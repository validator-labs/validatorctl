package services

import (
	"time"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

type Env struct {
	HTTPProxy       string  `yaml:"httpProxy,omitempty"`
	HTTPSProxy      string  `yaml:"httpsProxy,omitempty"`
	NoProxy         string  `yaml:"noProxy,omitempty"`
	PodCIDR         *string `yaml:"podCIDR"`
	ProxyCaCertData string  `yaml:"proxyCaCertData,omitempty"`
	ProxyCaCertName string  `yaml:"proxyCaCertName,omitempty"`
	ProxyCaCertPath string  `yaml:"proxyCaCertPath,omitempty"`
	ServiceIPRange  *string `yaml:"serviceIPRange"`
}

func ReadProxyProps(e *Env) (err error) {
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
		e.NoProxy, err = prompts.EditFileValidatedByLine(cfg.NoProxyPrompt, e.NoProxy, ",", prompts.ValidateNoProxy, -1)
		if err != nil {
			return
		}

		// Proxy CA certificate
		caCertPath, caCertName, caCertData, err := prompts.ReadCACert("Proxy CA certificate filepath", e.ProxyCaCertPath, "")
		if err != nil {
			return err
		}
		e.ProxyCaCertData = string(caCertData)
		e.ProxyCaCertName = caCertName
		e.ProxyCaCertPath = caCertPath
	}

	return
}
