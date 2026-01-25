# Webhook Configuration Guide

The Jupyter Notebook Validator Operator supports optional admission webhooks for validating `NotebookValidationJob` resources before they are created or updated.

## Overview

Webhooks provide:
- **Validation**: Ensure NotebookValidationJob resources have valid configurations
- **Mutation**: Apply default values to resources

**Webhooks are optional** - the operator functions fully without them. They add an extra layer of validation but require cert-manager or manual certificate management.

## Configuration

### Environment Variable

Set the `ENABLE_WEBHOOKS` environment variable:

```yaml
env:
- name: ENABLE_WEBHOOKS
  value: "true"   # Enable webhooks
  # value: "false" # Disable webhooks (default)
```

Accepted values: `true`, `1`, `yes` (case-insensitive) to enable; any other value or unset to disable.

### Command-Line Flag

Alternatively, use the `--enable-webhooks` flag:

```bash
/manager --enable-webhooks=true
```

**Note**: The environment variable takes precedence if set.

## Deployment Scenarios

### Scenario 1: Vanilla Kubernetes (No cert-manager)

**Recommended: Disable webhooks**

```yaml
# In your deployment
env:
- name: ENABLE_WEBHOOKS
  value: "false"
```

The operator will work without webhooks. Validation is performed by the controller during reconciliation.

### Scenario 2: Kubernetes with cert-manager

**Enable webhooks with cert-manager managing certificates**

1. Install cert-manager:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
   ```

2. Deploy with webhooks enabled:
   ```yaml
   env:
   - name: ENABLE_WEBHOOKS
     value: "true"
   ```

3. Create a Certificate resource for the webhook:
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: Certificate
   metadata:
     name: webhook-server-cert
     namespace: jupyter-notebook-validator-operator-system
   spec:
     secretName: webhook-server-cert
     dnsNames:
     - notebook-validator-webhook-service.jupyter-notebook-validator-operator-system.svc
     - notebook-validator-webhook-service.jupyter-notebook-validator-operator-system.svc.cluster.local
     issuerRef:
       name: selfsigned-issuer
       kind: ClusterIssuer
   ```

### Scenario 3: OpenShift with Service Serving Certificates

OpenShift can automatically manage certificates via service serving certificates:

1. Annotate the webhook service:
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: notebook-validator-webhook-service
     annotations:
       service.beta.openshift.io/serving-cert-secret-name: webhook-server-cert
   ```

2. Deploy with webhooks enabled:
   ```yaml
   env:
   - name: ENABLE_WEBHOOKS
     value: "true"
   ```

### Scenario 4: OperatorHub.io / Community Operators

**Webhooks are disabled by default** for OperatorHub.io deployments because:
- The CI environment doesn't have cert-manager
- Users may not have cert-manager installed

To enable webhooks after installing from OperatorHub:
1. Install cert-manager
2. Edit the operator deployment to set `ENABLE_WEBHOOKS=true`
3. Create the webhook certificate

## Manual Configuration When Webhooks Are Disabled

When webhooks are disabled (`ENABLE_WEBHOOKS=false`), the mutating webhook that applies default values does not run. Users must manually specify certain fields that would otherwise be auto-populated.

### What Changes Without Webhooks

| Feature | With Webhooks (Automatic) | Without Webhooks (Manual Required) |
|---------|---------------------------|-----------------------------------|
| **Credentials shorthand** | `credentials: ["my-secret"]` auto-converts to `envFrom` | Must use verbose `envFrom` syntax |
| **ServiceAccount** | Auto-defaults to `"default"` | Must explicitly specify `serviceAccountName` |
| **Timeout** | Auto-defaults to `"30m"` | Must explicitly specify `timeout` |
| **Validation timing** | Immediate rejection on CREATE/UPDATE | Deferred to controller reconciliation |
| **Error visibility** | Admission error on `kubectl apply` | Errors appear in `.status.conditions` |

### Required Manual Configuration

#### 1. ServiceAccount Name (Required)

Without webhooks, you **must** specify `serviceAccountName` in your NotebookValidationJob:

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
    serviceAccountName: "default"  # REQUIRED when webhooks disabled
  timeout: "30m"                    # REQUIRED when webhooks disabled
```

#### 2. Credentials Configuration (Verbose Syntax Required)

The `credentials` shorthand field is **not available** without webhooks. You must use the standard Kubernetes `envFrom` syntax:

**With webhooks (shorthand available):**
```yaml
spec:
  podConfig:
    credentials:
      - "aws-credentials"
      - "database-credentials"
```

