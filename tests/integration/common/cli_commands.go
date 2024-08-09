package common

import (
	"bytes"
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/validator-labs/validatorctl/cmd"
	"github.com/validator-labs/validatorctl/pkg/cmd/validator"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

// InitCmd initializes a new cobra command with the given arguments
// and returns the command and a buffer to capture the output.
func InitCmd(args []string) (*cobra.Command, *bytes.Buffer) {
	b := bytes.NewBufferString("")
	rootCmd := cmd.InitRootCmd()
	rootCmd.SetOut(b)
	rootCmd.SetArgs(args)
	return rootCmd, b
}

// ExecCLI executes the given cobra command and returns a test result.
func ExecCLI(cmd *cobra.Command, buffer *bytes.Buffer, log *log.Entry, expectValidationErr bool) (tr *test.TestResult) {
	if err := cmd.Execute(); err != nil {
		isValidationFailed := errors.Is(err, validator.ErrValidationFailed{})
		if !isValidationFailed || (isValidationFailed && !expectValidationErr) {
			return test.Failure(err.Error())
		}
	}
	out, err := io.ReadAll(buffer)
	if err != nil {
		return test.Failure(err.Error())
	}
	log.Print(string(out))
	return test.Success()
}
