// Package components provides functions for managing the validator components.
package components

import (
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
	"github.com/validator-labs/validatorctl/pkg/utils/crypto"
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
func (c *ValidatorConfig) decrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}
	if err := c.SinkConfig.decrypt(); err != nil {
		return errors.Wrap(err, "failed to decrypt Sink configuration")
	}

	if c.AWSPlugin != nil {
		if err := c.AWSPlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt AWS plugin configuration")
		}
	}
	if c.AzurePlugin != nil {
		if err := c.AzurePlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt Azure plugin configuration")
		}
	}
	if c.MaasPlugin != nil {
		if err := c.MaasPlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt MAAS plugin configuration")
		}
	}
	if c.NetworkPlugin != nil {
		if err := c.NetworkPlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt Network plugin configuration")
		}
	}
	if c.OCIPlugin != nil {
		if err := c.OCIPlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt OCI plugin configuration")
		}
	}
	if c.VspherePlugin != nil {
		if err := c.VspherePlugin.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt vSphere plugin configuration")
		}
	}

	return nil
}

// nolint:dupl
func (c *ValidatorConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}
	if err := c.SinkConfig.encrypt(); err != nil {
		return errors.Wrap(err, "failed to encrypt Sink configuration")
	}

	if c.AWSPlugin != nil {
		if err := c.AWSPlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt AWS plugin configuration")
		}
	}
	if c.AzurePlugin != nil {
		if err := c.AzurePlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt Azure plugin configuration")
		}
	}
	if c.MaasPlugin != nil {
		if err := c.MaasPlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt MAAS plugin configuration")
		}
	}
	if c.NetworkPlugin != nil {
		if err := c.NetworkPlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt Network plugin configuration")
		}
	}
	if c.OCIPlugin != nil {
		if err := c.OCIPlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt OCI plugin configuration")
		}
	}
	if c.VspherePlugin != nil {
		if err := c.VspherePlugin.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt vSphere plugin configuration")
		}
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

func (c *SinkConfig) encrypt() error {
	if c.Values == nil {
		return nil
	}
	for k, v := range c.Values {
		if v == "" {
			continue
		}
		value, err := crypto.EncryptB64([]byte(v))
		if err != nil {
			return errors.Wrapf(err, "failed to encrypt SinkConfig key %s", k)
		}
		c.Values[k] = value
	}
	return nil
}

func (c *SinkConfig) decrypt() error {
	if c.Values == nil {
		return nil
	}
	for k := range c.Values {
		if c.Values[k] == "" {
			continue
		}
		bytes, err := crypto.DecryptB64(c.Values[k])
		if err != nil {
			return errors.Wrapf(err, "failed to decrypt SinkConfig key %s", k)
		}
		c.Values[k] = string(*bytes)
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

func (c *AWSPluginConfig) encrypt() error {
	accessKey, err := crypto.EncryptB64([]byte(c.AccessKeyID))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt access key id")
	}
	c.AccessKeyID = accessKey

	secretKey, err := crypto.EncryptB64([]byte(c.SecretAccessKey))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt secret access key")
	}
	c.SecretAccessKey = secretKey

	sessionToken, err := crypto.EncryptB64([]byte(c.SessionToken))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt session token")
	}
	c.SessionToken = sessionToken

	return nil
}

func (c *AWSPluginConfig) decrypt() error {
	bytes, err := crypto.DecryptB64(c.AccessKeyID)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt access key id")
	}
	c.AccessKeyID = string(*bytes)

	bytes, err = crypto.DecryptB64(c.SecretAccessKey)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt secret access key")
	}
	c.SecretAccessKey = string(*bytes)

	bytes, err = crypto.DecryptB64(c.SessionToken)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt session token")
	}
	c.SessionToken = string(*bytes)

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

func (c *AzurePluginConfig) encrypt() error {
	clientSecret, err := crypto.EncryptB64([]byte(c.ClientSecret))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt Azure Client Secret")
	}
	c.ClientSecret = clientSecret

	return nil
}

func (c *AzurePluginConfig) decrypt() error {
	bytes, err := crypto.DecryptB64(c.ClientSecret)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt Azure Client Secret")
	}
	c.ClientSecret = string(*bytes)

	return nil
}

// MaasPluginConfig represents the MAAS plugin configuration.
type MaasPluginConfig struct {
	Enabled   bool                    `yaml:"enabled"`
	Release   *validator.HelmRelease  `yaml:"helmRelease"`
	Validator *maas.MaasValidatorSpec `yaml:"validator"`
}

func (c *MaasPluginConfig) encrypt() error {
	if c.Validator == nil {
		return nil
	}

	token, err := crypto.EncryptB64([]byte(c.Validator.Auth.APIToken))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt token")
	}
	c.Validator.Auth.APIToken = token

	return nil
}

func (c *MaasPluginConfig) decrypt() error {
	if c.Validator == nil {
		return nil
	}

	bytes, err := crypto.DecryptB64(c.Validator.Auth.APIToken)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt token")
	}
	c.Validator.Auth.APIToken = string(*bytes)

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

