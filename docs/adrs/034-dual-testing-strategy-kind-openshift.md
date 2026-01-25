# ADR-034: Dual Testing Strategy with Kind and OpenShift

**Status**: Implemented  
**Date**: 2025-11-11  
**Updated**: 2026-01-24  
**Authors**: Sophia (AI Assistant), User Feedback  
**Related**: ADR-032 (CI Testing), ADR-033 (E2E Testing), ADR-035 (Test Tier Organization)

## Context

The Jupyter Notebook Validator Operator needs comprehensive testing across different environments to ensure reliability and catch platform-specific issues early in the development cycle.

### Current Situation

- **Manual Testing**: E2E tests performed manually on OpenShift cluster
- **No Local Testing**: Developers cannot test locally before pushing
- **Slow Feedback Loop**: Waiting for CI/CD to catch basic issues
- **OpenShift-Specific Features**: Some features only work on OpenShift (S2I, Tekton, SCCs)

### Problem Statement

We need a testing strategy that provides:
1. **Fast Local Feedback**: Developers can test basic functionality locally
2. **Comprehensive Validation**: Full OpenShift feature testing in CI/CD
3. **Cost Efficiency**: Minimize expensive OpenShift cluster usage
4. **Clear Separation**: Know what can be tested locally vs. what requires OpenShift

### Test Repository

External test suite: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks` (private)

**Current Structure**:
- `notebooks/tier1-simple/` - 4 basic notebooks (< 30s execution)
- `notebooks/tier2-intermediate/` - Empty (planned for data science)
- `notebooks/tier3-complex/` - Empty (planned for model training)
- `model-training/` - ML training notebooks
- `model-validation/` - Model inference testing notebooks
- `eso-integration/` - External secrets testing notebooks
- `deployments/` - Model deployment manifests
- `lib/` - Shared Python libraries

## Decision

Implement a **dual testing strategy** using Kind for local development and OpenShift for comprehensive CI/CD testing.

### Testing Environments

#### Environment 1: Kind (Local Development)
- **Purpose**: Fast feedback for basic functionality
- **Scope**: Tier 1 tests only (simple notebooks, no builds, no models)
- **Runtime**: < 2 minutes
- **Use Case**: Pre-commit validation, rapid iteration

#### Environment 2: OpenShift (CI/CD)
- **Purpose**: Comprehensive validation of all features
- **Scope**: All tiers (Tier 1, 2, 3)
- **Runtime**: 10-15 minutes
- **Use Case**: PR validation, release testing

### Test Tier Mapping

| Tier | Kind | OpenShift | Features Tested |
|------|------|-----------|-----------------|
| **Tier 1** | ✅ Yes | ✅ Yes | Basic notebook execution, git clone, simple validation |
| **Tier 2** | ❌ No | ✅ Yes | S2I builds, Tekton builds, dependencies, model training |
| **Tier 3** | ❌ No | ✅ Yes | Model inference, external secrets, KServe, OpenShift AI |

### Implementation Details

#### Kind Testing Setup

```bash
#!/bin/bash
# scripts/test-local-kind.sh

set -e

echo "=== Setting up Kind cluster for local testing ==="

# Create Kind cluster with Kubernetes 1.31.10
kind create cluster --name jupyter-validator-test --image kindest/node:v1.31.10

# Install cert-manager (required for webhooks)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager

# Install operator
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Wait for operator ready
kubectl wait --for=condition=available --timeout=300s \
  deployment/notebook-validator-controller-manager \
  -n jupyter-notebook-validator-operator

# Create test namespace
kubectl create namespace e2e-tests

# Create git credentials secret (for private test repo)
kubectl create secret generic git-https-credentials \
  --from-literal=username=${GIT_USERNAME} \
  --from-literal=password=${GIT_TOKEN} \
  -n e2e-tests

# Run Tier 1 tests
echo "=== Running Tier 1 tests on Kind ==="
cd ../jupyter-notebook-validator-test-notebooks
./scripts/run-tier1-tests.sh

# Cleanup
kind delete cluster --name jupyter-validator-test
```

#### OpenShift Testing Setup

```yaml
# .github/workflows/e2e-tests.yaml

name: E2E Tests

on:
  push:
    branches: [main, release-*]
  pull_request:
    branches: [main, release-*]

