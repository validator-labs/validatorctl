// Package clouds provides utility functions for interacting with clouds.
package clouds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
	"gopkg.in/ini.v1"

	"github.com/spectrocloud-labs/prompts-tui/prompts"
	"github.com/validator-labs/validator-plugin-aws/pkg/aws"
	"github.com/validator-labs/validatorctl/pkg/components"
)

const (
	awsCredsFilename = "credentials"
	awsNoCredsErr    = "get identity: get credentials: "
)

// ValidateAwsCreds validates the AWS credentials and returns an error if they are not valid.
func ValidateAwsCreds(c *components.AWSPluginConfig) error {
	api, err := aws.NewAPI(c.Validator.Auth, c.Validator.DefaultRegion)
	if err != nil {
		return err
	}
	_, err = api.IAM.GetUser(context.TODO(), nil)
	if err != nil && strings.Contains(err.Error(), awsNoCredsErr) {
		return err
	}
	return nil
}

// ReadAwsProfile reads the AWS credentials profile from the local .aws directory.
func ReadAwsProfile(c *components.AWSPluginConfig) (bool, error) {
	profiles, err := loadCredsProfiles()
	if err != nil || len(profiles) == 0 {
		return true, nil
	}

	profileNames := maps.Keys(profiles)
	profileNames = slices.Insert(profileNames, 0, "N/A")

	profile, err := prompts.Select("AWS Profile (select N/A to enter manually)", profileNames)
	if err != nil {
		return false, err
	}
	if profile == "N/A" {
		return false, nil
	}

	c.AccessKeyID = profiles[profile]["aws_access_key_id"]
	c.SecretAccessKey = profiles[profile]["aws_secret_access_key"]
	c.SessionToken = profiles[profile]["aws_session_token"]

	return false, nil
}

func loadCredsProfiles() (map[string]map[string]string, error) {
	credentialsPath := buildAwsFilePath(awsCredsFilename)
	creds, err := ini.Load(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS credentials file: %w", err)
	}

	awsProfiles := make(map[string]map[string]string, 0)
	for _, section := range creds.Sections() {
		awsProfiles[section.Name()] = map[string]string{
			"aws_access_key_id":     section.Key("aws_access_key_id").String(),
			"aws_secret_access_key": section.Key("aws_secret_access_key").String(),
			"aws_session_token":     section.Key("aws_session_token").String(),
		}
	}

	// dedupe default profile
	maps.DeleteFunc(awsProfiles, func(k string, _ map[string]string) bool {
		return k == "DEFAULT"
	})
	return awsProfiles, nil
}

func buildAwsFilePath(filename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aws", filename)
}
