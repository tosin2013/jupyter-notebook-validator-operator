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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestBuildPapermillValidationContainer(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &NotebookValidationJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	tests := []struct {
		name           string
		job            *mlopsv1alpha1.NotebookValidationJob
		containerImage string
		expectedPath   string
		description    string
	}{
		{
			name: "built image path",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "quay.io/test/notebook:latest",
					},
				},
			},
			containerImage: "image-registry.openshift-image-registry.svc:5000/mlops/test-job:latest",
			expectedPath:   "/opt/app-root/src/notebooks/test.ipynb",
			description:    "Should use S2I path for built images",
		},
		{
			name: "git-cloned path",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "quay.io/test/notebook:latest",
					},
				},
			},
			containerImage: "quay.io/test/notebook:latest",
			expectedPath:   "/workspace/repo/notebooks/test.ipynb",
			description:    "Should use git-clone path for pre-built images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			container := reconciler.buildPapermillValidationContainer(ctx, tt.job, tt.containerImage)

			if container.Name != "validator" {
				t.Errorf("%s: container name = %v, want validator", tt.description, container.Name)
			}

			// Check that the script contains the expected path
			if len(container.Command) < 3 {
				t.Errorf("%s: container command is too short", tt.description)
				return
			}

			script := container.Command[2]
			if !strings.Contains(script, tt.expectedPath) {
				t.Errorf("%s: script does not contain expected path %s", tt.description, tt.expectedPath)
			}

			// Verify script contains key components
			if !strings.Contains(script, "Jupyter Notebook Validator - Papermill") {
				t.Error("Script should contain Papermill header")
			}
			if !strings.Contains(script, "python -m papermill") {
				t.Error("Script should contain papermill execution command")
			}
			if !strings.Contains(script, "/workspace/results.json") {
				t.Error("Script should reference results.json")
			}
		})
	}
}

func TestBuildPapermillValidationContainerResources(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &NotebookValidationJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	job := &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Path: "notebooks/test.ipynb",
			},
			PodConfig: mlopsv1alpha1.PodConfigSpec{
				ContainerImage: "quay.io/test/notebook:latest",
				Resources: &mlopsv1alpha1.ResourceRequirements{
					Requests: map[string]string{
						"cpu":    "500m",
						"memory": "1Gi",
					},
					Limits: map[string]string{
						"cpu":    "2",
						"memory": "4Gi",
					},
				},
			},
		},
	}

	ctx := context.Background()
	container := reconciler.buildPapermillValidationContainer(ctx, job, "")

	// Verify resource requests
	cpuRequest := container.Resources.Requests[corev1.ResourceCPU]
	if cpuRequest.Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("CPU request = %v, want 500m", cpuRequest)
	}
	memRequest := container.Resources.Requests[corev1.ResourceMemory]
	if memRequest.Cmp(resource.MustParse("1Gi")) != 0 {
		t.Errorf("Memory request = %v, want 1Gi", memRequest)
	}

	// Verify resource limits
	cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
	if cpuLimit.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("CPU limit = %v, want 2", cpuLimit)
	}
	memLimit := container.Resources.Limits[corev1.ResourceMemory]
	if memLimit.Cmp(resource.MustParse("4Gi")) != 0 {
		t.Errorf("Memory limit = %v, want 4Gi", memLimit)
	}
}

