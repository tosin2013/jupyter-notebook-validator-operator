# External Volume Support Status - release-4.18

**Date**: 2025-12-02
**Branch**: release-4.18
**Status**: ✅ **SUPPORTED** (v1.0.5+)

## Summary

The `release-4.18` branch **NOW SUPPORTS** attaching external volumes (PVC, ConfigMap, Secret, EmptyDir) to NotebookValidationJob validation pods as of version **1.0.5-ocp4.18-volumes**.

### Validation Results

**Test Date**: 2025-12-02
**Image**: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes`
**Test Job**: `test-external-volumes`
**Result**: ✅ **PASSED** - All volume types mounted successfully

**Volumes Tested**:
- ✅ PersistentVolumeClaim (PVC) - Mounted at `/models`
- ✅ ConfigMap - Mounted at `/config` (read-only)
- ✅ EmptyDir with size limit - Mounted at `/tmp/data`

**Notebook Execution**: 8/8 cells succeeded (100% success rate)

## Current Volume Support

### ✅ Supported: Tekton Workspace Volumes

**What Works**: Tekton builds automatically create and mount PVC volumes for build workspaces.

**Example**:
```yaml
spec:
  podConfig:
    buildConfig:
      enabled: true
      strategy: "tekton"
```

**Result**: 
- PVC created: `<job-name>-build-workspace`
- Mounted in Tekton TaskRun pods at `/workspace/source`
- Used for build artifacts and intermediate files

### ❌ Not Supported: External Volume Mounting

**What Doesn't Work**: Users cannot attach their own volumes to validation pods.

**Missing CRD Fields**:
```yaml
# This does NOT exist in release-4.18
spec:
  podConfig:
    volumes: []        # ❌ Not available
    volumeMounts: []   # ❌ Not available
```

## Code Evidence

### Current PodConfigSpec (api/v1alpha1/notebookvalidationjob_types.go)

```go
type PodConfigSpec struct {
    ContainerImage string `json:"containerImage"`
    Resources *ResourceRequirements `json:"resources,omitempty"`
    ServiceAccountName string `json:"serviceAccountName,omitempty"`
    Env []EnvVar `json:"env,omitempty"`
    EnvFrom []EnvFromSource `json:"envFrom,omitempty"`
    Credentials []string `json:"credentials,omitempty"`
    BuildConfig *BuildConfigSpec `json:"buildConfig,omitempty"`
    
    // ❌ NO volume fields
}
```

### Hardcoded Volumes in Validation Pods

From `internal/controller/notebookvalidationjob_controller.go` (lines 817-836):

```go
Volumes: []corev1.Volume{
    {
        Name: "workspace",
        VolumeSource: corev1.VolumeSource{
            EmptyDir: &corev1.EmptyDirVolumeSource{},
        },
    },
    {
        Name: "jovyan-home",
        VolumeSource: corev1.VolumeSource{
            EmptyDir: &corev1.EmptyDirVolumeSource{},
        },
    },
    // ❌ No user-defined volumes
}
```

## Test Notebooks Requiring Volume Support

The following test notebooks **cannot be run** on release-4.18:

### tier3-complex/06-pvc-model-output-test.ipynb

**Requirements**:
- PVC named `trained-models-pvc` mounted at `/models`
- Write trained ML models to persistent storage
- Enable KServe to load models from `pvc://` storageUri

**Status**: ❌ Cannot run - no PVC mounting support

### tier3-complex/07-shared-dataset-volumes-test.ipynb

**Requirements**:
- PVC `shared-datasets-pvc` mounted at `/data` (read-only)
- ConfigMap `notebook-config` mounted at `/config`
- EmptyDir mounted at `/scratch`

**Status**: ❌ Cannot run - no volume mounting support

## What Would Be Needed

To support external volumes, the following changes would be required:

### 1. CRD Changes

```go
type PodConfigSpec struct {
    // ... existing fields ...
    
    // Volumes to mount in the validation pod
    // +optional
    Volumes []Volume `json:"volumes,omitempty"`
    
    // VolumeMounts for the validation container
    // +optional
    VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}

type Volume struct {
    Name string `json:"name"`
    VolumeSource VolumeSource `json:"volumeSource"`
}

type VolumeSource struct {
    PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
    ConfigMap *ConfigMapVolumeSource `json:"configMap,omitempty"`
    Secret *SecretVolumeSource `json:"secret,omitempty"`
    EmptyDir *EmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

type VolumeMount struct {
    Name string `json:"name"`
    MountPath string `json:"mountPath"`
    ReadOnly bool `json:"readOnly,omitempty"`
}
```

### 2. Controller Changes

Modify `createValidationPod()` to append user-defined volumes:

```go
// Start with default volumes
volumes := []corev1.Volume{
    {Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
    {Name: "jovyan-home", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
}

// Append user-defined volumes
for _, vol := range job.Spec.PodConfig.Volumes {
    volumes = append(volumes, convertToK8sVolume(vol))
}

pod.Spec.Volumes = volumes
```

### 3. RBAC Changes

Add PVC permissions to operator ServiceAccount (if not already present).

## Recommendations

### For release-4.18 Users

**Workaround**: Use environment variables and credentials instead of volumes where possible.

**Example**: Instead of mounting a ConfigMap, inject config as environment variables:
```yaml
spec:
  podConfig:
    envFrom:
      - configMapRef:
          name: notebook-config
```

### For Future Releases

Check `main`, `release-4.19`, or `release-4.20` branches to see if volume support has been added.

## Validation Cluster Test Results

✅ **Tekton Workspace Volumes**: PASSED  
- PVC created and mounted successfully
- Build artifacts persisted across TaskRuns

❌ **External Volume Mounting**: NOT SUPPORTED  
- Cannot test tier3 volume notebooks
- Feature not available in release-4.18

