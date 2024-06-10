package integration

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/validator-labs/validatorctl/cmd"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	validator "github.com/validator-labs/validatorctl/tests/integration/_validator"
	"github.com/validator-labs/validatorctl/tests/integration/common"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

func TestIntegrationSuite(t *testing.T) {
	if err := setup(); err != nil {
		t.Errorf("failed to setup integration test suite: %v", err)
	}
	runSuite(t)
}

func runSuite(t *testing.T) {
	fmt.Println("Validator CLI Integration Test Suite")

	testCtx := test.NewTestContext()
	err := test.Flow(testCtx).
		Test(common.NewSingleFuncTest("validator-test", validator.Execute)).
		Summarize().TearDown().Audit()

	if err != nil {
		t.Error(err)
	}
}

func setup() error {
	// Set CLI version
	version := os.Getenv("CLI_VERSION")
	if version == "" && cmd.Version == "" {
		log.Fatal("CLI_VERSION environment variable or ldflags must be set")
	}
	cmd.Version = version

	// Wipe out the default config & workspace location
	defaultWorkspace, err := cfg.DefaultWorkspaceLoc()
	if err != nil {
		log.Fatal(err.Error())
	}
	if err := os.RemoveAll(defaultWorkspace); err != nil {
		log.Fatal(err.Error())
	}

	// Initialze config, workspace, binaries, logger
	cmd.InitConfig()
	return nil
}
