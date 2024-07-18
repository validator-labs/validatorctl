package components

import (
	"fmt"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

// Env represents the environment configuration.
type Env struct {
	HTTPProxy      string  `yaml:"httpProxy,omitempty"`
	HTTPSProxy     string  `yaml:"httpsProxy,omitempty"`
	NoProxy        string  `yaml:"noProxy,omitempty"`
	PodCIDR        *string `yaml:"podCIDR"`
	ProxyCACert    *CACert `yaml:"proxyCaCert,omitempty"`
	ServiceIPRange *string `yaml:"serviceIPRange"`
}

// CACert represents a CA certificate.
type CACert struct {
	Data string `yaml:"data"`
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// Hauler represents the hauler configuration for air-gapped installs.
type Hauler struct {
	Host                  string     `yaml:"host"`
	Port                  int        `yaml:"port"`
	BasicAuth             *BasicAuth `yaml:"basicAuth,omitempty"`
	InsecureSkipTLSVerify bool       `yaml:"insecureSkipTLSVerify"`
	CACert                *CACert    `yaml:"caCert,omitempty"`
	ReuseProxyCACert      bool       `yaml:"reuseProxyCACert,omitempty"`
}

// Endpoint returns the base hauler registry URL.
func (h *Hauler) Endpoint() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

// KindImage returns the image with the local hauler registry endpoint.
func (h *Hauler) KindImage(image string) string {
	return fmt.Sprintf("localhost:%d/%s", h.Port, image)
}

// ChartEndpoint returns the hauler chart repository URL.
func (h *Hauler) ChartEndpoint() string {
	return fmt.Sprintf("oci://%s/hauler", h.Endpoint())
}

// ImageEndpoint returns the hauler image repository URL.
func (h *Hauler) ImageEndpoint() string {
	return fmt.Sprintf("%s/%s", h.Endpoint(), cfg.ValidatorImageRepository)
}
