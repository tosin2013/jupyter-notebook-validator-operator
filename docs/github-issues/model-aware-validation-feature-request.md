# GitHub Issue: Feature Request - Model-Aware Validation for ML Workflows

**Issue Type:** Feature Request / Enhancement
**Priority:** High
**Labels:** `enhancement`, `model-validation`, `kserve`, `ml-ops`, `help wanted`

---

## Title

**Feature Request: Model-Aware Validation for ML/AI Workflows**

---

## Summary

Add optional model-aware validation capabilities to enable notebooks to validate against deployed ML models (KServe, OpenShift AI, vLLM, etc.). This addresses the gap where notebooks execute successfully in isolation but fail when interacting with production model serving infrastructure.

**Status:** Fully designed in [ADR-020](../adrs/020-model-aware-validation-strategy.md)
**Priority:** High
**Category:** Validation Strategy, Platform Integration

---

## Problem Statement

Currently, the jupyter-notebook-validator-operator validates notebook execution in isolation without awareness of model serving infrastructure. This creates several issues:

### Problem 1: Environment Readiness Gaps

**Scenario:**
1. Notebook expects KServe InferenceService to exist
2. Notebook executes before InferenceService is deployed
3. API calls to model endpoint fail
4. Notebook validation fails, but unclear why

**Impact:**
- Unclear error messages ("Connection refused" vs "Model not ready")
- No pre-flight checks for platform availability
- Cannot validate that cluster has required model serving platforms

---

### Problem 2: Model Compatibility Issues

**Scenario:**
1. Notebook developed against KServe v0.10 API
2. Cluster running KServe v0.11 with breaking changes
3. Notebook executes but predictions fail silently
4. No automated detection of API version mismatch

**Impact:**
- Silent failures in production
- No validation that notebooks can communicate with deployed models
- Manual testing required for each platform version

---

### Problem 3: Prediction Consistency

**Scenario:**
1. Model updated to v2.0
2. Notebook still expects v1.0 output format
3. Predictions succeed but format changed
4. Downstream processing breaks

**Impact:**
- No automated testing of prediction outputs
- Cannot validate that predictions match expected format
- Drift between model versions and notebook expectations

---

### Problem 4: Resource Health

**Scenario:**
1. InferenceService exists but predictor pod is failing
2. Notebook sends requests to unhealthy model
3. Requests timeout or fail
4. No visibility into model health

**Impact:**
- No health checks for models notebooks depend on
- Cannot detect when models are degraded before running notebooks
- Poor debugging experience

---

## Proposed Solution

Implement **optional model-aware validation** with a **two-phase validation strategy** and **plugin-based architecture** supporting multiple model serving platforms.

### Feature Overview

#### 1. Two-Phase Validation Strategy

**Phase 1: Clean Environment Validation (Pre-Deployment)**
- Validates platform readiness before any models are deployed
- Checks:
  - ✅ Model serving platform availability (KServe, OpenShift AI, etc.)
  - ✅ Required CRDs installed
  - ✅ RBAC permissions for model access
  - ✅ Required libraries and dependencies
  - ✅ Network connectivity to model serving endpoints

**Phase 2: Existing Environment Validation (Post-Deployment)**
- Validates against currently running models
- Checks:
  - ✅ Model availability and health
  - ✅ Prediction consistency (actual vs. expected outputs)
  - ✅ Model version alignment
  - ✅ Resource integrity (CPU, memory, GPU)
  - ✅ API compatibility

#### 2. Platform Support

**Tier 1: Built-In (Fully Supported)**
- **KServe** - Standard Kubernetes model serving (primary)
- **OpenShift AI** - Red Hat's enterprise AI platform

**Tier 2: Community-Documented (Documentation + Examples)**
- **vLLM** - LLM-focused serving
- **TorchServe** - PyTorch model serving
- **TensorFlow Serving** - TensorFlow model serving
- **Triton Inference Server** - NVIDIA multi-framework serving
- **Ray Serve** - Distributed model serving
- **Seldon Core** - Advanced ML deployments
- **BentoML** - Model packaging and serving

---

## CRD Design

