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
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/platform"
)

// ModelValidationConfig contains configuration for model validation with multi-user support
type ModelValidationConfig struct {
	// JobNamespace is the namespace where the NotebookValidationJob is running
	JobNamespace string
	// AllowCrossNamespace enables cross-namespace model access
	AllowCrossNamespace bool
	// AllowedNamespaces is a list of namespaces that can be accessed for models
	// Empty list means all namespaces (when AllowCrossNamespace is true)
	AllowedNamespaces []string
	// TimeoutSeconds is the timeout for model validation operations
	TimeoutSeconds int
}

// performModelValidation performs platform detection and model validation with multi-user namespace isolation
func (r *NotebookValidationJobReconciler) performModelValidation(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error {
	startTime := time.Now()
	logger := log.FromContext(ctx)

	// Check if model validation is enabled
	if job.Spec.ModelValidation == nil || job.Spec.ModelValidation.Enabled == nil || !*job.Spec.ModelValidation.Enabled {
		logger.V(1).Info("Model validation not enabled, skipping")
		return nil
	}

	targetPlatform := job.Spec.ModelValidation.Platform
	logger.Info("Performing model validation",
		"platform", targetPlatform,
		"phase", job.Spec.ModelValidation.Phase,
		"jobNamespace", job.Namespace,
		"targetModels", job.Spec.ModelValidation.TargetModels)

	// Create discovery client for platform detection
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(r.RestConfig)
	if err != nil {
		logger.Error(err, "Failed to create discovery client for platform detection")
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create platform detector
	detector := platform.NewDetector(r.Client, discoveryClient)

	// Detect platform and record metrics
	detectionStart := time.Now()
	platformInfo, err := detector.DetectPlatform(ctx, job.Spec.ModelValidation.Platform)
	detectionDuration := time.Since(detectionStart).Seconds()

	if err != nil {
		logger.Error(err, "Failed to detect model serving platform",
			"platformHint", job.Spec.ModelValidation.Platform,
			"namespace", job.Namespace)

		// Record platform detection failure
		recordPlatformDetection(job.Namespace, targetPlatform, false, detectionDuration)
		recordModelValidationDuration(job.Namespace, targetPlatform, "error", time.Since(startTime).Seconds())

		// Update status with detection failure
		job.Status.ModelValidationResult = &mlopsv1alpha1.ModelValidationResult{
			Phase:            getPhaseString(job.Spec.ModelValidation.Phase),
			Platform:         job.Spec.ModelValidation.Platform,
			PlatformDetected: false,
			Success:          false,
			Message:          fmt.Sprintf("Platform detection failed in namespace %s: %v", job.Namespace, err),
		}

		return fmt.Errorf("platform detection failed: %w", err)
	}

	// Record successful platform detection
	recordPlatformDetection(job.Namespace, targetPlatform, platformInfo.Available, detectionDuration)

	logger.Info("Platform detected successfully",
		"platform", platformInfo.Platform,
		"available", platformInfo.Available,
		"detectedCRDs", platformInfo.CRDs,
		"namespace", job.Namespace)

	// Update status with platform detection result
	if job.Status.ModelValidationResult == nil {
		job.Status.ModelValidationResult = &mlopsv1alpha1.ModelValidationResult{}
	}

	job.Status.ModelValidationResult.Phase = getPhaseString(job.Spec.ModelValidation.Phase)
	job.Status.ModelValidationResult.Platform = string(platformInfo.Platform)
	job.Status.ModelValidationResult.PlatformDetected = platformInfo.Available

	if !platformInfo.Available {
		job.Status.ModelValidationResult.Success = false
		job.Status.ModelValidationResult.Message = fmt.Sprintf("Platform %s not available in cluster (checked from namespace %s)", platformInfo.Platform, job.Namespace)
		logger.Info("Platform not available", "platform", platformInfo.Platform, "namespace", job.Namespace)

		// Record model validation failure due to platform unavailability
		recordModelValidationDuration(job.Namespace, targetPlatform, "platform_unavailable", time.Since(startTime).Seconds())

		return fmt.Errorf("platform %s not available", platformInfo.Platform)
	}

	// Perform namespace-aware model health checks if target models are specified
	if len(job.Spec.ModelValidation.TargetModels) > 0 {
		err = r.performNamespaceAwareModelHealthChecks(ctx, job, detector, platformInfo)
		if err != nil {
			logger.Error(err, "Model health checks failed",
				"namespace", job.Namespace,
				"targetModels", job.Spec.ModelValidation.TargetModels)
			recordModelValidationDuration(job.Namespace, targetPlatform, "health_check_failed", time.Since(startTime).Seconds())
			return err
		}
	}

	logger.Info("Model validation platform ready",
		"platform", platformInfo.Platform,
		"phase", job.Spec.ModelValidation.Phase,
		"namespace", job.Namespace)

	// Record successful model validation setup
	recordModelValidationDuration(job.Namespace, targetPlatform, "success", time.Since(startTime).Seconds())

	return nil
}

// performNamespaceAwareModelHealthChecks performs health checks on target models with namespace isolation
func (r *NotebookValidationJobReconciler) performNamespaceAwareModelHealthChecks(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	detector *platform.Detector,
	platformInfo *platform.PlatformInfo,
) error {
	logger := log.FromContext(ctx)

	// Build validation config based on job settings
	// By default, only allow same-namespace access
	config := &ModelValidationConfig{
		JobNamespace:        job.Namespace,
		AllowCrossNamespace: false,
		AllowedNamespaces:   []string{},
		TimeoutSeconds:      300, // 5 minute default
	}

	// Parse timeout if specified
	if job.Spec.ModelValidation.Timeout != "" {
		if timeout, err := time.ParseDuration(job.Spec.ModelValidation.Timeout); err == nil {
			config.TimeoutSeconds = int(timeout.Seconds())
		}
	}

	// Create model resolver with namespace isolation
	resolver := platform.NewDefaultModelResolver(job.Namespace)

	// Resolve model references
	modelRefs, err := resolver.ResolveModelReferences(ctx, job.Spec.ModelValidation.TargetModels, job.Namespace)
	if err != nil {
		logger.Error(err, "Failed to resolve model references",
			"targetModels", job.Spec.ModelValidation.TargetModels,
			"namespace", job.Namespace)

		job.Status.ModelValidationResult.Success = false
		job.Status.ModelValidationResult.Message = fmt.Sprintf("Failed to resolve model references in namespace %s: %v", job.Namespace, err)
		return err
	}

	logger.Info("Resolved model references",
		"count", len(modelRefs),
		"namespace", job.Namespace)

	// Group models by namespace for efficient checking
	modelsByNamespace := platform.GroupByNamespace(modelRefs)

	// Log which namespaces will be checked
	namespaces := make([]string, 0, len(modelsByNamespace))
	for ns := range modelsByNamespace {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	logger.V(1).Info("Models grouped by namespace",
		"namespaces", namespaces,
		"jobNamespace", job.Namespace)

	// Check health of all models
	healthConfig := &platform.ModelHealthCheckConfig{
		Namespace:           job.Namespace,
		AllowCrossNamespace: config.AllowCrossNamespace,
		AllowedNamespaces:   config.AllowedNamespaces,
		TimeoutSeconds:      config.TimeoutSeconds,
	}

	healthResults, err := detector.CheckMultipleModelsHealth(ctx, modelRefs, platformInfo.Platform, job.Namespace, healthConfig)
	if err != nil {
		logger.Error(err, "Model health checks failed",
			"namespace", job.Namespace)

		job.Status.ModelValidationResult.Success = false
		job.Status.ModelValidationResult.Message = fmt.Sprintf("Model health checks failed in namespace %s: %v", job.Namespace, err)
		return err
	}

	// Update status with detailed health check results
	modelCheckResults := make([]mlopsv1alpha1.ModelCheckResult, 0, len(healthResults))
	allHealthy := true
	var unhealthyModels []string

	for _, healthStatus := range healthResults {
		modelResult := mlopsv1alpha1.ModelCheckResult{
			ModelName: healthStatus.ModelName,
			Available: healthStatus.Available,
			Healthy:   healthStatus.Ready,
			Version:   "", // Version detection not implemented yet
			Message:   healthStatus.Message,
		}

		if healthStatus.Namespace != job.Namespace {
			modelResult.Message = fmt.Sprintf("[namespace: %s] %s", healthStatus.Namespace, healthStatus.Message)
		}

		modelCheckResults = append(modelCheckResults, modelResult)

		if !healthStatus.Ready {
			allHealthy = false
			unhealthyModels = append(unhealthyModels, fmt.Sprintf("%s/%s", healthStatus.Namespace, healthStatus.ModelName))
		}
	}

	// Update existing environment check or create new one
	if job.Status.ModelValidationResult.ExistingEnvironmentCheck == nil {
		job.Status.ModelValidationResult.ExistingEnvironmentCheck = &mlopsv1alpha1.ExistingEnvironmentCheckResult{}
	}

	job.Status.ModelValidationResult.ExistingEnvironmentCheck.ModelsChecked = modelCheckResults
	job.Status.ModelValidationResult.ExistingEnvironmentCheck.Success = allHealthy

	if allHealthy {
		job.Status.ModelValidationResult.ExistingEnvironmentCheck.Message = fmt.Sprintf("All %d model(s) are healthy in namespace %s", len(modelCheckResults), job.Namespace)
		job.Status.ModelValidationResult.Success = true
		job.Status.ModelValidationResult.Message = "Model validation successful"
	} else {
		job.Status.ModelValidationResult.ExistingEnvironmentCheck.Message = fmt.Sprintf("%d of %d model(s) are unhealthy: %v", len(unhealthyModels), len(modelCheckResults), unhealthyModels)
		job.Status.ModelValidationResult.Success = false
		job.Status.ModelValidationResult.Message = fmt.Sprintf("Model health check failed: unhealthy models: %v", unhealthyModels)
	}

	logger.Info("Model health checks completed",
		"totalModels", len(modelCheckResults),
		"healthyModels", len(modelCheckResults)-len(unhealthyModels),
		"unhealthyModels", len(unhealthyModels),
		"namespace", job.Namespace)

	if !allHealthy {
		return fmt.Errorf("some models are unhealthy: %v", unhealthyModels)
	}

	return nil
}

// buildModelValidationEnvVars builds environment variables for model validation with namespace context
func (r *NotebookValidationJobReconciler) buildModelValidationEnvVars(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) []corev1.EnvVar {
	logger := log.FromContext(ctx)

	// Check if model validation is enabled
	if job.Spec.ModelValidation == nil || job.Spec.ModelValidation.Enabled == nil || !*job.Spec.ModelValidation.Enabled {
		return nil
	}

	logger.V(1).Info("Building model validation environment variables",
		"namespace", job.Namespace)

	envVars := []corev1.EnvVar{
		{
			Name:  "MODEL_VALIDATION_ENABLED",
			Value: "true",
		},
		{
			Name:  "MODEL_VALIDATION_PLATFORM",
			Value: job.Spec.ModelValidation.Platform,
		},
		// Add namespace context for multi-user isolation
		{
			Name:  "MODEL_VALIDATION_NAMESPACE",
			Value: job.Namespace,
		},
	}

	// Add phase
	if job.Spec.ModelValidation.Phase != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_PHASE",
			Value: job.Spec.ModelValidation.Phase,
		})
	}

	// Add target models with namespace resolution
	if len(job.Spec.ModelValidation.TargetModels) > 0 {
		// Resolve and normalize model references
		resolver := platform.NewDefaultModelResolver(job.Namespace)
		resolvedRefs, _ := resolver.ResolveModelReferences(ctx, job.Spec.ModelValidation.TargetModels, job.Namespace)

		// Build comma-separated list of resolved model references (namespace/model format)
		resolvedModels := make([]string, 0, len(resolvedRefs))
		for _, ref := range resolvedRefs {
			resolvedModels = append(resolvedModels, ref.FormatModelReference())
		}

		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_TARGET_MODELS",
			Value: strings.Join(resolvedModels, ","),
		})

		// Also provide the original model list for backward compatibility
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_TARGET_MODELS_ORIGINAL",
			Value: strings.Join(job.Spec.ModelValidation.TargetModels, ","),
		})

		// Add unique namespaces being accessed
		namespaces := platform.GetUniqueNamespaces(resolvedRefs)
		envVars = append(envVars, corev1.EnvVar{
			Name:  "MODEL_VALIDATION_TARGET_NAMESPACES",
			Value: strings.Join(namespaces, ","),
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

	logger.Info("Built model validation environment variables",
		"count", len(envVars),
		"namespace", job.Namespace)
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
