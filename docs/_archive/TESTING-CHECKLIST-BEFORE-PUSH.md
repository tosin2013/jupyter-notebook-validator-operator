# Testing Checklist Before Git Push

**Date Created:** 2025-11-11  
**Status:** ⚠️ CRITICAL - DO NOT PUSH TO GIT UNTIL ALL TESTS PASS

## Overview

This document provides a step-by-step checklist for testing the Jupyter Notebook Validator Operator locally before pushing code to the repository. Both Kind (local Kubernetes) and OpenShift tests must pass.

## Prerequisites

### 1. Disk Space ⚠️ REQUIRED
- **Current:** 5.9GB available in /var/tmp
- **Required:** ~10GB free space
- **Action:** Add more disk space before proceeding

### 2. Environment Verification
```bash
# Check disk space
df -h /var/tmp

# Verify OpenShift cluster access
oc cluster-info
oc whoami

# Verify Kind and Podman
kind version
podman --version
```

## Test 1: Kind Local Testing

### Location
```bash
cd /home/lab-user/jupyter-notebook-validator-operator
```

### Test Command
```bash
./scripts/test-local-kind.sh --podman-rootful --skip-cleanup
```

### Expected Results

#### ✅ Cluster Creation (Already Verified)
- [ ] Kind cluster creates successfully (~17 seconds)
- [ ] Kubernetes v1.31.12 running
- [ ] All core components healthy (coredns, etcd, kube-apiserver, etc.)

#### ✅ cert-manager Installation (Already Verified)
- [ ] cert-manager namespace created
- [ ] All cert-manager deployments ready
- [ ] Webhooks configured

#### ⏳ Operator Deployment (Needs Testing)
- [ ] Operator image builds successfully with `sudo podman build`
- [ ] Image loads into Kind cluster
- [ ] Operator deployment created
- [ ] Operator pod reaches Running state
- [ ] Operator pod passes readiness checks
- [ ] No errors in operator logs

**Verification Commands:**
```bash
# Check operator pod
sudo kubectl get pods -n jupyter-notebook-validator-operator-system

# Check operator logs
sudo kubectl logs -n jupyter-notebook-validator-operator-system \
  -l control-plane=controller-manager --tail=50

# Verify CRD
sudo kubectl get crd notebookvalidationjobs.mlops.mlops.dev
```

#### ⏳ Test Environment Setup (Needs Testing)
- [ ] Test namespace created
- [ ] Git credentials secret created (if GIT_USERNAME/GIT_TOKEN set)
- [ ] Service account created

#### ⏳ Tier 1 Tests (Needs Testing)
- [ ] Test 1: 01-hello-world.ipynb passes
- [ ] Test 2: 02-basic-math.ipynb passes
- [ ] Test 3: 03-data-validation.ipynb passes
- [ ] All tests complete in < 2 minutes total
- [ ] No errors in test execution

**Verification Commands:**
```bash
# Check test jobs
sudo kubectl get notebookvalidationjobs -n test-notebooks

# Check test status
sudo kubectl get notebookvalidationjobs -n test-notebooks -o wide

# View test details
sudo kubectl describe notebookvalidationjob <job-name> -n test-notebooks
```

#### ⏳ Cleanup (Needs Testing)
- [ ] Cluster deletes successfully
- [ ] No orphaned resources

**Cleanup Command:**
```bash
sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind delete cluster --name jupyter-validator-test
```

### Success Criteria
- ✅ All 3 Tier 1 tests pass (3/3)
- ✅ Operator logs show no errors
- ✅ Cluster cleanup successful
- ✅ Total execution time < 5 minutes

---

## Test 2: OpenShift Testing

### Location
```bash
cd /home/lab-user/jupyter-notebook-validator-operator
```

### Step 1: Verify Cluster Access
```bash
# Login to OpenShift (if needed)
oc login https://api.cluster-c4r4z.c4r4z.sandbox5156.opentlc.com:6443

# Verify access
oc cluster-info
oc whoami
oc get nodes
```

**Checklist:**
- [ ] Successfully logged in to OpenShift
- [ ] Cluster is accessible
- [ ] User has appropriate permissions

### Step 2: Build and Push Operator Image
```bash
# Set your image registry (replace with your registry)
export IMG=quay.io/<your-username>/jupyter-notebook-validator-operator:test

# Build image
make docker-build IMG=$IMG

# Push to registry
make docker-push IMG=$IMG
```

**Checklist:**
- [ ] Image builds successfully
- [ ] Image pushes to registry
- [ ] Image is accessible from OpenShift

### Step 3: Deploy Operator
```bash
# Deploy operator
make deploy IMG=$IMG

# Verify deployment
oc get pods -n jupyter-notebook-validator-operator-system
oc logs -n jupyter-notebook-validator-operator-system \
  -l control-plane=controller-manager --tail=50
```

**Checklist:**
- [ ] Operator namespace created
- [ ] Operator pod is Running
- [ ] Operator pod passes readiness checks
- [ ] No errors in operator logs
- [ ] CRD is installed

