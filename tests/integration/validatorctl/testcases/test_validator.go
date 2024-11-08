package validatorctl

import (
	"bytes"
	"fmt"
	"os"
	"slices"

	log "github.com/sirupsen/logrus"

	maasclient "github.com/canonical/gomaasclient/client"
	"github.com/spectrocloud-labs/prompts-tui/prompts"
	tuimocks "github.com/spectrocloud-labs/prompts-tui/prompts/mocks"

	"github.com/validator-labs/validator-plugin-vsphere/api/vcenter"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
	"github.com/validator-labs/validatorctl/tests/integration/common"
	file_utils "github.com/validator-labs/validatorctl/tests/utils/file"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

var kindClusterName string

func Execute(ctx *test.TestContext) error {
	return test.Flow(ctx).
		Test(NewValidatorTest("validator-integration-test")).
		TearDown().Audit()
}

type ValidatorTest struct {
	*test.BaseTest
	log *log.Entry
}

func NewValidatorTest(description string) *ValidatorTest {
	return &ValidatorTest{
		log:      log.WithField("test", "validator-integration-test"),
		BaseTest: test.NewBaseTest("validator", description, nil),
	}
}

func (t *ValidatorTest) Execute(ctx *test.TestContext) (tr *test.TestResult) {
	t.log.Printf("Executing %s and %s", t.GetName(), t.GetDescription())
	if tr := t.PreRequisite(ctx); tr.IsFailed() {
		return tr
	}
	if result := t.testInstallInteractive(ctx); result.IsFailed() {
		return result
	}
	if result := t.testInstallInteractiveApply(ctx); result.IsFailed() {
		return result
	}
	if result := t.testInstallSilent(); result.IsFailed() {
		return result
	}
	if result := t.testInstallSilentWait(); result.IsFailed() {
		return result
	}
	if result := t.testRulesCheck(); result.IsFailed() {
		return result
	}
	if result := t.testRulesCheckCustomResource(); result.IsFailed() {
		return result
	}
	if result := t.testDescribe(); result.IsFailed() {
		return result
	}
	if result := t.testUndeploy(); result.IsFailed() {
		return result
	}
	if result := t.testInstallUpdatePasswords(); result.IsFailed() {
		return result
	}
	return test.Success()
}

func (t *ValidatorTest) initVsphereDriver(ctx *test.TestContext) {
	vsphereDriverMock := vsphere.MockVsphereDriver{
		Datacenters: []string{"DC0"},
		Clusters:    []string{"C0", "C1", "C2", "C3", "C4"},
		VMFolders:   []string{"spectro-templates", "test"},
		HostSystems: map[string][]vcenter.HostSystem{
			"DC0_C0": {
				{
					Name:      "DC0_C0_H0",
					Reference: "",
				},
				{
					Name:      "DC0_C0_H1",
					Reference: "",
				},
				{
					Name:      "DC0_C0_H2",
					Reference: "",
				},
			},
		},
	}
	vsphereDriverFunc := clouds.GetVSphereDriver
	ctx.Put("vsphereDriverFunc", vsphereDriverFunc)
	clouds.GetVSphereDriver = func(account vcenter.Account) (vsphere.Driver, error) {
		return vsphereDriverMock, nil
	}
}

func (t *ValidatorTest) testInstallInteractive(ctx *test.TestContext) (tr *test.TestResult) {
	t.log.Printf("Executing testInstallInteractive")

	interactiveCmd, buffer := common.InitCmd([]string{"install", "-o", "-l", "debug"})

	// Base values
	tuiVals := t.validatorValues(ctx, "Alertmanager")

	// Install values
	tuiVals = t.awsPluginInstallValues(ctx, tuiVals)
	tuiVals = t.azurePluginInstallValues(ctx, tuiVals)
	tuiVals = t.maasPluginInstallValues(ctx, tuiVals)
	tuiVals = t.networkPluginInstallValues(ctx, tuiVals)
	tuiVals = t.ociPluginInstallValues(ctx, tuiVals)
	tuiVals = t.vspherePluginInstallValues(ctx, tuiVals)

	// Finalization
	tuiVals = t.finalizationValues(tuiVals)

	prompts.Tui = &tuimocks.MockTUI{
		Values: tuiVals,
	}

	return common.ExecCLI(interactiveCmd, buffer, t.log, false)
}

func (t *ValidatorTest) testInstallInteractiveApply(ctx *test.TestContext) (tr *test.TestResult) {
	t.log.Printf("Executing testInstallInteractiveApply")

	interactiveCmd, buffer := common.InitCmd([]string{"install", "-o", "--apply", "-l", "debug"})

	// Base values
	tuiVals := t.validatorValues(ctx, "Slack")

	// Install values
	tuiVals = t.awsPluginInstallValues(ctx, tuiVals)
	tuiVals = t.azurePluginInstallValues(ctx, tuiVals)
	tuiVals = t.maasPluginInstallValues(ctx, tuiVals)
	tuiVals = t.networkPluginInstallValues(ctx, tuiVals)
	tuiVals = t.ociPluginInstallValues(ctx, tuiVals)
	tuiVals = t.vspherePluginInstallValues(ctx, tuiVals)
	tuiVals = t.finalizationValues(tuiVals)

	// Plugin values
	tuiSliceVals := make([][]string, 0)
	tuiVals, tuiSliceVals = t.awsPluginValues(tuiVals, tuiSliceVals)
	tuiVals, tuiSliceVals = t.azurePluginValues(tuiVals, tuiSliceVals)
	tuiVals, tuiSliceVals = t.maasPluginValues(tuiVals, tuiSliceVals)
	tuiVals, tuiSliceVals = t.networkPluginValues(tuiVals, tuiSliceVals)
	tuiVals, tuiSliceVals = t.ociPluginValues(tuiVals, tuiSliceVals)
	tuiVals, tuiSliceVals = t.vspherePluginValues(tuiVals, tuiSliceVals)
	tuiVals = t.finalizationValues(tuiVals)

	prompts.Tui = &tuimocks.MockTUI{
		Values:      tuiVals,
		SliceValues: tuiSliceVals,
	}

	return common.ExecCLI(interactiveCmd, buffer, t.log, false)
}

func (t *ValidatorTest) validatorValues(ctx *test.TestContext, sinkType string) []string {
	vals := []string{
		// Kind
		"y", // provision & use kind cluster

		// Proxy
		"n", // Configure an HTTP proxy

		// Air-gapped
		"n", // enable air-gapped mode

		// Private OCI registry
		"n", // enable private OCI registry

		// Image registry
		"quay.io/validator-labs", // validator image registry

		// Helm registry
		cfg.ValidatorHelmRegistry, // validator helm registry
		"y",                       // allow insecure connection
		"n",                       // configure basic auth

		// Sink
		"y", // Configure a sink
	}
	switch sinkType {
	case "Alertmanager":
		vals = append(vals, []string{
			"Alertmanager",                 // Sink type
			"sink-secret",                  // Sink secret name
			"https://alertmanager.io:9093", // Alertmanager endpoint
			"y",                            // Alertmanager insecureSkipVerify
			"foo",                          // Alertmanager username
			"bar",                          // Alertmanager password
		}...)
	case "Slack":
		vals = append(vals, []string{
			"Slack",             // Sink type
			"sink-secret",       // Sink secret name
			"xoxb-xxx",          // Slack bot token
			"slack-channel-xyz", // Slack channel id
		}...)
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		vals = append(vals, cfg.ValidatorChartVersions[cfg.Validator]) // validator helm chart version
	}
	return vals
}

func (t *ValidatorTest) finalizationValues(vals []string) []string {
	vals = append(vals, []string{
		"n", // restart configuration
		"n", // reconfigure plugin(s)
	}...)
	return vals
}

func (t *ValidatorTest) awsPluginInstallValues(ctx *test.TestContext, vals []string) []string {
	awsVals := []string{
		"y",               // enable AWS plugin
		"n",               // use implicit auth
		"aws-creds",       // AWS secret name
		"accesskey",       // AWS Access Key ID
		"secretaccesskey", // AWS Secret Access Key
		"",                // AWS Session Token
		"y",               // Configure STS
		"arn",             // AWS STS Role Arn
		"abc",             // AWS STS Session Name
		"3600",            // AWS STS Duration Seconds
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		awsVals = slices.Insert(awsVals, 1,
			cfg.ValidatorChartVersions[cfg.ValidatorPluginAws], // validator-plugin-aws helm chart version
		)
	}
	vals = append(vals, awsVals...)
	return vals
}

func (t *ValidatorTest) awsPluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	awsVals := []any{
		"us-west-2",                     // default region
		"y",                             // enable IAM role validation
		"SpectroCloudRole",              // IAM role name
		"Local Filepath",                // Policy Document Source
		t.filePath("awsIAMPolicy.json"), // Policy Document File
		"n",                             // add another policy document
		"n",                             // add another IAM role rule
		"y",                             // enable IAM user validation
		"SpectroCloudUser",              // IAM user name
		"Local Filepath",                // Policy Document Source
		t.filePath("awsIAMPolicy.json"), // Policy Document File
		"n",                             // add another policy document
		"n",                             // add another IAM user rule
		"y",                             // enable IAM group validation
		"SpectroCloudGroup",             // IAM group name
		"Local Filepath",                // Policy Document Source
		t.filePath("awsIAMPolicy.json"), // Policy Document File
		"n",                             // add another policy document
		"n",                             // add another IAM group rule
		"y",                             // enable IAM policy validation
		"arn:aws:iam::account-num:policy/some-policy", // IAM policy ARN
		"Local Filepath",                // Policy Document Source
		t.filePath("awsIAMPolicy.json"), // Policy Document File
		"n",                             // add another policy document
		"n",                             // add another IAM policy rule
		"y",                             // enable service quota validation
		"EC2",                           // rule name
		"EC2-VPC Elastic IPs",           // service quota type
		"us-west-2",                     // service quota region #1
		"5",                             // service quota buffer #1
		"n",                             // add another service quota rule
		"y",                             // enable subnet tag validation
		"subnet",                        // tag resource type
		"elb tag rule",                  // rule name
		"us-west-2",                     // subnet tag region #1
		"foo",                           // subnet tag key #1
		"bar",                           // subnet tag value #1
		[]string{"arn-1"},               // subnet arns
		"n",                             // add another subnet tag rule
		"n",                             // add another tag rule
		"y",                             // enable AMI validation
		"ami rule",                      // rule name
		"us-west-2",                     // ami region
		[]string{"ami-1", "ami-2"},      // AMI ids
		"y",                             // add an AMI filter
		"foo",                           // filter tag
		[]string{"bar", "baz"},          // filter values
		"n",                             // is this a tag filter
		"n",                             // add another filter
		[]string{""},                    // owners
		"n",                             // add another AMI rule
	}
	return interleave(vals, sliceVals, awsVals)
}

