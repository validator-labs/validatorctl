package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/validator-labs/validatorctl/pkg/cmd/validator"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	cfgmanager "github.com/validator-labs/validatorctl/pkg/config/manager"
	cmdutils "github.com/validator-labs/validatorctl/pkg/utils/cmd"
	"github.com/validator-labs/validatorctl/pkg/utils/embed"
	"github.com/validator-labs/validatorctl/pkg/utils/exec"
)

// NewInstallValidatorCmd returns a new cobra command for installing validator & validator plugin(s)
// nolint:dupl
func NewInstallValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{CliVersion: Version}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install validator & validator plugin(s)",
		Long: `Install validator & validator plugin(s).

The validator CLI will install the validator and any configured plugins
to a Kubernetes cluster or your choosing.

Optionally provide the --apply and/or --wait flags to configure and apply
plugin rules and wait for validation, in addition to installation. This
is equivalent to first running 'validatorctl install', then running 
'validatorctl rules apply --config-file <config-file> --wait'.

Run 'validatorctl install --reconfigure --config-file <config-file>' to
reconfigure the validator and plugin(s) prior to installation.

Run 'validatorctl install --update-passwords --config-file <config-file>' to
update passwords in the validator configuration file. Optionally add
the --apply flag to update passwords for plugin(s) as well.

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := exec.CheckBinaries(exec.AllBins); err != nil {
				return err
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := c.Save(""); err != nil {
				return err
			}
			if err := validator.InstallValidatorCommand(c, tc); err != nil {
				return fmt.Errorf("failed to install validator: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Install using a configuration file (optional)")
	flags.BoolVarP(&tc.CreateConfigOnly, "config-only", "o", false, "Generate configuration file only. Do not proceed with installation. Default: false.")
	flags.BoolVarP(&tc.UpdatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with installation. The --config-file flag must be provided. Default: false.")
	flags.BoolVarP(&tc.Reconfigure, "reconfigure", "r", false, "Re-configure validator and plugin(s) prior to installation. The --config-file flag must be provided. Default: false.")

	flags.BoolVar(&tc.Apply, "apply", false, "Configure and apply validator plugin rules. Default: false")
	flags.BoolVar(&tc.Wait, "wait", false, "Wait for validation to succeed and describe results. Only applies when --apply is set. Default: false")

	cmd.MarkFlagsMutuallyExclusive("config-only", "wait")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "wait")

	return cmd
}

// NewValidatorRulesCmd returns a new cobra command which is a container for rule configuration subcommands
// nolint:dupl
func NewValidatorRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Configure & apply, or directly evaluate validator plugin rules.",
		Long: `Configure & apply, or directly evaluate validator plugin rules.

To configure and apply validator plugin rules, use 'validatorctl rules apply'.
Running 'validatorctl rules apply' requires a configuration file, which can be
generated using 'validatorctl install'.

To directly evaluate validator plugin rules, use 'validatorctl rules check'.
This does not require a configuration file, but one can be provided if desired.

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
	}

	cmd.AddCommand(NewApplyValidatorCmd())
	cmd.AddCommand(NewCheckValidatorCmd())

	return cmd
}

// NewApplyValidatorCmd returns a new cobra command for configuring and applying rules for validator plugins
func NewApplyValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{
		CliVersion: Version,
	}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Configure & apply validator plugin rules.",
		Long: `Configure & apply validator plugin rules.

Plugin-specific custom resources containing rule definitions will be
generated and applied to a Kubernetes cluster or your choosing. Useful
for continuous validation and alerting.

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := exec.CheckBinaries([]exec.Binary{exec.HelmBin, exec.KubectlBin}); err != nil {
				return err
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := c.Save(""); err != nil {
				return err
			}
			if err := validator.ConfigureOrCheckCommand(c, tc); err != nil {
				return fmt.Errorf("failed to configure and apply validator rules: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Validator configuration file (required).")
	flags.BoolVarP(&tc.CreateConfigOnly, "config-only", "o", false, "Update configuration file only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.UpdatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.Reconfigure, "reconfigure", "r", false, "Re-configure plugin rules prior to running checks. Default: false.")
	flags.BoolVar(&tc.Wait, "wait", false, "Wait for validation to succeed and describe results. Default: false")

	cmdutils.MarkFlagRequired(cmd, "config-file")

	cmd.MarkFlagsMutuallyExclusive("config-only", "wait")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "wait")

	return cmd
}

// NewCheckValidatorCmd returns a new cobra command for directly evaluating rules for validator plugins
func NewCheckValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{
		CliVersion: Version,
		Direct:     true,
	}

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Directly evaluate rules for validator plugin(s)",
		Long: `Directly evaluate rules for validator plugin(s).

