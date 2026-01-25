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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	smarterrors "github.com/tosin2013/jupyter-notebook-validator-operator/pkg/errors"
)

func TestSetCondition(t *testing.T) {
	job := &mlopsv1alpha1.NotebookValidationJob{
		Status: mlopsv1alpha1.NotebookValidationJobStatus{},
	}

	// Set a new condition
	SetCondition(job, ConditionTypeBuildReady, metav1.ConditionTrue, ReasonBuildComplete, "Build succeeded")

	assert.Len(t, job.Status.Conditions, 1)
	assert.Equal(t, ConditionTypeBuildReady, job.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, job.Status.Conditions[0].Status)
	assert.Equal(t, ReasonBuildComplete, job.Status.Conditions[0].Reason)
	assert.Equal(t, "Build succeeded", job.Status.Conditions[0].Message)

	// Update existing condition
	SetCondition(job, ConditionTypeBuildReady, metav1.ConditionFalse, ReasonBuildFailed, "Build failed")

	assert.Len(t, job.Status.Conditions, 1) // Still 1 condition
	assert.Equal(t, metav1.ConditionFalse, job.Status.Conditions[0].Status)
	assert.Equal(t, ReasonBuildFailed, job.Status.Conditions[0].Reason)
}

func TestSetConditionFromSmartError(t *testing.T) {
	tests := []struct {
		name             string
		smartErr         *smarterrors.SmartError
		expectedType     string
		expectedReason   string
		expectedContains string
	}{
		{
			name: "RBAC error",
			smartErr: smarterrors.NewSmartError(
				smarterrors.CategoryRBAC,
				"RBAC_ERROR",
				"Permission denied",
				nil,
			),
			expectedType:     ConditionTypeBuildReady,
			expectedReason:   ReasonRBACError,
			expectedContains: "RBAC",
		},
		{
			name: "Tekton param mismatch",
			smartErr: smarterrors.NewSmartError(
				smarterrors.CategoryTekton,
				"TEKTON_PARAM_MISMATCH",
				"Parameter mismatch",
				nil,
			),
			expectedType:     ConditionTypeBuildReady,
			expectedReason:   ReasonParameterMismatch,
			expectedContains: "Tekton",
		},
		{
			name: "Authentication error",
			smartErr: smarterrors.NewSmartError(
				smarterrors.CategoryAuthentication,
				"AUTH_ERROR",
				"Git auth failed",
				nil,
			),
			expectedType:     ConditionTypeBuildReady,
			expectedReason:   ReasonGitAuthFailed,
			expectedContains: "Authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &mlopsv1alpha1.NotebookValidationJob{
				Status: mlopsv1alpha1.NotebookValidationJobStatus{},
			}

			SetConditionFromSmartError(job, tt.smartErr)

			assert.NotEmpty(t, job.Status.Conditions)
			condition := GetCondition(job, tt.expectedType)
			assert.NotNil(t, condition)
			assert.Equal(t, tt.expectedReason, condition.Reason)
			assert.Contains(t, condition.Message, tt.expectedContains)
		})
	}
}

func TestGetCondition(t *testing.T) {
	job := &mlopsv1alpha1.NotebookValidationJob{
		Status: mlopsv1alpha1.NotebookValidationJobStatus{
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeBuildReady,
					Status: metav1.ConditionTrue,
					Reason: ReasonBuildComplete,
				},
				{
					Type:   ConditionTypeValidationReady,
					Status: metav1.ConditionFalse,
					Reason: ReasonValidationInProgress,
				},
			},
		},
	}

	buildCondition := GetCondition(job, ConditionTypeBuildReady)
	assert.NotNil(t, buildCondition)
	assert.Equal(t, metav1.ConditionTrue, buildCondition.Status)

	validationCondition := GetCondition(job, ConditionTypeValidationReady)
	assert.NotNil(t, validationCondition)
	assert.Equal(t, metav1.ConditionFalse, validationCondition.Status)

	notExisting := GetCondition(job, "NotExisting")
	assert.Nil(t, notExisting)
}

func TestIsConditionTrue(t *testing.T) {
	job := &mlopsv1alpha1.NotebookValidationJob{
		Status: mlopsv1alpha1.NotebookValidationJobStatus{
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeBuildReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   ConditionTypeValidationReady,
					Status: metav1.ConditionFalse,
				},
			},
		},
	}

	assert.True(t, IsConditionTrue(job, ConditionTypeBuildReady))
	assert.False(t, IsConditionTrue(job, ConditionTypeValidationReady))
	assert.False(t, IsConditionTrue(job, "NotExisting"))
}

func TestIsConditionFalse(t *testing.T) {
	job := &mlopsv1alpha1.NotebookValidationJob{
		Status: mlopsv1alpha1.NotebookValidationJobStatus{
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeBuildReady,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   ConditionTypeValidationReady,
					Status: metav1.ConditionFalse,
				},
			},
		},
	}

	assert.False(t, IsConditionFalse(job, ConditionTypeBuildReady))
	assert.True(t, IsConditionFalse(job, ConditionTypeValidationReady))
	assert.False(t, IsConditionFalse(job, "NotExisting"))
}

