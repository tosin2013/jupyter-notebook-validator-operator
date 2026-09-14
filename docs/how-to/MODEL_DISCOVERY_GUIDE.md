# Model Discovery Guide for Jupyter Notebook Validator Operator

**Date:** 2025-11-08  
**Status:** Active  
**Audience:** Data Scientists, ML Engineers, Platform Teams

---

## Overview

This guide explains how to create notebooks that discover and validate against **real deployed models** in your cluster, rather than making assumptions about what models exist.

## Available Platforms on This Cluster

Based on `oc get csv`, this OpenShift cluster has:

- ✅ **Red Hat OpenShift AI (v2.22.2)** - Enterprise AI platform with KServe
- ✅ **KServe CRDs** - InferenceServices, ServingRuntimes, Predictors
- ✅ **Knative Serving** - Foundation for KServe
- ✅ **NVIDIA GPU Operator (v24.9.2)** - GPU support for models
- ✅ **External Secrets Operator (v0.11.0)** - Credential management

## Model Discovery Helper Library

### Installation

Add to your test notebooks repository at `model_discovery.py`:

```python
"""
Model Discovery Helper Library for Jupyter Notebook Validator Operator

This library helps notebooks discover and interact with deployed models
without hardcoding model names or endpoints.

Usage:
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
        
        print(f"⚠ Prediction failed: {response.status_code} - {response.text}")
        return None
    except Exception as e:
        print(f"⚠ Prediction request failed: {e}")
        return None
```

## Real Model Deployments

### OpenShift AI Sentiment Analysis Model

Create a real sentiment analysis model using a pre-trained transformer:

**File:** `deployments/openshift-ai-sentiment-model.yaml`

```yaml
---
# ServingRuntime for ONNX models on OpenShift AI
apiVersion: serving.kserve.io/v1alpha1
kind: ServingRuntime
metadata:
  name: onnx-runtime
  namespace: mlops
spec:
  supportedModelFormats:
    - name: onnx
      version: "1"
  containers:
    - name: kserve-container
      image: mcr.microsoft.com/onnxruntime/server:latest
      args:
        - --model_path=/mnt/models
      resources:
        requests:
          cpu: "500m"
          memory: "512Mi"
        limits:
          cpu: "1"
          memory: "1Gi"

---
# InferenceService for sentiment analysis
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: sentiment-analysis-model
  namespace: mlops
  annotations:
    serving.kserve.io/deploymentMode: "Serverless"
spec:
  predictor:
    model:
      modelFormat:
        name: onnx
      runtime: onnx-runtime
      storageUri: "https://huggingface.co/optimum/distilbert-base-uncased-finetuned-sst-2-english/resolve/main/model.onnx"
      resources:
        requests:
          cpu: "500m"
          memory: "512Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
```

### KServe Fraud Detection Model

Create a scikit-learn fraud detection model:

**File:** `deployments/kserve-fraud-detection-model.yaml`

```yaml
---
# InferenceService for fraud detection (sklearn)
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: fraud-detection-model
  namespace: mlops
spec:
  predictor:
    sklearn:
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
```

## Deployment Instructions

### Step 1: Create Namespace and RBAC

```bash
# Create namespace
oc create namespace mlops

# Create ServiceAccount
oc create serviceaccount model-validator-sa -n mlops

# Create Role for model access
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-validator-role
  namespace: mlops
rules:
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices", "servingruntimes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get", "list"]
EOF

# Create RoleBinding
oc create rolebinding model-validator-binding \
  --role=model-validator-role \
  --serviceaccount=mlops:model-validator-sa \
  -n mlops
```

### Step 2: Deploy Models

```bash
# Deploy OpenShift AI sentiment model
oc apply -f deployments/openshift-ai-sentiment-model.yaml

# Deploy KServe fraud detection model
oc apply -f deployments/kserve-fraud-detection-model.yaml

# Wait for models to be ready
oc wait --for=condition=Ready inferenceservice/sentiment-analysis-model -n mlops --timeout=5m
oc wait --for=condition=Ready inferenceservice/fraud-detection-model -n mlops --timeout=5m
```

### Step 3: Verify Deployment

```bash
# Check InferenceServices
oc get inferenceservices -n mlops

# Check model endpoints
oc get inferenceservice sentiment-analysis-model -n mlops -o jsonpath='{.status.url}'
oc get inferenceservice fraud-detection-model -n mlops -o jsonpath='{.status.url}'
```

## Updated Test Notebooks

See the updated notebooks in the test repository:
- `model-validation/openshift-ai/sentiment-analysis-test.ipynb` - Uses real sentiment model
- `model-validation/kserve/model-inference-kserve.ipynb` - Uses real fraud detection model

Both notebooks now:
1. ✅ Discover models dynamically using `model_discovery.py`
2. ✅ Check model health before making predictions
3. ✅ Handle cases where models aren't deployed (graceful degradation)
4. ✅ Provide clear feedback about model availability

## Troubleshooting

### Models Not Discovered

```python
# Check RBAC permissions
from kubernetes import client, config
config.load_incluster_config()
api = client.RbacAuthorizationV1Api()

# List roles
roles = api.list_namespaced_role('mlops')
for role in roles.items:
    print(f"Role: {role.metadata.name}")
```

### Model Not Ready

```bash
# Check InferenceService status
oc describe inferenceservice sentiment-analysis-model -n mlops

# Check pods
oc get pods -n mlops -l serving.kserve.io/inferenceservice=sentiment-analysis-model

# Check logs
oc logs -n mlops -l serving.kserve.io/inferenceservice=sentiment-analysis-model
```

---

**Next Steps:**
1. Deploy the model discovery library to your test notebooks repository
2. Deploy the real models to your OpenShift cluster
3. Update test notebooks to use model discovery
4. Run validation jobs to test end-to-end

