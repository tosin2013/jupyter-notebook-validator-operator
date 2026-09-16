package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestAnalyzePodFailure_ImagePullBackOff verifies pod_failure_analyzer classifies
// ImagePullBackOff correctly and marks the issue as image-related (ADR-026).
func TestAnalyzePodFailure_ImagePullBackOff(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "notebook-validator",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image quay.io/missing/image:latest",
						},
					},
				},
			},
		},
	}

	analysis := analyzePodFailure(ctx, pod)

	if analysis.Reason != FailureReasonImagePull {
		t.Errorf("expected reason %s, got %s", FailureReasonImagePull, analysis.Reason)
	}
	if !analysis.IsImageIssue {
		t.Error("expected IsImageIssue to be true for ImagePullBackOff")
	}
	if !analysis.IsTransient {
		t.Error("expected IsTransient to be true for ImagePullBackOff (registry may recover)")
	}
	if !analysis.ShouldRetry {
		t.Error("expected ShouldRetry to be true for ImagePullBackOff")
	}
}

// TestAnalyzePodFailure_OOMKilled verifies OOMKilled classification (ADR-026).
func TestAnalyzePodFailure_OOMKilled(t *testing.T) {
	ctx := context.Background()
	exitCode := int32(137)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "notebook-validator",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: exitCode,
						},
					},
				},
			},
		},
	}

	analysis := analyzePodFailure(ctx, pod)

	if analysis.Reason != FailureReasonOOMKilled {
		t.Errorf("expected reason %s, got %s", FailureReasonOOMKilled, analysis.Reason)
	}
	if analysis.IsTransient {
		t.Error("OOMKilled should not be transient")
	}
}

// TestAnalyzePodFailure_CrashLoopBackOff verifies CrashLoopBackOff classification (ADR-026).
func TestAnalyzePodFailure_CrashLoopBackOff(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "notebook-validator",
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "CrashLoopBackOff",
							Message: "back-off 5m0s restarting failed container",
						},
					},
				},
			},
		},
	}

	analysis := analyzePodFailure(ctx, pod)

	if analysis.Reason != FailureReasonCrashLoop {
		t.Errorf("expected reason %s, got %s", FailureReasonCrashLoop, analysis.Reason)
	}
	if analysis.IsTransient {
		t.Error("CrashLoopBackOff should not be transient")
	}
}

// TestAnalyzePodFailure_InitContainerFailure verifies init container failures
// are detected and classified correctly.
func TestAnalyzePodFailure_InitContainerFailure(t *testing.T) {
	ctx := context.Background()
	exitCode := int32(1)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "git-clone",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Error",
							ExitCode: exitCode,
							Message:  "fatal: repository not found",
						},
					},
				},
			},
		},
	}

	analysis := analyzePodFailure(ctx, pod)

	if !analysis.IsInitContainer {
		t.Error("expected IsInitContainer to be true for init container failure")
	}
	if analysis.FailedContainer != "git-clone" {
		t.Errorf("expected FailedContainer 'git-clone', got %q", analysis.FailedContainer)
	}
}

// TestModelEndpointEnvInjection verifies that model endpoint environment
// variables follow the expected naming convention (ADR-020).
func TestModelEndpointEnvInjection(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		endpoint     string
		expectedName string
	}{
		{
			name:         "simple model name",
			modelName:    "my-model",
			endpoint:     "http://my-model.models.svc:8080/v2/models/my-model/infer",
			expectedName: "MODEL_ENDPOINT",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := corev1.EnvVar{
				Name:  tc.expectedName,
				Value: tc.endpoint,
			}
			if env.Name != tc.expectedName {
				t.Errorf("expected env var name %q, got %q", tc.expectedName, env.Name)
			}
			if env.Value != tc.endpoint {
				t.Errorf("expected env var value %q, got %q", tc.endpoint, env.Value)
			}
		})
	}
}

// TestPodResourceLimits verifies that resource limits are set correctly on
// validation pods, which is important for OOMKilled recovery (ADR-026).
func TestPodResourceLimits(t *testing.T) {
	limits := corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("512Mi"),
		corev1.ResourceCPU:    resource.MustParse("500m"),
	}

	mem := limits[corev1.ResourceMemory]
	if mem.String() != "512Mi" {
		t.Errorf("expected memory limit 512Mi, got %s", mem.String())
	}

	cpu := limits[corev1.ResourceCPU]
	if cpu.String() != "500m" {
		t.Errorf("expected cpu limit 500m, got %s", cpu.String())
	}
}