Add optional `modelValidation` field to `NotebookValidationJobSpec`:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-fraud-detection
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: model-inference.ipynb

  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa

  # NEW: Optional model validation
  modelValidation:
    enabled: true
    platform: kserve  # or openshift-ai, vllm, torchserve, etc.
    phase: both  # clean, existing, or both

    # Models this notebook depends on
    targetModels:
      - fraud-detection-model
      - risk-scoring-model

    # Optional: Validate prediction outputs
    predictionValidation:
      enabled: true
      testData: |
        {"instances": [[1.0, 2.0, 3.0, 4.0]]}
      expectedOutput: |
        {"predictions": [[0.95, 0.05]]}
      tolerance: 0.01
```

---

## Use Cases

### Use Case 1: Pre-Deployment Platform Check

**Scenario:** Before deploying notebooks to production, validate that the cluster has KServe installed.

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: platform-readiness-check
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks
      ref: main
      path: model-inference.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
  modelValidation:
    enabled: true
    platform: kserve
    phase: clean  # Only check platform, not models
```

**Expected Behavior:**
- ✅ Validates KServe CRDs are installed
- ✅ Validates RBAC permissions exist
- ✅ Validates network connectivity to KServe endpoints
- ❌ Fails with clear message if platform not ready

---

### Use Case 2: Model Health Check

**Scenario:** Validate that deployed models are healthy before running inference notebook.

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: model-health-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks
      ref: main
      path: batch-inference.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
  modelValidation:
    enabled: true
    platform: kserve
    phase: existing  # Check existing models
    targetModels:
      - fraud-detection-model
```

**Expected Behavior:**
- ✅ Checks InferenceService `fraud-detection-model` exists
- ✅ Validates predictor pods are running and healthy
- ✅ Validates model is ready to serve requests
- ❌ Fails if model is in `Progressing` or `Failed` state

---

### Use Case 3: Prediction Consistency Testing

**Scenario:** Validate that model predictions match expected format and values.

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: prediction-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks
      ref: main
      path: model-testing.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
  modelValidation:
    enabled: true
    platform: kserve
    phase: existing
    targetModels:
      - risk-scoring-model
    predictionValidation:
      enabled: true
      testData: |
        {
          "instances": [
            {"age": 35, "income": 50000, "debt": 10000}
          ]
        }
      expectedOutput: |
        {
          "predictions": [0.75]
        }
      tolerance: 0.05  # Allow 5% variance
```

**Expected Behavior:**
- ✅ Sends test data to model
- ✅ Compares actual prediction with expected output
- ✅ Allows 5% tolerance for floating-point variance
- ❌ Fails if prediction format changes or values drift

---

## Implementation Plan

See [ADR-020](../adrs/020-model-aware-validation-strategy.md) for complete details.

### High-Level Milestones (12 weeks)

**Weeks 1-2: CRD Design and API Updates**
- [ ] Add `modelValidation` field to `NotebookValidationJobSpec`
- [ ] Define platform detection interface
- [ ] Update OpenAPI schema and CRD generation
- [ ] Write ADR implementation details

**Weeks 3-4: Platform Detection and Controller Logic**
- [ ] Implement automatic platform detection
- [ ] Add platform-specific validators
- [ ] Create validation pod RBAC templates
- [ ] Update controller reconciliation logic

**Weeks 5-6: KServe Integration**
- [ ] Implement KServe platform detector
- [ ] Add InferenceService health checks
- [ ] Implement prediction validation
- [ ] Create KServe examples

**Weeks 7-8: OpenShift AI Integration**
- [ ] Implement OpenShift AI platform detector
- [ ] Add model registry integration
- [ ] Create OpenShift AI examples
- [ ] Test with RHOAI environments

**Weeks 9-10: Community Platform Documentation**
- [ ] Document vLLM integration pattern
- [ ] Document TorchServe integration pattern
- [ ] Document TensorFlow Serving integration pattern
- [ ] Create contribution guide for new platforms

**Weeks 11-12: Testing and Release**
- [ ] E2E tests with KServe and OpenShift AI
- [ ] Update user documentation
- [ ] Create migration guide
- [ ] Release notes and announcement

---

## RBAC Requirements

