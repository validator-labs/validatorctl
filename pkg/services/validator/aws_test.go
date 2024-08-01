package validator

import (
	"testing"

	"github.com/mohae/deepcopy"
	aws "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	"github.com/validator-labs/validator/api/v1alpha1"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/spectrocloud-labs/prompts-tui/prompts/mocks"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

var awsDummyConfig = &components.ValidatorConfig{
	RegistryConfig: &components.RegistryConfig{
		Enabled: false,
	},
	AWSPlugin: &components.AWSPluginConfig{
		Release: &v1alpha1.HelmRelease{
			Chart: v1alpha1.HelmChart{},
		},
		ReleaseSecret: &components.Secret{},
		Validator: &aws.AwsValidatorSpec{
			Auth: aws.AwsAuth{},
		},
	},
	Release: &v1alpha1.HelmRelease{
		Chart: v1alpha1.HelmChart{},
	},
	ReleaseSecret: &components.Secret{},
}

func Test_readAwsPlugin(t *testing.T) {
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
			vc:   deepcopy.Copy(awsDummyConfig).(*components.ValidatorConfig),
			returnVals: []string{
				cfg.ValidatorHelmRepository,                        // validator-plugin-aws helm chart repo
				cfg.ValidatorChartVersions[cfg.ValidatorPluginAws], // validator-plugin-aws helm chart version
				"y",         // Re-use validator chart security configuration
				"y",         // use implicit auth
				"",          // service account name
				"us-east-1", // region
				"n",         // enable IAM role validation
				"n",         // enable IAM user validation
				"n",         // enable IAM group validation
				"n",         // enable IAM policy validation
				"n",         // enable service quota validation
				"n",         // enable tag validation
				"n",         // enable AMI validation
			},
			wantErr: true,
			err:     errNoRulesEnabled,
		},
	}
	for _, tt := range tts {
		prompts.Tui = &mocks.MockTUI{Values: tt.returnVals}
		t.Run(tt.name, func(t *testing.T) {
			err := readAwsPlugin(tt.vc, tt.k8sClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("readAwsPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("readAwsPlugin() got error %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
