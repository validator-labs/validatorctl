package components

import (
	"os"

	"emperror.dev/errors"
	"gopkg.in/yaml.v2"

	aws "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	azure "github.com/validator-labs/validator-plugin-azure/api/v1alpha1"
	network "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	oci "github.com/validator-labs/validator-plugin-oci/api/v1alpha1"
	vsphere "github.com/validator-labs/validator-plugin-vsphere/api/v1alpha1"
	vsphere_cloud "github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
	validator "github.com/validator-labs/validator/api/v1alpha1"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	env "github.com/validator-labs/validatorctl/pkg/services"
	"github.com/validator-labs/validatorctl/pkg/utils/crypto"
)

type ValidatorConfig struct {
	Release          *validator.HelmRelease `yaml:"helmRelease"`
	ReleaseSecret    *Secret                `yaml:"helmReleaseSecret"`
	KindConfig       KindConfig             `yaml:"kindConfig"`
	Kubeconfig       string                 `yaml:"kubeconfig"`
	SinkConfig       *SinkConfig            `yaml:"sinkConfig"`
	ProxyConfig      *ProxyConfig           `yaml:"proxyConfig"`
	ImageRegistry    string                 `yaml:"imageRegistry"`
	UseFixedVersions bool                   `yaml:"useFixedVersions"`

	AWSPlugin     *AWSPluginConfig     `yaml:"awsPlugin,omitempty"`
	NetworkPlugin *NetworkPluginConfig `yaml:"networkPlugin,omitempty"`
	OCIPlugin     *OCIPluginConfig     `yaml:"ociPlugin,omitempty"`
	VspherePlugin *VspherePluginConfig `yaml:"vspherePlugin,omitempty"`
	AzurePlugin   *AzurePluginConfig   `yaml:"azurePlugin,omitempty"`
}

func NewValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		// Base config
		Release:       &validator.HelmRelease{},
		ReleaseSecret: &Secret{},
		KindConfig: KindConfig{
			UseKindCluster: false,
		},
		SinkConfig: &SinkConfig{},
		ProxyConfig: &ProxyConfig{
			Env: &env.Env{},
		},
		// Plugin config
		AWSPlugin: &AWSPluginConfig{
			Release:       &validator.HelmRelease{},
			ReleaseSecret: &Secret{},
			IamCheck:      &IamCheck{},
			Validator:     &aws.AwsValidatorSpec{},
		},
		AzurePlugin: &AzurePluginConfig{
			Release:                &validator.HelmRelease{},
			ReleaseSecret:          &Secret{},
			Validator:              &azure.AzureValidatorSpec{},
			RuleTypes:              make(map[int]string),
			PlacementTypes:         make(map[int]string),
			StaticDeploymentTypes:  make(map[int]string),
			StaticDeploymentValues: make(map[int]*AzureStaticDeploymentValues),
		},
		NetworkPlugin: &NetworkPluginConfig{
			Release:       &validator.HelmRelease{},
			ReleaseSecret: &Secret{},
			Validator:     &network.NetworkValidatorSpec{},
		},
		OCIPlugin: &OCIPluginConfig{
			Release:       &validator.HelmRelease{},
			ReleaseSecret: &Secret{},
			Validator:     &oci.OciValidatorSpec{},
			CaCertPaths:   make(map[int]string),
		},
		VspherePlugin: &VspherePluginConfig{
			Release:       &validator.HelmRelease{},
			ReleaseSecret: &Secret{},
			Validator:     &vsphere.VsphereValidatorSpec{},
			Account:       &vsphere_cloud.VsphereCloudAccount{},
		},
	}
}

func (c *ValidatorConfig) AnyPluginEnabled() bool {
	return c.AWSPlugin.Enabled || c.NetworkPlugin.Enabled || c.VspherePlugin.Enabled || c.OCIPlugin.Enabled || c.AzurePlugin.Enabled
}

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

type KindConfig struct {
	UseKindCluster  bool   `yaml:"useKindCluster"`
	KindClusterName string `yaml:"kindClusterName"`
}

type ProxyConfig struct {
	Enabled bool     `yaml:"enabled"`
	Env     *env.Env `yaml:"env"`
}

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

