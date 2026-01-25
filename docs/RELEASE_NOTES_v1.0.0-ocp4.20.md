# Release Notes: v1.0.0-ocp4.20

## Overview

Initial release of the Jupyter Notebook Validator Operator for OpenShift 4.20. This release brings full compatibility with OpenShift 4.20 (Kubernetes 1.33) and includes upgraded Tekton Pipeline support for OpenShift Pipelines 1.20.

## Supported Platforms

- **OpenShift**: 4.20.x (tested on 4.20.3)
- **Kubernetes**: 1.33.x
- **OpenShift Pipelines**: 1.20.x (Tekton Pipelines v0.68.0)
- **Architectures**: amd64, arm64

## What's New

### OpenShift 4.20 Compatibility
- Full support for OpenShift 4.20 with Kubernetes 1.33
- Validated with cert-manager on 4.20
- Upgraded Tekton Pipeline support to v0.68.0 for OpenShift Pipelines 1.20

### Tekton Pipeline v0.68.0 Upgrade
- Updated Tekton Pipeline integration for compatibility with OpenShift Pipelines 1.20
- Improved build strategy performance and reliability
- Support for latest Tekton Pipeline API features

### Volume and PVC Support (ADR-045) - NEW
This release introduces full Kubernetes volume support for validation pods, enabling MLOps training-to-serving workflows:

- **PersistentVolumeClaim (PVC) Support**: Mount PVCs to save trained models for model serving platforms
- **ConfigMap Volumes**: Mount configuration files as volumes
- **Secret Volumes**: Mount certificates and sensitive files securely
- **EmptyDir Volumes**: Temporary scratch space for intermediate files

**Use Cases**:
- Train models in notebooks and save to PVC for KServe (`pvc://` storageUri)
- Share datasets across multiple validation jobs using ReadWriteMany PVCs
- Mount configuration files for parameterized notebook execution
- Air-gapped deployments without cloud storage dependencies

**Example**:
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
          readOnly: true
    volumeMounts:
      - name: model-output
        mountPath: /models
      - name: training-data
        mountPath: /data
        readOnly: true
```

See [ADR-045](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.20/docs/adrs/045-volume-and-pvc-support-for-validation-pods.md) for complete documentation.

### Inherited Features from 4.19
- Jupyter notebook validation with Papermill
- Git repository integration (HTTPS and SSH)
- Configurable pod resources and environment variables
- Golden notebook comparison with tolerance settings
- Tekton build integration for dependency management
- S2I (Source-to-Image) build support
- KServe model inference validation
- Multi-platform model serving support (KServe, OpenShift AI, vLLM, TorchServe, TensorFlow Serving, Triton, Ray Serve, Seldon, BentoML)
- Credential injection with user-friendly syntax
- External Secrets Operator (ESO) integration
- Comprehensive error handling and status tracking
- Optional admission webhooks (can be disabled via ENABLE_WEBHOOKS=false)

## Installation

### Using Helm (Recommended)
```bash
helm repo add jupyter-notebook-validator https://tosin2013.github.io/jupyter-notebook-validator-operator
helm install jupyter-notebook-validator jupyter-notebook-validator/jupyter-notebook-validator-operator \
  --namespace jupyter-notebook-validator-system \
  --create-namespace
```

### Using Kustomize
```bash
kubectl apply -k https://github.com/tosin2013/jupyter-notebook-validator-operator/config/default?ref=v1.0.0-ocp4.20
```

### Using Container Image
```bash
docker pull quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.20
```

## Container Images

- `quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.20`

## Documentation

- [Installation Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.20/README.md)
- [Architecture Decision Records](https://github.com/tosin2013/jupyter-notebook-validator-operator/tree/release-4.20/docs/adrs)
- [API Reference](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.20/docs/api-reference.md)
- [Credentials Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.20/docs/NOTEBOOK_CREDENTIALS_GUIDE.md)
- [Model Discovery Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.20/docs/MODEL_DISCOVERY_GUIDE.md)

## Breaking Changes

None. This release adds new optional fields but is fully backward compatible with v1.0.x-ocp4.19.

## New API Fields

The following new optional fields were added to `PodConfigSpec` (ADR-045):
- `volumes`: Define volumes to mount in the validation pod
- `volumeMounts`: Define where to mount volumes in the container

These fields are optional and existing NotebookValidationJob resources will continue to work without modification.

## Known Issues

None at this time.

## Upgrade Path

If upgrading from v1.0.x-ocp4.19:
1. Ensure your cluster is running OpenShift 4.20
2. Update the operator image to v1.0.0-ocp4.20
3. Apply updated CRDs: `kubectl apply -f config/crd/bases/` (new volume fields added)
4. Existing NotebookValidationJobs will continue to work

If upgrading from v1.0.x-ocp4.18:
1. Ensure your cluster is running OpenShift 4.20
2. Update the operator image to v1.0.0-ocp4.20
3. Apply updated CRDs: `kubectl apply -f config/crd/bases/` (new volume fields added)
4. Existing NotebookValidationJobs will continue to work

## Platform Requirements

| Component | Minimum Version | Tested Version |
|-----------|-----------------|----------------|
| OpenShift | 4.20.0 | 4.20.3 |
| Kubernetes | 1.33.0 | 1.33.5 |
| OpenShift Pipelines | 1.20.0 | 1.20.1 |
| Tekton Pipelines | v0.68.0 | v0.68.0 |

## Contributors

Thanks to all contributors who helped validate and test this release on OpenShift 4.20.
