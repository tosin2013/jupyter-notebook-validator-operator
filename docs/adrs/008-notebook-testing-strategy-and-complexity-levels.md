# ADR 008: Notebook Testing Strategy and Complexity Levels

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must be tested against real-world notebook scenarios that represent the diverse use cases of MLOps workflows. Testing only with trivial notebooks would fail to uncover issues related to:

- **Long-running computations**: Model training, data processing
- **External dependencies**: Database connections, API calls, file I/O
- **Resource constraints**: Memory usage, CPU utilization
- **Error handling**: Failed cells, exceptions, timeouts
- **Reproducibility**: Deterministic outputs, random seeds
- **Data validation**: Input/output verification, schema validation

### User Research Findings

Based on research into MLOps and notebook validation patterns, we identified common user needs:

1. **Data Scientists**: Need to validate exploratory notebooks before committing to version control
2. **ML Engineers**: Need to ensure training notebooks produce consistent models
3. **Platform Teams**: Need to validate notebooks work across different environments
4. **CI/CD Pipelines**: Need automated notebook testing as part of deployment workflows
5. **Compliance Teams**: Need to verify notebooks meet regulatory requirements

### Industry Patterns

- **Papermill**: Industry-standard tool for parameterized notebook execution
- **nbval**: pytest plugin for notebook validation
- **Netflix**: Uses notebooks extensively in production with multi-tier testing
- **Airbnb**: Validates notebooks in CI/CD pipelines before deployment
- **Google Cloud**: Recommends notebook testing as part of ML best practices

### Testing Challenges

1. **Complexity Spectrum**: Notebooks range from simple data exploration to complex model training
2. **Execution Time**: Some notebooks take seconds, others take hours
3. **Resource Requirements**: Memory and CPU needs vary dramatically
4. **External Dependencies**: Databases, APIs, cloud services, file systems
5. **Non-Determinism**: Random seeds, timestamps, network latency
6. **Golden Notebook Comparison**: Determining acceptable output differences

## Decision

We will implement a **Three-Tier Notebook Testing Strategy** with representative test notebooks at each complexity level. This strategy will be used for:

1. **Operator Development**: Validate operator functionality during development
2. **CI/CD Pipeline**: Automated testing on every PR and release
3. **User Documentation**: Provide examples of supported notebook patterns
4. **Performance Benchmarking**: Measure operator performance across complexity levels

### Tier 1: Simple Notebooks (Seconds to Execute)

**Purpose**: Validate basic operator functionality, fast feedback in CI/CD

**Characteristics**:
- Execution time: <30 seconds
- No external dependencies
- Deterministic outputs
- Minimal resource usage (<100Mi memory, <100m CPU)
- Pure Python computations

**Test Scenarios**:

#### 1.1 Hello World Notebook
```python
# Cell 1: Basic computation
result = 2 + 2
print(f"Result: {result}")
assert result == 4

# Cell 2: String manipulation
message = "Hello from Jupyter Notebook Validator"
print(message)
assert len(message) > 0

# Cell 3: List operations
numbers = [1, 2, 3, 4, 5]
squared = [n**2 for n in numbers]
print(f"Squared: {squared}")
assert squared == [1, 4, 9, 16, 25]
```

**Expected Outcome**: All cells execute successfully, outputs match golden notebook

#### 1.2 Data Validation Notebook
```python
# Cell 1: Import libraries
import pandas as pd
import numpy as np

# Cell 2: Create sample data
data = {
    'name': ['Alice', 'Bob', 'Charlie'],
    'age': [25, 30, 35],
    'score': [85.5, 92.0, 78.5]
}
df = pd.DataFrame(data)
print(df)

# Cell 3: Validate data
assert len(df) == 3
assert df['age'].mean() == 30
assert df['score'].min() >= 0
assert df['score'].max() <= 100
print("✓ Data validation passed")

# Cell 4: Simple transformation
df['age_group'] = df['age'].apply(lambda x: 'young' if x < 30 else 'adult')
print(df)
assert df['age_group'].tolist() == ['young', 'adult', 'adult']
```

**Expected Outcome**: Data validation passes, transformations correct

#### 1.3 Error Handling Notebook
```python
# Cell 1: Successful cell
print("This cell succeeds")
result = 10 / 2
assert result == 5

# Cell 2: Intentional error (for testing error handling)
# This cell should fail gracefully
try:
    result = 10 / 0
    print("This should not print")
except ZeroDivisionError as e:
    print(f"✓ Caught expected error: {e}")

# Cell 3: Recovery after error
print("Execution continues after handled error")
assert True
```

**Expected Outcome**: Operator handles errors gracefully, continues execution

### Tier 2: Intermediate Notebooks (Minutes to Execute)

**Purpose**: Validate operator handles realistic data science workflows

