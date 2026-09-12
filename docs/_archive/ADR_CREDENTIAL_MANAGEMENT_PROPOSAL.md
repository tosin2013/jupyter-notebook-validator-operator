# ADR Proposal: Notebook Credential Management

**Date:** 2025-11-08  
**Author:** Tosin Akinosho (takinosh@redhat.com)  
**Status:** Proposed  
**Priority:** High - Critical for production notebook workflows

---

## Executive Summary

This proposal introduces **6 new ADRs (ADR-014 through ADR-019)** to formalize how users inject credentials into Jupyter notebooks for accessing external services like AWS S3, databases, and APIs. The current architecture already supports credential injection via `spec.podConfig.env`, but lacks comprehensive documentation, examples, and integration patterns for enterprise secret management systems (ESO, Vault).

**Key Objectives:**
1. Document credential injection patterns for notebooks
2. Provide examples for S3, database, and API access
3. Integrate with External Secrets Operator (ESO)
4. Integrate with HashiCorp Vault for dynamic secrets
5. Define security best practices and RBAC policies

---

## Background

### Current State ✅

The operator **already supports** credential injection through:
- ✅ `spec.podConfig.env` with `valueFrom.secretKeyRef`
- ✅ Native Kubernetes Secrets
- ✅ ESO integration (transparent - no code changes needed)
- ✅ Vault integration (via ESO or sidecar pattern)

**Example (Already Works):**
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

### Gap Analysis ❌

What's **missing**:
- ❌ Comprehensive documentation of credential injection patterns
- ❌ Examples for S3, database, and API access
- ❌ ESO integration examples (AWS, Azure, GCP)
- ❌ Vault integration examples (dynamic secrets, sidecar pattern)
- ❌ Security best practices guide
- ❌ RBAC policies for secret access
- ❌ Secret rotation procedures

---

## Proposed ADRs

### ADR-014: Notebook Credential Injection Strategy
**Category:** Security  
**Priority:** **Critical**

**Decision:**  
Formalize a multi-tier credential injection strategy:
1. **Tier 1:** Static secrets in environment variables (simple use-cases, POCs)
2. **Tier 2:** Kubernetes Secrets + ESO (enterprise secret stores)
3. **Tier 3:** Vault dynamic secrets (short-lived credentials, auto-rotation)

**Rationale:**
- Provides clear on-ramp for users (simple → advanced)
- Supports both simple and enterprise use-cases
- Enables gradual adoption path
- Improves security posture

**Consequences:**
- **Pros:** Consistency, clear guidance, improved security
- **Cons:** Increased documentation, learning curve

---

### ADR-015: Environment-Variable Pattern for Notebook Credentials
**Category:** Security / Deployment  
**Priority:** **High**

**Decision:**  
Standardize environment variable naming conventions:
- **AWS:** `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `S3_BUCKET`
- **Database:** `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- **APIs:** `OPENAI_API_KEY`, `HUGGINGFACE_TOKEN`, `MLFLOW_TRACKING_URI`

**Rationale:**
- Follows industry standards (AWS SDK, boto3, psycopg2)
- Reduces user confusion
- Enables portable notebooks

**Consequences:**
- **Pros:** Simple to adopt, minimal infrastructure needs
- **Cons:** No automated rotation, secrets still static

---

### ADR-016: External Secret Operator (ESO) Integration
**Category:** Security / Deployment  
**Priority:** **High**

**Decision:**  
Use ESO to automatically sync secrets from external vaults into Kubernetes Secrets.

**Example:**
```yaml
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
    - secretKey: AWS_ACCESS_KEY_ID
      remoteRef:
        key: prod/notebook/aws_access_key_id
```

**Rationale:**
- Declarative secret sync
- Supports AWS, Azure, GCP, Vault, 1Password
- Automatic refresh
- Audit trail from external store

**Consequences:**
- **Pros:** Centralized secret management, auto-rotation
- **Cons:** Adds ESO dependency, static secrets still need manual rotation upstream

---

### ADR-017: Vault Dynamic-Secrets Injection Pattern
**Category:** Security / Deployment  
**Priority:** **High**

**Decision:**  
Adopt the "Vault Agent Sidecar" pattern for dynamic secrets.

**Pattern:**
1. Sidecar container runs Vault Agent
2. Agent renders credentials into shared volume
3. Notebook container reads credentials from volume or env vars
4. Leverage Kubernetes auth method (ServiceAccount → Vault role)

**Rationale:**
- Short-lived credentials (reduced blast radius)
- Automatic rotation
- Just-in-time credential generation
- Audit trail in Vault

**Consequences:**
- **Pros:** Best security posture, automatic rotation
- **Cons:** More complex Pod spec, requires Vault infrastructure

---

### ADR-018: Secret Rotation & Lifecycle Management
**Category:** Security  
**Priority:** Medium

**Decision:**  
Define rotation policies:
- **Static secrets:** Rotate at least quarterly via ESO configuration
- **Dynamic secrets:** Rely on Vault TTL and renew logic
- **Compromised secrets:** Immediate revocation procedures

