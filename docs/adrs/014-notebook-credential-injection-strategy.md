# ADR 014: Notebook Credential Injection Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator executes notebooks that often need to access external services during validation:

1. **AWS S3**: Loading training data, saving model artifacts
2. **Databases**: Querying feature stores, validating data pipelines
3. **APIs**: Calling ML services (OpenAI, Hugging Face, MLflow)
4. **Cloud Services**: Accessing cloud-native ML platforms

Without a clear, unified credential injection strategy, teams will adopt ad hoc methods, leading to:
- **Security gaps**: Hardcoded credentials, exposed secrets
- **Inconsistency**: Different patterns across teams
- **Operational overhead**: Manual secret management
- **Compliance issues**: No audit trail, no rotation

### Current Architecture

The operator **already supports** credential injection through `spec.podConfig.env`:

```yaml
spec:
  podConfig:
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
```

However, there is **no documentation** of:
- Recommended patterns for different use-cases
- Integration with enterprise secret management (ESO, Vault)
- Security best practices
- Secret rotation procedures

### User Stories

- **US-10**: "As a data scientist, I need my notebook to load training data from S3 during validation"
- **US-11**: "As an ML engineer, I need my notebook to query a database for feature engineering"
- **US-12**: "As a platform engineer, I need to integrate with our enterprise secret store (Vault)"
- **US-13**: "As a security engineer, I need to enforce least-privilege access to secrets"

### Industry Patterns (2024)

Based on research, the Kubernetes ecosystem has converged on several patterns:

1. **Environment Variables**: Simple, works everywhere, but static
2. **Kubernetes Secrets**: Built-in, but no rotation or external sync
3. **External Secrets Operator (ESO)**: Sync from external stores (AWS, Azure, GCP, Vault)
4. **Vault Agent Sidecar**: Dynamic secrets with automatic rotation
5. **Workload Identity**: Cloud-native IAM (IRSA, Workload Identity, Managed Identity)

## Decision

We adopt a **multi-tier credential injection strategy** that provides a clear adoption path from simple to advanced:

### Tier 1: Static Secrets in Environment Variables
**Target**: POCs, development, simple use-cases

**Pattern**: Use Kubernetes Secrets with `spec.podConfig.env`

**Example**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
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
            name: aws-credentials
            key: access-key-id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: secret-access-key
      - name: AWS_REGION
        value: us-east-1
```

**Pros**:
- Simple to adopt
- Works out-of-the-box
- No additional dependencies
- Well-understood by users

**Cons**:
- No automatic rotation
- Secrets are static
- Manual management required

**When to Use**: Development, POCs, simple validation workflows

### Tier 2: External Secrets Operator (ESO)
**Target**: Enterprise users with existing secret stores

**Pattern**: Use ESO to sync secrets from external vaults into Kubernetes Secrets

**Example**:
```yaml
# 1. Create ExternalSecret (ESO syncs to K8s Secret)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: aws-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: aws-credentials
  data:
    - secretKey: access-key-id
      remoteRef:
        key: prod/notebook/aws_access_key_id
    - secretKey: secret-access-key
      remoteRef:
        key: prod/notebook/aws_secret_access_key

# 2. Reference in NotebookValidationJob (same as Tier 1)
spec:
  podConfig:
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
```

**Pros**:
- Centralized secret management
- Automatic sync from external stores
- Supports AWS, Azure, GCP, Vault, 1Password
- Audit trail from external store
- Automatic refresh (configurable interval)

**Cons**:
- Requires ESO installation
- Secrets still static (rotation happens upstream)
- Additional complexity

**When to Use**: Production, enterprise environments, centralized secret management

### Tier 3: Vault Dynamic Secrets
**Target**: High-security environments, production workloads

**Pattern**: Use Vault Agent sidecar to inject dynamic, short-lived credentials

**Example**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: database-pipeline.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: vault-notebook-sa  # Annotated with Vault role
    # Vault Agent sidecar injects credentials automatically
    # No explicit env vars needed - credentials appear in shared volume
```

