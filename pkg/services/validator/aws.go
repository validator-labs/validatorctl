package validator

import (
	"reflect"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	vpawsapi "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

var (
	region             = "us-east-1"
	iamRole            = "SpectroCloudRole"
	stsDurationSeconds = "3600"
	awsSecretName      = "aws-creds"
)

type awsRule interface {
	*vpawsapi.ServiceQuotaRule | *vpawsapi.TagRule
}

func readAwsPlugin(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	var err error
	c := vc.AWSPlugin

	if err := readHelmRelease(cfg.ValidatorPluginAws, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return err
	}
	if err := readAwsCredentials(c, k8sClient); err != nil {
		return errors.Wrap(err, "failed to read AWS credentials")
	}

	if c.Validator.DefaultRegion != "" {
		region = c.Validator.DefaultRegion
	}
	c.Validator.DefaultRegion, err = prompts.ReadText("Default AWS region", region, false, -1)
	if err != nil {
		return err
	}

	ruleNames := make([]string, 0)

	if err := configureIamRule(c); err != nil {
		return err
	}
	if err := configureServiceQuotaRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureAwsTagRules(c, &ruleNames); err != nil {
		return err
	}

	if !c.IamCheck.Enabled && c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}
	return nil
}

func configureIamRule(c *components.AWSPluginConfig) error {
	log.InfoCLI(`
	AWS IAM validation ensures that a certain IAM Role has every permission
	specified in one of Spectro Cloud's predefined IAM policies.

	Different permission sets are required to deploy clusters via Spectro Cloud,
	depending on placement type (static vs. dynamic) and other factors.
	`)

	var err error
	c.IamCheck.Enabled, err = prompts.ReadBool("Enable IAM validation", true)
	if err != nil {
		return err
	}
	if !c.IamCheck.Enabled {
		return nil
	}
	if c.IamCheck.IamRoleName != "" {
		iamRole = c.IamCheck.IamRoleName
	}
	c.IamCheck.IamRoleName, err = prompts.ReadText("IAM Role Name", iamRole, false, -1)
	if err != nil {
		return err
	}
	checkType, err := prompts.Select("IAM check type", cfg.ValidatorIamCheckTypes())
	if err != nil {
		return err
	}
	c.IamCheck.Type = cfg.IamCheckType(checkType)
	return nil
}

func configureServiceQuotaRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS service quota validation ensures that the usage for specific AWS resource quotas
	remains below a specific buffer.
	`)

	validateQuotas, err := prompts.ReadBool("Enable Service Quota validation", true)
	if err != nil {
		return err
	}
	if !validateQuotas {
		c.Validator.ServiceQuotaRules = nil
		return nil
	}
	for i, r := range c.Validator.ServiceQuotaRules {
		r := r
		if err := readServiceQuotaRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.ServiceQuotaRules == nil {
		c.Validator.ServiceQuotaRules = make([]vpawsapi.ServiceQuotaRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another service quota rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readServiceQuotaRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another service quota rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func configureAwsTagRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS tag validation ensures that specific tags are set on selected AWS resources.
	`)

	validateTags, err := prompts.ReadBool("Enable Tag validation", true)
	if err != nil {
		return err
	}
	if !validateTags {
		c.Validator.TagRules = nil
		return nil
	}
	for i, r := range c.Validator.TagRules {
		r := r
		switch r.ResourceType {
		case "subnet":
			if err := readSubnetTagRule(c, &r, i, ruleNames); err != nil {
				return err
			}
		}
	}
	addRules := true
	if c.Validator.TagRules == nil {
		c.Validator.TagRules = make([]vpawsapi.TagRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another subnet tag rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		resourceType, err := prompts.Select("AWS resource type", []string{"subnet"})
		if err != nil {
			return err
		}
		switch resourceType {
		case "subnet":
			if err := readSubnetTagRule(c, nil, -1, ruleNames); err != nil {
				return err
			}
		}
		add, err := prompts.ReadBool("Add another tag rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readAwsCredentials(c *components.AWSPluginConfig, k8sClient kubernetes.Interface) error {
	var err error
	c.Validator.Auth.Implicit, err = prompts.ReadBool("Use implicit AWS auth", true)
	if err != nil {
		return err
	}
	if c.Validator.Auth.Implicit {
		c.ServiceAccountName, err = services.ReadServiceAccount(k8sClient, cfg.Validator)
		if err != nil {
			return err
		}
		return nil
	}

	// always create AWS credential secret if creating a new kind cluster
	createSecret := true

	if k8sClient != nil {
		log.InfoCLI(`
	Either specify AWS credentials or provide the name of a secret in the target K8s cluster's %s namespace.
	If using an existing secret, it must contain the following keys: %+v.
	`, cfg.Validator, cfg.ValidatorPluginAwsKeys,
		)
		createSecret, err = prompts.ReadBool("Create AWS credential secret", true)
		if err != nil {
			return err
		}
	}

	if createSecret {
		if c.Validator.Auth.SecretName != "" {
			awsSecretName = c.Validator.Auth.SecretName
		}
		c.Validator.Auth.SecretName, err = prompts.ReadText("AWS credentials secret name", awsSecretName, false, -1)
		if err != nil {
			return err
		}
		c.AccessKeyId, err = prompts.ReadPassword("AWS Access Key ID", c.AccessKeyId, false, -1)
		if err != nil {
			return err
		}
		c.SecretAccessKey, err = prompts.ReadPassword("AWS Secret Access Key", c.SecretAccessKey, false, -1)
		if err != nil {
			return err
		}
		c.SessionToken, err = prompts.ReadPassword("AWS Session Token", c.SessionToken, true, -1)
		if err != nil {
			return err
		}
	} else {
		secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, cfg.ValidatorPluginAwsKeys)
		if err != nil {
			return err
		}
		c.Validator.Auth.SecretName = secret.Name
		c.AccessKeyId = string(secret.Data["AWS_ACCESS_KEY_ID"])
		c.SecretAccessKey = string(secret.Data["AWS_SECRET_ACCESS_KEY"])
		c.SessionToken = string(secret.Data["AWS_SESSION_TOKEN"])
	}

	useSTS, err := prompts.ReadBool("Configure Credentials for STS", false)
	if err != nil {
		return err
	}
	if useSTS {
		c.Validator.Auth.StsAuth = &vpawsapi.AwsSTSAuth{}
		c.Validator.Auth.StsAuth.RoleArn, err = prompts.ReadText("AWS STS Role ARN", c.Validator.Auth.StsAuth.RoleArn, false, -1)
		if err != nil {
			return err
		}
		c.Validator.Auth.StsAuth.RoleSessionName, err = prompts.ReadText("AWS STS Session Name", c.Validator.Auth.StsAuth.RoleSessionName, false, -1)
		if err != nil {
			return err
		}
		duration := stsDurationSeconds
		if c.Validator.Auth.StsAuth.DurationSeconds != 0 {
			duration = intToStringDefault(c.Validator.Auth.StsAuth.DurationSeconds)
		}
		c.Validator.Auth.StsAuth.DurationSeconds, err = prompts.ReadInt("AWS STS Session Duration", duration, 900, 43200)
		if err != nil {
			return err
		}
	}

	return nil
}

func initAwsRule[R awsRule](r R, ruleType string, ruleNames *[]string) error {
	name := reflect.ValueOf(r).Elem().FieldByName("Name").String()
	if name != "" {
		log.InfoCLI("Reconfiguring %s rule: %s", ruleType, name)
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

func readServiceQuotaRule(c *components.AWSPluginConfig, r *vpawsapi.ServiceQuotaRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpawsapi.ServiceQuotaRule{
			ServiceQuotas: []vpawsapi.ServiceQuota{
				{
					Name:   "",
					Buffer: 1,
				},
			},
		}
	}
	err := initAwsRule(r, "service quota", ruleNames)
	if err != nil {
		return err
	}
	if r.ServiceCode == "" {
		quota, err := prompts.SelectID("Service quota type", cfg.ValidatorPluginAwsServiceQuotas)
		if err != nil {
			return err
		}
		r.ServiceCode = quota.ID
		r.ServiceQuotas[0].Name = quota.Name
	}
	if r.Region != "" {
		region = r.Region
	}
	r.Region, err = prompts.ReadText("AWS Region", region, false, -1)
	if err != nil {
		return err
	}
	r.ServiceQuotas[0].Buffer, err = prompts.ReadInt("Buffer", intToStringDefault(r.ServiceQuotas[0].Buffer), 1, -1)
	if err != nil {
		return err
	}
	if idx == -1 {
		c.Validator.ServiceQuotaRules = append(c.Validator.ServiceQuotaRules, *r)
	} else {
		c.Validator.ServiceQuotaRules[idx] = *r
	}
	return nil
}

func readSubnetTagRule(c *components.AWSPluginConfig, r *vpawsapi.TagRule, idx int, ruleNames *[]string) error {
	for {
		if r == nil {
			r = &vpawsapi.TagRule{
				ResourceType: "subnet",
				ARNs:         make([]string, 0),
			}
		}
		err := initAwsRule(r, "subnet tag", ruleNames)
		if err != nil {
			return err
		}
		if r.Region != "" {
			region = r.Region
		}
		r.Region, err = prompts.ReadText("AWS Region", region, false, -1)
		if err != nil {
			return err
		}
		r.Key, err = prompts.ReadText("Subnet Tag key", r.Key, false, -1)
		if err != nil {
			return err
		}
		r.ExpectedValue, err = prompts.ReadText("Subnet Tag value", r.ExpectedValue, false, -1)
		if err != nil {
			return err
		}
		for i, a := range r.ARNs {
			arn, err := prompts.ReadText("Subnet ARN", a, false, -1)
			if err != nil {
				return err
			}
			r.ARNs[i] = arn
		}
		addArns := true
		if len(r.ARNs) > 0 {
			addArns, err = prompts.ReadBool("Add another subnet ARN", false)
			if err != nil {
				return err
			}
		}
		if addArns {
			for {
				arn, err := prompts.ReadText("Subnet ARN", "", false, -1)
				if err != nil {
					return err
				}
				r.ARNs = append(r.ARNs, arn)

				add, err := prompts.ReadBool("Add another subnet ARN", false)
				if err != nil {
					return err
				}
				if !add {
					break
				}
			}
		}
		if idx == -1 {
			c.Validator.TagRules = append(c.Validator.TagRules, *r)
		} else {
			c.Validator.TagRules[idx] = *r
			break
		}
		add, err := prompts.ReadBool("Add another subnet tag rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}
