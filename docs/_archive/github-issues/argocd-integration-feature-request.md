# GitHub Issue: Feature Request - ArgoCD Integration for GitOps Workflows

**Issue Type:** Feature Request / Enhancement
**Priority:** High
**Labels:** `enhancement`, `argocd`, `gitops`, `integration`, `help wanted`

---

## Title

**Feature Request: ArgoCD Integration for GitOps Workflows**

---

## Summary

The jupyter-notebook-validator-operator is a production-ready operator with comprehensive ML model validation capabilities. However, it currently lacks ArgoCD integration, creating coordination gaps in GitOps workflows. This is especially problematic for KServe model serving scenarios where notebook completion should trigger downstream resource updates.

This feature request proposes adding five ArgoCD integration features to enable complete GitOps workflows for ML model training and serving.

---

## Current Problems

### Problem 1: No Coordination Between Notebooks and InferenceServices

**Scenario:**
1. NotebookValidationJob trains model → saves to PVC → completes successfully
2. InferenceService is already running with old model
3. InferenceService has no idea a new model is available
4. **Manual intervention required:** `kubectl delete pod` to trigger reload

**Impact:**
- Models don't auto-update after training
- Requires manual kubectl commands
- Breaks GitOps declarative workflow

### Problem 2: ArgoCD Application Shows "Synced" But Models Are Stale

**Scenario:**
1. ArgoCD syncs → deploys InferenceService
2. Notebook runs 10 minutes later → trains new model
3. ArgoCD still shows "Synced" (technically correct for K8s resources)
4. But model in production is outdated

**Impact:**
- False sense of "everything is current"
- No visibility into model freshness
- Drift between code sync and model sync

### Problem 3: No Dependency Ordering in GitOps

**Scenario:**
1. Want to ensure: Notebook completes → Model validated → InferenceService deployed
2. Current options:
   - Manual sync wave ordering (brittle)
   - Wait/sleep commands (timing issues)
   - Manual intervention (not GitOps)

**Impact:**
- Race conditions
- Failed deployments if models not ready
- Cannot express "deploy InferenceService AFTER notebook succeeds"

### Problem 4: No Observability in ArgoCD UI

**Scenario:**
1. User checks ArgoCD Application health
2. Sees InferenceService status but not NotebookValidationJob status
3. Can't tell if model training is in progress/failed
4. Must switch to `oc get notebookvalidationjob` in separate terminal

**Impact:**
- Poor UX for operators
- Hidden failures in notebook validation
- No single pane of glass

---

## Real-World Context

