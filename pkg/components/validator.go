// Package components provides functions for managing the validator components.
package components

import (
	"encoding/base64"
	"fmt"
	"os"

	"emperror.dev/errors"
	"gopkg.in/yaml.v2"

	aws "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	azure "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"
	maas "github.com/validator-labs/validator-plugin-maas/api/v1alpha1"
	network "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	oci "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"
	vsphereapi "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	validator "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// ValidatorConfig represents the validator configuration.
type ValidatorConfig struct {
	HelmConfig       *validator.HelmConfig  `yaml:"helmConfig"`
	Release          *validator.HelmRelease `yaml:"helmRelease"`
	ReleaseSecret    *Secret                `yaml:"helmReleaseSecret"`
	KindConfig       KindConfig             `yaml:"kindConfig"`
	Kubeconfig       string                 `yaml:"kubeconfig"`
	RegistryConfig   *RegistryConfig        `yaml:"registryConfig"`
	SinkConfig       *SinkConfig            `yaml:"sinkConfig"`
	ProxyConfig      *ProxyConfig           `yaml:"proxyConfig"`
	ImageRegistry    string                 `yaml:"imageRegistry"`
	UseFixedVersions bool                   `yaml:"useFixedVersions"`

	AWSPlugin     *AWSPluginConfig     `yaml:"awsPlugin,omitempty"`
	AzurePlugin   *AzurePluginConfig   `yaml:"azurePlugin,omitempty"`
	MaasPlugin    *MaasPluginConfig    `yaml:"maasPlugin,omitempty"`
	NetworkPlugin *NetworkPluginConfig `yaml:"networkPlugin,omitempty"`
	OCIPlugin     *OCIPluginConfig     `yaml:"ociPlugin,omitempty"`
	VspherePlugin *VspherePluginConfig `yaml:"vspherePlugin,omitempty"`
}

// NewValidatorConfig creates a new ValidatorConfig object.
func NewValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		// Base config
		HelmConfig: &validator.HelmConfig{},
		Release:    &validator.HelmRelease{},
		ReleaseSecret: &Secret{
			BasicAuth: &BasicAuth{},
			Data:      make(map[string]string),
		},
		KindConfig: KindConfig{
			UseKindCluster: false,
		},
		RegistryConfig: &RegistryConfig{
			Registry: &Registry{
				BasicAuth: &BasicAuth{},
				CACert:    &CACert{},
			},
		},
		SinkConfig: &SinkConfig{},
		ProxyConfig: &ProxyConfig{
			Env: &Env{
				ProxyCACert: &CACert{},
			},
		},
		// Plugin config
		AWSPlugin: &AWSPluginConfig{
			Release:   &validator.HelmRelease{},
			Validator: &aws.AwsValidatorSpec{},
		},
		AzurePlugin: &AzurePluginConfig{
			Release:   &validator.HelmRelease{},
			Validator: &azure.AzureValidatorSpec{},
		},
		MaasPlugin: &MaasPluginConfig{
			Release:   &validator.HelmRelease{},
			Validator: &maas.MaasValidatorSpec{},
		},
		NetworkPlugin: &NetworkPluginConfig{
			Release:       &validator.HelmRelease{},
			HTTPFileAuths: make([][]string, 0),
			Validator: &network.NetworkValidatorSpec{
				CACerts: network.CACertificates{},
			},
		},
		OCIPlugin: &OCIPluginConfig{
			Release:     &validator.HelmRelease{},
			Validator:   &oci.OciValidatorSpec{},
			CaCertPaths: make(map[int]string),
		},
		VspherePlugin: &VspherePluginConfig{
			Release:   &validator.HelmRelease{},
			Validator: &vsphereapi.VsphereValidatorSpec{},
		},
	}
}

// AnyPluginEnabled returns true if any plugin is enabled.
func (c *ValidatorConfig) AnyPluginEnabled() bool {
	return c.AWSPlugin.Enabled || c.NetworkPlugin.Enabled || c.VspherePlugin.Enabled || c.OCIPlugin.Enabled || c.AzurePlugin.Enabled || c.MaasPlugin.Enabled
}

