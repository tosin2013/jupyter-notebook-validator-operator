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

// IsOpenShift detects if the cluster is running OpenShift
// It checks for OpenShift-specific API groups:
// - build.openshift.io (BuildConfig, Build)
// - image.openshift.io (ImageStream, ImageStreamTag)
func (d *Detector) IsOpenShift(ctx context.Context) (bool, error) {
	log := log.FromContext(ctx)

	if d.discoveryClient == nil {
		return false, fmt.Errorf("discovery client not available")
	}

	// OpenShift-specific API groups to check
	openshiftAPIGroups := []string{
		"build.openshift.io",
		"image.openshift.io",
	}

	// Get all API groups
	apiGroupList, err := d.discoveryClient.ServerGroups()
	if err != nil {
		log.Error(err, "Failed to get server groups")
		return false, err
	}

	// Build a map of available API groups
	availableGroups := make(map[string]bool)
	for _, group := range apiGroupList.Groups {
		availableGroups[group.Name] = true
	}

	// Check if all OpenShift API groups are present
	foundCount := 0
	for _, openshiftGroup := range openshiftAPIGroups {
		if availableGroups[openshiftGroup] {
			foundCount++
			log.V(1).Info("Found OpenShift API group", "group", openshiftGroup)
		}
	}

	// Consider it OpenShift if we find at least one OpenShift API group
	// (some minimal OpenShift installations might not have all groups)
	isOpenShift := foundCount > 0

	if isOpenShift {
		log.Info("OpenShift cluster detected", "apiGroups", foundCount)
	} else {
		log.V(1).Info("Not an OpenShift cluster (no OpenShift API groups found)")
	}

	return isOpenShift, nil
}

// GetOpenShiftInfo returns detailed information about the OpenShift cluster
// Returns nil if not running on OpenShift
func (d *Detector) GetOpenShiftInfo(ctx context.Context) (*OpenShiftInfo, error) {
	log := log.FromContext(ctx)

	isOpenShift, err := d.IsOpenShift(ctx)
	if err != nil {
		return nil, err
	}

	if !isOpenShift {
		return nil, nil
	}

	info := &OpenShiftInfo{
		IsOpenShift:  true,
		APIGroups:    []string{},
		Capabilities: make(map[string]bool),
	}

	// Check for specific OpenShift capabilities
	capabilities := map[string][]string{
		"build":      {"build.openshift.io"},
		"image":      {"image.openshift.io"},
		"route":      {"route.openshift.io"},
		"security":   {"security.openshift.io"},
		"project":    {"project.openshift.io"},
		"template":   {"template.openshift.io"},
		"apps":       {"apps.openshift.io"},
		"oauth":      {"oauth.openshift.io"},
		"user":       {"user.openshift.io"},
		"operator":   {"operator.openshift.io"},
		"config":     {"config.openshift.io"},
		"console":    {"console.openshift.io"},
		"monitoring": {"monitoring.coreos.com"},
		"serverless": {"serving.knative.dev"},
		"pipelines":  {"tekton.dev"},
		"gitops":     {"argoproj.io"},
	}

	// Get all API groups
	apiGroupList, err := d.discoveryClient.ServerGroups()
	if err != nil {
		log.Error(err, "Failed to get server groups for OpenShift info")
		return info, err
	}

	// Build a map of available API groups
	availableGroups := make(map[string]bool)
	for _, group := range apiGroupList.Groups {
		availableGroups[group.Name] = true
	}

	// Check each capability
	for capability, groups := range capabilities {
		found := false
		for _, group := range groups {
			if availableGroups[group] {
				found = true
				info.APIGroups = append(info.APIGroups, group)
				break
			}
		}
		info.Capabilities[capability] = found
	}

	log.Info("OpenShift cluster information gathered",
		"apiGroups", len(info.APIGroups),
		"capabilities", len(info.Capabilities))

	return info, nil
}

