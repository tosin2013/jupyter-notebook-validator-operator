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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/build"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/logging"
)

const (
	// Phases - State Machine (ADR-037)
	PhaseInitializing      = "Initializing"      // Initial state when job is created
	PhaseBuilding          = "Building"          // Build in progress (waiting for build to complete)
	PhaseBuildComplete     = "BuildComplete"     // Build completed successfully (ready for validation)
	PhaseValidationRunning = "ValidationRunning" // Validation pod executing notebook
	PhaseSucceeded         = "Succeeded"         // Terminal success state
	PhaseFailed            = "Failed"            // Terminal failure state
	PhasePending           = "Pending"           // Legacy state (backward compatibility)
	PhaseRunning           = "Running"           // Legacy state (backward compatibility)

	// Condition types
	ConditionTypeReady              = "Ready"
	ConditionTypeGitCloned          = "GitCloned"
	ConditionTypeValidationStarted  = "ValidationStarted"
	ConditionTypeValidationComplete = "ValidationComplete"
	ConditionTypeEnvironmentReady   = "EnvironmentReady"

	// Condition reasons
	ReasonInitializing            = "Initializing"
	ReasonGitCloneInProgress      = "GitCloneInProgress"
	ReasonGitCloneSucceeded       = "GitCloneSucceeded"
	ReasonGitCloneFailed          = "GitCloneFailed"
	ReasonPodCreated              = "PodCreated"
	ReasonPodRunning              = "PodRunning"
	ReasonPodSucceeded            = "PodSucceeded"
	ReasonPodFailed               = "PodFailed"
	ReasonValidationComplete      = "ValidationComplete"
	ReasonEnvironmentSetupFailed  = "EnvironmentSetupFailed"
	ReasonDependencyInstallFailed = "DependencyInstallFailed"
	ReasonNotebookExecutionFailed = "NotebookExecutionFailed"
	ReasonConfigurationError      = "ConfigurationError"

	// Defaults
	DefaultTimeout = 30 * time.Minute
	MaxRetries     = 3
)

