# ADR 010: Observability and Monitoring Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must provide comprehensive observability to enable:

1. **Debugging**: Troubleshoot issues in development and production
2. **Monitoring**: Track operator health and performance
3. **Alerting**: Detect and respond to failures quickly
4. **Compliance**: Audit trails for security and regulatory requirements
5. **User Experience**: Provide clear status and error messages

### PRD Requirements

**US-5**: "View structured, cell-by-cell results of validation runs, including error messages and output diffs"

**US-7**: "Update status of NotebookValidationJob with final result and detailed conditions"

**Section 6**: "Report errors clearly in CR status" for all edge cases

### Observability Pillars

The three pillars of observability:

1. **Logs**: Structured events for debugging and audit trails
2. **Metrics**: Time-series data for monitoring and alerting
3. **Traces**: Distributed request tracking (future consideration)

### Technical Challenges

1. **Log Volume**: High-frequency reconciliation can generate excessive logs
2. **Metric Cardinality**: Too many label combinations can overwhelm Prometheus
3. **Status Complexity**: CRD status must be both machine and human-readable
4. **Secret Leakage**: Logs must never contain credentials or sensitive data
5. **Performance**: Observability overhead must be minimal

## Decision

We will implement a **Three-Pillar Observability Strategy** with structured logging, Prometheus metrics, and comprehensive status conditions.

### Pillar 1: Structured Logging

**Framework**: `sigs.k8s.io/controller-runtime/pkg/log` (logr interface)
**Format**: JSON structured logging
**Levels**: Error, Info, Debug (V-levels)

#### Log Levels

```go
// Error: Unexpected errors requiring attention
log.Error(err, "Failed to create validation pod",
    "namespace", req.Namespace,
    "name", req.Name,
)

// Info: Important state changes and milestones
log.Info("Validation job completed successfully",
    "namespace", req.Namespace,
    "name", req.Name,
    "duration", duration.Seconds(),
)

// Debug (V=1): Detailed operational information
log.V(1).Info("Fetching notebook from Git",
    "url", sanitizeURL(gitURL),
    "ref", gitRef,
)
```

#### Structured Log Format

```json
{
  "level": "info",
  "ts": "2025-11-07T20:45:00.123Z",
  "logger": "controller.notebookvalidationjob",
  "msg": "Validation job completed successfully",
  "namespace": "mlops",
  "name": "validate-model-training",
  "duration": 45.2,
  "phase": "Succeeded",
  "cellsExecuted": 12,
  "cellsFailed": 0
}
```

#### Log Sanitization

```go
// pkg/logging/sanitize.go
package logging

import (
    "net/url"
    "strings"
)

// SanitizeURL removes credentials from URLs for logging
func SanitizeURL(rawURL string) string {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "[invalid-url]"
    }
    
    // Remove user info (credentials)
    u.User = nil
    
    return u.String()
}

// SanitizeError removes sensitive information from error messages
func SanitizeError(err error, sensitiveStrings ...string) error {
    if err == nil {
        return nil
    }
    
    msg := err.Error()
    for _, sensitive := range sensitiveStrings {
        if sensitive != "" {
            msg = strings.ReplaceAll(msg, sensitive, "[REDACTED]")
        }
    }
    
    return fmt.Errorf("%s", msg)
}
```

### Pillar 2: Prometheus Metrics

**Framework**: `sigs.k8s.io/controller-runtime/pkg/metrics`
**Exposition**: `/metrics` endpoint on port 8080
**Format**: Prometheus text format

#### Core Metrics

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    // Reconciliation metrics
    ReconciliationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "notebook_validation_reconciliation_duration_seconds",
            Help: "Duration of NotebookValidationJob reconciliation in seconds",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
        },
        []string{"namespace", "result"}, // result: success, error, requeue
    )

    // Validation job metrics
    ValidationJobsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "notebook_validation_jobs_total",
            Help: "Total number of notebook validation jobs",
        },
        []string{"namespace", "phase"}, // phase: Pending, Running, Succeeded, Failed
    )

    ValidationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "notebook_validation_duration_seconds",
            Help: "Duration of notebook validation execution in seconds",
            Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800},
        },
        []string{"namespace", "result"}, // result: success, failure, timeout
    )

    // Cell execution metrics
    CellsExecutedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "notebook_cells_executed_total",
            Help: "Total number of notebook cells executed",
        },
        []string{"namespace", "status"}, // status: success, failure
    )

    // Git operation metrics
    GitCloneDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "notebook_git_clone_duration_seconds",
            Help: "Duration of Git clone operations in seconds",
            Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
        },
        []string{"result"}, // result: success, failure
    )

    // Queue metrics
    WorkQueueDepth = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "notebook_validation_workqueue_depth",
            Help: "Current depth of the NotebookValidationJob work queue",
        },
    )

    // Resource metrics
    ActiveValidationPods = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "notebook_validation_active_pods",
            Help: "Number of active validation pods",
        },
        []string{"namespace"},
    )
)

