# Jupyter Notebook Validator Operator - Testing Guide

## Overview

This guide explains the three-tier testing strategy for the Jupyter Notebook Validator Operator. Each tier represents increasing complexity and execution time, allowing for comprehensive validation while maintaining fast CI/CD feedback loops.

## Testing Philosophy

Our testing strategy is based on real-world MLOps workflows, progressing from simple data validation to complex model training scenarios. This approach ensures:

1. **Fast Feedback**: Tier 1 tests run in seconds for quick validation
2. **Realistic Scenarios**: Tier 2 tests represent typical data science workflows
3. **Production Readiness**: Tier 3 tests validate production-like ML pipelines
4. **Comprehensive Coverage**: All tiers combined cover the full spectrum of notebook complexity

## Three-Tier Testing Strategy

### 📊 Comparison Matrix

| Aspect | Tier 1: Simple | Tier 2: Intermediate | Tier 3: Complex |
|--------|----------------|----------------------|-----------------|
| **Execution Time** | <30 seconds | 1-5 minutes | 5-15 minutes |
| **Resource Usage** | <100Mi / <100m CPU | <500Mi / <500m CPU | <2Gi / <1000m CPU |
| **Dependencies** | None (pure Python) | CSV files, small datasets | Datasets, model artifacts |
| **Use Case** | Basic validation | Data analysis | Model training |
| **CI/CD Stage** | Every PR | Every PR | Nightly / On-demand |
| **Determinism** | Fully deterministic | Controlled (random seeds) | Controlled (random seeds) |

## Tier 1: Simple Notebooks

**Purpose**: Validate basic operator functionality with fast execution

### Example 1: Hello World

```python
# Validates: Basic execution, assertions, output capture
result = 2 + 2
print(f"Result: {result}")
assert result == 4
```

**What it tests**:
- ✅ Operator can execute basic Python code
- ✅ Cell outputs are captured correctly
- ✅ Assertions work as expected
- ✅ Execution completes successfully

### Example 2: Data Validation

```python
import pandas as pd

# Create sample data
data = {'name': ['Alice', 'Bob'], 'age': [25, 30]}
df = pd.DataFrame(data)

# Validate
assert len(df) == 2
assert df['age'].mean() == 27.5
print("✓ Data validation passed")
```

**What it tests**:
- ✅ Pandas library available
- ✅ DataFrame operations work
- ✅ Statistical computations correct
- ✅ Validation logic executes

### Example 3: Error Handling

```python
# Test graceful error handling
try:
    result = 10 / 0
except ZeroDivisionError as e:
    print(f"✓ Caught expected error: {e}")

# Execution continues
print("Execution continues after handled error")
```

**What it tests**:
- ✅ Operator handles exceptions gracefully
- ✅ Execution continues after handled errors
- ✅ Error messages captured in output

### Running Tier 1 Tests

```bash
# Run all Tier 1 tests
make test-tier1

# Run specific test
kubectl apply -f - <<EOF
apiVersion: mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: test-hello-world
spec:
  notebook:
    git:
      url: https://github.com/your-org/jupyter-notebook-validator-operator.git
      ref: main
      path: tests/notebooks/tier1-simple/01-hello-world.ipynb
  golden:
    path: tests/notebooks/tier1-simple/golden/01-hello-world-golden.ipynb
EOF
```

## Tier 2: Intermediate Notebooks

**Purpose**: Validate realistic data science workflows

### Example 1: Data Analysis

```python
from sklearn.datasets import load_iris
import pandas as pd

# Load dataset
iris = load_iris()
df = pd.DataFrame(iris.data, columns=iris.feature_names)

# Statistical analysis
summary = df.describe()
print(summary)

# Correlation analysis
correlation = df.corr()
print(correlation)

# Assertions
assert df.shape[0] == 150
assert correlation.shape == (4, 4)
```

**What it tests**:
- ✅ Scikit-learn integration
- ✅ Statistical computations
- ✅ Data exploration workflows
- ✅ Multi-cell execution

### Example 2: Feature Engineering

```python
import numpy as np
from sklearn.preprocessing import StandardScaler

np.random.seed(42)  # Reproducibility

# Generate data
data = np.random.randn(1000, 3)

# Scale features
scaler = StandardScaler()
scaled_data = scaler.fit_transform(data)

# Validate scaling
assert scaled_data.mean() < 0.01  # Near zero mean
assert abs(scaled_data.std() - 1.0) < 0.01  # Unit variance
print("✓ Feature scaling validated")
```

**What it tests**:
- ✅ Data preprocessing pipelines
- ✅ Reproducibility with random seeds
- ✅ Numerical validation
- ✅ Moderate execution time handling