// NotebookValidationJobReconciler reconciles a NotebookValidationJob object
type NotebookValidationJobReconciler struct {
	client.Client
	APIReader  client.Reader // Non-cached client for direct API access (used for SCCs)
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

//+kubebuilder:rbac:groups=mlops.mlops.dev,resources=notebookvalidationjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mlops.mlops.dev,resources=notebookvalidationjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mlops.mlops.dev,resources=notebookvalidationjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;update;use,resourceNames=pipelines-scc;privileged
//+kubebuilder:rbac:groups=serving.kserve.io,resources=inferenceservices,verbs=get;list;watch
//+kubebuilder:rbac:groups=serving.kserve.io,resources=servingruntimes,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=ray.io,resources=rayservices;rayclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=machinelearning.seldon.io,resources=seldondeployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=serving.yatai.ai,resources=bentos;bentodeployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=build.openshift.io,resources=buildconfigs;builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines;pipelineruns;taskruns;tasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// The reconciliation loop follows this workflow:
// 1. Fetch the NotebookValidationJob resource
// 2. Initialize status if needed
// 3. Check if validation is already complete
// 4. Clone Git repository with credentials
// 5. Create validation pod
// 6. Monitor pod execution
// 7. Update status with results
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *NotebookValidationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NotebookValidationJob", "name", req.Name, "namespace", req.Namespace)
	logger.V(1).Info("Reconciliation started",
		"namespace", req.Namespace,
		"name", req.Name,
		"timestamp", startTime.Format(time.RFC3339))

	// Fetch the NotebookValidationJob instance
	job := &mlopsv1alpha1.NotebookValidationJob{}
	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			logger.V(1).Info("NotebookValidationJob resource not found, ignoring since object must be deleted",
				"namespace", req.Namespace,
				"name", req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		logger.Error(err, "Failed to get NotebookValidationJob",
			"namespace", req.Namespace,
			"name", req.Name)
		return ctrl.Result{}, err
	}

	logger.V(2).Info("NotebookValidationJob fetched successfully",
		"namespace", req.Namespace,
		"name", req.Name,
		"phase", job.Status.Phase,
		"generation", job.Generation,
		"resourceVersion", job.ResourceVersion)

	// Initialize status if needed (ADR-037: State Machine)
	if job.Status.Phase == "" {
		logger.Info("Initializing NotebookValidationJob status",
			"namespace", req.Namespace,
			"name", req.Name)
		job.Status.Phase = PhaseInitializing
		job.Status.StartTime = &metav1.Time{Time: time.Now()}
		job.Status.RetryCount = 0

		// Set initial condition
		condition := metav1.Condition{
			Type:               ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonInitializing,
			Message:            "NotebookValidationJob is initializing",
			LastTransitionTime: metav1.Now(),
		}
		job.Status.Conditions = []metav1.Condition{condition}

		if err := r.Status().Update(ctx, job); err != nil {
			logger.Error(err, "Failed to update NotebookValidationJob status")
			return ctrl.Result{}, err
		}

		// Requeue to continue processing
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if validation is already complete
	if job.Status.Phase == PhaseSucceeded || job.Status.Phase == PhaseFailed {
		// Sanity check: If phase is terminal but completion time is not set,
		// this might be stale cache data from a conflict error. Re-fetch from etcd.
		if job.Status.CompletionTime == nil {
			logger.Info("Terminal phase detected without completion time - re-fetching from etcd to verify",
				"phase", job.Status.Phase,
				"resourceVersion", job.ResourceVersion)

			// Force fetch from etcd (bypass cache)
			freshJob := &mlopsv1alpha1.NotebookValidationJob{}
			if err := r.Client.Get(ctx, req.NamespacedName, freshJob); err != nil {
				logger.Error(err, "Failed to re-fetch job from etcd")
				return ctrl.Result{}, err
			}

			// Update our local reference to the fresh object
			*job = *freshJob
			logger.Info("Re-fetched job from etcd",
				"phase", job.Status.Phase,
				"completionTime", job.Status.CompletionTime,
				"resourceVersion", job.ResourceVersion)

			// If still in terminal phase with completion time, it's legitimate
			if (job.Status.Phase == PhaseSucceeded || job.Status.Phase == PhaseFailed) && job.Status.CompletionTime != nil {
				logger.Info("NotebookValidationJob already complete (verified from etcd)", "phase", job.Status.Phase)
				return ctrl.Result{}, nil
			}

			// Otherwise, continue processing with fresh data
			logger.Info("Job was not actually complete - continuing with fresh data", "phase", job.Status.Phase)
		} else {
			logger.Info("NotebookValidationJob already complete", "phase", job.Status.Phase)
			return ctrl.Result{}, nil
		}
	}

	// Check retry limit
	if job.Status.RetryCount >= MaxRetries {
		logger.Info("Max retries exceeded", "retryCount", job.Status.RetryCount)
		return r.updateJobPhase(ctx, job, PhaseFailed, "Maximum retry attempts exceeded")
	}

	// ADR-037: State Machine Dispatch
	// Dispatch to appropriate reconciliation function based on current phase
	logger.V(1).Info("Dispatching based on phase", "phase", job.Status.Phase)

	var result ctrl.Result
	var err error

	switch job.Status.Phase {
	case PhaseInitializing:
		logger.Info("Phase: Initializing - Setting up initial state")
		result, err = r.reconcileInitializing(ctx, job)

	case PhaseBuilding:
		logger.Info("Phase: Building - Checking build status")
		result, err = r.reconcileBuilding(ctx, job)

	case PhaseBuildComplete:
		logger.Info("Phase: BuildComplete - Transitioning to validation")
		result, err = r.reconcileBuildComplete(ctx, job)

	case PhaseValidationRunning:
		logger.Info("Phase: ValidationRunning - Monitoring validation pod")
		result, err = r.reconcileValidationRunning(ctx, job)

	case PhasePending, PhaseRunning:
		// Legacy states - migrate to new state machine
		logger.Info("Legacy phase detected, migrating to new state machine", "oldPhase", job.Status.Phase)
		if isBuildEnabled(job) {
			// If build is enabled, start from Building phase
			result, err = r.transitionPhase(ctx, job, PhaseBuilding, "Migrating from legacy phase to Building")
		} else {
			// If no build, start from ValidationRunning phase
			result, err = r.transitionPhase(ctx, job, PhaseValidationRunning, "Migrating from legacy phase to ValidationRunning")
		}

	default:
		logger.Error(nil, "Unknown phase", "phase", job.Status.Phase)
		return r.updateJobPhase(ctx, job, PhaseFailed, fmt.Sprintf("Unknown phase: %s", job.Status.Phase))
	}

	if err != nil {
		logger.Error(err, "Error during phase reconciliation", "phase", job.Status.Phase)
		// Record reconciliation duration with error result
		recordReconciliationDuration(req.Namespace, "error", time.Since(startTime).Seconds())
		// Classify error and handle accordingly (ADR-011)
		return r.handleReconcileError(ctx, job, err)
	}

	// Record reconciliation duration with success result
	recordReconciliationDuration(req.Namespace, "success", time.Since(startTime).Seconds())
	return result, nil
}

// ADR-037: State Machine Reconciliation Functions

// transitionPhase transitions the job to a new phase with logging and status update
func (r *NotebookValidationJobReconciler) transitionPhase(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, newPhase, message string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Transitioning phase", "oldPhase", job.Status.Phase, "newPhase", newPhase, "message", message)

	// Save original phase in case update fails
	originalPhase := job.Status.Phase

	job.Status.Phase = newPhase
	if err := r.Status().Update(ctx, job); err != nil {
		// Restore original phase on failure to prevent stale in-memory state
		job.Status.Phase = originalPhase

		logger.Error(err, "Failed to update job phase", "newPhase", newPhase)
		return ctrl.Result{}, err
	}

	// Requeue immediately to process new phase
	return ctrl.Result{Requeue: true}, nil
}

// reconcileInitializing handles the Initializing phase
// Sets up initial state and determines whether to build or validate directly
func (r *NotebookValidationJobReconciler) reconcileInitializing(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Initializing phase")

	// Determine next phase based on build configuration
	if isBuildEnabled(job) {
		// Build is enabled - transition to Building phase
		logger.Info("Build enabled, transitioning to Building phase")
		return r.transitionPhase(ctx, job, PhaseBuilding, "Build enabled, starting build workflow")
	}

	// No build needed - transition directly to ValidationRunning
	logger.Info("Build not enabled, transitioning to ValidationRunning phase")
	return r.transitionPhase(ctx, job, PhaseValidationRunning, "No build required, starting validation")
}

// reconcileBuilding handles the Building phase
// Checks build status and waits for completion with 30-second requeue
func (r *NotebookValidationJobReconciler) reconcileBuilding(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Building phase")

	// Initialize build status if needed
	if job.Status.BuildStatus == nil {
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
			Phase:     "Pending",
			Message:   "Build initialization",
			StartTime: &metav1.Time{Time: time.Now()},
		}
		if err := r.Status().Update(ctx, job); err != nil {
			logger.Error(err, "Failed to initialize build status")
			return ctrl.Result{}, err
		}
	}

	// Initialize build registry with all available strategies
	registry := build.NewStrategyRegistry(r.Client, r.APIReader, r.Scheme)

	// Get the configured strategy
	strategyName := job.Spec.PodConfig.BuildConfig.Strategy
	if strategyName == "" {
		strategyName = "s2i" // Default to S2I
	}

	strategy := registry.GetStrategy(strategyName)
	if strategy == nil {
		logger.Error(nil, "Build strategy not found", "strategy", strategyName)
		return r.transitionPhase(ctx, job, PhaseFailed, fmt.Sprintf("Build strategy not found: %s", strategyName))
	}

	// Check if build needs to be created
	buildName := fmt.Sprintf("%s-build", job.Name)
	// ADR-042: Use GetLatestBuild for build discovery
	// This works for both Tekton (PipelineRuns) and S2I (Builds with -1, -2 suffixes)
	buildInfo, err := strategy.GetLatestBuild(ctx, buildName)

	if err != nil {
		// Build doesn't exist yet, create it
		logger.Info("Build not found, creating new build", "buildName", buildName)

		// Validate strategy availability
		available, detectErr := strategy.Detect(ctx, r.Client)
		if detectErr != nil || !available {
			logger.Error(detectErr, "Build strategy not available", "strategy", strategyName)
			return r.transitionPhase(ctx, job, PhaseFailed, fmt.Sprintf("Build strategy not available: %s", strategyName))
		}

		// Validate configuration
		if validateErr := strategy.ValidateConfig(job.Spec.PodConfig.BuildConfig); validateErr != nil {
			logger.Error(validateErr, "Invalid build configuration")
			return r.transitionPhase(ctx, job, PhaseFailed, fmt.Sprintf("Invalid build configuration: %v", validateErr))
		}

		// Create build
		buildInfo, err = strategy.CreateBuild(ctx, job)
		if err != nil {
			logger.Error(err, "Failed to create build")
			return r.transitionPhase(ctx, job, PhaseFailed, fmt.Sprintf("Failed to create build: %v", err))
		}

		// Update build status
		originalBuildStatus := job.Status.BuildStatus
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
			Phase:     "Running",
			Message:   "Build created and started",
			BuildName: buildInfo.Name,
			Strategy:  strategyName,
			StartTime: &metav1.Time{Time: time.Now()},
		}
		if err := r.Status().Update(ctx, job); err != nil {
			// Restore original build status on failure
			job.Status.BuildStatus = originalBuildStatus
			logger.Error(err, "Failed to update build status")
			return ctrl.Result{}, err
		}

		// Requeue to check build status
		logger.Info("Build created, requeuing to check status", "buildName", buildInfo.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Build exists, check its status
	logger.Info("Checking build status", "buildName", buildName, "status", buildInfo.Status)

	// Calculate duration if build has started
	var duration string
	if job.Status.BuildStatus != nil && job.Status.BuildStatus.StartTime != nil {
		elapsed := time.Since(job.Status.BuildStatus.StartTime.Time)
		duration = elapsed.Round(time.Second).String()
	}

	switch buildInfo.Status {
	case build.BuildStatusComplete:
		// Build completed successfully
		logger.Info("Build completed successfully", "image", buildInfo.ImageReference, "duration", duration)

		// Update build status with completion details
		originalBuildStatus := job.Status.BuildStatus
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
			Phase:          "Complete",
			Message:        "Build completed successfully",
			BuildName:      buildInfo.Name,
			Strategy:       strategyName,
			ImageReference: buildInfo.ImageReference,
			StartTime:      job.Status.BuildStatus.StartTime,
			CompletionTime: &metav1.Time{Time: time.Now()},
			Duration:       duration,
		}
		if err := r.Status().Update(ctx, job); err != nil {
			// Restore original build status on failure
			job.Status.BuildStatus = originalBuildStatus
			logger.Error(err, "Failed to update build status")
			return ctrl.Result{}, err
		}

		// Transition to BuildComplete phase
		return r.transitionPhase(ctx, job, PhaseBuildComplete, "Build completed, ready for validation")

	case build.BuildStatusFailed:
		// Build failed
		logger.Error(nil, "Build failed", "message", buildInfo.Message, "duration", duration)

		// Update build status with failure details
		originalBuildStatus := job.Status.BuildStatus
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
			Phase:          "Failed",
			Message:        fmt.Sprintf("Build failed: %s", buildInfo.Message),
			BuildName:      buildInfo.Name,
			Strategy:       strategyName,
			StartTime:      job.Status.BuildStatus.StartTime,
			CompletionTime: &metav1.Time{Time: time.Now()},
			Duration:       duration,
		}
		if err := r.Status().Update(ctx, job); err != nil {
			// Restore original build status on failure
			job.Status.BuildStatus = originalBuildStatus
			logger.Error(err, "Failed to update build status")
			return ctrl.Result{}, err
		}

		// Transition to Failed phase
		return r.transitionPhase(ctx, job, PhaseFailed, fmt.Sprintf("Build failed: %s", buildInfo.Message))

	case build.BuildStatusPending, build.BuildStatusRunning:
		// Build still in progress
		logger.Info("Build in progress, requeuing", "status", buildInfo.Status, "duration", duration)

		// Update build status with current progress
		if job.Status.BuildStatus != nil {
			originalBuildStatus := job.Status.BuildStatus.DeepCopy()
			job.Status.BuildStatus.Phase = string(buildInfo.Status)
			job.Status.BuildStatus.Message = fmt.Sprintf("Build %s", buildInfo.Status)
			job.Status.BuildStatus.Duration = duration
			if err := r.Status().Update(ctx, job); err != nil {
				// Restore original build status on failure
				job.Status.BuildStatus = originalBuildStatus
				logger.Error(err, "Failed to update build status")
			}
		}

		// Requeue after 30 seconds to check again
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	default:
		logger.Info("Unknown build status, requeuing", "status", buildInfo.Status)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
}

