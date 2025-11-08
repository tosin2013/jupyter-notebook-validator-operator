# Error Handling and Categorization Guide

## Overview

The Jupyter Notebook Validator Operator implements comprehensive error categorization to help users quickly identify and resolve issues. This guide explains the error categories, how to interpret them, and recommended solutions.

## Error Categories

### 1. `configuration_error`

**Description**: Issues with the NotebookValidationJob configuration or notebook file location.

**Common Causes**:
- Notebook file not found at specified path
- Invalid Git repository URL
- Missing required configuration fields
- Invalid service account name

**Example Error**:
```
Notebook not found at path: /workspace/repo/notebooks/missing.ipynb
```

**Resolution**:
- Verify the notebook path in your NotebookValidationJob spec
- Check that the Git repository URL is correct
- Ensure the notebook exists in the specified branch
- Review the sample configurations in `config/samples/`

### 2. `dependency_install_failed`

**Description**: Papermill or other Python dependencies failed to install.

**Common Causes**:
- Permission issues in the container (most common on OpenShift)
- Missing Python development tools (pip, setuptools)
- Network connectivity issues
- Incompatible Python version

**Example Error**:
```
Failed to install Papermill. This may be due to: 
1) Insufficient permissions in the container
2) Missing Python development tools
3) Network connectivity issues
Check that the base image supports user-level pip installs or consider using a custom image with Papermill pre-installed.
```

**Resolution**:

#### Option A: Use a Custom Image (Recommended)
Create a custom Jupyter image with Papermill pre-installed:

```dockerfile
FROM quay.io/jupyter/scipy-notebook:latest

USER root
RUN pip install --no-cache-dir papermill nbformat nbconvert
USER ${NB_UID}
```

Build and push:
```bash
podman build -t quay.io/your-org/jupyter-papermill:latest .
podman push quay.io/your-org/jupyter-papermill:latest
```

Update your NotebookValidationJob:
```yaml
spec:
  podConfig:
    containerImage: quay.io/your-org/jupyter-papermill:latest
```

#### Option B: Verify Environment Variables
The operator sets these environment variables for OpenShift compatibility:
- `HOME=/workspace`
- `PYTHONUSERBASE=/workspace/.local`
- `PIP_USER=1`
- `PIP_NO_CACHE_DIR=1`

If you're still seeing permission errors, check the pod logs:
```bash
oc logs <validation-pod-name> -n <namespace>
```

Look for the environment diagnostics:
```
Environment: HOME=/workspace, PYTHONUSERBASE=/workspace/.local
User: 1000920000:1000920000
Writable check: YES
```

### 3. `environment_setup_failed`

**Description**: The validation environment could not be properly configured.

**Common Causes**:
- Permission errors writing to directories
- Security Context Constraints (SCC) violations on OpenShift
- Volume mount issues
- Insufficient disk space

**Example Error**:
```
Notebook execution failed due to permission errors. 
Check that the container has write access to required directories.
```

**Resolution**:

#### For OpenShift Users:
1. **Verify SCC Compliance**: The operator is designed to work with OpenShift's `restricted-v2` SCC. Do NOT use `anyuid` SCC.

2. **Check Service Account**: Ensure you're using the correct service account:
```yaml
spec:
  podConfig:
    serviceAccountName: notebook-validator-jupyter-notebook-validator-runner
```

3. **Verify Volume Mounts**: Check that the workspace volume is writable:
```bash
oc describe pod <validation-pod-name> -n <namespace>
```

#### For Kubernetes Users:
1. **Check Security Context**: Verify the pod's security context allows writing to `/workspace`

2. **Verify PVC**: If using persistent volumes, ensure they're writable

### 4. `notebook_execution_failed`

**Description**: The notebook code itself failed during execution.

**Common Causes**:
- Python code errors (NameError, TypeError, AttributeError)
- Missing Python modules not caught by dependency check
- Logic errors in notebook cells
- Resource constraints (OOM, timeout)

**Example Error**:
```
Notebook execution failed due to code errors. 
Review the notebook code for issues.
```

**Resolution**:

1. **Review Execution Logs**: Check the detailed execution log:
```bash
oc logs <validation-pod-name> -n <namespace>
```