// TestConvertVolumes tests the volume conversion function (ADR-045)
func TestConvertVolumes(t *testing.T) {
	tests := []struct {
		name     string
		input    []mlopsv1alpha1.Volume
		expected int
		checks   func(t *testing.T, volumes []corev1.Volume)
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: 0,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes != nil {
					t.Error("Expected nil output for nil input")
				}
			},
		},
		{
			name:     "empty input",
			input:    []mlopsv1alpha1.Volume{},
			expected: 0,
			checks:   nil,
		},
		{
			name: "PVC volume",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "model-output",
					PersistentVolumeClaim: &mlopsv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName: "trained-models-pvc",
						ReadOnly:  false,
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes[0].Name != "model-output" {
					t.Errorf("Volume name = %s, want model-output", volumes[0].Name)
				}
				if volumes[0].PersistentVolumeClaim == nil {
					t.Error("PVC volume source should not be nil")
					return
				}
				if volumes[0].PersistentVolumeClaim.ClaimName != "trained-models-pvc" {
					t.Errorf("ClaimName = %s, want trained-models-pvc", volumes[0].PersistentVolumeClaim.ClaimName)
				}
				if volumes[0].PersistentVolumeClaim.ReadOnly != false {
					t.Error("ReadOnly should be false")
				}
			},
		},
		{
			name: "PVC volume read-only",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "shared-data",
					PersistentVolumeClaim: &mlopsv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName: "shared-datasets-pvc",
						ReadOnly:  true,
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if !volumes[0].PersistentVolumeClaim.ReadOnly {
					t.Error("ReadOnly should be true")
				}
			},
		},
		{
			name: "ConfigMap volume",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "config",
					ConfigMap: &mlopsv1alpha1.ConfigMapVolumeSource{
						Name: "notebook-config",
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes[0].ConfigMap == nil {
					t.Error("ConfigMap volume source should not be nil")
					return
				}
				if volumes[0].ConfigMap.Name != "notebook-config" {
					t.Errorf("ConfigMap name = %s, want notebook-config", volumes[0].ConfigMap.Name)
				}
			},
		},
		{
			name: "ConfigMap volume with items",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "config-items",
					ConfigMap: &mlopsv1alpha1.ConfigMapVolumeSource{
						Name: "my-config",
						Items: []mlopsv1alpha1.KeyToPath{
							{Key: "config.yaml", Path: "app-config.yaml"},
						},
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if len(volumes[0].ConfigMap.Items) != 1 {
					t.Errorf("ConfigMap items length = %d, want 1", len(volumes[0].ConfigMap.Items))
					return
				}
				if volumes[0].ConfigMap.Items[0].Key != "config.yaml" {
					t.Errorf("Item key = %s, want config.yaml", volumes[0].ConfigMap.Items[0].Key)
				}
				if volumes[0].ConfigMap.Items[0].Path != "app-config.yaml" {
					t.Errorf("Item path = %s, want app-config.yaml", volumes[0].ConfigMap.Items[0].Path)
				}
			},
		},
		{
			name: "Secret volume",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "creds",
					Secret: &mlopsv1alpha1.SecretVolumeSource{
						SecretName: "api-credentials",
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes[0].Secret == nil {
					t.Error("Secret volume source should not be nil")
					return
				}
				if volumes[0].Secret.SecretName != "api-credentials" {
					t.Errorf("SecretName = %s, want api-credentials", volumes[0].Secret.SecretName)
				}
			},
		},
		{
			name: "EmptyDir volume",
			input: []mlopsv1alpha1.Volume{
				{
					Name:     "scratch",
					EmptyDir: &mlopsv1alpha1.EmptyDirVolumeSource{},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes[0].EmptyDir == nil {
					t.Error("EmptyDir volume source should not be nil")
				}
			},
		},
		{
			name: "EmptyDir volume with memory medium",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "ramdisk",
					EmptyDir: &mlopsv1alpha1.EmptyDirVolumeSource{
						Medium:    "Memory",
						SizeLimit: "1Gi",
					},
				},
			},
			expected: 1,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				if volumes[0].EmptyDir.Medium != corev1.StorageMediumMemory {
					t.Errorf("Medium = %s, want Memory", volumes[0].EmptyDir.Medium)
				}
				if volumes[0].EmptyDir.SizeLimit == nil {
					t.Error("SizeLimit should not be nil")
					return
				}
				if volumes[0].EmptyDir.SizeLimit.Cmp(resource.MustParse("1Gi")) != 0 {
					t.Errorf("SizeLimit = %v, want 1Gi", volumes[0].EmptyDir.SizeLimit)
				}
			},
		},
		{
			name: "multiple volumes",
			input: []mlopsv1alpha1.Volume{
				{
					Name: "data",
					PersistentVolumeClaim: &mlopsv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName: "data-pvc",
					},
				},
				{
					Name: "config",
					ConfigMap: &mlopsv1alpha1.ConfigMapVolumeSource{
						Name: "app-config",
					},
				},
				{
					Name:     "temp",
					EmptyDir: &mlopsv1alpha1.EmptyDirVolumeSource{},
				},
			},
			expected: 3,
			checks: func(t *testing.T, volumes []corev1.Volume) {
				// Check first volume is PVC
				if volumes[0].PersistentVolumeClaim == nil {
					t.Error("First volume should be PVC")
				}
				// Check second volume is ConfigMap
				if volumes[1].ConfigMap == nil {
					t.Error("Second volume should be ConfigMap")
				}
				// Check third volume is EmptyDir
				if volumes[2].EmptyDir == nil {
					t.Error("Third volume should be EmptyDir")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVolumes(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Error("Expected nil result for nil input")
				}
				if tt.checks != nil {
					tt.checks(t, result)
				}
				return
			}

			if len(result) != tt.expected {
				t.Errorf("convertVolumes() returned %d volumes, want %d", len(result), tt.expected)
			}

			if tt.checks != nil {
				tt.checks(t, result)
			}
		})
	}
}

