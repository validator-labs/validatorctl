package validator

import (
	"fmt"
	"reflect"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/services/clouds"

	vpmaasapi "github.com/validator-labs/validator-plugin-maas/api/v1alpha1"
)

var (
	maasSecretName = "maas-creds"
	maasTokenKey   = "MAAS_API_KEY"
)

type maasRule interface {
	*vpmaasapi.ResourceAvailabilityRule | *vpmaasapi.ImageRule | *vpmaasapi.InternalDNSRule | *vpmaasapi.UpstreamDNSRule
}

func initMaasRule[R maasRule](r R, ruleType string, ruleNames *[]string) error {
	name := reflect.ValueOf(r).Elem().FieldByName("Name").String()
	if name != "" {
		// not all maasRules have a Name field, for now we can create a unique name for them
		if name == "<invalid Value>" {
			name = ruleType + " - " + time.Now().Format("20060102T150405.000000000")
		}
		log.InfoCLI("\nReconfiguring %s validation rule: %s", ruleType, name)
		*ruleNames = append(*ruleNames, name)
	} else {
		name, err := getRuleName(ruleNames)
		if err != nil {
			return err
		}
		reflect.ValueOf(r).Elem().FieldByName("Name").Set(reflect.ValueOf(name))
	}
	return nil
}

func readMaasPlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	c := vc.MaasPlugin

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginMaas, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}
	if err := readMaasCredentials(c, tc, k8sClient); err != nil {
		return fmt.Errorf("failed to read MAAS credentials: %w", err)
	}
	return nil
}

// nolint:dupl
func readMaasPluginRules(vc *components.ValidatorConfig, _ *cfg.TaskConfig, _ kubernetes.Interface) error {
	log.Header("MAAS Plugin Rule Configuration")
	c := vc.MaasPlugin
	ruleNames := make([]string, 0)

	if err := configureMaasResourceRules(c, &ruleNames); err != nil {
		return err
	}

	if err := configureMaasImageRules(c, &ruleNames); err != nil {
		return err
	}

	if err := configureMaasInternalDNSRules(c, &ruleNames); err != nil {
		return err
	}

	if err := configureMaasUpstreamDNSRules(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}
	return nil
}

// nolint:dupl
func readMaasCredentials(c *components.MaasPluginConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	var err error

	// always create MAAS credential secret if creating a new kind cluster
	createSecret := true

	if k8sClient != nil {
		log.InfoCLI(`
		Either specify MAAS credentials or provide the name of a secret in the target K8s cluster's %s namespace.
		`, cfg.Validator,
		)
		createSecret, err = prompts.ReadBool("Create MAAS credential secret", true)
		if err != nil {
			return fmt.Errorf("failed to create MAAS credential secret: %w", err)
		}
	}

	if createSecret {
		if c.Validator.Auth.SecretName != "" {
			maasSecretName = c.Validator.Auth.SecretName
		}

		if !tc.Direct {
			c.Validator.Auth.SecretName, err = prompts.ReadK8sName("MAAS credentials secret name", maasSecretName, false)
			if err != nil {
				return fmt.Errorf("failed to prompt for text for MAAS credentials secret name: %w", err)
			}
			c.Validator.Auth.TokenKey, err = prompts.ReadText("MAAS API token key", maasTokenKey, false, -1)
			if err != nil {
				return fmt.Errorf("failed to prompt for text for MAAS API token key: %w", err)
			}
		}

		if err := clouds.ReadMaasClientProps(c); err != nil {
			return err
		}

	} else {
		c.Validator.Auth.TokenKey, err = prompts.ReadText("MAAS API token key", maasTokenKey, false, -1)
		if err != nil {
			return fmt.Errorf("failed to prompt for text for MAAS API token key: %w", err)
		}
		secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, []string{c.Validator.Auth.TokenKey})
		if err != nil {
			return fmt.Errorf("failed to read k8s Secret: %w", err)
		}
		c.Validator.Auth.SecretName = secret.Name
		c.MaasAPIToken = string(secret.Data[c.Validator.Auth.TokenKey])
	}

	return nil
}

