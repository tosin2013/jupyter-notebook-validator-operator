# ADR 006: Version Support Roadmap and Testing Strategy

## Status
Accepted

## Context

The PRD outlines a phased approach for platform support, starting with OpenShift 4.18 and expanding to 4.19, 4.20, and upstream Kubernetes. This ADR provides a detailed roadmap and testing strategy for implementing this multi-phase support plan.

### PRD Requirements

From Section 8 (Platform Support & Evolution Plan):
- **Phase 1**: OpenShift 4.18 (initial target)
- **Phase 2**: OpenShift 4.19 & 4.20
- **Phase 3**: Latest stable Kubernetes release (community support)

### Strategic Goals

1. **Minimize Risk**: Start with single platform, expand incrementally
2. **Validate Stability**: Ensure operator is production-ready before expanding
3. **Community Adoption**: Enable broader Kubernetes community usage
4. **Enterprise Confidence**: Demonstrate commitment to long-term support
5. **API Compatibility**: Proactively address API deprecations

### Current State

- **Available Cluster**: OpenShift 4.18 cluster accessible via `oc cluster-info`
- **Initial Development**: Starting from scratch (greenfield project)
- **Target Users**: MLOps engineers, data scientists, platform teams

## Decision

We will implement a **Three-Phase Version Support Roadmap** with explicit milestones, testing requirements, and success criteria for each phase.

### Phase 1: OpenShift 4.18 Foundation (Months 1-3)

#### Objectives
- Establish operator core functionality
- Achieve production-ready stability on OpenShift 4.18
- Build comprehensive test suite
- Document installation and usage

#### Deliverables
1. **Operator Implementation**
   - Complete CRD implementation (v1alpha1)
   - Controller reconciliation logic
   - Pod orchestration for notebook validation
   - Git integration for notebook fetching
   - Status reporting and error handling

2. **Testing Infrastructure**
   - Unit tests (>80% coverage)
   - Integration tests with envtest
   - E2E tests against OpenShift 4.18 cluster
   - Performance benchmarks

3. **Documentation**
   - Installation guide for OpenShift 4.18
   - User guide with examples
   - Troubleshooting guide
   - API reference

4. **CI/CD Pipeline**
   - Automated testing on every PR
   - Container image builds
   - OLM bundle generation
   - Release automation

#### Success Criteria
- ✅ All unit tests passing (>80% coverage)
- ✅ E2E tests passing on OpenShift 4.18
- ✅ Operator successfully validates notebooks end-to-end
- ✅ OLM bundle installs cleanly via OperatorHub
- ✅ Documentation complete and reviewed
- ✅ At least 3 successful production deployments

#### Testing Strategy
```yaml
# .github/workflows/phase1-tests.yml
name: Phase 1 - OpenShift 4.18 Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Run unit tests
        run: make test
      
      - name: Check coverage
        run: make coverage
        # Fail if coverage < 80%

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Setup envtest
        run: make envtest
      
      - name: Run integration tests
        run: make test-integration

  e2e-openshift-4-18:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to OpenShift 4.18
        run: make deploy IMG=${{ env.IMAGE }}
      
      - name: Run E2E tests
        run: make test-e2e PLATFORM=openshift VERSION=4.18
      
      - name: Collect logs
        if: failure()
        run: make logs
```

### Phase 2: OpenShift 4.19 & 4.20 Expansion (Months 4-6)

#### Objectives
- Validate compatibility with newer OpenShift versions
- Address API deprecations and breaking changes
- Expand test matrix to cover multiple versions
- Update documentation for version-specific considerations

#### Prerequisites
- Phase 1 complete and stable
- OpenShift 4.19 and 4.20 clusters available for testing
- No critical bugs in Phase 1 deployment

#### Deliverables
1. **Compatibility Testing**
   - API deprecation analysis for 4.19 and 4.20
   - Migration plan for deprecated APIs
   - Version-specific feature detection

2. **Expanded Test Matrix**
   - E2E tests against OpenShift 4.18, 4.19, 4.20
   - Cross-version upgrade testing
   - Backward compatibility validation

3. **Documentation Updates**
   - Version compatibility matrix
   - Version-specific installation notes
   - Upgrade guides between versions

4. **CI/CD Enhancements**
   - Multi-version test matrix
   - Automated compatibility checks
   - Version-specific release artifacts

