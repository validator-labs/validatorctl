// Package validator provides functions for interacting with the validator and its plugins.
package validator

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"emperror.dev/errors"
	vtypes "github.com/validator-labs/validator/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	awsconsts "github.com/validator-labs/validator-plugin-aws/pkg/constants"
	azureconsts "github.com/validator-labs/validator-plugin-azure/pkg/constants"
	maasconsts "github.com/validator-labs/validator-plugin-maas/pkg/constants"
	netconsts "github.com/validator-labs/validator-plugin-network/pkg/constants"
	ociconsts "github.com/validator-labs/validator-plugin-oci/pkg/constants"
	vsphereconsts "github.com/validator-labs/validator-plugin-vsphere/pkg/constants"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/utils/exec"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	"github.com/validator-labs/validatorctl/pkg/utils/kube"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
)

type pluginFuncMap map[string]func(*components.ValidatorConfig, *cfg.TaskConfig, kubernetes.Interface) error

var (
	pluginInstallFuncs = pluginFuncMap{
		awsconsts.PluginCode:     readAwsPlugin,
		azureconsts.PluginCode:   readAzurePlugin,
		maasconsts.PluginCode:    readMaasPlugin,
		netconsts.PluginCode:     readNetworkPlugin,
		ociconsts.PluginCode:     readOciPlugin,
		vsphereconsts.PluginCode: readVspherePlugin,
	}
	pluginRuleFuncs = pluginFuncMap{
		awsconsts.PluginCode:     readAwsPluginRules,
		azureconsts.PluginCode:   readAzurePluginRules,
		maasconsts.PluginCode:    readMaasPluginRules,
		netconsts.PluginCode:     readNetworkPluginRules,
		ociconsts.PluginCode:     readOciPluginRules,
		vsphereconsts.PluginCode: readVspherePluginRules,
	}
	plugins = make([]string, 0, len(pluginInstallFuncs))
)

func init() {
	for k := range pluginInstallFuncs {
		plugins = append(plugins, k)
	}
	slices.Sort(plugins)
}

// ReadValidatorConfig prompts the user to configure installation settings for validator and its plugins.
// nolint:gocyclo
func ReadValidatorConfig(c *cfg.Config, tc *cfg.TaskConfig, vc *components.ValidatorConfig) error {
	log.Header("Enter Validator Configuration")
	log.InfoCLI(`
	You will be prompted for the following configuration:

	  - Kubernetes cluster configuration
	  - Proxy configuration
	  - Artifact registry configuration
	  - Sink configuration
	  - Validator plugin(s) to install

	If you make a mistake at any point you will have to option
	to revisit any configuration step at the end.
	`)

	var err error
	var kClient kubernetes.Interface

	log.Header("Kind Configuration")
	vc.KindConfig.UseKindCluster, err = prompts.ReadBool("Provision & use kind cluster", true)
	if err != nil {
		return err
	}
	if vc.KindConfig.UseKindCluster {
		if err := exec.CheckBinaries([]exec.Binary{exec.DockerBin, exec.KindBin}); err != nil {
			return err
		}
		if err := kind.ValidateClusters("Validator installation"); err != nil {
			return err
		}
		// only set kubeconfig if a kind cluster will be created
		if !tc.CreateConfigOnly {
			vc.Kubeconfig = filepath.Join(c.RunLoc, "kind-cluster.kubeconfig")
		}
	} else {
		kClient, vc.Kubeconfig, err = services.ReadKubeconfig()
		if err != nil {
			return err
		}
	}

	log.Header("Proxy Configuration")
	if err := readProxyConfig(vc); err != nil {
		return err
	}

	log.Header("Artifact Registry Configuration")
	if err := readRegistryConfig(vc); err != nil {
		return err
	}

	log.Header("Helm Configuration")
	if err := readHelmConfig(cfg.Validator, kClient, vc, vc.ReleaseSecret); err != nil {
		return err
	}

	log.Header("Sink Configuration")
	log.InfoCLI(`
	If sink configuration is provided, validator will upload all plugin validation
	results to either Slack or Alertmanager. Results are hashed so that new events
	are emitted only when the validation result changes.
	`)
	if err := readSinkConfig(vc, kClient); err != nil {
		return err
	}

	// Configure validator HelmRelease
	if err := readHelmRelease(cfg.Validator, vc, vc.Release); err != nil {
		return err
	}

	log.Header("Validator Plugin Configuration")
	log.InfoCLI(`
	Validator plugins provide informative, actionable validation results pertaining
	to infrastructure, networking, kubernetes cluster internals, and more.

	Pick and choose from them to craft a validation profile that meets your
	organization's requirements.
	`)

	// Enable plugin(s) & read install config
	if err := handlePlugins(vc, tc, kClient, "Install", true, pluginInstallFuncs); err != nil {
		return err
	}

	log.Header("Finalize Installation Configuration")
	restart, err := prompts.ReadBool("Restart configuration", false)
	if err != nil {
		return err
	}
	if restart {
		return ReadValidatorConfig(c, tc, vc)
	}
	for {
		revisit, err := prompts.ReadBool("Reconfigure plugin(s)", false)
		if err != nil {
			return err
		}
		if revisit {
			pluginCode, err := prompts.Select("Plugin", plugins)
			if err != nil {
				return err
			}
			if err := pluginInstallFuncs[pluginCode](vc, tc, kClient); err != nil {
				return err
			}
			continue
		}
		break
	}

	return nil
}

