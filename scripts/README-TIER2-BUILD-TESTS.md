# Tier 2 E2E Tests - Build Integration Testing

## Overview

Tier 2 tests validate the **complete build workflow** for S2I and Tekton build strategies as defined in the ADRs:

- **ADR-009**: Secret Management and Git Credentials
- **ADR-016**: External Secrets Operator Integration  
- **ADR-023**: S2I Build Integration (OpenShift)
- **ADR-027**: S2I Build Strategy for Git Integration
- **ADR-028**: Tekton Task Strategy (Custom vs Cluster Tasks)
- **ADR-031**: Tekton Build (Dockerfile vs Base Image)
- **ADR-033**: E2E Testing on Live OpenShift Cluster

## What These Tests Validate

### S2I Build Strategy
1. ✅ BuildConfig creation with Git source
2. ✅ Git credentials handling for private repositories
3. ✅ S2I build execution and monitoring
4. ✅ ImageStream creation and tagging
5. ✅ Built image usage in validation pod (no git-clone init container)

### Tekton Build Strategy
1. ✅ Pipeline creation with git-clone and buildah tasks
2. ✅ Task copying from openshift-pipelines namespace to user namespace
3. ✅ Git credentials handling (basic-auth workspace format)
4. ✅ PipelineRun execution and monitoring
5. ✅ Built image usage in validation pod (no git-clone init container)

## Prerequisites

### 1. OpenShift Cluster Access
```bash
oc cluster-info
# Should show your OpenShift cluster API server
```

### 2. Operator Deployed
```bash
oc get deployment notebook-validator-controller-manager -n jupyter-notebook-validator-operator
# Should show the operator deployment
```

### 3. Git Credentials (REQUIRED)

The test repository **https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git** is **PRIVATE** and requires authentication.

#### Create GitHub Personal Access Token

1. Go to: https://github.com/settings/tokens
2. Click **"Generate new token (classic)"**
3. Give it a name: `jupyter-notebook-validator-tests`
4. Select scopes:
   - ✅ **repo** (Full control of private repositories)
5. Click **"Generate token"**
6. **Copy the token** (starts with `ghp_`) - you won't see it again!

#### Set Up Credentials

```bash
# Export your GitHub credentials
export GITHUB_USERNAME=your-github-username
export GITHUB_TOKEN=ghp_your_personal_access_token

# Run the setup script
./scripts/setup-git-credentials.sh e2e-tests
```

This creates two secrets:
- **git-credentials**: For S2I builds (kubernetes.io/basic-auth format)
- **git-credentials-tekton**: For Tekton builds (basic-auth workspace format)

### 4. S2I BuildConfig API (OpenShift)
```bash
oc api-resources | grep buildconfigs
# Should show: buildconfigs, bc
```

### 5. Tekton Pipeline API (OpenShift Pipelines)
```bash
oc api-resources | grep tekton.dev
# Should show: pipelines, pipelineruns, tasks, taskruns, etc.
```

If Tekton is not installed:
```bash
# Install OpenShift Pipelines Operator from OperatorHub
# Or via CLI:
oc apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

## Running the Tests

### Quick Start

```bash
# 1. Set up Git credentials (one-time setup)
export GITHUB_USERNAME=your-github-username
export GITHUB_TOKEN=ghp_your_token
./scripts/setup-git-credentials.sh e2e-tests

# 2. Run Tier 2 build tests
./scripts/tier2-build-tests.sh
```

### What Happens During Tests

#### Test 1: S2I Build Strategy

1. Creates NotebookValidationJob with `buildConfig.strategy: s2i`
2. Operator creates BuildConfig with Git source
3. BuildConfig automatically triggers a Build
4. Build clones private repository using git-credentials secret
5. S2I builds image from base image + notebooks
6. Built image is pushed to internal registry
7. Operator creates validation pod using built image
8. **Verification**: Pod should NOT have git-clone init container

#### Test 2: Tekton Build Strategy

1. Creates NotebookValidationJob with `buildConfig.strategy: tekton`
2. Operator copies git-clone and buildah Tasks to e2e-tests namespace
3. Operator creates Pipeline with those Tasks
4. Operator creates PipelineRun to execute Pipeline
5. git-clone Task clones private repository using git-credentials-tekton secret
6. buildah Task builds image from base image + notebooks
7. Built image is pushed to internal registry
8. Operator creates validation pod using built image
9. **Verification**: Pod should NOT have git-clone init container

### Expected Output

```
========================================
Tier 2 E2E Tests - Build Integration
========================================
ℹ️  Testing S2I and Tekton build workflows per ADR-023, ADR-027, ADR-028, ADR-031, ADR-033

========================================
Checking Prerequisites
========================================
✅ oc CLI found
✅ Connected to OpenShift cluster
✅ Operator deployed in namespace: jupyter-notebook-validator-operator
✅ Test namespace exists: e2e-tests
✅ Git credentials secret exists: git-credentials
✅ S2I BuildConfig API available
✅ Tekton Pipeline API available
✅ Required Tekton tasks (git-clone, buildah) found in openshift-pipelines namespace

