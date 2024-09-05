// Package clouds provides utility functions for interacting with clouds.
package clouds

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	string_utils "github.com/validator-labs/validatorctl/pkg/utils/string"
)

// GetVSphereDriver enables monkey-patching the vSphere driver for integration tests
var GetVSphereDriver = getVSphereDriver

// ReadVsphereAccountProps prompts the user to configure vSphere account properties
func ReadVsphereAccountProps(account *vsphere.Account) error {
	vcenterServer := account.Host
	username := account.Username
	password := account.Password

	// Identity Endpoint
	vcenterServer, err := prompts.ReadDomainsOrIPs(
		"vSphere Endpoint", vcenterServer, "vSphere Endpoint should be a valid FQDN or IP", false, 1,
	)
	if err != nil {
		return err
	}
	vcenterServer = string_utils.MultiTrim(vcenterServer, cfg.HTTPSchemes, []string{"/"})
	account.Host = vcenterServer

	// Username
	username, err = prompts.ReadTextRegex(
		"vSphere Username (with domain)", username, "Invalid username", cfg.VSphereUsernameRegex,
	)
	if err != nil {
		return err
	}
	account.Username = username

	// Password
	password, err = prompts.ReadPassword("vSphere Password", password, false, -1)
	if err != nil {
		return err
	}
	account.Password = password

	// Allow Insecure Connection
	insecure, err := prompts.ReadBool("Allow Insecure Connection (Bypass x509 Verification)", true)
	if err != nil {
		return err
	}
	account.Insecure = insecure

	// Validate
	if err := ValidateCloudAccountVsphere(*account); err != nil {
		val, err := handleCloudAccountError(err)
		if err != nil {
			return err
		}
		if val == "Continue" {
			return ReadVsphereAccountProps(account)
		}

		os.Exit(0)
	}

	return nil
}

func getVSphereDriver(account vsphere.Account) (vsphere.Driver, error) {
	return vsphere.NewVSphereDriver(account, "", logr.Logger{})
}

// ValidateCloudAccountVsphere validates that the provided vSphere cloud account is valid
func ValidateCloudAccountVsphere(account vsphere.Account) error {
	driver, err := GetVSphereDriver(account)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	isValid, err := driver.IsValidVSphereCredentials()
	if err != nil {
		return err
	}
	if !isValid {
		return errors.New("vSphere cloud account is not valid")
	}

	// ensure we have permissions to get tags
	_, err = driver.GetResourceTags(ctx, "Datacenter")
	if err != nil {
		return errors.Wrap(err, "vSphere cloud account failed to get tags")
	}

	return nil
}

func handleCloudAccountError(err error) (string, error) {
	errMsg := fmt.Sprintf("Cloud Account validation failed with error: %v. Please update account properties.", err)
	val, err := prompts.Select(errMsg, []string{"Continue", "Exit"})
	if err != nil {
		return "", err
	}
	return val, nil
}
