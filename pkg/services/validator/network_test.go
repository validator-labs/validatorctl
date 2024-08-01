package validator

import (
	"testing"

	"github.com/mohae/deepcopy"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/spectrocloud-labs/prompts-tui/prompts/mocks"
	network "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	"github.com/validator-labs/validator/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

var networkDummyConfig = &components.ValidatorConfig{
	RegistryConfig: &components.RegistryConfig{
		Enabled: false,
	},
	NetworkPlugin: &components.NetworkPluginConfig{
		Release: &v1alpha1.HelmRelease{
			Chart: v1alpha1.HelmChart{},
		},
		ReleaseSecret: &components.Secret{},
		Validator:     &network.NetworkValidatorSpec{},
	},
	Release: &v1alpha1.HelmRelease{
		Chart: v1alpha1.HelmChart{},
	},
	ReleaseSecret: &components.Secret{},
}

func Test_readNetworkPlugin(t *testing.T) {
	tts := []struct {
		name       string
		returnVals []string
		vc         *components.ValidatorConfig
		k8sClient  kubernetes.Interface
		wantErr    bool
		err        error
	}{
		{
			name: "Fail - no rules",
			vc:   deepcopy.Copy(networkDummyConfig).(*components.ValidatorConfig),
			returnVals: []string{
				cfg.ValidatorHelmRepository,                            // validator-plugin-network helm chart repo
				cfg.ValidatorChartVersions[cfg.ValidatorPluginNetwork], // validator-plugin-network helm chart version
				"y", // Re-use validator chart security configuration
				"n", // enable DNS validation
				"n", // enable ICMP validation
				"n", // enable IP range validation
				"n", // enable MTU validation
				"n", // enable TCP connection validation
				"n", // enable HTTPFile validation
			},
			wantErr: true,
			err:     errNoRulesEnabled,
		},
	}
	for _, tt := range tts {
		prompts.Tui = &mocks.MockTUI{Values: tt.returnVals}
		t.Run(tt.name, func(t *testing.T) {
			err := readNetworkPlugin(tt.vc, tt.k8sClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("readNetworkPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("readNetworkPlugin() got error %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
