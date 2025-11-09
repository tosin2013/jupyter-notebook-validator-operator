package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// fakeDiscoveryClient is a simple fake discovery client for testing OpenShift detection
// It only implements the ServerGroups method which is all we need for OpenShift detection
type fakeDiscoveryClient struct {
	discovery.DiscoveryInterface
	apiGroups []metav1.APIGroup
}

func (f *fakeDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	return &metav1.APIGroupList{
		Groups: f.apiGroups,
	}, nil
}

func TestGetPlatformDefinition(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		wantNil  bool
	}{
		{
			name:     "KServe platform",
			platform: PlatformKServe,
			wantNil:  false,
		},
		{
			name:     "OpenShift AI platform",
			platform: PlatformOpenShiftAI,
			wantNil:  false,
		},
		{
			name:     "vLLM platform",
			platform: PlatformVLLM,
			wantNil:  false,
		},
		{
			name:     "Unknown platform",
			platform: PlatformUnknown,
			wantNil:  true,
		},
		{
			name:     "Invalid platform",
			platform: Platform("invalid"),
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := GetPlatformDefinition(tt.platform)
			if tt.wantNil {
				assert.Nil(t, def)
			} else {
				assert.NotNil(t, def)
				assert.Equal(t, tt.platform, def.Platform)
				assert.NotEmpty(t, def.Description)
			}
		})
	}
}

func TestListSupportedPlatforms(t *testing.T) {
	platforms := ListSupportedPlatforms()
	assert.NotEmpty(t, platforms)
	assert.GreaterOrEqual(t, len(platforms), 9) // At least 9 built-in platforms

	// Verify KServe is in the list
	found := false
	for _, p := range platforms {
		if p.Platform == PlatformKServe {
			found = true
			assert.Equal(t, "serving.kserve.io", p.APIGroup)
			assert.Equal(t, "inferenceservices", p.ResourceType)
			break
		}
	}
	assert.True(t, found, "KServe platform should be in supported platforms")
}

func TestNewDetector(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	discoveryClient := clientset.Discovery()

	detector := NewDetector(nil, discoveryClient)
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.discoveryClient)
}

func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		name            string
		platform        Platform
		setupDiscovery  func(*fakediscovery.FakeDiscovery)
		wantAvailable   bool
		wantDetected    bool
		wantErrContains string
	}{
		{
			name:     "KServe platform available",
			platform: PlatformKServe,
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				// Mock KServe CRDs as available
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "serving.kserve.io/v1beta1",
						APIResources: []metav1.APIResource{
							{Name: "inferenceservices", Kind: "InferenceService"},
						},
					},
				}
			},
			wantAvailable: true,
			wantDetected:  false, // Explicitly specified, not auto-detected
		},
		{
			name:     "vLLM platform (no CRDs required)",
			platform: PlatformVLLM,
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				// vLLM doesn't require specific CRDs
			},
			wantAvailable: true, // Available because no CRDs required
			wantDetected:  false,
		},
		{
			name:            "Unknown platform",
			platform:        Platform("invalid"),
			setupDiscovery:  func(fd *fakediscovery.FakeDiscovery) {},
			wantAvailable:   false,
			wantDetected:    false,
			wantErrContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			fakeDiscovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
			tt.setupDiscovery(fakeDiscovery)

			detector := NewDetector(nil, fakeDiscovery)
			ctx := context.Background()

			info, err := detector.validatePlatform(ctx, tt.platform)

			if tt.wantErrContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrContains)
			} else {
				assert.NotNil(t, info)
				assert.Equal(t, tt.platform, info.Platform)
				assert.Equal(t, tt.wantAvailable, info.Available)
				assert.Equal(t, tt.wantDetected, info.Detected)
			}
		})
	}
}