### Step 4: Create Test Namespace
```bash
# Create test namespace
oc create namespace test-notebooks

# Create git credentials (if testing private repos)
oc create secret generic git-https-credentials \
  --from-literal=username="$GIT_USERNAME" \
  --from-literal=password="$GIT_TOKEN" \
  -n test-notebooks
```

**Checklist:**
- [ ] Test namespace created
- [ ] Credentials secret created (if needed)

### Step 5: Run Tier 1 Tests
Test repository: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks

Create NotebookValidationJob CRs for each test:

**Test 1: Hello World**
```bash
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-hello-world
  namespace: test-notebooks
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
  timeout: "5m"
EOF
```

**Test 2: Basic Math**
```bash
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-basic-math
  namespace: test-notebooks
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/02-basic-math.ipynb"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
  timeout: "5m"
EOF
```

**Test 3: Data Validation**
```bash
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-data-validation
  namespace: test-notebooks
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/03-data-validation.ipynb"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
  timeout: "5m"
EOF
```

**Monitor Tests:**
```bash
# Watch test status
watch oc get notebookvalidationjobs -n test-notebooks

# Check specific test
oc get notebookvalidationjob tier1-hello-world -n test-notebooks -o yaml

# View logs
oc logs -n test-notebooks -l job-name=tier1-hello-world
```

**Checklist:**
- [ ] tier1-hello-world: Status = Succeeded
- [ ] tier1-basic-math: Status = Succeeded
- [ ] tier1-data-validation: Status = Succeeded
- [ ] All tests complete in < 2 minutes
- [ ] No errors in operator logs

### Step 6: Run Tier 2 Test (Build Integration)
```bash
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier2-train-sentiment
  namespace: test-notebooks
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier2-intermediate/01-train-sentiment-model.ipynb"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
  buildConfig:
    enabled: true
    strategy: "tekton"
  timeout: "10m"
EOF
```

**Monitor:**
```bash
# Watch test
watch oc get notebookvalidationjobs -n test-notebooks

# Check Tekton pipeline
oc get pipelineruns -n test-notebooks

# View logs
oc logs -n test-notebooks -l job-name=tier2-train-sentiment
```

**Checklist:**
- [ ] Tekton pipeline created
- [ ] Build completes successfully
- [ ] Notebook executes with built image
- [ ] Test status = Succeeded
- [ ] Test completes in < 5 minutes

### Step 7: Cleanup
```bash
# Delete test resources
oc delete notebookvalidationjobs --all -n test-notebooks
oc delete namespace test-notebooks

# Undeploy operator
make undeploy

# Verify cleanup
oc get all -n jupyter-notebook-validator-operator-system
```

**Checklist:**
- [ ] All test jobs deleted
- [ ] Test namespace deleted
- [ ] Operator undeployed
- [ ] No orphaned resources

### Success Criteria
- ✅ All 3 Tier 1 tests pass (3/3)
- ✅ Tier 2 build test passes (1/1)
- ✅ No errors in operator logs
- ✅ Resources cleanup successfully
- ✅ Total execution time < 10 minutes

---

## Final Checklist Before Git Push

- [ ] ✅ Disk space added (~10GB free)
- [ ] ✅ Kind test passed completely
- [ ] ✅ OpenShift Tier 1 tests passed (3/3)
- [ ] ✅ OpenShift Tier 2 test passed (1/1)
- [ ] ✅ All operator logs reviewed (no errors)
- [ ] ✅ Test results documented
- [ ] ✅ Cleanup verified on both platforms

## Git Push Commands (After All Tests Pass)

```bash
cd /home/lab-user/jupyter-notebook-validator-operator

# Review commits
git log --oneline -5

# Push to remote
git push origin release-4.18
```

## Current Commits Ready to Push

1. **3fedc83** - Update Kind testing to use Kubernetes v1.31.12 and add Podman rootful mode support
2. **0a465c3** - Add Podman support to Kind testing script with proper container runtime detection
3. **f0d8ee6** - Update IMPLEMENTATION-PLAN.md with local testing requirements before git push

## Notes

- Test repository: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks
- OpenShift cluster: https://api.cluster-c4r4z.c4r4z.sandbox5156.opentlc.com:6443
- Kind script: `./scripts/test-local-kind.sh --podman-rootful`
- All tests must pass before pushing to prevent breaking the CI/CD pipeline

## Troubleshooting

### Kind Test Fails
- Check disk space: `df -h /var/tmp`
- Check Podman images: `sudo podman images`
- Check operator logs: `sudo kubectl logs -n jupyter-notebook-validator-operator-system -l control-plane=controller-manager`

### OpenShift Test Fails
- Check operator logs: `oc logs -n jupyter-notebook-validator-operator-system -l control-plane=controller-manager`
- Check test job status: `oc describe notebookvalidationjob <name> -n test-notebooks`
- Check Tekton pipeline: `oc get pipelineruns -n test-notebooks`

---

**Last Updated:** 2025-11-11  
**Status:** Ready for testing after disk space is added

