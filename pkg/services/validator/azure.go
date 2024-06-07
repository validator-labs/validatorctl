package validator

import (
	"fmt"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	plug "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const (
	singleCluster                 = "Single cluster"
	multipleClustersResourceGroup = "Multiple clusters in a single resource group"
	multipleClustersSubscription  = "Multiple clusters in a single subscription"

	rbacRuleActionTypeAction      = "Action"
	rbacRuleActionTypeDataAction  = "DataAction"
	rbacRuleTypeClusterDeployment = "Cluster Deployment"
	rbacRuleTypeCustom            = "Custom"

	mustBeValidUUID = "must be valid UUID"
)

var (
	azureSecretName = "azure-creds"

	rbacRuleTypes = []string{
		rbacRuleTypeClusterDeployment,
		rbacRuleTypeCustom,
	}

	staticDeploymentTypes = []string{
		singleCluster,
		multipleClustersResourceGroup,
		multipleClustersSubscription,
	}
)

func readAzurePlugin(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	c := vc.AzurePlugin

	if err := readHelmRelease(cfg.ValidatorPluginAzure, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return fmt.Errorf("failed to read Helm release: %w", err)
	}
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
	log.InfoCLI(`
	Azure RBAC validation ensures that a certain service principal has every
	permission specified in one of Spectro Cloud's predefined permission sets.

	Different permission sets are required to deploy clusters via Spectro Cloud,
	depending on placement type (static vs. dynamic) and other factors.
	`)

	var err error
	addRules := true
	ruleNames := make([]string, 0)

	for i, r := range c.Validator.RBACRules {
		r := r
		ruleType := c.RuleTypes[i]
		log.InfoCLI("Reconfiguring Azure RBAC %s rule: %s", ruleType, r.Name)

		switch ruleType {
		case rbacRuleTypeClusterDeployment:
			if err := configureClusterDeploymentAzureRBACRule(c, &ruleNames, &r, i); err != nil {
				return fmt.Errorf("failed to configure cluster deployment RBAC rule: %w", err)
			}
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
		case rbacRuleTypeClusterDeployment:
			if err := configureClusterDeploymentAzureRBACRule(c, &ruleNames, rule, ruleIdx); err != nil {
				return fmt.Errorf("failed to configure cluster deployment RBAC rule: %w", err)
			}
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

// Allows the user to configure an Azure RBAC rule for the situation where they want to deploy a
// cluster. They must specify cluster deployment type and then remaining info common to all cluster
// deployment scenarios. The info they specify is added to one of our presets.
func configureClusterDeploymentAzureRBACRule(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	var err error

	placementType, ok := c.PlacementTypes[idx]
	if !ok {
		placementType, err = prompts.Select("Placement type", cfg.PlacementTypes)
		if err != nil {
			return fmt.Errorf("failed to prompt for selection for placement type: %w", err)
		}
		c.PlacementTypes[idx] = placementType
	}

	switch placementType {
	case cfg.PlacementTypeStatic:
		if err := configureAzureRBACRuleClusterDeploymentStatic(c, ruleNames, r, idx); err != nil {
			return fmt.Errorf("failed to configure (%s placement type): %w", placementType, err)
		}
	case cfg.PlacementTypeDynamic:
		if err := configureAzureRBACRuleClusterDeploymentDynamic(c, ruleNames, r, idx); err != nil {
			return fmt.Errorf("failed to configure (%s placement type): %w", placementType, err)
		}
	default:
		return fmt.Errorf("unknown placement type (%s)", placementType)
	}

	return nil
}

// Builds an RBAC rule for deploying clusters with the static placement type. There are multiple
// ways to deploy clusters this way and the user specifies further.
func configureAzureRBACRuleClusterDeploymentStatic(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	var err error

	deploymentType, ok := c.StaticDeploymentTypes[idx]
	if !ok {
		deploymentType, err = prompts.Select("Static deployment type", staticDeploymentTypes)
		if err != nil {
			return fmt.Errorf("failed to prompt for selection for static deployment type: %w", err)
		}
		c.StaticDeploymentTypes[idx] = deploymentType
	}

	switch deploymentType {
	case singleCluster:
		if err := configureAzureSingleCluster(c, ruleNames, r, idx); err != nil {
			return fmt.Errorf("failed to configure (%s): %w", deploymentType, err)
		}
	case multipleClustersResourceGroup:
		if err := configureAzureMultipleClustersResourceGroup(c, ruleNames, r, idx); err != nil {
			return fmt.Errorf("failed to configure (%s): %w", deploymentType, err)
		}
	case multipleClustersSubscription:
		if err := configureAzureMultipleClustersSubscription(c, ruleNames, r, idx); err != nil {
			return fmt.Errorf("failed to configure (%s): %w", deploymentType, err)
		}
	default:
		return fmt.Errorf("unknown deployment type (%s)", deploymentType)
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

func staticRbacRuleValues(c *components.AzurePluginConfig, r *plug.RBACRule, idx int) (*components.AzureStaticDeploymentValues, error) {
	values, ok := c.StaticDeploymentValues[idx]
	if !ok {
		if r.PrincipalID != "" { // failure reconfiguring a rule
			return nil, fmt.Errorf("failed to get static deployment values for rule %s with index: %d", r.Name, idx)
		}
		return &components.AzureStaticDeploymentValues{}, nil
	}
	return values, nil
}

// Builds an RBAC rule for deploying a cluster with dynamic placement based on data provided by the user.
func configureAzureRBACRuleClusterDeploymentDynamic(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	if err := initRbacRule(ruleNames, r); err != nil {
		return err
	}
	v, err := staticRbacRuleValues(c, r, idx)
	if err != nil {
		return err
	}

	// For this use case, the security principal will always be a service principal (not user etc).
	logToCollect("service principal that will deploy cluster resources", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Service principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	logToCollect("subscription to deploy cluster resources into", formatAzureGUID)
	v.Subscription, err = prompts.ReadTextRegex("Subscription", v.Subscription, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for subscription: %w", err)
	}
	scope := fmt.Sprintf("/subscriptions/%s", v.Subscription)

	actionStrs := []plug.ActionStr{}
	for _, a := range cfg.ValidatorAzurePluginDynamicPlacementActions {
		actionStrs = append(actionStrs, plug.ActionStr(a))
	}

	r.Permissions = []plug.PermissionSet{
		// Permission set can be made narrower in scope in the future. Right now, we're making
		// this match our docs by having a wide set of permissions checked at the subscription
		// scope. We should be able to reduce the set of permissions and use a narrower scope
		// for some of them. For now, this gets Palette users going.
		{
			Actions: actionStrs,
			// This works for subscriptions that are in management groups and ones that aren't.
			Scope: scope,
		},
	}

	c.StaticDeploymentValues[idx] = v

	return nil
}

// Builds an RBAC rule for deploying a single cluster with static placement based on data provided
// by the user. Because it's a single cluster, it is expected that the user know which low level
// resources (e.g. virtual network subnet) will be used and they must provide them.
func configureAzureSingleCluster(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	if err := initRbacRule(ruleNames, r); err != nil {
		return err
	}
	v, err := staticRbacRuleValues(c, r, idx)
	if err != nil {
		return err
	}

	// For this use case, the security principal will always be a service principal (not user etc).
	logToCollect("service principal that will deploy cluster resources", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Service principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	logToCollect("subscription to deploy cluster resources into", formatAzureGUID)
	v.Subscription, err = prompts.ReadTextRegex("Subscription", v.Subscription, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for subscription: %w", err)
	}

	logToCollect("name of resource group (within configured subscription) to deploy cluster resources into", formatResourceGroupName)
	v.ResourceGroup, err = prompts.ReadText("Resource group", v.ResourceGroup, false, 1000)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for resource group: %w", err)
	}
	rgScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", v.Subscription, v.ResourceGroup)

	logToCollect("name of virtual network (within configured resource group) to use", formatVirtualNetworkName)
	v.VirtualNetwork, err = prompts.ReadText("Virtual network", v.VirtualNetwork, false, 1000)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for virtual network: %w", err)
	}
	vnScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s", v.Subscription, v.ResourceGroup, v.VirtualNetwork)

	logToCollect("name of subnet (within configured virtual network) to use", formatSubnetName)
	v.Subnet, err = prompts.ReadText("Subnet", v.Subnet, false, 1000)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for subnet: %w", err)
	}
	subnetScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", v.Subscription, v.ResourceGroup, v.VirtualNetwork, v.Subnet)

	logToCollect("Azure Compute Gallery (within configured resource group) to use for machine images", formatComputeGalleryName)
	v.ComputeGallery, err = prompts.ReadText("Compute Gallery", v.ComputeGallery, false, 1000)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for compute gallery: %w", err)
	}
	cgScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/galleries/%s", v.Subscription, v.ResourceGroup, v.ComputeGallery)

	resourceGroupLevelActions := []plug.ActionStr{}
	for _, a := range cfg.ValidatorAzurePluginStaticPlacementResourceGroupLevelActions {
		resourceGroupLevelActions = append(resourceGroupLevelActions, plug.ActionStr(a))
	}

	virtualNetworkLevelActions := []plug.ActionStr{}
	for _, a := range cfg.ValidatorAzurePluginStaticPlacementVirtualNetworkLevelActions {
		virtualNetworkLevelActions = append(virtualNetworkLevelActions, plug.ActionStr(a))
	}

	subnetLevelActions := []plug.ActionStr{}
	for _, a := range cfg.ValidatorAzurePluginStaticPlacementSubnetLevelActions {
		subnetLevelActions = append(subnetLevelActions, plug.ActionStr(a))
	}

	azureComputeGalleryLevelActions := []plug.ActionStr{}
	for _, a := range cfg.ValidatorAzurePluginStaticPlacementComputeGalleryLevelActions {
		azureComputeGalleryLevelActions = append(azureComputeGalleryLevelActions, plug.ActionStr(a))
	}

	r.Permissions = []plug.PermissionSet{
		// Each slice item corresponds to a group of actions needed for a particular
		// resource. We provide the actions and the user provides the resource which becomes
		// the scope of the permission set.
		{
			Actions: resourceGroupLevelActions,
			Scope:   rgScope,
		},
		{
			Actions: virtualNetworkLevelActions,
			Scope:   vnScope,
		},
		{
			Actions: subnetLevelActions,
			Scope:   subnetScope,
		},
		{
			Actions: azureComputeGalleryLevelActions,
			Scope:   cgScope,
		},
	}

	c.StaticDeploymentValues[idx] = v

	return nil
}

// Builds an RBAC rule for deploying multiple clusters to a single resource group with static
// placement based on data provided by the user. Because it's multiple clusters, we use the resource
// group as the scope for every permission so that it validates that the user can create all the
// resources (they do not know every resource down to its lowest level scope). It is expected that
// the user know which resource group is to be used and they must provide it.
func configureAzureMultipleClustersResourceGroup(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	if err := initRbacRule(ruleNames, r); err != nil {
		return err
	}
	v, err := staticRbacRuleValues(c, r, idx)
	if err != nil {
		return err
	}

	// For this use case, the security principal will always be a service principal (not user etc).
	logToCollect("service principal that will deploy cluster resources", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Service principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	logToCollect("subscription to deploy cluster resources into", formatAzureGUID)
	v.Subscription, err = prompts.ReadTextRegex("Subscription", v.Subscription, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for subscription: %w", err)
	}

	logToCollect("name of resource group (within configured subscription) to deploy cluster resources into", formatResourceGroupName)
	v.ResourceGroup, err = prompts.ReadText("Resource group", v.ResourceGroup, false, 1000)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for resource group: %w", err)
	}
	rgScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", v.Subscription, v.ResourceGroup)

	allPerms := []string{}
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementResourceGroupLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementVirtualNetworkLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementSubnetLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementComputeGalleryLevelActions...)

	allPermActionStrs := []plug.ActionStr{}
	for _, a := range allPerms {
		allPermActionStrs = append(allPermActionStrs, plug.ActionStr(a))
	}

	r.Permissions = []plug.PermissionSet{
		// We provide the actions. The user provides the resource group.
		{
			Actions: allPermActionStrs,
			Scope:   rgScope,
		},
	}

	c.StaticDeploymentValues[idx] = v

	return nil
}

// Builds an RBAC rule for deploying multiple clusters to a single subscription with static
// placement based on data provided by the user. Because it's multiple clusters, we use the
// subscription as the scope for every permission so that it validates that the user can create all
// the resources (they do not know every resource down to its lowest level scope). It is expected
// that the user know which subscription is to be used and they must provide it.
func configureAzureMultipleClustersSubscription(c *components.AzurePluginConfig, ruleNames *[]string, r *plug.RBACRule, idx int) error {
	if err := initRbacRule(ruleNames, r); err != nil {
		return err
	}
	v, err := staticRbacRuleValues(c, r, idx)
	if err != nil {
		return err
	}

	// For this use case, the security principal will always be a service principal (not user etc).
	logToCollect("service principal that will deploy cluster resources", formatAzureGUID)
	r.PrincipalID, err = prompts.ReadTextRegex("Service principal", r.PrincipalID, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for service principal: %w", err)
	}

	logToCollect("subscription to deploy cluster resources into", formatAzureGUID)
	v.Subscription, err = prompts.ReadTextRegex("Subscription", v.Subscription, mustBeValidUUID, prompts.UUIDRegex)
	if err != nil {
		return fmt.Errorf("failed to prompt for text for subscription: %w", err)
	}

	allPerms := []string{}
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementResourceGroupLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementVirtualNetworkLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementSubnetLevelActions...)
	allPerms = append(allPerms, cfg.ValidatorAzurePluginStaticPlacementComputeGalleryLevelActions...)

	allPermActionStrs := []plug.ActionStr{}
	for _, a := range allPerms {
		allPermActionStrs = append(allPermActionStrs, plug.ActionStr(a))
	}

	r.Permissions = []plug.PermissionSet{
		// We provide the actions. The user provides the subscription.
		{
			Actions: allPermActionStrs,
			Scope:   v.Subscription,
		},
	}

	c.StaticDeploymentValues[idx] = v

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
