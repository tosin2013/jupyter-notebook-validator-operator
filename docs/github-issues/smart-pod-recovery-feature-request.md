# GitHub Issue: Feature Request - Smart Validation Pod Recovery and Build Strategy Fallback

**Issue Type:** Feature Request / Enhancement
**Priority:** Medium-High
**Labels:** `enhancement`, `reliability`, `pod-recovery`, `build-strategy`, `help wanted`

---

## Title

**Feature Request: Smart Validation Pod Recovery with Build Strategy Fallback**

---

## Summary

Add intelligent pod failure detection and recovery logic with automatic build strategy fallback (S2I → Tekton → Pre-built image). This addresses production scenarios where validation pods fail due to image pull errors, SCC violations, or build strategy incompatibilities.

**Status:** Fully designed in [ADR-026](../adrs/026-smart-validation-pod-recovery.md)
**Priority:** Medium-High
**Category:** Error Handling, Reliability, Build Strategy

---

## Problem Statement

Validation pods can fail for various reasons in production, and the current simple retry logic (delete + recreate with same config) doesn't handle all failure scenarios intelligently.

### Problem 1: Image Pull Failures

**Scenario:**
1. NotebookValidationJob triggers S2I build
2. Build succeeds, creates image `image-registry.openshift-image-registry.svc:5000/ns/notebook:abc123`
3. Validation pod tries to pull image
4. Image pull fails: `ImagePullBackOff` (registry issues, rate limits, authentication)
5. Operator retries 3 times with same configuration
6. All retries fail with same error

**Impact:**
- Wasted retry attempts (same failure repeated)
- No fallback to alternative image sources
- Unclear whether issue is transient or permanent

---

### Problem 2: OpenShift SCC Violations

**Scenario:**
1. User specifies `containerImage: jupyter/scipy-notebook:latest`
2. Operator creates validation pod with git-clone init container
3. Git-clone init container uses `bitnami/git:latest` (runs as UID 1001)
4. OpenShift SCC rejects pod: `RunContainerError` (must run as allocated UID range)
5. Operator retries 3 times
6. All retries fail with same SCC error

**Impact:**
- SCC violations not detected automatically
- No fallback to remove problematic init container
- Manual intervention required to fix configuration

---

### Problem 3: Build Strategy Incompatibility

**Scenario:**
1. User specifies `buildConfig.strategy: s2i`
2. S2I build fails (base image doesn't support S2I)
3. Operator reports build failure
4. No automatic fallback to Tekton build

**Impact:**
- User must manually change build strategy
- Not all base images are S2I-compatible
- Tekton is more flexible but not tried automatically

---

### Problem 4: CrashLoopBackOff

**Scenario:**
1. Validation pod starts
2. Container crashes immediately (missing dependency, bad notebook code)
3. Kubernetes enters `CrashLoopBackOff`
4. Operator waits indefinitely
5. No intelligent detection of application-level failures

**Impact:**
- Unclear whether failure is transient (should retry) or permanent (should fail)
- No distinction between infrastructure issues and notebook code issues
- Poor debugging experience

---

## Proposed Solution

Implement **smart pod failure detection** with **automatic strategy fallback** and **intelligent retry logic**.

### Feature Overview

#### 1. Enhanced Pod Failure Detection

Classify failures into categories:

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
    IsTransient     bool           // Should retry with same config
    ShouldRetry     bool           // Should attempt recovery
    SuggestedAction string         // What to do next
    FailedContainer string         // Which container failed
    ErrorMessage    string         // Detailed error
}
```

#### 2. Smart Retry Decision Tree

```
┌─────────────────────────────────┐
│  Pod Failure Detected           │
└──────────┬──────────────────────┘
           │
           ▼
   ┌───────────────┐
   │  Analyze      │
   │  Failure      │
   └───────┬───────┘
           │
           ▼
    ┌─────────────────────────────────────┐
    │  Classify Failure Type              │
    └─────────────────────────────────────┘
            │
            ├─── ImagePullBackOff ────────► Try fallback image
            │
            ├─── RunContainerError (SCC) ──► Remove init container
            │
            ├─── CrashLoopBackOff ────────► Try different build strategy
            │
            ├─── InitContainerError ──────► Use pre-built image
            │
            └─── Unknown ─────────────────► Standard retry
```

#### 3. Build Strategy Fallback Chain

```
S2I Build
   │
   ├─── Success ─────────────────────► Use built image
   │
   └─── Failure
         │
         ▼
      Tekton Build
         │
         ├─── Success ─────────────────► Use built image
         │
         └─── Failure
               │
               ▼
            Pre-built Image
               │
               └─── Use spec.podConfig.containerImage
