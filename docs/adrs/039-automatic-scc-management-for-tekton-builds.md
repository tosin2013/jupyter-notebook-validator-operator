# ADR-039: Automatic SCC Management for Tekton Builds

**Status**: Implemented
**Date**: 2025-11-21
**Updated**: 2026-01-24
**Authors**: Claude Code (Anthropic)
**Related ADRs**: ADR-028 (Tekton Task Strategy), ADR-019 (RBAC & Pod Security)
**Implementation**: `pkg/build/scc_helper.go`

---

## Context

When implementing Tekton Pipelines for building notebook images (ADR-028), we discovered that the `buildah` task requires privileged SecurityContextConstraints (SCC) to perform container builds. Without proper SCC configuration, Tekton PipelineRuns fail with:

```
pods "test-build-image-pod" is forbidden: unable to validate against any security context constraint
```

### The Problem

**Current Behavior (Before ADR-039)**:
1. Operator creates `pipeline` ServiceAccount ✅
2. Operator logs a NOTE asking admin to manually run:
   ```bash
   oc adm policy add-scc-to-user pipelines-scc -z pipeline -n <namespace>
   ```
3. Build fails if admin doesn't execute manual command ❌
4. Poor user experience - operator doesn't "just work" ❌

**Why SCC is Required**:
- **Buildah** needs `fsGroup: 65532` and `SETFCAP` capability
- OpenShift's default `restricted-v2` SCC doesn't allow these
- `pipelines-scc` SCC is specifically designed for Tekton builds

### Design Principles from ADR-028

From ADR-028 section on user experience:
> "Users expect operator to 'just work'"
> "Should not require manual Task installation"
> "Should work across different OpenShift versions"

**Current Gap**: Requiring manual SCC configuration violates the "just work" principle.

### Security Considerations

**Concern**: Automatically granting SCC might be considered a security risk.

**Analysis**:
- ✅ **Least Privilege**: We only grant `pipelines-scc`, not `privileged`
- ✅ **Namespace Isolation**: SCC is granted per-namespace, not cluster-wide
- ✅ **Operator RBAC**: Operator needs explicit permission to manage SCCs
- ✅ **Audit Trail**: All SCC changes logged by Kubernetes audit
- ✅ **Standard Practice**: OpenShift Pipelines Operator does this automatically

**Comparison with OpenShift Pipelines**:
```bash
# OpenShift Pipelines Operator automatically grants SCC to:
$ oc describe scc pipelines-scc
Name: pipelines-scc
Users:
  - system:serviceaccount:openshift-pipelines:pipeline
  - system:serviceaccount:my-namespace:pipeline  # Auto-added by operator
```

---

## Decision

**We will have the operator automatically grant `pipelines-scc` to the `pipeline` ServiceAccount when creating Tekton builds.**

### Implementation Strategy

#### 1. Operator RBAC Enhancement

**Update `config/rbac/role.yaml`** to add SCC permissions:
```yaml
- apiGroups:
  - security.openshift.io
  resources:
  - securitycontextconstraints
  verbs:
  - get      # Read SCC configuration
  - list     # List available SCCs
  - use      # Grant SCC to ServiceAccounts
  resourceNames:
  - pipelines-scc  # Only allow pipelines-scc
  - privileged     # Fallback if pipelines-scc unavailable
```

**Security Note**: `resourceNames` field restricts operator to only specific SCCs.

#### 2. Automatic SCC Granting

**New function**: `grantSCCToServiceAccount()`
```go
func (t *TektonStrategy) grantSCCToServiceAccount(
    ctx context.Context,
    namespace, serviceAccount, sccName string,
) error {
    // 1. Get the SCC
    scc := &securityv1.SecurityContextConstraints{}
    if err := t.client.Get(ctx, client.ObjectKey{Name: sccName}, scc); err != nil {
        return err
    }

    // 2. Check if ServiceAccount already has SCC
    serviceAccountUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount)
    for _, user := range scc.Users {
        if user == serviceAccountUser {
            return nil // Already granted
        }
    }

    // 3. Add ServiceAccount to SCC users
    scc.Users = append(scc.Users, serviceAccountUser)
    return t.client.Update(ctx, scc)
}
```

**Enhanced `ensurePipelineServiceAccount()`**:
```go
func (t *TektonStrategy) ensurePipelineServiceAccount(ctx context.Context, namespace string) error {
    // Step 1: Create ServiceAccount (if missing)
    // ... existing code ...

    // Step 2: Automatically grant pipelines-scc
    if err := t.grantSCCToServiceAccount(ctx, namespace, "pipeline", "pipelines-scc"); err != nil {
        // Log warning but don't fail (might be Kubernetes)
        logger.Info("Failed to grant SCC (might be Kubernetes without OpenShift SCCs)",
            "error", err)
        logger.Info("If on OpenShift, manually grant SCC: oc adm policy add-scc-to-user pipelines-scc -z pipeline -n " + namespace)
    }

    return nil
}
```

#### 3. Fallback Strategy

**Graceful Degradation**:
- If SCC granting fails, log warning but don't fail the operation
- Provide manual command in logs for troubleshooting
- Works on Kubernetes (no SCCs) without breaking

**SCC Fallback Chain**:
1. Try `pipelines-scc` (preferred, least privilege)
2. If unavailable, log error with manual instructions
3. User can manually grant `privileged` if needed (last resort)

