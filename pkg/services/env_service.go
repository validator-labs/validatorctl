// Package services provides utility functions for interacting with various services.
package services

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
	"github.com/validator-labs/validatorctl/pkg/utils/network"
)

// ReadProxyProps prompts the user to configure proxy settings.
func ReadProxyProps(e *components.Env) error {
	var err error

	// https_proxy
	e.HTTPSProxy, err = prompts.ReadURL("HTTPS Proxy", e.HTTPSProxy, "HTTPS Proxy should be a valid URL", true)
	if err != nil {
		return err
	}

	// http_proxy
	e.HTTPProxy, err = prompts.ReadURL("HTTP Proxy", e.HTTPProxy, "HTTP Proxy should be a valid URL", true)
	if err != nil {
		return err
	}

	if e.HTTPProxy != "" || e.HTTPSProxy != "" {
		// no_proxy
		log.InfoCLI("Configure NO_PROXY")
		time.Sleep(2 * time.Second)
		e.NoProxy, err = prompts.EditFileValidatedByLine(cfg.NoProxyPrompt, e.NoProxy, ",", prompts.ValidateNoProxy, -1)
		if err != nil {
			return err
		}

		// Proxy CA certificate
		if e.ProxyCACert == nil {
			e.ProxyCACert = &components.CACert{}
		}
		caCertPath, caCertName, caCertData, err := prompts.ReadCACert("Proxy CA certificate filepath", e.ProxyCACert.Path, "")
		if err != nil {
			return err
		}
		e.ProxyCACert.Data = string(caCertData)
		e.ProxyCACert.Name = caCertName
		e.ProxyCACert.Path = caCertPath
	}

	return nil
}

// ReadHaulerProps prompts the user to configure hauler settings.
func ReadHaulerProps(h *components.Registry, e *components.Env) error {
	var err error

	// registry
	if h.Host == "" {
		h.Host = network.GetDefaultHostAddress()
	}
	h.Host, err = prompts.ReadText("Hauler Host (IPv4 address of primary NIC)", h.Host, false, -1)
	if err != nil {
		return err
	}
	if h.Port == 0 {
		h.Port = 5000
	}
	h.Port, err = prompts.ReadInt("Hauler Port", fmt.Sprintf("%d", h.Port), 1024, 65535)
	if err != nil {
		return err
	}

	err = readAuthTLSProps(h)
	if err != nil {
		return err
	}

	// ca cert
	if e.ProxyCACert.Path != "" {
		h.ReuseProxyCACert, err = prompts.ReadBool("Reuse proxy CA cert for Hauler registry", true)
		if err != nil {
			return err
		}
	}
	if h.CACert == nil {
		h.CACert = &components.CACert{}
	}
	if h.ReuseProxyCACert {
		h.CACert = e.ProxyCACert
		return nil
	}
	caCertPath, caCertName, caCertData, err := prompts.ReadCACert("Hauler CA certificate filepath", h.CACert.Path, "")
	if err != nil {
		return err
	}

	if caCertPath == "" {
		h = nil
	} else {
		h.CACert.Data = string(caCertData)
		h.CACert.Name = caCertName
		h.CACert.Path = caCertPath
	}

	return nil
}

// ReadRegistryProps prompts the user to configure custom private registry settings.
func ReadRegistryProps(r *components.Registry, e *components.Env) error {
	ociURL, err := prompts.ReadURL(
		"Registry Endpoint", "", "Invalid Registry Endpoint. A scheme is required, e.g.: 'https://'.", false,
	)
	if err != nil {
		return err
	}

	parsedURL, err := url.Parse(ociURL)
	if err != nil {
		return err
	}
	r.Host = parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		r.Port = components.UnspecifiedPort
	} else {
		r.Port, err = strconv.Atoi(port)
		if err != nil {
			return err
		}
	}

	baseContentPath, err := prompts.ReadText("Registry Base Content Path", "", true, -1)
	if err != nil {
		return err
	}
	r.BaseContentPath = baseContentPath

	err = readAuthTLSProps(r)
	if err != nil {
		return err
	}

	// ca cert
	if e.ProxyCACert.Path != "" {
		r.ReuseProxyCACert, err = prompts.ReadBool("Reuse proxy CA cert for OCI registry", true)
		if err != nil {
			return err
		}
	}
	if r.CACert == nil {
		r.CACert = &components.CACert{}
	}
	if r.ReuseProxyCACert {
		r.CACert = e.ProxyCACert
	} else {
		caCertPath, caCertName, caCertData, err := prompts.ReadCACert("OCI registry CA certificate filepath", r.CACert.Path, "")
		if err != nil {
			return err
		}

		if caCertPath != "" {
			r.CACert.Data = string(caCertData)
			r.CACert.Name = caCertName
			r.CACert.Path = caCertPath
		}
	}

	return ensureDockerOciCaConfig(r.CACert, r.Host)
}

func readAuthTLSProps(r *components.Registry) error {
	var err error

	// basic auth
	if r.BasicAuth == nil {
		r.BasicAuth = &components.BasicAuth{}
	}
	r.BasicAuth.Username, r.BasicAuth.Password, err = prompts.ReadBasicCreds(
		"Username", "Password", r.BasicAuth.Username, r.BasicAuth.Password, true, false,
	)
	if err != nil {
		return err
	}

	// tls verification
	r.InsecureSkipTLSVerify, err = prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return err
	}
	if r.InsecureSkipTLSVerify {
		return nil
	}

	return nil
}

func ensureDockerOciCaConfig(caCert *components.CACert, endpoint string) error {
	// TODO: mock this function properly
	if os.Getenv("IS_TEST") == "true" {
		return nil
	}

	dockerOciCaDir := fmt.Sprintf("/etc/docker/certs.d/%s", endpoint)
	dockerOciCaPath := fmt.Sprintf("%s/%s", dockerOciCaDir, caCert.Name)

	if _, err := os.Stat(dockerOciCaPath); err != nil {
		log.InfoCLI("OCI CA configuration for Docker not found")

		if err := ensureDockerCACertDir(dockerOciCaDir); err != nil {
			return err
		}

		cmd := exec.Command("sudo", "cp", caCert.Path, dockerOciCaPath) //#nosec G204
		_, stderr, err := exec_utils.Execute(true, cmd)
		if err != nil {
			log.InfoCLI("Failed to configure OCI CA certificate")
			return errors.Wrap(err, stderr)
		}
		log.InfoCLI("Copied OCA CA certificate from %s to %s", caCert.Path, dockerOciCaPath)

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
			return errors.Wrap(err, stderr)
		}
		return createDockerCACertDir(path)
	}
	return nil
}

func createDockerCACertDir(path string) error {
	cmd := exec.Command("sudo", "mkdir", "-p", path) //#nosec G204
	_, stderr, err := exec_utils.Execute(true, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	log.InfoCLI("Created Docker OCI CA certificate directory: %s", path)
	return nil
}
