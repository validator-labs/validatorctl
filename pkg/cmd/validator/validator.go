package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	vapi "github.com/validator-labs/validator/api/v1alpha1"
	"github.com/validator-labs/validator/pkg/helm"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services/validator"
	embed_utils "github.com/validator-labs/validatorctl/pkg/utils/embed"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	"github.com/validator-labs/validatorctl/pkg/utils/kube"
)

func DeployValidatorCommand(c *cfg.Config, tc *cfg.TaskConfig, reconfigure bool) error {
	var vc *components.ValidatorConfig
	var err error
	var saveConfig bool

	if tc.ConfigFile == "" && reconfigure {
		log.FatalCLI("Cannot reconfigure validator without providing a configuration file.")
	}

	if tc.ConfigFile != "" && !reconfigure {
		// Silent Mode
		vc, err = components.NewValidatorFromConfig(tc)
		if err != nil {
			return errors.Wrap(err, "failed to load validator configuration file")
		}
		if tc.UpdatePasswords {
			log.Header("Updating credentials in validator configuration file")
			if err := validator.UpdateValidatorCredentials(vc); err != nil {
				return err
			}
			saveConfig = true
		}
		if vc.Kubeconfig == "" {
			vc.Kubeconfig = filepath.Join(c.RunLoc, "kind-cluster.kubeconfig")
			saveConfig = true
		}
	} else {
		// Interactive mode
		if reconfigure {
			vc, err = components.NewValidatorFromConfig(tc)
			if err != nil {
				return errors.Wrap(err, "failed to load validator configuration file")
			}
		} else {
			vc = components.NewValidatorConfig()
		}

		// for dev build versions, we allow selection of specific validator and plugin versions
		// for all other builds, we set a fixed version for the validator and plugins
		vc.UseFixedVersions = !strings.HasSuffix(tc.CliVersion, "-dev")
		if err := validator.ReadValidatorConfig(c, tc, vc); err != nil {
			return errors.Wrap(err, "failed to create new validator configuration")
		}
		tc.ConfigFile = filepath.Join(c.RunLoc, cfg.ValidatorConfigFile)
		saveConfig = true
	}

	// save / print validator config file
	if saveConfig {
		if err := components.SaveValidatorConfig(vc, tc); err != nil {
			return err
		}
	} else {
		log.InfoCLI("validator configuration file: %s", tc.ConfigFile)
	}

	if tc.CreateConfigOnly || tc.UpdatePasswords {
		return nil
	}

	if vc.UseKindCluster {
		if err := createKindCluster(c, vc); err != nil {
			return err
		}
	}

	return applyValidatorAndPlugins(c, vc)
}

func UpgradeValidatorCommand(c *cfg.Config, taskConfig *cfg.TaskConfig) error {
	vc, err := components.NewValidatorFromConfig(taskConfig)
	if err != nil {
		return errors.Wrap(err, "failed to load validator configuration file")
	}
	if vc.Kubeconfig == "" {
		return errors.New("invalid validator configuration: kubeconfig is required")
	}
	return applyValidatorAndPlugins(c, vc)
}

func UndeployValidatorCommand(taskConfig *cfg.TaskConfig, deleteCluster bool) error {
	vc, err := components.NewValidatorFromConfig(taskConfig)
	if err != nil {
		return errors.Wrap(err, "failed to load validator configuration file")
	}

	log.Header("Uninstalling validator")
	helmClient, err := getHelmClient(vc)
	if err != nil {
		return err
	}
	if err := helmClient.Delete(cfg.Validator, cfg.Validator); err != nil {
		return errors.Wrap(err, "failed to delete validator Helm release")
	}
	log.InfoCLI("\nUninstalled validator and validator plugin(s) successfully")

	if vc.UseKindCluster && deleteCluster {
		return kind.DeleteCluster(cfg.ValidatorKindClusterName)
	}

	return nil
}

