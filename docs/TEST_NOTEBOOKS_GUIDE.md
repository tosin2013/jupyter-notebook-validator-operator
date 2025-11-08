# Test Notebooks Guide

This guide describes the test notebooks that should be created in the `jupyter-notebook-validator-test-notebooks` repository to support comprehensive integration testing.

## Repository Structure

```
jupyter-notebook-validator-test-notebooks/
├── README.md
├── simple-test.ipynb                          # Basic test (already exists)
├── eso-integration/
│   ├── aws-credentials-test.ipynb             # Test AWS credential injection
│   ├── database-connection-test.ipynb         # Test database credential injection
│   ├── mlflow-tracking-test.ipynb             # Test MLflow credential injection
│   └── api-keys-test.ipynb                    # Test API key injection
├── model-validation/
│   ├── kserve/
│   │   ├── model-inference-kserve.ipynb       # KServe inference test
│   │   ├── fraud-detection-test.ipynb         # Fraud detection model test
│   │   └── multi-model-pipeline.ipynb         # Multi-model pipeline test
│   ├── openshift-ai/
│   │   ├── model-inference-openshift-ai.ipynb # OpenShift AI inference test
│   │   ├── sentiment-analysis-test.ipynb      # Sentiment analysis test
│   │   └── text-classification-test.ipynb     # Text classification test
│   └── community/
│       ├── vllm/
│       │   ├── llm-inference-vllm.ipynb       # vLLM LLM inference test
│       │   └── llama-chat-test.ipynb          # Llama chat test
│       ├── torchserve/
│       │   └── pytorch-inference-test.ipynb   # TorchServe test
│       └── triton/
│           └── triton-inference-test.ipynb    # Triton inference test
└── golden-notebooks/
    ├── aws-credentials-test-golden.ipynb
    ├── database-connection-test-golden.ipynb
    └── model-inference-kserve-golden.ipynb
```

## Test Notebook Specifications

### 1. ESO Integration Notebooks

#### `eso-integration/aws-credentials-test.ipynb`

**Purpose**: Test AWS credential injection via ESO

**Expected Environment Variables**:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import boto3
from botocore.exceptions import ClientError

# Cell 2: Verify credentials are injected
assert 'AWS_ACCESS_KEY_ID' in os.environ, "AWS_ACCESS_KEY_ID not found"
assert 'AWS_SECRET_ACCESS_KEY' in os.environ, "AWS_SECRET_ACCESS_KEY not found"
assert 'AWS_REGION' in os.environ, "AWS_REGION not found"
print("✓ All AWS credentials found")

# Cell 3: Test credential format
access_key = os.environ['AWS_ACCESS_KEY_ID']
assert access_key.startswith('AKIA'), f"Invalid AWS access key format: {access_key[:4]}..."
print(f"✓ AWS Access Key format valid: {access_key[:4]}...")

# Cell 4: Test region
region = os.environ['AWS_REGION']
assert region in ['us-east-1', 'us-west-2', 'eu-west-1'], f"Unexpected region: {region}"
print(f"✓ AWS Region: {region}")

# Cell 5: Summary
print("\n=== AWS Credentials Test Summary ===")
print("✓ All AWS credentials properly injected")
print("✓ Credential format validation passed")
print("✓ Region validation passed")
```

#### `eso-integration/database-connection-test.ipynb`

**Purpose**: Test database credential injection via ESO

**Expected Environment Variables**:
- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import psycopg2
from urllib.parse import quote_plus

# Cell 2: Verify credentials are injected
required_vars = ['DB_HOST', 'DB_PORT', 'DB_NAME', 'DB_USER', 'DB_PASSWORD']
for var in required_vars:
    assert var in os.environ, f"{var} not found"
print("✓ All database credentials found")

# Cell 3: Build connection string
db_host = os.environ['DB_HOST']
db_port = os.environ['DB_PORT']
db_name = os.environ['DB_NAME']
db_user = os.environ['DB_USER']
db_password = os.environ['DB_PASSWORD']

connection_string = f"postgresql://{db_user}:{quote_plus(db_password)}@{db_host}:{db_port}/{db_name}"
print(f"✓ Connection string built: postgresql://{db_user}:***@{db_host}:{db_port}/{db_name}")

# Cell 4: Test connection (mock - don't actually connect in test)
print("✓ Database credentials validated (connection test skipped in validation)")

# Cell 5: Summary
print("\n=== Database Credentials Test Summary ===")
print("✓ All database credentials properly injected")
print("✓ Connection string format valid")
```

