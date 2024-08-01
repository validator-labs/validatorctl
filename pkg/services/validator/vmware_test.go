package validator

import (
	"testing"

	"github.com/mohae/deepcopy"
	"github.com/spectrocloud-labs/prompts-tui/prompts"
	tuimocks "github.com/spectrocloud-labs/prompts-tui/prompts/mocks"
	"k8s.io/client-go/kubernetes"

	vsphereapi "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
	"github.com/validator-labs/validator/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"
)

var vSphereDummyConfig = &components.ValidatorConfig{
	RegistryConfig: &components.RegistryConfig{
		Enabled: false,
	},
	VspherePlugin: &components.VspherePluginConfig{
		Release: &v1alpha1.HelmRelease{
			Chart: v1alpha1.HelmChart{},
		},
		ReleaseSecret: &components.Secret{},
		Account:       &vsphere.CloudAccount{},
		Validator:     &vsphereapi.VsphereValidatorSpec{},
	},
	Release: &v1alpha1.HelmRelease{
		Chart: v1alpha1.HelmChart{},
	},
	ReleaseSecret: &components.Secret{},
}

var (
	tui               prompts.TUI
	vSphereDriverFunc func(account *vsphere.CloudAccount) (vsphere.Driver, error)
)

func setup(returnVals []string) {
	tui = prompts.Tui
	prompts.Tui = &tuimocks.MockTUI{Values: returnVals}

	vSphereDriverFunc = clouds.GetVSphereDriver
	clouds.GetVSphereDriver = func(account *vsphere.CloudAccount) (vsphere.Driver, error) {
		return vsphere.MockVsphereDriver{}, nil
	}
}

func teardown() {
	prompts.Tui = tui
	clouds.GetVSphereDriver = vSphereDriverFunc
}

func Test_readVspherePlugin(t *testing.T) {
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
			vc:   deepcopy.Copy(vSphereDummyConfig).(*components.ValidatorConfig),
			returnVals: []string{
				"DC0", // datacenter
				"n",   // enable NTP validation
				"n",   // enable role privilege validation
				"n",   // enable entity privilege validation
				"n",   // enable resource requirement validation
				"n",   // enable tag validation
			},
			wantErr: true,
			err:     errNoRulesEnabled,
		},
	}
	for _, tt := range tts {
		setup(tt.returnVals)
		defer teardown()

		t.Run(tt.name, func(t *testing.T) {
			err := readVspherePluginRules(tt.vc, tt.k8sClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("readVspherePlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("readVspherePlugin() got error %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