// EnabledPluginsHaveRules returns true if all enabled plugins have at least one rule configured.
func (c *ValidatorConfig) EnabledPluginsHaveRules() (bool, []string) {
	var ok bool
	invalidPlugins := []string{}
	if c.AWSPlugin != nil && c.AWSPlugin.Enabled && c.AWSPlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.AWSPlugin.Validator.PluginCode())
	}
	if c.AzurePlugin != nil && c.AzurePlugin.Enabled && c.AzurePlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.AzurePlugin.Validator.PluginCode())
	}
	if c.MaasPlugin != nil && c.MaasPlugin.Enabled && c.MaasPlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.MaasPlugin.Validator.PluginCode())
	}
	if c.NetworkPlugin != nil && c.NetworkPlugin.Enabled && c.NetworkPlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.NetworkPlugin.Validator.PluginCode())
	}
	if c.OCIPlugin != nil && c.OCIPlugin.Enabled && c.OCIPlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.OCIPlugin.Validator.PluginCode())
	}
	if c.VspherePlugin != nil && c.VspherePlugin.Enabled && c.VspherePlugin.Validator.ResultCount() == 0 {
		invalidPlugins = append(invalidPlugins, c.VspherePlugin.Validator.PluginCode())
	}
	if len(invalidPlugins) == 0 {
		ok = true
	}
	return ok, invalidPlugins
}

// nolint:dupl
func (c *ValidatorConfig) decode() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decode(); err != nil {
			return errors.Wrap(err, "failed to decode release secret configuration")
		}
	}
	if err := c.SinkConfig.decode(); err != nil {
		return errors.Wrap(err, "failed to decode Sink configuration")
	}

	if c.AWSPlugin != nil {
		if err := c.AWSPlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode AWS plugin configuration")
		}
	}
	if c.AzurePlugin != nil {
		if err := c.AzurePlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode Azure plugin configuration")
		}
	}
	if c.MaasPlugin != nil {
		if err := c.MaasPlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode MAAS plugin configuration")
		}
	}
	if c.NetworkPlugin != nil {
		if err := c.NetworkPlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode Network plugin configuration")
		}
	}
	if c.OCIPlugin != nil {
		if err := c.OCIPlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode OCI plugin configuration")
		}
	}
	if c.VspherePlugin != nil {
		if err := c.VspherePlugin.decode(); err != nil {
			return errors.Wrap(err, "failed to decode vSphere plugin configuration")
		}
	}

	return nil
}

// nolint:dupl
func (c *ValidatorConfig) encode() error {
	if c.ReleaseSecret != nil {
		c.ReleaseSecret.encode()
	}
	c.SinkConfig.encode()

	if c.AWSPlugin != nil {
		c.AWSPlugin.encode()
	}
	if c.AzurePlugin != nil {
		c.AzurePlugin.encode()
	}
	if c.MaasPlugin != nil {
		c.MaasPlugin.encode()
	}
	if c.NetworkPlugin != nil {
		c.NetworkPlugin.encode()
	}
	if c.OCIPlugin != nil {
		c.OCIPlugin.encode()
	}
	if c.VspherePlugin != nil {
		c.VspherePlugin.encode()
	}

	return nil
}

// RegistryConfig represents the artifact registry configuration.
type RegistryConfig struct {
	Enabled  bool      `yaml:"enabled"`
	Registry *Registry `yaml:"registry"`
}

// ToHelmConfig converts the RegistryConfig to a HelmConfig.
func (c *RegistryConfig) ToHelmConfig() *validator.HelmConfig {
	hc := &validator.HelmConfig{
		Registry:              c.Registry.ChartEndpoint(),
		InsecureSkipTLSVerify: c.Registry.InsecureSkipTLSVerify,
	}

	if c.Registry.CACert != nil {
		hc.CAFile = c.Registry.CACert.Path
	}

	if c.BasicAuthEnabled() {
		hc.AuthSecretName = cfg.ValidatorHelmReleaseName
	}

	return hc
}

