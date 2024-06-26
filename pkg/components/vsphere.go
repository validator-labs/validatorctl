package components

import (
	"fmt"

	vsphere_api "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
	vapi "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

type VsphereConfig struct {
	Username                     string
	Password                     string
	VcenterServer                string
	Datacenter                   string
	ClusterName                  string
	ImageTemplateFolder          string
	NodePoolResourceRequirements []vsphere_api.NodepoolResourceRequirement
	TagValidationRules           []vsphere_api.TagValidationRule
	Privileges                   []string
}

func ConfigureVspherePlugin(vc *ValidatorConfig, config VsphereConfig) error {
	vc.VspherePlugin = &VspherePluginConfig{
		Enabled: true,
		Release: &vapi.HelmRelease{
			Chart: vapi.HelmChart{
				Name:                  cfg.ValidatorPluginVsphere,
				Repository:            fmt.Sprintf("%s/%s", cfg.ValidatorHelmRepository, cfg.ValidatorPluginVsphere),
				Version:               cfg.ValidatorChartVersions[cfg.ValidatorPluginVsphere],
				InsecureSkipTlsVerify: true,
			},
		},
		ReleaseSecret: &Secret{
			Name: fmt.Sprintf("validator-helm-release-%s", cfg.ValidatorPluginVsphere),
		},
		Account: &vsphere.VsphereCloudAccount{
			Insecure:      true,
			Username:      config.Username,
			Password:      config.Password,
			VcenterServer: config.VcenterServer,
		},
		Validator: &vsphere_api.VsphereValidatorSpec{
			Auth: vsphere_api.VsphereAuth{
				SecretName: "vsphere-creds",
			},
			Datacenter: config.Datacenter,
			ComputeResourceRules: []vsphere_api.ComputeResourceRule{
				{
					Name:                         "Cluster Compute Resource Availability",
					ClusterName:                  config.ClusterName,
					Scope:                        "cluster",
					EntityName:                   config.ClusterName,
					NodepoolResourceRequirements: config.NodePoolResourceRequirements,
				},
			},
			EntityPrivilegeValidationRules: []vsphere_api.EntityPrivilegeValidationRule{
				{
					Name:       "Create folder: image template folder",
					Username:   config.Username,
					EntityType: "folder",
					EntityName: config.ImageTemplateFolder,
					Privileges: []string{"Folder.Create"},
				},
			},
			RolePrivilegeValidationRules: []vsphere_api.GenericRolePrivilegeValidationRule{
				{
					Username:   config.Username,
					Privileges: config.Privileges,
				},
			},
			TagValidationRules: config.TagValidationRules,
		},
	}
	return nil
}
