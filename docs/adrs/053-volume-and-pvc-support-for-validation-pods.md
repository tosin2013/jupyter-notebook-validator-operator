# ADR-053: Volume and PVC Support for Validation Pods

**Date**: 2025-11-29
**Status**: Implemented
**Updated**: 2026-01-24
**Deciders**: Development Team, User Community
**Technical Story**: Enable notebooks to persist trained models and access shared datasets via PVC volumes
**Supersedes**: Previously numbered as ADR-045 (renumbered to resolve duplicate)

## Context and Problem Statement

Users have requested the ability to mount Persistent Volume Claims (PVCs) to NotebookValidationJob pods. The primary use case is **MLOps training-to-serving workflows** where:

1. A notebook trains a machine learning model
2. The trained model is saved to a PVC
3. A model serving platform (KServe, OpenShift AI, etc.) loads the model from that PVC

Currently, the `PodConfigSpec` in our API supports:
- Container image and resources
- Environment variables and credentials
- Build configuration

**Missing**: Volume mounts for persistent storage access.

### User Story

> "I need volumes for the NotebookValidationJob pods so if a notebook is training a model, it can save the output to a PVC. Then a KServe model can pull the model from the PVC instead of S3."

This is a common pattern in:
- Air-gapped environments without cloud storage access
- On-premises deployments using NFS/Ceph storage
- Multi-stage ML pipelines where training and serving share storage
- Cost-conscious deployments avoiding cloud storage egress fees

## Decision Drivers

- **MLOps Workflow Integration**: Enable end-to-end training-to-serving pipelines
- **Platform Compatibility**: Support KServe `pvc://` storageUri, OpenShift AI, and other platforms
- **Flexibility**: Support various volume types (PVC, ConfigMap, Secret, EmptyDir)
- **Kubernetes Native**: Use standard Kubernetes volume/volumeMount patterns
- **Air-Gap Support**: Enable deployments without cloud storage dependencies
- **Backward Compatibility**: Existing NotebookValidationJobs must continue to work

## Considered Options

### Option 1: Simple PVC-Only Support

Add a `persistentVolumeClaims` field that only supports PVCs:

```yaml
spec:
  podConfig:
    persistentVolumeClaims:
      - name: model-output
        claimName: trained-models-pvc
        mountPath: /models
      - name: training-data
        claimName: datasets-pvc
        mountPath: /data
        readOnly: true
```

**Pros**:
- Simple API surface
- Covers 90% of use cases
- Easy to implement

**Cons**:
- Limited to PVCs only
- Cannot mount ConfigMaps or Secrets as files
- Non-standard pattern (not Kubernetes native)

### Option 2: Full Kubernetes Volume Support ✅ **Selected**

Add `volumes` and `volumeMounts` fields that mirror Kubernetes Pod spec:

```yaml
spec:
  podConfig:
    volumes:
      - name: model-output
        persistentVolumeClaim:
          claimName: trained-models-pvc
      - name: training-data
        persistentVolumeClaim:
          claimName: datasets-pvc
      - name: config
        configMap:
          name: notebook-config
      - name: scratch
        emptyDir:
          sizeLimit: 10Gi
    volumeMounts:
      - name: model-output
        mountPath: /models
      - name: training-data
        mountPath: /data
        readOnly: true
      - name: config
        mountPath: /config
      - name: scratch
        mountPath: /tmp/scratch
```

**Pros**:
- Full Kubernetes compatibility
- Supports all volume types (PVC, ConfigMap, Secret, EmptyDir, HostPath, NFS, etc.)
- Familiar pattern for Kubernetes users
- Future-proof for new volume types

**Cons**:
- More complex API
- More validation required
- Larger implementation effort

### Option 3: Hybrid Approach

Add simplified `storage` field with common patterns plus escape hatch for advanced use:

```yaml
spec:
  podConfig:
    storage:
      outputs:
        pvcName: trained-models-pvc
        mountPath: /models
      inputs:
        pvcName: datasets-pvc
        mountPath: /data
        readOnly: true
    # Escape hatch for advanced use cases
    additionalVolumes: [...]
    additionalVolumeMounts: [...]
```

**Pros**:
- Simple for common cases
- Still flexible for advanced use

**Cons**:
- Non-standard API
- Confusing having two ways to do the same thing
- More documentation burden

## Decision Outcome

**Chosen option**: **Option 2 - Full Kubernetes Volume Support**

This provides the best balance of:
- **Flexibility**: Supports all Kubernetes volume types
- **Familiarity**: Uses standard Kubernetes patterns
- **Future-Proof**: No API changes needed for new volume types
- **Ecosystem Compatibility**: Works with any storage solution

## API Design

### New Types in `api/v1alpha1/notebookvalidationjob_types.go`