// TestConvertVolumeMounts tests the volume mount conversion function (ADR-045)
func TestConvertVolumeMounts(t *testing.T) {
	tests := []struct {
		name     string
		input    []mlopsv1alpha1.VolumeMount
		expected int
		checks   func(t *testing.T, mounts []corev1.VolumeMount)
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: 0,
			checks: func(t *testing.T, mounts []corev1.VolumeMount) {
				if mounts != nil {
					t.Error("Expected nil output for nil input")
				}
			},
		},
		{
			name:     "empty input",
			input:    []mlopsv1alpha1.VolumeMount{},
			expected: 0,
			checks:   nil,
		},
		{
			name: "basic mount",
			input: []mlopsv1alpha1.VolumeMount{
				{
					Name:      "model-output",
					MountPath: "/models",
				},
			},
			expected: 1,
			checks: func(t *testing.T, mounts []corev1.VolumeMount) {
				if mounts[0].Name != "model-output" {
					t.Errorf("Name = %s, want model-output", mounts[0].Name)
				}
				if mounts[0].MountPath != "/models" {
					t.Errorf("MountPath = %s, want /models", mounts[0].MountPath)
				}
				if mounts[0].ReadOnly != false {
					t.Error("ReadOnly should default to false")
				}
			},
		},
		{
			name: "read-only mount",
			input: []mlopsv1alpha1.VolumeMount{
				{
					Name:      "shared-data",
					MountPath: "/data",
					ReadOnly:  true,
				},
			},
			expected: 1,
			checks: func(t *testing.T, mounts []corev1.VolumeMount) {
				if !mounts[0].ReadOnly {
					t.Error("ReadOnly should be true")
				}
			},
		},
		{
			name: "mount with subpath",
			input: []mlopsv1alpha1.VolumeMount{
				{
					Name:      "config",
					MountPath: "/app/config.yaml",
					SubPath:   "config.yaml",
				},
			},
			expected: 1,
			checks: func(t *testing.T, mounts []corev1.VolumeMount) {
				if mounts[0].SubPath != "config.yaml" {
					t.Errorf("SubPath = %s, want config.yaml", mounts[0].SubPath)
				}
			},
		},
		{
			name: "multiple mounts",
			input: []mlopsv1alpha1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/data",
					ReadOnly:  true,
				},
				{
					Name:      "models",
					MountPath: "/models",
					ReadOnly:  false,
				},
				{
					Name:      "config",
					MountPath: "/config",
				},
				{
					Name:      "scratch",
					MountPath: "/scratch",
				},
			},
			expected: 4,
			checks: func(t *testing.T, mounts []corev1.VolumeMount) {
				// Check all mounts have correct paths
				paths := map[string]string{
					"data":    "/data",
					"models":  "/models",
					"config":  "/config",
					"scratch": "/scratch",
				}
				for _, mount := range mounts {
					expectedPath, ok := paths[mount.Name]
					if !ok {
						t.Errorf("Unexpected mount name: %s", mount.Name)
						continue
					}
					if mount.MountPath != expectedPath {
						t.Errorf("Mount %s has path %s, want %s", mount.Name, mount.MountPath, expectedPath)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertVolumeMounts(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Error("Expected nil result for nil input")
				}
				if tt.checks != nil {
					tt.checks(t, result)
				}
				return
			}

			if len(result) != tt.expected {
				t.Errorf("convertVolumeMounts() returned %d mounts, want %d", len(result), tt.expected)
			}

			if tt.checks != nil {
				tt.checks(t, result)
			}
		})
	}
}

