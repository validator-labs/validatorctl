package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	plug "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-azure/pkg/utils/azure"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const (
	mustBeValidUUID = "must be valid UUID"

	azureNoCredsErr = "DefaultAzureCredential: "

	mockAzureScope = "00000000-0000-0000-0000-000000000000"

	mockAzureRoleAssignment = "00000000-000000-0000000000"

	regexOneCharString = ".+"
)

var (
	azureSecretName = "azure-creds"
)

func readAzurePlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	c := vc.AzurePlugin

	log.InfoCLI("Select the Azure cloud environment to connect to.")
	var err error
	vc.AzurePlugin.Cloud, err = prompts.Select("Azure cloud", cfg.ValidatorAzureClouds)
	if err != nil {
		return err
	}

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginAzure, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}
	if err := readAzureCredentials(c, tc, k8sClient); err != nil {
		return fmt.Errorf("failed to read Azure credentials: %w", err)
	}

	return nil
}

// readAzurePluginRules reads Azure plugin configuration and rules from the user.
func readAzurePluginRules(vc *components.ValidatorConfig, _ *cfg.TaskConfig, _ kubernetes.Interface) error {
	log.Header("Azure Plugin Rule Configuration")

	c := vc.AzurePlugin
	ruleNames := make([]string, 0)

	if err := configureRBACRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureCommunityGalleryImageRules(c, &ruleNames); err != nil {
		return err
	}
	if err := configureQuotaRules(c, &ruleNames); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}

	return nil
}

func readAzureCredentials(c *components.AzurePluginConfig, tc *cfg.TaskConfig, k8sClient kubernetes.Interface) error {
	if tc.Direct {
		return readDirectAzureCredentials(c)
	}
	return readInstallAzureCredentials(c, k8sClient)
}

func readDirectAzureCredentials(c *components.AzurePluginConfig) error {
	// check if credentials are already configured
	api, err := azure.NewAzureAPI()
	if err != nil {
		return fmt.Errorf("failed to create Azure API: %w", err)
	}
	_, err = api.RoleAssignmentsClient.Get(context.Background(), mockAzureScope, mockAzureRoleAssignment, nil)
	// auth toolchain is configured, skip prompting for credentials
	if err == nil || !strings.Contains(err.Error(), azureNoCredsErr) {
		return nil
	}

	err = readAzureCredsHelper(c)
	if err != nil {
		return err
	}

	err = os.Setenv("AZURE_ENVIRONMENT", c.Cloud)
	if err != nil {
		return fmt.Errorf("failed to set AZURE_ENVIRONMENT: %w", err)
	}
	err = os.Setenv("AZURE_TENANT_ID", c.TenantID)
	if err != nil {
		return fmt.Errorf("failed to set AZURE_TENANT_ID: %w", err)
	}
	err = os.Setenv("AZURE_CLIENT_ID", c.ClientID)
	if err != nil {
		return fmt.Errorf("failed to set AZURE_CLIENT_ID: %w", err)
	}
	err = os.Setenv("AZURE_CLIENT_SECRET", c.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to set AZURE_CLIENT_SECRET: %w", err)
	}

	return nil
}

func readInstallAzureCredentials(c *components.AzurePluginConfig, k8sClient kubernetes.Interface) error {
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
		return nil
	}
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

		err = readAzureCredsHelper(c)
		if err != nil {
			return err
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

	return nil
}

func readAzureCredsHelper(c *components.AzurePluginConfig) error {
	var err error
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
	return nil
}

