// Package platform provides model serving platform detection and validation
package platform

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Platform represents a model serving platform
type Platform string

const (
	// PlatformKServe represents KServe model serving
	PlatformKServe Platform = "kserve"
	// PlatformOpenShiftAI represents OpenShift AI model serving
	PlatformOpenShiftAI Platform = "openshift-ai"
	// PlatformVLLM represents vLLM model serving
	PlatformVLLM Platform = "vllm"
	// PlatformTorchServe represents TorchServe model serving
	PlatformTorchServe Platform = "torchserve"
	// PlatformTensorFlowServing represents TensorFlow Serving
	PlatformTensorFlowServing Platform = "tensorflow-serving"
	// PlatformTriton represents Triton Inference Server
	PlatformTriton Platform = "triton"
	// PlatformRayServe represents Ray Serve
	PlatformRayServe Platform = "ray-serve"
	// PlatformSeldon represents Seldon Core
	PlatformSeldon Platform = "seldon"
	// PlatformBentoML represents BentoML
	PlatformBentoML Platform = "bentoml"
	// PlatformCustom represents a custom platform
	PlatformCustom Platform = "custom"
	// PlatformUnknown represents an unknown platform
	PlatformUnknown Platform = "unknown"
)

// PlatformInfo contains information about a detected platform
type PlatformInfo struct {
	// Platform is the detected platform type
	Platform Platform
	// Detected indicates whether the platform was auto-detected
	Detected bool
	// CRDs lists the detected CRDs for this platform
	CRDs []string
	// APIGroup is the Kubernetes API group for the platform
	APIGroup string
	// ResourceType is the primary resource type for models
	ResourceType string
	// Available indicates if the platform is available in the cluster
	Available bool
	// Message provides additional information
	Message string
}

// PlatformDefinition defines the characteristics of a model serving platform
type PlatformDefinition struct {
	// Platform is the platform identifier
	Platform Platform
	// APIGroup is the Kubernetes API group
	APIGroup string
	// ResourceType is the primary resource type for models
	ResourceType string
	// CRDNames lists the CRDs that indicate this platform
	CRDNames []string
	// Description is a human-readable description
	Description string
}

// builtInPlatforms defines the built-in platform definitions
var builtInPlatforms = []PlatformDefinition{
	{
		Platform:     PlatformKServe,
		APIGroup:     "serving.kserve.io",
		ResourceType: "inferenceservices",
		CRDNames:     []string{"inferenceservices.serving.kserve.io"},
		Description:  "KServe - Standard Kubernetes model serving",
	},
	{
		Platform:     PlatformOpenShiftAI,
		APIGroup:     "serving.kserve.io",
		ResourceType: "inferenceservices",
		CRDNames: []string{
			"inferenceservices.serving.kserve.io",
			"servingruntime.serving.kserve.io",
		},
		Description: "OpenShift AI - Red Hat's enterprise AI platform",
	},
	{
		Platform:     PlatformVLLM,
		APIGroup:     "apps",
		ResourceType: "deployments",
		CRDNames:     []string{}, // vLLM typically uses standard Deployments
		Description:  "vLLM - LLM-focused serving",
	},
	{
		Platform:     PlatformTorchServe,
		APIGroup:     "apps",
		ResourceType: "deployments",
		CRDNames:     []string{}, // TorchServe typically uses standard Deployments
		Description:  "TorchServe - PyTorch model serving",
	},
	{
		Platform:     PlatformTensorFlowServing,
		APIGroup:     "apps",
		ResourceType: "deployments",
		CRDNames:     []string{}, // TensorFlow Serving typically uses standard Deployments
		Description:  "TensorFlow Serving - TensorFlow model serving",
	},
	{
		Platform:     PlatformTriton,
		APIGroup:     "apps",
		ResourceType: "deployments",
		CRDNames:     []string{}, // Triton typically uses standard Deployments
		Description:  "Triton Inference Server - NVIDIA multi-framework serving",
	},
	{
		Platform:     PlatformRayServe,
		APIGroup:     "ray.io",
		ResourceType: "rayservices",
		CRDNames:     []string{"rayservices.ray.io", "rayclusters.ray.io"},
		Description:  "Ray Serve - Distributed model serving",
	},
	{
		Platform:     PlatformSeldon,
		APIGroup:     "machinelearning.seldon.io",
		ResourceType: "seldondeployments",
		CRDNames:     []string{"seldondeployments.machinelearning.seldon.io"},
		Description:  "Seldon Core - Advanced ML deployments",
	},
	{
		Platform:     PlatformBentoML,
		APIGroup:     "serving.yatai.ai",
		ResourceType: "bentos",
		CRDNames:     []string{"bentos.serving.yatai.ai", "bentodeployments.serving.yatai.ai"},
		Description:  "BentoML - Model packaging and serving",
	},
}

// Detector provides platform detection capabilities
type Detector struct {
	client          client.Client
	discoveryClient discovery.DiscoveryInterface
}

// NewDetector creates a new platform detector
func NewDetector(c client.Client, discoveryClient discovery.DiscoveryInterface) *Detector {
	return &Detector{
		client:          c,
		discoveryClient: discoveryClient,
	}
}

// DetectPlatform detects the model serving platform in the cluster
// If platformHint is provided, it validates that platform; otherwise, it auto-detects
func (d *Detector) DetectPlatform(ctx context.Context, platformHint string) (*PlatformInfo, error) {
	log := log.FromContext(ctx)

	// If platform hint is provided, validate it
	if platformHint != "" {
		log.V(1).Info("Validating specified platform", "platform", platformHint)
		return d.validatePlatform(ctx, Platform(platformHint))
	}

	// Auto-detect platform
	log.V(1).Info("Auto-detecting model serving platform")
	return d.autoDetectPlatform(ctx)
}