// TestConvertTolerations tests the toleration conversion function (GitHub Issue #13)
func TestConvertTolerations(t *testing.T) {
	int64Ptr := func(i int64) *int64 { return &i }

	tests := []struct {
		name     string
		input    []mlopsv1alpha1.Toleration
		expected int
		checks   func(t *testing.T, tolerations []corev1.Toleration)
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: 0,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				if tolerations != nil {
					t.Error("Expected nil output for nil input")
				}
			},
		},
		{
			name:     "empty input",
			input:    []mlopsv1alpha1.Toleration{},
			expected: 0,
			checks:   nil,
		},
		{
			name: "GPU node toleration with Exists operator",
			input: []mlopsv1alpha1.Toleration{
				{
					Key:      "nvidia.com/gpu",
					Operator: "Exists",
					Effect:   "NoSchedule",
				},
			},
			expected: 1,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				if tolerations[0].Key != "nvidia.com/gpu" {
					t.Errorf("Key = %s, want nvidia.com/gpu", tolerations[0].Key)
				}
				if tolerations[0].Operator != corev1.TolerationOpExists {
					t.Errorf("Operator = %v, want Exists", tolerations[0].Operator)
				}
				if tolerations[0].Effect != corev1.TaintEffectNoSchedule {
					t.Errorf("Effect = %v, want NoSchedule", tolerations[0].Effect)
				}
			},
		},
		{
			name: "toleration with Equal operator",
			input: []mlopsv1alpha1.Toleration{
				{
					Key:      "gpu",
					Operator: "Equal",
					Value:    "true",
					Effect:   "NoSchedule",
				},
			},
			expected: 1,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				if tolerations[0].Key != "gpu" {
					t.Errorf("Key = %s, want gpu", tolerations[0].Key)
				}
				if tolerations[0].Operator != corev1.TolerationOpEqual {
					t.Errorf("Operator = %v, want Equal", tolerations[0].Operator)
				}
				if tolerations[0].Value != "true" {
					t.Errorf("Value = %s, want true", tolerations[0].Value)
				}
			},
		},
		{
			name: "toleration with NoExecute and TolerationSeconds",
			input: []mlopsv1alpha1.Toleration{
				{
					Key:               "node.kubernetes.io/not-ready",
					Operator:          "Exists",
					Effect:            "NoExecute",
					TolerationSeconds: int64Ptr(300),
				},
			},
			expected: 1,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				if tolerations[0].Effect != corev1.TaintEffectNoExecute {
					t.Errorf("Effect = %v, want NoExecute", tolerations[0].Effect)
				}
				if tolerations[0].TolerationSeconds == nil {
					t.Error("TolerationSeconds should not be nil")
					return
				}
				if *tolerations[0].TolerationSeconds != 300 {
					t.Errorf("TolerationSeconds = %d, want 300", *tolerations[0].TolerationSeconds)
				}
			},
		},
		{
			name: "multiple tolerations",
			input: []mlopsv1alpha1.Toleration{
				{
					Key:      "nvidia.com/gpu",
					Operator: "Exists",
					Effect:   "NoSchedule",
				},
				{
					Key:      "kubernetes.io/spot",
					Operator: "Exists",
					Effect:   "NoSchedule",
				},
				{
					Key:      "team",
					Operator: "Equal",
					Value:    "ml",
					Effect:   "NoSchedule",
				},
			},
			expected: 3,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				keys := []string{"nvidia.com/gpu", "kubernetes.io/spot", "team"}
				for i, tol := range tolerations {
					if tol.Key != keys[i] {
						t.Errorf("Toleration %d key = %s, want %s", i, tol.Key, keys[i])
					}
				}
			},
		},
		{
			name: "toleration with PreferNoSchedule effect",
			input: []mlopsv1alpha1.Toleration{
				{
					Key:      "high-memory",
					Operator: "Exists",
					Effect:   "PreferNoSchedule",
				},
			},
			expected: 1,
			checks: func(t *testing.T, tolerations []corev1.Toleration) {
				if tolerations[0].Effect != corev1.TaintEffectPreferNoSchedule {
					t.Errorf("Effect = %v, want PreferNoSchedule", tolerations[0].Effect)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTolerations(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Error("Expected nil result for nil input")
				}
				if tt.checks != nil {
					tt.checks(t, result)
				}
				return
			}

			if len(result) != tt.expected {
				t.Errorf("convertTolerations() returned %d tolerations, want %d", len(result), tt.expected)
			}

			if tt.checks != nil {
				tt.checks(t, result)
			}
		})
	}
}