// reconcileBuildComplete handles the BuildComplete phase
// Immediately transitions to ValidationRunning phase
func (r *NotebookValidationJobReconciler) reconcileBuildComplete(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling BuildComplete phase")

	// Verify build status contains image reference
	if job.Status.BuildStatus == nil || job.Status.BuildStatus.ImageReference == "" {
		logger.Error(nil, "BuildComplete phase but no image reference found")
		return r.transitionPhase(ctx, job, PhaseFailed, "Build completed but no image reference available")
	}

	logger.Info("Build completed with image, transitioning to validation", "image", job.Status.BuildStatus.ImageReference)

	// Transition to ValidationRunning phase
	return r.transitionPhase(ctx, job, PhaseValidationRunning, "Build complete, starting validation")
}

// reconcileValidationRunning handles the ValidationRunning phase
// Delegates to the existing reconcileValidation logic
func (r *NotebookValidationJobReconciler) reconcileValidationRunning(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling ValidationRunning phase")

	// Use existing reconcileValidation logic
	// This function already handles pod creation, monitoring, and result collection
	return r.reconcileValidation(ctx, job)
}

// reconcileValidation handles the main validation workflow
func (r *NotebookValidationJobReconciler) reconcileValidation(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Step 1: Perform model validation if enabled (Phase 4.4: Model-Aware Validation)
	if isModelValidationEnabled(job) {
		logger.Info("Model validation enabled, performing platform detection")
		if err := r.performModelValidation(ctx, job); err != nil {
			logger.Error(err, "Model validation failed")
			// Update status but don't fail the job - model validation is optional
			if updateErr := r.updateModelValidationStatus(ctx, job, false, fmt.Sprintf("Model validation failed: %v", err)); updateErr != nil {
				logger.Error(updateErr, "Failed to update model validation status")
			}
			// Continue with notebook validation even if model validation fails
			logger.Info("Continuing with notebook validation despite model validation failure")
		} else {
			logger.Info("Model validation completed successfully")
			if updateErr := r.updateModelValidationStatus(ctx, job, true, "Model validation completed successfully"); updateErr != nil {
				logger.Error(updateErr, "Failed to update model validation status")
			}
		}
	}

	// Step 2: Determine container image to use (ADR-037: Use built image from BuildStatus if available)
	containerImage := job.Spec.PodConfig.ContainerImage
	if job.Status.BuildStatus != nil && job.Status.BuildStatus.ImageReference != "" {
		// Use built image from BuildStatus (set by reconcileBuilding)
		logger.Info("Using built image from BuildStatus", "image", job.Status.BuildStatus.ImageReference)
		containerImage = job.Status.BuildStatus.ImageReference
	} else if isBuildEnabled(job) {
		// Build is enabled but no built image yet (should not happen in state machine flow)
		// This is a safety check for legacy jobs
		logger.Info("Build enabled but no built image in BuildStatus, using spec image", "image", containerImage)
	}

	// Step 3: Check if validation pod already exists
	podName := fmt.Sprintf("%s-validation", job.Name)
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: job.Namespace}, pod)

	if err != nil && errors.IsNotFound(err) {
		// Pod doesn't exist, create it
		logger.Info("Creating validation pod", "podName", podName)

		// Update phase to Running
		if job.Status.Phase != PhaseRunning {
			if _, err := r.updateJobPhase(ctx, job, PhaseRunning, "Starting validation"); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Create the validation pod with the container image (built or spec)
		pod, err := r.createValidationPod(ctx, job, containerImage)
		if err != nil {
			logger.Error(err, "Failed to create validation pod")
			// Record pod creation failure
			recordPodCreation(job.Namespace, StatusFailed)
			return r.updateJobPhase(ctx, job, PhaseFailed, fmt.Sprintf("Failed to create validation pod: %v", err))
		}

		// Record successful pod creation
		recordPodCreation(job.Namespace, "success")

		// Update status with pod name
		job.Status.ValidationPodName = pod.Name
		if err := r.Status().Update(ctx, job); err != nil {
			logger.Error(err, "Failed to update job status with pod name")
			return ctrl.Result{}, err
		}

		logger.Info("Validation pod created successfully", "podName", pod.Name)

		// Requeue to check pod status
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	} else if err != nil {
		// Error fetching pod
		logger.Error(err, "Failed to get validation pod")
		return ctrl.Result{}, err
	}

	// Pod exists, check its status
	logger.Info("Checking validation pod status", "podName", pod.Name, "phase", pod.Status.Phase)

	switch pod.Status.Phase {
	case corev1.PodPending:
		logger.Info("Validation pod is pending")

		// ADR-019: Check if pod is stuck due to init container failures
		// Analyze pod to detect ImagePullBackOff, SCC violations, etc.
		analysis := analyzePodFailure(ctx, pod)
		if analysis.Reason != FailureReasonUnknown {
			logger.Info("Detected failure in pending pod",
				"reason", analysis.Reason,
				"failedContainer", analysis.FailedContainer,
				"isInitContainer", analysis.IsInitContainer,
				"suggestedAction", analysis.SuggestedAction)

			// Treat as pod failure and handle recovery
			return r.handlePodFailure(ctx, job, pod)
		}

		// Update active pod gauge
		setActivePods(job.Namespace, "pending", 1)
		setActivePods(job.Namespace, "running", 0)
		// Requeue to check again
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

	case corev1.PodRunning:
		logger.Info("Validation pod is running")
		// Update active pod gauge
		setActivePods(job.Namespace, "pending", 0)
		setActivePods(job.Namespace, "running", 1)
		// Requeue to check again
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil

	case corev1.PodSucceeded:
		logger.Info("Validation pod succeeded, collecting logs and results")
		// Clear active pod gauges
		setActivePods(job.Namespace, "pending", 0)
		setActivePods(job.Namespace, "running", 0)
		// Collect logs and parse results
		return r.handlePodSuccess(ctx, job, pod)

	case corev1.PodFailed:
		logger.Info("Validation pod failed, collecting logs and handling retry")
		// Clear active pod gauges
		setActivePods(job.Namespace, "pending", 0)
		setActivePods(job.Namespace, "running", 0)
		// Handle pod failure with retry logic
		return r.handlePodFailure(ctx, job, pod)

	default:
		logger.Info("Validation pod in unknown state", "phase", pod.Status.Phase)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

// createValidationPod creates a pod for notebook validation
// containerImage parameter allows using a custom built image (Phase 4.5: S2I Build Integration)
func (r *NotebookValidationJobReconciler) createValidationPod(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, containerImage string) (*corev1.Pod, error) {
	logger := log.FromContext(ctx)

	podName := fmt.Sprintf("%s-validation", job.Name)

	logger.V(1).Info("Creating validation pod",
		"namespace", job.Namespace,
		"name", job.Name,
		"podName", podName)

	// ADR-019: Smart Validation Pod Recovery
	// Check if we should skip git-clone init container
	// When using a built image (S2I/Tekton), the notebook is already in the image
	var initContainers []corev1.Container

	if shouldSkipGitClone(containerImage, job.Spec.PodConfig.ContainerImage) {
		logger.Info("Using built image - notebook already in image, skipping git-clone init container",
			"builtImage", containerImage,
			"specImage", job.Spec.PodConfig.ContainerImage)
		// No init containers needed - notebook is in the built image
		initContainers = []corev1.Container{}
	} else {
		logger.Info("Using pre-built image - adding git-clone init container",
			"image", containerImage)

		// Resolve Git credentials (ADR-009)
		logger.V(1).Info("Resolving Git credentials",
			"namespace", job.Namespace,
			"name", job.Name)
		creds, err := r.resolveGitCredentials(ctx, job)
		if err != nil {
			logger.Error(logging.SanitizeError(err), "Failed to resolve Git credentials",
				"namespace", job.Namespace,
				"name", job.Name)
			return nil, fmt.Errorf("failed to resolve Git credentials: %w", err)
		}

		logger.V(2).Info("Git credentials resolved",
			"namespace", job.Namespace,
			"name", job.Name,
			"credentialType", creds.Type)

		// Build Git clone init container (ADR-009)
		logger.Info("Building Git clone init container")
		gitCloneContainer, err := r.buildGitCloneInitContainer(ctx, job, creds)
		if err != nil {
			logger.Error(err, "Failed to build Git clone init container")
			return nil, fmt.Errorf("failed to build Git clone init container: %w", err)
		}

		// Build init containers list
		initContainers = []corev1.Container{gitCloneContainer}

		// Add golden notebook init container if specified (Phase 3: Golden Notebook Comparison)
		if job.Spec.GoldenNotebook != nil {
			logger.Info("Building golden notebook Git clone init container")
			goldenCreds, err := r.resolveGoldenGitCredentials(ctx, job)
			if err != nil {
				logger.Error(err, "Failed to resolve golden notebook credentials")
				return nil, fmt.Errorf("failed to resolve golden notebook credentials: %w", err)
			}

			goldenCloneContainer, err := r.buildGoldenGitCloneInitContainer(ctx, job, goldenCreds)
			if err != nil {
				logger.Error(err, "Failed to build golden Git clone init container")
				return nil, fmt.Errorf("failed to build golden Git clone init container: %w", err)
			}

			initContainers = append(initContainers, goldenCloneContainer)
			logger.Info("Added golden notebook init container")
		}
	}

	// Build pod spec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                          "jupyter-notebook-validator",
				"notebookvalidationjob":        job.Name,
				"app.kubernetes.io/name":       "jupyter-notebook-validator-operator",
				"app.kubernetes.io/component":  "validation-pod",
				"app.kubernetes.io/managed-by": "jupyter-notebook-validator-operator",
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: job.Spec.PodConfig.ServiceAccountName,
			RestartPolicy:      corev1.RestartPolicyNever,
			// ADR-005: OpenShift Compatibility
			// SecurityContext is intentionally omitted at pod level
			// OpenShift automatically assigns appropriate UID/GID/fsGroup from namespace ranges
			// Container-level SecurityContext (in buildPapermillValidationContainer) ensures:
			//   - RunAsNonRoot: true
			//   - AllowPrivilegeEscalation: false
			//   - Capabilities dropped
			// Combined with HOME=/workspace and PYTHONUSERBASE=/workspace/.local env vars,
			// this ensures notebooks work with OpenShift's default restricted SCC
			InitContainers: initContainers,
			Containers: []corev1.Container{
				r.buildPapermillValidationContainer(ctx, job, containerImage),
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					// ADR-005: OpenShift Compatibility
					// Jupyter containers expect /home/jovyan to exist and be writable during startup
					// Mount an emptyDir at /home/jovyan to satisfy this requirement
					// OpenShift automatically makes emptyDir writable by the assigned UID
					// Combined with HOME=/workspace env var, this prevents startup failures
					// while redirecting actual work to the workspace volume
					Name: "jovyan-home",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	// Build environment variables list
	envVars := make([]corev1.EnvVar, 0)

	// Add model validation environment variables if enabled (Phase 4.4: Model-Aware Validation)
	if isModelValidationEnabled(job) {
		logger.Info("Adding model validation environment variables")
		modelValidationEnvVars := r.buildModelValidationEnvVars(ctx, job)
		envVars = append(envVars, modelValidationEnvVars...)
	}

	// Add user-specified environment variables
	if len(job.Spec.PodConfig.Env) > 0 {
		for _, env := range job.Spec.PodConfig.Env {
			envVar := corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			}

			// Handle valueFrom
			if env.ValueFrom != nil {
				envVar.ValueFrom = &corev1.EnvVarSource{}
				if env.ValueFrom.SecretKeyRef != nil {
					envVar.ValueFrom.SecretKeyRef = &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: env.ValueFrom.SecretKeyRef.Name,
						},
						Key: env.ValueFrom.SecretKeyRef.Key,
					}
				}
				if env.ValueFrom.ConfigMapKeyRef != nil {
					envVar.ValueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: env.ValueFrom.ConfigMapKeyRef.Name,
						},
						Key: env.ValueFrom.ConfigMapKeyRef.Key,
					}
				}
			}

			envVars = append(envVars, envVar)
		}
	}

	// Set environment variables on container
	if len(envVars) > 0 {
		pod.Spec.Containers[0].Env = envVars
	}

	// Add envFrom if specified (Phase 4: Credential Management)
	envFromSources := make([]corev1.EnvFromSource, 0)

	// First, add explicit envFrom entries
	if len(job.Spec.PodConfig.EnvFrom) > 0 {
		for _, envFrom := range job.Spec.PodConfig.EnvFrom {
			envFromSource := corev1.EnvFromSource{}

			// Handle secretRef
			if envFrom.SecretRef != nil {
				envFromSource.SecretRef = &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: envFrom.SecretRef.Name,
					},
				}
			}

			// Handle configMapRef
			if envFrom.ConfigMapRef != nil {
				envFromSource.ConfigMapRef = &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: envFrom.ConfigMapRef.Name,
					},
				}
			}

			envFromSources = append(envFromSources, envFromSource)
		}
	}

	// Then, add credentials as secretRef entries (syntactic sugar)
	// This allows users to simply specify: credentials: ["aws-credentials", "database-credentials"]
	// instead of the more verbose envFrom syntax
	if len(job.Spec.PodConfig.Credentials) > 0 {
		logger.Info("Converting credentials to envFrom", "credentials", job.Spec.PodConfig.Credentials)
		for _, credentialName := range job.Spec.PodConfig.Credentials {
			envFromSource := corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: credentialName,
					},
				},
			}
			envFromSources = append(envFromSources, envFromSource)
		}
	}

	// Apply all envFrom sources to the pod
	if len(envFromSources) > 0 {
		pod.Spec.Containers[0].EnvFrom = envFromSources
	}

	// Add resource requirements if specified
	if job.Spec.PodConfig.Resources != nil {
		resources := corev1.ResourceRequirements{}

		if job.Spec.PodConfig.Resources.Requests != nil {
			resources.Requests = make(corev1.ResourceList)
			for k, v := range job.Spec.PodConfig.Resources.Requests {
				resources.Requests[corev1.ResourceName(k)] = parseQuantity(v)
			}
		}

		if job.Spec.PodConfig.Resources.Limits != nil {
			resources.Limits = make(corev1.ResourceList)
			for k, v := range job.Spec.PodConfig.Resources.Limits {
				resources.Limits[corev1.ResourceName(k)] = parseQuantity(v)
			}
		}

		pod.Spec.Containers[0].Resources = resources
	}

	// Set owner reference for garbage collection
	if err := ctrl.SetControllerReference(job, pod, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return nil, err
	}

	// Create the pod
	if err := r.Create(ctx, pod); err != nil {
		logger.Error(err, "Failed to create pod")
		return nil, err
	}

	logger.Info("Pod created successfully", "podName", pod.Name)
	return pod, nil
}

