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
	"github.com/vmware/govmomi/object"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"
	"github.com/validator-labs/validatorctl/pkg/utils/embed"
)

var (
	vSphereSecretName  = "vsphere-creds" //#nosec G101
	dataCenter         = "Datacenter"
	vsphereEntityTypes []string
)

func init() {
	for _, v := range cfg.ValidatorPluginVsphereEntityMap {
		vsphereEntityTypes = append(vsphereEntityTypes, v)
	}
}

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

	vSphereCloudDriver, err := clouds.GetVSphereDriver(c.Validator.Auth.CloudAccount)
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

	if err := configureNtpRules(ctx, c, vSphereCloudDriver, &ruleNames); err != nil {
		return err
	}
	if err := configureRolePrivilegeRules(c, &ruleNames, vSphereCloudDriver); err != nil {
		return err
	}
	if err := configureEntityPrivilegeRules(ctx, c, vSphereCloudDriver, &ruleNames, vSphereCloudDriver); err != nil {
		return err
	}
	if err := configureResourceRequirementRules(ctx, c, vSphereCloudDriver, &ruleNames); err != nil {
		return err
	}
	if err := configureVsphereTagRules(ctx, c, vSphereCloudDriver, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	return nil
}

func readVsphereCredentials(c *components.VspherePluginConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	var err error
	c.Validator.Auth.CloudAccount = &vsphere.CloudAccount{}
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
		if err := clouds.ReadVsphereAccountProps(c.Validator.Auth.CloudAccount); err != nil {
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
		c.Validator.Auth.CloudAccount.VcenterServer = string(secret.Data["vcenterServer"])
		c.Validator.Auth.CloudAccount.Username = string(secret.Data["username"])
		c.Validator.Auth.CloudAccount.Password = string(secret.Data["password"])
		c.Validator.Auth.CloudAccount.Insecure = insecure
	}

	// validate vSphere version
	vSphereCloudDriver, err := clouds.GetVSphereDriver(c.Validator.Auth.CloudAccount)
	if err != nil {
		return err
	}
	if err := vSphereCloudDriver.ValidateVsphereVersion(cfg.ValidatorVsphereVersionConstraint); err != nil {
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
		_, r.ClusterName, err = getClusterScopedInfo(ctx, c.Validator.Datacenter, cfg.ValidatorVsphereEntityHost, driver)
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
	hosts, err := driver.GetVSphereHostSystems(ctx, datacenter, clusterName)
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

func configureRolePrivilegeRules(c *components.VspherePluginConfig, ruleNames *[]string, vSphereCloudDriver vsphere.Driver) error {
	log.InfoCLI(`
	Role privilege validation ensures that a vSphere user has a
	specific set of root-level vSphere privileges.
	`)

	validateRolePrivileges, err := prompts.ReadBool("Enable role privilege validation", true)
	if err != nil {
		return err
	}
	if !validateRolePrivileges {
		c.VsphereRolePrivilegeRules = nil
		c.Validator.RolePrivilegeValidationRules = nil
		return nil
	}

	isAdmin, err := vSphereCloudDriver.IsAdminAccount(context.Background())
	if err != nil {
		return err
	}

	for i, r := range c.VsphereRolePrivilegeRules {
		r := r
		if err := readRolePrivilegeRule(c, &r, i, ruleNames, isAdmin); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.VsphereRolePrivilegeRules) == 0 {
		c.VsphereRolePrivilegeRules = make([]components.VsphereRolePrivilegeRule, 0)
		c.Validator.RolePrivilegeValidationRules = make([]v1alpha1.GenericRolePrivilegeValidationRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another role privilege validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readRolePrivilegeRule(c, &components.VsphereRolePrivilegeRule{}, -1, ruleNames, isAdmin); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another role privilege validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readRolePrivilegeRule(c *components.VspherePluginConfig, r *components.VsphereRolePrivilegeRule, idx int, ruleNames *[]string, isAdmin bool) error {
	var err error
	var initMsg string
	reconfigurePrivileges := true

	if r.Name() != "" {
		reconfigurePrivileges, err = prompts.ReadBool("Reconfigure privilege set for role privilege rule", false)
		if err != nil {
			return err
		}
		if reconfigurePrivileges {
			initMsg = "The rule's vSphere privilege set will be replaced."
		}
	}
	if err := initRule(r, "role privilege", initMsg, ruleNames); err != nil {
		return err
	}

	if isAdmin {
		r.Username, err = prompts.ReadTextRegex("vSphere username for privilege validation", r.Username, "Invalid vSphere username", cfg.VSphereUsernameRegex)
		if err != nil {
			return err
		}
	} else {
		log.InfoCLI(`Privilege validation rule will be applied for username %s`, c.Validator.Auth.CloudAccount.Username)
		r.Username = c.Validator.Auth.CloudAccount.Username
	}

	if reconfigurePrivileges {
		r.Privileges, err = readPrivileges(r.Privileges)
		if err != nil {
			return err
		}
	}

	if idx == -1 {
		c.VsphereRolePrivilegeRules = append(c.VsphereRolePrivilegeRules, *r)
		c.Validator.RolePrivilegeValidationRules = append(c.Validator.RolePrivilegeValidationRules, r.GenericRolePrivilegeValidationRule)
	} else {
		c.VsphereRolePrivilegeRules[idx] = *r
		c.Validator.RolePrivilegeValidationRules[idx] = r.GenericRolePrivilegeValidationRule
	}

	return nil
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
		if strings.HasPrefix(input, "#") {
			return nil
		}
		if !slices.Contains(privileges, strings.TrimSpace(input)) {
			log.ErrorCLI("failed to read vCenter privileges", "invalidPrivilege", input)
			return prompts.ErrValidationFailed
		}
		return nil
	}

	return strings.Join(privileges, "\n"), validate, nil
}

func readPrivileges(rulePrivileges []string) ([]string, error) {
	defaultPrivileges, validate, err := loadPrivileges(cfg.ValidatorVspherePrivilegeFile)
	if err != nil {
		return nil, err
	}
	if len(rulePrivileges) > 0 {
		defaultPrivileges = strings.Join(rulePrivileges, "\n")
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
	if inputType == cfg.LocalFilepath {
		return readPrivilegesFromFile(validate)
	}

	return readPrivilegesFromEditor(defaultPrivileges, validate)
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
	for _, p := range privileges {
		if err := validate(p); err != nil {
			retry, err := prompts.ReadBool("Reconfigure privileges", true)
			if err != nil {
				return nil, err
			}
			if retry {
				return readPrivilegesFromFile(validate)
			}
			return nil, err
		}
	}
	return privileges, nil
}

func configureEntityPrivilegeRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string, vSphereCloudDriver vsphere.Driver) error {
	log.InfoCLI(`
	Entity privilege validation ensures that a vSphere user has certain
	privileges with respect to a specific vSphere resource.
	`)

	validateEntityPrivileges, err := prompts.ReadBool("Enable entity privilege validation", true)
	if err != nil {
		return err
	}
	if !validateEntityPrivileges {
		c.VsphereEntityPrivilegeRules = nil
		c.Validator.EntityPrivilegeValidationRules = nil
		return nil
	}

	isAdmin, err := vSphereCloudDriver.IsAdminAccount(context.Background())
	if err != nil {
		return err
	}

	for i, r := range c.VsphereEntityPrivilegeRules {
		r := r
		if err := readEntityPrivilegeRule(ctx, c, &r, driver, i, ruleNames, isAdmin); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.EntityPrivilegeValidationRules) == 0 {
		c.VsphereEntityPrivilegeRules = make([]components.VsphereEntityPrivilegeRule, 0)
		c.Validator.EntityPrivilegeValidationRules = make([]v1alpha1.EntityPrivilegeValidationRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another entity privilege validation rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readEntityPrivilegeRule(ctx, c, &components.VsphereEntityPrivilegeRule{}, driver, -1, ruleNames, isAdmin); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another entity privilege validation rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readEntityPrivilegeRule(ctx context.Context, c *components.VspherePluginConfig, r *components.VsphereEntityPrivilegeRule, driver vsphere.Driver, idx int, ruleNames *[]string, isAdmin bool) error {
	var err error
	var initMsg string
	reconfigureEntity := true

	if r.Name() != "" {
		reconfigureEntity, err = prompts.ReadBool("Reconfigure entity and privilege set for entity privilege rule", false)
		if err != nil {
			return err
		}
		if reconfigureEntity {
			initMsg = "The rule's entity and privilege set will be replaced."
		}
	}
	if err := initRule(r, "entity privilege", initMsg, ruleNames); err != nil {
		return err
	}

	if err := readEntityPrivileges(ctx, c, r, driver, isAdmin, reconfigureEntity); err != nil {
		return err
	}

	if idx == -1 {
		c.VsphereEntityPrivilegeRules = append(c.VsphereEntityPrivilegeRules, *r)
		c.Validator.EntityPrivilegeValidationRules = append(c.Validator.EntityPrivilegeValidationRules, r.EntityPrivilegeValidationRule)
	} else {
		c.VsphereEntityPrivilegeRules[idx] = *r
		c.Validator.EntityPrivilegeValidationRules[idx] = r.EntityPrivilegeValidationRule
	}

	return nil
}

func readEntityPrivileges(ctx context.Context, c *components.VspherePluginConfig, r *components.VsphereEntityPrivilegeRule, driver vsphere.Driver, isAdmin, reconfigureEntity bool) error {
	var err error

	if isAdmin {
		r.Username, err = prompts.ReadTextRegex("vSphere username to validate entity privileges for", r.Username, "Invalid vSphere username", cfg.VSphereUsernameRegex)
		if err != nil {
			return err
		}
	} else {
		log.InfoCLI(`Privilege validation rule will be applied for username %s`, c.Validator.Auth.CloudAccount.Username)
		r.Username = c.Validator.Auth.CloudAccount.Username
	}

	if reconfigureEntity {
		r.EntityType, r.EntityName, r.ClusterName, err = getEntityInfo(ctx, "", "Entity Type", c.Validator.Datacenter, cfg.ValidatorPluginVsphereEntities, driver)
		if err != nil {
			return err
		}
		r.Privileges, err = readPrivileges(r.Privileges)
		if err != nil {
			return err
		}
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
	err := initRule(r, "resource requirement", "", ruleNames)
	if err != nil {
		return err
	}

	r.Scope, r.EntityName, r.ClusterName, err = getEntityInfo(ctx, r.Scope, "Scope", c.Validator.Datacenter, cfg.ValidatorPluginVsphereDeploymentDestination, driver)
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

func configureVsphereTagRules(ctx context.Context, c *components.VspherePluginConfig, driver vsphere.Driver, ruleNames *[]string) error {
	log.InfoCLI(`
	Tag validation ensures that a specific tag is present on a particular vSphere resource.
	`)

	validateTags, err := prompts.ReadBool("Enable tag validation", true)
	if err != nil {
		return err
	}
	if !validateTags {
		c.VsphereTagRules = nil
		c.Validator.TagValidationRules = nil
		return nil
	}

	for i, r := range c.VsphereTagRules {
		r := r
		if err := readVsphereTagRule(ctx, c, &r, driver, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.VsphereTagRules) == 0 {
		c.VsphereTagRules = make([]components.VsphereTagRule, 0)
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
		if err := readVsphereTagRule(ctx, c, &components.VsphereTagRule{}, driver, -1, ruleNames); err != nil {
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

func readVsphereTagRule(ctx context.Context, c *components.VspherePluginConfig, r *components.VsphereTagRule, driver vsphere.Driver, idx int, ruleNames *[]string) error {
	err := initRule(r, "tag", "", ruleNames)
	if err != nil {
		return err
	}

	if err := readCustomVsphereTagRule(ctx, c, r, driver); err != nil {
		return err
	}

	if idx == -1 {
		c.VsphereTagRules = append(c.VsphereTagRules, *r)
		c.Validator.TagValidationRules = append(c.Validator.TagValidationRules, r.TagValidationRule)
	} else {
		c.VsphereTagRules[idx] = *r
		c.Validator.TagValidationRules[idx] = r.TagValidationRule
	}

	return nil
}

func readCustomVsphereTagRule(ctx context.Context, c *components.VspherePluginConfig, r *components.VsphereTagRule, driver vsphere.Driver) error {
	var err error
	r.EntityType, r.EntityName, r.ClusterName, err = getEntityInfo(ctx, r.EntityType, "Entity Type", c.Validator.Datacenter, cfg.ValidatorPluginVsphereEntities, driver)
	if err != nil {
		return err
	}
	r.Tag, err = prompts.ReadText("Tag", r.Tag, false, -1)
	if err != nil {
		return err
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
	clusterList, err := driver.GetVSphereClusters(ctx, datacenter)
	if err != nil {
		return "", errors.Wrap(err, "failed to list vSphere clusters")
	}
	clusterName, err := prompts.Select("Cluster", clusterList)
	if err != nil {
		return "", err
	}
	return clusterName, nil
}

func getEntityInfo(ctx context.Context, entityType, entityTypePrompt, datacenter string, entityTypesList []string, driver vsphere.Driver) (string, string, string, error) {
	var err error
	if entityType == "" {
		entityType, err = prompts.Select(entityTypePrompt, entityTypesList)
		if err != nil {
			return "", "", "", err
		}
	}
	entityName, clusterName, err := getEntityAndClusterInfo(ctx, entityType, driver, datacenter)
	if err != nil {
		return "", "", "", err
	}

	// Convert pretty entity type to entity type compatible with the vSphere validator plugin
	validatorEntityType, ok := cfg.ValidatorPluginVsphereEntityMap[entityType]
	if !ok {
		// entity type will already be converted if we're reconfiguring a rule
		if !slices.Contains(vsphereEntityTypes, entityType) {
			return "", "", "", fmt.Errorf("invalid entity type: %s", entityType)
		}
	}

	return validatorEntityType, entityName, clusterName, err
}

func getEntityAndClusterInfo(ctx context.Context, entityType string, driver vsphere.Driver, datacenter string) (entityName, clusterName string, err error) {
	switch entityType {
	case cfg.ValidatorVsphereEntityCluster, "cluster":
		entityName, err = getClusterName(ctx, datacenter, driver)
		if err != nil {
			return "", "", err
		}
		return entityName, entityName, nil
	case cfg.ValidatorVsphereEntityDatacenter, "datacenter":
		entityName, err = handleDatacenterEntity(ctx, driver)
		if err != nil {
			return "", "", err
		}
		return entityName, clusterName, nil
	case cfg.ValidatorVsphereEntityFolder, "folder":
		entityName, err = handleFolderEntity(ctx, driver, datacenter)
		if err != nil {
			return "", "", err
		}
		return entityName, clusterName, nil
	case cfg.ValidatorVsphereEntityHost, "host":
		return handleHostEntity(ctx, driver, datacenter, entityType)
	case cfg.ValidatorVsphereEntityResourcePool, "resourcepool":
		return handleResourcePoolEntity(ctx, driver, datacenter, entityType)
	case cfg.ValidatorVsphereEntityVirtualApp, "vapp":
		entityName, err = handleVAppEntity(ctx, driver)
		if err != nil {
			return "", "", err
		}
		return entityName, "", nil
	case cfg.ValidatorVsphereEntityVirtualMachine, "vm":
		return handleVMEntity(ctx, driver, datacenter, entityType)
	default:
		return "", "", fmt.Errorf("invalid entity type: %s", entityType)
	}
}

func handleDatacenterEntity(ctx context.Context, driver vsphere.Driver) (string, error) {
	dcList, err := driver.GetVSphereDatacenters(ctx)
	if err != nil {
		return "", err
	}
	dcName, err := prompts.Select("Datacenter", dcList)
	if err != nil {
		return "", err
	}
	return dcName, nil
}

func handleFolderEntity(ctx context.Context, driver vsphere.Driver, datacenter string) (string, error) {
	folderList, err := driver.GetVSphereVMFolders(ctx, datacenter)
	if err != nil {
		return "", err
	}
	folderName, err := prompts.Select("Folder", folderList)
	if err != nil {
		return "", err
	}
	return folderName, nil
}

func handleHostEntity(ctx context.Context, driver vsphere.Driver, datacenter, entityType string) (string, string, error) {
	_, clusterName, err := getClusterScopedInfo(ctx, datacenter, entityType, driver)
	if err != nil {
		return "", "", err
	}
	hosts, err := driver.GetVSphereHostSystems(ctx, datacenter, clusterName)
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

func handleResourcePoolEntity(ctx context.Context, driver vsphere.Driver, datacenter string, entityType string) (string, string, error) {
	allResourcePools := make([]*object.ResourcePool, 0)

	clusterScoped, clusterName, err := getClusterScopedInfo(ctx, datacenter, entityType, driver)
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

func handleVAppEntity(ctx context.Context, driver vsphere.Driver) (string, error) {
	vApps, err := driver.GetVapps(ctx)
	if err != nil {
		return "", err
	}
	vAppList := make([]string, 0, len(vApps))
	for _, vapp := range vApps {
		vAppList = append(vAppList, vapp.Name)
	}
	vAppName, err := prompts.Select("Virtual App", vAppList)
	if err != nil {
		return "", err
	}
	return vAppName, nil
}

func handleVMEntity(ctx context.Context, driver vsphere.Driver, datacenter string, entityType string) (string, string, error) {
	clusterScoped, clusterName, err := getClusterScopedInfo(ctx, datacenter, entityType, driver)
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

	vms, err := driver.GetVSphereVms(ctx, datacenter)
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
