# ArgoCD Integration Guide

This guide covers integrating the jupyter-notebook-validator-operator with ArgoCD for GitOps workflows. It implements features described in [ADR-048: ArgoCD Integration Strategy](./adrs/048-argocd-integration-strategy.md).

## Table of Contents

- [Overview](#overview)
- [Feature 1: Health Assessment (Available Now)](#feature-1-health-assessment-available-now)
- [Feature 2: Post-Success Resource Hooks (Planned)](#feature-2-post-success-resource-hooks-planned)
- [Feature 3: Sync Wave Coordination (Planned)](#feature-3-sync-wave-coordination-planned)
- [End-to-End Examples](#end-to-end-examples)
- [Troubleshooting](#troubleshooting)

---

## Overview

The jupyter-notebook-validator-operator manages the validation and execution of Jupyter Notebooks in Kubernetes. When integrated with ArgoCD, it enables:

1. **Health Visibility**: See notebook validation status in ArgoCD UI
2. **Automatic Coordination**: Trigger downstream resource updates when notebooks complete
3. **Dependency Ordering**: Ensure notebooks complete before deploying model serving infrastructure
4. **Centralized Monitoring**: Single pane of glass for GitOps + ML workflows

### Why ArgoCD Integration Matters

**Problem:** Model training notebooks complete successfully, but InferenceServices don't know when to reload new models.

**Solution:** ArgoCD integration enables automatic coordination between notebook execution (training) and model serving (inference).

**Benefits:**
- ✅ No manual intervention required
- ✅ Fully declarative GitOps workflow
- ✅ Prevents race conditions between training and serving
- ✅ Better observability in ArgoCD UI

---

## Feature 1: Health Assessment (Available Now)

ArgoCD can display custom health status for NotebookValidationJob resources using Lua-based health checks.

### Status Mapping

| NotebookValidationJob Phase | ArgoCD Health Status | Description |
|-----------------------------|---------------------|-------------|
| `Succeeded` | `Healthy` | Notebook validation completed successfully |
| `Failed` | `Degraded` | Notebook validation failed after retries |
| `ValidationRunning` | `Progressing` | Actively executing notebook cells |
| `Building` | `Progressing` | Building validation environment |
| `BuildComplete` | `Progressing` | Build complete, preparing to run |
| `Initializing` | `Progressing` | Initializing resources |
| `Pending` | `Progressing` | Waiting to start |

### Installation

#### Step 1: Configure ArgoCD Health Check

Add the following to your ArgoCD ConfigMap (`argocd-cm` in the `argocd` namespace):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
data:
  # Health assessment for NotebookValidationJob
  resource.customizations.health.mlops.mlops.dev_NotebookValidationJob: |
    hs = {}

    -- Handle missing status
    if obj.status == nil or obj.status.phase == nil then
      hs.status = "Progressing"
      hs.message = "Initializing notebook validation"
      return hs
    end

    -- Map phase to ArgoCD health status
    local phase = obj.status.phase

    if phase == "Succeeded" then
      hs.status = "Healthy"
      hs.message = "Notebook validation succeeded"

      -- Add completion timestamp if available
      if obj.status.completionTime ~= nil then
        hs.message = hs.message .. " at " .. obj.status.completionTime
      end

    elseif phase == "Failed" then
      hs.status = "Degraded"
      hs.message = obj.status.message or "Notebook validation failed"

      -- Show retry information
      if obj.status.retryCount ~= nil then
        hs.message = hs.message .. " (retry " .. obj.status.retryCount .. "/3)"
      end

    elseif phase == "ValidationRunning" then
      hs.status = "Progressing"
      hs.message = "Validating notebook"

      -- Show pod name if available
      if obj.status.validationPodName ~= nil and obj.status.validationPodName ~= "" then
        hs.message = hs.message .. " (pod: " .. obj.status.validationPodName .. ")"
      end

    elseif phase == "Building" then
      hs.status = "Progressing"
      hs.message = "Building validation environment"

    elseif phase == "BuildComplete" then
      hs.status = "Progressing"
      hs.message = "Build complete, preparing validation"

    elseif phase == "Initializing" or phase == "Pending" or phase == "Running" then
      hs.status = "Progressing"
      hs.message = "Initializing notebook validation"

    else
      -- Unknown phase
      hs.status = "Progressing"
      hs.message = "Phase: " .. phase
    end

    return hs
```

#### Step 2: Apply the ConfigMap

```bash
kubectl apply -f argocd-cm.yaml
```

#### Step 3: Restart ArgoCD Server (if needed)

Some ArgoCD versions require a restart to pick up ConfigMap changes:

```bash
kubectl rollout restart deployment argocd-server -n argocd
```

#### Step 4: Verify Health Check

1. Open ArgoCD UI
2. Navigate to your Application containing NotebookValidationJob resources
3. You should see health status displayed for each NotebookValidationJob

**Expected Output:**
- **Healthy** (green): Succeeded notebooks
- **Degraded** (red): Failed notebooks
- **Progressing** (blue): Running/building notebooks

---

### Example: NotebookValidationJob in ArgoCD

#### Manifest

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: model-training-job
  namespace: ml-platform
  labels:
    app.kubernetes.io/part-of: ml-platform
spec:
  notebook:
    git:
      url: https://github.com/example/ml-models
      ref: main
    path: notebooks/train-model.ipynb
  podConfig:
    containerImage: quay.io/ml-platform/notebook-runtime:latest
    resources:
      requests:
        memory: "2Gi"
        cpu: "1"
      limits:
        memory: "4Gi"
        cpu: "2"
```

#### ArgoCD UI Display

**When Running:**
```
NotebookValidationJob/model-training-job
Status: Progressing
Message: Validating notebook (pod: model-training-job-validation-abc123)
```

**When Succeeded:**
```
NotebookValidationJob/model-training-job
Status: Healthy
Message: Notebook validation succeeded at 2026-01-24T22:30:00Z
```

**When Failed:**
```
NotebookValidationJob/model-training-job
Status: Degraded
Message: Cell execution failed: ModuleNotFoundError (retry 3/3)
```

---

### Advanced Health Check (With Cell-Level Details)

For more detailed health information, you can extend the health check to show cell execution progress:

```yaml
resource.customizations.health.mlops.mlops.dev_NotebookValidationJob: |
  hs = {}

  if obj.status == nil or obj.status.phase == nil then
    hs.status = "Progressing"
    hs.message = "Initializing"
    return hs
  end

  local phase = obj.status.phase

  if phase == "Succeeded" then
    hs.status = "Healthy"

    -- Show cell execution summary
    if obj.status.results ~= nil then
      local totalCells = #obj.status.results
      hs.message = "Succeeded - Executed " .. totalCells .. " cells"
    else
      hs.message = "Notebook validation succeeded"
    end

  elseif phase == "Failed" then
    hs.status = "Degraded"

    -- Find which cell failed
    if obj.status.results ~= nil then
      for i, result in ipairs(obj.status.results) do
        if result.status == "error" then
          hs.message = "Failed at cell " .. result.cellIndex .. ": " .. (result.errorType or "Error")
          return hs
        end
      end
    end

    hs.message = obj.status.message or "Validation failed"

  elseif phase == "ValidationRunning" then
    hs.status = "Progressing"

    -- Show progress through cells
    if obj.status.results ~= nil then
      local completedCells = 0
      for _, result in ipairs(obj.status.results) do
        if result.status == "success" then
          completedCells = completedCells + 1
        end
      end
      hs.message = "Executing notebook (" .. completedCells .. " cells completed)"
    else
      hs.message = "Validating notebook"
    end

  else
    hs.status = "Progressing"
    hs.message = "Phase: " .. phase
  end

  return hs
```

---

## Feature 2: Post-Success Resource Hooks (Planned)

> **Status:** Planned for Phase 2 implementation
>
> **Tracking:** See [GitHub Issue: ArgoCD Integration](./github-issues/argocd-integration-feature-request.md)

### Proposed Usage

When implemented, you'll be able to automatically trigger resource updates when notebooks succeed:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: predictive-analytics-training
  annotations:
    # Automatically restart InferenceService when model training completes
    mlops.dev/on-success-trigger: |
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: predictive-analytics
        namespace: ml-platform
        action: restart
spec:
  # ... notebook spec
```

**Supported Actions (Planned):**
- `restart`: Delete pods to force reload (for KServe, Deployments)
- `sync`: Trigger ArgoCD Application sync
- `refresh`: Add annotation to force resource refresh

---

## Feature 3: Sync Wave Coordination (Planned)

> **Status:** Planned for Phase 3 implementation

### Proposed Usage

Ensure notebooks complete before dependent resources deploy:

```yaml
# Wave 3: Train model
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: model-training
  annotations:
    argocd.argoproj.io/sync-wave: "3"
    mlops.dev/block-wave: "4"  # Block wave 4 until succeeded
spec:
  # ... training notebook

---
# Wave 4: Deploy InferenceService (waits for wave 3)
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: trained-model
  annotations:
    argocd.argoproj.io/sync-wave: "4"
spec:
  # ... inference service
```

---

## End-to-End Examples

### Example 1: Simple Model Training Pipeline

This example shows a basic GitOps workflow for ML model training:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ml-training-pipeline
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/example/ml-pipeline
    targetRevision: main
    path: manifests
  destination:
    server: https://kubernetes.default.svc
    namespace: ml-platform
  syncPolicy:
    automated:
      prune: true
      selfHeal: true

---
# NotebookValidationJob for training
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: fraud-detection-training
  namespace: ml-platform
  labels:
    app.kubernetes.io/part-of: ml-training-pipeline
spec:
  notebook:
    git:
      url: https://github.com/example/ml-models
      ref: main
    path: notebooks/fraud-detection/train.ipynb
  podConfig:
    containerImage: quay.io/ml-platform/sklearn-notebook:latest
    resources:
      requests:
        memory: "4Gi"
        cpu: "2"
    volumeMounts:
    - name: model-storage
      mountPath: /mnt/models
  volumes:
  - name: model-storage
    persistentVolumeClaim:
      claimName: ml-model-storage
```

**ArgoCD Behavior:**
1. Application syncs → creates NotebookValidationJob
2. Operator executes notebook → trains model → saves to PVC
3. ArgoCD shows **Progressing** while notebook runs
4. ArgoCD shows **Healthy** when training succeeds
5. ArgoCD shows **Degraded** if training fails

---

### Example 2: Multi-Stage ML Pipeline with Sync Waves

This example demonstrates coordinated deployment of training → validation → serving:

```yaml
# Stage 1 (Wave 0): Infrastructure
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ml-model-storage
  annotations:
    argocd.argoproj.io/sync-wave: "0"
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 50Gi

---
# Stage 2 (Wave 1): Data preprocessing
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: data-preprocessing
  annotations:
    argocd.argoproj.io/sync-wave: "1"
spec:
  notebook:
    git:
      url: https://github.com/example/ml-pipeline
      ref: main
    path: notebooks/01-preprocess-data.ipynb
  podConfig:
    containerImage: quay.io/ml/pandas-notebook:latest

---
# Stage 3 (Wave 2): Model training
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: model-training
  annotations:
    argocd.argoproj.io/sync-wave: "2"
spec:
  notebook:
    git:
      url: https://github.com/example/ml-pipeline
      ref: main
    path: notebooks/02-train-model.ipynb
  podConfig:
    containerImage: quay.io/ml/sklearn-notebook:latest
    volumeMounts:
    - name: model-storage
      mountPath: /mnt/models
  volumes:
  - name: model-storage
    persistentVolumeClaim:
      claimName: ml-model-storage

---
# Stage 4 (Wave 3): Model validation
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: model-validation
  annotations:
    argocd.argoproj.io/sync-wave: "3"
spec:
  notebook:
    git:
      url: https://github.com/example/ml-pipeline
      ref: main
    path: notebooks/03-validate-model.ipynb
  podConfig:
    containerImage: quay.io/ml/sklearn-notebook:latest
    volumeMounts:
    - name: model-storage
      mountPath: /mnt/models
  volumes:
  - name: model-storage
    persistentVolumeClaim:
      claimName: ml-model-storage

---
# Stage 5 (Wave 4): Deploy InferenceService
# NOTE: Currently deploys after wave 3, but doesn't wait for completion
# Feature 3 (sync wave coordination) will add waiting capability
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: fraud-detection-model
  annotations:
    argocd.argoproj.io/sync-wave: "4"
spec:
  predictor:
    sklearn:
      storageUri: pvc://ml-model-storage/fraud-detection
```

**Current Behavior:**
- Waves deploy sequentially
- ArgoCD shows health for each NotebookValidationJob
- **Limitation:** Wave 4 starts when wave 3 resources are *created*, not when notebooks *complete*

**Future with Feature 3:**
- Wave 4 waits for wave 3 NotebookValidationJobs to reach `Succeeded` phase
- No race conditions between training and serving

---

### Example 3: KServe Model Serving with Manual Reload

Until Feature 2 (auto-restart) is implemented, use this pattern for model updates:

```yaml
# NotebookValidationJob trains new model
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: retrain-model
  namespace: ml-platform
spec:
  notebook:
    git:
      url: https://github.com/example/models
      ref: main
    path: notebooks/retrain.ipynb
  podConfig:
    volumeMounts:
    - name: model-storage
      mountPath: /mnt/models
  volumes:
  - name: model-storage
    persistentVolumeClaim:
      claimName: kserve-model-storage

---
# InferenceService serves model
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: my-model
  namespace: ml-platform
spec:
  predictor:
    containers:
    - name: kserve-container
      image: my-model-server:latest
      volumeMounts:
      - name: model-storage
        mountPath: /mnt/models
  volumes:
  - name: model-storage
    persistentVolumeClaim:
      claimName: kserve-model-storage
```

**Manual Reload Process:**
1. Check notebook completion in ArgoCD UI (should show **Healthy**)
2. Delete InferenceService pods to reload new model:
   ```bash
   kubectl delete pods -n ml-platform -l serving.kserve.io/inferenceservice=my-model
   ```
3. Verify new model is loaded in predictor logs

**With Feature 2 (future):**
```yaml
metadata:
  annotations:
    mlops.dev/on-success-trigger: |
      - apiVersion: serving.kserve.io/v1beta1
        kind: InferenceService
        name: my-model
        namespace: ml-platform
        action: restart
```
→ Automatic reload, no manual steps required

---

## Troubleshooting

### Health Check Not Showing in ArgoCD

**Symptom:** NotebookValidationJob resources appear in Application, but no health status is displayed

**Diagnosis:**
1. Check ArgoCD ConfigMap exists:
   ```bash
   kubectl get configmap argocd-cm -n argocd
   ```

2. Verify health check is configured:
   ```bash
   kubectl get configmap argocd-cm -n argocd -o yaml | grep NotebookValidationJob
   ```

3. Check ArgoCD server logs for errors:
   ```bash
   kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server | grep health
   ```

**Solutions:**
- Ensure ConfigMap key is exactly: `resource.customizations.health.mlops.mlops.dev_NotebookValidationJob`
- Note the underscore (`_`) between group and kind
- Restart ArgoCD server after ConfigMap changes:
  ```bash
  kubectl rollout restart deployment argocd-server -n argocd
  ```

---

### Health Shows "Progressing" Instead of "Healthy"

**Symptom:** Notebook completed successfully, but ArgoCD still shows Progressing

**Diagnosis:**
```bash
kubectl get notebookvalidationjob <name> -o jsonpath='{.status.phase}'
```

**Possible Causes:**
1. **Phase not set correctly**: Check controller logs
   ```bash
   kubectl logs -n jupyter-notebook-validator-system deployment/jupyter-notebook-validator-operator-controller-manager
   ```

2. **Health check Lua error**: Check ArgoCD application controller logs
   ```bash
   kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller | grep -A 5 NotebookValidationJob
   ```

3. **Status not updated**: Verify status conditions:
   ```bash
   kubectl get notebookvalidationjob <name> -o yaml | grep -A 10 status:
   ```

---

### Application Shows "Degraded" But Notebooks Succeeded

**Symptom:** ArgoCD Application health is Degraded even though NotebookValidationJobs succeeded

**Diagnosis:**
Check for failed notebooks:
```bash
kubectl get notebookvalidationjob -n <namespace> -o json | \
  jq -r '.items[] | select(.status.phase=="Failed") | .metadata.name'
```

**Solutions:**
1. **Fix failed notebooks**: Check logs for the failed jobs
   ```bash
   kubectl logs <validation-pod-name>
   ```

2. **Delete failed jobs** (if they're old/irrelevant):
   ```bash
   kubectl delete notebookvalidationjob <failed-job-name>
   ```

3. **Retry failed jobs**: Delete and let ArgoCD recreate
   ```bash
   kubectl delete notebookvalidationjob <failed-job-name>
   # ArgoCD will recreate if syncPolicy.automated is enabled
   ```

---

### Sync Waves Not Waiting for Notebook Completion

**Symptom:** Wave 4 resources deploy before wave 3 notebooks finish executing

**Explanation:**
ArgoCD sync waves wait for resources to be *created* (from Kubernetes API perspective), not for operator-managed workloads to *complete*.

A NotebookValidationJob is "synced" as soon as the CR is created, even if the notebook is still running.

**Current Workaround:**
Use ArgoCD PreSync hooks to wait for completion:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: wait-for-notebook-completion
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/sync-wave: "4"
spec:
  template:
    spec:
      serviceAccountName: argocd-sync-waiter  # needs RBAC to read NotebookValidationJobs
      containers:
      - name: wait
        image: bitnami/kubectl:latest
        command: ["/bin/bash", "-c"]
        args:
        - |
          echo "Waiting for notebook validation to complete..."

          until kubectl get notebookvalidationjob model-training -n ml-platform \
            -o jsonpath='{.status.phase}' | grep -E "Succeeded|Failed"; do
            echo "Still running..."
            sleep 10
          done

          # Check if succeeded
          PHASE=$(kubectl get notebookvalidationjob model-training -n ml-platform \
            -o jsonpath='{.status.phase}')

          if [ "$PHASE" != "Succeeded" ]; then
            echo "Notebook validation failed!"
            exit 1
          fi

          echo "Notebook validation succeeded, proceeding with sync"
      restartPolicy: Never
  backoffLimit: 3
```

**Future with Feature 3:**
Operator will automatically set `mlops.dev/wave-complete` annotation when notebooks succeed, simplifying the wait logic.

---

## References

- **ADR-048**: [ArgoCD Integration Strategy](./adrs/048-argocd-integration-strategy.md)
- **GitHub Issue**: [Feature Request](./github-issues/argocd-integration-feature-request.md)
- **ArgoCD Docs**: [Health Assessment](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/)
- **ArgoCD Docs**: [Sync Waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/)
- **Operator Docs**: [End-to-End ML Workflow](./END_TO_END_ML_WORKFLOW.md)

---

## Contributing

Found issues with ArgoCD integration or have ideas for improvements?

1. Check existing [GitHub issues](./github-issues/)
2. Review [ADR-048](./adrs/048-argocd-integration-strategy.md) for planned features
3. Open a new issue with your findings

---

## Version History

- **2026-01-24**: Initial version with Feature 1 (Health Assessment) documentation
- **Future**: Feature 2 (Resource Hooks), Feature 3 (Sync Waves), Feature 4 (Application Status), Feature 5 (Notifications)
