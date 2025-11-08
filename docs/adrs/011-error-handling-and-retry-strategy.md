# ADR 011: Error Handling and Retry Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must handle errors gracefully and retry transient failures automatically. This is critical for:

1. **Reliability**: Recover from transient failures without manual intervention
2. **User Experience**: Provide clear error messages and actionable guidance
3. **Resource Efficiency**: Avoid infinite retry loops and resource exhaustion
4. **Observability**: Track error patterns for debugging and alerting

### PRD Requirements

**Section 6 (Edge Cases)**: The operator must handle:
- Invalid Git URLs or missing files
- Pod failures (OOMKilled, ImagePullBackOff, CrashLoopBackOff)
- Notebook execution timeouts
- Network failures during Git clone
- Insufficient RBAC permissions
- Resource quota exceeded

**US-7**: "Update status of NotebookValidationJob with final result and detailed conditions"

### Technical Challenges

1. **Error Classification**: Distinguish transient vs. terminal errors
2. **Retry Limits**: Prevent infinite retry loops
3. **Backoff Strategy**: Balance responsiveness with resource usage
4. **Status Reporting**: Provide actionable error messages to users
5. **Reconciliation Complexity**: Handle errors at multiple stages (Git clone, pod creation, execution, comparison)

### Controller-Runtime Behavior

The `controller-runtime` library provides built-in retry mechanisms:

```go
// Returning an error triggers exponential backoff retry
return ctrl.Result{}, err

// Returning Result with Requeue triggers immediate retry
return ctrl.Result{Requeue: true}, nil

// Returning Result with RequeueAfter schedules retry after duration
return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil

// Returning empty Result with no error stops reconciliation
return ctrl.Result{}, nil
```

**Default Backoff**: 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.28s, 2.56s, 5.12s, 10.24s, 20.48s, 40.96s, 81.92s, 163.84s, 327.68s, 655.36s (max ~11 minutes)

## Decision

We will implement a **Tiered Error Handling Strategy** with three error categories: Transient, Retriable, and Terminal.

### Error Classification

#### Tier 1: Transient Errors (Automatic Retry with Backoff)
**Definition**: Temporary failures that are likely to resolve on their own

**Examples**:
- Network timeouts during Git clone
- Kubernetes API server unavailable
- Pod scheduling delays (insufficient resources)
- Image pull rate limiting

**Handling**: Return error to trigger controller-runtime's exponential backoff
```go
return ctrl.Result{}, fmt.Errorf("transient error: %w", err)
```

**Max Retries**: Unlimited (controller-runtime handles backoff)
**Status**: Update condition with `Reason: "Retrying"`, `Status: "False"`

#### Tier 2: Retriable Errors (Manual Retry with Limits)
**Definition**: Failures that may resolve with retry, but require tracking

**Examples**:
- Git authentication failures (credentials may be updated)
- Pod failures (OOMKilled, may need resource adjustment)
- Notebook execution errors (may be intermittent)
- Timeout during execution (may succeed with longer timeout)

**Handling**: Track retry count in status, requeue with delay
```go
if job.Status.RetryCount < maxRetries {
    job.Status.RetryCount++
    return ctrl.Result{RequeueAfter: calculateBackoff(job.Status.RetryCount)}, nil
}
// Max retries exceeded, mark as terminal
return ctrl.Result{}, nil
```

**Max Retries**: 3 attempts (configurable)
**Backoff**: 1m, 2m, 5m
**Status**: Update condition with `Reason: "RetryLimitExceeded"` after max retries

#### Tier 3: Terminal Errors (No Retry)
**Definition**: Permanent failures that cannot be resolved by retry

**Examples**:
- Invalid Git URL (malformed)
- Git repository not found (404)
- Missing notebook file in repository
- Invalid CRD spec (validation errors)
- RBAC permission denied (operator lacks permissions)
- Notebook syntax errors (invalid Python)

**Handling**: Update status to Failed, do not requeue
```go
return ctrl.Result{}, nil // No retry
```

**Status**: Update condition with `Reason: "ValidationFailed"`, `Status: "False"`, detailed error message

### Error Handling Implementation

