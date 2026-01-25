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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	smarterrors "github.com/tosin2013/jupyter-notebook-validator-operator/pkg/errors"
)

// Condition types for NotebookValidationJob
// ADR-030: Smart Error Messages and User Feedback - Phase 4: Status Conditions
const (
	// ConditionTypeBuildReady indicates whether the build phase has completed
	ConditionTypeBuildReady = "BuildReady"

	// ConditionTypeValidationReady indicates whether validation is ready or complete
	ConditionTypeValidationReady = "ValidationReady"

	// ConditionTypeProgressing indicates the job is making progress
	ConditionTypeProgressing = "Progressing"

	// ConditionTypeAvailable indicates the job has completed successfully
	ConditionTypeAvailable = "Available"
)

// Condition reasons - ADR-030
// Note: Some reasons like ReasonBuildInProgress, ReasonBuildTimeout, ReasonPodFailed,
// and ReasonValidationComplete are defined in other files
const (
	// Build reasons (not duplicating those in build_integration_helper.go)
	ReasonBuildPending      = "BuildPending"
	ReasonBuildComplete     = "BuildComplete"
	ReasonBuildFailed       = "BuildFailed"
	ReasonBuildSkipped      = "BuildSkipped"
	ReasonRBACError         = "RBACPermissionDenied"
	ReasonTaskNotFound      = "TaskNotFound"
	ReasonParameterMismatch = "ParameterMismatch"
	ReasonGitAuthFailed     = "GitAuthenticationFailed"

	// Validation reasons (not duplicating those in notebookvalidationjob_controller.go)
	ReasonValidationPending    = "ValidationPending"
	ReasonValidationInProgress = "ValidationInProgress"
	ReasonValidationFailed     = "ValidationFailed"
	ReasonValidationTimeout    = "ValidationTimeout"
	ReasonWaitingForBuild      = "WaitingForBuild"
	ReasonPodCreationFailed    = "PodCreationFailed"

	// Progress reasons
	ReasonInProgress           = "InProgress"
	ReasonStalled              = "Stalled"
	ReasonWaitingForDependency = "WaitingForDependency"

	// Availability reasons
	ReasonSucceeded = "Succeeded"
	ReasonFailed    = "Failed"
	ReasonUnknown   = "Unknown"
)

// SetCondition sets a condition on the NotebookValidationJob status
// ADR-030: Phase 4 - Status Conditions
func SetCondition(job *mlopsv1alpha1.NotebookValidationJob, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	for i, c := range job.Status.Conditions {
		if c.Type == conditionType {
			// Only update LastTransitionTime if status actually changed
			if c.Status == status {
				newCondition.LastTransitionTime = c.LastTransitionTime
			}
			job.Status.Conditions[i] = newCondition
			return
		}
	}

	// Append new condition
	job.Status.Conditions = append(job.Status.Conditions, newCondition)
}

// SetConditionFromSmartError sets a condition from a SmartError
// ADR-030: Integration between SmartError and Status Conditions
func SetConditionFromSmartError(job *mlopsv1alpha1.NotebookValidationJob, smartErr *smarterrors.SmartError) {
	// Determine condition type based on error category
	var conditionType string
	var reason string

	switch smartErr.Category {
	case smarterrors.CategoryRBAC:
		conditionType = ConditionTypeBuildReady
		reason = ReasonRBACError
	case smarterrors.CategoryResource:
		conditionType = ConditionTypeBuildReady
		reason = ReasonTaskNotFound
	case smarterrors.CategoryTekton:
		conditionType = ConditionTypeBuildReady
		if smartErr.Code == "TEKTON_PARAM_MISMATCH" {
			reason = ReasonParameterMismatch
		} else {
			reason = ReasonBuildFailed
		}
	case smarterrors.CategoryAuthentication:
		conditionType = ConditionTypeBuildReady
		reason = ReasonGitAuthFailed
	default:
		conditionType = ConditionTypeProgressing
		reason = ReasonFailed
	}

	SetCondition(job, conditionType, metav1.ConditionFalse, reason, smartErr.UserFriendlyMessage())
}

// SetBuildCondition sets the BuildReady condition
func SetBuildCondition(job *mlopsv1alpha1.NotebookValidationJob, status metav1.ConditionStatus, reason, message string) {
	SetCondition(job, ConditionTypeBuildReady, status, reason, message)
}

