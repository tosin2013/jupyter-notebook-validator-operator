// Package platform provides model serving platform detection and validation
package platform

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ModelReference represents a reference to a model with namespace resolution
type ModelReference struct {
	// Name is the name of the model
	Name string
	// Namespace is the namespace where the model is located
	// Empty string means same namespace as the NotebookValidationJob
	Namespace string
	// OriginalRef is the original reference string before parsing
	OriginalRef string
}

// ModelResolutionConfig contains configuration for model namespace resolution
type ModelResolutionConfig struct {
	// AllowCrossNamespace enables cross-namespace model access
	// When false, models are always resolved to the job's namespace
	AllowCrossNamespace bool
	// DefaultNamespace is the fallback namespace when not specified
	DefaultNamespace string
	// AllowedNamespaces is a list of namespaces that can be accessed
	// Empty list means all namespaces (when AllowCrossNamespace is true)
	AllowedNamespaces []string
}

// ModelResolver handles namespace-aware model reference resolution
type ModelResolver struct {
	config ModelResolutionConfig
}

// NewModelResolver creates a new ModelResolver with the given configuration
func NewModelResolver(config ModelResolutionConfig) *ModelResolver {
	return &ModelResolver{
		config: config,
	}
}

// NewDefaultModelResolver creates a ModelResolver with default settings
// (namespace-scoped, no cross-namespace access)
func NewDefaultModelResolver(defaultNamespace string) *ModelResolver {
	return &ModelResolver{
		config: ModelResolutionConfig{
			AllowCrossNamespace: false,
			DefaultNamespace:    defaultNamespace,
			AllowedNamespaces:   []string{},
		},
	}
}

// ResolveModelReference parses a model reference string and resolves the namespace
// Supported formats:
//   - "model-name" - resolves to default namespace
//   - "namespace/model-name" - explicit namespace (requires AllowCrossNamespace)
func (r *ModelResolver) ResolveModelReference(ctx context.Context, modelRef string, jobNamespace string) (*ModelReference, error) {
	logger := log.FromContext(ctx)

	if modelRef == "" {
		return nil, fmt.Errorf("model reference cannot be empty")
	}

	// Determine the effective default namespace
	defaultNS := jobNamespace
	if defaultNS == "" {
		defaultNS = r.config.DefaultNamespace
	}

	// Parse the model reference
	ref := &ModelReference{
		OriginalRef: modelRef,
	}

	// Check if the reference contains a namespace prefix (namespace/model)
	if strings.Contains(modelRef, "/") {
		parts := strings.SplitN(modelRef, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid model reference format: %q (expected 'namespace/model' or 'model')", modelRef)
		}

		// Validate that the model name doesn't contain additional slashes (e.g., ns//model or ns/sub/model)
		if strings.Contains(parts[1], "/") || strings.HasPrefix(parts[1], "/") {
			return nil, fmt.Errorf("invalid model reference format: %q (model name cannot contain slashes)", modelRef)
		}

		ref.Namespace = parts[0]
		ref.Name = parts[1]

		// Check if cross-namespace access is allowed
		if ref.Namespace != jobNamespace {
			if !r.config.AllowCrossNamespace {
				logger.Info("Cross-namespace model access denied",
					"requestedNamespace", ref.Namespace,
					"jobNamespace", jobNamespace,
					"model", ref.Name)
				return nil, fmt.Errorf("cross-namespace model access is not allowed: model %q is in namespace %q but job is in namespace %q",
					ref.Name, ref.Namespace, jobNamespace)
			}

			// Check if the namespace is in the allowed list
			if len(r.config.AllowedNamespaces) > 0 {
				allowed := false
				for _, ns := range r.config.AllowedNamespaces {
					if ns == ref.Namespace {
						allowed = true
						break
					}
				}
				if !allowed {
					logger.Info("Model namespace not in allowed list",
						"requestedNamespace", ref.Namespace,
						"allowedNamespaces", r.config.AllowedNamespaces,
						"model", ref.Name)
					return nil, fmt.Errorf("namespace %q is not in the allowed namespaces list for model access", ref.Namespace)
				}
			}

			logger.V(1).Info("Cross-namespace model reference resolved",
				"model", ref.Name,
				"namespace", ref.Namespace,
				"jobNamespace", jobNamespace)
		}
	} else {
		// Simple model name without namespace prefix
		ref.Name = modelRef
		ref.Namespace = defaultNS
		logger.V(1).Info("Model reference resolved to job namespace",
			"model", ref.Name,
			"namespace", ref.Namespace)
	}

	return ref, nil
}

// ResolveModelReferences resolves multiple model references
func (r *ModelResolver) ResolveModelReferences(ctx context.Context, modelRefs []string, jobNamespace string) ([]*ModelReference, error) {
	logger := log.FromContext(ctx)

	if len(modelRefs) == 0 {
		return []*ModelReference{}, nil
	}

	resolved := make([]*ModelReference, 0, len(modelRefs))
	var errors []string

	for _, ref := range modelRefs {
		modelRef, err := r.ResolveModelReference(ctx, ref, jobNamespace)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", ref, err))
			continue
		}
		resolved = append(resolved, modelRef)
	}

	if len(errors) > 0 {
		logger.Error(nil, "Some model references failed to resolve",
			"errors", errors,
			"totalRefs", len(modelRefs),
			"resolved", len(resolved))
		return resolved, fmt.Errorf("failed to resolve %d model reference(s): %s", len(errors), strings.Join(errors, "; "))
	}

	logger.V(1).Info("All model references resolved",
		"count", len(resolved),
		"jobNamespace", jobNamespace)

	return resolved, nil
}

// GroupByNamespace groups model references by namespace for efficient batch operations
func GroupByNamespace(refs []*ModelReference) map[string][]*ModelReference {
	result := make(map[string][]*ModelReference)
	for _, ref := range refs {
		result[ref.Namespace] = append(result[ref.Namespace], ref)
	}
	return result
}

// GetUniqueNamespaces returns a list of unique namespaces from model references
func GetUniqueNamespaces(refs []*ModelReference) []string {
	namespaceMap := make(map[string]bool)
	for _, ref := range refs {
		namespaceMap[ref.Namespace] = true
	}

	namespaces := make([]string, 0, len(namespaceMap))
	for ns := range namespaceMap {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// FormatModelReference formats a ModelReference back to string representation
func (ref *ModelReference) FormatModelReference() string {
	if ref.Namespace != "" {
		return fmt.Sprintf("%s/%s", ref.Namespace, ref.Name)
	}
	return ref.Name
}

// String implements the Stringer interface
func (ref *ModelReference) String() string {
	return ref.FormatModelReference()
}

// IsCrossNamespace returns true if the model is in a different namespace than the job
func (ref *ModelReference) IsCrossNamespace(jobNamespace string) bool {
	return ref.Namespace != "" && ref.Namespace != jobNamespace
}
