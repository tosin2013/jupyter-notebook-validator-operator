# ADR-048: ArgoCD Integration Strategy for GitOps Workflows

**Status**: Proposed
**Date**: 2026-01-24
**Deciders**: TBD
**Context**: GitOps Workflow Coordination and Model Serving Integration

## Context and Problem Statement

The jupyter-notebook-validator-operator is a mature, production-ready operator with comprehensive ML model validation capabilities. However, it currently has **zero ArgoCD integration**, creating coordination gaps in GitOps workflows, especially for KServe model serving where notebook completion should trigger downstream resources.

### Current Operational Issues

**Cluster Status:**
- InferenceServices: 1/2 operational (anomaly-detector ✅, predictive-analytics ❌)
- NotebookValidationJobs: 30/32 succeeded (93.75% success rate)
- Root Issue: Model training notebooks complete, but InferenceServices don't know when to reload models
- Manual intervention required: delete pods to trigger model reload

**ArgoCD Integration Gaps:**
- No Application status reporting
- No sync wave annotations/awareness
- No ArgoCD event/notification integration
- No automatic downstream resource triggering
- No health assessment customization
- No synchronization with Argo Application lifecycle

## Decision Drivers

1. **GitOps Workflow Completeness**: Current manual intervention breaks declarative GitOps principles
2. **Model Lifecycle Coordination**: Need automatic synchronization between model training and serving
3. **Operator Maturity**: 47 ADRs, production-ready operator needs enterprise GitOps features
4. **Platform Detection Exists**: `pkg/platform/detector.go` already provides extensibility foundation
5. **ADR-007 Vision**: Explicitly mentions "Integrate with ArgoCD, Flux, Tekton" as strategic goal
6. **Production Impact**: Real deployment issues in openshift-aiops-platform requiring this integration

## Problems Addressed

### Problem 1: No Coordination Between Notebooks and InferenceServices

**Scenario:**
1. NotebookValidationJob trains model → saves to PVC → completes successfully
2. InferenceService is already running with old model
3. InferenceService has no idea a new model is available
4. Manual intervention required: `kubectl delete pod` to trigger reload

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

## Proposed Features

### Feature 1: ArgoCD Health Assessment Hook ⭐ **HIGH PRIORITY**

**What:**
Custom health check that reports NotebookValidationJob status to ArgoCD

**Why:**
ArgoCD Application health would include notebook validation status

**Implementation:**
ArgoCD supports custom health checks via Lua scripts in ConfigMap. The operator will document the health assessment configuration:

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
      if obj.status.completionTime ~= nil then
        hs.message = hs.message .. " at " .. obj.status.completionTime
      end
    elseif obj.status.phase == "Failed" then
      hs.status = "Degraded"
      hs.message = obj.status.message or "Notebook validation failed"
      if obj.status.retryCount ~= nil then
        hs.message = hs.message .. " (retry " .. obj.status.retryCount .. "/3)"
      end
    elseif obj.status.phase == "ValidationRunning" then
      hs.status = "Progressing"
      hs.message = "Validating notebook..."
      if obj.status.validationPodName ~= nil then
        hs.message = hs.message .. " (pod: " .. obj.status.validationPodName .. ")"
      end
    elseif obj.status.phase == "Building" or obj.status.phase == "BuildComplete" then
      hs.status = "Progressing"
      hs.message = "Building validation environment"
    elseif obj.status.phase == "Initializing" or obj.status.phase == "Pending" then
      hs.status = "Progressing"
      hs.message = "Initializing..."
    else
      hs.status = "Progressing"
      hs.message = "Phase: " .. obj.status.phase
    end

    return hs
```

**Status Mapping:**
- Phase `Succeeded` → ArgoCD `Healthy`
- Phase `Failed` → ArgoCD `Degraded`
- Phase `ValidationRunning`, `Building`, `BuildComplete`, `Initializing`, `Pending` → ArgoCD `Progressing`

**Benefit:**
- ArgoCD UI shows notebook status
- Application health reflects model training state
- Operators see issues immediately in single pane of glass

**Implementation Complexity:** **LOW** (documentation only, no code changes)
**Estimated Effort:** 1 week (testing and documentation)

---

### Feature 2: Post-Success Resource Hook ⭐ **HIGH PRIORITY**

**What:**
Operator annotation to trigger resource updates when notebook succeeds

**Why:**
Automatically restart InferenceServices when new models are trained

**Implementation:**
Add annotation-based triggering system to controller:

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
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: anomaly-detector
        namespace: self-healing-platform
        action: refresh
spec:
  # ... existing spec
```

