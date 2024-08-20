// Package validator provides functions to manage the validator and its plugins
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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	awsapi "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	awsval "github.com/validator-labs/validator-plugin-aws/pkg/validate"
	azureapi "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"
	azureval "github.com/validator-labs/validator-plugin-azure/pkg/validate"
	maasapi "github.com/validator-labs/validator-plugin-maas/api/v1alpha1"
	maasval "github.com/validator-labs/validator-plugin-maas/pkg/validate"
	netapi "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	netval "github.com/validator-labs/validator-plugin-network/pkg/validate"
	ociapi "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"
	ociauth "github.com/validator-labs/validator-plugin-oci/pkg/auth"
	ocic "github.com/validator-labs/validator-plugin-oci/pkg/ociclient"
	ocival "github.com/validator-labs/validator-plugin-oci/pkg/validate"
	vsphereapi "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	vsphereval "github.com/validator-labs/validator-plugin-vsphere/pkg/validate"
	vapi "github.com/validator-labs/validator/api/v1alpha1"
	"github.com/validator-labs/validator/pkg/helm"
	"github.com/validator-labs/validator/pkg/sinks"
	"github.com/validator-labs/validator/pkg/types"
	vres "github.com/validator-labs/validator/pkg/validationresult"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services/validator"
	"github.com/validator-labs/validatorctl/pkg/utils/embed"
	"github.com/validator-labs/validatorctl/pkg/utils/exec"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	"github.com/validator-labs/validatorctl/pkg/utils/kube"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
)

// ErrValidationFailed is returned when one or more validation checks failed
type ErrValidationFailed struct{}

// Error returns the error message for ErrValidationFailed
func (e ErrValidationFailed) Error() string {
	return "one or more validation checks failed"
}

// InitWorkspace initializes a workspace directory with subdirectories
func InitWorkspace(c *cfg.Config, workspaceDir string, subdirs []string, timestamped bool) error {
	if err := c.CreateWorkspace(workspaceDir, subdirs, timestamped); err != nil {
		return fmt.Errorf("failed to initialize workspace: %v", err)
	}
	log.SetOutput(c.RunLoc)
	return nil
}

