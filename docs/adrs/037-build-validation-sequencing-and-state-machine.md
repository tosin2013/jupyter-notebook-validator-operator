# ADR 037: Build-Validation Sequencing and State Machine

## Status
Implemented

**Updated**: 2026-01-24
**Implementation**: State machine phases (`Building`, `BuildComplete`, `ValidationRunning`, `Succeeded`, `Failed`) in `api/v1alpha1/notebookvalidationjob_types.go`

## Context

### Problem Statement
In production deployments, a critical race condition has been identified where notebook validation begins before the container image build completes. This results in 100% false negative validation results when custom builds are enabled, undermining the operator's core value proposition.

**Timeline of the Race Condition**:
```
04:48:05 - NotebookValidationJob created
04:48:05 - Tekton PipelineRun started (building image with custom dependencies)
04:48:05 - Validation pod ALSO created immediately ❌
04:48:06 - Validation fails: "ModuleNotFoundError: No module named 'seaborn'"
05:10:00 - Build completes successfully (too late - validation already failed)
```

### Root Cause
The operator has two independent reconciliation loops that run in parallel:
1. **Build controller** → Starts Tekton/S2I build
2. **Validation controller** → Starts validation pod

These loops are not sequenced, causing the validation pod to use a fallback/base image that lacks the custom dependencies being built.

### Impact
- **Severity**: 🔴 Critical (blocks production use)
- **Frequency**: 100% when buildConfig.enabled = true
- **User Experience**: Complete breakdown of validation workflow
- **Workaround**: Manual pre-building of images (defeats automation purpose)

### Current Implementation
```go
// Simplified current logic (PROBLEMATIC)
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // These happen in parallel - WRONG!
    if job.Spec.BuildConfig.Enabled {
        r.reconcileBuild(ctx, job)  // Starts build
    }

    r.reconcileValidation(ctx, job)  // Starts validation immediately
    // No waiting for build completion!
}
```

## Decision

We will implement a **state machine-based reconciliation loop** that enforces strict sequencing between build and validation phases.

### State Machine Design

```
┌─────────────┐
│ Initializing│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Building   │ ← Wait here until build completes
└──────┬──────┘
       │ Build Complete
       ▼
┌──────────────┐
│BuildComplete │ ← Update imageReference
└──────┬───────┘
       │
       ▼
┌────────────────┐
│ValidationRunning│
└──────┬─────────┘
       │
       ├─▶ Succeeded
       └─▶ Failed
```

### State Definitions

| Phase | Description | Next Phase | Requeue |
|-------|-------------|------------|---------|
| `Initializing` | Job created, Git clone in progress | `Building` or `ValidationRunning` | 10s |
| `Building` | Build in progress (Tekton/S2I) | `BuildComplete` | 30s |
| `BuildComplete` | Build succeeded, imageReference updated | `ValidationRunning` | 0s |
| `ValidationRunning` | Validation pod executing | `Succeeded` or `Failed` | 10s |
| `Succeeded` | Validation passed | Terminal | N/A |
| `Failed` | Validation failed | Terminal | N/A |

### Implementation Strategy

#### 1. Add Status Fields to CRD

```go
// api/v1alpha1/notebookvalidationjob_types.go
type NotebookValidationJobStatus struct {
    // Phase represents the current phase of the validation job
    // +kubebuilder:validation:Enum=Initializing;Building;BuildComplete;ValidationRunning;Succeeded;Failed
    Phase string `json:"phase,omitempty"`

    // BuildStatus tracks the build progress
    BuildStatus *BuildStatus `json:"buildStatus,omitempty"`

    // Conditions for Kubernetes-style status
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // ... existing fields
}

type BuildStatus struct {
    // Phase of the build process
    // +kubebuilder:validation:Enum=Pending;Running;Complete;Failed
    Phase string `json:"phase,omitempty"`

    // ImageReference is the full image reference after build completes
    ImageReference string `json:"imageReference,omitempty"`

    // StartTime is when the build started
    StartTime *metav1.Time `json:"startTime,omitempty"`

    // CompletionTime is when the build completed
    CompletionTime *metav1.Time `json:"completionTime,omitempty"`

    // Duration is the build duration in human-readable format
    Duration string `json:"duration,omitempty"`
}
```

#### 2. Implement State Machine in Controller

