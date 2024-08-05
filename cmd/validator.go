package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/validator-labs/validatorctl/pkg/cmd/validator"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	cfgmanager "github.com/validator-labs/validatorctl/pkg/config/manager"
	log "github.com/validator-labs/validatorctl/pkg/logging"
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
				return fmt.Errorf("failed to install validator: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Install using a configuration file (optional)")
	flags.BoolVarP(&tc.CreateConfigOnly, "config-only", "o", false, "Generate configuration file only. Do not proceed with installation. Default: false.")
	flags.BoolVarP(&tc.UpdatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with installation. The --config-file flag must be provided. Default: false.")
	flags.BoolVarP(&tc.Reconfigure, "reconfigure", "r", false, "Re-configure validator and plugin(s) prior to installation. The --config-file flag must be provided. Default: false.")

	flags.BoolVar(&tc.Check, "check", false, "Configure rules for validator plugin(s). Default: false")
	flags.BoolVar(&tc.Wait, "wait", false, "Wait for validation to succeed and describe results. Only applies when --check is set. Default: false")

	cmd.MarkFlagsMutuallyExclusive("config-only", "wait")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "wait")

	return cmd
}

// NewConfigureValidatorCmd returns a new cobra command for configuring and applying rules for validator plugins
// nolint:dupl
func NewConfigureValidatorCmd() *cobra.Command {
	c := cfgmanager.Config()
	var tc = &cfg.TaskConfig{
		CliVersion: Version,
	}

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Configure & apply rules for validator plugin(s)",
		Long: `Configure & apply rules for validator plugin(s).

For more information about validator, see: https://github.com/validator-labs/validator.
`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if tc.ConfigFile == "" && !tc.Direct {
				return fmt.Errorf(`required flag "config-file" not set`)
			}
			if !tc.Direct {
				if err := exec.CheckBinaries([]exec.Binary{exec.HelmBin, exec.KubectlBin}); err != nil {
					return err
				}
			} else {
				// enables 'validatorctl check --direct' without '-r'
				if tc.ConfigFile == "" {
					tc.Reconfigure = true
				}
			}
			return validator.InitWorkspace(c, cfg.Validator, cfg.ValidatorSubdirs, true)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := c.Save(""); err != nil {
				return err
			}
			if err := validator.ConfigureValidatorCommand(c, tc); err != nil {
				return fmt.Errorf("failed to configure validator: %v", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&tc.ConfigFile, "config-file", "f", "", "Validator configuration file. Required unless using --direct.")
	flags.BoolVarP(&tc.CreateConfigOnly, "config-only", "o", false, "Update configuration file only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.Direct, "direct", "d", false, "Execute checks directly; no validator installation required. Default: false")
	flags.BoolVarP(&tc.UpdatePasswords, "update-passwords", "p", false, "Update passwords only. Do not proceed with checks. Default: false.")
	flags.BoolVarP(&tc.Reconfigure, "reconfigure", "r", false, "Re-configure plugin rules prior to running checks. Default: false.")
	flags.BoolVar(&tc.Wait, "wait", false, "Wait for validation to succeed and describe results. Default: false")

	cmd.MarkFlagsMutuallyExclusive("config-only", "wait")
	cmd.MarkFlagsMutuallyExclusive("direct", "config-only")
	cmd.MarkFlagsMutuallyExclusive("direct", "update-passwords")
	cmd.MarkFlagsMutuallyExclusive("direct", "wait")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "reconfigure")
	cmd.MarkFlagsMutuallyExclusive("update-passwords", "wait")

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
				return fmt.Errorf("failed to upgrade validator: %v", err)
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
				return fmt.Errorf("failed to uninstall validator: %v", err)
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
				return fmt.Errorf("failed to describe validation results: %v", err)
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
			return embed.EFS.PrintTableTemplate(log.Out(), args, cfg.Validator, "docs.tmpl")
		},
	}

	return cmd
}