// InstallValidatorCommand deploys the validator and its plugins
func InstallValidatorCommand(c *cfg.Config, tc *cfg.TaskConfig) error {
	var vc *components.ValidatorConfig
	var err error
	var saveConfig bool

	if tc.ConfigFile == "" && tc.Reconfigure {
		log.FatalCLI("invalid arguments", "error", "cannot reconfigure validator without providing a configuration file")
	}

	configProvided := tc.ConfigFile != ""

	if configProvided && !tc.Reconfigure {
		// Silent Mode
		vc, err = components.NewValidatorFromConfig(tc)
		if err != nil {
			return errors.Wrap(err, "failed to load validator configuration file")
		}
		if vc.KindConfig.UseKindCluster {
			if err := exec.CheckBinaries([]exec.Binary{exec.DockerBin, exec.KindBin}); err != nil {
				return err
			}
		}
		if tc.UpdatePasswords {
			log.Header("Updating credentials in validator configuration file")
			if err := validator.UpdateValidatorCredentials(vc); err != nil {
				return err
			}
			if tc.Apply {
				if err := validator.UpdateValidatorPluginCredentials(vc, tc); err != nil {
					return err
				}
			}
			saveConfig = true
		}
		if vc.Kubeconfig == "" {
			vc.Kubeconfig = filepath.Join(c.RunLoc, "kind-cluster.kubeconfig")
			saveConfig = true
		}
	} else {
		// Interactive mode
		if tc.Reconfigure {
			vc, err = components.NewValidatorFromConfig(tc)
			if err != nil {
				return errors.Wrap(err, "failed to load validator configuration file")
			}
		} else {
			vc = components.NewValidatorConfig()
		}

		// for dev build versions, we allow selection of specific validator and plugin versions
		// for all other builds, we set a fixed version for the validator and plugins
		vc.UseFixedVersions = !string_utils.IsDevVersion(tc.CliVersion)

		if err := validator.ReadValidatorConfig(c, tc, vc); err != nil {
			return errors.Wrap(err, "failed to configure validator")
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

	if tc.CreateConfigOnly && tc.Apply {
		if !configProvided {
			tc.Reconfigure = true
		}
		if err := ConfigureOrCheckCommand(c, tc); err != nil {
			return err
		}
	}

	if tc.CreateConfigOnly || tc.UpdatePasswords {
		return nil
	}

	if vc.KindConfig.UseKindCluster {
		if err := createKindCluster(c, vc); err != nil {
			return err
		}
	}

	if tc.Apply {
		if !configProvided {
			tc.Reconfigure = true
		}
		if err := ConfigureOrCheckCommand(c, tc); err != nil {
			return err
		}
	}

	if err := deployValidatorAndPlugins(c, vc); err != nil {
		return err
	}

	log.InfoCLI(`
	Configure plugin rules and apply them to a cluster via the following command:

	validator rules apply --config-file %s --reconfigure
	`, tc.ConfigFile)
	return nil
}

// ConfigureOrCheckCommand configures and applies/executes validator plugin rules
// nolint:dupl
func ConfigureOrCheckCommand(c *cfg.Config, tc *cfg.TaskConfig) error {
	var vc *components.ValidatorConfig
	var err error
	var saveConfig bool

	if !tc.Reconfigure {
		// Silent Mode
		vc, err = components.NewValidatorFromConfig(tc)
		if err != nil {
			return errors.Wrap(err, "failed to load validator configuration file")
		}
		if tc.UpdatePasswords {
			log.Header("Updating plugin credentials in validator configuration file")
			if err := validator.UpdateValidatorPluginCredentials(vc, tc); err != nil {
				return err
			}
			saveConfig = true
		}
	} else {
		// Interactive mode
		if tc.Direct && tc.ConfigFile == "" {
			vc = components.NewValidatorConfig()
			tc.ConfigFile = filepath.Join(c.RunLoc, cfg.ValidatorConfigFile)
		} else {
			vc, err = components.NewValidatorFromConfig(tc)
			if err != nil {
				return errors.Wrap(err, "failed to load validator configuration file")
			}
		}
		if err := validator.ReadValidatorPluginConfig(c, tc, vc); err != nil {
			return errors.Wrap(err, "failed to configure validator plugin(s)")
		}
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

	ok, invalidPlugins := vc.EnabledPluginsHaveRules()
	if !ok {
		log.FatalCLI("invalid validator configuration", "error",
			fmt.Sprintf("the following plugins are enabled, but have no rules configured: %v", invalidPlugins),
		)
	}

	if tc.Direct {
		return executePlugins(c, vc)
	}

	// upgrade the validator helm release so that plugin rule secrets
	// are created, e.g., OCI registry secrets, Network basic auth secrets, etc.
	if err := applyValidator(c, vc); err != nil {
		return err
	}
	if err := configurePlugins(c, vc, tc); err != nil {
		return err
	}
	if tc.Wait {
		log.Header("Waiting for validation to complete")
		_, err := WatchValidationResults(tc)
		return err
	}
	return nil
}

// UpgradeValidatorCommand upgrades validator and its plugins
func UpgradeValidatorCommand(c *cfg.Config, tc *cfg.TaskConfig) error {
	vc, err := components.NewValidatorFromConfig(tc)
	if err != nil {
		return errors.Wrap(err, "failed to load validator configuration file")
	}
	if vc.Kubeconfig == "" {
		return errors.New("invalid validator configuration: kubeconfig is required")
	}
	return deployValidatorAndPlugins(c, vc)
}

// UndeployValidatorCommand undeploys validator and its plugins
func UndeployValidatorCommand(tc *cfg.TaskConfig) error {
	vc, err := components.NewValidatorFromConfig(tc)
	if err != nil {
		return errors.Wrap(err, "failed to load validator configuration file")
	}
	if vc.KindConfig.UseKindCluster && tc.DeleteCluster {
		if err := exec.CheckBinaries([]exec.Binary{exec.KindBin}); err != nil {
			return err
		}
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

	if vc.KindConfig.UseKindCluster && tc.DeleteCluster {
		return kind.DeleteCluster(cfg.ValidatorKindClusterName)
	}

	return nil
}

// DescribeValidationResultsCommand prints the validation results
func DescribeValidationResultsCommand(tc *cfg.TaskConfig) error {
	kClient, err := getValidationResultsCRDClient(tc)
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

// WatchValidationResults watches the validation results until all have either succeeded or failed
func WatchValidationResults(tc *cfg.TaskConfig) (bool, error) {
	log.InfoCLI("\nWatching validation results, waiting for all to succeed...")
	kClient, err := getValidationResultsCRDClient(tc)
	if err != nil {
		return false, errors.Wrap(err, "failed to get validation result client")
	}

	watchFunc := func(_ metav1.ListOptions) (watch.Interface, error) {
		return kClient.Watch(context.Background(), metav1.ListOptions{})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return false, errors.Wrap(err, "failed to create retry watcher for validation results")
	}

	var hasValidationSucceeded bool
	validationStates := make(map[string]vapi.ValidationState)

	if os.Getenv("IS_TEST") == "true" {
		return true, nil
	}

	for event := range watcher.ResultChan() {
		vrObj := event.Object.(*unstructured.Unstructured)

		vr := &vapi.ValidationResult{}
		bytes, err := vrObj.MarshalJSON()
		if err != nil {
			return false, err
		}
		if err := json.Unmarshal(bytes, vr); err != nil {
			return false, err
		}

		prevValidationState := validationStates[vr.Name]
		validationStates[vr.Name] = vr.Status.State
		if event.Type != watch.Modified {
			continue
		}

		hasValidationSucceeded = true
		if prevValidationState != vr.Status.State {
			log.InfoCLI("\nValidation result for %s updated:", vr.Name)
			err = printValidationResults([]unstructured.Unstructured{*vrObj})
			if err != nil {
				return false, err
			}

			finished := true
			vrWaiting := make([]string, 0)
			for vName, state := range validationStates {
				if state == vapi.ValidationFailed {
					hasValidationSucceeded = false
				}
				if state != vapi.ValidationSucceeded && state != vapi.ValidationFailed {
					vrWaiting = append(vrWaiting, vName)
					finished = false
					break
				}
			}
			if finished {
				break
			}

			log.InfoCLI("\nWatching for updates to validation results for %s...", vrWaiting)
		}
	}
	log.InfoCLI("\nAll validations have completed.")
	return hasValidationSucceeded, nil
}

func getValidationResultsCRDClient(tc *cfg.TaskConfig) (dynamic.NamespaceableResourceInterface, error) {
	if tc.ConfigFile != "" {
		vc, err := components.NewValidatorFromConfig(tc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load validator configuration file")
		}
		if err := os.Setenv("KUBECONFIG", vc.Kubeconfig); err != nil {
			return nil, err
		}
		log.Debug("Using kubeconfig from validator configuration file: %s", vc.Kubeconfig)
	}

	gv := kube.GetGroupVersion(vapi.GroupVersion.Group, vapi.GroupVersion.Version)
	kClient, err := kube.GetCRDClient(gv, vapi.ValidationResultGroupResource)
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
			vals = append(vals, c.Reason)
			break
		}
	}

	args := map[string]interface{}{
		"Keys":   keys,
		"Values": vals,
	}

	if err := embed.EFS.PrintTableTemplate(sb, args, cfg.Validator, "validation-result.tmpl"); err != nil {
		return "", err
	}

	sb.WriteString("\n------------\nRule Results\n------------\n")
	for _, c := range vr.Status.ValidationConditions {
		args := map[string]interface{}{
			"Keys":   []string{"Validation Rule", "Validation Type", "Status", "Last Validated", "Message"},
			"Values": []string{c.ValidationRule, c.ValidationType, string(c.Status), c.LastValidationTime.Format(time.RFC3339), strings.TrimSpace(c.Message)},
		}

		if err := embed.EFS.PrintTableTemplate(sb, args, cfg.Validator, "validation-result.tmpl"); err != nil {
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

// deployValidatorAndPlugins installs/upgrades validator + plugin(s)
func deployValidatorAndPlugins(c *cfg.Config, vc *components.ValidatorConfig) error {
	log.Header("Installing/Upgrading validator and validator plugin(s)")

	if err := applyValidator(c, vc); err != nil {
		return err
	}

	log.InfoCLI("\nvalidator and validator plugin(s) installed successfully")
	return nil
}

// configurePlugins applies/updates validator CRs for each plugin
func configurePlugins(c *cfg.Config, vc *components.ValidatorConfig, tc *cfg.TaskConfig) error {
	log.Header("Configuring validator plugin(s)")

	if err := applyPlugins(c, vc); err != nil {
		return err
	}

	log.Header("Validation in progress")

	log.InfoCLI("\nPlugins will now execute validation checks.")
	log.InfoCLI("\nYou can list validation results via the following command:")
	log.InfoCLI("\nkubectl -n validator get validationresults --kubeconfig %s", vc.Kubeconfig)

	log.InfoCLI("\nAnd you can view all validation result details via the following command:")
	log.InfoCLI("\nvalidator describe -f %s", tc.ConfigFile)
	return nil
}

func executePlugins(c *cfg.Config, vc *components.ValidatorConfig) error {
	log.Header("Executing validator plugin(s)")

	// Initialize a new logr.Logger that writes to the same
	// debug log file as the global logrus.Logger
	l := zap.New(zap.WriteTo(log.Out()))

	ok := true
	results := make([]*vapi.ValidationResult, 0)

	if vc.AWSPlugin.Enabled {
		v := &awsapi.AwsValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aws-validator",
				Namespace: "N/A",
			},
			Spec: *vc.AWSPlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := awsval.Validate(*vc.AWSPlugin.Validator, l)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		if vrOk := validationResponseOk(vrr, l); !vrOk {
			ok = false
		}
		results = append(results, vr)
	}

	if vc.AzurePlugin.Enabled {
		v := &azureapi.AzureValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "azure-validator",
				Namespace: "N/A",
			},
			Spec: *vc.AzurePlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := azureval.Validate(context.Background(), *vc.AzurePlugin.Validator, l)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		if vrOk := validationResponseOk(vrr, l); !vrOk {
			ok = false
		}
		results = append(results, vr)
	}

	if vc.MaasPlugin.Enabled {
		v := &maasapi.MaasValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "maas-validator",
				Namespace: "N/A",
			},
			Spec: *vc.MaasPlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := maasval.Validate(*vc.MaasPlugin.Validator, vc.MaasPlugin.Validator.Host, vc.MaasPlugin.MaasAPIToken, l)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		results = append(results, vr)
	}

	if vc.NetworkPlugin.Enabled {
		v := &netapi.NetworkValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "network-validator",
				Namespace: "N/A",
			},
			Spec: *vc.NetworkPlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := netval.Validate(*vc.NetworkPlugin.Validator,
			vc.NetworkPlugin.Validator.CACerts.RawCerts(),
			vc.NetworkPlugin.HTTPFileAuthBytes(), l,
		)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		if vrOk := validationResponseOk(vrr, l); !vrOk {
			ok = false
		}
		results = append(results, vr)
	}

	if vc.OCIPlugin.Enabled {
		v := &ociapi.OciValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci-validator",
				Namespace: "N/A",
			},
			Spec: *vc.OCIPlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := ocival.Validate(*vc.OCIPlugin.Validator,
			vc.OCIPlugin.BasicAuths(),
			vc.OCIPlugin.AllPubKeys(), l,
		)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		if vrOk := validationResponseOk(vrr, l); !vrOk {
			ok = false
		}
		results = append(results, vr)
	}

	if vc.VspherePlugin.Enabled {
		v := &vsphereapi.VsphereValidator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphere-validator",
				Namespace: "N/A",
			},
			Spec: *vc.VspherePlugin.Validator,
		}
		vr := vres.Build(v)
		vrr := vsphereval.Validate(context.Background(), *vc.VspherePlugin.Validator, vc.VspherePlugin.Account, l)
		if err := vres.Finalize(vr, vrr, l); err != nil {
			return err
		}
		if vrOk := validationResponseOk(vrr, l); !vrOk {
			ok = false
		}
		results = append(results, vr)
	}

	// Optionally emit results to a sink
	if vc.SinkConfig.Enabled {
		if err := emitToSink(vc, results, l); err != nil {
			return err
		}
	}

	// Convert results to unstructured objects
	us := make([]unstructured.Unstructured, 0, len(results))
	for _, vr := range results {
		u, err := kube.ToUnstructured(vr)
		if err != nil {
			return err
		}
		us = append(us, *u)

		// Write result to disk
		bs, err := yaml.Marshal(vr)
		if err != nil {
			return err
		}
		out := filepath.Join(c.RunLoc, fmt.Sprintf("%s-validation-result.yaml", vr.Name))
		if err := os.WriteFile(out, bs, 0600); err != nil {
			return err
		}
	}

	if err := printValidationResults(us); err != nil {
		return err
	}

	if !ok {
		return ErrValidationFailed{}
	}

	return nil
}

