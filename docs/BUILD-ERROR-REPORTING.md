# Build Error Reporting

## Overview

This document describes how build strategy detection errors are surfaced to users through the NotebookValidationJob status, similar to how Jupyter cell execution errors are reported.

## Problem Statement

Previously, when S2I or Tekton build strategies failed to detect or initialize, users would only see generic error messages in the operator logs:

```
"Build strategy not available, falling back to container image"
"error": "build strategy not available: s2i"
```

Users had no visibility into:
- **Why** the build strategy failed (missing CRDs, permission issues, etc.)
- **Which** strategy was attempted
- **What** specific error occurred during detection

## Solution

### 1. Enhanced Detection Logging

Both S2I and Tekton strategies now include detailed logging in their `Detect()` methods:

**S2I Strategy** (`pkg/build/s2i_strategy.go`):
```go
func (s *S2IStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
    logger := log.FromContext(ctx)
    
    // Changed from Get() to List() for more reliable detection
    buildConfigList := &buildv1.BuildConfigList{}
    err := client.List(ctx, buildConfigList, &client2.ListOptions{
        Namespace: "default",
        Limit:     1,
    })

    if err != nil {
        logger.V(1).Info("S2I detection: error listing BuildConfigs", 
            "error", err,
            "errorType", fmt.Sprintf("%T", err),
            "isNotFound", errors.IsNotFound(err),
            "isNotRegistered", runtime.IsNotRegisteredError(err))
        
        // Check for specific error types
        if runtime.IsNotRegisteredError(err) {
            logger.Info("S2I not available: BuildConfig CRD not registered")
            return false, nil
        }
        
        if strings.Contains(err.Error(), "no matches for kind") {
            logger.Info("S2I not available: BuildConfig API not found")
            return false, nil
        }
        
        logger.Error(err, "S2I detection failed with unexpected error")
        return false, err
    }

    logger.Info("S2I available: BuildConfig API detected", "buildConfigCount", len(buildConfigList.Items))
    return true, nil
}
```

**Tekton Strategy** (`pkg/build/tekton_strategy.go`):
- Similar implementation using TaskRunList
- Same detailed error logging and classification

### 2. BuildStatus Error Reporting

The controller now captures and surfaces detection errors in the BuildStatus:

**Detection Error** (`internal/controller/build_integration_helper.go`):
```go
available, err := strategy.Detect(ctx, r.Client)
if err != nil {
    logger.Error(err, "Failed to check strategy availability", "strategy", strategyName)
    // Include detailed error in build status
    errorMsg := fmt.Sprintf("Strategy detection failed for %s: %v", strategyName, err)
    if updateErr := r.updateBuildStatus(ctx, job, "Failed", errorMsg, ""); updateErr != nil {
        logger.Error(updateErr, "Failed to update build status")
    }
    return job.Spec.PodConfig.ContainerImage, fmt.Errorf("failed to check strategy availability: %w", err)
}
```

**Strategy Not Available**:
```go
if !available {
    logger.Info("Build strategy not available, falling back to container image", "strategy", strategyName)
    // Provide helpful message about why strategy is not available
    errorMsg := fmt.Sprintf("Strategy not available: %s. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster.", strategyName)
    if updateErr := r.updateBuildStatus(ctx, job, "Failed", errorMsg, ""); updateErr != nil {
        logger.Error(updateErr, "Failed to update build status")
    }
    return job.Spec.PodConfig.ContainerImage, fmt.Errorf("build strategy not available: %s", strategyName)
}
```

### 3. Strategy Field in BuildStatus

The `updateBuildStatus()` method now sets the strategy field so users can see which strategy was attempted:

```go
func (r *NotebookValidationJobReconciler) updateBuildStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, status, message, imageReference string) error {
    // ... initialization ...
    
    // Set strategy from build config if available
    if job.Spec.PodConfig != nil && job.Spec.PodConfig.BuildConfig != nil {
        job.Status.BuildStatus.Strategy = job.Spec.PodConfig.BuildConfig.Strategy
    }
    
    // ... rest of method ...
}
```

## User Experience

### Before

Users would see:
```bash
$ oc get notebookvalidationjob notebookvalidationjob-s2i-sample -o yaml
status:
  phase: Running
  # No build status information
```

And only generic logs:
```
"Build strategy not available, falling back to container image"
```

### After

Users now see detailed status:
```bash
$ oc get notebookvalidationjob notebookvalidationjob-s2i-sample -o yaml
status:
  phase: Running
  buildStatus:
    phase: Failed
    strategy: s2i
    message: "Strategy not available: s2i. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster."
    startTime: "2025-11-09T03:00:00Z"
    completionTime: "2025-11-09T03:00:01Z"
```

Or for detection errors:
```yaml
buildStatus:
  phase: Failed
  strategy: s2i
  message: "Strategy detection failed for s2i: no matches for kind BuildConfig in group build.openshift.io"
```

## Error Categories

### 1. CRD Not Registered
**Cause**: The CRD is not installed in the cluster
**Message**: "S2I not available: BuildConfig CRD not registered"
**Resolution**: Install OpenShift Build API or Tekton Pipelines

### 2. API Not Found
**Cause**: The API group is not available
**Message**: "S2I not available: BuildConfig API not found"
**Resolution**: Verify API server configuration

### 3. Permission Issues
**Cause**: ServiceAccount lacks RBAC permissions
**Message**: "Strategy detection failed for s2i: [permission error details]"
**Resolution**: Check ClusterRole and RoleBinding

### 4. Unexpected Errors
**Cause**: Other errors (network, API server issues, etc.)
**Message**: Full error details from the API
**Resolution**: Check operator logs for detailed stack trace

## Testing

To test error reporting:

1. **Test with S2I unavailable**:
```bash
# On a cluster without OpenShift Build API
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml
oc get notebookvalidationjob notebookvalidationjob-s2i-sample -o jsonpath='{.status.buildStatus}'
```

2. **Test with Tekton unavailable**:
```bash
# On a cluster without Tekton Pipelines
# Create a sample with strategy: tekton
oc get notebookvalidationjob <name> -o jsonpath='{.status.buildStatus}'
```

3. **Check operator logs**:
```bash
oc logs -n jupyter-notebook-validator-operator deployment/notebook-validator-controller-manager -c manager --tail=50
```

## Benefits

1. **User Visibility**: Users can see exactly why builds are failing without checking operator logs
2. **Troubleshooting**: Clear error messages guide users to the root cause
3. **Consistency**: Build errors are reported the same way as cell execution errors
4. **Automation**: CI/CD pipelines can parse BuildStatus to detect and handle build failures

## Related Documentation

- [ADR-016: S2I Build Strategy for Git Integration](adr/ADR-016-S2I-Build-Strategy-for-Git-Integration.md)
- [Git Clone Approaches](GIT-CLONE-APPROACHES.md)
- [Testing Status](../TESTING-STATUS.md)

## Future Enhancements

1. Add retry logic for transient detection errors
2. Include suggested remediation steps in error messages
3. Add metrics for build detection success/failure rates
4. Create alerts for repeated detection failures

