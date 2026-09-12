# GitHub Issue: Feature Request - Automatic Tekton Git Credentials Synchronization

**Issue Type:** Feature Request / Enhancement
**Priority:** Medium
**Labels:** `enhancement`, `tekton`, `credentials`, `automation`, `good first issue`

---

## Title

**Feature Request: Automatic Tekton Git Credentials Synchronization**

---

## Summary

Add automatic synchronization of git credentials when source secrets are updated. Currently, the operator converts secrets from Kubernetes format to Tekton format on creation, but doesn't watch for updates to the source secret.

**Status:** TODO identified in [ADR-042](../adrs/042-automatic-tekton-git-credentials-conversion.md)
**Priority:** Medium
**Category:** Secret Management, Tekton Integration

---

## Problem Statement

### Current Behavior

When using Tekton builds with git credentials:

1. User creates source secret (Kubernetes format):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: github-credentials
   type: kubernetes.io/basic-auth
   stringData:
     username: myuser
     password: ghp_xxxxxxxxxxxx
   ```

2. Operator creates Tekton secret (Tekton format):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: github-credentials-tekton
     annotations:
       tekton.dev/git-0: https://github.com
   type: kubernetes.io/basic-auth
   stringData:
     username: myuser
     password: ghp_xxxxxxxxxxxx
   ```

3. NotebookValidationJob references source secret
4. Tekton build uses `-tekton` secret
5. ✅ Build succeeds

### The Problem

**Scenario:**
1. Source secret `github-credentials` is updated (new token)
2. Tekton secret `github-credentials-tekton` is NOT updated
3. Next build uses stale credentials from `-tekton` secret
4. ❌ Build fails with authentication error

