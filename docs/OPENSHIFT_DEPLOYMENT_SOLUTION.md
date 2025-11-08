# OpenShift Deployment Solution

## Problem Summary

The Jupyter Notebook Validator Operator was experiencing permission issues when deploying to OpenShift due to Security Context Constraints (SCC) and the way Jupyter notebook images are designed.

### Root Cause

1. **Jupyter Images Expect UID 1000**: Standard Jupyter images (like `quay.io/jupyter/scipy-notebook:latest`) are designed to run as UID 1000 with `/home/jovyan` as the home directory.

2. **OpenShift Assigns Random UIDs**: OpenShift's `restricted-v2` SCC assigns random UIDs from the namespace's allocated range (e.g., 1000920000-1000919999) for security.

3. **Permission Mismatch**: When the container runs as UID 1000920000, it cannot write to `/home/jovyan/.local` (owned by UID 1000), causing pip installations to fail.

4. **False Positive Success**: The original script didn't properly detect pip failures because pip can exit with code 0 even when installation fails with permission errors.

## Solution: Use Custom Container Images

The **recommended solution** is to create a custom container image with Papermill pre-installed. This eliminates the need for runtime pip installations and avoids permission issues entirely.

### Step 1: Create a Custom Dockerfile

Create a file named `Dockerfile.jupyter-papermill`:

```dockerfile
FROM quay.io/jupyter/scipy-notebook:latest

# Switch to root to install system packages if needed
USER root

# Install Papermill and common dependencies as root
# This ensures they're installed in system site-packages, not user site-packages
RUN pip install --no-cache-dir \
    papermill==2.6.0 \
    nbformat==5.10.4 \
    nbconvert==7.16.6 \
    scikit-learn \
    pandas \
    numpy \
    matplotlib \
    seaborn

# Switch back to the notebook user
USER ${NB_UID}

# Verify installation
RUN python -c "import papermill; print(f'Papermill {papermill.__version__} installed successfully')"
```

### Step 2: Build and Push the Image

```bash
# Build the image
podman build -f Dockerfile.jupyter-papermill -t quay.io/your-org/jupyter-papermill:latest .

# Login to Quay.io
podman login quay.io

# Push the image
podman push quay.io/your-org/jupyter-papermill:latest
```

### Step 3: Update Your NotebookValidationJob

Update your NotebookValidationJob to use the custom image:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: notebookvalidationjob-sample
  namespace: jupyter-notebook-validator-operator
spec:
  notebook:
    gitRepo:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      branch: "main"
    path: "notebooks/01.01-Help-And-Documentation.ipynb"
  
  podConfig:
    # Use your custom image with Papermill pre-installed
    containerImage: "quay.io/your-org/jupyter-papermill:latest"
    
    serviceAccountName: "notebook-validator-jupyter-notebook-validator-runner"
    
    resources:
      limits:
        memory: "2Gi"
        cpu: "1"
      requests:
        memory: "1Gi"
        cpu: "500m"
```

### Step 4: Deploy and Test

```bash
# Apply the updated job
oc apply -f your-notebookvalidationjob.yaml

# Watch the pod
oc get pods -n jupyter-notebook-validator-operator -w

