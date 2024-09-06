package validator

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	netapi "github.com/validator-labs/validator-plugin-network/api/v1alpha1"
	vapi "github.com/validator-labs/validator/api/v1alpha1"
	"github.com/validator-labs/validator/pkg/plugins"
)

func TestBuildValidationResultString(t *testing.T) {
	type testCase struct {
		name          string
		vrObj         unstructured.Unstructured
		expectedVrStr string
	}

	testCases := []testCase{{name: "valid vr json with multiple validation conditions",
		vrObj: unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "validator-plugin-oci-rules",
					"namespace": "validator",
				},
				"spec": map[string]interface{}{
					"plugin": "OCI",
				},
				"status": map[string]interface{}{
					"state": vapi.ValidationFailed,
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   vapi.SinkEmission,
							"reason": string(vapi.SinkEmitSucceeded),
						},
					},
					"validationConditions": []interface{}{
						map[string]interface{}{
							"validationType": "oci-registry",
							"validationRule": "success-rule",
							"message":        "All oci-registry checks passed",
							"details":        []interface{}{"detail-a", "detail-b"},
							"status":         corev1.ConditionTrue,
						},
						map[string]interface{}{
							"validationType": "oci-registry",
							"validationRule": "failure-rule",
							"message":        "Some oci-registry checks failed",
							"details":        []interface{}{"detail-a", "detail-b", "detail-c"},
							"failures":       []interface{}{"failure-a", "failure-b"},
							"status":         corev1.ConditionFalse,
						},
					},
				},
			},
		},
		expectedVrStr: `
=================
Validation Result
=================

Plugin:            OCI
Name:              validator-plugin-oci-rules
Namespace:         validator
State:             Failed
Sink State:        SinkEmitSucceeded

------------
Rule Results
------------

Validation Rule:        success-rule
Validation Type:        oci-registry
Status:                 True
Last Validated:         0001-01-01T00:00:00Z
Message:                All oci-registry checks passed

-------
Details
-------
- detail-a
- detail-b

Validation Rule:        failure-rule
Validation Type:        oci-registry
Status:                 False
Last Validated:         0001-01-01T00:00:00Z
Message:                Some oci-registry checks failed

-------
Details
-------
- detail-a
- detail-b
- detail-c

--------
Failures
--------
- failure-a
- failure-b
`,
	},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vrStr, _ := buildValidationResultString(tc.vrObj)

			if vrStr != tc.expectedVrStr {
				t.Errorf("\nexpected vrStr:\n%s\nactual vrStr:\n%s", tc.expectedVrStr, vrStr)
			}
		})
	}
}

func TestUnmarshalPluginSpec(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		expectedSpec plugins.PluginSpec
		expectedErr  error
	}{
		{
			name: "Valid NetworkValidator spec",
			input: []byte(
				`apiVersion: validation.spectrocloud.labs/v1alpha1
kind: NetworkValidator
metadata:
  name: network-validator-combined-network-rules
spec:
  dnsRules:
  - name: Resolve Google
    host: google.com
`),
			expectedSpec: &netapi.NetworkValidatorSpec{
				DNSRules: []netapi.DNSRule{
					{RuleName: "Resolve Google", Host: "google.com"},
				},
			},
			expectedErr: nil,
		},
		{
			name:         "Unknown plugin kind",
			input:        []byte(`kind: SomeRandomKind`),
			expectedSpec: nil,
			expectedErr:  errors.New("unknown plugin kind"),
		},
		{
			name: "Kind not set",
			input: []byte(
				`spec:
  dnsRules:
  - name: Resolve Google
    host: google.com
`),
			expectedSpec: nil,
			expectedErr:  errors.New("plugin kind is not set"),
		},
		{
			name:         "Invalid YAML format",
			input:        []byte("hello"),
			expectedSpec: nil,
			expectedErr:  errors.New("cannot unmarshal"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := unmarshalPluginSpec(tt.input)

			// If an error is expected
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSpec, spec)
			}
		})
	}
}
