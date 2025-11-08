# ADR 009: Secret Management and Git Credentials

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must securely handle Git credentials to fetch notebooks from private repositories. This is a critical security requirement that affects:

1. **User Experience**: Users need a simple, secure way to provide Git credentials
2. **Security Posture**: Credentials must never be logged or exposed
3. **Enterprise Requirements**: Integration with existing secret management systems
4. **Multi-Tenancy**: Isolation of secrets between namespaces/tenants
5. **Compliance**: Audit trails and secret rotation capabilities

### PRD Requirements

**Section 4 (AC-2)**: "The `spec.notebook.git` section accepts credentials via a secret"

**Section 7 (Open Questions)**: "A secure and flexible strategy for handling Git credentials and other secrets is required"

### User Stories

- **US-3**: "Fetch notebooks from private or public Git repositories"
- **Implicit**: Support multiple authentication methods (SSH, HTTPS, tokens)
- **Implicit**: Enable secret rotation without operator restart
- **Implicit**: Integrate with enterprise secret stores (Vault, AWS Secrets Manager)

### Technical Challenges

1. **Multiple Auth Methods**: SSH keys, HTTPS tokens, deploy keys, OAuth apps
2. **Secret Rotation**: Credentials may change without operator knowledge
3. **Multi-Cluster**: Secrets don't replicate across clusters
4. **GitOps Compatibility**: Secrets can't be stored in Git plaintext
5. **Namespace Isolation**: Secrets must be scoped appropriately
6. **Audit Requirements**: Track secret access and usage

### Industry Patterns (2024)

Based on research, the Kubernetes ecosystem has converged on several patterns:

1. **Native Kubernetes Secrets**: Simple, built-in, but limited
2. **External Secrets Operator (ESO)**: Sync from external stores (Vault, AWS, GCP, Azure)
3. **Sealed Secrets**: Encrypt secrets for GitOps workflows
4. **SOPS + Age**: Encrypt secrets in Git with age/PGP keys
5. **Secrets Store CSI Driver**: Mount secrets from external stores as volumes
6. **HashiCorp Vault**: Enterprise-grade secret management with dynamic secrets

### Available Options

#### Option 1: Native Kubernetes Secrets Only
**Approach**: Use only `kubectl create secret` and reference in CRD

**Pros**:
- Simple, no additional dependencies
- Works out-of-the-box on any Kubernetes cluster
- Well-understood by users
- No additional RBAC complexity

**Cons**:
- Secrets stored base64-encoded (not encrypted at rest by default)
- No secret rotation without manual intervention
- Can't be stored in Git (GitOps unfriendly)
- No integration with enterprise secret stores
- Limited audit capabilities

**Example**:
```yaml
# Create secret manually
kubectl create secret generic git-credentials \
  --from-literal=username=myuser \
  --from-literal=password=mytoken

# Reference in CRD
spec:
  notebook:
    git:
      url: https://github.com/org/repo.git
      credentialsSecret: git-credentials
```

#### Option 2: External Secrets Operator (ESO)
**Approach**: Sync secrets from external stores into Kubernetes Secrets

**Pros**:
- Integrates with 20+ secret backends (Vault, AWS, GCP, Azure, 1Password)
- Automatic secret rotation
- Centralized secret management
- Audit trails from external store
- Secrets never stored in Git
- Dynamic secret generation (Vault)

**Cons**:
- Requires ESO installation (additional dependency)
- Complexity for simple use cases
- Requires external secret store
- Additional RBAC for ESO

**Example**:
```yaml
# ExternalSecret syncs from Vault to K8s Secret
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: git-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: git-credentials
    creationPolicy: Owner
  data:
    - secretKey: username
      remoteRef:
        key: secret/data/git/credentials
        property: username
    - secretKey: password
      remoteRef:
        key: secret/data/git/credentials
        property: password

# Reference in CRD (same as Option 1)
spec:
  notebook:
    git:
      credentialsSecret: git-credentials
```

#### Option 3: Sealed Secrets
**Approach**: Encrypt secrets for storage in Git, decrypt in-cluster

**Pros**:
- GitOps-friendly (encrypted secrets in Git)
- No external dependencies
- Cluster-scoped encryption keys
- Simple workflow for developers

