# Tekton Build Setup Guide

## Overview

The Jupyter Notebook Validator Operator supports building custom container images using Tekton Pipelines (OpenShift Pipelines). This guide explains the prerequisites and setup required for Tekton builds to work correctly.

## Prerequisites

### 1. OpenShift Pipelines Operator

Tekton builds require the OpenShift Pipelines operator to be installed:

```bash
# Check if OpenShift Pipelines is installed
oc get operators | grep openshift-pipelines

# If not installed, install it from OperatorHub
# Navigate to: Operators → OperatorHub → Search for "OpenShift Pipelines"
```

### 2. Pipeline ServiceAccount with pipelines-scc

The buildah task used for building container images requires privileged access. The operator automatically creates a `pipeline` ServiceAccount in each namespace where Tekton builds are used, but **you must grant it the `pipelines-scc` Security Context Constraint**.

#### Why is this needed?

- The buildah task needs to run containers with elevated privileges to build images
- OpenShift's default `restricted-v2` SCC doesn't allow the necessary capabilities
- The `pipelines-scc` SCC is specifically designed for Tekton build tasks

#### Setup Instructions

**For each namespace where you'll run Tekton builds**, grant the `pipelines-scc` to the `pipeline` ServiceAccount:

```bash
# Replace <namespace> with your target namespace
NAMESPACE="your-namespace"

# Grant pipelines-scc to the pipeline ServiceAccount
oc adm policy add-scc-to-user pipelines-scc -z pipeline -n $NAMESPACE
```

**Example for e2e-tests namespace:**

```bash
oc adm policy add-scc-to-user pipelines-scc -z pipeline -n e2e-tests
```

#### Verification

Verify the ServiceAccount has the correct SCC:

```bash
# Check if pipeline ServiceAccount exists
oc get sa pipeline -n <namespace>

# Verify SCC binding
oc describe sa pipeline -n <namespace>
```

## How It Works

### Operator Behavior

When you create a NotebookValidationJob with Tekton build enabled:

1. **Operator checks for pipeline ServiceAccount**
   - If it doesn't exist, the operator creates it automatically
   - The ServiceAccount is labeled with `app.kubernetes.io/managed-by: jupyter-notebook-validator-operator`

2. **Operator logs a warning if SCC is not granted**
   - The operator cannot grant SCC permissions itself (requires cluster-admin)
   - You'll see a log message: "NOTE: The pipeline ServiceAccount needs pipelines-scc..."

3. **PipelineRun uses the pipeline ServiceAccount**
   - All Tekton tasks run with the `pipeline` ServiceAccount
   - The buildah task requires the `pipelines-scc` to function

### What Happens Without pipelines-scc?

If the `pipeline` ServiceAccount doesn't have `pipelines-scc`, the buildah task will fail with:

```
pods "...-build-image-pod" is forbidden: unable to validate against any security context constraint
```

This error indicates that the pod cannot run because it violates all available SCCs.

## Troubleshooting

### Issue: buildah task fails with SCC violation

**Symptoms:**
```
pods "tier2-tekton-build-test-build-build-image-pod" is forbidden: unable to validate against any security context constraint
```

**Solution:**
Grant `pipelines-scc` to the `pipeline` ServiceAccount:

```bash
oc adm policy add-scc-to-user pipelines-scc -z pipeline -n <namespace>
```

### Issue: pipeline ServiceAccount doesn't exist

**Symptoms:**
```
Error from server (NotFound): serviceaccounts "pipeline" not found
```

**Solution:**
The operator creates the ServiceAccount automatically when you create a NotebookValidationJob with Tekton build enabled. If it doesn't exist:

1. Check operator logs for errors:
   ```bash
   oc logs -n jupyter-notebook-validator-operator -l control-plane=controller-manager -c manager --tail=50
   ```

2. Manually create the ServiceAccount:
   ```bash
   oc create sa pipeline -n <namespace>
   oc adm policy add-scc-to-user pipelines-scc -z pipeline -n <namespace>
   ```

### Issue: Tekton tasks not found

**Symptoms:**
```
Error: tasks.tekton.dev "git-clone" not found
Error: tasks.tekton.dev "buildah" not found
```

**Solution:**
The operator automatically copies required tasks from `openshift-pipelines` namespace. If this fails:

1. Verify OpenShift Pipelines is installed:
   ```bash
   oc get tasks -n openshift-pipelines | grep -E "git-clone|buildah"
   ```

2. Check operator logs for task copy errors:
   ```bash
   oc logs -n jupyter-notebook-validator-operator -l control-plane=controller-manager -c manager --tail=100 | grep "Task"
   ```

## Security Considerations

### Why pipelines-scc is Safe

The `pipelines-scc` SCC is designed specifically for Tekton build workloads:

- **Scoped to specific ServiceAccount**: Only the `pipeline` ServiceAccount has these privileges
- **Namespace-isolated**: Each namespace has its own `pipeline` ServiceAccount
- **Build-only**: Only used for building images, not for running validation pods
- **OpenShift standard**: This is the standard approach for Tekton builds on OpenShift

### Principle of Least Privilege

- **Validation pods** run with the `default` ServiceAccount (restricted-v2 SCC)
- **Build pods** run with the `pipeline` ServiceAccount (pipelines-scc SCC)
- The operator itself runs with minimal RBAC permissions

## Example: Complete Setup

Here's a complete example of setting up a namespace for Tekton builds:

```bash
# 1. Create namespace
oc create namespace my-notebooks

# 2. Grant pipelines-scc to pipeline ServiceAccount
# (The operator will create the ServiceAccount automatically)
oc adm policy add-scc-to-user pipelines-scc -z pipeline -n my-notebooks

# 3. Create Git credentials secret (if using private repo)
oc create secret generic git-credentials \
  --from-literal=username=your-github-username \
  --from-literal=password=your-github-token \
  --type=kubernetes.io/basic-auth \
  -n my-notebooks

# 4. Create NotebookValidationJob with Tekton build
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-notebook-validation
  namespace: my-notebooks
spec:
  notebook:
    git:
      url: "https://github.com/your-org/your-repo.git"
      ref: "main"
      credentialsSecret: "git-credentials"
    path: "notebooks/my-notebook.ipynb"
  
  podConfig:
    containerImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
    buildConfig:
      enabled: true
      strategy: "tekton"
      baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
  
  timeout: "30m"
EOF
```

## Related Documentation

- [ADR-028: Tekton Task Strategy](adrs/028-tekton-task-strategy.md)
- [ADR-031: Tekton Build Strategy](adrs/031-tekton-build-strategy.md)
- [Namespace Setup Guide](NAMESPACE_SETUP.md)
- [OpenShift Pipelines Documentation](https://docs.openshift.com/pipelines/latest/about/understanding-openshift-pipelines.html)