func DescribeValidationResultsCommand(taskConfig *cfg.TaskConfig) error {
	kClient, err := getValidationResultsCRDClient(taskConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get validation result client")
	}

	vrs, err := kClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list validation results")
	}

	if err := printValidationResults(vrs.Items); err != nil {
		return err
	}

	return nil
}
func getValidationResultsCRDClient(taskConfig *cfg.TaskConfig) (dynamic.NamespaceableResourceInterface, error) {
	if taskConfig.ConfigFile != "" {
		vc, err := components.NewValidatorFromConfig(taskConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load validator configuration file")
		}
		if err := os.Setenv("KUBECONFIG", vc.Kubeconfig); err != nil {
			return nil, err
		}
		log.InfoCLI("Using kubeconfig from validator configuration file: %s", vc.Kubeconfig)
	}

	gv := kube.GetGroupVersion("validation.spectrocloud.labs", "v1alpha1")
	kClient, err := kube.GetCRDClient(gv, "validationresults")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validation result client")
	}

	return kClient, nil
}

func printValidationResults(validationResults []unstructured.Unstructured) error {
	for _, vrObj := range validationResults {
		vrStr, err := buildValidationResultString(vrObj)
		if err != nil {
			return err
		}
		log.InfoCLI(vrStr)
	}

	return nil
}