**Cons**:
- Secrets tied to specific cluster (can't move between clusters easily)
- Key management complexity
- No integration with enterprise secret stores
- Manual rotation process

**Example**:
```yaml
# Sealed secret (encrypted, safe for Git)
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: git-credentials
spec:
  encryptedData:
    username: AgBh8... (encrypted)
    password: AgCx9... (encrypted)

# Decrypted to regular Secret in-cluster
# Reference in CRD (same as Option 1)
```

#### Option 4: Hybrid Approach (Recommended)
**Approach**: Support native Secrets + optional ESO integration

**Pros**:
- Flexibility: Simple for basic use, powerful for enterprise
- No forced dependencies
- Gradual adoption path
- Best of both worlds

**Cons**:
- More documentation needed
- Testing complexity (multiple paths)
- Support burden for multiple patterns

## Decision

We will implement a **Hybrid Secret Management Strategy** with three tiers:

### Tier 1: Native Kubernetes Secrets (Required)
**Target**: All users, simple use cases

The operator will **always** support native Kubernetes Secrets as the baseline:

```yaml
apiVersion: mlops.dev/v1alpha1
kind: NotebookValidationJob
spec:
  notebook:
    git:
      url: https://github.com/org/repo.git
      credentialsSecret: my-git-credentials  # References K8s Secret
```

**Secret Format**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-git-credentials
type: Opaque
stringData:
  # HTTPS authentication
  username: myuser
  password: ghp_mytoken123
  
  # OR SSH authentication
  ssh-privatekey: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
    -----END OPENSSH PRIVATE KEY-----
  ssh-knownhosts: |
    github.com ssh-rsa AAAAB3NzaC1yc2E...
```

**Supported Auth Methods**:
1. **HTTPS with username/password**: `username` + `password` keys
2. **HTTPS with token**: `password` key only (username optional)
3. **SSH with private key**: `ssh-privatekey` + optional `ssh-knownhosts`

### Tier 2: External Secrets Operator (Optional)
**Target**: Enterprise users with existing secret stores

The operator will **detect and support** ESO if installed:

```yaml
# User creates ExternalSecret (ESO syncs to K8s Secret)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: git-credentials
spec:
  secretStoreRef:
    name: vault-backend
  target:
    name: git-credentials  # Creates this K8s Secret
  data:
    - secretKey: username
      remoteRef:
        key: git/credentials
        property: username

# Operator uses the synced Secret (transparent)
spec:
  notebook:
    git:
      credentialsSecret: git-credentials
```

**Benefits**:
- Automatic secret rotation (ESO handles sync)
- Centralized secret management
- Audit trails from external store
- No operator code changes needed (uses K8s Secrets)

### Tier 3: Sealed Secrets (Optional)
**Target**: GitOps users

The operator will **transparently support** Sealed Secrets:

```yaml
# User commits SealedSecret to Git
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: git-credentials
spec:
  encryptedData:
    username: AgBh8...
    password: AgCx9...

# Sealed Secrets controller decrypts to K8s Secret
# Operator uses the decrypted Secret (transparent)
```

**Benefits**:
- GitOps-friendly (encrypted secrets in Git)
- No operator changes needed
- Works with existing GitOps workflows

## Implementation Details

### Secret Resolution Logic

```go
// pkg/git/credentials.go
package git

import (
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type Credentials struct {
    Username      string
    Password      string
    SSHPrivateKey string
    SSHKnownHosts string
}

// ResolveCredentials fetches and parses Git credentials from a Secret
func ResolveCredentials(ctx context.Context, c client.Client, namespace, secretName string) (*Credentials, error) {
    secret := &corev1.Secret{}
    if err := c.Get(ctx, client.ObjectKey{
        Namespace: namespace,
        Name:      secretName,
    }, secret); err != nil {
        return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
    }

    creds := &Credentials{
        Username:      string(secret.Data["username"]),
        Password:      string(secret.Data["password"]),
        SSHPrivateKey: string(secret.Data["ssh-privatekey"]),
        SSHKnownHosts: string(secret.Data["ssh-knownhosts"]),
    }

    // Validate credentials
    if err := creds.Validate(); err != nil {
        return nil, fmt.Errorf("invalid credentials in secret %s/%s: %w", namespace, secretName, err)
    }

    return creds, nil
}

func (c *Credentials) Validate() error {
    hasHTTPS := c.Password != ""
    hasSSH := c.SSHPrivateKey != ""

    if !hasHTTPS && !hasSSH {
        return fmt.Errorf("secret must contain either 'password' (HTTPS) or 'ssh-privatekey' (SSH)")
    }

    if hasHTTPS && hasSSH {
        return fmt.Errorf("secret cannot contain both HTTPS and SSH credentials")
    }

    return nil
}

// IsHTTPS returns true if credentials are for HTTPS authentication
func (c *Credentials) IsHTTPS() bool {
    return c.Password != ""
}

// IsSSH returns true if credentials are for SSH authentication
func (c *Credentials) IsSSH() bool {
    return c.SSHPrivateKey != ""
}
```

### Git Clone with Credentials

```go
// pkg/git/clone.go
package git

import (
    "context"
    "fmt"
    "os"
    "os/exec"
)

// CloneRepository clones a Git repository with credentials
func CloneRepository(ctx context.Context, url, ref, targetDir string, creds *Credentials) error {
    if creds.IsHTTPS() {
        return cloneHTTPS(ctx, url, ref, targetDir, creds)
    }
    return cloneSSH(ctx, url, ref, targetDir, creds)
}

func cloneHTTPS(ctx context.Context, url, ref, targetDir string, creds *Credentials) error {
    // Inject credentials into URL
    authenticatedURL := injectCredentials(url, creds.Username, creds.Password)
    
    cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", ref, authenticatedURL, targetDir)
    
    // Ensure credentials don't leak in logs
    cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        // Sanitize error message (remove credentials)
        return fmt.Errorf("git clone failed: %w", sanitizeError(err, creds.Password))
    }
    
    return nil
}

func cloneSSH(ctx context.Context, url, ref, targetDir string, creds *Credentials) error {
    // Write SSH key to temporary file
    keyFile, err := writeSSHKey(creds.SSHPrivateKey)
    if err != nil {
        return err
    }
    defer os.Remove(keyFile)

    // Configure SSH to use the key
    cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", ref, url, targetDir)
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyFile),
    )

    if creds.SSHKnownHosts != "" {
        // Write known_hosts file
        knownHostsFile, err := writeKnownHosts(creds.SSHKnownHosts)
        if err != nil {
            return err
        }
        defer os.Remove(knownHostsFile)
        
        cmd.Env = append(cmd.Env,
            fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o UserKnownHostsFile=%s", keyFile, knownHostsFile),
        )
    }

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git clone failed: %w (output: %s)", err, output)
    }

    return nil
}
```

### Security Best Practices

1. **Never Log Credentials**
```go
// BAD: Credentials in logs
log.Info("Cloning repository", "url", authenticatedURL)

