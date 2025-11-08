# ADR-020: Model-Aware Validation Strategy

**Status:** Proposed  
**Date:** 2025-11-08  
**Decision Makers:** Architecture Team, Platform Team  
**Priority:** High  
**Category:** Validation Strategy, Platform Integration  

---

## Context and Problem Statement

Jupyter notebooks in ML/AI workflows often interact with deployed models for inference, testing, and validation. Currently, the Jupyter Notebook Validator Operator validates notebook execution in isolation without awareness of the model serving infrastructure or deployed models in the cluster.

This creates several challenges:

1. **Environment Readiness**: No validation that the cluster has required model serving platforms (KServe, OpenShift AI, vLLM, etc.) before deploying notebooks
2. **Model Compatibility**: No verification that notebooks can successfully communicate with deployed models
3. **Prediction Consistency**: No automated testing of prediction outputs against expected results
4. **Resource Integrity**: No health checks for deployed models that notebooks depend on
5. **Platform Fragmentation**: Different model serving platforms (KServe, vLLM, TorchServe, Triton, etc.) require different integration approaches

**Key Questions:**
- Should the operator be aware of model serving platforms in the cluster?
- How can we validate notebooks against deployed models without tight coupling?
- How do we support multiple model serving platforms (KServe, OpenShift AI, vLLM, TorchServe, etc.)?
- Should this be optional or mandatory?
- How do we enable community contributions for additional platforms?

---

## Decision Drivers

### Business Value
- **Reduced Deployment Failures**: Catch model integration issues before production deployment
- **Faster Feedback Loops**: Validate model compatibility during notebook development
- **Improved Reliability**: Ensure notebooks work with actual deployed models, not just mock data
- **Cost Savings**: Prevent failed deployments and reduce debugging time
- **Compliance**: Validate that notebooks meet model governance requirements

### Technical Drivers
- **Platform Diversity**: Support multiple model serving platforms (KServe, OpenShift AI, vLLM, TorchServe, TensorFlow Serving, Triton, Ray Serve, Seldon, BentoML)
- **Kubernetes Native**: Leverage Kubernetes CRDs for model discovery
- **Extensibility**: Enable community contributions for new platforms
- **Backward Compatibility**: Must not break existing validation workflows
- **Optional Feature**: Should be opt-in, not mandatory

### User Needs
- **Data Scientists**: Need confidence that notebooks will work with deployed models
- **ML Engineers**: Need automated validation of model integration
- **Platform Teams**: Need visibility into notebook-model dependencies
- **DevOps Teams**: Need deployment readiness checks

---

## Decision

We will implement **optional model-aware validation** with a **two-phase validation strategy** and a **plugin-based architecture** supporting multiple model serving platforms.

### Core Decision Components

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

#### 2. Built-In Platform Support

**Tier 1: Built-In (Fully Supported)**
- **KServe** - Standard Kubernetes model serving (primary)
- **OpenShift AI** - Red Hat's enterprise AI platform (development reference)

**Rationale:**
- KServe is the most widely adopted open-source standard
- OpenShift AI is our development/test environment
- Covers 80% of use cases
- Provides strong foundation for community contributions

#### 3. Community Platform Support

**Tier 2: Community-Documented (Documentation + Examples)**
- **vLLM** - LLM-focused serving
- **TorchServe** - PyTorch model serving
- **TensorFlow Serving** - TensorFlow model serving
- **Triton Inference Server** - NVIDIA multi-framework serving
- **Ray Serve** - Distributed model serving
- **Seldon Core** - Advanced ML deployments
- **BentoML** - Model packaging and serving

**Rationale:**
- Enables community contributions
- Provides clear integration patterns
- Reduces maintenance burden
- Allows ecosystem growth

#### 4. CRD Design

Add optional `modelValidation` field to `NotebookValidationJobSpec`:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-with-models
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
    targetModels:
      - fraud-detection-model
      - risk-scoring-model
    predictionValidation:
      enabled: true
      testData: |
        {"instances": [[1.0, 2.0, 3.0, 4.0]]}
      expectedOutput: |
        {"predictions": [[0.95, 0.05]]}
      tolerance: 0.01
