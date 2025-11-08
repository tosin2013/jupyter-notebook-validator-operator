# ADR 019: RBAC & Pod Security Policies for Notebook Secret Access

## Status
Accepted

## Context

ADR-014 through ADR-018 established credential injection patterns and rotation policies. This ADR addresses **access control and security hardening** for notebook secret access.

### Problem

Notebooks execute arbitrary user code, which creates security risks:
- **Unauthorized secret access**: Notebooks could access secrets they shouldn't
- **Privilege escalation**: Compromised notebooks could escalate privileges
- **Lateral movement**: Compromised notebooks could access other namespaces
- **No audit trail**: No visibility into which notebooks accessed which secrets

### Security Principles

1. **Least Privilege**: Grant minimum permissions required
2. **Defense in Depth**: Multiple layers of security controls
3. **Separation of Concerns**: Isolate workloads by namespace
4. **Audit Trail**: Log all secret access
5. **Pod Security Standards**: Enforce baseline security policies

## Decision

We implement RBAC policies and Pod Security Standards to enforce least-privilege access to secrets.

### RBAC Strategy

**Principle**: Each notebook namespace has its own ServiceAccount with access only to secrets in that namespace.

**Architecture**:
```
┌─────────────────────────────────────────────────────────────┐
│                    Namespace: notebook-validation            │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │  ServiceAccount: notebook-validator-sa             │    │
│  │  - Can read secrets in this namespace only         │    │
│  │  - Cannot read secrets in other namespaces         │    │
│  │  - Cannot create/update/delete secrets             │    │
│  └────────────────────────────────────────────────────┘    │
│                          │                                   │
│                          │ RBAC Role binding                 │
│                          ▼                                   │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Role: notebook-secret-reader                      │    │
│  │  - get, list secrets                               │    │
│  │  - Only in this namespace                          │    │
│  └────────────────────────────────────────────────────┘    │
│                          │                                   │
│                          │ Mounts secrets                    │
│                          ▼                                   │
│  ┌────────────────────────────────────────────────────┐    │
│  │  NotebookValidationJob Pod                         │    │
│  │  - Uses notebook-validator-sa                      │    │
│  │  - Can only access secrets in this namespace       │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### RBAC Resources

**1. ServiceAccount** (per namespace):
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: notebook-validator-sa
  namespace: notebook-validation
automountServiceAccountToken: true
```

**2. Role** (namespace-scoped):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: notebook-secret-reader
  namespace: notebook-validation
rules:
  # Allow reading secrets (for env var injection)
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  # Allow reading configmaps (for configuration)
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
```

**3. RoleBinding** (binds ServiceAccount to Role):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: notebook-validator-secret-reader
  namespace: notebook-validation
subjects:
  - kind: ServiceAccount
    name: notebook-validator-sa
    namespace: notebook-validation
roleRef:
  kind: Role
  name: notebook-secret-reader
  apiGroup: rbac.authorization.k8s.io
```

**4. NotebookValidationJob** (uses ServiceAccount):
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
    serviceAccountName: notebook-validator-sa  # Uses least-privilege SA
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
```

### Pod Security Standards

**Principle**: Enforce baseline security policies for all notebook pods.

**Policy**: Use **Restricted** Pod Security Standard (most secure)

**Namespace Label**:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: notebook-validation
  labels:
    # Enforce Restricted Pod Security Standard
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

**Restricted Policy Requirements**:
- ✅ Run as non-root user
- ✅ Drop all capabilities
- ✅ No privilege escalation
- ✅ Read-only root filesystem (where possible)
- ✅ Seccomp profile (RuntimeDefault)
- ✅ No host namespaces (hostNetwork, hostPID, hostIPC)

**Pod Security Context** (enforced by operator):
```yaml
spec:
  podConfig:
    securityContext:
      runAsNonRoot: true
      runAsUser: 1000
      fsGroup: 1000
      seccompProfile:
        type: RuntimeDefault
    containers:
      - name: notebook-validator
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
              - ALL
          readOnlyRootFilesystem: false  # Notebooks need to write to /tmp
```

### Network Policies (Optional)

**Principle**: Restrict network access to only required services.

**Example**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: notebook-validator-netpol
  namespace: notebook-validation
spec:
  podSelector:
    matchLabels:
      app: notebook-validator
  policyTypes:
    - Ingress
    - Egress
  ingress: []  # No ingress (notebooks don't receive traffic)
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow HTTPS to external services (S3, APIs)
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
    # Allow database access
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432
```

