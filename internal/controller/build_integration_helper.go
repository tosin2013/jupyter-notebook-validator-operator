package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/build"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// PHASE 1: Uses smart build discovery to find latest completed build
// PHASE 2: Auto-triggers stuck builds and checks ImageStream fallback
// PHASE 3: Implements retry logic and cleanup
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

	// Extract BuildConfig name from build name (remove -1, -2, etc. suffix)
	buildConfigName := buildName
	if idx := strings.LastIndex(buildName, "-"); idx > 0 {
		// Check if the suffix is a number
		suffix := buildName[idx+1:]
		if _, err := fmt.Sscanf(suffix, "%d", new(int)); err == nil {
			buildConfigName = buildName[:idx]
		}
	}

	logger.Info("Waiting for build to complete", "buildName", buildName, "buildConfigName", buildConfigName, "timeout", timeout)

	stuckBuildCheckTime := time.Now().Add(2 * time.Minute) // Check for stuck builds after 2 minutes
	stuckBuildTriggered := false

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

			// PHASE 1: Smart Build Discovery - Check for latest completed build
			latestBuild, err := strategy.GetLatestBuild(ctx, buildConfigName)
			if err != nil {
				logger.V(1).Info("No latest build found yet", "buildConfigName", buildConfigName, "error", err)
				// Fall back to checking specific build
				buildInfo, err := strategy.GetBuildStatus(ctx, buildName)
				if err != nil {
					logger.Error(err, "Failed to get build status", "buildName", buildName)
					continue
				}
				latestBuild = buildInfo
			} else {
				logger.V(1).Info("Found latest build", "latestBuildName", latestBuild.Name, "status", latestBuild.Status)
			}

			logger.V(1).Info("Build status check", "buildName", latestBuild.Name, "status", latestBuild.Status, "message", latestBuild.Message)

			switch latestBuild.Status {
			case build.BuildStatusComplete:
				logger.Info("Build completed successfully", "buildName", latestBuild.Name, "image", latestBuild.ImageReference)
				if updateErr := r.updateBuildStatus(ctx, job, "Complete", "Build completed successfully", latestBuild.ImageReference); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}

				// PHASE 3: Cleanup old builds (keep last 3)
				if cleanupErr := strategy.CleanupOldBuilds(ctx, buildConfigName, 3); cleanupErr != nil {
					logger.Error(cleanupErr, "Failed to cleanup old builds")
				}

				return latestBuild.ImageReference, nil

			case build.BuildStatusFailed:
				logger.Error(nil, "Build failed", "buildName", latestBuild.Name, "message", latestBuild.Message)
				if updateErr := r.updateBuildStatus(ctx, job, "Failed", fmt.Sprintf("Build failed: %s", latestBuild.Message), ""); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build failed: %s", latestBuild.Message)

			case build.BuildStatusCancelled:
				logger.Info("Build was cancelled", "buildName", latestBuild.Name)
				if updateErr := r.updateBuildStatus(ctx, job, "Failed", "Build was cancelled", ""); updateErr != nil {
					logger.Error(updateErr, "Failed to update build status")
				}
				return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build was cancelled")

			case build.BuildStatusPending, build.BuildStatusRunning:
				// PHASE 2: Auto-trigger stuck builds
				if latestBuild.Status == build.BuildStatusPending && !stuckBuildTriggered && time.Now().After(stuckBuildCheckTime) {
					logger.Info("Build stuck in Pending status, attempting to trigger", "buildName", latestBuild.Name)
					if triggerErr := strategy.TriggerBuild(ctx, latestBuild.Name); triggerErr != nil {
						logger.Error(triggerErr, "Failed to trigger stuck build")
					} else {
						logger.Info("Successfully triggered stuck build", "buildName", latestBuild.Name)
						stuckBuildTriggered = true
					}
				}

				// Continue waiting
				logger.V(2).Info("Build still in progress", "buildName", latestBuild.Name, "status", latestBuild.Status)

				// PHASE 2: Check ImageStream fallback after 5 minutes
				if time.Now().After(time.Now().Add(-5 * time.Minute)) {
					imageStreamName := buildConfigName
					if imageRef, imgErr := strategy.GetImageFromImageStream(ctx, imageStreamName); imgErr == nil {
						logger.Info("Found image in ImageStream as fallback", "imageStreamName", imageStreamName, "imageRef", imageRef)
						if updateErr := r.updateBuildStatus(ctx, job, "Complete", "Using image from ImageStream", imageRef); updateErr != nil {
							logger.Error(updateErr, "Failed to update build status")
						}
						return imageRef, nil
					}
				}

			default:
				logger.Info("Unknown build status", "buildName", latestBuild.Name, "status", latestBuild.Status)
			}
		}
	}
}

// updateBuildStatus updates the build status in the job with smart retry logic
// Implements exponential backoff for Kubernetes resource version conflicts
func (r *NotebookValidationJobReconciler) updateBuildStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, status, message, imageReference string) error {
	logger := log.FromContext(ctx)

	// Retry configuration for handling resource conflicts
	maxRetries := 5
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			logger.V(1).Info("Retrying status update after conflict", "attempt", attempt, "delay", delay)
			time.Sleep(delay)

			// Fetch latest version of the resource
			latestJob := &mlopsv1alpha1.NotebookValidationJob{}
			if err := r.Get(ctx, client.ObjectKeyFromObject(job), latestJob); err != nil {
				logger.Error(err, "Failed to fetch latest job version for retry")
				return err
			}
			job = latestJob
		}

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

		// Attempt status update
		if err := r.Status().Update(ctx, job); err != nil {
			// Check if this is a resource conflict error
			if apierrors.IsConflict(err) {
				logger.V(1).Info("Resource conflict detected, will retry", "attempt", attempt+1, "maxRetries", maxRetries)
				if attempt < maxRetries-1 {
					continue // Retry
				}
				// Max retries exceeded
				logger.Error(err, "Failed to update build status after max retries", "attempts", maxRetries)
				return fmt.Errorf("resource conflict after %d retries: %w", maxRetries, err)
			}

			// Non-conflict error - fail immediately
			logger.Error(err, "Failed to update build status (non-conflict error)")
			return err
		}

		// Success!
		logger.Info("Build status updated successfully", "status", status, "message", message, "attempts", attempt+1)
		return nil
	}

	// Should never reach here, but just in case
	return fmt.Errorf("failed to update build status after %d attempts", maxRetries)
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
