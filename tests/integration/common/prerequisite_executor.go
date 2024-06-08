package common

import (
	"os"

	"github.com/spf13/viper"

	cfgmanager "github.com/validator-labs/validatorctl/pkg/config/manager"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

type ExecuteRequisite func(testCtx *test.TestContext) error

func PreRequisiteFun() ExecuteRequisite {
	return func(testCtx *test.TestContext) error {
		if err := os.Setenv("NAMESPACE", ""); err != nil {
			return err
		}
		if err := os.Unsetenv("KUBECONFIG"); err != nil {
			return err
		}
		return nil
	}
}

func TearDownFun() ExecuteRequisite {
	return func(testCtx *test.TestContext) error {
		// Delete the CLI configuration between all tests
		cfgFile := viper.GetViper().ConfigFileUsed()
		if cfgFile != "" {
			if _, err := os.Stat(cfgFile); err == nil {
				if err := os.Remove(cfgFile); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Wipe viper's in memory config
		viper.Reset()
		cfgmanager.Reset()

		return nil
	}
}
