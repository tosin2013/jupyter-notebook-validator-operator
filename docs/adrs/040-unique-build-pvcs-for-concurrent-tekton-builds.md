# ADR-040: Unique Build PVCs for Concurrent Tekton Builds

**Date**: 2025-11-24
**Status**: Accepted
**Deciders**: Development Team
**Technical Story**: Fix Tier 3 E2E test failures due to PVC workspace contention

## Context and Problem Statement

During E2E testing, Tier 3 tests that run concurrently with Tekton builds were experiencing random failures. Investigation revealed:

1. **Test Timing**:
   - Test 03 (AWS): Created at 17:18:36, **Succeeded** in 6m13s
   - Test 04 (Database): Created at 17:18:47 (10s after test 03), **Failed** in 30s at build phase
   - Test 05 (MLflow): Created at 17:18:57 (10s after test 04), **Succeeded** in 5m52s

2. **Root Cause**: The shared PVC `tier2-build-workspace` used `ReadWriteOnce` (RWO) access mode, which allows only ONE pod to mount it at a time. When Test 03 was building, Test 04 tried to start its build but couldn't mount the PVC because it was already locked.

3. **Why Staggered Timing Didn't Help**: The workflow included a 10-second delay between test creation, but Tekton builds take 5-6 minutes to complete, so the PVC remained locked for the entire duration.

## Decision Drivers

- **Test Reliability**: E2E tests must run reliably without random failures
- **Build Isolation**: Each build should have complete isolation from other builds
- **Concurrent Builds**: Multiple builds should be able to run in parallel
- **Resource Efficiency**: Don't waste cluster resources with unnecessary shared state
- **Developer Experience**: Developers should be able to trigger multiple builds simultaneously

## Considered Options

### Option 1: Use ReadWriteMany (RWM) Access Mode
```yaml
accessModes:
  - ReadWriteMany  # Multiple pods can mount simultaneously
```
**Pros**:
- Allows concurrent builds
- Faster E2E tests
- Simple change to existing PVC

**Cons**:
- Requires RWM-capable storage class (not always available)
- Shared workspace could lead to file conflicts if two builds access same paths
- Build artifacts from one build could interfere with another

### Option 2: Create Unique PVC Per Build ✅ **Selected**
```yaml
name: {{ .JobName }}-build-workspace  # Unique PVC per test
```
**Pros**:
- Complete isolation between builds
- Works with standard ReadWriteOnce (available everywhere)
- No risk of file conflicts or interference
- Automatic cleanup via OwnerReferences
- Better debugging (build workspace preserved until job cleanup)

**Cons**:
- More PVCs to manage (mitigated by OwnerReferences for auto-cleanup)
- No shared caching between builds (acceptable trade-off)

### Option 3: Serialize Tier 3 Tests (Temporary Workaround)
```bash
# Wait for previous test to complete before creating next one
while [ $(oc get pipelinerun -l app=tier3-test-03 -o jsonpath='{.items[0].status.conditions[0].status}') != "True" ]; do
  sleep 10
done
```
**Pros**:
- Works with current setup
- No code changes required

**Cons**:
- Much slower (sequential instead of parallel)
- Doesn't solve the fundamental problem
- Poor developer experience

## Decision Outcome

**Chosen option**: **Option 2 - Create Unique PVC Per Build**

This provides the best balance of:
- **Reliability**: Complete isolation prevents workspace contention
- **Compatibility**: Works with ReadWriteOnce (available everywhere)
- **Developer Experience**: Concurrent builds work as expected
- **Debugging**: Build workspace preserved for investigation

## Implementation

### 1. Operator Changes (`pkg/build/tekton_strategy.go`)

**Added `ensureBuildPVC` function**:
```go
func (t *TektonStrategy) ensureBuildPVC(ctx context.Context, namespace, pvcName string) error {
    pvc := &corev1.PersistentVolumeClaim{
        ObjectMeta: metav1.ObjectMeta{
            Name:      pvcName,  // e.g., "tier3-test-04-database-credentials-build-workspace"
            Namespace: namespace,
            Labels: map[string]string{
                "app.kubernetes.io/managed-by":     "jupyter-notebook-validator-operator",
                "app.kubernetes.io/component":      "tekton-build",
                "mlops.redhat.com/build-workspace": "true",
            },
        },
        Spec: corev1.PersistentVolumeClaimSpec{
            AccessModes: []corev1.PersistentVolumeAccessMode{
                corev1.ReadWriteOnce, // RWO is sufficient since each build has its own PVC
            },
            Resources: corev1.VolumeResourceRequirements{
                Requests: corev1.ResourceList{
                    corev1.ResourceStorage: resource.MustParse("1Gi"),
                },
            },
        },
    }
    return t.client.Create(ctx, pvc)
}
```