```go
// PodConfigSpec defines the pod configuration for validation execution
type PodConfigSpec struct {
    // ... existing fields ...

    // Volumes defines volumes to mount in the validation pod
    // Supports PersistentVolumeClaim, ConfigMap, Secret, EmptyDir, and other volume types
    // +optional
    Volumes []Volume `json:"volumes,omitempty"`

    // VolumeMounts defines where to mount volumes in the validation container
    // +optional
    VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}

// Volume represents a named volume in a pod
// Simplified version of corev1.Volume with common volume sources
type Volume struct {
    // Name is the volume name, must match a VolumeMount name
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
    Name string `json:"name"`

    // PersistentVolumeClaim represents a reference to a PVC in the same namespace
    // +optional
    PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`

    // ConfigMap represents a ConfigMap to mount as a volume
    // +optional
    ConfigMap *ConfigMapVolumeSource `json:"configMap,omitempty"`

    // Secret represents a Secret to mount as a volume
    // +optional
    Secret *SecretVolumeSource `json:"secret,omitempty"`

    // EmptyDir represents a temporary directory that shares a pod's lifetime
    // +optional
    EmptyDir *EmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

// PersistentVolumeClaimVolumeSource references a PVC in the same namespace
type PersistentVolumeClaimVolumeSource struct {
    // ClaimName is the name of a PersistentVolumeClaim in the same namespace
    // +kubebuilder:validation:Required
    ClaimName string `json:"claimName"`

    // ReadOnly will force the volume to be mounted read-only
    // +optional
    ReadOnly bool `json:"readOnly,omitempty"`
}

// ConfigMapVolumeSource adapts a ConfigMap into a volume
type ConfigMapVolumeSource struct {
    // Name is the name of the ConfigMap
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Items if unspecified, each key-value pair becomes a file
    // +optional
    Items []KeyToPath `json:"items,omitempty"`

    // DefaultMode is the mode bits for created files (default 0644)
    // +optional
    DefaultMode *int32 `json:"defaultMode,omitempty"`

    // Optional specifies whether the ConfigMap must exist
    // +optional
    Optional *bool `json:"optional,omitempty"`
}

// SecretVolumeSource adapts a Secret into a volume
type SecretVolumeSource struct {
    // SecretName is the name of the Secret
    // +kubebuilder:validation:Required
    SecretName string `json:"secretName"`

    // Items if unspecified, each key-value pair becomes a file
    // +optional
    Items []KeyToPath `json:"items,omitempty"`

    // DefaultMode is the mode bits for created files (default 0644)
    // +optional
    DefaultMode *int32 `json:"defaultMode,omitempty"`

    // Optional specifies whether the Secret must exist
    // +optional
    Optional *bool `json:"optional,omitempty"`
}

// EmptyDirVolumeSource is a temporary directory that shares a pod's lifetime
type EmptyDirVolumeSource struct {
    // Medium is the storage medium type (default "" uses node's default, "Memory" uses tmpfs)
    // +kubebuilder:validation:Enum="";Memory
    // +optional
    Medium string `json:"medium,omitempty"`

    // SizeLimit is the maximum size of the emptyDir volume
    // +optional
    SizeLimit string `json:"sizeLimit,omitempty"`
}

// KeyToPath maps a key to a file path
type KeyToPath struct {
    // Key is the key to project
    // +kubebuilder:validation:Required
    Key string `json:"key"`

    // Path is the relative path of the file to map the key to
    // +kubebuilder:validation:Required
    Path string `json:"path"`

    // Mode is the file mode bits
    // +optional
    Mode *int32 `json:"mode,omitempty"`
}

