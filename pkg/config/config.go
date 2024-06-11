package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"emperror.dev/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func NewConfig() *Config {
	return &Config{
		WorkspaceLoc: "",
	}
}

type Config struct {
	RunLoc       string `yaml:"runLoc"`
	WorkspaceLoc string `yaml:"workspaceLoc"`
}

type TaskConfig struct {
	CliVersion       string
	ConfigFile       string
	CreateConfigOnly bool
	Silent           bool
	UpdatePasswords  bool
	UpdateTokens     bool
}

func NewTaskConfig(cliVersion, configFile string, configOnly, silent, updatePasswords, updateTokens bool) *TaskConfig {
	return &TaskConfig{
		CliVersion:       cliVersion,
		ConfigFile:       configFile,
		CreateConfigOnly: configOnly,
		Silent:           silent,
		UpdatePasswords:  updatePasswords,
		UpdateTokens:     updateTokens,
	}
}

func DefaultWorkspaceLoc() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, WorkspaceLoc), nil
}

func (c *Config) initWorkspaceLoc() (err error) {
	if c.WorkspaceLoc == "" {
		c.WorkspaceLoc, err = DefaultWorkspaceLoc()
	}
	if viper.ConfigFileUsed() == "" {
		viper.SetConfigFile(filepath.Join(c.WorkspaceLoc, ConfigFile))
	}
	return
}

func (c *Config) CreateWorkspace(folder string, subdirs []string, timestamped bool) error {
	// Derive base dir
	if err := c.initWorkspaceLoc(); err != nil {
		return err
	}
	c.RunLoc = c.WorkspaceLoc

	if timestamped {
		t := time.Now()
		c.RunLoc = filepath.Join(c.WorkspaceLoc, fmt.Sprintf("%s-%s", folder, t.Format(TimeFormat)))
	}

	// Create subdirs
	for _, s := range subdirs {
		d := filepath.Join(c.RunLoc, s)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err = os.MkdirAll(d, 0700); err != nil {
				return err
			}
		}
	}

	return nil
}

// restoreGlobalDefaults resets ephemeral values in the global config before updating it on disk
func (c *Config) restoreGlobalDefaults() (err error) {
	c.WorkspaceLoc, err = DefaultWorkspaceLoc()
	return
}

func (c *Config) Decrypt() error {
	return nil
}

func (c *Config) Encrypt() error {
	return nil
}

func (c *Config) Save(path string) error {
	if err := c.restoreGlobalDefaults(); err != nil {
		return err
	}
	if err := c.Encrypt(); err != nil {
		return errors.Wrap(err, "failed to encrypt config")
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}
	if err := c.Decrypt(); err != nil {
		return errors.Wrap(err, "failed to decrypt config")
	}
	if path == "" {
		path = viper.ConfigFileUsed()
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		return errors.Wrap(err, "failed to create config file")
	}
	return nil
}

func (c *Config) Load() error {
	if err := viper.Unmarshal(&c); err != nil {
		return err
	}
	return c.Decrypt()
}

func (c *Config) Kubeconfig() string {
	return filepath.Join(c.RunLoc, "kubeconfig")
}

func (c *Config) ManifestDir() string {
	return filepath.Join(c.RunLoc, "manifests")
}