func TestAutoDetectPlatform(t *testing.T) {
	tests := []struct {
		name           string
		setupDiscovery func(*fakediscovery.FakeDiscovery)
		wantPlatform   Platform
		wantDetected   bool
		wantAvailable  bool
	}{
		{
			name: "KServe detected",
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "serving.kserve.io/v1beta1",
						APIResources: []metav1.APIResource{
							{Name: "inferenceservices", Kind: "InferenceService"},
						},
					},
				}
			},
			wantPlatform:  PlatformKServe,
			wantDetected:  true,
			wantAvailable: true,
		},
		{
			name: "Ray Serve detected",
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "ray.io/v1",
						APIResources: []metav1.APIResource{
							{Name: "rayservices", Kind: "RayService"},
						},
					},
				}
			},
			wantPlatform:  PlatformRayServe,
			wantDetected:  true,
			wantAvailable: true,
		},
		{
			name:           "No platform detected",
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {},
			wantPlatform:   PlatformUnknown,
			wantDetected:   false,
			wantAvailable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			fakeDiscovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
			tt.setupDiscovery(fakeDiscovery)

			detector := NewDetector(nil, fakeDiscovery)
			ctx := context.Background()

			info, err := detector.autoDetectPlatform(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, tt.wantPlatform, info.Platform)
			assert.Equal(t, tt.wantDetected, info.Detected)
			assert.Equal(t, tt.wantAvailable, info.Available)
		})
	}
}

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name           string
		platformHint   string
		setupDiscovery func(*fakediscovery.FakeDiscovery)
		wantPlatform   Platform
		wantDetected   bool
	}{
		{
			name:         "With platform hint - KServe",
			platformHint: "kserve",
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "serving.kserve.io/v1beta1",
						APIResources: []metav1.APIResource{
							{Name: "inferenceservices", Kind: "InferenceService"},
						},
					},
				}
			},
			wantPlatform: PlatformKServe,
			wantDetected: false, // Explicitly specified
		},
		{
			name:         "Without platform hint - auto-detect",
			platformHint: "",
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "serving.kserve.io/v1beta1",
						APIResources: []metav1.APIResource{
							{Name: "inferenceservices", Kind: "InferenceService"},
						},
					},
				}
			},
			wantPlatform: PlatformKServe,
			wantDetected: true, // Auto-detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			fakeDiscovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
			tt.setupDiscovery(fakeDiscovery)

			detector := NewDetector(nil, fakeDiscovery)
			ctx := context.Background()

			info, err := detector.DetectPlatform(ctx, tt.platformHint)

			assert.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, tt.wantPlatform, info.Platform)
			assert.Equal(t, tt.wantDetected, info.Detected)
		})
	}
}

func TestCheckCRDs(t *testing.T) {
	tests := []struct {
		name           string
		crdNames       []string
		setupDiscovery func(*fakediscovery.FakeDiscovery)
		wantCRDs       []string
		wantErr        bool
	}{
		{
			name:     "CRDs found",
			crdNames: []string{"inferenceservices.serving.kserve.io"},
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				fd.Resources = []*metav1.APIResourceList{
					{
						GroupVersion: "serving.kserve.io/v1beta1",
						APIResources: []metav1.APIResource{
							{Name: "inferenceservices", Kind: "InferenceService"},
						},
					},
				}
			},
			wantCRDs: []string{"inferenceservices.serving.kserve.io"},
			wantErr:  false,
		},
		{
			name:           "CRDs not found",
			crdNames:       []string{"nonexistent.example.com"},
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {},
			wantCRDs:       nil,
			wantErr:        false, // Not finding CRDs is not an error
		},
		{
			name:     "No discovery client",
			crdNames: []string{"test.example.com"},
			setupDiscovery: func(fd *fakediscovery.FakeDiscovery) {
				// Will be set to nil in test
			},
			wantCRDs: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var discoveryClient discovery.DiscoveryInterface
			if tt.name != "No discovery client" {
				clientset := fake.NewSimpleClientset()
				fakeDiscovery := clientset.Discovery().(*fakediscovery.FakeDiscovery)
				tt.setupDiscovery(fakeDiscovery)
				discoveryClient = fakeDiscovery
			}

			detector := NewDetector(nil, discoveryClient)
			ctx := context.Background()

			crds, err := detector.checkCRDs(ctx, tt.crdNames)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantCRDs == nil {
					assert.Empty(t, crds)
				} else {
					assert.Equal(t, tt.wantCRDs, crds)
				}
			}
		})
	}
}

