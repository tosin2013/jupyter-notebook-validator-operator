# ADR-019: Smart Validation Pod Recovery and Build Strategy Selection

## Status
Proposed

## Context

### Problem Statement
Validation pods can fail for various reasons in production:
1. **Image Pull Errors**: `ImagePullBackOff`, `ErrImagePull` (rate limits, authentication, registry issues)
2. **Permission Errors**: `RunContainerError` (OpenShift SCC violations, non-root requirements)
3. **Container Errors**: `CrashLoopBackOff` (application crashes, missing dependencies)
4. **Init Container Failures**: Git clone failures, credential issues
5. **Build Strategy Incompatibility**: S2I may not work for all base images

### Current Limitations
- Simple retry logic: deletes pod and recreates with same configuration
- No detection of specific failure reasons
- No build strategy fallback (S2I → Tekton)
- No intelligent decision-making based on failure patterns
- Git clone init container still uses bitnami/git (causes OpenShift SCC issues)

### Research Findings

**Kubernetes Operator Best Practices**:
1. **Detect specific failure reasons** from pod status and container statuses
2. **Classify failures** as transient (retry same config) vs. persistent (change strategy)
3. **Exponential backoff** for retries (already implemented for resource conflicts)
4. **Delete and recreate** pods for certain failure types
5. **Fallback strategies** when primary approach fails

**OpenShift S2I vs Tekton Decision Matrix**:
- **S2I**: Best for standard Jupyter images with S2I support, simpler, faster
- **Tekton**: More flexible, works with any base image, better for complex builds
- **Fallback**: If S2I fails, try Tekton; if both fail, use pre-built image

## Decision

### 1. Enhanced Pod Failure Detection

Implement detailed failure reason detection:

```go
type PodFailureReason string

const (
    FailureReasonImagePull         PodFailureReason = "ImagePullBackOff"
    FailureReasonImagePullError    PodFailureReason = "ErrImagePull"
    FailureReasonCrashLoop         PodFailureReason = "CrashLoopBackOff"
    FailureReasonRunContainer      PodFailureReason = "RunContainerError"
    FailureReasonCreateContainer   PodFailureReason = "CreateContainerConfigError"
    FailureReasonInitContainer     PodFailureReason = "InitContainerError"
    FailureReasonPermission        PodFailureReason = "PermissionDenied"
    FailureReasonUnknown           PodFailureReason = "Unknown"
)

type PodFailureAnalysis struct {
    Reason          PodFailureReason
    IsTransient     bool
    ShouldRetry     bool
    SuggestedAction string
    FailedContainer string
    ErrorMessage    string
}
```

### 2. Smart Retry Logic with Strategy Fallback

```go
// Retry decision tree:
// 1. ImagePullBackOff → Check if using built image → Try fallback to pre-built image
// 2. RunContainerError (SCC) → Check if init container → Remove git-clone, use built image
// 3. CrashLoopBackOff → Check retry count → Try different build strategy
// 4. InitContainerError → Check if git-clone → Try without init container (use built image)
```

### 3. Build Strategy Selection Rules

**Workflow Rules**:
```yaml
# Priority 1: User-specified strategy
if spec.podConfig.buildConfig.strategy != "":
  use specified strategy

# Priority 2: OpenShift AI detection
if OpenShift AI installed && image in redhat-ods-applications:
  prefer S2I with OpenShift AI image

# Priority 3: Base image S2I capability
if base image has S2I labels:
  prefer S2I
else:
  prefer Tekton

# Priority 4: Failure-based fallback
if S2I build failed:
  retry with Tekton
if Tekton build failed:
  fallback to pre-built image (spec.podConfig.containerImage)
```

### 4. Remove Git Clone Init Container for Built Images

**Key Insight**: When using S2I or Tekton builds, the notebook is **already in the built image**. The git-clone init container is unnecessary and causes OpenShift SCC issues.

