# ADR 018: Secret Rotation & Lifecycle Management

## Status
Accepted

## Context

ADR-014, ADR-015, ADR-016, and ADR-017 established credential injection patterns. This ADR addresses **secret rotation and lifecycle management** across all three tiers.

### Problem

Without clear rotation policies, secrets become stale and increase security risk:
- **Static secrets** (Tier 1, Tier 2): Never rotated, long-lived
- **Compromised secrets**: No clear revocation procedure
- **Expired secrets**: Notebooks fail unexpectedly
- **Compliance**: Regulations require regular rotation (SOC 2, PCI-DSS)

### Industry Standards

- **NIST**: Rotate secrets at least annually, more frequently for high-risk systems
- **PCI-DSS**: Rotate credentials every 90 days
- **SOC 2**: Document rotation procedures and audit trail
- **CIS Benchmarks**: Rotate secrets quarterly

## Decision

We define rotation policies for each tier and establish procedures for secret lifecycle management.

### Tier 1: Static Secrets (Manual Rotation)

**Policy**: Rotate at least **quarterly** (every 90 days)

**Procedure**:

1. **Generate new credentials** in external system (AWS, database)
2. **Update Kubernetes Secret**:
   ```bash
   kubectl create secret generic aws-credentials \
     --from-literal=access-key-id=NEW_KEY \
     --from-literal=secret-access-key=NEW_SECRET \
     --dry-run=client -o yaml | kubectl apply -f -
   ```
3. **Verify rotation**: Test with new credentials
4. **Revoke old credentials** in external system
5. **Document rotation**: Update audit log

**Automation** (optional):
```bash
#!/bin/bash
# rotate-aws-credentials.sh

# 1. Generate new AWS credentials
NEW_KEY=$(aws iam create-access-key --user-name notebook-validator --query 'AccessKey.AccessKeyId' --output text)
NEW_SECRET=$(aws iam create-access-key --user-name notebook-validator --query 'AccessKey.SecretAccessKey' --output text)

# 2. Update Kubernetes Secret
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=$NEW_KEY \
  --from-literal=secret-access-key=$NEW_SECRET \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. Wait for propagation (30 seconds)
sleep 30

# 4. Test new credentials
kubectl apply -f test-notebook-job.yaml
kubectl wait --for=condition=complete job/test-notebook-job --timeout=5m

# 5. Revoke old credentials
OLD_KEY=$(aws iam list-access-keys --user-name notebook-validator --query 'AccessKeyMetadata[?CreateDate<`2024-01-01`].AccessKeyId' --output text)
aws iam delete-access-key --user-name notebook-validator --access-key-id $OLD_KEY

# 6. Log rotation
echo "$(date): Rotated AWS credentials for notebook-validator" >> /var/log/secret-rotation.log
```

### Tier 2: ESO-Synced Secrets (Automatic Sync, Manual Rotation)

**Policy**: Rotate at least **quarterly** (every 90 days) in external vault

**Procedure**:

1. **Rotate secret in external vault** (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager)
2. **ESO automatically syncs** new secret to Kubernetes (based on `refreshInterval`)
3. **Verify sync**: Check ExternalSecret status
4. **Test with new credentials**
5. **Document rotation**: Audit trail in external vault

**Example: AWS Secrets Manager**:
```bash
# 1. Rotate secret in AWS Secrets Manager
aws secretsmanager rotate-secret --secret-id prod/notebook/aws

# 2. ESO syncs automatically (within refreshInterval, e.g., 1 hour)
# Check sync status
kubectl get externalsecret aws-credentials -n notebook-validation -o yaml

# 3. Verify new credentials in K8s Secret
kubectl get secret aws-credentials -n notebook-validation -o jsonpath='{.data.access-key-id}' | base64 -d
```

**ESO Configuration for Automatic Refresh**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: aws-credentials
spec:
  refreshInterval: 1h  # Sync every hour
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: aws-credentials
    creationPolicy: Owner
  data:
    - secretKey: access-key-id
      remoteRef:
        key: prod/notebook/aws
        property: access_key_id
```

### Tier 3: Vault Dynamic Secrets (Automatic Rotation)

**Policy**: Rely on Vault TTL and automatic renewal

**Configuration**:
- **Default TTL**: 1 hour (credentials expire after 1 hour)
- **Max TTL**: 24 hours (maximum lease duration)
- **Renewal**: Vault Agent automatically renews at 80% of TTL

**Vault Configuration**:
```bash
# Database role with 1-hour TTL
vault write database/roles/readonly \
    db_name=my-postgresql-database \
    creation_statements="..." \
    default_ttl="1h" \
    max_ttl="24h"

# AWS role with 1-hour TTL
vault write aws/roles/s3-readonly \
    credential_type=assumed_role \
    role_arns=arn:aws:iam::123456789012:role/S3ReadOnlyRole \
    default_ttl=1h \
    max_ttl=24h
```

**Automatic Renewal**:
- Vault Agent monitors lease TTL
- Renews at 80% of TTL (e.g., 48 minutes for 1-hour TTL)
- If renewal fails, fetches new credentials
- Notebook always has valid credentials

**No manual rotation required** - Vault handles everything automatically.

### Compromised Secret Revocation

**Immediate Actions**:

1. **Revoke in external system** (AWS, database, Vault)
2. **Delete Kubernetes Secret**:
   ```bash
   kubectl delete secret aws-credentials -n notebook-validation
   ```
3. **Generate new credentials** (follow rotation procedure)
4. **Audit access logs** to determine scope of compromise
5. **Document incident** in security log

**Example: AWS Credentials Compromised**:
```bash
# 1. Revoke compromised credentials
aws iam delete-access-key --user-name notebook-validator --access-key-id COMPROMISED_KEY

