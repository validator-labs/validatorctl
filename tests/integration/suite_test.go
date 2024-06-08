package integration

import (
	"fmt"
	"log"
	"os"
	"testing"

	/*
		"github.com/spectrocloud/palette-cli/tests/external/http"
		"github.com/spectrocloud/palette-cli/tests/integration/helper"
	*/
	"github.com/validator-labs/validatorctl/cmd"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	validator "github.com/validator-labs/validatorctl/tests/integration/_validator"
	"github.com/validator-labs/validatorctl/tests/integration/common"
	"github.com/validator-labs/validatorctl/tests/utils/test"
	"github.com/validator-labs/validatorctl/tests/utils/test/manager"
	//file_utils "github.com/validator-labs/validatorctl/tests/utils/file"
)

var (
	tm *manager.Manager

	/*
		hubbleServer = http.NewTestServer("HUBBLE", file_utils.HubbleRepoPath(), common.HubblePort)
		scarServer   = http.NewTestServer("SCAR", file_utils.ScarRepoPath(), common.ScarPort)
		maasServer   = http.NewTestServer("MAAS", file_utils.MaasRepoPath(), common.MaasPort)
	*/
)

func TestIntegrationSuite(t *testing.T) {
	if err := setup(); err != nil {
		t.Errorf("failed to setup integration test suite: %v", err)
	}
	runSuite(t)
	if err := teardown(); err != nil {
		t.Errorf("failed to teardown integration test suite: %v", err)
	}
}

/*
func startMockServers() {
	hubbleServer.Start()
	maasServer.Start()
	scarServer.Start()
}

func stopMockServers() {
	hubbleServer.Shutdown()
	maasServer.Shutdown()
	scarServer.Shutdown()
}
*/

func runSuite(t *testing.T) {
	fmt.Println("Palette CLI Integration Test Suite")

	//startMockServers()

	testCtx := test.NewTestContext()
	err := test.Flow(testCtx).
		Test(common.NewSingleFuncTest("validator-test", validator.Execute)).
		Summarize().TearDown().Audit()

	//stopMockServers()

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

	// Initialize subcommands, config, workspace, binaries, logger
	cmd.Subcommands = "ALL"
	cmd.InitConfig()

	// Initialize envtest cluster
	crds := manager.KubeVirtCRDs
	tm = manager.NewTestManager()
	return tm.InitCluster(crds)
}

func teardown() error {
	if tm != nil { // tm may be nil if setup() failed to initialize it
		return tm.DestroyEnvironment()
	}
	return nil
}
