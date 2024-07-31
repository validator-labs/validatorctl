package validator

import (
	"reflect"

	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	vpnetworkapi "github.com/validator-labs/validator-plugin-network/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

type networkRule interface {
	*vpnetworkapi.DNSRule | *vpnetworkapi.ICMPRule | *vpnetworkapi.IPRangeRule | *vpnetworkapi.MTURule | *vpnetworkapi.TCPConnRule
}

func readNetworkPlugin(vc *components.ValidatorConfig, _ kubernetes.Interface) error {
	c := vc.NetworkPlugin

	if err := readHelmRelease(cfg.ValidatorPluginNetwork, vc, c.Release); err != nil {
		return err
	}

	log.Header("Network Configuration")
	ruleNames := make([]string, 0)

	if err := configureDNSRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureIcmpRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureIPRangeRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureMtuRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureTCPConnRules(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}
	return nil
}

// nolint:dupl
func configureDNSRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	DNS validation rules ensure that DNS lookups succeed for the specified host(s).
	`)

	validateDNS, err := prompts.ReadBool("Enable DNS validation", true)
	if err != nil {
		return err
	}
	if !validateDNS {
		c.Validator.DNSRules = nil
		return nil
	}
	for i, r := range c.Validator.DNSRules {
		r := r
		if err := readDNSRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.DNSRules == nil {
		c.Validator.DNSRules = make([]vpnetworkapi.DNSRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another DNS rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readDNSRule(c, &vpnetworkapi.DNSRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another DNS rule", false)
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
func configureIcmpRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	ICMP validation rules ensure that ICMP pings succeed for the specified host(s).
	`)

	validateIcmp, err := prompts.ReadBool("Enable ICMP validation", true)
	if err != nil {
		return err
	}
	if !validateIcmp {
		c.Validator.ICMPRules = nil
		return nil
	}
	for i, r := range c.Validator.ICMPRules {
		r := r
		if err := readIcmpRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.ICMPRules == nil {
		c.Validator.ICMPRules = make([]vpnetworkapi.ICMPRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another ICMP rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIcmpRule(c, &vpnetworkapi.ICMPRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another ICMP rule", false)
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
func configureIPRangeRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	IP range validation rules ensure that a specific range of
	IP addresses (starting IP + next N IPs) are unallocated.
	`)

	validateIPRange, err := prompts.ReadBool("Enable IP range validation", true)
	if err != nil {
		return err
	}
	if !validateIPRange {
		c.Validator.IPRangeRules = nil
		return nil
	}
	for i, r := range c.Validator.IPRangeRules {
		r := r
		if err := readIPRangeRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.IPRangeRules == nil {
		c.Validator.IPRangeRules = make([]vpnetworkapi.IPRangeRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another IP range rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIPRangeRule(c, &vpnetworkapi.IPRangeRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another IP range rule", false)
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
func configureMtuRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	MTU validation rules ensure that the default NIC has an
	MTU of at least X, where X is the provided MTU.
	`)

	validateMTU, err := prompts.ReadBool("Enable MTU validation", true)
	if err != nil {
		return err
	}
	if !validateMTU {
		c.Validator.MTURules = nil
		return nil
	}
	for i, r := range c.Validator.MTURules {
		r := r
		if err := readMtuRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.MTURules == nil {
		c.Validator.MTURules = make([]vpnetworkapi.MTURule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another MTU rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readMtuRule(c, &vpnetworkapi.MTURule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another MTU rule", false)
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
func configureTCPConnRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	TCP connection validation rules ensure that TCP connections
	to the specified host(s) and port(s) are successful.
	`)

	validateTCP, err := prompts.ReadBool("Enable TCP connection validation", true)
	if err != nil {
		return err
	}
	if !validateTCP {
		c.Validator.TCPConnRules = nil
		return nil
	}
	for i, r := range c.Validator.TCPConnRules {
		r := r
		if err := readTCPConnRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.TCPConnRules == nil {
		c.Validator.TCPConnRules = make([]vpnetworkapi.TCPConnRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another TCP connection rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readTCPConnRule(c, &vpnetworkapi.TCPConnRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another TCP connection rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func initNetworkRule[R networkRule](r R, ruleType string, ruleNames *[]string) error {
	name := reflect.ValueOf(r).Elem().FieldByName("RuleName").String()
	if name != "" {
		log.InfoCLI("Reconfiguring %s rule: %s", ruleType, name)
		*ruleNames = append(*ruleNames, name)
	} else {
		name, err := getRuleName(ruleNames)
		if err != nil {
			return err
		}
		reflect.ValueOf(r).Elem().FieldByName("RuleName").Set(reflect.ValueOf(name))
	}
	return nil
}

func readDNSRule(c *components.NetworkPluginConfig, r *vpnetworkapi.DNSRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "DNS", ruleNames)
	if err != nil {
		return err
	}
	r.Host, err = prompts.ReadText("Host to resolve", r.Host, false, -1)
	if err != nil {
		return err
	}
	r.Server, err = prompts.ReadText("Nameserver (optional)", r.Server, true, -1)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.DNSRules = append(c.Validator.DNSRules, *r)
	} else {
		c.Validator.DNSRules[idx] = *r
	}
	return nil
}

func readIcmpRule(c *components.NetworkPluginConfig, r *vpnetworkapi.ICMPRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "ICMP", ruleNames)
	if err != nil {
		return err
	}
	r.Host, err = prompts.ReadText("Host to ping", r.Host, false, -1)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.ICMPRules = append(c.Validator.ICMPRules, *r)
	} else {
		c.Validator.ICMPRules[idx] = *r
	}
	return nil
}

func readIPRangeRule(c *components.NetworkPluginConfig, r *vpnetworkapi.IPRangeRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "IP range", ruleNames)
	if err != nil {
		return err
	}
	r.StartIP, err = prompts.ReadIPs("First IPv4 in range", r.StartIP, "invalid IPv4", false, 1)
	if err != nil {
		return err
	}
	r.Length, err = prompts.ReadInt("Length of IPv4 range", intToStringDefault(r.Length), 1, -1)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.IPRangeRules = append(c.Validator.IPRangeRules, *r)
	} else {
		c.Validator.IPRangeRules[idx] = *r
	}
	return nil
}

func readMtuRule(c *components.NetworkPluginConfig, r *vpnetworkapi.MTURule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "MTU", ruleNames)
	if err != nil {
		return err
	}
	r.Host, err = prompts.ReadText("Host to ping", r.Host, false, -1)
	if err != nil {
		return err
	}
	r.MTU, err = prompts.ReadInt("Minimum MTU", intToStringDefault(r.MTU), 1, -1)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.MTURules = append(c.Validator.MTURules, *r)
	} else {
		c.Validator.MTURules[idx] = *r
	}
	return nil
}

func readTCPConnRule(c *components.NetworkPluginConfig, r *vpnetworkapi.TCPConnRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "TCP connection", ruleNames)
	if err != nil {
		return err
	}
	r.Host, err = prompts.ReadText("Host to connect to", r.Host, false, -1)
	if err != nil {
		return err
	}
	addPorts := true
	for i, p := range r.Ports {
		port, err := prompts.ReadInt("Port", intToStringDefault(p), 1, -1)
		if err != nil {
			return err
		}
		r.Ports[i] = port
	}
	if r.Ports == nil {
		r.Ports = make([]int, 0)
	} else {
		addPorts, err = prompts.ReadBool("Add another port", false)
		if err != nil {
			return err
		}
	}
	if addPorts {
		for {
			port, err := prompts.ReadInt("Port", "", 1, -1)
			if err != nil {
				return err
			}
			r.Ports = append(r.Ports, port)
			add, err := prompts.ReadBool("Add another port", false)
			if err != nil {
				return err
			}
			if !add {
				break
			}
		}
	}

	r.InsecureSkipTLSVerify, err = prompts.ReadBool("Skip TLS certificate verification", true)
	if err != nil {
		return err
	}

	r.Timeout, err = prompts.ReadInt("Timeout in seconds", "5", 1, -1)
	if err != nil {
		return err
	}

	if idx == -1 {
		c.Validator.TCPConnRules = append(c.Validator.TCPConnRules, *r)
	} else {
		c.Validator.TCPConnRules[idx] = *r
	}
	return nil
}