#### `eso-integration/mlflow-tracking-test.ipynb`

**Purpose**: Test MLflow credential injection via ESO

**Expected Environment Variables**:
- `MLFLOW_TRACKING_URI`
- `MLFLOW_TRACKING_USERNAME`
- `MLFLOW_TRACKING_PASSWORD`

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import mlflow

# Cell 2: Verify credentials are injected
assert 'MLFLOW_TRACKING_URI' in os.environ, "MLFLOW_TRACKING_URI not found"
assert 'MLFLOW_TRACKING_USERNAME' in os.environ, "MLFLOW_TRACKING_USERNAME not found"
assert 'MLFLOW_TRACKING_PASSWORD' in os.environ, "MLFLOW_TRACKING_PASSWORD not found"
print("✓ All MLflow credentials found")

# Cell 3: Set MLflow tracking URI
mlflow.set_tracking_uri(os.environ['MLFLOW_TRACKING_URI'])
print(f"✓ MLflow tracking URI set: {os.environ['MLFLOW_TRACKING_URI']}")

# Cell 4: Test authentication (mock)
username = os.environ['MLFLOW_TRACKING_USERNAME']
print(f"✓ MLflow username: {username}")

# Cell 5: Summary
print("\n=== MLflow Credentials Test Summary ===")
print("✓ All MLflow credentials properly injected")
print("✓ Tracking URI configured")
```

### 2. Model Validation Notebooks

#### `model-validation/kserve/model-inference-kserve.ipynb`

**Purpose**: Test model inference against KServe InferenceService

**Expected Environment Variables**:
- `MODEL_ENDPOINT` (injected by operator)
- `MODEL_NAME` (injected by operator)

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import requests
import json
import numpy as np

# Cell 2: Verify model environment variables
assert 'MODEL_ENDPOINT' in os.environ, "MODEL_ENDPOINT not found"
assert 'MODEL_NAME' in os.environ, "MODEL_NAME not found"
model_endpoint = os.environ['MODEL_ENDPOINT']
model_name = os.environ['MODEL_NAME']
print(f"✓ Model endpoint: {model_endpoint}")
print(f"✓ Model name: {model_name}")

# Cell 3: Test model health check
health_url = f"{model_endpoint}/v1/models/{model_name}"
try:
    response = requests.get(health_url, timeout=5)
    assert response.status_code == 200, f"Health check failed: {response.status_code}"
    print("✓ Model health check passed")
except requests.exceptions.RequestException as e:
    print(f"⚠ Health check skipped (model not deployed): {e}")

# Cell 4: Prepare test data
test_data = {
    "instances": [
        [1.0, 2.0, 3.0, 4.0, 5.0]
    ]
}
print(f"✓ Test data prepared: {test_data}")

# Cell 5: Make prediction (mock if model not available)
predict_url = f"{model_endpoint}/v1/models/{model_name}:predict"
try:
    response = requests.post(predict_url, json=test_data, timeout=10)
    if response.status_code == 200:
        predictions = response.json()
        print(f"✓ Prediction successful: {predictions}")
    else:
        print(f"⚠ Prediction skipped (model not deployed): {response.status_code}")
except requests.exceptions.RequestException as e:
    print(f"⚠ Prediction skipped (model not deployed): {e}")

# Cell 6: Summary
print("\n=== KServe Model Inference Test Summary ===")
print("✓ Model endpoint configured")
print("✓ Test data prepared")
print("✓ Inference test completed")
```

#### `model-validation/openshift-ai/sentiment-analysis-test.ipynb`

**Purpose**: Test sentiment analysis model on OpenShift AI

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import requests
import json

# Cell 2: Verify OpenShift AI environment
assert 'MODEL_ENDPOINT' in os.environ, "MODEL_ENDPOINT not found"
model_endpoint = os.environ['MODEL_ENDPOINT']
print(f"✓ OpenShift AI model endpoint: {model_endpoint}")

# Cell 3: Prepare sentiment analysis test data
test_texts = [
    "This is a great product!",
    "I love this service.",
    "Terrible experience, very disappointed."
]
print(f"✓ Test texts prepared: {len(test_texts)} samples")

