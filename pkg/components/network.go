package components

import (
	network_api "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
)

// NetworkConfig represents the network plugin configuration.
type NetworkConfig struct {
	VcenterServer string
	IPRangeRules  []network_api.IPRangeRule
	TCPConnRules  []network_api.TCPConnRule
}

// ConfigureNetworkPlugin configures the network plugin.
func ConfigureNetworkPlugin(vc *ValidatorConfig, config NetworkConfig) {
	vc.NetworkPlugin = &NetworkPluginConfig{
		Enabled: true,
		Validator: &network_api.NetworkValidatorSpec{
			IPRangeRules: config.IPRangeRules,
			TCPConnRules: config.TCPConnRules,
		},
	}
}