func (t *ValidatorTest) azurePluginInstallValues(ctx *test.TestContext, vals []string) []string {
	azureVals := []string{
		"y",                                    // enable plugin
		"AzureCloud",                           // cloud to connect to
		"n",                                    // implicit plugin auth
		"azure-creds",                          // k8s secret name
		"d551b7b1-78ae-43df-9d61-4935c843a454", // tenant id
		"d551b7b1-78ae-43df-9d61-4935c843a454", // client id
		"test_client_secret",                   // client secret
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		azureVals = slices.Insert(azureVals, 2,
			cfg.ValidatorChartVersions[cfg.ValidatorPluginAzure], // validator-plugin-azure helm chart version
		)
	}
	vals = append(vals, azureVals...)
	return vals
}

func (t *ValidatorTest) azurePluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	azureVals := []any{
		"y",                                    // enable RBAC validation
		"rule-1",                               // rule name
		"d551b7b1-78ae-43df-9d61-4935c843a454", // security principal
		"Local Filepath",                       // Add permission sets via
		t.filePath("azureRbacPermissionSets.json"), // Permission sets file
		"n", // add RBAC rule

		"y",                                    // enable community gallery image validation
		"rule-2",                               // rule name
		"westus",                               // location
		"testgallery",                          // gallery name
		[]string{"a"},                          // images
		"d551b7b1-78ae-43df-9d61-4935c843a454", // subscription
		"n",                                    // add community gallery image rule

		"y",              // enable quota validation
		"rule-3",         // rule name
		"Local Filepath", // Add resource sets via
		t.filePath("azureQuotaResourceSets.json"), // Resource sets file
		"n", // add quota rule
	}
	return interleave(vals, sliceVals, azureVals)
}

