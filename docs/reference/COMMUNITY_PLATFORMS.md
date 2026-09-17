# Community-Contributed Model Serving Platforms

> **🚀 We Need Your Help!** This document outlines our vision for supporting multiple model serving platforms. We have built-in support for **KServe** and **OpenShift AI**, but we need the community to help us build integrations for other platforms like **vLLM**, **TorchServe**, **TensorFlow Serving**, **Triton**, **Ray Serve**, **Seldon**, and **BentoML**.
>
> **Your contributions will help thousands of data scientists and ML engineers validate their notebooks against real deployed models!**

---

## 🎯 Why Contribute?

### Impact
- **Help the ML Community**: Enable notebook validation for your favorite model serving platform
- **Showcase Your Expertise**: Demonstrate your knowledge of model serving platforms
- **Build Your Portfolio**: Contribute to a growing open-source project
- **Learn Kubernetes**: Gain hands-on experience with Kubernetes operators and CRDs

### Recognition
- **Contributor Badge**: Get recognized in our `CONTRIBUTORS.md` file
- **Platform Maintainer**: Become the maintainer for your platform integration
- **Community Spotlight**: Featured in our monthly community newsletter
- **Conference Talks**: Opportunity to present your work at KubeCon, MLOps conferences

### Support
- **Mentorship**: Get guidance from core maintainers
- **Code Reviews**: Learn best practices through detailed code reviews
- **Testing Environment**: Access to test clusters for integration testing
- **Documentation Help**: Assistance with writing documentation and examples

---

## 📋 Table of Contents