// BasicAuthEnabled returns true if basic auth is enabled on the RegistryConfig.
func (c *RegistryConfig) BasicAuthEnabled() bool {
	return c.Registry.BasicAuth != nil &&
		(c.Registry.BasicAuth.Username != "" || c.Registry.BasicAuth.Password != "")
}

// KindConfig represents the kind configuration.
type KindConfig struct {
	UseKindCluster  bool   `yaml:"useKindCluster"`
	KindClusterName string `yaml:"kindClusterName"`
}

// ProxyConfig represents the proxy configuration.
type ProxyConfig struct {
	Enabled bool `yaml:"enabled"`
	Env     *Env `yaml:"env"`
}

// SinkConfig represents the sink configuration.
type SinkConfig struct {
	Enabled      bool              `yaml:"enabled"`
	CreateSecret bool              `yaml:"createSecret"`
	SecretName   string            `yaml:"secretName"`
	Type         string            `yaml:"type"`
	Values       map[string]string `yaml:"values"`
}

func (c *SinkConfig) encode() {
	if c.Values == nil {
		return
	}
	for k, v := range c.Values {
		if v == "" {
			continue
		}
		value := base64.StdEncoding.EncodeToString([]byte(v))
		c.Values[k] = value
	}
}

func (c *SinkConfig) decode() error {
	if c.Values == nil {
		return nil
	}
	for k := range c.Values {
		if c.Values[k] == "" {
			continue
		}
		bytes, err := base64.StdEncoding.DecodeString(c.Values[k])
		if err != nil {
			return errors.Wrapf(err, "failed to decode SinkConfig key %s", k)
		}
		c.Values[k] = string(bytes)
	}
	return nil
}

// AWSPluginConfig represents the AWS plugin configuration.
type AWSPluginConfig struct {
	Enabled            bool                   `yaml:"enabled"`
	Release            *validator.HelmRelease `yaml:"helmRelease"`
	AccessKeyID        string                 `yaml:"accessKeyId,omitempty"`
	SecretAccessKey    string                 `yaml:"secretAccessKey,omitempty"`
	SessionToken       string                 `yaml:"sessionToken,omitempty"`
	ServiceAccountName string                 `yaml:"serviceAccountName,omitempty"`
	Validator          *aws.AwsValidatorSpec  `yaml:"validator"`
}

func (c *AWSPluginConfig) encode() {
	accessKey := base64.StdEncoding.EncodeToString([]byte(c.AccessKeyID))
	c.AccessKeyID = accessKey

	secretKey := base64.StdEncoding.EncodeToString([]byte(c.SecretAccessKey))
	c.SecretAccessKey = secretKey

	sessionToken := base64.StdEncoding.EncodeToString([]byte(c.SessionToken))
	c.SessionToken = sessionToken
}

func (c *AWSPluginConfig) decode() error {
	bytes, err := base64.StdEncoding.DecodeString(c.AccessKeyID)
	if err != nil {
		return errors.Wrap(err, "failed to decode access key id")
	}
	c.AccessKeyID = string(bytes)

	bytes, err = base64.StdEncoding.DecodeString(c.SecretAccessKey)
	if err != nil {
		return errors.Wrap(err, "failed to decode secret access key")
	}
	c.SecretAccessKey = string(bytes)

	bytes, err = base64.StdEncoding.DecodeString(c.SessionToken)
	if err != nil {
		return errors.Wrap(err, "failed to decode session token")
	}
	c.SessionToken = string(bytes)

	return nil
}

// AzurePluginConfig represents the Azure plugin configuration.
type AzurePluginConfig struct {
	Enabled            bool                      `yaml:"enabled"`
	Release            *validator.HelmRelease    `yaml:"helmRelease"`
	ServiceAccountName string                    `yaml:"serviceAccountName,omitempty"`
	Cloud              string                    `yaml:"cloud"`
	TenantID           string                    `yaml:"tenantId"`
	ClientID           string                    `yaml:"clientId"`
	ClientSecret       string                    `yaml:"clientSecret"`
	Validator          *azure.AzureValidatorSpec `yaml:"validator"`
}

