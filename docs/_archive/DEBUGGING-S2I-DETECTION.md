# Debugging S2I Detection Issue - A Case Study

## Problem Statement

S2I build strategy detection was consistently failing with generic error messages:
```
"Build strategy not available, falling back to container image"
"error": "build strategy not available: s2i"
```

Users had no visibility into **why** the detection was failing.

## Investigation Journey

### Phase 1: Enhanced Error Reporting

**User Request**: "we may want that error to show in the status of the job just like we did with other code like jupyter cell status"

**Implementation** (Commits: `dd254e5`, `19f51dd`):
1. Added detailed logging to `Detect()` methods in both S2I and Tekton strategies
2. Changed from `Get()` to `List()` for more reliable CRD detection
3. Surfaced detection errors in `BuildStatus.Message` field
4. Added `BuildStatus.Strategy` field to show which strategy was attempted

**Key Code Changes**:
```go
// pkg/build/s2i_strategy.go
func (s *S2IStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
    logger := log.FromContext(ctx)
    
    buildConfigList := &buildv1.BuildConfigList{}
    err := client.List(ctx, buildConfigList)

    if err != nil {
        logger.V(1).Info("S2I detection: error listing BuildConfigs", 
            "error", err,
            "errorType", fmt.Sprintf("%T", err),
            "isNotFound", errors.IsNotFound(err),
            "isNotRegistered", runtime.IsNotRegisteredError(err))
        
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

    logger.Info("S2I available: BuildConfig API detected")
    return true, nil
}
```

### Phase 2: Fixing Import and Type Issues

**Problem**: Initial implementation had compilation errors

**Fixes** (Commit: `19f51dd`):
- Added missing `strings` import
- Added missing `log` import from `sigs.k8s.io/controller-runtime/pkg/log`
- Fixed `client.List()` calls (removed invalid `&client.ListOptions{}`)
- Fixed `PodConfig` nil check (it's a struct, not a pointer)

### Phase 3: Root Cause Discovery

**Deployment**: Deployed operator with enhanced error reporting (image: `release-4.18-19f51dd`)

**Test**:
```bash
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml
oc get notebookvalidationjob notebookvalidationjob-s2i-sample -o jsonpath='{.status.buildStatus}' | jq
```

**Result - BuildStatus**:
```json
{
  "completionTime": "2025-11-09T03:24:40Z",
  "message": "Strategy not available: s2i. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster.",
  "phase": "Failed",
  "strategy": "s2i"
}
```

**Result - Operator Logs**:
```
DEBUG S2I detection: error listing BuildConfigs
  error: "no kind is registered for the type v1.BuildConfigList in scheme \"pkg/runtime/scheme.go:100\""
  errorType: "*runtime.notRegisteredErr"
  isNotFound: false
  isNotRegistered: true

INFO S2I not available: BuildConfig CRD not registered
```

**🎯 Root Cause Identified**: The BuildConfig and TaskRun types were **not registered in the operator's runtime scheme**!

This was NOT a cluster issue - it was a code issue in the operator itself.

### Phase 4: The Fix

**Problem**: The operator's scheme didn't include OpenShift Build or Tekton Pipeline types.

**Solution** (Commit: `82470e3`):

Added scheme registration in `cmd/main.go`:

```go
import (
    // ... existing imports ...
    buildv1 "github.com/openshift/api/build/v1"
    tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func init() {
    utilruntime.Must(clientgoscheme.AddToScheme(scheme))
    utilruntime.Must(mlopsv1alpha1.AddToScheme(scheme))

    // Register OpenShift Build API for S2I support
    utilruntime.Must(buildv1.AddToScheme(scheme))

    // Register Tekton Pipeline API for Tekton support
    utilruntime.Must(tektonv1.AddToScheme(scheme))

    //+kubebuilder:scaffold:scheme
}
```

## Key Learnings

### 1. Enhanced Error Reporting is Critical

Without detailed error logging, we would have spent much longer debugging. The enhanced error reporting immediately showed:
- The exact error type: `*runtime.notRegisteredErr`
- The specific error message: "no kind is registered for the type v1.BuildConfigList"
- Whether it was a NotFound vs NotRegistered error

### 2. User Feedback Drives Better Design

The user's suggestion to "show errors in job status like we do with Jupyter cell status" led to:
- Better user experience (errors visible in `oc get` output)
- Faster debugging (detailed logs with error classification)
- Consistent error handling across the operator

### 3. Scheme Registration is Easy to Forget

When adding support for external CRDs (OpenShift Build, Tekton), it's easy to forget to register them in the scheme. This should be part of the checklist when adding new API integrations.

### 4. Test-Driven Development Would Have Caught This

A unit test that actually tries to list BuildConfigs would have caught this immediately:

```go
func TestS2IDetection(t *testing.T) {
    scheme := runtime.NewScheme()
    utilruntime.Must(clientgoscheme.AddToScheme(scheme))
    utilruntime.Must(buildv1.AddToScheme(scheme)) // This line would be missing!
    
    client := fake.NewClientBuilder().WithScheme(scheme).Build()
    strategy := NewS2IStrategy(client, scheme)
    
    available, err := strategy.Detect(context.Background(), client)
    // Test would fail without buildv1.AddToScheme
}
```

## Timeline

1. **Initial Problem**: S2I detection failing with generic errors
2. **User Request**: Surface errors in job status (like cell execution errors)
3. **Implementation**: Enhanced error reporting with detailed logging
4. **Deployment**: Tested with new operator image
5. **Discovery**: Logs showed `*runtime.notRegisteredErr`
6. **Fix**: Added scheme registration for BuildConfig and TaskRun
7. **Next**: Deploy and verify S2I builds work end-to-end

## Documentation Created

1. **`docs/BUILD-ERROR-REPORTING.md`**: How error reporting works
2. **`docs/TROUBLESHOOTING-BUILD-DETECTION.md`**: Step-by-step troubleshooting guide
3. **`docs/DEBUGGING-S2I-DETECTION.md`**: This case study

## Next Steps

1. Wait for GitHub Actions to build image `release-4.18-82470e3`
2. Deploy the new operator image
3. Test S2I detection - should now succeed
4. Test actual S2I build creation and execution
5. Verify Tekton detection also works
6. Update test plan and mark S2I tests as ready

## Commands for Testing

```bash
# Deploy new operator
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:release-4.18-82470e3

# Test S2I detection
oc delete notebookvalidationjob notebookvalidationjob-s2i-sample -n default
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml

# Check BuildStatus (should show success now)
oc get notebookvalidationjob notebookvalidationjob-s2i-sample -o jsonpath='{.status.buildStatus}' | jq

# Check operator logs
oc logs -n jupyter-notebook-validator-operator deployment/notebook-validator-controller-manager -c manager --tail=50 | grep -i "s2i\|detect"

# Verify BuildConfig was created
oc get buildconfigs -n default

# Watch build progress
oc get builds -w
```

## Conclusion

This debugging journey demonstrates the value of:
1. **User-driven improvements**: The user's suggestion led to better error reporting
2. **Detailed logging**: Error type classification helped identify the root cause immediately
3. **Methodological pragmatism**: We didn't assume the cluster was wrong - we verified the operator code
4. **Documentation**: Capturing the journey helps future developers avoid the same issue

The fix was simple (3 lines of code), but finding it required systematic investigation and good tooling.

