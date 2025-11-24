# ADR-042: Fix S2I Build Status Monitoring

**Date**: 2025-11-24
**Status**: Proposed
**Deciders**: Development Team
**Technical Story**: Fix bug where S2I builds stuck in "Building" phase even when failed

## Context and Problem Statement

During Tier 4 E2E testing with S2I/BuildConfig strategy, discovered that NotebookValidationJobs remain stuck in "Building" phase even when the actual OpenShift Build fails.

### Observed Behavior

```bash
$ oc get notebookvalidationjobs -n e2e-tests -l tier=4
NAME                                    PHASE      NOTEBOOK                                               AGE
tier4-test-01-s2i-python-build          Building   notebooks/tier4-s2i/01-s2i-python-build.ipynb          5m10s
tier4-test-02-s2i-custom-requirements   Building   notebooks/tier4-s2i/02-s2i-custom-requirements.ipynb   5m

$ oc get builds -n e2e-tests
NAME                                            TYPE     FROM                     STATUS                       STARTED         DURATION
tier4-test-01-s2i-python-build-build-1          Docker   Dockerfile,Git@b53bb5b   Failed (DockerBuildFailed)   5 minutes ago   2m25s
tier4-test-02-s2i-custom-requirements-build-1   Docker   Dockerfile,Git@b53bb5b   Failed (DockerBuildFailed)   5 minutes ago   2m44s
```

**Problem**: NotebookValidationJob shows "Building" but actual Build shows "Failed"

### Root Cause

**Controller calls wrong method**: `notebookvalidationjob_controller.go:356`

```go
buildName := fmt.Sprintf("%s-build", job.Name)
buildInfo, err := strategy.GetBuildStatus(ctx, buildName)  // ❌ WRONG for S2I!
```

**Why it fails**:
1. BuildConfig name: `tier4-test-01-s2i-python-build-build`
2. Actual Build name: `tier4-test-01-s2i-python-build-build-1` (with `-1` suffix)
3. `GetBuildStatus()` looks for exact match → "build not found"
4. Controller thinks build doesn't exist → creates new BuildConfig (already exists) → loops forever

**Operator logs show the loop**:
```
Build not found, creating new build
BuildConfig already exists
BuildConfig created with ConfigChange trigger - build will start automatically
Build created, requeuing to check status
Build not found, creating new build  // ← LOOP!
```

## Decision Drivers

- **Build failure detection**: Must detect when Build fails and update job status
- **Build discovery**: Must find builds created by BuildConfig (with `-1`, `-2` suffixes)
- **Strategy-specific logic**: S2I needs different lookup than Tekton
- **Existing method**: `GetLatestBuild()` already exists and works correctly

## Decision Outcome

**Fix controller to use `GetLatestBuild()` for S2I builds**

### Solution

**Change**: `internal/controller/notebookvalidationjob_controller.go:356`

```go
// Before (BROKEN):
buildInfo, err := strategy.GetBuildStatus(ctx, buildName)

// After (FIXED):
buildInfo, err := strategy.GetLatestBuild(ctx, buildName)
```

**Why this works**:
- `GetLatestBuild()` looks for builds with `buildconfig=<name>` label
- Finds actual Build resources created by BuildConfig
- Returns most recent build (handles multiple builds)
- Properly detects Failed/Complete/Running states

### Implementation

**File**: `internal/controller/notebookvalidationjob_controller.go`