// VolumeMount describes a mount point for a Volume
type VolumeMount struct {
    // Name must match the Name of a Volume
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // MountPath is the path within the container to mount the volume
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^/.*$`
    MountPath string `json:"mountPath"`

    // SubPath is a sub-path inside the volume to mount (optional)
    // +optional
    SubPath string `json:"subPath,omitempty"`

    // ReadOnly mounts the volume as read-only when true
    // +optional
    ReadOnly bool `json:"readOnly,omitempty"`
}
```

### Example Usage

#### Use Case 1: Training Model Output to PVC for KServe

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: train-sentiment-model
spec:
  notebook:
    git:
      url: "https://github.com/myorg/ml-notebooks.git"
      ref: "main"
    path: "training/sentiment-model.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/pytorch-notebook:latest"
    resources:
      requests:
        memory: "8Gi"
        cpu: "4"
        nvidia.com/gpu: "1"
    volumes:
      - name: model-output
        persistentVolumeClaim:
          claimName: trained-models-pvc
    volumeMounts:
      - name: model-output
        mountPath: /models
  timeout: "2h"
---
# KServe InferenceService using the trained model
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: sentiment-model
spec:
  predictor:
    pytorch:
      storageUri: "pvc://trained-models-pvc/sentiment/v1"
```

#### Use Case 2: Shared Dataset Access (ReadWriteMany)

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: analyze-dataset
spec:
  notebook:
    git:
      url: "https://github.com/myorg/analysis-notebooks.git"
      ref: "main"
    path: "analysis/explore-dataset.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/datascience-notebook:latest"
    volumes:
      - name: shared-data
        persistentVolumeClaim:
          claimName: ml-datasets-pvc
          readOnly: true
      - name: scratch
        emptyDir:
          sizeLimit: 50Gi
    volumeMounts:
      - name: shared-data
        mountPath: /data
        readOnly: true
      - name: scratch
        mountPath: /tmp/working
  timeout: "30m"
```

#### Use Case 3: Configuration Files from ConfigMap

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: configurable-validation
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "validation/parameterized.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    volumes:
      - name: config
        configMap:
          name: validation-config
      - name: certs
        secret:
          secretName: model-endpoint-certs
    volumeMounts:
      - name: config
        mountPath: /config
      - name: certs
        mountPath: /certs
        readOnly: true
  timeout: "10m"
```

## Implementation Plan

### Phase 1: API Types (Release 4.20)
1. Add `Volume`, `VolumeMount`, and supporting types to `api/v1alpha1/`
2. Update CRD generation with `make manifests generate`
3. Add webhook validation for volume/mount consistency
4. Update documentation and samples

### Phase 2: Controller Implementation (Release 4.20)
1. Update `papermill_helper.go` to add volumes/mounts to validation pod spec
2. Convert custom types to `corev1.Volume` and `corev1.VolumeMount`
3. Add RBAC permissions if needed (PVC list/get may be required)
4. Add unit tests for volume conversion

### Phase 3: Integration Testing (Release 4.20)
1. Add E2E test for PVC volume mount
2. Test with KServe integration (save model, serve from PVC)
3. Test with shared datasets (ReadWriteMany)
4. Test ConfigMap/Secret volume mounts

### Phase 4: Backport (Releases 4.18, 4.19)
1. Cherry-pick API changes
2. Cherry-pick controller changes
3. Run E2E tests on each release branch
4. Update release notes

## Consequences

### Positive

1. **Enables MLOps Workflows**: Complete training-to-serving pipelines
2. **KServe Integration**: Direct compatibility with `pvc://` storageUri
3. **Air-Gap Support**: Works without cloud storage
4. **Flexible Storage**: Supports PVC, ConfigMap, Secret, EmptyDir
5. **Standard Patterns**: Familiar Kubernetes API design
6. **Backward Compatible**: Existing jobs continue to work

### Negative

1. **API Complexity**: More fields to understand and validate
2. **Storage Dependencies**: Users must provision PVCs separately
3. **Access Mode Considerations**: ReadWriteOnce vs ReadWriteMany confusion
4. **Security Surface**: Mounting arbitrary PVCs/Secrets requires RBAC consideration

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Volume name collision with operator-created volumes | Pod fails to start | Validate volume names don't conflict with `git-clone`, `notebook-data` |
| PVC doesn't exist | Pod stuck in Pending | Add webhook validation to check PVC exists |
| Wrong access mode (RWO vs RWX) | Concurrent jobs fail | Document access mode requirements clearly |
| Large PVC slows pod startup | Timeout failures | Document storage class performance considerations |

## Security Considerations

1. **PVC Access**: Users can only mount PVCs they have access to (namespace-scoped)
2. **Secret Volumes**: Same security model as existing `env.valueFrom.secretKeyRef`
3. **HostPath**: Not supported in this ADR to prevent security issues
4. **RBAC**: Operator may need `get`/`list` on PVCs for validation (optional)

## Documentation Updates

1. **API Reference**: Document new `volumes` and `volumeMounts` fields
2. **User Guide**: "Mounting Persistent Storage" section
3. **Integration Guide**: "Training Models for KServe" tutorial
4. **Samples**: Add examples in `config/samples/`

## References

- [KServe PVC Storage](https://kserve.github.io/website/0.8/modelserving/storage/pvc/pvc/)
- [Kubeflow Notebook Volumes](https://thenewstack.io/tutorial-configure-storage-volumes-for-kubeflow-notebook-servers/)
- [OpenShift AI Model Serving](https://docs.redhat.com/en/documentation/red_hat_openshift_ai_self-managed/2.16/html-single/serving_models/index)
- [Kubernetes Volumes](https://kubernetes.io/docs/concepts/storage/volumes/)
- **Related ADRs**:
  - ADR-040: Unique Build PVCs for Concurrent Tekton Builds
  - ADR-031: Tekton Build Dockerfile vs Base Image
