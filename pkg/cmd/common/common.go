package common

import (
	"fmt"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

func InitWorkspace(c *cfg.Config, workspaceDir string, subdirs []string, timestamped bool) error {
	// Create workspace
	if err := c.CreateWorkspace(workspaceDir, subdirs, timestamped); err != nil {
		return fmt.Errorf("failed to initialize workspace: %v", err)
	}
	return nil
}