func (t *ValidatorTest) networkPluginInstallValues(ctx *test.TestContext, vals []string) []string {
	networkVals := []string{
		"y", // enable Network plugin
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		networkVals = slices.Insert(networkVals, 1,
			cfg.ValidatorChartVersions[cfg.ValidatorPluginNetwork], // validator-plugin-network helm chart version
		)
	}
	vals = append(vals, networkVals...)
	return vals
}

func (t *ValidatorTest) networkPluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	networkVals := []any{
		"y",                              // enable DNS validation
		"resolve foo",                    // DNS rule name
		"foo",                            // DNS host
		"",                               // DNS nameserver
		"n",                              // add another DNS rule
		"y",                              // enable ICMP validation
		"ping foo",                       // ICMP rule name
		"foo",                            // ICMP host
		"n",                              // add another ICMP rule
		"y",                              // enable IP range validation
		"check ips",                      // IP range rule name
		"10.10.10.10",                    // first IPv4 in range
		"1",                              // length of IPv4 range
		"n",                              // add another IP range rule
		"y",                              // enable MTU validation
		"check mtu",                      // MTU rule name
		"foo",                            // MTU host
		"1500",                           // minimum MTU
		"n",                              // add another MTU rule
		"y",                              // enable TCP connection validation
		"check tcp",                      // TCP connection rule name
		"foo",                            // TCP connection host
		[]string{"80"},                   // TCP connection ports
		"y",                              // InsecureSkipTLSVerify
		"5",                              // TCP connection timeout
		"n",                              // add another TCP connection rule
		"y",                              // enable HTTP file validation
		"check http file",                // HTTP file rule name
		[]string{"https://foo.com/file"}, // paths
		"y",                              // configure basic auth for http file rule
		"y",                              // create http file credential secret
		"username",                       // username key
		"password",                       // password key
		"y",                              // skip TLS verification
		"n",                              // add another HTTP file rule
		"n",                              // add local CA certs
		"y",                              // add CA cert secret refs
		"ca-certs",                       // secret name
		"ca.crt",                         // cert key
		"n",                              // add another CA cert secret ref
	}
	return interleave(vals, sliceVals, networkVals)
}

