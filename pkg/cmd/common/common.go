package common

import (
	"fmt"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	embed_utils "github.com/validator-labs/validatorctl/pkg/utils/embed"
)

func InitWorkspace(c *cfg.Config, workspaceDir string, subdirs []string, timestamped bool) error {
	// Create workspace
	if err := c.CreateWorkspace(workspaceDir, subdirs, timestamped); err != nil {
		return fmt.Errorf("failed to initialize workspace: %v", err)
	}

	// Unpack binaries
	embed_utils.InitBinaries(c)

	// Initialize logger
	log.SetOutput(c.RunLoc)

	return nil
}