Plugin rules will be evaluated directly, in-process. Useful for preflight checks or debugging.

Exit codes:
- 0 indicates that all rules passed validation.
- 1 indicates that an unexpected error occurred.
- 2 indicates that one or more rules failed validation.

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// enable 'validatorctl rules check --direct' without '-r'
			if tc.ConfigFile == "" {
				tc.Reconfigure = true
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := c.Save(""); err != nil {
				return err
			}
			if err := validator.ConfigureOrCheckCommand(c, tc); err != nil {
				if errors.Is(err, validator.ErrValidationFailed{}) {
					cmd.SilenceUsage = true
				}
				return fmt.Errorf("failed to check validator: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Validator configuration file.")
	flags.BoolVarP(&tc.CreateConfigOnly, "config-only", "o", false, "Update configuration file only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.UpdatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.Reconfigure, "reconfigure", "r", false, "Re-configure plugin rules prior to running checks. Default: false.")

	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")

	return cmd
}

// NewUpgradeValidatorCmd returns a new cobra command for upgrading the validator
func NewUpgradeValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{CliVersion: Version}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade validator & re-configure validator plugin(s)",
		Long: `Upgrade validator & re-configure validator plugin(s).

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := exec.CheckBinaries([]exec.Binary{exec.HelmBin, exec.KubectlBin}); err != nil {
				return err
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validator.UpgradeValidatorCommand(c, tc); err != nil {
				return fmt.Errorf("failed to upgrade validator: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Upgrade using a configuration file")

	cmdutils.MarkFlagRequired(cmd, "config-file")

	return cmd
}

// NewUndeployValidatorCmd returns a new cobra command for undeploying the validator
func NewUndeployValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{CliVersion: Version}

	cmd := &cobra.Command{
		Use:           "uninstall",
		Short:         "Uninstall validator & all validator plugin(s)",
		Long:          "Uninstall validator & all validator plugin(s)",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if err := exec.CheckBinaries([]exec.Binary{exec.HelmBin, exec.KindBin}); err != nil {
				return err
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validator.UndeployValidatorCommand(tc); err != nil {
				return fmt.Errorf("failed to uninstall validator: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Validator configuration file (required)")
	flags.BoolVarP(&tc.DeleteCluster, "delete-cluster", "d", true, "Delete the validator kind cluster. Does not apply if using a preexisting K8s cluster. Default: true.")

	cmdutils.MarkFlagRequired(cmd, "config-file")

	return cmd
}

// NewDescribeValidationResultsCmd returns a new cobra command for describing validation results
func NewDescribeValidationResultsCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{CliVersion: Version}

	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe all validation results in a Kubernetes cluster",
		Long: `Describe all validation results in a Kubernetes cluster

Validation results in the cluster specified by the KUBECONFIG environment variable will be described.
If the --config-file flag is specified, the KUBECONFIG specified in the validator configuration file will be used instead.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validator.DescribeValidationResultsCommand(tc); err != nil {
				return fmt.Errorf("failed to describe validation results: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Validator configuration file to read kubeconfig from (optional)")

	return cmd
}

// NewValidatorDocsCmd returns a new cobra command for displaying information about validator plugins
func NewValidatorDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Display information about supported validator plugins",
		Long: `Display information about supported validator plugins.

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		RunE: func(_ *cobra.Command, _ []string) error {
			args := map[string]interface{}{
				"Version":          Version,
				"ValidatorVersion": cfg.ValidatorChartVersions[cfg.Validator],
				"Plugins":          cfg.ValidatorChartVersions,
			}
			return embed.EFS.PrintTableTemplate(os.Stdout, args, cfg.Validator, "docs.tmpl")
		},
	}

	return cmd
}
