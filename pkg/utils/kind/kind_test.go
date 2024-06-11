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
	subtests := []struct {
		name     string
		env      *env.Env
		expected string
	}{
		{
			name: "Kind config w/ proxy CA cert",
			env: &env.Env{
				PodCIDR:         &cfg.DefaultPodCIDR,
				ServiceIPRange:  &cfg.DefaultServiceIPRange,
				ProxyCaCertName: "hosts",
				ProxyCaCertPath: "/etc/hosts",
			},
			expected: "kindconfig-shared-ca.yaml",
		},
		{
			name: "Kind config basic",
			env: &env.Env{
				PodCIDR:        &cfg.DefaultPodCIDR,
				ServiceIPRange: &cfg.DefaultServiceIPRange,
			},
			expected: "kindconfig-basic.yaml",
		},
	}
	for _, subtest := range subtests {
		kindConfig := file.UnitTestFile("kindconfig.tmp")
		if err := AdvancedConfig(subtest.env, kindConfig); err != nil {
			t.Fatalf("Command Execution Failed. %v", err)
		}
		expectedBytes, err := os.ReadFile(file.UnitTestFile(subtest.expected))
		if err != nil {
			t.Fatalf("failed to read expected file: %s: %v", subtest.expected, err)
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