#### Success Criteria
- ✅ All tests passing on OpenShift 4.18, 4.19, 4.20
- ✅ No API deprecation warnings in any supported version
- ✅ Successful upgrades between operator versions
- ✅ Version-specific documentation complete
- ✅ At least 2 production deployments on 4.19 or 4.20

#### Testing Strategy
```yaml
# .github/workflows/phase2-tests.yml
name: Phase 2 - Multi-Version OpenShift Tests

on: [push, pull_request]

jobs:
  e2e-matrix:
    strategy:
      matrix:
        openshift-version: ['4.18', '4.19', '4.20']
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to OpenShift ${{ matrix.openshift-version }}
        run: make deploy IMG=${{ env.IMAGE }} VERSION=${{ matrix.openshift-version }}
      
      - name: Run E2E tests
        run: make test-e2e PLATFORM=openshift VERSION=${{ matrix.openshift-version }}
      
      - name: Check API deprecations
        run: make check-deprecations VERSION=${{ matrix.openshift-version }}

  upgrade-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Install previous version
        run: make deploy IMG=${{ env.PREVIOUS_IMAGE }}
      
      - name: Create test resources
        run: kubectl apply -f config/samples/
      
      - name: Upgrade to current version
        run: make deploy IMG=${{ env.CURRENT_IMAGE }}
      
      - name: Verify resources still work
        run: make test-upgrade
```

#### API Deprecation Tracking
```go
// pkg/version/compatibility.go
package version

import (
    "k8s.io/apimachinery/pkg/version"
)

// DeprecationWarnings tracks known API deprecations
var DeprecationWarnings = map[string][]string{
    "4.19": {
        "batch/v1beta1 CronJob is deprecated, use batch/v1",
        "policy/v1beta1 PodSecurityPolicy is removed",
    },
    "4.20": {
        "flowcontrol.apiserver.k8s.io/v1beta1 is deprecated",
    },
}

// CheckDeprecations validates operator doesn't use deprecated APIs
func CheckDeprecations(clusterVersion *version.Info) []string {
    // Implementation
}
```

### Phase 3: Kubernetes Community Support (Months 7-9)

#### Objectives
- Enable operator usage on vanilla Kubernetes
- Support multiple Kubernetes distributions (GKE, EKS, AKS, kind)
- Establish community distribution channels
- Build community adoption and feedback loop

#### Prerequisites
- Phase 2 complete and stable
- No critical bugs in OpenShift deployments
- Community interest validated (GitHub stars, issues, discussions)

#### Deliverables
1. **Kubernetes Compatibility**
   - Remove OpenShift-specific dependencies
   - Test on vanilla Kubernetes 1.25+
   - Support multiple distributions (GKE, EKS, AKS, kind)
   - Helm chart for Kubernetes deployment

2. **Community Distribution**
   - Publish to OperatorHub.io (community catalog)
   - Publish Helm chart to Artifact Hub
   - GitHub Releases with installation manifests
   - Container images on public registry (Quay.io, Docker Hub)

3. **Documentation Expansion**
   - Kubernetes installation guide
   - Distribution-specific notes (GKE, EKS, AKS)
   - Local development with kind/minikube
   - Community contribution guide

4. **CI/CD for Kubernetes**
   - Tests against latest Kubernetes (kind)
   - Tests against managed K8s (GKE/EKS)
   - Automated Helm chart publishing
   - Community catalog updates

#### Success Criteria
- ✅ All tests passing on Kubernetes 1.25+
- ✅ Successful deployments on GKE, EKS, AKS
- ✅ Operator listed on OperatorHub.io
- ✅ Helm chart published to Artifact Hub
- ✅ At least 5 community deployments
- ✅ Active community engagement (issues, PRs, discussions)

#### Testing Strategy
```yaml
# .github/workflows/phase3-tests.yml
name: Phase 3 - Kubernetes Community Tests

on: [push, pull_request]

jobs:
  e2e-kubernetes:
    strategy:
      matrix:
        k8s-version: ['1.25', '1.26', '1.27', '1.28', 'latest']
        distribution: ['kind', 'gke', 'eks']
    runs-on: ubuntu-latest
    steps:
      - name: Setup ${{ matrix.distribution }} cluster
        run: make setup-cluster DIST=${{ matrix.distribution }} VERSION=${{ matrix.k8s-version }}
      
      - name: Install via Helm
        run: |
          helm install jupyter-validator ./charts/jupyter-notebook-validator-operator \
            --namespace jupyter-validator-system \
            --create-namespace
      
      - name: Run E2E tests
        run: make test-e2e PLATFORM=kubernetes VERSION=${{ matrix.k8s-version }}

  helm-chart-validation:
    runs-on: ubuntu-latest
    steps:
      - name: Lint Helm chart
        run: helm lint charts/jupyter-notebook-validator-operator
      
      - name: Template Helm chart
        run: helm template charts/jupyter-notebook-validator-operator
      
      - name: Validate manifests
        run: kubectl apply --dry-run=client -f <(helm template charts/jupyter-notebook-validator-operator)
```