jobs:
  # Job 1: Fast local testing with Kind
  tier1-kind:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - name: Checkout operator
        uses: actions/checkout@v4

      - name: Setup Kind
        uses: helm/kind-action@v1
        with:
          version: v0.20.0
          node_image: kindest/node:v1.31.10

      - name: Run Tier 1 tests
        env:
          GIT_USERNAME: ${{ secrets.TEST_REPO_USERNAME }}
          GIT_TOKEN: ${{ secrets.TEST_REPO_TOKEN }}
        run: |
          ./scripts/test-local-kind.sh

  # Job 2: Comprehensive OpenShift testing
  all-tiers-openshift:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - name: Checkout operator
        uses: actions/checkout@v4

      - name: Install OpenShift CLI
        run: |
          curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz
          tar -xzf openshift-client-linux.tar.gz
          sudo mv oc /usr/local/bin/

      - name: Login to OpenShift
        env:
          OPENSHIFT_TOKEN: ${{ secrets.OPENSHIFT_TOKEN }}
          OPENSHIFT_SERVER: ${{ secrets.OPENSHIFT_SERVER }}
        run: |
          oc login --token=$OPENSHIFT_TOKEN --server=$OPENSHIFT_SERVER --insecure-skip-tls-verify=true

      - name: Create test namespace
        run: |
          oc new-project jupyter-validator-e2e-${{ github.run_id }} || true

      - name: Install operator
        run: |
          make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:${{ github.sha }}

      - name: Setup test infrastructure
        run: |
          # Grant SCC for Tekton builds
          oc adm policy add-scc-to-user pipelines-scc -z pipeline -n jupyter-validator-e2e-${{ github.run_id }}
          
          # Deploy test models for Tier 3
          cd ../jupyter-notebook-validator-test-notebooks/deployments
          ./setup-models.sh

      - name: Run Tier 1 tests
        run: |
          cd ../jupyter-notebook-validator-test-notebooks
          ./scripts/run-tier1-tests.sh

      - name: Run Tier 2 tests (builds)
        run: |
          ./scripts/run-tier2-tests.sh

      - name: Run Tier 3 tests (models)
        run: |
          ./scripts/run-tier3-tests.sh

      - name: Collect results
        if: always()
        run: |
          oc get notebookvalidationjob -A -o yaml > test-results.yaml

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: e2e-test-results
          path: test-results.yaml

      - name: Cleanup
        if: always()
        run: |
          oc delete project jupyter-validator-e2e-${{ github.run_id }}
```

### Developer Workflow

#### Local Development (Before Push)
```bash
# 1. Make code changes
vim pkg/controller/notebookvalidationjob_controller.go

# 2. Run unit tests
make test

# 3. Run local Kind tests (Tier 1 only)
./scripts/test-local-kind.sh

# 4. Commit and push
git commit -am "Fix: Update validation logic"
git push origin feature-branch
```

#### CI/CD Validation (After Push)
1. **Tier 1 Kind Job** runs (2-3 minutes) - Fast feedback
2. **All Tiers OpenShift Job** runs (10-15 minutes) - Comprehensive validation
3. Both jobs must pass before merge

## Consequences

### Positive

- ✅ **Fast Local Feedback**: Developers get feedback in < 2 minutes
- ✅ **Reduced CI/CD Load**: Kind tests catch basic issues before OpenShift tests
- ✅ **Cost Efficiency**: Minimize expensive OpenShift cluster usage
- ✅ **Clear Separation**: Developers know what can be tested locally
- ✅ **Comprehensive Coverage**: OpenShift tests validate all features
- ✅ **Parallel Testing**: Kind and OpenShift tests can run in parallel

### Negative

- ❌ **Maintenance Overhead**: Two testing environments to maintain
- ❌ **Setup Complexity**: Developers need to install Kind locally
- ❌ **Limited Local Testing**: Tier 2/3 features cannot be tested locally
- ❌ **Potential Drift**: Kind and OpenShift environments may diverge

### Neutral

- ⚠️ **Documentation**: Clear documentation needed for both environments
- ⚠️ **Training**: Developers need to understand which tests run where

## Alternatives Considered

### Alternative 1: OpenShift Only
- **Pros**: Single environment, no drift
- **Cons**: Slow feedback (10-15 min), expensive, no local testing
- **Rejected**: Too slow for rapid iteration

### Alternative 2: Kind Only
- **Pros**: Fast, cheap, local testing
- **Cons**: Cannot test OpenShift-specific features (S2I, Tekton, SCCs)
- **Rejected**: Insufficient coverage

### Alternative 3: Minikube + OpenShift
- **Pros**: Similar to Kind approach
- **Cons**: Minikube is slower and less reliable than Kind
- **Rejected**: Kind is the industry standard for local Kubernetes testing

## Implementation Plan

### Phase 1: Kind Testing Infrastructure (Week 1)
1. ✅ Create `scripts/test-local-kind.sh`
2. ✅ Document Kind setup in `docs/DEVELOPMENT.md`
3. ✅ Test Tier 1 notebooks on Kind
4. ✅ Validate git authentication works

### Phase 2: CI/CD Integration (Week 2)
5. Create `.github/workflows/e2e-tests.yaml`
6. Add Tier 1 Kind job
7. Add All Tiers OpenShift job
8. Configure GitHub secrets

### Phase 3: Documentation and Training (Week 3)
9. Update `docs/TESTING.md` with dual strategy
10. Create troubleshooting guide
11. Train team on local testing workflow
12. Document CI/CD pipeline

## Verification

### Success Criteria
- [ ] Kind tests run locally in < 2 minutes
- [ ] OpenShift tests run in CI/CD in < 15 minutes
- [ ] Both test suites pass on every PR
- [ ] Documentation complete
- [ ] Team trained on local testing

### Testing
```bash
# Verify Kind setup
kind create cluster --name test
kubectl cluster-info

# Verify operator deployment
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Verify Tier 1 tests
cd ../jupyter-notebook-validator-test-notebooks
./scripts/run-tier1-tests.sh
```

## References

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Test Notebooks Repository](https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks)
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- ADR-033: End-to-End Testing Against Live OpenShift Cluster
- ADR-035: Test Tier Organization and Scope

## Notes

- Kind cluster uses Kubernetes 1.31.10 to match OpenShift 4.18
- Tier 1 tests should be platform-agnostic (work on both Kind and OpenShift)
- Tier 2/3 tests require OpenShift-specific features
- Git credentials required for private test repository access
- Consider adding performance benchmarks to track test execution time