func validationResponseOk(vr types.ValidationResponse, log logr.Logger) bool {
	for _, vrr := range vr.ValidationRuleResults {
		if vrr.State != nil && *vrr.State == vapi.ValidationFailed {
			log.V(0).Info("validation rule failed",
				"rule", vrr.Condition.ValidationRule,
				"type", vrr.Condition.ValidationType,
				"message", vrr.Condition.Message,
			)
			return false
		}
	}
	return true
}

func emitToSink(vc *components.ValidatorConfig, results []*vapi.ValidationResult, log logr.Logger) error {
	sink := sinks.NewSink(types.SinkType(vc.SinkConfig.Type), log)

	sinkConfig := make(map[string][]byte, len(vc.SinkConfig.Values))
	for k, v := range vc.SinkConfig.Values {
		sinkConfig[k] = []byte(v)
	}

	c := sinks.NewClient(30 * time.Second)
	if err := sink.Configure(*c, sinkConfig); err != nil {
		return fmt.Errorf("failed to configure sink: %w", err)
	}
	for _, vr := range results {
		if err := sink.Emit(*vr); err != nil {
			return fmt.Errorf("failed to emit ValidationResult to sink: %w", err)
		}
	}

	return nil
}

func createReleaseSecretCmd(secret *components.Secret) []string {
	args := []string{
		"create", "secret", "generic", secret.Name, "-n", "validator",
		// include empty username/password, even if unset, to avoid error in validator
		fmt.Sprintf("--from-literal=username=%s", secret.BasicAuth.Username),
		fmt.Sprintf("--from-literal=password=%s", secret.BasicAuth.Password),
	}
	if secret.CaCertFile != "" {
		args = append(args, fmt.Sprintf("--from-file=caCert=%s", secret.CaCertFile))
	}
	return args
}

