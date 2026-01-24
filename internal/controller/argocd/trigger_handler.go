package argocd

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const (
	// AnnotationOnSuccessTrigger is the annotation key for post-success triggers
	AnnotationOnSuccessTrigger = "mlops.dev/on-success-trigger"
	// AnnotationRestartTimestamp is used for refresh action
	AnnotationRestartTimestamp = "mlops.dev/restart-timestamp"
)

// TriggerHandler handles resource triggering on job completion
type TriggerHandler struct {
	client.Client
}

// NewTriggerHandler creates a new trigger handler
func NewTriggerHandler(c client.Client) *TriggerHandler {
	return &TriggerHandler{Client: c}
}

// ExecuteTriggers parses and executes all triggers from the job annotations
func (h *TriggerHandler) ExecuteTriggers(ctx context.Context, job client.Object) error {
	logger := log.FromContext(ctx)

	annotations := job.GetAnnotations()
	if annotations == nil {
		return nil
	}

	triggerYAML := annotations[AnnotationOnSuccessTrigger]
	if triggerYAML == "" {
		// No triggers configured
		return nil
	}

	// Parse triggers from YAML
	triggers, err := parseTriggers(triggerYAML)
	if err != nil {
		return fmt.Errorf("failed to parse triggers: %w", err)
	}

	logger.Info("Executing resource triggers", "count", len(triggers))

	// Execute each trigger
	for i, trigger := range triggers {
		logger.V(1).Info("Executing trigger",
			"index", i,
			"apiVersion", trigger.APIVersion,
			"kind", trigger.Kind,
			"name", trigger.Name,
			"namespace", trigger.Namespace,
			"action", trigger.Action)

		if err := h.executeTrigger(ctx, job, trigger); err != nil {
			logger.Error(err, "Failed to execute trigger",
				"index", i,
				"kind", trigger.Kind,
				"name", trigger.Name)
			// Continue with other triggers even if one fails
			continue
		}

		logger.Info("Successfully executed trigger",
			"index", i,
			"kind", trigger.Kind,
			"name", trigger.Name,
			"action", trigger.Action)
	}

	return nil
}

// parseTriggers parses triggers from YAML annotation
func parseTriggers(yamlStr string) ([]ResourceTrigger, error) {
	var triggers []ResourceTrigger

	// Parse YAML list
	if err := yaml.Unmarshal([]byte(yamlStr), &triggers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal triggers YAML: %w", err)
	}

	// Validate triggers
	for i, trigger := range triggers {
		if trigger.APIVersion == "" {
			return nil, fmt.Errorf("trigger[%d]: apiVersion is required", i)
		}
		if trigger.Kind == "" {
			return nil, fmt.Errorf("trigger[%d]: kind is required", i)
		}
		if trigger.Name == "" {
			return nil, fmt.Errorf("trigger[%d]: name is required", i)
		}
		if trigger.Action == "" {
			return nil, fmt.Errorf("trigger[%d]: action is required", i)
		}

		// Validate action
		switch trigger.Action {
		case ActionRestart, ActionSync, ActionRefresh:
			// Valid actions
		default:
			return nil, fmt.Errorf("trigger[%d]: invalid action '%s' (must be restart, sync, or refresh)", i, trigger.Action)
		}
	}

	return triggers, nil
}

// executeTrigger executes a single trigger
func (h *TriggerHandler) executeTrigger(ctx context.Context, job client.Object, trigger ResourceTrigger) error {
	// Determine namespace (default to job namespace)
	namespace := trigger.Namespace
	if namespace == "" {
		namespace = job.GetNamespace()
	}

	// Execute action based on trigger type
	switch trigger.Action {
	case ActionRestart:
		return h.restartResource(ctx, trigger, namespace)
	case ActionSync:
		return h.syncArgoApplication(ctx, trigger, namespace)
	case ActionRefresh:
		return h.refreshResource(ctx, trigger, namespace)
	default:
		return fmt.Errorf("unknown action: %s", trigger.Action)
	}
}

// restartResource restarts a resource by deleting its pods
func (h *TriggerHandler) restartResource(ctx context.Context, trigger ResourceTrigger, namespace string) error {
	logger := log.FromContext(ctx)

	// Build GVR from APIVersion and Kind
	gv, err := schema.ParseGroupVersion(trigger.APIVersion)
	if err != nil {
		return fmt.Errorf("invalid apiVersion: %w", err)
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    trigger.Kind,
	}

	// Handle different resource types
	switch {
	case gvk.Group == "serving.kserve.io" && gvk.Kind == "InferenceService":
		// KServe InferenceService: delete predictor pods
		return h.restartKServeInferenceService(ctx, trigger.Name, namespace)

	case gvk.Group == "apps" && gvk.Kind == "Deployment":
		// Kubernetes Deployment: delete pods
		return h.restartDeployment(ctx, trigger.Name, namespace)

	default:
		// Generic resource: try to find and delete pods by owner reference
		logger.Info("Generic resource restart - attempting to find pods by owner reference",
			"kind", trigger.Kind,
			"name", trigger.Name)
		return h.restartGenericResource(ctx, trigger, namespace)
	}
}

// restartKServeInferenceService restarts a KServe InferenceService by deleting predictor pods
func (h *TriggerHandler) restartKServeInferenceService(ctx context.Context, name, namespace string) error {
	logger := log.FromContext(ctx)

	// KServe InferenceServices create pods with label: serving.kserve.io/inferenceservice=<name>
	labelSelector := labels.SelectorFromSet(map[string]string{
		"serving.kserve.io/inferenceservice": name,
	})

	podList := &corev1.PodList{}
	if err := h.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		logger.Info("No pods found for InferenceService", "name", name, "namespace", namespace)
		return nil
	}

	logger.Info("Deleting pods to restart InferenceService", "count", len(podList.Items), "name", name)

	// Delete all pods
	for _, pod := range podList.Items {
		if err := h.Delete(ctx, &pod); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
		}
		logger.V(1).Info("Deleted pod", "pod", pod.Name)
	}

	return nil
}

