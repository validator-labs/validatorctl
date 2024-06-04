package services

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/pkg/repo"
	crypto_utils "github.com/validator-labs/validatorctl/pkg/utils/crypto"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
	models "github.com/validator-labs/validatorctl/pkg/utils/extra"
	palette_utils "github.com/validator-labs/validatorctl/pkg/utils/extra" // TODO: rename this
	"github.com/validator-labs/validatorctl/pkg/utils/file"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
)

func ReadEnvProps(env *models.V1Env) error {

	log.Header("Enter Environment Configuration")

	// Proxy env vars & CA cert
	if err := ReadProxyProps(env); err != nil {
		return err
	}

	// Pod CIDR
	podCIDR, err := prompts.ReadCIDRs("Pod CIDR", *env.PodCIDR, "Invalid Pod CIDR", false, 1)
	if err != nil {
		return err
	}
	env.PodCIDR = &podCIDR

	// Service CIDR
	serviceCIDR, err := prompts.ReadCIDRs("Service IP Range", *env.ServiceIPRange, "Invalid Service IP Range", false, 1)
	if err != nil {
		return err
	}
	env.ServiceIPRange = &serviceCIDR

	return nil
}

func ReadProxyProps(e *models.V1Env) (err error) {
	// https_proxy
	e.HTTPSProxy, err = prompts.ReadURL("HTTPS Proxy", e.HTTPSProxy, "HTTPS Proxy should be a valid URL", true)
	if err != nil {
		return
	}

	// http_proxy
	e.HTTPProxy, err = prompts.ReadURL("HTTP Proxy", e.HTTPProxy, "HTTP Proxy should be a valid URL", true)
	if err != nil {
		return
	}

	if e.HTTPProxy != "" || e.HTTPSProxy != "" {
		// no_proxy
		log.InfoCLI("Configure NO_PROXY")
		time.Sleep(2 * time.Second)
		e.NoProxy, err = file.EditFileValidated(cfg.NoProxyPrompt, e.NoProxy, ",", prompts.ValidateNoProxy, -1)
		if err != nil {
			return
		}

		// Proxy CA certificate
		caCertPath, caCertName, caCertData, err := crypto_utils.ReadCACert("Proxy CA certificate filepath", e.ProxyCaCertPath, "")
		if err != nil {
			return err
		}
		e.ProxyCaCertData = string(caCertData)
		e.ProxyCaCertName = caCertName
		e.ProxyCaCertPath = caCertPath
	}

	return
}

func ReadRegistryProps(e *models.V1Env, p *repo.ScarProps, tc *cfg.TaskConfig) error {

	log.Header("Enter Pack & Image Registry Configuration")

	defaultPackRegistry, defaultImageRegistryType, defaultPackRegistryType := p.DefaultRegistryMeta()
	if defaultPackRegistry != "" {
		prompt := fmt.Sprintf("Use default Pack Registry (%s)", defaultPackRegistry)
		useDefaultPackRegistry, err := prompts.ReadBool(prompt, true)
		if err != nil {
			return err
		}
		if useDefaultPackRegistry {
			p.ImageRegistryType = defaultImageRegistryType
			p.PackRegistryType = defaultPackRegistryType
			if err := p.UpdateAuthJson(); err != nil {
				return err
			}
			return nil
		}
	}

	// OCI pack registry
	var err error
	var packRegistryType *repo.RegistryType

	packRegistryType, p.OCIPackRegistry, err = ReadOCIRegistry(e, "Pack", tc)
	if err != nil {
		return err
	}
	p.PackRegistryType = *packRegistryType

	// OCI image registry
	log.InfoCLI("Enter 'Y' to pull images from public registries or 'N' to specify an OCI image registry")
	usePublicImageRegistries, err := prompts.ReadBool("Pull images from public registries", true)
	if err != nil {
		return err
	}
	if !usePublicImageRegistries {
		// For now, only OCI is supported for images - not OCI ECR
		p.ImageRegistryType = repo.RegistryTypeOCI

		var sharedRegistry bool
		if p.PackRegistryType == repo.RegistryTypeOCI {
			sharedRegistry, err = prompts.ReadBool("Use the same OCI Registry for packs & images", true)
			if err != nil {
				return err
			}
		}
		if !sharedRegistry {
			// Either pack registry is OCI ECR, or user wants separate OCI registries for packs & images
			_, p.OCIImageRegistry, err = ReadOCIRegistry(e, "Image", tc)
			if err != nil {
				return err
			}
		} else {
			p.OCIImageRegistry = p.OCIPackRegistry
		}
	} else {
		p.ImageRegistryType = repo.RegistryTypeSpectro
	}

	return nil
}

