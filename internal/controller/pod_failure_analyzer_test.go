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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnalyzePodFailure(t *testing.T) {
	tests := []struct {
		name                string
		pod                 *corev1.Pod
		expectedReason      PodFailureReason
		expectedTransient   bool
		expectedShouldRetry bool
		expectedSCCViolation bool
		expectedImageIssue  bool
		expectedInitContainer bool
		description         string
	}{
		{
			name: "init container image pull backoff",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "git-clone",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "ImagePullBackOff",
									Message: "Back-off pulling image \"registry.example.com/git:latest\"",
								},
							},
						},
					},
				},
			},
			expectedReason:        FailureReasonImagePull,
			expectedTransient:     true,
			expectedShouldRetry:   true,
			expectedImageIssue:    true,
			expectedInitContainer: true,
			description:           "Should detect init container image pull backoff as transient",
		},
		{
			name: "init container SCC violation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "git-clone",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "RunContainerError",
									Message: "container has runAsNonRoot and image will run as root",
								},
							},
						},
					},
				},
			},
			expectedReason:        FailureReasonRunContainer,
			expectedTransient:     false,
			expectedShouldRetry:   true,
			expectedSCCViolation:  true,
			expectedInitContainer: true,
			description:           "Should detect OpenShift SCC violation in init container",
		},
		{
			name: "init container crash loop",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "git-clone",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "Back-off restarting failed container",
								},
							},
						},
					},
				},
			},
			expectedReason:        FailureReasonCrashLoop,
			expectedTransient:     false,
			expectedShouldRetry:   true,
			expectedInitContainer: true,
			description:           "Should detect init container crash loop as non-transient",
		},
		{
			name: "init container OOM killed",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "git-clone",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 137,
									Reason:   "OOMKilled",
									Message:  "Container killed due to out of memory",
								},
							},
						},
					},
				},
			},
			expectedReason:        FailureReasonOOMKilled,
			expectedTransient:     false,
			expectedShouldRetry:   true,
			expectedInitContainer: true,
			description:           "Should detect OOM killed init container",
		},
		{
			name: "init container permission denied",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "git-clone",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 126,
									Reason:   "Error",
									Message:  "Permission denied: cannot execute /usr/bin/git",
								},
							},
						},
					},
				},
			},
			expectedReason:        FailureReasonPermission,
			expectedTransient:     false,
			expectedShouldRetry:   false, // Terminal error
			expectedInitContainer: true,
			description:           "Should detect permission denied in init container",
		},
		{
			name: "main container image pull error",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "notebook-validator",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "ErrImagePull",
									Message: "Failed to pull image \"quay.io/test/notebook:latest\": manifest unknown",
								},
							},
						},
					},
				},
			},
			expectedReason:      FailureReasonImagePullError,
			expectedTransient:   true,
			expectedShouldRetry: true,
			expectedImageIssue:  true,
			description:         "Should detect main container image pull error",
		},
		{
			name: "main container permission denied exit code 126",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "notebook-validator",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 126,
									Reason:   "Error",
									Message:  "Permission denied: cannot execute /usr/local/bin/papermill",
								},
							},
						},
					},
				},
			},
			expectedReason:      FailureReasonPermission,
			expectedTransient:   false,
			expectedShouldRetry: false, // Terminal error
			description:         "Should detect permission denied errors with exit code 126",
		},
		{
			name: "main container permission denied message",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "notebook-validator",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 1,
									Reason:   "Error",
									Message:  "PermissionError: [Errno 13] Permission denied: '/workspace/.local'",
								},
							},
						},
					},
				},
			},
			expectedReason:      FailureReasonPermission,
			expectedTransient:   false,
			expectedShouldRetry: false, // Terminal error
			description:         "Should detect permission denied errors from message",
		},
		{
			name: "main container command not found",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "notebook-validator",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 127,
									Reason:   "Error",
									Message:  "bash: papermill: command not found",
								},
							},
						},
					},
				},
			},
			expectedReason:      FailureReasonPermission,
			expectedTransient:   false,
			expectedShouldRetry: false, // Terminal error
			description:         "Should detect command not found errors",
		},
		{
			name: "unschedulable pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:    corev1.PodScheduled,
							Status:  corev1.ConditionFalse,
							Reason:  "Unschedulable",
							Message: "0/3 nodes are available: insufficient memory",
						},
					},
				},
			},
			expectedReason:      FailureReasonUnknown,
			expectedTransient:   true,
			expectedShouldRetry: true,
			description:         "Should detect unschedulable pod as transient",
		},
		{
			name: "unknown failure",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			expectedReason:      FailureReasonUnknown,
			expectedTransient:   false,
			expectedShouldRetry: true,
			description:         "Should handle unknown failures gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			analysis := analyzePodFailure(ctx, tt.pod)

			if analysis.Reason != tt.expectedReason {
				t.Errorf("%s: Reason = %v, want %v", tt.description, analysis.Reason, tt.expectedReason)
			}
			if analysis.IsTransient != tt.expectedTransient {
				t.Errorf("%s: IsTransient = %v, want %v", tt.description, analysis.IsTransient, tt.expectedTransient)
			}
			if analysis.ShouldRetry != tt.expectedShouldRetry {
				t.Errorf("%s: ShouldRetry = %v, want %v", tt.description, analysis.ShouldRetry, tt.expectedShouldRetry)
			}
			if analysis.IsSCCViolation != tt.expectedSCCViolation {
				t.Errorf("%s: IsSCCViolation = %v, want %v", tt.description, analysis.IsSCCViolation, tt.expectedSCCViolation)
			}
			if analysis.IsImageIssue != tt.expectedImageIssue {
				t.Errorf("%s: IsImageIssue = %v, want %v", tt.description, analysis.IsImageIssue, tt.expectedImageIssue)
			}
			if analysis.IsInitContainer != tt.expectedInitContainer {
				t.Errorf("%s: IsInitContainer = %v, want %v", tt.description, analysis.IsInitContainer, tt.expectedInitContainer)
			}
			if analysis.SuggestedAction == "" {
				t.Errorf("%s: SuggestedAction should not be empty", tt.description)
			}
		})
	}
}

