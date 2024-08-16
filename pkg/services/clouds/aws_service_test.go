package clouds

import (
	"reflect"
	"testing"

	vpawsapi "github.com/validator-labs/validator-plugin-aws/api/v1alpha1"
	"github.com/validator-labs/validatorctl/tests/utils/file"
)

func TestLoadAwsSTSProfiles(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     map[string]vpawsapi.AwsSTSAuth
		wantErr  bool
	}{
		{
			name:     "test present",
			filepath: "aws_config",
			want: map[string]vpawsapi.AwsSTSAuth{
				"profile test": {
					RoleArn:         "test_arn",
					RoleSessionName: "test_name",
					DurationSeconds: 3600,
					ExternalID:      "test_id",
				},
			},
			wantErr: false,
		},
		{
			name:     "test not present",
			filepath: "aws_config_dne",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := file.UnitTestFile(tt.filepath)
			profiles, err := loadAwsSTSProfiles(filepath)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(profiles, tt.want) {
					t.Fatalf("profiles = %v, want %v", profiles, tt.want)
				}
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			}
		})
	}
}

func TestLoadAwsCredsProfiles(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     map[string]map[string]string
		wantErr  bool
	}{
		{
			name:     "test present",
			filepath: "aws_credentials",
			want: map[string]map[string]string{
				"default": {
					"aws_access_key_id":     "test_id",
					"aws_secret_access_key": "test_key",
					"aws_session_token":     "test_token",
				},
			},
			wantErr: false,
		},
		{
			name:     "test not present",
			filepath: "aws_credentials_dne",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filepath := file.UnitTestFile(tt.filepath)
			profiles, err := loadAwsCredsProfiles(filepath)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("failed to load AWS credentials profiles: %v", err)
				}
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(profiles, tt.want) {
					t.Fatalf("profiles = %v, want %v", profiles, tt.want)
				}
			}
		})
	}
}