func ReadOCIRegistry(e *models.V1Env, registryType string, tc *cfg.TaskConfig) (*repo.RegistryType, *repo.OCIRegistry, error) {
	log.InfoCLI("\n==== %s OCI Registry Configuration ====", registryType)

	/*
		if tc.AirgapConfig != nil {
			useAirgapReg, err = prompts.ReadBool(fmt.Sprintf("Use local, air-gapped %s Registry", registryType), true)
			if err != nil {
				return nil, nil, err
			}
			caPathOverride = tc.AirgapConfig.CaCertPath
		}
	*/

	var registryOciType repo.RegistryType
	if registryType == "Pack" /*&& !useAirgapReg*/ {
		ociType, err := prompts.Select("Registry Type", repo.OCIRegistryTypes)
		if err != nil {
			return nil, nil, err
		}
		registryOciType = repo.RegistryType(ociType)
	} else {
		registryOciType = repo.RegistryTypeOCI // OCI ECR not supported for images
	}

	name, err := prompts.ReadText("Registry Name", "", false, -1)
	if err != nil {
		return nil, nil, err
	}

	endpoint, err := prompts.ReadURL(
		"Registry Endpoint", "", "Invalid Registry Endpoint. A scheme is required, e.g.: 'https://'.", false,
	)
	if err != nil {
		return nil, nil, err
	}
	endpoint = string_utils.MultiTrim(endpoint, cfg.HTTPSchemes, nil)

	baseContentPath, err := prompts.ReadText("Registry Base Content Path", "", true, -1)
	if err != nil {
		return nil, nil, err
	}
	insecure, err := prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return nil, nil, err
	}

	var caCertData []byte
	var caCertName, caPath string
	var reuseProxyCaCert bool

	if !insecure {
		if e.ProxyCaCertPath != "" {
			prompt := fmt.Sprintf("Reuse proxy CA cert for %s registry", registryType)
			reuseProxyCaCert, err = prompts.ReadBool(prompt, true)
			if err != nil {
				return nil, nil, err
			}
		}
		if reuseProxyCaCert {
			caCertData = []byte(e.ProxyCaCertData)
			caCertName = e.ProxyCaCertName
			caPath = e.ProxyCaCertPath
		} else {
			caPath, caCertName, caCertData, err = crypto_utils.ReadCACert("Registry CA certificate Filepath", "", "")
			if err != nil {
				return nil, nil, err
			}
		}
		// ensure Docker OCI CA config regardless of whether we are reusing the proxy CA cert
		if registryType == "Image" {
			if err := EnsureDockerOciCaConfig(caCertName, caPath, endpoint); err != nil {
				return nil, nil, err
			}
		}
	}

	var mirrorRegistries string

	if registryType == "Image" {
		var err error
		log.InfoCLI("Configure registry mirror(s)")
		time.Sleep(2 * time.Second)
		defaultMirrorRegistries := generateMirrorRegistries(endpoint, baseContentPath)
		mirrorRegistries, err = file.EditFileValidated(
			cfg.RegistryMirrorPrompt, defaultMirrorRegistries, ",", validateMirrorRegistry, -1,
		)
		if err != nil {
			log.Error("Error auto-generating Registry Mirror config: %v", err)
			return nil, nil, err
		}
	}

	auth := &palette_utils.Auth{
		Tls: palette_utils.TlsConfig{
			Ca:                 string(caCertData),
			InsecureSkipVerify: insecure,
		},
	}

	ociRegistry := &repo.OCIRegistry{}

	switch registryOciType {
	case repo.RegistryTypeOCI:
		ociRegistry.OCIRegistryBasic = &repo.OCIRegistryBasic{}
		ociRegistry.OCIRegistryBasic.Name = name
		ociRegistry.OCIRegistryBasic.Endpoint = endpoint
		ociRegistry.OCIRegistryBasic.BaseContentPath = baseContentPath
		ociRegistry.OCIRegistryBasic.InsecureSkipVerify = insecure
		ociRegistry.OCIRegistryBasic.CACertData = base64.StdEncoding.EncodeToString(caCertData)
		ociRegistry.OCIRegistryBasic.CACertName = caCertName
		ociRegistry.OCIRegistryBasic.CACertPath = caPath
		ociRegistry.OCIRegistryBasic.ReusedProxyCACert = reuseProxyCaCert
		ociRegistry.OCIRegistryBasic.MirrorRegistries = mirrorRegistries
		if err := ociRegistry.OCIRegistryBasic.ReadCredentials(auth); err != nil {
			return nil, nil, err
		}
	case repo.RegistryTypeOCIECR:
		ociRegistry.OCIRegistryECR = &repo.OCIRegistryECR{}
		ociRegistry.OCIRegistryECR.Name = name
		ociRegistry.OCIRegistryECR.Endpoint = endpoint
		ociRegistry.OCIRegistryECR.BaseContentPath = baseContentPath
		ociRegistry.OCIRegistryECR.InsecureSkipVerify = insecure
		ociRegistry.OCIRegistryECR.CACertData = base64.StdEncoding.EncodeToString(caCertData)
		ociRegistry.OCIRegistryECR.CACertName = caCertName
		ociRegistry.OCIRegistryECR.CACertPath = caPath
		ociRegistry.OCIRegistryECR.ReusedProxyCACert = reuseProxyCaCert
		if err := ociRegistry.OCIRegistryECR.ReadCredentials(auth); err != nil {
			return nil, nil, err
		}
	}

	if err := ociRegistry.UpdateAuthJson(endpoint, auth); err != nil {
		return nil, nil, err
	}

	return &registryOciType, ociRegistry, nil
}

