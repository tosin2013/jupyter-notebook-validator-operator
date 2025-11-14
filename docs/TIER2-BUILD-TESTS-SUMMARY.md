# Tier 2 E2E Build Tests - Summary

## Overview

This document summarizes the Tier 2 E2E testing work for S2I and Tekton build integration, including fixes applied and next steps.

## Test Results

### ✅ S2I Build Test: **PASSED**

**Test Configuration:**
- Repository: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git` (PRIVATE)
- Base Image: `image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1`
- Git Credentials: `git-credentials` secret (kubernetes.io/basic-auth format)
- Notebook: `notebooks/tier1-simple/01-hello-world.ipynb`

**What Worked:**
1. ✅ Git credentials setup with `scripts/setup-git-credentials.sh`
2. ✅ Private repository cloned successfully using `git-credentials` secret
3. ✅ S2I build completed with proper S2I-enabled base image
4. ✅ Dependencies installed from `requirements.txt`
5. ✅ Built image pushed to internal registry
6. ✅ Validation pod used built image (NO git-clone init container)
7. ✅ Notebook validation succeeded: `4/4 cells succeeded (100.0% success rate)`

**Key Learnings:**
- Must use S2I-enabled images (with `/usr/libexec/s2i/assemble` script)
- `quay.io/jupyter/minimal-notebook:latest` is NOT S2I-enabled
- OpenShift provides S2I-enabled Jupyter images in `redhat-ods-applications` namespace:
  - `s2i-minimal-notebook:2025.1`
  - `s2i-generic-data-science-notebook:2025.1`

### ⚠️ Tekton Build Test: **FAILED** → **FIXED**

**Initial Failure:**
```
Error: container has runAsNonRoot and image will run as root
(pod: "tier2-tekton-build-test-build-generate-dockerfile-pod")
```

**Root Cause:**
The `generate-dockerfile` inline task used `registry.access.redhat.com/ubi9/ubi-minimal:latest` which runs as root by default. This violated OpenShift's `restricted-v2` SCC which requires `runAsNonRoot=true`.

**Fix Applied:**
Added `SecurityContext` to the `generate-dockerfile` task step:

```go
SecurityContext: &corev1.SecurityContext{
    RunAsNonRoot: func() *bool { b := true; return &b }(),
    RunAsUser:    func() *int64 { uid := int64(65532); return &uid }(), // Standard non-root user
},
```

**Status:**
- ✅ Fix committed and pushed to GitHub
- ✅ GitHub Actions build completed successfully
- ✅ New image pushed to `quay.io/takinosh/jupyter-notebook-validator-operator:latest`
- ⏳ **Needs testing**: Operator needs to be redeployed with new image and Tekton test re-run

## Files Created/Modified

### New Scripts

1. **`scripts/tier2-build-tests.sh`**
   - Comprehensive Tier 2 E2E test script
   - Tests both S2I and Tekton build strategies
   - Includes prerequisite checks, colored output, detailed logging
   - Verifies build artifacts (checks for absence of git-clone init container)

2. **`scripts/setup-git-credentials.sh`**
   - Sets up Git credentials for both S2I and Tekton builds
   - Creates two secret formats:
     - `git-credentials`: For S2I builds (kubernetes.io/basic-auth)
     - `git-credentials-tekton`: For Tekton builds (basic-auth workspace)
   - Reads from environment variables (`GITHUB_USERNAME`, `GITHUB_TOKEN`)

3. **`scripts/setup-git-push.sh`**
   - Helper script for configuring Git authentication
   - Detects HTTPS vs SSH authentication
   - Provides guidance for setting up credentials
   - Can switch between HTTPS and SSH

4. **`scripts/README-TIER2-BUILD-TESTS.md`**
   - Comprehensive documentation for Tier 2 tests
   - Prerequisites, setup instructions, troubleshooting

### Modified Files

1. **`pkg/build/tekton_strategy.go`**
   - Added `SecurityContext` to `generate-dockerfile` task
   - Fixes `CreateContainerConfigError` in Tekton pipeline

2. **`Makefile`**
   - Added `git-push-rebuild` target for automated GitHub push
   - Added `git-status` target for quick status check

3. **`.gitignore`**
   - Added `.env` and `.env.*` to prevent secret leakage
   - Added `*.log`, `test-notebooks/`, `helm/` for test artifacts

4. **`config/manager/kustomization.yaml`**
   - Updated image reference (if modified during testing)

## Next Steps

### 1. Redeploy Operator with Fix

```bash
# Deploy operator with new image
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Or use kustomize directly
cd config/manager && kustomize edit set image controller=quay.io/takinosh/jupyter-notebook-validator-operator:latest
kustomize build config/default | oc apply -f -
```

### 2. Re-run Tekton Build Test

```bash
# Clean up previous test
oc delete notebookvalidationjob tier2-tekton-build-test -n e2e-tests
oc delete pipeline tier2-tekton-build-test-pipeline -n e2e-tests
oc delete pipelinerun tier2-tekton-build-test-build -n e2e-tests

