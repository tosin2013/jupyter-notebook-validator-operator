# Integration Testing Guide

This guide explains how to run integration tests for the Jupyter Notebook Validator Operator against a real OpenShift or Kubernetes cluster.

## Overview

Integration tests validate that the build strategies work correctly with actual cluster resources:

- **S2I Strategy**: Tests OpenShift Source-to-Image builds
- **Tekton Strategy**: Tests Tekton Pipeline builds
- **Strategy Detection**: Tests automatic detection of available build backends
- **Strategy Selection**: Tests automatic strategy selection based on cluster capabilities

## Related ADRs

- **[ADR-032](adrs/032-github-actions-ci-testing-kubernetes-version.md)**: GitHub Actions CI Testing Against Kubernetes 1.31.10
- **[ADR-033](adrs/033-e2e-testing-live-openshift-cluster.md)**: End-to-End Testing Against Live OpenShift Cluster
- **[ADR-031](adrs/031-tekton-build-dockerfile-vs-base-image.md)**: Tekton Build Dockerfile vs Base Image Support
- **[ADR-027](adrs/027-s2i-build-strategy-for-git-integration.md)**: S2I Build Strategy for Git Integration

## Prerequisites

### Required

1. **Cluster Access**: Access to an OpenShift 4.18+ or Kubernetes 1.31+ cluster
2. **CLI Tools**: `oc` (OpenShift) or `kubectl` (Kubernetes)
3. **Kubeconfig**: Valid kubeconfig with cluster admin or sufficient permissions
4. **Go**: Go 1.21+ for running tests

### Permissions Required

The test user needs permissions to:
- Create/delete namespaces
- Create/delete BuildConfigs and Builds (for S2I tests)
- Create/delete Pipelines and PipelineRuns (for Tekton tests)
- List cluster resources for detection

## Quick Start

### Using the Test Runner Script

The easiest way to run integration tests:

```bash
# Run all integration tests
./scripts/run-integration-tests.sh

# Run with custom namespace
TEST_NAMESPACE=my-test-ns ./scripts/run-integration-tests.sh

# Skip build tests (only run detection tests)
SKIP_S2I_BUILD_TEST=true SKIP_TEKTON_BUILD_TEST=true ./scripts/run-integration-tests.sh
```

### Manual Test Execution

Run integration tests directly with Go:

```bash
# Run all integration tests
go test -v -tags=integration -timeout=10m ./pkg/build/... -count=1

# Run specific test
go test -v -tags=integration -timeout=10m ./pkg/build/... -run TestIntegrationS2IDetection -count=1

# Run with environment variables
SKIP_S2I_BUILD_TEST=true go test -v -tags=integration -timeout=10m ./pkg/build/... -count=1
```

## Test Categories

### 1. Detection Tests

These tests verify that the operator can detect available build strategies:

- `TestIntegrationS2IDetection`: Detects if OpenShift S2I is available
- `TestIntegrationTektonDetection`: Detects if Tekton Pipelines is available
- `TestIntegrationStrategyRegistry`: Tests strategy registry and detection

**Expected Results:**
- On OpenShift: S2I should be detected
- On Kubernetes with Tekton: Tekton should be detected
- On vanilla Kubernetes: No strategies detected (expected)

### 2. Build Creation Tests

These tests create actual builds on the cluster:

- `TestIntegrationS2IBuild`: Creates an S2I build
- `TestIntegrationTektonBuild`: Creates a Tekton pipeline build
- `TestIntegrationBuildWithCustomRegistry`: Tests custom registry configuration

**Note:** Build tests can be skipped using environment variables:
```bash
SKIP_S2I_BUILD_TEST=true
SKIP_TEKTON_BUILD_TEST=true
```

### 3. Strategy Selection Tests

These tests verify automatic strategy selection:

- `TestIntegrationAutoStrategySelection`: Tests auto-selection of available strategies

### 4. Cluster Info Tests

These tests verify cluster connectivity and information:

