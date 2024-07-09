// Package cmd provides the CLI commands for valdatorctl.
package cmd

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	cfgmanager "github.com/validator-labs/validatorctl/pkg/config/manager"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

var (
	cfgFile      string
	logLevel     string
	workspaceLoc string
	rootCmd      *cobra.Command

	// Version is the version validatorctl
	Version string
)

func init() {
	InitRootCmd()
	cobra.OnInitialize(InitConfig)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		exit(err)
	}
}

// InitRootCmd initializes the root command and adds all enabled subcommands
func InitRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "validator",
		Short: "Welcome to the Validator CLI",
		Long: `Welcome to the Validator CLI.
Install validator & configure validator plugins.
Use 'validator help <sub-command>' to explore all of the functionality the Validator CLI has to offer.`,
		SilenceUsage: false,
	}

	globalFlags := rootCmd.PersistentFlags()
	globalFlags.StringVarP(&cfgFile, "config", "c", "", "Validator CLI config file location")
	globalFlags.StringVarP(&logLevel, "log-level", "l", "info", "Log level. One of: [panic fatal error warn info debug trace]")
	globalFlags.StringVarP(&workspaceLoc, "workspace", "w", "", `Workspace location for staging runtime configurations and logs (default "$HOME/.validator")`)

	if err := viper.BindPFlag("logLevel", globalFlags.Lookup("log-level")); err != nil {
		exit(err)
	}

	// Verify required binaries exist
	if err := exec_utils.CheckBinaries(); err != nil {
		exit(err)
	}

	// add base commands
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewDeployValidatorCmd())
	rootCmd.AddCommand(NewUpgradeValidatorCmd())
	rootCmd.AddCommand(NewUndeployValidatorCmd())
	rootCmd.AddCommand(NewDescribeValidationResultsCmd())

	return rootCmd
}

// InitConfig reads in config file and ENV variables if set
func InitConfig() {
	log.SetLevel(viper.GetString("logLevel"))

	if cfgFile != "" {
		// Use config file from the --config flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		cfgPath, err := cfg.DefaultWorkspaceLoc()
		cobra.CheckErr(err)

		// Search for config under home directory
		viper.AddConfigPath(cfgPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName(cfg.ConfigFile)
	}
	viper.SetEnvPrefix("VALIDATOR_CTL")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it
	if err := viper.ReadInConfig(); err == nil {
		viper.OnConfigChange(func(e fsnotify.Event) {
			fmt.Println("Config file changed:", e.Name)
			// This is actually a noop - the updated config will be
			// written to disk separately, but still nice to notify
			// the user that something changed!
		})
		viper.WatchConfig()
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.InfoCLI("No validator cli config file detected. One will be created.")
		default:
			log.FatalCLI("Failed to initialize Validator CLI config", "error", err)
		}
	}

	// Instantiate config
	if err := cfgmanager.Init(); err != nil {
		exit(err)
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
