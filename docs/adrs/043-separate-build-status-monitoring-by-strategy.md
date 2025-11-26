# ADR-043: Separate Build Status Monitoring by Strategy

**Date**: 2025-11-24
**Status**: Implemented
**Deciders**: Development Team
**Technical Story**: Split build status monitoring into strategy-specific functions to handle naming differences

## Context and Problem Statement

During Tier 2 E2E testing with Tekton strategy, discovered that NotebookValidationJobs remain stuck in "Building" phase forever, even when the PipelineRun has completed successfully.

### Observed Behavior

```bash
$ oc get notebookvalidationjobs -n e2e-tests -l tier=2
NAME                          PHASE      NOTEBOOK
tier2-test-01-sentiment-model Building   notebooks/tier2-intermediate/01-...

$ oc get pipelineruns -n e2e-tests
NAME                                  SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
tier2-test-01-sentiment-model-build   True        Succeeded   11m         6m8s
```

**Problem**: NotebookValidationJob stuck in "Building" but PipelineRun shows "Succeeded" (completed 6 minutes ago)

### Root Cause

**Generic interface doesn't fit both strategies**:

The controller used a single `reconcileBuilding()` function that tried to use the same logic for both S2I and Tekton:

```go
buildName := fmt.Sprintf("%s-build", job.Name)  // "tier2-test-01-sentiment-model-build"
buildInfo, err := strategy.GetLatestBuild(ctx, buildName)
```

**Why this failed**:

**S2I/BuildConfig**:
- BuildConfig name: `tier2-test-01-sentiment-model-build`
- Creates Builds: `tier2-test-01-sentiment-model-build-1`, `-2`, etc.
- `GetLatestBuild()` searches by label: `buildconfig=tier2-test-01-sentiment-model-build`
- ✅ **Works** - finds builds by BuildConfig label

**Tekton/Pipeline**:
- Pipeline name: `tier2-test-01-sentiment-model-pipeline`
- PipelineRun name: `tier2-test-01-sentiment-model-build`
- `GetLatestBuild()` searches by label: `tekton.dev/pipeline=<passed-name>`
- ❌ **Fails** - controller passes `tier2-test-01-sentiment-model-build`, but pipeline label is `tier2-test-01-sentiment-model-pipeline`

**Operator logs showed infinite loop**:
```
INFO  Build not found, creating new build  buildName=tier2-test-01-sentiment-model-build
INFO  Build created, requeuing to check status
[30 seconds later]
INFO  Build not found, creating new build  ← LOOP!
```

The controller never found the completed PipelineRun, so it kept trying to create new builds.

## Decision Drivers

- **Naming conventions differ**: S2I uses BuildConfig name, Tekton uses Pipeline name
- **Label-based lookups**: Each strategy uses different labels for discovery
- **Strategy interface limitations**: Generic interface forces incompatible patterns
- **Code clarity**: Strategy-specific logic easier to understand and maintain

## Decision Outcome

**Split build status monitoring into strategy-specific functions**

### Architecture

```
reconcileBuilding()
  ├─ Dispatcher based on strategy
  ├─> reconcileBuildingS2I()    ← S2I-specific logic
  ├─> reconcileBuildingTekton()  ← Tekton-specific logic
  └─> handleBuildStatus()        ← Common status handling
```

### Implementation

**File**: `internal/controller/notebookvalidationjob_controller.go`

#### 1. Dispatcher Function

```go
func (r *NotebookValidationJobReconciler) reconcileBuilding(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.Info("Reconciling Building phase")

	// Initialize build status if needed
	if job.Status.BuildStatus == nil {
		job.Status.BuildStatus = &mlopsv1alpha1.BuildStatus{
			Phase:     "Pending",
			Message:   "Build initialization",
			StartTime: &metav1.Time{Time: time.Now()},
		}
		if err := r.Status().Update(ctx, job); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get the configured strategy
	strategyName := job.Spec.PodConfig.BuildConfig.Strategy
	if strategyName == "" {
		strategyName = "s2i" // Default to S2I
	}

	// ADR-043: Dispatch to strategy-specific reconciliation
	switch strategyName {
	case "s2i":
		return r.reconcileBuildingS2I(ctx, job)
	case "tekton":
		return r.reconcileBuildingTekton(ctx, job)
	default:
		return r.transitionPhase(ctx, job, PhaseFailed,
			fmt.Sprintf("Unknown build strategy: %s", strategyName))
	}
}
```

#### 2. S2I-Specific Function

```go
func (r *NotebookValidationJobReconciler) reconcileBuildingS2I(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.Info("Reconciling S2I build")

	registry := build.NewStrategyRegistry(r.Client, r.APIReader, r.Scheme)
	strategy := registry.GetStrategy("s2i")
	if strategy == nil {
		return r.transitionPhase(ctx, job, PhaseFailed,
			"S2I build strategy not available")
	}

	// S2I: BuildConfig creates Builds with -1, -2 suffixes
	// Use GetLatestBuild to find by buildconfig label
	buildName := fmt.Sprintf("%s-build", job.Name)
	buildInfo, err := strategy.GetLatestBuild(ctx, buildName)

	return r.handleBuildStatus(ctx, job, strategy, "s2i", buildName, buildInfo, err)
}
```

#### 3. Tekton-Specific Function

