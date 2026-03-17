package aws

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestHasDedicatedHost(t *testing.T) {
	tests := []struct {
		name     string
		machine  *unstructured.Unstructured
		expected bool
	}{
		{
			name: "machine with dedicated host",
			machine: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"providerSpec": map[string]interface{}{
							"value": map[string]interface{}{
								"placement": map[string]interface{}{
									"host": map[string]interface{}{
										"affinity": "dedicated-host",
										"dedicatedHost": map[string]interface{}{
											"allocationStrategy": "user-provided",
											"id":                 "h-1234567890abcdef0",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "machine without placement",
			machine: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"providerSpec": map[string]interface{}{
							"value": map[string]interface{}{
								"instanceType": "m5.large",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "machine with placement but no host",
			machine: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"providerSpec": map[string]interface{}{
							"value": map[string]interface{}{
								"placement": map[string]interface{}{
									"tenancy": "default",
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "machine without providerSpec",
			machine: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDedicatedHost(tt.machine)
			if result != tt.expected {
				t.Errorf("hasDedicatedHost() = %v, want %v", result, tt.expected)
			}
		})
	}
}