**Solution**:
```go
// In createValidationPod():
if containerImage != job.Spec.PodConfig.ContainerImage {
    // Using a built image - notebook is already in the image
    // Skip git-clone init container
    logger.Info("Using built image, skipping git-clone init container", "image", containerImage)
    initContainers = []corev1.Container{} // No init containers needed
} else {
    // Using pre-built image - need git-clone
    logger.Info("Using pre-built image, adding git-clone init container")
    gitCloneContainer, err := r.buildGitCloneInitContainer(ctx, job, creds)
    // ... add init container
}
```

## Implementation Plan

### Phase 1: Enhanced Failure Detection (IMMEDIATE)
1. Add `analyzePodFailure()` function to detect specific failure reasons
2. Inspect pod status, container statuses, and init container statuses
3. Extract error messages and classify failures

### Phase 2: Smart Retry with Strategy Fallback (THIS WEEK)
1. Modify `handlePodFailure()` to use failure analysis
2. Implement strategy fallback logic:
   - S2I failure → Retry with Tekton
   - Tekton failure → Fallback to pre-built image
   - Init container failure → Skip init container, use built image
3. Track failure history in job status

### Phase 3: Remove Git Clone for Built Images (THIS WEEK)
1. Modify `createValidationPod()` to detect built vs. pre-built images
2. Skip git-clone init container when using built images
3. Update documentation

### Phase 4: Build Strategy Selection Rules (FUTURE)
1. Implement intelligent strategy selection based on:
   - Environment capabilities (OpenShift AI, Tekton, S2I)
   - Base image capabilities (S2I labels)
   - Previous failure patterns
2. Add strategy recommendation to job status

## Consequences

### Positive
- **Resilient**: Automatically recovers from common failure scenarios
- **Intelligent**: Makes smart decisions based on failure analysis
- **Flexible**: Falls back to alternative strategies when needed
- **OpenShift-Compatible**: Removes SCC-violating init containers for built images
- **Production-Ready**: Handles real-world failure scenarios

### Negative
- **Complexity**: More complex retry logic
- **Build Time**: Fallback strategies may increase total time to success
- **Resource Usage**: Multiple build attempts consume more resources

### Risks
- **Infinite Loops**: Must have max retry limits per strategy
- **Cost**: Multiple builds could increase cloud costs
- **Confusion**: Users may not understand why strategy changed

### Mitigations
- **Max Retries**: Limit retries per strategy (3 per strategy, 9 total max)
- **Status Transparency**: Log all strategy changes and reasons in job status
- **Documentation**: Clear docs on failure recovery and strategy selection
- **Metrics**: Track failure reasons and recovery success rates

## Examples

### Example 1: ImagePullBackOff Recovery
```
1. Create validation pod with built image
2. Pod fails: ImagePullBackOff (registry rate limit)
3. Analyze failure: Image pull error, transient
4. Delete pod, retry with exponential backoff
5. If still fails after 3 retries: fallback to pre-built image
```

### Example 2: SCC Permission Error Recovery
```
1. Create validation pod with git-clone init container
2. Pod fails: RunContainerError (runAsNonRoot violation)
3. Analyze failure: Init container permission error
4. Delete pod, recreate WITHOUT git-clone (use built image)
5. Pod succeeds
```

### Example 3: S2I Build Failure → Tekton Fallback
```
1. S2I build fails (base image not S2I-compatible)
2. Analyze failure: Build strategy incompatibility
3. Delete build resources
4. Retry with Tekton strategy
5. Tekton build succeeds
6. Validation pod created with Tekton-built image
```

## References
- [Kubernetes Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [OpenShift S2I Documentation](https://docs.openshift.com/container-platform/latest/cicd/builds/understanding-image-builds.html)
- [Tekton Pipelines](https://tekton.dev/docs/pipelines/)
- [Operator SDK Best Practices](https://sdk.operatorframework.io/docs/best-practices/)

