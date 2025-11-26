package build

import (
	"context"
	"testing"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestBuildStatusConstants tests that BuildStatus constants are defined correctly
func TestBuildStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   BuildStatus
		expected string
	}{
		{"Pending status", BuildStatusPending, "Pending"},
		{"Running status", BuildStatusRunning, "Running"},
		{"Complete status", BuildStatusComplete, "Complete"},
		{"Failed status", BuildStatusFailed, "Failed"},
		{"Cancelled status", BuildStatusCancelled, "Cancelled"},
		{"Unknown status", BuildStatusUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("BuildStatus %s = %v, want %v", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestBuildInfo tests BuildInfo structure
func TestBuildInfo(t *testing.T) {
	now := time.Now()
	buildInfo := &BuildInfo{
		Name:           "test-build",
		Status:         BuildStatusRunning,
		Message:        "Build in progress",
		ImageReference: "registry.example.com/test:latest",
		StartTime:      &now,
		CompletionTime: nil,
		Logs:           "Building...",
	}

	if buildInfo.Name != "test-build" {
		t.Errorf("BuildInfo.Name = %v, want test-build", buildInfo.Name)
	}
	if buildInfo.Status != BuildStatusRunning {
		t.Errorf("BuildInfo.Status = %v, want %v", buildInfo.Status, BuildStatusRunning)
	}
	if buildInfo.StartTime == nil {
		t.Error("BuildInfo.StartTime should not be nil")
	}
	if buildInfo.CompletionTime != nil {
		t.Error("BuildInfo.CompletionTime should be nil for running build")
	}
}

// TestStrategyRegistry tests the strategy registry
func TestStrategyRegistry(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	registry := NewStrategyRegistry(fakeClient, fakeClient, scheme)

	if registry == nil {
		t.Fatal("NewStrategyRegistry returned nil")
	}

	// Test that strategies are registered
	strategies := registry.ListStrategies()
	if len(strategies) == 0 {
		t.Error("No strategies registered")
	}

	// Check for expected strategies
	expectedStrategies := []string{"s2i", "tekton"}
	for _, expected := range expectedStrategies {
		found := false
		for _, strategy := range strategies {
			if strategy == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected strategy %s not found in registry", expected)
		}
	}
}

// TestGetStrategy tests retrieving strategies from registry
func TestGetStrategy(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	registry := NewStrategyRegistry(fakeClient, fakeClient, scheme)

	tests := []struct {
		name         string
		strategyName string
		shouldExist  bool
		expectedName string
	}{
		{"Get S2I strategy", "s2i", true, "s2i"},
		{"Get Tekton strategy", "tekton", true, "tekton"},
		{"Get non-existent strategy", "nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := registry.GetStrategy(tt.strategyName)
			if tt.shouldExist {
				if strategy == nil {
					t.Errorf("GetStrategy(%s) returned nil, expected strategy", tt.strategyName)
				} else if strategy.Name() != tt.expectedName {
					t.Errorf("Strategy.Name() = %v, want %v", strategy.Name(), tt.expectedName)
				}
			} else {
				if strategy != nil {
					t.Errorf("GetStrategy(%s) returned %v, expected nil", tt.strategyName, strategy)
				}
			}
		})
	}
}

// TestDetectAvailableStrategies tests auto-detection of available strategies
func TestDetectAvailableStrategies(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	registry := NewStrategyRegistry(fakeClient, fakeClient, scheme)
	ctx := context.Background()

	available, err := registry.DetectAvailableStrategies(ctx)
	if err != nil {
		t.Fatalf("DetectAvailableStrategies() error = %v", err)
	}

	// In a test environment without real cluster resources, we expect empty or error
	// This test mainly ensures the method doesn't panic
	t.Logf("Detected %d available strategies", len(available))
}

// TestSelectStrategy tests strategy selection logic
func TestSelectStrategy(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	registry := NewStrategyRegistry(fakeClient, fakeClient, scheme)
	ctx := context.Background()

	tests := []struct {
		name          string
		buildConfig   *mlopsv1alpha1.BuildConfigSpec
		expectedError bool
		expectedName  string
	}{
		{
			name: "Select S2I strategy explicitly",
			buildConfig: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "s2i",
			},
			expectedError: false,
			expectedName:  "s2i",
		},
		{
			name: "Select Tekton strategy explicitly",
			buildConfig: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "tekton",
			},
			expectedError: false,
			expectedName:  "tekton",
		},
		{
			name: "Select non-existent strategy",
			buildConfig: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "nonexistent",
			},
			expectedError: true,
		},
		{
			name: "Auto-select strategy (empty)",
			buildConfig: &mlopsv1alpha1.BuildConfigSpec{
				Strategy: "",
			},
			expectedError: true, // In test environment, no strategies are available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := registry.SelectStrategy(ctx, tt.buildConfig)
			if tt.expectedError {
				if err == nil {
					t.Error("SelectStrategy() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SelectStrategy() unexpected error = %v", err)
				}
				if strategy != nil && tt.expectedName != "" && strategy.Name() != tt.expectedName {
					t.Errorf("Strategy.Name() = %v, want %v", strategy.Name(), tt.expectedName)
				}
			}
		})
	}
}

// mockStrategy is a mock implementation of Strategy interface for testing
type mockStrategy struct {
	name              string
	detectResult      bool
	detectError       error
	createBuildResult *BuildInfo
	createBuildError  error
}

func (m *mockStrategy) Name() string {
	return m.name
}

func (m *mockStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
	return m.detectResult, m.detectError
}

func (m *mockStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
	return m.createBuildResult, m.createBuildError
}

func (m *mockStrategy) GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error) {
	return m.createBuildResult, nil
}

func (m *mockStrategy) WaitForCompletion(ctx context.Context, buildName string, timeout time.Duration) (*BuildInfo, error) {
	return m.createBuildResult, nil
}

func (m *mockStrategy) GetBuildLogs(ctx context.Context, buildName string) (string, error) {
	return "mock logs", nil
}

func (m *mockStrategy) DeleteBuild(ctx context.Context, buildName string) error {
	return nil
}

func (m *mockStrategy) ValidateConfig(config *mlopsv1alpha1.BuildConfigSpec) error {
	return nil
}

// TestMockStrategy tests the mock strategy implementation
func TestMockStrategy(t *testing.T) {
	mock := &mockStrategy{
		name:         "mock",
		detectResult: true,
		detectError:  nil,
		createBuildResult: &BuildInfo{
			Name:   "mock-build",
			Status: BuildStatusComplete,
		},
	}

	if mock.Name() != "mock" {
		t.Errorf("Name() = %v, want mock", mock.Name())
	}

	ctx := context.Background()
	detected, err := mock.Detect(ctx, nil)
	if err != nil {
		t.Errorf("Detect() error = %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true")
	}

	buildInfo, err := mock.CreateBuild(ctx, nil)
	if err != nil {
		t.Errorf("CreateBuild() error = %v", err)
	}
	if buildInfo.Name != "mock-build" {
		t.Errorf("BuildInfo.Name = %v, want mock-build", buildInfo.Name)
	}
}