```go
func (r *NotebookValidationJobReconciler) reconcileBuilding(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	strategy build.Strategy) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.Info("Reconciling Building phase")

	buildName := fmt.Sprintf("%s-build", job.Name)

	// Use GetLatestBuild for build discovery
	// This works for both Tekton (PipelineRuns) and S2I (Builds)
	buildInfo, err := strategy.GetLatestBuild(ctx, buildName)

	if err != nil {
		// Build doesn't exist yet, create it
		logger.Info("Build not found, creating new build", "buildName", buildName)
		buildInfo, err = strategy.CreateBuild(ctx, job)
		if err != nil {
			return r.updateStatusWithError(ctx, job, "BuildCreationFailed",
				fmt.Sprintf("Failed to create build: %v", err))
		}

		// Requeue to check build status
		logger.Info("Build created, requeuing to check status", "buildName", buildInfo.Name)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Build exists, check status
	logger.Info("Checking build status", "buildName", buildInfo.Name, "status", buildInfo.Status)

	switch buildInfo.Status {
	case build.BuildStatusComplete:
		logger.Info("Build completed successfully", "imageReference", buildInfo.ImageReference)
		// ... transition to Running phase ...

	case build.BuildStatusFailed, build.BuildStatusCancelled:
		// ✅ NOW DETECTS FAILURES!
		logger.Info("Build failed", "status", buildInfo.Status, "message", buildInfo.Message)
		return r.updateStatusWithError(ctx, job, "BuildFailed",
			fmt.Sprintf("Build failed: %s", buildInfo.Message))

	case build.BuildStatusRunning, build.BuildStatusPending:
		logger.V(1).Info("Build still in progress", "status", buildInfo.Status)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

	default:
		logger.Info("Unknown build status", "status", buildInfo.Status)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}
```

## Consequences

### Positive

1. ✅ **Build failures detected**: Jobs transition to Failed when Build fails
2. ✅ **Works for both strategies**: `GetLatestBuild()` works for Tekton and S2I
3. ✅ **Handles multiple builds**: Prioritizes Complete > Running > Pending > Failed
4. ✅ **Better error messages**: Users see actual build failure reasons
5. ✅ **No infinite loops**: Controller stops retrying after build fails

### Negative

1. ⚠️ **Slight behavior change**: Uses "latest" build instead of exact name match
2. ⚠️ **Multiple builds**: If user manually triggers multiple builds, uses most recent

### Neutral

1. 📝 **Test coverage**: Add E2E test for build failure scenarios
2. 📝 **Monitoring**: Watch for any regressions in Tekton builds (already using GetLatestBuild)
3. 📝 **Documentation**: Update ARCHITECTURE_OVERVIEW.md with build status monitoring

## Validation

### Before Fix

```bash
# Create S2I job with bad requirements (Python 3.13 incompatible)
$ oc apply -f tier4-test-01.yaml

# Wait 5 minutes...
$ oc get notebookvalidationjobs
NAME                          PHASE      NOTEBOOK
tier4-test-01                 Building   notebooks/tier4-s2i/01-...

$ oc get builds
NAME                           STATUS    Started
tier4-test-01-build-build-1    Failed    5m ago

# ❌ Job stuck in Building, never detects failure
```

### After Fix

```bash
# Create S2I job with bad requirements
$ oc apply -f tier4-test-01.yaml

# Wait for build to fail (~2-3 minutes)
$ oc get notebookvalidationjobs
NAME                          PHASE    NOTEBOOK
tier4-test-01                 Failed   notebooks/tier4-s2i/01-...

$ oc describe notebookvalidationjob tier4-test-01
Status:
  Phase: Failed
  Message: Build failed: error building at STEP "RUN pip install..."
  Build Status:
    Phase: Failed
    Duration: 2m25s

# ✅ Job correctly shows Failed status with build error details
```

## References

- **Build Strategy Interface**: `pkg/build/strategy.go`
- **S2I Strategy**: `pkg/build/s2i_strategy.go:362` (`GetLatestBuild` implementation)
- **Tekton Strategy**: `pkg/build/tekton_strategy.go:1032` (`GetLatestBuild` implementation)
- **Controller**: `internal/controller/notebookvalidationjob_controller.go:356`
- **Related ADRs**:
  - ADR-040: Unique Build PVCs for Concurrent Tekton Builds
  - ADR-039: Automatic SCC management for Tekton builds
  - ADR-038: Requirements.txt auto-detection

## Notes

This bug only affected S2I builds because:
- Tekton creates PipelineRuns with exact names matching Pipeline name
- S2I creates Builds with suffixes (`-1`, `-2`) appended to BuildConfig name
- `GetBuildStatus()` expects exact name match
- `GetLatestBuild()` uses labels to find builds regardless of suffix
