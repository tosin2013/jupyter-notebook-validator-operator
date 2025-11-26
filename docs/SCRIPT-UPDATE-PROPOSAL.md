# Script Update Proposal: test-local-kind.sh for ADR-037

**Date**: 2025-11-20
**Purpose**: Update `scripts/test-local-kind.sh` to test ADR-037 state machine with Tekton builds

---

## What Changed (Week 1, Day 1-9)

### ✅ Completed Changes

1. **New Phase Constants** (Day 1-2):
   - `Initializing`, `Building`, `BuildComplete`, `ValidationRunning`, `Succeeded`, `Failed`
   - Legacy: `Pending`, `Running`

2. **BuildStatus Enhancement** (Day 1-2):
   - Added `duration` field (human-readable, e.g., "5m30s")
   - Existing fields: `phase`, `imageReference`, `startTime`, `completionTime`

3. **State Machine Implementation** (Day 3-5):
   - Non-blocking reconciliation with 30s requeue during builds
   - BuildStatus as single source of truth for image reference

4. **Unit Tests** (Day 8-9):
   - 9 tests covering state transitions, requeue logic, legacy migration
   - All passing ✅

---

## Current Script Limitations

### 1. Phase Monitoring
**Current**: Only checks `Succeeded` or `Failed`
```bash
case "$phase" in
    "Succeeded")
        log_success "✅ Test passed: $notebook"
        ;;
    "Failed")
        log_error "❌ Test failed: $notebook"
        ;;
esac
```

**Issue**: Doesn't observe intermediate phases (Initializing, Building, BuildComplete, ValidationRunning)

---

### 2. Build Testing Coverage
**Current**: Only runs Tier 1 tests (simple notebooks, no builds)
```bash
local tier1_notebooks=(
    "notebooks/tier1-simple/01-hello-world.ipynb"
    "notebooks/tier1-simple/02-basic-math.ipynb"
    "notebooks/tier1-simple/03-data-validation.ipynb"
)
```

**Issue**: Doesn't test the build workflow that ADR-037 fixes

---

### 3. Timeout Values
**Current**: 120s (2 minutes) for Tier 1 tests
```bash
local timeout=120
```

**Issue**: Build tests need 5-10 minutes (Tekton pipeline execution)

---

### 4. Tekton Prerequisites
**Current**: Installs cert-manager only
```bash
install_cert_manager() {
    # ... cert-manager installation
}
```

**Issue**: Doesn't install Tekton CRDs/operators needed for Tekton builds

---

## Recommended Updates

### Update 1: Add Tekton Installation Function

```bash
# Install Tekton Pipelines
install_tekton() {
    log_info "Installing Tekton Pipelines for build testing..."

    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    # Install Tekton Pipelines v0.53.0 (compatible with Kubernetes 1.31)
    $KUBECTL_CMD apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.53.0/release.yaml

    # Wait for Tekton to be ready
    log_info "Waiting for Tekton to be ready..."
    $KUBECTL_CMD wait --for=condition=Available --timeout=300s \
        -n tekton-pipelines deployment/tekton-pipelines-controller \
        deployment/tekton-pipelines-webhook

    log_success "Tekton installed successfully"
}
```

**Call in main()**: After `install_cert_manager`, add:
```bash
# Install Tekton (for build testing)
install_tekton
```

---

### Update 2: Add Build Test Function

