# Jupyter Notebook Validator Operator Catalog

File-Based Catalog (FBC) for the Jupyter Notebook Validator Operator.

## Versions

All versions use semantic version **1.0.7** with OpenShift-specific image tags:

- **v1.0.7-ocp4.18**: OpenShift 4.18 (with volume support)
- **v1.0.7-ocp4.19**: OpenShift 4.19
- **v1.0.7-ocp4.20**: OpenShift 4.20

**Note**: The operator version (1.0.7) is consistent across all platforms. OpenShift version appears only in container image tags.

## Build Catalog

```bash
podman build -f catalog/Dockerfile -t quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest catalog/
podman push quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest
```

## Deploy Catalog

```bash
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: jupyter-notebook-validator-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest
  displayName: Jupyter Notebook Validator Operator
  publisher: Community
  updateStrategy:
    registryPoll:
      interval: 10m
