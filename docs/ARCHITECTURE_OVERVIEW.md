# Jupyter Notebook Validator Operator - Architecture Overview

## Executive Summary

The Jupyter Notebook Validator Operator is a Kubernetes-native tool designed to automate the validation of Jupyter Notebooks within MLOps workflows. This document provides a high-level overview of the architectural decisions documented in our ADRs.

## Quick Reference

- **Language**: Go 1.21+
- **Framework**: Operator SDK v1.32.0+
- **Primary Platform**: OpenShift 4.18 (initial), expanding to 4.19, 4.20, and Kubernetes 1.25+
- **CRD**: `NotebookValidationJob` (mlops.dev/v1alpha1)
- **Deployment**: OLM Bundle (primary), Helm Chart (secondary), Raw Manifests (tertiary)
- **Distribution**: OpenShift OperatorHub, Red Hat Catalog, OperatorHub.io, Artifact Hub

## Architectural Decisions

### 1. Technology Stack (ADR 001)

**Decision**: Operator SDK v1.32.0+ with Go Modules

**Why**: 
- Long-term API stability
- Native OpenShift integration
- Comprehensive tooling and community support
- Built-in controller-runtime

**Impact**: Standardized project structure, reduced boilerplate, seamless OLM integration

### 2. Platform Support Strategy (ADR 002, ADR 006)

**Decision**: Phased rollout with hybrid support model

**Phases**:
1. **Phase 1 (Months 1-3)**: OpenShift 4.18 foundation
2. **Phase 2 (Months 4-6)**: OpenShift 4.19 & 4.20 expansion
3. **Phase 3 (Months 7-9)**: Kubernetes 1.25+ community support

**Support Tiers**:
- **Tier 1**: OpenShift 4.18-4.20 (certified, full support)
- **Tier 2**: Kubernetes 1.25+ (community, best-effort)

**Why**: Balances enterprise needs with community adoption, manages risk through incremental expansion

**Impact**: Clear roadmap, manageable testing complexity, broad platform coverage

### 3. API Design (ADR 003)

**Decision**: Multi-version CRD with conversion webhooks

**API Versioning**:
- **v1alpha1**: Initial experimental version
- **v1beta1**: Stable API (future)
- **v1**: Production-ready (future)

**Key Features**:
- OpenAPI v3 schema validation
- Status subresource separation
- Server-side defaulting
- Conversion webhooks for smooth upgrades

**Why**: Enables API evolution without breaking users, follows Kubernetes best practices

**Impact**: Smooth upgrade path, strong validation, backward compatibility

### 4. Deployment & Packaging (ADR 004, ADR 007)

**Decision**: Multi-format packaging with phased distribution

**Packaging Formats**:
1. **OLM Bundle**: Primary for OpenShift, automatic upgrades
2. **Helm Chart**: Secondary for Kubernetes, templating support
3. **Raw Manifests**: Tertiary for GitOps/CI/CD, maximum flexibility

**Distribution Channels** (by phase):

| Phase | OpenShift OperatorHub | Red Hat Catalog | OperatorHub.io | Artifact Hub |
|-------|----------------------|-----------------|----------------|--------------|
| 1     | ✅ Community         | ❌              | ❌             | ❌           |
| 2     | ✅ Community         | ✅ Certified    | ❌             | ❌           |
| 3     | ✅ Community         | ✅ Certified    | ✅ Community   | ✅ Helm      |

**Why**: Best-in-class UX for each platform, maximum discoverability, broad adoption

**Impact**: Maintenance burden across formats, but superior user experience

### 5. Security & RBAC (ADR 005)

**Decision**: Least-privilege RBAC with hybrid deployment model

**Service Accounts**:
- **Operator SA**: `jupyter-notebook-validator-operator` (cluster or namespace-scoped)
- **Validation Pod SA**: `jupyter-notebook-validator-runner` (minimal permissions)

**Deployment Modes**:
- **Cluster-Wide**: Operator watches all namespaces (platform teams)
- **Namespace-Scoped**: Operator watches single namespace (self-service)
- **Multi-Namespace**: Operator watches specific namespaces (advanced)

**Security Features**:
- Non-root containers
- Read-only root filesystem
- Dropped capabilities
- OpenShift SCC compliance

**Why**: Follows security best practices, supports multiple deployment scenarios, audit-friendly

**Impact**: Flexible deployment, enhanced security posture, compliance-ready

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes/OpenShift Cluster              │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Operator Namespace                                 │    │
│  │                                                     │    │
│  │  ┌──────────────────────────────────────────┐     │    │
│  │  │  Jupyter Notebook Validator Operator     │     │    │
│  │  │  (Deployment)                            │     │    │
│  │  │                                          │     │    │
│  │  │  - Watches NotebookValidationJob CRs    │     │    │
│  │  │  - Reconciles desired state             │     │    │
│  │  │  - Creates validation pods              │     │    │
│  │  │  - Updates CR status                    │     │    │
│  │  └──────────────────────────────────────────┘     │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │  User Namespace                                     │    │
│  │                                                     │    │
│  │  ┌──────────────────────────────────────────┐     │    │
│  │  │  NotebookValidationJob CR                │     │    │
│  │  │  (Custom Resource)                       │     │    │
│  │  │                                          │     │    │
│  │  │  spec:                                   │     │    │
│  │  │    notebook:                             │     │    │
│  │  │      git: {url, ref, path}              │     │    │
│  │  │    podConfig: {image, resources}        │     │    │
│  │  └──────────────────────────────────────────┘     │    │
│  │                                                     │    │
│  │  ┌──────────────────────────────────────────┐     │    │
│  │  │  Validation Pod (ephemeral)              │     │    │
│  │  │                                          │     │    │
│  │  │  1. Clone Git repository                │     │    │
│  │  │  2. Execute notebook cells              │     │    │
│  │  │  3. Compare with golden (optional)      │     │    │
│  │  │  4. Report results                      │     │    │
│  │  └──────────────────────────────────────────┘     │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Validation Workflow