**Characteristics**:
- Execution time: 1-5 minutes
- External dependencies: CSV files, small datasets
- Some non-determinism (controlled with random seeds)
- Moderate resource usage (<500Mi memory, <500m CPU)
- Data processing and visualization

**Test Scenarios**:

#### 2.1 Data Analysis Notebook
```python
# Cell 1: Import libraries
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from sklearn.datasets import load_iris

# Cell 2: Load dataset
iris = load_iris()
df = pd.DataFrame(iris.data, columns=iris.feature_names)
df['species'] = iris.target
print(f"Dataset shape: {df.shape}")
print(df.head())

# Cell 3: Statistical analysis
summary = df.describe()
print(summary)
assert df.shape[0] == 150  # Iris dataset has 150 samples
assert df.shape[1] == 5    # 4 features + 1 target

# Cell 4: Data visualization
fig, axes = plt.subplots(2, 2, figsize=(12, 10))
for idx, col in enumerate(iris.feature_names):
    ax = axes[idx // 2, idx % 2]
    ax.hist(df[col], bins=20, edgecolor='black')
    ax.set_title(col)
    ax.set_xlabel('Value')
    ax.set_ylabel('Frequency')
plt.tight_layout()
plt.savefig('/tmp/iris_histograms.png')
print("✓ Visualizations saved")

# Cell 5: Correlation analysis
correlation = df.iloc[:, :-1].corr()
print("Correlation matrix:")
print(correlation)
assert correlation.shape == (4, 4)
```

**Expected Outcome**: Analysis completes, visualizations generated, assertions pass

#### 2.2 Feature Engineering Notebook
```python
# Cell 1: Setup
import pandas as pd
import numpy as np
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.model_selection import train_test_split

np.random.seed(42)  # Ensure reproducibility

# Cell 2: Generate synthetic data
n_samples = 1000
data = {
    'feature1': np.random.randn(n_samples),
    'feature2': np.random.randn(n_samples) * 2 + 5,
    'feature3': np.random.choice(['A', 'B', 'C'], n_samples),
    'target': np.random.choice([0, 1], n_samples)
}
df = pd.DataFrame(data)
print(f"Generated {len(df)} samples")

# Cell 3: Encode categorical features
le = LabelEncoder()
df['feature3_encoded'] = le.fit_transform(df['feature3'])
print("Categorical encoding complete")
print(df.head())

# Cell 4: Scale numerical features
scaler = StandardScaler()
df[['feature1_scaled', 'feature2_scaled']] = scaler.fit_transform(
    df[['feature1', 'feature2']]
)
print("Feature scaling complete")
print(df.describe())

# Cell 5: Train/test split
X = df[['feature1_scaled', 'feature2_scaled', 'feature3_encoded']]
y = df['target']
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42
)
print(f"Train set: {len(X_train)} samples")
print(f"Test set: {len(X_test)} samples")
assert len(X_train) == 800
assert len(X_test) == 200
```

**Expected Outcome**: Feature engineering pipeline completes, data split correctly

### Tier 3: Complex Notebooks (Model Training, 5-15 Minutes)

**Purpose**: Validate operator handles production-like ML workflows

**Characteristics**:
- Execution time: 5-15 minutes
- External dependencies: Datasets, model artifacts
- Controlled non-determinism (random seeds, cross-validation)
- High resource usage (<2Gi memory, <1000m CPU)
- Model training, evaluation, and persistence

**Test Scenarios**:

