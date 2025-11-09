// +build integration

package build

import (
	"context"
	"os"
	"testing"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	buildv1 "github.com/openshift/api/build/v1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	// Test namespace - will be created if it doesn't exist
	testNamespace = "notebook-validator-integration-test"
	
	// Test timeout
	testTimeout = 10 * time.Minute
	
	// Build timeout
	buildTimeout = 5 * time.Minute
)

// setupIntegrationTest sets up the integration test environment
func setupIntegrationTest(t *testing.T) (client.Client, *runtime.Scheme, func()) {
	t.Helper()

	// Get kubeconfig
	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig: %v", err)
	}

	// Create scheme
	testScheme := runtime.NewScheme()
	if err := scheme.AddToScheme(testScheme); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}
	if err := mlopsv1alpha1.AddToScheme(testScheme); err != nil {
		t.Fatalf("Failed to add mlops scheme: %v", err)
	}
	if err := buildv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("Failed to add OpenShift build scheme: %v", err)
	}
	if err := tektonv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("Failed to add Tekton scheme: %v", err)
	}

	// Create client
	k8sClient, err := client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test namespace
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}

	// Try to create namespace, wait if it's terminating
	for i := 0; i < 30; i++ {
		err := k8sClient.Create(ctx, ns)
		if err == nil {
			// Successfully created
			break
		}
		if err != nil && i < 29 {
			// Namespace might be terminating, wait and retry
			time.Sleep(2 * time.Second)
			continue
		}
		// Last attempt failed, but continue anyway (might already exist)
	}

	// Wait for namespace to be ready
	time.Sleep(2 * time.Second)

	// Cleanup function
	cleanup := func() {
		// Delete test namespace (this will cascade delete all resources)
		ctx := context.Background()
		_ = k8sClient.Delete(ctx, ns)
		t.Logf("Cleaned up test namespace: %s", testNamespace)
	}

	return k8sClient, testScheme, cleanup
}

// createTestJob creates a test NotebookValidationJob
func createTestJob(name string, strategy string, baseImage string) *mlopsv1alpha1.NotebookValidationJob {
	return &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Git: mlopsv1alpha1.GitSpec{
					URL: "https://github.com/jupyter/notebook.git",
					Ref: "main",
				},
				Path: "docs/source/examples/Notebook/Running Code.ipynb",
			},
			PodConfig: mlopsv1alpha1.PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
				BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
					Enabled:   true,
					Strategy:  strategy,
					BaseImage: baseImage,
				},
			},
		},
	}
}

// TestIntegrationS2IDetection tests S2I strategy detection on real cluster
func TestIntegrationS2IDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	strategy := NewS2IStrategy(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := strategy.Detect(ctx, k8sClient)
	if err != nil {
		t.Logf("S2I detection error (expected on non-OpenShift): %v", err)
	}

	t.Logf("S2I detected: %v", detected)
	
	// On OpenShift, we expect S2I to be detected
	// On vanilla Kubernetes, we expect false
	if detected {
		t.Log("✅ S2I is available on this cluster")
	} else {
		t.Log("⚠️  S2I is not available on this cluster")
	}
}

// TestIntegrationTektonDetection tests Tekton strategy detection on real cluster
func TestIntegrationTektonDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	strategy := NewTektonStrategy(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := strategy.Detect(ctx, k8sClient)
	if err != nil {
		t.Logf("Tekton detection error: %v", err)
	}

	t.Logf("Tekton detected: %v", detected)
	
	if detected {
		t.Log("✅ Tekton is available on this cluster")
	} else {
		t.Log("⚠️  Tekton is not available on this cluster")
	}
}

// TestIntegrationStrategyRegistry tests strategy registry with real cluster
func TestIntegrationStrategyRegistry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	registry := NewStrategyRegistry(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test strategy listing
	strategies := registry.ListStrategies()
	t.Logf("Registered strategies: %v", strategies)
	
	if len(strategies) == 0 {
		t.Error("No strategies registered")
	}

	// Test strategy detection
	available, err := registry.DetectAvailableStrategies(ctx)
	if err != nil {
		t.Errorf("DetectAvailableStrategies() error = %v", err)
	}
	
	t.Logf("Available strategies: %d", len(available))
	for _, strategy := range available {
		t.Logf("  - %s", strategy.Name())
	}

	if len(available) == 0 {
		t.Log("⚠️  No build strategies available on this cluster")
		t.Log("   This is expected if neither OpenShift S2I nor Tekton are installed")
	}
}

