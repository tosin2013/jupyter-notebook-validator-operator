# Namespace Setup for NotebookValidationJobs

## Overview

When creating `NotebookValidationJob` resources in a namespace other than the operator's namespace, you must first set up the required RBAC resources in that namespace.

## Why is this required?

The Jupyter Notebook Validator Operator follows the **principle of least privilege** and uses namespace-scoped ServiceAccounts for validation pods. This ensures:

- **Security isolation**: Validation pods only have access to resources in their namespace
- **Multi-tenancy support**: Different teams can use the operator in their own namespaces
- **Audit trail**: Clear separation of permissions per namespace

## Quick Setup

Run the following commands to set up a namespace for NotebookValidationJobs:

```bash
# Set your namespace
NAMESPACE="your-namespace"

# Create the namespace (if it doesn't exist)
oc create namespace ${NAMESPACE}

# Create ServiceAccount
oc apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: jupyter-notebook-validator-runner
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: validation-runner
EOF

# Create Role
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jupyter-notebook-validator-runner-role
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: validation-runner
rules:
  # Allow reading secrets for Git credentials
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  # Allow reading configmaps for configuration
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
EOF

# Create RoleBinding
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jupyter-notebook-validator-runner-rolebinding
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: validation-runner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: jupyter-notebook-validator-runner-role
subjects:
  - kind: ServiceAccount
    name: jupyter-notebook-validator-runner
    namespace: ${NAMESPACE}
EOF
```

## Verification

Verify the setup:

```bash
# Check ServiceAccount
oc get serviceaccount jupyter-notebook-validator-runner -n ${NAMESPACE}

# Check Role
oc get role jupyter-notebook-validator-runner-role -n ${NAMESPACE}

# Check RoleBinding
oc get rolebinding jupyter-notebook-validator-runner-rolebinding -n ${NAMESPACE}
```

## Create Your First NotebookValidationJob

Once the namespace is set up, you can create NotebookValidationJobs:

```bash
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-first-validation
  namespace: ${NAMESPACE}
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
```

## Troubleshooting

### Error: serviceaccount "jupyter-notebook-validator-runner" not found

**Symptom:**
```
Failed to create validation pod: pods "my-validation-validation" is forbidden: 
error looking up service account <namespace>/jupyter-notebook-validator-runner: 
serviceaccount "jupyter-notebook-validator-runner" not found
```

**Solution:**
Run the Quick Setup commands above to create the ServiceAccount in your namespace.

### Error: pods is forbidden

**Symptom:**
```
Failed to create validation pod: pods is forbidden: User "system:serviceaccount:..." 
cannot create resource "pods" in API group "" in the namespace "..."
```

**Solution:**
Ensure the Role and RoleBinding are created correctly. The operator's ServiceAccount needs permission to create pods in your namespace.

## Using a Custom ServiceAccount

If you want to use a different ServiceAccount (e.g., with additional permissions), specify it in your NotebookValidationJob:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: custom-sa-validation
  namespace: ${NAMESPACE}
spec:
  notebook:
    git:
      url: "https://github.com/example/notebooks.git"
      ref: "main"
    path: "notebooks/my-notebook.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    serviceAccountName: "my-custom-serviceaccount"  # Custom SA
  timeout: "5m"
```

**Note:** The custom ServiceAccount must exist in the same namespace and have the necessary permissions.

## Automation with Helm

If you're using the Helm chart, you can configure it to create the validation runner ServiceAccount in multiple namespaces:

```yaml
# values.yaml
validationRunner:
  serviceAccount:
    create: true
    name: "jupyter-notebook-validator-runner"
  
  # Create validation runner RBAC in additional namespaces
  additionalNamespaces:
    - name: "team-a"
    - name: "team-b"
    - name: "e2e-tests"
```

## Security Considerations

The validation runner ServiceAccount has **minimal permissions**:
- ✅ Read-only access to Secrets (for Git credentials)
- ✅ Read-only access to ConfigMaps (for configuration)
- ❌ No write access to any resources
- ❌ No access to other namespaces

This follows the **principle of least privilege** and ensures validation pods cannot modify cluster resources.

## Related Documentation

- [ADR-005: RBAC and Service Account Model](adrs/005-rbac-and-service-account-model.md)
- [Operator Installation Guide](../README.md#installation)
- [NotebookValidationJob API Reference](../api/v1alpha1/notebookvalidationjob_types.go)

