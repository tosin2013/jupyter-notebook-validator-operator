# External Volume Support Implementation

## Summary

External volume support has been successfully implemented in the `release-4.18` branch. This feature allows users to mount PVCs, ConfigMaps, Secrets, and EmptyDirs in validation pods.

## Changes Made

### 1. CRD Updates (`api/v1alpha1/notebookvalidationjob_types.go`)

Added new fields to `PodConfigSpec`:
- `Volumes []PodVolume` - List of volumes to mount
- `VolumeMounts []VolumeMount` - List of volume mount points

Added new types:
- `PodVolume` - Represents a volume (PVC, ConfigMap, Secret, EmptyDir)
- `PersistentVolumeClaimVolumeSource`
- `ConfigMapVolumeSource`
- `SecretVolumeSource`
- `EmptyDirVolumeSource`
- `VolumeMount` - Describes where to mount volumes

### 2. Controller Updates

**`internal/controller/notebookvalidationjob_controller.go`:**
- Added `buildPodVolumes()` function to build volume list
- Converts user-defined volumes to Kubernetes volume specs
- Appends to default volumes (workspace, jovyan-home)

**`internal/controller/papermill_helper.go`:**
- Updated `buildPapermillValidationContainer()` to append user-defined volume mounts
- Maintains default mounts (workspace, jovyan-home) plus user mounts

### 3. Generated Artifacts

- CRD manifests regenerated with `make manifests generate`
- New volume fields added to OpenAPI schema

## Usage Example

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: test-external-volumes
spec:
  notebook:
    git:
      url: "https://github.com/org/repo.git"
      ref: "main"
    path: "notebooks/test.ipynb"
  
  podConfig:
    containerImage: "quay.io/jupyter/scipy-notebook:latest"
    
    # Define volumes
    volumes:
      - name: model-output
        persistentVolumeClaim:
          claimName: model-output-pvc
          readOnly: false
      
      - name: config-data
        configMap:
          name: notebook-config
          optional: true
      
      - name: temp-data
        emptyDir:
          sizeLimit: "1Gi"
    
    # Define mount points
    volumeMounts:
      - name: model-output
        mountPath: /models
      
      - name: config-data
        mountPath: /config
        readOnly: true
      
      - name: temp-data
        mountPath: /tmp/data
```

## Deployment Steps

### 1. Build and Push Image

```bash
# Build
podman build -t quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes .

# Push
podman push quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes
```

### 2. Update Deployment

```bash
# Undeploy current version
make undeploy

# Install CRDs
make install

# Deploy new version
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes
```

### 3. Create Test Resources

```bash
# Create PVC for testing
oc apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: model-output-pvc
  namespace: jupyter-validator-volume-test
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

# Create ConfigMap for testing
oc create configmap notebook-config \
  --from-literal=model_name=test-model \
  --from-literal=threshold=0.95 \
  -n jupyter-validator-volume-test

# Apply test CR
oc apply -f config/samples/test-external-volumes.yaml
```

### 4. Verify

```bash
# Check job status
oc get notebookvalidationjob test-external-volumes -n jupyter-validator-volume-test

# Check pod
oc get pods -n jupyter-validator-volume-test -l job-name=test-external-volumes

# Verify volumes are mounted
oc describe pod <pod-name> -n jupyter-validator-volume-test | grep -A 20 "Volumes:"
```

## Testing Notebooks

Now you can test the tier3 notebooks that require external volumes:
- `tier3-complex/06-pvc-model-output-test.ipynb` - Requires PVC at `/models`
- `tier3-complex/07-shared-dataset-volumes-test.ipynb` - Requires PVC, ConfigMap, EmptyDir

## Next Steps

1. Push the image to quay.io
2. Deploy to validation cluster
3. Create test PVC and ConfigMap
4. Run test CR with external volumes
5. Verify volumes are mounted correctly
6. Test tier3 notebooks

## Documentation Updates Needed

- Update user guide with volume examples
- Add volume support to ADR documentation
- Update Helm chart with volume examples