### Running Tier 2 Tests

```bash
# Run all Tier 2 tests
make test-tier2

# Run with custom timeout
make test-tier2 TIMEOUT=5m
```

## Tier 3: Complex Notebooks

**Purpose**: Validate production-like ML workflows

### Example: Model Training Pipeline

```python
from sklearn.datasets import load_breast_cancer
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split, cross_val_score
from sklearn.metrics import roc_auc_score
import joblib

np.random.seed(42)

# Load data
data = load_breast_cancer()
X_train, X_test, y_train, y_test = train_test_split(
    data.data, data.target, test_size=0.2, random_state=42
)

# Train model
model = RandomForestClassifier(n_estimators=100, random_state=42)
model.fit(X_train, y_train)

# Cross-validation
cv_scores = cross_val_score(model, X_train, y_train, cv=5, scoring='roc_auc')
print(f"CV ROC-AUC: {cv_scores.mean():.4f}")

# Evaluate
y_pred_proba = model.predict_proba(X_test)[:, 1]
test_auc = roc_auc_score(y_test, y_pred_proba)
print(f"Test ROC-AUC: {test_auc:.4f}")

# Validate performance
assert cv_scores.mean() > 0.90
assert test_auc > 0.90

# Save model
joblib.dump(model, '/tmp/model.joblib')
print("✓ Model training complete")
```

**What it tests**:
- ✅ Full ML training pipeline
- ✅ Cross-validation workflows
- ✅ Model persistence
- ✅ Performance validation
- ✅ Long-running computations
- ✅ Resource management

### Running Tier 3 Tests

```bash
# Run all Tier 3 tests (typically in nightly builds)
make test-tier3

# Run with explicit trigger
git commit -m "feat: new feature [test-tier3]"
```

## Golden Notebook Comparison

Golden notebooks represent the expected output for each test notebook. The operator compares execution results against golden notebooks to detect regressions.

### What Gets Compared

1. **Cell Outputs**: Text output, print statements
2. **Execution Status**: Success/failure of each cell
3. **Assertions**: All assertions must pass
4. **Execution Order**: Cells execute in correct sequence

### What Gets Ignored

1. **Timestamps**: Execution timestamps vary
2. **Memory Addresses**: Object memory addresses differ
3. **Floating Point Precision**: Minor numerical differences (<1e-6)
4. **Matplotlib Figures**: Visual outputs (compared separately)

### Updating Golden Notebooks

```bash
# After intentional changes, update golden notebooks
make update-golden-notebooks TIER=1

# Review changes
git diff tests/notebooks/tier1-simple/golden/

# Commit updated golden notebooks
git add tests/notebooks/*/golden/
git commit -m "test: update golden notebooks for new feature"
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Notebook Tests

on: [push, pull_request]

jobs:
  tier1:
    name: Tier 1 (Fast)
    runs-on: ubuntu-latest
    steps:
      - name: Run Tier 1 tests
        run: make test-tier1
        timeout-minutes: 5

  tier2:
    name: Tier 2 (Intermediate)
    runs-on: ubuntu-latest
    needs: tier1
    steps:
      - name: Run Tier 2 tests
        run: make test-tier2
        timeout-minutes: 10

  tier3:
    name: Tier 3 (Complex)
    runs-on: ubuntu-latest
    if: github.event_name == 'schedule'
    steps:
      - name: Run Tier 3 tests
        run: make test-tier3
        timeout-minutes: 30
```

### Test Execution Flow

```
┌─────────────────────────────────────────────────────────┐
│                    Pull Request                          │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│  Tier 1: Simple Tests (<30s)                            │
│  ✓ Hello World                                          │
│  ✓ Data Validation                                      │
│  ✓ Error Handling                                       │
└─────────────────────────────────────────────────────────┘
                          ↓ (if pass)
┌─────────────────────────────────────────────────────────┐
│  Tier 2: Intermediate Tests (1-5min)                    │
│  ✓ Data Analysis                                        │
│  ✓ Feature Engineering                                  │
└─────────────────────────────────────────────────────────┘
                          ↓ (if pass)
┌─────────────────────────────────────────────────────────┐
│  Merge to Main                                          │
└─────────────────────────────────────────────────────────┘
                          ↓ (nightly)
┌─────────────────────────────────────────────────────────┐
│  Tier 3: Complex Tests (5-15min)                        │
│  ✓ Model Training                                       │
│  ✓ Hyperparameter Tuning                                │
└─────────────────────────────────────────────────────────┘
```

## Resource Requirements

