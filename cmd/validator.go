package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/validator-labs/validatorctl/pkg/cmd/common"
	"github.com/validator-labs/validatorctl/pkg/cmd/validator"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	cfgmanager "github.com/validator-labs/validatorctl/pkg/config/manager"
	cmdutils "github.com/validator-labs/validatorctl/pkg/utils/cmd"
)

func NewDeployValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var configFile string
	var configOnly, updatePasswords, reconfigure bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install validator & configure validator plugin(s)",
		Long: `Install validator & configure validator plugin(s).

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:         cobra.NoArgs,
		SilenceUsage: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return common.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskConfig := cfg.NewTaskConfig(
				Version, configFile, configOnly, false, updatePasswords, false, false,
			)
			if err := c.Save(""); err != nil {
				return err
			}

			if err := validator.DeployValidatorCommand(c, taskConfig, reconfigure); err != nil {
				return fmt.Errorf("failed to install validator: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&configFile, "config-file", "f", "", "Install using a configuration file (optional)")
	flags.BoolVarP(&configOnly, "config-only", "o", false, "Generate configuration file only. Do not proceed with installation. Default: false.")
	flags.BoolVarP(&updatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with installation. The --config-file flag must be provided. Default: false.")
	flags.BoolVarP(&reconfigure, "reconfigure", "r", false, "Re-configure validator and plugin(s) prior to installation. The --config-file flag must be provided. Default: false.")

	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")

	return cmd
}

func NewUpgradeValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var configFile string

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade validator & re-configure validator plugin(s)",
		Long: `Upgrade validator & re-configure validator plugin(s).

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:         cobra.NoArgs,
		SilenceUsage: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return common.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskConfig := cfg.NewTaskConfig(
				Version, configFile, false, false, false, false, false,
			)
			if err := validator.UpgradeValidatorCommand(c, taskConfig); err != nil {
				return fmt.Errorf("failed to upgrade validator: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&configFile, "config-file", "f", "", "Upgrade using a configuration file")

	cmdutils.MarkFlagRequired(cmd, "config-file")

	return cmd
}

func NewUndeployValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var configFile string
	var deleteCluster bool

	cmd := &cobra.Command{
		Use:          "uninstall",
		Short:        "Uninstall validator & all validator plugin(s)",
		Long:         "Uninstall validator & all validator plugin(s)",
		Args:         cobra.NoArgs,
		SilenceUsage: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return common.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskConfig := cfg.NewTaskConfig(
				Version, configFile, false, false, false, false, false,
			)
			if err := validator.UndeployValidatorCommand(taskConfig, deleteCluster); err != nil {
				return fmt.Errorf("failed to uninstall validator: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&configFile, "config-file", "f", "", "Validator configuration file (required)")
	flags.BoolVarP(&deleteCluster, "delete-cluster", "d", true, "Delete the validator kind cluster. Does not apply if using a preexisting K8s cluster. Default: true.")

	cmdutils.MarkFlagRequired(cmd, "config-file")

	return cmd
}

func NewDescribeValidationResultsCmd() *cobra.Command {
	c := cfgmanager.Config()
	var configFile string

	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe all validation results in a Kubernetes cluster",
		Long: `Describe all validation results in a Kubernetes cluster

Validation results in the cluster specified by the KUBECONFIG environment variable will be described.
If the --config-file flag is specified, the KUBECONFIG specified in the validator configuration file will be used instead.
`,
		Args:         cobra.NoArgs,
		SilenceUsage: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return common.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			taskConfig := cfg.NewTaskConfig(
				Version, configFile, false, false, false, false, false,
			)
			if err := validator.DescribeValidationResultsCommand(taskConfig); err != nil {
				return fmt.Errorf("failed to describe validation results: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&configFile, "config-file", "f", "", "Validator configuration file to read kubeconfig from (optional)")

	return cmd
}
