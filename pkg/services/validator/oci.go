package validator

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	plug "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

const notApplicable = "N/A"

func readOciPlugin(vc *components.ValidatorConfig, tc *cfg.TaskConfig, _ kubernetes.Interface) error {
	c := vc.OCIPlugin

	if !tc.Direct {
		if err := readHelmRelease(cfg.ValidatorPluginOci, vc, c.Release); err != nil {
			return fmt.Errorf("failed to read Helm release: %w", err)
		}
	}

	return nil
}

func readOciPluginRules(vc *components.ValidatorConfig, _ *cfg.TaskConfig, kClient kubernetes.Interface) error {
	log.Header("OCI Plugin Rule Configuration")
	c := vc.OCIPlugin

	authSecretNames, err := configureAuthSecrets(c, kClient)
	if err != nil {
		return err
	}
	sigVerificationSecretNames, err := configureSigVerificationSecrets(c, kClient)
	if err != nil {
		return err
	}
	if err := configureOciRegistryRules(c, authSecretNames, sigVerificationSecretNames); err != nil {
		return err
	}

	// impossible at present. uncomment if/when additional OCI rule types are added.
	// if c.Validator.ResultCount() == 0 {
	// 	return errNoRulesEnabled
	// }
	return nil
}

// nolint:dupl
// configureAuthSecrets prompts the user to configure credentials for OCI registries.
func configureAuthSecrets(c *components.OCIPluginConfig, kClient kubernetes.Interface) ([]string, error) {
	log.InfoCLI("Optionally configure secret(s) for private OCI registry authentication.")

	var err error
	addSecrets := true
	secretNames := make([]string, 0)
	secretNames = append(secretNames, notApplicable) // always provide the option to not use any secret

	for i, s := range c.Secrets {
		s := s
		if err := readOciSecret(s); err != nil {
			return nil, err
		}
		c.Secrets[i] = s
		secretNames = append(secretNames, s.Name)
	}

	if c.Secrets == nil {
		c.Secrets = make([]*components.Secret, 0)
	} else {
		addSecrets, err = prompts.ReadBool("Add another private OCI registry secret", false)
		if err != nil {
			return nil, err
		}
	}
	if !addSecrets {
		return secretNames, nil
	}

	adjective := "a"
	for {
		add, err := prompts.ReadBool(fmt.Sprintf("Add %s private OCI registry secret", adjective), false)
		if err != nil {
			return nil, err
		}
		if !add {
			break
		}
		adjective = "another"

		s := &components.Secret{}
		if err := readOciSecret(s); err != nil {
			return nil, err
		}
		c.Secrets = append(c.Secrets, s)
		secretNames = append(secretNames, s.Name)
	}

	if kClient != nil {
		existingSecrets, err := services.GetSecretsWithKeys(kClient, cfg.Validator, cfg.ValidatorBasicAuthKeys)
		if err != nil {
			return nil, err
		}

		for _, s := range existingSecrets {
			secretNames = append(secretNames, s.Name)
		}
	}

	return secretNames, nil
}

