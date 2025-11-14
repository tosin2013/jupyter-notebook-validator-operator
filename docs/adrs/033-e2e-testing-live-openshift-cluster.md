# ADR-033: End-to-End Testing Against Live OpenShift Cluster

**Status**: Accepted
**Date**: 2025-11-09
**Updated**: 2025-11-11 (Added test tier organization)
**Authors**: Sophia (AI Assistant), User Feedback
**Related**: ADR-032 (CI Testing), ADR-034 (Dual Testing), ADR-035 (Test Tier Organization), ADR-036 (Private Test Repository)

## Context

Beyond local KinD tests (ADR-032), we need E2E tests against a real OpenShift cluster to validate the complete operator workflow including Tekton builds, validation pods, and notebook execution.

### Current Situation

- **Local Testing**: KinD cluster with Kubernetes v1.31.10 (ADR-032)
- **Manual Testing**: E2E tests performed manually on OpenShift cluster
- **Test Notebooks**: External repository with comprehensive test suite
- **Gap**: No automated E2E testing in CI/CD pipeline

### Problem Statement

Local KinD testing cannot replicate OpenShift-specific features:
1. **ImageStreams**: OpenShift-specific image registry and tagging
2. **BuildConfigs**: S2I builds with OpenShift Build API
3. **Tekton Pipelines**: OpenShift Pipelines operator integration
4. **Security Contexts**: OpenShift SCCs (Security Context Constraints)
5. **Service Mesh**: OpenShift Service Mesh integration
6. **Networking**: OpenShift SDN and route configuration

### Test Notebook Repository

External test suite: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks` (private)

**Test Coverage** (See ADR-035 for detailed tier organization):

#### Tier 1: Simple Validation (< 30 seconds)
- **Environment**: Kind + OpenShift
- **Infrastructure**: None (no builds, no models)
- **Notebooks**: 4 notebooks (hello-world, basic-math, data-validation, error-test)
- **Purpose**: Basic notebook execution and validation

#### Tier 2: Intermediate Complexity (1-5 minutes)
- **Environment**: OpenShift only
- **Infrastructure**: S2I/Tekton builds, custom images
- **Notebooks**: 4+ notebooks (model training, data preprocessing, feature engineering)
- **Purpose**: Build integration and dependency management

#### Tier 3: Complex Integration (5-30 minutes)
- **Environment**: OpenShift only
- **Infrastructure**: Deployed models (KServe/OpenShift AI), External Secrets Operator
- **Notebooks**: 5+ notebooks (model inference, external secrets integration)
- **Purpose**: Model inference and external integrations

## Decision

Configure a separate GitHub Actions job for **E2E testing against a live OpenShift 4.18 cluster**.

### Workflow Steps

1. **Authentication**: Pull OpenShift API token from GitHub Secret (`OPENSHIFT_TOKEN`)
2. **Login**: Execute `oc login` to pre-provisioned OCP 4.18 cluster
3. **Operator Installation**: Deploy operator via manifests/bundle
4. **Test Execution**: Clone test-notebooks repo and run validation jobs
5. **Results Validation**: Verify notebook execution results
6. **Cleanup**: Remove test resources
7. **Reporting**: Report test results to GitHub Actions

### Implementation Details

```yaml
name: E2E Tests - OpenShift

on:
  push:
    branches: [main, release-*]
  pull_request:
    branches: [main, release-*]

jobs:
  e2e-openshift:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout operator code
        uses: actions/checkout@v4

      - name: Install OpenShift CLI
        run: |
          curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz
          tar -xzf openshift-client-linux.tar.gz
          sudo mv oc /usr/local/bin/
          oc version

      - name: Login to OpenShift cluster
        env:
          OPENSHIFT_TOKEN: ${{ secrets.OPENSHIFT_TOKEN }}
          OPENSHIFT_SERVER: ${{ secrets.OPENSHIFT_SERVER }}
        run: |
          oc login --token=$OPENSHIFT_TOKEN --server=$OPENSHIFT_SERVER --insecure-skip-tls-verify=true

      - name: Create test namespace
        run: |
          oc new-project jupyter-notebook-validator-e2e-${{ github.run_id }} || true
          oc project jupyter-notebook-validator-e2e-${{ github.run_id }}

      - name: Install operator
        run: |
          make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:${{ github.sha }}

      - name: Wait for operator ready
        run: |
          oc wait --for=condition=available --timeout=300s \
            deployment/notebook-validator-controller-manager \
            -n jupyter-notebook-validator-operator

      - name: Clone test notebooks
        run: |
          git clone https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
          cd jupyter-notebook-validator-test-notebooks

      - name: Setup test infrastructure
        run: |
          # Grant SCC for Tekton builds (Tier 2)
          oc adm policy add-scc-to-user pipelines-scc -z pipeline -n jupyter-notebook-validator-e2e-${{ github.run_id }}

          # Deploy test models for Tier 3
          cd ../jupyter-notebook-validator-test-notebooks/deployments
          ./setup-models.sh

      - name: Run Tier 1 tests (Simple - no builds)
        run: |
          cd ../jupyter-notebook-validator-test-notebooks
          ./scripts/run-tier1-tests.sh

      - name: Run Tier 2 tests (Intermediate - with builds)
        run: |
          ./scripts/run-tier2-tests.sh

      - name: Run Tier 3 tests (Complex - with models)
        run: |
          ./scripts/run-tier3-tests.sh

      - name: Collect test results
        if: always()
        run: |
          oc get notebookvalidationjob -A -o yaml > test-results.yaml
          oc get pod -A -o yaml > pod-status.yaml

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: e2e-test-results
          path: |
            test-results.yaml
            pod-status.yaml

      - name: Cleanup
        if: always()
        run: |
          oc delete project jupyter-notebook-validator-e2e-${{ github.run_id }}