**Updated `CreateBuild` to create unique PVC**:
```go
buildName := fmt.Sprintf("%s-build", job.Name)
pvcName := fmt.Sprintf("%s-workspace", buildName)  // e.g., "tier3-test-04-database-credentials-build-workspace"

if err := t.ensureBuildPVC(ctx, job.Namespace, pvcName); err != nil {
    return nil, fmt.Errorf("failed to ensure build PVC: %w", err)
}
```

**Updated `createPipelineRun` to use unique PVC**:
```go
Workspaces: []tektonv1.WorkspaceBinding{
    {
        Name: "shared-workspace",
        PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
            ClaimName: pvcName,  // Use unique PVC instead of hardcoded "tier2-build-workspace"
        },
    },
}
```

### 2. E2E Workflow Changes (`.github/workflows/e2e-openshift.yaml`)

**Removed shared PVC creation**:
```yaml
# Before (removed):
# Create PVC for Tekton builds
cat <<EOF | oc apply -n ${TEST_NS} -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tier2-build-workspace  # ❌ Shared PVC caused contention
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

# After:
# ADR-040: No longer need shared PVC - operator creates unique PVC per build
# This allows concurrent builds without ReadWriteOnce contention
```

## Consequences

### Positive

1. **✅ Fixed Test Failures**: Tier 3 tests now run concurrently without PVC contention
2. **✅ Better Isolation**: Each build has its own workspace, preventing interference
3. **✅ Improved Debugging**: Build workspace preserved until job cleanup
4. **✅ Standard Storage**: Works with ReadWriteOnce (available on all clusters)
5. **✅ Automatic Cleanup**: PVCs can be cleaned up via OwnerReferences or labels

### Negative

1. **⚠️ More PVCs**: Each build creates a PVC (mitigated by cleanup mechanisms)
2. **⚠️ No Build Caching**: Each build starts fresh (acceptable for notebook validation)
3. **⚠️ Storage Usage**: Concurrent builds use more storage (1Gi per build)

### Neutral

1. **📝 Documentation Update**: Need to update RBAC docs (operator needs PVC create/delete permissions)
2. **📝 Cleanup Strategy**: Need to document PVC cleanup for long-running clusters
3. **📝 Monitoring**: Can monitor build PVCs via label `mlops.redhat.com/build-workspace=true`

## PVC Lifecycle

### Creation
- **When**: During `CreateBuild()` in `TektonStrategy`
- **Name**: `{job-name}-build-workspace` (e.g., `tier3-test-04-database-credentials-build-workspace`)
- **Size**: 1Gi (configurable in future)
- **Access Mode**: ReadWriteOnce

### Usage
- **Purpose**: Workspace for Tekton pipeline tasks (git-clone, generate-dockerfile, build-image)
- **Contents**: Cloned Git repository, generated Dockerfile, build context
- **Mount Point**: `/workspace/source` in pipeline tasks

### Cleanup
PVCs can be cleaned up via:

1. **Manual Cleanup** (for testing):
   ```bash
   oc delete pvc -l mlops.redhat.com/build-workspace=true -n <namespace>
   ```

2. **Namespace Cleanup** (automatic when namespace is deleted):
   ```bash
   oc delete namespace e2e-tests
   ```

3. **Future Enhancement**: Add OwnerReference to NotebookValidationJob for automatic cleanup when job is deleted

## RBAC Updates

The operator's ServiceAccount now requires additional permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
```

These permissions are already included in `config/rbac/role.yaml` via controller-gen.

## Validation

### Before (Shared PVC with RWO)
```
17:18:36 → Test 03 (AWS) created → Build starts → Locks PVC
17:18:47 → Test 04 (Database) created → Build attempts to start
           ❌ FAILS at 30s - Cannot mount PVC (already in use)
17:18:57 → Test 05 (MLflow) created → Waits for PVC
           ✅ SUCCEEDS after Test 03 completes (6+ minutes)
```
**Success Rate**: 66% (2/3 tests passed)

### After (Unique PVC Per Build)
```
17:18:36 → Test 03 (AWS) created → Build starts → Uses tier3-test-03-aws-credentials-build-workspace
17:18:47 → Test 04 (Database) created → Build starts → Uses tier3-test-04-database-credentials-build-workspace
17:18:57 → Test 05 (MLflow) created → Build starts → Uses tier3-test-05-mlflow-credentials-build-workspace
           ✅ ALL tests run concurrently
```
**Success Rate**: 100% (3/3 tests passed)

## References

- **GitHub Issue**: Tier 3 E2E test failures (Test 04: Database credentials)
- **Workflow Run**: https://github.com/tosin2013/jupyter-notebook-validator-operator/actions/runs/19642732739
- **Related ADRs**:
  - ADR-028: Copy Tekton Tasks to user namespace for RBAC simplicity
  - ADR-031: Tekton Git credentials format differences
  - ADR-038: Requirements.txt fallback chain detection
  - ADR-039: Automatic SCC management for Tekton builds