### Kubernetes Resource Quotas

```yaml
# Tier 1: Simple notebooks
resources:
  requests:
    memory: "100Mi"
    cpu: "100m"
  limits:
    memory: "500Mi"
    cpu: "500m"

# Tier 2: Intermediate notebooks
resources:
  requests:
    memory: "500Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"

# Tier 3: Complex notebooks
resources:
  requests:
    memory: "1Gi"
    cpu: "1000m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

## Best Practices

### 1. Use Random Seeds

```python
import numpy as np
import random

# Set seeds for reproducibility
np.random.seed(42)
random.seed(42)
```

### 2. Add Validation Assertions

```python
# Validate intermediate results
assert df.shape[0] > 0, "DataFrame is empty"
assert model_score > 0.8, f"Model score too low: {model_score}"
```

### 3. Print Progress

```python
print("Loading data...")
# ... data loading code ...
print("✓ Data loaded successfully")

print("Training model...")
# ... training code ...
print("✓ Model training complete")
```

### 4. Handle Errors Gracefully

```python
try:
    # Risky operation
    result = perform_operation()
except Exception as e:
    print(f"⚠ Warning: {e}")
    # Fallback or recovery
```

### 5. Clean Up Resources

```python
import tempfile
import os

# Use temporary files
with tempfile.NamedTemporaryFile(delete=True) as tmp:
    # Work with tmp.name
    pass  # Automatically cleaned up
```

## Troubleshooting

### Test Failures

**Symptom**: Tier 1 test fails
```bash
# Check operator logs
kubectl logs -n jupyter-validator-system deployment/jupyter-notebook-validator-operator

# Check validation job status
kubectl describe notebookvalidationjob test-hello-world

# Check validation pod logs
kubectl logs -l job-name=test-hello-world
```

**Symptom**: Golden notebook mismatch
```bash
# Compare outputs
make compare-outputs JOB=test-hello-world

# Update golden notebook if change is intentional
make update-golden NOTEBOOK=01-hello-world
```

**Symptom**: Timeout
```bash
# Increase timeout for specific test
kubectl patch notebookvalidationjob test-model-training \
  --type merge \
  -p '{"spec":{"timeoutSeconds":1800}}'  # 30 minutes
```

## Coverage Standards

### Baseline (measured April 2026, commit b329281)

| Package | Tracked Lines | Coverage |
|---|---|---|
| `api/v1alpha1` | 1095 | 5.21% |
| `internal/controller` | 4388 | 45.58% |
| `pkg` | 2924 | 47.81% |
| **Overall** | **8407** | **41.09%** |

### Thresholds

| Gate | Current Target | Milestone |
|---|---|---|
| Project overall | **41%** | v1.0.10 → 50%, v1.1.0 → 60%, stretch → 75% |
| Patch (new code) | **−5% max** | A PR must not drop coverage by more than 5 points |

These thresholds are enforced automatically on every PR by Codecov. The project gate is set at the current baseline so PRs are not blocked unfairly — raise `.codecov.yml` `target:` as coverage improves milestone by milestone. The `api/v1alpha1` package is the biggest gap (5.21%); adding generated-type validation tests there will have the largest impact on the overall number.

### Running Coverage Locally

```bash
# Full coverage via make (sets up envtest automatically)
make test
go tool cover -func=cover.out | tail -1   # total line

# HTML report in browser
go tool cover -html=cover.out
```

### Roadmap to 75%

1. Fix `internal/controller` BeforeSuite flakiness so all 14 controller specs run reliably in CI
2. Add table-driven unit tests for the reconciler happy-path and error branches
3. Bring `pkg/build` from ~35% to ~60% with additional build-strategy unit tests
4. Each new feature PR is expected to include tests that keep patch coverage ≥ 0%

## Next Steps

1. **Review ADR 008**: [Notebook Testing Strategy and Complexity Levels](adrs/008-notebook-testing-strategy-and-complexity-levels.md)
2. **Explore Test Notebooks**: Browse `tests/notebooks/` directory
3. **Run Tests Locally**: `make test-all`
4. **Contribute Test Cases**: Add new test notebooks for edge cases

## References

- [ADR 008: Notebook Testing Strategy](adrs/008-notebook-testing-strategy-and-complexity-levels.md)
- [Papermill Documentation](https://papermill.readthedocs.io/)
- [Scikit-learn Testing Guide](https://scikit-learn.org/stable/developers/develop.html)
- [Netflix Notebook Testing](https://netflixtechblog.com/notebook-innovation-591ee3221233)

---

**Last Updated**: 2025-11-07
**Version**: 1.0

