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

	// Validator constants
	ValidatorConfigFile      = "validator.yaml"
	ValidatorKindClusterName = "validator-kind-cluster"
	ValidatorHelmRepository  = "https://validator-labs.github.io"
	ValidatorImageRegistry   = "quay.io"
	ValidatorImageRepository = "validator-labs"

	ValidatorPluginAws     = "validator-plugin-aws"
	ValidatorPluginAzure   = "validator-plugin-azure"
	ValidatorPluginNetwork = "validator-plugin-network"
	ValidatorPluginOci     = "validator-plugin-oci"
	ValidatorPluginVsphere = "validator-plugin-vsphere"

	ValidatorPluginAwsTemplate     = "validator-rules-aws.tmpl"
	ValidatorPluginAzureTemplate   = "validator-rules-azure.tmpl"
	ValidatorPluginNetworkTemplate = "validator-rules-network.tmpl"
	ValidatorPluginOciTemplate     = "validator-rules-oci.tmpl"
	ValidatorPluginVsphereTemplate = "validator-rules-vsphere.tmpl"

	ValidatorVsphereEntityDatacenter     = "Datacenter"
	ValidatorVsphereEntityCluster        = "Cluster"
	ValidatorVsphereEntityFolder         = "Folder"
	ValidatorVsphereEntityResourcePool   = "Resource Pool"
	ValidatorVsphereEntityHost           = "ESXi Host"
	ValidatorVsphereEntityVirtualMachine = "Virtual Machine"
	ValidatorVsphereEntityVirtualApp     = "Virtual App"
	ValidatorVsphereVersionConstraint    = ">= 6.0, < 9.0"
	ValidatorVspherePrivilegeFile        = "vsphere-root-level-privileges-all.yaml"

	AWSPolicyDocumentPrompt = "# Provide the AWS policy document for IAM validation rule. The policy document should be in JSON format. Type :wq to save and exit (if using vi).\n"

	DefaultStorageClassAnnotation string = "storageclass.kubernetes.io/is-default-class"

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
)