#### 3.1 Model Training Notebook
```python
# Cell 1: Import libraries
import pandas as pd
import numpy as np
from sklearn.datasets import load_breast_cancer
from sklearn.model_selection import train_test_split, cross_val_score
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import classification_report, confusion_matrix, roc_auc_score
import joblib
import json
from datetime import datetime

np.random.seed(42)

# Cell 2: Load and prepare data
print("Loading breast cancer dataset...")
data = load_breast_cancer()
X = pd.DataFrame(data.data, columns=data.feature_names)
y = pd.Series(data.target)

print(f"Dataset shape: {X.shape}")
print(f"Class distribution: {y.value_counts().to_dict()}")

# Cell 3: Train/test split
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)
print(f"Training samples: {len(X_train)}")
print(f"Test samples: {len(X_test)}")

# Cell 4: Model training
print("Training Random Forest model...")
model = RandomForestClassifier(
    n_estimators=100,
    max_depth=10,
    random_state=42,
    n_jobs=-1
)
model.fit(X_train, y_train)
print("✓ Model training complete")

# Cell 5: Cross-validation
print("Performing cross-validation...")
cv_scores = cross_val_score(model, X_train, y_train, cv=5, scoring='roc_auc')
print(f"CV ROC-AUC scores: {cv_scores}")
print(f"Mean CV ROC-AUC: {cv_scores.mean():.4f} (+/- {cv_scores.std():.4f})")
assert cv_scores.mean() > 0.90  # Expect good performance

# Cell 6: Model evaluation
print("Evaluating model on test set...")
y_pred = model.predict(X_test)
y_pred_proba = model.predict_proba(X_test)[:, 1]

test_auc = roc_auc_score(y_test, y_pred_proba)
print(f"Test ROC-AUC: {test_auc:.4f}")
assert test_auc > 0.90

print("\nClassification Report:")
print(classification_report(y_test, y_pred))

print("\nConfusion Matrix:")
print(confusion_matrix(y_test, y_pred))

# Cell 7: Feature importance
feature_importance = pd.DataFrame({
    'feature': X.columns,
    'importance': model.feature_importances_
}).sort_values('importance', ascending=False)

print("\nTop 10 Most Important Features:")
print(feature_importance.head(10))

# Cell 8: Save model and metadata
model_path = '/tmp/breast_cancer_model.joblib'
joblib.dump(model, model_path)
print(f"✓ Model saved to {model_path}")

metadata = {
    'model_type': 'RandomForestClassifier',
    'training_date': datetime.now().isoformat(),
    'n_samples_train': len(X_train),
    'n_samples_test': len(X_test),
    'cv_roc_auc_mean': float(cv_scores.mean()),
    'cv_roc_auc_std': float(cv_scores.std()),
    'test_roc_auc': float(test_auc),
    'n_features': X.shape[1],
    'random_seed': 42
}

metadata_path = '/tmp/model_metadata.json'
with open(metadata_path, 'w') as f:
    json.dump(metadata, f, indent=2)
print(f"✓ Metadata saved to {metadata_path}")

# Cell 9: Validation checks
print("\n=== Final Validation ===")
assert test_auc > 0.90, "Model performance below threshold"
assert len(feature_importance) == X.shape[1], "Feature importance mismatch"
print("✓ All validation checks passed")
print(f"✓ Model ready for deployment (ROC-AUC: {test_auc:.4f})")
```

**Expected Outcome**: 
- Model trains successfully
- Performance metrics meet thresholds
- Model and metadata saved
- All assertions pass

#### 3.2 Hyperparameter Tuning Notebook
```python
# Cell 1: Setup
import pandas as pd
import numpy as np
from sklearn.datasets import load_digits
from sklearn.model_selection import GridSearchCV, train_test_split
from sklearn.svm import SVC
from sklearn.metrics import classification_report
import time

np.random.seed(42)

# Cell 2: Load data
print("Loading digits dataset...")
digits = load_digits()
X_train, X_test, y_train, y_test = train_test_split(
    digits.data, digits.target, test_size=0.2, random_state=42
)
print(f"Training samples: {len(X_train)}")
print(f"Test samples: {len(X_test)}")

# Cell 3: Define hyperparameter grid
param_grid = {
    'C': [0.1, 1, 10],
    'gamma': ['scale', 'auto', 0.001, 0.01],
    'kernel': ['rbf', 'linear']
}
print(f"Hyperparameter grid: {param_grid}")
print(f"Total combinations: {3 * 4 * 2} = 24")

# Cell 4: Grid search
print("\nStarting grid search...")
start_time = time.time()

grid_search = GridSearchCV(
    SVC(random_state=42),
    param_grid,
    cv=3,
    scoring='accuracy',
    n_jobs=-1,
    verbose=1
)
grid_search.fit(X_train, y_train)

elapsed_time = time.time() - start_time
print(f"✓ Grid search complete in {elapsed_time:.2f} seconds")

# Cell 5: Best parameters
print(f"\nBest parameters: {grid_search.best_params_}")
print(f"Best CV score: {grid_search.best_score_:.4f}")
assert grid_search.best_score_ > 0.95

# Cell 6: Test set evaluation
best_model = grid_search.best_estimator_
test_score = best_model.score(X_test, y_test)
print(f"\nTest set accuracy: {test_score:.4f}")
assert test_score > 0.95

y_pred = best_model.predict(X_test)
print("\nClassification Report:")
print(classification_report(y_test, y_pred))

print("✓ Hyperparameter tuning complete")
```

**Expected Outcome**: Grid search completes, best model found, performance validated

## Consequences

### Positive
- **Comprehensive Coverage**: Tests span full complexity spectrum
- **Real-World Validation**: Notebooks represent actual MLOps use cases
- **Performance Benchmarking**: Measure operator performance across tiers
- **User Documentation**: Test notebooks serve as usage examples
- **CI/CD Integration**: Fast Tier 1 tests, thorough Tier 2/3 tests
- **Regression Detection**: Golden notebook comparison catches unexpected changes

