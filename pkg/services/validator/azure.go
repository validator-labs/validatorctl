package validator

import (
	"fmt"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"

	plug "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const (
	rbacRuleActionTypeAction     = "Action"
	rbacRuleActionTypeDataAction = "DataAction"
	rbacRuleTypeCustom           = "Custom"

	mustBeValidUUID = "must be valid UUID"
)

var (
	azureSecretName = "azure-creds"

	rbacRuleTypes = []string{
		rbacRuleTypeCustom,
	}
)

func readAzurePlugin(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	c := vc.AzurePlugin

	if err := readHelmRelease(cfg.ValidatorPluginAzure, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return fmt.Errorf("failed to read Helm release: %w", err)
	}

	log.Header("Azure Configuration")

	if err := readAzureCredentials(c, k8sClient); err != nil {
		return errors.Wrap(err, "failed to read Azure credentials")
	}

	// Configure RBAC rules. Unlike how other plugins are styled, no prompt for whether the user
	// wants to configure this rule type because right now it is the only rule type for the plugin.
	if err := configureAzureRBACRules(c); err != nil {
		return fmt.Errorf("failed to configure RBAC rules: %w", err)
	}

	// impossible at present. uncomment if/when additional azure rule types are added.
	// if c.Validator.ResultCount() == 0 {
	// 	return errNoRulesEnabled
	// }
	return nil
}

func readAzureCredentials(c *components.AzurePluginConfig, k8sClient kubernetes.Interface) error {
	var err error

	c.Validator.Auth.Implicit, err = prompts.ReadBool("Use implicit Azure auth", true)
	if err != nil {
		return fmt.Errorf("failed to prompt for bool for use implicit Azure auth: %w", err)
	}
	if c.Validator.Auth.Implicit {
		c.ServiceAccountName, err = services.ReadServiceAccount(k8sClient, cfg.Validator)
		if err != nil {
			return fmt.Errorf("failed to read k8s ServiceAccount: %w", err)
		}
	} else {
		// always create Azure credential secret if creating a new kind cluster
		createSecret := true

		if k8sClient != nil {
			log.InfoCLI(`
	Either specify Azure credentials or provide the name of a secret in the target K8s cluster's %s namespace.
	If using an existing secret, it must contain the following keys: %+v.
	`, cfg.Validator, cfg.ValidatorPluginAzureKeys,
			)
			createSecret, err = prompts.ReadBool("Create Azure credential secret", true)
			if err != nil {
				return fmt.Errorf("failed to create Azure credential secret: %w", err)
			}
		}

		if createSecret {
			if c.Validator.Auth.SecretName != "" {
				azureSecretName = c.Validator.Auth.SecretName
			}
			c.Validator.Auth.SecretName, err = prompts.ReadText("Azure credentials secret name", azureSecretName, false, -1)
			if err != nil {
				return fmt.Errorf("failed to prompt for text for Azure credentials secret name: %w", err)
			}
			c.TenantID, err = prompts.ReadTextRegex("Azure Tenant ID", c.TenantID, mustBeValidUUID, prompts.UUIDRegex)
			if err != nil {
				return fmt.Errorf("failed to prompt for text for Azure Tenant ID: %w", err)
			}
			c.ClientID, err = prompts.ReadTextRegex("Azure Client ID", c.ClientID, mustBeValidUUID, prompts.UUIDRegex)
			if err != nil {
				return fmt.Errorf("failed to prompt for text for Azure Client ID: %w", err)
			}
			c.ClientSecret, err = prompts.ReadPassword("Azure Client Secret", c.ClientSecret, false, -1)
			if err != nil {
				return fmt.Errorf("failed to prompt for password for Azure Client Secret: %w", err)
			}
		} else {
			secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, cfg.ValidatorPluginAzureKeys)
			if err != nil {
				return fmt.Errorf("failed to read k8s Secret: %w", err)
			}
			c.Validator.Auth.SecretName = secret.Name
			c.TenantID = string(secret.Data["AZURE_TENANT_ID"])
			c.ClientID = string(secret.Data["AZURE_CLIENT_ID"])
			c.ClientSecret = string(secret.Data["AZURE_CLIENT_SECRET"])
		}
	}

	return nil
}