func buildValidationResultString(vrObj unstructured.Unstructured) (string, error) {
	vr := &vapi.ValidationResult{}
	bytes, err := vrObj.MarshalJSON()
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(bytes, vr); err != nil {
		return "", err
	}

	sb := &strings.Builder{}
	sb.WriteString("\n=================\nValidation Result\n=================\n")
	keys := []string{"Plugin", "Name", "Namespace", "State"}
	vals := []string{vr.Spec.Plugin, vr.Name, vr.Namespace, string(vr.Status.State)}

	for _, c := range vr.Status.Conditions {
		if c.Type == vapi.SinkEmission {
			keys = append(keys, "Sink State")
			vals = append(vals, string(c.Reason))
			break
		}
	}

	args := map[string]interface{}{
		"Keys":   keys,
		"Values": vals,
	}

	if err := embed_utils.PrintTableTemplate(sb, args, cfg.Validator, "validation-result.tmpl"); err != nil {
		return "", err
	}

	sb.WriteString("\n------------\nRule Results\n------------\n")
	for _, c := range vr.Status.ValidationConditions {
		args := map[string]interface{}{
			"Keys":   []string{"Validation Rule", "Validation Type", "Status", "Last Validated", "Message"},
			"Values": []string{c.ValidationRule, c.ValidationType, string(c.Status), c.LastValidationTime.Format(time.RFC3339), strings.TrimSpace(c.Message)},
		}

		if err := embed_utils.PrintTableTemplate(sb, args, cfg.Validator, "validation-result.tmpl"); err != nil {
			return "", err
		}

		for i, d := range c.Details {
			if i == 0 {
				sb.WriteString("\n-------\nDetails\n-------\n")
			}
			sb.WriteString(fmt.Sprintf("- %s\n", d))
		}
		for i, f := range c.Failures {
			if i == 0 {
				sb.WriteString("\n--------\nFailures\n--------\n")
			}
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
	}
	return sb.String(), nil
}

// applyValidatorAndPlugins installs/upgrades validator + plugin(s), then applies/updates validator CRs for each plugin
func applyValidatorAndPlugins(c *cfg.Config, vc *components.ValidatorConfig) error {
	log.Header("Installing/Upgrading validator and validator plugin(s)")

	if err := applyValidator(c, vc); err != nil {
		return err
	}
	log.InfoCLI("\nvalidator installed successfully")

	if err := applyPlugins(c, vc); err != nil {
		return err
	}
	log.Header("validator plugin(s) installed successfully")
	log.InfoCLI("\nPlugins will now execute validation checks.")

	log.InfoCLI("\nYou can list validation results via the following command:")
	log.InfoCLI("kubectl -n validator get validationresults --kubeconfig %s", vc.Kubeconfig)

	log.InfoCLI("\nAnd you can view all validation result details via the following command:")
	log.InfoCLI("kubectl -n validator describe validationresults --kubeconfig %s", vc.Kubeconfig)
	return nil
}

func createReleaseSecretCmd(secret *components.Secret) []string {
	args := []string{
		"create", "secret", "generic", secret.Name, "-n", "validator",
		// include empty username/password, even if unset, to avoid error in validator
		fmt.Sprintf("--from-literal=username=%s", secret.Username),
		fmt.Sprintf("--from-literal=password=%s", secret.Password),
	}
	if secret.CaCertFile != "" {
		args = append(args, fmt.Sprintf("--from-file=caCert=%s", secret.CaCertFile))
	}
	return args
}

func applyValidator(c *cfg.Config, vc *components.ValidatorConfig) error {
	kubecommands, kubecommandsPre := [][]string{}, [][]string{}
	kClient, err := kube.GetKubeClientset(vc.Kubeconfig)
	if err != nil {
		return err
	}

	// build validator plugin spec
	validatorSpec := vapi.ValidatorConfigSpec{
		Plugins: make([]vapi.HelmRelease, 0),
	}

	if vc.AWSPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.AWSPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-aws-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin aws values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.AWSPlugin.Release.Chart,
			Values: string(values),
		})
		if vc.AWSPlugin.ReleaseSecret != nil && vc.AWSPlugin.ReleaseSecret.ShouldCreate() {
			kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.AWSPlugin.ReleaseSecret))
		}
		kubecommands = append(kubecommands, cfg.ValidatorPluginAwsWaitCmd)
	}

	if vc.AzurePlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.AzurePlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-azure-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin azure values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.AzurePlugin.Release.Chart,
			Values: string(values),
		})
		if vc.AzurePlugin.ReleaseSecret != nil && vc.AzurePlugin.ReleaseSecret.ShouldCreate() {
			kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.AzurePlugin.ReleaseSecret))
		}
		kubecommands = append(kubecommands, cfg.ValidatorPluginAzureWaitCmd)
	}

	if vc.NetworkPlugin.Enabled {
		args := map[string]interface{}{
			"Tag":           vc.NetworkPlugin.Release.Chart.Version,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-network-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin network values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.NetworkPlugin.Release.Chart,
			Values: string(values),
		})
		if vc.NetworkPlugin.ReleaseSecret != nil && vc.NetworkPlugin.ReleaseSecret.ShouldCreate() {
			kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.NetworkPlugin.ReleaseSecret))
		}
		kubecommands = append(kubecommands, cfg.ValidatorPluginNetworkWaitCmd)
	}

	if vc.OCIPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.OCIPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-oci-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin oci values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.OCIPlugin.Release.Chart,
			Values: string(values),
		})
		if vc.OCIPlugin.ReleaseSecret != nil && vc.OCIPlugin.ReleaseSecret.ShouldCreate() {
			kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.OCIPlugin.ReleaseSecret))
		}
		kubecommands = append(kubecommands, cfg.ValidatorPluginOciWaitCmd)
	}

	if vc.VspherePlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.VspherePlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-vsphere-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin vsphere values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.VspherePlugin.Release.Chart,
			Values: string(values),
		})
		if vc.VspherePlugin.ReleaseSecret != nil && vc.VspherePlugin.ReleaseSecret.ShouldCreate() {
			kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.VspherePlugin.ReleaseSecret))
		}
		kubecommands = append(kubecommands, cfg.ValidatorPluginVsphereWaitCmd)
	}

	if !vc.AnyPluginEnabled() {
		log.FatalCLI("Invalid validator config: at least one plugin must be enabled!")
	}

	// concatenate base validator values w/ plugin values
	args := map[string]interface{}{
		"ImageRegistry": vc.ImageRegistry,
		"Tag":           vc.Release.Chart.Version,
		"ProxyConfig":   vc.ProxyConfig,
		"SinkConfig":    vc.SinkConfig,
		"AWSPlugin":     vc.AWSPlugin,
		"VspherePlugin": vc.VspherePlugin,
		"OCIPlugin":     vc.OCIPlugin,
		"AzurePlugin":   vc.AzurePlugin,
	}
	if vc.ProxyConfig.Enabled {
		args["ProxyCaCertData"] = strings.Split(vc.ProxyConfig.Env.ProxyCaCertData, "\n")
	}

	values, err := embed_utils.RenderTemplateBytes(args, cfg.Validator, "validator-base-values.tmpl")
	if err != nil {
		return errors.Wrap(err, "failed to render validator base values.yaml")
	}
	pluginValues, err := yaml.Marshal(validatorSpec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal validator plugin YAML")
	}
	pluginValues = bytes.ReplaceAll(pluginValues, []byte("sink: null"), nil)
	values = append(values, pluginValues...)
	finalValues := string(values)
	log.Debug("applying validator helm chart with values:")
	log.Debug(finalValues)

	// install validator helm chart

	if len(kubecommandsPre) > 0 {
		_, err := kClient.CoreV1().Namespaces().Get(context.Background(), cfg.Validator, metav1.GetOptions{})
		if err != nil && apierrs.IsNotFound(err) {
			kubecommandsPre = append([][]string{{"create", "namespace", cfg.Validator}}, kubecommandsPre...)
		}
		for _, c := range kubecommandsPre {
			if _, stderr, err := kube.KubectlCommand(c, vc.Kubeconfig); err != nil {
				// ignore already exists errors when creating release secrets
				if !strings.HasSuffix(strings.TrimSpace(stderr), "already exists") {
					return errors.Wrap(err, stderr)
				}
				log.Debug(stderr)
			}
		}
	}

	helmClient, err := getHelmClient(vc)
	if err != nil {
		return err
	}
	opts := helm.Options{
		Chart:                 vc.Release.Chart.Name,
		Repo:                  vc.Release.Chart.Repository,
		CaFile:                vc.Release.Chart.CaFile,
		InsecureSkipTlsVerify: vc.Release.Chart.InsecureSkipTlsVerify,
		Version:               vc.Release.Chart.Version,
		Values:                finalValues,
		CreateNamespace:       true,
	}
	if vc.ReleaseSecret != nil {
		opts.Username = vc.ReleaseSecret.Username
		opts.Password = vc.ReleaseSecret.Password
	}

	var cleanupLocalChart bool
	if strings.HasPrefix(opts.Repo, "oci://") {
		log.InfoCLI("\n==== Pulling validator Helm chart from OCI repository %s ====", opts.Repo)

		opts.Untar = true
		opts.UntarDir = c.RunLoc
		opts.Version = strings.TrimPrefix(opts.Version, "v")

		if err := helmClient.Pull(opts); err != nil {
			return err
		}
		log.InfoCLI("Pulled plugin Helm chart %s from OCI repository", opts.Chart)

		opts.Path = fmt.Sprintf("%s/%s", c.RunLoc, opts.Chart)
		opts.Chart = ""
		cleanupLocalChart = true
		log.InfoCLI("Reconfigured Helm options to deploy local chart")
	}

	log.InfoCLI("\n==== Installing/upgrading validator Helm chart ====")
	if err := helmClient.Upgrade(cfg.Validator, cfg.Validator, opts); err != nil {
		return errors.Wrap(err, "failed to install validator helm chart")
	}
	if cleanupLocalChart {
		if err := os.RemoveAll(opts.Path); err != nil {
			return errors.Wrap(err, "failed to remove local chart directory")
		}
		log.InfoCLI("Cleaned up local chart directory: %s", opts.Path)
	}

	// wait for validator to be ready
	if _, stderr, err := kube.KubectlCommand(cfg.ValidatorWaitCmd, vc.Kubeconfig); err != nil {
		return errors.Wrap(err, stderr)
	}
	log.InfoCLI("Pausing for 20s for validator to establish a lease & begin plugin installation")
	time.Sleep(20 * time.Second)

	// wait for validator plugin(s) to be ready
	for _, c := range kubecommands {
		if _, stderr, err := kube.KubectlCommand(c, vc.Kubeconfig); err != nil {
			return errors.Wrap(err, stderr)
		}
	}

	return nil
}

