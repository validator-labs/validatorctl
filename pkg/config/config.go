// Package config provides utility functions for managing the validator config.
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

// NewConfig creates a new Config object.
func NewConfig() *Config {
	return &Config{
		WorkspaceLoc: "",
	}
}

// Config represents the validator config.
type Config struct {
	RunLoc       string `yaml:"runLoc"`
	WorkspaceLoc string `yaml:"workspaceLoc"`
}

// TaskConfig represents the validator task config.
// CLI flags are bound to this struct.
type TaskConfig struct {
	CliVersion       string
	ConfigFile       string
	CustomResources  string
	Apply            bool
	CreateConfigOnly bool
	DeleteCluster    bool
	Direct           bool
	Reconfigure      bool
	UpdatePasswords  bool
	Wait             bool
}

// DefaultWorkspaceLoc returns the default workspace location.
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

// CreateWorkspace creates a new workspace with the specified folder and subdirs.
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

// Decrypt decrypts the config.
func (c *Config) Decrypt() error {
	return nil
}

// Encrypt encrypts the config.
func (c *Config) Encrypt() error {
	return nil
}

// Save saves the config to the specified path.
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

// Load loads the decrypted config file.
func (c *Config) Load() error {
	if err := viper.Unmarshal(&c); err != nil {
		return err
	}
	return c.Decrypt()
}