1. **User creates NotebookValidationJob CR**
   ```yaml
   apiVersion: mlops.dev/v1alpha1
   kind: NotebookValidationJob
   metadata:
     name: validate-my-notebook
   spec:
     notebook:
       git:
         url: https://github.com/org/repo.git
         ref: main
         path: notebooks/analysis.ipynb
   ```

2. **Operator reconciles CR**
   - Validates CR spec
   - Fetches notebook from Git
   - Creates validation pod with appropriate configuration

3. **Validation pod executes**
   - Clones Git repository
   - Executes notebook cells sequentially
   - Captures outputs and errors
   - Compares with golden notebook (if specified)

4. **Operator updates CR status**
   ```yaml
   status:
     phase: Succeeded  # or Failed
     results:
       - cellIndex: 0
         status: Success
       - cellIndex: 1
         status: Success
   ```

## Development Roadmap

### Phase 1: OpenShift 4.18 Foundation (Months 1-3)

**Goals**:
- ✅ Core operator functionality
- ✅ Production-ready on OpenShift 4.18
- ✅ OLM bundle published to community catalog
- ✅ Comprehensive test suite

**Deliverables**:
- Operator v0.1.0
- CRD v1alpha1
- Documentation and examples
- CI/CD pipeline

### Phase 2: OpenShift Expansion (Months 4-6)

**Goals**:
- ✅ Support OpenShift 4.19 & 4.20
- ✅ Red Hat certification
- ✅ Multi-version testing
- ✅ API stability improvements

**Deliverables**:
- Operator v0.2.0
- Certified operator bundle
- Version compatibility matrix
- Upgrade guides

### Phase 3: Kubernetes Community (Months 7-9)

**Goals**:
- ✅ Support Kubernetes 1.25+
- ✅ Helm chart distribution
- ✅ Community adoption
- ✅ Multiple distribution channels

**Deliverables**:
- Operator v0.3.0
- Helm chart on Artifact Hub
- OperatorHub.io listing
- Community documentation

## Testing Strategy

### Test Pyramid

```
                    ┌─────────────┐
                    │   E2E Tests │  (Multi-version, multi-platform)
                    └─────────────┘
                  ┌───────────────────┐
                  │ Integration Tests │  (envtest, CRD validation)
                  └───────────────────┘
              ┌─────────────────────────────┐
              │       Unit Tests            │  (>80% coverage)
              └─────────────────────────────┘
```

### Test Matrix (Phase 3)

| Test Type | OpenShift 4.18 | OpenShift 4.19 | OpenShift 4.20 | K8s 1.25+ |
|-----------|----------------|----------------|----------------|-----------|
| Unit      | ✅             | ✅             | ✅             | ✅        |
| Integration | ✅           | ✅             | ✅             | ✅        |
| E2E       | ✅             | ✅             | ✅             | ✅        |

## Key Metrics

### Quality Metrics
- Test coverage: >80%
- E2E test pass rate: >95%
- Mean time to recovery: <4 hours
- Bug escape rate: <5%

### Adoption Metrics
- Active installations
- GitHub stars and forks
- Community contributions
- Support ticket volume

### Performance Metrics
- Reconciliation duration: <5s
- Validation job success rate: >90%
- Resource utilization: <200Mi memory, <200m CPU

## Next Steps

1. **Initialize Project** (Week 1)
   ```bash
   operator-sdk init --domain mlops.dev --repo github.com/your-org/jupyter-notebook-validator-operator
   operator-sdk create api --group mlops --version v1alpha1 --kind NotebookValidationJob --resource --controller
   ```

2. **Implement Core Features** (Weeks 2-8)
   - CRD types and validation
   - Controller reconciliation logic
   - Git integration
   - Pod orchestration
   - Status reporting

3. **Build Test Suite** (Weeks 9-10)
   - Unit tests
   - Integration tests
   - E2E tests on OpenShift 4.18

4. **Create OLM Bundle** (Week 11)
   - ClusterServiceVersion
   - Bundle manifests
   - Submit to community catalog

5. **Documentation** (Week 12)
   - Installation guide
   - User guide
   - API reference
   - Troubleshooting

## References

### ADRs
- [ADR 001: Operator Framework and SDK Version](adrs/001-operator-framework-and-sdk-version.md)
- [ADR 002: Platform Version Support Strategy](adrs/002-platform-version-support-strategy.md)
- [ADR 003: CRD Schema Design and Versioning](adrs/003-crd-schema-design-and-versioning.md)
- [ADR 004: Deployment and Packaging Strategy](adrs/004-deployment-and-packaging-strategy.md)
- [ADR 005: RBAC and Service Account Model](adrs/005-rbac-and-service-account-model.md)
- [ADR 006: Version Support Roadmap and Testing](adrs/006-version-support-roadmap-and-testing.md)
- [ADR 007: Distribution and Catalog Strategy](adrs/007-distribution-and-catalog-strategy.md)

### External Resources
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [OpenShift Documentation](https://docs.openshift.com/)
- [OLM Documentation](https://olm.operatorframework.io/)

## Questions?

For questions about the architecture or ADRs:
- Open an issue in the GitHub repository
- Review the detailed ADRs in `docs/adrs/`
- Contact the platform team

---

**Last Updated**: 2025-11-07
**Version**: 1.0
**Status**: Living Document

