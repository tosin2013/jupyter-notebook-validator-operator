# S2I Build Integration Plan

## Overview

This document summarizes the architectural decisions and implementation plan for integrating OpenShift Source-to-Image (S2I) builds into the Jupyter Notebook Validator Operator.

**Date:** 2025-01-08  
**Status:** Planning Complete, Implementation Pending  
**Priority:** High - Resolves critical OpenShift SCC permission issues

## Problem Statement

### Current Issue
The operator currently attempts runtime pip installation in validation pods, which fails on OpenShift due to Security Context Constraints (SCC):

- OpenShift's `restricted-v2` SCC assigns random UIDs (e.g., 1000920000) to containers
- Standard Jupyter images expect UID 1000 and write to `/home/jovyan`
- Runtime `pip install` fails with "Permission denied" errors
- Environment variable workarounds are unreliable

### Impact
- **100% failure rate** for notebooks with custom dependencies on OpenShift
- Poor user experience (confusing error messages)
- Blocks adoption on OpenShift clusters
- Inconsistent behavior between vanilla Kubernetes and OpenShift

## Solution: S2I Build Integration

### Approach
Pre-build container images with dependencies before notebook execution using OpenShift's native S2I capability:

1. **Detect OpenShift** platform via API group detection
2. **Trigger S2I build** when `buildConfig` is specified in CRD
3. **Wait for build completion** and monitor build logs
4. **Use built image** for validation pod (no runtime pip installation)
5. **Handle missing requirements.txt** with multi-tiered fallback strategy

### Benefits
- ✅ **Eliminates runtime pip failures** on OpenShift (100% resolution)
- ✅ **Faster notebook execution** (no per-job installation overhead)
- ✅ **Better security** (dependencies installed at build time, not runtime)
- ✅ **Improved reproducibility** (immutable images)
- ✅ **Aligns with industry patterns** (Azure ML, SageMaker, Vertex AI)
- ✅ **Leverages native OpenShift tooling** (no external CI/CD needed)

## Architectural Decisions

### ADR-023: Strategy for Source-to-Image (S2I) Build Integration on OpenShift
**File:** `docs/adrs/023-s2i-build-integration-openshift.md`

**Decision:** Adopt OpenShift S2I as the primary build mechanism for creating notebook validation container images.

**Key Points:**
- Optional `buildConfig` field in CRD (opt-in behavior)
- Detect OpenShift by checking for `build.openshift.io` API group
- Create S2I BuildConfig referencing notebook's Git repository
- Use Jupyter-compatible S2I builder image as base
- Wait for build completion before creating validation pod

**CRD Schema:**
```yaml
spec:
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"  # Optional
    buildConfig:  # Optional: trigger S2I build
      enabled: true
      strategy: "s2i"
      baseImage: "quay.io/jupyter/minimal-notebook:latest"
      autoGenerateRequirements: false
      requirementsFile: "requirements.txt"
```

### ADR-024: Fallback Strategy for Notebooks Missing requirements.txt
**File:** `docs/adrs/024-fallback-strategy-missing-requirements.md`

**Decision:** Implement multi-tiered fallback strategy for handling notebooks without `requirements.txt`.

**Fallback Tiers:**
1. **Tier 1:** Use existing requirements.txt (primary)
2. **Tier 2:** Auto-generate with pipreqs (opt-in via `autoGenerateRequirements`)
3. **Tier 3:** Detect inline pip magic commands (`%pip install`, `!pip install`)
4. **Tier 4:** Proceed with base image only (with warning)
5. **Tier 5:** Clear error messages if validation fails

**Key Points:**
- pipreqs analyzes Python imports using AST
- Filters standard library imports
- Opt-in approach for auto-generation
- Clear error messages guide users to solutions

### ADR-025: Community-Contributed Build Methods and Extension Framework
**File:** `docs/adrs/025-community-build-methods-extension-framework.md`

**Decision:** Establish community contribution framework for alternative build methods.

**Key Points:**
- S2I is the only officially supported method (core)
- Define `BuildStrategy` interface for pluggable backends
- Community directory structure for alternative implementations
- Reference implementations for Tekton and Kaniko
- Feature flag system for enabling community strategies

**BuildStrategy Interface:**
```go
type BuildStrategy interface {
    Name() string
    Detect(ctx context.Context, client client.Client) (bool, error)
    CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error
    GetBuildStatus(ctx context.Context, buildName string) (BuildStatus, error)
    GetBuiltImage(ctx context.Context, buildName string) (string, error)
    DeleteBuild(ctx context.Context, buildName string) error
}
```

## Implementation Plan

### Phase 4.5: S2I Build Integration (6 weeks)