```go
// internal/controller/notebookvalidationjob_controller.go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Fetch the job
    job := &mlopsv1alpha1.NotebookValidationJob{}
    if err := r.Get(ctx, req.NamespacedName, job); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // State machine dispatch
    switch job.Status.Phase {
    case "": // Initial state
        return r.reconcileInitializing(ctx, job)

    case "Initializing":
        return r.reconcileInitializing(ctx, job)

    case "Building":
        return r.reconcileBuilding(ctx, job)

    case "BuildComplete":
        return r.reconcileBuildComplete(ctx, job)

    case "ValidationRunning":
        return r.reconcileValidationRunning(ctx, job)

    case "Succeeded", "Failed":
        // Terminal states - no further reconciliation
        return ctrl.Result{}, nil

    default:
        log.Error(fmt.Errorf("unknown phase: %s", job.Status.Phase), "Invalid phase")
        return ctrl.Result{}, nil
    }
}

func (r *Reconciler) reconcileBuilding(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Check if build is enabled
    if !job.Spec.PodConfig.BuildConfig.Enabled {
        // Skip to validation
        job.Status.Phase = "ValidationRunning"
        if err := r.Status().Update(ctx, job); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }

    // Check build status
    buildInfo, err := r.getBuildStatus(ctx, job)
    if err != nil {
        return ctrl.Result{}, err
    }

    // Update build status
    job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
        Phase:          buildInfo.Phase,
        ImageReference: buildInfo.ImageReference,
        StartTime:      buildInfo.StartTime,
        CompletionTime: buildInfo.CompletionTime,
        Duration:       buildInfo.Duration,
    }

    // Check if build is complete
    if buildInfo.Phase == "Complete" {
        log.Info("Build completed successfully", "imageReference", buildInfo.ImageReference)

        // Transition to BuildComplete
        job.Status.Phase = "BuildComplete"

        // CRITICAL: Update the container image to use the built image
        job.Spec.ContainerImage = buildInfo.ImageReference

        if err := r.Status().Update(ctx, job); err != nil {
            return ctrl.Result{}, err
        }

        // Immediate requeue to start validation
        return ctrl.Result{Requeue: true}, nil
    }

    if buildInfo.Phase == "Failed" {
        log.Error(fmt.Errorf("build failed"), "Build failed")
        job.Status.Phase = "Failed"
        job.Status.Message = "Build failed: " + buildInfo.Message
        if err := r.Status().Update(ctx, job); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{}, nil
    }

    // Build still in progress - requeue after 30 seconds
    log.Info("Build in progress, waiting...", "buildPhase", buildInfo.Phase)
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *Reconciler) reconcileBuildComplete(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {
    // Build complete - start validation immediately
    job.Status.Phase = "ValidationRunning"
    if err := r.Status().Update(ctx, job); err != nil {
        return ctrl.Result{}, err
    }

    // Immediate requeue to start validation pod
    return ctrl.Result{Requeue: true}, nil
}
```

#### 3. Build Status Query Helper

```go
// internal/controller/build_integration_helper.go
func (r *Reconciler) getBuildStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
    strategy := job.Spec.PodConfig.BuildConfig.Strategy

    switch strategy {
    case "tekton":
        return r.getTektonBuildStatus(ctx, job)
    case "s2i":
        return r.getS2IBuildStatus(ctx, job)
    default:
        return nil, fmt.Errorf("unknown build strategy: %s", strategy)
    }
}

func (r *Reconciler) getTektonBuildStatus(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
    // Query Tekton PipelineRun
    pipelineRun := &tektonv1.PipelineRun{}
    pipelineRunName := fmt.Sprintf("%s-build", job.Name)

    err := r.Get(ctx, client.ObjectKey{
        Name:      pipelineRunName,
        Namespace: job.Namespace,
    }, pipelineRun)

    if err != nil {
        return nil, err
    }

    // Map Tekton status to BuildInfo
    buildInfo := &BuildInfo{
        StartTime: pipelineRun.Status.StartTime,
    }

    for _, condition := range pipelineRun.Status.Conditions {
        if condition.Type == "Succeeded" {
            if condition.Status == "True" {
                buildInfo.Phase = "Complete"
                buildInfo.CompletionTime = pipelineRun.Status.CompletionTime

                // Extract image reference from PipelineRun results
                for _, result := range pipelineRun.Status.PipelineResults {
                    if result.Name == "IMAGE_URL" {
                        buildInfo.ImageReference = result.Value.StringVal
                        break
                    }
                }
            } else if condition.Status == "False" {
                buildInfo.Phase = "Failed"
                buildInfo.Message = condition.Message
            } else {
                buildInfo.Phase = "Running"
            }
        }
    }

    // Calculate duration
    if buildInfo.StartTime != nil && buildInfo.CompletionTime != nil {
        duration := buildInfo.CompletionTime.Sub(buildInfo.StartTime.Time)
        buildInfo.Duration = duration.Round(time.Second).String()
    }

    return buildInfo, nil
}
```

