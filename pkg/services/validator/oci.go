package validator

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"

	plug "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const (
	basicAuth = "Basic"
	ecrAuth   = "ECR"
)

func readOciPlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, _ kubernetes.Interface) error {
	c := vc.OCIPlugin

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginOci, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}

	return nil
}

func readOciPluginRules(vc *components.ValidatorConfig, tc *cfg.TaskConfig, kClient kubernetes.Interface) error {
	log.Header("OCI Plugin Rule Configuration")
	c := vc.OCIPlugin
	ruleNames := make([]string, 0)
	authSecretNames := make([]string, 0)
	sigSecretNames := make([]string, 0)

	if err := configureOciRegistryRules(c, &ruleNames, &authSecretNames, &sigSecretNames, kClient, tc.Direct); err != nil {
		return err
	}

	if c.Validator.ResultCount() == 0 {
		return errNoRulesEnabled
	}
	return nil
}

// configureAuthInline prompts the user to configure their OCI registry authentication details.
func configureAuthInline(r *plug.OciRegistryRule) error {
	r.Auth.SecretName = nil

	authType, err := prompts.Select("Authentication type", []string{basicAuth, ecrAuth})
	if err != nil {
		return err
	}

	if authType == basicAuth {
		r.Auth.ECR = nil
		if r.Auth.Basic == nil {
			r.Auth.Basic = &plug.BasicAuth{}
		}
		r.Auth.Basic.Username, r.Auth.Basic.Password, err = prompts.ReadBasicCreds(
			"Username", "Password", r.Auth.Basic.Username, r.Auth.Basic.Password, false, false,
		)
		if err != nil {
			return err
		}
		return nil
	}

	r.Auth.Basic = nil
	if r.Auth.ECR == nil {
		r.Auth.ECR = &plug.ECRAuth{}
	}

	r.Auth.ECR.AccessKeyID, r.Auth.ECR.SecretAccessKey, r.Auth.ECR.SessionToken, err = readAwsCreds(r.Auth.ECR.AccessKeyID, r.Auth.ECR.SecretAccessKey, r.Auth.ECR.SessionToken)
	if err != nil {
		return err
	}

	return nil
}

// configureAuthSecrets prompts the user to configure secrets containing authentication details.
func configureAuthSecrets(c *components.OCIPluginConfig, r *plug.OciRegistryRule, kClient kubernetes.Interface, authSecretNames *[]string) error {
	allSecretNames := []string{cfg.OciCreateNewAuthSecPrompt} // provide the option to create a new secret
	allSecretNames = append(allSecretNames, *authSecretNames...)
	if kClient != nil {
		existingAuthSecrets, err := services.GetSecretsWithKeys(kClient, cfg.Validator, cfg.ValidatorBasicAuthKeys)
		if err != nil {
			return err
		}
		for _, s := range existingAuthSecrets {
			allSecretNames = append(allSecretNames, s.Name)
		}
	}

	var err error
	useSecretName := cfg.OciCreateNewAuthSecPrompt
	if len(allSecretNames) > 1 {
		useSecretName, err = prompts.Select("Registry authentication secret name", allSecretNames)
		if err != nil {
			return err
		}
	}

	if useSecretName == cfg.OciCreateNewAuthSecPrompt {
		secret := &components.Secret{}
		if err := readOciSecret(secret); err != nil {
			return err
		}
		c.Secrets = append(c.Secrets, secret)
		useSecretName = secret.Name
		*authSecretNames = append(*authSecretNames, useSecretName)
	}
	r.Auth.SecretName = &useSecretName
	return nil
}

// configureSigVerification prompts the user to configure their public keys for signature verification.
func configureSigVerification(r *plug.OciRegistryRule) error {
	r.SignatureVerification.SecretName = ""

	pubKeys, err := configurePublicKeys()
	if err != nil {
		return err
	}

	r.SignatureVerification.PublicKeys = pubKeys
	return nil
}

