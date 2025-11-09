# ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10

**Status**: Proposed  
**Date**: 2025-11-09  
**Authors**: Sophia (AI Assistant), User Feedback  
**Related**: ADR-031 (Tekton Build), ADR-005 (OpenShift Compatibility), ADR-022 (Testing Strategy)

## Context

We must ensure that unit and integration tests run against the same Kubernetes API version that OpenShift 4.18 uses (Kubernetes v1.31.10). Discrepancies between test-cluster versions and the production OCP version risk breakages that only surface at release time.

### Current Situation

- **Production Environment**: OpenShift 4.18.21 (Kubernetes 1.31.10)
- **Current CI Testing**: May use different Kubernetes versions (latest stable)
- **Risk**: API compatibility issues not detected until production deployment
- **Evidence**: ADR-031 Tekton implementation required specific Kubernetes API features

### Problem Statement

Without version-pinned CI testing:
1. Tests may pass on newer/older Kubernetes versions but fail on OpenShift 4.18
2. API deprecations or changes go undetected
3. Tekton Pipeline API compatibility issues emerge late
4. Operator SDK generated code may use incompatible APIs
5. CRD validation and webhook behavior differs across versions

## Decision

Configure GitHub Actions workflows to spin up a **KinD (Kubernetes-in-Docker) cluster pinned to Kubernetes v1.31.10** for all unit and integration tests.

### Implementation Details

1. **GitHub Actions Workflow Configuration**:
   ```yaml
   - name: Create KinD cluster
     uses: helm/kind-action@v1
     with:
       version: v0.20.0
       node_image: kindest/node:v1.31.10
       cluster_name: test-cluster
   ```

2. **Test Scope**:
   - Unit tests for controller logic
   - Integration tests for CRD validation
   - Tekton Pipeline/Task creation and execution
   - Operator deployment and lifecycle
   - Webhook validation

3. **Version Maintenance**:
   - Monitor OpenShift 4.18.x releases for Kubernetes version updates
   - Update KinD node image when OpenShift upgrades Kubernetes
   - Document version mapping in `docs/INTEGRATION_TESTING.md`

## Consequences

### Positive

- ✅ **Early Detection**: Kubernetes API compatibility issues caught in CI pipeline
- ✅ **Consistent Environment**: Test cluster mirrors production OpenShift 4.18.21
- ✅ **Reduced Risk**: Runtime failures due to API version drift eliminated
- ✅ **Confident Releases**: Higher confidence in production deployments
- ✅ **Faster Debugging**: Issues identified before manual testing phase

### Negative

- ❌ **Maintenance Overhead**: Must update KinD version when OpenShift upgrades
- ❌ **Longer Setup Time**: Version-specific cluster provisioning adds ~30s to CI
- ❌ **Monitoring Required**: Need to track OpenShift version updates
- ❌ **Version Lag**: May lag behind latest Kubernetes features

### Neutral

- ⚠️ **Documentation**: Requires clear documentation of version mapping
- ⚠️ **Team Training**: Team must understand version pinning rationale

## Alternatives Considered

### Alternative 1: Latest Stable KinD/K3s
- **Pros**: Easier to maintain, no version tracking needed
- **Cons**: Test cluster drift vs. production, API compatibility issues
- **Rejected**: High risk of production failures

### Alternative 2: Minikube Pinned Version
- **Pros**: Similar version control
- **Cons**: Slower startup (~2-3 minutes), similar drift risk
- **Rejected**: Slower CI, no significant advantage

### Alternative 3: No Version Pinning
- **Pros**: Fastest setup, always latest features
- **Cons**: Highest risk of compatibility issues
- **Rejected**: Unacceptable production risk

## Implementation Plan

### Phase 1: Setup (Week 1)
1. Create `.github/workflows/ci-unit-tests.yaml`
2. Configure KinD with Kubernetes v1.31.10
3. Run existing unit tests
4. Validate test results

### Phase 2: Integration (Week 2)
5. Add integration tests for Tekton Pipelines
6. Test operator deployment and lifecycle
7. Validate CRD and webhook behavior
8. Document CI environment in `docs/INTEGRATION_TESTING.md`

### Phase 3: Monitoring (Ongoing)
9. Set up alerts for OpenShift 4.18.x releases
10. Create process for version updates
11. Maintain version mapping documentation

## Verification

### Success Criteria
- [ ] CI tests run on Kubernetes v1.31.10
- [ ] All existing tests pass
- [ ] Tekton Pipeline tests execute successfully
- [ ] Documentation updated
- [ ] Team trained on version pinning

### Testing
```bash
# Verify KinD cluster version
kubectl version --short

# Expected output:
# Server Version: v1.31.10
```

## References

- [OpenShift 4.18 Release Notes](https://docs.openshift.com/container-platform/4.18/release_notes/ocp-4-18-release-notes.html)
- [KinD Node Images](https://github.com/kubernetes-sigs/kind/releases)
- [Kubernetes Version Skew Policy](https://kubernetes.io/releases/version-skew-policy/)
- ADR-031: Tekton Build Dockerfile vs Base Image Support
- ADR-005: OpenShift Compatibility
- ADR-022: Testing Strategy

## Notes

- OpenShift 4.18.21 verified to run Kubernetes 1.31.10
- KinD node image `kindest/node:v1.31.10` available
- GitHub Actions `helm/kind-action@v1` supports version pinning
- Estimated CI time increase: 30-60 seconds per workflow run