## Consequences

### Positive
- ✅ **Eliminates race condition**: Validation only starts after build completes
- ✅ **Accurate validation**: Validation pod uses the exact built image with custom dependencies
- ✅ **Clear status tracking**: Users can see build progress in status
- ✅ **Production-ready**: No false negatives due to missing dependencies
- ✅ **Improved observability**: Separate build and validation durations tracked

### Negative
- ⏱️ **Increased total time**: Validation waits for build (unavoidable, but necessary)
- 🔄 **Increased complexity**: State machine logic more complex than previous parallel approach
- 📝 **CRD migration**: Existing jobs need status migration (handled by conversion webhook)

### Neutral
- 🔧 **Requeue intervals**: 30s requeue during build may need tuning for very fast/slow builds
- 📊 **Metrics impact**: Need to update metrics to track build and validation separately

## Implementation Notes

### Testing Strategy

#### Unit Tests
- [ ] Test state transitions with mock build statuses
- [ ] Test timeout enforcement at each phase
- [ ] Test error handling for build failures

#### Integration Tests
- [ ] Test with Tekton build strategy
- [ ] Test with S2I build strategy
- [ ] Test fallback to base image when build disabled

#### E2E Tests
- [ ] Create NotebookValidationJob with buildConfig.enabled=true
- [ ] Verify build starts before validation
- [ ] Verify validation uses built image
- [ ] Verify validation succeeds with custom dependencies
- [ ] Measure total time (build + validation)

### Rollout Strategy

#### Phase 1: CRD Update (Week 1)
- [ ] Add new status fields to CRD
- [ ] Run `make manifests generate`
- [ ] Deploy updated CRD to cluster
- [ ] Verify existing jobs not affected

#### Phase 2: Controller Implementation (Week 1-2)
- [ ] Implement state machine logic
- [ ] Add build status query helpers
- [ ] Update metrics collection
- [ ] Test with sample jobs

#### Phase 3: E2E Validation (Week 2)
- [ ] Run E2E tests against live cluster
- [ ] Measure performance impact
- [ ] Validate with production notebooks

### Migration Path

#### For Existing Users
1. **CRD Update**: Automatically handled by `kubectl apply -f config/crd/`
2. **Status Migration**: Existing jobs in terminal states (`Succeeded`/`Failed`) not affected
3. **Active Jobs**: Will transition to new state machine on next reconciliation
4. **Backward Compatibility**: Jobs without `buildConfig.enabled` skip directly to validation

### Performance Considerations

| Metric | Before (Parallel) | After (Sequential) | Change |
|--------|-------------------|-------------------|--------|
| Build Start | T+0s | T+0s | Same |
| Validation Start | T+0s | T+Build Duration | +Build Time |
| Total Time | max(Build, Validation) | Build + Validation | +Build Time |
| Success Rate | 0% (race condition) | 100% | +100% |

**Trade-off**: We accept increased total time to ensure correctness. This is the right trade-off for production use.

### Requeue Intervals

| Phase | Requeue Interval | Rationale |
|-------|-----------------|-----------|
| `Initializing` | 10s | Git clone typically fast |
| `Building` | 30s | Builds take 5-20 minutes, 30s is reasonable polling |
| `BuildComplete` | 0s (immediate) | Transition immediately to validation |
| `ValidationRunning` | 10s | Pod execution monitoring |

## References

- [OPERATOR-FEEDBACK.md](../../OPERATOR-FEEDBACK.md) - Bug #1: Race Condition
- [Kubernetes Controller Best Practices](https://kubernetes.io/docs/concepts/architecture/controller/)
- [Operator SDK Reconciliation Guide](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#reconcile-loop)

## Related ADRs

- [ADR-011: Error Handling and Retry Strategy](011-error-handling-and-retry-strategy.md) - Retry logic
- [ADR-023: S2I Build Integration](023-s2i-build-integration-openshift.md) - S2I build status
- [ADR-028: Tekton Task Strategy](028-tekton-task-strategy-custom-vs-cluster-tasks.md) - Tekton build status
- [ADR-038: Requirements.txt Auto-Detection](038-requirements-auto-detection.md) - Depends on build completion

## Revision History

| Date | Author | Description |
|------|--------|-------------|
| 2025-11-20 | Claude Code | Initial proposal based on production feedback |
