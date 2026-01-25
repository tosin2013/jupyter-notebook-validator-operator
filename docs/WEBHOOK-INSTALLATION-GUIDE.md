# Webhook Installation Guide

## Overview

The Jupyter Notebook Validator Operator supports **admission webhooks** for enhanced validation and user experience. Webhooks are **optional** but **highly recommended** for production deployments.

## Why Use Webhooks? 🎯

### ✅ **With Webhooks Enabled** (Recommended)

**Mutating Webhook** (Automatic Defaults):
- ✅ Automatically sets `serviceAccountName: "default"` if not specified
- ✅ Automatically sets `timeout: "30m"` if not specified
- ✅ Converts `credentials: [secret-name]` to `envFrom` (simplified syntax)

**Validating Webhook** (Early Error Detection):
- ✅ **Prevents reserved volume names** (`git-clone`, `notebook-data`, `source`)
- ✅ **Catches duplicate volume names** at creation time
- ✅ **Validates volume mounts** reference existing volumes
- ✅ **Clear error messages** instead of cryptic pod failures

**Example - Simplified Syntax**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/analysis.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    credentials:  # ← Simplified syntax (webhook converts to envFrom)
      - my-credentials-secret
  # ← No serviceAccountName needed (webhook sets default)
  # ← No timeout needed (webhook sets 30m default)
```

### ⚠️ **Without Webhooks** (Manual Mode)

**Required Manual Configuration**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/analysis.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    serviceAccountName: "default"  # ← REQUIRED!
    envFrom:  # ← Verbose syntax required
      - secretRef:
          name: my-credentials-secret
  timeout: "30m"  # ← REQUIRED!
```

**Limitations**:
- ❌ No automatic defaults (must specify `serviceAccountName` and `timeout`)
- ❌ No volume validation (errors happen at pod runtime, not creation time)
- ❌ Verbose credential syntax required
- ❌ Cryptic pod failure messages for volume conflicts

## Installation Options

### Option 1: Webhooks Enabled (Recommended for Production)

#### Prerequisites

1. **Install cert-manager** (for automatic TLS certificate management):

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=Available --timeout=300s \
  deployment/cert-manager -n cert-manager
kubectl wait --for=condition=Available --timeout=300s \
  deployment/cert-manager-webhook -n cert-manager
kubectl wait --for=condition=Available --timeout=300s \
  deployment/cert-manager-cainjector -n cert-manager
```

#### Install Operator with Webhooks

```bash
# Install from OperatorHub.io (webhooks enabled by default)
kubectl create -f https://operatorhub.io/install/jupyter-notebook-validator-operator.yaml

# Or install via OLM
kubectl create -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: operators
spec:
  channel: alpha
  name: jupyter-notebook-validator-operator
  source: operatorhubio-catalog
  sourceNamespace: olm
EOF
```

#### Verify Webhook Installation

```bash
# Check webhook configurations
kubectl get validatingwebhookconfigurations | grep jupyter-notebook
kubectl get mutatingwebhookconfigurations | grep jupyter-notebook

# Check operator logs
kubectl logs -n operators deployment/jupyter-notebook-validator-operator-controller-manager | grep webhook
```

Expected output:
```
webhooks enabled - configuring webhook server
```

### Option 2: Webhooks Disabled (CI/Testing/Minimal Setup)

If you cannot install cert-manager or prefer a simpler setup:

#### Install Operator without Webhooks

```bash
# Install from OperatorHub.io
kubectl create -f https://operatorhub.io/install/jupyter-notebook-validator-operator.yaml

# Disable webhooks by patching the deployment
kubectl patch deployment jupyter-notebook-validator-operator-controller-manager \
  -n operators \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/env/0/value", "value": "false"}]'
```

Or create a custom Subscription with webhooks disabled:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: operators
spec:
  channel: alpha
  name: jupyter-notebook-validator-operator
  source: operatorhubio-catalog
  sourceNamespace: olm
  config:
    env:
      - name: ENABLE_WEBHOOKS
        value: "false"
```

## Comparison Table

| Feature | Webhooks Enabled | Webhooks Disabled |
|---------|------------------|-------------------|
| **Setup Complexity** | Medium (requires cert-manager) | Low (no dependencies) |
| **Startup Time** | ~60-90s (cert injection) | ~10-20s |
| **User Experience** | ✅ Excellent (automatic defaults) | ⚠️ Manual (verbose config) |
| **Volume Validation** | ✅ Early (at creation) | ❌ Late (at pod runtime) |
| **Error Messages** | ✅ Clear validation errors | ❌ Cryptic pod failures |
| **Credential Syntax** | ✅ Simple (`credentials: [...]`) | ⚠️ Verbose (`envFrom: [...]`) |
| **Production Ready** | ✅ Yes | ⚠️ Requires careful documentation |

## Troubleshooting

### Webhook Certificate Issues

If you see errors like `x509: certificate signed by unknown authority`:

```bash
# Check cert-manager is running
kubectl get pods -n cert-manager

# Check certificate was created
kubectl get certificate -n operators

# Check webhook service
kubectl get service -n operators | grep webhook

# Restart operator to trigger cert injection
kubectl rollout restart deployment/jupyter-notebook-validator-operator-controller-manager -n operators
```

### Webhook Timeout Issues

If operator pod fails to start with webhook errors:

```bash
# Check webhook server is listening
kubectl logs -n operators deployment/jupyter-notebook-validator-operator-controller-manager | grep "webhook server"

# Check webhook port is accessible
kubectl port-forward -n operators deployment/jupyter-notebook-validator-operator-controller-manager 9443:9443
```

## Recommendation

✅ **For Production**: Use **webhooks enabled** with cert-manager
- Better user experience
- Early error detection
- Prevents common configuration mistakes

⚠️ **For CI/Testing**: Use **webhooks disabled**
- Faster startup
- No cert-manager dependency
- Simpler troubleshooting

## Additional Resources

- [Webhook Implementation](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/api/v1alpha1/notebookvalidationjob_webhook.go)
- [Architecture Overview](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/docs/ARCHITECTURE_OVERVIEW.md)
- [cert-manager Documentation](https://cert-manager.io/docs/)