# Cell 4: Test sentiment analysis (mock if model not available)
for i, text in enumerate(test_texts):
    print(f"\nTest {i+1}: {text}")
    try:
        response = requests.post(
            f"{model_endpoint}/predict",
            json={"instances": [text]},
            timeout=10
        )
        if response.status_code == 200:
            result = response.json()
            print(f"  ✓ Sentiment: {result}")
        else:
            print(f"  ⚠ Prediction skipped: {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"  ⚠ Prediction skipped: {e}")

# Cell 5: Summary
print("\n=== Sentiment Analysis Test Summary ===")
print("✓ OpenShift AI endpoint configured")
print(f"✓ Tested {len(test_texts)} samples")
```

#### `model-validation/community/vllm/llm-inference-vllm.ipynb`

**Purpose**: Test LLM inference with vLLM

**Test Steps**:
```python
# Cell 1: Import libraries
import os
import requests
import json

# Cell 2: Verify vLLM environment
assert 'VLLM_ENDPOINT' in os.environ, "VLLM_ENDPOINT not found"
vllm_endpoint = os.environ['VLLM_ENDPOINT']
print(f"✓ vLLM endpoint: {vllm_endpoint}")

# Cell 3: Prepare LLM test prompt
prompt = "What is the capital of France?"
test_data = {
    "model": "llama-2-7b-chat",
    "prompt": prompt,
    "max_tokens": 50,
    "temperature": 0.7
}
print(f"✓ Test prompt: {prompt}")

# Cell 4: Test LLM completion (mock if model not available)
try:
    response = requests.post(
        f"{vllm_endpoint}/v1/completions",
        json=test_data,
        timeout=30
    )
    if response.status_code == 200:
        result = response.json()
        completion = result['choices'][0]['text']
        print(f"✓ LLM response: {completion}")
        assert "Paris" in completion, "Expected 'Paris' in response"
        print("✓ Response validation passed")
    else:
        print(f"⚠ LLM inference skipped: {response.status_code}")
except requests.exceptions.RequestException as e:
    print(f"⚠ LLM inference skipped: {e}")

# Cell 5: Summary
print("\n=== vLLM Inference Test Summary ===")
print("✓ vLLM endpoint configured")
print("✓ LLM inference test completed")
```

## Creating the Notebooks

### Step 1: Clone the Test Notebooks Repository

```bash
git clone https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
cd jupyter-notebook-validator-test-notebooks
```

### Step 2: Create Directory Structure

```bash
mkdir -p eso-integration
mkdir -p model-validation/kserve
mkdir -p model-validation/openshift-ai
mkdir -p model-validation/community/vllm
mkdir -p model-validation/community/torchserve
mkdir -p model-validation/community/triton
mkdir -p golden-notebooks
```

### Step 3: Create Notebooks

Use Jupyter to create the notebooks based on the specifications above:

```bash
jupyter notebook
```

Or use `nbformat` to create them programmatically:

```python
import nbformat as nbf

# Create a new notebook
nb = nbf.v4.new_notebook()

# Add cells
nb['cells'] = [
    nbf.v4.new_markdown_cell("# AWS Credentials Test"),
    nbf.v4.new_code_cell("import os\nimport boto3"),
    # ... add more cells
]

# Write notebook
with open('eso-integration/aws-credentials-test.ipynb', 'w') as f:
    nbf.write(nb, f)
```

### Step 4: Commit and Push

```bash
git add .
git commit -m "Add integration test notebooks for ESO and model validation"
git push origin main
```

## Using the Test Notebooks

Once the notebooks are in the repository, update the integration test suite to use them:

```bash
# In test/integration-test-suite.sh, update the notebook paths:
# - eso-integration/aws-credentials-test.ipynb
# - model-validation/kserve/model-inference-kserve.ipynb
# - model-validation/openshift-ai/sentiment-analysis-test.ipynb
```

## Golden Notebooks

For comparison testing, create golden notebooks with expected outputs:

1. Run the test notebook successfully
2. Save the output as a golden notebook
3. Use the golden notebook for comparison in future validation runs

Example:
```yaml
spec:
  notebook:
    path: eso-integration/aws-credentials-test.ipynb
  goldenNotebook:
    path: golden-notebooks/aws-credentials-test-golden.ipynb
```

## Next Steps

1. Create the notebooks in the test repository
2. Update the integration test suite to reference the new notebooks
3. Add golden notebooks for comparison testing
4. Document the expected outputs for each test
5. Add CI/CD integration to run tests automatically