## Consequences

### Positive
- **Risk Mitigation**: Phased approach reduces risk of premature expansion
- **Quality Focus**: Each phase has clear success criteria
- **Community Building**: Phase 3 enables broader adoption
- **Enterprise Confidence**: Demonstrates commitment to stability and support
- **Clear Roadmap**: Stakeholders understand timeline and priorities

### Negative
- **Delayed Community Access**: Kubernetes users must wait for Phase 3
- **Resource Intensive**: Multi-version testing requires significant CI/CD resources
- **Maintenance Burden**: Supporting multiple versions increases complexity
- **Timeline Risk**: Delays in early phases cascade to later phases

### Neutral
- **Flexibility**: Phases can be adjusted based on feedback and priorities
- **Incremental Value**: Each phase delivers value independently

## Implementation Notes

### Phase Transition Criteria

#### Phase 1 → Phase 2 Transition
```bash
# Checklist before starting Phase 2
□ All Phase 1 success criteria met
□ No P0/P1 bugs in production
□ Test coverage ≥ 80%
□ Documentation complete
□ At least 3 production deployments
□ OpenShift 4.19/4.20 clusters provisioned
□ Team capacity available for Phase 2
```

#### Phase 2 → Phase 3 Transition
```bash
# Checklist before starting Phase 3
□ All Phase 2 success criteria met
□ No P0/P1 bugs across all OpenShift versions
□ Successful cross-version upgrades
□ Community interest validated (>50 GitHub stars)
□ Kubernetes test infrastructure ready
□ Helm chart developed and tested
□ Team capacity available for Phase 3
```

### Version Support Matrix (Post-Phase 3)

| Operator Version | OpenShift 4.18 | OpenShift 4.19 | OpenShift 4.20 | Kubernetes 1.25+ |
|------------------|----------------|----------------|----------------|------------------|
| v0.1.x (Phase 1) | ✅ Tier 1      | ❌             | ❌             | ❌               |
| v0.2.x (Phase 2) | ✅ Tier 1      | ✅ Tier 1      | ✅ Tier 1      | ❌               |
| v0.3.x (Phase 3) | ✅ Tier 1      | ✅ Tier 1      | ✅ Tier 1      | ✅ Tier 2        |

### Monitoring and Metrics

Track these metrics throughout all phases:

```yaml
# Metrics to track
quality_metrics:
  - test_coverage_percentage
  - e2e_test_pass_rate
  - mean_time_to_recovery
  - bug_escape_rate

adoption_metrics:
  - active_installations
  - github_stars
  - community_contributions
  - support_ticket_volume

performance_metrics:
  - reconciliation_duration
  - validation_job_success_rate
  - resource_utilization
```

### Risk Mitigation

| Risk | Mitigation |
|------|------------|
| API deprecations break operator | Automated deprecation checks in CI/CD |
| Phase delays cascade | Build buffer time between phases |
| Community adoption lower than expected | Early community engagement, beta program |
| Resource constraints for testing | Prioritize critical paths, use ephemeral clusters |
| Breaking changes between versions | Comprehensive upgrade testing, rollback procedures |

## References

- [Kubernetes Version Skew Policy](https://kubernetes.io/releases/version-skew-policy/)
- [OpenShift Life Cycle Policy](https://access.redhat.com/support/policy/updates/openshift)
- [Operator Maturity Model](https://sdk.operatorframework.io/docs/overview/operator-capabilities/)
- [Testing Best Practices](https://kubernetes.io/docs/tasks/debug/debug-cluster/)

## Related ADRs

- ADR 002: Platform Version Support Strategy (defines support tiers)
- ADR 007: Distribution and Catalog Strategy (defines where operator is published)
- ADR 008: CI/CD Pipeline Integration (implements testing infrastructure)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial roadmap |