// configureAzureRBACRules sets up zero or more RBAC rules based on user input. To save the users
// some typing, we allow the user to choose between using a preset and making custom rules.
func configureAzureRBACRules(c *components.AzurePluginConfig) error {
	var err error
	addRules := true
	ruleNames := make([]string, 0)

	for i, r := range c.Validator.RBACRules {
		r := r
		ruleType := c.RuleTypes[i]
		log.InfoCLI("Reconfiguring Azure RBAC %s rule: %s", ruleType, r.Name)

		switch ruleType {
		case rbacRuleTypeCustom:
			if err = configureCustomAzureRBACRule(&ruleNames, &r); err != nil {
				return fmt.Errorf("failed to configure custom RBAC rule: %w", err)
			}
		default:
			return fmt.Errorf("unknown rule type (%s)", ruleType)
		}

		c.Validator.RBACRules[i] = r
	}

	if c.Validator.RBACRules == nil {
		c.Validator.RBACRules = make([]plug.RBACRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another RBAC rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}

	log.InfoCLI("Note: You must configure at least one rule for plugin configuration.")
	ruleIdx := len(c.Validator.RBACRules)

	for {
		log.InfoCLI("Note: Collecting input for rule #%d", ruleIdx+1)

		// Type is determined first because this is likely how a user would think while they're
		// using the CLI. This causes some repeated code in other functions called from this one
		// for name and principal, but this is intentional.
		ruleType, err := prompts.Select("Rule type", rbacRuleTypes)
		if err != nil {
			return fmt.Errorf("failed to prompt for selection for rule type: %w", err)
		}
		c.RuleTypes[ruleIdx] = ruleType

		rule := &plug.RBACRule{
			Permissions: []plug.PermissionSet{},
		}

		switch ruleType {
		case rbacRuleTypeCustom:
			if err := configureCustomAzureRBACRule(&ruleNames, rule); err != nil {
				return fmt.Errorf("failed to configure custom RBAC rule: %w", err)
			}
		default:
			return fmt.Errorf("unknown rule type (%s)", ruleType)
		}

		c.Validator.RBACRules = append(c.Validator.RBACRules, *rule)

		addRBACRule, err := prompts.ReadBool("Add additional RBAC rule", false)
		if err != nil {
			return fmt.Errorf("failed to prompt for bool for add an RBAC rule: %w", err)
		}
		if !addRBACRule {
			break
		}
		ruleIdx++
	}

	return nil
}

func initRbacRule(ruleNames *[]string, r *plug.RBACRule) error {
	var err error
	if r.Name != "" {
		log.InfoCLI("Reconfiguring RBAC rule: %s", r.Name)
		*ruleNames = append(*ruleNames, r.Name)
	} else {
		r.Name, err = getRuleName(ruleNames)
		if err != nil {
			return err
		}
	}
	return nil
}

// Allows the user to configure an Azure RBAC rule where they specify every detail.
func configureCustomAzureRBACRule(ruleNames *[]string, r *plug.RBACRule) error {
	err := initRbacRule(ruleNames, r)
	if err != nil {
		return err
	}

	logToCollect("service principal", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Service principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	if err := configureAzureRBACRulePermissionSets(r); err != nil {
		return fmt.Errorf("failed to configure permission sets: %w", err)
	}
	return nil
}

func configureAzureRBACRulePermissionSets(r *plug.RBACRule) error {
	permissionSets := []plug.PermissionSet{}

	log.InfoCLI("Note: You must configure at least one permission set for rule.")
	log.InfoCLI("If you're updating an existing custom RBAC rule, its permission sets will be replaced.")
	// TODO: consider reconfiguration supported for custom RBAC rules. It's just not worth the effort right now.

	for {
		log.InfoCLI("Note: Collecting input for permission set #%d.", len(permissionSets)+1)

		set, err := configureAzureRBACRulePermissionSet()
		if err != nil {
			return fmt.Errorf("failed to configure permission set: %w", err)
		}
		permissionSets = append(permissionSets, set)

		add, err := prompts.ReadBool("Add additional permission set", false)
		if err != nil {
			return fmt.Errorf("failed to prompt for bool for add permission set: %w", err)
		}
		if !add {
			break
		}
	}

	r.Permissions = permissionSets

	return nil
}

func configureAzureRBACRulePermissionSet() (plug.PermissionSet, error) {
	permissionSet := plug.PermissionSet{}

	logToCollect("scope that the the permissions must apply to", formatFullyQualifiedAzureResourceName)
	scope, err := prompts.ReadText("Scope", "", false, 500)
	if err != nil {
		return plug.PermissionSet{}, fmt.Errorf("failed to prompt for text for scope: %w", err)
	}
	permissionSet.Scope = scope

	actions, err := configureAzureRBACRulePermissionSetActions(rbacRuleActionTypeAction)
	if err != nil {
		return plug.PermissionSet{}, fmt.Errorf("failed to configure Actions: %w", err)
	}
	actionStrs := []plug.ActionStr{}
	for _, a := range actions {
		actionStrs = append(actionStrs, plug.ActionStr(a))
	}
	permissionSet.Actions = actionStrs

	dataActions, err := configureAzureRBACRulePermissionSetActions(rbacRuleActionTypeDataAction)
	if err != nil {
		return plug.PermissionSet{}, fmt.Errorf("failed to configure DataActions: %w", err)
	}
	dataActionStrs := []plug.ActionStr{}
	for _, da := range dataActions {
		dataActionStrs = append(dataActionStrs, plug.ActionStr(da))
	}
	permissionSet.DataActions = dataActionStrs

	if len(actions) == 0 && len(dataActions) == 0 {
		log.InfoCLI("You must configure at least one Action or one DataAction for each permission set. Please try again.")
		return configureAzureRBACRulePermissionSet()
	}

	return permissionSet, nil
}

func configureAzureRBACRulePermissionSetActions(actionType string) ([]string, error) {
	actions := make([]string, 0)

	for {
		log.InfoCLI("Enter configuration for %s #%d.", actionType, len(actions)+1)

		action, err := prompts.ReadText(actionType, "", true, 100)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for text for %s: %w", actionType, err)
		}
		if action != "" {
			actions = append(actions, action)
		}

		add, err := prompts.ReadBool(fmt.Sprintf("Add another %s", actionType), false)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for bool for add %s: %w", actionType, err)
		}
		if !add {
			break
		}
	}

	return actions, nil
}