// validatePlatform validates that a specific platform is available
func (d *Detector) validatePlatform(ctx context.Context, platform Platform) (*PlatformInfo, error) {
	log := log.FromContext(ctx)

	// Find platform definition
	var platformDef *PlatformDefinition
	for i := range builtInPlatforms {
		if builtInPlatforms[i].Platform == platform {
			platformDef = &builtInPlatforms[i]
			break
		}
	}

	if platformDef == nil {
		return &PlatformInfo{
			Platform:  platform,
			Detected:  false,
			Available: false,
			Message:   fmt.Sprintf("Unknown platform: %s", platform),
		}, nil
	}

	// Check if platform CRDs are installed
	detectedCRDs, err := d.checkCRDs(ctx, platformDef.CRDNames)
	if err != nil {
		log.Error(err, "Failed to check CRDs", "platform", platform)
		return &PlatformInfo{
			Platform:  platform,
			Detected:  false,
			Available: false,
			Message:   fmt.Sprintf("Failed to check CRDs: %v", err),
		}, err
	}

	available := len(detectedCRDs) > 0 || len(platformDef.CRDNames) == 0
	message := fmt.Sprintf("Platform %s validated", platform)
	if !available {
		message = fmt.Sprintf("Platform %s not available (CRDs not found)", platform)
	}

	return &PlatformInfo{
		Platform:     platform,
		Detected:     false, // Explicitly specified, not auto-detected
		CRDs:         detectedCRDs,
		APIGroup:     platformDef.APIGroup,
		ResourceType: platformDef.ResourceType,
		Available:    available,
		Message:      message,
	}, nil
}

// autoDetectPlatform attempts to auto-detect the platform
func (d *Detector) autoDetectPlatform(ctx context.Context) (*PlatformInfo, error) {
	log := log.FromContext(ctx)

	// Try to detect each platform in priority order
	for _, platformDef := range builtInPlatforms {
		if len(platformDef.CRDNames) == 0 {
			// Skip platforms without CRDs (they can't be auto-detected)
			continue
		}

		detectedCRDs, err := d.checkCRDs(ctx, platformDef.CRDNames)
		if err != nil {
			log.V(2).Info("Error checking CRDs", "platform", platformDef.Platform, "error", err)
			continue
		}

		if len(detectedCRDs) > 0 {
			log.Info("Platform auto-detected", "platform", platformDef.Platform, "crds", detectedCRDs)
			return &PlatformInfo{
				Platform:     platformDef.Platform,
				Detected:     true,
				CRDs:         detectedCRDs,
				APIGroup:     platformDef.APIGroup,
				ResourceType: platformDef.ResourceType,
				Available:    true,
				Message:      fmt.Sprintf("Auto-detected %s platform", platformDef.Platform),
			}, nil
		}
	}

	// No platform detected
	log.Info("No model serving platform detected")
	return &PlatformInfo{
		Platform:  PlatformUnknown,
		Detected:  false,
		Available: false,
		Message:   "No model serving platform detected in cluster",
	}, nil
}

// checkCRDs checks if the specified CRDs are installed in the cluster
func (d *Detector) checkCRDs(ctx context.Context, crdNames []string) ([]string, error) {
	logger := log.FromContext(ctx)

	if d.discoveryClient == nil {
		return nil, fmt.Errorf("discovery client not available")
	}

	var detectedCRDs []string

	// Get all API resources
	_, apiResourceLists, err := d.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		// Partial errors are acceptable (some API groups may be unavailable)
		// Continue with whatever resources we got
		logger.V(1).Info("Partial error getting API resources", "error", err)
	}

	// Build a map of available resources
	availableResources := make(map[string]bool)
	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			// Extract group from GroupVersion (format: "group/version")
			group := ""
			if gv := apiResourceList.GroupVersion; gv != "" {
				parts := splitGroupVersion(gv)
				if len(parts) > 0 {
					group = parts[0]
				}
			}

			// Build resource name in format: <resource>.<group>
			resourceName := apiResource.Name
			if group != "" {
				resourceName = apiResource.Name + "." + group
			}
			availableResources[resourceName] = true
		}
	}

	// Check which CRDs are available
	for _, crdName := range crdNames {
		if availableResources[crdName] {
			detectedCRDs = append(detectedCRDs, crdName)
		}
	}

	return detectedCRDs, nil
}

// splitGroupVersion splits a GroupVersion string into [group, version]
func splitGroupVersion(gv string) []string {
	// Handle both "group/version" and "version" formats
	if gv == "" {
		return []string{}
	}

	// Check if it contains a slash
	for i, c := range gv {
		if c == '/' {
			return []string{gv[:i], gv[i+1:]}
		}
	}

	// No slash means it's just a version (core API)
	return []string{"", gv}
}

// GetPlatformDefinition returns the platform definition for a given platform
func GetPlatformDefinition(platform Platform) *PlatformDefinition {
	for i := range builtInPlatforms {
		if builtInPlatforms[i].Platform == platform {
			return &builtInPlatforms[i]
		}
	}
	return nil
}

// ListSupportedPlatforms returns a list of all supported platforms
func ListSupportedPlatforms() []PlatformDefinition {
	return builtInPlatforms
}