func init() {
    // Register metrics with controller-runtime
    metrics.Registry.MustRegister(
        ReconciliationDuration,
        ValidationJobsTotal,
        ValidationDuration,
        CellsExecutedTotal,
        GitCloneDuration,
        WorkQueueDepth,
        ActiveValidationPods,
    )
}
```

#### Metric Usage in Controller

```go
// controllers/notebookvalidationjob_controller.go
func (r *NotebookValidationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    start := time.Now()
    log := log.FromContext(ctx)

    defer func() {
        duration := time.Since(start).Seconds()
        result := "success"
        if err != nil {
            result = "error"
        }
        metrics.ReconciliationDuration.WithLabelValues(req.Namespace, result).Observe(duration)
    }()

    // ... reconciliation logic ...

    // Update metrics
    metrics.ValidationJobsTotal.WithLabelValues(req.Namespace, job.Status.Phase).Inc()
    metrics.ActiveValidationPods.WithLabelValues(req.Namespace).Set(float64(activePods))

    return ctrl.Result{}, nil
}
```

### Pillar 3: Status Conditions

**Framework**: Kubernetes Conditions (metav1.Condition)
**Purpose**: Machine-readable status for automation, human-readable messages for users

#### Status Structure

```go
// api/v1alpha1/notebookvalidationjob_types.go
type NotebookValidationJobStatus struct {
    // Phase represents the current phase of the validation job
    // +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
    Phase string `json:"phase,omitempty"`

    // Conditions represent the latest available observations of the job's state
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // StartTime is when the validation started
    StartTime *metav1.Time `json:"startTime,omitempty"`

    // CompletionTime is when the validation completed
    CompletionTime *metav1.Time `json:"completionTime,omitempty"`

    // Results contains cell-by-cell execution results
    Results []CellResult `json:"results,omitempty"`

    // ValidationPodName is the name of the pod executing the validation
    ValidationPodName string `json:"validationPodName,omitempty"`

    // Message provides a human-readable summary
    Message string `json:"message,omitempty"`
}

type CellResult struct {
    // CellIndex is the zero-based index of the cell
    CellIndex int `json:"cellIndex"`

    // Status is the execution status of the cell
    // +kubebuilder:validation:Enum=Success;Failure;Skipped
    Status string `json:"status"`

    // ExecutionTime is how long the cell took to execute
    ExecutionTime *metav1.Duration `json:"executionTime,omitempty"`

    // Output is the cell's stdout/stderr (truncated if too long)
    Output string `json:"output,omitempty"`

    // ErrorMessage is the error message if the cell failed
    ErrorMessage string `json:"errorMessage,omitempty"`
}
```

#### Condition Types

```go
const (
    // ConditionTypeReady indicates the job is ready to execute
    ConditionTypeReady = "Ready"

    // ConditionTypeGitCloned indicates Git repository was cloned successfully
    ConditionTypeGitCloned = "GitCloned"

    // ConditionTypeValidationComplete indicates validation execution completed
    ConditionTypeValidationComplete = "ValidationComplete"

    // ConditionTypeGoldenComparison indicates golden notebook comparison completed
    ConditionTypeGoldenComparison = "GoldenComparison"
)

