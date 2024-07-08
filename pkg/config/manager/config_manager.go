// Package manager provides functions manage the validator config
package manager

import (
	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

var c *cfg.Config

func init() {
	Reset()
}

// Init loads the validator config
func Init() error {
	return c.Load()
}

// Config returns the validator config
func Config() *cfg.Config {
	return c
}

// Reset resets the validator config
func Reset() {
	c = cfg.NewConfig()
}