```go
// pkg/errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

// ErrorType represents the category of error
type ErrorType string

const (
    ErrorTypeTransient ErrorType = "Transient"
    ErrorTypeRetriable ErrorType = "Retriable"
    ErrorTypeTerminal  ErrorType = "Terminal"
)

// ValidationError wraps an error with classification
type ValidationError struct {
    Type    ErrorType
    Reason  string
    Message string
    Err     error
}

func (e *ValidationError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ValidationError) Unwrap() error {
    return e.Err
}

// NewTransientError creates a transient error
func NewTransientError(reason, message string, err error) *ValidationError {
    return &ValidationError{
        Type:    ErrorTypeTransient,
        Reason:  reason,
        Message: message,
        Err:     err,
    }
}

// NewRetriableError creates a retriable error
func NewRetriableError(reason, message string, err error) *ValidationError {
    return &ValidationError{
        Type:    ErrorTypeRetriable,
        Reason:  reason,
        Message: message,
        Err:     err,
    }
}

// NewTerminalError creates a terminal error
func NewTerminalError(reason, message string, err error) *ValidationError {
    return &ValidationError{
        Type:    ErrorTypeTerminal,
        Reason:  reason,
        Message: message,
        Err:     err,
    }
}

// ClassifyError determines the error type from a generic error
func ClassifyError(err error) ErrorType {
    var validationErr *ValidationError
    if errors.As(err, &validationErr) {
        return validationErr.Type
    }

    // Classify common Kubernetes errors
    if apierrors.IsNotFound(err) {
        return ErrorTypeTerminal // Resource doesn't exist
    }
    if apierrors.IsUnauthorized(err) || apierrors.IsForbidden(err) {
        return ErrorTypeTerminal // RBAC issue
    }
    if apierrors.IsInvalid(err) {
        return ErrorTypeTerminal // Validation error
    }
    if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
        return ErrorTypeTransient // Network timeout
    }
    if apierrors.IsServiceUnavailable(err) {
        return ErrorTypeTransient // API server unavailable
    }

    // Default to retriable for unknown errors
    return ErrorTypeRetriable
}
```

### Reconciliation Error Handling

```go
// controllers/notebookvalidationjob_controller.go
func (r *NotebookValidationJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    
    // Fetch the NotebookValidationJob
    job := &mlopsv1alpha1.NotebookValidationJob{}
    if err := r.Get(ctx, req.NamespacedName, job); err != nil {
        if apierrors.IsNotFound(err) {
            // Resource deleted, stop reconciliation
            return ctrl.Result{}, nil
        }
        // Transient error, retry with backoff
        return ctrl.Result{}, err
    }

    // Check if job is already completed
    if job.Status.Phase == "Succeeded" || job.Status.Phase == "Failed" {
        return ctrl.Result{}, nil
    }

    // Execute validation workflow
    result, err := r.executeValidation(ctx, job)
    if err != nil {
        return r.handleError(ctx, job, err)
    }

    return result, nil
}

func (r *NotebookValidationJobReconciler) handleError(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, err error) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    
    // Classify error
    errorType := errors.ClassifyError(err)
    
    var validationErr *errors.ValidationError
    if !errors.As(err, &validationErr) {
        // Wrap generic error
        validationErr = &errors.ValidationError{
            Type:    errorType,
            Reason:  "UnknownError",
            Message: err.Error(),
            Err:     err,
        }
    }

    switch validationErr.Type {
    case errors.ErrorTypeTransient:
        // Update status with retry condition
        conditions.SetCondition(&job.Status.Conditions,
            "Ready",
            metav1.ConditionFalse,
            validationErr.Reason,
            fmt.Sprintf("Transient error, retrying: %s", validationErr.Message),
        )
        
        if err := r.Status().Update(ctx, job); err != nil {
            log.Error(err, "Failed to update status")
        }
        
        // Return error to trigger exponential backoff
        log.Info("Transient error, will retry with backoff",
            "reason", validationErr.Reason,
            "message", validationErr.Message,
        )
        return ctrl.Result{}, validationErr

    case errors.ErrorTypeRetriable:
        // Check retry limit
        if job.Status.RetryCount >= maxRetries {
            // Max retries exceeded, treat as terminal
            log.Info("Max retries exceeded, marking as failed",
                "retryCount", job.Status.RetryCount,
                "maxRetries", maxRetries,
            )
            return r.markAsFailed(ctx, job, "RetryLimitExceeded",
                fmt.Sprintf("Failed after %d retries: %s", maxRetries, validationErr.Message))
        }

        // Increment retry count
        job.Status.RetryCount++
        backoff := calculateBackoff(job.Status.RetryCount)
        
        conditions.SetCondition(&job.Status.Conditions,
            "Ready",
            metav1.ConditionFalse,
            validationErr.Reason,
            fmt.Sprintf("Retry %d/%d: %s", job.Status.RetryCount, maxRetries, validationErr.Message),
        )
        
        if err := r.Status().Update(ctx, job); err != nil {
            log.Error(err, "Failed to update status")
        }
        
        log.Info("Retriable error, scheduling retry",
            "retryCount", job.Status.RetryCount,
            "backoff", backoff,
            "reason", validationErr.Reason,
        )
        return ctrl.Result{RequeueAfter: backoff}, nil

    case errors.ErrorTypeTerminal:
        // Terminal error, mark as failed
        log.Info("Terminal error, marking as failed",
            "reason", validationErr.Reason,
            "message", validationErr.Message,
        )
        return r.markAsFailed(ctx, job, validationErr.Reason, validationErr.Message)
    }

    // Should never reach here
    return ctrl.Result{}, fmt.Errorf("unhandled error type: %s", errorType)
}

func (r *NotebookValidationJobReconciler) markAsFailed(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, reason, message string) (ctrl.Result, error) {
    job.Status.Phase = "Failed"
    job.Status.CompletionTime = &metav1.Time{Time: time.Now()}
    job.Status.Message = message
    
    conditions.SetCondition(&job.Status.Conditions,
        "ValidationComplete",
        metav1.ConditionFalse,
        reason,
        message,
    )
    
    if err := r.Status().Update(ctx, job); err != nil {
        return ctrl.Result{}, err
    }
    
    // Do not requeue
    return ctrl.Result{}, nil
}

const maxRetries = 3

func calculateBackoff(retryCount int) time.Duration {
    // Exponential backoff: 1m, 2m, 5m
    backoffs := []time.Duration{
        1 * time.Minute,
        2 * time.Minute,
        5 * time.Minute,
    }
    
    if retryCount <= 0 || retryCount > len(backoffs) {
        return backoffs[len(backoffs)-1]
    }
    
    return backoffs[retryCount-1]
}
```

