# ADR-048: Pre-Submission Validation Test Rig Strategy

**Status**: Accepted  
**Date**: 2025-11-12  
**Authors**: Sophia (AI Assistant), User Feedback  
**Related**: ADR-032 (CI Testing), ADR-033 (E2E Testing), ADR-034 (Dual Testing Strategy), ADR-006 (Version Support Roadmap)

## Context

The Community Operators pipeline is strict, and waiting for their CI to fail is frustrating. Developers need a way to validate operator bundles locally before submission to catch issues early and avoid the slow feedback loop of Community Operators CI.

### Current Situation

- **Community Operators CI**: Strict validation pipeline that tests against multiple Kubernetes versions
- **Existing Local Testing**: `test-local-kind.sh` tests against a single Kubernetes version (v1.33.1)
- **Existing Bundle Validation**: `test-bundle-with-operator-sdk.sh` validates bundles but only against one version
- **Gap**: No systematic way to test against the Kubernetes version matrix that Community Operators actually uses
- **Problem**: API compatibility issues (e.g., deprecated v1beta1 APIs) only surface after submission

### Problem Statement

Without pre-submission validation:

1. **Slow Feedback Loop**: Waiting for Community Operators CI to fail takes hours/days
2. **Version Compatibility Unknown**: No visibility into which Kubernetes versions will pass/fail
3. **API Deprecation Blind Spots**: Removed fields (e.g., v1beta1) that exist in 4.18 but are gone in 4.20's underlying K8s version go undetected
4. **Resource Constraints**: Running three concurrent full OpenShift clusters (4.18, 4.19, 4.20) on one local VM is impossible due to massive resource overhead (~16GB+ RAM per cluster)

### Community Operators Testing Matrix

The Community Operators repository tests against upstream Kubernetes versions that map to OpenShift releases:

| OpenShift | Kubernetes | Community Operators Test |
|-----------|------------|-------------------------|
| 4.18      | 1.31       | ✅ Tested |
| 4.19      | 1.32       | ✅ Tested |
| 4.20      | 1.33       | ✅ Tested |

Additionally, Community Operators validates against earlier Kubernetes versions (1.29, 1.30) to ensure backward compatibility.

## Decision

Implement a **"Confidence Builder" CentOS Test Rig** that simulates Community Operators validation locally using sequential Kind clusters and the Operator Framework validation suite.

### Architecture Overview

The test rig does **not** run full OpenShift clusters. Instead, it:

1. **Uses Kind for Kubernetes Version Matrix**: Spins up sequential Kind clusters with different Kubernetes versions (1.29, 1.30, 1.31, 1.33)
2. **Uses Operator Framework Validator**: Runs `operator-sdk bundle validate` with `suite=operatorframework` to match Community Operators CI checks
3. **Runs Sequentially**: Tests one Kubernetes version at a time to avoid resource exhaustion
4. **OS-Aware**: Detects and adapts to the host operating system (RHEL/CentOS vs others)

### Implementation Strategy

#### Phase 1: CentOS Test Rig Setup

**Recommended VM Specs**: 4-8 CPUs, 16GB+ RAM, 100GB Disk

**Core Tools Installation**:
- Docker (or Podman) for container runtime
- Kind for Kubernetes-in-Docker clusters
- kubectl for cluster management
- operator-sdk for bundle validation

#### Phase 2: Kubernetes Version Matrix Testing

Test against the Kubernetes versions that Community Operators validates:

```bash
# Kubernetes versions to test (mapping to OpenShift)
declare -a K8S_VERSIONS=("v1.29.0" "v1.30.0" "v1.31.0" "v1.33.0")
```

**Test Flow**:
1. Create Kind cluster with specific Kubernetes version
2. Install OLM (Operator Lifecycle Manager)
3. Load operator bundle image
4. Run `operator-sdk bundle validate` with `suite=operatorframework`
5. Clean up cluster
6. Repeat for next version

#### Phase 3: Operator Framework Validation Suite

Use the same validation suite that Community Operators uses:

```bash
operator-sdk bundle validate ./bundle \
  --select-optional suite=operatorframework \
  --optional-values=k8s-version=1.31
```

**Why This Works**: If this passes with `suite=operatorframework`, you are 95% safe for OpenShift. The validator checks:
- Bundle structure and metadata
- CRD schema validation
- API compatibility
- Operator SDK best practices
- OLM bundle format compliance

### OpenShift Validation Strategy

**Cannot easily spin up OpenShift 4.18/4.19/4.20 on a local CentOS VM** due to resource constraints. Instead:

1. **Use Kubernetes Version Testing**: Test against the underlying Kubernetes versions (1.31, 1.32, 1.33)
2. **Use Red Hat Validator**: The `operatorframework` suite mimics Community Pipeline checks
3. **Consider Operating System**: Take into account the operating system the cluster runs on (RHEL/CentOS vs others)

**Alternative Options** (if full OpenShift testing needed):
- MicroShift (lightweight OpenShift for edge/IoT)
- Code-Ready Containers (CRC) for local OpenShift
- Rely on the strict validator suite (recommended)

