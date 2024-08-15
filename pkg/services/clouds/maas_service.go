package clouds

import (
	"fmt"
	"os"

	maasclient "github.com/canonical/gomaasclient/client"
	"github.com/canonical/gomaasclient/entity"
	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/validator-labs/validatorctl/pkg/components"
)

var (
	// GetMaasClient is defined to enable monkeypatching during testing
	GetMaasClient = getMaasClient
	host          = "https://maas.io/MAAS"
)

// ReadMaasClientProps gathers and validates MAAS client credentials
func ReadMaasClientProps(c *components.MaasPluginConfig) error {
	var err error
	c.MaasAPIToken, err = prompts.ReadPassword("MAAS API token", c.MaasAPIToken, false, -1)
	if err != nil {
		return fmt.Errorf("failed to prompt for password for MAAS API token: %w", err)
	}

	if c.Validator.Host != "" {
		host = c.Validator.Host
	}
	c.Validator.Host, err = prompts.ReadText("MAAS Domain", host, false, -1)
	if err != nil {
		return err
	}

	if err := validateMaasClient(c.Validator.Host, c.MaasAPIToken); err != nil {
		val, err := handleMaasClientError(err)
		if err != nil {
			return err
		}
		if val == "Continue" {
			return ReadMaasClientProps(c)
		}
		os.Exit(0)
	}

	return nil
}

func validateMaasClient(maasURL, maasToken string) error {
	client, err := GetMaasClient(maasURL, maasToken)
	if err != nil {
		return err
	}
	// gomaasclient doesnt provide a direct way to validate, so we manually check that requests return success
	_, err = client.Account.ListAuthorisationTokens()
	if err != nil {
		return err
	}
	if client == nil {
		return err
	}
	return nil
}

func handleMaasClientError(err error) (string, error) {
	errMsg := fmt.Sprintf("MAAS credentials validation failed with error: %v. Please update your credentials.", err)
	val, err := prompts.Select(errMsg, []string{"Continue", "Exit"})
	if err != nil {
		return "", err
	}
	return val, nil
}

// GetMaasResourcePools fetches a list of resource pools in the cluster
func GetMaasResourcePools(c *components.MaasPluginConfig) ([]string, error) {
	client, err := GetMaasClient(c.Validator.Host, c.MaasAPIToken)
	if err != nil {
		return nil, err
	}

	poolsObj, err := client.ResourcePools.Get()
	if err != nil {
		return nil, err
	}
	pools := make([]string, len(poolsObj))
	for i, p := range poolsObj {
		pools[i] = p.Name
	}

	return pools, nil
}

// GetMaasZones fetches a list of availability zones in the cluster
func GetMaasZones(c *components.MaasPluginConfig) ([]string, error) {
	client, err := GetMaasClient(c.Validator.Host, c.MaasAPIToken)
	if err != nil {
		return nil, err
	}
	zonesObj, err := client.Zones.Get()
	if err != nil {
		return nil, err
	}
	zones := make([]string, len(zonesObj))
	for i, z := range zonesObj {
		zones[i] = z.Name
	}
	return zones, nil
}

func getMaasClient(url, token string) (*maasclient.Client, error) {
	client, err := maasclient.GetClient(url, token, "2.0")
	if err != nil {
		return &maasclient.Client{}, err
	}
	return client, nil
}

// GetMockMaasClient returns a mock MAAS client for testing
func GetMockMaasClient(_, _ string) (*maasclient.Client, error) {
	mockClient := &maasclient.Client{
		Account: &MockMaasAccount{},
		ResourcePools: &MockMaasResourcePools{
			resourcePools: []entity.ResourcePool{
				{
					Name: "pool1",
					ID:   1,
				},
			},
		},
		Zones: &MockMaasZones{
			zones: []entity.Zone{
				{
					Name: "az1",
					ID:   1,
				},
			},
		},
	}
	return mockClient, nil
}

// MockMaasAccount replaces the maasclient.Account struct for integration testing
type MockMaasAccount struct {
	maasclient.Account
}

// ListAuthorisationTokens replaces the maasclient.Account.ListAuthorisationTokens method for integration testing
func (a *MockMaasAccount) ListAuthorisationTokens() ([]entity.AuthorisationTokenListItem, error) {
	return []entity.AuthorisationTokenListItem{}, nil
}

// MockMaasResourcePools replaces the maasclient.ResourcePools struct for integration testing
type MockMaasResourcePools struct {
	maasclient.ResourcePools
	resourcePools []entity.ResourcePool
}

// Get replaces the maasclient.ResourcePools.Get method for integration testing
func (r *MockMaasResourcePools) Get() ([]entity.ResourcePool, error) {
	return r.resourcePools, nil
}

// MockMaasZones replaces the maasclient.Zones struct for integration testing
type MockMaasZones struct {
	maasclient.Zones
	zones []entity.Zone
}

// Get replaces the maasclient.Zones.Get method for integration testing
func (z *MockMaasZones) Get() ([]entity.Zone, error) {
	return z.zones, nil
}
