# ADR 017: Vault Dynamic-Secrets Injection Pattern

## Status
Accepted

## Context

ADR-014 established a multi-tier credential injection strategy. This ADR focuses on **Tier 3: Vault Dynamic Secrets**, which provides the highest security posture through short-lived, automatically rotated credentials.

### Problem

Static secrets (Tier 1 and Tier 2) have inherent security risks:
- **Long-lived credentials**: Increased blast radius if compromised
- **Manual rotation**: Requires coordination and downtime
- **Credential sprawl**: Secrets copied to multiple locations
- **No automatic revocation**: Compromised credentials remain valid until manually revoked

### HashiCorp Vault Dynamic Secrets

Vault can generate **dynamic, short-lived credentials** for:
- **Databases**: PostgreSQL, MySQL, MongoDB (credentials expire after TTL)
- **AWS**: IAM credentials with specific permissions (STS AssumeRole)
- **Cloud APIs**: Azure, GCP credentials
- **SSH**: Dynamic SSH keys

**Key Benefits**:
- **Short-lived**: Credentials expire automatically (e.g., 1 hour TTL)
- **Just-in-time**: Credentials generated when needed
- **Automatic rotation**: No manual intervention required
- **Reduced blast radius**: Compromised credentials expire quickly
- **Audit trail**: Vault logs all credential generation

### Vault Agent Sidecar Pattern

The **Vault Agent Sidecar** pattern is the recommended way to inject Vault secrets into Kubernetes pods:

1. **Sidecar container** runs Vault Agent
2. **Vault Agent** authenticates to Vault using Kubernetes ServiceAccount
3. **Vault Agent** fetches secrets and renders them into a shared volume
4. **Application container** reads secrets from the shared volume

## Decision

We adopt the **Vault Agent Sidecar** pattern for dynamic secret injection (Tier 3).

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    HashiCorp Vault                           │
│  - Database secrets engine (dynamic DB credentials)          │
│  - AWS secrets engine (dynamic IAM credentials)              │
│  - Kubernetes auth method (ServiceAccount → Vault role)      │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ 1. Authenticate with ServiceAccount JWT
                              │ 2. Fetch secrets
                              │ 3. Renew lease before expiry
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Vault Agent Sidecar Container                   │
│  - Authenticates to Vault                                    │
│  - Fetches dynamic secrets                                   │
│  - Renders secrets to shared volume                          │
│  - Automatically renews leases                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Shared volume (/vault/secrets)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Notebook Validation Container                   │
│  - Reads credentials from /vault/secrets/db                  │
│  - Credentials are always fresh (auto-renewed)               │
└─────────────────────────────────────────────────────────────┘
```

### Pattern: Dynamic Database Credentials

**1. Configure Vault** (Vault admin):

```bash
# Enable database secrets engine
vault secrets enable database

# Configure PostgreSQL connection
vault write database/config/my-postgresql-database \
    plugin_name=postgresql-database-plugin \
    allowed_roles="readonly" \
    connection_url="postgresql://{{username}}:{{password}}@postgres:5432/mydb" \
    username="vault" \
    password="vault-password"

# Create role with dynamic credentials (1 hour TTL)
vault write database/roles/readonly \
    db_name=my-postgresql-database \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    default_ttl="1h" \
    max_ttl="24h"

# Enable Kubernetes auth
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc:443"

# Create policy
vault policy write notebook-validator - <<EOF
path "database/creds/readonly" {
  capabilities = ["read"]
}
EOF

# Create Kubernetes role
vault write auth/kubernetes/role/notebook-validator \
    bound_service_account_names=vault-notebook-sa \
    bound_service_account_namespaces=notebook-validation \
    policies=notebook-validator \
    ttl=1h
```

**2. Create ServiceAccount with Vault annotations** (user):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-notebook-sa
  namespace: notebook-validation
  annotations:
    # Vault Agent Injector annotations
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "notebook-validator"
    vault.hashicorp.com/agent-inject-secret-db: "database/creds/readonly"
    vault.hashicorp.com/agent-inject-template-db: |
      {{- with secret "database/creds/readonly" -}}
      export DB_USERNAME="{{ .Data.username }}"
      export DB_PASSWORD="{{ .Data.password }}"
      export DB_HOST="postgres.example.com"
      export DB_PORT="5432"
      export DB_NAME="mydb"
      {{- end }}
```

**3. Create NotebookValidationJob** (user):

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: database-pipeline-validation
  namespace: notebook-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: database-pipeline.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: vault-notebook-sa  # Uses Vault Agent sidecar
    # No explicit env vars needed - credentials in /vault/secrets/db
```

**4. Notebook code** (reads from Vault-injected file):

```python
# Cell 1: Import libraries
import psycopg2
import os

# Cell 2: Source credentials from Vault-injected file
with open('/vault/secrets/db', 'r') as f:
    exec(f.read())  # Sets DB_USERNAME, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME

# Cell 3: Connect to database (credentials are fresh, auto-renewed)
conn = psycopg2.connect(
    host=os.environ['DB_HOST'],
    port=os.environ['DB_PORT'],
    database=os.environ['DB_NAME'],
    user=os.environ['DB_USERNAME'],
    password=os.environ['DB_PASSWORD']
)

# Cell 4: Query data
cursor = conn.cursor()
cursor.execute("SELECT * FROM features LIMIT 10")
rows = cursor.fetchall()
```

### Pattern: Dynamic AWS Credentials

**1. Configure Vault** (Vault admin):

```bash
# Enable AWS secrets engine
vault secrets enable aws

