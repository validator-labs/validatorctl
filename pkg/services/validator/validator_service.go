// Package validator provides functions for interacting with the validator and its plugins.
package validator

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	vtypes "github.com/validator-labs/validator/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
)

var (
	pluginFuncs = map[string]func(*components.ValidatorConfig, kubernetes.Interface) error{
		"AWS":     readAwsPlugin,
		"Azure":   readAzurePlugin,
		"Network": readNetworkPlugin,
		"OCI":     readOciPlugin,
		"vSphere": readVspherePlugin,
	}
	plugins = make([]string, 0, len(pluginFuncs))
)

func init() {
	for k := range pluginFuncs {
		plugins = append(plugins, k)
	}
	slices.Sort(plugins)
}

// ReadValidatorConfig prompts the user to configure the validator and its plugins
// nolint:gocyclo
func ReadValidatorConfig(c *cfg.Config, tc *cfg.TaskConfig, vc *components.ValidatorConfig) error {
	log.Header("Enter Validator Configuration")

	var err error
	var k8sClient kubernetes.Interface

	log.Header("Kind Configuration")
	vc.KindConfig.UseKindCluster, err = prompts.ReadBool("Provision & use kind cluster", true)
	if err != nil {
		return err
	}
	if vc.KindConfig.UseKindCluster {
		if err := kind.ValidateClusters("Validator installation"); err != nil {
			return err
		}
		vc.Kubeconfig = filepath.Join(c.RunLoc, "kind-cluster.kubeconfig")
	} else {
		k8sClient, vc.Kubeconfig, err = services.ReadKubeconfig()
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

	log.Header("Sink Configuration")
	if err := readSinkConfig(vc, k8sClient); err != nil {
		return err
	}

	if err := readHelmRelease(cfg.Validator, k8sClient, vc, vc.Release, vc.ReleaseSecret); err != nil {
		return err
	}

	log.Header("Validator Plugin Configuration")

	vc.AWSPlugin.Enabled, err = prompts.ReadBool("Enable AWS plugin", true)
	if err != nil {
		return err
	}
	if vc.AWSPlugin.Enabled {
		if err = readAwsPlugin(vc, k8sClient); err != nil {
			return err
		}
	}

	vc.AzurePlugin.Enabled, err = prompts.ReadBool("Enable Azure plugin", true)
	if err != nil {
		return fmt.Errorf("failed to prompt for bool for enable Azure plugin: %w", err)
	}
	if vc.AzurePlugin.Enabled {
		if err = readAzurePlugin(vc, k8sClient); err != nil {
			return err
		}
	}

	vc.NetworkPlugin.Enabled, err = prompts.ReadBool("Enable Network plugin", true)
	if err != nil {
		return err
	}
	if vc.NetworkPlugin.Enabled {
		if err = readNetworkPlugin(vc, k8sClient); err != nil {
			return err
		}
	}

	vc.OCIPlugin.Enabled, err = prompts.ReadBool("Enable OCI plugin", true)
	if err != nil {
		return err
	}
	if vc.OCIPlugin.Enabled {
		if err = readOciPlugin(vc, k8sClient); err != nil {
			return err
		}
	}

	vc.VspherePlugin.Enabled, err = prompts.ReadBool("Enable vSphere plugin", true)
	if err != nil {
		return err
	}
	if vc.VspherePlugin.Enabled {
		if err = readVspherePlugin(vc, k8sClient); err != nil {
			return err
		}
	}

	log.Header("Finalize Configuration")
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
			pluginFunc, err := prompts.Select("Plugin", plugins)
			if err != nil {
				return err
			}
			if err := pluginFuncs[pluginFunc](vc, k8sClient); err != nil {
				return err
			}
			continue
		}
		break
	}

	return nil
}

// UpdateValidatorCredentials updates the validator credentials
func UpdateValidatorCredentials(c *components.ValidatorConfig) error {
	var err error
	var k8sClient kubernetes.Interface

	if !c.KindConfig.UseKindCluster {
		k8sClient, c.Kubeconfig, err = services.ReadKubeconfig()
		if err != nil {
			return err
		}
	}

	if c.RegistryConfig.Enabled {
		if err := readRegistryConfig(c); err != nil {
			return err
		}
	}

	if err := readHelmCredentials(c.Release, c.ReleaseSecret, k8sClient, c); err != nil {
		return err
	}

	if c.AWSPlugin != nil && c.AWSPlugin.Enabled {
		if err := readHelmCredentials(c.AWSPlugin.Release, c.AWSPlugin.ReleaseSecret, k8sClient, c); err != nil {
			return err
		}
		if err := readAwsCredentials(c.AWSPlugin, k8sClient); err != nil {
			return err
		}
	}
	if c.AzurePlugin != nil && c.AzurePlugin.Enabled {
		if err := readHelmCredentials(c.AzurePlugin.Release, c.AzurePlugin.ReleaseSecret, k8sClient, c); err != nil {
			return err
		}
		if err := readAzureCredentials(c.AzurePlugin, k8sClient); err != nil {
			return err
		}
	}
	if c.OCIPlugin != nil && c.OCIPlugin.Enabled {
		if err := readHelmCredentials(c.OCIPlugin.Release, c.OCIPlugin.ReleaseSecret, k8sClient, c); err != nil {
			return err
		}
		for _, secret := range c.OCIPlugin.Secrets {
			if err := readSecret(secret); err != nil {
				return err
			}
		}
	}
	if c.VspherePlugin != nil && c.VspherePlugin.Enabled {
		if err = readHelmCredentials(c.VspherePlugin.Release, c.VspherePlugin.ReleaseSecret, k8sClient, c); err != nil {
			return err
		}
		if err := readVsphereCredentials(c.VspherePlugin, k8sClient); err != nil {
			return err
		}
	}

	return nil
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