// nolint:dupl
func configureMaasResourceRules(c *components.MaasPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	Resource Availability validation checks that the required number of machines
	matching certain criteria are "Ready" for use in an availability zone. 
	Each availability zone should have no more than 1 rule configured.
	`)

	validateResources, err := prompts.ReadBool("Enable Resource Availability validation", true)
	if err != nil {
		return err
	}

	if !validateResources {
		c.Validator.ResourceAvailabilityRules = nil
		return nil
	}

	for i, r := range c.Validator.ResourceAvailabilityRules {
		r := r
		if err := readMaasResourceRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}

	addRules := true
	if c.Validator.ResourceAvailabilityRules == nil {
		c.Validator.ResourceAvailabilityRules = make([]vpmaasapi.ResourceAvailabilityRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another Resource Availability rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readMaasResourceRule(c, &vpmaasapi.ResourceAvailabilityRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another Resource Availability rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// nolint:dupl
func readMaasResourceRule(c *components.MaasPluginConfig, r *vpmaasapi.ResourceAvailabilityRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpmaasapi.ResourceAvailabilityRule{
			Name:      "",
			AZ:        "",
			Resources: []vpmaasapi.Resource{},
		}
	}

	err := initMaasRule(r, "Resource Availability", ruleNames)
	if err != nil {
		return err
	}

	if r.AZ == "" {
		az, err := prompts.ReadText("Availability Zone", "az1", false, -1)
		if err != nil {
			return err
		}
		r.AZ = az
	}

	addResources := true

	for addResources {
		resource, err := readMaasResource(c)
		if err != nil {
			return err
		}

		r.Resources = append(r.Resources, resource)
		addResources, err = prompts.ReadBool("Add another resource", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.ResourceAvailabilityRules = append(c.Validator.ResourceAvailabilityRules, *r)
	} else {
		c.Validator.ResourceAvailabilityRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func readMaasResource(c *components.MaasPluginConfig) (vpmaasapi.Resource, error) {
	res := vpmaasapi.Resource{}

	numMachines, err := prompts.ReadInt("Minimum number of machines", "1", 1, -1)
	if err != nil {
		return res, err
	}
	res.NumMachines = numMachines

	numCPU, err := prompts.ReadInt("Minimum CPU cores per machine", "4", 1, -1)
	if err != nil {
		return res, err
	}
	res.NumCPU = numCPU

	ram, err := prompts.ReadInt("Minimum RAM in GB", "16", 1, -1)
	if err != nil {
		return res, err
	}
	res.RAM = ram

	disk, err := prompts.ReadInt("Minimum Disk capacity in GB", "256", 1, -1)
	if err != nil {
		return res, err
	}
	res.Disk = disk

	resourcePools, err := clouds.GetMaasResourcePools(c)
	if err != nil {
		return res, err
	}
	pool, err := prompts.Select("Machine pool", resourcePools)
	if err != nil {
		return res, err
	}
	if pool != "" {
		res.Pool = pool
	}

	tags, err := prompts.ReadTextSlice("Machine tags", "", "Invalid tag values", "", true)
	if err != nil {
		return res, err
	}
	if len(tags) > 0 {
		res.Tags = tags
	}

	return res, nil
}

// nolint:dupl
func configureMaasImageRules(c *components.MaasPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	OS Image validation ensures that the specified images are available for use.
	`)

	validateImages, err := prompts.ReadBool("Enable OS Image validation", true)
	if err != nil {
		return err
	}

	if !validateImages {
		c.Validator.ImageRules = nil
		return nil
	}
	for i, r := range c.Validator.ImageRules {
		r := r
		if err := readMaasImageRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}

	addRules := true
	if c.Validator.ImageRules == nil {
		c.Validator.ImageRules = make([]vpmaasapi.ImageRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another OS Image rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readMaasImageRule(c, &vpmaasapi.ImageRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another OS Image rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// nolint:dupl
func readMaasImageRule(c *components.MaasPluginConfig, r *vpmaasapi.ImageRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpmaasapi.ImageRule{
			Name:   "",
			Images: []vpmaasapi.Image{},
		}
	}

	err := initMaasRule(r, "OS Image", ruleNames)
	if err != nil {
		return err
	}

	addImages := true

	for addImages {
		image, err := readMaasImage()
		if err != nil {
			return err
		}

		r.Images = append(r.Images, image)
		addImages, err = prompts.ReadBool("Add another OS Image", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.ImageRules = append(c.Validator.ImageRules, *r)
	} else {
		c.Validator.ImageRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func readMaasImage() (vpmaasapi.Image, error) {
	img := vpmaasapi.Image{}

	name, err := prompts.ReadText("Image name (standard or custom)", "ubuntu/jammy", false, -1)
	if err != nil {
		return img, err
	}
	img.Name = name

	arch, err := prompts.ReadText("Architecture formatted as <platform>/<release> (e.g amd64/ga-22.04, arm64/hwe-20.04-edge)", "amd64/ga-22.04", false, -1)
	if err != nil {
		return img, err
	}
	img.Architecture = arch

	return img, nil
}

// nolint:dupl
func configureMaasInternalDNSRules(c *components.MaasPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	Internal DNS validation ensures that the expected DNS setting are configured inside your MAAS cluster.
	`)

	validateIDNS, err := prompts.ReadBool("Enable Internal DNS validation", true)
	if err != nil {
		return err
	}

	if !validateIDNS {
		c.Validator.InternalDNSRules = nil
		return nil
	}
	for i, r := range c.Validator.InternalDNSRules {
		r := r
		if err := readMaasInternalDNSRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}

	addRules := true
	if c.Validator.InternalDNSRules == nil {
		c.Validator.InternalDNSRules = make([]vpmaasapi.InternalDNSRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another Internal DNS rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readMaasInternalDNSRule(c, &vpmaasapi.InternalDNSRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another Internal DNS rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// nolint:dupl
func readMaasInternalDNSRule(c *components.MaasPluginConfig, r *vpmaasapi.InternalDNSRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpmaasapi.InternalDNSRule{
			MaasDomain:   "",
			DNSResources: []vpmaasapi.DNSResource{},
		}
	}

	err := initMaasRule(r, "Internal DNS", ruleNames)
	if err != nil {
		return err
	}

	if r.MaasDomain == "" {
		domain, err := prompts.ReadText("MAAS Domain", "maas.io", false, -1)
		if err != nil {
			return err
		}
		r.MaasDomain = domain
	}

	addResources := true

	for addResources {
		resource, err := readMaasDNSResource()
		if err != nil {
			return err
		}

		r.DNSResources = append(r.DNSResources, resource)
		addResources, err = prompts.ReadBool("Add another DNS resource", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.InternalDNSRules = append(c.Validator.InternalDNSRules, *r)
	} else {
		c.Validator.InternalDNSRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func readMaasDNSResource() (vpmaasapi.DNSResource, error) {
	res := vpmaasapi.DNSResource{}

	fqdn, err := prompts.ReadDomains("FQDN", "www.maas.io", "must be a valid FQDN", false, 1)
	if err != nil {
		return res, err
	}
	res.FQDN = fqdn

	addRecord := true

	for addRecord {
		record, err := readMaasDNSRecord()
		if err != nil {
			return res, err
		}

		res.DNSRecords = append(res.DNSRecords, record)
		addRecord, err = prompts.ReadBool("Add another DNS record", false)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// nolint:dupl
func readMaasDNSRecord() (vpmaasapi.DNSRecord, error) {
	log.InfoCLI("Add a DNS Resource Record")
	rec := vpmaasapi.DNSRecord{}

	ip, err := prompts.ReadIPs("IP", "10.10.10.10", "IP should be a valid IPv4", false, 1)
	if err != nil {
		return rec, err
	}
	rec.IP = ip

	recType, err := prompts.Select("Record type", cfg.DNSRecordTypes)
	if err != nil {
		return rec, err
	}
	rec.Type = recType

	ttl, err := prompts.ReadInt("TTL in seconds (optional, enter -1 to skip)", "-1", -1, -1)
	if err != nil {
		return rec, err
	}
	rec.TTL = ttl

	return rec, nil
}

// nolint:dupl
func configureMaasUpstreamDNSRules(c *components.MaasPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	Upstream DNS validation ensures that the expected number of upstream DNS server are configured.
	`)

	validateUDNS, err := prompts.ReadBool("Enable Upstream DNS validation", true)
	if err != nil {
		return err
	}

	if !validateUDNS {
		c.Validator.UpstreamDNSRules = nil
		return nil
	}
	for i, r := range c.Validator.UpstreamDNSRules {
		r := r
		if err := readMaasUpstreamDNSRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}

	addRules := true
	if c.Validator.UpstreamDNSRules == nil {
		c.Validator.UpstreamDNSRules = make([]vpmaasapi.UpstreamDNSRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another Upstream DNS rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readMaasUpstreamDNSRule(c, &vpmaasapi.UpstreamDNSRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another Upstream DNS rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// nolint:dupl
func readMaasUpstreamDNSRule(c *components.MaasPluginConfig, r *vpmaasapi.UpstreamDNSRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpmaasapi.UpstreamDNSRule{
			Name:          "",
			NumDNSServers: 0,
		}
	}

	err := initMaasRule(r, "Upstream DNS", ruleNames)
	if err != nil {
		return err
	}

	if r.NumDNSServers <= 0 {
		num, err := prompts.ReadInt("Expected number of DNS servers", "1", 1, -1)
		if err != nil {
			return err
		}
		r.NumDNSServers = num
	}

	if idx == -1 {
		c.Validator.UpstreamDNSRules = append(c.Validator.UpstreamDNSRules, *r)
	} else {
		c.Validator.UpstreamDNSRules[idx] = *r
	}

	return nil
}