// handlePlugins supports three distinct logical flows:
// 1. validator install
//   - prompt to enable plugins
//   - verb is Install & tc.Direct will never be true, so readXYZPlugin calls are never hit
//
// 2. validator check (without --direct)
//   - no prompt to enable plugins as they're already enabled in the validator config file
//   - verb is Enable & tc.Direct is false, so readXYZPlugin are never called, but readXYZPluginRules are
//
// 3. validator check --direct
//   - prompt to enable plugins (since we don't have a validator config file)
//   - verb is Enable & tc.Direct is true, so readXYZPlugin and readXYZPluginRules are called
//
// nolint:gocyclo
func handlePlugins(vc *components.ValidatorConfig, tc *cfg.TaskConfig, kClient kubernetes.Interface, verb string, enablePlugins bool, funcMap pluginFuncMap) error {
	var err error

	if enablePlugins {
		log.Header("AWS Plugin")
		log.InfoCLI(`
	The AWS validator plugin reconciles AwsValidator custom resources to perform the
	following validations against your AWS environment:

	- Ensure that one or more EC2 AMI(s) exist in a particular region.
	- Compare the IAM permissions associated with an IAM user / group / role / policy
	  against an expected permission set.
	- Compare the usage for a particular service quota against the active quota to
	  avoid unexpectedly hitting quota limits.
	- Compare the tags associated with a subnet against an expected tag set.
		`)
		vc.AWSPlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s AWS plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.AWSPlugin.Enabled {
		if tc.Direct {
			if err = readAwsPlugin(vc, tc, kClient); err != nil {
				return err
			}
		}
		if err := funcMap[awsconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	// TODO: support image gallery rules
	// - Verify that images in community image galleries exist.
	if enablePlugins {
		log.Header("Azure Plugin")
		log.InfoCLI(`
	The Azure validator plugin reconciles AzureValidator custom resources to perform
	the following validations against your Azure environment:

	- Compare the Azure RBAC permissions associated with a security principal against
	  an expected permission set.
		`)
		vc.AzurePlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s Azure plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.AzurePlugin.Enabled {
		if tc.Direct {
			if err = readAzurePlugin(vc, tc, kClient); err != nil {
				return err
			}
		}

		if err := funcMap[azureconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	if enablePlugins {
		log.Header("MAAS Plugin")
		log.InfoCLI(`
	The MAAS validator plugin reconciles MaasValidator custom resources to perform
	the following validation against your MAAS cluster:

	- Check that a certain number of Upstream DNS servers are configured.
	- Check that the expected Internal DNS settings are present.
	- Check that the expected number of Machines matching a certain spec are "Ready" for use in each Availability Zone.
	- Check that all required OS Images are "Synced" to the cluster.
		`)
		vc.MaasPlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s MAAS plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.MaasPlugin.Enabled {
		if tc.Direct {
			if err = readMaasPlugin(vc, tc, kClient); err != nil {
				return err
			}
		}
		if err := funcMap[maasconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	if enablePlugins {
		log.Header("Network Plugin")
		log.InfoCLI(`
	The Network validator plugin reconciles NetworkValidator custom resources to perform
	the following validations against your network:

	- Execute DNS lookups.
	- Execute ICMP pings.
	- Validate TCP connections to arbitrary host + port(s).
	- Check each IP in an IP range to ensure that they're all unallocated.
	- Check that the default NIC has an MTU greater than or equal to a specified value.
	- Check that each file in a list of URLs is available and publicly accessible
	  via an HTTP HEAD request, with optional basic auth.
		`)
		vc.NetworkPlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s Network plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.NetworkPlugin.Enabled {
		if tc.Direct {
			if err := exec.CheckBinaries([]exec.Binary{exec.NslookupBin, exec.PingBin}); err != nil {
				return err
			}
			if err = readNetworkPlugin(vc, tc, kClient); err != nil {
				return err
			}
		}
		if err := funcMap[netconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	if enablePlugins {
		log.Header("OCI Plugin")
		log.InfoCLI(`
	The OCI validator plugin reconciles OciValidator custom resources to perform the
	following validations against your OCI registry:

	- Validate OCI registry authentication.
	- Validate the existence of arbitrary OCI artifacts, with optional signature
	  verification.
	- Validate downloading arbitrary OCI artifacts.
		`)
		vc.OCIPlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s OCI plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.OCIPlugin.Enabled {
		if tc.Direct {
			if err = readOciPlugin(vc, tc, kClient); err != nil {
				return err
			}
		}
		if err := funcMap[ociconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	if enablePlugins {
		log.Header("vSphere Plugin")
		log.InfoCLI(`
	The vSphere validator plugin reconciles VsphereValidator custom resources to perform
	the following validations against your vSphere environment:

	- Compare the privileges associated with a user against an expected privileges set.
	- Compare the privileges associated with a user against an expected privileges set
	  on a particular entity (cluster, resourcepool, folder, vapp, host).
	- Verify availability of compute resources on an ESXi host, resourcepool, or cluster.
	- Compare the tags associated with a datacenter, cluster, host, vm, resourcepool or vm
	  against an expected tag set.
	- Verify that a set of ESXi hosts have valid NTP configuration.
		`)
		vc.VspherePlugin.Enabled, err = prompts.ReadBool(fmt.Sprintf("%s vSphere plugin", verb), true)
		if err != nil {
			return err
		}
	}
	if vc.VspherePlugin.Enabled {
		if tc.Direct {
			if err = readVspherePlugin(vc, tc, kClient); err != nil {
				return err
			}
		}
		if err := funcMap[vsphereconsts.PluginCode](vc, tc, kClient); err != nil {
			return err
		}
	}

	return nil
}

// ReadValidatorPluginConfig prompts the user to configure validator plugins rule(s).
func ReadValidatorPluginConfig(c *cfg.Config, tc *cfg.TaskConfig, vc *components.ValidatorConfig) error {
	var err error
	var enablePlugins bool
	var kClient kubernetes.Interface

	if tc.Direct {
		enablePlugins = true
	} else if !tc.CreateConfigOnly {
		if vc.Kubeconfig == "" {
			if vc.KindConfig.UseKindCluster {
				return errors.New(`config file has kindConfig.useKindCluster set to true, but no kubeconfig path was provided. Have you run "validator install" yet?`)
			}
			kClient, vc.Kubeconfig, err = services.ReadKubeconfig()
			if err != nil {
				return err
			}
		} else {
			kClient, err = kube.GetKubeClientset(vc.Kubeconfig)
			if err != nil {
				return err
			}
		}
		log.InfoCLI("")
	}

	log.Header("Validator Plugin Configuration")
	if tc.Direct {
		log.InfoCLI(`
	You will be prompted to enable validator plugins and 
	configure rules for each enabled plugin. The rules will be
	evaluated directly, in-process. Results will be saved to
	disk and printed to the console.

	If you make a mistake at any point you will have to option
	to revisit any configuration step at the end.
		`)
	} else {
		log.InfoCLI(`
	You will be prompted to configure validator plugin rules
	for each enabled plugin in your validator configuration file.

	Custom Resouces containing plugin rules will be applied to the
	Kubernetes cluster specified by the KUBECONFIG environment variable.

	If you make a mistake at any point you will have to option
	to revisit any configuration step at the end.
		`)
	}

	// If direct, enable plugin(s) & read partial install config & rule config.
	// If not direct, read rule config for enabled plugin(s).
	if err := handlePlugins(vc, tc, kClient, "Enable", enablePlugins, pluginRuleFuncs); err != nil {
		return err
	}

	log.Header("Finalize Plugin Rule Configuration")
	restart, err := prompts.ReadBool("Restart configuration", false)
	if err != nil {
		return err
	}
	if restart {
		return ReadValidatorPluginConfig(c, tc, vc)
	}
	for {
		revisit, err := prompts.ReadBool("Reconfigure plugin(s)", false)
		if err != nil {
			return err
		}
		if revisit {
			pluginCode, err := prompts.Select("Plugin", plugins)
			if err != nil {
				return err
			}
			if err := pluginRuleFuncs[pluginCode](vc, tc, kClient); err != nil {
				return err
			}
			continue
		}
		break
	}

	return nil
}

// UpdateValidatorCredentials updates validator credentials
func UpdateValidatorCredentials(c *components.ValidatorConfig) error {
	if c.RegistryConfig.Enabled {
		if err := readRegistryConfig(c); err != nil {
			return fmt.Errorf("failed to update registry config: %w", err)
		}
	}
	k8sClient, err := k8sClientFromConfig(c)
	if err != nil {
		return err
	}
	if err := readHelmConfig(cfg.Validator, k8sClient, c, c.ReleaseSecret); err != nil {
		return fmt.Errorf("failed to update Helm configuration: %w", err)
	}
	return nil
}

// UpdateValidatorPluginCredentials updates validator plugin credentials
func UpdateValidatorPluginCredentials(c *components.ValidatorConfig, tc *cfg.TaskConfig) error {
	k8sClient, err := k8sClientFromConfig(c)
	if err != nil {
		return err
	}
	if c.AWSPlugin != nil && c.AWSPlugin.Enabled {
		if err := readAwsCredentials(c.AWSPlugin, tc, k8sClient); err != nil {
			return fmt.Errorf("failed to update AWS credentials: %w", err)
		}
	}
	if c.AzurePlugin != nil && c.AzurePlugin.Enabled {
		if err := readAzureCredentials(c.AzurePlugin, tc, k8sClient); err != nil {
			return fmt.Errorf("failed to update Azure credentials: %w", err)
		}
	}
	if c.MaasPlugin != nil && c.MaasPlugin.Enabled {
		if err := readMaasCredentials(c.MaasPlugin, tc, k8sClient); err != nil {
			return fmt.Errorf("failed to update MAAS credentials: %w", err)
		}
	}
	if c.OCIPlugin != nil && c.OCIPlugin.Enabled {
		for _, secret := range c.OCIPlugin.Secrets {
			if err := readOciSecret(secret); err != nil {
				return fmt.Errorf("failed to update OCI secret: %w", err)
			}
		}
	}
	if c.VspherePlugin != nil && c.VspherePlugin.Enabled {
		if err := readVsphereCredentials(c.VspherePlugin, tc, k8sClient); err != nil {
			return fmt.Errorf("failed to update vSphere credentials: %w", err)
		}
	}
	return nil
}

func k8sClientFromConfig(c *components.ValidatorConfig) (kubernetes.Interface, error) {
	var err error
	var k8sClient kubernetes.Interface

	if !c.KindConfig.UseKindCluster {
		k8sClient, c.Kubeconfig, err = services.ReadKubeconfig()
		if err != nil {
			return nil, err
		}
	}

	return k8sClient, nil
}

func readRegistryConfig(vc *components.ValidatorConfig) (err error) {
	airgapped, err := prompts.ReadBool("Configure Hauler for air-gapped installation", false)
	if err != nil {
		return err
	}
	if airgapped {
		vc.RegistryConfig.Enabled = true
		vc.RegistryConfig.Registry.IsAirgapped = true
		vc.UseFixedVersions = true
		if err = services.ReadHaulerProps(vc.RegistryConfig.Registry, vc.ProxyConfig.Env); err != nil {
			return err
		}
		vc.ImageRegistry = vc.RegistryConfig.Registry.ImageEndpoint()
		return nil
	}

	privateRegistry, err := prompts.ReadBool("Configure private OCI registry", false)
	if err != nil {
		return err
	}
	if privateRegistry {
		vc.RegistryConfig.Enabled = true
		if err := services.ReadRegistryProps(vc.RegistryConfig.Registry, vc.ProxyConfig.Env); err != nil {
			return err
		}
		vc.ImageRegistry = vc.RegistryConfig.Registry.ImageEndpoint()
		return nil
	}

	// public registry configuration
	imageRegistry := cfg.ValidatorImagePath()
	if vc.ImageRegistry != "" {
		imageRegistry = vc.ImageRegistry
	}
	vc.ImageRegistry, err = prompts.ReadText("Validator image registry", imageRegistry, false, -1)
	if err != nil {
		return err
	}
	return nil

}

func readProxyConfig(vc *components.ValidatorConfig) error {
	vc.ProxyConfig.Env.PodCIDR = &cfg.DefaultPodCIDR
	vc.ProxyConfig.Env.ServiceIPRange = &cfg.DefaultServiceIPRange

	configureProxy, err := prompts.ReadBool("Configure an HTTP proxy", false)
	if err != nil {
		return err
	}
	if !configureProxy {
		vc.ProxyConfig.Enabled = false
		return nil
	}
	if err := services.ReadProxyProps(vc.ProxyConfig.Env); err != nil {
		return err
	}
	vc.ProxyConfig.Enabled = vc.ProxyConfig.Env.ProxyCACert.Path != ""

	return nil
}

func readSinkConfig(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	var err error
	vc.SinkConfig.Enabled, err = prompts.ReadBool("Configure a sink", false)
	if err != nil {
		return err
	}
	if !vc.SinkConfig.Enabled {
		return nil
	}

	sinkType, err := prompts.Select("Sink Type", sinkTypes())
	if err != nil {
		return err
	}
	vc.SinkConfig.Type = strings.ToLower(sinkType)

	// always create sink credential secret if creating a new kind cluster
	vc.SinkConfig.CreateSecret = true

	if k8sClient != nil {
		keys := cfg.ValidatorSinkKeys[vtypes.SinkType(vc.SinkConfig.Type)]
		log.InfoCLI(`
	Either specify sink credentials or provide the name of a secret in the target K8s cluster's %s namespace.
	If using an existing secret, it must contain the following keys: %+v.
	`, cfg.Validator, keys,
		)
		vc.SinkConfig.CreateSecret, err = prompts.ReadBool("Create sink credential secret", true)
		if err != nil {
			return err
		}
		if !vc.SinkConfig.CreateSecret {
			secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, keys)
			if err != nil {
				return err
			}
			vc.SinkConfig.SecretName = secret.Name
			return nil
		}
	}

	vc.SinkConfig.SecretName, err = prompts.ReadText("Sink credentials secret name", "sink-creds", false, -1)
	if err != nil {
		return err
	}

	switch vc.SinkConfig.Type {
	case string(vtypes.SinkTypeAlertmanager):
		if vc.SinkConfig.Values == nil {
			vc.SinkConfig.Values = map[string]string{
				"endpoint": "",
				"caCert":   "",
				"username": "",
				"password": "",
			}
		}

		endpoint, err := prompts.ReadURL(
			"Alertmanager endpoint", vc.SinkConfig.Values["endpoint"], "Alertmanager endpoint must be a valid URL", false,
		)
		if err != nil {
			return err
		}
		vc.SinkConfig.Values["endpoint"] = endpoint

		insecure, err := prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
		if err != nil {
			return err
		}
		vc.SinkConfig.Values["insecureSkipVerify"] = strconv.FormatBool(insecure)

		if !insecure {
			var caCertData []byte
			_, _, caCertData, err = prompts.ReadCACert("Alertmanager CA certificate filepath", vc.SinkConfig.Values["caCert"], "")
			if err != nil {
				return err
			}
			vc.SinkConfig.Values["caCert"] = string(caCertData)
		}

		username, password, err := prompts.ReadBasicCreds(
			"Alertmanager Username", "Alertmanager Password",
			vc.SinkConfig.Values["username"], vc.SinkConfig.Values["password"], true, false,
		)
		if err != nil {
			return err
		}
		vc.SinkConfig.Values["username"] = username
		vc.SinkConfig.Values["password"] = password

	case string(vtypes.SinkTypeSlack):
		if vc.SinkConfig.Values == nil {
			vc.SinkConfig.Values = map[string]string{
				"apiToken":  "",
				"channelID": "",
			}
		}

		botToken, err := prompts.ReadPassword("Bot token", vc.SinkConfig.Values["apiToken"], false, -1)
		if err != nil {
			return err
		}
		vc.SinkConfig.Values["apiToken"] = botToken

		channelID, err := prompts.ReadText("Channel ID", vc.SinkConfig.Values["channelID"], false, -1)
		if err != nil {
			return err
		}
		vc.SinkConfig.Values["channelID"] = channelID
	}

	return nil
}

func sinkTypes() []string {
	return []string{
		string_utils.Capitalize(string(vtypes.SinkTypeAlertmanager)),
		string_utils.Capitalize(string(vtypes.SinkTypeSlack)),
	}
}