// GOOD: Sanitized logging
log.Info("Cloning repository", "url", sanitizeURL(url))
```

2. **Temporary Files for SSH Keys**
```go
func writeSSHKey(key string) (string, error) {
    tmpFile, err := os.CreateTemp("", "git-ssh-key-*")
    if err != nil {
        return "", err
    }
    
    // Set restrictive permissions (SSH requires this)
    if err := tmpFile.Chmod(0600); err != nil {
        return "", err
    }
    
    if _, err := tmpFile.WriteString(key); err != nil {
        return "", err
    }
    
    return tmpFile.Name(), tmpFile.Close()
}
```

3. **Secret Access RBAC**
```yaml
# Operator needs read access to Secrets
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jupyter-notebook-validator-operator
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
```

## Consequences

### Positive
- **Flexibility**: Supports simple and enterprise use cases
- **No Forced Dependencies**: Works with native Secrets out-of-the-box
- **Enterprise Ready**: Integrates with existing secret stores via ESO
- **GitOps Compatible**: Works with Sealed Secrets and SOPS
- **Security**: Credentials never logged or exposed
- **Multi-Auth**: Supports HTTPS and SSH authentication

### Negative
- **Documentation Burden**: Must document multiple secret patterns
- **Testing Complexity**: Must test native Secrets, ESO, and Sealed Secrets
- **Support Complexity**: Users may be confused by multiple options

### Neutral
- **Gradual Adoption**: Users can start simple, add ESO later
- **Operator Simplicity**: Operator only reads K8s Secrets (transparent to ESO/Sealed Secrets)

## Migration and Compatibility

### Secret Rotation

**Native Secrets**: Manual rotation
```bash
# Update secret
kubectl create secret generic git-credentials \
  --from-literal=password=new-token \
  --dry-run=client -o yaml | kubectl apply -f -

# Operator picks up new secret on next reconciliation
```

**ESO**: Automatic rotation
```yaml
# ESO syncs every refreshInterval
spec:
  refreshInterval: 1h  # Sync from Vault every hour