# Check logs
oc logs <validation-pod-name> -n jupyter-notebook-validator-operator
```

You should now see:
```
[2025-11-08 10:00:00] Checking Papermill installation...
[2025-11-08 10:00:00] ✓ Papermill already installed
[2025-11-08 10:00:00] Python version: Python 3.13.9
[2025-11-08 10:00:00] Papermill version: 2.6.0
```

## Alternative Approaches (Not Recommended)

### Approach 1: Use anyuid SCC (Security Risk)

**NOT RECOMMENDED** - This bypasses OpenShift's security model:

```bash
oc adm policy add-scc-to-user anyuid -z notebook-validator-jupyter-notebook-validator-runner
```

**Why not recommended**: 
- Violates security best practices
- May not be allowed in production environments
- Defeats the purpose of OpenShift's security model

### Approach 2: Runtime Environment Variable Overrides (Doesn't Work)

We attempted to set `HOME=/workspace` and `PYTHONUSERBASE=/workspace/.local` to redirect pip installations, but this approach has limitations:

- Environment variables may not be properly propagated
- Jupyter images have hardcoded paths in various places
- Still requires writable directories that may not exist
- Adds complexity and potential failure points

## Error Detection Improvements

We've enhanced the operator to properly detect and report pip installation failures:

### Before (False Positive)
```
[2025-11-08 09:21:46] Installing Papermill...
ERROR: Could not install packages due to an OSError: [Errno 13] Permission denied: '/home/jovyan/.local'
[2025-11-08 09:21:46] ✓ Papermill installed
Status: succeeded
```

### After (Proper Error Detection)
```
[2025-11-08 10:00:00] Installing Papermill...
ERROR: Could not install packages due to an OSError: [Errno 13] Permission denied: '/home/jovyan/.local'
[2025-11-08 10:00:00] ERROR: Pip installation failed. Log contents:
...
[2025-11-08 10:00:00] ERROR: Failed to install Papermill due to permission errors.
SOLUTION: Use a custom container image with Papermill pre-installed.
See docs/ERROR_HANDLING_GUIDE.md for instructions.
Status: failed
Error Category: dependency_install_failed
```

## Best Practices

### 1. Always Use Custom Images for Production

Pre-install all dependencies in your custom image:

```dockerfile
FROM quay.io/jupyter/scipy-notebook:latest

USER root

# Install all required packages
RUN pip install --no-cache-dir \
    papermill \
    nbformat \
    nbconvert \
    # Add your notebook's dependencies here
    tensorflow \
    torch \
    transformers \
    scikit-learn

USER ${NB_UID}
```

### 2. Version Pin Your Dependencies

```dockerfile
RUN pip install --no-cache-dir \
    papermill==2.6.0 \
    nbformat==5.10.4 \
    nbconvert==7.16.6
```

### 3. Test Your Custom Image Locally

```bash
# Run your custom image
podman run -it --rm \
  -v $(pwd)/notebooks:/workspace/notebooks:Z \
  quay.io/your-org/jupyter-papermill:latest \
  bash

# Test papermill
papermill /workspace/notebooks/test.ipynb /workspace/output.ipynb
```

### 4. Use Image Digests for Reproducibility

```yaml
spec:
  podConfig:
    containerImage: "quay.io/your-org/jupyter-papermill@sha256:abc123..."
```

### 5. Maintain Separate Images for Different Use Cases

```
quay.io/your-org/jupyter-ml:latest          # ML/AI notebooks
quay.io/your-org/jupyter-data-science:latest # Data science notebooks
quay.io/your-org/jupyter-basic:latest        # Basic Python notebooks
```

## Troubleshooting

### Issue: "Papermill already installed" but execution fails

**Cause**: The image has papermill but it's in a location not in PATH.

**Solution**: Verify papermill is in system site-packages:
```dockerfile
RUN pip install --no-cache-dir papermill && \
    which papermill && \
    papermill --version
```

### Issue: Custom image works locally but fails on OpenShift

**Cause**: Image may have hardcoded UIDs or paths.

**Solution**: Ensure your Dockerfile doesn't hardcode UIDs:
```dockerfile
# Good - uses the image's default user
USER ${NB_UID}

# Bad - hardcodes UID
USER 1000
```

### Issue: Image is too large

**Cause**: Installing too many dependencies.

**Solution**: Use multi-stage builds or create specialized images:
```dockerfile
FROM quay.io/jupyter/scipy-notebook:latest as base

# Only install what you need
RUN pip install --no-cache-dir papermill nbformat nbconvert

# Don't install unnecessary packages
```

## Summary

✅ **DO**: Create custom images with Papermill pre-installed
✅ **DO**: Version pin your dependencies
✅ **DO**: Test images locally before deploying
✅ **DO**: Use OpenShift's standard `restricted-v2` SCC

❌ **DON'T**: Rely on runtime pip installations
❌ **DON'T**: Use `anyuid` SCC in production
❌ **DON'T**: Hardcode UIDs in Dockerfiles
❌ **DON'T**: Assume environment variable overrides will work

## References

- [ERROR_HANDLING_GUIDE.md](ERROR_HANDLING_GUIDE.md) - Comprehensive error handling documentation
- [ADR-005: OpenShift Compatibility](adrs/005-openshift-compatibility.md) - Architectural decision record
- [Jupyter Docker Stacks](https://jupyter-docker-stacks.readthedocs.io/) - Official Jupyter images
- [OpenShift Security Context Constraints](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)