func (t *ValidatorTest) ociPluginInstallValues(ctx *test.TestContext, vals []string) []string {
	ociVals := []string{
		"y", // enable OCI plugin
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		ociVals = append(ociVals, cfg.ValidatorChartVersions[cfg.ValidatorPluginOci]) // validator-plugin-oci helm chart version
	}
	vals = append(vals, ociVals...)
	return vals
}

func (t *ValidatorTest) ociPluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	ociVals := []any{
		"private quay registry",               // OCI rule name
		"quay.io",                             // registry host
		"y",                                   // configure registry authentication
		"oci-creds",                           // secret name
		"y",                                   // configure basic auth
		"user1",                               // username
		"pa$$w0rd",                            // password
		"n",                                   // add env vars
		[]string{"quay.io/myartifact:latest"}, // artifact references
		"none",                                // validation type
		"y",                                   // add signature verification secret
		"cosign-pubkeys",                      // secret name
		t.filePath("cosign.pub"),              // public key file
		"n",                                   // add another public key to this secret
		"",                                    // ca certificate
		"n",                                   // add another registry rule
	}
	return interleave(vals, sliceVals, ociVals)
}

func (t *ValidatorTest) vspherePluginInstallValues(ctx *test.TestContext, vals []string) []string {
	vsphereVals := []string{
		"y",                // enable vsphere plugin
		"vsphere-creds",    // vSphere secret name
		"fake.vsphere.com", // vSphere domain
		"bob@vsphere.com",  // vSphere username
		"password",         // vSphere password
		"y",                // insecure skip verify
	}
	if string_utils.IsDevVersion(ctx.Get("version")) {
		vsphereVals = slices.Insert(vsphereVals, 1,
			cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere], // validator-plugin-vsphere helm chart version
		)
	}
	vals = append(vals, vsphereVals...)
	return vals
}

