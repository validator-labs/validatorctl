package repo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"emperror.dev/errors"
	"github.com/pterm/pterm"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	aws_utils "github.com/validator-labs/validatorctl/pkg/utils/aws"
	"github.com/validator-labs/validatorctl/pkg/utils/crypto"
	palette_utils "github.com/validator-labs/validatorctl/pkg/utils/extra" // TODO: fix this name
)

// Scar Props
type ScarProps struct {
	ImageRegistryType RegistryType `yaml:"imageRegistryType"`
	PackRegistryType  RegistryType `yaml:"packRegistryType"`
	MongoDbProps      MongoDbProps `yaml:"mongo"`

	// unmarshalled from <scar_endpoint>/config/onprem-properties.yaml
	Accounts         Accounts     `yaml:"accounts"`
	Secrets          Secrets      `yaml:"secrets"`
	OCIImageRegistry *OCIRegistry `yaml:"ociImageRegistry"`
	OCIPackRegistry  *OCIRegistry `yaml:"ociPackRegistry"`
}

func (p *ScarProps) Decrypt() error {
	if p == nil {
		return nil
	}
	if p.OCIImageRegistry != nil {
		if err := p.OCIImageRegistry.Decrypt(); err != nil {
			return err
		}
	}
	if p.OCIPackRegistry != nil {
		if err := p.OCIPackRegistry.Decrypt(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ScarProps) Encrypt() error {
	if p == nil {
		return nil
	}
	if p.OCIImageRegistry != nil {
		if err := p.OCIImageRegistry.Encrypt(); err != nil {
			return err
		}
	}
	if p.OCIPackRegistry != nil {
		if err := p.OCIPackRegistry.Encrypt(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ScarProps) UpdateAuthTokens() error {
	if p == nil {
		return nil
	}
	if p.OCIPackRegistry != nil {
		if err := p.OCIPackRegistry.UpdateAuthTokens(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ScarProps) UpdatePasswords() error {
	if p == nil {
		return nil
	}
	log.Header("OCI Configuration")
	if p.OCIImageRegistry != nil {
		if err := p.OCIImageRegistry.UpdatePasswords(); err != nil {
			return err
		}
	}
	if p.OCIPackRegistry != nil {
		if err := p.OCIPackRegistry.UpdatePasswords(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ScarProps) DefaultRegistryMeta() (string, RegistryType, RegistryType) {
	var endpoint string
	imageRegistryType, packRegistryType := RegistryTypeSpectro, RegistryTypeSpectro

	if p.OCIPackRegistry != nil {
		if p.OCIPackRegistry.OCIRegistryECR != nil && p.OCIPackRegistry.OCIRegistryECR.Endpoint != "" {
			endpoint = p.OCIPackRegistry.OCIRegistryECR.Endpoint
			packRegistryType = RegistryTypeOCIECR
		} else if p.OCIPackRegistry.OCIRegistryBasic != nil && p.OCIPackRegistry.OCIRegistryBasic.Endpoint != "" {
			endpoint = p.OCIPackRegistry.OCIRegistryBasic.Endpoint
			packRegistryType = RegistryTypeOCI
		}
	}
	if p.OCIImageRegistry != nil {
		if p.OCIImageRegistry.OCIRegistryECR != nil && p.OCIImageRegistry.OCIRegistryECR.Endpoint != "" {
			imageRegistryType = RegistryTypeOCIECR
		} else if p.OCIPackRegistry.OCIRegistryBasic != nil && p.OCIImageRegistry.OCIRegistryBasic.Endpoint != "" {
			imageRegistryType = RegistryTypeOCI
		}
	}
	return endpoint, imageRegistryType, packRegistryType
}

// UpdateAuthJson generates credentials for the packserver-credential secret
func (p *ScarProps) UpdateAuthJson() error {
	if p.OCIImageRegistry != nil {
		if err := p.OCIImageRegistry.UpdateAuthJsonByType(p.ImageRegistryType); err != nil {
			return err
		}
	}
	if p.OCIPackRegistry != nil {
		if err := p.OCIPackRegistry.UpdateAuthJsonByType(p.PackRegistryType); err != nil {
			return err
		}
	}
	return nil
}

func (p *ScarProps) ImageUrl(name, tag string) string {
	url := fmt.Sprintf("%s/%s:%s", p.ociImageRegistryBasePath(), name, tag)
	return strings.ReplaceAll(url, "//", "/")
}

func (p *ScarProps) PackUrl(name, tag string) string {
	url := fmt.Sprintf("%s/spectro-packs/archive/%s:%s", p.ociPackRegistryBasePath(), name, tag)
	return strings.ReplaceAll(url, "//", "/")
}

func (p *ScarProps) ociImageRegistryBasePath() string {
	return fmt.Sprintf("%s/%s",
		p.OCIImageRegistry.Endpoint(p.ImageRegistryType),
		p.OCIImageRegistry.BaseContentPath(p.ImageRegistryType),
	)
}

func (p *ScarProps) ociPackRegistryBasePath() string {
	return fmt.Sprintf("%s/%s",
		p.OCIPackRegistry.Endpoint(p.PackRegistryType),
		p.OCIPackRegistry.BaseContentPath(p.PackRegistryType),
	)
}

// Secrets
type Secrets struct {
	ImagePull string `yaml:"imagePull"`
}

// Registry
type Registry struct {
	RegistryBase `yaml:",inline"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

type RegistryType string

const (
	RegistryTypeOCI     RegistryType = "OCI"
	RegistryTypeOCIECR  RegistryType = "OCI ECR"
	RegistryTypeSpectro RegistryType = "spectro"
)

var OCIRegistryTypes = []string{
	string(RegistryTypeOCI),
	string(RegistryTypeOCIECR),
}

type OCIRegistry struct {
	AuthJson         string            `yaml:"authJson"`
	OCIRegistryBasic *OCIRegistryBasic `yaml:"ociRegistry,omitempty"`
	OCIRegistryECR   *OCIRegistryECR   `yaml:"ociEcrRegistry,omitempty"`
}

func (r *OCIRegistry) Decrypt() error {
	bytes, err := crypto.DecryptB64(r.AuthJson)
	if err != nil {
		return err
	}
	r.AuthJson = string(*bytes)

	if r.OCIRegistryBasic != nil {
		bytes, err := crypto.DecryptB64(r.OCIRegistryBasic.Password)
		if err != nil {
			return err
		}
		r.OCIRegistryBasic.Password = string(*bytes)
	}

	if r.OCIRegistryECR != nil {
		bytes, err := crypto.DecryptB64(r.OCIRegistryECR.AccessKey)
		if err != nil {
			return err
		}
		r.OCIRegistryECR.AccessKey = string(*bytes)

		bytes, err = crypto.DecryptB64(r.OCIRegistryECR.SecretKey)
		if err != nil {
			return err
		}
		r.OCIRegistryECR.SecretKey = string(*bytes)
	}

	return nil
}

func (r *OCIRegistry) Encrypt() error {
	authJson, err := crypto.EncryptB64([]byte(r.AuthJson))
	if err != nil {
		return err
	}
	r.AuthJson = authJson

	if r.OCIRegistryBasic != nil {
		ociPassword, err := crypto.EncryptB64([]byte(r.OCIRegistryBasic.Password))
		if err != nil {
			return err
		}
		r.OCIRegistryBasic.Password = ociPassword
	}

	if r.OCIRegistryECR != nil {
		accessKey, err := crypto.EncryptB64([]byte(r.OCIRegistryECR.AccessKey))
		if err != nil {
			return err
		}
		r.OCIRegistryECR.AccessKey = accessKey

		secretKey, err := crypto.EncryptB64([]byte(r.OCIRegistryECR.SecretKey))
		if err != nil {
			return err
		}
		r.OCIRegistryECR.SecretKey = secretKey
	}

	return nil
}

func (r *OCIRegistry) UpdateAuthJsonByType(registryType RegistryType) error {
	switch registryType {
	case RegistryTypeOCI:
		auth, err := r.OCIRegistryBasic.Auth()
		if err != nil {
			return err
		}
		return r.UpdateAuthJson(r.OCIRegistryBasic.Endpoint, auth)
	case RegistryTypeOCIECR:
		auth, err := r.OCIRegistryECR.Auth()
		if err != nil {
			return err
		}
		return r.UpdateAuthJson(r.OCIRegistryECR.Endpoint, auth)
	}
	return nil
}

func (r *OCIRegistry) UpdateAuthJson(endpoint string, auth *palette_utils.Auth) error {
	registryCreds := palette_utils.RegistryAuthMap{endpoint: *auth}
	registryCredsJson, err := json.Marshal(registryCreds)
	if err != nil {
		return err
	}
	r.AuthJson = base64.StdEncoding.EncodeToString(registryCredsJson)
	return nil
}

func (r *OCIRegistry) UpdateAuthTokens() error {
	if r.OCIRegistryECR != nil {
		auth, err := r.OCIRegistryECR.Auth()
		if err != nil {
			return err
		}
		if err := r.UpdateAuthJson(r.OCIRegistryECR.Endpoint, auth); err != nil {
			return err
		}
	}
	return nil
}

func (r *OCIRegistry) UpdatePasswords() error {
	auth, err := r.decodeAuth()
	if err != nil {
		return err
	}
	if r.OCIRegistryBasic != nil {
		if err := r.OCIRegistryBasic.ReadCredentials(auth); err != nil {
			return err
		}
		if err := r.UpdateAuthJson(r.OCIRegistryBasic.Endpoint, auth); err != nil {
			return err
		}
	} else if r.OCIRegistryECR != nil {
		if err := r.OCIRegistryECR.ReadCredentials(auth); err != nil {
			return err
		}
		if err := r.UpdateAuthJson(r.OCIRegistryECR.Endpoint, auth); err != nil {
			return err
		}
	}
	return nil
}

func (r *OCIRegistry) decodeAuth() (*palette_utils.Auth, error) {
	authBytes, err := base64.StdEncoding.DecodeString(r.AuthJson)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode authJson")
	}
	authMap := &palette_utils.RegistryAuthMap{}
	if err := json.Unmarshal(authBytes, authMap); err != nil {
		return nil, err
	}
	var auth *palette_utils.Auth
	for _, a := range *authMap {
		a := a
		auth = &a
		break
	}
	return auth, nil
}

func (r *OCIRegistry) Endpoint(registryType RegistryType) string {
	var ep string
	switch registryType {
	case RegistryTypeOCI:
		ep = r.OCIRegistryBasic.Endpoint
	case RegistryTypeOCIECR:
		ep = r.OCIRegistryECR.Endpoint
	}
	ep = strings.TrimSuffix(ep, "/")
	ep = strings.TrimSuffix(ep, "/v2")
	return ep
}

func (r *OCIRegistry) BaseContentPath(registryType RegistryType) string {
	var baseContentPath string
	switch registryType {
	case RegistryTypeOCI:
		baseContentPath = r.OCIRegistryBasic.BaseContentPath
	case RegistryTypeOCIECR:
		baseContentPath = r.OCIRegistryECR.BaseContentPath
	}
	return strings.Trim(baseContentPath, "/")
}

func (r *OCIRegistry) CACertData(registryType RegistryType) string {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.CACertData
	case RegistryTypeOCIECR:
		return r.OCIRegistryECR.CACertData
	default:
		return ""
	}
}

func (r *OCIRegistry) CACertName(registryType RegistryType) string {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.CACertName
	case RegistryTypeOCIECR:
		return r.OCIRegistryECR.CACertName
	default:
		return ""
	}
}

func (r *OCIRegistry) CACertPath(registryType RegistryType) string {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.CACertPath
	case RegistryTypeOCIECR:
		return r.OCIRegistryECR.CACertPath
	default:
		return ""
	}
}

func (r *OCIRegistry) ReusedProxyCACert(registryType RegistryType) bool {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.ReusedProxyCACert
	case RegistryTypeOCIECR:
		return r.OCIRegistryECR.ReusedProxyCACert
	default:
		return false
	}
}

func (r *OCIRegistry) InsecureSkipVerify(registryType RegistryType) bool {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.InsecureSkipVerify
	case RegistryTypeOCIECR:
		return r.OCIRegistryECR.InsecureSkipVerify
	default:
		return false
	}
}

func (r *OCIRegistry) Username(registryType RegistryType) (string, error) {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.Username, nil
	case RegistryTypeOCIECR:
		username, _, err := aws_utils.GetECRCredentials(
			r.OCIRegistryECR.AccessKey,
			r.OCIRegistryECR.SecretKey,
			r.OCIRegistryECR.Region,
		)
		return username, err
	default:
		return "", nil
	}
}

func (r *OCIRegistry) Password(registryType RegistryType) (string, error) {
	switch registryType {
	case RegistryTypeOCI:
		return r.OCIRegistryBasic.Password, nil
	case RegistryTypeOCIECR:
		_, password, err := aws_utils.GetECRCredentials(
			r.OCIRegistryECR.AccessKey,
			r.OCIRegistryECR.SecretKey,
			r.OCIRegistryECR.Region,
		)
		return password, err
	default:
		return "", nil
	}
}

// OCI Registry - standard
type OCIRegistryBasic struct {
	RegistryBase     `yaml:",inline"`
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	BaseContentPath  string `yaml:"baseContentPath"`
	MirrorRegistries string `yaml:"mirrorRegistries"`
}

func (r *OCIRegistryBasic) Auth() (*palette_utils.Auth, error) {
	auth, err := initAuth(r.CACertData, r.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}
	auth.Username = r.Username
	auth.Password = r.Password
	return auth, nil
}

func (r *OCIRegistryBasic) ReadCredentials(auth *palette_utils.Auth) error {
	var err error
	r.Username, r.Password, err = prompts.ReadBasicCreds(
		"Registry Username", "Registry Password", auth.Username, auth.Password, true, false,
	)
	if err != nil {
		return err
	}
	auth.Username = r.Username
	auth.Password = r.Password
	return nil
}

// OCI Registry - ECR
type OCIRegistryECR struct {
	RegistryBase    `yaml:",inline"`
	AccessKey       string `yaml:"accessKey"`
	SecretKey       string `yaml:"secretKey"`
	Region          string `yaml:"region"`
	BaseContentPath string `yaml:"baseContentPath"`
	IsPrivate       bool   `yaml:"isPrivate"`
}

func (r *OCIRegistryECR) Auth() (*palette_utils.Auth, error) {
	auth, err := initAuth(r.CACertData, r.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}
	auth.Username, auth.Password, err = aws_utils.GetECRCredentials(
		r.AccessKey, r.SecretKey, r.Region,
	)
	if err != nil {
		return nil, err
	}
	return auth, nil
}

func (r *OCIRegistryECR) ReadCredentials(auth *palette_utils.Auth) error {
	var err error
	r.AccessKey, r.SecretKey, err = prompts.ReadBasicCreds(
		"Registry AccessKey", "Registry SecretKey", auth.Username, auth.Password, false, true,
	)
	if err != nil {
		return err
	}
	r.Region, err = prompts.ReadText("Registry Region", "", false, -1)
	if err != nil {
		return err
	}
	r.IsPrivate, err = prompts.ReadBool("ECR Registry is private", true)
	if err != nil {
		return err
	}
	auth.Username, auth.Password, err = aws_utils.GetECRCredentials(
		r.AccessKey, r.SecretKey, r.Region,
	)
	if err != nil {
		pterm.DefaultLogger.Error("failed to get ECR credentials", pterm.DefaultLogger.Args("error", err))
		return r.ReadCredentials(auth)
	}
	return nil
}

func initAuth(caCertBase64 string, insecure bool) (*palette_utils.Auth, error) {
	caCert, err := base64.StdEncoding.DecodeString(caCertBase64)
	if err != nil {
		return nil, err
	}
	auth := &palette_utils.Auth{
		Tls: palette_utils.TlsConfig{
			Ca:                 string(caCert),
			InsecureSkipVerify: insecure,
		},
	}
	return auth, nil
}

type RegistryBase struct {
	Name               string `yaml:"name"`
	Endpoint           string `yaml:"endpoint"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	CACertData         string `yaml:"caCert"`
	CACertName         string `yaml:"caCertName"`
	CACertPath         string `yaml:"caCertPath"`
	ReusedProxyCACert  bool   `yaml:"reusedProxyCACert"`
}

// MongoDB
type MongoDbProps struct {
	Url         string `yaml:"url"`
	CpuLimit    string `yaml:"cpuLimit"`
	MemoryLimit string `yaml:"memoryLimit"`
	PvcSize     string `yaml:"pvcSize"`
}

// Accounts
type Accounts struct {
	DevOps DevOps `yaml:"devOps"`
}

// DevOps
type DevOps struct {
	Aws       Aws       `yaml:"aws"`
	Azure     Azure     `yaml:"azure"`
	Gcp       Gcp       `yaml:"gcp"`
	Maas      Maas      `yaml:"maas"`
	Openstack Openstack `yaml:"openstack"`
	Vsphere   Vsphere   `yaml:"vsphere"`
}

// Aws
type Aws struct {
	AccessKey         string `yaml:"accessKey"`
	SecretKey         string `yaml:"secretKey"`
	GoldenImageRegion string `yaml:"goldenImageRegion"`
}

// Azure
type Azure struct {
	ClientId       string       `yaml:"clientId"`
	ClientSecret   string       `yaml:"clientSecret"`
	TenantId       string       `yaml:"tenantId"`
	SubscriptionId string       `yaml:"subscriptionId"`
	Storage        AzureStorage `yaml:"azureStorage"`
}

type AzureStorage struct {
	AccessKey   string `yaml:"accessKey"`
	StorageName string `yaml:"storageName"`
	Container   string `yaml:"container"`
}

// Gcp
type Gcp struct {
	JsonCredentials string `yaml:"jsonCredentials"`
	ImageProject    string `yaml:"imageProject"`
}

// Maas
type Maas struct {
	ImagesHostEndpoint string `yaml:"imagesHostEndpoint"`
}

// Openstack
type Openstack struct {
	ImagesHostEndpoint string `yaml:"imagesHostEndpoint"`
}

// vSphere
type Vsphere struct {
	ImagesHostEndpoint  string `yaml:"imagesHostEndpoint"`
	OverlordOvaLocation string `yaml:"overlordOvaLocation"`
}