========================================
Test 1: S2I Build Strategy (ADR-023, ADR-027)
========================================
ℹ️  Creating NotebookValidationJob with S2I build...
✅ S2I build job created
✅ BuildConfig created: tier2-s2i-build-test-build
ℹ️  Build started: tier2-s2i-build-test-build-1
ℹ️  Monitoring build progress...
✅ Job succeeded: tier2-s2i-build-test

========================================
Verifying Build Artifacts for tier2-s2i-build-test
========================================
✅ Validation pod found: tier2-s2i-build-test-validation
✅ Pod does NOT have git-clone init container - using built image ✓
✅ Built image: image-registry.openshift-image-registry.svc:5000/e2e-tests/tier2-s2i-build-test-build@sha256:...

========================================
Test 2: Tekton Build Strategy (ADR-028, ADR-031)
========================================
ℹ️  Creating NotebookValidationJob with Tekton build...
✅ Tekton build job created
✅ Pipeline created: tier2-tekton-build-test-pipeline
ℹ️  PipelineRun started: tier2-tekton-build-test-build
✅ Job succeeded: tier2-tekton-build-test

========================================
Verifying Build Artifacts for tier2-tekton-build-test
========================================
✅ Validation pod found: tier2-tekton-build-test-validation
✅ Pod does NOT have git-clone init container - using built image ✓
✅ Built image: image-registry.openshift-image-registry.svc:5000/e2e-tests/tier2-tekton-build-test-build@sha256:...

========================================
Test Summary
========================================
✅ S2I Build Test: PASSED
✅ Tekton Build Test: PASSED
✅ All Tier 2 build tests passed!
```

## Troubleshooting

### Error: Git credentials secret not found

```
❌ Git credentials secret not found: git-credentials
```

**Solution**: Run the setup script:
```bash
export GITHUB_USERNAME=your-username
export GITHUB_TOKEN=ghp_your_token
./scripts/setup-git-credentials.sh e2e-tests
```

### Error: failed to fetch requested repository

```
error: failed to fetch requested repository "https://github.com/..." with provided credentials
```

**Possible causes**:
1. **Invalid token**: Token may be expired or revoked
2. **Insufficient permissions**: Token needs `repo` scope
3. **Wrong username**: Verify GITHUB_USERNAME matches your GitHub account

**Solution**: Generate a new token and re-run setup:
```bash
# Generate new token at: https://github.com/settings/tokens
export GITHUB_USERNAME=your-username
export GITHUB_TOKEN=ghp_new_token
./scripts/setup-git-credentials.sh e2e-tests
```

### Error: Tekton Pipeline API not available

```
⚠️  Tekton Pipeline API not available (tests will be skipped)
```

**Solution**: Install OpenShift Pipelines Operator:
1. Open OpenShift Console
2. Go to **Operators** → **OperatorHub**
3. Search for **"Red Hat OpenShift Pipelines"**
4. Click **Install**
5. Wait for installation to complete
6. Re-run tests

### Error: Build timeout

```
❌ Timeout waiting for job: tier2-s2i-build-test
```

**Possible causes**:
1. **Slow network**: Image pulls taking too long
2. **Resource constraints**: Cluster under heavy load
3. **Build stuck**: Build pod not starting

**Solution**: Check build logs:
```bash
# For S2I builds
oc logs build/tier2-s2i-build-test-build-1 -n e2e-tests

# For Tekton builds
oc logs -n e2e-tests -l tekton.dev/pipelineRun=tier2-tekton-build-test-build --all-containers
```

## Advanced Usage

### Test with Custom Repository

```bash
export TEST_REPO=https://github.com/your-org/your-private-repo.git
export TEST_REPO_REF=main
export NOTEBOOK_PATH=notebooks/your-notebook.ipynb
./scripts/tier2-build-tests.sh
```

### Test Only S2I or Tekton

Edit `scripts/tier2-build-tests.sh` and comment out the test you don't want to run.

### Keep Resources for Debugging

By default, the script does NOT clean up resources. To manually clean up:

```bash
oc delete notebookvalidationjobs -n e2e-tests -l tier=2
oc delete buildconfigs -n e2e-tests -l tier=2
oc delete pipelines -n e2e-tests -l tier=2
oc delete pipelineruns -n e2e-tests -l tier=2
```

## Related Documentation

- [ADR-009: Secret Management and Git Credentials](../docs/adrs/009-secret-management-and-git-credentials.md)
- [ADR-016: External Secrets Operator Integration](../docs/adrs/016-external-secrets-operator-integration.md)
- [ADR-023: S2I Build Integration](../docs/adrs/023-s2i-build-integration-openshift.md)
- [ADR-028: Tekton Task Strategy](../docs/adrs/028-tekton-task-strategy-custom-vs-cluster-tasks.md)
- [ADR-031: Tekton Build Strategy](../docs/adrs/031-tekton-build-dockerfile-vs-base-image.md)
- [Git Credentials Setup Guide](../config/samples/git-credentials-example.yaml)
- [ESO Git Credentials for Tekton](../config/samples/eso-git-credentials-tekton.yaml)