**Vault ServiceAccount**:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-notebook-sa
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "notebook-validator"
    vault.hashicorp.com/agent-inject-secret-db: "database/creds/readonly"
    vault.hashicorp.com/agent-inject-template-db: |
      {{- with secret "database/creds/readonly" -}}
      export DB_USERNAME="{{ .Data.username }}"
      export DB_PASSWORD="{{ .Data.password }}"
      {{- end }}
```

**Pros**:
- Dynamic, short-lived credentials
- Automatic rotation
- Reduced blast radius (credentials expire quickly)
- Best security posture
- Audit trail in Vault

**Cons**:
- Most complex setup
- Requires Vault infrastructure
- Requires Vault Agent sidecar
- More complex Pod spec

**When to Use**: Production, high-security environments, compliance requirements

## Consequences

### Positive

1. **Clear Adoption Path**: Users can start simple (Tier 1) and graduate to advanced (Tier 2/3)
2. **Consistency**: Standardized patterns across teams
3. **Security**: Best practices built-in, least-privilege access
4. **Flexibility**: Supports simple and enterprise use-cases
5. **Transparency**: Operator doesn't need to change - patterns are user-facing

### Negative

1. **Documentation Overhead**: Need comprehensive guides for each tier
2. **Learning Curve**: Users need to understand 3 different patterns
3. **Complexity**: Tier 3 (Vault) is complex to set up

### Neutral

1. **No Code Changes**: Operator already supports all patterns via `spec.podConfig.env`
2. **Optional Features**: ESO and Vault are optional - users can stay on Tier 1

## Implementation

### Phase 1: Documentation (Week 4)
- [ ] Create comprehensive credential guide (`docs/NOTEBOOK_CREDENTIALS_GUIDE.md`)
- [ ] Document Tier 1 pattern with examples
- [ ] Document Tier 2 pattern with ESO examples
- [ ] Document Tier 3 pattern with Vault examples
- [ ] Create security best practices guide

### Phase 2: Examples (Week 4)
- [ ] Create example notebooks (S3, database, API)
- [ ] Create sample CRD manifests for each tier
- [ ] Create secret templates
- [ ] Create ESO configuration examples
- [ ] Create Vault configuration examples

### Phase 3: Testing (Week 5)
- [ ] Test Tier 1 with static secrets
- [ ] Test Tier 2 with ESO (AWS, Azure, GCP)
- [ ] Test Tier 3 with Vault Agent sidecar
- [ ] Verify security best practices

### Phase 4: Publication (Week 5)
- [ ] Publish documentation
- [ ] Update README with credential patterns
- [ ] Create user training materials
- [ ] Host brown-bag session

## Alternatives Considered

### Alternative 1: Only Vault Dynamic Secrets
**Rejected**: Too heavy for simple use-cases, high barrier to entry

### Alternative 2: Only Environment Variables
**Rejected**: No secret rotation, no enterprise integration, security gaps

### Alternative 3: Custom Secret Injection Mechanism
**Rejected**: Reinventing the wheel, Kubernetes already provides `env` and `envFrom`

## Related ADRs

- **ADR-009**: Secret Management and Git Credentials (Git clone credentials)
- **ADR-015**: Environment-Variable Pattern for Notebook Credentials (Tier 1 details)
- **ADR-016**: External Secret Operator Integration (Tier 2 details)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Tier 3 details)
- **ADR-018**: Secret Rotation & Lifecycle Management (rotation policies)
- **ADR-019**: RBAC & Pod Security Policies (access control)

## References

- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [External Secrets Operator](https://external-secrets.io/)
- [HashiCorp Vault](https://www.vaultproject.io/)
- [Vault Agent Sidecar](https://www.vaultproject.io/docs/platform/k8s/injector)
- [AWS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
- [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [Azure Managed Identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

