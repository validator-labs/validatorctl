package components

import (
	"fmt"

	oci_api "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"
)

// OciConfig represents the OCI plugin configuration.
type OciConfig struct {
	// HostRefs is a map of hostnames to a list of artifact references
	HostRefs map[string][]string
}

// ConfigureOciPlugin configures the OCI plugin.
func ConfigureOciPlugin(vc *ValidatorConfig, config OciConfig) {
	vc.OCIPlugin = &OCIPluginConfig{
		Enabled: true,
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
