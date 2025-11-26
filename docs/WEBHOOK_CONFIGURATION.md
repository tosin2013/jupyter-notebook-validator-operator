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