func (c *AzurePluginConfig) encode() {
	clientSecret := base64.StdEncoding.EncodeToString([]byte(c.ClientSecret))
	c.ClientSecret = clientSecret
}

func (c *AzurePluginConfig) decode() error {
	bytes, err := base64.StdEncoding.DecodeString(c.ClientSecret)
	if err != nil {
		return errors.Wrap(err, "failed to decode Client Secret")
	}
	c.ClientSecret = string(bytes)

	return nil
}

// MaasPluginConfig represents the MAAS plugin configuration.
type MaasPluginConfig struct {
	Enabled   bool                    `yaml:"enabled"`
	Release   *validator.HelmRelease  `yaml:"helmRelease"`
	Validator *maas.MaasValidatorSpec `yaml:"validator"`
}

func (c *MaasPluginConfig) encode() {
	if c.Validator == nil {
		return
	}

	token := base64.StdEncoding.EncodeToString([]byte(c.Validator.Auth.APIToken))
	c.Validator.Auth.APIToken = token
}

func (c *MaasPluginConfig) decode() error {
	if c.Validator == nil {
		return nil
	}

	bytes, err := base64.StdEncoding.DecodeString(c.Validator.Auth.APIToken)
	if err != nil {
		return errors.Wrap(err, "failed to decode token")
	}
	c.Validator.Auth.APIToken = string(bytes)

	return nil
}

// NetworkPluginConfig represents the network plugin configuration.
type NetworkPluginConfig struct {
	Enabled       bool                          `yaml:"enabled"`
	Release       *validator.HelmRelease        `yaml:"helmRelease"`
	HTTPFileAuths [][]string                    `yaml:"httpFileAuths,omitempty"`
	Validator     *network.NetworkValidatorSpec `yaml:"validator"`
}

// AddDummyHTTPFileAuth adds a dummy HTTP file auth to the NetworkPluginConfig.
// This keeps the slice in sync when reconfiguring the plugin.
func (c *NetworkPluginConfig) AddDummyHTTPFileAuth() {
	c.HTTPFileAuths = append(c.HTTPFileAuths, []string{"", ""})
}

func (c *NetworkPluginConfig) encode() {
	if c.HTTPFileAuths == nil {
		return
	}

	for i, auth := range c.HTTPFileAuths {
		password := base64.StdEncoding.EncodeToString([]byte(auth[1]))
		c.HTTPFileAuths[i][1] = password
	}
}

func (c *NetworkPluginConfig) decode() error {
	if c.HTTPFileAuths == nil {
		return nil
	}

	for i, auth := range c.HTTPFileAuths {
		bytes, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			return fmt.Errorf("failed to decode password: %w", err)
		}
		c.HTTPFileAuths[i][1] = string(bytes)
	}

	return nil
}

// OCIPluginConfig represents the OCI plugin configuration.
type OCIPluginConfig struct {
	Enabled          bool                   `yaml:"enabled"`
	Release          *validator.HelmRelease `yaml:"helmRelease"`
	Secrets          []*Secret              `yaml:"secrets,omitempty"`
	PublicKeySecrets []*PublicKeySecret     `yaml:"publicKeySecrets,omitempty"`
	CaCertPaths      map[int]string         `yaml:"caCertPaths,omitempty"`
	Validator        *oci.OciValidatorSpec  `yaml:"validator"`
}

func (c *OCIPluginConfig) encode() {
	for _, s := range c.Secrets {
		if s != nil {
			s.encode()
		}
	}
}

func (c *OCIPluginConfig) decode() error {
	for _, s := range c.Secrets {
		if s != nil {
			if err := s.decode(); err != nil {
				return err
			}
		}
	}
	return nil
}

// VspherePluginConfig represents the vSphere plugin configuration.
type VspherePluginConfig struct {
	Enabled   bool                             `yaml:"enabled"`
	Release   *validator.HelmRelease           `yaml:"helmRelease"`
	Validator *vsphereapi.VsphereValidatorSpec `yaml:"validator"`
}