// updateJobPhase updates the job phase and completion time
func (r *NotebookValidationJobReconciler) updateJobPhase(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, phase string, message string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Save original values in case update fails
	originalPhase := job.Status.Phase
	originalMessage := job.Status.Message
	originalCompletionTime := job.Status.CompletionTime
	originalConditions := make([]metav1.Condition, len(job.Status.Conditions))
	copy(originalConditions, job.Status.Conditions)

	job.Status.Phase = phase
	job.Status.Message = message

	if phase == PhaseSucceeded || phase == PhaseFailed {
		now := metav1.Now()
		job.Status.CompletionTime = &now
	}

	// Update condition
	condition := metav1.Condition{
		Type:               ConditionTypeValidationComplete,
		LastTransitionTime: metav1.Now(),
	}

	if phase == PhaseSucceeded {
		condition.Status = metav1.ConditionTrue
		condition.Reason = ReasonValidationComplete
		condition.Message = message
	} else if phase == PhaseFailed {
		condition.Status = metav1.ConditionFalse
		condition.Reason = ReasonPodFailed
		condition.Message = message
	} else {
		condition.Status = metav1.ConditionUnknown
		condition.Reason = ReasonPodRunning
		condition.Message = message
	}

	// Update or append condition
	job.Status.Conditions = updateCondition(job.Status.Conditions, condition)

	if err := r.Status().Update(ctx, job); err != nil {
		// Restore original values on failure to prevent stale in-memory state
		// from being inadvertently persisted by subsequent reconcile loops
		job.Status.Phase = originalPhase
		job.Status.Message = originalMessage
		job.Status.CompletionTime = originalCompletionTime
		job.Status.Conditions = originalConditions

		logger.Error(err, "Failed to update job status")
		return ctrl.Result{}, err
	}

	logger.Info("Job phase updated", "phase", phase, "message", message)
	return ctrl.Result{}, nil
}

