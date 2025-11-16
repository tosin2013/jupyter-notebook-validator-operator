package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodFailureReason represents specific pod failure reasons
type PodFailureReason string

const (
	FailureReasonImagePull       PodFailureReason = "ImagePullBackOff"
	FailureReasonImagePullError  PodFailureReason = "ErrImagePull"
	FailureReasonCrashLoop       PodFailureReason = "CrashLoopBackOff"
	FailureReasonRunContainer    PodFailureReason = "RunContainerError"
	FailureReasonCreateContainer PodFailureReason = "CreateContainerConfigError"
	FailureReasonInitContainer   PodFailureReason = "InitContainerError"
	FailureReasonPermission      PodFailureReason = "PermissionDenied"
	FailureReasonOOMKilled       PodFailureReason = "OOMKilled"
	FailureReasonUnknown         PodFailureReason = "Unknown"
)

// PodFailureAnalysis contains detailed analysis of pod failure
type PodFailureAnalysis struct {
	Reason          PodFailureReason
	IsTransient     bool   // Can be retried with same configuration
	ShouldRetry     bool   // Should we retry at all
	SuggestedAction string // Human-readable suggestion
	FailedContainer string // Name of failed container
	ErrorMessage    string // Detailed error message
	IsInitContainer bool   // Whether failure is in init container
	IsSCCViolation  bool   // Whether failure is OpenShift SCC violation
	IsImageIssue    bool   // Whether failure is image-related
}

