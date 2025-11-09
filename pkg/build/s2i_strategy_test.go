package build

import (
	"context"
	"testing"

	buildv1 "github.com/openshift/api/build/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestNewS2IStrategy tests S2I strategy creation
func TestNewS2IStrategy(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := NewS2IStrategy(fakeClient, scheme)

	if strategy == nil {
		t.Fatal("NewS2IStrategy returned nil")
	}

	if strategy.Name() != "s2i" {
		t.Errorf("Name() = %v, want s2i", strategy.Name())
	}
}

// TestS2IStrategyName tests the Name method
func TestS2IStrategyName(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := NewS2IStrategy(fakeClient, scheme)

	if strategy.Name() != "s2i" {
		t.Errorf("Name() = %v, want s2i", strategy.Name())
	}
}

// TestS2IStrategyDetect tests the Detect method
func TestS2IStrategyDetect(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := NewS2IStrategy(fakeClient, scheme)
	ctx := context.Background()

	// In a test environment without real OpenShift, detection should return false or error
	detected, err := strategy.Detect(ctx, fakeClient)

	// We expect either false (not detected) or an error (CRD not registered)
	if detected {
		t.Log("S2I detected in test environment (unexpected but not an error)")
	}
	if err != nil {
		t.Logf("Detect() error = %v (expected in test environment)", err)
	}
}

// TestS2IStrategyValidateConfig tests configuration validation
func TestS2IStrategyValidateConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := NewS2IStrategy(fakeClient, scheme)

	tests := []struct {
		name        string
		config      *mlopsv1alpha1.BuildConfigSpec
		expectError bool
	}{
		{
			name: "Valid config with base image",
			config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy:  "s2i",
				BaseImage: "quay.io/jupyter/minimal-notebook:latest",
			},
			expectError: false,
		},
		{
			name: "Valid config without base image (uses default)",
			config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "s2i",
			},
			expectError: false,
		},
		{
			name: "Valid config with strategy config",
			config: &mlopsv1alpha1.BuildConfigSpec{
				Strategy:  "s2i",
				BaseImage: "custom-image:latest",
				StrategyConfig: map[string]string{
					"incremental": "true",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := strategy.ValidateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("ValidateConfig() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateConfig() unexpected error = %v", err)
			}
		})
	}
}

// TestS2IStrategyCreateBuild tests build creation
func TestS2IStrategyCreateBuild(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := NewS2IStrategy(fakeClient, scheme)
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
							Strategy:  "s2i",
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
							Strategy: "s2i",
							// BaseImage not specified - should use default
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
				}
			}
		})
	}
}

// TestS2IStrategyGetBuildStatus tests getting build status
func TestS2IStrategyGetBuildStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)

	// Create a fake build with the label our code looks for
	build := &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
			Labels: map[string]string{
				"mlops.redhat.com/notebook-validation": "true",
			},
		},
		Status: buildv1.BuildStatus{
			Phase: buildv1.BuildPhaseComplete,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(build).
		Build()

	strategy := NewS2IStrategy(fakeClient, scheme)
	ctx := context.Background()

	buildInfo, err := strategy.GetBuildStatus(ctx, "test-build")
	if err != nil {
		t.Errorf("GetBuildStatus() error = %v", err)
	}
	if buildInfo == nil {
		t.Fatal("GetBuildStatus() returned nil BuildInfo")
	}
	if buildInfo.Status != BuildStatusComplete {
		t.Errorf("BuildInfo.Status = %v, want %v", buildInfo.Status, BuildStatusComplete)
	}
}

// TestS2IStrategyDeleteBuild tests build deletion
func TestS2IStrategyDeleteBuild(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = buildv1.AddToScheme(scheme)

	// Create a fake build
	build := &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(build).
		Build()

	strategy := NewS2IStrategy(fakeClient, scheme)
	ctx := context.Background()

	err := strategy.DeleteBuild(ctx, "test-build")
	if err != nil {
		t.Errorf("DeleteBuild() error = %v", err)
	}
}