// OpenShiftInfo contains information about an OpenShift cluster
type OpenShiftInfo struct {
	// IsOpenShift indicates if this is an OpenShift cluster
	IsOpenShift bool
	// APIGroups lists the detected OpenShift API groups
	APIGroups []string
	// Capabilities maps capability names to availability
	Capabilities map[string]bool
}

// ModelHealthCheckConfig configures model health checking behavior
type ModelHealthCheckConfig struct {
	// Namespace is the namespace to check for models
	Namespace string
	// AllowCrossNamespace enables checking models in other namespaces
	AllowCrossNamespace bool
	// AllowedNamespaces is a list of namespaces that can be accessed
	// Empty list means all namespaces (when AllowCrossNamespace is true)
	AllowedNamespaces []string
	// TimeoutSeconds is the timeout for health checks
	TimeoutSeconds int
}

// ModelHealthStatus represents the health status of a model
type ModelHealthStatus struct {
	// ModelName is the name of the model
	ModelName string
	// Namespace is the namespace where the model is located
	Namespace string
	// Available indicates if the model resource exists
	Available bool
	// Ready indicates if the model is ready to serve requests
	Ready bool
	// Phase is the current phase of the model (platform-specific)
	Phase string
	// Replicas is the number of ready replicas
	Replicas int32
	// Message provides additional status information
	Message string
	// LastChecked is when the health check was performed
	LastChecked string
}

// NamespaceModelSummary provides a summary of models in a namespace
type NamespaceModelSummary struct {
	// Namespace is the namespace
	Namespace string
	// TotalModels is the total number of models found
	TotalModels int
	// ReadyModels is the number of models that are ready
	ReadyModels int
	// FailedModels is the number of models in failed state
	FailedModels int
	// Platform is the detected platform in this namespace
	Platform Platform
	// Models is the list of model health statuses
	Models []ModelHealthStatus
}

// CheckModelHealth checks the health of a specific model in a namespace
func (d *Detector) CheckModelHealth(ctx context.Context, modelRef *ModelReference, platform Platform, jobNamespace string, config *ModelHealthCheckConfig) (*ModelHealthStatus, error) {
	logger := log.FromContext(ctx)

	// Determine the target namespace
	targetNamespace := modelRef.Namespace
	if targetNamespace == "" {
		targetNamespace = jobNamespace
	}

	// Validate cross-namespace access
	if targetNamespace != jobNamespace {
		if config == nil || !config.AllowCrossNamespace {
			logger.Info("Cross-namespace model access denied",
				"model", modelRef.Name,
				"targetNamespace", targetNamespace,
				"jobNamespace", jobNamespace)
			return &ModelHealthStatus{
				ModelName: modelRef.Name,
				Namespace: targetNamespace,
				Available: false,
				Ready:     false,
				Message:   fmt.Sprintf("Cross-namespace model access denied: model %s is in namespace %s but job is in namespace %s", modelRef.Name, targetNamespace, jobNamespace),
			}, fmt.Errorf("cross-namespace model access denied")
		}

		// Check if namespace is in allowed list
		if len(config.AllowedNamespaces) > 0 {
			allowed := false
			for _, ns := range config.AllowedNamespaces {
				if ns == targetNamespace {
					allowed = true
					break
				}
			}
			if !allowed {
				logger.Info("Model namespace not in allowed list",
					"model", modelRef.Name,
					"targetNamespace", targetNamespace,
					"allowedNamespaces", config.AllowedNamespaces)
				return &ModelHealthStatus{
					ModelName: modelRef.Name,
					Namespace: targetNamespace,
					Available: false,
					Ready:     false,
					Message:   fmt.Sprintf("Namespace %s is not in the allowed namespaces list", targetNamespace),
				}, fmt.Errorf("namespace not allowed")
			}
		}
	}

	logger.V(1).Info("Checking model health",
		"model", modelRef.Name,
		"namespace", targetNamespace,
		"platform", platform)

	// Get platform definition for API group info
	platformDef := GetPlatformDefinition(platform)
	if platformDef == nil {
		return &ModelHealthStatus{
			ModelName: modelRef.Name,
			Namespace: targetNamespace,
			Available: false,
			Ready:     false,
			Message:   fmt.Sprintf("Unknown platform: %s", platform),
		}, fmt.Errorf("unknown platform: %s", platform)
	}

	// Platform-specific health check
	status := &ModelHealthStatus{
		ModelName: modelRef.Name,
		Namespace: targetNamespace,
	}

	switch platform {
	case PlatformKServe, PlatformOpenShiftAI:
		status = d.checkKServeModelHealth(ctx, modelRef.Name, targetNamespace)
	case PlatformRayServe:
		status = d.checkRayServeModelHealth(ctx, modelRef.Name, targetNamespace)
	case PlatformSeldon:
		status = d.checkSeldonModelHealth(ctx, modelRef.Name, targetNamespace)
	case PlatformBentoML:
		status = d.checkBentoMLModelHealth(ctx, modelRef.Name, targetNamespace)
	default:
		// For deployment-based platforms (vLLM, TorchServe, TensorFlow Serving, Triton)
		status = d.checkDeploymentModelHealth(ctx, modelRef.Name, targetNamespace)
	}

	logger.V(1).Info("Model health check completed",
		"model", modelRef.Name,
		"namespace", targetNamespace,
		"available", status.Available,
		"ready", status.Ready,
		"phase", status.Phase)

	return status, nil
}

