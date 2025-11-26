# End-to-End Testing Guide

This guide covers end-to-end (E2E) testing for the Jupyter Notebook Validator Operator, including build integration workflows.

## Overview

E2E tests validate complete workflows from start to finish:

1. **Build Workflow Tests**: Test S2I and Tekton build creation, monitoring, and completion
2. **Complete Validation Tests**: Test full notebook validation with build integration
3. **Error Handling Tests**: Test failure scenarios and recovery
4. **Performance Tests**: Test concurrent builds and resource usage

## Prerequisites

### Required

- Access to OpenShift 4.18+ or Kubernetes 1.31+ cluster
- `kubectl` or `oc` CLI configured
- Go 1.21+ installed
- Cluster admin permissions (for creating namespaces and resources)

### Optional (for specific tests)

- **S2I Tests**: OpenShift cluster with BuildConfig API
- **Tekton Tests**: Tekton Pipelines installed (v0.65.0+)
- **Image Registry**: Access to push/pull images (internal or external)

## Running E2E Tests

### Quick Start

```bash
# Run all E2E tests
./scripts/run-e2e-tests.sh
```

### Manual Test Execution

```bash
# Run S2I workflow test
go test -v -tags=e2e -timeout=15m ./pkg/build/... -run "TestE2ES2IWorkflow" -count=1

# Run Tekton workflow test
go test -v -tags=e2e -timeout=15m ./pkg/build/... -run "TestE2ETektonWorkflow" -count=1

# Run all E2E tests
go test -v -tags=e2e -timeout=15m ./pkg/build/... -count=1
```

### Environment Variables

```bash
# Custom test namespace
export TEST_NAMESPACE="my-e2e-tests"

# Custom timeout
export TEST_TIMEOUT="20m"

# Skip cleanup (for debugging)
export SKIP_CLEANUP="true"

# Run tests
./scripts/run-e2e-tests.sh
```

## Test Scenarios

### 1. S2I Build Workflow Test

**Test**: `TestE2ES2IWorkflow`

**What it tests**:
- S2I availability detection
- BuildConfig creation
- Build triggering
- Build status monitoring
- Build completion
- Image reference retrieval
- Resource cleanup

**Expected duration**: 5-10 minutes

**Success criteria**:
- ✅ S2I detected on cluster
- ✅ BuildConfig created successfully
- ✅ Build completes with status "Complete"
- ✅ Image reference is set
- ✅ Resources cleaned up

### 2. Tekton Build Workflow Test

**Test**: `TestE2ETektonWorkflow`

**What it tests**:
- Tekton availability detection
- Pipeline creation
- PipelineRun creation
- Build status monitoring
- Build completion
- Image reference retrieval
- Resource cleanup

**Expected duration**: 5-10 minutes

**Success criteria**:
- ✅ Tekton detected on cluster
- ✅ Pipeline and PipelineRun created successfully
- ✅ Build completes with status "Complete"
- ✅ Image reference is set
- ✅ Resources cleaned up

### 3. Complete Notebook Validation with S2I

**Test**: `TestE2ECompleteNotebookValidationWithS2I`

**Status**: 🚧 Pending controller integration

**What it will test**:
- NotebookValidationJob creation with build enabled
- Automatic build triggering
- Build completion waiting
- Validation pod creation with built image
- Notebook execution
- Results collection
- Status updates

**Expected duration**: 10-15 minutes

### 4. Complete Notebook Validation with Tekton

**Test**: `TestE2ECompleteNotebookValidationWithTekton`

**Status**: 🚧 Pending controller integration

**What it will test**:
- NotebookValidationJob creation with build enabled
- Automatic build triggering via Tekton
- Build completion waiting
- Validation pod creation with built image
- Notebook execution
- Results collection
- Status updates

**Expected duration**: 10-15 minutes

## Test Infrastructure

### Test Namespace

E2E tests create a dedicated namespace: `notebook-validator-e2e-test`

This namespace is:
- Created automatically before tests
- Cleaned up automatically after tests (unless `SKIP_CLEANUP=true`)
- Isolated from other test runs

### Test Resources

Each test creates:
- NotebookValidationJob CR
- Build resources (BuildConfig/Build or Pipeline/PipelineRun)
- Pods (for validation)
- Secrets (for credentials)
- ConfigMaps (for configuration)