type IamCheck struct {
	Enabled     bool             `yaml:"enabled"`
	IamRoleName string           `yaml:"iamRoleName"`
	Type        cfg.IamCheckType `yaml:"type"`
}

type AWSPluginConfig struct {
	Enabled            bool                   `yaml:"enabled"`
	Release            *validator.HelmRelease `yaml:"helmRelease"`
	ReleaseSecret      *Secret                `yaml:"helmReleaseSecret"`
	AccessKeyId        string                 `yaml:"accessKeyId,omitempty"`
	SecretAccessKey    string                 `yaml:"secretAccessKey,omitempty"`
	SessionToken       string                 `yaml:"sessionToken,omitempty"`
	ServiceAccountName string                 `yaml:"serviceAccountName,omitempty"`
	IamCheck           *IamCheck              `yaml:"iamCheck"`
	Validator          *aws.AwsValidatorSpec  `yaml:"validator"`
}

func (c *AWSPluginConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}

	accessKey, err := crypto.EncryptB64([]byte(c.AccessKeyId))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt access key id")
	}
	c.AccessKeyId = accessKey

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
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}

	bytes, err := crypto.DecryptB64(c.AccessKeyId)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt access key id")
	}
	c.AccessKeyId = string(*bytes)

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

type AzurePluginConfig struct {
	Enabled                bool                                 `yaml:"enabled"`
	Release                *validator.HelmRelease               `yaml:"helmRelease"`
	ReleaseSecret          *Secret                              `yaml:"helmReleaseSecret"`
	ServiceAccountName     string                               `yaml:"serviceAccountName,omitempty"`
	TenantID               string                               `yaml:"tenantId"`
	ClientID               string                               `yaml:"clientId"`
	ClientSecret           string                               `yaml:"clientSecret"`
	RuleTypes              map[int]string                       `yaml:"ruleTypes"`
	PlacementTypes         map[int]string                       `yaml:"placementTypes"`
	StaticDeploymentTypes  map[int]string                       `yaml:"staticDeploymentTypes"`
	StaticDeploymentValues map[int]*AzureStaticDeploymentValues `yaml:"staticDeploymentValues"`
	Validator              *azure.AzureValidatorSpec            `yaml:"validator"`
}

func (c *AzurePluginConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}

	clientSecret, err := crypto.EncryptB64([]byte(c.ClientSecret))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt Azure Client Secret")
	}
	c.ClientSecret = clientSecret

	return nil
}

func (c *AzurePluginConfig) decrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}

	bytes, err := crypto.DecryptB64(c.ClientSecret)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt Azure Client Secret")
	}
	c.ClientSecret = string(*bytes)

	return nil
}

type AzureStaticDeploymentValues struct {
	Subscription   string `yaml:"subscriptionUuid"`
	ResourceGroup  string `yaml:"resourceGroupUuid"`
	VirtualNetwork string `yaml:"virtualNetworkUuid"`
	Subnet         string `yaml:"subnetUuid"`
	ComputeGallery string `yaml:"computeGalleryUuid"`
}

type NetworkPluginConfig struct {
	Enabled       bool                          `yaml:"enabled"`
	Release       *validator.HelmRelease        `yaml:"helmRelease"`
	ReleaseSecret *Secret                       `yaml:"helmReleaseSecret"`
	Validator     *network.NetworkValidatorSpec `yaml:"validator"`
}

func (c *NetworkPluginConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}
	return nil
}

func (c *NetworkPluginConfig) decrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}
	return nil
}

type OCIPluginConfig struct {
	Enabled          bool                   `yaml:"enabled"`
	Release          *validator.HelmRelease `yaml:"helmRelease"`
	ReleaseSecret    *Secret                `yaml:"helmReleaseSecret"`
	Secrets          []*Secret              `yaml:"secrets,omitempty"`
	PublicKeySecrets []*PublicKeySecret     `yaml:"publicKeySecrets,omitempty"`
	CaCertPaths      map[int]string         `yaml:"caCertPaths,omitempty"`
	Validator        *oci.OciValidatorSpec  `yaml:"validator"`
}

