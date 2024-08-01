package components

import (
	"fmt"

	oci_api "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"
	vapi "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

// OciConfig represents the OCI plugin configuration.
type OciConfig struct {
	// HostRefs is a map of hostnames to a list of artifact references
	HostRefs map[string][]string
}

// ConfigureOciPlugin configures the OCI plugin.
func ConfigureOciPlugin(vc *ValidatorConfig, config OciConfig) {
	// TODO: prompt for chart version if !vc.UseFixedVersions
	vc.OCIPlugin = &OCIPluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:       cfg.ValidatorPluginOci,
				Repository: cfg.ValidatorPluginOci,
				Version:    cfg.ValidatorChartVersions[cfg.ValidatorPluginOci],
			},
		},
		Validator: &oci_api.OciValidatorSpec{
			OciRegistryRules: generateOciRegistryRules(config.HostRefs),
		},
	}
}

func generateOciRegistryRules(hostRefs map[string][]string) []oci_api.OciRegistryRule {
	rules := make([]oci_api.OciRegistryRule, 0, len(hostRefs))
	for host, refs := range hostRefs {
		rule := oci_api.OciRegistryRule{
			RuleName: fmt.Sprintf("artifacts on %s", host),
			Host:     host,
		}

		artifacts := []oci_api.Artifact{}
		for _, ref := range refs {
			artifacts = append(artifacts, oci_api.Artifact{
				Ref:             ref,
				LayerValidation: true,
			})
		}
		rule.Artifacts = artifacts

		rules = append(rules, rule)
	}
	return rules
}