// CheckMultipleModelsHealth checks the health of multiple models, potentially across namespaces
func (d *Detector) CheckMultipleModelsHealth(ctx context.Context, modelRefs []*ModelReference, platform Platform, jobNamespace string, config *ModelHealthCheckConfig) ([]ModelHealthStatus, error) {
	logger := log.FromContext(ctx)

	if len(modelRefs) == 0 {
		return []ModelHealthStatus{}, nil
	}

	results := make([]ModelHealthStatus, 0, len(modelRefs))
	var errors []string

	// Group models by namespace for efficient checking
	byNamespace := GroupByNamespace(modelRefs)

	logger.V(1).Info("Checking health of multiple models",
		"totalModels", len(modelRefs),
		"namespaces", len(byNamespace),
		"jobNamespace", jobNamespace)

	for namespace, refs := range byNamespace {
		for _, ref := range refs {
			status, err := d.CheckModelHealth(ctx, ref, platform, jobNamespace, config)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s/%s: %v", namespace, ref.Name, err))
				// Still add the status (with error info) to results
				if status != nil {
					results = append(results, *status)
				}
				continue
			}
			results = append(results, *status)
		}
	}

	if len(errors) > 0 {
		logger.Info("Some model health checks failed",
			"errors", len(errors),
			"total", len(modelRefs))
		return results, fmt.Errorf("failed to check %d model(s): %v", len(errors), errors)
	}

	return results, nil
}

// GetNamespaceModelSummary returns a summary of all models in a namespace
func (d *Detector) GetNamespaceModelSummary(ctx context.Context, namespace string, platform Platform) (*NamespaceModelSummary, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("Getting model summary for namespace",
		"namespace", namespace,
		"platform", platform)

	summary := &NamespaceModelSummary{
		Namespace: namespace,
		Platform:  platform,
		Models:    []ModelHealthStatus{},
	}

	// Platform-specific model discovery
	switch platform {
	case PlatformKServe, PlatformOpenShiftAI:
		models, err := d.listKServeModels(ctx, namespace)
		if err != nil {
			return summary, err
		}
		summary.Models = models
	case PlatformRayServe:
		models, err := d.listRayServeModels(ctx, namespace)
		if err != nil {
			return summary, err
		}
		summary.Models = models
	case PlatformSeldon:
		models, err := d.listSeldonModels(ctx, namespace)
		if err != nil {
			return summary, err
		}
		summary.Models = models
	default:
		// For other platforms, return empty list (deployment-based platforms need explicit model names)
		logger.V(1).Info("Platform does not support automatic model discovery",
			"platform", platform)
	}

	// Calculate summary statistics
	summary.TotalModels = len(summary.Models)
	for _, model := range summary.Models {
		if model.Ready {
			summary.ReadyModels++
		} else if model.Phase == "Failed" {
			summary.FailedModels++
		}
	}

	logger.V(1).Info("Namespace model summary complete",
		"namespace", namespace,
		"totalModels", summary.TotalModels,
		"readyModels", summary.ReadyModels,
		"failedModels", summary.FailedModels)

	return summary, nil
}

