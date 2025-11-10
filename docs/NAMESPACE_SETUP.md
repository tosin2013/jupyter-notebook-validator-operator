# Namespace Setup for NotebookValidationJobs

## Overview

The Jupyter Notebook Validator Operator uses a **mutating admission webhook** to automatically inject the `default` ServiceAccount into `NotebookValidationJob` resources. This means **no manual ServiceAccount setup is required** in most cases.

## Automatic ServiceAccount Injection

When you create a `NotebookValidationJob`, the operator's mutating webhook automatically:

1. Detects if `spec.podConfig.serviceAccountName` is not specified
2. Injects `serviceAccountName: default` into the resource
3. The validation pod runs using the `default` ServiceAccount in your namespace

This design follows industry best practices from:
- OpenTelemetry Operator (annotation-based injection)
- Istio (sidecar injection)
- Vault Agent Injector (secret injection)

## Benefits

- **Zero configuration**: Works out of the box in any namespace
- **No manual RBAC setup**: The `default` ServiceAccount exists in all namespaces
- **Security isolation**: Validation pods only have access to resources in their namespace
- **Multi-tenancy support**: Different teams can use the operator in their own namespaces
- **Audit trail**: Clear separation of permissions per namespace

## Default ServiceAccount Permissions

The `default` ServiceAccount in each namespace has minimal permissions by default. If your notebooks need additional permissions (e.g., to access Secrets or ConfigMaps), you can grant them to the `default` ServiceAccount:

```bash
# Example: Grant permission to read secrets in a namespace
NAMESPACE="your-namespace"

oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: notebook-validator-permissions
  namespace: ${NAMESPACE}
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

oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: notebook-validator-permissions
  namespace: ${NAMESPACE}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: notebook-validator-permissions
subjects:
  - kind: ServiceAccount
    name: default
    namespace: ${NAMESPACE}
EOF
```

## Custom ServiceAccount (Advanced)

If you need to use a custom ServiceAccount instead of `default`, you can specify it in your `NotebookValidationJob`:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation-job
  namespace: my-namespace
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
    path: my-notebook.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: my-custom-sa  # Override the default
```

In this case, you must create the custom ServiceAccount manually:

## Manual ServiceAccount Setup (Legacy/Advanced)

If you need to create a custom ServiceAccount for specific use cases, follow these steps:

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

Then specify it in your NotebookValidationJob:

```yaml
spec:
  podConfig:
    serviceAccountName: jupyter-notebook-validator-runner  # Use your custom SA
```

## Verification

Verify the webhook is working:

```bash
# Check that the operator webhook is running
oc get pods -n jupyter-notebook-validator-operator

# Check webhook configuration
oc get mutatingwebhookconfiguration | grep notebook-validator

# Check cert-manager certificate
oc get certificate -n jupyter-notebook-validator-operator
```

## Create Your First NotebookValidationJob

You can create NotebookValidationJobs in any namespace without any setup:

```bash
# Create a namespace (if it doesn't exist)
oc create namespace my-namespace

# Create a NotebookValidationJob - no ServiceAccount setup needed!
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-first-validation
  namespace: my-namespace
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    # serviceAccountName is automatically injected as "default" by the webhook
  timeout: "5m"
EOF
```

Verify the webhook injected the ServiceAccount:

```bash
# Check the created resource
oc get notebookvalidationjob my-first-validation -n my-namespace -o yaml | grep serviceAccountName