#### Week 1: CRD Updates and Platform Detection
- [ ] Add `buildConfig` field to CRD
- [ ] Extend platform detector to identify OpenShift
- [ ] Update OpenAPI schema and regenerate manifests
- [ ] Unit tests for platform detection

#### Week 2: S2I Build Orchestration
- [ ] Create `internal/controller/build_helper.go`
- [ ] Implement S2I BuildConfig creation
- [ ] Implement build status monitoring
- [ ] Implement build log streaming
- [ ] Unit tests for build orchestration

#### Week 3: Requirements.txt Fallback Strategies
- [ ] Integrate pipreqs for auto-generation
- [ ] Implement inline pip magic detection
- [ ] Implement standard library filtering
- [ ] Create error message templates
- [ ] Unit tests for fallback logic

#### Week 4: Controller Integration
- [ ] Update reconciliation loop
- [ ] Add build status conditions
- [ ] Integrate build orchestration
- [ ] Handle build failures gracefully
- [ ] Integration tests on OpenShift

#### Week 5: Community Framework and Documentation
- [ ] Define BuildStrategy interface
- [ ] Implement S2I strategy using interface
- [ ] Create community contribution framework
- [ ] Write comprehensive documentation
- [ ] Create reference implementations (Tekton, Kaniko)

#### Week 6: E2E Testing and Refinement
- [ ] Create test notebooks (with/without requirements.txt)
- [ ] E2E tests on OpenShift cluster
- [ ] Performance testing
- [ ] Documentation review and updates
- [ ] User acceptance testing

## Success Criteria

### Functional Requirements
- ✅ CRD supports optional `buildConfig` field
- ✅ Platform detection identifies OpenShift correctly
- ✅ S2I builds trigger automatically when enabled
- ✅ Built images used for validation pods
- ✅ No runtime pip installation failures on OpenShift
- ✅ requirements.txt auto-generation works with pipreqs
- ✅ Inline pip magic commands detected and processed
- ✅ Clear error messages when dependencies missing

### Non-Functional Requirements
- ✅ Build completion within 5 minutes for typical notebooks
- ✅ Build logs streamed to operator events
- ✅ Graceful handling of build failures
- ✅ Image caching for repeated builds
- ✅ Comprehensive documentation
- ✅ 80%+ test coverage for new code

### User Experience
- ✅ Simple opt-in via `buildConfig.enabled: true`
- ✅ Automatic requirements.txt generation (opt-in)
- ✅ Clear error messages with actionable guidance
- ✅ Troubleshooting guide for common issues
- ✅ Examples for all use cases

## Dependencies

### Prerequisites
- ✅ OpenShift cluster with BuildConfig API (available)
- ✅ Image registry access (internal or external)
- ✅ ADR-023, ADR-024, ADR-025 created
- ⏳ S2I builder image with pipreqs (to be created)

### Related Work
- Phase 4.4: Model-Aware Validation (complete)
- ADR-009: Secret Management (complete)
- ADR-011: Error Handling (complete)

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Build failures harder to debug | Medium | Medium | Stream build logs; clear error messages; troubleshooting guide |
| S2I builder images outdated | Medium | Low | Version and maintain builder images; update with Jupyter releases |
| Network policies block registry | High | Low | Document required permissions; provide cluster admin guidance |
| Build queue delays | Medium | Medium | Implement timeout and retry logic; document expected build times |
| pipreqs incomplete requirements | Medium | High | Document limitations; recommend explicit requirements.txt for production |

## Next Steps

1. **Review ADRs with team** (Week 1)
2. **Begin CRD updates** (Week 1)
3. **Implement platform detection** (Week 1)
4. **Create S2I builder image** (Week 2)
5. **Implement build orchestration** (Week 2-3)
6. **Integrate into controller** (Week 4)
7. **Test on OpenShift cluster** (Week 5-6)
8. **Document and release** (Week 6)

## References

- **ADR-023:** `docs/adrs/023-s2i-build-integration-openshift.md`
- **ADR-024:** `docs/adrs/024-fallback-strategy-missing-requirements.md`
- **ADR-025:** `docs/adrs/025-community-build-methods-extension-framework.md`
- **Implementation Plan:** `docs/IMPLEMENTATION-PLAN.md` (Phase 4.5)
- **OpenShift S2I Documentation:** https://docs.openshift.com/container-platform/latest/openshift_images/using_images/using-s21-images.html
- **jupyter-on-openshift Project:** https://github.com/jupyter-on-openshift/jupyter-notebooks
- **pipreqs Documentation:** https://github.com/bndr/pipreqs

## Approval

- [ ] Architecture Review: _______________ Date: ___________
- [ ] Technical Lead: _______________ Date: ___________
- [ ] Product Owner: _______________ Date: ___________