func (t *ValidatorTest) vspherePluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	vsphereVals := []any{
		"DC0",                               // datacenter
		"y",                                 // Enable NTP check
		"ntpd",                              // NTP rule name
		"y",                                 // are hosts cluster scoped
		"C0",                                // cluster name
		"DC0_C0_H0",                         // host1
		"y",                                 // add more hosts
		"DC0_C0_H1",                         // host2
		"n",                                 // add more hosts
		"n",                                 // add more validation rules
		"y",                                 // Enable privilege validation
		"entity rule 1",                     // privilege rule name
		"Folder",                            // entity type
		"spectro-templates",                 // folder name
		"Local Filepath",                    // vCenter privileges Source
		t.filePath("vCenterPrivileges.txt"), // privileges File
		"y",                                 // enable propagation
		[]string{""},                        // group principals
		"y",                                 // propagated
		"n",                                 // add more entity privilege rules
		"y",                                 // Enable compute resource validation
		"resource requirement rule 1",       // resource requirement rule name
		"Cluster",                           // select cluster for resource check
		"C0",                                // cluster name for resource check
		"master-pool",                       // node pool name
		"1",                                 // number of nodes
		"2GHz",                              // per node cpu
		"4Gi",                               // per node memory
		"10Gi",                              // per node storage
		"y",                                 // add another node pool
		"worker-pool",                       // node pool name
		"3",                                 // number of nodes
		"3GHz",                              // per node cpu
		"8Gi",                               // per node memory
		"20Gi",                              // per node storage
		"n",                                 // add more node pools
		"n",                                 // add more resource requirement checks
		"y",                                 // Enable tags validation
		"tag rule 1",                        // tag rule name
		"Datacenter",                        // entity type
		"DC0",                               // datacenter name
		"k8s-region",                        // tag
		"y",                                 // add another tag rule
		"tag rule 2",                        // tag rule name
		"Cluster",                           // entity type
		"C0",                                // cluster name
		"k8s-zone",                          // tag
		"n",                                 // add another tag rule
	}
	return interleave(vals, sliceVals, vsphereVals)
}

func (t *ValidatorTest) maasPluginInstallValues(ctx *test.TestContext, vals []string) []string {
	maasVals := []string{
		"y",                   // install MAAS plugin
		"maas-creds",          // MAAS credentials secret name
		"MAAS_API_KEY",        // MAAS API token key
		"fake:maasapi:token",  // MAAS API token
		"http://maas.io/MAAS", // MAAS Domain
	}

	if string_utils.IsDevVersion(ctx.Get("version")) {
		maasVals = slices.Insert(maasVals, 1,
			cfg.ValidatorChartVersions[cfg.ValidatorPluginMaas],
		)
	}

	vals = append(vals, maasVals...)
	return vals

}

func (t *ValidatorTest) maasPluginValues(vals []string, sliceVals [][]string) ([]string, [][]string) {
	maasVals := []any{
		"y",                 // Enable Resource Availibility validation
		"res-rule-1",        // Rule name
		"az1",               // Availability Zone
		"1",                 // Minimum number of machines
		"4",                 // Minimum CPU cores per machine
		"16",                // Minimum RAM in GB
		"256",               // Minimum Disk capacity in GB
		"pool1",             // Machine pool
		[]string{"tag1"},    // Tags
		"n",                 // Add another resource
		"n",                 // Add another resource rule
		"y",                 // Enable os image validation
		"os-rule-1",         // Rule name
		"ubuntu/jammy",      // image name
		"amd64/ga-22.04",    // image architecture
		"n",                 // Add another image
		"n",                 // Add another image rule
		"y",                 // Enable internal DNS validation
		"maas.io",           // MAAS Domain
		"subdomain.maas.io", // FQDN
		"10.10.10.10",       // IP
		"A",                 // Record type
		"10",                // ttl
		"n",                 // add another record
		"n",                 // add another resource
		"n",                 // add another internal DNS rule
		"y",                 // Enable upstream DNS validation
		"udns-rule-1",       // Rule name
		"1",                 // Expected number of servers
		"n",                 // Add another upstream dns rule
	}
	return interleave(vals, sliceVals, maasVals)
}