```

---

## Implementation Details

### Phase 1: Failure Detection (Weeks 1-2)

```go
// Analyze pod failure and determine recovery action
func (r *Reconciler) analyzePodFailure(pod *corev1.Pod) PodFailureAnalysis {
    analysis := PodFailureAnalysis{
        Reason:      FailureReasonUnknown,
        IsTransient: false,
        ShouldRetry: true,
    }

    // Check container statuses
    for _, cs := range pod.Status.ContainerStatuses {
        if cs.State.Waiting != nil {
            switch cs.State.Waiting.Reason {
            case "ImagePullBackOff", "ErrImagePull":
                analysis.Reason = FailureReasonImagePull
                analysis.IsTransient = false
                analysis.SuggestedAction = "Try fallback to pre-built image"
                analysis.FailedContainer = cs.Name
                analysis.ErrorMessage = cs.State.Waiting.Message

            case "CrashLoopBackOff":
                analysis.Reason = FailureReasonCrashLoop
                analysis.IsTransient = false
                analysis.SuggestedAction = "Try different build strategy"
                analysis.FailedContainer = cs.Name
                analysis.ErrorMessage = cs.State.Waiting.Message

            case "RunContainerError", "CreateContainerConfigError":
                analysis.Reason = FailureReasonRunContainer
                analysis.IsTransient = false

                // Check if SCC-related
                if strings.Contains(cs.State.Waiting.Message, "scc") ||
                   strings.Contains(cs.State.Waiting.Message, "unable to validate against any security context constraint") {
                    analysis.SuggestedAction = "Remove git-clone init container (SCC violation)"
                } else {
                    analysis.SuggestedAction = "Check container configuration"
                }
                analysis.FailedContainer = cs.Name
                analysis.ErrorMessage = cs.State.Waiting.Message
            }
        }
    }

    // Check init container statuses
    for _, ics := range pod.Status.InitContainerStatuses {
        if ics.State.Waiting != nil {
            analysis.Reason = FailureReasonInitContainer
            analysis.IsTransient = false
            analysis.SuggestedAction = "Use pre-built image without git-clone init container"
            analysis.FailedContainer = ics.Name
            analysis.ErrorMessage = ics.State.Waiting.Message
        }
    }

    return analysis
}
```

### Phase 2: Smart Recovery (Weeks 3-4)

```go
// Implement recovery actions based on failure analysis
func (r *Reconciler) recoverFromPodFailure(ctx context.Context, job *NotebookValidationJob, analysis PodFailureAnalysis) error {
    switch analysis.Reason {
    case FailureReasonImagePull:
        // Fallback to pre-built image
        return r.retryWithPreBuiltImage(ctx, job)

    case FailureReasonRunContainer:
        if strings.Contains(analysis.SuggestedAction, "SCC") {
            // Remove git-clone init container
            return r.retryWithoutInitContainer(ctx, job)
        }
        return r.standardRetry(ctx, job)

    case FailureReasonCrashLoop:
        // Try different build strategy
        return r.retryWithFallbackBuildStrategy(ctx, job)

    case FailureReasonInitContainer:
        // Use pre-built image
        return r.retryWithPreBuiltImage(ctx, job)

    default:
        // Standard retry
        return r.standardRetry(ctx, job)
    }
}
```

### Phase 3: Build Strategy Fallback (Weeks 5-6)

```go
// Try fallback build strategies
func (r *Reconciler) retryWithFallbackBuildStrategy(ctx context.Context, job *NotebookValidationJob) error {
    currentStrategy := job.Status.BuildStrategy

    switch currentStrategy {
    case "s2i":
        logger.Info("S2I build failed, falling back to Tekton")
        job.Status.BuildStrategy = "tekton"
        job.Status.BuildAttempts = 0  // Reset counter for new strategy

    case "tekton":
        logger.Info("Tekton build failed, falling back to pre-built image")
        job.Status.BuildStrategy = "prebuilt"
        job.Status.BuildAttempts = 0

    default:
        return fmt.Errorf("no fallback strategy available")
    }

    return r.Status().Update(ctx, job)
}
```

---

## Use Cases

### Use Case 1: ImagePullBackOff Recovery

**Before:**
```
1. S2I builds image successfully
2. Pod tries to pull image: ImagePullBackOff (registry unreachable)
3. Retry 1: Same error
4. Retry 2: Same error
5. Retry 3: Same error
6. Job fails after 3 retries
```

**After (with smart recovery):**
```
1. S2I builds image successfully
2. Pod tries to pull image: ImagePullBackOff (registry unreachable)
3. Operator detects: Non-transient image pull failure
4. Operator switches to pre-built image from spec.podConfig.containerImage
5. Pod starts successfully with fallback image
6. Notebook executes successfully
```

---

### Use Case 2: OpenShift SCC Violation Recovery

**Before:**
```
1. Pod created with git-clone init container (bitnami/git)
2. OpenShift rejects: RunContainerError (SCC violation)
3. Retry 1: Same error
4. Retry 2: Same error
5. Retry 3: Same error
6. Job fails after 3 retries
```

**After (with smart recovery):**
```
1. Pod created with git-clone init container (bitnami/git)
2. OpenShift rejects: RunContainerError (SCC violation)
3. Operator detects: SCC-related failure
4. Operator removes git-clone init container
5. Operator uses built image (already has notebook from build)
6. Pod starts successfully
7. Notebook executes successfully
```

---

### Use Case 3: Build Strategy Fallback

**Before:**
```
1. User specifies buildConfig.strategy: s2i
2. Base image doesn't support S2I
3. Build fails
4. Job marked as Failed
5. User must manually change to Tekton
```

**After (with smart recovery):**
```
1. User specifies buildConfig.strategy: s2i
2. Base image doesn't support S2I
3. Build fails
4. Operator detects: S2I build failure
5. Operator falls back to Tekton build
6. Tekton build succeeds
7. Notebook executes successfully
```

---

## Status Message Improvements

**Before:**
```
Phase: Failed
Message: Pod validation-pod-abc123 failed
```

**After:**
```
Phase: Failed
Message: Validation pod failed: ImagePullBackOff (image not found). Attempted recovery: fallback to pre-built image. Final status: Success
```

---

## Implementation Plan

See [ADR-026](../adrs/026-smart-validation-pod-recovery.md) for complete details.

### High-Level Milestones (6 weeks)

**Weeks 1-2: Failure Detection**
- [ ] Implement `analyzePodFailure()` function
- [ ] Add pod status inspection logic
- [ ] Classify failure types
- [ ] Unit tests for failure detection

**Weeks 3-4: Recovery Actions**
- [ ] Implement `recoverFromPodFailure()` function
- [ ] Add fallback to pre-built image
- [ ] Add init container removal
- [ ] Add standard retry logic
- [ ] Integration tests

**Weeks 5-6: Build Strategy Fallback**
- [ ] Implement build strategy fallback chain
- [ ] Add S2I → Tekton fallback
- [ ] Add Tekton → Pre-built fallback
- [ ] Update status messages
- [ ] E2E tests
- [ ] Documentation

---

## Benefits

### For Users
- ✅ Automatic recovery from common failure scenarios
- ✅ Reduced manual intervention
- ✅ Better error messages with suggested actions
- ✅ Higher success rate for notebook validations

### For Platform Teams
- ✅ Reduced support burden (fewer SCC-related tickets)
- ✅ Better observability into failure patterns
- ✅ Automatic build strategy optimization
- ✅ More resilient operator behavior

### For Developers
- ✅ Clear failure classification
- ✅ Extensible recovery framework
- ✅ Better testing capabilities
- ✅ Improved debugging experience

---

## Consequences

### Positive
- ✅ Improved reliability (automatic recovery)
- ✅ Better user experience (clear error messages)
- ✅ Reduced operational burden (fewer manual interventions)
- ✅ Intelligent retry logic (avoids wasted retries)

### Negative
- ⚠️ Increased complexity (more decision logic)
- ⚠️ More code paths to test
- ⚠️ Potential for unexpected behavior (fallback surprises)

### Mitigation
- Comprehensive unit and integration tests
- Clear status messages showing what recovery actions were taken
- Configuration option to disable automatic fallback
- Detailed logging of recovery decisions

---

## Configuration Options

Add optional configuration to control recovery behavior:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-notebook
spec:
  # ... existing fields

  # Optional: Control recovery behavior
  recoveryConfig:
    enableSmartRecovery: true  # Default: true
    enableBuildFallback: true  # Default: true
    maxRecoveryAttempts: 3     # Default: 3
    disabledRecoveryActions:   # Optional: Disable specific recovery actions
      - "fallback-to-prebuilt"
```

