package components

import (
	vsphereapi "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
)

// VsphereConfig represents the vSphere plugin configuration.
type VsphereConfig struct {
	Username                     string
	Password                     string
	VcenterServer                string
	Insecure                     bool
	Datacenter                   string
	ClusterName                  string
	ImageTemplateFolder          string
	NodePoolResourceRequirements []vsphereapi.NodepoolResourceRequirement
	TagValidationRules           []vsphereapi.TagValidationRule
	Privileges                   []string
}

// ConfigureVspherePlugin configures the vSphere plugin.
func ConfigureVspherePlugin(vc *ValidatorConfig, config VsphereConfig) {
	vc.VspherePlugin = &VspherePluginConfig{
		Enabled: true,
		Account: &vsphere.CloudAccount{
			Insecure:      config.Insecure,
			Username:      config.Username,
			Password:      config.Password,
			VcenterServer: config.VcenterServer,
		},
		Validator: &vsphereapi.VsphereValidatorSpec{
			Datacenter: config.Datacenter,
			ComputeResourceRules: []vsphereapi.ComputeResourceRule{
				{
					Name:                         "Cluster Compute Resource Availability",
					ClusterName:                  config.ClusterName,
					Scope:                        "cluster",
					EntityName:                   config.ClusterName,
					NodepoolResourceRequirements: config.NodePoolResourceRequirements,
				},
			},
			EntityPrivilegeValidationRules: []vsphereapi.EntityPrivilegeValidationRule{
				{
					Name:       "Create folder: image template folder",
					Username:   config.Username,
					EntityType: "folder",
					EntityName: config.ImageTemplateFolder,
					Privileges: []string{"Folder.Create"},
				},
			},
			RolePrivilegeValidationRules: []vsphereapi.GenericRolePrivilegeValidationRule{
				{
					Username:   config.Username,
					Privileges: config.Privileges,
				},
			},
			TagValidationRules: config.TagValidationRules,
		},
	}
}