// handleReconcileError handles errors during reconciliation (ADR-011)
func (r *NotebookValidationJobReconciler) handleReconcileError(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Classify error type
	errorType := classifyError(err)

	switch errorType {
	case "Transient":
		// Transient errors: retry with exponential backoff
		logger.Info("Transient error detected, will retry", "error", err)
		return ctrl.Result{RequeueAfter: time.Minute}, nil

	case "Retriable":
		// Retriable errors: increment retry count and retry with backoff
		job.Status.RetryCount++
		job.Status.LastRetryTime = &metav1.Time{Time: time.Now()}
		if err := r.Status().Update(ctx, job); err != nil {
			return ctrl.Result{}, err
		}

		if job.Status.RetryCount >= MaxRetries {
			logger.Info("Max retries exceeded for retriable error")
			return r.updateJobPhase(ctx, job, PhaseFailed, fmt.Sprintf("Maximum retries exceeded: %v", err))
		}

		// Exponential backoff: 1m, 2m, 5m
		backoff := time.Minute * time.Duration(1<<uint(job.Status.RetryCount-1))
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}

		logger.Info("Retriable error detected, will retry", "error", err, "retryCount", job.Status.RetryCount, "backoff", backoff)
		return ctrl.Result{RequeueAfter: backoff}, nil

	case "Terminal":
		// Terminal errors: mark as failed immediately
		logger.Error(err, "Terminal error detected, marking job as failed")
		return r.updateJobPhase(ctx, job, PhaseFailed, fmt.Sprintf("Terminal error: %v", err))

	default:
		// Unknown error type: treat as retriable
		logger.Error(err, "Unknown error type, treating as retriable")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
}

