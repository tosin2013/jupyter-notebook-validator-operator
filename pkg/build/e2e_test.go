//go:build e2e
// +build e2e

package build

import (
	"context"
	"fmt"
	"testing"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	e2eTestNamespace = "notebook-validator-e2e-test"
	e2eTestTimeout   = 15 * time.Minute // E2E tests can take longer
)

// setupE2ETest sets up the e2e test environment
func setupE2ETest(t *testing.T) (client.Client, *runtime.Scheme, func()) {
	// Load kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create scheme
	testScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(testScheme)
	_ = mlopsv1alpha1.AddToScheme(testScheme)
	_ = buildv1.AddToScheme(testScheme)
	_ = tektonv1.AddToScheme(testScheme)

	// Create client
	k8sClient, err := client.New(config, client.Options{Scheme: testScheme})
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create test namespace
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: e2eTestNamespace,
		},
	}

	// Try to create namespace, wait if it's terminating
	for i := 0; i < 30; i++ {
		err := k8sClient.Create(ctx, ns)
		if err == nil {
			break
		}
		if err != nil && i < 29 {
			t.Logf("Waiting for namespace to be ready (attempt %d/30)...", i+1)
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
			t.Fatalf("Failed to create test namespace: %v", err)
		}
	}

	// Wait for namespace to be ready
	time.Sleep(2 * time.Second)
	t.Logf("Created test namespace: %s", e2eTestNamespace)

	// Cleanup function
	cleanup := func() {
		ctx := context.Background()
		_ = k8sClient.Delete(ctx, ns)
		t.Logf("Cleaned up test namespace: %s", e2eTestNamespace)
	}

	return k8sClient, testScheme, cleanup
}

// TestE2ES2IWorkflow tests the complete S2I build workflow
func TestE2ES2IWorkflow(t *testing.T) {
	k8sClient, testScheme, cleanup := setupE2ETest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), e2eTestTimeout)
	defer cancel()

	// Create S2I strategy
	s2iStrategy := NewS2IStrategy(k8sClient, testScheme)

	// Check if S2I is available
	available, err := s2iStrategy.IsAvailable(ctx)
	if err != nil {
		t.Fatalf("Failed to check S2I availability: %v", err)
	}
	if !available {
		t.Skip("S2I not available on this cluster, skipping e2e test")
	}

	// Create test NotebookValidationJob
	job := &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-s2i-test",
			Namespace: e2eTestNamespace,
		},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Git: mlopsv1alpha1.GitSpec{
					URL: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git",
					Ref: "main",
				},
				Path: "simple-notebook.ipynb",
			},
			PodConfig: mlopsv1alpha1.PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
				BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
					Enabled:   true,
					Strategy:  "s2i",
					BaseImage: "quay.io/jupyter/minimal-notebook:latest",
				},
			},
		},
	}

	// Validate configuration
	if err := s2iStrategy.ValidateConfig(job); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Create build
	t.Log("Creating S2I build...")
	buildName, err := s2iStrategy.CreateBuild(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create build: %v", err)
	}
	t.Logf("Build created: %s", buildName)

	// Wait for build to complete
	t.Log("Waiting for build to complete...")
	buildInfo, err := waitForBuildCompletion(ctx, s2iStrategy, buildName, 10*time.Minute)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify build completed successfully
	if buildInfo.Status != StatusComplete {
		t.Fatalf("Build did not complete successfully: status=%s, message=%s", buildInfo.Status, buildInfo.Message)
	}
	t.Logf("Build completed successfully: %s", buildInfo.ImageReference)

	// Verify image reference is set
	if buildInfo.ImageReference == "" {
		t.Fatal("Build completed but image reference is empty")
	}

	// Cleanup build
	t.Log("Cleaning up build...")
	if err := s2iStrategy.DeleteBuild(ctx, buildName); err != nil {
		t.Errorf("Failed to cleanup build: %v", err)
	}
}