func TestIsOpenShift(t *testing.T) {
	tests := []struct {
		name          string
		apiGroups     []metav1.APIGroup
		wantOpenShift bool
		wantError     bool
	}{
		{
			name: "OpenShift cluster with build and image API groups",
			apiGroups: []metav1.APIGroup{
				{
					Name: "build.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "build.openshift.io/v1", Version: "v1"},
					},
				},
				{
					Name: "image.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "image.openshift.io/v1", Version: "v1"},
					},
				},
			},
			wantOpenShift: true,
			wantError:     false,
		},
		{
			name: "OpenShift cluster with only build API group",
			apiGroups: []metav1.APIGroup{
				{
					Name: "build.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "build.openshift.io/v1", Version: "v1"},
					},
				},
			},
			wantOpenShift: true,
			wantError:     false,
		},
		{
			name: "OpenShift cluster with only image API group",
			apiGroups: []metav1.APIGroup{
				{
					Name: "image.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "image.openshift.io/v1", Version: "v1"},
					},
				},
			},
			wantOpenShift: true,
			wantError:     false,
		},
		{
			name: "Vanilla Kubernetes cluster",
			apiGroups: []metav1.APIGroup{
				{
					Name: "apps",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "apps/v1", Version: "v1"},
					},
				},
				{
					Name: "batch",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "batch/v1", Version: "v1"},
					},
				},
			},
			wantOpenShift: false,
			wantError:     false,
		},
		{
			name: "OpenShift with additional API groups",
			apiGroups: []metav1.APIGroup{
				{
					Name: "apps",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "apps/v1", Version: "v1"},
					},
				},
				{
					Name: "build.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "build.openshift.io/v1", Version: "v1"},
					},
				},
				{
					Name: "image.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "image.openshift.io/v1", Version: "v1"},
					},
				},
				{
					Name: "route.openshift.io",
					Versions: []metav1.GroupVersionForDiscovery{
						{GroupVersion: "route.openshift.io/v1", Version: "v1"},
					},
				},
			},
			wantOpenShift: true,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake discovery client with custom API groups
			fakeDiscovery := &fakeDiscoveryClient{
				apiGroups: tt.apiGroups,
			}

			// Create detector
			detector := NewDetector(nil, fakeDiscovery)

			// Test IsOpenShift
			isOpenShift, err := detector.IsOpenShift(context.Background())

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOpenShift, isOpenShift)
			}
		})
	}
}

func TestGetOpenShiftInfo(t *testing.T) {
	tests := []struct {
		name              string
		apiGroups         []metav1.APIGroup
		wantNil           bool
		wantCapabilities  map[string]bool
		wantAPIGroupCount int
	}{
		{
			name: "Full OpenShift cluster with common capabilities",
			apiGroups: []metav1.APIGroup{
				{Name: "build.openshift.io"},
				{Name: "image.openshift.io"},
				{Name: "route.openshift.io"},
				{Name: "security.openshift.io"},
				{Name: "project.openshift.io"},
				{Name: "apps.openshift.io"},
				{Name: "tekton.dev"},
			},
			wantNil: false,
			wantCapabilities: map[string]bool{
				"build":     true,
				"image":     true,
				"route":     true,
				"security":  true,
				"project":   true,
				"apps":      true,
				"pipelines": true,
			},
			wantAPIGroupCount: 7,
		},
		{
			name: "Minimal OpenShift cluster",
			apiGroups: []metav1.APIGroup{
				{Name: "build.openshift.io"},
			},
			wantNil: false,
			wantCapabilities: map[string]bool{
				"build": true,
			},
			wantAPIGroupCount: 1,
		},
		{
			name: "Vanilla Kubernetes cluster",
			apiGroups: []metav1.APIGroup{
				{Name: "apps"},
				{Name: "batch"},
			},
			wantNil:           true,
			wantCapabilities:  nil,
			wantAPIGroupCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake discovery client with custom API groups
			fakeDiscovery := &fakeDiscoveryClient{
				apiGroups: tt.apiGroups,
			}

			// Create detector
			detector := NewDetector(nil, fakeDiscovery)

			// Test GetOpenShiftInfo
			info, err := detector.GetOpenShiftInfo(context.Background())

			assert.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, info)
			} else {
				assert.NotNil(t, info)
				assert.True(t, info.IsOpenShift)
				assert.Equal(t, tt.wantAPIGroupCount, len(info.APIGroups))

				// Check specific capabilities
				for capability, expected := range tt.wantCapabilities {
					actual, exists := info.Capabilities[capability]
					assert.True(t, exists, "Capability %s should exist", capability)
					assert.Equal(t, expected, actual, "Capability %s should be %v", capability, expected)
				}
			}
		})
	}
}

func TestIsOpenShift_NoDiscoveryClient(t *testing.T) {
	// Create detector without discovery client
	detector := NewDetector(nil, nil)

	// Test IsOpenShift
	isOpenShift, err := detector.IsOpenShift(context.Background())

	assert.Error(t, err)
	assert.False(t, isOpenShift)
	assert.Contains(t, err.Error(), "discovery client not available")
}

func TestGetOpenShiftInfo_NoDiscoveryClient(t *testing.T) {
	// Create detector without discovery client
	detector := NewDetector(nil, nil)

	// Test GetOpenShiftInfo
	info, err := detector.GetOpenShiftInfo(context.Background())

	assert.Error(t, err)
	assert.Nil(t, info)
}
