package validator

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	network "github.com/validator-labs/validator-plugin-network/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

type networkRule interface {
	*network.DNSRule | *network.ICMPRule | *network.IPRangeRule | *network.MTURule | *network.TCPConnRule | *network.HTTPFileRule
}

func readNetworkPluginInstall(vc *components.ValidatorConfig, _ kubernetes.Interface) error {
	c := vc.NetworkPlugin

	if err := readHelmRelease(cfg.ValidatorPluginNetwork, vc, c.Release); err != nil {
		return fmt.Errorf("failed to read Helm release: %w", err)
	}

	return nil
}

func readNetworkPluginRules(vc *components.ValidatorConfig, _ kubernetes.Interface) error {
	log.Header("Network Plugin Rule Configuration")

	c := vc.NetworkPlugin
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
	if err := configureHTTPFileRules(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	if len(c.Validator.TCPConnRules) > 0 || len(c.Validator.HTTPFileRules) > 0 {
		if err := readCACertificates(c); err != nil {
			return err
		}
	}

	return nil
}

// readCACertificates reads CA certificates for TLS verification.
// Certs are always overwritten / reconfiguration intentionally unsupported.
func readCACertificates(c *components.NetworkPluginConfig) error {
	if err := readLocalCACertificates(c); err != nil {
		return err
	}
	if err := readSecretCACertificates(c); err != nil {
		return err
	}
	return nil
}

func readLocalCACertificates(c *components.NetworkPluginConfig) error {
	c.Validator.CACerts.Certs = make([]network.Certificate, 0)

	addCerts, err := prompts.ReadBool("Add CA certificate(s) for TLS verification", true)
	if err != nil {
		return err
	}
	if !addCerts {
		return nil
	}
	for {
		_, _, caBytes, err := prompts.ReadCACert("CA certificate for TLS verification", "", "")
		if err != nil {
			return err
		}
		c.Validator.CACerts.Certs = append(c.Validator.CACerts.Certs, network.Certificate(caBytes))

		add, err := prompts.ReadBool("Add another CA certificate", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readSecretCACertificates(c *components.NetworkPluginConfig) error {
	c.Validator.CACerts.SecretRefs = make([]network.CASecretReference, 0)

	addCerts, err := prompts.ReadBool("Add CA certificate secret reference(s) for TLS verification", true)
	if err != nil {
		return err
	}
	if !addCerts {
		return nil
	}
	for {
		ref := network.CASecretReference{}
		ref.Name, err = prompts.ReadK8sName("CA secret name", "", false)
		if err != nil {
			return err
		}
		ref.Key, err = prompts.ReadText("Key for CA certificate in secret", "ca.crt", false, -1)
		if err != nil {
			return err
		}
		c.Validator.CACerts.SecretRefs = append(c.Validator.CACerts.SecretRefs, ref)

		add, err := prompts.ReadBool("Add another CA certificate secret reference", false)
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
		c.Validator.DNSRules = make([]network.DNSRule, 0)
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
		if err := readDNSRule(c, &network.DNSRule{}, -1, ruleNames); err != nil {
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
		c.Validator.ICMPRules = make([]network.ICMPRule, 0)
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
		if err := readIcmpRule(c, &network.ICMPRule{}, -1, ruleNames); err != nil {
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
		c.Validator.IPRangeRules = make([]network.IPRangeRule, 0)
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
		if err := readIPRangeRule(c, &network.IPRangeRule{}, -1, ruleNames); err != nil {
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
		c.Validator.MTURules = make([]network.MTURule, 0)
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
		if err := readMtuRule(c, &network.MTURule{}, -1, ruleNames); err != nil {
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
		c.Validator.TCPConnRules = make([]network.TCPConnRule, 0)
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
		if err := readTCPConnRule(c, &network.TCPConnRule{}, -1, ruleNames); err != nil {
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

// nolint:dupl
func configureHTTPFileRules(c *components.NetworkPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	HTTP file rules ensure that specific files are accessible via HTTP HEAD requests,
	optionally with basic authentication.
	`)

	validateFiles, err := prompts.ReadBool("Enable HTTP file validation", true)
	if err != nil {
		return err
	}
	if !validateFiles {
		c.Validator.HTTPFileRules = nil
		return nil
	}
	for i, r := range c.Validator.HTTPFileRules {
		r := r
		if err := readHTTPFileRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.HTTPFileRules == nil {
		c.Validator.HTTPFileRules = make([]network.HTTPFileRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another HTTP file rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readHTTPFileRule(c, &network.HTTPFileRule{}, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another HTTP file rule", false)
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

func readDNSRule(c *components.NetworkPluginConfig, r *network.DNSRule, idx int, ruleNames *[]string) error {
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

func readIcmpRule(c *components.NetworkPluginConfig, r *network.ICMPRule, idx int, ruleNames *[]string) error {
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

func readIPRangeRule(c *components.NetworkPluginConfig, r *network.IPRangeRule, idx int, ruleNames *[]string) error {
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

func readMtuRule(c *components.NetworkPluginConfig, r *network.MTURule, idx int, ruleNames *[]string) error {
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

func readTCPConnRule(c *components.NetworkPluginConfig, r *network.TCPConnRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "TCP connection", ruleNames)
	if err != nil {
		return err
	}
	r.Host, err = prompts.ReadText("Host to connect to", r.Host, false, -1)
	if err != nil {
		return err
	}
	r.Ports, err = prompts.ReadIntSlice("Port", intsToStringDefault(r.Ports), false)
	if err != nil {
		return err
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

func readHTTPFileRule(c *components.NetworkPluginConfig, r *network.HTTPFileRule, idx int, ruleNames *[]string) error {
	err := initNetworkRule(r, "HTTP file", ruleNames)
	if err != nil {
		return err
	}
	r.Paths, err = prompts.ReadURLSlice("Paths", strings.Join(r.Paths, "\n"), "Invalid path; must be a valid URL", false)
	if err != nil {
		return err
	}
	if r.AuthSecretRef == nil {
		r.AuthSecretRef = &network.BasicAuthSecretReference{}
	}
	r.AuthSecretRef.Name, err = prompts.ReadK8sName("Secret name for basic authentication", r.AuthSecretRef.Name, true)
	if err != nil {
		return err
	}
	if r.AuthSecretRef.Name != "" {
		r.AuthSecretRef.UsernameKey, err = prompts.ReadText("Key for username in secret", r.AuthSecretRef.UsernameKey, false, -1)
		if err != nil {
			return err
		}
		r.AuthSecretRef.PasswordKey, err = prompts.ReadText("Key for password in secret", r.AuthSecretRef.PasswordKey, false, -1)
		if err != nil {
			return err
		}
	}
	r.InsecureSkipTLSVerify, err = prompts.ReadBool("Skip TLS certificate verification", true)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.HTTPFileRules = append(c.Validator.HTTPFileRules, *r)
	} else {
		c.Validator.HTTPFileRules[idx] = *r
	}
	return nil
}
