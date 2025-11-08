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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/platform"
)

// performModelValidation performs platform detection and model validation
func (r *NotebookValidationJobReconciler) performModelValidation(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error {
	logger := log.FromContext(ctx)

	// Check if model validation is enabled
	if job.Spec.ModelValidation == nil || job.Spec.ModelValidation.Enabled == nil || !*job.Spec.ModelValidation.Enabled {
		logger.V(1).Info("Model validation not enabled, skipping")
		return nil
	}

	logger.Info("Performing model validation",
		"platform", job.Spec.ModelValidation.Platform,
		"phase", job.Spec.ModelValidation.Phase)

	// Create discovery client for platform detection
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(r.RestConfig)
	if err != nil {
		logger.Error(err, "Failed to create discovery client for platform detection")
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create platform detector
	detector := platform.NewDetector(r.Client, discoveryClient)

	// Detect platform
	platformInfo, err := detector.DetectPlatform(ctx, job.Spec.ModelValidation.Platform)
	if err != nil {
		logger.Error(err, "Failed to detect model serving platform",
			"platformHint", job.Spec.ModelValidation.Platform)

		// Update status with detection failure
		job.Status.ModelValidationResult = &mlopsv1alpha1.ModelValidationResult{
			Phase:            getPhaseString(job.Spec.ModelValidation.Phase),
			Platform:         job.Spec.ModelValidation.Platform,
			PlatformDetected: false,
			Success:          false,
			Message:          fmt.Sprintf("Platform detection failed: %v", err),
		}

		return fmt.Errorf("platform detection failed: %w", err)
	}

	logger.Info("Platform detected successfully",
		"platform", platformInfo.Platform,
		"available", platformInfo.Available,
		"detectedCRDs", platformInfo.CRDs)

	// Update status with platform detection result
	if job.Status.ModelValidationResult == nil {
		job.Status.ModelValidationResult = &mlopsv1alpha1.ModelValidationResult{}
	}

	job.Status.ModelValidationResult.Phase = getPhaseString(job.Spec.ModelValidation.Phase)
	job.Status.ModelValidationResult.Platform = string(platformInfo.Platform)
	job.Status.ModelValidationResult.PlatformDetected = platformInfo.Available

	if !platformInfo.Available {
		job.Status.ModelValidationResult.Success = false
		job.Status.ModelValidationResult.Message = fmt.Sprintf("Platform %s not available in cluster", platformInfo.Platform)
		logger.Info("Platform not available", "platform", platformInfo.Platform)
		return fmt.Errorf("platform %s not available", platformInfo.Platform)
	}

	logger.Info("Model validation platform ready",
		"platform", platformInfo.Platform,
		"phase", job.Spec.ModelValidation.Phase)

	return nil
}

// buildModelValidationEnvVars builds environment variables for model validation
func (r *NotebookValidationJobReconciler) buildModelValidationEnvVars(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) []corev1.EnvVar {
	logger := log.FromContext(ctx)

	// Check if model validation is enabled
	if job.Spec.ModelValidation == nil || job.Spec.ModelValidation.Enabled == nil || !*job.Spec.ModelValidation.Enabled {
		return nil
	}

	logger.V(1).Info("Building model validation environment variables")

	envVars := []corev1.EnvVar{
		{
			Name:  "MODEL_VALIDATION_ENABLED",
			Value: "true",
		},
		{
			Name:  "MODEL_VALIDATION_PLATFORM",
			Value: job.Spec.ModelValidation.Platform,
		},
	}

	// Add phase
	if job.Spec.ModelValidation.Phase != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_PHASE",
			Value: job.Spec.ModelValidation.Phase,
		})
	}

	// Add target models
	if len(job.Spec.ModelValidation.TargetModels) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_TARGET_MODELS",
			Value: strings.Join(job.Spec.ModelValidation.TargetModels, ","),
		})
	}

	// Add prediction validation settings
	if job.Spec.ModelValidation.PredictionValidation != nil &&
		job.Spec.ModelValidation.PredictionValidation.Enabled != nil &&
		*job.Spec.ModelValidation.PredictionValidation.Enabled {

		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_PREDICTION_ENABLED",
			Value: "true",
		})

		if job.Spec.ModelValidation.PredictionValidation.TestData != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_PREDICTION_TEST_DATA",
				Value: job.Spec.ModelValidation.PredictionValidation.TestData,
			})
		}

		if job.Spec.ModelValidation.PredictionValidation.ExpectedOutput != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_PREDICTION_EXPECTED_OUTPUT",
				Value: job.Spec.ModelValidation.PredictionValidation.ExpectedOutput,
			})
		}

		if job.Spec.ModelValidation.PredictionValidation.Tolerance != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_PREDICTION_TOLERANCE",
				Value: job.Spec.ModelValidation.PredictionValidation.Tolerance,
			})
		}
	}

	// Add custom platform configuration
	if job.Spec.ModelValidation.CustomPlatform != nil {
		if job.Spec.ModelValidation.CustomPlatform.APIGroup != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_CUSTOM_API_GROUP",
				Value: job.Spec.ModelValidation.CustomPlatform.APIGroup,
			})
		}

		if job.Spec.ModelValidation.CustomPlatform.ResourceType != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_CUSTOM_RESOURCE_TYPE",
				Value: job.Spec.ModelValidation.CustomPlatform.ResourceType,
			})
		}

		if job.Spec.ModelValidation.CustomPlatform.HealthCheckEndpoint != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_CUSTOM_HEALTH_ENDPOINT",
				Value: job.Spec.ModelValidation.CustomPlatform.HealthCheckEndpoint,
			})
		}

		if job.Spec.ModelValidation.CustomPlatform.PredictionEndpoint != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "MODEL_VALIDATION_CUSTOM_PREDICTION_ENDPOINT",
				Value: job.Spec.ModelValidation.CustomPlatform.PredictionEndpoint,
			})
		}
	}

	// Add timeout
	if job.Spec.ModelValidation.Timeout != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_TIMEOUT",
			Value: job.Spec.ModelValidation.Timeout,
		})
	}

	logger.Info("Built model validation environment variables", "count", len(envVars))
	return envVars
}

// getPhaseString returns the phase string, defaulting to "both" if empty
func getPhaseString(phase string) string {
	if phase == "" {
		return "both"
	}
	return phase
}

// isModelValidationEnabled checks if model validation is enabled
func isModelValidationEnabled(job *mlopsv1alpha1.NotebookValidationJob) bool {
	return job.Spec.ModelValidation != nil &&
		job.Spec.ModelValidation.Enabled != nil &&
		*job.Spec.ModelValidation.Enabled
}

// updateModelValidationStatus updates the model validation status
func (r *NotebookValidationJobReconciler) updateModelValidationStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, success bool, message string) error {
	logger := log.FromContext(ctx)

	if job.Status.ModelValidationResult == nil {
		job.Status.ModelValidationResult = &mlopsv1alpha1.ModelValidationResult{}
	}

	job.Status.ModelValidationResult.Success = success
	job.Status.ModelValidationResult.Message = message

	if err := r.Status().Update(ctx, job); err != nil {
		logger.Error(err, "Failed to update model validation status")
		return err
	}

	logger.Info("Model validation status updated", "success", success, "message", message)
	return nil
}
