//revive:disable

package config

import (
	"github.com/spectrocloud-labs/prompts-tui/prompts"

	vtypes "github.com/validator-labs/validator/pkg/types"
)

const (
	ConfigFile   = "validatorctl.yaml"
	TimeFormat   = "20060102150405"
	WorkspaceLoc = ".validator"

	ClusterConfigTemplate = "cluster-configuration.tmpl"
	KindImage             = "kindest/node"
	KindImageTag          = "v1.30.2"
	NoProxyPrompt         = "# Default NO_PROXY values are on the lines below.\n# Edit as you see fit (comments are ignored). The file should contain a list of NO_PROXY values, newline separated.\n# Type :wq to save and exit (if using vi).\n\n"

	LocalFilepath = "Local Filepath"
	FileEditor    = "File Editor"

	// Validator constants
	ValidatorConfigFile      = "validator.yaml"
	ValidatorKindClusterName = "validator-kind-cluster"
	ValidatorHelmRegistry    = "https://validator-labs.github.io"
	ValidatorImageRegistry   = "quay.io"
	ValidatorImageRepository = "validator-labs"
	ValidatorHelmReleaseName = "validator-helm-release"

	ValidatorPluginAws     = "validator-plugin-aws"
	ValidatorPluginAzure   = "validator-plugin-azure"
	ValidatorPluginMaas    = "validator-plugin-maas"
	ValidatorPluginNetwork = "validator-plugin-network"
	ValidatorPluginOci     = "validator-plugin-oci"
	ValidatorPluginVsphere = "validator-plugin-vsphere"

	ValidatorPluginAwsKind     = "AwsValidator"
	ValidatorPluginAzureKind   = "AzureValidator"
	ValidatorPluginMaasKind    = "MaasValidator"
	ValidatorPluginNetworkKind = "NetworkValidator"
	ValidatorPluginOciKind     = "OciValidator"
	ValidatorPluginVsphereKind = "VsphereValidator"

	ValidatorPluginAwsTemplate     = "validator-rules-aws.tmpl"
	ValidatorPluginAzureTemplate   = "validator-rules-azure.tmpl"
	ValidatorPluginMaasTemplate    = "validator-rules-maas.tmpl"
	ValidatorPluginNetworkTemplate = "validator-rules-network.tmpl"
	ValidatorPluginOciTemplate     = "validator-rules-oci.tmpl"
	ValidatorPluginVsphereTemplate = "validator-rules-vsphere.tmpl"

	ValidatorVsphereVersionConstraint = ">= 6.0, < 9.0"
	ValidatorVspherePrivilegeFile     = "vsphere-privileges-7.x.yaml"

	AWSPolicyDocumentPrompt   = "# Provide the AWS policy document for IAM validation rule. The policy document should be in JSON format. Type :wq to save and exit (if using vi).\n"
	AzurePermissionSetPrompt  = "# Provide the Azure permission set for IAM validation rule. The permission set should be in JSON format. Type :wq to save and exit (if using vi).\n"
	VcenterPrivilegePrompt    = "# All valid vCenter privileges are on the lines below.\n# Edit as you see fit (comments are ignored). The file should contain a list of privileges, newline separated.\n# Type :wq to save and exit (if using vi).\n\n"
	OciCreateNewAuthSecPrompt = "Create a new registry authentication secret"
	OciCreateNewSigSecPrompt  = "Create a new signature verification secret"

	// Embed dirs
	Kind      string = "kind"
	Validator string = "validator"

	// Regex
	DomainRegex          = "([a-zA-Z0-9]{1,63}|[a-zA-Z0-9][a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])(\\.[a-zA-Z0-9]{1,63}|\\.[a-zA-Z0-9][a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9]){0,10}\\.([a-zA-Z0-9][a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]{0,30}[a-zA-Z0-9]\\.[a-zA-Z]{2,})"
	UsernameRegex        = "[a-zA-Z0-9]+(?:\\.[a-zA-Z0-9]+)*(?:-[a-zA-Z0-9]+)*(?:_[a-zA-Z0-9]+)*"
	VSphereUsernameRegex = "^" + UsernameRegex + "@" + DomainRegex + "$"
	CPUReqRegex          = "(^\\d+\\.?\\d*[M,G]Hz)"
	MemoryReqRegex       = "(^\\d+\\.?\\d*[M,G,T]i)"
	DiskReqRegex         = "(^\\d+\\.?\\d*[M,G,T]i)"
	PolicyArnRegex       = "^arn:aws:iam::.*:policy/.*$"

	// Env vars
	AwsAccessKey       = "AWS_ACCESS_KEY_ID"     // #nosec
	AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY" // #nosec
	AwsSessionToken    = "AWS_SESSION_TOKEN"     // #nosec
)

var (
	// Misc.
	DefaultPodCIDR          = "192.168.0.0/16"
	DefaultServiceIPRange   = "10.96.0.0/12"
	HTTPSchemes             = []string{"https://", "http://"}
	RegistryMirrors         = []string{"docker.io", "gcr.io", "ghcr.io", "k8s.gcr.io", "registry.k8s.io", "quay.io", "*"}
	RegistryMirrorSeparator = "::"
	FileInputs              = []string{LocalFilepath, FileEditor}
	DNSRecordTypes          = []string{"A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "SSHFP"}

	// Command dirs
	ValidatorSubdirs = []string{"logs", "manifests"}

	// Validator
	ValidatorImagePath = func() string {
		return ValidatorImageRegistry + "/" + ValidatorImageRepository
	}
	ValidatorWaitCmd       = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-controller-manager", "-n", "validator"}
	ValidatorBasicAuthKeys = []string{"username", "password"}
	ValidatorSinkKeys      = map[vtypes.SinkType][]string{
		vtypes.SinkTypeAlertmanager: {"endpoint", "insecureSkipVerify", "username", "password", "caCert"},
		vtypes.SinkTypeSlack:        {"apiToken", "channelID"},
	}
	ValidatorPluginAwsKeys                     = []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"}
	ValidatorPluginAzureKeys                   = []string{"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET"}
	ValidatorPluginVsphereKeys                 = []string{"username", "password", "vcenterServer", "insecureSkipVerify"}
	ValidatorPluginOciSigVerificationKeysRegex = ".pub$"

	ValidatorPluginAwsServiceQuotas = []prompts.ChoiceItem{
		{
			ID:   "ec2",
			Name: "EC2-VPC Elastic IPs",
		},
		{
			ID:   "ec2",
			Name: "Public AMIs",
		},
		{
			ID:   "elasticfilesystem",
			Name: "File systems per account",
		},
		{
			ID:   "elasticloadbalancing",
			Name: "Application Load Balancers per Region",
		},
		{
			ID:   "elasticloadbalancing",
			Name: "Classic Load Balancers per Region",
		},
		{
			ID:   "elasticloadbalancing",
			Name: "Network Load Balancers per Region",
		},
		{
			ID:   "vpc",
			Name: "Internet gateways per Region",
		},
		{
			ID:   "vpc",
			Name: "Network interfaces per Region",
		},
		{
			ID:   "vpc",
			Name: "VPCs per Region",
		},
		{
			ID:   "vpc",
			Name: "Subnets per VPC",
		},
		{
			ID:   "vpc",
			Name: "NAT gateways per Availability Zone",
		},
	}

	ValidatorAzureClouds = []string{
		"AzureCloud",
		"AzureUSGovernment",
		"AzureChinaCloud",
	}
)