**Controller Logic:**
```go
// In notebookvalidationjob_controller.go
func (r *NotebookValidationJobReconciler) handleSuccess(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error {
    // Parse on-success-trigger annotation
    triggers, err := parseSuccessTriggers(job.Annotations["mlops.dev/on-success-trigger"])
    if err != nil {
        return fmt.Errorf("failed to parse success triggers: %w", err)
    }

    for _, trigger := range triggers {
        switch trigger.Action {
        case "restart":
            // Delete pods to trigger reload
            if err := r.restartResource(ctx, trigger); err != nil {
                r.Recorder.Event(job, corev1.EventTypeWarning, "TriggerFailed",
                    fmt.Sprintf("Failed to restart %s/%s: %v", trigger.Kind, trigger.Name, err))
            } else {
                r.Recorder.Event(job, corev1.EventTypeNormal, "TriggerSuccess",
                    fmt.Sprintf("Restarted %s/%s", trigger.Kind, trigger.Name))
            }
        case "sync":
            // Patch Application to trigger ArgoCD sync
            if err := r.triggerArgoSync(ctx, trigger); err != nil {
                r.Recorder.Event(job, corev1.EventTypeWarning, "TriggerFailed",
                    fmt.Sprintf("Failed to sync Application %s: %v", trigger.Name, err))
            }
        case "refresh":
            // Update resource annotation to force refresh
            if err := r.forceRefresh(ctx, trigger); err != nil {
                r.Recorder.Event(job, corev1.EventTypeWarning, "TriggerFailed",
                    fmt.Sprintf("Failed to refresh %s/%s: %v", trigger.Kind, trigger.Name, err))
            }
        }
    }

    return nil
}

func (r *NotebookValidationJobReconciler) restartResource(ctx context.Context, trigger Trigger) error {
    // For InferenceServices, delete predictor pods to force reload
    if trigger.Kind == "InferenceService" {
        labelSelector := fmt.Sprintf("serving.kserve.io/inferenceservice=%s", trigger.Name)
        return r.deletePodsByLabel(ctx, trigger.Namespace, labelSelector)
    }

    // For other resources, add restart annotation
    return r.addRestartAnnotation(ctx, trigger)
}
```

**Supported Actions:**
- `restart`: Delete pods to trigger reload (for KServe, deployments, etc.)
- `sync`: Trigger ArgoCD Application sync
- `refresh`: Add annotation to force resource refresh

**Benefit:**
- Models auto-update after training
- No manual intervention
- Declarative dependency expression
- Works across different resource types

**Implementation Complexity:** **MEDIUM**
**Estimated Effort:** 2-3 weeks (controller logic, tests, documentation)

---

### Feature 3: ArgoCD Sync Wave Awareness ⭐ **MEDIUM PRIORITY**

**What:**
Operator respects and coordinates with ArgoCD sync waves

**Why:**
Ensure notebooks complete before dependent resources deploy

**Implementation:**
Add sync-wave blocking capabilities:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: predictive-analytics-kserve-validation
  annotations:
    argocd.argoproj.io/sync-wave: "3"
    mlops.dev/block-wave: "4"  # Block wave 4 until this succeeds
spec:
  # ...
```

**Controller Logic:**
1. When job phase = Succeeded, update annotation:
   ```yaml
   mlops.dev/wave-complete: "3"
   mlops.dev/completion-time: "2026-01-24T22:30:00Z"
   ```

2. When job phase = Failed, update annotation:
   ```yaml
   mlops.dev/wave-failed: "3"
   mlops.dev/failure-time: "2026-01-24T22:30:00Z"
   ```

3. ArgoCD sync hooks can check for `mlops.dev/wave-complete` before proceeding

**Integration with ArgoCD:**
Users can create PreSync hooks that wait for wave completion:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: wait-for-notebook-wave3
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/sync-wave: "4"
spec:
  template:
    spec:
      containers:
      - name: wait
        image: bitnami/kubectl:latest
        command:
        - /bin/bash
        - -c
        - |
          until kubectl get notebookvalidationjob predictive-analytics-kserve-validation \
            -o jsonpath='{.metadata.annotations.mlops\.dev/wave-complete}' | grep -q "3"; do
            echo "Waiting for notebook validation wave 3 to complete..."
            sleep 5
          done
          echo "Wave 3 complete, proceeding with wave 4"
      restartPolicy: Never
```

**Benefit:**
- Guaranteed ordering: notebooks → models → InferenceServices
- Prevents race conditions
- GitOps-native dependency management
- Prevents deploying InferenceServices when models aren't ready

**Implementation Complexity:** **LOW-MEDIUM** (annotation management)
**Estimated Effort:** 1-2 weeks