**Rationale:**
- Reduces risk of credential compromise
- Enables compliance (SOC 2, PCI-DSS)
- Provides audit trail

**Consequences:**
- **Pros:** Improved security posture, auditability
- **Cons:** Operational overhead, potential notebook interruptions

---

### ADR-019: RBAC & Pod Security Policies for Notebook Secret Access
**Category:** Security / Process  
**Priority:** Medium

**Decision:**  
- Define RBAC roles limiting access to secret resources
- Enforce Pod Security Standards (PSS)
- ServiceAccount per notebook namespace with least privilege

**Rationale:**
- Prevents unauthorized secret access
- Aligns with principle of least privilege
- Reduces attack surface

**Consequences:**
- **Pros:** Minimizes unauthorized access, compliance-ready
- **Cons:** Additional RBAC policy management

---

## Implementation Plan

### Phase 4.2: Notebook Credential Management

**Timeline:** Weeks 4-5  
**Priority:** High

#### Tasks

**1. ADR Creation (6 ADRs)**
- [ ] Create ADR-014: Notebook Credential Injection Strategy
- [ ] Create ADR-015: Environment-Variable Pattern
- [ ] Create ADR-016: ESO Integration
- [ ] Create ADR-017: Vault Dynamic-Secrets Pattern
- [ ] Create ADR-018: Secret Rotation & Lifecycle
- [ ] Create ADR-019: RBAC & Pod Security Policies

**2. Documentation**
- [ ] Create `docs/NOTEBOOK_CREDENTIALS_GUIDE.md`
  - Overview of credential injection patterns
  - AWS S3 access examples
  - Database connection examples
  - API key injection examples
  - ESO integration examples
  - Vault integration examples
  - Security best practices
  - Troubleshooting guide

**3. Examples**
- [ ] Create example notebooks
  - S3 data pipeline notebook
  - Database feature engineering notebook
  - Multi-service notebook (S3 + DB + API + MLflow)
- [ ] Create sample CRD manifests
  - `mlops_v1alpha1_notebookvalidationjob_s3.yaml`
  - `mlops_v1alpha1_notebookvalidationjob_database.yaml`
  - `mlops_v1alpha1_notebookvalidationjob_multi_service.yaml`
- [ ] Create ESO configuration examples
  - AWS Secrets Manager integration
  - Azure Key Vault integration
  - GCP Secret Manager integration
  - Vault integration
- [ ] Create secret templates
  - `aws-credentials-secret.yaml`
  - `database-credentials-secret.yaml`
  - `api-keys-secret.yaml`

**4. Integration (Optional)**
- [ ] Test ESO with AWS Secrets Manager
- [ ] Test ESO with Azure Key Vault
- [ ] Test ESO with GCP Secret Manager
- [ ] Test Vault Agent sidecar pattern
- [ ] Test dynamic database credentials
- [ ] Test dynamic AWS credentials

---

## Success Criteria

### Documentation
- ✅ All 6 ADRs created and reviewed
- ✅ Comprehensive credential guide published
- ✅ Examples for S3, database, and API access
- ✅ ESO integration examples for AWS/Azure/GCP
- ✅ Vault integration examples

### Functionality
- ✅ Notebooks can access S3 with credentials
- ✅ Notebooks can connect to databases with credentials
- ✅ Notebooks can use API keys from secrets
- ✅ ESO examples work with AWS/Azure/GCP
- ✅ Vault dynamic secrets work with sidecar pattern

### Security
- ✅ RBAC policies enforce least privilege
- ✅ Pod Security Standards enforced
- ✅ Secret rotation procedures documented
- ✅ Audit trail for secret access

---

## Benefits

### For Users
- **Clear guidance** on credential injection patterns
- **Examples** for common use-cases (S3, DB, API)
- **Security best practices** built-in
- **Gradual adoption path** (simple → advanced)

### For Enterprise
- **ESO integration** for centralized secret management
- **Vault integration** for dynamic secrets
- **RBAC policies** for access control
- **Audit trail** for compliance

### For Security
- **Least privilege** access to secrets
- **Secret rotation** procedures
- **Pod Security Standards** enforcement
- **Reduced attack surface**

---

## Next Steps

1. **Review & Approve** - Circulate this proposal for feedback
2. **Create ADRs** - Write the 6 ADRs (ADR-014 through ADR-019)
3. **Create Documentation** - Write the comprehensive credential guide
4. **Create Examples** - Build example notebooks and manifests
5. **Test Integration** - Verify ESO and Vault integration
6. **Publish** - Make documentation accessible to users

---

## References

- **ADR-009:** Secret Management and Git Credentials (existing)
- **External Secrets Operator:** https://external-secrets.io/
- **HashiCorp Vault:** https://www.vaultproject.io/
- **AWS Secrets Manager:** https://aws.amazon.com/secrets-manager/
- **Azure Key Vault:** https://azure.microsoft.com/en-us/services/key-vault/
- **GCP Secret Manager:** https://cloud.google.com/secret-manager

---

**Document Generated:** 2025-11-08  
**Author:** Tosin Akinosho (takinosh@redhat.com)  
**Status:** Proposed - Awaiting Review