```

### Multi-Cluster Secrets

For multi-cluster deployments, use ESO with centralized secret store:
```yaml
# Cluster A and Cluster B both sync from same Vault
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
spec:
  provider:
    vault:
      server: https://vault.example.com
      path: secret
      auth:
        kubernetes:
          mountPath: kubernetes
          role: jupyter-validator
```

## Notebook Credential Injection

**Added:** 2025-11-08

In addition to Git credentials, notebooks often need credentials to access external services during validation (AWS S3, databases, ML services). This section extends the secret management strategy to cover notebook runtime credentials.

### Credential Types

Notebooks may require credentials for:

1. **Cloud Storage**: AWS S3, Azure Blob, GCP Cloud Storage
2. **Databases**: PostgreSQL, MySQL, MongoDB, Redis
3. **ML Services**: MLflow, Weights & Biases, Hugging Face
4. **APIs**: OpenAI, Anthropic, custom REST APIs

### Implementation Approach

The operator supports two methods for injecting credentials into validation pods:

#### Method 1: Individual Environment Variables (`spec.podConfig.env`)

**Use Case**: Fine-grained control, mixing secrets and plain values

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

#### Method 2: Bulk Injection (`spec.podConfig.envFrom`)

**Use Case**: Injecting all keys from a Secret or ConfigMap

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
    envFrom:
      - secretRef:
          name: aws-credentials
      - secretRef:
          name: database-config
      - configMapRef:
          name: app-config
```

### Security Considerations

1. **Separation of Concerns**: Git credentials and notebook credentials are separate
   - Git credentials: Used by init container to clone repository
   - Notebook credentials: Used by validation container to access services

2. **Least Privilege**: Grant only necessary secret access
   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: notebook-validation-secrets
   rules:
     - apiGroups: [""]
       resources: ["secrets"]
       resourceNames: ["aws-credentials", "database-config"]
       verbs: ["get"]
   ```

3. **Log Sanitization**: All credentials are sanitized in logs (see ADR-010)
   - URLs with credentials are sanitized
   - Secret values are never logged
   - Environment variables with sensitive names are masked

4. **Secret Rotation**: Same patterns as Git credentials
   - Native Secrets: Manual rotation via `kubectl apply`
   - ESO: Automatic rotation via `refreshInterval`
   - Vault: Dynamic secrets with TTL-based rotation

### Integration with External Secret Stores

Notebook credentials can use the same ESO/Vault patterns as Git credentials:

**ESO Example**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: aws-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: aws-credentials
  data:
    - secretKey: AWS_ACCESS_KEY_ID
      remoteRef:
        key: mlops/aws-credentials
        property: access-key-id
    - secretKey: AWS_SECRET_ACCESS_KEY
      remoteRef:
        key: mlops/aws-credentials
        property: secret-access-key
```

**Vault Dynamic Secrets Example**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-config
spec:
  refreshInterval: 15m
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: database-config
  data:
    - secretKey: DB_CONNECTION_STRING
      remoteRef:
        key: database/creds/mlops-readonly
        property: connection_string
```

### Documentation References

For detailed guidance on notebook credential injection:
- **ADR-014**: Notebook Credential Injection Strategy (multi-tier approach)
- **ADR-015**: Environment-Variable Pattern for Notebook Credentials (naming conventions)
- **ADR-016**: External Secrets Operator Integration (ESO setup and examples)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Vault Agent sidecar)
- **ADR-018**: Secret Rotation & Lifecycle Management (rotation policies)
- **ADR-019**: RBAC & Pod Security Policies for Notebook Secret Access (security)
- **docs/NOTEBOOK_CREDENTIALS_GUIDE.md**: Comprehensive implementation guide

## References

- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
- [External Secrets Operator](https://external-secrets.io/)
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- [SOPS](https://github.com/getsops/sops)
- [HashiCorp Vault](https://www.vaultproject.io/)
- [Git Credential Storage](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage)

## Related ADRs

- ADR 005: RBAC and Service Account Model (defines secret access permissions)
- ADR 010: Observability and Monitoring (secret access audit logging)
- ADR 014: Notebook Credential Injection Strategy (multi-tier credential approach)
- ADR 015: Environment-Variable Pattern for Notebook Credentials (naming conventions)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial secret management strategy |
| 2025-11-08 | Team   | Added notebook credential injection section |