---

### Feature 4: ArgoCD Application Status Integration ⭐ **MEDIUM PRIORITY**

**What:**
Operator publishes aggregated status to ArgoCD Application annotations

**Why:**
Centralized visibility of notebook validation state

**Implementation:**
Controller updates parent Application with aggregated status:

```yaml
# Controller updates parent Application
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
        "failedJobs": [
          "predictive-analytics-kserve-validation",
          "multi-cluster-healing-validation"
        ]
      }
    mlops.dev/notebook-summary: "30/32 succeeded (93.75% success rate)"
```

**Controller Logic:**
```go
func (r *NotebookValidationJobReconciler) updateApplicationStatus(ctx context.Context, namespace string) error {
    // List all NotebookValidationJobs in namespace
    var jobs mlopsv1alpha1.NotebookValidationJobList
    if err := r.List(ctx, &jobs, client.InNamespace(namespace)); err != nil {
        return err
    }

    // Aggregate status
    status := aggregateStatus(jobs.Items)

    // Find parent ArgoCD Application (look for app.kubernetes.io/part-of label)
    appName := findParentApplication(jobs.Items)
    if appName == "" {
        return nil // No parent Application found
    }

    // Update Application annotation
    app := &unstructured.Unstructured{}
    app.SetGroupVersionKind(schema.GroupVersionKind{
        Group:   "argoproj.io",
        Version: "v1alpha1",
        Kind:    "Application",
    })

    if err := r.Get(ctx, client.ObjectKey{Name: appName, Namespace: "argocd"}, app); err != nil {
        return client.IgnoreNotFound(err)
    }

    annotations := app.GetAnnotations()
    if annotations == nil {
        annotations = make(map[string]string)
    }

    statusJSON, _ := json.Marshal(status)
    annotations["mlops.dev/notebook-status"] = string(statusJSON)
    annotations["mlops.dev/notebook-summary"] = status.Summary()
    app.SetAnnotations(annotations)

    return r.Update(ctx, app)
}
```

**Update Triggers:**
- After each job status change
- Debounced to avoid excessive updates (max 1 update per minute)
- Background reconciliation every 5 minutes

**Benefit:**
- ArgoCD UI shows aggregated notebook status
- No need to run separate kubectl commands
- Better UX for operators
- Single source of truth

**Implementation Complexity:** **MEDIUM**
**Estimated Effort:** 2 weeks

---

### Feature 5: ArgoCD Notification Hooks ⭐ **LOW PRIORITY**

**What:**
Trigger ArgoCD notifications on job completion

**Why:**
Alert operators when model training fails

**Implementation:**
Operator creates Kubernetes Events that ArgoCD can consume:

```yaml
# Controller creates Event when job fails
apiVersion: v1
kind: Event
metadata:
  name: notebook-validation-failed-predictive-analytics
  namespace: self-healing-platform
  labels:
    app.kubernetes.io/part-of: self-healing-platform
    mlops.dev/notification-type: validation-failure
involvedObject:
  apiVersion: mlops.mlops.dev/v1alpha1
  kind: NotebookValidationJob
  name: predictive-analytics-kserve-validation
  namespace: self-healing-platform
type: Warning
reason: ValidationFailed
message: |
  NotebookValidationJob 'predictive-analytics-kserve-validation' failed:
  ModuleNotFoundError: No module named 'kserve_wrapper'

  Retry count: 3/3 (exhausted)
  Duration: 41 seconds
  Phase: Failed

  Suggested action: Check notebook cell execution for model serialization issues
  Logs: kubectl logs predictive-analytics-kserve-validation-pod
```

**ArgoCD Notification Integration:**
Users can configure ArgoCD notifications to trigger on these events:

```yaml
# argocd-notifications-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  trigger.notebook-validation-failed: |
    - when: app.status.health.status == 'Degraded'
      oncePer: app.status.sync.revision
      send: [notebook-failure-slack, notebook-failure-email]

  template.notebook-failure-slack: |
    message: |
      :warning: Notebook Validation Failed in {{.app.metadata.name}}

      Check failed notebook validations with:
      `kubectl get notebookvalidationjob -n {{.app.spec.destination.namespace}}`
```

**Benefit:**
- Proactive alerts
- Faster incident response
- Integration with existing ArgoCD notification channels (Slack, email, PagerDuty, etc.)

**Implementation Complexity:** **LOW** (uses existing Event API)
**Estimated Effort:** 1 week

---

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2) - **Immediate Priority**
**Goal:** Add basic ArgoCD awareness