func TestConvertPodFailureToSmartError(t *testing.T) {
	tests := []struct {
		name             string
		analysis         *PodFailureAnalysis
		expectedCategory smarterrors.ErrorCategory
		expectedRetry    bool
	}{
		{
			name:             "nil analysis",
			analysis:         nil,
			expectedCategory: "",
		},
		{
			name: "image pull error",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonImagePull,
				FailedContainer: "validation",
				ShouldRetry:     true,
				IsTransient:     true,
			},
			expectedCategory: smarterrors.CategoryBuild,
			expectedRetry:    true,
		},
		{
			name: "SCC violation",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonPermission,
				FailedContainer: "validation",
				IsSCCViolation:  true,
				ShouldRetry:     false,
			},
			expectedCategory: smarterrors.CategoryPlatform,
			expectedRetry:    false,
		},
		{
			name: "git clone failure",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonInitContainer,
				FailedContainer: GitCloneContainerName,
				IsInitContainer: true,
				ShouldRetry:     false,
			},
			expectedCategory: smarterrors.CategoryAuthentication,
			expectedRetry:    false,
		},
		{
			name: "OOM killed",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonOOMKilled,
				FailedContainer: "validation",
				ShouldRetry:     true,
			},
			expectedCategory: smarterrors.CategoryResource,
			expectedRetry:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := ConvertPodFailureToSmartError(tt.analysis)

			if tt.analysis == nil {
				assert.Nil(t, smartErr)
				return
			}

			assert.NotNil(t, smartErr)
			assert.Equal(t, tt.expectedCategory, smartErr.Category)
			assert.Equal(t, tt.expectedRetry, smartErr.Retryable)
		})
	}
}

func TestSetConditionsForPhase(t *testing.T) {
	tests := []struct {
		name               string
		phase              string
		expectedProgStatus metav1.ConditionStatus
		expectedAvailable  metav1.ConditionStatus
	}{
		{
			name:               "Initializing",
			phase:              "Initializing",
			expectedProgStatus: metav1.ConditionTrue,
			expectedAvailable:  metav1.ConditionFalse,
		},
		{
			name:               "Building",
			phase:              "Building",
			expectedProgStatus: metav1.ConditionTrue,
			expectedAvailable:  metav1.ConditionFalse,
		},
		{
			name:               "Succeeded",
			phase:              "Succeeded",
			expectedProgStatus: metav1.ConditionFalse,
			expectedAvailable:  metav1.ConditionTrue,
		},
		{
			name:               "Failed",
			phase:              "Failed",
			expectedProgStatus: metav1.ConditionFalse,
			expectedAvailable:  metav1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &mlopsv1alpha1.NotebookValidationJob{
				Status: mlopsv1alpha1.NotebookValidationJobStatus{
					Phase: tt.phase,
				},
			}

			SetConditionsForPhase(job)

			progCondition := GetCondition(job, ConditionTypeProgressing)
			assert.NotNil(t, progCondition)
			assert.Equal(t, tt.expectedProgStatus, progCondition.Status)

			availCondition := GetCondition(job, ConditionTypeAvailable)
			assert.NotNil(t, availCondition)
			assert.Equal(t, tt.expectedAvailable, availCondition.Status)
		})
	}
}

func TestSetBuildFailedFromPipelineRun(t *testing.T) {
	job := &mlopsv1alpha1.NotebookValidationJob{
		Status: mlopsv1alpha1.NotebookValidationJobStatus{
			BuildStatus: &mlopsv1alpha1.BuildStatus{
				Phase: "Running",
			},
		},
	}

	SetBuildFailedFromPipelineRun(job, "my-pipelinerun", "missing values for these params: [URL]")

	// Check conditions
	buildCondition := GetCondition(job, ConditionTypeBuildReady)
	assert.NotNil(t, buildCondition)
	assert.Equal(t, metav1.ConditionFalse, buildCondition.Status)
	assert.Equal(t, ReasonBuildFailed, buildCondition.Reason)
	assert.Contains(t, buildCondition.Message, "my-pipelinerun")

	// Check job message contains guidance
	assert.NotEmpty(t, job.Status.Message)
	assert.Contains(t, job.Status.Message, "Error")

	// Check build status updated
	assert.Equal(t, "Failed", job.Status.BuildStatus.Phase)
}

func TestSetValidationFailedFromPod(t *testing.T) {
	tests := []struct {
		name           string
		analysis       *PodFailureAnalysis
		expectedReason string
	}{
		{
			name:           "nil analysis",
			analysis:       nil,
			expectedReason: ReasonPodFailed,
		},
		{
			name: "git clone failure",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonInitContainer,
				FailedContainer: GitCloneContainerName,
				IsInitContainer: true,
				SuggestedAction: "Use Tekton build",
			},
			expectedReason: ReasonGitAuthFailed,
		},
		{
			name: "generic pod failure",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonCrashLoop,
				FailedContainer: "validation",
				SuggestedAction: "Check logs",
			},
			expectedReason: ReasonPodFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &mlopsv1alpha1.NotebookValidationJob{
				Status: mlopsv1alpha1.NotebookValidationJobStatus{},
			}

			SetValidationFailedFromPod(job, tt.analysis)

			validationCondition := GetCondition(job, ConditionTypeValidationReady)
			assert.NotNil(t, validationCondition)
			assert.Equal(t, metav1.ConditionFalse, validationCondition.Status)
			assert.Equal(t, tt.expectedReason, validationCondition.Reason)
		})
	}
}
