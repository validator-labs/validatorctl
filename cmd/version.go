package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd returns the cobra command that outputs the Validator CLI version
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Prints the Validator CLI version",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Printf("Validator CLI version: %s\n", Version)
		},
	}
}
