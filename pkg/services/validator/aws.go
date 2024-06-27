package validator

import (
	"os"
	"reflect"
	"time"

	"emperror.dev/errors"
	awspolicy "github.com/L30Bola/aws-policy"
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
	stsDurationSeconds = "3600"
	awsSecretName      = "aws-creds"
)

type awsRule interface {
	*vpawsapi.ServiceQuotaRule | *vpawsapi.TagRule | *vpawsapi.IamRoleRule | *vpawsapi.IamUserRule | *vpawsapi.IamGroupRule | *vpawsapi.IamPolicyRule
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

	if err := configureIamRoleRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureIamUserRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureIamGroupRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureIamPolicyRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureServiceQuotaRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureAwsTagRules(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}
	return nil
}

// nolint:dupl
func configureIamRoleRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS IAM Role validation ensures that specified IAM Roles have every permission
	specified in the provided policy document(s).
	`)

	validateRoles, err := prompts.ReadBool("Enable Iam Role validation", true)
	if err != nil {
		return err
	}
	if !validateRoles {
		c.Validator.IamRoleRules = nil
		return nil
	}
	for i, r := range c.Validator.IamRoleRules {
		r := r
		if err := readIamRoleRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.IamRoleRules == nil {
		c.Validator.IamRoleRules = make([]vpawsapi.IamRoleRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another role rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIamRoleRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another role rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readIamRoleRule(c *components.AWSPluginConfig, r *vpawsapi.IamRoleRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpawsapi.IamRoleRule{
			IamRoleName: "",
			Policies:    []vpawsapi.PolicyDocument{},
		}
	}
	err := initAwsRule(r, "IAM role", ruleNames)
	if err != nil {
		return err
	}
	if r.IamRoleName == "" {
		roleName, err := prompts.ReadText("IAM Role Name", "", false, -1)
		if err != nil {
			return err
		}
		r.IamRoleName = roleName
	}

	addPolicies := true
	for addPolicies {
		policyDoc, err := readIamPolicy()
		if err != nil {
			return err
		}

		r.Policies = append(r.Policies, policyDoc)
		addPolicies, err = prompts.ReadBool("Add another policy document", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.IamRoleRules = append(c.Validator.IamRoleRules, *r)
	} else {
		c.Validator.IamRoleRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func configureIamUserRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS IAM User validation ensures that specified IAM Users have every permission
	specified in the provided policy document(s).
	`)

	validateUsers, err := prompts.ReadBool("Enable Iam User validation", true)
	if err != nil {
		return err
	}
	if !validateUsers {
		c.Validator.IamUserRules = nil
		return nil
	}
	for i, r := range c.Validator.IamUserRules {
		r := r
		if err := readIamUserRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.IamUserRules == nil {
		c.Validator.IamUserRules = make([]vpawsapi.IamUserRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another user rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIamUserRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another user rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readIamUserRule(c *components.AWSPluginConfig, r *vpawsapi.IamUserRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpawsapi.IamUserRule{
			IamUserName: "",
			Policies:    []vpawsapi.PolicyDocument{},
		}
	}
	err := initAwsRule(r, "IAM user", ruleNames)
	if err != nil {
		return err
	}
	if r.IamUserName == "" {
		userName, err := prompts.ReadText("IAM User Name", "", false, -1)
		if err != nil {
			return err
		}
		r.IamUserName = userName
	}

	addPolicies := true
	for addPolicies {
		policyDoc, err := readIamPolicy()
		if err != nil {
			return err
		}

		r.Policies = append(r.Policies, policyDoc)
		addPolicies, err = prompts.ReadBool("Add another policy document", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.IamUserRules = append(c.Validator.IamUserRules, *r)
	} else {
		c.Validator.IamUserRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func configureIamGroupRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS IAM Group validation ensures that specified IAM Groups have every permission
	specified in the provided policy document(s).
	`)

	validateGroups, err := prompts.ReadBool("Enable Iam Group validation", true)
	if err != nil {
		return err
	}
	if !validateGroups {
		c.Validator.IamGroupRules = nil
		return nil
	}
	for i, r := range c.Validator.IamGroupRules {
		r := r
		if err := readIamGroupRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.IamGroupRules == nil {
		c.Validator.IamGroupRules = make([]vpawsapi.IamGroupRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another group rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIamGroupRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another group rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readIamGroupRule(c *components.AWSPluginConfig, r *vpawsapi.IamGroupRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpawsapi.IamGroupRule{
			IamGroupName: "",
			Policies:     []vpawsapi.PolicyDocument{},
		}
	}
	err := initAwsRule(r, "IAM group", ruleNames)
	if err != nil {
		return err
	}
	if r.IamGroupName == "" {
		groupName, err := prompts.ReadText("IAM Group Name", "", false, -1)
		if err != nil {
			return err
		}
		r.IamGroupName = groupName
	}

	addPolicies := true
	for addPolicies {
		policyDoc, err := readIamPolicy()
		if err != nil {
			return err
		}

		r.Policies = append(r.Policies, policyDoc)
		addPolicies, err = prompts.ReadBool("Add another policy document", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.IamGroupRules = append(c.Validator.IamGroupRules, *r)
	} else {
		c.Validator.IamGroupRules[idx] = *r
	}
	return nil
}

// nolint:dupl
func configureIamPolicyRules(c *components.AWSPluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	AWS IAM Policy validation ensures that specified IAM Policies have every permission
	specified in the provided policy document(s).
	`)

	validatePolicies, err := prompts.ReadBool("Enable Iam Policy validation", true)
	if err != nil {
		return err
	}
	if !validatePolicies {
		c.Validator.IamPolicyRules = nil
		return nil
	}
	for i, r := range c.Validator.IamPolicyRules {
		r := r
		if err := readIamPolicyRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.IamPolicyRules == nil {
		c.Validator.IamPolicyRules = make([]vpawsapi.IamPolicyRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another policy rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readIamPolicyRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another policy rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readIamPolicyRule(c *components.AWSPluginConfig, r *vpawsapi.IamPolicyRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &vpawsapi.IamPolicyRule{
			IamPolicyARN: "",
			Policies:     []vpawsapi.PolicyDocument{},
		}
	}
	err := initAwsRule(r, "IAM policy", ruleNames)
	if err != nil {
		return err
	}
	if r.IamPolicyARN == "" {
		policyArn, err := prompts.ReadTextRegex("IAM Policy ARN", "", "Invalid Policy ARN", cfg.PolicyArnRegex)
		if err != nil {
			return err
		}
		r.IamPolicyARN = policyArn
	}

	addPolicies := true
	for addPolicies {
		policyDoc, err := readIamPolicy()
		if err != nil {
			return err
		}

		r.Policies = append(r.Policies, policyDoc)
		addPolicies, err = prompts.ReadBool("Add another policy document", false)
		if err != nil {
			return err
		}
	}
	if idx == -1 {
		c.Validator.IamPolicyRules = append(c.Validator.IamPolicyRules, *r)
	} else {
		c.Validator.IamPolicyRules[idx] = *r
	}
	return nil
}

func readIamPolicy() (vpawsapi.PolicyDocument, error) {
	policyDoc := vpawsapi.PolicyDocument{}
	inputType, err := prompts.Select("Add policy document via", []string{"Local Filepath", "File Editor"})
	if err != nil {
		return policyDoc, err
	}

	for {
		var policyBytes []byte
		if inputType == "Local Filepath" {
			policyFile, err := prompts.ReadFilePath("Policy Document Filepath", "", "Invalid policy document path", false)
			if err != nil {
				return policyDoc, err
			}

			policyBytes, err = os.ReadFile(policyFile) //#nosec
			if err != nil {
				return policyDoc, err
			}
		} else {
			log.InfoCLI("Configure Policy Document")
			time.Sleep(2 * time.Second)
			policyFile, err := prompts.EditFileValidatedByFullContent(cfg.AWSPolicyDocumentPrompt, "", prompts.ValidateJson, 1)
			if err != nil {
				return policyDoc, err
			}
			policyBytes = []byte(policyFile)
		}

		var policy awspolicy.Policy
		errUnmarshal := policy.UnmarshalJSON(policyBytes)
		if errUnmarshal != nil {
			log.ErrorCLI("Failed to unmarshal the provided policy document", "err", errUnmarshal)
			retry, err := prompts.ReadBool("Reconfigure policy document", true)
			if err != nil {
				return policyDoc, err
			}

			if retry {
				continue
			}
			return policyDoc, errUnmarshal
		}

		policyDoc.Name = policy.ID
		policyDoc.Version = policy.Version
		policyDoc.Statements = convertStatements(policy.Statements)

		return policyDoc, nil
	}
}

// Convert statements from awspolicy to v1alpha1
func convertStatements(statements []awspolicy.Statement) []vpawsapi.StatementEntry {
	result := make([]vpawsapi.StatementEntry, 0)
	for _, s := range statements {
		result = append(result, vpawsapi.StatementEntry{
			Condition: vpawsapi.Condition(s.Condition),
			Effect:    s.Effect,
			Actions:   s.Action,
			Resources: s.Resource,
		})
	}
	return result
}

// nolint:dupl
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
		// not all awsRules have a Name field, for now we can create a unique name for them
		if name == "<invalid Value>" {
			name = ruleType + " - " + time.Now().Format("20060102T150405.000000000")
		}
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