# Configure AWS credentials
vault write aws/config/root \
    access_key=AKIAIOSFODNN7EXAMPLE \
    secret_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
    region=us-east-1

# Create role with dynamic STS credentials (1 hour TTL)
vault write aws/roles/s3-readonly \
    credential_type=assumed_role \
    role_arns=arn:aws:iam::123456789012:role/S3ReadOnlyRole \
    default_ttl=1h \
    max_ttl=24h
```

**2. Create ServiceAccount**:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-notebook-sa
  namespace: notebook-validation
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "notebook-validator"
    vault.hashicorp.com/agent-inject-secret-aws: "aws/creds/s3-readonly"
    vault.hashicorp.com/agent-inject-template-aws: |
      {{- with secret "aws/creds/s3-readonly" -}}
      export AWS_ACCESS_KEY_ID="{{ .Data.access_key }}"
      export AWS_SECRET_ACCESS_KEY="{{ .Data.secret_key }}"
      export AWS_SESSION_TOKEN="{{ .Data.security_token }}"
      export AWS_REGION="us-east-1"
      {{- end }}
```

### How Vault Agent Sidecar Works

**Pod Lifecycle**:

1. **Pod starts**: Vault Agent sidecar starts first
2. **Authentication**: Vault Agent authenticates to Vault using ServiceAccount JWT
3. **Secret fetch**: Vault Agent fetches dynamic secrets from Vault
4. **Template rendering**: Vault Agent renders secrets using template
5. **File write**: Vault Agent writes secrets to `/vault/secrets/` (shared volume)
6. **Application starts**: Notebook container starts and reads secrets
7. **Lease renewal**: Vault Agent automatically renews leases before expiry
8. **Pod terminates**: Vault Agent revokes leases

**Automatic Renewal**:
- Vault Agent monitors lease TTL
- Renews leases before expiry (e.g., at 80% of TTL)
- If renewal fails, fetches new credentials
- Notebook always has valid credentials

## Consequences

### Positive

1. **Short-lived credentials**: Reduced blast radius (credentials expire in 1 hour)
2. **Automatic rotation**: No manual intervention required
3. **Just-in-time**: Credentials generated when needed
4. **Automatic revocation**: Credentials revoked when pod terminates
5. **Audit trail**: Vault logs all credential generation and usage
6. **Best security posture**: Industry best practice for production workloads

### Negative

1. **Complexity**: Most complex setup (requires Vault infrastructure)
2. **Vault dependency**: Requires Vault installation and configuration
3. **Sidecar overhead**: Additional container in pod (small resource overhead)
4. **Learning curve**: Users need to understand Vault concepts

### Neutral

1. **No operator changes**: Operator doesn't need to change - pattern is user-facing
2. **Optional**: Users can stay on Tier 1 or Tier 2 if Vault not available

## Implementation

### Phase 1: Documentation
- [ ] Document Vault Agent sidecar pattern
- [ ] Create examples for database credentials
- [ ] Create examples for AWS credentials
- [ ] Create troubleshooting guide

### Phase 2: Testing
- [ ] Test with PostgreSQL dynamic credentials
- [ ] Test with MySQL dynamic credentials
- [ ] Test with AWS STS credentials
- [ ] Verify automatic renewal
- [ ] Verify automatic revocation

### Phase 3: Publication
- [ ] Publish Vault integration guide
- [ ] Create example manifests
- [ ] Update README

## Troubleshooting

### Issue: Vault Agent sidecar not injecting secrets

**Symptoms**: No `/vault/secrets/` directory in pod

**Diagnosis**:
```bash
kubectl describe pod <pod-name> -n notebook-validation
kubectl logs <pod-name> -c vault-agent -n notebook-validation
```

**Common Causes**:
- ServiceAccount not annotated correctly
- Vault Injector not installed
- Vault role not configured
- Kubernetes auth not configured

### Issue: Credentials expired

**Symptoms**: Database connection fails with authentication error

**Diagnosis**: Check Vault Agent logs for renewal failures

**Solution**: Increase TTL or check Vault Agent renewal logic

## Alternatives Considered

### Alternative 1: Vault CSI Provider
**Considered**: Vault CSI Provider mounts secrets as volumes
**Status**: Valid alternative, but Vault Agent Sidecar is more flexible

### Alternative 2: Direct Vault API Calls
**Rejected**: Requires Vault client in notebook, more complex

### Alternative 3: ESO with Vault
**Considered**: ESO can sync from Vault, but secrets are static (not dynamic)

## Related ADRs

- **ADR-014**: Notebook Credential Injection Strategy (overall strategy)
- **ADR-015**: Environment-Variable Pattern (Tier 1)
- **ADR-016**: External Secret Operator Integration (Tier 2)
- **ADR-018**: Secret Rotation & Lifecycle Management (rotation policies)
- **ADR-019**: RBAC & Pod Security Policies (access control)

## References

- [HashiCorp Vault](https://www.vaultproject.io/)
- [Vault Agent Sidecar Injector](https://www.vaultproject.io/docs/platform/k8s/injector)
- [Vault Database Secrets Engine](https://www.vaultproject.io/docs/secrets/databases)
- [Vault AWS Secrets Engine](https://www.vaultproject.io/docs/secrets/aws)
- [Vault Kubernetes Auth](https://www.vaultproject.io/docs/auth/kubernetes)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