All resources are labeled with:
```yaml
labels:
  mlops.redhat.com/notebook-validation: "true"
  test-run: "e2e"
```

### Cleanup

Cleanup happens automatically:
1. After each test completes
2. On test failure
3. On script exit (via trap)

To skip cleanup for debugging:
```bash
export SKIP_CLEANUP=true
./scripts/run-e2e-tests.sh
```

## Troubleshooting

### Test Timeout

**Problem**: Tests timeout before completion

**Solution**:
```bash
# Increase timeout
export TEST_TIMEOUT="30m"
./scripts/run-e2e-tests.sh
```

### Build Failures

**Problem**: Builds fail during E2E tests

**Debugging**:
```bash
# Check build logs (S2I)
oc logs -f build/e2e-s2i-test-build-1

# Check build logs (Tekton)
tkn pipelinerun logs e2e-tekton-test-build -f

# Check build status
oc get builds
oc get pipelineruns
```

### Namespace Stuck in Terminating

**Problem**: Test namespace stuck in "Terminating" state

**Solution**:
```bash
# Force delete namespace
kubectl delete namespace notebook-validator-e2e-test --grace-period=0 --force

# Or wait for it to terminate
kubectl wait --for=delete namespace/notebook-validator-e2e-test --timeout=5m
```

### Permission Errors

**Problem**: Tests fail with permission errors

**Solution**:
```bash
# Ensure you have cluster admin permissions
oc adm policy add-cluster-role-to-user cluster-admin $(oc whoami)

# Or use a service account with sufficient permissions
```

### Image Registry Access

**Problem**: Builds fail to push images

**Solution**:
```bash
# For OpenShift internal registry
oc policy add-role-to-user registry-editor system:serviceaccount:notebook-validator-e2e-test:default

# For external registry, create image pull secret
kubectl create secret docker-registry regcred \
  --docker-server=<registry> \
  --docker-username=<username> \
  --docker-password=<password> \
  -n notebook-validator-e2e-test
```

## CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on:
  pull_request:
    branches: [ main, release-* ]
  push:
    branches: [ main, release-* ]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Set up test cluster
        run: |
          # Set up kind/minikube or connect to test cluster
          
      - name: Run E2E tests
        run: ./scripts/run-e2e-tests.sh
        env:
          TEST_TIMEOUT: "20m"
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        TEST_NAMESPACE = "notebook-validator-e2e-${BUILD_NUMBER}"
        TEST_TIMEOUT = "20m"
    }
    
    stages {
        stage('E2E Tests') {
            steps {
                sh './scripts/run-e2e-tests.sh'
            }
        }
    }
    
    post {
        always {
            sh 'kubectl delete namespace ${TEST_NAMESPACE} --ignore-not-found=true'
        }
    }
}
```

## Performance Benchmarks

### Expected Test Times

| Test | Duration | Notes |
|------|----------|-------|
| S2I Workflow | 5-10 min | Depends on image size and registry speed |
| Tekton Workflow | 5-10 min | Depends on image size and registry speed |
| Complete Validation (S2I) | 10-15 min | Includes build + notebook execution |
| Complete Validation (Tekton) | 10-15 min | Includes build + notebook execution |

### Resource Usage

| Resource | Typical Usage | Peak Usage |
|----------|---------------|------------|
| CPU | 1-2 cores | 4 cores |
| Memory | 2-4 GB | 8 GB |
| Storage | 5-10 GB | 20 GB |

## Next Steps

After E2E tests pass:

1. **Controller Integration**: Integrate build strategies into controller reconciliation loop
2. **Error Handling Tests**: Test failure scenarios and recovery
3. **Concurrent Build Tests**: Test multiple builds running simultaneously
4. **Performance Tests**: Measure build times and resource usage
5. **Load Tests**: Test operator under high load

## Related Documentation

- [Integration Testing Guide](INTEGRATION_TESTING.md)
- [Build Strategy Implementation](BUILD_STRATEGY_IMPLEMENTATION.md)
- [ADR-023: S2I Build Integration](adr/ADR-023-s2i-build-integration.md)
- [ADR-025: Community Build Methods](adr/ADR-025-community-build-methods.md)

