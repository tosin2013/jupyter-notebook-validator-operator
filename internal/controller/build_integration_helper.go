package controller

import (
	"context"
	"fmt"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/build"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Build condition types
	ConditionTypeBuildStarted  = "BuildStarted"
	ConditionTypeBuildComplete = "BuildComplete"
	ConditionTypeBuildFailed   = "BuildFailed"

	// Build condition reasons
	ReasonBuildCreated      = "BuildCreated"
	ReasonBuildInProgress   = "BuildInProgress"
	ReasonBuildSucceeded    = "BuildSucceeded"
	ReasonBuildFailedReason = "BuildFailed"
	ReasonBuildTimeout      = "BuildTimeout"
	ReasonStrategyNotFound  = "StrategyNotFound"
	ReasonConfigInvalid     = "ConfigInvalid"
	ReasonBuildNotEnabled   = "BuildNotEnabled"

	// Build defaults
	DefaultBuildTimeout = 15 * time.Minute
)

// isBuildEnabled checks if build is enabled in the job spec
func isBuildEnabled(job *mlopsv1alpha1.NotebookValidationJob) bool {
	return job.Spec.PodConfig.BuildConfig != nil && job.Spec.PodConfig.BuildConfig.Enabled
}

// handleBuildIntegration handles the build integration workflow
// Returns the image to use for validation pod (built image or fallback)
func (r *NotebookValidationJobReconciler) handleBuildIntegration(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (string, error) {
	logger := log.FromContext(ctx)

	// Check if build is enabled
	if !isBuildEnabled(job) {
		logger.V(1).Info("Build not enabled, using container image from spec")
		return job.Spec.PodConfig.ContainerImage, nil
	}

	logger.Info("Build enabled, starting build integration workflow",
		"strategy", job.Spec.PodConfig.BuildConfig.Strategy,
		"baseImage", job.Spec.PodConfig.BuildConfig.BaseImage)

	// Initialize build registry
	registry := build.NewStrategyRegistry(r.Client, r.Scheme)

	// Register available strategies
	s2iStrategy := build.NewS2IStrategy(r.Client, r.Scheme)
	tektonStrategy := build.NewTektonStrategy(r.Client, r.Scheme)

	registry.Register(s2iStrategy)
	registry.Register(tektonStrategy)

	// Get the configured strategy
	strategyName := job.Spec.PodConfig.BuildConfig.Strategy
	if strategyName == "" {
		strategyName = "s2i" // Default to S2I
	}

	strategy := registry.GetStrategy(strategyName)
	if strategy == nil {
		logger.Error(nil, "Failed to get build strategy", "strategy", strategyName)
		if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Strategy not found: %s", strategyName), ""); updateErr != nil {
			logger.Error(updateErr, "Failed to update build status")
		}
		// Fall back to container image
		return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build strategy not found: %s", strategyName)
	}

	// Check if strategy is available
	available, err := strategy.Detect(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to check strategy availability", "strategy", strategyName)
		// Include detailed error in build status
		errorMsg := fmt.Sprintf("Strategy detection failed for %s: %v", strategyName, err)
		if updateErr := r.updateBuildStatus(ctx, job, "Failed", errorMsg, ""); updateErr != nil {
			logger.Error(updateErr, "Failed to update build status")
		}
		return job.Spec.PodConfig.ContainerImage, fmt.Errorf("failed to check strategy availability: %w", err)
	}

	if !available {
		logger.Info("Build strategy not available, falling back to container image", "strategy", strategyName)
		// Provide more helpful message about why strategy is not available
		errorMsg := fmt.Sprintf("Strategy not available: %s. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster.", strategyName)
		if updateErr := r.updateBuildStatus(ctx, job, "Failed", errorMsg, ""); updateErr != nil {
			logger.Error(updateErr, "Failed to update build status")
		}
		return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build strategy not available: %s", strategyName)
	}

	// Validate configuration
	if err := strategy.ValidateConfig(job.Spec.PodConfig.BuildConfig); err != nil {
		logger.Error(err, "Build configuration validation failed")
		if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Invalid configuration: %v", err), ""); updateErr != nil {
			logger.Error(updateErr, "Failed to update build status")
		}
		return job.Spec.PodConfig.ContainerImage, fmt.Errorf("invalid build configuration: %w", err)
	}

	// Check if build already exists
	buildName := fmt.Sprintf("%s-build", job.Name)
	existingBuild, err := strategy.GetBuildStatus(ctx, buildName)
	if err == nil && existingBuild != nil {
		// Build exists, check its status
		logger.Info("Existing build found", "buildName", buildName, "status", existingBuild.Status)

		switch existingBuild.Status {
		case build.BuildStatusComplete:
			logger.Info("Build already complete, using built image", "image", existingBuild.ImageReference)
			if updateErr := r.updateBuildStatus(ctx, job, "Complete", "Build completed successfully", existingBuild.ImageReference); updateErr != nil {
				logger.Error(updateErr, "Failed to update build status")
			}
			return existingBuild.ImageReference, nil

		case build.BuildStatusFailed:
			logger.Info("Previous build failed, creating new build")
			// Delete failed build and create new one
			if deleteErr := strategy.DeleteBuild(ctx, buildName); deleteErr != nil {
				logger.Error(deleteErr, "Failed to delete failed build")
			}

		case build.BuildStatusPending, build.BuildStatusRunning:
			logger.Info("Build in progress, waiting for completion", "status", existingBuild.Status)
			// Wait for build to complete
			return r.waitForBuildCompletion(ctx, job, strategy, buildName)

		case build.BuildStatusCancelled:
			logger.Info("Previous build was cancelled, creating new build")
			if deleteErr := strategy.DeleteBuild(ctx, buildName); deleteErr != nil {
				logger.Error(deleteErr, "Failed to delete cancelled build")
			}
		}
	}

	// Create new build
	logger.Info("Creating new build", "buildName", buildName, "strategy", strategyName)
	buildInfo, err := strategy.CreateBuild(ctx, job)
	if err != nil {
		logger.Error(err, "Failed to create build")
		if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Build creation failed: %v", err), ""); updateErr != nil {
			logger.Error(updateErr, "Failed to update build status")
		}
		return job.Spec.PodConfig.ContainerImage, fmt.Errorf("failed to create build: %w", err)
	}

	logger.Info("Build created successfully", "buildName", buildInfo.Name)
	if updateErr := r.updateBuildStatus(ctx, job, "Running", "Build created and started", ""); updateErr != nil {
		logger.Error(updateErr, "Failed to update build status")
	}

	// Wait for build to complete
	return r.waitForBuildCompletion(ctx, job, strategy, buildInfo.Name)
}