// TestConvertAffinity tests the affinity conversion function (GitHub Issue #13)
// Tests are split into sub-tests to reduce cyclomatic complexity
func TestConvertAffinity(t *testing.T) {
	t.Run("nil input", testConvertAffinityNil)
	t.Run("node affinity with required terms", testConvertAffinityNodeRequired)
	t.Run("node affinity with preferred terms", testConvertAffinityNodePreferred)
	t.Run("pod anti-affinity", testConvertAffinityPodAnti)
	t.Run("pod affinity with required terms", testConvertAffinityPodRequired)
	t.Run("combined node and pod affinity", testConvertAffinityCombined)
}

func testConvertAffinityNil(t *testing.T) {
	result := convertAffinity(nil)
	if result != nil {
		t.Error("Expected nil output for nil input")
	}
}

func testConvertAffinityNodeRequired(t *testing.T) {
	input := &mlopsv1alpha1.Affinity{
		NodeAffinity: &mlopsv1alpha1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &mlopsv1alpha1.NodeSelector{
				NodeSelectorTerms: []mlopsv1alpha1.NodeSelectorTerm{
					{
						MatchExpressions: []mlopsv1alpha1.NodeSelectorRequirement{
							{Key: "nvidia.com/gpu.present", Operator: "In", Values: []string{"true"}},
						},
					},
				},
			},
		},
	}
	result := convertAffinity(input)
	assertAffinityNotNil(t, result)
	assertNodeAffinityNotNil(t, result)

	required := result.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		t.Fatal("RequiredDuringSchedulingIgnoredDuringExecution should not be nil")
	}
	if len(required.NodeSelectorTerms) != 1 {
		t.Fatalf("NodeSelectorTerms length = %d, want 1", len(required.NodeSelectorTerms))
	}
	assertNodeSelectorExpression(t, required.NodeSelectorTerms[0].MatchExpressions,
		"nvidia.com/gpu.present", corev1.NodeSelectorOpIn, []string{"true"})
}

func testConvertAffinityNodePreferred(t *testing.T) {
	input := &mlopsv1alpha1.Affinity{
		NodeAffinity: &mlopsv1alpha1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []mlopsv1alpha1.PreferredSchedulingTerm{
				{
					Weight: 100,
					Preference: mlopsv1alpha1.NodeSelectorTerm{
						MatchExpressions: []mlopsv1alpha1.NodeSelectorRequirement{
							{Key: "nvidia.com/gpu.memory", Operator: "Gt", Values: []string{"16000"}},
						},
					},
				},
			},
		},
	}
	result := convertAffinity(input)
	assertAffinityNotNil(t, result)
	assertNodeAffinityNotNil(t, result)

	preferred := result.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	if len(preferred) != 1 {
		t.Fatalf("PreferredDuringSchedulingIgnoredDuringExecution length = %d, want 1", len(preferred))
	}
	if preferred[0].Weight != 100 {
		t.Errorf("Weight = %d, want 100", preferred[0].Weight)
	}
	if preferred[0].Preference.MatchExpressions[0].Operator != corev1.NodeSelectorOpGt {
		t.Errorf("Operator = %v, want Gt", preferred[0].Preference.MatchExpressions[0].Operator)
	}
}