### Timeout Handling

```go
// pkg/validation/executor.go
func (e *Executor) ExecuteNotebook(ctx context.Context, notebookPath string, timeout time.Duration) error {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Execute notebook with Papermill
    cmd := exec.CommandContext(ctx, "papermill", notebookPath, outputPath)
    
    if err := cmd.Run(); err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return errors.NewRetriableError(
                "ExecutionTimeout",
                fmt.Sprintf("Notebook execution exceeded timeout of %s", timeout),
                err,
            )
        }
        return errors.NewTerminalError(
            "ExecutionFailed",
            "Notebook execution failed",
            err,
        )
    }

    return nil
}
```

### Status Updates

```go
// api/v1alpha1/notebookvalidationjob_types.go
type NotebookValidationJobStatus struct {
    // ... existing fields ...

    // RetryCount tracks the number of retry attempts
    RetryCount int `json:"retryCount,omitempty"`

    // LastRetryTime is when the last retry occurred
    LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`
}
```

## Configuration

```go
// config/manager/manager.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: operator-config
data:
  # Maximum retry attempts for retriable errors
  maxRetries: "3"
  
  # Timeout for notebook execution (default: 30 minutes)
  executionTimeout: "30m"
  
  # Timeout for Git clone operations (default: 5 minutes)
  gitCloneTimeout: "5m"
```

## Consequences

### Positive
- **Resilient**: Automatically recovers from transient failures
- **User-Friendly**: Clear error messages guide users to resolution
- **Resource Efficient**: Retry limits prevent infinite loops
- **Observable**: Error classification enables better monitoring
- **Predictable**: Exponential backoff prevents thundering herd

### Negative
- **Complexity**: Three-tier error classification adds code complexity
- **Tuning**: Retry limits and backoff timings may need adjustment
- **Status Churn**: Frequent status updates during retries

### Neutral
- **Standard Pattern**: Follows Kubernetes controller best practices
- **Configurable**: Retry limits and timeouts can be adjusted

## References

- [Building Resilient Kubernetes Controllers](https://medium.com/@vamshitejanizam/building-resilient-kubernetes-controllers-a-practical-guide-to-retry-mechanisms-0d689160fa51)
- [controller-runtime Rate Limiting](https://danielmangum.com/posts/controller-runtime-client-go-rate-limiting/)
- [Kubernetes API Conventions - Errors](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#error-codes)

## Related ADRs

- ADR 010: Observability and Monitoring (error logging and metrics)
- ADR 003: CRD Schema Design (status field structure)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial error handling strategy |

