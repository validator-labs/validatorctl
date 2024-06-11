package validator

import (
	"encoding/base64"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	plug "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/utils/crypto"
)

const notApplicable = "N/A"

func readOciPlugin(vc *components.ValidatorConfig, k8sClient kubernetes.Interface) error {
	c := vc.OCIPlugin

	if err := readHelmRelease(cfg.ValidatorPluginOci, k8sClient, vc, c.Release, c.ReleaseSecret); err != nil {
		return err
	}
	authSecretNames, err := configureAuthSecrets(c, k8sClient)
	if err != nil {
		return err
	}
	sigVerificationSecretNames, err := configureSigVerificationSecrets(c, k8sClient)
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

// configureAuthSecrets prompts the user to configure credentials for OCI registries.
func configureAuthSecrets(c *components.OCIPluginConfig, k8sClient kubernetes.Interface) ([]string, error) {
	log.InfoCLI("Optionally configure secret(s) for private OCI registry authentication.")

	var err error
	addSecrets := true
	secretNames := make([]string, 0)
	secretNames = append(secretNames, notApplicable) // always provide the option to not use any secret

	for i, s := range c.Secrets {
		s := s
		if err := readSecret(s); err != nil {
			return nil, err
		}
		c.Secrets[i] = s
		secretNames = append(secretNames, s.Name)
	}

	if c.Secrets == nil {
		c.Secrets = make([]*components.Secret, 0)
	} else {
		addSecrets, err = prompts.ReadBool("Add another private OCI registry", false)
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
		if err := readSecret(s); err != nil {
			return nil, err
		}
		c.Secrets = append(c.Secrets, s)
		secretNames = append(secretNames, s.Name)
	}

	if k8sClient != nil {
		existingSecrets, err := services.GetSecretsWithKeys(k8sClient, cfg.Validator, cfg.ValidatorBasicAuthKeys)
		if err != nil {
			return nil, err
		}

		for _, s := range existingSecrets {
			secretNames = append(secretNames, s.Name)
		}
	}

	return secretNames, nil
}

// configureSigVerificationSecrets prompts the user to configure secrets containing public keys for use in signature verification.
func configureSigVerificationSecrets(c *components.OCIPluginConfig, k8sClient kubernetes.Interface) ([]string, error) {
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

	if k8sClient != nil {
		existingSecrets, err := services.GetSecretsWithRegexKeys(k8sClient, cfg.Validator, cfg.ValidatorPluginOciSigVerificationKeysRegex)
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

	r.Host, err = prompts.ReadText("Registry host", r.Host, false, -1)
	if err != nil {
		return err
	}

	authSecretName, err := prompts.Select("Registry authentication secret name, select N/A for public registries", authSecretNames)
	if err != nil {
		return err
	}
	if authSecretName != notApplicable {
		r.Auth = plug.Auth{SecretName: authSecretName}
	}

	if err := configureArtifacts(r); err != nil {
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

	caCertPath := c.CaCertPaths[idx]
	caCertPath, _, caCertData, err := crypto.ReadCACert("Registry CA certificate filepath", caCertPath, "")
	if err != nil {
		return err
	}
	r.CaCert = base64.StdEncoding.EncodeToString(caCertData)
	c.CaCertPaths[idx] = caCertPath

	return nil
}

func configureArtifacts(r *plug.OciRegistryRule) error {
	log.InfoCLI("Configure one or more OCI artifact(s) to validate.")

	var err error
	addArtifacts := true

	for i, a := range r.Artifacts {
		a := a
		if err := readArtifactRef(r, &a, i); err != nil {
			return err
		}
	}
	if r.Artifacts == nil {
		r.Artifacts = make([]plug.Artifact, 0)
	} else {
		addArtifacts, err = prompts.ReadBool("Add another artifact reference", false)
		if err != nil {
			return err
		}
	}
	if !addArtifacts {
		return nil
	}
	for {
		if err := readArtifactRef(r, &plug.Artifact{}, -1); err != nil {
			return err
		}
		add, err := prompts.ReadBool("Add another artifact reference", false)
		if err != nil {
			return err
		}
		if !add {
			break
		}
	}
	return nil
}

func readArtifactRef(r *plug.OciRegistryRule, a *plug.Artifact, idx int) error {
	var err error
	a.Ref, err = prompts.ReadTextRegex("Artifact ref", a.Ref, "Invalid artifact ref", prompts.ArtifactRefRegex)
	if err != nil {
		return err
	}
	a.LayerValidation, err = prompts.ReadBool("Enable full layer validation", false)
	if err != nil {
		return err
	}
	if idx == -1 {
		r.Artifacts = append(r.Artifacts, *a)
	} else {
		r.Artifacts[idx] = *a
	}
	return nil
}