**Impact:**
- Credentials get out of sync
- Builds fail unexpectedly after credential rotation
- Users must manually update both secrets
- Not GitOps-friendly (secret updates don't propagate)

---

## Current Code Location

**File:** `pkg/build/tekton_strategy.go:252`

```go
if err == nil {
    // Secret exists, check if it's managed by us
    if existingSecret.Labels["app.kubernetes.io/managed-by"] == "jupyter-notebook-validator-operator" {
        logger.V(1).Info("Tekton Git credentials secret already exists", "secret", tektonSecretName, "namespace", namespace)
        // TODO: Check if source secret has changed and update if needed
    } else {
        logger.Info("Tekton Git credentials secret exists but not managed by operator, skipping", "secret", tektonSecretName, "namespace", namespace)
    }
    return nil
}
```

---

## Proposed Solution

Implement automatic synchronization of Tekton secrets when source secrets change.

### Approach 1: Watch Source Secrets (Recommended)

Add a watch for secrets referenced by NotebookValidationJobs:

```go
// In controller setup
func (r *NotebookValidationJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&mlopsv1alpha1.NotebookValidationJob{}).
        Owns(&corev1.Pod{}).
        Owns(&buildv1.BuildConfig{}).
        Owns(&pipelinev1.PipelineRun{}).
        // NEW: Watch secrets to detect credential updates
        Watches(
            &corev1.Secret{},
            handler.EnqueueRequestsFromMapFunc(r.findNotebookValidationJobsForSecret),
            builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
        ).
        Complete(r)
}

// Map secret changes to NotebookValidationJobs
func (r *NotebookValidationJobReconciler) findNotebookValidationJobsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
    // Find NotebookValidationJobs that reference this secret
    var jobs mlopsv1alpha1.NotebookValidationJobList
    if err := r.List(ctx, &jobs, client.InNamespace(secret.GetNamespace())); err != nil {
        return nil
    }

    var requests []reconcile.Request
    for _, job := range jobs.Items {
        if job.Spec.Notebook.Git.CredentialsSecret == secret.GetName() {
            requests = append(requests, reconcile.Request{
                NamespacedName: types.NamespacedName{
                    Name:      job.Name,
                    Namespace: job.Namespace,
                },
            })
        }
    }

    return requests
}
```

### Approach 2: Reconcile-Time Sync (Alternative)

Check and update on every reconciliation:

```go
func (r *TektonBuildStrategy) ensureTektonGitCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, namespace string) error {
    // ... existing code to find source secret

    tektonSecretName := sourceSecretName + "-tekton"

    // Check if Tekton secret exists
    var existingSecret corev1.Secret
    err := r.Client.Get(ctx, client.ObjectKey{
        Name:      tektonSecretName,
        Namespace: namespace,
    }, &existingSecret)

    if err == nil {
        // Secret exists, check if it's managed by us
        if existingSecret.Labels["app.kubernetes.io/managed-by"] == "jupyter-notebook-validator-operator" {
            logger.V(1).Info("Tekton Git credentials secret exists", "secret", tektonSecretName)

            // NEW: Check if source secret has changed
            sourceSecret := &corev1.Secret{}
            if err := r.Client.Get(ctx, client.ObjectKey{Name: sourceSecretName, Namespace: namespace}, sourceSecret); err == nil {
                if needsUpdate := r.secretNeedsUpdate(&existingSecret, sourceSecret); needsUpdate {
                    logger.Info("Source secret changed, updating Tekton secret", "secret", tektonSecretName)
                    return r.updateTektonSecret(ctx, &existingSecret, sourceSecret)
                }
            }

            return nil
        }
        // Not managed by us, skip
        return nil
    }

    // ... existing code to create secret
}

// Check if Tekton secret needs update
func (r *TektonBuildStrategy) secretNeedsUpdate(tektonSecret, sourceSecret *corev1.Secret) bool {
    // Compare data fields
    if !bytes.Equal(tektonSecret.Data["username"], sourceSecret.Data["username"]) {
        return true
    }
    if !bytes.Equal(tektonSecret.Data["password"], sourceSecret.Data["password"]) {
        return true
    }
    if !bytes.Equal(tektonSecret.Data[".gitconfig"], sourceSecret.Data[".gitconfig"]) {
        return true
    }

    // Check if git URL annotation needs update
    gitURL := extractGitHost(sourceSecret)
    currentAnnotation := tektonSecret.Annotations["tekton.dev/git-0"]
    if currentAnnotation != gitURL {
        return true
    }

    return false
}

// Update Tekton secret with new data from source
func (r *TektonBuildStrategy) updateTektonSecret(ctx context.Context, tektonSecret, sourceSecret *corev1.Secret) error {
    // Copy data from source secret
    tektonSecret.Data = map[string][]byte{
        "username": sourceSecret.Data["username"],
        "password": sourceSecret.Data["password"],
    }

    // Update git URL annotation if needed
    gitURL := extractGitHost(sourceSecret)
    if tektonSecret.Annotations == nil {
        tektonSecret.Annotations = make(map[string]string)
    }
    tektonSecret.Annotations["tekton.dev/git-0"] = gitURL

    // Update the secret
    if err := r.Client.Update(ctx, tektonSecret); err != nil {
        return fmt.Errorf("failed to update Tekton secret %s: %w", tektonSecret.Name, err)
    }

    return nil
}
```

---

## Use Cases

### Use Case 1: Token Rotation

**Scenario:** GitHub token needs to be rotated for security.

**Before:**
```bash
# 1. Update source secret
kubectl create secret generic github-credentials \
  --from-literal=username=myuser \
  --from-literal=password=ghp_NEW_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Manually update Tekton secret
kubectl create secret generic github-credentials-tekton \
  --from-literal=username=myuser \
  --from-literal=password=ghp_NEW_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl annotate secret github-credentials-tekton \
  tekton.dev/git-0=https://github.com

# 3. Restart builds to pick up new token
```

**After (with auto-sync):**
```bash
# 1. Update source secret
kubectl create secret generic github-credentials \
  --from-literal=username=myuser \
  --from-literal=password=ghp_NEW_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Operator automatically updates github-credentials-tekton
# 3. Next build uses new token automatically
```

---

### Use Case 2: GitOps Secret Management

**Scenario:** Secrets managed via External Secrets Operator or Sealed Secrets.

**Before:**
```yaml
# External Secret creates/updates source secret
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: github-credentials
spec:
  secretStoreRef:
    name: vault-backend
  target:
    name: github-credentials
  data:
  - secretKey: username
    remoteRef:
      key: github/credentials
      property: username
  - secretKey: password
    remoteRef:
      key: github/credentials
      property: token

# Problem: github-credentials-tekton doesn't update
# Must manually trigger operator to recreate Tekton secret
```

**After (with auto-sync):**
```yaml
# External Secret creates/updates source secret
# Operator watches for changes and updates github-credentials-tekton automatically
# Fully GitOps-compatible workflow
```

---

## Implementation Plan

### Week 1: Design and Setup
- [ ] Choose approach (Watch vs Reconcile-time)
- [ ] Design secret comparison logic
- [ ] Define update triggers
- [ ] Write design doc

### Week 2: Implementation
- [ ] Implement secret watch (if Approach 1)
- [ ] Implement `secretNeedsUpdate()` function
- [ ] Implement `updateTektonSecret()` function
- [ ] Add unit tests

### Week 3: Testing
- [ ] Integration tests for secret updates
- [ ] E2E tests with credential rotation
- [ ] Test with External Secrets Operator
- [ ] Test with GitOps workflows

### Week 4: Documentation
- [ ] Update ADR-042 with implementation details
- [ ] Update user documentation
- [ ] Create credential rotation guide
- [ ] Add troubleshooting section

---

## Future Enhancements (from ADR-042)

Once basic synchronization is implemented, consider:

1. **Multi-Provider Support**: Extract Git host from `spec.notebook.git.url` for annotation
   ```go
   // Auto-detect git provider from URL
   gitURL := job.Spec.Notebook.Git.URL
   gitHost := extractGitHost(gitURL)
   // e.g., https://github.com/user/repo → https://github.com
   ```

2. **SSH Key Support**: Extend to handle SSH-based authentication
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: github-ssh
   type: kubernetes.io/ssh-auth
   data:
     ssh-privatekey: base64-encoded-key
   ```

3. **Cleanup on Deletion**: Delete `-tekton` secret when source secret is deleted
   ```go
   // Add finalizer to source secret
   // Clean up Tekton secret on source secret deletion
   ```

---

## Benefits

### For Users
- ✅ Automatic credential synchronization
- ✅ No manual secret management
- ✅ GitOps-friendly (secrets update automatically)
- ✅ Reduced build failures from stale credentials

### For Platform Teams
- ✅ Easier secret rotation
- ✅ Better integration with External Secrets Operator
- ✅ Reduced operational burden
- ✅ Better security posture (credentials stay in sync)

### For Security
- ✅ Supports credential rotation best practices
- ✅ No stale credentials lingering
- ✅ Works with secret management tools (Vault, etc.)

---

## Consequences

### Positive
- ✅ Improved automation (no manual secret updates)
- ✅ Better GitOps integration
- ✅ Reduced operational burden
- ✅ Fewer build failures from stale credentials

### Negative
- ⚠️ Additional watch overhead (if using Approach 1)
- ⚠️ Potential for unexpected updates if source secret changes frequently
- ⚠️ Need to handle watch conflicts

### Mitigation
- Use resource version to detect actual changes (not just reconcile triggers)
- Add rate limiting to prevent excessive updates
- Clear logging when sync occurs

---

## Acceptance Criteria

- [ ] Tekton secret updates when source secret changes
- [ ] Watch or reconcile-time detection implemented
- [ ] `secretNeedsUpdate()` compares username, password, .gitconfig
- [ ] `updateTektonSecret()` updates data and annotations
- [ ] Unit tests for comparison logic
- [ ] Integration tests for sync behavior
- [ ] E2E tests with credential rotation
- [ ] Documentation updated
- [ ] Migration guide for existing users

---

## References

- **ADR-042**: [Automatic Tekton Git Credentials Conversion](../adrs/042-automatic-tekton-git-credentials-conversion.md)
- **Code Location**: `pkg/build/tekton_strategy.go:252`
- **Tekton Auth Docs**: https://tekton.dev/docs/pipelines/auth/
- **External Secrets Operator**: https://external-secrets.io/

---

## Community Input Requested

- Which approach is preferred (Watch vs Reconcile-time)?
- Should sync be configurable (opt-in/opt-out)?
- What should happen if manual changes are made to `-tekton` secret?

---

## Labels

Please add:
- `enhancement`
- `tekton`
- `credentials`
- `automation`
- `good first issue`
- `priority: medium`

---

## CC

@tosin2013 (operator maintainer)

---

**Thank you for considering this feature request!** This would improve the operator's GitOps integration and secret management.