// classifyError classifies errors according to ADR-011
func classifyError(err error) string {
	if err == nil {
		return ""
	}

	// Transient errors - these should requeue without incrementing retry count
	// Conflicts are transient because they occur when multiple reconciliation loops
	// try to update the same resource - the next reconciliation will succeed
	if errors.IsConflict(err) || errors.IsServerTimeout(err) || errors.IsTimeout(err) || errors.IsServiceUnavailable(err) {
		return "Transient"
	}

	// Terminal errors
	if errors.IsInvalid(err) || errors.IsBadRequest(err) || errors.IsForbidden(err) {
		return "Terminal"
	}

	// Default to retriable
	return "Retriable"
}

// updateCondition updates or appends a condition to the condition list
func updateCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	for i, condition := range conditions {
		if condition.Type == newCondition.Type {
			conditions[i] = newCondition
			return conditions
		}
	}
	return append(conditions, newCondition)
}

// parseQuantity is a helper to parse resource quantities
func parseQuantity(value string) resource.Quantity {
	quantity, _ := resource.ParseQuantity(value)
	return quantity
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotebookValidationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mlopsv1alpha1.NotebookValidationJob{}).
		Owns(&corev1.Pod{}). // Watch pods owned by NotebookValidationJob
		Complete(r)
}
