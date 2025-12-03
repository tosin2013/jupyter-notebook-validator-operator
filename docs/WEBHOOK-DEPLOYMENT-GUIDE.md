# Webhook Deployment Guide

**Date**: 2025-12-03  
**Purpose**: Explain webhook configuration and certificate management for different deployment methods

## Overview

The Jupyter Notebook Validator Operator uses admission webhooks (mutating and validating) to provide defaults and validate NotebookValidationJob resources. Webhook configuration and certificate management differ based on deployment method.

## Deployment Methods

### Method 1: OLM Bundle Deployment (Production)

**Use Case**: Production deployments, OperatorHub installations, community-operators submissions

**Certificate Management**: OLM automatically handles all certificate management

#### How It Works

1. **OLM Generates Certificates**
   - Creates self-signed CA certificates (valid for 2 years)
   - Automatically rotates certificates before expiration
   - Mounts certificates to deployment at standard paths

2. **Certificate Locations**
   ```
   /tmp/k8s-webhook-server/serving-certs/tls.cert
   /tmp/k8s-webhook-server/serving-certs/tls.key
   ```

3. **OLM Creates Webhook Configurations**
   - Reads `webhookdefinitions` from ClusterServiceVersion (CSV)
   - Creates MutatingWebhookConfiguration and ValidatingWebhookConfiguration
   - Injects CA bundle into webhook configurations

4. **No Additional Setup Required**
   - No cert-manager needed
   - No manual webhook configuration
   - No service-ca annotations

#### Bundle Configuration

The operator bundle includes webhook definitions in the CSV:

```yaml
# bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml
webhookdefinitions:
- type: MutatingAdmissionWebhook
  admissionReviewVersions: [v1]
  containerPort: 443
  targetPort: 9443
  deploymentName: notebook-validator-controller-manager
  failurePolicy: Fail
  generateName: mnotebookvalidationjob.kb.io
  webhookPath: /mutate-mlops-mlops-dev-v1alpha1-notebookvalidationjob
  rules:
  - apiGroups: [mlops.mlops.dev]
    apiVersions: [v1alpha1]
    operations: [CREATE, UPDATE]
    resources: [notebookvalidationjobs]
  sideEffects: None

- type: ValidatingAdmissionWebhook
  admissionReviewVersions: [v1]
  containerPort: 443
  targetPort: 9443
  deploymentName: notebook-validator-controller-manager
  failurePolicy: Fail
  generateName: vnotebookvalidationjob.kb.io
  webhookPath: /validate-mlops-mlops-dev-v1alpha1-notebookvalidationjob
  rules:
  - apiGroups: [mlops.mlops.dev]
    apiVersions: [v1alpha1]
    operations: [CREATE, UPDATE]
    resources: [notebookvalidationjobs]
  sideEffects: None
```

#### Deployment Manifest

The CSV deployment includes webhook server configuration:

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        volumeMounts:
        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: cert
          readOnly: true
      volumes:
      - name: cert
        secret:
          defaultMode: 420
          secretName: webhook-server-cert  # OLM creates this
```

#### Installation

```bash
# Install via OLM (OperatorHub)
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-notebook-validator-operator
spec:
  channel: stable
  name: jupyter-notebook-validator-operator
  source: jupyter-notebook-validator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF
```

### Method 2: Direct Deployment with OpenShift Service-CA (Development/Testing)

**Use Case**: Local development, quick testing on OpenShift, CI/CD pipelines

**Certificate Management**: OpenShift service-ca-operator handles certificates

#### How It Works

1. **Service-CA Annotation**
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: notebook-validator-webhook-service
     annotations:
       service.beta.openshift.io/serving-cert-secret-name: webhook-server-cert
   ```

2. **Service-CA Creates Secret**
   - Generates TLS certificate for service
   - Creates `webhook-server-cert` secret
   - Automatically rotates certificates

3. **CA Bundle Injection**
   ```yaml
   apiVersion: admissionregistration.k8s.io/v1
   kind: MutatingWebhookConfiguration
   metadata:
     annotations:
       service.beta.openshift.io/inject-cabundle: "true"
   ```

#### Installation

```bash
# Deploy operator
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18

# Apply webhook configuration
oc apply -f config/quick-deploy/webhook-service.yaml
```

### Method 3: Helm Chart with cert-manager (Kubernetes)

**Use Case**: Kubernetes clusters without OpenShift, GitOps deployments

**Certificate Management**: cert-manager handles certificates

#### Prerequisites

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

#### Installation

```bash
helm install jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace \
  --set certManager.enabled=true \
  --set operator.webhook.enabled=true
```

## Comparison Matrix

| Feature | OLM Bundle | OpenShift Service-CA | Helm + cert-manager |
|---------|-----------|---------------------|---------------------|
| **Certificate Management** | OLM | service-ca-operator | cert-manager |
| **Webhook Configuration** | OLM (from CSV) | Manual (YAML) | Helm templates |
| **Certificate Rotation** | Automatic (OLM) | Automatic (service-ca) | Automatic (cert-manager) |
| **Platform** | OpenShift/K8s | OpenShift only | Kubernetes |
| **Use Case** | Production | Development/Testing | Kubernetes Production |
| **Setup Complexity** | Low (OLM handles) | Medium (manual YAML) | Medium (cert-manager) |
| **OperatorHub Ready** | ✅ Yes | ❌ No | ❌ No |

## Troubleshooting

### Webhook Not Working (OLM Deployment)

```bash
# Check CSV status
oc get csv -n jupyter-notebook-validator-operator

# Check webhook configurations
oc get mutatingwebhookconfiguration | grep notebook-validator
oc get validatingwebhookconfiguration | grep notebook-validator

# Check certificate secret
oc get secret -n jupyter-notebook-validator-operator | grep webhook-server-cert

# Check operator logs
oc logs -n jupyter-notebook-validator-operator \
  deployment/notebook-validator-controller-manager -c manager
```

### Certificate Issues

```bash
# Check certificate expiration (OLM)
oc get secret webhook-server-cert -n jupyter-notebook-validator-operator \
  -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates

# Force certificate rotation (delete secret, OLM recreates)
oc delete secret webhook-server-cert -n jupyter-notebook-validator-operator
```

## References

- [OLM Webhook Documentation](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/)
- [OpenShift Service-CA](https://docs.openshift.com/container-platform/latest/security/certificates/service-serving-certificate.html)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Kubebuilder Webhook Guide](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html)