var (
	// Misc.
	DefaultPodCIDR          = "192.168.0.0/16"
	DefaultServiceIPRange   = "10.96.0.0/12"
	HTTPSchemes             = []string{"https://", "http://"}
	RegistryMirrors         = []string{"docker.io", "gcr.io", "ghcr.io", "k8s.gcr.io", "registry.k8s.io", "quay.io", "*"}
	RegistryMirrorSeparator = "::"

	// Command dirs
	ValidatorSubdirs = []string{"manifests"}

	// Validator
	ValidatorImagePath = func() string {
		return ValidatorImageRegistry + "/" + ValidatorImageRepository
	}

	PlacementTypeStatic  = "Static"
	PlacementTypeDynamic = "Dynamic"
	PlacementTypes       = []string{PlacementTypeStatic, PlacementTypeDynamic}

	// TODO: centralize these in a single place referenced by validator & validatorctl
	ValidatorChartVersions = map[string]string{
		Validator:              "v0.0.46",
		ValidatorPluginAws:     "v0.1.1",
		ValidatorPluginAzure:   "v0.0.12",
		ValidatorPluginNetwork: "v0.0.17",
		ValidatorPluginOci:     "v0.0.10",
		ValidatorPluginVsphere: "v0.0.26",
	}

	ValidatorWaitCmd              = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-controller-manager", "-n", "validator"}
	ValidatorPluginAwsWaitCmd     = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-plugin-aws-controller-manager", "-n", "validator"}
	ValidatorPluginVsphereWaitCmd = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-plugin-vsphere-controller-manager", "-n", "validator"}
	ValidatorPluginNetworkWaitCmd = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-plugin-network-controller-manager", "-n", "validator"}
	ValidatorPluginOciWaitCmd     = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-plugin-oci-controller-manager", "-n", "validator"}
	ValidatorPluginAzureWaitCmd   = []string{"wait", "--for=condition=available", "--timeout=600s", "deployment/validator-plugin-azure-controller-manager", "-n", "validator"}

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

	ValidatorPluginVsphereEntities = []string{
		ValidatorVsphereEntityCluster,
		ValidatorVsphereEntityDatacenter,
		ValidatorVsphereEntityHost,
		ValidatorVsphereEntityFolder,
		ValidatorVsphereEntityResourcePool,
		ValidatorVsphereEntityVirtualApp,
		ValidatorVsphereEntityVirtualMachine,
	}
	ValidatorPluginVsphereEntityMap = map[string]string{
		ValidatorVsphereEntityCluster:        "cluster",
		ValidatorVsphereEntityDatacenter:     "datacenter",
		ValidatorVsphereEntityHost:           "host",
		ValidatorVsphereEntityFolder:         "folder",
		ValidatorVsphereEntityResourcePool:   "resourcepool",
		ValidatorVsphereEntityVirtualApp:     "vapp",
		ValidatorVsphereEntityVirtualMachine: "vm",
	}
	ValidatorPluginVsphereDeploymentDestination = []string{
		ValidatorVsphereEntityCluster,
		ValidatorVsphereEntityHost,
		ValidatorVsphereEntityResourcePool,
	}

	ValidatorAzurePluginStaticPlacementResourceGroupLevelActions = []string{
		"Microsoft.Compute/disks/delete",
		"Microsoft.Compute/disks/read",
		"Microsoft.Compute/disks/write",
		"Microsoft.Compute/virtualMachines/delete",
		"Microsoft.Compute/virtualMachines/extensions/delete",
		"Microsoft.Compute/virtualMachines/extensions/read",
		"Microsoft.Compute/virtualMachines/extensions/write",
		"Microsoft.Compute/virtualMachines/read",
		"Microsoft.Compute/virtualMachines/write",
		"Microsoft.Network/loadBalancers/backendAddressPools/join/action",
		"Microsoft.Network/loadBalancers/delete",
		"Microsoft.Network/loadBalancers/inboundNatRules/delete",
		"Microsoft.Network/loadBalancers/inboundNatRules/join/action",
		"Microsoft.Network/loadBalancers/inboundNatRules/read",
		"Microsoft.Network/loadBalancers/inboundNatRules/write",
		"Microsoft.Network/loadBalancers/read",
		"Microsoft.Network/loadBalancers/write",
		"Microsoft.Network/networkInterfaces/delete",
		"Microsoft.Network/networkInterfaces/join/action",
		"Microsoft.Network/networkInterfaces/read",
		"Microsoft.Network/networkInterfaces/write",
		"Microsoft.Network/networkSecurityGroups/read",
		"Microsoft.Network/networkSecurityGroups/securityRules/delete",
		"Microsoft.Network/networkSecurityGroups/securityRules/read",
		"Microsoft.Network/networkSecurityGroups/securityRules/write",
		"Microsoft.Network/privateDnsZones/A/delete",
		"Microsoft.Network/privateDnsZones/A/read",
		"Microsoft.Network/privateDnsZones/A/write",
		"Microsoft.Network/privateDnsZones/delete",
		"Microsoft.Network/privateDnsZones/read",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/delete",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/read",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/write",
		"Microsoft.Network/privateDnsZones/write",
		"Microsoft.Network/publicIPAddresses/delete",
		"Microsoft.Network/publicIPAddresses/join/action",
		"Microsoft.Network/publicIPAddresses/read",
		"Microsoft.Network/publicIPAddresses/write",
		"Microsoft.Network/routeTables/delete",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Resources/subscriptions/resourceGroups/read",
	}
	ValidatorAzurePluginStaticPlacementVirtualNetworkLevelActions = []string{
		"Microsoft.Network/virtualNetworks/read",
	}
	ValidatorAzurePluginStaticPlacementSubnetLevelActions = []string{
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
	}
	ValidatorAzurePluginStaticPlacementComputeGalleryLevelActions = []string{
		"Microsoft.Compute/galleries/images/read",
		"Microsoft.Compute/galleries/images/versions/read",
	}
	ValidatorAzurePluginDynamicPlacementActions = []string{
		"Microsoft.Compute/disks/delete",
		"Microsoft.Compute/disks/read",
		"Microsoft.Compute/disks/write",
		"Microsoft.Compute/virtualMachines/delete",
		"Microsoft.Compute/virtualMachines/extensions/delete",
		"Microsoft.Compute/virtualMachines/extensions/read",
		"Microsoft.Compute/virtualMachines/extensions/write",
		"Microsoft.Compute/virtualMachines/read",
		"Microsoft.Compute/virtualMachines/write",
		"Microsoft.Network/loadBalancers/backendAddressPools/join/action",
		"Microsoft.Network/loadBalancers/delete",
		"Microsoft.Network/loadBalancers/inboundNatRules/delete",
		"Microsoft.Network/loadBalancers/inboundNatRules/join/action",
		"Microsoft.Network/loadBalancers/inboundNatRules/read",
		"Microsoft.Network/loadBalancers/inboundNatRules/write",
		"Microsoft.Network/loadBalancers/read",
		"Microsoft.Network/loadBalancers/write",
		"Microsoft.Network/networkInterfaces/delete",
		"Microsoft.Network/networkInterfaces/join/action",
		"Microsoft.Network/networkInterfaces/read",
		"Microsoft.Network/networkInterfaces/write",
		"Microsoft.Network/networkSecurityGroups/read",
		"Microsoft.Network/networkSecurityGroups/securityRules/delete",
		"Microsoft.Network/networkSecurityGroups/securityRules/read",
		"Microsoft.Network/networkSecurityGroups/securityRules/write",
		"Microsoft.Network/publicIPAddresses/delete",
		"Microsoft.Network/publicIPAddresses/join/action",
		"Microsoft.Network/publicIPAddresses/read",
		"Microsoft.Network/publicIPAddresses/write",
		"Microsoft.Network/routeTables/delete",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
		"Microsoft.Resources/subscriptions/resourceGroups/read",
		"Microsoft.Network/privateDnsZones/read",
		"Microsoft.Network/privateDnsZones/write",
		"Microsoft.Network/privateDnsZones/delete",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/read",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/write",
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/delete",
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/privateDnsZones/A/write",
		"Microsoft.Network/privateDnsZones/A/read",
		"Microsoft.Network/privateDnsZones/A/delete",
		"Microsoft.Storage/storageAccounts/blobServices/containers/write",
		"Microsoft.Storage/storageAccounts/blobServices/containers/read",
		"Microsoft.Storage/storageAccounts/write",
		"Microsoft.Storage/storageAccounts/read",
		"Microsoft.Storage/storageAccounts/blobServices/listKeys/action",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/delete",
		"Microsoft.Network/virtualNetworks/virtualMachines/read",
		"Microsoft.Network/virtualNetworks/virtualNetworkPeerings/read",
		"Microsoft.Network/virtualNetworks/virtualNetworkPeerings/write",
		"Microsoft.Network/virtualNetworks/virtualNetworkPeerings/delete",
		"Microsoft.Network/virtualNetworks/peer/action",
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/joinLoadBalancer/action",
		"Microsoft.Network/virtualNetworks/subnets/write",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/delete",
		"Microsoft.Network/virtualNetworks/subnets/virtualMachines/read",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/joinLoadBalancer/action",
		"Microsoft.Compute/images/write",
		"Microsoft.Compute/images/read",
		"Microsoft.Compute/galleries/write",
		"Microsoft.Compute/galleries/read",
		"Microsoft.Compute/galleries/images/write",
		"Microsoft.Compute/galleries/images/read",
		"Microsoft.Compute/galleries/images/versions/read",
		"Microsoft.Compute/galleries/images/versions/write",
	}
)