```

#### 5. Platform Detection

Automatic platform detection via Kubernetes API:
- Query for platform-specific CRDs (`serving.kserve.io`, `ray.io`, etc.)
- Detect OpenShift AI via model registry CRDs
- Fall back to explicit platform specification if auto-detect fails

#### 6. RBAC Requirements

Validation pods need additional permissions:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-validator
rules:
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices/status"]
    verbs: ["get"]
```

---

## Consequences

### Positive Consequences

✅ **Improved Reliability**
- Notebooks validated against actual deployed models
- Catches integration issues early in development
- Reduces production deployment failures

✅ **Platform Flexibility**
- Supports multiple model serving platforms
- Works on standard Kubernetes and OpenShift
- Enables community contributions

✅ **Backward Compatibility**
- Optional feature (opt-in)
- Existing workflows unaffected
- Gradual adoption path

✅ **Developer Experience**
- Clear validation phases (clean vs. existing)
- Automated prediction consistency checks
- Fast feedback on model compatibility

✅ **Operational Visibility**
- Platform readiness checks
- Model health monitoring
- Dependency tracking

✅ **Community Growth**
- Clear contribution guidelines
- Extensible architecture
- Documentation for 7+ platforms

### Negative Consequences

⚠️ **Increased Complexity**
- Additional CRD fields
- More controller logic
- Platform-specific code paths

⚠️ **RBAC Management**
- Additional permissions required
- ServiceAccount configuration
- Security considerations

⚠️ **Maintenance Burden**
- Multiple platform integrations
- API compatibility tracking
- Documentation updates

⚠️ **Testing Overhead**
- Need test environments for each platform
- Integration test complexity
- CI/CD pipeline updates

### Mitigation Strategies

1. **Complexity**: Use plugin architecture to isolate platform-specific logic
2. **RBAC**: Provide clear RBAC templates and examples
3. **Maintenance**: Focus on built-in platforms; community maintains others
4. **Testing**: Prioritize KServe/OpenShift AI; community tests others

---

## Alternatives Considered

### Alternative 1: No Model Awareness (Status Quo)
**Rejected**: Leaves gap in validation coverage; notebooks may fail in production

### Alternative 2: Mandatory Model Validation
**Rejected**: Too restrictive; not all notebooks interact with models

### Alternative 3: Only KServe Support
**Rejected**: Limits platform choice; doesn't support OpenShift AI or community platforms

### Alternative 4: External Validation Service
**Rejected**: Adds operational complexity; breaks Kubernetes-native pattern

### Alternative 5: Webhook-Based Validation
**Rejected**: Doesn't fit operator pattern; harder to debug

---

## Implementation Plan

See `docs/IMPLEMENTATION-PLAN.md` Phase 4.3 for detailed tasks.

**High-Level Milestones:**
1. **Week 1-2**: CRD design and API updates
2. **Week 3-4**: Platform detection and controller logic
3. **Week 5-6**: KServe and OpenShift AI integration
4. **Week 7-8**: Community platform documentation
5. **Week 9-10**: Testing and examples
6. **Week 11-12**: Documentation and release

---

## Related ADRs

- **ADR-001**: Operator SDK and Go as Primary Development Framework
- **ADR-002**: Custom Resource Definition (CRD) Design for NotebookValidationJob
- **ADR-009**: Secret Management and Git Credentials
- **ADR-014**: Notebook Credential Injection Strategy
- **ADR-016**: External Secrets Operator Integration

---

## References

- [KServe Documentation](https://kserve.github.io/website/)
- [OpenShift AI Documentation](https://docs.redhat.com/en/documentation/red_hat_openshift_ai_self-managed/)
- [vLLM Documentation](https://docs.vllm.ai/)
- [Model Serving Platform Comparison](https://www.hopsworks.ai/dictionary/model-serving)
- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)

---

## Approval

- [ ] Architecture Team
- [ ] Platform Team
- [ ] Security Team
- [ ] ML Engineering Team

---

**Last Updated:** 2025-11-08  
**Next Review:** 2025-12-08

