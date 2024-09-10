package validator

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/vmware/govmomi/object"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/api/vcenter"
	"github.com/validator-labs/validator-plugin-vsphere/api/vcenter/entity"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/validators/tags"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"
	"github.com/validator-labs/validatorctl/pkg/utils/embed"
)

var (
	vSphereSecretName = "vsphere-creds" //#nosec G101
	dataCenter        = "Datacenter"
)

func readVspherePlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	c := vc.VspherePlugin

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginVsphere, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}
	if err := readVsphereCredentials(c, tc, k8sClient); err != nil {
		return fmt.Errorf("failed to read vSphere credentials: %w", err)
	}

	return nil
}

func readVspherePluginRules(vc *components.ValidatorConfig, _ *cfg.TaskConfig, _ kubernetes.Interface) error {
	log.Header("vSphere Plugin Rule Configuration")
	c := vc.VspherePlugin

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	driver, err := clouds.GetVSphereDriver(*c.Validator.Auth.Account)
	if err != nil {
		return err
	}

	if c.Validator.Datacenter != "" {
		dataCenter = c.Validator.Datacenter
	}
	c.Validator.Datacenter, err = prompts.ReadText("Datacenter", dataCenter, false, -1)
	if err != nil {
		return err
	}

	ruleNames := make([]string, 0)

	if err := configureNtpRules(ctx, c, driver, &ruleNames); err != nil {
		return err
	}
	if err := configurePrivilegeRules(ctx, c, driver, &ruleNames); err != nil {
		return err
	}
	if err := configureResourceRequirementRules(ctx, c, driver, &ruleNames); err != nil {
		return err
	}
	if err := configureVsphereTagRules(ctx, c, driver, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	return nil
}

func readVsphereCredentials(c *components.VspherePluginConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	var err error
	c.Validator.Auth.Account = &vcenter.Account{}
	// always create vSphere credential secret if creating a new kind cluster
	createSecret := true

	if k8sClient != nil {
		log.InfoCLI(`
	Either specify vSphere credentials or provide the name of a secret in the target K8s cluster's %s namespace.
	If using an existing secret, it must contain the following keys: %+v.
	`, cfg.Validator, cfg.ValidatorPluginVsphereKeys,
		)
		createSecret, err = prompts.ReadBool("Create vSphere credential secret", true)
		if err != nil {
			return fmt.Errorf("failed to create vSphere credential secret: %w", err)
		}
	}

	if createSecret {
		if c.Validator.Auth.SecretName != "" {
			vSphereSecretName = c.Validator.Auth.SecretName
		}
		if !tc.Direct {
			c.Validator.Auth.SecretName, err = prompts.ReadText("vSphere credentials secret name", vSphereSecretName, false, -1)
			if err != nil {
				return err
			}
		}
		if err := clouds.ReadVsphereAccountProps(c.Validator.Auth.Account); err != nil {
			return err
		}
	} else {
		secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, cfg.ValidatorPluginVsphereKeys)
		if err != nil {
			return err
		}
		c.Validator.Auth.SecretName = secret.Name
		insecure, err := strconv.ParseBool(string(secret.Data["insecureSkipVerify"]))
		if err != nil {
			return err
		}
		c.Validator.Auth.Account.Host = string(secret.Data["vcenterServer"])
		c.Validator.Auth.Account.Username = string(secret.Data["username"])
		c.Validator.Auth.Account.Password = string(secret.Data["password"])
		c.Validator.Auth.Account.Insecure = insecure
	}

	// validate vSphere version
	driver, err := clouds.GetVSphereDriver(*c.Validator.Auth.Account)
	if err != nil {
		return err
	}
	if err := driver.ValidateVersion(cfg.ValidatorVsphereVersionConstraint); err != nil {
		return err
	}
	log.InfoCLI("Validated vSphere version %s", cfg.ValidatorVsphereVersionConstraint)

	return nil
}