// HTTPFileAuthBytes converts a slice of basic authentication details from
// a [][]string to a [][][]byte. The former is required for YAML marshalling,
// encryption, and decryption, while the latter is required by the plugin's
// Validate method.
// TODO: refactor Network plugin to use [][]string.
func (c *NetworkPluginConfig) HTTPFileAuthBytes() [][][]byte {
	auths := make([][][]byte, len(c.HTTPFileAuths))
	for i, auth := range c.HTTPFileAuths {
		auths[i] = [][]byte{
			[]byte(auth[0]),
			[]byte(auth[1]),
		}
	}
	return auths
}

func (c *NetworkPluginConfig) encrypt() error {
	if c.HTTPFileAuths == nil {
		return nil
	}

	for i, auth := range c.HTTPFileAuths {
		password, err := crypto.EncryptB64([]byte(auth[1]))
		if err != nil {
			return fmt.Errorf("failed to encrypt password': %w", err)
		}
		c.HTTPFileAuths[i][1] = password
	}

	return nil
}

func (c *NetworkPluginConfig) decrypt() error {
	if c.HTTPFileAuths == nil {
		return nil
	}

	for i, auth := range c.HTTPFileAuths {
		bytes, err := crypto.DecryptB64(auth[1])
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		c.HTTPFileAuths[i][1] = string(*bytes)
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

// BasicAuths returns a slice of basic authentication details for each rule.
func (c *OCIPluginConfig) BasicAuths() map[string][]string {
	auths := make(map[string][]string, 0)

	for _, r := range c.Validator.OciRegistryRules {
		if r.Auth.SecretName != nil {
			for _, s := range c.Secrets {
				if s.Name != *r.Auth.SecretName {
					continue
				}

				if s.BasicAuth != nil {
					auths[r.Name()] = []string{s.BasicAuth.Username, s.BasicAuth.Password}
				}
			}
			continue
		}

		if r.Auth.Basic != nil {
			auths[r.Name()] = []string{r.Auth.Basic.Username, r.Auth.Basic.Password}
			continue
		}
	}

	return auths
}

// AllPubKeys returns a slice of public keys for each public key secret.
func (c *OCIPluginConfig) AllPubKeys() map[string][][]byte {
	pubKeys := make(map[string][][]byte, len(c.PublicKeySecrets))
	for _, s := range c.PublicKeySecrets {
		s := s
		keys := make([][]byte, len(s.Keys))
		for i, k := range s.Keys {
			keys[i] = []byte(k)
		}
		pubKeys[s.Name] = keys
	}
	return pubKeys
}

func (c *OCIPluginConfig) encrypt() error {
	for _, s := range c.Secrets {
		if s != nil {
			if err := s.encrypt(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *OCIPluginConfig) decrypt() error {
	for _, s := range c.Secrets {
		if s != nil {
			if err := s.decrypt(); err != nil {
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

func (c *VspherePluginConfig) encrypt() error {
	if c.Validator == nil {
		return nil
	}
	if c.Validator.Auth.Account == nil {
		return nil
	}

	password, err := crypto.EncryptB64([]byte(c.Validator.Auth.Account.Password))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt password")
	}
	c.Validator.Auth.Account.Password = password

	return nil
}

func (c *VspherePluginConfig) decrypt() error {
	if c.Validator == nil {
		return nil
	}
	if c.Validator.Auth.Account == nil {
		return nil
	}

	bytes, err := crypto.DecryptB64(c.Validator.Auth.Account.Password)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt password")
	}
	c.Validator.Auth.Account.Password = string(*bytes)

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

func (s *Secret) encrypt() error {
	if s.BasicAuth != nil {
		if err := s.BasicAuth.encrypt(); err != nil {
			return err
		}
	}
	for k, v := range s.Data {
		v, err := crypto.EncryptB64([]byte(v))
		if err != nil {
			return fmt.Errorf("failed to encrypt value for secret key '%s': %w", k, err)
		}
		s.Data[k] = v
	}
	return nil
}

func (s *Secret) decrypt() error {
	if s.BasicAuth != nil {
		if err := s.BasicAuth.decrypt(); err != nil {
			return err
		}
	}
	for k := range s.Data {
		bytes, err := crypto.DecryptB64(s.Data[k])
		if err != nil {
			return fmt.Errorf("failed to decrypt value for secret key '%s': %w", k, err)
		}
		s.Data[k] = string(*bytes)
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

func (ba *BasicAuth) encrypt() error {
	password, err := crypto.EncryptB64([]byte(ba.Password))
	if err != nil {
		return fmt.Errorf("failed to encrypt password': %w", err)
	}
	ba.Password = password

	return nil
}

func (ba *BasicAuth) decrypt() error {
	bytes, err := crypto.DecryptB64(ba.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}
	ba.Password = string(*bytes)

	return nil
}

// NewValidatorFromConfig loads a validator configuration file from disk and decrypts it
func NewValidatorFromConfig(tc *cfg.TaskConfig) (*ValidatorConfig, error) {
	c, err := LoadValidatorConfig(tc)
	if err != nil {
		return nil, err
	}
	if err := c.decrypt(); err != nil {
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
	if err := c.encrypt(); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to marshal validator config")
	}
	if err := c.decrypt(); err != nil {
		return err
	}
	if err = os.WriteFile(tc.ConfigFile, b, 0600); err != nil {
		return errors.Wrap(err, "failed to create validator config file")
	}
	log.InfoCLI("\nvalidator configuration file saved: %s", tc.ConfigFile)
	return nil
}