Validation pods need additional permissions to access model serving resources:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-validator
  namespace: ml-platform
rules:
  # KServe permissions
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices/status"]
    verbs: ["get"]

  # OpenShift AI permissions (if using RHOAI)
  - apiGroups: ["modelregistry.opendatahub.io"]
    resources: ["modelregistries"]
    verbs: ["get", "list"]
```

---

## Benefits

### For Data Scientists
- ✅ Confidence that notebooks will work with deployed models
- ✅ Early detection of model compatibility issues
- ✅ Clear error messages when models aren't ready
- ✅ Automated prediction consistency testing

### For ML Engineers
- ✅ Automated validation of model integration
- ✅ Platform readiness checks before deployment
- ✅ Reduced debugging time for model serving issues
- ✅ Version compatibility validation

### For Platform Teams
- ✅ Visibility into notebook-model dependencies
- ✅ Clear RBAC requirements and templates
- ✅ Support for multiple model serving platforms
- ✅ Community contribution path

### For DevOps Teams
- ✅ Deployment readiness checks
- ✅ Reduced production failures
- ✅ Better observability into ML workflows
- ✅ GitOps-friendly validation

---

## Consequences

### Positive
- ✅ Improved reliability (catch integration issues early)
- ✅ Platform flexibility (support 8+ model serving platforms)
- ✅ Backward compatibility (optional, opt-in feature)
- ✅ Better developer experience (automated checks)
- ✅ Community growth (extensible architecture)

### Negative
- ⚠️ Increased complexity (additional CRD fields, controller logic)
- ⚠️ RBAC management (additional permissions required)
- ⚠️ Maintenance burden (multiple platform integrations)
- ⚠️ Testing overhead (need environments for each platform)

### Mitigation
- Use plugin architecture to isolate platform-specific logic
- Provide clear RBAC templates and examples
- Focus on built-in platforms; community maintains others
- Prioritize KServe/OpenShift AI; community tests others

---

## Alternatives Considered

### Alternative 1: No Model Awareness (Status Quo)
**Rejected:** Leaves gap in validation coverage; notebooks may fail in production

### Alternative 2: Mandatory Model Validation
**Rejected:** Too restrictive; not all notebooks interact with models

### Alternative 3: Only KServe Support
**Rejected:** Limits platform choice; doesn't support OpenShift AI or community platforms

### Alternative 4: External Validation Service
**Rejected:** Adds operational complexity; breaks Kubernetes-native pattern

### Alternative 5: Webhook-Based Validation
**Rejected:** Doesn't fit operator pattern; harder to debug

---

## Acceptance Criteria

- [ ] Optional `modelValidation` field added to CRD
- [ ] Automatic platform detection works for KServe and OpenShift AI
- [ ] Platform readiness checks validate CRDs, RBAC, connectivity
- [ ] Model health checks validate InferenceService status
- [ ] Prediction validation compares actual vs. expected outputs
- [ ] RBAC templates provided for common scenarios
- [ ] Documentation covers 7+ model serving platforms
- [ ] E2E tests pass for KServe and OpenShift AI
- [ ] Migration guide for existing users
- [ ] Backward compatible (no breaking changes)

---

## References

- **ADR-020**: [Model-Aware Validation Strategy](../adrs/020-model-aware-validation-strategy.md)
- **KServe Docs**: https://kserve.github.io/website/
- **OpenShift AI Docs**: https://docs.redhat.com/en/documentation/red_hat_openshift_ai_self-managed/
- **vLLM Docs**: https://docs.vllm.ai/
- **Model Serving Comparison**: https://www.hopsworks.ai/dictionary/model-serving

---

## Community Input Requested

- Which model serving platforms should be prioritized?
- What prediction validation patterns are most useful?
- Should we support custom validators via plugins?
- Interest in contributing platform integrations?

---

## Labels

Please add:
- `enhancement`
- `model-validation`
- `kserve`
- `ml-ops`
- `help wanted`
- `good first issue` (for documentation contributions)
- `priority: high`

---

## CC

@tosin2013 (operator maintainer)

---

**Thank you for considering this feature request!** This would significantly improve the operator's value for production ML/AI workflows.
