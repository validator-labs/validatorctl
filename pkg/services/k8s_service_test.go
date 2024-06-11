package services

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
)

func TestGetSecretsWithKeys(t *testing.T) {
	tests := []struct {
		name                string
		keys                []string
		secrets             []corev1.Secret
		expectedSecretNames []string
	}{
		{
			name: "Matching validator basic auth keys",
			keys: cfg.ValidatorBasicAuthKeys,
			secrets: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-auth-secret-1", Namespace: "validator"},
					Data: map[string][]byte{
						"username": []byte("user1"),
						"password": []byte("pass1"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-auth-secret-2", Namespace: "validator"},
					Data: map[string][]byte{
						"username": []byte("user2"),
						"password": []byte("pass2"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "wrong-namespace-auth-secret", Namespace: "default"},
					Data: map[string][]byte{
						"username": []byte("user2"),
						"password": []byte("pass2"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "invalid-auth-secret", Namespace: "validator"},
					Data: map[string][]byte{
						"username": []byte("user2"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "some-secret", Namespace: "validator"},
					Data: map[string][]byte{
						"some-key": []byte("some-value"),
					},
				},
			},
			expectedSecretNames: []string{"valid-auth-secret-1", "valid-auth-secret-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeSecrets := make([]runtime.Object, 0)
			for _, s := range tt.secrets {
				s := s
				runtimeSecrets = append(runtimeSecrets, &s)
			}

			k8sClient := fake.NewSimpleClientset(runtimeSecrets...)
			secrets, err := GetSecretsWithKeys(k8sClient, "validator", tt.keys)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(secrets) != len(tt.expectedSecretNames) {
				t.Errorf("expected %d secrets, got %d", len(tt.expectedSecretNames), len(secrets))
			}

			for _, s := range secrets {
				if !slices.Contains(tt.expectedSecretNames, s.Name) {
					t.Errorf("unexpected secret: %s", s.Name)
				}
			}
		})
	}
}

func TestGetSecretsWithRegexKeys(t *testing.T) {
	tests := []struct {
		name                string
		keyExpr             string
		secrets             []corev1.Secret
		expectedSecretNames []string
	}{
		{
			name:    "Matching oci signature verification public key regex",
			keyExpr: cfg.ValidatorPluginOciSigVerificationKeysRegex,
			secrets: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-pubkey-secret-1", Namespace: "validator"},
					Data: map[string][]byte{
						"key.pub": []byte("public-key"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-pubkey-secret-2", Namespace: "validator"},
					Data: map[string][]byte{
						"key1.pub": []byte("public-key-1"),
						"key2.pub": []byte("public-key-2"),
						"key3.pub": []byte("public-key-3"),
						"key4.pub": []byte("public-key-4"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "valid-pubkey-secret-3", Namespace: "validator"},
					Data: map[string][]byte{
						"key.pub":   []byte("public-key"),
						"other-key": []byte("other-value"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "wrong-namespace-pubkey-secret", Namespace: "default"},
					Data: map[string][]byte{
						"key.pub": []byte("public-key"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "some-secret", Namespace: "validator"},
					Data: map[string][]byte{
						"some-key": []byte("some-value"),
					},
				},
			},
			expectedSecretNames: []string{"valid-pubkey-secret-1", "valid-pubkey-secret-2", "valid-pubkey-secret-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeSecrets := make([]runtime.Object, 0)
			for _, s := range tt.secrets {
				s := s
				runtimeSecrets = append(runtimeSecrets, &s)
			}

			k8sClient := fake.NewSimpleClientset(runtimeSecrets...)
			secrets, err := GetSecretsWithRegexKeys(k8sClient, "validator", tt.keyExpr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(secrets) != len(tt.expectedSecretNames) {
				t.Errorf("expected %d secrets, got %d", len(tt.expectedSecretNames), len(secrets))
			}
			for _, s := range secrets {
				if !slices.Contains(tt.expectedSecretNames, s.Name) {
					t.Errorf("unexpected secret: %s", s.Name)
				}
			}
		})
	}
}
