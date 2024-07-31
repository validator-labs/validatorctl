package validator

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"

	"emperror.dev/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/repo"
	"sigs.k8s.io/yaml"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	vapi "github.com/validator-labs/validator/api/v1alpha1"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/services"
)

var errNoRulesEnabled = errors.New("no validation rules enabled")

func readHelmConfig(name string, k8sClient kubernetes.Interface, vc *components.ValidatorConfig, rs *components.Secret) error {
	var err error

	rs.Name = fmt.Sprintf("validator-helm-release-%s", name)
	if vc.RegistryConfig.Enabled {
		vc.HelmConfig.Registry = vc.RegistryConfig.Registry.ChartEndpoint() // TODO: verify this is correct. it should just be the endpoint with a scheme at the beginning (ie Registry.Endpoint?)
		log.InfoCLI("Using helm repository: %s", vc.RegistryConfig.Registry.ChartEndpoint())
	} else {
		vc.HelmConfig.Registry, err = prompts.ReadText("Helm registry", cfg.ValidatorHelmRepository, false, -1)
		if err != nil {
			return err
		}
	}

	vc.HelmConfig.InsecureSkipTLSVerify, err = prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return err
	}
	if !vc.HelmConfig.InsecureSkipTLSVerify {
		vc.HelmConfig.CAFile, _, _, err = prompts.ReadCACert("Helm repository CA certificate filepath", vc.HelmConfig.CAFile, "")
		if err != nil {
			return err
		}
	}

	useBasicAuth, err := prompts.ReadBool("Configure Helm basic authentication", false)
	if err != nil {
		return err
	}
	if useBasicAuth {
		var useExistingSecret bool

		if k8sClient != nil {
			log.InfoCLI(`
	Either specify credentials for basic authentication or provide
	the name of a secret in the target K8s cluster's %s namespace.
	If using an existing secret, it must contain the following keys: %+v.
	`, cfg.Validator, cfg.ValidatorBasicAuthKeys,
			)
			useExistingSecret, err = prompts.ReadBool("Use existing secret", true)
			if err != nil {
				return err
			}
			if useExistingSecret {
				secret, err := services.ReadSecret(k8sClient, cfg.Validator, false, cfg.ValidatorBasicAuthKeys)
				if err != nil {
					return err
				}
				rs.Name = secret.Name
				rs.BasicAuth.Username = string(secret.Data["username"])
				rs.BasicAuth.Password = string(secret.Data["password"])
				rs.Exists = true
			}
		}

		if k8sClient == nil || !useExistingSecret {
			if err = readBasicAuthSecret(rs); err != nil {
				return err
			}
		}
	}

	// Helm credentials and/or CA cert provided
	if rs.BasicAuth.Username != "" || rs.BasicAuth.Password != "" || rs.CaCertFile != "" {
		vc.HelmConfig.AuthSecretName = rs.Name
	}

	return nil
}

func readHelmRelease(name string, vc *components.ValidatorConfig, c *vapi.HelmRelease) error {
	log.Header(fmt.Sprintf("%s Helm Chart Configuration", name))

	c.Chart.Name = name
	c.Chart.Repository = name
	repoURL := fmt.Sprintf("%s/%s", vc.HelmConfig.Registry, c.Chart.Repository)

	if vc.UseFixedVersions {
		c.Chart.Version = cfg.ValidatorChartVersions[name]
		log.InfoCLI("Using fixed version: %s for %s chart", c.Chart.Version, repoURL)
	} else {
		versionPrompt := fmt.Sprintf("%s version", name)
		availableVersions, err := getReleasesFromHelmRepo(repoURL)
		// Ignore error and fall back to reading version from the command line.
		// Errors may occur in air-gapped environments or misconfigured helm repos.
		if err != nil {
			log.InfoCLI("Failed to fetch chart versions from Helm repo due to error: %v. Falling back to manual input.", err)
		}
		if availableVersions != nil {
			c.Chart.Version, err = prompts.Select(versionPrompt, availableVersions)
			if err != nil {
				return err
			}
		} else {
			c.Chart.Version, err = prompts.ReadSemVer(versionPrompt, c.Chart.Version, "invalid Helm version")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type indexFile struct {
	repo.IndexFile `yaml:",inline"`
}

func getReleasesFromHelmRepo(repoURL string) ([]string, error) {
	var helmIndexFile indexFile
	var versions []string

	indexURL := fmt.Sprintf("%s/index.yaml", repoURL)
	log.Debug("Fetching releases from Helm repository index: %s", indexURL)

	resp, err := http.Get(indexURL) //#nosec G107
	if err != nil {
		return nil, err // can happen in air-gapped scenarios
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("Failed to close response body for helm repo index: %v", err)
		}
	}()

	// if there is a failure in fetching the index.yaml, return err so the version can be picked manually
	// Don't have to worry about resp being nil as http.Get documentation mentions - When err is nil, resp always contains a non-nil resp.Body.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code: %d from repository", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(body, &helmIndexFile)
	if err != nil {
		return nil, err
	}

	for _, entry := range helmIndexFile.Entries {
		for _, chartVersion := range entry {
			versions = append(versions, fmt.Sprintf("v%s", chartVersion.Metadata.Version))
		}
	}

	return versions, nil
}

func readBasicAuthSecret(secret *components.Secret) error {
	var err error
	if secret.Name == "" {
		secret.Name, err = prompts.ReadK8sName("Secret Name", "", false)
		if err != nil {
			return err
		}
	} else {
		log.InfoCLI("Reconfiguring secret: %s", secret.Name)
	}

	if secret.BasicAuth == nil {
		secret.BasicAuth = &components.BasicAuth{}
	}

	secret.BasicAuth.Username, secret.BasicAuth.Password, err = prompts.ReadBasicCreds(
		"Username", "Password", secret.BasicAuth.Username, secret.BasicAuth.Password, false, false,
	)
	if err != nil {
		return err
	}

	return nil
}

// Gets a unique rule name from the user for a validator. Continues prompting the user until they
// provide a unique name.
//   - ruleNames - The rules that exist in the validator so far.
func getRuleName(ruleNames *[]string) (string, error) {
	for {
		name, err := prompts.ReadText("Rule name", "", false, 63)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read rule name")
		}
		// Ensure unique rule names
		if slices.Contains(*ruleNames, name) {
			log.ErrorCLI("Rule names must be unique", "current rule names", *ruleNames)
			continue
		}
		*ruleNames = append(*ruleNames, name)
		return name, nil
	}
}

func intToStringDefault(x int) string {
	var s string
	if x != 0 {
		s = strconv.FormatInt(int64(x), 10)
	}
	return s
}