func (c *VspherePluginConfig) encode() {
	if c.Validator == nil {
		return
	}
	if c.Validator.Auth.Account == nil {
		return
	}

	password := base64.StdEncoding.EncodeToString([]byte(c.Validator.Auth.Account.Password))
	c.Validator.Auth.Account.Password = password
}

func (c *VspherePluginConfig) decode() error {
	if c.Validator == nil {
		return nil
	}
	if c.Validator.Auth.Account == nil {
		return nil
	}

	bytes, err := base64.StdEncoding.DecodeString(c.Validator.Auth.Account.Password)
	if err != nil {
		return errors.Wrap(err, "failed to decode password")
	}
	c.Validator.Auth.Account.Password = string(bytes)

	return nil
}

// PublicKeySecret represents a public key secret.
type PublicKeySecret struct {
	Name string   `yaml:"name"`
	Keys []string `yaml:"keys"`
}

// Secret represents a k8s secret.
type Secret struct {
	Name       string            `yaml:"name"`
	BasicAuth  *BasicAuth        `yaml:"basicAuth,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
	CaCertFile string            `yaml:"caCertFile,omitempty"`
	Exists     bool              `yaml:"exists"`
}

// ShouldCreate returns true if the secret should be created.
func (s *Secret) ShouldCreate() bool {
	return !s.Exists && (s.BasicAuth.Configured() || len(s.Data) > 0 || s.CaCertFile != "")
}

func (s *Secret) encode() {
	if s.BasicAuth != nil {
		s.BasicAuth.encode()
	}
	for k, v := range s.Data {
		v := base64.StdEncoding.EncodeToString([]byte(v))
		s.Data[k] = v
	}
}

func (s *Secret) decode() error {
	if s.BasicAuth != nil {
		if err := s.BasicAuth.decode(); err != nil {
			return err
		}
	}
	for k := range s.Data {
		bytes, err := base64.StdEncoding.DecodeString(s.Data[k])
		if err != nil {
			return fmt.Errorf("failed to decode value for secret key '%s': %w", k, err)
		}
		s.Data[k] = string(bytes)
	}
	return nil
}

// BasicAuth represents basic authentication credentials.
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Configured returns true if the basic auth is non-empty.
func (ba *BasicAuth) Configured() bool {
	return ba != nil && ba.Username != "" && ba.Password != ""
}

func (ba *BasicAuth) encode() {
	password := base64.StdEncoding.EncodeToString([]byte(ba.Password))
	ba.Password = password
}

func (ba *BasicAuth) decode() error {
	bytes, err := base64.StdEncoding.DecodeString(ba.Password)
	if err != nil {
		return fmt.Errorf("failed to decode password: %w", err)
	}
	ba.Password = string(bytes)

	return nil
}

// NewValidatorFromConfig loads a validator configuration file from disk and decrypts it
func NewValidatorFromConfig(tc *cfg.TaskConfig) (*ValidatorConfig, error) {
	c, err := LoadValidatorConfig(tc)
	if err != nil {
		return nil, err
	}
	if err := c.decode(); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadValidatorConfig loads a validator configuration file from disk
func LoadValidatorConfig(tc *cfg.TaskConfig) (*ValidatorConfig, error) {
	bytes, err := os.ReadFile(tc.ConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read validator config file")
	}
	c := &ValidatorConfig{}
	if err = yaml.Unmarshal(bytes, c); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal validator config")
	}
	return c, nil
}

// SaveValidatorConfig saves a validator configuration file to disk
func SaveValidatorConfig(c *ValidatorConfig, tc *cfg.TaskConfig) error {
	if err := c.encode(); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to marshal validator config")
	}
	if err := c.decode(); err != nil {
		return err
	}
	if err = os.WriteFile(tc.ConfigFile, b, 0600); err != nil {
		return errors.Wrap(err, "failed to create validator config file")
	}
	log.InfoCLI("\nvalidator configuration file saved: %s", tc.ConfigFile)
	return nil
}