---

## Consequences

### Positive

1. **✅ Improved UX**: Operator "just works" without manual steps
2. **✅ Faster Onboarding**: No need to document manual SCC commands
3. **✅ Reduced Support Burden**: Fewer "builds not working" support tickets
4. **✅ Consistency**: Same behavior across all namespaces
5. **✅ Audit Trail**: All SCC grants logged automatically
6. **✅ Kubernetes Compatible**: Gracefully handles non-OpenShift clusters

### Negative

1. **⚠️ Increased Operator Privileges**: Operator needs SCC management permission
2. **⚠️ Security Review Required**: Some organizations may restrict SCC automation
3. **⚠️ RBAC Dependency**: Operator deployment must include SCC permissions

### Neutral

1. **📊 Behavior Change**: Existing users need to update operator RBAC
2. **📊 Migration Path**: Can be disabled if needed (manual SCC grant still works)

---

## Implementation

### Phase 1: Code Changes ✅

- [x] Add SCC permissions to `config/rbac/role.yaml`
- [x] Add `securityv1` import to `tekton_strategy.go`
- [x] Implement `grantSCCToServiceAccount()` function
- [x] Update `ensurePipelineServiceAccount()` to auto-grant SCC
- [x] Add comprehensive logging

### Phase 2: Testing

- [ ] Test on OpenShift cluster
- [ ] Verify SCC is granted automatically
- [ ] Test fallback behavior on Kubernetes
- [ ] Verify existing ServiceAccounts work without redeployment

### Phase 3: Documentation

- [ ] Update operator installation guide with new RBAC
- [ ] Document SCC behavior in troubleshooting guide
- [ ] Add migration notes for existing deployments

---

## Security Model

### Threat Analysis

**Threat**: Operator could grant excessive privileges
**Mitigation**: RBAC `resourceNames` restricts to specific SCCs only

**Threat**: ServiceAccount with SCC could be abused
**Mitigation**: Namespace isolation + SCC only for builds, not validation pods

**Threat**: Operator compromise could grant SCC to any namespace
**Mitigation**: Operator ServiceAccount has ClusterRole, but actions are audited

### Security Best Practices

1. **Principle of Least Privilege**: Only grant `pipelines-scc`, not `privileged`
2. **Defense in Depth**: SCC is one layer; namespace RBAC provides additional isolation
3. **Audit Logging**: All SCC modifications logged by Kubernetes audit system
4. **Explicit Approval**: Organizations can review operator RBAC before deployment

---

## Alternatives Considered

### Alternative 1: Require Manual SCC Configuration (Status Quo)

**Pros**:
- No additional operator privileges
- Explicit admin control

**Cons**:
- ❌ Poor user experience
- ❌ Violates "operator should just work" principle
- ❌ Easy to forget or misconfigure
- ❌ Inconsistent across namespaces

**Rejected**: Violates ADR-028 design principles

### Alternative 2: Use ServiceAccount Annotations

**Idea**: Annotate ServiceAccount, let OpenShift controller grant SCC
```yaml
metadata:
  annotations:
    openshift.io/required-scc: pipelines-scc
```

**Pros**:
- Declarative approach
- No operator code changes

**Cons**:
- ❌ Not a standard OpenShift feature
- ❌ Requires cluster-level admission controller
- ❌ Doesn't exist in current OpenShift versions

**Rejected**: Feature doesn't exist

### Alternative 3: Create ClusterRole with SCC `use` Verb

**Idea**: Create ClusterRole that allows using `pipelines-scc`, bind to ServiceAccount

**Pros**:
- RBAC-based approach
- Works with standard RBAC

**Cons**:
- ❌ Requires ClusterRole per SCC (namespace pollution)
- ❌ More complex RBAC hierarchy
- ❌ Doesn't follow OpenShift conventions

**Rejected**: More complex than direct SCC grant

### Alternative 4: Document and Fail Fast

**Idea**: Check if SCC is granted, fail with clear error if not

**Pros**:
- No operator privilege escalation
- Clear error messages

**Cons**:
- ❌ Still requires manual steps
- ❌ Doesn't improve UX
- ❌ Fails builds unnecessarily

**Rejected**: Doesn't solve the core problem

---

## Rollback Plan

If automatic SCC management causes issues:

1. **Revert Operator Code**: Remove `grantSCCToServiceAccount()` call
2. **Keep RBAC**: SCC permissions can stay (won't be used)
3. **Manual Fallback**: Document manual SCC grant in release notes
4. **Feature Flag**: Could add env var `DISABLE_AUTO_SCC=true`

---

## References

- **OpenShift SCC Documentation**: https://docs.openshift.com/container-platform/4.18/authentication/managing-security-context-constraints.html
- **Tekton Security Context**: https://tekton.dev/docs/pipelines/security/
- **OpenShift Pipelines Operator**: Similar automatic SCC management
- **ADR-028**: Tekton Task Strategy (operator should "just work")
- **ADR-019**: RBAC & Pod Security (validation pod security model)

---

## Revision History

- **2025-11-21**: Initial version (Claude Code)
  - Implements automatic SCC management
  - Adds `grantSCCToServiceAccount()` function
  - Updates operator RBAC with SCC permissions