### Audit Logging

**Principle**: Log all secret access for audit trail.

**Kubernetes Audit Policy**:
```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  # Log secret access
  - level: RequestResponse
    resources:
      - group: ""
        resources: ["secrets"]
    namespaces: ["notebook-validation"]
  # Log pod creation (to track which notebooks ran)
  - level: Metadata
    resources:
      - group: ""
        resources: ["pods"]
    namespaces: ["notebook-validation"]
    verbs: ["create"]
```

**Audit Log Query** (find secret access):
```bash
# Find all secret access in last 24 hours
kubectl logs -n kube-system kube-apiserver-* | \
  jq 'select(.objectRef.resource=="secrets" and .objectRef.namespace=="notebook-validation")'
```

## Consequences

### Positive

1. **Least Privilege**: Notebooks can only access secrets in their namespace
2. **Defense in Depth**: Multiple security layers (RBAC, PSS, NetworkPolicy)
3. **Audit Trail**: All secret access logged
4. **Compliance**: Meets SOC 2, PCI-DSS requirements
5. **Reduced Attack Surface**: Restricted capabilities, no privilege escalation

### Negative

1. **Operational Overhead**: More RBAC resources to manage
2. **Complexity**: More complex security model
3. **Potential Breakage**: Restrictive policies may break some notebooks

### Neutral

1. **No Code Changes**: Operator enforces security policies automatically
2. **Optional NetworkPolicy**: Users can add NetworkPolicy if needed

## Implementation

### Phase 1: RBAC Setup
- [ ] Create ServiceAccount template
- [ ] Create Role template
- [ ] Create RoleBinding template
- [ ] Document RBAC setup

### Phase 2: Pod Security Standards
- [ ] Update operator to enforce Restricted PSS
- [ ] Add securityContext to Pod spec
- [ ] Test with Restricted PSS

### Phase 3: Audit Logging
- [ ] Configure Kubernetes audit policy
- [ ] Create audit log queries
- [ ] Document audit procedures

### Phase 4: Network Policies (Optional)
- [ ] Create NetworkPolicy templates
- [ ] Test network isolation
- [ ] Document NetworkPolicy setup

## Security Checklist

**Before deploying notebooks**:
- [ ] ServiceAccount created with least-privilege RBAC
- [ ] Namespace labeled with Pod Security Standard (Restricted)
- [ ] Secrets created with appropriate access controls
- [ ] Audit logging enabled
- [ ] NetworkPolicy configured (optional)
- [ ] Security context enforced (runAsNonRoot, drop capabilities)

## Troubleshooting

### Issue: Pod fails with "forbidden: violates PodSecurity"

**Symptoms**: Pod creation fails with PSS violation

**Diagnosis**:
```bash
kubectl describe pod <pod-name> -n notebook-validation
```

**Common Causes**:
- Running as root (runAsNonRoot: false)
- Privilege escalation allowed
- Capabilities not dropped

**Solution**: Update Pod securityContext to meet Restricted PSS

### Issue: "forbidden: User cannot get resource 'secrets'"

**Symptoms**: Pod fails to mount secret

**Diagnosis**:
```bash
kubectl auth can-i get secrets --as=system:serviceaccount:notebook-validation:notebook-validator-sa -n notebook-validation
```

**Solution**: Create Role and RoleBinding for ServiceAccount

## Alternatives Considered

### Alternative 1: No RBAC (use default ServiceAccount)
**Rejected**: Security risk, no least-privilege

### Alternative 2: ClusterRole (cluster-wide access)
**Rejected**: Violates least-privilege, allows cross-namespace access

### Alternative 3: Privileged Pod Security Standard
**Rejected**: Too permissive, allows privilege escalation

## Related ADRs

- **ADR-005**: RBAC and Security Model (operator RBAC)
- **ADR-014**: Notebook Credential Injection Strategy (overall strategy)
- **ADR-015**: Environment-Variable Pattern (Tier 1)
- **ADR-016**: External Secret Operator Integration (Tier 2)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Tier 3)
- **ADR-018**: Secret Rotation & Lifecycle Management (rotation policies)

## References

- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [Kubernetes Audit Logging](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)
- [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

