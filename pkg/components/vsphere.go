package components

import (
	"fmt"

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
	// TODO: properly handle TLS, helm, and air-gap config
	vc.VspherePlugin = &VspherePluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:                  cfg.ValidatorPluginVsphere,
				Repository:            fmt.Sprintf("%s/%s", cfg.ValidatorHelmRepository, cfg.ValidatorPluginVsphere),
				Version:               cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere],
				InsecureSkipTLSVerify: true,
			},
		},
		ReleaseSecret: &Secret{
			Name:      fmt.Sprintf("validator-helm-release-%s", cfg.ValidatorPluginVsphere),
			BasicAuth: &BasicAuth{},
		},
		Account: &vsphere.CloudAccount{
			Insecure:      true,
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