// getHelmClient gets a helm client w/ a monkey-patched path to the embedded kind binary
func getHelmClient(vc *components.ValidatorConfig) (helm.HelmClient, error) {
	apiCfg, err := kube.GetAPIConfig(vc.Kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API config from kubeconfig")
	}
	helm.CommandPath = exec_utils.Helm
	helmClient := helm.NewHelmClient(apiCfg)
	return helmClient, nil
}

func applyPlugins(c *cfg.Config, vc *components.ValidatorConfig) error {
	if vc.AWSPlugin.Enabled {
		log.InfoCLI("\n==== Applying AWS plugin validator(s) ====")

		// render & apply any AWS validator rules other than IAM
		if vc.AWSPlugin.Validator.ResultCount() > 0 {
			if err := createValidator(
				vc.Kubeconfig, c.RunLoc, "rules", cfg.ValidatorPluginAwsTemplate, *vc.AWSPlugin.Validator,
			); err != nil {
				return err
			}
		}

		// render & apply AWS IAM validator
		if vc.AWSPlugin.IamCheck.Enabled {
			template := cfg.ValidatorPluginAwsIamMap[vc.AWSPlugin.IamCheck.Type]
			outputPath := strings.ReplaceAll(filepath.Join(c.RunLoc, "manifests", template), ".tmpl", ".yaml")
			auth, err := yaml.Marshal(vc.AWSPlugin.Validator.Auth)
			if err != nil {
				return errors.Wrap(err, "failed to marshal AWS plugin auth")
			}
			args := map[string]interface{}{
				"Auth":        indent(auth, 4),
				"IamRoleName": vc.AWSPlugin.IamCheck.IamRoleName,
			}
			if err := embed_utils.RenderTemplate(args, cfg.Validator, template, outputPath); err != nil {
				return err
			}
			if err := applyValidatorManifest(vc.Kubeconfig, cfg.ValidatorPluginAws, outputPath); err != nil {
				return err
			}
		}
	}

	if vc.VspherePlugin.Enabled {
		log.InfoCLI("\n==== Applying vSphere plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, "rules", cfg.ValidatorPluginVsphereTemplate, *vc.VspherePlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.NetworkPlugin.Enabled {
		log.InfoCLI("\n==== Applying Network plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, "rules", cfg.ValidatorPluginNetworkTemplate, *vc.NetworkPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.OCIPlugin.Enabled {
		log.InfoCLI("\n==== Applying OCI plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, "rules", cfg.ValidatorPluginOciTemplate, *vc.OCIPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.AzurePlugin.Enabled {
		log.InfoCLI("\n==== Applying Azure plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, "rules", cfg.ValidatorPluginAzureTemplate, *vc.AzurePlugin.Validator,
		); err != nil {
			return err
		}
	}

	return nil
}

func createValidator(kubeconfig, runLoc, name, template string, validator interface{}) error {
	spec, err := yaml.Marshal(validator)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to marshal %s validator", name))
	}
	args := map[string]interface{}{
		"Name":      name,
		"Namespace": cfg.Validator,
		"Spec":      indent(spec, 2),
	}
	path := filepath.Join(runLoc, "manifests", fmt.Sprintf("%s.yaml", name))
	if err := embed_utils.RenderTemplate(args, cfg.Validator, template, path); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to render %s validator manifest", name))
	}
	return applyValidatorManifest(kubeconfig, name, path)
}

func indent(bs []byte, indent int) string {
	b := bytes.Buffer{}
	for _, l := range bytes.Split(bs, []byte("\n")) {
		for i := 0; i < indent; i++ {
			b.Write([]byte(" "))
		}
		l = append(l, []byte("\n")...)
		b.Write(l)
	}
	return b.String()
}

func applyValidatorManifest(kubeconfig, name, path string) error {
	cmd := []string{"apply", "-f", path}
	if _, stderr, err := kube.KubectlCommand(cmd, kubeconfig); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to apply %s validator: %s", name, stderr))
	}
	return nil
}

func createKindCluster(c *cfg.Config, vc *components.ValidatorConfig) error {
	clusterConfig := filepath.Join(c.RunLoc, "kind-cluster-config.yaml")
	if err := kind.AdvancedConfig(vc.ProxyConfig.Env, clusterConfig); err != nil {
		return err
	}
	if err := kind.StartCluster(cfg.ValidatorKindClusterName, clusterConfig, vc.Kubeconfig); err != nil {
		return errors.Wrap(err, "failed to start validator kind cluster")
	}
	if err := os.Setenv("KUBECONFIG", vc.Kubeconfig); err != nil {
		return errors.Wrap(err, "failed to set KUBECONFIG env var")
	}
	log.InfoCLI("\nCreated kind cluster. kubeconfig: %s", vc.Kubeconfig)
	return nil
}