**Tasks:**
1. ✅ Document ArgoCD health assessment configuration (Feature 1)
2. ✅ Create ADR-048
3. ✅ Add health check examples to operator documentation
4. Test health checks with existing NotebookValidationJobs in openshift-aiops-platform
5. Update operator README with ArgoCD integration section
6. Create example manifests for common use cases

**Deliverable:** ArgoCD can display NotebookValidationJob health

---

### Phase 2: Resource Triggering (Weeks 3-5) - **High Priority**
**Goal:** Implement post-success hooks (Feature 2)

**Tasks:**
1. Design annotation format for `mlops.dev/on-success-trigger`
2. Implement controller logic to parse and execute triggers
3. Add support for "restart", "sync", "refresh" actions
4. Add RBAC permissions for cross-resource management
5. Write unit tests for trigger parsing
6. Write e2e tests for InferenceService restart
7. Create ADR for resource triggering patterns
8. Document supported actions and examples

**Deliverable:** InferenceServices auto-restart after model training

---

### Phase 3: Sync Wave Integration (Weeks 6-7) - **Medium Priority**
**Goal:** Coordinate with ArgoCD sync waves (Feature 3)

**Tasks:**
1. Implement sync-wave awareness in controller
2. Add `wave-complete` and `wave-failed` annotation on job completion
3. Document wave blocking strategy with PreSync hooks
4. Create example ArgoCD Application with sync waves
5. Test with multi-wave Application in openshift-aiops-platform
6. Update ADR with sync wave patterns

**Deliverable:** Notebooks block subsequent waves until completion

---

### Phase 4: Observability (Weeks 8-10) - **Medium Priority**
**Goal:** Improve visibility in ArgoCD UI (Features 4 & 5)

**Tasks:**
1. Implement Application status aggregation (Feature 4)
2. Add debounced update logic
3. Add Prometheus metrics for ArgoCD scraping
4. Implement notification event creation (Feature 5)
5. Create notification templates for common use cases
6. Update documentation with screenshots
7. Create troubleshooting guide

**Deliverable:** Full observability in ArgoCD UI with notifications

---

## Example: End-to-End Workflow

### Before (Current State)
```yaml
# Wave 3: Notebook runs, trains model
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: predictive-analytics-kserve-validation
  annotations:
    argocd.argoproj.io/sync-wave: "3"
spec:
  notebook:
    git:
      url: https://github.com/example/models
      ref: main
    path: notebooks/predictive-analytics.ipynb
  # ... trains model to /mnt/models/predictive-analytics/model.pkl

---
# Wave 4: InferenceService deploys (races with notebook!)
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: predictive-analytics
  annotations:
    argocd.argoproj.io/sync-wave: "4"
spec:
  predictor:
    containers:
    - name: kserve-container
      volumeMounts:
      - name: model-storage
        mountPath: /mnt/models
```

**Problems:**
- ❌ InferenceService may deploy before model exists
- ❌ If model changes, InferenceService doesn't reload
- ❌ No visibility in ArgoCD about notebook status
- ❌ Manual pod deletion required after model training

---

### After (With ArgoCD Integration)
```yaml
# Wave 3: Notebook runs, trains model
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: predictive-analytics-kserve-validation
  annotations:
    argocd.argoproj.io/sync-wave: "3"
    mlops.dev/block-wave: "4"  # ← Feature 3: Block wave 4 until success
    mlops.dev/on-success-trigger: |  # ← Feature 2: Auto-restart InferenceService
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: predictive-analytics
        namespace: self-healing-platform
        action: restart
spec:
  notebook:
    git:
      url: https://github.com/example/models
      ref: main
    path: notebooks/predictive-analytics.ipynb
  # ... trains model to /mnt/models/predictive-analytics/model.pkl

# Operator automatically sets when job succeeds:
# mlops.dev/wave-complete: "3"
# mlops.dev/completion-time: "2026-01-24T22:30:00Z"

---
# Wave 4: Wait for notebook completion
apiVersion: batch/v1
kind: Job
metadata:
  name: wait-for-models
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/sync-wave: "4"
spec:
  template:
    spec:
      containers:
      - name: wait
        image: bitnami/kubectl:latest
        command: ["/bin/bash", "-c"]
        args:
        - |
          until kubectl get notebookvalidationjob predictive-analytics-kserve-validation \
            -o jsonpath='{.metadata.annotations.mlops\.dev/wave-complete}' | grep -q "3"; do
            echo "Waiting for notebook validation to complete..."
            sleep 5
          done
      restartPolicy: Never

---
# Wave 4: InferenceService deploys ONLY AFTER notebook succeeds
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: predictive-analytics
  annotations:
    argocd.argoproj.io/sync-wave: "4"
spec:
  predictor:
    containers:
    - name: kserve-container
      volumeMounts:
      - name: model-storage
        mountPath: /mnt/models
```