// restartDeployment restarts a Deployment by deleting its pods
func (h *TriggerHandler) restartDeployment(ctx context.Context, name, namespace string) error {
	logger := log.FromContext(ctx)

	// Get the Deployment
	deployment := &appsv1.Deployment{}
	if err := h.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Deployment not found", "name", name, "namespace", namespace)
			return nil
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get pods owned by this deployment
	labelSelector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
	podList := &corev1.PodList{}
	if err := h.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		logger.Info("No pods found for Deployment", "name", name, "namespace", namespace)
		return nil
	}

	logger.Info("Deleting pods to restart Deployment", "count", len(podList.Items), "name", name)

	// Delete all pods
	for _, pod := range podList.Items {
		if err := h.Delete(ctx, &pod); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
		}
		logger.V(1).Info("Deleted pod", "pod", pod.Name)
	}

	return nil
}

// restartGenericResource attempts to restart a generic resource by finding and deleting its pods
func (h *TriggerHandler) restartGenericResource(ctx context.Context, trigger ResourceTrigger, namespace string) error {
	logger := log.FromContext(ctx)

	// Get the resource as unstructured
	gv, err := parseGroupVersion(trigger.APIVersion)
	if err != nil {
		return fmt.Errorf("invalid apiVersion: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    trigger.Kind,
	})

	if err := h.Get(ctx, types.NamespacedName{Name: trigger.Name, Namespace: namespace}, obj); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found", "kind", trigger.Kind, "name", trigger.Name)
			return nil
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Try to find pods by owner reference
	podList := &corev1.PodList{}
	if err := h.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Find pods owned by this resource
	var podsToDelete []corev1.Pod
	for _, pod := range podList.Items {
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.UID == obj.GetUID() {
				podsToDelete = append(podsToDelete, pod)
				break
			}
		}
	}

	if len(podsToDelete) == 0 {
		logger.Info("No pods found for resource", "kind", trigger.Kind, "name", trigger.Name)
		return nil
	}

	logger.Info("Deleting pods to restart resource", "count", len(podsToDelete), "kind", trigger.Kind, "name", trigger.Name)

	// Delete all pods
	for _, pod := range podsToDelete {
		if err := h.Delete(ctx, &pod); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
		}
		logger.V(1).Info("Deleted pod", "pod", pod.Name)
	}

	return nil
}

// syncArgoApplication triggers an ArgoCD Application sync
func (h *TriggerHandler) syncArgoApplication(ctx context.Context, trigger ResourceTrigger, namespace string) error {
	logger := log.FromContext(ctx)

	// ArgoCD Applications are in the argoproj.io API group
	// The namespace is typically "argocd" but can be configured
	appNamespace := namespace
	if appNamespace == "" {
		appNamespace = "argocd"
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	if err := h.Get(ctx, types.NamespacedName{Name: trigger.Name, Namespace: appNamespace}, obj); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ArgoCD Application not found", "name", trigger.Name, "namespace", appNamespace)
			return nil
		}
		return fmt.Errorf("failed to get ArgoCD Application: %w", err)
	}

	// Trigger sync by adding/updating annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// ArgoCD sync annotation
	annotations["argocd.argoproj.io/refresh"] = "normal"
	annotations["argocd.argoproj.io/sync-wave"] = "0" // Force immediate sync

	obj.SetAnnotations(annotations)

	if err := h.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to trigger ArgoCD sync: %w", err)
	}

	logger.Info("Triggered ArgoCD Application sync", "name", trigger.Name, "namespace", appNamespace)
	return nil
}

// refreshResource adds a restart annotation to force resource refresh
func (h *TriggerHandler) refreshResource(ctx context.Context, trigger ResourceTrigger, namespace string) error {
	logger := log.FromContext(ctx)

	// Build GVR
	gv, err := schema.ParseGroupVersion(trigger.APIVersion)
	if err != nil {
		return fmt.Errorf("invalid apiVersion: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    trigger.Kind,
	})

	if err := h.Get(ctx, types.NamespacedName{Name: trigger.Name, Namespace: namespace}, obj); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found", "kind", trigger.Kind, "name", trigger.Name)
			return nil
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Add restart timestamp annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[AnnotationRestartTimestamp] = time.Now().Format(time.RFC3339)
	obj.SetAnnotations(annotations)

	if err := h.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to refresh resource: %w", err)
	}

	logger.Info("Refreshed resource", "kind", trigger.Kind, "name", trigger.Name)
	return nil
}

// Helper function to parse group version
func parseGroupVersion(apiVersion string) (schema.GroupVersion, error) {
	if apiVersion == "" {
		return schema.GroupVersion{}, fmt.Errorf("apiVersion cannot be empty")
	}
	parts := strings.Split(apiVersion, "/")
	if len(parts) == 1 {
		// Core API group (e.g., "v1")
		return schema.GroupVersion{Version: parts[0]}, nil
	} else if len(parts) == 2 {
		// Group/Version format (e.g., "apps/v1")
		return schema.ParseGroupVersion(apiVersion)
	}
	return schema.GroupVersion{}, fmt.Errorf("invalid apiVersion format: %s", apiVersion)
}