```

### Security Configuration

**GitHub Secrets Required**:
- `OPENSHIFT_TOKEN`: Service account token with cluster-admin or appropriate RBAC
- `OPENSHIFT_SERVER`: OpenShift API server URL (e.g., `https://api.cluster.example.com:6443`)

**Token Rotation**:
- Rotate tokens every 90 days
- Use service account tokens (not user tokens)
- Limit token scope to test namespace

## Consequences

### Positive

- ✅ **Full-Stack Validation**: Tests complete operator workflow end-to-end
- ✅ **OpenShift Features**: Validates ImageStreams, Tekton, security contexts
- ✅ **Real-World Issues**: Catches issues that local testing cannot replicate
- ✅ **Confident Releases**: Higher confidence for production deployments
- ✅ **Automated Testing**: Eliminates manual E2E testing burden
- ✅ **External Test Suite**: Leverages comprehensive test notebook repository

### Negative

- ❌ **Cluster Dependency**: Requires stable access to shared OCP cluster
- ❌ **Security Management**: Secure token management and rotation required
- ❌ **Longer Runtime**: E2E tests take 10-15 minutes vs. 2-3 minutes for unit tests
- ❌ **Test Flakiness**: Potential for flakiness due to cluster state
- ❌ **Infrastructure Cost**: Cost of maintaining test cluster

### Neutral

- ⚠️ **Maintenance**: Requires monitoring of test cluster health
- ⚠️ **Documentation**: Clear documentation of E2E test setup required

## Alternatives Considered

### Alternative 1: Mocked OCP API
- **Pros**: Faster, no cluster dependency
- **Cons**: Does not catch cluster-specific issues
- **Rejected**: Insufficient validation coverage

### Alternative 2: Self-Provision OCP Cluster Per Test
- **Pros**: More isolated, no shared cluster issues
- **Cons**: 20-30 minute cluster provisioning time, high infrastructure cost
- **Rejected**: Too slow for CI/CD pipeline

### Alternative 3: Manual E2E Testing Only
- **Pros**: No CI/CD complexity
- **Cons**: Slower feedback loop, human error prone, not scalable
- **Rejected**: Does not meet automation requirements

## Implementation Plan

### Phase 1: Infrastructure Setup (Week 1)
1. Provision dedicated OpenShift 4.18 test cluster
2. Create service account with appropriate RBAC
3. Generate and store OpenShift token in GitHub Secrets
4. Document cluster access and token rotation process

### Phase 2: Workflow Development (Week 2)
5. Create `.github/workflows/e2e-openshift.yaml`
6. Implement operator installation steps
7. Integrate test notebook repository
8. Add result collection and reporting

### Phase 3: Test Integration (Week 3)
9. Run Tier 1 tests (simple notebooks)
10. Add Tier 2 tests (intermediate notebooks)
11. Add Tier 3 tests (advanced notebooks)
12. Validate test results and fix issues

### Phase 4: Documentation and Training (Week 4)
13. Update `docs/INTEGRATION_TESTING.md`
14. Document troubleshooting procedures
15. Train team on E2E testing workflow
16. Establish monitoring and alerting

## Verification

### Success Criteria
- [ ] E2E workflow runs successfully on every PR
- [ ] All tier 1-3 tests pass
- [ ] Test results uploaded as artifacts
- [ ] Cleanup executes successfully
- [ ] Documentation complete
- [ ] Team trained

### Testing
```bash
# Verify operator deployment
oc get deployment -n jupyter-notebook-validator-operator

# Verify test execution
oc get notebookvalidationjob -A

# Check test results
oc get notebookvalidationjob <name> -o jsonpath='{.status.phase}'
```

## References

- [Test Notebooks Repository](https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks) (private)
- [OpenShift CI/CD Best Practices](https://docs.openshift.com/container-platform/4.18/cicd/index.html)
- [GitHub Actions OpenShift Integration](https://github.com/redhat-actions)
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- ADR-034: Dual Testing Strategy with Kind and OpenShift
- ADR-035: Test Tier Organization and Scope
- ADR-036: Private Test Repository Strategy

## Notes

- Test cluster should be OpenShift 4.18.21 to match production
- Service account token should have limited scope (test namespace only)
- Test notebooks repository should be pinned to specific commit/tag for reproducibility
- E2E tests should run on every PR and merge to main/release branches
- Consider adding performance benchmarks to E2E tests