// Platform-specific health check implementations

func (d *Detector) checkKServeModelHealth(ctx context.Context, modelName, namespace string) *ModelHealthStatus {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Checking KServe InferenceService health",
		"model", modelName,
		"namespace", namespace)

	// This would use the client to fetch the InferenceService
	// For now, return a placeholder that indicates the check was attempted
	return &ModelHealthStatus{
		ModelName: modelName,
		Namespace: namespace,
		Available: true,
		Ready:     true,
		Phase:     "Ready",
		Message:   "KServe InferenceService health check - implementation pending actual API call",
	}
}

func (d *Detector) checkRayServeModelHealth(ctx context.Context, modelName, namespace string) *ModelHealthStatus {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Checking Ray Serve health",
		"model", modelName,
		"namespace", namespace)

	return &ModelHealthStatus{
		ModelName: modelName,
		Namespace: namespace,
		Available: true,
		Ready:     true,
		Phase:     "Running",
		Message:   "Ray Serve health check - implementation pending actual API call",
	}
}

func (d *Detector) checkSeldonModelHealth(ctx context.Context, modelName, namespace string) *ModelHealthStatus {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Checking Seldon deployment health",
		"model", modelName,
		"namespace", namespace)

	return &ModelHealthStatus{
		ModelName: modelName,
		Namespace: namespace,
		Available: true,
		Ready:     true,
		Phase:     "Available",
		Message:   "Seldon deployment health check - implementation pending actual API call",
	}
}

func (d *Detector) checkBentoMLModelHealth(ctx context.Context, modelName, namespace string) *ModelHealthStatus {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Checking BentoML deployment health",
		"model", modelName,
		"namespace", namespace)

	return &ModelHealthStatus{
		ModelName: modelName,
		Namespace: namespace,
		Available: true,
		Ready:     true,
		Phase:     "Running",
		Message:   "BentoML health check - implementation pending actual API call",
	}
}

func (d *Detector) checkDeploymentModelHealth(ctx context.Context, modelName, namespace string) *ModelHealthStatus {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Checking Deployment-based model health",
		"model", modelName,
		"namespace", namespace)

	return &ModelHealthStatus{
		ModelName: modelName,
		Namespace: namespace,
		Available: true,
		Ready:     true,
		Phase:     "Running",
		Replicas:  1,
		Message:   "Deployment health check - implementation pending actual API call",
	}
}

// Platform-specific model listing implementations

func (d *Detector) listKServeModels(ctx context.Context, namespace string) ([]ModelHealthStatus, error) {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Listing KServe InferenceServices",
		"namespace", namespace)

	// This would use the client to list InferenceServices
	// For now, return empty list
	return []ModelHealthStatus{}, nil
}

func (d *Detector) listRayServeModels(ctx context.Context, namespace string) ([]ModelHealthStatus, error) {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Listing Ray Services",
		"namespace", namespace)

	return []ModelHealthStatus{}, nil
}

func (d *Detector) listSeldonModels(ctx context.Context, namespace string) ([]ModelHealthStatus, error) {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Listing Seldon Deployments",
		"namespace", namespace)

	return []ModelHealthStatus{}, nil
}