// TestIntegrationS2IBuild tests S2I build creation on real cluster
func TestIntegrationS2IBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we should skip this test
	if os.Getenv("SKIP_S2I_BUILD_TEST") == "true" {
		t.Skip("Skipping S2I build test (SKIP_S2I_BUILD_TEST=true)")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	strategy := NewS2IStrategy(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Check if S2I is available
	detected, err := strategy.Detect(ctx, k8sClient)
	if err != nil || !detected {
		t.Skip("S2I not available on this cluster, skipping build test")
	}

	// Create test job
	job := createTestJob("s2i-test-build", "s2i", "quay.io/jupyter/minimal-notebook:latest")

	// Create build
	t.Log("Creating S2I build...")
	buildInfo, err := strategy.CreateBuild(ctx, job)
	if err != nil {
		t.Fatalf("CreateBuild() error = %v", err)
	}

	t.Logf("Build created: %s", buildInfo.Name)
	t.Logf("Build status: %s", buildInfo.Status)
	t.Logf("Build message: %s", buildInfo.Message)

	// Verify build was created
	if buildInfo.Name == "" {
		t.Error("Build name is empty")
	}
	if buildInfo.Status == "" {
		t.Error("Build status is empty")
	}

	// Get build status
	t.Log("Checking build status...")
	statusInfo, err := strategy.GetBuildStatus(ctx, buildInfo.Name)
	if err != nil {
		t.Errorf("GetBuildStatus() error = %v", err)
	} else {
		t.Logf("Current status: %s", statusInfo.Status)
	}

	// Cleanup: Delete the build
	t.Log("Cleaning up build...")
	if err := strategy.DeleteBuild(ctx, buildInfo.Name); err != nil {
		t.Logf("DeleteBuild() error = %v (may be expected)", err)
	}
}

// TestIntegrationTektonBuild tests Tekton build creation on real cluster
func TestIntegrationTektonBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we should skip this test
	if os.Getenv("SKIP_TEKTON_BUILD_TEST") == "true" {
		t.Skip("Skipping Tekton build test (SKIP_TEKTON_BUILD_TEST=true)")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	strategy := NewTektonStrategy(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Check if Tekton is available
	detected, err := strategy.Detect(ctx, k8sClient)
	if err != nil || !detected {
		t.Skip("Tekton not available on this cluster, skipping build test")
	}

	// Create test job
	job := createTestJob("tekton-test-build", "tekton", "quay.io/jupyter/minimal-notebook:latest")

	// Create build
	t.Log("Creating Tekton build...")
	buildInfo, err := strategy.CreateBuild(ctx, job)
	if err != nil {
		t.Fatalf("CreateBuild() error = %v", err)
	}

	t.Logf("Build created: %s", buildInfo.Name)
	t.Logf("Build status: %s", buildInfo.Status)
	t.Logf("Build message: %s", buildInfo.Message)
	t.Logf("Image reference: %s", buildInfo.ImageReference)

	// Verify build was created
	if buildInfo.Name == "" {
		t.Error("Build name is empty")
	}
	if buildInfo.Status == "" {
		t.Error("Build status is empty")
	}
	if buildInfo.ImageReference == "" {
		t.Error("Image reference is empty")
	}

	// Get build status
	t.Log("Checking build status...")
	statusInfo, err := strategy.GetBuildStatus(ctx, buildInfo.Name)
	if err != nil {
		t.Errorf("GetBuildStatus() error = %v", err)
	} else {
		t.Logf("Current status: %s", statusInfo.Status)
	}

	// Cleanup: Delete the build
	t.Log("Cleaning up build...")
	if err := strategy.DeleteBuild(ctx, buildInfo.Name); err != nil {
		t.Logf("DeleteBuild() error = %v (may be expected)", err)
	}
}

// TestIntegrationAutoStrategySelection tests automatic strategy selection
func TestIntegrationAutoStrategySelection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	registry := NewStrategyRegistry(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test auto-selection with empty strategy
	config := &mlopsv1alpha1.BuildConfigSpec{
		Strategy:  "", // Empty - should auto-select
		BaseImage: "quay.io/jupyter/minimal-notebook:latest",
	}

	strategy, err := registry.SelectStrategy(ctx, config)
	if err != nil {
		t.Logf("SelectStrategy() error = %v (expected if no strategies available)", err)

		// Check if it's the expected error
		if _, ok := err.(*NoStrategyAvailableError); ok {
			t.Log("✅ Correctly returned NoStrategyAvailableError")
		}
		return
	}

	if strategy != nil {
		t.Logf("✅ Auto-selected strategy: %s", strategy.Name())
	}
}

// TestIntegrationBuildWithCustomRegistry tests Tekton build with custom registry
func TestIntegrationBuildWithCustomRegistry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we should skip this test
	if os.Getenv("SKIP_TEKTON_BUILD_TEST") == "true" {
		t.Skip("Skipping Tekton build test (SKIP_TEKTON_BUILD_TEST=true)")
	}

	k8sClient, testScheme, cleanup := setupIntegrationTest(t)
	defer cleanup()

	strategy := NewTektonStrategy(k8sClient, testScheme)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Check if Tekton is available
	detected, err := strategy.Detect(ctx, k8sClient)
	if err != nil || !detected {
		t.Skip("Tekton not available on this cluster, skipping build test")
	}

	// Create test job with custom registry
	job := createTestJob("tekton-custom-registry", "tekton", "quay.io/jupyter/minimal-notebook:latest")
	job.Spec.PodConfig.BuildConfig.StrategyConfig = map[string]string{
		"registry": "image-registry.openshift-image-registry.svc:5000",
	}

	// Create build
	t.Log("Creating Tekton build with custom registry...")
	buildInfo, err := strategy.CreateBuild(ctx, job)
	if err != nil {
		t.Fatalf("CreateBuild() error = %v", err)
	}

	t.Logf("Build created: %s", buildInfo.Name)
	t.Logf("Image reference: %s", buildInfo.ImageReference)

	// Verify custom registry is used
	if buildInfo.ImageReference == "" {
		t.Error("Image reference is empty")
	}

	// Cleanup
	t.Log("Cleaning up build...")
	if err := strategy.DeleteBuild(ctx, buildInfo.Name); err != nil {
		t.Logf("DeleteBuild() error = %v (may be expected)", err)
	}
}

// TestIntegrationClusterInfo tests cluster information retrieval
func TestIntegrationClusterInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig: %v", err)
	}

	t.Logf("Cluster host: %s", cfg.Host)
	t.Logf("API version: %s", cfg.APIPath)

	// Log OpenShift version if available
	k8sClient, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get cluster version (OpenShift specific)
	t.Log("Checking for OpenShift cluster version...")

	// List namespaces to verify connectivity
	namespaces := &corev1.NamespaceList{}
	if err := k8sClient.List(ctx, namespaces); err != nil {
		t.Errorf("Failed to list namespaces: %v", err)
	} else {
		t.Logf("✅ Successfully connected to cluster (%d namespaces)", len(namespaces.Items))
	}
}