// nolint:gocyclo
func applyValidator(c *cfg.Config, vc *components.ValidatorConfig) error {
	pluginCount := 0
	kubecommandsPre := [][]string{}

	kClient, err := kube.GetKubeClientset(vc.Kubeconfig)
	if err != nil {
		return err
	}

	// build validator plugin spec
	validatorSpec := vapi.ValidatorConfigSpec{
		HelmConfig: *vc.HelmConfig,
		Plugins:    make([]vapi.HelmRelease, 0),
	}

	if vc.ReleaseSecret != nil && vc.ReleaseSecret.ShouldCreate() {
		kubecommandsPre = append(kubecommandsPre, createReleaseSecretCmd(vc.ReleaseSecret))
	}

	if vc.AWSPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.AWSPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-aws-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin aws values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.AWSPlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if vc.AzurePlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.AzurePlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-azure-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin azure values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.AzurePlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if vc.MaasPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.MaasPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-maas-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin maas values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.MaasPlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if vc.NetworkPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.NetworkPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-network-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin network values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.NetworkPlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if vc.OCIPlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.OCIPlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-oci-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin oci values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.OCIPlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if vc.VspherePlugin.Enabled {
		args := map[string]interface{}{
			"Config":        vc.VspherePlugin,
			"ImageRegistry": vc.ImageRegistry,
		}
		values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-plugin-vsphere-values.tmpl")
		if err != nil {
			return errors.Wrap(err, "failed to render validator plugin vsphere values.yaml")
		}
		validatorSpec.Plugins = append(validatorSpec.Plugins, vapi.HelmRelease{
			Chart:  vc.VspherePlugin.Release.Chart,
			Values: string(values),
		})
		pluginCount++
	}

	if !vc.AnyPluginEnabled() {
		log.FatalCLI("invalid validator config", "error", "at least one plugin must be enabled")
	}

	// concatenate base validator values w/ plugin values
	args := map[string]interface{}{
		"ImageRegistry": vc.ImageRegistry,
		"Tag":           vc.Release.Chart.Version,
		"ProxyConfig":   vc.ProxyConfig,
		"SinkConfig":    vc.SinkConfig,
		"AWSPlugin":     vc.AWSPlugin,
		"AzurePlugin":   vc.AzurePlugin,
		"MAASPlugin":    vc.MaasPlugin,
		"NetworkPlugin": vc.NetworkPlugin,
		"OCIPlugin":     vc.OCIPlugin,
		"VspherePlugin": vc.VspherePlugin,
	}
	if vc.ProxyConfig.Enabled {
		args["ProxyCaCertData"] = strings.Split(vc.ProxyConfig.Env.ProxyCACert.Data, "\n")
	}

	values, err := embed.EFS.RenderTemplateBytes(args, cfg.Validator, "validator-base-values.tmpl")
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
		Registry:              vc.HelmConfig.Registry,
		CaFile:                vc.HelmConfig.CAFile,
		InsecureSkipTLSVerify: vc.HelmConfig.InsecureSkipTLSVerify,
		Version:               vc.Release.Chart.Version,
		Values:                finalValues,
		CreateNamespace:       true,
	}
	if vc.ReleaseSecret != nil && vc.ReleaseSecret.BasicAuth != nil {
		opts.Username = vc.ReleaseSecret.BasicAuth.Username
		opts.Password = vc.ReleaseSecret.BasicAuth.Password
	}

	var cleanupLocalChart bool
	if strings.HasPrefix(opts.Registry, ocic.Scheme) {
		log.InfoCLI("\n==== Pulling validator Helm chart from OCI registry %s ====", opts.Registry)

		opts.Path = fmt.Sprintf("%s/%s", c.RunLoc, opts.Chart)
		opts.Version = strings.TrimPrefix(opts.Version, "v")

		ociClient, err := ocic.NewOCIClient(
			ocic.WithBasicAuth(opts.Username, opts.Password),
			ocic.WithMultiAuth(ociauth.GetKeychain(opts.Registry)),
			ocic.WithTLSConfig(opts.InsecureSkipTLSVerify, "", opts.CaFile),
		)
		if err != nil {
			return fmt.Errorf("failed to create OCI client: %w", err)
		}
		ociOpts := ocic.ImageOptions{
			Ref:     fmt.Sprintf("%s/%s:%s", strings.TrimPrefix(opts.Registry, ocic.Scheme), opts.Chart, opts.Version),
			OutDir:  opts.Path,
			OutFile: opts.Chart,
		}
		if err := ociClient.PullChart(ociOpts); err != nil {
			return fmt.Errorf("failed to pull Helm chart from OCI registry: %w", err)
		}

		opts.Path = fmt.Sprintf("%s/%s.tgz", opts.Path, opts.Chart)
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
	pluginsOk, err := watchValidatorConfig(pluginCount)
	if err != nil {
		return err
	}
	if !pluginsOk {
		return errors.New("one or more validator plugin(s) failed to install/upgrade")
	}

	return nil
}

