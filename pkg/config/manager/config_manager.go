package manager

import (
	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

var c *cfg.Config

func init() {
	Reset()
}

func Init() error {
	return c.Load()
}

func Config() *cfg.Config {
	return c
}

func Reset() {
	c = cfg.NewConfig()
}
