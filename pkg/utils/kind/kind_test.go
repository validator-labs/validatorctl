package kind

import (
	"bytes"
	"os"
	"testing"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	"github.com/validator-labs/validatorctl/tests/utils/file"
)

func TestRenderKindConfig(t *testing.T) {
	tests := []struct {
		name     string
		vc       *components.ValidatorConfig
		expected string
	}{
		{
			name: "Kind config basic",
			vc: &components.ValidatorConfig{
				ProxyConfig: &components.ProxyConfig{
					Env: &components.Env{
						ProxyCACert:    &components.CACert{},
						PodCIDR:        &cfg.DefaultPodCIDR,
						ServiceIPRange: &cfg.DefaultServiceIPRange,
					},
				},
				RegistryConfig: &components.RegistryConfig{
					Enabled: false,
				},
			},
			expected: "kindconfig-basic.yaml",
		},
		{
			name: "Kind config w/ proxy CA cert",
			vc: &components.ValidatorConfig{
				ProxyConfig: &components.ProxyConfig{
					Env: &components.Env{
						PodCIDR:        &cfg.DefaultPodCIDR,
						ServiceIPRange: &cfg.DefaultServiceIPRange,
						ProxyCACert: &components.CACert{
							Name: "hosts",
							Path: "/etc/hosts",
						},
					},
				},
				RegistryConfig: &components.RegistryConfig{
					Enabled: false,
				},
			},
			expected: "kindconfig-shared-ca.yaml",
		},
		{
			name: "Kind config basic w/ custom registry",
			vc: &components.ValidatorConfig{
				ProxyConfig: &components.ProxyConfig{
					Env: &components.Env{
						ProxyCACert:    &components.CACert{},
						PodCIDR:        &cfg.DefaultPodCIDR,
						ServiceIPRange: &cfg.DefaultServiceIPRange,
					},
				},
				RegistryConfig: &components.RegistryConfig{
					Enabled: true,
					Registry: &components.Registry{
						Host: "registry.example.com",
						Port: 5000,
						BasicAuth: &components.BasicAuth{
							Username: "user",
							Password: "password",
						},
						InsecureSkipTLSVerify: true,
						ReuseProxyCACert:      false,
						BaseContentPath:       "base-path",
						IsAirgapped:           false,
					},
				},
			},
			expected: "kindconfig-custom-registry.yaml",
		},
		{
			name: "Kind config basic w/ airgapped registry",
			vc: &components.ValidatorConfig{
				ProxyConfig: &components.ProxyConfig{
					Env: &components.Env{
						ProxyCACert:    &components.CACert{},
						PodCIDR:        &cfg.DefaultPodCIDR,
						ServiceIPRange: &cfg.DefaultServiceIPRange,
					},
				},
				RegistryConfig: &components.RegistryConfig{
					Enabled: true,
					Registry: &components.Registry{
						Host: "registry.example.com",
						Port: 5000,
						BasicAuth: &components.BasicAuth{
							Username: "user",
							Password: "password",
						},
						InsecureSkipTLSVerify: true,
						ReuseProxyCACert:      false,
						IsAirgapped:           true,
					},
				},
			},
			expected: "kindconfig-airgapped.yaml",
		},
	}
	for _, tt := range tests {
		kindConfig := file.UnitTestFile("kindconfig.tmp")
		if err := RenderKindConfig(tt.vc, kindConfig); err != nil {
			t.Fatalf("Command Execution Failed. %v", err)
		}
		expectedBytes, err := os.ReadFile(file.UnitTestFile(tt.expected))
		if err != nil {
			t.Fatalf("failed to read expected file: %s: %v", tt.expected, err)
		}
		renderedBytes, err := os.ReadFile(kindConfig)
		if err != nil {
			t.Fatalf("failed to read rendered file: %s: %v", kindConfig, err)
		}

		if !bytes.Equal(renderedBytes, expectedBytes) {
			t.Fatalf("expected: %s, but got: %s", string(expectedBytes), string(renderedBytes))
		}
	}
}
