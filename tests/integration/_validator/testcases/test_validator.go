package validator

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	tuimocks "github.com/spectrocloud-labs/prompts-tui/prompts/mocks"

	vsphere_cloud "github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"
	"github.com/validator-labs/validatorctl/pkg/utils/kind"
	"github.com/validator-labs/validatorctl/tests/integration/common"
	file_utils "github.com/validator-labs/validatorctl/tests/utils/file"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

func Execute() error {
	testCtx := test.NewTestContext()
	return test.Flow(testCtx).
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
	if result := t.testDeployInteractive(ctx); result.IsFailed() {
		return result
	}
	/*
		if result := t.testDeploySilent(); result.IsFailed() {
			return result
		}
		if result := t.testDescribe(); result.IsFailed() {
			return result
		}
		if result := t.testUndeploy(); result.IsFailed() {
			return result
		}
		if result := t.testUpdatePasswords(); result.IsFailed() {
			return result
		}
	*/
	return test.Success()
}

func (t *ValidatorTest) testDeployInteractive(ctx *test.TestContext) (tr *test.TestResult) {

	interactiveCmd, buffer := common.InitCmd([]string{"install", "-o", "-l", "debug"})

	vsphereDriverMock := vsphere_cloud.MockVsphereDriver{
		Datacenters: []string{"DC0"},
		Clusters:    []string{"C0", "C1", "C2", "C3", "C4"},
		VMFolders:   []string{"spectro-templates", "test"},
		HostSystems: map[string][]vsphere_cloud.VSphereHostSystem{
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
	clouds.GetVSphereDriver = func(account *vsphere_cloud.VsphereCloudAccount) (vsphere_cloud.VsphereDriver, error) {
		return vsphereDriverMock, nil
	}

	prompts.Tui = &tuimocks.MockTUI{
		ReturnVals: []string{
			// Kind
			"y", // provision & use kind cluster

			// Image registry
			"quay.io/validator-labs", // validator image registry

			// Proxy
			"n", // Configure an HTTP proxy

			// Sink
			"y",                            // Configure a sink
			"Alertmanager",                 // Sink type
			"sink-secret",                  // Sink secret name
			"https://alertmanager.io:9093", // Alertmanager endpoint
			"y",                            // Alertmanager insecureSkipVerify
			"foo",                          // Alertmanager username
			"bar",                          // Alertmanager password

			// Helm repo
			cfg.ValidatorHelmRepository,               // validator helm chart repo
			cfg.ValidatorChartVersions[cfg.Validator], // validator helm chart version
			"y",   // insecure skip verify
			"y",   // use basic auth
			"bob", // release secret username
			"dog", // release secret password

			// AWS plugin
			"y",                         // enable AWS plugin
			cfg.ValidatorHelmRepository, // validator-plugin-aws helm chart repo
			cfg.ValidatorChartVersions[cfg.ValidatorPluginAws], // validator-plugin-aws helm chart version
			"y",                   // Re-use validator chart security configuration
			"n",                   // use implicit auth
			"aws-creds",           // AWS secret name
			"secretkey",           // AWS Secret Key ID
			"secretaccesskey",     // AWS Secret Access Key
			"",                    // AWS Session Token
			"y",                   // Configure STS
			"arn",                 // AWS STS Role Arn
			"abc",                 // AWS STS Session Name
			"3600",                // AWS STS Duration Seconds
			"us-west-2",           // default region
			"y",                   // enable IAM validation
			"SpectroCloudRole",    // IAM role name
			"Base",                // IAM check type
			"y",                   // enable service quota validation
			"EC2",                 // rule name
			"EC2-VPC Elastic IPs", // service quota type
			"us-west-2",           // service quota region #1
			"5",                   // service quota buffer #1
			"n",                   // add another service quota rule
			"y",                   // enable subnet tag validation
			"subnet",              // tag resource type
			"elb tag rule",        // rule name
			"us-west-2",           // subnet tag region #1
			"foo",                 // subnet tag key #1
			"bar",                 // subnet tag value #1
			"baz",                 // subnet arn #1
			"n",                   // add another subnet arn
			"n",                   // add another subnet tag rule
			"n",                   // add another tag rule

			// Azure plugin
			"y",                         // enable plugin
			cfg.ValidatorHelmRepository, // helm chart repo
			cfg.ValidatorChartVersions[cfg.ValidatorPluginAzure], // helm chart version
			"n",                                    // Re-use validator chart security configuration
			"y",                                    // insecure skip verify
			"n",                                    // use basic auth
			"n",                                    // implicit plugin auth
			"azure-creds",                          // k8s secret name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // tenant id
			"d551b7b1-78ae-43df-9d61-4935c843a454", // client id
			"test_client_secret",                   // client secret
			"Cluster Deployment",                   // rule type (select)
			"Static",                               // placement type (select)
			"Single cluster",                       // static deployment style (select)
			"rule-1",                               // rule name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // service principal
			"d551b7b1-78ae-43df-9d61-4935c843a454", // subscription
			"rg",                                   // resource group
			"vn",                                   // virtual network
			"s",                                    // subnet
			"acg",                                  // azure compute gallery
			"y",                                    // add RBAC rule
			"Cluster Deployment",                   // rule type (select)
			"Static",                               // placement type (select)
			"Multiple clusters in a single resource group", // static deployment style (select)
			"rule-2",                               // rule name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // service principal
			"d551b7b1-78ae-43df-9d61-4935c843a454", // subscription
			"rg",                                   // resource group
			"y",                                    // add RBAC rule
			"Cluster Deployment",                   // rule type (select)
			"Static",                               // placement type (select)
			"Multiple clusters in a single subscription", // static deployment style (select)
			"rule-3",                               // rule name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // service principal
			"d551b7b1-78ae-43df-9d61-4935c843a454", // subscription
			"y",                                    // add RBAC rule
			"Cluster Deployment",                   // rule type (select)
			"Dynamic",                              // placement type (select)
			"rule-4",                               // rule name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // service principal
			"d551b7b1-78ae-43df-9d61-4935c843a454", // subscription
			"y",                                    // add RBAC rule
			"Custom",                               // rule type (select)
			"rule-5",                               // rule name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // security principal
			"s",                                    // scope
			"a",                                    // Action
			"n",                                    // add Action
			"da",                                   // DataAction
			"n",                                    // add DataAction
			"n",                                    // add permission set
			"n",                                    // add RBAC rule

			// Network plugin
			"y",                         // enable Network plugin
			cfg.ValidatorHelmRepository, // validator-plugin-network helm chart repo
			cfg.ValidatorChartVersions[cfg.ValidatorPluginNetwork], // validator-plugin-network helm chart version
			"y",           // Re-use validator chart security configuration
			"y",           // enable DNS validation
			"resolve foo", // DNS rule name
			"foo",         // DNS host
			"",            // DNS nameserver
			"n",           // add another DNS rule
			"y",           // enable ICMP validation
			"ping foo",    // ICMP rule name
			"foo",         // ICMP host
			"n",           // add another ICMP rule
			"y",           // enable IP range validation
			"check ips",   // IP range rule name
			"10.10.10.10", // first IPv4 in range
			"10",          // length of IPv4 range
			"n",           // add another IP range rule
			"y",           // enable MTU validation
			"check mtu",   // MTU rule name
			"foo",         // MTU host
			"1500",        // minimum MTU
			"n",           // add another MTU rule
			"y",           // enable TCP connection validation
			"check tcp",   // TCP connection rule name
			"foo",         // TCP connection host
			"80",          // TCP connection port
			"n",           // add another port
			"n",           // add another TCP connection rule

			// OCI plugin
			"y",                         // enable OCI plugin
			cfg.ValidatorHelmRepository, // validator-plugin-oci helm chart repo
			cfg.ValidatorChartVersions[cfg.ValidatorPluginOci], // validator-plugin-oci helm chart version
			"y",                        // Re-use validator chart security configuration
			"y",                        // add registry credentials
			"oci-creds",                // secret name
			"user1",                    // username
			"pa$$w0rd",                 // password
			"n",                        // add another registry credential
			"y",                        // add signature verification secret
			"cosign-pubkeys",           // secret name
			t.filePath("cosign.pub"),   // public key file
			"n",                        // add another public key to this secret
			"n",                        // add another signature verification secret
			"public ecr registry",      // rule name
			"public.ecr.aws",           // registry host
			"N/A",                      // registry auth secret name
			"u5n5j0b4/oci-test-public", // artifact ref
			"n",                        // enable full layer validation
			"n",                        // add another artifact
			"N/A",                      // signature verification secret name
			"",                         // ca certificate
			"n",                        // add another registry rule

			// vSphere plugin
			"y",                         // enable vsphere plugin
			cfg.ValidatorHelmRepository, // validator-plugin-vsphere helm chart repo
			cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere], // validator-plugin-vsphere helm chart version
			"y",                                // Re-use validator chart security configuration
			"vsphere-creds",                    // vSphere secret name
			"fake.vsphere.com",                 // vSphere domain
			"bob@vsphere.com",                  // vSphere username
			"password",                         // vSphere password
			"y",                                // insecure skip verify
			"DC0",                              // datacenter
			"y",                                // Enable NTP check
			"ntpd",                             // NTP rule name
			"y",                                // are hosts cluster scoped
			"C0",                               // cluster name
			"DC0_C0_H0",                        // host1
			"y",                                // add more hosts
			"DC0_C0_H1",                        // host2
			"n",                                // add more hosts
			"n",                                // add more validation rules
			"y",                                // Check role privileges
			"role rule 1",                      // Role privilege rule name
			"user1@vsphere.local",              // user to check role privileges against
			cfg.SpectroRootLevelPrivilegesV7_0, // vSphere permission version
			"n",                                // add more role privilege checks
			"y",                                // check entity privileges
			"entity rule 1",                    // entity privilege rule name
			cfg.SpectroEntityPrivileges,        // entity level permissions
			"Read folder: spectro-templates",   // spectro entity permission
			"user2@vsphere.local",              // user to check entity privileges against
			"n",                                // add more entity permission checks
			"y",                                // check compute resource requirements
			"resource requirement rule 1",      // resource requirement rule name
			"Cluster",                          // select cluster for resource check
			"C0",                               // cluster name for resource check
			"master-pool",                      // node pool name
			"1",                                // number of nodes
			"2GHz",                             // per node cpu
			"4Gi",                              // per node memory
			"10Gi",                             // per node storage
			"y",                                // add another node pool
			"worker-pool",                      // node pool name
			"3",                                // number of nodes
			"3GHz",                             // per node cpu
			"8Gi",                              // per node memory
			"20Gi",                             // per node storage
			"n",                                // add more node pools
			"n",                                // add more resource requirement checks
			"y",                                // check tags on entities
			"tag rule 1",                       // tag rule name
			cfg.SpectroCloudTags,               // zone & region tags
			"Cluster: k8s-zone (ensure that the selected cluster has a 'k8s-zone' tag)", // zone tag
			"C0", // cluster name
			"n",  // add another tag rule

			// Finalization
			"n", // restart configuration
			"n", // reconfigure plugin(s)
		},
	}

	return common.ExecCLI(interactiveCmd, buffer, t.log)
}

func (t *ValidatorTest) testDeploySilent() (tr *test.TestResult) {
	silentCmd, buffer := common.InitCmd([]string{
		"install", "-l", "debug", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log)
}

func (t *ValidatorTest) testDescribe() (tr *test.TestResult) {
	silentCmd, buffer := common.InitCmd([]string{
		"describe", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log)
}

func (t *ValidatorTest) testUndeploy() (tr *test.TestResult) {
	silentCmd, buffer := common.InitCmd([]string{
		"uninstall", "-f", t.filePath(cfg.ValidatorConfigFile),
	})
	return common.ExecCLI(silentCmd, buffer, t.log)
}

func (t *ValidatorTest) testUpdatePasswords() (tr *test.TestResult) {
	cmd, buffer := common.InitCmd([]string{
		"install", "-f", t.filePath(cfg.ValidatorConfigFile), "-p",
	})

	clouds.GetVSphereDriver = func(account *vsphere_cloud.VsphereCloudAccount) (vsphere_cloud.VsphereDriver, error) {
		return vsphere_cloud.MockVsphereDriver{}, nil
	}

	prompts.Tui = &tuimocks.MockTUI{
		ReturnVals: []string{
			// Validator
			"y",                // Allow Insecure Connection (Bypass x509 Verification)
			"y",                // Use Helm basic auth
			"validator-secret", // Helm secret name
			"admin",            // Helm username
			"welcome",          // Helm password

			// AWS validator
			"n",         // Re-use validator chart security configuration
			"y",         // Allow Insecure Connection (Bypass x509 Verification)
			"n",         // Use Helm basic auth
			"n",         // Use implicit AWS auth
			"aws-creds", // AWS credentials secret name
			"abc",       // AWS Access Key
			"xyz",       // AWS Secret Key
			"",          // AWS Session Token
			"n",         // Use STS

			// Azure validator
			"y",                                    // Re-use validator chart security configuration
			"n",                                    // Use implicit Azure auth
			"azure-creds",                          // Azure credentials secret name
			"d551b7b1-78ae-43df-9d61-4935c843a454", // Azure Tenant ID
			"d551b7b1-78ae-43df-9d61-4935c843a454", // Azure Client ID
			"test_azure_client_secret",             // Azure Client Secret

			// OCI validator
			"n",         // Re-use validator chart security configuration
			"y",         // Allow Insecure Connection (Bypass x509 Verification)
			"n",         // Use Helm basic auth
			"user2",     // Registry username
			"password2", // Registry password

			// vSphere validator
			"y",                // Re-use validator chart security configuration
			"vsphere-creds",    // vSphere credentials secret name
			"vcenter.test.dev", // vSphere endpoint
			"bob@vsphere.com",  // vSphere username
			"123",              // vSphere password
			"y",                // vSphere insecureSkipVerify
		},
	}

	return common.ExecCLI(cmd, buffer, t.log)
}

func (t *ValidatorTest) PreRequisite(ctx *test.TestContext) (tr *test.TestResult) {
	t.log.Printf("Executing ExecuteRequisite for %s and %s", t.GetName(), t.GetDescription())
	if err := common.PreRequisiteFun()(ctx); err != nil {
		return test.Failure(err.Error())
	}
	return test.Success()
}

func (t *ValidatorTest) TearDown(ctx *test.TestContext) {
	t.log.Printf("Executing TearDown for %s and %s ", t.GetName(), t.GetDescription())

	if err := kind.DeleteCluster(cfg.ValidatorKindClusterName); err != nil {
		t.log.Errorf("Failed to delete validator kind cluster: %v", err)
	}
	if err := common.TearDownFun()(ctx); err != nil {
		t.log.Errorf("Failed to run teardown fun: %v", err)
	}

	// restore clouds.GetVSphereDriver
	vsphereDriverFunc := ctx.Get("vsphereDriverFunc")
	clouds.GetVSphereDriver = vsphereDriverFunc.(func(account *vsphere_cloud.VsphereCloudAccount) (vsphere_cloud.VsphereDriver, error))
}

func (t *ValidatorTest) filePath(file string) string {
	return fmt.Sprintf("%s/%s/%s", file_utils.ValidatorTestCasesPath(), "data", file)
}