// watchValidatorConfig watches the validator config until all plugins have been installed
func watchValidatorConfig(numPlugins int) (bool, error) {
	log.InfoCLI("\nWatching validator config, waiting for plugins to be installed or failed")

	gv := kube.GetGroupVersion(vapi.GroupVersion.Group, vapi.GroupVersion.Version)
	kClient, err := kube.GetCRDClient(gv, vapi.ValidatorConfigGroupResource)
	if err != nil {
		return false, errors.Wrap(err, "failed to get validator config client")
	}

	watchFunc := func(_ metav1.ListOptions) (watch.Interface, error) {
		return kClient.Watch(context.Background(), metav1.ListOptions{})
	}
	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return false, errors.Wrap(err, "failed to create retry watcher for validator config")
	}

	pluginsOk := true

	for event := range watcher.ResultChan() {
		vrObj := event.Object.(*unstructured.Unstructured)
		bytes, err := vrObj.MarshalJSON()
		if err != nil {
			return false, err
		}

		vc := &vapi.ValidatorConfig{}
		if err := json.Unmarshal(bytes, vc); err != nil {
			return false, err
		}

		if len(vc.Status.Conditions) == numPlugins {
			for _, c := range vc.Status.Conditions {
				if c.Status == v1.ConditionFalse {
					pluginsOk = false
					log.ErrorCLI("Plugin failed to install", c.PluginName, c.Message)
				}
			}
			break
		}

		log.InfoCLI("\nFound %d/%d plugin conditions in validator config status. Waiting...", len(vc.Status.Conditions), numPlugins)
	}

	log.InfoCLI("\nPlugin conditions found. All ok: %t.", pluginsOk)
	return pluginsOk, nil
}