- [Built-In Platforms](#built-in-platforms-fully-supported)
- [Community Platforms - Help Wanted!](#community-platforms-help-wanted)
- [Platform Comparison Matrix](#platform-comparison-matrix)
- [Contributing a New Platform](#contributing-a-new-platform)
- [Testing Your Integration](#testing-your-platform-integration)
- [Community Support](#community-support)
- [Roadmap](#roadmap)

---

## ✅ Built-In Platforms (Fully Supported)

### KServe (Standard Kubernetes)
- **Status**: ✅ Built-in, fully supported
- **Kubernetes**: Any Kubernetes 1.25+
- **CRD**: `serving.kserve.io/v1beta1/InferenceService`
- **Documentation**: [KServe Docs](https://kserve.github.io/website/)
- **Example**: See `config/samples/model-validation-kserve.yaml`

**Why KServe?**
- Most widely adopted open-source model serving platform
- Works on any Kubernetes cluster
- Supports multiple frameworks (TensorFlow, PyTorch, ONNX, and others)
- Active community and regular updates

**Example Usage:**
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-kserve-notebook
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: model-inference.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
  
  modelValidation:
    enabled: true
    platform: kserve
    phase: both
    targetModels:
      - sklearn-iris-model
```

---

### OpenShift AI
- **Status**: ✅ Built-in, fully supported
- **Platform**: OpenShift 4.20+
- **CRD**: `serving.kserve.io/v1beta1/InferenceService` (KServe-based)
- **Documentation**: [OpenShift AI Docs](https://docs.redhat.com/en/documentation/red_hat_openshift_ai_self-managed/)
- **Example**: See `config/samples/model-validation-openshift-ai.yaml`

**Why OpenShift AI?**
- Enterprise-grade AI/ML platform from Red Hat
- Our development and test environment
- Includes model registry, workbenches, and pipelines
- Integrated security and compliance features

**Example Usage:**
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-openshift-ai-notebook
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: fraud-detection.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/pytorch-notebook:latest
    serviceAccountName: model-validator-sa
  
  modelValidation:
    enabled: true
    platform: openshift-ai
    phase: existing
    targetModels:
      - fraud-detection-model
    predictionValidation:
      enabled: true
      testData: |
        {"instances": [[1.0, 2.0, 3.0, 4.0]]}
      expectedOutput: |
        {"predictions": [[0.95, 0.05]]}
      tolerance: 0.01
```

---

## 📚 Community Platforms - Help Wanted!

> **🙋 We Need Contributors!** The following platforms are **waiting for community contributions**. Each platform needs:
> - Integration guide documentation
> - Example notebooks
> - Sample CRD manifests
> - Integration tests
>
> **Pick a platform you know and love, and help us build it!** See [Contributing a New Platform](#contributing-a-new-platform) for step-by-step guidance.

The following platforms are **documented for community contributions**. We welcome PRs to add full support!

### 🚀 vLLM (LLM Serving) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: Large Language Models (LLMs)
- **CRD**: `serving.kserve.io/v1beta1/InferenceService` (via KServe)
- **Runtime**: vLLM ServingRuntime
- **Documentation**: ⚠️ `docs/reference/vllm.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-vllm.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20vLLM%20integration)**

**Key Features:**
- Optimized for LLM inference (Llama, Mistral, GPT, and others)
- PagedAttention for efficient memory usage
- OpenAI-compatible API
- Continuous batching for high throughput

**Use Cases:**
- ✅ Validating LLM prompt engineering notebooks
- ✅ Testing chatbot integration
- ✅ Verifying RAG (Retrieval-Augmented Generation) pipelines
- ✅ Benchmarking LLM performance

**Example Notebook Validation:**
```python
# Cell 1: Test vLLM model availability
import requests
import os

model_url = os.environ.get('VLLM_MODEL_URL', 'http://llama-2-7b.default.svc.cluster.local')
health_response = requests.get(f"{model_url}/health")
assert health_response.status_code == 200, "vLLM model is not healthy"

# Cell 2: Test prediction
prompt = "What is machine learning?"
response = requests.post(
    f"{model_url}/v1/completions",
    json={"prompt": prompt, "max_tokens": 100}
)
assert response.status_code == 200, "Prediction failed"
print(f"Response: {response.json()['choices'][0]['text']}")
```

---

### 🔥 TorchServe (PyTorch Models) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: PyTorch model serving
- **CRD**: `serving.kserve.io/v1beta1/InferenceService` (via KServe)
- **Documentation**: ⚠️ `docs/reference/torchserve.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-torchserve.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20TorchServe%20integration)**

**Key Features:**
- Native PyTorch support
- Multi-model serving
- A/B testing support
- Metrics and logging

**Use Cases:**
- ✅ Computer vision model validation
- ✅ NLP model testing
- ✅ Custom PyTorch model deployment
- ✅ Model versioning and rollback

---

### 🧠 TensorFlow Serving - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: TensorFlow model serving
- **CRD**: `serving.kserve.io/v1beta1/InferenceService` (via KServe)
- **Documentation**: ⚠️ `docs/reference/tensorflow-serving.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-tensorflow.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20TensorFlow%20Serving%20integration)**

**Key Features:**
- Native TensorFlow support
- SavedModel format
- gRPC and REST APIs
- Model versioning

**Use Cases:**
- ✅ TensorFlow model validation
- ✅ Keras model deployment
- ✅ Production-grade serving
- ✅ High-throughput inference

---

### ⚡ Triton Inference Server (NVIDIA) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: Multi-framework, GPU-optimized serving
- **CRD**: `serving.kserve.io/v1beta1/InferenceService` (via KServe)
- **Documentation**: ⚠️ `docs/reference/triton.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-triton.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20Triton%20integration)**

**Key Features:**
- Multi-framework support (TensorFlow, PyTorch, ONNX, TensorRT)
- Dynamic batching
- Model ensembles
- GPU optimization

**Use Cases:**
- ✅ GPU-accelerated inference
- ✅ Multi-model pipelines
- ✅ High-performance serving
- ✅ ONNX model deployment

---

### 🌟 Ray Serve (Distributed Serving) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: Distributed, scalable model serving
- **CRD**: `ray.io/v1alpha1/RayService`
- **Documentation**: ⚠️ `docs/reference/ray-serve.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-ray-serve.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20Ray%20Serve%20integration)**

**Key Features:**
- Distributed serving across multiple nodes
- Python-native API
- Model composition
- Autoscaling

**Use Cases:**
- ✅ Large-scale distributed inference
- ✅ Complex model pipelines
- ✅ Multi-stage ML workflows
- ✅ Custom serving logic

---

### 🎯 Seldon Core (Advanced ML) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: Advanced ML deployments, explainability
- **CRD**: `machinelearning.seldon.io/v1/SeldonDeployment`
- **Documentation**: ⚠️ `docs/reference/seldon.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-seldon.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20Seldon%20Core%20integration)**

**Key Features:**
- Advanced deployment strategies (canary, shadow, A/B)
- Model explainability
- Outlier detection
- Drift detection

**Use Cases:**
- ✅ A/B testing validation
- ✅ Model explainability testing
- ✅ Drift detection validation
- ✅ Advanced deployment patterns

---

### 🍱 BentoML (Model Packaging) - 🙋 **HELP WANTED**
- **Status**: � **Needs Contributor** - Documentation and examples needed
- **Use Case**: Model packaging and serving
- **CRD**: `serving.yatai.ai/v1alpha1/BentoDeployment`
- **Documentation**: ⚠️ `docs/reference/bentoml.md` - **TO BE CREATED**
- **Example**: ⚠️ `config/samples/community/model-validation-bentoml.yaml` - **TO BE CREATED**
- **Volunteer**: 🙋 **[Claim this platform!](https://github.com/your-org/jupyter-notebook-validator-operator/issues/new?title=Volunteer%20for%20BentoML%20integration)**

**Key Features:**
- Model packaging and versioning
- Multi-framework support
- API generation
- Containerization

**Use Cases:**
- ✅ Model packaging validation
- ✅ API contract testing
- ✅ Deployment readiness checks
- ✅ Version compatibility

---

## 📊 Platform Comparison Matrix

| Platform | Use Case | Built-In | Community | Kubernetes | OpenShift | GPU Required | Complexity |
|----------|----------|----------|-----------|------------|-----------|--------------|------------|
| **KServe** | General ML | ✅ | - | ✅ | ✅ | Optional | Low |
| **OpenShift AI** | Enterprise ML | ✅ | - | ❌ | ✅ | Optional | Low |
| **vLLM** | LLMs | ❌ | 📚 | ✅ | ✅ | Recommended | Medium |
| **TorchServe** | PyTorch | ❌ | 📚 | ✅ | ✅ | Optional | Low |
| **TensorFlow Serving** | TensorFlow | ❌ | 📚 | ✅ | ✅ | Optional | Low |
| **Triton** | Multi-framework | ❌ | 📚 | ✅ | ✅ | Recommended | Medium |
| **Ray Serve** | Distributed | ❌ | 📚 | ✅ | ✅ | Optional | High |
| **Seldon** | Advanced ML | ❌ | 📚 | ✅ | ✅ | Optional | High |
| **BentoML** | Packaging | ❌ | 📚 | ✅ | ✅ | Optional | Medium |

---

## 🤝 Contributing a New Platform

> **🎉 Ready to contribute?** We've made it easy! Follow this step-by-step guide to add support for your favorite model serving platform.

### Quick Start: Claim Your Platform

1. **Pick a Platform**: Choose from the list above or propose a new one
2. **Claim It**: Open an issue using the "Volunteer" link or create a new issue
3. **Get Support**: Join our Slack channel and introduce yourself
4. **Start Building**: Follow the steps below

---

### Step-by-Step Contribution Guide

#### Step 1: Create Platform Documentation (30-60 minutes)

Create `docs/reference/{platform-name}.md` with:

**Required Sections:**
- ✅ **Platform Overview**: What is it? Why use it?
- ✅ **Key Features**: What makes it unique?
- ✅ **Installation Guide**: How to install on Kubernetes/OpenShift
- ✅ **CRD Schema**: What CRDs does it use?
- ✅ **API Endpoints**: Health check and prediction endpoints
- ✅ **Example Notebook**: Working validation example
- ✅ **Troubleshooting**: Common issues and solutions

**Template Available**: Copy from `docs/reference/TEMPLATE.md` (we'll create this for you!)

**Example Structure:**
```markdown
# {Platform Name} Integration Guide

## Overview
Brief description of the platform...

## Installation
```bash
kubectl apply -f https://...
```

## CRD Schema
```yaml
apiVersion: your.api.group/v1
kind: YourResource
...
```

## Health Check Endpoint
`GET /health` - Returns 200 if healthy

## Prediction Endpoint
`POST /predict` - Accepts JSON, returns predictions

## Example Notebook
See `config/samples/community/notebook-{platform}-validation.ipynb`

## Troubleshooting
...
```

---

#### Step 2: Add Platform Definition (15 minutes)

Update `pkg/platform/detector.go`:

```go
// In CommunityPlatforms map
"your-platform": {
    Name:              "Your Platform",
    APIGroup:          "your.api.group",
    APIVersion:        "v1",
    ResourceKind:      "YourResource",
    HealthEndpoint:    "/health",
    PredictionEndpoint: "/predict",
},
```

**Need Help?** Look at existing platform definitions for reference.

---

#### Step 3: Create Example Manifests (30 minutes)

Add to `config/samples/community/`:

**1. CRD Example** - `model-validation-{platform}.yaml`
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-{platform}-notebook
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: {platform}-inference.ipynb

  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa

  modelValidation:
    enabled: true
    platform: {platform}
    phase: both
    targetModels:
      - my-model
```

**2. Example Notebook** - `notebook-{platform}-validation.ipynb`
- Cell 1: Import libraries
- Cell 2: Check platform availability
- Cell 3: Test model health
- Cell 4: Test prediction
- Cell 5: Validate output

**3. Model Deployment Example** - `{platform}-inferenceservice.yaml`
- Example of deploying a model on your platform

---

#### Step 4: Add Tests (Optional but Recommended) (60 minutes)

Create `test/e2e/community/{platform}_test.go`:

```go
package community

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPlatformDetection(t *testing.T) {
    // Test platform detection logic
}

func TestHealthCheck(t *testing.T) {
    // Test health check endpoint
}

func TestPredictionValidation(t *testing.T) {
    // Test prediction validation
}
```

**Don't worry if you're not a Go expert!** We'll help you with the tests during code review.

---

#### Step 5: Submit Your PR (15 minutes)

**Checklist before submitting:**
- [ ] Platform documentation created (`docs/reference/{platform}.md`)
- [ ] Platform definition added (`pkg/platform/detector.go`)
- [ ] Example CRD manifest created
- [ ] Example notebook created
- [ ] Model deployment example created
- [ ] Tests added (optional)
- [ ] Updated `COMMUNITY_PLATFORMS.md` (this file) - change status from 🔴 to ✅
- [ ] Updated `docs/adrs/020-model-aware-validation-strategy.md`
- [ ] Added yourself to `CONTRIBUTORS.md`

**PR Template:**
```markdown
## Platform Integration: {Platform Name}

### Summary
This PR adds support for {platform name} model serving platform.

### What's Included
- [ ] Platform documentation
- [ ] Platform definition
- [ ] Example manifests
- [ ] Example notebook
- [ ] Tests (optional)

### Testing
- Tested on Kubernetes version: X.Y.Z
- Tested with {platform} version: X.Y.Z
- Example model deployed: {model name}

### Screenshots
[Optional: Add screenshots of successful validation]

### Related Issues
Closes #XXX
```

---

### 🎁 What You Get

**Recognition:**
- ✅ Your name in `CONTRIBUTORS.md`
- ✅ Platform maintainer badge
- ✅ Featured in monthly newsletter
- ✅ Invitation to monthly community call

**Support:**
- ✅ Code review from core maintainers
- ✅ Help with Go code and Kubernetes concepts
- ✅ Access to test clusters
- ✅ Documentation review and editing

**Growth:**
- ✅ Learn Kubernetes operators
- ✅ Build your open-source portfolio
- ✅ Network with ML/MLOps community
- ✅ Potential speaking opportunities

---

### 💬 Need Help?

**Before You Start:**
- Read the [Contribution Guidelines](../CONTRIBUTING.md)
- Join our [Slack channel](https://kubernetes.slack.com/archives/jupyter-notebook-validator)
- Attend [Office Hours](https://calendar.google.com/...) (Fridays 2-3 PM EST)

**During Development:**
- Ask questions in Slack
- Tag `@maintainers` in your PR for help
- Request a pairing session for complex issues

**We're here to help you succeed!** 🚀

---

## 🧪 Testing Your Platform Integration

### Phase 1: Clean Environment Test
```bash
# Deploy your platform
kubectl apply -f your-platform-install.yaml

# Run validation
kubectl apply -f config/samples/community/model-validation-your-platform.yaml

# Check results
kubectl get notebookvalidationjob -o yaml
kubectl logs -l job-name=your-validation-job
```

### Phase 2: Existing Environment Test
```bash
# Deploy a test model
kubectl apply -f your-test-model.yaml

# Wait for model to be ready
kubectl wait --for=condition=Ready inferenceservice/your-model --timeout=5m

# Run validation with prediction checks
kubectl apply -f config/samples/community/model-validation-your-platform-predictions.yaml

# Verify predictions
kubectl logs -l job-name=your-validation-job | grep "Prediction"
```

---

## 💬 Community Support

- **GitHub Discussions**: [Community Platforms](https://github.com/your-org/jupyter-notebook-validator-operator/discussions/categories/community-platforms)
- **Slack**: [#jupyter-notebook-validator](https://kubernetes.slack.com/archives/jupyter-notebook-validator)
- **Monthly Community Call**: First Tuesday of each month at 10 AM EST
- **Office Hours**: Every Friday 2-3 PM EST

---

## 🗺️ Roadmap

### Release 1.0 (Current - Q4 2025)
- ✅ KServe (built-in)
- ✅ OpenShift AI (built-in)
- 📚 Community platform documentation

### Release 1.1 (Q1 2026)
- 🎯 vLLM community integration (first community platform)
- 🎯 Enhanced prediction validation
- 🎯 Model registry integration

### Release 1.2+ (Q2 2026 - Community-Driven)
- 🎯 TorchServe, TensorFlow Serving, Triton (community PRs)
- 🎯 Ray Serve, Seldon, BentoML (community PRs)
- 🎯 Custom platform plugin system

---

**🎉 We welcome your contributions! Join us in building the most comprehensive notebook validation platform for Kubernetes.**

