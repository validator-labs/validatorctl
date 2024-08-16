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
	vpawsapi "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	"github.com/validator-labs/validator-plugin-aws/pkg/aws"
	"github.com/validator-labs/validatorctl/pkg/components"
)

const (
	awsCredsFilename  = "credentials"
	awsConfigFilename = "config"
	awsNoCredsErr     = "get identity: get credentials: "
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
	profiles, err := loadAwsCredsProfiles()
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

func loadAwsCredsProfiles() (map[string]map[string]string, error) {
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

// ReadAwsSTSProfile reads the AWS STS config from the local .aws directory.
func ReadAwsSTSProfile(c *components.AWSPluginConfig) error {
	profiles, err := loadAwsSTSProfiles()
	if err != nil || len(profiles) == 0 {
		return nil
	}

	profileNames := maps.Keys(profiles)
	profileNames = slices.Insert(profileNames, 0, "N/A")

	profile, err := prompts.Select("AWS STS profile (select N/A to enter manually)", profileNames)
	if err != nil {
		return err
	}
	if profile == "N/A" {
		return nil
	}

	c.Validator.Auth.StsAuth.RoleArn = profiles[profile].RoleArn
	c.Validator.Auth.StsAuth.RoleSessionName = profiles[profile].RoleSessionName
	c.Validator.Auth.StsAuth.DurationSeconds = profiles[profile].DurationSeconds
	c.Validator.Auth.StsAuth.ExternalID = profiles[profile].ExternalID

	return nil
}

func loadAwsSTSProfiles() (map[string]vpawsapi.AwsSTSAuth, error) {
	configPath := buildAwsFilePath(awsConfigFilename)
	config, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS config file: %w", err)
	}

	awsProfiles := make(map[string]vpawsapi.AwsSTSAuth, 0)
	for _, section := range config.Sections() {
		// filter out profiles that don't have all the required fields
		if section.Key("role_arn").String() != "" && section.Key("role_session_name").String() != "" && section.Key("duration_seconds").String() != "" {
			awsProfiles[section.Name()] = vpawsapi.AwsSTSAuth{
				RoleArn:         section.Key("role_arn").String(),
				RoleSessionName: section.Key("role_session_name").String(),
				DurationSeconds: section.Key("duration_seconds").MustInt(),
				ExternalID:      section.Key("external_id").String(),
			}
		}
	}

	return awsProfiles, nil
}

func buildAwsFilePath(filename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aws", filename)
}
