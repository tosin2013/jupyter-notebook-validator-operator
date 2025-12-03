# File-Based Catalog (FBC) Multi-Version Upgrade Plan

**Date**: 2025-12-02  
**Current Status**: Successfully tested on OpenShift 4.18  
**Goal**: Upgrade to 4.19 and 4.20 with proper FBC catalog management

## Overview

This plan implements a File-Based Catalog (FBC) to manage multiple operator versions across OpenShift 4.18, 4.19, and 4.20.

**Reference**: [OLM File-Based Catalogs](https://olm.operatorframework.io/docs/reference/file-based-catalogs/)

## Current State

### Existing Branches
- ✅ `release-4.18` - Tested and validated with volume support
- ✅ `release-4.19` - Exists but not tested
- ✅ `release-4.20` - Exists but not tested
- ✅ `main` - Development branch

### Current Bundle Structure
```
bundle/
├── manifests/
│   ├── jupyter-notebook-validator-operator.clusterserviceversion.yaml
│   └── mlops.mlops.dev_notebookvalidationjobs.yaml
├── metadata/
│   └── annotations.yaml
└── tests/
    └── scorecard/
```

### Current Versions
- **Operator SDK**: v1.37.0
- **Bundle Version**: 1.0.0 (in bundle CSV)
- **Makefile VERSION**: 0.0.1 (needs update)
- **Container Images**:
  - `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes` (4.18)
  - Need to build for 4.19 and 4.20

## File-Based Catalog (FBC) Structure

### Why FBC?

1. **Declarative**: Define catalog as YAML files
2. **Version Control**: Track catalog changes in Git
3. **Multi-Version**: Support multiple OpenShift versions in one catalog
4. **Upgrade Paths**: Define explicit upgrade paths between versions
5. **No Index Images**: Simpler than traditional index-based catalogs

### Proposed FBC Directory Structure

```
catalog/
├── catalog.yaml              # Main FBC definition
├── Dockerfile                # Catalog container image
└── README.md                 # Catalog documentation
```

## Implementation Plan

### Phase 1: Create FBC Infrastructure (Week 1)

#### Step 1.1: Create FBC Directory and Files

```bash
# Create catalog directory
mkdir -p catalog

# Create catalog.yaml with all versions
cat > catalog/catalog.yaml <<'EOF'
---
schema: olm.package
name: jupyter-notebook-validator-operator
defaultChannel: stable
description: |
  Kubernetes operator for validating Jupyter notebooks in MLOps workflows.
  Supports OpenShift 4.18, 4.19, and 4.20.
---
schema: olm.channel
package: jupyter-notebook-validator-operator
name: stable
entries:
  - name: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
  - name: jupyter-notebook-validator-operator.v1.0.6-ocp4.19
    replaces: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
  - name: jupyter-notebook-validator-operator.v1.0.7-ocp4.20
    replaces: jupyter-notebook-validator-operator.v1.0.6-ocp4.19
---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.5-ocp4.18
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
  - type: olm.bundle.object
    value:
      data: <base64-encoded-csv>
---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.6-ocp4.19
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.6-ocp4.19
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.7-ocp4.20
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.7-ocp4.20
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
EOF
```

#### Step 1.2: Create Catalog Dockerfile

```bash
cat > catalog/Dockerfile <<'EOF'
FROM scratch
COPY catalog.yaml /configs/catalog.yaml
LABEL operators.operatorframework.io.index.configs.v1=/configs
EOF
```

#### Step 1.3: Add Makefile Targets for FBC

Add to `Makefile`:

```makefile
##@ File-Based Catalog (FBC)

.PHONY: catalog-init
catalog-init: opm ## Initialize FBC directory structure
	mkdir -p catalog
	$(OPM) alpha render-template basic catalog/catalog.yaml \
		--output yaml > catalog/catalog-generated.yaml

.PHONY: catalog-validate
catalog-validate: opm ## Validate FBC catalog
	$(OPM) validate catalog

.PHONY: catalog-build-fbc
catalog-build-fbc: ## Build FBC catalog image
	podman build -f catalog/Dockerfile -t $(CATALOG_IMG) catalog/

.PHONY: catalog-push-fbc
catalog-push-fbc: ## Push FBC catalog image
	podman push $(CATALOG_IMG)
```

### Phase 2: Build and Test 4.19 (Week 2)

#### Step 2.1: Checkout and Update release-4.19

```bash
# Checkout release-4.19 branch
git checkout release-4.19

# Merge volume support from release-4.18
git merge release-4.18

# Update version in Makefile
sed -i 's/VERSION ?= .*/VERSION ?= 1.0.6-ocp4.19/' Makefile

# Update bundle annotations
sed -i 's/v4.18/v4.19/g' bundle/metadata/annotations.yaml
sed -i 's/1.0.5/1.0.6/g' bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml
```

#### Step 2.2: Build and Push 4.19 Images

```bash
# Build operator image
export PATH=$PATH:/usr/local/go/bin
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19
podman push quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Generate bundle
make bundle IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Build and push bundle
make bundle-build BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19
podman push quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19
```

#### Step 2.3: Test on OpenShift 4.19 Cluster

```bash
# Deploy to 4.19 cluster
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Run tier5 volume tests
oc apply -f config/samples/tier5-ml-training-volumes.yaml

# Verify results
oc get notebookvalidationjob -A
```

### Phase 3: Build and Test 4.20 (Week 3)

#### Step 3.1: Checkout and Update release-4.20

```bash
# Checkout release-4.20 branch
git checkout release-4.20

# Merge from release-4.19
git merge release-4.19

# Update version
sed -i 's/VERSION ?= .*/VERSION ?= 1.0.7-ocp4.20/' Makefile
sed -i 's/v4.19/v4.20/g' bundle/metadata/annotations.yaml
sed -i 's/1.0.6/1.0.7/g' bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml
```

#### Step 3.2: Build and Push 4.20 Images

```bash
# Build operator image
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20
podman push quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20

# Generate bundle
make bundle IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20

# Build and push bundle
make bundle-build BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20
podman push quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20
```

#### Step 3.3: Test on OpenShift 4.20 Cluster

```bash
# Deploy to 4.20 cluster
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20

# Run tier5 volume tests
oc apply -f config/samples/tier5-ml-training-volumes.yaml

# Verify results
oc get notebookvalidationjob -A
```

### Phase 4: Build and Deploy FBC Catalog (Week 4)

#### Step 4.1: Build FBC Catalog Image

```bash
# Build catalog image
make catalog-build-fbc CATALOG_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest

# Push catalog image
make catalog-push-fbc CATALOG_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest
```

#### Step 4.2: Deploy FBC Catalog to Cluster

```bash
# Create CatalogSource
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: jupyter-notebook-validator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest
  displayName: Jupyter Notebook Validator Operator
  publisher: Community
  updateStrategy:
    registryPoll:
      interval: 10m
EOF

# Wait for catalog to be ready
oc wait --for=condition=Ready catalogsource/jupyter-notebook-validator-catalog \
  -n openshift-marketplace --timeout=5m

# Verify catalog
oc get catalogsource jupyter-notebook-validator-catalog -n openshift-marketplace
oc get packagemanifests | grep jupyter-notebook-validator
```

#### Step 4.3: Install Operator via OLM

```bash
# Create namespace
oc create namespace jupyter-notebook-validator-operator

# Create OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: jupyter-notebook-validator-operator-group
  namespace: jupyter-notebook-validator-operator
spec:
  targetNamespaces:
  - jupyter-notebook-validator-operator
EOF

# Create Subscription
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-notebook-validator-operator
spec:
  channel: stable
  name: jupyter-notebook-validator-operator
  source: jupyter-notebook-validator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF

# Watch installation
oc get csv -n jupyter-notebook-validator-operator -w
```

## Container Image Versions Reference

### OpenShift 4.18
- **Operator**: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes`
- **Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18`
- **Status**: ✅ Built and tested

### OpenShift 4.19
- **Operator**: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19`
- **Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19`
- **Status**: ⏳ To be built

### OpenShift 4.20
- **Operator**: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20`
- **Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20`
- **Status**: ⏳ To be built

### FBC Catalog
- **Catalog**: `quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest`
- **Status**: ⏳ To be built

## Upgrade Path

```
v1.0.5-ocp4.18 → v1.0.6-ocp4.19 → v1.0.7-ocp4.20
```

## Testing Checklist

### Per-Version Testing
- [ ] Operator deploys successfully
- [ ] CRD is created
- [ ] Webhooks are configured
- [ ] Basic notebook validation works
- [ ] Git credentials work
- [ ] Tekton builds work (if available)
- [ ] Volume mounting works (tier5 test)
- [ ] Model validation works (if models available)

### FBC Catalog Testing
- [ ] Catalog builds successfully
- [ ] Catalog deploys to cluster
- [ ] PackageManifest appears in marketplace
- [ ] Operator installs via OLM
- [ ] Operator upgrades between versions
- [ ] Rollback works if needed

## Next Steps

1. **Immediate**: Create FBC infrastructure in `catalog/` directory
2. **Week 1**: Build and test 4.19 on validation cluster
3. **Week 2**: Build and test 4.20 on validation cluster
4. **Week 3**: Build FBC catalog with all versions
5. **Week 4**: Test OLM installation and upgrades

## References

- [OLM File-Based Catalogs](https://olm.operatorframework.io/docs/reference/file-based-catalogs/)
- [OLM Upgrade Graph](https://olm.operatorframework.io/docs/concepts/olm-architecture/operator-catalog/creating-an-update-graph/)
- [Operator SDK Bundle](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/)
- [OpenShift Operator Lifecycle Manager](https://docs.openshift.com/container-platform/4.18/operators/understanding/olm/olm-understanding-olm.html)


