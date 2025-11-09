//go:build integration
// +build integration

package platform

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestIsOpenShift_Integration tests OpenShift detection on a real cluster
// Run with: go test -v -tags=integration ./pkg/platform/...
func TestIsOpenShift_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create clientset: %v", err)
	}

	// Create detector
	detector := NewDetector(nil, clientset.Discovery())

	// Test IsOpenShift
	isOpenShift, err := detector.IsOpenShift(context.Background())
	assert.NoError(t, err)

	t.Logf("IsOpenShift: %v", isOpenShift)

	// If it's OpenShift, get detailed info
	if isOpenShift {
		info, err := detector.GetOpenShiftInfo(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.True(t, info.IsOpenShift)

		t.Logf("OpenShift API Groups: %v", info.APIGroups)
		t.Logf("OpenShift Capabilities: %v", info.Capabilities)

		// Verify build capability is present
		assert.True(t, info.Capabilities["build"], "OpenShift should have build capability")
	}
}

// TestGetOpenShiftInfo_Integration tests detailed OpenShift info on a real cluster
func TestGetOpenShiftInfo_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create clientset: %v", err)
	}

	// Create detector
	detector := NewDetector(nil, clientset.Discovery())

	// Get OpenShift info
	info, err := detector.GetOpenShiftInfo(context.Background())
	assert.NoError(t, err)

	if info != nil {
		t.Logf("OpenShift detected: %v", info.IsOpenShift)
		t.Logf("API Groups (%d): %v", len(info.APIGroups), info.APIGroups)
		t.Logf("Capabilities (%d):", len(info.Capabilities))
		for capability, available := range info.Capabilities {
			t.Logf("  - %s: %v", capability, available)
		}
	} else {
		t.Log("Not an OpenShift cluster")
	}
}

// TestAPIGroupDiscovery_Integration tests API group discovery on a real cluster
func TestAPIGroupDiscovery_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create discovery client: %v", err)
	}

	// Get all API groups
	apiGroupList, err := discoveryClient.ServerGroups()
	assert.NoError(t, err)
	assert.NotNil(t, apiGroupList)

	t.Logf("Total API groups: %d", len(apiGroupList.Groups))

	// Check for OpenShift-specific groups
	openshiftGroups := []string{
		"build.openshift.io",
		"image.openshift.io",
		"route.openshift.io",
		"security.openshift.io",
	}

	foundGroups := make(map[string]bool)
	for _, group := range apiGroupList.Groups {
		for _, osGroup := range openshiftGroups {
			if group.Name == osGroup {
				foundGroups[osGroup] = true
				t.Logf("Found OpenShift API group: %s", osGroup)
			}
		}
	}

	// Log which groups were found
	for _, osGroup := range openshiftGroups {
		if foundGroups[osGroup] {
			t.Logf("✓ %s: present", osGroup)
		} else {
			t.Logf("✗ %s: not found", osGroup)
		}
	}
}