// nolint:dupl
// configureSigVerificationSecrets prompts the user to configure secrets containing public keys for use in signature verification.
func configureSigVerificationSecrets(c *components.OCIPluginConfig, kClient kubernetes.Interface) ([]string, error) {
	log.InfoCLI("Optionally configure secret(s) for OCI artifact signature verification.")

	var err error
	addSecrets := true
	secretNames := make([]string, 0)
	secretNames = append(secretNames, notApplicable) // always provide the option to not use any secret

	for i, s := range c.PublicKeySecrets {
		s := s
		if err := readPublicKeySecret(s); err != nil {
			return nil, err
		}
		c.PublicKeySecrets[i] = s
		secretNames = append(secretNames, s.Name)
	}

	if c.PublicKeySecrets == nil {
		c.PublicKeySecrets = make([]*components.PublicKeySecret, 0)
	} else {
		addSecrets, err = prompts.ReadBool("Add another signature verification secret", false)
		if err != nil {
			return nil, err
		}
	}
	if !addSecrets {
		return secretNames, nil
	}

	adjective := "a"
	for {
		add, err := prompts.ReadBool(fmt.Sprintf("Add %s signature verification secret", adjective), false)
		if err != nil {
			return nil, err
		}
		if !add {
			break
		}
		adjective = "another"

		s := &components.PublicKeySecret{}
		if err := readPublicKeySecret(s); err != nil {
			return nil, err
		}
		c.PublicKeySecrets = append(c.PublicKeySecrets, s)
		secretNames = append(secretNames, s.Name)
	}

	if kClient != nil {
		existingSecrets, err := services.GetSecretsWithRegexKeys(kClient, cfg.Validator, cfg.ValidatorPluginOciSigVerificationKeysRegex)
		if err != nil {
			return nil, err
		}

		for _, s := range existingSecrets {
			secretNames = append(secretNames, s.Name)
		}
	}

	return secretNames, nil
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

	pubKeys := make([]string, 0)
	for {
		pubKeyPath, err := prompts.ReadFilePath("Public Key file", "", "Invalid public key path", false)
		if err != nil {
			return err
		}
		pubKeyBytes, err := os.ReadFile(pubKeyPath) //#nosec
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, string(pubKeyBytes))

		add, err := prompts.ReadBool("Add another public key to this secret", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	secret.Keys = pubKeys
	return nil
}

func configureOciRegistryRules(c *components.OCIPluginConfig, authSecretNames, sigVerificationSecretNames []string) error {
	log.InfoCLI("OCI registry rule(s) ensure that specific OCI artifacts are present in an OCI registry.")

	var err error
	addRules := true

	for i, r := range c.Validator.OciRegistryRules {
		r := r
		if err := readOciRegistryRule(c, &r, i, authSecretNames, sigVerificationSecretNames); err != nil {
			return err
		}
	}

	if c.Validator.OciRegistryRules == nil {
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

	ruleIdx := len(c.Validator.OciRegistryRules)
	for {
		r := &plug.OciRegistryRule{}
		if err := readOciRegistryRule(c, r, ruleIdx, authSecretNames, sigVerificationSecretNames); err != nil {
			return err
		}
		c.Validator.OciRegistryRules = append(c.Validator.OciRegistryRules, *r)

		add, err := prompts.ReadBool("Add another OCI registry rule", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
		ruleIdx++
	}

	return nil
}

func readOciRegistryRule(c *components.OCIPluginConfig, r *plug.OciRegistryRule, idx int, authSecretNames, sigVerificationSecretNames []string) error {
	var err error

	if r.RuleName != "" {
		log.InfoCLI("Reconfiguring OCI registry rule for name: %s", r.RuleName)
	}

	r.RuleName, err = prompts.ReadText("Rule name", r.RuleName, false, -1)
	if err != nil {
		return err
	}

	log.InfoCLI("Example OCI registry hosts: gcr.io, quay.io, oci://myregistry:5000")
	host, err := prompts.ReadText("Registry host", r.Host, false, -1)
	if err != nil {
		return err
	}
	r.Host = strings.TrimSuffix(host, "/")

	authSecretName, err := prompts.Select("Registry authentication secret name, select N/A for public registries", authSecretNames)
	if err != nil {
		return err
	}
	if authSecretName != notApplicable {
		r.Auth = plug.Auth{SecretName: &authSecretName}
	}

	if err := readArtifactRefs(r); err != nil {
		return err
	}

	sigVerificationSecretName, err := prompts.Select("Signature verification secret name, select N/A to skip signature verification", sigVerificationSecretNames)
	if err != nil {
		return err
	}
	if sigVerificationSecretName != notApplicable {
		r.SignatureVerification = plug.SignatureVerification{
			Provider:   "cosign",
			SecretName: sigVerificationSecretName,
		}
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

	return nil
}

func readArtifactRefs(r *plug.OciRegistryRule) error {
	log.InfoCLI("Configure one or more OCI artifact(s) to validate.")

	// We've intentionally opted to not support prompting for full layer validation overrides per artifact

	log.InfoCLI(`
	Artifact references must include the registry host for the current rule,
	e.g. 'gcr.io/someimage:latest', not 'someimage:latest'.
	`)
	var defaultArtifacts string
	for _, a := range r.Artifacts {
		defaultArtifacts += a.Ref + "\n"
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

	log.InfoCLI("Full layer validation is enabled by default for all artifacts.")
	r.SkipLayerValidation, err = prompts.ReadBool("Disable full layer validation for all artifacts", false)
	if err != nil {
		return err
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