### Negative
- **Maintenance Burden**: Must maintain test notebooks as dependencies evolve
- **Execution Time**: Tier 3 tests slow down CI/CD pipeline
- **Resource Requirements**: Need sufficient cluster resources for parallel testing
- **Non-Determinism**: Some notebooks may have acceptable output variations
- **Golden Notebook Drift**: Must update golden notebooks when intentional changes occur

### Neutral
- **Tiered Testing**: Can run different tiers in different CI/CD stages
- **Parameterization**: Can use Papermill to parameterize test notebooks

## Implementation Notes

### Test Notebook Repository Structure

```
tests/notebooks/
├── tier1-simple/
│   ├── 01-hello-world.ipynb
│   ├── 02-data-validation.ipynb
│   ├── 03-error-handling.ipynb
│   └── golden/
│       ├── 01-hello-world-golden.ipynb
│       ├── 02-data-validation-golden.ipynb
│       └── 03-error-handling-golden.ipynb
├── tier2-intermediate/
│   ├── 01-data-analysis.ipynb
│   ├── 02-feature-engineering.ipynb
│   └── golden/
│       ├── 01-data-analysis-golden.ipynb
│       └── 02-feature-engineering-golden.ipynb
└── tier3-complex/
    ├── 01-model-training.ipynb
    ├── 02-hyperparameter-tuning.ipynb
    └── golden/
        ├── 01-model-training-golden.ipynb
        └── 02-hyperparameter-tuning-golden.ipynb
```

### CI/CD Integration

```yaml
# .github/workflows/notebook-tests.yml
name: Notebook Validation Tests

on: [push, pull_request]

jobs:
  tier1-fast:
    name: Tier 1 - Simple Notebooks (Fast)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Deploy operator
        run: make deploy IMG=${{ env.OPERATOR_IMG }}
      
      - name: Run Tier 1 tests
        run: |
          for notebook in tests/notebooks/tier1-simple/*.ipynb; do
            kubectl apply -f - <<EOF
            apiVersion: mlops.dev/v1alpha1
            kind: NotebookValidationJob
            metadata:
              name: test-$(basename $notebook .ipynb)
            spec:
              notebook:
                git:
                  url: ${{ github.repository }}
                  ref: ${{ github.sha }}
                  path: $notebook
              golden:
                path: tests/notebooks/tier1-simple/golden/$(basename $notebook)
            EOF
          done
      
      - name: Wait for completion
        run: make wait-for-jobs TIMEOUT=60s
      
      - name: Check results
        run: make check-job-results

  tier2-intermediate:
    name: Tier 2 - Intermediate Notebooks
    runs-on: ubuntu-latest
    needs: tier1-fast
    steps:
      # Similar to tier1, but with longer timeout
      - name: Run Tier 2 tests
        run: make test-tier2 TIMEOUT=5m

  tier3-complex:
    name: Tier 3 - Complex Notebooks (Nightly)
    runs-on: ubuntu-latest
    if: github.event_name == 'schedule' || contains(github.event.head_commit.message, '[test-tier3]')
    steps:
      # Run only on schedule or explicit trigger
      - name: Run Tier 3 tests
        run: make test-tier3 TIMEOUT=15m
```

### Resource Quotas

```yaml
# config/samples/resource-quotas.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: notebook-validation-quota
spec:
  hard:
    # Tier 1: Simple notebooks
    requests.cpu.tier1: "500m"
    requests.memory.tier1: "500Mi"
    limits.cpu.tier1: "1000m"
    limits.memory.tier1: "1Gi"
    
    # Tier 2: Intermediate notebooks
    requests.cpu.tier2: "1000m"
    requests.memory.tier2: "1Gi"
    limits.cpu.tier2: "2000m"
    limits.memory.tier2: "2Gi"
    
    # Tier 3: Complex notebooks
    requests.cpu.tier3: "2000m"
    requests.memory.tier3: "2Gi"
    limits.cpu.tier3: "4000m"
    limits.memory.tier3: "4Gi"
```

## References

- [Papermill Documentation](https://papermill.readthedocs.io/)
- [nbval - pytest plugin for notebooks](https://nbval.readthedocs.io/)
- [Netflix Notebook Innovation](https://netflixtechblog.com/notebook-innovation-591ee3221233)
- [Google Cloud ML Best Practices](https://cloud.google.com/architecture/mlops-continuous-delivery-and-automation-pipelines-in-machine-learning)
- [Scikit-learn Model Evaluation](https://scikit-learn.org/stable/modules/model_evaluation.html)

## Related ADRs

- ADR 003: CRD Schema Design (defines NotebookValidationJob spec)
- ADR 006: Version Support Roadmap (defines testing phases)
- ADR 009: CI/CD Pipeline Integration (implements automated testing)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial three-tier testing strategy |