func TestGetFailureRecoveryAction(t *testing.T) {
	tests := []struct {
		name           string
		analysis       *PodFailureAnalysis
		retryCount     int
		expectedAction string
		description    string
	}{
		{
			name: "SCC violation",
			analysis: &PodFailureAnalysis{
				Reason:         FailureReasonRunContainer,
				IsSCCViolation: true,
				ShouldRetry:    true,
			},
			retryCount:     0,
			expectedAction: "use_openshift_compatible_image",
			description:    "Should use OpenShift compatible image for SCC violations",
		},
		{
			name: "init container with SCC violation",
			analysis: &PodFailureAnalysis{
				Reason:          FailureReasonRunContainer,
				IsInitContainer: true,
				IsSCCViolation:  true,
				ShouldRetry:     true,
			},
			retryCount:     0,
			expectedAction: "skip_init_container",
			description:    "Should skip init container for SCC violations",
		},
		{
			name: "OOM killed",
			analysis: &PodFailureAnalysis{
				Reason:      FailureReasonOOMKilled,
				ShouldRetry: true,
			},
			retryCount:     0,
			expectedAction: "increase_resources",
			description:    "Should increase resources for OOM killed",
		},
		{
			name: "transient error",
			analysis: &PodFailureAnalysis{
				Reason:      FailureReasonImagePull,
				IsTransient: true,
				ShouldRetry: true,
			},
			retryCount:     0,
			expectedAction: "retry_with_backoff",
			description:    "Should retry with backoff for transient errors",
		},
		{
			name: "max retries exceeded",
			analysis: &PodFailureAnalysis{
				Reason:      FailureReasonImagePull,
				IsTransient: true,
				ShouldRetry: true,
			},
			retryCount:     5,
			expectedAction: "max_retries_exceeded",
			description:    "Should not retry after max retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := getFailureRecoveryAction(tt.analysis, tt.retryCount)

			if action != tt.expectedAction {
				t.Errorf("%s: action = %v, want %v", tt.description, action, tt.expectedAction)
			}
		})
	}
}

func TestShouldSkipGitClone(t *testing.T) {
	tests := []struct {
		name           string
		containerImage string
		specImage      string
		expected       bool
		description    string
	}{
		{
			name:           "built image",
			containerImage: "image-registry.openshift-image-registry.svc:5000/mlops/test:latest",
			specImage:      "quay.io/test/notebook:latest",
			expected:       true,
			description:    "Should skip git-clone for built images",
		},
		{
			name:           "same image",
			containerImage: "quay.io/test/notebook:latest",
			specImage:      "quay.io/test/notebook:latest",
			expected:       false,
			description:    "Should not skip git-clone when using spec image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipGitClone(tt.containerImage, tt.specImage)

			if result != tt.expected {
				t.Errorf("%s: shouldSkipGitClone() = %v, want %v", tt.description, result, tt.expected)
			}
		})
	}
}