**Without webhooks (verbose syntax required):**
```yaml
spec:
  podConfig:
    envFrom:
      - secretRef:
          name: "aws-credentials"
      - secretRef:
          name: "database-credentials"
```

#### 3. Timeout (Required)

Without webhooks, always specify the `timeout` field:

```yaml
spec:
  timeout: "30m"  # Required - no default applied without webhooks
```

### Complete Example for OperatorHub.io Installations

Here's a complete NotebookValidationJob spec that works without webhooks:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-notebook-validation
  namespace: my-namespace
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
      # For private repos, reference a secret:
      # credentialsSecret: "git-credentials"
    path: "notebooks/analysis.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/scipy-notebook:latest"
    serviceAccountName: "default"  # REQUIRED
    # Use envFrom instead of credentials shorthand:
    envFrom:
      - secretRef:
          name: "my-credentials"
    env:
      - name: "MY_VAR"
        value: "my-value"
    resources:
      limits:
        cpu: "2"
        memory: "4Gi"
      requests:
        cpu: "500m"
        memory: "1Gi"
  timeout: "30m"  # REQUIRED
```

### CRD Schema Validation (Always Active)

Even without webhooks, the CRD's OpenAPI schema validation is **always enforced**:

- Git URL must match pattern: `^((https?|git|ssh)://|git@).*$`
- Notebook path must end with `.ipynb`
- Timeout must be a valid duration string (e.g., `30m`, `1h`, `90s`)

These validations happen at the Kubernetes API level and do not require webhooks.

### Checking for Validation Errors

Without webhooks, validation errors appear in the resource status after the controller processes it:

```bash
# Check job status
kubectl get notebookvalidationjob my-validation -o yaml

# Look at the status.conditions for any errors
kubectl get notebookvalidationjob my-validation -o jsonpath='{.status.conditions}'
```

### Enabling Webhooks After OperatorHub Installation

If you want the convenience of automatic defaults and early validation, you can enable webhooks after installation:

1. **Install cert-manager**:
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
   ```

2. **Create a self-signed ClusterIssuer**:
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: ClusterIssuer
   metadata:
     name: selfsigned-issuer
   spec:
     selfSigned: {}
   ```

3. **Create the webhook certificate**:
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: Certificate
   metadata:
     name: webhook-server-cert
     namespace: <operator-namespace>
   spec:
     secretName: webhook-server-cert
     dnsNames:
       - notebook-validator-webhook-service.<operator-namespace>.svc
       - notebook-validator-webhook-service.<operator-namespace>.svc.cluster.local
     issuerRef:
       name: selfsigned-issuer
       kind: ClusterIssuer
   ```

4. **Update the operator deployment**:
   ```bash
   kubectl set env deployment/notebook-validator-controller-manager \
     -n <operator-namespace> \
     ENABLE_WEBHOOKS=true
   ```

5. **Restart the operator** to pick up the new configuration:
   ```bash
   kubectl rollout restart deployment/notebook-validator-controller-manager \
     -n <operator-namespace>
   ```

## Helm Chart Configuration

The Helm chart supports webhook configuration:

```yaml
# values.yaml
webhooks:
  enabled: false  # Set to true to enable webhooks

certManager:
  enabled: false  # Set to true if using cert-manager
```

Install with webhooks:
```bash
helm install jupyter-notebook-validator ./helm/jupyter-notebook-validator-operator \
  --set webhooks.enabled=true \
  --set certManager.enabled=true
```

## Troubleshooting

### Operator crashes with "unable to create webhook"

**Cause**: Webhooks are enabled but the certificate secret doesn't exist.

**Solution**: Either:
1. Disable webhooks: Set `ENABLE_WEBHOOKS=false`
2. Install cert-manager and create the certificate

### Webhook returns "connection refused"

**Cause**: The webhook service isn't reachable or TLS isn't configured correctly.

**Solution**:
1. Verify the webhook service exists
2. Check the certificate secret is mounted
3. Review operator logs for TLS errors

### ValidationWebhookConfiguration not created

**Cause**: Webhooks are disabled in the operator.

**Solution**: This is expected when `ENABLE_WEBHOOKS=false`. Enable webhooks if you need them.

## Verification

Check if webhooks are enabled by viewing operator logs:

```bash
kubectl logs -n jupyter-notebook-validator-operator-system deployment/notebook-validator-controller-manager

# Look for:
# "webhooks enabled - configuring webhook server"
# or
# "webhooks disabled - operator will run without admission webhooks"
```

## Security Considerations

- Webhooks require TLS certificates
- Use cert-manager for automatic certificate rotation
- On OpenShift, use service serving certificates for seamless integration
- The webhook validates all CREATE and UPDATE operations on NotebookValidationJob resources