func testConvertAffinityPodAnti(t *testing.T) {
	input := &mlopsv1alpha1.Affinity{
		PodAntiAffinity: &mlopsv1alpha1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []mlopsv1alpha1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: mlopsv1alpha1.PodAffinityTerm{
						LabelSelector: &mlopsv1alpha1.LabelSelector{
							MatchLabels: map[string]string{"app": "jupyter-notebook-validator"},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
	result := convertAffinity(input)
	assertAffinityNotNil(t, result)
	if result.PodAntiAffinity == nil {
		t.Fatal("PodAntiAffinity should not be nil")
	}

	preferred := result.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	if len(preferred) != 1 {
		t.Fatalf("PreferredDuringSchedulingIgnoredDuringExecution length = %d, want 1", len(preferred))
	}
	assertWeightedPodAffinityTerm(t, preferred[0], 100, "kubernetes.io/hostname", "jupyter-notebook-validator")
}

func testConvertAffinityPodRequired(t *testing.T) {
	input := &mlopsv1alpha1.Affinity{
		PodAffinity: &mlopsv1alpha1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []mlopsv1alpha1.PodAffinityTerm{
				{
					LabelSelector: &mlopsv1alpha1.LabelSelector{
						MatchExpressions: []mlopsv1alpha1.LabelSelectorRequirement{
							{Key: "team", Operator: "In", Values: []string{"ml", "data-science"}},
						},
					},
					TopologyKey: "topology.kubernetes.io/zone",
					Namespaces:  []string{"team-ml", "team-ds"},
				},
			},
		},
	}
	result := convertAffinity(input)
	assertAffinityNotNil(t, result)
	if result.PodAffinity == nil {
		t.Fatal("PodAffinity should not be nil")
	}

	required := result.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(required) != 1 {
		t.Fatalf("RequiredDuringSchedulingIgnoredDuringExecution length = %d, want 1", len(required))
	}
	if required[0].TopologyKey != "topology.kubernetes.io/zone" {
		t.Errorf("TopologyKey = %s, want topology.kubernetes.io/zone", required[0].TopologyKey)
	}
	if len(required[0].Namespaces) != 2 {
		t.Errorf("Namespaces length = %d, want 2", len(required[0].Namespaces))
	}
}

func testConvertAffinityCombined(t *testing.T) {
	input := &mlopsv1alpha1.Affinity{
		NodeAffinity: &mlopsv1alpha1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &mlopsv1alpha1.NodeSelector{
				NodeSelectorTerms: []mlopsv1alpha1.NodeSelectorTerm{
					{
						MatchExpressions: []mlopsv1alpha1.NodeSelectorRequirement{
							{Key: "node-type", Operator: "In", Values: []string{"gpu"}},
						},
					},
				},
			},
		},
		PodAntiAffinity: &mlopsv1alpha1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []mlopsv1alpha1.WeightedPodAffinityTerm{
				{
					Weight: 50,
					PodAffinityTerm: mlopsv1alpha1.PodAffinityTerm{
						LabelSelector: &mlopsv1alpha1.LabelSelector{
							MatchLabels: map[string]string{"app": "gpu-workload"},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
	result := convertAffinity(input)
	assertAffinityNotNil(t, result)
	assertNodeAffinityNotNil(t, result)
	if result.PodAntiAffinity == nil {
		t.Error("PodAntiAffinity should not be nil")
	}
	if result.PodAffinity != nil {
		t.Error("PodAffinity should be nil (not specified)")
	}
}

// Helper functions for affinity test assertions
func assertAffinityNotNil(t *testing.T, affinity *corev1.Affinity) {
	t.Helper()
	if affinity == nil {
		t.Fatal("Affinity should not be nil")
	}
}

func assertNodeAffinityNotNil(t *testing.T, affinity *corev1.Affinity) {
	t.Helper()
	if affinity.NodeAffinity == nil {
		t.Fatal("NodeAffinity should not be nil")
	}
}

func assertNodeSelectorExpression(t *testing.T, exprs []corev1.NodeSelectorRequirement, key string, op corev1.NodeSelectorOperator, values []string) {
	t.Helper()
	if len(exprs) != 1 {
		t.Fatalf("MatchExpressions length = %d, want 1", len(exprs))
	}
	expr := exprs[0]
	if expr.Key != key {
		t.Errorf("Key = %s, want %s", expr.Key, key)
	}
	if expr.Operator != op {
		t.Errorf("Operator = %v, want %v", expr.Operator, op)
	}
	if len(expr.Values) != len(values) {
		t.Errorf("Values length = %d, want %d", len(expr.Values), len(values))
	}
}

func assertWeightedPodAffinityTerm(t *testing.T, term corev1.WeightedPodAffinityTerm, weight int32, topologyKey, appLabel string) {
	t.Helper()
	if term.Weight != weight {
		t.Errorf("Weight = %d, want %d", term.Weight, weight)
	}
	if term.PodAffinityTerm.TopologyKey != topologyKey {
		t.Errorf("TopologyKey = %s, want %s", term.PodAffinityTerm.TopologyKey, topologyKey)
	}
	if term.PodAffinityTerm.LabelSelector == nil {
		t.Fatal("LabelSelector should not be nil")
	}
	if term.PodAffinityTerm.LabelSelector.MatchLabels["app"] != appLabel {
		t.Errorf("MatchLabels app = %s, want %s", term.PodAffinityTerm.LabelSelector.MatchLabels["app"], appLabel)
	}
}
