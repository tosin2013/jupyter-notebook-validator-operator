# ADR 016: External Secrets Operator (ESO) Integration

## Status
Accepted

## Context

ADR-014 established a multi-tier credential injection strategy. This ADR focuses on **Tier 2: External Secrets Operator (ESO)**, which enables enterprise users to sync secrets from external vaults into Kubernetes Secrets.

### Problem

Enterprise organizations store secrets in centralized vaults:
- **AWS Secrets Manager**
- **Azure Key Vault**
- **GCP Secret Manager**
- **HashiCorp Vault**
- **1Password**

Without ESO integration, users must:
- Manually copy secrets from vaults to Kubernetes
- Keep secrets in sync manually
- Manage rotation manually
- Lose audit trail from external vault

### External Secrets Operator (ESO)

ESO is a Kubernetes operator that:
1. Syncs secrets from external vaults into Kubernetes Secrets
2. Automatically refreshes secrets on a schedule
3. Supports multiple secret stores (AWS, Azure, GCP, Vault, 1Password)
4. Provides declarative secret management
5. Maintains audit trail from external store

**Key Concepts**:
- **SecretStore**: Defines connection to external vault (namespace-scoped)
- **ClusterSecretStore**: Defines connection to external vault (cluster-scoped)
- **ExternalSecret**: Defines which secrets to sync and how

## Decision

We adopt External Secrets Operator (ESO) as the recommended pattern for enterprise secret management (Tier 2).

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    External Secret Store                     │
│  (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager) │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ ESO syncs secrets
                              │ (refreshInterval: 1h)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              External Secrets Operator (ESO)                 │
│  - Watches ExternalSecret resources                          │
│  - Fetches secrets from external store                       │
│  - Creates/updates Kubernetes Secrets                        │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Creates K8s Secret
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Secret                         │
│  (Synced from external store, auto-refreshed)               │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Mounts as env vars
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              NotebookValidationJob Pod                       │
│  - Reads credentials from env vars                           │
│  - Credentials are always up-to-date                         │
└─────────────────────────────────────────────────────────────┘
```

### Pattern: AWS Secrets Manager

**1. Install ESO** (cluster admin):
```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace
```

**2. Create ClusterSecretStore** (cluster admin):
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets-sa
            namespace: external-secrets-system
```

**3. Create ExternalSecret** (user):
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: aws-credentials
  namespace: notebook-validation
spec:
  refreshInterval: 1h  # Sync every hour
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: aws-credentials  # K8s Secret name
    creationPolicy: Owner
  data:
    - secretKey: access-key-id  # Key in K8s Secret
      remoteRef:
        key: prod/notebook/aws  # Path in AWS Secrets Manager
        property: access_key_id  # JSON property
    - secretKey: secret-access-key
      remoteRef:
        key: prod/notebook/aws
        property: secret_access_key
```

**4. Use in NotebookValidationJob** (user):
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: s3-pipeline-validation
  namespace: notebook-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: s3-pipeline.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials  # Synced by ESO
            key: access-key-id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: secret-access-key
```

### Pattern: Azure Key Vault

**1. Create ClusterSecretStore**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: azure-keyvault
spec:
  provider:
    azurekv:
      vaultUrl: https://my-vault.vault.azure.net
      authType: ManagedIdentity
      identityId: /subscriptions/.../resourceGroups/.../providers/Microsoft.ManagedIdentity/userAssignedIdentities/eso-identity
```

**2. Create ExternalSecret**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
  namespace: notebook-validation
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: azure-keyvault
    kind: ClusterSecretStore
  target:
    name: database-credentials
  data:
    - secretKey: host
      remoteRef:
        key: db-host
    - secretKey: username
      remoteRef:
        key: db-username
    - secretKey: password
      remoteRef:
        key: db-password
```

### Pattern: GCP Secret Manager

**1. Create ClusterSecretStore**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: gcp-secret-manager
spec:
  provider:
    gcpsm:
      projectID: my-project-id
      auth:
        workloadIdentity:
          clusterLocation: us-central1
          clusterName: my-cluster
          serviceAccountRef:
            name: external-secrets-sa
            namespace: external-secrets-system