# 2. Delete K8s Secret
kubectl delete secret aws-credentials -n notebook-validation

# 3. Generate new credentials
NEW_KEY=$(aws iam create-access-key --user-name notebook-validator --query 'AccessKey.AccessKeyId' --output text)
NEW_SECRET=$(aws iam create-access-key --user-name notebook-validator --query 'AccessKey.SecretAccessKey' --output text)

# 4. Create new K8s Secret
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=$NEW_KEY \
  --from-literal=secret-access-key=$NEW_SECRET \
  -n notebook-validation

# 5. Audit CloudTrail logs
aws cloudtrail lookup-events --lookup-attributes AttributeKey=AccessKeyId,AttributeValue=COMPROMISED_KEY

# 6. Document incident
echo "$(date): Revoked compromised AWS credentials COMPROMISED_KEY" >> /var/log/security-incidents.log
```

### Secret Expiration Handling

**Tier 1 & Tier 2**: Secrets don't expire automatically
- Monitor secret age
- Alert when secrets are > 80 days old (approaching 90-day rotation)
- Rotate before expiration

**Tier 3**: Vault handles expiration automatically
- Vault Agent renews leases before expiration
- If renewal fails, Vault Agent fetches new credentials
- Notebook always has valid credentials

**Monitoring Script**:
```bash
#!/bin/bash
# check-secret-age.sh

# Get secret creation time
SECRET_AGE=$(kubectl get secret aws-credentials -n notebook-validation -o jsonpath='{.metadata.creationTimestamp}')
SECRET_AGE_DAYS=$(( ($(date +%s) - $(date -d "$SECRET_AGE" +%s)) / 86400 ))

# Alert if > 80 days old
if [ $SECRET_AGE_DAYS -gt 80 ]; then
  echo "WARNING: Secret aws-credentials is $SECRET_AGE_DAYS days old (rotation due in $((90 - SECRET_AGE_DAYS)) days)"
  # Send alert (email, Slack, PagerDuty)
fi
```

## Consequences

### Positive

1. **Reduced risk**: Regular rotation reduces blast radius
2. **Compliance**: Meets SOC 2, PCI-DSS requirements
3. **Audit trail**: Rotation documented in logs
4. **Automation**: Tier 3 (Vault) fully automated
5. **Clear procedures**: Teams know what to do

### Negative

1. **Operational overhead**: Tier 1 and Tier 2 require manual rotation
2. **Potential downtime**: Rotation can cause brief interruptions
3. **Coordination**: Rotation requires coordination across teams

### Neutral

1. **No code changes**: Operator doesn't need to change
2. **Optional automation**: Users can automate Tier 1/2 rotation if desired

## Implementation

### Phase 1: Documentation
- [x] Document rotation policies in this ADR
- [ ] Create rotation procedures for each tier
- [ ] Create automation scripts for Tier 1
- [ ] Create monitoring scripts

### Phase 2: Automation
- [ ] Create rotation automation for Tier 1
- [ ] Create monitoring alerts for secret age
- [ ] Test rotation procedures

### Phase 3: Compliance
- [ ] Document audit trail requirements
- [ ] Create compliance reports
- [ ] Train teams on rotation procedures

## Rotation Schedule

| Tier | Rotation Frequency | Method | Automation |
|------|-------------------|--------|------------|
| **Tier 1** | Quarterly (90 days) | Manual | Optional script |
| **Tier 2** | Quarterly (90 days) | Manual (in external vault) | ESO auto-sync |
| **Tier 3** | Automatic (1 hour TTL) | Vault Agent | Fully automated |

## Compliance Mapping

| Standard | Requirement | How We Meet It |
|----------|-------------|----------------|
| **NIST** | Rotate annually | Quarterly rotation (exceeds requirement) |
| **PCI-DSS** | Rotate every 90 days | Quarterly rotation (meets requirement) |
| **SOC 2** | Document rotation | Audit logs, this ADR |
| **CIS** | Rotate quarterly | Quarterly rotation (meets requirement) |

## Alternatives Considered

### Alternative 1: No Rotation Policy
**Rejected**: Security risk, compliance issues

### Alternative 2: Monthly Rotation
**Rejected**: Too frequent for Tier 1/2, operational overhead

### Alternative 3: Annual Rotation
**Rejected**: Too infrequent, doesn't meet PCI-DSS

## Related ADRs

- **ADR-014**: Notebook Credential Injection Strategy (overall strategy)
- **ADR-015**: Environment-Variable Pattern (Tier 1)
- **ADR-016**: External Secret Operator Integration (Tier 2)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Tier 3)
- **ADR-019**: RBAC & Pod Security Policies (access control)

## References

- [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [PCI-DSS Requirements](https://www.pcisecuritystandards.org/)
- [SOC 2 Compliance](https://www.aicpa.org/interestareas/frc/assuranceadvisoryservices/aicpasoc2report.html)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [AWS Secrets Manager Rotation](https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets.html)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

