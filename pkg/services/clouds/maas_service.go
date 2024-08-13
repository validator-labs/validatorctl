package clouds

import (
	"fmt"
	"os"

	maasclient "github.com/canonical/gomaasclient/client"
	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/validator-labs/validatorctl/pkg/components"
)

var (
	// ReadMaasClientProps is defined to enable monkeypatching during testing
	ReadMaasClientProps = readMaasClientProps
	host                = "https://maas.io/MAAS"
)

func readMaasClientProps(c *components.MaasPluginConfig) error {
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
			return readMaasClientProps(c)
		}
		os.Exit(0)
	}

	return nil
}

func validateMaasClient(maasURL, maasToken string) error {
	client, err := maasclient.GetClient(maasURL, maasToken, "2.0")
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
	client, err := maasclient.GetClient(c.Validator.Host, c.MaasAPIToken, "2.0")
	if err != nil {
		return []string{}, err
	}
	poolsObj, err := client.ResourcePools.Get()
	if err != nil {
		return []string{}, err
	}

	pools := make([]string, len(poolsObj))
	for i, p := range poolsObj {
		pools[i] = p.Name
	}
	return pools, nil
}
