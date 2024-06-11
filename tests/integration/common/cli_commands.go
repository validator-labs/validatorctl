package common

import (
	"bytes"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/validator-labs/validatorctl/cmd"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

func InitCmd(args []string) (*cobra.Command, *bytes.Buffer) {
	b := bytes.NewBufferString("")
	rootCmd := cmd.InitRootCmd()
	rootCmd.SetOut(b)
	rootCmd.SetArgs(args)
	return rootCmd, b
}

func ExecCLI(cmd *cobra.Command, buffer *bytes.Buffer, log *log.Entry) (tr *test.TestResult) {
	if err := cmd.Execute(); err != nil {
		return test.Failure(err.Error())
	}
	out, err := io.ReadAll(buffer)
	if err != nil {
		return test.Failure(err.Error())
	}
	log.Print(string(out))
	return test.Success()
}
