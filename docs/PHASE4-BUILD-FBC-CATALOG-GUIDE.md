# Phase 4: Build and Deploy FBC Catalog

**Status**: Ready to Execute (After Phase 3)  
**Prerequisites**: All three versions (4.18, 4.19, 4.20) built and tested  
**Duration**: ~30 minutes

## Overview

This phase builds the File-Based Catalog (FBC) that contains all three operator versions and deploys it to OpenShift for OLM-based installation.

## Prerequisites Verification

Before proceeding, verify all bundle images exist:

```bash
# Check 4.18 bundle
podman pull quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18

# Check 4.19 bundle
podman pull quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19

# Check 4.20 bundle
podman pull quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20
```

All three should pull successfully. If any fail, go back and complete that phase.

## Step-by-Step Instructions

### Step 1: Checkout release-4.18 Branch

```bash
cd /home/lab-user/jupyter-notebook-validator-operator

# Checkout release-4.18 (where FBC was created)
git checkout release-4.18

# Verify catalog directory exists
ls -la catalog/
```

### Step 2: Validate FBC Catalog

```bash
# Install opm if not already installed
make opm

# Validate catalog structure
./bin/opm validate catalog

# Expected output: "Validation successful"
```

### Step 3: Build FBC Catalog Image

```bash
# Build catalog image
make catalog-build-fbc CATALOG_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest

# Verify image was built
podman images | grep catalog
```

### Step 4: Push FBC Catalog Image

```bash
# Login to quay.io (if not already logged in)
podman login quay.io

# Push catalog image
make catalog-push-fbc CATALOG_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest

# Verify push
echo "Catalog pushed: quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest"
```

### Step 5: Deploy FBC Catalog to OpenShift

```bash
# Login to your OpenShift cluster
oc login <cluster-url>

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

# Verify catalog pod is running
oc get pods -n openshift-marketplace | grep jupyter-notebook-validator
```

### Step 6: Verify Catalog in OperatorHub

```bash
# Check if PackageManifest appears
oc get packagemanifests -n openshift-marketplace | grep jupyter-notebook-validator

# Get detailed package info
oc get packagemanifest jupyter-notebook-validator-operator -n openshift-marketplace -o yaml

# Check available channels
oc get packagemanifest jupyter-notebook-validator-operator -n openshift-marketplace \
  -o jsonpath='{.status.channels[*].name}'

# Check available versions
oc get packagemanifest jupyter-notebook-validator-operator -n openshift-marketplace \
  -o jsonpath='{.status.channels[*].currentCSVDesc.version}'
```

### Step 7: Install Operator via OLM (Optional Test)

```bash
# Create test namespace
oc create namespace jupyter-validator-olm-test

# Create OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: jupyter-notebook-validator-og
  namespace: jupyter-validator-olm-test
spec:
  targetNamespaces:
  - jupyter-validator-olm-test
EOF

# Create Subscription
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-validator-olm-test
spec:
  channel: stable
  name: jupyter-notebook-validator-operator
  source: jupyter-notebook-validator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF

# Watch installation
oc get csv -n jupyter-validator-olm-test -w

# Verify operator is running
oc get pods -n jupyter-validator-olm-test
```

### Step 8: Test Operator Installation

```bash
# Create a test NotebookValidationJob
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: olm-test
  namespace: jupyter-validator-olm-test
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
  timeout: "5m"
EOF

# Monitor test
oc get notebookvalidationjob olm-test -n jupyter-validator-olm-test -w

# Check results
oc get notebookvalidationjob olm-test -n jupyter-validator-olm-test -o yaml | grep -A 10 "status:"
```

## Verification Checklist

- [ ] All three bundle images exist in quay.io
- [ ] FBC catalog validated with opm
- [ ] Catalog image built and pushed
- [ ] CatalogSource deployed to cluster
- [ ] Catalog pod running in openshift-marketplace
- [ ] PackageManifest appears in marketplace
- [ ] Operator installs via OLM
- [ ] Test NotebookValidationJob succeeds

## Container Images

- ✅ **4.18 Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18`
- ✅ **4.19 Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19`
- ✅ **4.20 Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20`
- ✅ **FBC Catalog**: `quay.io/takinosh/jupyter-notebook-validator-operator-catalog:latest`

## Upgrade Path Verification

The FBC catalog defines this upgrade path:

```
v1.0.5-ocp4.18 → v1.0.6-ocp4.19 → v1.0.7-ocp4.20
```

To test upgrades:
1. Install v1.0.5-ocp4.18
2. Update subscription to allow upgrades
3. Verify automatic upgrade to v1.0.6-ocp4.19
4. Verify automatic upgrade to v1.0.7-ocp4.20

## Troubleshooting

### Issue: Catalog Pod CrashLoopBackOff
```bash
# Check catalog pod logs
oc logs -n openshift-marketplace <catalog-pod-name>

# Common causes:
# - Invalid catalog.yaml syntax
# - Missing bundle images
# - Network issues pulling images
```

### Issue: PackageManifest Not Appearing
```bash
# Wait a few minutes for catalog to sync
sleep 60

# Check catalog source status
oc get catalogsource jupyter-notebook-validator-catalog -n openshift-marketplace -o yaml

# Check marketplace operator logs
oc logs -n openshift-marketplace deployment/marketplace-operator
```

### Issue: Operator Installation Fails
```bash
# Check InstallPlan
oc get installplan -n jupyter-validator-olm-test

# Check CSV status
oc get csv -n jupyter-validator-olm-test -o yaml

# Check operator pod logs
oc logs -n jupyter-validator-olm-test deployment/<operator-deployment>
```

## Next Steps

After successful FBC deployment:
1. Document the catalog in README.md
2. Create user guide for OLM installation
3. Consider submitting to OperatorHub.io
4. Set up CI/CD for automated catalog updates

## Success Criteria

✅ All phases complete when:
- All three operator versions (4.18, 4.19, 4.20) are built and tested
- FBC catalog is deployed and accessible in OperatorHub
- Operator installs successfully via OLM
- Test workloads run successfully
- Upgrade path works correctly

