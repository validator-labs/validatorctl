package validator

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"

	"emperror.dev/errors"
	"github.com/mohae/deepcopy"
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

func readHelmRelease(name string, k8sClient kubernetes.Interface, vc *components.ValidatorConfig, r *vapi.HelmRelease, rs *components.Secret) error {
	log.Header(fmt.Sprintf("%s Helm Chart Configuration", name))
	var err error

	defaultRepo := fmt.Sprintf("%s/%s", cfg.ValidatorHelmRepository, name)
	defaultVersion := ""
	if r != nil && r.Chart.Repository != "" {
		defaultRepo = r.Chart.Repository
		defaultVersion = r.Chart.Version
	}

	r.Chart.Name = name
	rs.Name = fmt.Sprintf("validator-helm-release-%s", name)

	if vc.AirgapConfig.Enabled {
		r.Chart.Repository = vc.AirgapConfig.Hauler.ChartEndpoint()
		log.InfoCLI("Using local Hauler repository: %s", vc.AirgapConfig.Hauler.ChartEndpoint())
	} else {
		r.Chart.Repository, err = prompts.ReadText(fmt.Sprintf("%s Helm repository", name), defaultRepo, false, -1)
		if err != nil {
			return err
		}
	}

	if vc.UseFixedVersions {
		r.Chart.Version = cfg.ValidatorChartVersions[name]
		log.InfoCLI("Using fixed version: %s for %s chart", r.Chart.Version, r.Chart.Name)
	} else {
		versionPrompt := fmt.Sprintf("%s version", name)
		availableVersions, err := getReleasesFromHelmRepo(r.Chart.Repository)
		// Ignore error and fall back to reading version from the command line.
		// Errors may occur in air-gapped environments or misconfigured helm repos.
		if err != nil {
			log.InfoCLI("Failed to fetch chart versions from Helm repo due to error: %v. Falling back to manual input.", err)
		}
		if availableVersions != nil {
			r.Chart.Version, err = prompts.Select(versionPrompt, availableVersions)
			if err != nil {
				return err
			}
		} else {
			r.Chart.Version, err = prompts.ReadSemVer(versionPrompt, defaultVersion, "invalid Helm version")
			if err != nil {
				return err
			}
		}
	}

	if err := readHelmCredentials(r, rs, k8sClient, vc); err != nil {
		return err
	}

	return nil
}

func readHelmCredentials(r *vapi.HelmRelease, rs *components.Secret, k8sClient kubernetes.Interface, vc *components.ValidatorConfig) error {
	copyChart := false
	var err error

	if vc.Release != nil && r.Chart.Name != cfg.Validator {
		copyChart, err = prompts.ReadBool("Re-use security configuration from validator chart", true)
		if err != nil {
			return err
		}
	}
	if copyChart {
		rsCp := deepcopy.Copy(vc.ReleaseSecret).(*components.Secret)
		*rs = *rsCp
		r.Chart.AuthSecretName = vc.Release.Chart.AuthSecretName
		r.Chart.CAFile = vc.Release.Chart.CAFile
		r.Chart.InsecureSkipTLSVerify = vc.Release.Chart.InsecureSkipTLSVerify
		return nil
	}

	if rs.BasicAuth == nil {
		rs.BasicAuth = &components.BasicAuth{}
	}

	insecure, err := prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return err
	}
	if !insecure {
		rs.CaCertFile, _, _, err = prompts.ReadCACert("Helm repository CA certificate filepath", rs.CaCertFile, "")
		if err != nil {
			return err
		}
		r.Chart.CAFile = rs.CaCertFile
	}
	r.Chart.InsecureSkipTLSVerify = insecure

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
		r.Chart.AuthSecretName = rs.Name
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
