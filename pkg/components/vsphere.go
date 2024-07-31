package components

import (
	vsphereapi "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
	vapi "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

// VsphereConfig represents the vSphere plugin configuration.
type VsphereConfig struct {
	Username                     string
	Password                     string
	VcenterServer                string
	Datacenter                   string
	ClusterName                  string
	ImageTemplateFolder          string
	NodePoolResourceRequirements []vsphereapi.NodepoolResourceRequirement
	TagValidationRules           []vsphereapi.TagValidationRule
	Privileges                   []string
}

// ConfigureVspherePlugin configures the vSphere plugin.
func ConfigureVspherePlugin(vc *ValidatorConfig, config VsphereConfig) {
	// TODO: prompt for chart version if !vc.UseFixedVersions
	vc.VspherePlugin = &VspherePluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:       cfg.ValidatorPluginVsphere,
				Repository: cfg.ValidatorPluginVsphere,
				Version:    cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere],
			},
		},
		Account: &vsphere.CloudAccount{
			Insecure:      true, // TODO: get this from VsphereConfig
			Username:      config.Username,
			Password:      config.Password,
			VcenterServer: config.VcenterServer,
		},
		Validator: &vsphereapi.VsphereValidatorSpec{
			Auth: vsphereapi.VsphereAuth{
				SecretName: "vsphere-creds",
			},
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