// analyzePodFailure performs detailed analysis of pod failure
// Implements ADR-019: Smart Validation Pod Recovery
func analyzePodFailure(ctx context.Context, pod *corev1.Pod) *PodFailureAnalysis {
	logger := log.FromContext(ctx)

	analysis := &PodFailureAnalysis{
		Reason:      FailureReasonUnknown,
		IsTransient: false,
		ShouldRetry: true,
	}

	logger.V(1).Info("Analyzing pod failure", "podName", pod.Name, "phase", pod.Status.Phase)

	// Check init container statuses first
	for _, initStatus := range pod.Status.InitContainerStatuses {
		if initStatus.State.Waiting != nil {
			waiting := initStatus.State.Waiting
			logger.V(1).Info("Init container waiting", "container", initStatus.Name, "reason", waiting.Reason, "message", waiting.Message)

			analysis.FailedContainer = initStatus.Name
			analysis.IsInitContainer = true
			analysis.ErrorMessage = waiting.Message

			switch waiting.Reason {
			case "ImagePullBackOff":
				analysis.Reason = FailureReasonImagePull
				analysis.IsTransient = true
				analysis.IsImageIssue = true
				analysis.SuggestedAction = "Image pull failed for init container. Check image registry, credentials, and rate limits. Consider using built image to skip init container."
				return analysis

			case "ErrImagePull":
				analysis.Reason = FailureReasonImagePullError
				analysis.IsTransient = true
				analysis.IsImageIssue = true
				analysis.SuggestedAction = "Image pull error for init container. Verify image exists and credentials are correct. Consider using built image to skip init container."
				return analysis

			case "CrashLoopBackOff":
				analysis.Reason = FailureReasonCrashLoop
				analysis.IsTransient = false
				analysis.SuggestedAction = "Init container crashing repeatedly. Check init container logs for errors. Consider using built image to skip init container."
				return analysis

			case "RunContainerError":
				analysis.Reason = FailureReasonRunContainer
				analysis.IsTransient = false
				// Check if it's an SCC violation
				if strings.Contains(strings.ToLower(waiting.Message), "runasnonroot") ||
					strings.Contains(strings.ToLower(waiting.Message), "scc") ||
					strings.Contains(strings.ToLower(waiting.Message), "security context") {
					analysis.IsSCCViolation = true
					analysis.SuggestedAction = `OpenShift SCC violation in init container. The git-clone container cannot run with current security settings.

RECOMMENDED SOLUTION: Use Tekton build strategy designed for OpenShift SCC compliance.

Quick Fix:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"

Why this works: Tekton builds use pipelines-scc during build and produce images that run under restricted SCC.
See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
				} else {
					analysis.SuggestedAction = "Init container failed to run. Check container configuration and logs."
				}
				return analysis

			case "CreateContainerConfigError":
				analysis.Reason = FailureReasonCreateContainer
				analysis.IsTransient = false
				analysis.SuggestedAction = "Init container configuration error. Check volume mounts, environment variables, and security context."
				return analysis
			}
		}

		if initStatus.State.Terminated != nil {
			terminated := initStatus.State.Terminated
			if terminated.ExitCode != 0 {
				logger.V(1).Info("Init container terminated with error", "container", initStatus.Name, "exitCode", terminated.ExitCode, "reason", terminated.Reason)

				analysis.FailedContainer = initStatus.Name
				analysis.IsInitContainer = true
				analysis.ErrorMessage = terminated.Message
				analysis.Reason = FailureReasonInitContainer
				analysis.IsTransient = false

				// Check for git authentication errors (exit code 2 or 128) - MOST COMMON FAILURE
				// This must be checked BEFORE generic permission errors
				if (terminated.ExitCode == 2 || terminated.ExitCode == 128) &&
					(initStatus.Name == "git-clone" || strings.Contains(initStatus.Name, "git")) {
					analysis.Reason = FailureReasonPermission
					analysis.ShouldRetry = false // Terminal error - requires user action

					// Provide specific guidance based on error patterns
					errorLower := strings.ToLower(terminated.Message)
					if strings.Contains(errorLower, "authentication") ||
						strings.Contains(errorLower, "permission denied") ||
						strings.Contains(errorLower, "could not read") ||
						strings.Contains(errorLower, "fatal: could not read from remote repository") {
						analysis.SuggestedAction = `Git authentication failed. The git-clone init container cannot access the repository.

ROOT CAUSE: Git credentials are missing, invalid, or insufficient.

COMMON ISSUES (2025):
- GitHub fine-grained token expired (max 1 year)
- Token missing required permissions (e.g., 'Contents' read)
- Missing tekton.dev/git- annotation for OpenShift Tekton
- Using SSH instead of HTTPS

RECOMMENDED SOLUTION: Use Tekton build with properly configured git credentials.

Quick Fix - Add to your NotebookValidationJob:
  spec:
    notebook:
      git:
        url: "https://github.com/your-org/your-repo.git"
        ref: "main"
        credentialsSecret: "git-credentials"  # ← Must exist with proper annotation!
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

Create git-credentials secret (OpenShift Tekton):
  kubectl create secret generic git-credentials \
    --from-literal=username=oauth2 \
    --from-literal=password=ghp_xxxxxxxxxxxx \
    --dry-run=client -o yaml | \
  kubectl annotate -f - \
    tekton.dev/git-0=https://github.com \
    --local -o yaml | \
  kubectl apply -f -

IMPORTANT: The tekton.dev/git-0 annotation is REQUIRED for OpenShift Tekton!

Generate GitHub fine-grained token (recommended):
  1. Go to: https://github.com/settings/tokens?type=beta
  2. Select repositories
  3. Grant 'Contents' read permission
  4. Set expiration (max 1 year)

Why this works: Tekton clones during build with pipelines-scc, then validation pod uses built image (no git-clone init container).

See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml
Docs: https://docs.redhat.com/en/documentation/red_hat_openshift_pipelines/1.16/html/securing_openshift_pipelines/authenticating-pipelines-repos-using-secrets`
					} else if strings.Contains(errorLower, "repository not found") ||
						strings.Contains(errorLower, "not found") {
						analysis.SuggestedAction = `Git repository not found. The specified repository URL is invalid or inaccessible.

ROOT CAUSE: Repository URL is incorrect, repository is private, or repository was deleted.

TROUBLESHOOTING:
1. Verify repository URL is correct
2. Check if repository is private (requires credentials)
3. Ensure repository exists and is accessible

RECOMMENDED SOLUTION: Use Tekton build with correct repository URL and credentials.

Quick Fix:
  spec:
    notebook:
      git:
        url: "https://github.com/your-org/your-repo.git"  # ← Verify this URL!
        ref: "main"
        credentialsSecret: "git-credentials"  # For private repos
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"

See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					} else {
						// Generic git error - provide comprehensive guidance
						analysis.SuggestedAction = fmt.Sprintf(`Git-clone init container failed (exit code %d). This typically indicates authentication or repository access issues.

COMMON CAUSES:
- Missing or invalid git credentials
- Private repository without credentials
- Repository URL is incorrect
- Network connectivity issues

RECOMMENDED SOLUTION: Use Tekton build with proper git credentials.

Quick Fix:
  spec:
    notebook:
      git:
        url: "https://github.com/your-org/your-repo.git"
        ref: "main"
        credentialsSecret: "git-credentials"  # Create this secret!
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

Create git-credentials secret:
  kubectl create secret generic git-credentials \
    --from-literal=username=oauth2 \
    --from-literal=password=your-github-token

See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`, terminated.ExitCode)
					}
					return analysis
				}

				// Check for permission errors in init containers (ADR-030: Smart Error Messages)
				if terminated.ExitCode == 126 ||
					strings.Contains(strings.ToLower(terminated.Message), "permission denied") ||
					strings.Contains(strings.ToLower(terminated.Message), "permissionerror") ||
					strings.Contains(strings.ToLower(terminated.Message), "cannot execute") ||
					strings.Contains(strings.ToLower(terminated.Message), "access denied") {
					analysis.Reason = FailureReasonPermission
					analysis.ShouldRetry = false // Terminal error - requires user action
					analysis.SuggestedAction = `Git-clone init container failed with permission denied. This is a common OpenShift SCC (Security Context Constraint) issue.

RECOMMENDED SOLUTION: Enable Tekton build to skip the git-clone init container. Tekton handles git cloning during the build phase with proper permissions.

Quick Fix - Add to your NotebookValidationJob:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

How it works:
1. Tekton clones your repo during build (with proper SCC permissions)
2. Builds custom image with notebook and dependencies
3. Validation pod uses built image (no git-clone init container needed)

See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					return analysis
				}

				// Check for command not found in init containers
				if terminated.ExitCode == 127 || strings.Contains(strings.ToLower(terminated.Message), "command not found") {
					analysis.Reason = FailureReasonPermission
					analysis.ShouldRetry = false // Terminal error - requires user action
					analysis.SuggestedAction = `Required command not found in init container. The git-clone image is missing necessary tools.

RECOMMENDED SOLUTION: Use Tekton build to avoid init container issues entirely.

Quick Fix:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"

Alternative: Use a pre-built image with notebooks baked in.
See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					return analysis
				}

				if terminated.Reason == "OOMKilled" {
					analysis.Reason = FailureReasonOOMKilled
					analysis.SuggestedAction = "Init container killed due to out of memory. Increase memory limits."
				} else {
					analysis.SuggestedAction = fmt.Sprintf("Init container failed with exit code %d. Check logs for details. Consider using built image to skip init container.", terminated.ExitCode)
				}
				return analysis
			}
		}
	}

	// Check main container statuses
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			waiting := containerStatus.State.Waiting
			logger.V(1).Info("Container waiting", "container", containerStatus.Name, "reason", waiting.Reason, "message", waiting.Message)

			analysis.FailedContainer = containerStatus.Name
			analysis.ErrorMessage = waiting.Message

			switch waiting.Reason {
			case "ImagePullBackOff":
				analysis.Reason = FailureReasonImagePull
				analysis.IsTransient = true
				analysis.IsImageIssue = true
				analysis.SuggestedAction = "Image pull failed. Check image registry, credentials, and rate limits. Consider fallback to pre-built image."
				return analysis

			case "ErrImagePull":
				analysis.Reason = FailureReasonImagePullError
				analysis.IsTransient = true
				analysis.IsImageIssue = true
				analysis.SuggestedAction = "Image pull error. Verify image exists and credentials are correct. Consider fallback to pre-built image."
				return analysis

			case "CrashLoopBackOff":
				analysis.Reason = FailureReasonCrashLoop
				analysis.IsTransient = false
				analysis.SuggestedAction = "Container crashing repeatedly. Check application logs and dependencies. May need different base image or build strategy."
				return analysis

			case "RunContainerError":
				analysis.Reason = FailureReasonRunContainer
				analysis.IsTransient = false
				// Check if it's an SCC violation
				if strings.Contains(strings.ToLower(waiting.Message), "runasnonroot") ||
					strings.Contains(strings.ToLower(waiting.Message), "scc") ||
					strings.Contains(strings.ToLower(waiting.Message), "security context") {
					analysis.IsSCCViolation = true
					analysis.SuggestedAction = `OpenShift Security Context Constraint (SCC) violation. The container cannot run with current security settings.

ROOT CAUSE: Base image may require privileged access or specific user IDs that conflict with OpenShift's restricted SCC policy.

RECOMMENDED SOLUTION: Use Tekton or BuildConfig build strategies designed for OpenShift SCC compliance.

Tekton Build (Recommended for OpenShift 4.x):
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

S2I Build (Alternative):
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "s2i"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

Why this works: Tekton/S2I builds create images that comply with OpenShift's restricted SCC by default.
See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
				} else {
					analysis.SuggestedAction = "Container failed to run. Check container configuration and logs."
				}
				return analysis

			case "CreateContainerConfigError":
				analysis.Reason = FailureReasonCreateContainer
				analysis.IsTransient = false
				analysis.SuggestedAction = "Container configuration error. Check volume mounts, environment variables, and security context."
				return analysis
			}
		}

		if containerStatus.State.Terminated != nil {
			terminated := containerStatus.State.Terminated
			if terminated.ExitCode != 0 {
				logger.V(1).Info("Container terminated with error", "container", containerStatus.Name, "exitCode", terminated.ExitCode, "reason", terminated.Reason)

				analysis.FailedContainer = containerStatus.Name
				analysis.ErrorMessage = terminated.Message
				analysis.IsTransient = false

				// Check for permission errors (ADR-030: Smart Error Messages)
				// Exit code 126 = Permission denied (cannot execute)
				// Exit code 127 = Command not found
				if terminated.ExitCode == 126 ||
					strings.Contains(strings.ToLower(terminated.Message), "permission denied") ||
					strings.Contains(strings.ToLower(terminated.Message), "permissionerror") ||
					strings.Contains(strings.ToLower(terminated.Message), "cannot execute") ||
					strings.Contains(strings.ToLower(terminated.Message), "access denied") {
					analysis.Reason = FailureReasonPermission
					analysis.ShouldRetry = false // Terminal error - requires user action

					// Provide specific guidance based on the error
					if strings.Contains(strings.ToLower(terminated.Message), "papermill") {
						analysis.SuggestedAction = `Permission denied executing Papermill. The base container image lacks required dependencies or has incorrect permissions.

RECOMMENDED SOLUTION: Enable automatic image building with Tekton or BuildConfig. The operator will build a custom image with all dependencies pre-installed.

Quick Fix:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"

Alternative: Manually create a custom image with Papermill pre-installed.
See: docs/ERROR_HANDLING_GUIDE.md and config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					} else if strings.Contains(strings.ToLower(terminated.Message), "pip") || strings.Contains(strings.ToLower(terminated.Message), "install") {
						analysis.SuggestedAction = `Permission denied during pip package installation. The container cannot write to Python site-packages directory due to OpenShift security constraints.

RECOMMENDED SOLUTION: Use Tekton or BuildConfig to build an image with dependencies pre-installed. This is the operator's native solution for OpenShift environments.

Example configuration:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/scipy-notebook:latest"
        requirementsFile: "requirements.txt"

Why this works: Tekton builds run with elevated permissions and create OpenShift-compatible images that work with restricted SCCs.
See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					} else {
						analysis.SuggestedAction = `Permission denied in container. Check that required files are executable and directories are writable.

RECOMMENDED SOLUTION: Use Tekton/S2I build to create image with correct permissions.

Quick Fix:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"

See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					}
					return analysis
				}

				// Check for command not found errors
				if terminated.ExitCode == 127 || strings.Contains(strings.ToLower(terminated.Message), "command not found") {
					analysis.Reason = FailureReasonPermission
					analysis.ShouldRetry = false // Terminal error - requires user action
					analysis.SuggestedAction = `Required command not found in container. The base container image is missing required dependencies.

RECOMMENDED SOLUTION: Enable automatic dependency installation with Tekton build.

Option 1 - Auto-detect dependencies from requirements.txt:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        baseImage: "quay.io/jupyter/minimal-notebook:latest"
        autoGenerateRequirements: true

Option 2 - Use existing requirements.txt:
  spec:
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"
        requirementsFile: "requirements.txt"

The operator will clone your repository, build a custom image with all dependencies installed, and use it for validation.
See: config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`
					return analysis
				}

				if terminated.Reason == "OOMKilled" {
					analysis.Reason = FailureReasonOOMKilled
					analysis.SuggestedAction = "Container killed due to out of memory. Increase memory limits in podConfig.resources."
				} else if terminated.Reason == "Error" {
					analysis.Reason = FailureReasonUnknown
					analysis.SuggestedAction = fmt.Sprintf("Container failed with exit code %d. Check logs for details.", terminated.ExitCode)
				} else {
					analysis.Reason = FailureReasonUnknown
					analysis.SuggestedAction = fmt.Sprintf("Container terminated: %s (exit code %d)", terminated.Reason, terminated.ExitCode)
				}
				return analysis
			}
		}
	}

	// Check pod conditions for additional insights
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse {
			logger.V(1).Info("Pod condition false", "type", condition.Type, "reason", condition.Reason, "message", condition.Message)

			if condition.Type == corev1.PodScheduled && condition.Reason == "Unschedulable" {
				analysis.Reason = FailureReasonUnknown
				analysis.IsTransient = true
				analysis.ErrorMessage = condition.Message
				analysis.SuggestedAction = "Pod cannot be scheduled. Check resource requests, node selectors, and cluster capacity."
				return analysis
			}
		}
	}

	// If we get here, we couldn't determine specific failure reason
	logger.Info("Could not determine specific pod failure reason", "podName", pod.Name)
	analysis.SuggestedAction = "Pod failed for unknown reason. Check pod events and logs for details."
	return analysis
}

// shouldSkipGitClone determines if git-clone init container should be skipped
// Returns true if using a built image (S2I/Tekton) where notebook is already in the image
func shouldSkipGitClone(containerImage, specImage string) bool {
	// If containerImage is different from spec image, it's a built image
	return containerImage != specImage
}

// getFailureRecoveryAction determines the recovery action based on failure analysis
func getFailureRecoveryAction(analysis *PodFailureAnalysis, retryCount int) string {
	if !analysis.ShouldRetry {
		return "do_not_retry"
	}

	// Max retries per action type
	if retryCount >= MaxRetries {
		return "max_retries_exceeded"
	}

	// Init container failures - try without init container
	if analysis.IsInitContainer {
		if analysis.IsSCCViolation || analysis.IsImageIssue {
			return "skip_init_container"
		}
		return "retry_same_config"
	}

	// Image pull issues - retry with backoff, then fallback
	if analysis.IsImageIssue {
		if retryCount < 2 {
			return "retry_with_backoff"
		}
		return "fallback_to_prebuilt_image"
	}

	// SCC violations - need different approach
	if analysis.IsSCCViolation {
		return "use_openshift_compatible_image"
	}

	// Crash loops - may need different build strategy
	if analysis.Reason == FailureReasonCrashLoop {
		if retryCount < 1 {
			return "retry_same_config"
		}
		return "try_different_build_strategy"
	}

	// OOM - need more resources
	if analysis.Reason == FailureReasonOOMKilled {
		return "increase_resources"
	}

	// Transient errors - retry with backoff
	if analysis.IsTransient {
		return "retry_with_backoff"
	}

	// Non-transient errors - may need manual intervention
	return "manual_intervention_required"
}
