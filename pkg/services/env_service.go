// Package services provides utility functions for interacting with various services.
package services

import (
	"fmt"
	"time"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
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
func ReadHaulerProps(h *components.Hauler, e *components.Env) error {
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

	// basic auth
	if h.BasicAuth == nil {
		h.BasicAuth = &components.BasicAuth{}
	}
	h.BasicAuth.Username, h.BasicAuth.Password, err = prompts.ReadBasicCreds(
		"Username", "Password", h.BasicAuth.Username, h.BasicAuth.Password, true, false,
	)
	if err != nil {
		return err
	}

	// tls verification
	h.InsecureSkipTLSVerify, err = prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return err
	}
	if h.InsecureSkipTLSVerify {
		return nil
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