const (
    ReasonGitCloneSuccess = "GitCloneSuccess"
    ReasonGitCloneFailed  = "GitCloneFailed"
    ReasonPodCreated      = "PodCreated"
    ReasonPodFailed       = "PodFailed"
    ReasonValidationSuccess = "ValidationSuccess"
    ReasonValidationFailed  = "ValidationFailed"
    ReasonTimeout         = "Timeout"
)
```

#### Setting Conditions

```go
// pkg/conditions/conditions.go
package conditions

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetCondition updates or adds a condition to the status
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
    now := metav1.Now()
    
    // Find existing condition
    for i, condition := range *conditions {
        if condition.Type == conditionType {
            // Update existing condition
            (*conditions)[i].Status = status
            (*conditions)[i].Reason = reason
            (*conditions)[i].Message = message
            (*conditions)[i].LastTransitionTime = now
            return
        }
    }
    
    // Add new condition
    *conditions = append(*conditions, metav1.Condition{
        Type:               conditionType,
        Status:             status,
        Reason:             reason,
        Message:            message,
        LastTransitionTime: now,
        ObservedGeneration: 0, // Set by controller
    })
}
```

#### Example Status

```yaml
status:
  phase: Succeeded
  startTime: "2025-11-07T20:45:00Z"
  completionTime: "2025-11-07T20:45:45Z"
  validationPodName: validate-model-training-abc123
  message: "Validation completed successfully. All 12 cells executed without errors."
  
  conditions:
    - type: Ready
      status: "True"
      reason: PodCreated
      message: "Validation pod created successfully"
      lastTransitionTime: "2025-11-07T20:45:00Z"
    
    - type: GitCloned
      status: "True"
      reason: GitCloneSuccess
      message: "Successfully cloned repository from https://github.com/org/repo.git"
      lastTransitionTime: "2025-11-07T20:45:05Z"
    
    - type: ValidationComplete
      status: "True"
      reason: ValidationSuccess
      message: "All 12 cells executed successfully in 40.2 seconds"
      lastTransitionTime: "2025-11-07T20:45:45Z"
    
    - type: GoldenComparison
      status: "True"
      reason: OutputsMatch
      message: "Notebook outputs match golden version"
      lastTransitionTime: "2025-11-07T20:45:45Z"
  
  results:
    - cellIndex: 0
      status: Success
      executionTime: 0.5s
      output: "Loaded dataset: 150 samples"
    
    - cellIndex: 1
      status: Success
      executionTime: 2.3s
      output: "Model training complete. Accuracy: 0.95"
    
    # ... more cells ...
```

## Implementation Notes

### ServiceMonitor for Prometheus

```yaml
# config/prometheus/monitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-validator-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Jupyter Notebook Validator Operator",
    "panels": [
      {
        "title": "Validation Job Success Rate",
        "targets": [
          {
            "expr": "rate(notebook_validation_jobs_total{phase=\"Succeeded\"}[5m]) / rate(notebook_validation_jobs_total[5m])"
          }
        ]
      },
      {
        "title": "Validation Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(notebook_validation_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Active Validation Pods",
        "targets": [
          {
            "expr": "sum(notebook_validation_active_pods)"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
# config/prometheus/alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: jupyter-notebook-validator-alerts
spec:
  groups:
    - name: jupyter-validator
      interval: 30s
      rules:
        - alert: HighValidationFailureRate
          expr: |
            rate(notebook_validation_jobs_total{phase="Failed"}[5m]) / 
            rate(notebook_validation_jobs_total[5m]) > 0.2
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High validation failure rate"
            description: "More than 20% of validation jobs are failing"
        
        - alert: ValidationJobStuck
          expr: |
            time() - max(notebook_validation_duration_seconds) > 1800
          for: 10m
          labels:
            severity: critical
          annotations:
            summary: "Validation job stuck"
            description: "A validation job has been running for more than 30 minutes"
```

## Consequences

### Positive
- **Comprehensive Visibility**: Logs, metrics, and status provide full observability
- **Production Ready**: Prometheus integration enables monitoring and alerting
- **User-Friendly**: Status conditions provide clear, actionable information
- **Debuggable**: Structured logs enable efficient troubleshooting
- **Secure**: Log sanitization prevents credential leakage

### Negative
- **Complexity**: Three observability pillars require maintenance
- **Storage**: Logs and metrics consume storage resources
- **Cardinality**: High-cardinality metrics can impact Prometheus performance

### Neutral
- **Standard Patterns**: Uses industry-standard observability tools
- **Extensible**: Can add OpenTelemetry tracing in future

## References

- [Operator SDK Observability Best Practices](https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/)
- [controller-runtime Logging](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/log)
- [Prometheus Operator](https://prometheus-operator.dev/)
- [Kubernetes API Conventions - Conditions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)

## Related ADRs

- ADR 009: Secret Management (log sanitization for credentials)
- ADR 011: Error Handling (error reporting in status)
- ADR 016: Performance and Scalability (metrics for performance monitoring)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial observability strategy |

