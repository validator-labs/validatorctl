package validator

import (
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"

	network "github.com/validator-labs/validator-plugin-network/api/v1alpha1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

func readNetworkPlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, _ kubernetes.Interface) error {
	c := vc.NetworkPlugin

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginNetwork, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}

	return nil
}

func readNetworkPluginRules(vc *components.ValidatorConfig, tc *cfg.TaskConfig, kClient kubernetes.Interface) error {
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
	if err := configureHTTPFileRules(c, tc, &ruleNames, kClient); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	if len(c.Validator.TCPConnRules) > 0 || len(c.Validator.HTTPFileRules) > 0 {
		if err := readCACertificates(c, tc); err != nil {
			return err
		}
	}

	return nil
}

// readCACertificates reads CA certificates for TLS verification.
// Certs are always overwritten / reconfiguration intentionally unsupported.
func readCACertificates(c *components.NetworkPluginConfig, tc *cfg.TaskConfig) error {
	if err := readLocalCACertificates(c); err != nil {
		return err
	}
	if !tc.Direct {
		if err := readSecretCACertificates(c); err != nil {
			return err
		}
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
	if len(c.Validator.DNSRules) == 0 {
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
	if len(c.Validator.ICMPRules) == 0 {
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
	if len(c.Validator.IPRangeRules) == 0 {
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
	if len(c.Validator.MTURules) == 0 {
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
	if len(c.Validator.TCPConnRules) == 0 {
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
func configureHTTPFileRules(c *components.NetworkPluginConfig, tc *cfg.TaskConfig, ruleNames *[]string, kClient kubernetes.Interface) error {
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
		if err := readHTTPFileRule(c, tc, &r, i, ruleNames, kClient); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.HTTPFileRules) == 0 {
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
		if err := readHTTPFileRule(c, tc, &network.HTTPFileRule{}, -1, ruleNames, kClient); err != nil {
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

func readDNSRule(c *components.NetworkPluginConfig, r *network.DNSRule, idx int, ruleNames *[]string) error {
	err := initRule(r, "DNS", "", ruleNames)
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
	err := initRule(r, "ICMP", "", ruleNames)
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
	err := initRule(r, "IP range", "", ruleNames)
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
	err := initRule(r, "MTU", "", ruleNames)
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
	err := initRule(r, "TCP connection", "", ruleNames)
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

func readHTTPFileRule(c *components.NetworkPluginConfig, tc *cfg.TaskConfig, r *network.HTTPFileRule, idx int, ruleNames *[]string, kClient kubernetes.Interface) error {
	err := initRule(r, "HTTP file", "", ruleNames)
	if err != nil {
		return err
	}
	r.Paths, err = prompts.ReadURLSlice("Paths", strings.Join(r.Paths, "\n"), "Invalid path; must be a valid URL", false)
	if err != nil {
		return err
	}
	configureAuth, err := prompts.ReadBool("Configure basic authentication for this HTTP file rule", false)
	if err != nil {
		return err
	}
	if configureAuth {
		// TODO
		// if direct,
		// else
		if err := readHTTPFileRuleCredentials(c, tc, r, idx, kClient); err != nil {
			return err
		}
	} else {
		c.AddDummyHTTPFileAuth()
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

// TODO: this should be the indirect way of doing it
func readHTTPFileRuleCredentials(c *components.NetworkPluginConfig, tc *cfg.TaskConfig, r *network.HTTPFileRule, idx int, kClient kubernetes.Interface) error {
	var err error
	var username, password string
	createSecret := true

	if r.Auth.SecretRef == nil {
		r.Auth.SecretRef = &network.BasicAuthSecretReference{}
	}
	if idx == -1 {
		// preallocate space if appending a new rule
		idx = len(c.HTTPFileAuths)
		c.AddDummyHTTPFileAuth()
	}
	if len(c.HTTPFileAuths) > idx {
		username = c.HTTPFileAuths[idx][0]
		password = c.HTTPFileAuths[idx][1]
	}

	if kClient != nil {
		log.InfoCLI(`
	Either specify basic authentication credentials or provide the name of a
	secret in the target K8s cluster's %s namespace and its keys that map to
	basic authentication credentials.
	`, cfg.Validator,
		)
		createSecret, err = prompts.ReadBool("Create HTTP file credential secret", false)
		if err != nil {
			return err
		}
	}

	if createSecret {
		if !tc.Direct {
			r.Auth.SecretRef.Name, err = prompts.ReadK8sName("Secret name for basic authentication", r.Auth.SecretRef.Name, false)
			if err != nil {
				return err
			}
		}
		r.Auth.SecretRef.UsernameKey = "username"
		r.Auth.SecretRef.PasswordKey = "password"

		username, password, err = prompts.ReadBasicCreds("Username", "Password", username, password, false, false)
		if err != nil {
			return err
		}
		c.HTTPFileAuths[idx] = []string{username, password}
	} else {
		usernameKey, err := prompts.ReadText("Key for username in secret", r.Auth.SecretRef.UsernameKey, false, -1)
		if err != nil {
			return err
		}
		passwordKey, err := prompts.ReadText("Key for password in secret", r.Auth.SecretRef.PasswordKey, false, -1)
		if err != nil {
			return err
		}
		secret, err := services.ReadSecret(kClient, cfg.Validator, false, []string{usernameKey, passwordKey})
		if err != nil {
			return err
		}
		r.Auth.SecretRef.Name = secret.Name
		r.Auth.SecretRef.UsernameKey = usernameKey
		r.Auth.SecretRef.PasswordKey = passwordKey
	}

	return nil
}