// getHelmClient gets a helm client w/ a monkey-patched path to the embedded kind binary
func getHelmClient(vc *components.ValidatorConfig) (helm.Client, error) {
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
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginAws, cfg.ValidatorPluginAwsTemplate, *vc.AWSPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.AzurePlugin.Enabled {
		log.InfoCLI("\n==== Applying Azure plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginAzure, cfg.ValidatorPluginAzureTemplate, *vc.AzurePlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.MaasPlugin.Enabled {
		log.InfoCLI("\n==== Applying MAAS plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginMaas, cfg.ValidatorPluginMaasTemplate, *vc.MaasPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.NetworkPlugin.Enabled {
		log.InfoCLI("\n==== Applying Network plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginNetwork, cfg.ValidatorPluginNetworkTemplate, *vc.NetworkPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.OCIPlugin.Enabled {
		log.InfoCLI("\n==== Applying OCI plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginOci, cfg.ValidatorPluginOciTemplate, *vc.OCIPlugin.Validator,
		); err != nil {
			return err
		}
	}

	if vc.VspherePlugin.Enabled {
		log.InfoCLI("\n==== Applying vSphere plugin validator(s) ====")
		if err := createValidator(
			vc.Kubeconfig, c.RunLoc, cfg.ValidatorPluginVsphere, cfg.ValidatorPluginVsphereTemplate, *vc.VspherePlugin.Validator,
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
	if err := embed.EFS.RenderTemplate(args, cfg.Validator, template, path); err != nil {
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
	if err := kind.RenderKindConfig(vc, clusterConfig); err != nil {
		return err
	}
	kindClusterName := vc.KindConfig.KindClusterName
	if kindClusterName == "" {
		kindClusterName = cfg.ValidatorKindClusterName
	}
	if err := kind.StartCluster(kindClusterName, clusterConfig, vc.Kubeconfig); err != nil {
		return errors.Wrap(err, "failed to start validator kind cluster")
	}
	if err := os.Setenv("KUBECONFIG", vc.Kubeconfig); err != nil {
		return errors.Wrap(err, "failed to set KUBECONFIG env var")
	}
	log.InfoCLI("\nCreated kind cluster; kubeconfig: %s", vc.Kubeconfig)
	return nil
}