- `TestIntegrationClusterInfo`: Retrieves and displays cluster information

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TEST_NAMESPACE` | `notebook-validator-integration-test` | Namespace for test resources |
| `SKIP_S2I_BUILD_TEST` | `false` | Skip S2I build creation tests |
| `SKIP_TEKTON_BUILD_TEST` | `false` | Skip Tekton build creation tests |
| `TEST_TIMEOUT` | `10m` | Test timeout duration |

## Test Workflow

1. **Setup Phase**
   - Connect to cluster
   - Create test namespace
   - Register schemes (core, OpenShift, Tekton)
   - Create Kubernetes client

2. **Detection Phase**
   - Detect available build strategies
   - Log cluster capabilities
   - Skip unavailable strategy tests

3. **Build Phase** (if not skipped)
   - Create test NotebookValidationJob
   - Trigger build using strategy
   - Verify build creation
   - Check build status
   - Clean up build resources

4. **Cleanup Phase**
   - Delete test resources
   - Optionally delete test namespace

## Expected Test Results

### On OpenShift 4.18+

```
✅ TestIntegrationS2IDetection - PASS (S2I detected)
✅ TestIntegrationTektonDetection - PASS or SKIP (depends on Tekton installation)
✅ TestIntegrationStrategyRegistry - PASS (S2I available)
✅ TestIntegrationS2IBuild - PASS (build created)
✅ TestIntegrationAutoStrategySelection - PASS (S2I selected)
✅ TestIntegrationClusterInfo - PASS
```

### On Kubernetes 1.31+ with Tekton

```
⚠️  TestIntegrationS2IDetection - PASS (S2I not detected - expected)
✅ TestIntegrationTektonDetection - PASS (Tekton detected)
✅ TestIntegrationStrategyRegistry - PASS (Tekton available)
⏭️  TestIntegrationS2IBuild - SKIP (S2I not available)
✅ TestIntegrationTektonBuild - PASS (build created)
✅ TestIntegrationAutoStrategySelection - PASS (Tekton selected)
✅ TestIntegrationClusterInfo - PASS
```

### On Vanilla Kubernetes

```
⚠️  TestIntegrationS2IDetection - PASS (S2I not detected - expected)
⚠️  TestIntegrationTektonDetection - PASS (Tekton not detected - expected)
⚠️  TestIntegrationStrategyRegistry - PASS (no strategies available - expected)
⏭️  TestIntegrationS2IBuild - SKIP (S2I not available)
⏭️  TestIntegrationTektonBuild - SKIP (Tekton not available)
⚠️  TestIntegrationAutoStrategySelection - PASS (error expected)
✅ TestIntegrationClusterInfo - PASS
```

## Troubleshooting

### Test Failures

**"Failed to get kubeconfig"**
- Ensure `KUBECONFIG` environment variable is set or `~/.kube/config` exists
- Verify kubeconfig is valid: `oc cluster-info` or `kubectl cluster-info`

**"Failed to create namespace"**
- Check permissions: `oc auth can-i create namespace`
- Try using an existing namespace: `TEST_NAMESPACE=existing-ns ./scripts/run-integration-tests.sh`

**"S2I not available"**
- Expected on non-OpenShift clusters
- On OpenShift, check: `oc get buildconfig -A`

**"Tekton not available"**
- Install Tekton: `kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml`
- Check installation: `kubectl get pipelinerun -A`

**"Build creation failed"**
- Check cluster resources: `oc get events -n $TEST_NAMESPACE`
- Verify permissions: `oc auth can-i create build`
- Check logs: `oc logs -n $TEST_NAMESPACE <build-pod>`

### Cleanup Issues

**Namespace stuck in "Terminating"**
```bash
# Force delete namespace
kubectl delete namespace $TEST_NAMESPACE --grace-period=0 --force
```

**Resources not cleaned up**
```bash
# Manual cleanup
oc delete all -l mlops.redhat.com/notebook-validation=true -n $TEST_NAMESPACE
oc delete pipelinerun -l mlops.redhat.com/notebook-validation=true -n $TEST_NAMESPACE
```

## CI/CD Integration

### Testing Strategy

Our CI/CD pipeline implements a two-tier testing approach:

1. **Unit & Integration Tests** (ADR-032): Run on KinD cluster with Kubernetes v1.31.10
2. **End-to-End Tests** (ADR-033): Run on live OpenShift 4.18 cluster

### Tier 1: Unit & Integration Tests (KinD)

**Purpose**: Fast feedback on Kubernetes API compatibility

**Environment**:
- KinD cluster with Kubernetes v1.31.10 (matches OpenShift 4.18.21)
- GitHub Actions runner
- ~5 minute runtime

**Workflow**: `.github/workflows/ci-unit-tests.yaml`

```yaml
name: Unit & Integration Tests