2. **Check Cell Results**: The results JSON includes per-cell execution status:
```json
{
  "status": "failed",
  "error_category": "notebook_execution_failed",
  "cells": [
    {
      "cell_index": 5,
      "cell_type": "code",
      "status": "failed",
      "error": "NameError: name 'undefined_variable' is not defined"
    }
  ]
}
```

3. **Test Locally**: Run the notebook locally with Papermill:
```bash
papermill input.ipynb output.ipynb --log-output
```

4. **Check Resource Limits**: Increase memory/CPU if needed:
```yaml
spec:
  podConfig:
    resources:
      limits:
        memory: "4Gi"
        cpu: "2"
      requests:
        memory: "2Gi"
        cpu: "1"
```

## Status Conditions

The operator sets Kubernetes conditions to track validation progress:

### Condition Types

1. **`Ready`**: Overall readiness of the validation job
2. **`GitCloned`**: Git repository clone status
3. **`ValidationStarted`**: Validation execution started
4. **`ValidationComplete`**: Validation finished (success or failure)
5. **`EnvironmentReady`**: Environment setup status

### Condition Reasons

- `Initializing`: Job is being initialized
- `GitCloneInProgress`: Cloning Git repository
- `GitCloneSucceeded`: Git clone completed successfully
- `GitCloneFailed`: Git clone failed
- `PodCreated`: Validation pod created
- `PodRunning`: Validation pod is running
- `PodSucceeded`: Validation completed successfully
- `PodFailed`: Validation failed
- `EnvironmentSetupFailed`: Environment setup failed
- `DependencyInstallFailed`: Dependency installation failed
- `NotebookExecutionFailed`: Notebook execution failed
- `ConfigurationError`: Configuration error detected

### Checking Conditions

```bash
oc get notebookvalidationjob <job-name> -o yaml | grep -A 10 conditions:
```

Example output:
```yaml
conditions:
- lastTransitionTime: "2025-11-08T10:00:00Z"
  message: "Failed to install Papermill. Consider using a custom image."
  reason: DependencyInstallFailed
  status: "False"
  type: EnvironmentReady
```

## Best Practices

### 1. Use Custom Images for Production

Pre-install all dependencies in a custom image:

```dockerfile
FROM quay.io/jupyter/scipy-notebook:latest

USER root

# Install Papermill and common ML libraries
RUN pip install --no-cache-dir \
    papermill \
    nbformat \
    nbconvert \
    scikit-learn \
    pandas \
    numpy \
    matplotlib

USER ${NB_UID}
```

### 2. Test Notebooks Locally First

Before deploying to the operator, test with Papermill locally:

```bash
papermill notebook.ipynb output.ipynb --log-output --progress-bar
```

### 3. Use Appropriate Resource Limits

Set realistic resource limits based on your notebook's requirements:

```yaml
spec:
  podConfig:
    resources:
      limits:
        memory: "8Gi"  # For ML training
        cpu: "4"
      requests:
        memory: "4Gi"
        cpu: "2"
```

### 4. Monitor Validation Jobs

Use the operator's observability features:

```bash
# Check job status
oc get notebookvalidationjob

# View detailed status
oc describe notebookvalidationjob <job-name>

# Check pod logs
oc logs <validation-pod-name>
```

### 5. Handle Transient Failures

The operator automatically retries transient failures with exponential backoff. For persistent failures, check the error category and follow the resolution steps above.

## Troubleshooting Workflow

1. **Check Job Status**:
   ```bash
   oc get notebookvalidationjob <job-name> -o yaml
   ```

2. **Identify Error Category**:
   Look for `error_category` in the status message or results JSON

3. **Review Pod Logs**:
   ```bash
   oc logs <validation-pod-name> -n <namespace>
   ```

4. **Check Conditions**:
   ```bash
   oc get notebookvalidationjob <job-name> -o jsonpath='{.status.conditions}'
   ```

5. **Apply Resolution**:
   Follow the resolution steps for the specific error category

6. **Retest**:
   Delete and recreate the NotebookValidationJob:
   ```bash
   oc delete notebookvalidationjob <job-name>
   oc apply -f your-job.yaml
   ```

## Getting Help

If you encounter issues not covered in this guide:

1. Check the [ADR documentation](docs/adrs/) for architectural decisions
2. Review the [OpenShift compatibility guide](docs/adrs/005-openshift-compatibility.md)
3. Open an issue on GitHub with:
   - Error category
   - Full error message
   - Pod logs
   - NotebookValidationJob YAML (sanitized)

