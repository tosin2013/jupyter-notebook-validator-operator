#!/bin/bash
# Update Test Notebooks Repository with Model Discovery
# This script copies the model discovery library and deployment manifests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_REPO="${TEST_REPO:-/home/lab-user/jupyter-notebook-validator-test-notebooks}"

echo "=========================================="
echo "Updating Test Notebooks Repository"
echo "=========================================="
echo "Project Root: ${PROJECT_ROOT}"
echo "Test Repo:    ${TEST_REPO}"
echo ""

# Check if test repo exists
if [ ! -d "${TEST_REPO}" ]; then
    echo "❌ Error: Test repository not found at ${TEST_REPO}"
    echo "   Please clone it first or set TEST_REPO environment variable"
    exit 1
fi

cd "${TEST_REPO}"

# Create directories
echo "Creating directory structure..."
mkdir -p deployments
mkdir -p lib
mkdir -p docs
echo "✓ Directories created"
echo ""

# Create model_discovery.py
echo "Creating model_discovery.py..."
cat > lib/model_discovery.py <<'EOFPY'
"""
Model Discovery Helper Library for Jupyter Notebook Validator Operator

This library helps notebooks discover and interact with deployed models
without hardcoding model names or endpoints.

Usage:
    import sys
    sys.path.append('/workspace/lib')
    from model_discovery import discover_models, get_model_endpoint
    
    models = discover_models(platform='kserve')
    endpoint = get_model_endpoint('my-model')
"""

import os
import requests
from typing import Dict, Optional, List

def get_platform() -> str:
    """Get the model serving platform from environment."""
    return os.environ.get('MODEL_VALIDATION_PLATFORM', 'unknown')

def get_target_models() -> List[str]:
    """Get target models from environment."""
    models_str = os.environ.get('MODEL_VALIDATION_TARGET_MODELS', '')
    return [m.strip() for m in models_str.split(',') if m.strip()]

def get_namespace() -> str:
    """Get the current namespace."""
    # Try to read from service account
    try:
        with open('/var/run/secrets/kubernetes.io/serviceaccount/namespace', 'r') as f:
            return f.read().strip()
    except:
        return os.environ.get('NAMESPACE', 'mlops')

def discover_kserve_models(namespace: Optional[str] = None) -> Dict:
    """
    Discover available KServe InferenceServices.
    
    Returns:
        dict: Available models with metadata
    """
    if namespace is None:
        namespace = get_namespace()
    
    try:
        from kubernetes import client, config
        config.load_incluster_config()
        api = client.CustomObjectsApi()
        
        inference_services = api.list_namespaced_custom_object(
            group="serving.kserve.io",
            version="v1beta1",
            namespace=namespace,
            plural="inferenceservices"
        )
        
        models = {}
        for svc in inference_services.get('items', []):
            name = svc['metadata']['name']
            status = svc.get('status', {})
            
            # Get URL from status
            url = status.get('url', '')
            if not url:
                url = f'http://{name}-predictor-default.{namespace}.svc.cluster.local'
            
            # Check readiness
            conditions = status.get('conditions', [])
            ready = any(c.get('type') == 'Ready' and c.get('status') == 'True' 
                       for c in conditions)
            
            models[name] = {
                'name': name,
                'url': url,
                'ready': ready,
                'platform': 'kserve',
                'namespace': namespace,
                'predictor': svc.get('spec', {}).get('predictor', {})
            }
        
        return models
    except Exception as e:
        print(f"⚠ Could not discover KServe models: {e}")
        print(f"   Make sure ServiceAccount has permissions to list InferenceServices")
        return {}

def discover_openshift_ai_models(namespace: Optional[str] = None) -> Dict:
    """
    Discover available OpenShift AI models.
    OpenShift AI uses KServe under the hood.
    """
    return discover_kserve_models(namespace)