// SetValidationCondition sets the ValidationReady condition
func SetValidationCondition(job *mlopsv1alpha1.NotebookValidationJob, status metav1.ConditionStatus, reason, message string) {
	SetCondition(job, ConditionTypeValidationReady, status, reason, message)
}

// SetProgressingCondition sets the Progressing condition
func SetProgressingCondition(job *mlopsv1alpha1.NotebookValidationJob, status metav1.ConditionStatus, reason, message string) {
	SetCondition(job, ConditionTypeProgressing, status, reason, message)
}

// SetAvailableCondition sets the Available condition
func SetAvailableCondition(job *mlopsv1alpha1.NotebookValidationJob, status metav1.ConditionStatus, reason, message string) {
	SetCondition(job, ConditionTypeAvailable, status, reason, message)
}

// GetCondition returns the condition with the given type
func GetCondition(job *mlopsv1alpha1.NotebookValidationJob, conditionType string) *metav1.Condition {
	for _, c := range job.Status.Conditions {
		if c.Type == conditionType {
			return &c
		}
	}
	return nil
}

// IsConditionTrue returns true if the condition with the given type is True
func IsConditionTrue(job *mlopsv1alpha1.NotebookValidationJob, conditionType string) bool {
	c := GetCondition(job, conditionType)
	return c != nil && c.Status == metav1.ConditionTrue
}

// IsConditionFalse returns true if the condition with the given type is False
func IsConditionFalse(job *mlopsv1alpha1.NotebookValidationJob, conditionType string) bool {
	c := GetCondition(job, conditionType)
	return c != nil && c.Status == metav1.ConditionFalse
}

// ConvertPodFailureToSmartError converts a PodFailureAnalysis to a SmartError
// ADR-030: Bridges existing pod analysis with new SmartError system
func ConvertPodFailureToSmartError(analysis *PodFailureAnalysis) *smarterrors.SmartError {
	if analysis == nil {
		return nil
	}

	// Determine category from failure reason
	var category smarterrors.ErrorCategory
	switch analysis.Reason {
	case FailureReasonImagePull, FailureReasonImagePullError:
		category = smarterrors.CategoryBuild
	case FailureReasonPermission:
		if analysis.IsSCCViolation {
			category = smarterrors.CategoryPlatform
		} else {
			category = smarterrors.CategoryAuthentication
		}
	case FailureReasonOOMKilled:
		category = smarterrors.CategoryResource
	case FailureReasonInitContainer:
		if analysis.FailedContainer == GitCloneContainerName {
			category = smarterrors.CategoryAuthentication
		} else {
			category = smarterrors.CategoryConfiguration
		}
	case FailureReasonCrashLoop:
		category = smarterrors.CategoryBuild
	case FailureReasonRunContainer, FailureReasonCreateContainer:
		category = smarterrors.CategoryConfiguration
	default:
		category = smarterrors.CategoryUnknown
	}

	// Create SmartError
	smartErr := smarterrors.NewSmartError(
		category,
		string(analysis.Reason),
		fmt.Sprintf("Container %s failed", analysis.FailedContainer),
		nil,
	).WithRootCause(analysis.ErrorMessage).
		WithRetryable(analysis.ShouldRetry)

	// Extract actions from SuggestedAction
	if analysis.SuggestedAction != "" {
		smartErr = smartErr.WithActions(analysis.SuggestedAction)
	}

	// Set severity based on transient nature
	if analysis.IsTransient {
		smartErr = smartErr.WithSeverity(smarterrors.SeverityWarning)
	} else {
		smartErr = smartErr.WithSeverity(smarterrors.SeverityError)
	}

	return smartErr
}

