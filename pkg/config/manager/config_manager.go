package manager

import (
	"github.com/validator-labs/validatorctl/pkg/cmd/common"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

var c *cfg.Config

func init() {
	Reset()
}

func Init() error {
	if err := c.Load(); err != nil {
		return err
	}
	return common.InitWorkspace(c, "", cfg.BaseDirs, false)
}

func Config() *cfg.Config {
	return c
}

func Reset() {
	c = cfg.NewConfig()
}