def discover_models(platform: Optional[str] = None, namespace: Optional[str] = None) -> Dict:
    """
    Discover available models based on platform.
    
    Args:
        platform: Model serving platform (kserve, openshift-ai, vllm, etc.)
                 If None, uses MODEL_VALIDATION_PLATFORM env var
        namespace: Kubernetes namespace to search
    
    Returns:
        dict: Available models with their metadata
    """
    if platform is None:
        platform = get_platform()
    
    if namespace is None:
        namespace = get_namespace()
    
    print(f"Discovering models on platform: {platform} in namespace: {namespace}")
    
    if platform in ['kserve', 'openshift-ai']:
        return discover_kserve_models(namespace)
    else:
        print(f"⚠ Unknown platform: {platform}")
        return {}

def check_model_health(model_url: str, timeout: int = 5) -> bool:
    """Check if a model endpoint is healthy."""
    try:
        # Try common health endpoints
        for path in ['/health', '/v1/health', '/healthz', '']:
            try:
                url = f"{model_url.rstrip('/')}{path}"
                response = requests.get(url, timeout=timeout)
                if response.status_code == 200:
                    return True
            except:
                continue
        return False
    except Exception as e:
        print(f"⚠ Health check failed: {e}")
        return False

def get_model_endpoint(model_name: str, platform: Optional[str] = None, 
                       namespace: Optional[str] = None) -> Optional[str]:
    """
    Get the endpoint URL for a specific model.
    
    Returns:
        str: Model endpoint URL, or None if model not found/ready
    """
    models = discover_models(platform, namespace)
    model = models.get(model_name)
    
    if model and model.get('ready'):
        return model['url']
    
    print(f"⚠ Model '{model_name}' not found or not ready")
    return None

def make_prediction(model_url: str, data: dict, timeout: int = 30) -> Optional[dict]:
    """
    Make a prediction request to a model endpoint.
    
    Args:
        model_url: Model endpoint URL
        data: Prediction data (format depends on model)
        timeout: Request timeout in seconds
    
    Returns:
        dict: Prediction response, or None if failed
    """
    try:
        # Try KServe v1 protocol
        response = requests.post(
            f"{model_url.rstrip('/')}/v1/models/:predict",
            json=data,
            timeout=timeout
        )
        if response.status_code == 200:
            return response.json()
        
        # Try KServe v2 protocol
        response = requests.post(
            f"{model_url.rstrip('/')}/v2/models/model/infer",
            json=data,
            timeout=timeout
        )
        if response.status_code == 200:
            return response.json()
        
        # Try simple /predict endpoint
        response = requests.post(
            f"{model_url.rstrip('/')}/predict",
            json=data,
            timeout=timeout
        )
        if response.status_code == 200:
            return response.json()
        
        print(f"⚠ Prediction failed: {response.status_code} - {response.text}")
        return None
    except Exception as e:
        print(f"⚠ Prediction request failed: {e}")
        return None
EOFPY
echo "✓ model_discovery.py created"
echo ""

# Update requirements.txt
echo "Updating requirements.txt..."
if ! grep -q "kubernetes" requirements.txt 2>/dev/null; then
    echo "kubernetes>=28.1.0" >> requirements.txt
fi
if ! grep -q "requests" requirements.txt 2>/dev/null; then
    echo "requests>=2.31.0" >> requirements.txt
fi
echo "✓ requirements.txt updated"
echo ""

# Create deployment manifests
echo "Creating deployment manifests..."
cat > deployments/README.md <<'EOFMD'
# Model Deployments for Testing

This directory contains Kubernetes manifests for deploying real models
to test the Jupyter Notebook Validator Operator.

## Quick Start

```bash
# Deploy all models
./setup-models.sh

# Or deploy individually
oc apply -f sentiment-analysis-model.yaml
oc apply -f fraud-detection-model.yaml
```

## Models

- **sentiment-analysis-model**: Sklearn-based sentiment classifier
- **fraud-detection-model**: Sklearn-based fraud detection model

Both models use KServe InferenceService CRDs and work with OpenShift AI.
EOFMD

