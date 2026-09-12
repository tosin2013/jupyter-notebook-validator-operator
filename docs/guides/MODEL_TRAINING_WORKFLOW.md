# Model Training Workflow with Jupyter Notebook Validator Operator

**Date:** 2025-11-08  
**Status:** ✅ Complete  
**Purpose:** Document how to use the operator to validate notebooks that train ML models

---

## Overview

The Jupyter Notebook Validator Operator can validate notebooks that **train machine learning models** from scratch. This enables:

1. **Reproducible Training** - Ensure training notebooks execute successfully
2. **CI/CD Integration** - Validate training pipelines in automated workflows
3. **Model Versioning** - Track model training as part of version control
4. **Quality Assurance** - Verify model training produces expected artifacts

## Training Notebook Structure

A typical model training notebook includes:

### 1. Data Preparation
```python
# Load and prepare training data
import pandas as pd
from sklearn.model_selection import train_test_split

# Create or load dataset
df = pd.DataFrame(training_data, columns=['text', 'sentiment'])
X_train, X_test, y_train, y_test = train_test_split(df['text'], df['sentiment'])
```

### 2. Model Training
```python
# Train model
from sklearn.linear_model import LogisticRegression

model = LogisticRegression()
model.fit(X_train_vec, y_train)
```

### 3. Model Evaluation
```python
# Evaluate model
from sklearn.metrics import accuracy_score

y_pred = model.predict(X_test_vec)
accuracy = accuracy_score(y_test, y_pred)
print(f"Model Accuracy: {accuracy:.2%}")
```

### 4. Model Persistence
```python
# Save model
import joblib

joblib.dump(model, '/tmp/sentiment-model/model.pkl')
joblib.dump(vectorizer, '/tmp/sentiment-model/vectorizer.pkl')
```

## Example: Train Sentiment Analysis Model

### Notebook: `train-sentiment-model.ipynb`

**Location:** `jupyter-notebook-validator-test-notebooks/model-training/train-sentiment-model.ipynb`

**What it does:**
1. Creates a sample sentiment analysis dataset (20 samples)
2. Trains a Logistic Regression model with TF-IDF features
3. Evaluates model accuracy on test set
4. Saves model, vectorizer, and metadata to disk
5. Tests the saved model can be loaded and used

**Expected outputs:**
- `/tmp/sentiment-model/model.pkl` - Trained model
- `/tmp/sentiment-model/vectorizer.pkl` - TF-IDF vectorizer
- `/tmp/sentiment-model/metadata.json` - Model metadata

**Expected accuracy:** ~80-100% (small dataset)

## NotebookValidationJob Configuration

### Basic Training Job

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: train-sentiment-model
  namespace: mlops
spec:
  notebook:
    git:
      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
      ref: main
    path: model-training/train-sentiment-model.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
    resources:
      requests:
        memory: "1Gi"
        cpu: "500m"
      limits:
        memory: "2Gi"
        cpu: "1000m"
  
  validation:
    timeout: 600  # 10 minutes for training
    
    expectedOutputs:
      - type: file
        path: /tmp/sentiment-model/model.pkl
      - type: file
        path: /tmp/sentiment-model/vectorizer.pkl
      - type: log
        pattern: "Model trained successfully"
      - type: log
        pattern: "Model Accuracy:"
```

### Advanced: Training with External Data

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: train-with-s3-data
  namespace: mlops
spec:
  notebook:
    git:
      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
      ref: main
    path: model-training/train-with-external-data.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
    
    # Inject AWS credentials via ESO
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: secret-access-key
      - name: TRAINING_DATA_S3_URI
        value: "s3://my-bucket/training-data/sentiment.csv"
      - name: MODEL_OUTPUT_S3_URI
        value: "s3://my-bucket/models/sentiment-v1/"
    
    resources:
      requests:
        memory: "2Gi"
        cpu: "1000m"
      limits:
        memory: "4Gi"
        cpu: "2000m"
  
  validation:
    timeout: 1800  # 30 minutes for larger training
```

## Deployment Workflow

### Step 1: Train Model

```bash
# Create training job
oc apply -f config/samples/model-training-job.yaml

# Watch training progress
oc get notebookvalidationjob train-sentiment-model -n mlops -w

# Check training logs
oc logs -n mlops -l job-name=train-sentiment-model-validation -f
```

### Step 2: Extract Trained Model