// waitForBuildCompletion waits for a build to complete and returns the built image
func (r *NotebookValidationJobReconciler) waitForBuildCompletion(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, strategy build.Strategy, buildName string) (string, error) {
	logger := log.FromContext(ctx)

	timeout := DefaultBuildTimeout
	if job.Spec.PodConfig.BuildConfig.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(job.Spec.PodConfig.BuildConfig.Timeout); err == nil {
			timeout = parsedTimeout
		}
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	logger.Info("Waiting for build to complete", "buildName", buildName, "timeout", timeout)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled while waiting for build")
			return job.Spec.PodConfig.ContainerImage, ctx.Err()

		case <-ticker.C:
			if time.Now().After(deadline) {
				logger.Error(nil, "Build timeout exceeded", "buildName", buildName, "timeout", timeout)
				if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Build timeout exceeded: %v", timeout), ""); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build timeout exceeded: %v", timeout)
			}

			buildInfo, err := strategy.GetBuildStatus(ctx, buildName)
			if err != nil {
				logger.Error(err, "Failed to get build status", "buildName", buildName)
				continue
			}

			logger.V(1).Info("Build status check", "buildName", buildName, "status", buildInfo.Status, "message", buildInfo.Message)

			switch buildInfo.Status {
			case build.BuildStatusComplete:
				logger.Info("Build completed successfully", "buildName", buildName, "image", buildInfo.ImageReference)
				if updateErr := r.updateBuildStatus(ctx, job, "Complete", "Build completed successfully", buildInfo.ImageReference); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return buildInfo.ImageReference, nil

			case build.BuildStatusFailed:
				logger.Error(nil, "Build failed", "buildName", buildName, "message", buildInfo.Message)
				if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Build failed: %s", buildInfo.Message), ""); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build failed: %s", buildInfo.Message)

			case build.BuildStatusCancelled:
				logger.Info("Build was cancelled", "buildName", buildName)
				if updateErr := r.updateBuildStatus(ctx, job, "Failed", "Build was cancelled", ""); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build was cancelled")

			case build.BuildStatusPending, build.BuildStatusRunning:
				// Continue waiting
				logger.V(2).Info("Build still in progress", "buildName", buildName, "status", buildInfo.Status)

			default:
				logger.Info("Unknown build status", "buildName", buildName, "status", buildInfo.Status)
			}
		}
	}
}

