package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

const (
	// AnnotationNotebookStatus is the annotation key for aggregated notebook status
	AnnotationNotebookStatus = "mlops.dev/notebook-status"
	// AnnotationNotebookSummary is a human-readable summary
	AnnotationNotebookSummary = "mlops.dev/notebook-summary"
	// LabelPartOf is the label key for identifying parent Application
	LabelPartOf = "app.kubernetes.io/part-of"
	// DebounceInterval is the minimum time between status updates
	DebounceInterval = 1 * time.Minute
)

// NotebookStatus represents aggregated status of NotebookValidationJobs
type NotebookStatus struct {
	Total      int      `json:"total"`
	Succeeded  int      `json:"succeeded"`
	Failed     int      `json:"failed"`
	Running    int      `json:"running"`
	LastUpdate string   `json:"lastUpdate"`
	FailedJobs []string `json:"failedJobs,omitempty"`
}

// StatusAggregator aggregates NotebookValidationJob statuses and updates ArgoCD Applications
type StatusAggregator struct {
	client.Client
	lastUpdateTimes map[string]time.Time
	mu              sync.Mutex
}

// NewStatusAggregator creates a new status aggregator
func NewStatusAggregator(c client.Client) *StatusAggregator {
	return &StatusAggregator{
		Client:          c,
		lastUpdateTimes: make(map[string]time.Time),
	}
}

// UpdateApplicationStatus updates the ArgoCD Application with aggregated notebook status
// This is debounced to prevent excessive updates (max 1 update per minute per Application)
func (a *StatusAggregator) UpdateApplicationStatus(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// List all NotebookValidationJobs in the namespace
	jobList := &mlopsv1alpha1.NotebookValidationJobList{}
	if err := a.List(ctx, jobList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list NotebookValidationJobs: %w", err)
	}

	// Find parent ArgoCD Application via part-of label
	appName := a.findParentApplication(jobList.Items)
	if appName == "" {
		// No parent Application found - this is OK
		logger.V(1).Info("No parent ArgoCD Application found", "namespace", namespace)
		return nil
	}

	// Check debounce
	appKey := fmt.Sprintf("%s/%s", namespace, appName)
	a.mu.Lock()
	lastUpdate, exists := a.lastUpdateTimes[appKey]
	if exists && time.Since(lastUpdate) < DebounceInterval {
		a.mu.Unlock()
		logger.V(1).Info("Skipping status update (debounced)", "app", appName, "namespace", namespace)
		return nil
	}
	a.mu.Unlock()

	// Aggregate status
	status := a.aggregateStatus(jobList.Items)

	// Update Application annotation
	if err := a.updateApplicationAnnotation(ctx, appName, namespace, status); err != nil {
		return fmt.Errorf("failed to update Application annotation: %w", err)
	}

	// Update last update time
	a.mu.Lock()
	a.lastUpdateTimes[appKey] = time.Now()
	a.mu.Unlock()

	logger.Info("Updated ArgoCD Application status",
		"app", appName,
		"namespace", namespace,
		"total", status.Total,
		"succeeded", status.Succeeded,
		"failed", status.Failed)

	return nil
}

// findParentApplication finds the parent ArgoCD Application from job labels
func (a *StatusAggregator) findParentApplication(jobs []mlopsv1alpha1.NotebookValidationJob) string {
	for _, job := range jobs {
		if partOf := job.Labels[LabelPartOf]; partOf != "" {
			return partOf
		}
	}
	return ""
}

// aggregateStatus aggregates status from all NotebookValidationJobs
func (a *StatusAggregator) aggregateStatus(jobs []mlopsv1alpha1.NotebookValidationJob) NotebookStatus {
	status := NotebookStatus{
		Total:      len(jobs),
		Succeeded:  0,
		Failed:     0,
		Running:    0,
		LastUpdate: time.Now().Format(time.RFC3339),
		FailedJobs: []string{},
	}

	for _, job := range jobs {
		switch job.Status.Phase {
		case "Succeeded":
			status.Succeeded++
		case "Failed":
			status.Failed++
			status.FailedJobs = append(status.FailedJobs, job.Name)
		case "ValidationRunning", "Building", "BuildComplete", "Initializing", "Pending", "Running":
			status.Running++
		}
	}

	return status
}

// updateApplicationAnnotation updates the ArgoCD Application with aggregated status
func (a *StatusAggregator) updateApplicationAnnotation(ctx context.Context, appName, namespace string, status NotebookStatus) error {
	logger := log.FromContext(ctx)

	// Try to find Application in argocd namespace first, then in the same namespace
	appNamespaces := []string{"argocd", namespace}

	var app *unstructured.Unstructured
	var err error

	for _, ns := range appNamespaces {
		app = &unstructured.Unstructured{}
		app.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "argoproj.io",
			Version: "v1alpha1",
			Kind:    "Application",
		})

		err = a.Get(ctx, types.NamespacedName{Name: appName, Namespace: ns}, app)
		if err == nil {
			logger.V(1).Info("Found ArgoCD Application", "name", appName, "namespace", ns)
			break
		}
		if !client.IgnoreNotFound(err) {
			return fmt.Errorf("failed to get Application: %w", err)
		}
	}

	if err != nil {
		// Application not found - this is OK
		logger.V(1).Info("ArgoCD Application not found", "name", appName)
		return nil
	}

	// Update annotations
	annotations := app.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Serialize status to JSON
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	annotations[AnnotationNotebookStatus] = string(statusJSON)
	annotations[AnnotationNotebookSummary] = status.Summary()

	app.SetAnnotations(annotations)

	// Update the Application
	if err := a.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update Application: %w", err)
	}

	return nil
}

// Summary returns a human-readable summary of the status
func (s NotebookStatus) Summary() string {
	if s.Total == 0 {
		return "No notebook validation jobs"
	}

	successRate := float64(s.Succeeded) / float64(s.Total) * 100
	return fmt.Sprintf("%d/%d succeeded (%.1f%% success rate)", s.Succeeded, s.Total, successRate)
}
