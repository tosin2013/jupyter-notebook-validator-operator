package argocd

// ResourceTrigger defines a resource to trigger on job completion
// This is parsed from the mlops.dev/on-success-trigger annotation
type ResourceTrigger struct {
	// APIVersion is the Kubernetes API version (e.g., "serving.kserve.io/v1beta1", "apps/v1")
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	// Kind is the Kubernetes resource kind (e.g., "InferenceService", "Deployment")
	Kind string `yaml:"kind" json:"kind"`
	// Name is the name of the resource
	Name string `yaml:"name" json:"name"`
	// Namespace is the namespace of the resource (optional, defaults to job namespace)
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	// Action is the action to perform: "restart", "sync", or "refresh"
	Action string `yaml:"action" json:"action"`
}

// TriggerAction constants
const (
	// ActionRestart deletes pods to trigger reload (for KServe, Deployments, etc.)
	ActionRestart = "restart"
	// ActionSync triggers ArgoCD Application sync
	ActionSync = "sync"
	// ActionRefresh adds annotation to force resource refresh
	ActionRefresh = "refresh"
)
