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
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestIsModelValidationEnabled(t *testing.T) {
	tests := []struct {
		name     string
		job      *mlopsv1alpha1.NotebookValidationJob
		expected bool
	}{
		{
			name: "model validation enabled",
			job: &mlopsv1alpha1.NotebookValidationJob{
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled: boolPtr(true),
					},
				},
			},
			expected: true,
		},
		{
			name: "model validation disabled",
			job: &mlopsv1alpha1.NotebookValidationJob{
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled: boolPtr(false),
					},
				},
			},
			expected: false,
		},
		{
			name: "model validation nil",
			job: &mlopsv1alpha1.NotebookValidationJob{
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: nil,
				},
			},
			expected: false,
		},
		{
			name: "model validation enabled nil",
			job: &mlopsv1alpha1.NotebookValidationJob{
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled: nil,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isModelValidationEnabled(tt.job)
			if result != tt.expected {
				t.Errorf("isModelValidationEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetPhaseString(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		expected string
	}{
		{
			name:     "empty phase defaults to both",
			phase:    "",
			expected: "both",
		},
		{
			name:     "clean phase",
			phase:    "clean",
			expected: "clean",
		},
		{
			name:     "existing phase",
			phase:    "existing",
			expected: "existing",
		},
		{
			name:     "both phase",
			phase:    "both",
			expected: "both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPhaseString(tt.phase)
			if result != tt.expected {
				t.Errorf("getPhaseString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildModelValidationEnvVars(t *testing.T) {
	reconciler := &NotebookValidationJobReconciler{}
	ctx := context.Background()

	tests := []struct {
		name     string
		job      *mlopsv1alpha1.NotebookValidationJob
		expected int // expected number of env vars
	}{
		{
			name: "model validation disabled",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled: boolPtr(false),
					},
				},
			},
			expected: 0,
		},
		{
			name: "basic model validation",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled:  boolPtr(true),
						Platform: "kserve",
					},
				},
			},
			expected: 3, // MODEL_VALIDATION_ENABLED, MODEL_VALIDATION_PLATFORM, MODEL_VALIDATION_NAMESPACE
		},
		{
			name: "model validation with phase and target models",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled:      boolPtr(true),
						Platform:     "kserve",
						Phase:        "both",
						TargetModels: []string{"model1", "model2"},
					},
				},
			},
			expected: 7, // + MODEL_VALIDATION_NAMESPACE, MODEL_VALIDATION_PHASE, MODEL_VALIDATION_TARGET_MODELS, MODEL_VALIDATION_TARGET_MODELS_ORIGINAL, MODEL_VALIDATION_TARGET_NAMESPACES
		},
		{
			name: "model validation with prediction validation",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled:  boolPtr(true),
						Platform: "kserve",
						PredictionValidation: &mlopsv1alpha1.PredictionValidationSpec{
							Enabled:        boolPtr(true),
							TestData:       `{"instances": [[1.0, 2.0]]}`,
							ExpectedOutput: `{"predictions": [[0.95]]}`,
							Tolerance:      "0.01",
						},
					},
				},
			},
			expected: 7, // + MODEL_VALIDATION_NAMESPACE + 4 prediction validation vars
		},
		{
			name: "model validation with custom platform",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					ModelValidation: &mlopsv1alpha1.ModelValidationSpec{
						Enabled:  boolPtr(true),
						Platform: "vllm",
						CustomPlatform: &mlopsv1alpha1.CustomPlatformSpec{
							APIGroup:            "apps",
							ResourceType:        "deployments",
							HealthCheckEndpoint: "http://{{.ModelName}}-vllm:8000/health",
							PredictionEndpoint:  "http://{{.ModelName}}-vllm:8000/v1/completions",
						},
					},
				},
			},
			expected: 7, // + MODEL_VALIDATION_NAMESPACE + 4 custom platform vars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := reconciler.buildModelValidationEnvVars(ctx, tt.job)
			if len(envVars) != tt.expected {
				t.Errorf("buildModelValidationEnvVars() returned %d env vars, want %d", len(envVars), tt.expected)
				for _, env := range envVars {
					t.Logf("  %s=%s", env.Name, env.Value)
				}
			}

			// Verify required env vars are present when enabled
			if isModelValidationEnabled(tt.job) {
				hasEnabled := false
				hasPlatform := false
				for _, env := range envVars {
					if env.Name == "MODEL_VALIDATION_ENABLED" && env.Value == "true" {
						hasEnabled = true
					}
					if env.Name == "MODEL_VALIDATION_PLATFORM" {
						hasPlatform = true
					}
				}
				if !hasEnabled {
					t.Error("MODEL_VALIDATION_ENABLED not found or not set to true")
				}
				if !hasPlatform {
					t.Error("MODEL_VALIDATION_PLATFORM not found")
				}
			}
		})
	}
}
