# Model Validation Multi-User Guide

This guide explains how to configure model-aware validation for multi-user and multi-environment deployments of the Jupyter Notebook Validator Operator.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Namespace Isolation](#namespace-isolation)
- [RBAC Configuration](#rbac-configuration)
- [Use Cases](#use-cases)
- [Configuration Examples](#configuration-examples)
- [Troubleshooting](#troubleshooting)

---

## Overview

The model-aware validation feature enables notebooks to validate against deployed ML models (KServe, OpenShift AI, vLLM, etc.). In multi-user environments, proper namespace isolation and RBAC configuration are essential for security and resource management.

### Key Concepts

1. **Namespace Isolation**: By default, validation jobs can only access models in their own namespace
2. **Cross-Namespace Access**: Optional, requires explicit RBAC configuration
3. **ServiceAccount Separation**: Each team/user can have their own ServiceAccount with appropriate permissions
4. **Model References**: Support for both `model-name` (same namespace) and `namespace/model-name` (cross-namespace) formats

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐        ┌─────────────────────────────────────┐│
│  │ Operator NS     │        │ shared-models Namespace             ││
│  │                 │        │                                     ││
│  │  ┌───────────┐  │        │  ┌─────────────────────────────┐   ││
│  │  │ Operator  │  │        │  │ InferenceService:           │   ││
│  │  │ Controller│  │        │  │   fraud-detection-model     │   ││
│  │  └───────────┘  │        │  │   risk-scoring-model        │   ││
│  │                 │        │  └─────────────────────────────┘   ││
│  └─────────────────┘        │                                     ││
│                             │  RoleBinding:                       ││
│  ┌─────────────────┐        │    team-a → model-validator         ││
│  │ team-a NS       │        │    team-b → model-validator         ││
│  │                 │        └─────────────────────────────────────┘│
│  │  ┌───────────┐  │                                               │
│  │  │ NVJob:    │──┼───────────────────────────────────────────────│
│  │  │ user1-val │  │        Access: shared-models/fraud-detection  │
│  │  └───────────┘  │                                               │
│  │                 │        ┌─────────────────────────────────────┐│
│  │  ServiceAccount:│        │ team-b Namespace                    ││
│  │  model-validator│        │                                     ││
│  └─────────────────┘        │  ┌───────────┐                      ││
│                             │  │ NVJob:    │──────────────────────┼│
│                             │  │ team-b-val│                      ││
│                             │  └───────────┘                      ││
│                             │  ServiceAccount: model-validator    ││
│                             └─────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

---

## Namespace Isolation

### Default Behavior (Same-Namespace Only)

By default, validation jobs can only access models within their own namespace. This provides security isolation between teams.

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation
  namespace: team-a  # Job runs here
spec:
  modelValidation:
    enabled: true
    platform: kserve
    targetModels:
      - fraud-detection-model  # Resolves to team-a/fraud-detection-model
      - risk-model            # Resolves to team-a/risk-model
```

### Cross-Namespace Access

To access models in other namespaces, use the `namespace/model` format and configure appropriate RBAC:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: cross-ns-validation
  namespace: team-a  # Job runs here
spec:
  modelValidation:
    enabled: true
    platform: kserve
    targetModels:
      - shared-models/fraud-detection-model  # Cross-namespace access
      - shared-models/risk-model             # Cross-namespace access
      - internal-model                       # Same namespace (team-a)
```

---

## RBAC Configuration

### Option 1: Namespace-Scoped (Default, Most Secure)

For single-namespace access, create a Role and RoleBinding in the validation namespace:

```yaml
# Role for model access within namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-validator-role
  namespace: team-a
rules:
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices/status"]
    verbs: ["get"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["services", "endpoints", "pods"]
    verbs: ["get", "list", "watch"]

---
# ServiceAccount for validation pods
apiVersion: v1
kind: ServiceAccount
metadata:
  name: model-validator-sa
  namespace: team-a

---
# Bind role to ServiceAccount
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: model-validator-binding
  namespace: team-a
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: model-validator-role
subjects:
  - kind: ServiceAccount
    name: model-validator-sa
    namespace: team-a
```

### Option 2: Cross-Namespace Access (Shared Models)

For accessing models in a shared namespace:

```yaml
# ClusterRole with model access permissions
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: model-validator-cross-namespace
rules:
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices/status"]
    verbs: ["get"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["services", "endpoints", "pods"]
    verbs: ["get", "list", "watch"]

---
# RoleBinding in the TARGET namespace (where models live)
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: team-a-model-access
  namespace: shared-models  # Target namespace with models
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: model-validator-cross-namespace
subjects:
  - kind: ServiceAccount
    name: model-validator-sa
    namespace: team-a  # Source namespace
```

### Option 3: Platform Team (Cluster-Wide)

For platform teams that need to validate models across all namespaces:

```yaml
# ClusterRoleBinding for cluster-wide access
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: platform-model-validator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: model-validator-cross-namespace
subjects:
  - kind: ServiceAccount
    name: platform-model-validator-sa
    namespace: ml-platform-admin
```

---

## Use Cases

### Use Case 1: Single Team with Shared Models

A data science team with their own namespace, accessing shared models:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: team-validation
  namespace: ds-team
spec:
  notebook:
    path: inference-notebook.ipynb
    git:
      url: https://github.com/example/notebooks.git
      ref: main
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
  
  modelValidation:
    enabled: true
    platform: kserve
    phase: existing
    targetModels:
      - ml-platform/production-model  # Shared production model
      - staging-model                 # Team's own staging model
    
    predictionValidation:
      enabled: true
      testData: '{"instances": [[1.0, 2.0, 3.0]]}'
      expectedOutput: '{"predictions": [[0.8]]}'
      tolerance: "0.05"
```

### Use Case 2: Multi-Environment Validation

Validating against models in different environments (staging, production):

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: multi-env-validation
  namespace: ci-cd
spec:
  notebook:
    path: integration-tests.ipynb
    git:
      url: https://github.com/example/notebooks.git
      ref: main
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: ci-model-validator-sa
  
  modelValidation:
    enabled: true
    platform: kserve
    phase: existing
    targetModels:
      - staging/fraud-model-v2    # Staging environment
      - production/fraud-model-v1 # Production environment (read-only check)
    timeout: "10m"
```

### Use Case 3: Platform Health Monitoring

Platform team checking model health across all namespaces:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: platform-health-check
  namespace: ml-platform
spec:
  notebook:
    path: health-check.ipynb
    git:
      url: https://github.com/example/platform-notebooks.git
      ref: main
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: platform-model-validator-sa
  
  modelValidation:
    enabled: true
    platform: kserve
    phase: both  # Check platform AND model health
    targetModels:
      - team-a/model-1
      - team-b/model-2
      - production/model-3
      - staging/model-4
    timeout: "15m"
```

---

## Configuration Examples

### Full Sample Files

See the following sample configurations in `config/samples/`:

| File | Description |
|------|-------------|
| `model-validation-kserve.yaml` | Basic KServe model validation (single namespace) |
| `model-validation-openshift-ai.yaml` | OpenShift AI integration |
| `model-validation-multi-user.yaml` | Multi-user same-namespace scenario |
| `model-validation-cross-namespace.yaml` | Cross-namespace model access |

### Environment Variables

The operator sets these environment variables in validation pods:

| Variable | Description |
|----------|-------------|
| `MODEL_VALIDATION_ENABLED` | "true" if model validation is enabled |
| `MODEL_VALIDATION_PLATFORM` | Platform type (kserve, openshift-ai, etc.) |
| `MODEL_VALIDATION_NAMESPACE` | Namespace where the validation job runs |
| `MODEL_VALIDATION_PHASE` | Validation phase (clean, existing, both) |
| `MODEL_VALIDATION_TARGET_MODELS` | Comma-separated list of resolved model refs |
| `MODEL_VALIDATION_TARGET_NAMESPACES` | Unique namespaces being accessed |
| `MODEL_VALIDATION_TIMEOUT` | Validation timeout |

---

## Troubleshooting

### Common Issues

#### 1. "Cross-namespace model access denied"

**Cause**: Attempting to access a model in a different namespace without proper RBAC.

**Solution**: Create a RoleBinding in the target namespace:

```bash
# Check if RoleBinding exists
kubectl get rolebinding -n <target-namespace> | grep model-validator

# Create RoleBinding if missing
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: allow-<source-namespace>-access
  namespace: <target-namespace>
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: model-validator-cross-namespace
subjects:
  - kind: ServiceAccount
    name: model-validator-sa
    namespace: <source-namespace>
EOF
```

#### 2. "Namespace not in allowed list"

**Cause**: Cross-namespace access is enabled but the target namespace is restricted.

**Solution**: This is a safety feature. Either:
- Add the namespace to the allowed list in your configuration
- Use the default same-namespace mode

#### 3. "Platform detection failed"

**Cause**: The specified model serving platform is not installed in the cluster.

**Solution**: Check if the platform CRDs are installed:

```bash
# Check for KServe
kubectl get crd | grep kserve

# Check for Ray Serve
kubectl get crd | grep ray

# Check for Seldon
kubectl get crd | grep seldon
```

#### 4. "Model health check failed"

**Cause**: The target model exists but is not healthy.

**Solution**: Check the model status:

```bash
# For KServe
kubectl get inferenceservice <model-name> -n <namespace> -o yaml

# Check pod status
kubectl get pods -n <namespace> -l serving.kserve.io/inferenceservice=<model-name>
```

### Debugging Commands

```bash
# View validation job status
kubectl get notebookvalidationjob <job-name> -n <namespace> -o yaml

# Check model validation result
kubectl get notebookvalidationjob <job-name> -n <namespace> \
  -o jsonpath='{.status.modelValidationResult}'

# List all InferenceServices in a namespace
kubectl get inferenceservice -n <namespace>

# Check ServiceAccount permissions
kubectl auth can-i get inferenceservices -n <namespace> \
  --as=system:serviceaccount:<job-namespace>:model-validator-sa
```

---

## Best Practices

1. **Use Namespace Isolation by Default**: Only enable cross-namespace access when necessary

2. **Separate ServiceAccounts per Team**: Each team should have their own ServiceAccount

3. **Minimal RBAC Permissions**: Grant only read access (get, list, watch) for model validation

4. **Use Labels for Organization**: Label your NotebookValidationJobs for easy filtering:
   ```yaml
   labels:
     team: team-a
     user: user1
     environment: staging
   ```

5. **Monitor with Metrics**: The operator exposes Prometheus metrics for model validation:
   - `model_validation_duration_seconds`
   - `platform_detection_success_total`
   - `model_health_check_total`

---

## Related Documentation

- [ADR-020: Model-Aware Validation Strategy](adrs/020-model-aware-validation-strategy.md)
- [NAMESPACE_SETUP.md](NAMESPACE_SETUP.md) - General namespace setup guide
- [ADR-005: RBAC and Service Account Model](adrs/005-rbac-and-service-account-model.md)
