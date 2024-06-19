package validator

import (
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/validator-labs/validatorctl/pkg/components"
)

func Test_readKubescapePlugin(t *testing.T) {
	type args struct {
		vc        *components.ValidatorConfig
		k8sClient kubernetes.Interface
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := readKubescapePlugin(tt.args.vc, tt.args.k8sClient); (err != nil) != tt.wantErr {
				t.Errorf("readKubescapePlugin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configureSeverityLimitRule(t *testing.T) {
	type args struct {
		c        *components.KubescapePluginConfig
		ruleName *[]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := configureSeverityLimitRule(tt.args.c, tt.args.ruleName); (err != nil) != tt.wantErr {
				t.Errorf("configureSeverityLimitRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configureFlagCVERule(t *testing.T) {
	type args struct {
		c        *components.KubescapePluginConfig
		ruleName *[]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := configureFlagCVERule(tt.args.c, tt.args.ruleName); (err != nil) != tt.wantErr {
				t.Errorf("configureFlagCVERule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configureIgnoreCVERule(t *testing.T) {
	type args struct {
		c        *components.KubescapePluginConfig
		ruleName *[]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := configureIgnoreCVERule(tt.args.c, tt.args.ruleName); (err != nil) != tt.wantErr {
				t.Errorf("configureIgnoreCVERule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
