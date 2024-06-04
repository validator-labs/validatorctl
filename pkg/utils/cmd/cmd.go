package cmd

import (
	"github.com/spf13/cobra"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

func MarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		log.FatalCLI(err.Error())
	}
}