// nolint:dupl
func configureNtpRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string) error {
	log.InfoCLI(`
	NTP validation ensures that ntpd is enabled and running on a set of ESXi hosts.
	If enabled, you will be prompted to select one or more of ESXi hosts.
	`)

	validateNtp, err := prompts.ReadBool("Enable NTP validation for ESXi host(s)", true)
	if err != nil {
		return err
	}
	if !validateNtp {
		c.Validator.NTPValidationRules = nil
		return nil
	}
	for i, r := range c.Validator.NTPValidationRules {
		r := r
		if err := readNtpRule(ctx, c, &r, driver, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.NTPValidationRules) == 0 {
		c.Validator.NTPValidationRules = make([]v1alpha1.NTPValidationRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another NTP validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readNtpRule(ctx, c, &v1alpha1.NTPValidationRule{}, driver, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another NTP validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readNtpRule(ctx context.Context, c *components.VspherePluginConfig, r *v1alpha1.NTPValidationRule, driver vsphere.Driver, idx int, ruleNames *[]string) error {
	err := initRule(r, "NTP", "The rule's ESXi host selection will be replaced.", ruleNames)
	if err != nil {
		return err
	}
	if r.ClusterName == "" {
		_, r.ClusterName, err = getClusterScopedInfo(ctx, c.Validator.Datacenter, entity.LabelMap[entity.Host], driver)
		if err != nil {
			return err
		}
	}
	r.Hosts, err = selectEsxiHosts(ctx, c.Validator.Datacenter, r.ClusterName, driver)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.NTPValidationRules = append(c.Validator.NTPValidationRules, *r)
	} else {
		c.Validator.NTPValidationRules[idx] = *r
	}
	return nil
}

func selectEsxiHosts(ctx context.Context, datacenter string, clusterName string, driver vsphere.Driver) ([]string, error) {
	hosts, err := driver.GetHostSystems(ctx, datacenter, clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list vSphere ESXi hosts")
	}

	hostList := make([]string, 0, len(hosts))
	selectedHosts := make([]string, 0, len(hosts))

	for _, host := range hosts {
		hostList = append(hostList, host.Name)
	}
	for {
		hostName, err := prompts.Select("ESXi Host Name", hostList)
		if err != nil {
			return nil, err
		}
		selectedHosts = append(selectedHosts, hostName)
		hostList = slices.DeleteFunc(hostList, func(s string) bool {
			return s == hostName
		})

		add, err := prompts.ReadBool("Add another ESXi Host", false)
		if err != nil {
			return nil, err
		}
		if !add {
			break
		}
	}
	return selectedHosts, nil
}

// loadPrivileges returns a slice of privilege IDs from the provided privilege file
func loadPrivileges(privilegeFile string) (string, func(string) error, error) {
	privilegeBytes, err := embed.EFS.ReadFile(cfg.Validator, privilegeFile)
	if err != nil {
		return "", nil, err
	}

	var privilegeMap map[string][]string
	if err := yaml.Unmarshal(privilegeBytes, &privilegeMap); err != nil {
		return "", nil, err
	}
	privileges := privilegeMap["privilegeIds"]
	slices.Sort(privileges)

	validate := func(input string) error {
		s := strings.TrimSpace(input)
		if s == "" || strings.HasPrefix(s, "#") {
			return nil
		}
		if !slices.Contains(privileges, s) {
			log.ErrorCLI("failed to read vCenter privileges", "invalidPrivilege", input)
			return prompts.ErrValidationFailed
		}
		return nil
	}

	return strings.Join(privileges, "\n"), validate, nil
}

func readPrivileges(rulePrivileges []v1alpha1.Privilege) ([]v1alpha1.Privilege, error) {
	defaultPrivileges, validate, err := loadPrivileges(cfg.ValidatorVspherePrivilegeFile)
	if err != nil {
		return nil, err
	}
	if len(rulePrivileges) > 0 {
		sb := strings.Builder{}
		for _, p := range rulePrivileges {
			sb.WriteString(p.Name + "\n")
		}
		defaultPrivileges = sb.String()
	}

	log.InfoCLI(`
	Configure vCenter privileges. Either provide a local file path to a
	file containing vCenter privileges or edit the privileges directly.

	If providing a local file path, the file should contain a list of
	vCenter privileges, newline separated. Lines starting with '#' are
	considered comments and are ignored.

	If editing the privileges directly, your default editor will be opened
	with all valid vCenter privileges prepopulated for you to edit.
	`)
	inputType, err := prompts.Select("Add privileges via", cfg.FileInputs)
	if err != nil {
		return nil, err
	}

	var privileges []string
	if inputType == cfg.LocalFilepath {
		privileges, err = readPrivilegesFromFile(validate)
	} else {
		privileges, err = readPrivilegesFromEditor(defaultPrivileges, validate)
	}
	if err != nil {
		log.ErrorCLI("failed to read vCenter privileges", "error", err)

		retry, err := prompts.ReadBool("Reconfigure privileges", true)
		if err != nil {
			return nil, err
		}
		if retry {
			return readPrivileges(rulePrivileges)
		}
	}

	// Disable propagation for all default privileges.
	// If validating propagation is required, users will need to produce a custom config.
	apiPrivileges := make([]v1alpha1.Privilege, len(privileges))
	for i, p := range privileges {
		apiPrivileges[i] = v1alpha1.Privilege{
			Name:       p,
			Propagated: false,
		}
	}

	return apiPrivileges, nil
}

func readPrivilegesFromEditor(defaultPrivileges string, validate func(string) error) ([]string, error) {
	log.InfoCLI("Configure vCenter privileges")
	time.Sleep(2 * time.Second)

	joinedPrivileges, err := prompts.EditFileValidatedByLine(cfg.VcenterPrivilegePrompt, defaultPrivileges, "\n", validate, 1)
	if err != nil {
		return nil, err
	}
	privileges := strings.Split(joinedPrivileges, "\n")

	return privileges, nil
}

func readPrivilegesFromFile(validate func(string) error) ([]string, error) {
	privilegeFile, err := prompts.ReadFilePath("Privilege file path", "", "Invalid file path", false)
	if err != nil {
		return nil, err
	}

	privilegeBytes, err := os.ReadFile(privilegeFile) //#nosec
	if err != nil {
		return nil, fmt.Errorf("failed to read privilege file: %w", err)
	}
	privileges := strings.Split(string(privilegeBytes), "\n")

	return prompts.FilterLines(privileges, validate)
}

// nolint:dupl
func configurePrivilegeRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string) error {
	log.InfoCLI(`
	Privilege validation ensures that a vSphere user has certain
	privileges with respect to a specific vSphere resource.
	`)

	validatePrivileges, err := prompts.ReadBool("Enable privilege validation", true)
	if err != nil {
		return err
	}
	if !validatePrivileges {
		c.Validator.PrivilegeValidationRules = nil
		return nil
	}

	for i, r := range c.Validator.PrivilegeValidationRules {
		r := r
		if err := readPrivilegeRule(ctx, c, &r, driver, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.PrivilegeValidationRules) == 0 {
		c.Validator.PrivilegeValidationRules = make([]v1alpha1.PrivilegeValidationRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another privilege validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readPrivilegeRule(ctx, c, &v1alpha1.PrivilegeValidationRule{}, driver, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another privilege validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readPrivilegeRule(ctx context.Context, c *components.VspherePluginConfig, r *v1alpha1.PrivilegeValidationRule, driver vsphere.Driver, idx int, ruleNames *[]string) error {
	var err error
	var initMsg string
	reconfigureEntity := true

	if r.Name() != "" {
		reconfigureEntity, err = prompts.ReadBool("Reconfigure entity and privilege set for privilege rule", false)
		if err != nil {
			return err
		}
		if reconfigureEntity {
			initMsg = "The rule's entity and privilege set will be replaced."
		}
	}
	if err := initRule(r, "privilege", initMsg, ruleNames); err != nil {
		return err
	}

	if reconfigureEntity {
		if err := readEntityPrivileges(ctx, c, r, driver); err != nil {
			return err
		}
	}

	if idx == -1 {
		c.Validator.PrivilegeValidationRules = append(c.Validator.PrivilegeValidationRules, *r)
	} else {
		c.Validator.PrivilegeValidationRules[idx] = *r
	}

	return nil
}

func readEntityPrivileges(ctx context.Context, c *components.VspherePluginConfig, r *v1alpha1.PrivilegeValidationRule, driver vsphere.Driver) error {
	var err error

	log.InfoCLI(`Privilege validation rule will be applied for username %s`, c.Validator.Auth.Account.Username)

	entityLabel, err := prompts.Select("Entity Type", entity.Labels)
	if err != nil {
		return err
	}
	r.EntityType = entity.Map[entityLabel]

	r.EntityName, r.ClusterName, err = getEntityAndClusterInfo(ctx, r.EntityType, driver, c.Validator.Datacenter)
	if err != nil {
		return err
	}

	r.Privileges, err = readPrivileges(r.Privileges)
	if err != nil {
		return err
	}

	log.InfoCLI(`
	Privileges are granted to users via permissions, which are scoped to either a user or a group
	principal. Provide a list of group principals to consider during validation. Group principals
	should be of the format, DOMAIN\group-name. It is recommended to provide a group principal for
	each group that the vCenter user is a member of. The user's own principal is included by default.
	`)
	r.GroupPrincipals, err = prompts.ReadTextSlice(
		"Group Principals", strings.Join(r.GroupPrincipals, "\n"), "", "", true,
	)
	if err != nil {
		return err
	}

	return nil
}

// nolint:dupl
func configureResourceRequirementRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string) error {
	log.InfoCLI(`
	Resource requirement validation ensures that sufficient capacity is available within
	a vSphere Datacenter or Cluster for a configurable number of VMs with specific CPU, RAM, and Storage minimums.
	`)

	validateResourceRequirements, err := prompts.ReadBool("Enable resource requirement validation", true)
	if err != nil {
		return err
	}
	if !validateResourceRequirements {
		c.Validator.ComputeResourceRules = nil
		return nil
	}
	for i, r := range c.Validator.ComputeResourceRules {
		r := r
		if err := readResourceRequirementRule(ctx, c, &r, driver, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.ComputeResourceRules) == 0 {
		c.Validator.ComputeResourceRules = make([]v1alpha1.ComputeResourceRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another resource requirement validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readResourceRequirementRule(ctx, c, &v1alpha1.ComputeResourceRule{}, driver, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another resource requirement validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readResourceRequirementRule(ctx context.Context, c *components.VspherePluginConfig, r *v1alpha1.ComputeResourceRule, driver vsphere.Driver, idx int, ruleNames *[]string) error {
	origName := r.Name()

	err := initRule(r, "resource requirement", "", ruleNames)
	if err != nil {
		return err
	}

	if origName == "" {
		entityLabel, err := prompts.Select("Scope", entity.ComputeResourceScopes)
		if err != nil {
			return err
		}
		r.Scope = entity.Map[entityLabel]
	}

	r.EntityName, r.ClusterName, err = getEntityAndClusterInfo(ctx, r.Scope, driver, c.Validator.Datacenter)
	if err != nil {
		return err
	}

	for i, n := range r.NodepoolResourceRequirements {
		n := n
		if err := readResourceRequirements(r, &n, i); err != nil {
			return err
		}
	}
	addNodePool := true
	if r.NodepoolResourceRequirements == nil {
		r.NodepoolResourceRequirements = make([]v1alpha1.NodepoolResourceRequirement, 0)
	} else {
		addNodePool, err = prompts.ReadBool("Add resource requirements for another node pool", false)
		if err != nil {
			return err
		}
	}
	if !addNodePool {
		return nil
	}
	for {
		if err := readResourceRequirements(r, &v1alpha1.NodepoolResourceRequirement{}, -1); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add resource requirements for another node pool", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}

	if idx == -1 {
		c.Validator.ComputeResourceRules = append(c.Validator.ComputeResourceRules, *r)
	} else {
		c.Validator.ComputeResourceRules[idx] = *r
	}

	return nil
}

func readResourceRequirements(r *v1alpha1.ComputeResourceRule, n *v1alpha1.NodepoolResourceRequirement, idx int) error {
	log.InfoCLI("Specify node pool size and resource requirements")

	var err error
	n.Name, err = prompts.ReadText("Node pool name", n.Name, false, -1)
	if err != nil {
		return err
	}

	numNodes := 1
	if n.NumberOfNodes != 0 {
		numNodes = n.NumberOfNodes
	}
	n.NumberOfNodes, err = prompts.ReadInt("Number of nodes in node pool", intToStringDefault(numNodes), 1, 1000)
	if err != nil {
		return err
	}

	n.CPU, err = prompts.ReadTextRegex("CPU requirement per node (Ex: 2.5GHz, 1000MHz)", n.CPU, "Invalid CPU requirement", cfg.CPUReqRegex)
	if err != nil {
		return err
	}
	n.Memory, err = prompts.ReadTextRegex("Memory requirement per node (Ex: 10.5Gi, 40Ti, 512Mi)", n.Memory, "Invalid memory requirement", cfg.MemoryReqRegex)
	if err != nil {
		return err
	}
	n.DiskSpace, err = prompts.ReadTextRegex("Storage request per node (Ex: 1000Mi, 5.5Gi, 2Ti)", n.DiskSpace, "Invalid storage requirement", cfg.DiskReqRegex)
	if err != nil {
		return err
	}

	if idx == -1 {
		r.NodepoolResourceRequirements = append(r.NodepoolResourceRequirements, *n)
	} else {
		r.NodepoolResourceRequirements[idx] = *n
	}

	return nil
}

// nolint:dupl
func configureVsphereTagRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string) error {
	log.InfoCLI(`
	Tag validation ensures that a specific tag is present on a particular vSphere resource.
	`)

	validateTags, err := prompts.ReadBool("Enable tag validation", true)
	if err != nil {
		return err
	}
	if !validateTags {
		c.Validator.TagValidationRules = nil
		return nil
	}

	for i, r := range c.Validator.TagValidationRules {
		r := r
		if err := readVsphereTagRule(ctx, c, &r, driver, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.TagValidationRules) == 0 {
		c.Validator.TagValidationRules = make([]v1alpha1.TagValidationRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another tag validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readVsphereTagRule(ctx, c, &v1alpha1.TagValidationRule{}, driver, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another tag validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readVsphereTagRule(ctx context.Context, c *components.VspherePluginConfig, r *v1alpha1.TagValidationRule, driver vsphere.Driver, idx int, ruleNames *[]string) error {
	origName := r.Name()

	err := initRule(r, "tag", "", ruleNames)
	if err != nil {
		return err
	}

	if origName == "" {
		entityLabel, err := prompts.Select("Entity Type", tags.SupportedEntities)
		if err != nil {
			return err
		}
		r.EntityType = entity.Map[entityLabel]
	}

	r.EntityName, r.ClusterName, err = getEntityAndClusterInfo(ctx, r.EntityType, driver, c.Validator.Datacenter)
	if err != nil {
		return err
	}

	r.Tag, err = prompts.ReadText("Tag", r.Tag, false, -1)
	if err != nil {
		return err
	}

	if idx == -1 {
		c.Validator.TagValidationRules = append(c.Validator.TagValidationRules, *r)
	} else {
		c.Validator.TagValidationRules[idx] = *r
	}

	return nil
}

func getClusterScopedInfo(ctx context.Context, datacenter, entityType string, driver vsphere.Driver) (bool, string, error) {
	var clusterName string

	clusterScopedPrompt := fmt.Sprintf("Is %s cluster scoped", entityType)
	clusterScoped, err := prompts.ReadBool(clusterScopedPrompt, false)
	if err != nil {
		return false, "", err
	}
	if clusterScoped {
		clusterName, err = getClusterName(ctx, datacenter, driver)
		if err != nil {
			return false, "", err
		}
	}

	return clusterScoped, clusterName, nil
}

func getClusterName(ctx context.Context, datacenter string, driver vsphere.Driver) (string, error) {
	clusterList, err := driver.GetClusters(ctx, datacenter)
	if err != nil {
		return "", errors.Wrap(err, "failed to list vSphere clusters")
	}
	clusterName, err := prompts.Select("Cluster", clusterList)
	if err != nil {
		return "", err
	}
	return clusterName, nil
}

func getEntityAndClusterInfo(ctx context.Context, entityType entity.Entity, driver vsphere.Driver, datacenter string) (entityName, clusterName string, err error) {
	switch entityType {
	case entity.Cluster:
		entityName, err = getClusterName(ctx, datacenter, driver)
		if err != nil {
			return "", "", err
		}
		return entityName, entityName, nil
	case entity.Datacenter:
		return handleDatacenterEntity(ctx, driver)
	case entity.Datastore:
		return handleDatastoreEntity(ctx, driver, datacenter)
	case entity.DistributedVirtualPortgroup:
		return handleDVPEntity(ctx, driver, datacenter)
	case entity.DistributedVirtualSwitch:
		return handleDVSEntity(ctx, driver, datacenter)
	case entity.Folder:
		return handleFolderEntity(ctx, driver, datacenter)
	case entity.Host:
		return handleHostEntity(ctx, driver, datacenter)
	case entity.Network:
		return handleNetworkEntity(ctx, driver, datacenter)
	case entity.ResourcePool:
		return handleResourcePoolEntity(ctx, driver, datacenter)
	case entity.VCenterRoot:
		return "", "", nil
	case entity.VirtualApp:
		return handleVAppEntity(ctx, driver)
	case entity.VirtualMachine:
		return handleVMEntity(ctx, driver, datacenter)
	default:
		return "", "", fmt.Errorf("invalid entity type: %s", entityType.String())
	}
}

func handleDatacenterEntity(ctx context.Context, driver vsphere.Driver) (string, string, error) {
	dcList, err := driver.GetDatacenters(ctx)
	if err != nil {
		return "", "", err
	}
	dcName, err := prompts.Select("Datacenter", dcList)
	if err != nil {
		return "", "", err
	}
	return dcName, "", nil
}

func handleDatastoreEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	datastores, err := driver.GetDatastores(ctx, datacenter)
	if err != nil {
		return "", "", err
	}
	datastore, err := prompts.Select("Datastore", datastores)
	if err != nil {
		return "", "", err
	}
	return datastore, "", nil
}

func handleDVPEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	dvpList, err := driver.GetDistributedVirtualPortgroups(ctx, datacenter)
	if err != nil {
		return "", "", err
	}
	dvp, err := prompts.Select("Distributed Port Group", dvpList)
	if err != nil {
		return "", "", err
	}
	return dvp, "", nil
}

func handleDVSEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	dvsList, err := driver.GetDistributedVirtualSwitches(ctx, datacenter)
	if err != nil {
		return "", "", err
	}
	dvs, err := prompts.Select("Distributed Switch", dvsList)
	if err != nil {
		return "", "", err
	}
	return dvs, "", nil
}

func handleFolderEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	folderList, err := driver.GetVMFolders(ctx, datacenter)
	if err != nil {
		return "", "", err
	}
	folderName, err := prompts.Select("Folder", folderList)
	if err != nil {
		return "", "", err
	}
	return folderName, "", nil
}

func handleHostEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	_, clusterName, err := getClusterScopedInfo(ctx, datacenter, entity.Host.String(), driver)
	if err != nil {
		return "", "", err
	}
	hosts, err := driver.GetHostSystems(ctx, datacenter, clusterName)
	if err != nil {
		return "", "", err
	}
	hostList := make([]string, 0, len(hosts))
	for _, host := range hosts {
		hostList = append(hostList, host.Name)
	}
	hostName, err := prompts.Select("ESXi Host", hostList)
	if err != nil {
		return "", "", err
	}
	return hostName, clusterName, nil
}

func handleNetworkEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	networkList, err := driver.GetNetworks(ctx, datacenter)
	if err != nil {
		return "", "", err
	}
	network, err := prompts.Select("Network", networkList)
	if err != nil {
		return "", "", err
	}
	return network, "", nil
}

func handleResourcePoolEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	allResourcePools := make([]*object.ResourcePool, 0)

	clusterScoped, clusterName, err := getClusterScopedInfo(ctx, datacenter, entity.ResourcePool.String(), driver)
	if err != nil {
		return "", "", err
	}
	if clusterScoped {
		// Get cluster resourcepools
		clusterRPs, err := driver.GetResourcePools(ctx, datacenter, clusterName)
		if err != nil {
			return "", "", err
		}
		allResourcePools = append(allResourcePools, clusterRPs...)
	}

	defaultRPs, err := driver.GetResourcePools(ctx, datacenter, "")
	if err != nil {
		return "", "", err
	}
	allResourcePools = append(allResourcePools, defaultRPs...)

	rpChoiceList := make([]prompts.ChoiceItem, 0, len(allResourcePools))
	rpClusterMapping := make(map[string]string)
	for _, rp := range allResourcePools {
		rpCluster := strings.Split(rp.InventoryPath, "/")[3]
		rpChoiceName := rp.Name()
		if rp.Name() == "Resources" {
			// if Cluster scoped has been chosen, handle resource pools not belonging to the chosen cluster
			if clusterScoped && (rpCluster != clusterName) {
				continue
			}
			rpChoiceName = fmt.Sprintf("Default Resource Pool (Cluster: %s)", rpCluster)
		}
		choiceItem := prompts.ChoiceItem{
			ID:   rp.Name(),
			Name: rpChoiceName,
		}
		rpClusterMapping[rpChoiceName] = rpCluster
		rpChoiceList = append(rpChoiceList, choiceItem)
	}

	choice, err := prompts.SelectID("Resource Pool", rpChoiceList)
	if err != nil {
		return "", "", err
	}

	return choice.ID, rpClusterMapping[choice.Name], nil
}

func handleVAppEntity(ctx context.Context, driver vsphere.Driver) (string, string, error) {
	vApps, err := driver.GetVApps(ctx)
	if err != nil {
		return "", "", err
	}
	vAppList := make([]string, 0, len(vApps))
	for _, vapp := range vApps {
		vAppList = append(vAppList, vapp.Name)
	}
	vAppName, err := prompts.Select("Virtual App", vAppList)
	if err != nil {
		return "", "", err
	}
	return vAppName, "", nil
}

func handleVMEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, string, error) {
	clusterScoped, clusterName, err := getClusterScopedInfo(ctx, datacenter, entity.VirtualMachine.String(), driver)
	if err != nil {
		return "", "", err
	}

	hostClusterMapping := make(map[string]string)
	if clusterScoped {
		// This way because govmomi just doesn't have a way to cheaply determine what cluster a VM belongs to :')
		hostClusterMapping, err = driver.GetHostClusterMapping(ctx)
		if err != nil {
			return "", "", err
		}
	}

	vms, err := driver.GetVMs(ctx, datacenter)
	if err != nil {
		return "", "", err
	}

	vmList := make([]string, 0, len(vms))
	for _, vm := range vms {
		if clusterScoped {
			hostLookupKey := vm.Host
			if hostClusterMapping[hostLookupKey] != clusterName {
				continue
			}
		}
		vmList = append(vmList, vm.Name)
	}
	sort.Slice(vmList, func(i, j int) bool {
		return vmList[i] < vmList[j]
	})
	vmName, err := prompts.Select("Virtual Machine", vmList)
	if err != nil {
		return "", "", err
	}

	return vmName, clusterName, nil
}