// configureRBACRules sets up zero or more RBAC rules based on pre-existing files or user input.
// nolint:dupl
func configureRBACRules(c *components.AzurePluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	RBAC validation rules ensure that security principals have the required permissions.
	`)

	validateRBAC, err := prompts.ReadBool("Enable Azure RBAC validation", true)
	if err != nil {
		return err
	}
	if !validateRBAC {
		c.Validator.RBACRules = nil
		return nil
	}
	for i, r := range c.Validator.RBACRules {
		r := r
		if err := readRBACRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.RBACRules) == 0 {
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
	for {
		if err := readRBACRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add additional RBAC rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// readRBACRule begins the process of reconfiguring or beginning a new RBAC rule.
// nolint:dupl
func readRBACRule(c *components.AzurePluginConfig, r *plug.RBACRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &plug.RBACRule{}
	}

	err := initRule(r, "RBAC", "", ruleNames)
	if err != nil {
		return err
	}

	logToCollect("security principal", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Security principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return err
	}

	if err := readRBACRulePermissionSets(r); err != nil {
		return err
	}

	if idx == -1 {
		c.Validator.RBACRules = append(c.Validator.RBACRules, *r)
	} else {
		c.Validator.RBACRules[idx] = *r
	}
	return nil
}

// readRBACRulePermissionSets begins the process of beginning a new list of permission sets. Users
// can provide input via file or prompts.
// nolint:dupl
func readRBACRulePermissionSets(r *plug.RBACRule) error {
	log.InfoCLI("Note: You must configure at least one permission set for the rule.")
	log.InfoCLI("If you're updating an existing RBAC rule, its permission sets will be replaced.")

	inputType, err := prompts.Select("Add permission sets via", cfg.FileInputs)
	if err != nil {
		return err
	}

	for {
		var permissionSetBytes []byte
		if inputType == cfg.LocalFilepath {
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
			permissionSetFile, err := prompts.EditFileValidatedByFullContent(cfg.AzurePermissionSetPrompt, "", prompts.ValidateJSON, 1)
			if err != nil {
				return fmt.Errorf("failed to configure permission sets: %w", err)
			}
			permissionSetBytes = []byte(permissionSetFile)
		}

		if err := json.Unmarshal(permissionSetBytes, &r.Permissions); err != nil {
			log.ErrorCLI("Failed to unmarshal the provided permission sets", "err", err)
			retry, err := prompts.ReadBool("Reconfigure permission sets", true)
			if err != nil {
				return err
			}
			if retry {
				continue
			}
			return fmt.Errorf("failed to unmarshal permission sets: %w", err)
		}

		return nil
	}
}

// configureCommunityGalleryImageRules sets up zero or more Community Gallery Image rules based on
// pre-existing files or user input.
// nolint:dupl
func configureCommunityGalleryImageRules(c *components.AzurePluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	Community gallery image validation rules ensure that images are publicly available via community galleries.
	`)

	validateCommunityGalleryImage, err := prompts.ReadBool("Enable Community Gallery Image validation", true)
	if err != nil {
		return err
	}
	if !validateCommunityGalleryImage {
		c.Validator.CommunityGalleryImageRules = nil
		return nil
	}
	for i, r := range c.Validator.CommunityGalleryImageRules {
		r := r
		if err := readCommunityGalleryImageRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if c.Validator.CommunityGalleryImageRules == nil {
		c.Validator.CommunityGalleryImageRules = make([]plug.CommunityGalleryImageRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another Community Gallery Image rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readCommunityGalleryImageRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add additional Community Gallery Image rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// readCommunityGalleryImageRule begins the process of reconfiguring or beginning a new Community
// Gallery Image rule.
// nolint:dupl
func readCommunityGalleryImageRule(c *components.AzurePluginConfig, r *plug.CommunityGalleryImageRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &plug.CommunityGalleryImageRule{}
	}

	err := initRule(r, "Community Gallery Image", "", ruleNames)
	if err != nil {
		return err
	}

	logToCollect("gallery location", formatAzureLocation)
	if r.Gallery.Location, err = prompts.ReadText("Gallery location", r.Gallery.Location, false, -1); err != nil {
		return err
	}

	if r.Gallery.Name, err = prompts.ReadText("Gallery name", r.Gallery.Name, false, -1); err != nil {
		return err
	}

	if r.Images, err = prompts.ReadTextSlice("Images", strings.Join(r.Images, "\n"), "image names must be at least one character", regexOneCharString, false); err != nil {
		return fmt.Errorf("failed to prompt for images: %w", err)
	}

	log.InfoCLI(`
 	Community gallery images are accessed via subscriptions.
	Provide the ID of the subscription you want to verify can access the community gallery image(s).
	This can be any subscription the security principal you authed the Azure plugin with has access to.
	`)
	logToCollect("subscription ID", formatAzureGUID)
	if r.SubscriptionID, err = prompts.ReadTextRegex("Subscription ID", r.SubscriptionID, mustBeValidUUID, prompts.UUIDRegex); err != nil {
		return err
	}

	if idx == -1 {
		c.Validator.CommunityGalleryImageRules = append(c.Validator.CommunityGalleryImageRules, *r)
	} else {
		c.Validator.CommunityGalleryImageRules[idx] = *r
	}
	return nil
}

// configureQuotaRules sets up zero or more quota rules based on pre-existing files or user input.
// nolint:dupl
func configureQuotaRules(c *components.AzurePluginConfig, ruleNames *[]string) error {
	log.InfoCLI(`
	Quota validation rules ensure that quota limits are set high enough to account for current usage plus a buffer.
	`)

	validateQuotas, err := prompts.ReadBool("Enable quota validation", true)
	if err != nil {
		return err
	}
	if !validateQuotas {
		c.Validator.RBACRules = nil
		return nil
	}
	for i, r := range c.Validator.QuotaRules {
		r := r
		if err := readQuotaRule(c, &r, i, ruleNames); err != nil {
			return err
		}
	}
	addRules := true
	if len(c.Validator.QuotaRules) == 0 {
		c.Validator.QuotaRules = make([]plug.QuotaRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another quota rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}
	for {
		if err := readQuotaRule(c, nil, -1, ruleNames); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add additional quota rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

// func readQuotaRule begins the process of reconfiguring or beginning a new quota rule.
// nolint:dupl
func readQuotaRule(c *components.AzurePluginConfig, r *plug.QuotaRule, idx int, ruleNames *[]string) error {
	if r == nil {
		r = &plug.QuotaRule{}
	}

	err := initRule(r, "quota", "", ruleNames)
	if err != nil {
		return err
	}

	if err := readQuotaRuleResourceSets(r); err != nil {
		return err
	}

	if idx == -1 {
		c.Validator.QuotaRules = append(c.Validator.QuotaRules, *r)
	} else {
		c.Validator.QuotaRules[idx] = *r
	}
	return nil
}

// readQuotaRuleResourceSets begins the process of setting the resource sets of the rule. Users can
// provide input via file or prompts.
// nolint:dupl
func readQuotaRuleResourceSets(r *plug.QuotaRule) error {
	log.InfoCLI("Note: You must configure at least one resource set for the rule.")
	log.InfoCLI("If you're updating an existing quota rule, its resource sets will be replaced.")

	inputType, err := prompts.Select("Add resource sets via", cfg.FileInputs)
	if err != nil {
		return err
	}

	for {
		var resourceSetBytes []byte
		if inputType == cfg.LocalFilepath {
			resourceSetFile, err := prompts.ReadFilePath("Resource sets file path", "", "Invalid file path", false)
			if err != nil {
				return err
			}
			resourceSetBytes, err = os.ReadFile(resourceSetFile) //#nosec
			if err != nil {
				return fmt.Errorf("failed to read resource sets file: %w", err)
			}
		} else {
			log.InfoCLI("Configure resource sets")
			time.Sleep(2 * time.Second)
			resourceSetFile, err := prompts.EditFileValidatedByFullContent(cfg.AzurePermissionSetPrompt, "", prompts.ValidateJSON, 1)
			if err != nil {
				return fmt.Errorf("failed to configure resource sets: %w", err)
			}
			resourceSetBytes = []byte(resourceSetFile)
		}

		if err := json.Unmarshal(resourceSetBytes, &r.ResourceSets); err != nil {
			log.ErrorCLI("Failed to unmarshal the provided resource sets", "err", err)
			retry, err := prompts.ReadBool("Reconfigure resource sets", true)
			if err != nil {
				return err
			}
			if retry {
				continue
			}
			return fmt.Errorf("failed to unmarshal resource sets: %w", err)
		}

		return nil
	}
}

const (
	formatAzureGUID     = iota
	formatAzureLocation = iota
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
	case formatAzureLocation:
		formatLabel = "Azure location"
		example = "westus"
	}

	log.InfoCLI("Format: %s", formatLabel)
	log.InfoCLI("Example: %s", example)
}
