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

// Registry represents the generic configuration for a registry.
// If IsAirgapped is true, a local Hauler registry is used.
type Registry struct {
	Host                  string     `yaml:"host"`
	Port                  int        `yaml:"port"`
	BasicAuth             *BasicAuth `yaml:"basicAuth,omitempty"`
	InsecureSkipTLSVerify bool       `yaml:"insecureSkipTLSVerify"`
	CACert                *CACert    `yaml:"caCert,omitempty"`
	ReuseProxyCACert      bool       `yaml:"reuseProxyCACert,omitempty"`
	BaseContentPath       string     `yaml:"baseContentPath"`
	IsAirgapped           bool       `yaml:"isAirgapped"`
}

// Endpoint returns the base registry URL.
func (r *Registry) Endpoint() string {
	if r.IsAirgapped {
		return fmt.Sprintf("%s:%d", r.Host, r.Port)
	}
	return r.Host
}

// KindImage returns the image with the registry endpoint.
func (r *Registry) KindImage(image string) string {
	if r.IsAirgapped {
		return fmt.Sprintf("localhost:%d/%s", r.Port, image)
	}
	return fmt.Sprintf("%s/%s/%s", r.Host, r.BaseContentPath, image)
}

// ChartEndpoint returns the chart repository URL.
func (r *Registry) ChartEndpoint() string {
	if r.IsAirgapped {
		return fmt.Sprintf("oci://%s/hauler", r.Endpoint())
	}
	if r.BaseContentPath == "" {
		return fmt.Sprintf("oci://%s/charts", r.Host)
	}
	return fmt.Sprintf("oci://%s/%s/charts", r.Host, r.BaseContentPath)
}

// ImageEndpoint returns the image repository URL.
func (r *Registry) ImageEndpoint() string {
	return fmt.Sprintf("%s/%s", r.Endpoint(), cfg.ValidatorImageRepository)
}
