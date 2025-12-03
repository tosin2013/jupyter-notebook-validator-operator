# Phase 2: Build and Test OpenShift 4.19 Support

**Status**: Ready to Execute  
**Prerequisites**: Phase 1 Complete (FBC infrastructure created)  
**Duration**: ~1-2 hours

## Overview

This phase builds the operator and bundle for OpenShift 4.19, incorporating all the volume support features from 4.18.

## Step-by-Step Instructions

### Step 1: Checkout release-4.19 Branch

```bash
cd /home/lab-user/jupyter-notebook-validator-operator

# Checkout release-4.19
git checkout release-4.19

# Check current state
git log --oneline -3
git status
```

### Step 2: Merge Volume Support from release-4.18

```bash
# Merge all volume support features from 4.18
git merge release-4.18

# Resolve any conflicts if they occur
# (Likely none since 4.19 should be behind 4.18)

# Verify merge
git log --oneline -5
```

### Step 3: Update Version Numbers

```bash
# Update Makefile VERSION
sed -i 's/VERSION ?= .*/VERSION ?= 1.0.6-ocp4.19/' Makefile

# Verify change
grep "VERSION ?=" Makefile
```

### Step 4: Update Bundle Metadata

```bash
# Update OpenShift version in bundle annotations
sed -i 's/v4.18/v4.19/g' bundle/metadata/annotations.yaml

# Update version in CSV
sed -i 's/1.0.5/1.0.6/g' bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml
sed -i 's/v1.0.5/v1.0.6/g' bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml

# Verify changes
grep "com.redhat.openshift.versions" bundle/metadata/annotations.yaml
grep "version:" bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml | head -5
```

### Step 5: Build Operator Image

```bash
# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin

# Build operator image
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Verify image was built
podman images | grep jupyter-notebook-validator-operator
```

### Step 6: Push Operator Image

```bash
# Login to quay.io (if not already logged in)
podman login quay.io

# Push operator image
podman push quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Verify push
echo "Image pushed: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19"
```

### Step 7: Generate Bundle Manifests

```bash
# Generate bundle with new image
make bundle IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Verify bundle was generated
ls -la bundle/manifests/
cat bundle/metadata/annotations.yaml | grep "com.redhat.openshift.versions"
```

### Step 8: Build Bundle Image

```bash
# Build bundle image
make bundle-build BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19

# Verify bundle image
podman images | grep bundle
```

### Step 9: Push Bundle Image

```bash
# Push bundle image
podman push quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19

# Verify push
echo "Bundle pushed: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19"
```

### Step 10: Commit Changes

```bash
# Add all changes
git add -A

# Commit
git commit -m "feat: Add OpenShift 4.19 support with volume features

- Merge volume support from release-4.18
- Update version to 1.0.6-ocp4.19
- Update bundle metadata for OpenShift 4.19
- Build and push operator image: 1.0.6-ocp4.19
- Build and push bundle image: 1.0.6-ocp4.19

Features included:
- External volume support (PVC, ConfigMap, Secret, EmptyDir)
- Tier5 ML training pipeline validation
- All 4.18 volume enhancements

Container Images:
- Operator: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19
- Bundle: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19"

# Push to GitHub
git push origin release-4.19
```

### Step 11: Test on OpenShift 4.19 Cluster (When Available)

**⚠️ WAIT FOR CLUSTER UPGRADE TO 4.19 BEFORE PROCEEDING**

Once you have access to an OpenShift 4.19 cluster:

```bash
# Login to 4.19 cluster
oc login <4.19-cluster-url>

# Deploy operator
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19

# Wait for operator to be ready
oc wait --for=condition=Ready pod -l control-plane=controller-manager \
  -n jupyter-notebook-validator-operator-system --timeout=120s

# Run tier5 volume test
oc apply -f config/samples/tier5-ml-training-volumes.yaml

# Monitor test
oc get notebookvalidationjob tier5-ml-training-volumes -w

# Verify results
oc get notebookvalidationjob tier5-ml-training-volumes -o yaml | grep -A 20 "status:"
```

## Verification Checklist

- [ ] release-4.19 branch checked out
- [ ] Volume support merged from release-4.18
- [ ] Version updated to 1.0.6-ocp4.19
- [ ] Bundle metadata updated for 4.19
- [ ] Operator image built and pushed
- [ ] Bundle image built and pushed
- [ ] Changes committed and pushed to GitHub
- [ ] **WAIT**: Cluster upgraded to 4.19
- [ ] Operator deployed to 4.19 cluster
- [ ] Tier5 volume test passed on 4.19

## Container Images Built

- ✅ **Operator**: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19`
- ✅ **Bundle**: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19`

## Next Steps

After successful testing on 4.19:
1. Proceed to Phase 3 (Build 4.20)
2. See `docs/PHASE3-BUILD-4.20-GUIDE.md`

## Troubleshooting

### Issue: Merge Conflicts
```bash
# If merge conflicts occur
git status
# Manually resolve conflicts in affected files
git add <resolved-files>
git commit
```

### Issue: Image Build Fails
```bash
# Check Go version
go version  # Should be 1.22+

# Check disk space
df -h

# Clean up old images
podman system prune -a
```

### Issue: Push Fails (Unauthorized)
```bash
# Re-login to quay.io
podman login quay.io
# Enter credentials and retry push
```