# You should see: serviceAccountName: default
```

## Troubleshooting

### Webhook Not Injecting ServiceAccount

**Symptom:**
The `serviceAccountName` field is not being set to `default` automatically.

**Solution:**
1. Verify the webhook is running:
   ```bash
   oc get pods -n jupyter-notebook-validator-operator
   ```

2. Check webhook configuration:
   ```bash
   oc get mutatingwebhookconfiguration notebook-validator-mutating-webhook-configuration -o yaml
   ```

3. Check cert-manager certificate:
   ```bash
   oc get certificate -n jupyter-notebook-validator-operator
   oc describe certificate serving-cert -n jupyter-notebook-validator-operator
   ```

4. Check webhook logs:
   ```bash
   oc logs -n jupyter-notebook-validator-operator deployment/notebook-validator-controller-manager
   ```

### Error: serviceaccount "default" not found

**Symptom:**
```
Failed to create validation pod: pods "my-validation-validation" is forbidden:
error looking up service account my-namespace/default:
serviceaccount "default" not found
```

**Solution:**
This is extremely rare as the `default` ServiceAccount is automatically created in all namespaces. If you encounter this:

1. Verify the namespace exists:
   ```bash
   oc get namespace my-namespace
   ```

2. Check if the default ServiceAccount exists:
   ```bash
   oc get serviceaccount default -n my-namespace
   ```

3. If it doesn't exist, create it:
   ```bash
   oc create serviceaccount default -n my-namespace
   ```

### Error: pods is forbidden

**Symptom:**
```
Failed to create validation pod: pods is forbidden: User "system:serviceaccount:..."
cannot create resource "pods" in API group "" in the namespace "..."
```

**Solution:**
This error is about the **operator's** ServiceAccount, not the validation pod's ServiceAccount. Ensure the operator has proper RBAC permissions. This should be handled automatically by the operator installation.

### Validation Pod Needs Additional Permissions

**Symptom:**
Your notebook needs to access Secrets or ConfigMaps, but the validation pod fails with permission errors.

**Solution:**
Grant additional permissions to the `default` ServiceAccount in your namespace (see "Default ServiceAccount Permissions" section above).

## Architecture Notes

### How the Webhook Works

1. **User creates NotebookValidationJob**: You create a NotebookValidationJob resource without specifying `serviceAccountName`
2. **Webhook intercepts**: The mutating webhook intercepts the CREATE/UPDATE request
3. **Default injection**: The webhook's `Default()` method checks if `serviceAccountName` is empty
4. **Sets default**: If empty, it sets `serviceAccountName: default`
5. **Resource created**: The modified resource is created in the cluster
6. **Pod creation**: When the operator creates the validation pod, it uses the injected ServiceAccount

### Why "default" ServiceAccount?

Based on research into production operators (OpenTelemetry, Istio, Vault):

- **Simplest approach**: Works immediately in any namespace
- **Zero configuration**: No manual setup required
- **Principle of least surprise**: `default` SA exists in all namespaces
- **Flexible**: Users can override by specifying a custom SA
- **Secure**: Namespace-scoped isolation is maintained

### Future Enhancements

Planned enhancements for ServiceAccount management:

1. **Annotation-based injection**: Use annotations to specify custom ServiceAccounts
   ```yaml
   metadata:
     annotations:
       notebook-validator.mlops.dev/service-account: my-custom-sa
   ```

2. **Namespace-level defaults**: Configure default ServiceAccount per namespace
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: notebook-validator-config
     namespace: my-namespace
   data:
     defaultServiceAccount: my-team-sa
   ```

3. **Automatic RBAC provisioning**: Automatically create minimal RBAC for validation pods

## Security Considerations

The `default` ServiceAccount has **minimal permissions** by default:
- ❌ No access to Secrets
- ❌ No access to ConfigMaps
- ❌ No write access to any resources
- ❌ No access to other namespaces

This follows the **principle of least privilege** and ensures validation pods cannot modify cluster resources unless explicitly granted permissions.

If your notebooks need additional permissions, grant them explicitly to the `default` ServiceAccount in your namespace (see "Default ServiceAccount Permissions" section above).

## Related Documentation

- [ADR-005: RBAC and Service Account Model](adrs/005-rbac-and-service-account-model.md)
- [Mutating Admission Webhooks (Kubernetes Docs)](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Operator Installation Guide](../README.md#installation)
- [NotebookValidationJob API Reference](../api/v1alpha1/notebookvalidationjob_types.go)