This feature request comes from production use in the [openshift-aiops-platform](https://github.com/tosin2013/openshift-aiops-platform) project:

**Current Status:**
- NotebookValidationJobs: 30/32 succeeded (93.75% success rate)
- InferenceServices: 1/2 operational (anomaly-detector ✅, predictive-analytics ❌)
- **Root Issue:** Model training notebooks complete, but InferenceServices don't reload models
- **Workaround:** Manual `kubectl delete pod` commands required

**Related PR:**
- [PR #23](https://github.com/tosin2013/openshift-aiops-platform/pull/23) - KServe webhook compatibility fix (demonstrates model loading issues)

---

## Proposed Features

### Feature 1: ArgoCD Health Assessment Hook ⭐ **HIGH PRIORITY**

**What:**
Custom health check configuration that reports NotebookValidationJob status to ArgoCD

**Why:**
ArgoCD Application health would include notebook validation status

**How It Works:**
ArgoCD supports custom health checks via Lua scripts in ConfigMap. The operator would document the health assessment configuration:

```yaml
# ConfigMap: argocd-cm (in argocd namespace)
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
data:
  resource.customizations.health.mlops.mlops.dev_NotebookValidationJob: |
    hs = {}
    if obj.status == nil or obj.status.phase == nil then
      hs.status = "Progressing"
      hs.message = "Initializing notebook validation"
      return hs
    end

    if obj.status.phase == "Succeeded" then
      hs.status = "Healthy"
      hs.message = "Notebook validation succeeded"
    elseif obj.status.phase == "Failed" then
      hs.status = "Degraded"
      hs.message = obj.status.message or "Notebook validation failed"
    elseif obj.status.phase == "ValidationRunning" then
      hs.status = "Progressing"
      hs.message = "Validating notebook..."
    else
      hs.status = "Progressing"
      hs.message = "Phase: " .. obj.status.phase
    end

    return hs
```

**Benefits:**
- ✅ ArgoCD UI shows notebook status
- ✅ Application health reflects model training state
- ✅ Operators see issues immediately

**Implementation Complexity:** **LOW** (documentation only, no code changes)
**Estimated Effort:** 1 week

---

### Feature 2: Post-Success Resource Hook ⭐ **HIGH PRIORITY**

**What:**
Operator annotation to trigger resource updates when notebook succeeds

**Why:**
Automatically restart InferenceServices when new models are trained

**How It Works:**
Add annotation-based triggering system:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: predictive-analytics-kserve-validation
  annotations:
    mlops.dev/on-success-trigger: |
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: predictive-analytics
        namespace: self-healing-platform
        action: restart
spec:
  # ... existing spec
```

**Controller Logic:**
When NotebookValidationJob phase = `Succeeded`:
1. Parse `mlops.dev/on-success-trigger` annotation
2. For each trigger:
   - `action: restart` → Delete pods to trigger reload
   - `action: sync` → Patch ArgoCD Application to trigger sync
   - `action: refresh` → Add annotation to force resource refresh
3. Record Kubernetes Events for observability

**Benefits:**
- ✅ Models auto-update after training
- ✅ No manual intervention
- ✅ Declarative dependency expression

**Implementation Complexity:** **MEDIUM**
**Estimated Effort:** 2-3 weeks

---

### Feature 3: ArgoCD Sync Wave Awareness ⭐ **MEDIUM PRIORITY**

**What:**
Operator respects and coordinates with ArgoCD sync waves

**Why:**
Ensure notebooks complete before dependent resources deploy

**How It Works:**
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"
    mlops.dev/block-wave: "4"  # Block wave 4 until this succeeds
spec:
  # ...
```

**Controller Updates:**
- When job succeeds → Add annotation: `mlops.dev/wave-complete: "3"`
- When job fails → Add annotation: `mlops.dev/wave-failed: "3"`
- ArgoCD PreSync hooks can wait for `wave-complete` before proceeding

**Benefits:**
- ✅ Guaranteed ordering: notebooks → models → InferenceServices
- ✅ Prevents race conditions
- ✅ GitOps-native dependency management

**Implementation Complexity:** **LOW-MEDIUM**
**Estimated Effort:** 1-2 weeks

---

### Feature 4: ArgoCD Application Status Integration ⭐ **MEDIUM PRIORITY**

**What:**
Operator publishes aggregated status to ArgoCD Application annotations

**Why:**
Centralized visibility of notebook validation state

**How It Works:**
Controller updates parent Application:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: self-healing-platform
  annotations:
    mlops.dev/notebook-status: |
      {
        "total": 32,
        "succeeded": 30,
        "failed": 2,
        "running": 0,
        "lastUpdate": "2026-01-24T22:30:00Z",
        "failedJobs": ["predictive-analytics-kserve-validation"]
      }
```

**Benefits:**
- ✅ ArgoCD UI shows aggregated notebook status
- ✅ No need to run separate kubectl commands
- ✅ Single source of truth

**Implementation Complexity:** **MEDIUM**
**Estimated Effort:** 2 weeks

---

### Feature 5: ArgoCD Notification Hooks ⭐ **LOW PRIORITY**

**What:**
Trigger ArgoCD notifications on job completion

**Why:**
Alert operators when model training fails

**How It Works:**
Operator creates Kubernetes Events that ArgoCD notifications can consume:

```yaml
apiVersion: v1
kind: Event
metadata:
  labels:
    mlops.dev/notification-type: validation-failure
type: Warning
reason: ValidationFailed
message: |
  NotebookValidationJob 'predictive-analytics-kserve-validation' failed
  Retry count: 3/3 (exhausted)
  Suggested action: Check notebook cell execution
```

**Benefits:**
- ✅ Proactive alerts
- ✅ Integration with Slack, email, PagerDuty, etc.
- ✅ Faster incident response

**Implementation Complexity:** **LOW**
**Estimated Effort:** 1 week

---

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2) - **Immediate Priority**
**Goal:** Add basic ArgoCD awareness

**Tasks:**
1. Document ArgoCD health assessment configuration (Feature 1)
2. Create ADR-048
3. Add health check examples to operator documentation
4. Test with existing deployments
5. Update README with ArgoCD integration section

**Deliverable:** ArgoCD can display NotebookValidationJob health

---

### Phase 2: Resource Triggering (Weeks 3-5) - **High Priority**
**Goal:** Implement post-success hooks (Feature 2)

**Tasks:**
1. Design annotation format
2. Implement controller logic
3. Add RBAC permissions
4. Write unit + e2e tests
5. Document supported actions

**Deliverable:** InferenceServices auto-restart after model training

---

### Phase 3: Sync Wave Integration (Weeks 6-7) - **Medium Priority**
**Goal:** Coordinate with ArgoCD sync waves (Feature 3)

**Tasks:**
1. Implement annotation management
2. Document PreSync hook patterns
3. Test with multi-wave Applications
4. Create example manifests

**Deliverable:** Notebooks block subsequent waves until completion

---

### Phase 4: Observability (Weeks 8-10) - **Medium Priority**
**Goal:** Improve visibility (Features 4 & 5)

**Tasks:**
1. Implement Application status aggregation
2. Add Prometheus metrics
3. Create notification templates
4. Update documentation

**Deliverable:** Full observability in ArgoCD UI

---

## End-to-End Example

### Before (Current State - Manual Intervention Required)
```yaml
# Wave 3: Notebook
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"
# ... trains model

---
# Wave 4: InferenceService (races with notebook!)
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "4"
```

**Problems:**
- ❌ Race conditions
- ❌ Manual pod deletion required
- ❌ No visibility in ArgoCD

---

### After (With ArgoCD Integration - Fully Automated)
```yaml
# Wave 3: Notebook
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "3"
    mlops.dev/block-wave: "4"  # ← Feature 3
    mlops.dev/on-success-trigger: |  # ← Feature 2
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: predictive-analytics
        action: restart

---
# Wave 4: Wait job (uses Feature 3 annotations)
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/sync-wave: "4"
spec:
  template:
    spec:
      containers:
      - name: wait
        command: ["/bin/bash", "-c"]
        args:
        - |
          until kubectl get notebookvalidationjob ... \
            -o jsonpath='{.metadata.annotations.mlops\.dev/wave-complete}'; do
            sleep 5
          done

---
# Wave 4: InferenceService (deploys ONLY after notebook succeeds)
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "4"
```

**Benefits:**
- ✅ No race conditions (Feature 3)
- ✅ Auto-restart on model update (Feature 2)
- ✅ Health visible in ArgoCD UI (Feature 1)
- ✅ No manual intervention
- ✅ Fully declarative GitOps

---

## Why This Matters

### Aligns with Operator Strategy
From [ADR-007 Distribution and Catalog Strategy](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/docs/adrs/007-distribution-catalog.md):

> **Integration Points**: Integrate with ArgoCD, Flux, Tekton for continuous validation in CI/CD pipelines

This has been a strategic goal from the beginning but not yet implemented.

### Production Use Case
The openshift-aiops-platform demonstrates this need with:
- 32 NotebookValidationJobs managing model training
- 2 KServe InferenceServices for model serving
- Manual coordination currently required between them
- ArgoCD managing the entire platform deployment

### Enterprise GitOps Standard
ArgoCD is the de facto GitOps tool for Kubernetes/OpenShift. Supporting it makes the operator more valuable for enterprise MLOps workflows.

---

## Acceptance Criteria

After implementing these features:

✅ **Visibility:**
- [ ] ArgoCD Application health reflects notebook validation status
- [ ] Operators can see notebook progress in ArgoCD UI
- [ ] No separate kubectl commands needed

✅ **Automation:**
- [ ] InferenceServices automatically restart when models are trained
- [ ] No manual intervention required
- [ ] Models stay current without drift

✅ **Reliability:**
- [ ] Notebooks complete before InferenceServices deploy
- [ ] No race conditions
- [ ] Sync waves guarantee ordering

✅ **Observability:**
- [ ] Failed validations trigger alerts
- [ ] Application status shows aggregated notebook health
- [ ] Single pane of glass for GitOps + ML workflows

---

## References

- **ADR-048**: ArgoCD Integration Strategy (created alongside this issue)
- **openshift-aiops-platform**: https://github.com/tosin2013/openshift-aiops-platform
- **ArgoCD Health Checks**: https://argo-cd.readthedocs.io/en/stable/operator-manual/health/
- **ArgoCD Sync Waves**: https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/
- **ArgoCD Notifications**: https://argocd-notifications.readthedocs.io/

---

## Community Input Requested

- Are there other ArgoCD integration patterns we should support?
- Should we also support Flux for those not using ArgoCD?
- What priority would you assign to each feature?
- Any concerns about the annotation-based approach?

---

## Next Steps

1. ✅ Create ADR-048
2. ✅ Open this GitHub issue
3. Prototype Feature 1 (health assessment)
4. Test in production environment
5. Gather community feedback
6. Begin Phase 1 implementation if approved

---

## Labels

Please add:
- `enhancement`
- `argocd`
- `gitops`
- `integration`
- `help wanted`
- `good first issue` (for Feature 1 documentation)
- `priority: high` (for Features 1 & 2)

---

## CC

@tosin2013 (operator maintainer)

---

**Thank you for considering this feature request!** This would significantly improve the GitOps experience for ML workflows using this operator.
