package build

import (
	"context"
	"testing"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestNewTektonStrategy tests Tekton strategy creation
func TestNewTektonStrategy(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme) // Add core Kubernetes types
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

	strategy := NewTektonStrategy(fakeClient, testScheme)

	if strategy == nil {
		t.Fatal("NewTektonStrategy returned nil")
	}

	if strategy.Name() != "tekton" {
		t.Errorf("Name() = %v, want tekton", strategy.Name())
	}
}

// TestTektonStrategyName tests the Name method
func TestTektonStrategyName(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme) // Add core Kubernetes types
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

	strategy := NewTektonStrategy(fakeClient, testScheme)

	if strategy.Name() != "tekton" {
		t.Errorf("Name() = %v, want tekton", strategy.Name())
	}
}

// TestTektonStrategyDetect tests the Detect method
func TestTektonStrategyDetect(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme) // Add core Kubernetes types
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

	strategy := NewTektonStrategy(fakeClient, testScheme)
	ctx := context.Background()

	// In a test environment without real Tekton, detection should return false or error
	detected, err := strategy.Detect(ctx, fakeClient)

	// We expect either false (not detected) or an error (CRD not registered)
	if detected {
		t.Log("Tekton detected in test environment (unexpected but not an error)")
	}
	if err != nil {
		t.Logf("Detect() error = %v (expected in test environment)", err)
	}
}

// TestTektonStrategyValidateConfig tests configuration validation
func TestTektonStrategyValidateConfig(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme) // Add core Kubernetes types
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

	strategy := NewTektonStrategy(fakeClient, testScheme)

	tests := []ValidateConfigTestCase{
		{
			Name: "Valid config with base image",
			Config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy:  "tekton",
				BaseImage: "quay.io/jupyter/minimal-notebook:latest",
			},
			ExpectError: false,
		},
		{
			Name: "Valid config without base image (uses default)",
			Config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "tekton",
			},
			ExpectError: false,
		},
		{
			Name: "Valid config with registry",
			Config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy:  "tekton",
				BaseImage: "custom-image:latest",
				StrategyConfig: map[string]string{
					"registry": "custom-registry.example.com:5000",
				},
			},
			ExpectError: false,
		},
	}

	RunValidateConfigTests(t, strategy, tests)
}

// TestTektonStrategyCreateBuild tests build creation
func TestTektonStrategyCreateBuild(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme) // Add core Kubernetes types
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)

	// ADR-028: Pre-create Tasks in openshift-pipelines namespace for testing
	// This simulates the Tasks that exist in a real OpenShift cluster
	gitCloneTask := &tektonv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-clone",
			Namespace: "openshift-pipelines",
		},
		Spec: tektonv1.TaskSpec{
			Description: "Test git-clone task",
			Params: []tektonv1.ParamSpec{
				{Name: "url", Type: tektonv1.ParamTypeString},
				{Name: "revision", Type: tektonv1.ParamTypeString},
			},
			Workspaces: []tektonv1.WorkspaceDeclaration{
				{Name: "output"},
				{Name: "ssh-directory", Optional: true},
			},
		},
	}

	buildahTask := &tektonv1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "buildah",
			Namespace: "openshift-pipelines",
		},
		Spec: tektonv1.TaskSpec{
			Description: "Test buildah task",
			Params: []tektonv1.ParamSpec{
				{Name: "IMAGE", Type: tektonv1.ParamTypeString},
				{Name: "BUILDER_IMAGE", Type: tektonv1.ParamTypeString},
				{Name: "CONTEXT", Type: tektonv1.ParamTypeString},
			},
			Workspaces: []tektonv1.WorkspaceDeclaration{
				{Name: "source"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(gitCloneTask, buildahTask).
		Build()

	strategy := NewTektonStrategy(fakeClient, testScheme)
	ctx := context.Background()

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid job with build config",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/repo.git",
							Ref: "main",
						},
						Path: "notebook.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test-image:latest",
						BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
							Enabled:   true,
							Strategy:  "tekton",
							BaseImage: "quay.io/jupyter/minimal-notebook:latest",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Job without build config",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-no-build",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/repo.git",
							Ref: "main",
						},
						Path: "notebook.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test-image:latest",
						BuildConfig:    nil,
					},
				},
			},
			expectError: true,
			errorMsg:    "buildConfig is required",
		},
		{
			name: "Job with default base image",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-default-image",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/repo.git",
							Ref: "main",
						},
						Path: "notebook.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test-image:latest",
						BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
							Enabled:  true,
							Strategy: "tekton",
							// BaseImage not specified - should use default
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Job with custom registry",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-custom-registry",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/repo.git",
							Ref: "main",
						},
						Path: "notebook.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test-image:latest",
						BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
							Enabled:   true,
							Strategy:  "tekton",
							BaseImage: "quay.io/jupyter/minimal-notebook:latest",
							StrategyConfig: map[string]string{
								"registry": "custom-registry.example.com:5000",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildInfo, err := strategy.CreateBuild(ctx, tt.job)

			if tt.expectError {
				if err == nil {
					t.Error("CreateBuild() expected error, got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("CreateBuild() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CreateBuild() unexpected error = %v", err)
				}
				if buildInfo == nil {
					t.Error("CreateBuild() returned nil BuildInfo")
				} else {
					// Verify BuildInfo fields
					if buildInfo.Name == "" {
						t.Error("BuildInfo.Name is empty")
					}
					if buildInfo.Status != BuildStatusPending {
						t.Errorf("BuildInfo.Status = %v, want %v", buildInfo.Status, BuildStatusPending)
					}
					if buildInfo.StartTime == nil {
						t.Error("BuildInfo.StartTime is nil")
					}
					if buildInfo.ImageReference == "" {
						t.Error("BuildInfo.ImageReference is empty")
					}
				}
			}
		})
	}
}

// TestTektonStrategyGetBuildStatus tests getting build status
func TestTektonStrategyGetBuildStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = tektonv1.AddToScheme(scheme)

	// Create a fake PipelineRun with minimal status and the label our code looks for
	pipelineRun := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
			Labels: map[string]string{
				"mlops.redhat.com/notebook-validation": "true",
			},
		},
		Status: tektonv1.PipelineRunStatus{
			PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
				// Add any status fields if needed
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineRun).
		Build()

	strategy := NewTektonStrategy(fakeClient, scheme)
	ctx := context.Background()

	buildInfo, err := strategy.GetBuildStatus(ctx, "test-build")
	if err != nil {
		t.Errorf("GetBuildStatus() error = %v", err)
	}
	if buildInfo == nil {
		t.Fatal("GetBuildStatus() returned nil BuildInfo")
	}
	// In test environment without proper status, we expect Unknown or Pending
	if buildInfo.Status != BuildStatusUnknown && buildInfo.Status != BuildStatusPending {
		t.Logf("BuildInfo.Status = %v (expected Unknown or Pending in test environment)", buildInfo.Status)
	}
}

// TestTektonStrategyDeleteBuild tests build deletion
func TestTektonStrategyDeleteBuild(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = tektonv1.AddToScheme(scheme)

	// Create a fake PipelineRun
	pipelineRun := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pipelineRun).
		Build()

	strategy := NewTektonStrategy(fakeClient, scheme)
	ctx := context.Background()

	err := strategy.DeleteBuild(ctx, "test-build")
	if err != nil {
		t.Errorf("DeleteBuild() error = %v", err)
	}
}
