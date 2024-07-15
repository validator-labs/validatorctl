package kind

import (
	"bytes"
	"os"
	"testing"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	env "github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/tests/utils/file"
)

func TestRenderKindConfig(t *testing.T) {
	tests := []struct {
		name     string
		env      *env.Env
		hauler   *env.Hauler
		expected string
	}{
		{
			name: "Kind config w/ proxy CA cert",
			env: &env.Env{
				PodCIDR:        &cfg.DefaultPodCIDR,
				ServiceIPRange: &cfg.DefaultServiceIPRange,
				ProxyCACert: &env.CACert{
					Name: "hosts",
					Path: "/etc/hosts",
				},
			},
			expected: "kindconfig-shared-ca.yaml",
		},
		{
			name: "Kind config basic",
			env: &env.Env{
				ProxyCACert:    &env.CACert{},
				PodCIDR:        &cfg.DefaultPodCIDR,
				ServiceIPRange: &cfg.DefaultServiceIPRange,
			},
			expected: "kindconfig-basic.yaml",
		},
	}
	for _, tt := range tests {
		kindConfig := file.UnitTestFile("kindconfig.tmp")
		if err := RenderKindConfig(tt.env, tt.hauler, kindConfig); err != nil {
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