on:
  push:
    branches: [ main, release-* ]
  pull_request:
    branches: [ main, release-* ]

jobs:
  unit-integration-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Create KinD cluster (Kubernetes v1.31.10)
        uses: helm/kind-action@v1
        with:
          version: v0.20.0
          node_image: kindest/node:v1.31.10
          cluster_name: test-cluster

      - name: Verify Kubernetes version
        run: |
          kubectl version --short
          # Expected: Server Version: v1.31.10

      - name: Run unit tests
        run: make test

      - name: Run integration tests
        run: |
          export KUBECONFIG=~/.kube/config
          ./scripts/run-integration-tests.sh
        env:
          SKIP_S2I_BUILD_TEST: "true"  # S2I not available on KinD
          SKIP_TEKTON_BUILD_TEST: "false"

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: unit-test-results
          path: |
            coverage.out
            test-results.xml
```

### Tier 2: End-to-End Tests (OpenShift)

**Purpose**: Validate complete operator workflow on real OpenShift cluster

**Environment**:
- Live OpenShift 4.18.21 cluster
- External test notebooks: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks`
- ~15 minute runtime

**Workflow**: `.github/workflows/e2e-openshift.yaml`

```yaml
name: E2E Tests - OpenShift

on:
  push:
    branches: [ main, release-* ]
  pull_request:
    branches: [ main, release-* ]

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
          oc cluster-info

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

      - name: Run Tier 1 tests (Simple)
        run: |
          cd jupyter-notebook-validator-test-notebooks
          ./scripts/run-e2e-tests.sh tier1

      - name: Run Tier 2 tests (Intermediate)
        run: |
          cd jupyter-notebook-validator-test-notebooks
          ./scripts/run-e2e-tests.sh tier2

      - name: Run Tier 3 tests (Advanced)
        run: |
          cd jupyter-notebook-validator-test-notebooks
          ./scripts/run-e2e-tests.sh tier3

      - name: Collect test results
        if: always()
        run: |
          oc get notebookvalidationjob -A -o yaml > test-results.yaml
          oc get pod -A -o yaml > pod-status.yaml
          oc get pipelinerun.tekton.dev -A -o yaml > tekton-results.yaml

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: e2e-test-results
          path: |
            test-results.yaml
            pod-status.yaml
            tekton-results.yaml

      - name: Cleanup
        if: always()
        run: |
          oc delete project jupyter-notebook-validator-e2e-${{ github.run_id }}
```

### GitHub Secrets Configuration

**Required Secrets**:

| Secret Name | Description | Example |
|-------------|-------------|---------|
| `OPENSHIFT_TOKEN` | Service account token with cluster-admin or appropriate RBAC | `sha256~...` |
| `OPENSHIFT_SERVER` | OpenShift API server URL | `https://api.cluster.example.com:6443` |

**Token Setup**:

```bash
# Create service account
oc create serviceaccount ci-test-runner -n default

# Grant permissions
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:default:ci-test-runner

# Get token
oc create token ci-test-runner -n default --duration=8760h

# Add to GitHub Secrets
# Settings → Secrets and variables → Actions → New repository secret
```

**Token Rotation**: Rotate tokens every 90 days for security.

## Best Practices

1. **Use Dedicated Test Namespace**: Always use a separate namespace for testing
2. **Clean Up Resources**: Always clean up test resources after tests
3. **Skip Long-Running Tests**: Use skip flags for quick validation
4. **Monitor Cluster Resources**: Watch cluster resources during tests
5. **Test on Target Platform**: Test on the same platform as production

## Next Steps

After successful integration tests:

1. Review test results and logs
2. Fix any failures or issues
3. Run tests on different cluster configurations
4. Proceed to end-to-end testing
5. Prepare for production deployment

## Support

For issues or questions:
- Check logs: `oc logs -n $TEST_NAMESPACE <pod-name>`
- Review events: `oc get events -n $TEST_NAMESPACE`
- Check ADR documentation: `docs/adrs/`
- Open an issue on GitHub

