package components

import (
	network_api "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	vapi "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

// NetworkConfig represents the network plugin configuration.
type NetworkConfig struct {
	VcenterServer string
	IPRangeRules  []network_api.IPRangeRule
	TCPConnRules  []network_api.TCPConnRule
}

// ConfigureNetworkPlugin configures the network plugin.
func ConfigureNetworkPlugin(vc *ValidatorConfig, config NetworkConfig) {
	// TODO: prompt for chart version if !vc.UseFixedVersions
	vc.NetworkPlugin = &NetworkPluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:       cfg.ValidatorPluginNetwork,
				Repository: cfg.ValidatorPluginNetwork,
				Version:    cfg.ValidatorChartVersions[cfg.ValidatorPluginNetwork],
			},
		},
		Validator: &network_api.NetworkValidatorSpec{
			IPRangeRules: config.IPRangeRules,
			TCPConnRules: config.TCPConnRules,
		},
	}
}