// SetConditionsForPhase sets all conditions appropriate for the current phase
// ADR-030: Comprehensive condition management based on job phase
func SetConditionsForPhase(job *mlopsv1alpha1.NotebookValidationJob) {
	switch job.Status.Phase {
	case "Initializing":
		SetProgressingCondition(job, metav1.ConditionTrue, ReasonInProgress, "Job is initializing")
		SetBuildCondition(job, metav1.ConditionUnknown, ReasonBuildPending, "Build not started")
		SetValidationCondition(job, metav1.ConditionUnknown, ReasonValidationPending, "Waiting for initialization")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonUnknown, "Job not yet available")

	case "Building":
		SetProgressingCondition(job, metav1.ConditionTrue, ReasonInProgress, "Build is in progress")
		SetBuildCondition(job, metav1.ConditionFalse, ReasonBuildInProgress, "Building container image")
		SetValidationCondition(job, metav1.ConditionFalse, ReasonWaitingForBuild, "Waiting for build to complete")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonUnknown, "Build in progress")

	case "BuildComplete":
		SetProgressingCondition(job, metav1.ConditionTrue, ReasonInProgress, "Build complete, starting validation")
		SetBuildCondition(job, metav1.ConditionTrue, ReasonBuildComplete, "Build completed successfully")
		SetValidationCondition(job, metav1.ConditionFalse, ReasonValidationPending, "Validation starting")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonUnknown, "Preparing validation")

	case "ValidationRunning", "Running":
		SetProgressingCondition(job, metav1.ConditionTrue, ReasonInProgress, "Validation is running")
		SetValidationCondition(job, metav1.ConditionFalse, ReasonValidationInProgress, "Notebook validation in progress")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonUnknown, "Validation in progress")

	case "Succeeded":
		SetProgressingCondition(job, metav1.ConditionFalse, ReasonSucceeded, "Job completed successfully")
		SetValidationCondition(job, metav1.ConditionTrue, ReasonValidationComplete, "Validation completed successfully")
		SetAvailableCondition(job, metav1.ConditionTrue, ReasonSucceeded, "Job succeeded")

	case "Failed":
		SetProgressingCondition(job, metav1.ConditionFalse, ReasonFailed, "Job failed")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonFailed, job.Status.Message)

	case "Pending":
		SetProgressingCondition(job, metav1.ConditionFalse, ReasonWaitingForDependency, "Waiting for resources")
		SetAvailableCondition(job, metav1.ConditionFalse, ReasonUnknown, "Job pending")
	}
}

// SetBuildFailedFromPipelineRun sets build failed condition from PipelineRun error
// ADR-030: Phase 1.5 - Surface PipelineRun errors in NotebookValidationJob status
func SetBuildFailedFromPipelineRun(job *mlopsv1alpha1.NotebookValidationJob, pipelineRunName string, errorMessage string) {
	// Analyze the error to get smart guidance
	smartErr := smarterrors.AnalyzeError(fmt.Errorf("%s", errorMessage))

	// Set build condition with analyzed error
	SetBuildCondition(job, metav1.ConditionFalse, ReasonBuildFailed,
		fmt.Sprintf("PipelineRun '%s' failed: %s", pipelineRunName, smartErr.UserFriendlyMessage()))

	// Also set progressing to false
	SetProgressingCondition(job, metav1.ConditionFalse, ReasonStalled,
		fmt.Sprintf("Build failed: %s", errorMessage))

	// Set available to false
	SetAvailableCondition(job, metav1.ConditionFalse, ReasonFailed, "Build phase failed")

	// Update job message with detailed guidance
	job.Status.Message = smartErr.DetailedMessage()

	// Update build status if exists
	if job.Status.BuildStatus != nil {
		job.Status.BuildStatus.Phase = "Failed"
		job.Status.BuildStatus.Message = smartErr.UserFriendlyMessage()
	}
}

// SetValidationFailedFromPod sets validation failed condition from pod failure analysis
// ADR-030: Surface pod errors in status with smart analysis
func SetValidationFailedFromPod(job *mlopsv1alpha1.NotebookValidationJob, analysis *PodFailureAnalysis) {
	if analysis == nil {
		SetValidationCondition(job, metav1.ConditionFalse, ReasonPodFailed, "Validation pod failed")
		return
	}

	// Convert to SmartError for richer guidance
	smartErr := ConvertPodFailureToSmartError(analysis)

	// Set validation condition
	reason := ReasonPodFailed
	if analysis.IsInitContainer && analysis.FailedContainer == GitCloneContainerName {
		reason = ReasonGitAuthFailed
	}

	SetValidationCondition(job, metav1.ConditionFalse, reason, smartErr.UserFriendlyMessage())
	SetProgressingCondition(job, metav1.ConditionFalse, ReasonStalled, "Validation pod failed")
	SetAvailableCondition(job, metav1.ConditionFalse, ReasonFailed, "Validation failed")

	// Update job message with detailed guidance
	job.Status.Message = analysis.SuggestedAction
}
