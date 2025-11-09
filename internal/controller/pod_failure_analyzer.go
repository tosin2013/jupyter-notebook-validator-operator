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
	IsTransient     bool          // Can be retried with same configuration
	ShouldRetry     bool          // Should we retry at all
	SuggestedAction string        // Human-readable suggestion
	FailedContainer string        // Name of failed container
	ErrorMessage    string        // Detailed error message
	IsInitContainer bool          // Whether failure is in init container
	IsSCCViolation  bool          // Whether failure is OpenShift SCC violation
	IsImageIssue    bool          // Whether failure is image-related
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
					analysis.SuggestedAction = "OpenShift SCC violation in init container. Use built image to skip git-clone init container."
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
					analysis.SuggestedAction = "OpenShift SCC violation. Check security context and use S2I/Tekton build with OpenShift-compatible base image."
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

