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
	cfg "github.com/validator-labs/validatorctl/pkg/config"
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
		Account:       &vsphere.VsphereCloudAccount{},
		Validator:     &vsphereapi.VsphereValidatorSpec{},
	},
	Release: &v1alpha1.HelmRelease{
		Chart: v1alpha1.HelmChart{},
	},
	ReleaseSecret: &components.Secret{},
}

var (
	tui               prompts.TUI
	vSphereDriverFunc func(account *vsphere.VsphereCloudAccount) (vsphere.VsphereDriver, error)
)

func setup(returnVals []string) {
	tui = prompts.Tui
	prompts.Tui = &tuimocks.MockTUI{ReturnVals: returnVals}

	vSphereDriverFunc = clouds.GetVSphereDriver
	clouds.GetVSphereDriver = func(account *vsphere.VsphereCloudAccount) (vsphere.VsphereDriver, error) {
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
				cfg.ValidatorHelmRepository,                            // validator-plugin-vsphere helm chart repo
				cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere], // validator-plugin-vsphere helm chart version
				"y",                // Re-use validator chart security configuration
				"vsphere-creds",    // vSphere secret name
				"fake.vsphere.com", // vSphere domain
				"bob@vsphere.com",  // vSphere username
				"password",         // vSphere password
				"y",                // insecure skip verify
				"DC0",              // datacenter
				"n",                // enable NTP validation
				"n",                // enable role privilege validation
				"n",                // enable entity privilege validation
				"n",                // enable resource requirement validation
				"n",                // enable tag validation
			},
			wantErr: true,
			err:     errNoRulesEnabled,
		},
	}
	for _, tt := range tts {
		setup(tt.returnVals)
		defer teardown()

		t.Run(tt.name, func(t *testing.T) {
			err := readVspherePlugin(tt.vc, tt.k8sClient)
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
