# Release Notes: v1.0.0-ocp4.19

## Overview

Initial release of the Jupyter Notebook Validator Operator for OpenShift 4.19. This release brings full compatibility with OpenShift 4.19 (Kubernetes 1.32) while maintaining all existing features from the 4.18 release.

## Supported Platforms

- **OpenShift**: 4.19.x (tested on 4.19.17)
- **Kubernetes**: 1.32.x
- **Architectures**: amd64, arm64

## What's New

### OpenShift 4.19 Compatibility
- Full support for OpenShift 4.19 with Kubernetes 1.32
- Validated with cert-manager on 4.19
- All E2E test tiers pass (Tier 1-4)

### Inherited Features from 4.18
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
kubectl apply -k https://github.com/tosin2013/jupyter-notebook-validator-operator/config/default?ref=v1.0.0-ocp4.19
```

### Using Container Image
```bash
docker pull quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.19
```

## Container Images

- `quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.19`

## Documentation

- [Installation Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.19/README.md)
- [Architecture Decision Records](https://github.com/tosin2013/jupyter-notebook-validator-operator/tree/release-4.19/docs/adrs)
- [API Reference](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.19/docs/api-reference.md)
- [Credentials Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.19/docs/NOTEBOOK_CREDENTIALS_GUIDE.md)
- [Model Discovery Guide](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/release-4.19/docs/MODEL_DISCOVERY_GUIDE.md)

## Breaking Changes

None. This is a compatible release from v1.0.x-ocp4.18.

## Known Issues

None at this time.

## Upgrade Path

If upgrading from v1.0.x-ocp4.18:
1. Ensure your cluster is running OpenShift 4.19
2. Update the operator image to v1.0.0-ocp4.19
3. No CRD changes required

## Contributors

Thanks to all contributors who helped validate and test this release on OpenShift 4.19.
