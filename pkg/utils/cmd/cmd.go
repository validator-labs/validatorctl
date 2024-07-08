// Package cmd provides utility functions for cobra commands.
package cmd

import (
	"github.com/spf13/cobra"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// MarkFlagRequired marks a flag as required.
func MarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		log.FatalCLI(err.Error())
	}
}
