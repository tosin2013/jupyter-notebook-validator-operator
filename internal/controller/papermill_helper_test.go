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