const (
	formatAzureGUID = iota
	formatResourceGroupName
	formatVirtualNetworkName
	formatSubnetName
	formatComputeGalleryName
	formatFullyQualifiedAzureResourceName
)

// logToCollect logs a few messages to guide the user when we need to collect data from them.
//   - dataToCollect - A string used in a message. Should not begin with a capital letter unless the
//     name of the data to be collected is a proper noun.
//   - format - An enum value to indicate what the format of the data to be collected is.
func logToCollect(dataToCollect string, format int) {
	log.InfoCLI("Enter %s.", dataToCollect)

	var formatLabel string
	var example string

	exampleGUID := "d6df0bba-800d-492f-802e-d04a38c80786"
	exampleResourceGroupName := "rg1"
	exampleVirtualNetworkName := "vnet1"
	exampleSubnetName := "subnet1"
	exampleGalleryName := "gallery1"
	exampleFullyQualifiedAzureResourceName := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", exampleGUID, exampleResourceGroupName)

	switch format {
	case formatAzureGUID:
		formatLabel = "Azure GUID"
		example = exampleGUID
	case formatResourceGroupName:
		formatLabel = "Resource group name"
		example = exampleResourceGroupName
	case formatVirtualNetworkName:
		formatLabel = "Virtual network name"
		example = exampleVirtualNetworkName
	case formatSubnetName:
		formatLabel = "Subnet name"
		example = exampleSubnetName
	case formatComputeGalleryName:
		formatLabel = "Gallery name"
		example = exampleGalleryName
	case formatFullyQualifiedAzureResourceName:
		formatLabel = "Fully-qualified Azure resource name"
		example = exampleFullyQualifiedAzureResourceName
	}

	log.InfoCLI("Format: %s", formatLabel)
	log.InfoCLI("Example: %s", example)
}