// configureSigVerificationSecrets prompts the user to configure secrets containing public keys for use in signature verification.
func configureSigVerificationSecrets(c *components.OCIPluginConfig, r *plug.OciRegistryRule, kClient kubernetes.Interface, sigSecretNames *[]string) error {
	allSecretNames := []string{cfg.OciCreateNewSigSecPrompt} // provide the option to create a new secret
	allSecretNames = append(allSecretNames, *sigSecretNames...)

	if kClient != nil {
		existingSigSecrets, err := services.GetSecretsWithRegexKeys(kClient, cfg.Validator, cfg.ValidatorPluginOciSigVerificationKeysRegex)
		if err != nil {
			return err
		}
		for _, s := range existingSigSecrets {
			allSecretNames = append(allSecretNames, s.Name)
		}
	}

	var err error
	useSecretName := cfg.OciCreateNewSigSecPrompt
	if len(allSecretNames) > 1 {
		useSecretName, err = prompts.Select("Signature verification secret name", allSecretNames)
		if err != nil {
			return err
		}
	}

	if useSecretName == cfg.OciCreateNewSigSecPrompt {
		secret := &components.PublicKeySecret{}
		if err := readPublicKeySecret(secret); err != nil {
			return err
		}
		c.PublicKeySecrets = append(c.PublicKeySecrets, secret)
		useSecretName = secret.Name
		*sigSecretNames = append(*sigSecretNames, useSecretName)
	}
	r.SignatureVerification = plug.SignatureVerification{
		Provider:   "cosign",
		SecretName: useSecretName,
	}
	return nil
}

func readPublicKeySecret(secret *components.PublicKeySecret) error {
	var err error
	if secret.Name == "" {
		secret.Name, err = prompts.ReadK8sName("Secret Name", "", false)
		if err != nil {
			return err
		}
	} else {
		log.InfoCLI("Reconfiguring secret: %s", secret.Name)
	}

	pubKeys, err := configurePublicKeys()
	if err != nil {
		return err
	}

	secret.Keys = pubKeys
	return nil
}

// configurePublicKeys prompts the user to configure a list of public keys.
func configurePublicKeys() ([]string, error) {
	pubKeys := make([]string, 0)
	for {
		pubKeyPath, err := prompts.ReadFilePath("Public Key file", "", "Invalid public key path", false)
		if err != nil {
			return nil, err
		}
		pubKeyBytes, err := os.ReadFile(pubKeyPath) //#nosec
		if err != nil {
			return nil, err
		}
		pubKeys = append(pubKeys, string(pubKeyBytes))

		add, err := prompts.ReadBool("Add another public key", false)
		if err != nil {
			return nil, err
		}
		if !add {
			break
		}
	}
	return pubKeys, nil
}

func configureOciRegistryRules(c *components.OCIPluginConfig, ruleNames, authSecretNames, sigSecretNames *[]string, kClient kubernetes.Interface, direct bool) error {
	log.InfoCLI(`
	OCI registry rule(s) ensure that specific OCI artifacts are present in an OCI registry.
	`)

	for i, r := range c.Validator.OciRegistryRules {
		r := r
		if err := readOciRegistryRule(c, &r, i, ruleNames, authSecretNames, sigSecretNames, kClient, direct); err != nil {
			return err
		}
	}

	var err error
	addRules := true

	if len(c.Validator.OciRegistryRules) == 0 {
		c.Validator.OciRegistryRules = make([]plug.OciRegistryRule, 0)
	} else {
		addRules, err = prompts.ReadBool("Add another OCI registry rule", false)
		if err != nil {
			return err
		}
	}
	if !addRules {
		return nil
	}

	for {
		if err := readOciRegistryRule(c, &plug.OciRegistryRule{}, -1, ruleNames, authSecretNames, sigSecretNames, kClient, direct); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another OCI registry rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}

	return nil
}

