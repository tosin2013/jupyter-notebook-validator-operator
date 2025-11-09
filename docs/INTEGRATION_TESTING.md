# Integration Testing Guide

This guide explains how to run integration tests for the Jupyter Notebook Validator Operator against a real OpenShift or Kubernetes cluster.

## Overview

Integration tests validate that the build strategies work correctly with actual cluster resources:

- **S2I Strategy**: Tests OpenShift Source-to-Image builds
- **Tekton Strategy**: Tests Tekton Pipeline builds
- **Strategy Detection**: Tests automatic detection of available build backends
- **Strategy Selection**: Tests automatic strategy selection based on cluster capabilities

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

### GitHub Actions Example

```yaml
name: Integration Tests

on:
  push:
    branches: [ release-4.18 ]
  pull_request:
    branches: [ release-4.18 ]

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Set up OpenShift CLI
        run: |
          curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux.tar.gz
          tar xvf openshift-client-linux.tar.gz
          sudo mv oc /usr/local/bin/
      
      - name: Configure kubeconfig
        run: |
          echo "${{ secrets.KUBECONFIG }}" > kubeconfig
          export KUBECONFIG=kubeconfig
      
      - name: Run Integration Tests
        run: |
          export KUBECONFIG=kubeconfig
          ./scripts/run-integration-tests.sh
        env:
          SKIP_S2I_BUILD_TEST: "false"
          SKIP_TEKTON_BUILD_TEST: "false"
```

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