# Re-run test
source .env && ./scripts/tier2-build-tests.sh
```

### 3. Verify Tekton Build Success

Expected outcomes:
- ✅ `generate-dockerfile` task pod should start successfully (no CreateContainerConfigError)
- ✅ Dockerfile should be generated from baseImage
- ✅ `buildah` task should build and push image
- ✅ Validation pod should use built image (NO git-clone init container)
- ✅ Notebook validation should succeed

### 4. Test Custom Dockerfile Support (ADR-031 Phase 2)

```bash
# Test with custom Dockerfile path
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_tekton_custom_dockerfile.yaml
```

### 5. Document Results

After successful Tekton test:
- Update this document with Tekton test results
- Create ADR update if needed
- Update E2E test documentation

## Git Workflow

### Pushing Changes to Trigger Rebuild

```bash
# Check status
make git-status

# Commit and push (triggers GitHub Actions build)
make git-push-rebuild MSG="your commit message"
```

### Monitoring Build

```bash
# Watch GitHub Actions build
gh run list --branch release-4.18 --limit 3
gh run watch <run-id>

# Or visit:
# https://github.com/tosin2013/jupyter-notebook-validator-operator/actions
```

## Configuration Samples

### S2I Build Sample

See: `config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml`

Key points:
- Use S2I-enabled base image from `redhat-ods-applications` namespace
- Specify `strategy: "s2i"`
- Use `git-credentials` secret for private repos

### Tekton Build Sample

See: `config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`

Key points:
- Use `strategy: "tekton"`
- Specify `baseImage` for auto-generated Dockerfile
- Use `git-credentials` secret for validation pod
- Tekton build automatically uses `git-credentials-tekton` secret

### Tekton with Custom Dockerfile

See: `config/samples/mlops_v1alpha1_notebookvalidationjob_tekton_custom_dockerfile.yaml`

Key points:
- Specify `dockerfile: "path/to/Dockerfile"` in buildConfig
- If Dockerfile not found, falls back to baseImage generation
- Supports both Dockerfile and Containerfile

## Troubleshooting

### S2I Build Fails with "No such file or directory: /usr/libexec/s2i/assemble"

**Solution**: Use S2I-enabled base image:
```yaml
baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
```

### Tekton Build Fails with "CreateContainerConfigError"

**Solution**: Ensure operator is using latest image with SecurityContext fix:
```bash
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest
```

### Git Clone Fails with "Permission denied"

**Solution**: Set up Git credentials:
```bash
export GITHUB_USERNAME=your_username
export GITHUB_TOKEN=ghp_your_token
./scripts/setup-git-credentials.sh e2e-tests
```

### Validation Pod Has git-clone Init Container

**Issue**: Build may have failed, causing fallback to init container approach.

**Solution**: Check build logs:
```bash
# For S2I
oc logs build/tier2-s2i-build-test-build-1 -n e2e-tests

# For Tekton
tkn pipelinerun logs tier2-tekton-build-test-build -n e2e-tests
```

## References

- **ADR-023**: S2I Build Integration on OpenShift
- **ADR-027**: S2I Build Strategy for Git Integration
- **ADR-028**: Tekton Task Strategy
- **ADR-031**: Tekton Build Strategy (Dockerfile vs baseImage)
- **ADR-033**: E2E Testing on Live OpenShift Cluster

