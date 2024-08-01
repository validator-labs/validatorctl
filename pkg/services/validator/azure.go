package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"

	plug "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const (
	ruleTypeRBAC = "RBAC"

	mustBeValidUUID = "must be valid UUID"
)

var (
	azureSecretName = "azure-creds"
)

func readAzurePluginInstall(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	log.Header("Azure Plugin Installation Configuration")
	c := vc.AzurePlugin

	if err := readHelmRelease(cfg.ValidatorPluginAzure, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return fmt.Errorf("failed to read Helm release: %w", err)
	}
	if err := readAzureCredentials(c, k8sClient); err != nil {
		return fmt.Errorf("failed to read Azure credentials: %w", err)
	}

	return nil
}

func readAzurePluginRules(vc *components.ValidatorConfig, _ kubernetes.Interface) error {
	log.Header("Azure Plugin Rule Configuration")

	// Configure RBAC rules. Unlike how other plugins are styled, no prompt for whether the user
	// wants to configure this rule type because right now it is the only rule type for the plugin.
	if err := configureAzureRBACRules(vc.AzurePlugin); err != nil {
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

// configureAzureRBACRules sets up zero or more RBAC rules based on pre-existing files or user
// input.
func configureAzureRBACRules(c *components.AzurePluginConfig) error {
	var err error
	addRules := true
	ruleNames := make([]string, 0)

	for i, r := range c.Validator.RBACRules {
		r := r
		ruleType := c.RuleTypes[i]
		log.InfoCLI("Reconfiguring Azure RBAC %s rule: %s", ruleType, r.Name)

		if err = configureAzureRBACRule(&ruleNames, &r); err != nil {
			return fmt.Errorf("failed to configure RBAC rule: %w", err)
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

		// This is intentional. We only have one rule type in the Azure plugin now, so we don't
		// prompt the user for rule type. But we are keeping the rest of the type-oriented code here
		// (e.g. the rule types tracked in the YAML config file) to avoid less refactor work later
		// when we add more rule types for Azure.
		ruleType := ruleTypeRBAC
		c.RuleTypes[ruleIdx] = ruleType

		rule := &plug.RBACRule{
			Permissions: []plug.PermissionSet{},
		}

		switch ruleType {
		case ruleTypeRBAC:
			if err := configureAzureRBACRule(&ruleNames, rule); err != nil {
				return fmt.Errorf("failed to configure RBAC rule: %w", err)
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
func configureAzureRBACRule(ruleNames *[]string, r *plug.RBACRule) error {
	err := initRbacRule(ruleNames, r)
	if err != nil {
		return err
	}

	logToCollect("security principal", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Security principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	if err := configureAzureRBACRulePermissionSets(r); err != nil {
		return fmt.Errorf("failed to configure permission sets: %w", err)
	}
	return nil
}

func configureAzureRBACRulePermissionSets(r *plug.RBACRule) error {
	log.InfoCLI("Note: You must configure at least one permission set for rule.")
	log.InfoCLI("If you're updating an existing RBAC rule, its permission sets will be replaced.")

	inputType, err := prompts.Select("Add permission sets via", []string{"Local Filepath", "File Editor"})
	if err != nil {
		return err
	}

	for {
		var permissionSetBytes []byte
		if inputType == "Local Filepath" {
			permissionSetFile, err := prompts.ReadFilePath("Permission sets file path", "", "Invalid file path", false)
			if err != nil {
				return err
			}

			permissionSetBytes, err = os.ReadFile(permissionSetFile) //#nosec
			if err != nil {
				return fmt.Errorf("failed to read permission sets file: %w", err)
			}
		} else {
			log.InfoCLI("Configure permission sets")
			time.Sleep(2 * time.Second)
			permissionSetFile, err := prompts.EditFileValidatedByFullContent(cfg.AzurePermissionSetPrompt, "", prompts.ValidateJson, 1)
			if err != nil {
				return fmt.Errorf("failed to configure permission sets: %w", err)
			}
			permissionSetBytes = []byte(permissionSetFile)
		}

		var permissionSets []plug.PermissionSet
		errUnmarshal := json.Unmarshal(permissionSetBytes, &permissionSets)
		if errUnmarshal != nil {
			log.ErrorCLI("Failed to unmarshal the provided permission sets", "err", errUnmarshal)
			retry, err := prompts.ReadBool("Reconfigure permission sets", true)
			if err != nil {
				return fmt.Errorf("failed to prompt for reconfiguration of permission sets: %w", err)
			}

			if retry {
				continue
			}
			return fmt.Errorf("failed to unmarshal permission sets: %w", errUnmarshal)
		}

		r.Permissions = permissionSets
		return nil
	}
}

const (
	formatAzureGUID = iota
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

	switch format {
	case formatAzureGUID:
		formatLabel = "Azure GUID"
		example = exampleGUID
	}

	log.InfoCLI("Format: %s", formatLabel)
	log.InfoCLI("Example: %s", example)
}
