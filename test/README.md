# Integration Test Suite

This directory contains integration tests for the Jupyter Notebook Validator Operator.

## Test Coverage

The integration test suite validates the following features:

1. **ESO (External Secrets Operator) Integration**
   - Verifies ESO installation and CRD availability
   - Tests secret synchronization from external stores
   - Validates `envFrom` field injection in NotebookValidationJob
   - Tests against 4 secret types: AWS, Database, MLflow, API keys

2. **KServe Model-Aware Validation**
   - Verifies KServe installation and CRD availability
   - Tests platform detection for KServe
   - Validates model validation configuration
   - Tests prediction validation against InferenceServices

3. **OpenShift AI Model-Aware Validation**
   - Verifies OpenShift AI installation
   - Tests platform detection for OpenShift AI
   - Validates model validation with OpenShift AI models
   - Tests ServingRuntime integration

## Prerequisites

### Required Tools

- `oc` - OpenShift CLI
- `jq` - JSON processor
- `bash` - Shell (version 4.0+)

### Required Cluster Components

- **OpenShift/Kubernetes Cluster**: Access to a running cluster
- **Jupyter Notebook Validator Operator**: Installed and running
- **External Secrets Operator** (for ESO tests): v0.11.0+
- **KServe** (for KServe tests): v0.10.0+ (optional, tests will skip if not installed)
- **OpenShift AI** (for OpenShift AI tests): Latest version (optional, tests will skip if not installed)

## Running the Tests

### Run All Tests

```bash
./test/integration-test-suite.sh
```

### Run with Custom Namespace

```bash
NAMESPACE=my-namespace ./test/integration-test-suite.sh
```

### Run with Custom Timeout

```bash
TIMEOUT=600 ./test/integration-test-suite.sh  # 10 minutes
```

## Test Structure

### Test 1: ESO Integration

**File**: `config/samples/eso-fake-secretstore.yaml`

**What it tests**:
- ESO SecretStore with Fake provider
- 4 ExternalSecrets syncing credentials
- NotebookValidationJob with `envFrom` field
- Secret injection into validation pods

**Expected outcome**:
- All 4 secrets created successfully
- NotebookValidationJob has 4 `envFrom` references
- Job completes successfully

### Test 2: KServe Model Validation

**File**: `config/samples/model-validation-kserve.yaml`

**What it tests**:
- Platform detection for KServe
- Model validation against InferenceServices
- Prediction consistency validation
- RBAC permissions for model access

**Expected outcome**:
- Platform detected as `kserve`
- Model validation runs (Phase 1 and Phase 2)
- Job completes successfully

**Note**: If KServe is not installed, the test will be skipped with a warning.

### Test 3: OpenShift AI Model Validation

**File**: `config/samples/model-validation-openshift-ai.yaml`

**What it tests**:
- Platform detection for OpenShift AI
- Model validation with OpenShift AI models
- ServingRuntime integration
- RBAC permissions for OpenShift AI resources

**Expected outcome**:
- Platform detected as `openshift-ai`
- Model validation runs
- Job completes successfully

**Note**: If OpenShift AI is not installed, the test will be skipped with a warning.

## Test Output

The test suite provides colored output:

- **BLUE**: Informational messages
- **GREEN**: Success messages
- **YELLOW**: Warning messages (e.g., skipped tests)
- **RED**: Error messages

### Example Output

```
========================================
Jupyter Notebook Validator Operator
Integration Test Suite
========================================

[INFO] Checking prerequisites...
[SUCCESS] oc command found
[SUCCESS] jq command found
[SUCCESS] Connected to OpenShift cluster
[SUCCESS] Namespace 'mlops' exists
[SUCCESS] CRD applied

[INFO] Starting integration tests...

========================================
Test 1: ESO (External Secrets Operator) Integration
========================================
[INFO] Checking ESO installation...
[SUCCESS] ESO CRD found
[INFO] Applying ESO test resources...
[SUCCESS] ESO test resources applied
[INFO] Waiting for ExternalSecrets to sync...
[SUCCESS] Secret 'aws-credentials-eso' created successfully
[SUCCESS] Secret 'database-config-eso' created successfully
[SUCCESS] Secret 'mlflow-credentials-eso' created successfully
[SUCCESS] Secret 'api-keys-eso' created successfully
[INFO] Waiting for job 'eso-integration-test' to complete (timeout: 300s)...
[SUCCESS] Job 'eso-integration-test' completed successfully
[SUCCESS] envFrom field contains 4 secret references
[SUCCESS] ESO integration test passed

========================================
Test Summary
========================================
Tests Run:    3
Tests Passed: 3
Tests Failed: 0

[SUCCESS] All tests passed!
```

## Troubleshooting

### Test Fails with "ESO CRD not found"

**Solution**: Install External Secrets Operator:

```bash
# Install ESO using Helm
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace

# Or install using OperatorHub (OpenShift)
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: external-secrets-operator
  namespace: openshift-operators
spec:
  channel: stable
  name: external-secrets-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
EOF
```

### Test Fails with "KServe CRD not found"

**Solution**: The test will skip KServe tests if KServe is not installed. To install KServe:

```bash
# Install KServe
kubectl apply -f https://github.com/kserve/kserve/releases/download/v0.11.0/kserve.yaml
```

### Test Fails with "Job timed out"

**Solution**: Increase the timeout:

```bash
TIMEOUT=600 ./test/integration-test-suite.sh  # 10 minutes
```

### Test Fails with "Not connected to OpenShift cluster"

**Solution**: Login to your OpenShift cluster:

```bash
oc login https://api.your-cluster.com:6443 --token=your-token
```

## Cleanup

To clean up test resources:

```bash
# Delete ESO test resources
oc delete -f config/samples/eso-fake-secretstore.yaml

# Delete KServe test resources
oc delete -f config/samples/model-validation-kserve.yaml

# Delete OpenShift AI test resources
oc delete -f config/samples/model-validation-openshift-ai.yaml

# Delete namespace (if desired)
oc delete namespace mlops
```

## CI/CD Integration

The integration test suite can be integrated into CI/CD pipelines:

### GitHub Actions Example

```yaml
name: Integration Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install OpenShift CLI
        run: |
          curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux.tar.gz
          tar -xzf openshift-client-linux.tar.gz
          sudo mv oc /usr/local/bin/
      
      - name: Install jq
        run: sudo apt-get install -y jq
      
      - name: Login to OpenShift
        run: oc login ${{ secrets.OPENSHIFT_SERVER }} --token=${{ secrets.OPENSHIFT_TOKEN }}
      
      - name: Run Integration Tests
        run: ./test/integration-test-suite.sh
```

## Contributing

When adding new tests:

1. Add a new test function following the naming convention: `test_<feature_name>()`
2. Use the helper functions: `log_info()`, `log_success()`, `log_error()`, `log_warning()`
3. Increment test counters appropriately
4. Add cleanup logic using `cleanup_test()`
5. Update this README with the new test documentation

## Related Documentation

- [ADR-014: Notebook Credential Injection Strategy](../docs/adrs/014-notebook-credential-injection-strategy.md)
- [ADR-015: Environment Variable Pattern](../docs/adrs/015-environment-variable-pattern-for-notebook-credentials.md)
- [ADR-016: External Secrets Operator Integration](../docs/adrs/016-external-secrets-operator-integration.md)
- [ADR-020: Model-Aware Validation Strategy](../docs/adrs/020-model-aware-validation-strategy.md)
- [ESO Integration Documentation](../docs/ESO_INTEGRATION_COMPLETE.md)