**Benefits:**
- ✅ InferenceService waits for model to exist (Feature 3)
- ✅ InferenceService auto-restarts when model updates (Feature 2)
- ✅ ArgoCD UI shows notebook validation status (Feature 1)
- ✅ Application health includes notebook state (Feature 4)
- ✅ Operators get notified if training fails (Feature 5)
- ✅ No manual intervention required
- ✅ Fully declarative GitOps workflow

---

## Consequences

### Positive
1. **Complete GitOps Workflow**: Notebooks become first-class citizens in ArgoCD
2. **Automatic Model Updates**: No manual intervention for model reloading
3. **Better Observability**: Single pane of glass in ArgoCD UI
4. **Race Condition Prevention**: Sync waves guarantee ordering
5. **Production Ready**: Aligns with enterprise GitOps practices
6. **Platform Agnostic**: Works across OpenShift, vanilla Kubernetes with ArgoCD
7. **Incremental Adoption**: Features can be adopted independently

### Negative
1. **Increased Complexity**: More annotation-based configuration
2. **ArgoCD Dependency**: Features only work with ArgoCD (not Flux, etc.)
3. **RBAC Requirements**: Controller needs permissions to update Applications, delete pods
4. **Testing Burden**: Need e2e tests with ArgoCD environment
5. **Documentation Overhead**: Multiple integration patterns to document

### Neutral
1. **Backward Compatible**: All features are opt-in via annotations
2. **Feature Flags**: Can gate features behind configuration
3. **Migration Path**: Existing deployments unaffected

---

## Related ADRs

- **ADR-007**: Distribution and Catalog Strategy - Mentions ArgoCD integration as strategic goal
- **ADR-010**: Observability and Monitoring Strategy - Status reporting foundation
- **ADR-011**: Error Handling and Retry Strategy - Failure notifications build on retry logic
- **ADR-042**: ArgoCD Deployment Lessons Learned (openshift-aiops-platform) - Real-world issues driving this

---

## References

- ArgoCD Health Assessment: https://argo-cd.readthedocs.io/en/stable/operator-manual/health/
- ArgoCD Sync Waves: https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/
- ArgoCD Notifications: https://argocd-notifications.readthedocs.io/
- KServe InferenceService: https://kserve.github.io/website/latest/
- NotebookValidationJob CRD: `api/v1alpha1/notebookvalidationjob_types.go`

---

## Implementation Notes

### RBAC Additions Required

For Feature 2 (Resource Triggering), the operator ClusterRole needs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: jupyter-notebook-validator-operator-manager
rules:
# Existing rules...

# Feature 2: Resource triggering
- apiGroups: ["serving.kserve.io"]
  resources: ["inferenceservices"]
  verbs: ["get", "list", "patch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["list", "delete"]

# Feature 4: Application status updates
- apiGroups: ["argoproj.io"]
  resources: ["applications"]
  verbs: ["get", "list", "patch"]
```

### Metrics to Add

For Prometheus/ArgoCD integration:

```go
var (
    notebookValidationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "notebook_validation_duration_seconds",
            Help: "Duration of notebook validation jobs",
        },
        []string{"job_name", "phase"},
    )

    notebookValidationTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "notebook_validation_total",
            Help: "Total number of notebook validations",
        },
        []string{"phase"},
    )

    argocdTriggersTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "argocd_triggers_total",
            Help: "Total number of ArgoCD-related triggers",
        },
        []string{"action", "status"},
    )
)
```

---

## Next Steps

1. ✅ **Create ADR-048** (this document)
2. ✅ **Create GitHub issue** with feature request
3. **Prototype Feature 1** - Health assessment documentation
4. **Test in openshift-aiops-platform** - Validate health checks work
5. **Community Feedback** - Get input from operator maintainers
6. **Phase 1 Implementation** - Begin coding if approved

---

## Success Criteria

After implementing these features:

✅ **Visibility:**
- ArgoCD Application health reflects notebook validation status
- Operators can see which notebooks are running/failed in ArgoCD UI
- No need to run kubectl commands separately

✅ **Automation:**
- InferenceServices automatically restart when new models are trained
- No manual intervention required
- Models stay current without drift

✅ **Reliability:**
- Notebooks complete before InferenceServices deploy
- No race conditions
- Sync waves guarantee ordering

✅ **Observability:**
- Failed notebook validations trigger alerts
- Application status shows aggregated notebook health
- Single pane of glass for GitOps + ML workflows
