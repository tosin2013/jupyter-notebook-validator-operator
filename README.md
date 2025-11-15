# Jupyter Notebook Validator Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/tosin2013/jupyter-notebook-validator-operator)](https://goreportcard.com/report/github.com/tosin2013/jupyter-notebook-validator-operator)
[![CI](https://github.com/tosin2013/jupyter-notebook-validator-operator/actions/workflows/ci.yml/badge.svg)](https://github.com/tosin2013/jupyter-notebook-validator-operator/actions/workflows/ci.yml)
[![Tier 1 Tests](https://github.com/tosin2013/jupyter-notebook-validator-operator/actions/workflows/ci-unit-tests.yaml/badge.svg)](https://github.com/tosin2013/jupyter-notebook-validator-operator/actions/workflows/ci-unit-tests.yaml)
[![OpenShift](https://img.shields.io/badge/OpenShift-4.18+-red.svg)](https://www.openshift.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.31+-blue.svg)](https://kubernetes.io/)

A Kubernetes-native operator that automates Jupyter Notebook validation in MLOps workflows. Built with Operator SDK and Go, it provides Git integration, pod orchestration for notebook execution, golden notebook comparison for regression testing, and model-aware validation for ML/AI workloads.

## Overview

The Jupyter Notebook Validator Operator enables automated testing and validation of Jupyter notebooks in Kubernetes and OpenShift environments. It's designed for data science teams, ML engineers, and platform teams who need to ensure notebook reliability, reproducibility, and integration with deployed ML models.

### Key Features

- **🔄 Automated Notebook Execution** - Execute notebooks in isolated Kubernetes pods with Papermill
- **📊 Golden Notebook Comparison** - Regression testing with cell-by-cell output comparison
- **🔐 Credential Management** - Secure injection of credentials (AWS, databases, APIs) via Secrets, ESO, or Vault
- **🤖 Model-Aware Validation** - Validate notebooks against deployed models (KServe, OpenShift AI, vLLM, etc.)
- **🔍 Git Integration** - Clone notebooks from Git repositories (HTTPS and SSH authentication)
- **📈 Observability** - Prometheus metrics and structured logging with credential sanitization
- **🎯 Platform Detection** - Auto-detect model serving platforms (9 platforms supported)
- **🔒 Security** - RBAC, Pod Security Standards, secret rotation, and audit logging

## Quick Start

### Prerequisites

- **Kubernetes/OpenShift Cluster:** OpenShift 4.18+ (recommended) or Kubernetes 1.31+
- **Command-line Tools:** kubectl or oc CLI, make (for building from source)
- **Optional:** External Secrets Operator (ESO), KServe or OpenShift AI, Tekton Pipelines (for build integration)

### Installation

```bash
# Install CRDs
make install

# Build and push image
make docker-build docker-push IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:v0.1.0

# Deploy operator
make deploy IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:v0.1.0
```

### Verify Installation

```bash
kubectl get pods -n jupyter-notebook-validator-operator-system
kubectl get crd notebookvalidationjobs.mlops.mlops.dev
```

## Usage Examples

See [config/samples/](config/samples/) for complete examples.

### Basic Validation

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: simple-validation
spec:
  notebook:
    git:
      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
      ref: main
    path: notebooks/tier1-simple/01-hello-world.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
```

## Documentation

- **[Architecture Overview](docs/ARCHITECTURE_OVERVIEW.md)** - System design
- **[Testing Guide](docs/TESTING_GUIDE.md)** - Testing procedures
- **[Notebook Credentials Guide](docs/NOTEBOOK_CREDENTIALS_GUIDE.md)** - Credential injection
- **[Model Discovery Guide](docs/MODEL_DISCOVERY_GUIDE.md)** - Model validation
- **[Community Platforms](docs/COMMUNITY_PLATFORMS.md)** - Supported platforms
- **[ADRs](docs/adrs/)** - Architectural decisions

## Supported Platforms

- **Model Serving:** KServe, OpenShift AI, vLLM, TorchServe, TensorFlow Serving, Triton, Ray Serve, Seldon, BentoML
- **Credential Management:** Kubernetes Secrets, External Secrets Operator (ESO), HashiCorp Vault

## License

Copyright 2025 Tosin Akinosho. Licensed under the Apache License, Version 2.0.