// updateBuildStatus updates the build status in the job
func (r *NotebookValidationJobReconciler) updateBuildStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, status, message, imageReference string) error {
	logger := log.FromContext(ctx)

	// Initialize build status if needed
	if job.Status.BuildStatus == nil {
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{}
	}

	job.Status.BuildStatus.Phase = status
	job.Status.BuildStatus.Message = message
	job.Status.BuildStatus.ImageReference = imageReference

	// Set strategy from build config if available
	if job.Spec.PodConfig.BuildConfig != nil {
		job.Status.BuildStatus.Strategy = job.Spec.PodConfig.BuildConfig.Strategy
	}

	// Populate available images from OpenShift AI (if installed and not already populated)
	if len(job.Status.BuildStatus.AvailableImages) == 0 {
		if err := r.populateAvailableImages(ctx, job); err != nil {
			logger.V(1).Info("Could not populate available images", "error", err)
			// Don't fail the update - this is informational only
		}
	}

	if status == "Running" && job.Status.BuildStatus.StartTime == nil {
		now := metav1.Now()
		job.Status.BuildStatus.StartTime = &now
	}

	if status == "Complete" || status == "Failed" {
		now := metav1.Now()
		job.Status.BuildStatus.CompletionTime = &now
	}

	// Update condition
	conditionType := ConditionTypeBuildStarted
	conditionStatus := metav1.ConditionTrue
	reason := ReasonBuildInProgress

	if status == "Complete" {
		conditionType = ConditionTypeBuildComplete
		reason = ReasonBuildSucceeded
	} else if status == "Failed" {
		conditionType = ConditionTypeBuildFailed
		conditionStatus = metav1.ConditionFalse
		reason = ReasonBuildFailedReason
	}

	condition := metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	job.Status.Conditions = updateCondition(job.Status.Conditions, condition)

	if err := r.Status().Update(ctx, job); err != nil {
		logger.Error(err, "Failed to update build status")
		return err
	}

	logger.Info("Build status updated", "status", status, "message", message)
	return nil
}

// populateAvailableImages populates the available images from OpenShift AI
func (r *NotebookValidationJobReconciler) populateAvailableImages(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error {
	logger := log.FromContext(ctx)

	// Create OpenShift AI helper
	aiHelper := build.NewOpenShiftAIHelper(r.Client)

	// Check if OpenShift AI is installed
	if !aiHelper.IsInstalled(ctx) {
		logger.V(1).Info("OpenShift AI not installed, skipping image population")
		return nil
	}

	// List S2I-enabled images
	s2iImages, err := aiHelper.ListS2IImageStreams(ctx)
	if err != nil {
		return fmt.Errorf("failed to list S2I images: %w", err)
	}

	// Convert to API type
	var availableImages []mlopsv1alpha1.AvailableImageInfo
	for _, img := range s2iImages {
		availableImages = append(availableImages, mlopsv1alpha1.AvailableImageInfo{
			Name:        img.Name,
			DisplayName: img.DisplayName,
			Description: img.Description,
			ImageRef:    img.ImageRef,
			S2IEnabled:  img.S2IEnabled,
			Tags:        img.Tags,
		})
	}

	job.Status.BuildStatus.AvailableImages = availableImages

	// Set recommended image (first S2I image, typically s2i-minimal-notebook)
	if len(s2iImages) > 0 {
		job.Status.BuildStatus.RecommendedImage = s2iImages[0].ImageRef
		logger.Info("Recommended S2I image", "image", s2iImages[0].ImageRef, "displayName", s2iImages[0].DisplayName)
	}

	logger.Info("Populated available images", "count", len(availableImages))
	return nil
}