```bash
# Run build integration tests (ADR-037)
run_build_tests() {
    log_info "Running build integration tests (ADR-037 - Tekton)..."

    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    local job_name="build-test-tekton-seaborn"
    local failed=0

    log_info "Testing: Tekton build with custom requirements (seaborn)"

    # Create NotebookValidationJob with Tekton build
    cat <<EOF | $KUBECTL_CMD apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: ${job_name}
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_REF}"
    path: "notebooks/tier2-data/01-pandas-analysis.ipynb"
  podConfig:
    buildConfig:
      enabled: true
      strategy: "tekton"
      baseImage: "quay.io/jupyter/minimal-notebook:latest"
      requirementsFile: "requirements.txt"
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    serviceAccountName: jupyter-notebook-validator-runner
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "2Gi"
        cpu: "2000m"
  timeout: "15m"
EOF

    # Wait for job to complete (max 10 minutes for build + validation)
    local timeout=600
    local elapsed=0
    local interval=10
    local last_phase=""

    while [ $elapsed -lt $timeout ]; do
        local phase=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        # Show phase transitions
        if [ "$phase" != "$last_phase" ]; then
            log_info "Phase transition: $last_phase -> $phase"
            last_phase="$phase"

            # Show BuildStatus details when in Building/BuildComplete
            if [[ "$phase" == "Building" ]] || [[ "$phase" == "BuildComplete" ]]; then
                local build_phase=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.buildStatus.phase}' 2>/dev/null || echo "Unknown")
                local build_duration=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.buildStatus.duration}' 2>/dev/null || echo "N/A")
                log_info "  BuildStatus: phase=$build_phase, duration=$build_duration"
            fi
        fi

        case "$phase" in
            "Succeeded")
                log_success "✅ Build test passed"

                # Verify BuildStatus was set
                local image_ref=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.buildStatus.imageReference}' 2>/dev/null)
                if [ -n "$image_ref" ]; then
                    log_success "  Built image: $image_ref"
                else
                    log_warning "  BuildStatus.imageReference not set (unexpected)"
                fi

                return 0
                ;;
            "Failed")
                log_error "❌ Build test failed"
                $KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
                return 1
                ;;
            *)
                sleep $interval
                elapsed=$((elapsed + interval))
                ;;
        esac
    done

    if [ $elapsed -ge $timeout ]; then
        log_error "❌ Build test timeout (phase: $phase)"
        $KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
        return 1
    fi
}
```

**Call in main()**: After `run_tier1_tests`, add:
```bash
# Run build integration tests (ADR-037)
if run_build_tests; then
    log_success "🎉 Build tests passed!"
    BUILD_TEST_RESULT=0
else
    log_error "❌ Build tests failed"
    BUILD_TEST_RESULT=1
    TEST_RESULT=1
fi
```

---

### Update 3: Add Verbose Phase Monitoring (Optional)

Add flag to enable verbose phase logging:

```bash
VERBOSE_PHASES=false

# Parse arguments (add to existing case statement)
--verbose-phases)
    VERBOSE_PHASES=true
    shift
    ;;
```

Update wait loop in `run_tier1_tests()`:
```bash
while [ $elapsed -lt $timeout ]; do
    local phase=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

    # Verbose phase logging
    if [ "$VERBOSE_PHASES" = true ] && [ "$phase" != "$last_phase" ]; then
        log_info "  Phase: $phase"
        last_phase="$phase"
    fi

    # ... rest of switch statement
done
```

---

## Testing Strategy

### Option A: Full Test (Recommended for CI)
Run both Tier 1 (simple) and Build tests:
```bash
./scripts/test-local-kind.sh --skip-cleanup
```

### Option B: Build Tests Only (Day 10 Focus)
Add a `--build-tests-only` flag:
```bash
./scripts/test-local-kind.sh --build-tests-only --skip-cleanup
```

### Option C: Interactive/Debug Mode
Run setup, then manually test:
```bash
./scripts/test-local-kind.sh --skip-cleanup
kubectl apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml
kubectl get notebookvalidationjob -w
```

---

## Summary of Changes

| Section | Change | Priority | Complexity |
|---------|--------|----------|------------|
| Add `install_tekton()` | Install Tekton Pipelines for Kind | 🔴 High | Low |
| Add `run_build_tests()` | Test ADR-037 state machine | 🔴 High | Medium |
| Update main() flow | Call new functions | 🔴 High | Low |
| Add `--verbose-phases` | Optional phase logging | 🟡 Medium | Low |
| Add `--build-tests-only` | Skip Tier 1 tests | 🟢 Low | Low |

---

## Recommended Execution Plan

1. **Review this proposal** ✅ (You are here)
2. **Make updates** to `scripts/test-local-kind.sh`
3. **Test locally** with `--skip-cleanup` flag
4. **Verify** ADR-037 state machine phases
5. **Create commit** for Week 1 work
6. **Push to GitHub** for CI testing

---

## Questions for Discussion

1. **Tekton version**: Use v0.53.0 or latest v0.62.0?
2. **Test complexity**: Start with Tier 1 + one build test, or comprehensive build testing?
3. **Cleanup strategy**: Keep cluster for debugging or clean up automatically?
4. **CI integration**: Update GitHub Actions workflow to use new script?

---

**Next Steps**: Should I proceed with implementing these updates?