---

## Acceptance Criteria

- [ ] Pod failures classified into 7+ categories
- [ ] Automatic fallback to pre-built image on ImagePullBackOff
- [ ] Automatic removal of init container on SCC violation
- [ ] Build strategy fallback chain (S2I → Tekton → Pre-built)
- [ ] Clear status messages showing recovery actions taken
- [ ] Unit tests for all failure types
- [ ] Integration tests for recovery actions
- [ ] E2E tests for build strategy fallback
- [ ] Documentation with examples
- [ ] Backward compatible (existing behavior unchanged if smart recovery disabled)

---

## References

- **ADR-026**: [Smart Validation Pod Recovery](../adrs/026-smart-validation-pod-recovery.md)
- **ADR-011**: Error Handling and Retry Strategy
- **ADR-023**: S2I Build Strategy
- **ADR-031**: Tekton Build Strategy
- **OpenShift SCC Docs**: https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

---

## Community Input Requested

- What other failure scenarios should be detected?
- Should recovery actions be configurable per-job or operator-wide?
- Interest in contributing additional recovery strategies?

---

## Labels

Please add:
- `enhancement`
- `reliability`
- `pod-recovery`
- `build-strategy`
- `help wanted`
- `priority: medium-high`

---

## CC

@tosin2013 (operator maintainer)

---

**Thank you for considering this feature request!** This would significantly improve the operator's reliability in production environments.