cat > deployments/sentiment-analysis-model.yaml <<'EOFYAML'
---
# OpenShift AI Sentiment Analysis Model
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: sentiment-analysis-model
  namespace: mlops
  annotations:
    serving.kserve.io/deploymentMode: "Serverless"
    openshift.io/display-name: "Sentiment Analysis Model"
    openshift.io/description: "Test sentiment analysis model for notebook validation"
spec:
  predictor:
    sklearn:
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "512Mi"
EOFYAML

cat > deployments/fraud-detection-model.yaml <<'EOFYAML'
---
# KServe Fraud Detection Model
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: fraud-detection-model
  namespace: mlops
  annotations:
    serving.kserve.io/deploymentMode: "Serverless"
    openshift.io/display-name: "Fraud Detection Model"
    openshift.io/description: "Test fraud detection model for notebook validation"
spec:
  predictor:
    sklearn:
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "512Mi"
EOFYAML

cat > deployments/setup-models.sh <<'EOFSH'
#!/bin/bash
# Deploy all test models
set -e

NAMESPACE="${NAMESPACE:-mlops}"

echo "Deploying models to namespace: ${NAMESPACE}"

# Create namespace if needed
oc create namespace "${NAMESPACE}" 2>/dev/null || true

# Deploy models
oc apply -f sentiment-analysis-model.yaml
oc apply -f fraud-detection-model.yaml

echo "✓ Models deployed. Waiting for ready status..."
oc wait --for=condition=Ready inferenceservice/sentiment-analysis-model -n "${NAMESPACE}" --timeout=5m || true
oc wait --for=condition=Ready inferenceservice/fraud-detection-model -n "${NAMESPACE}" --timeout=5m || true

echo "✓ Done!"
EOFSH
chmod +x deployments/setup-models.sh

echo "✓ Deployment manifests created"
echo ""

# Create documentation
echo "Creating documentation..."
cat > docs/MODEL_DISCOVERY.md <<'EOFDOC'
# Model Discovery for Test Notebooks

This guide explains how to use the model discovery library in your test notebooks.

## Installation

The `model_discovery.py` library is located in the `lib/` directory.

## Usage in Notebooks

```python
# Cell 1: Import library
import sys
sys.path.append('/workspace/lib')
from model_discovery import discover_models, get_model_endpoint, make_prediction

# Cell 2: Discover models
models = discover_models(platform='openshift-ai')
print(f"Found {len(models)} models:")
for name, info in models.items():
    status = "✓ Ready" if info['ready'] else "⚠ Not Ready"
    print(f"  {status} {name}: {info['url']}")

# Cell 3: Get specific model endpoint
endpoint = get_model_endpoint('sentiment-analysis-model')
if endpoint:
    print(f"Model endpoint: {endpoint}")
    
    # Make prediction
    result = make_prediction(endpoint, {"instances": [[1.0, 2.0, 3.0, 4.0]]})
    print(f"Prediction: {result}")
else:
    print("Model not available - using mock data")
```

## Environment Variables

The operator injects these variables:
- `MODEL_VALIDATION_PLATFORM`: Platform type (kserve, openshift-ai, etc.)
- `MODEL_VALIDATION_TARGET_MODELS`: Comma-separated list of model names
- `NAMESPACE`: Current namespace

## Deploying Test Models

```bash
cd deployments
./setup-models.sh
```

This deploys:
- `sentiment-analysis-model` - For OpenShift AI tests
- `fraud-detection-model` - For KServe tests
EOFDOC
echo "✓ Documentation created"
echo ""

# Git status
echo "=========================================="
echo "Files created/updated:"
echo "=========================================="
git status --short

echo ""
echo "=========================================="
echo "✓ Update Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Test the model discovery library locally"
echo "3. Deploy models: cd deployments && ./setup-models.sh"
echo "4. Update test notebooks to use model discovery"
echo "5. Commit and push: git add . && git commit -m 'Add model discovery' && git push"
echo ""