// generateMirrorRegistries returns a comma-separated string of registry mirrors
func generateMirrorRegistries(registryEndpoint string, baseContentPath string) string {
	if registryEndpoint == "" {
		return ""
	}

	mirrorRegistries := make([]string, 0)
	for _, registry := range cfg.RegistryMirrors {
		// Generate the endpoint for OCI format (with /v2)
		registryMirrorEndpoint := fmt.Sprintf("%s/v2", registryEndpoint)
		if baseContentPath != "" {
			// Optionally add a base content path
			registryMirrorEndpoint = fmt.Sprintf("%s/%s", registryMirrorEndpoint, baseContentPath)
		}
		mirrorRegistries = append(mirrorRegistries,
			fmt.Sprintf("%s%s%s", registry, cfg.RegistryMirrorSeparator, registryMirrorEndpoint),
		)
	}
	return strings.Join(mirrorRegistries, ",")
}

func validateMirrorRegistry(s string) error {
	parts := strings.Split(s, cfg.RegistryMirrorSeparator)
	if len(parts) != 2 {
		log.InfoCLI("Invalid registry mirror: %s. Missing separator: '%s'", s, cfg.RegistryMirrorSeparator)
		return prompts.ValidationError
	}
	_, err := url.Parse(parts[1])
	if err != nil {
		log.InfoCLI("Invalid registry mirror: %s. Failed to parse endpoint %s: %v", s, parts[1], err)
		return prompts.ValidationError
	}
	return nil
}

func EnsureDockerOciCaConfig(caCertName, caPath, endpoint string) error {
	// TODO: mock this function properly
	if os.Getenv("IS_TEST") == "true" {
		return nil
	}

	dockerOciCaDir := fmt.Sprintf("/etc/docker/certs.d/%s", endpoint)
	dockerOciCaPath := fmt.Sprintf("%s/%s", dockerOciCaDir, caCertName)

	if _, err := os.Stat(dockerOciCaPath); err != nil {
		log.InfoCLI("OCI CA configuration for Docker not found")

		if err := ensureDockerCACertDir(dockerOciCaDir); err != nil {
			return err
		}

		cmd := exec.Command("sudo", "cp", caPath, dockerOciCaPath) //#nosec G204
		_, stderr, err := exec_utils.Execute(true, cmd)
		if err != nil {
			log.InfoCLI("Failed to configure OCI CA certificate")
			return errors.Wrap(err, stderr)
		}
		log.InfoCLI("Copied OCA CA certificate from %s to %s", caPath, dockerOciCaPath)

		log.InfoCLI("Restarting Docker...")
		cmd = exec.Command("sudo", "systemctl", "daemon-reload")
		_, stderr, err = exec_utils.Execute(true, cmd)
		if err != nil {
			log.InfoCLI("Failed to reload systemd manager configuration")
			log.InfoCLI("Please execute 'sudo systemctl daemon-reload' manually and retry")
			return errors.Wrap(err, stderr)
		}

		cmd = exec.Command("sudo", "systemctl", "restart", "docker")
		_, stderr, err = exec_utils.Execute(true, cmd)
		if err != nil {
			log.InfoCLI("Failed to restart Docker")
			log.InfoCLI("Please execute 'sudo systemctl restart docker' manually and retry")
			return errors.Wrap(err, stderr)
		}
		log.InfoCLI("Configured OCA CA certificate for Docker")
	}
	return nil
}

func ensureDockerCACertDir(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return createDockerCACertDir(path)
	}
	if !fi.IsDir() {
		cmd := exec.Command("sudo", "rm", "-f", path) //#nosec G204
		_, stderr, err := exec_utils.Execute(true, cmd)
		if err != nil {
			return errors.Wrapf(err, stderr)
		}
		return createDockerCACertDir(path)
	}
	return nil
}

func createDockerCACertDir(path string) error {
	cmd := exec.Command("sudo", "mkdir", "-p", path) //#nosec G204
	_, stderr, err := exec_utils.Execute(true, cmd)
	if err != nil {
		return errors.Wrapf(err, stderr)
	}
	log.InfoCLI("Created Docker OCI CA certificate directory: %s", path)
	return nil
}