func (c *OCIPluginConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}
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
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}
	for _, s := range c.Secrets {
		if s != nil {
			if err := s.decrypt(); err != nil {
				return err
			}
		}
	}
	return nil
}

type VspherePluginConfig struct {
	Enabled                     bool                               `yaml:"enabled"`
	Release                     *validator.HelmRelease             `yaml:"helmRelease"`
	ReleaseSecret               *Secret                            `yaml:"helmReleaseSecret"`
	Account                     *vsphere_cloud.VsphereCloudAccount `yaml:"account"`
	Validator                   *vsphere.VsphereValidatorSpec      `yaml:"validator"`
	VsphereEntityPrivilegeRules []VsphereEntityPrivilegeRule       `yaml:"vsphereEntityPrivilegeRules"`
	VsphereRolePrivilegeRules   []VsphereRolePrivilegeRule         `yaml:"vsphereRolePrivilegeRules"`
	VsphereTagRules             []VsphereTagRule                   `yaml:"vsphereTagRules"`
}

func (c *VspherePluginConfig) encrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.encrypt(); err != nil {
			return errors.Wrap(err, "failed to encrypt release secret configuration")
		}
	}
	if c.Account != nil {
		password, err := crypto.EncryptB64([]byte(c.Account.Password))
		if err != nil {
			return errors.Wrap(err, "failed to encrypt password")
		}
		c.Account.Password = password
	}
	return nil
}

func (c *VspherePluginConfig) decrypt() error {
	if c.ReleaseSecret != nil {
		if err := c.ReleaseSecret.decrypt(); err != nil {
			return errors.Wrap(err, "failed to decrypt release secret configuration")
		}
	}
	if c.Account != nil {
		bytes, err := crypto.DecryptB64(c.Account.Password)
		if err != nil {
			return errors.Wrap(err, "failed to decrypt password")
		}
		c.Account.Password = string(*bytes)
	}
	return nil
}

type VsphereEntityPrivilegeRule struct {
	vsphere.EntityPrivilegeValidationRule `yaml:",inline"`
	ClusterScoped                         bool   `yaml:"clusterScoped"`
	RuleType                              string `yaml:"ruleType"`
}

type VsphereRolePrivilegeRule struct {
	vsphere.GenericRolePrivilegeValidationRule `yaml:",inline"`
	Name                                       string `yaml:"name"`
}

type VsphereTagRule struct {
	vsphere.TagValidationRule `yaml:",inline"`
	RuleType                  string `yaml:"ruleType"`
}

type PublicKeySecret struct {
	Name string   `yaml:"name"`
	Keys []string `yaml:"keys"`
}

type Secret struct {
	Name       string `yaml:"name"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	CaCertFile string `yaml:"caCertFile"`
	Exists     bool   `yaml:"exists"`
}

func (s *Secret) ShouldCreate() bool {
	return !s.Exists && (s.Username != "" || s.Password != "" || s.CaCertFile != "")
}

func (s *Secret) encrypt() error {
	password, err := crypto.EncryptB64([]byte(s.Password))
	if err != nil {
		return errors.Wrap(err, "failed to encrypt password")
	}
	s.Password = password

	return nil
}

func (s *Secret) decrypt() error {
	bytes, err := crypto.DecryptB64(s.Password)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt password")
	}
	s.Password = string(*bytes)

	return nil
}

// NewValidatorFromConfig loads a validator configuration file from disk and decrypts it
func NewValidatorFromConfig(taskConfig *cfg.TaskConfig) (*ValidatorConfig, error) {
	c, err := LoadValidatorConfig(taskConfig)
	if err != nil {
		return nil, err
	}
	if err := c.decrypt(); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadValidatorConfig loads a validator configuration file from disk
func LoadValidatorConfig(taskConfig *cfg.TaskConfig) (*ValidatorConfig, error) {
	bytes, err := os.ReadFile(taskConfig.ConfigFile)
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
func SaveValidatorConfig(c *ValidatorConfig, taskConfig *cfg.TaskConfig) error {
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
	if err = os.WriteFile(taskConfig.ConfigFile, b, 0600); err != nil {
		return errors.Wrap(err, "failed to create validator config file")
	}
	log.InfoCLI("validator configuration file saved: %s", taskConfig.ConfigFile)
	return nil
}
