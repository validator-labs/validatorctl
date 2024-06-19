package validator

import (
	"k8s.io/client-go/kubernetes"

	"github.com/validator-labs/validator-plugin-kubescape/api/v1alpha1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

func readKubescapePlugin(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	var err error

	c := vc.KubescapePlugin

	if err := readHelmRelease(cfg.ValidatorPluginKubescape, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return err
	}

	ruleNames := make([]string, 0)

	if c.Validator.Namespace, err = prompts.ReadText("Kubescape Namespace", "kubescape", false, 9999); err != nil {
		return err
	}

	if err := configureSeverityLimitRule(c, &ruleNames); err != nil {
		return err
	}

	if err := configureFlagCVERule(c, &ruleNames); err != nil {
		return err
	}

	if err := configureIgnoreCVERule(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	return nil
}

func configureSeverityLimitRule(c *components.KubescapePluginConfig, ruleName *[]string) error {
	log.InfoCLI(`
	Severity Limit Rule sets a threshold for vulnerabilities found within the cluster.
	`)

	validateSeverity, err := prompts.ReadBool("Enable Severity Limit Validation", true)
	if err != nil {
		return err
	}

	if !validateSeverity {
		return nil
	}

	c.Validator.SeverityLimitRule = v1alpha1.SeverityLimitRule{}

	var critical, high, medium, low, negligible, unknown int

	if critical, err = prompts.ReadInt("Limit for Critical Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.Critical = &critical

	if high, err = prompts.ReadInt("Limit for High Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.High = &high

	if medium, err = prompts.ReadInt("Limit for Medium Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.Medium = &medium

	if low, err = prompts.ReadInt("Limit for Low Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.Low = &low

	if negligible, err = prompts.ReadInt("Limit for Neglible Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.Negligible = &negligible

	if unknown, err = prompts.ReadInt("Limit for Unknown Vulnerabilities", "", 0, 9999999); err != nil {
		return err
	}
	c.Validator.SeverityLimitRule.Unknown = &unknown

	return nil
}

func configureFlagCVERule(c *components.KubescapePluginConfig, ruleName *[]string) error {
	// forever iterate
	log.InfoCLI(`
	Flag CVE Rules ensures that specified vulnerabilities are flagged within the cluster.
	`)

	enableFlagCVERule, err := prompts.ReadBool("Enable CVE Flag Rule", true)
	if err != nil {
		return err
	}

	if !enableFlagCVERule {
		return nil
	}

	c.Validator.FlaggedCVERule = []v1alpha1.FlaggedCVE{}

	for {
		cve, err := prompts.ReadText("Vulnerability", "", false, 32)
		if err != nil {
			return err
		}

		c.Validator.FlaggedCVERule = append(c.Validator.FlaggedCVERule, v1alpha1.FlaggedCVE(cve))

		add, err := prompts.ReadBool("Add another Vulnerability Flag rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}

	return nil
}

func configureIgnoreCVERule(c *components.KubescapePluginConfig, ruleName *[]string) error {

	// forever iterate
	log.InfoCLI(`
	Ignore specified CVE in any validation rule
	`)

	enableFlagCVERule, err := prompts.ReadBool("Enable ignore CVE list", true)
	if err != nil {
		return err
	}

	if !enableFlagCVERule {
		return nil
	}

	c.Validator.IgnoredCVERule = []string{}

	for {
		cve, err := prompts.ReadText("Vulnerability", "", false, 32)
		if err != nil {
			return err
		}

		c.Validator.IgnoredCVERule = append(c.Validator.IgnoredCVERule, cve)

		add, err := prompts.ReadBool("Add another Vulnerability to ignore", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}

	return nil
}