func interleave(vals []string, sliceVals [][]string, inputVals []any) ([]string, [][]string) {
	for _, val := range inputVals {
		switch v := val.(type) {
		case string:
			vals = append(vals, v)
		case []string:
			sliceVals = append(sliceVals, v)
		}
	}
	return vals, sliceVals
}

func (t *ValidatorTest) testInstallSilent() (tr *test.TestResult) {
	t.log.Printf("Executing testInstallSilent")

	kindClusterName = fmt.Sprintf("%s-%s", cfg.ValidatorKindClusterName, string_utils.RandStr(5))
	tokens := map[string]string{
		"<kind_cluster_name>": kindClusterName, // ensure concurrent tests use unique kind cluster names
	}
	if err := t.updateTestData(tokens); err != nil {
		return test.Failure(err.Error())
	}
	silentCmd, buffer := common.InitCmd([]string{
		"install", "-l", "debug", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log, false)
}

func (t *ValidatorTest) testInstallSilentWait() (tr *test.TestResult) {
	t.log.Printf("Executing testInstallSilentWait")

	tokens := map[string]string{
		"useKindCluster: true": "useKindCluster: false", // re-use the existing kind cluster
	}
	if err := t.updateTestData(tokens); err != nil {
		return test.Failure(err.Error())
	}
	silentCmd, buffer := common.InitCmd([]string{
		"install", "-l", "debug", "-f", t.filePath(cfg.ValidatorConfigFile),
		"--apply", "--wait",
	})
	return common.ExecCLI(silentCmd, buffer, t.log, false)
}

func (t *ValidatorTest) testRulesCheck() (tr *test.TestResult) {
	t.log.Printf("Executing testRulesCheck")

	tokens := map[string]string{
		`sinkConfig:
  enabled: true`: `sinkConfig:
  enabled: false`, // disable sink
	}
	if err := t.updateTestData(tokens); err != nil {
		return test.Failure(err.Error())
	}

	checkCmd, buffer := common.InitCmd([]string{
		"rules", "check", "-l", "debug", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(checkCmd, buffer, t.log, true)
}

func (t *ValidatorTest) testRulesCheckCustomResource() (tr *test.TestResult) {
	t.log.Printf("Executing testRulesCheckCustomResource")

	checkCmd, buffer := common.InitCmd([]string{
		"rules", "check", "-l", "debug", "--custom-resources", t.filePath("validator-crs"),
	})
	return common.ExecCLI(checkCmd, buffer, t.log, true)
}

func (t *ValidatorTest) testDescribe() (tr *test.TestResult) {
	t.log.Printf("Executing testDescribe")

	silentCmd, buffer := common.InitCmd([]string{
		"describe", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log, false)
}

func (t *ValidatorTest) testUndeploy() (tr *test.TestResult) {
	t.log.Printf("Executing testUndeploy")

	silentCmd, buffer := common.InitCmd([]string{
		"uninstall", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log, false)
}

func (t *ValidatorTest) testInstallUpdatePasswords() (tr *test.TestResult) {
	t.log.Printf("Executing testInstallUpdatePasswords")

	cmd, buffer := common.InitCmd([]string{
		"install", "-f", t.filePath(cfg.ValidatorConfigFile), "-p",
	})

	clouds.GetVSphereDriver = func(account vcenter.Account) (vsphere.Driver, error) {
		return vsphere.MockVsphereDriver{}, nil
	}

	prompts.Tui = &tuimocks.MockTUI{
		Values: []string{
			// Helm config
			cfg.ValidatorHelmRegistry, // Helm registry
			"y",                       // Allow Insecure Connection (Bypass x509 Verification)
			"y",                       // Use Helm basic auth
			"n",                       // Use existing secret
			"admin",                   // Helm username
			"welcome",                 // Helm password

			// AWS validator
			"n",         // Use implicit AWS auth
			"aws-creds", // AWS credentials secret name
			"abc",       // AWS Access Key
			"xyz",       // AWS Secret Key
			"",          // AWS Session Token
			"n",         // Use STS

			// Azure validator
			"n",                                    // Use implicit Azure auth
			"azure-creds",                          // Azure credentials secret name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // Azure Tenant ID
			"d551b7b1-78ae-43df-9d61-4935c843a454", // Azure Client ID
			"test_azure_client_secret",             // Azure Client Secret

			// MAAS validator
			"maas-creds",         // MAAS credentials secret name
			"MAAS_API_KEY",       // MAAS API token key
			"fake:maasapi:token", // MAAS API token

			// OCI validator
			"y",         // Add basic auth credentials
			"user2",     // Registry username
			"password2", // Registry password
			"y",         // Add an environment variable
			"FOO",       // Environment variable key
			"BAR",       // Environment variable value
			"n",         // Add another environment variable

			// vSphere validator
			"vsphere-creds",    // vSphere credentials secret name
			"vcenter.test.dev", // vSphere endpoint
			"bob@vsphere.com",  // vSphere username
			"123",              // vSphere password
			"y",                // vSphere insecureSkipVerify
		},
	}

	return common.ExecCLI(cmd, buffer, t.log, false)
}

func (t *ValidatorTest) PreRequisite(ctx *test.TestContext) (tr *test.TestResult) {
	t.log.Printf("Executing ExecuteRequisite for %s and %s", t.GetName(), t.GetDescription())
	if err := common.PreRequisiteFun()(ctx); err != nil {
		return test.Failure(err.Error())
	}

	t.initVsphereDriver(ctx)
	t.overrideMaasClient(ctx)

	return test.Success()
}

func (t *ValidatorTest) TearDown(ctx *test.TestContext) {
	t.log.Printf("Executing TearDown for %s and %s ", t.GetName(), t.GetDescription())

	if err := kind.DeleteCluster(kindClusterName); err != nil {
		t.log.Errorf("Failed to delete validator kind cluster: %v", err)
	}
	if err := common.TearDownFun()(ctx); err != nil {
		t.log.Errorf("Failed to run teardown fun: %v", err)
	}

	// restore clouds.GetVSphereDriver
	vsphereDriverFunc := ctx.Get("vsphereDriverFunc")
	clouds.GetVSphereDriver = vsphereDriverFunc.(func(account vcenter.Account) (vsphere.Driver, error))

	// restore clouds.GetMaasClient
	maasClientFunc := ctx.Get("maasClientFunc")
	clouds.GetMaasClient = maasClientFunc.(func(maasURL, maasToken string) (*maasclient.Client, error))
}

// updateTestData updates the hard-coded validator config used for silent installation tests
func (t *ValidatorTest) updateTestData(tokens map[string]string) error {
	testData := t.filePath(cfg.ValidatorConfigFile)
	bs, err := os.ReadFile(testData) //#nosec G304
	if err != nil {
		return err
	}
	for k, v := range tokens {
		bs = bytes.ReplaceAll(bs, []byte(k), []byte(v))
	}
	return os.WriteFile(testData, bs, 0600)
}

func (t *ValidatorTest) filePath(file string) string {
	return fmt.Sprintf("%s/%s/%s", file_utils.ValidatorTestCasesPath(), "data", file)
}

func (t *ValidatorTest) overrideMaasClient(ctx *test.TestContext) {
	maasClientFunc := clouds.GetMaasClient
	ctx.Put("maasClientFunc", maasClientFunc)
	clouds.GetMaasClient = clouds.GetMockMaasClient
}
