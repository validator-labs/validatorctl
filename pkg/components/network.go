package components

import (
	"fmt"

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
	vc.NetworkPlugin = &NetworkPluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:                  cfg.ValidatorPluginNetwork,
				Repository:            fmt.Sprintf("%s/%s", cfg.ValidatorHelmRepository, cfg.ValidatorPluginNetwork),
				Version:               cfg.ValidatorChartVersions[cfg.ValidatorPluginNetwork],
				InsecureSkipTlsVerify: true,
			},
		},
		ReleaseSecret: &Secret{
			Name: fmt.Sprintf("validator-helm-release-%s", cfg.ValidatorPluginNetwork),
		},
		Validator: &network_api.NetworkValidatorSpec{
			IPRangeRules: config.IPRangeRules,
			TCPConnRules: config.TCPConnRules,
		},
	}
}