```

**2. Create ExternalSecret**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: api-keys
  namespace: notebook-validation
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcp-secret-manager
    kind: ClusterSecretStore
  target:
    name: api-keys
  data:
    - secretKey: openai-api-key
      remoteRef:
        key: openai-api-key
        version: latest
    - secretKey: huggingface-token
      remoteRef:
        key: huggingface-token
        version: latest
```

## Consequences

### Positive

1. **Centralized Management**: Secrets managed in external vault (single source of truth)
2. **Automatic Sync**: ESO automatically syncs secrets to Kubernetes
3. **Automatic Refresh**: Secrets refreshed on schedule (e.g., every hour)
4. **Audit Trail**: External vault provides audit trail
5. **Multi-Cloud**: Supports AWS, Azure, GCP, Vault, 1Password
6. **Declarative**: ExternalSecret resources are declarative (GitOps-friendly)
7. **Transparent**: NotebookValidationJob doesn't need to change - just references K8s Secret

### Negative

1. **Dependency**: Requires ESO installation (cluster admin)
2. **Complexity**: More complex than static secrets
3. **Latency**: Secrets synced on schedule (not real-time)
4. **Static Secrets**: Secrets still static (rotation happens upstream in vault)

### Neutral

1. **No Code Changes**: Operator doesn't need to change - ESO is transparent
2. **Optional**: Users can stay on Tier 1 (static secrets) if ESO not available

## Implementation

### Phase 1: Documentation
- [ ] Document ESO installation procedure
- [ ] Create examples for AWS Secrets Manager
- [ ] Create examples for Azure Key Vault
- [ ] Create examples for GCP Secret Manager
- [ ] Create troubleshooting guide

### Phase 2: Testing
- [ ] Test with AWS Secrets Manager
- [ ] Test with Azure Key Vault
- [ ] Test with GCP Secret Manager
- [ ] Verify automatic refresh
- [ ] Test secret rotation

### Phase 3: Publication
- [ ] Publish ESO integration guide
- [ ] Create example manifests
- [ ] Update README

## Troubleshooting

### Issue: ExternalSecret not syncing

**Symptoms**: K8s Secret not created, ExternalSecret shows error

**Diagnosis**:
```bash
kubectl describe externalsecret aws-credentials -n notebook-validation
kubectl logs -n external-secrets-system -l app.kubernetes.io/name=external-secrets
```

**Common Causes**:
- SecretStore not configured correctly
- IAM permissions missing (AWS IRSA, Azure Managed Identity, GCP Workload Identity)
- Secret doesn't exist in external vault
- Network connectivity issues

### Issue: Secrets not refreshing

**Symptoms**: Old credentials still in use after rotation

**Diagnosis**:
```bash
kubectl get externalsecret aws-credentials -n notebook-validation -o yaml
```

**Solution**: Check `refreshInterval` and `status.refreshTime`

## Alternatives Considered

### Alternative 1: Manual Secret Sync
**Rejected**: No automation, error-prone, no audit trail

### Alternative 2: Custom Secret Sync Operator
**Rejected**: Reinventing the wheel, ESO is mature and well-supported

### Alternative 3: Sealed Secrets
**Rejected**: Doesn't sync from external vaults, secrets still in Git

## Related ADRs

- **ADR-009**: Secret Management and Git Credentials (Git credentials)
- **ADR-014**: Notebook Credential Injection Strategy (overall strategy)
- **ADR-015**: Environment-Variable Pattern (Tier 1)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Tier 3)
- **ADR-018**: Secret Rotation & Lifecycle Management (rotation policies)

## References

- [External Secrets Operator](https://external-secrets.io/)
- [ESO AWS Provider](https://external-secrets.io/latest/provider/aws-secrets-manager/)
- [ESO Azure Provider](https://external-secrets.io/latest/provider/azure-key-vault/)
- [ESO GCP Provider](https://external-secrets.io/latest/provider/google-secrets-manager/)
- [AWS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
- [Azure Managed Identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity)
- [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