## Consequences

### Positive

- ✅ **Fast Local Feedback**: Catch issues in minutes instead of waiting for Community Operators CI
- ✅ **Version Compatibility Visibility**: Know which Kubernetes versions pass/fail before submission
- ✅ **API Deprecation Detection**: Identify removed fields early (e.g., v1beta1 → v1 migrations)
- ✅ **Resource Efficient**: Sequential testing avoids resource exhaustion
- ✅ **95% Confidence**: Operator Framework validator suite matches Community Operators checks
- ✅ **OS-Aware**: Adapts to RHEL/CentOS environments
- ✅ **Reduced Frustration**: Avoid slow Community Operators CI feedback loop

### Negative

- ❌ **Not 100% Identical**: Local testing cannot perfectly replicate Community Operators CI environment
- ❌ **Sequential Execution**: Takes longer than parallel testing (but necessary due to resources)
- ❌ **Maintenance Overhead**: Must keep Kubernetes version matrix updated as Community Operators evolves
- ❌ **No Full OpenShift Testing**: Cannot test OpenShift-specific features (S2I, ImageStreams) locally
- ❌ **Setup Complexity**: Requires CentOS VM setup and tool installation

### Neutral

- ⚠️ **Documentation**: Requires clear documentation of version mapping and setup
- ⚠️ **Team Training**: Developers need to understand when to run pre-submission validation
- ⚠️ **Version Tracking**: Must monitor Community Operators for version matrix changes

## Alternatives Considered

### Alternative 1: Full OpenShift Clusters on VM
- **Pros**: 100% identical to production
- **Cons**: Impossible to run 3 concurrent clusters (48GB+ RAM needed), too slow for rapid iteration
- **Rejected**: Resource constraints make this impractical

### Alternative 2: Wait for Community Operators CI
- **Pros**: No local setup needed, guaranteed to match CI
- **Cons**: Slow feedback (hours/days), frustrating developer experience
- **Rejected**: Does not solve the problem of slow feedback

### Alternative 3: Single Kubernetes Version Testing
- **Pros**: Fast, simple
- **Cons**: Misses version compatibility issues, doesn't match Community Operators matrix
- **Rejected**: Insufficient coverage

### Alternative 4: Parallel Kind Clusters
- **Pros**: Faster execution
- **Cons**: Resource exhaustion on typical VMs (16GB RAM insufficient for 3+ clusters)
- **Rejected**: Sequential approach is more practical

## Implementation Plan

### Phase 1: Test Rig Infrastructure (Week 1)
1. ✅ Create `scripts/test-k8s-matrix.sh` - Kubernetes version matrix testing
2. ✅ Create `scripts/pre-submission-validate.sh` - Full pre-submission validation workflow
3. Document CentOS VM setup requirements
4. Document tool installation steps

### Phase 2: Integration (Week 2)
5. Integrate with existing `test-bundle-with-operator-sdk.sh`
6. Add version matrix to CI/CD pipeline (optional)
7. Create developer workflow documentation

### Phase 3: Documentation and Training (Week 3)
8. Update `docs/TESTING.md` with pre-submission validation
9. Create troubleshooting guide
10. Train team on pre-submission workflow
11. Document version mapping strategy

## Verification

### Success Criteria
- [ ] `test-k8s-matrix.sh` tests against all required Kubernetes versions
- [ ] `pre-submission-validate.sh` runs full validation suite
- [ ] Validation catches API deprecation issues before Community Operators submission
- [ ] Documentation complete
- [ ] Team trained on pre-submission workflow

### Testing
```bash
# Verify Kubernetes version matrix
./scripts/test-k8s-matrix.sh

# Verify full pre-submission validation
./scripts/pre-submission-validate.sh

# Expected: All Kubernetes versions pass validation
```

## References

- [Community Operators Repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- [Operator Framework Bundle Validation](https://sdk.operatorframework.io/docs/olm-integration/cli-overview/#bundle-validation)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [OpenShift Version Support Matrix](docs/OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md)
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- ADR-034: Dual Testing Strategy with Kind and OpenShift
- ADR-006: Version Support Roadmap and Testing Strategy

## Notes

- **Kubernetes Version Mapping**: Community Operators tests against upstream Kubernetes, not OpenShift directly. If it breaks on upstream K8s, it will break on OpenShift.
- **95% Confidence**: The `operatorframework` validation suite provides high confidence but not 100% identical to Community Operators CI
- **Resource Constraints**: Sequential execution is necessary on typical VMs (16GB RAM)
- **OS Consideration**: The operating system the cluster runs on should be taken into account (RHEL/CentOS vs others)
- **Version Updates**: Monitor Community Operators repository for changes to the Kubernetes version matrix
- **Integration**: This complements existing `test-local-kind.sh` (Tier 1 tests) and `test-bundle-with-operator-sdk.sh` (single version validation)

## Revision History

| Date | Author | Description |
|------|--------|-------------|
| 2025-11-12 | Sophia (AI Assistant) | Initial decision |
