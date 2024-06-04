package extra

// TODO: put these in a better spot, currently just importing this path and naming it models to avoid changes
type V1Env struct {
	HTTPProxy       string  `yaml:"httpProxy,omitempty"`
	HTTPSProxy      string  `yaml:"httpsProxy,omitempty"`
	NoProxy         string  `yaml:"noProxy,omitempty"`
	PodCIDR         *string `yaml:"podCIDR"`
	ProxyCaCertData string  `yaml:"proxyCaCertData,omitempty"`
	ProxyCaCertName string  `yaml:"proxyCaCertName,omitempty"`
	ProxyCaCertPath string  `yaml:"proxyCaCertPath,omitempty"`
	ServiceIPRange  *string `yaml:"serviceIPRange"`
}

type V1VsphereCloudAccount struct {
	Insecure      bool    `yaml:"insecure"`
	Password      *string `yaml:"password"`
	Username      *string `yaml:"username"`
	VcenterServer *string `yaml:"vcenterServer"`
}

// Auth username/password/tls or other auth
type Auth struct {
	Username string    `json:"username"`
	Password string    `json:"password"`
	Tls      TlsConfig `json:"tls"`
}

// TlsConfig config
type TlsConfig struct {
	Enabled            bool   `json:"enabled" bson:"enabled"`
	Certificate        string `json:"certificate" bson:"certificate"`
	Key                string `json:"key" bson:"key"`
	Ca                 string `json:"ca" bson:"ca"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify" bson:"insecureSkipVerify"`
	CaFile             string `json:"caFile" bson:"caFile"`
}

type RegistryAuthMap map[string]Auth
