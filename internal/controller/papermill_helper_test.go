/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestConvertResourceRequirements(t *testing.T) {
	tests := []struct {
		name           string
		customResources *mlopsv1alpha1.ResourceRequirements
		expected       corev1.ResourceRequirements
	}{
		{
			name:           "nil resources",
			customResources: nil,
			expected:       corev1.ResourceRequirements{},
		},
		{
			name: "empty resources",
			customResources: &mlopsv1alpha1.ResourceRequirements{
				Limits:   make(map[string]string),
				Requests: make(map[string]string),
			},
			expected: corev1.ResourceRequirements{
				Limits:   make(corev1.ResourceList),
				Requests: make(corev1.ResourceList),
			},
		},
		{
			name: "valid CPU and memory",
			customResources: &mlopsv1alpha1.ResourceRequirements{
				Limits: map[string]string{
					"cpu":    "2",
					"memory": "4Gi",
				},
				Requests: map[string]string{
					"cpu":    "1",
					"memory": "2Gi",
				},
			},
			expected: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "invalid quantity should be skipped",
			customResources: &mlopsv1alpha1.ResourceRequirements{
				Limits: map[string]string{
					"cpu":    "2",
					"memory": "invalid",
				},
				Requests: map[string]string{
					"cpu": "1",
				},
			},
			expected: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
			},
		},
		{
			name: "ephemeral storage",
			customResources: &mlopsv1alpha1.ResourceRequirements{
				Limits: map[string]string{
					"ephemeral-storage": "10Gi",
				},
				Requests: map[string]string{
					"ephemeral-storage": "5Gi",
				},
			},
			expected: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceEphemeralStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertResourceRequirements(tt.customResources)

			// Compare limits
			if len(result.Limits) != len(tt.expected.Limits) {
				t.Errorf("Limits length mismatch: got %d, want %d", len(result.Limits), len(tt.expected.Limits))
			}
			for key, expectedValue := range tt.expected.Limits {
				if actualValue, exists := result.Limits[key]; !exists {
					t.Errorf("Missing limit for resource %s", key)
				} else if !actualValue.Equal(expectedValue) {
					t.Errorf("Limit for %s: got %s, want %s", key, actualValue.String(), expectedValue.String())
				}
			}

			// Compare requests
			if len(result.Requests) != len(tt.expected.Requests) {
				t.Errorf("Requests length mismatch: got %d, want %d", len(result.Requests), len(tt.expected.Requests))
			}
			for key, expectedValue := range tt.expected.Requests {
				if actualValue, exists := result.Requests[key]; !exists {
					t.Errorf("Missing request for resource %s", key)
				} else if !actualValue.Equal(expectedValue) {
					t.Errorf("Request for %s: got %s, want %s", key, actualValue.String(), expectedValue.String())
				}
			}
		})
	}
}

func TestConvertEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		customEnvs  []mlopsv1alpha1.EnvVar
		expected    []corev1.EnvVar
		expectedLen int
	}{
		{
			name:        "nil env vars",
			customEnvs:   nil,
			expected:    nil,
			expectedLen: 0,
		},
		{
			name:        "empty env vars",
			customEnvs:  []mlopsv1alpha1.EnvVar{},
			expected:    []corev1.EnvVar{},
			expectedLen: 0,
		},
		{
			name: "simple env vars",
			customEnvs: []mlopsv1alpha1.EnvVar{
				{
					Name:  "ENV1",
					Value: "value1",
				},
				{
					Name:  "ENV2",
					Value: "value2",
				},
			},
			expected: []corev1.EnvVar{
				{
					Name:  "ENV1",
					Value: "value1",
				},
				{
					Name:  "ENV2",
					Value: "value2",
				},
			},
			expectedLen: 2,
		},
		{
			name: "env var with secret key ref",
			customEnvs: []mlopsv1alpha1.EnvVar{
				{
					Name: "SECRET_ENV",
					ValueFrom: &mlopsv1alpha1.EnvVarSource{
						SecretKeyRef: &mlopsv1alpha1.SecretKeySelector{
							Name: "my-secret",
							Key:  "secret-key",
						},
					},
				},
			},
			expected: []corev1.EnvVar{
				{
					Name: "SECRET_ENV",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-secret",
							},
							Key: "secret-key",
						},
					},
				},
			},
			expectedLen: 1,
		},
		{
			name: "env var with config map key ref",
			customEnvs: []mlopsv1alpha1.EnvVar{
				{
					Name: "CONFIG_ENV",
					ValueFrom: &mlopsv1alpha1.EnvVarSource{
						ConfigMapKeyRef: &mlopsv1alpha1.ConfigMapKeySelector{
							Name: "my-config",
							Key:  "config-key",
						},
					},
				},
			},
			expected: []corev1.EnvVar{
				{
					Name: "CONFIG_ENV",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-config",
							},
							Key: "config-key",
						},
					},
				},
			},
			expectedLen: 1,
		},
		{
			name: "mixed env vars",
			customEnvs: []mlopsv1alpha1.EnvVar{
				{
					Name:  "SIMPLE",
					Value: "simple-value",
				},
				{
					Name: "SECRET",
					ValueFrom: &mlopsv1alpha1.EnvVarSource{
						SecretKeyRef: &mlopsv1alpha1.SecretKeySelector{
							Name: "secret",
							Key:  "key",
						},
					},
				},
				{
					Name: "CONFIG",
					ValueFrom: &mlopsv1alpha1.EnvVarSource{
						ConfigMapKeyRef: &mlopsv1alpha1.ConfigMapKeySelector{
							Name: "config",
							Key:  "key",
						},
					},
				},
			},
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertEnvVars(tt.customEnvs)

			if tt.expectedLen != len(result) {
				t.Errorf("Length mismatch: got %d, want %d", len(result), tt.expectedLen)
			}

			if tt.expected != nil {
				for i, expected := range tt.expected {
					if i >= len(result) {
						t.Errorf("Missing env var at index %d", i)
						continue
					}

					actual := result[i]
					if actual.Name != expected.Name {
						t.Errorf("Env var %d name: got %s, want %s", i, actual.Name, expected.Name)
					}

					if actual.Value != expected.Value {
						t.Errorf("Env var %d value: got %s, want %s", i, actual.Value, expected.Value)
					}

					// Check ValueFrom
					if expected.ValueFrom != nil {
						if actual.ValueFrom == nil {
							t.Errorf("Env var %d: expected ValueFrom but got nil", i)
							continue
						}

						if expected.ValueFrom.SecretKeyRef != nil {
							if actual.ValueFrom.SecretKeyRef == nil {
								t.Errorf("Env var %d: expected SecretKeyRef but got nil", i)
							} else {
								if actual.ValueFrom.SecretKeyRef.Name != expected.ValueFrom.SecretKeyRef.Name {
									t.Errorf("Env var %d SecretKeyRef name: got %s, want %s",
										i, actual.ValueFrom.SecretKeyRef.Name, expected.ValueFrom.SecretKeyRef.Name)
								}
								if actual.ValueFrom.SecretKeyRef.Key != expected.ValueFrom.SecretKeyRef.Key {
									t.Errorf("Env var %d SecretKeyRef key: got %s, want %s",
										i, actual.ValueFrom.SecretKeyRef.Key, expected.ValueFrom.SecretKeyRef.Key)
								}
							}
						}

						if expected.ValueFrom.ConfigMapKeyRef != nil {
							if actual.ValueFrom.ConfigMapKeyRef == nil {
								t.Errorf("Env var %d: expected ConfigMapKeyRef but got nil", i)
							} else {
								if actual.ValueFrom.ConfigMapKeyRef.Name != expected.ValueFrom.ConfigMapKeyRef.Name {
									t.Errorf("Env var %d ConfigMapKeyRef name: got %s, want %s",
										i, actual.ValueFrom.ConfigMapKeyRef.Name, expected.ValueFrom.ConfigMapKeyRef.Name)
								}
								if actual.ValueFrom.ConfigMapKeyRef.Key != expected.ValueFrom.ConfigMapKeyRef.Key {
									t.Errorf("Env var %d ConfigMapKeyRef key: got %s, want %s",
										i, actual.ValueFrom.ConfigMapKeyRef.Key, expected.ValueFrom.ConfigMapKeyRef.Key)
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "true",
			input:    true,
			expected: true,
		},
		{
			name:     "false",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boolPtr(tt.input)
			if result == nil {
				t.Fatal("boolPtr returned nil")
			}
			if *result != tt.expected {
				t.Errorf("boolPtr(%v) = %v, want %v", tt.input, *result, tt.expected)
			}
		})
	}
}