// TestE2ETektonWorkflow tests the complete Tekton build workflow
func TestE2ETektonWorkflow(t *testing.T) {
	k8sClient, testScheme, cleanup := setupE2ETest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), e2eTestTimeout)
	defer cancel()

	// Create Tekton strategy
	tektonStrategy := NewTektonStrategy(k8sClient, k8sClient, testScheme)

	// Check if Tekton is available
	available, err := tektonStrategy.IsAvailable(ctx)
	if err != nil {
		t.Fatalf("Failed to check Tekton availability: %v", err)
	}
	if !available {
		t.Skip("Tekton not available on this cluster, skipping e2e test")
	}

	// Create test NotebookValidationJob
	job := &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-tekton-test",
			Namespace: e2eTestNamespace,
		},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Git: mlopsv1alpha1.GitSpec{
					URL: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git",
					Ref: "main",
				},
				Path: "simple-notebook.ipynb",
			},
			PodConfig: mlopsv1alpha1.PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
				BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
					Enabled:   true,
					Strategy:  "tekton",
					BaseImage: "quay.io/jupyter/minimal-notebook:latest",
				},
			},
		},
	}

	// Validate configuration
	if err := tektonStrategy.ValidateConfig(job); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Create build
	t.Log("Creating Tekton build...")
	buildName, err := tektonStrategy.CreateBuild(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create build: %v", err)
	}
	t.Logf("Build created: %s", buildName)

	// Wait for build to complete
	t.Log("Waiting for build to complete...")
	buildInfo, err := waitForBuildCompletion(ctx, tektonStrategy, buildName, 10*time.Minute)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify build completed successfully
	if buildInfo.Status != StatusComplete {
		t.Fatalf("Build did not complete successfully: status=%s, message=%s", buildInfo.Status, buildInfo.Message)
	}
	t.Logf("Build completed successfully: %s", buildInfo.ImageReference)

	// Verify image reference is set
	if buildInfo.ImageReference == "" {
		t.Fatal("Build completed but image reference is empty")
	}

	// Cleanup build
	t.Log("Cleaning up build...")
	if err := tektonStrategy.DeleteBuild(ctx, buildName); err != nil {
		t.Errorf("Failed to cleanup build: %v", err)
	}
}

// TestE2ECompleteNotebookValidationWithS2I tests the complete workflow:
// 1. Create NotebookValidationJob with build enabled
// 2. Build triggers automatically
// 3. Built image is used for validation
// 4. Notebook executes successfully
// 5. Results are collected
func TestE2ECompleteNotebookValidationWithS2I(t *testing.T) {
	t.Skip("Requires full controller deployment - implement after controller integration")
	// This test will be implemented after integrating build strategies into the controller
}

// TestE2ECompleteNotebookValidationWithTekton tests the complete workflow with Tekton
func TestE2ECompleteNotebookValidationWithTekton(t *testing.T) {
	t.Skip("Requires full controller deployment - implement after controller integration")
	// This test will be implemented after integrating build strategies into the controller
}

// waitForBuildCompletion waits for a build to complete
func waitForBuildCompletion(ctx context.Context, strategy Strategy, buildName string, timeout time.Duration) (*BuildInfo, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		buildInfo, err := strategy.GetBuildStatus(ctx, buildName)
		if err != nil {
			return nil, fmt.Errorf("failed to get build status: %w", err)
		}

		switch buildInfo.Status {
		case StatusComplete:
			return buildInfo, nil
		case StatusFailed:
			return nil, fmt.Errorf("build failed: %s", buildInfo.Message)
		case StatusCancelled:
			return nil, fmt.Errorf("build was cancelled: %s", buildInfo.Message)
		case StatusPending, StatusRunning:
			// Continue waiting
			time.Sleep(10 * time.Second)
		default:
			return nil, fmt.Errorf("unknown build status: %s", buildInfo.Status)
		}
	}

	return nil, fmt.Errorf("build did not complete within timeout (%v)", timeout)
}