```go
func (r *NotebookValidationJobReconciler) reconcileBuildingTekton(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.Info("Reconciling Tekton build")

	registry := build.NewStrategyRegistry(r.Client, r.APIReader, r.Scheme)
	strategy := registry.GetStrategy("tekton")
	if strategy == nil {
		return r.transitionPhase(ctx, job, PhaseFailed,
			"Tekton build strategy not available")
	}

	// Tekton: PipelineRuns are named {job-name}-build,
	// but Pipeline is {job-name}-pipeline
	// GetLatestBuild searches by tekton.dev/pipeline label,
	// so we need to pass pipeline name
	pipelineName := fmt.Sprintf("%s-pipeline", job.Name)
	buildInfo, err := strategy.GetLatestBuild(ctx, pipelineName)

	// For CreateBuild, we still use buildName format
	buildName := fmt.Sprintf("%s-build", job.Name)
	return r.handleBuildStatus(ctx, job, strategy, "tekton", buildName, buildInfo, err)
}
```

#### 4. Common Status Handler

```go
func (r *NotebookValidationJobReconciler) handleBuildStatus(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	strategy build.Strategy,
	strategyName string,
	buildName string,
	buildInfo *build.BuildInfo,
	err error) (ctrl.Result, error) {

	// Common logic for:
	// - Build creation if not found
	// - Status checking (Complete, Failed, Pending, Running)
	// - Status updates
	// - Phase transitions
	// ... (same as before, but now called by both strategies)
}
```

## Consequences

### Positive

1. ✅ **Tekton builds now detected**: Jobs correctly transition from Building → BuildComplete
2. ✅ **S2I builds still work**: No regression in S2I functionality (verified by ADR-042 fix)
3. ✅ **Strategy-specific logic**: Each strategy has its own lookup pattern
4. ✅ **Code clarity**: Easier to understand and maintain than generic interface
5. ✅ **Extensibility**: Adding new build strategies is clearer (create new function)
6. ✅ **No infinite loops**: Controller finds completed builds immediately

### Negative

1. ⚠️ **Code duplication**: Some duplication between S2I and Tekton functions (mitigated by common handler)
2. ⚠️ **More functions**: Three functions instead of one (but each is clearer)

### Neutral

1. 📝 **Future strategies**: Will need their own reconciliation functions
2. 📝 **Testing**: Need to test both S2I and Tekton paths independently
3. 📝 **Documentation**: Update ARCHITECTURE_OVERVIEW.md with new flow

## Validation

### Before Fix (Tekton)

```bash
# Create Tekton job
$ oc apply -f tier2-test-01.yaml

# PipelineRun completes successfully
$ oc get pipelineruns
NAME                                  SUCCEEDED   COMPLETIONTIME
tier2-test-01-sentiment-model-build   True        6m ago

# But job stuck in Building forever
$ oc get notebookvalidationjobs
NAME                          PHASE      AGE
tier2-test-01-sentiment-model Building   12m

# ❌ Job never detects build completion
```

### After Fix (Tekton)

```bash
# Create Tekton job
$ oc apply -f tier2-test-01.yaml

# PipelineRun completes
$ oc get pipelineruns
NAME                                  SUCCEEDED   COMPLETIONTIME
tier2-test-01-sentiment-model-build   True        3m ago

# Job correctly detects completion
$ oc get notebookvalidationjobs
NAME                          PHASE          AGE
tier2-test-01-sentiment-model BuildComplete  5m

# ✅ Job transitioned to BuildComplete, validation starting
```

### S2I Still Works (Regression Test)

```bash
# S2I builds from ADR-042 still work correctly
$ oc get notebookvalidationjobs -l tier=4
NAME                                  PHASE       NOTEBOOK
tier4-test-01-s2i-python-build        Succeeded   notebooks/tier4-s2i/01-...
tier4-test-02-s2i-custom-requirements Succeeded   notebooks/tier4-s2i/02-...

# ✅ No regression in S2I functionality
```

## Comparison: ADR-042 vs ADR-043

### ADR-042 (S2I Fix)
- **Problem**: S2I Builds with `-1` suffixes not found
- **Solution**: Use `GetLatestBuild()` instead of `GetBuildStatus()`
- **Scope**: S2I only

### ADR-043 (Strategy Separation)
- **Problem**: Tekton naming doesn't match S2I naming
- **Solution**: Split into strategy-specific functions
- **Scope**: Both S2I and Tekton
- **Builds on**: ADR-042's insights about lookup patterns

## References

- **Controller**: `internal/controller/notebookvalidationjob_controller.go:322-545`
- **S2I Strategy**: `pkg/build/s2i_strategy.go:362` (`GetLatestBuild` implementation)
- **Tekton Strategy**: `pkg/build/tekton_strategy.go:1032` (`GetLatestBuild` implementation)
- **Related ADRs**:
  - ADR-042: Fix S2I Build Status Monitoring (GetLatestBuild for S2I)
  - ADR-040: Unique Build PVCs for Concurrent Tekton Builds
  - ADR-039: Automatic SCC management for Tekton builds

## Notes

**Why not fix the interface instead?**

We considered making the interface smarter (e.g., auto-detect naming patterns), but:
1. Each strategy has fundamentally different lookup logic
2. Hiding complexity in the interface makes debugging harder
3. Explicit strategy-specific functions are clearer and more maintainable

**Strategy-specific logic belongs in strategy-specific functions**, not forced through a generic interface.