func readOciRegistryRule(c *components.OCIPluginConfig, r *plug.OciRegistryRule, idx int, ruleNames, authSecretNames, sigSecretNames *[]string, kClient kubernetes.Interface, direct bool) error {
	err := initRule(r, "OCI", "", ruleNames)
	if err != nil {
		return err
	}

	log.InfoCLI("Example OCI registry hosts: gcr.io, quay.io, oci://myregistry:5000")
	host, err := prompts.ReadText("Registry host", r.Host, false, -1)
	if err != nil {
		return err
	}
	r.Host = strings.TrimSuffix(host, "/")

	shouldConfigureAuth, err := prompts.ReadBool("Configure registry authentication", r.Auth.SecretName != nil)
	if err != nil {
		return err
	}
	if shouldConfigureAuth {
		if direct {
			if err := configureAuthInline(r); err != nil {
				return err
			}
		} else {
			if err := configureAuthSecrets(c, r, kClient, authSecretNames); err != nil {
				return err
			}
		}
	} else {
		r.Auth = plug.Auth{}
	}

	if err := readArtifactRefs(r); err != nil {
		return err
	}

	log.InfoCLI(`
    The following validation types are available:
    - 'none': only the existence of the artifacts in the registry is validated
    - 'fast': the artifacts are pulled and fast layer, manifest, and config validation is performed
    - 'full': the artifacts are pulled and full layer, manifest, and config validation is performed
    `)
	vType, err := prompts.Select("Validation type", []string{string(plug.ValidationTypeNone), string(plug.ValidationTypeFast), string(plug.ValidationTypeFull)})
	if err != nil {
		return err
	}
	r.ValidationType = plug.ValidationType(vType)

	shouldConfigureSigVerification, err := prompts.ReadBool("Configure signature verification", r.SignatureVerification.SecretName != "")
	if err != nil {
		return err
	}
	if shouldConfigureSigVerification {
		if direct {
			err := configureSigVerification(r)
			if err != nil {
				return err
			}
		} else {
			err := configureSigVerificationSecrets(c, r, kClient, sigSecretNames)
			if err != nil {
				return err
			}
		}
	} else {
		r.SignatureVerification = plug.SignatureVerification{}
	}

	if c.CaCertPaths == nil {
		c.CaCertPaths = make(map[int]string, 0)
	}
	caCertPath := c.CaCertPaths[idx]
	caCertPath, _, caCertData, err := prompts.ReadCACert("Registry CA certificate filepath", caCertPath, "")
	if err != nil {
		return err
	}
	r.CaCert = base64.StdEncoding.EncodeToString(caCertData)
	c.CaCertPaths[idx] = caCertPath

	if idx == -1 {
		c.Validator.OciRegistryRules = append(c.Validator.OciRegistryRules, *r)
	} else {
		c.Validator.OciRegistryRules[idx] = *r
	}
	return nil
}

func readArtifactRefs(r *plug.OciRegistryRule) error {
	log.InfoCLI("Configure one or more OCI artifact(s) to validate.")

	// We've intentionally opted to not support prompting for validation type overrides per artifact

	log.InfoCLI(`
	Artifact references must include the registry host for the current rule,
	e.g. 'gcr.io/someimage:latest', not 'someimage:latest'.
	`)
	var defaultArtifacts string
	for _, a := range r.Artifacts {
		defaultArtifacts += r.Host + "/" + a.Ref + "\n"
	}
	artifacts, err := prompts.ReadTextSlice(
		"Artifact references", defaultArtifacts, "invalid artifact refs", `^`+r.Host+`/.*$`, false,
	)
	if err != nil {
		return err
	}
	r.Artifacts = make([]plug.Artifact, len(artifacts))
	for i, a := range artifacts {
		r.Artifacts[i] = plug.Artifact{
			Ref: strings.TrimPrefix(a, fmt.Sprintf("%s/", r.Host)),
		}
	}

	return nil
}

func readOciSecret(secret *components.Secret) error {
	var err error
	if secret.Name == "" {
		secret.Name, err = prompts.ReadK8sName("Secret Name", "", false)
		if err != nil {
			return err
		}
	} else {
		log.InfoCLI("Reconfiguring secret: %s", secret.Name)
	}

	// reconfigure and/or add basic auth
	if secret.BasicAuth == nil {
		secret.BasicAuth = &components.BasicAuth{}
	}
	addBasicAuth, err := prompts.ReadBool("Add basic auth to this secret", false)
	if err != nil {
		return err
	}
	if addBasicAuth {
		secret.BasicAuth.Username, secret.BasicAuth.Password, err = prompts.ReadBasicCreds(
			"Username", "Password", secret.BasicAuth.Username, secret.BasicAuth.Password, false, false,
		)
		if err != nil {
			return err
		}
	} else {
		secret.BasicAuth = nil
	}

	// reconfigure and/or add env vars
	if secret.Data == nil {
		secret.Data = make(map[string]string)
	}
	for k, v := range secret.Data {
		k, v, err = readKeyValue(k, v)
		if err != nil {
			return err
		}
		secret.Data[k] = v
	}
	addEnvVars, err := prompts.ReadBool("Add environment variables to this secret", false)
	if err != nil {
		return err
	}
	if !addEnvVars {
		return nil
	}
	for {
		k, v, err := readKeyValue("", "")
		if err != nil {
			return err
		}
		secret.Data[k] = v

		add, err := prompts.ReadBool("Add another environment variable to this secret", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readKeyValue(k, v string) (string, string, error) {
	key, err := prompts.ReadText("Key", k, false, -1)
	if err != nil {
		return "", "", err
	}
	value, err := prompts.ReadText("Value", v, false, -1)
	if err != nil {
		return "", "", err
	}
	return key, value, nil
}