```bash
# Get the validation pod name
POD=$(oc get pods -n mlops -l job-name=train-sentiment-model-validation -o jsonpath='{.items[0].metadata.name}')

# Copy model files from pod
oc cp mlops/$POD:/tmp/sentiment-model ./trained-model/

# Verify model files
ls -la ./trained-model/
# model.pkl
# vectorizer.pkl
# metadata.json
```

### Step 3: Upload to Storage

```bash
# Upload to S3
aws s3 cp ./trained-model/ s3://my-bucket/models/sentiment-v1/ --recursive

# Or upload to PVC
oc cp ./trained-model/ mlops/model-storage-pvc:/models/sentiment-v1/
```

### Step 4: Deploy Model

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: trained-sentiment-model
  namespace: mlops
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
        version: "1"
      runtime: mlserver-sklearn
      storageUri: "s3://my-bucket/models/sentiment-v1/"
```

### Step 5: Validate Deployed Model

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-trained-model
  namespace: mlops
spec:
  notebook:
    git:
      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
      ref: main
    path: model-validation/openshift-ai/sentiment-analysis-test.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
  
  modelValidation:
    enabled: true
    platform: openshift-ai
    targetModels:
      - trained-sentiment-model
    predictionValidation:
      enabled: true
      testData: |
        {"instances": [["This is amazing!"]]}
      expectedOutput: |
        {"predictions": [[1]]}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Train and Deploy Model

on:
  push:
    paths:
      - 'notebooks/training/**'

jobs:
  train-model:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Login to OpenShift
        run: |
          oc login --token=${{ secrets.OC_TOKEN }} --server=${{ secrets.OC_SERVER }}
      
      - name: Create training job
        run: |
          oc apply -f config/samples/model-training-job.yaml
      
      - name: Wait for training completion
        run: |
          oc wait --for=condition=Complete notebookvalidationjob/train-sentiment-model \
            -n mlops --timeout=10m
      
      - name: Extract model
        run: |
          POD=$(oc get pods -n mlops -l job-name=train-sentiment-model-validation \
            -o jsonpath='{.items[0].metadata.name}')
          oc cp mlops/$POD:/tmp/sentiment-model ./trained-model/
      
      - name: Upload to S3
        run: |
          aws s3 cp ./trained-model/ s3://my-bucket/models/sentiment-${{ github.sha }}/ \
            --recursive
      
      - name: Deploy model
        run: |
          envsubst < config/samples/inference-service.yaml | oc apply -f -
```

## Best Practices

### 1. Resource Allocation

- **Small datasets (<1000 samples):** 1Gi memory, 500m CPU
- **Medium datasets (1000-100k samples):** 2-4Gi memory, 1-2 CPU
- **Large datasets (>100k samples):** 4-8Gi memory, 2-4 CPU

### 2. Timeout Configuration

- **Simple models (linear, logistic):** 5-10 minutes
- **Tree-based models (RF, XGBoost):** 10-30 minutes
- **Deep learning models:** 30-120 minutes

### 3. Model Validation

Always validate:
- ✅ Model file exists and is loadable
- ✅ Model accuracy meets minimum threshold
- ✅ Model can make predictions on test data
- ✅ Model metadata is saved

### 4. Error Handling

```python
# In training notebook
try:
    model.fit(X_train, y_train)
    print("✓ Model trained successfully")
except Exception as e:
    print(f"✗ Training failed: {e}")
    raise

# Validate accuracy threshold
if accuracy < 0.7:
    raise ValueError(f"Model accuracy {accuracy:.2%} below threshold 70%")
```

## Troubleshooting

### Training Job Fails

```bash
# Check job status
oc describe notebookvalidationjob train-sentiment-model -n mlops

# Check pod logs
oc logs -n mlops -l job-name=train-sentiment-model-validation

# Check events
oc get events -n mlops --sort-by='.lastTimestamp'
```

### Out of Memory

```yaml
# Increase memory limits
podConfig:
  resources:
    limits:
      memory: "4Gi"  # Increase from 2Gi
```

### Training Timeout

```yaml
# Increase timeout
validation:
  timeout: 1800  # 30 minutes instead of 10
```

## Next Steps

1. ✅ Create training notebook (`train-sentiment-model.ipynb`)
2. ✅ Create NotebookValidationJob for training
3. 📝 Test training job on OpenShift
4. 📝 Extract and deploy trained model
5. 📝 Validate deployed model with inference notebook

---

**See Also:**
- [Model Discovery Guide](MODEL_DISCOVERY_GUIDE.md)
- [Real Model Deployment](REAL_MODEL_DEPLOYMENT_COMPLETE.md)
- [Test Notebooks Guide](TEST_NOTEBOOKS_GUIDE.md)

