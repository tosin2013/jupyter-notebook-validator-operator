# Release Notes: v1.0.0-ocp4.18

**Release Date:** January 9, 2025  
**Target Platform:** OpenShift 4.18 (Kubernetes 1.31)  
**Branch:** `release-4.18`

## 🎉 First Stable Release for OpenShift 4.18

This is the first stable release of the Jupyter Notebook Validator Operator, specifically built and tested for OpenShift 4.18.21. This release includes comprehensive build integration with S2I and Tekton, model-aware validation, and golden notebook comparison.

## ✨ Key Features

### 1. **S2I and Tekton Build Integration** (Phase 4.5)
- **Automatic Build Strategy Detection**: Detects available build backends (S2I, Tekton) on the cluster
- **Custom Image Building**: Build custom container images with notebook dependencies
- **Build Status Tracking**: Real-time build status in CRD status with conditions
- **Build Reuse**: Intelligently reuses existing builds when available
- **Fallback Mechanism**: Falls back to spec container image on build failure
- **Timeout Handling**: Configurable build timeout (default: 15 minutes)

**Supported Build Strategies:**
- **S2I (Source-to-Image)**: OpenShift native build strategy
- **Tekton Pipelines**: Cloud-native CI/CD build strategy

### 2. **Model-Aware Validation** (Phase 4.4)
- **Platform Detection**: Automatic detection of model serving platforms
- **Supported Platforms**: KServe, Seldon Core, Ray Serve, BentoML, Custom
- **Two-Phase Validation**:
  - **Phase 1 (Clean)**: Validate notebook trains and deploys model from scratch
  - **Phase 2 (Existing)**: Validate notebook works with existing deployed model
- **Prediction Validation**: Test model predictions with sample data

### 3. **Golden Notebook Comparison** (Phase 3)
- **Output Comparison**: Compare notebook execution outputs against golden notebook
- **Flexible Comparison**: Exact or normalized comparison strategies
- **Floating-Point Tolerance**: Configurable tolerance for numerical comparisons
- **Cell-by-Cell Results**: Detailed comparison results for each cell

### 4. **Comprehensive Testing**
- **Unit Tests**: 26 tests, 52.1% coverage, all passing ✅
- **Integration Tests**: 8 tests, validated on OpenShift 4.18.21 ✅
- **E2E Test Infrastructure**: Complete workflow tests ready for deployment ✅

## 🔧 Platform Requirements

### Supported Versions

| Component | Supported Version | Notes |
|-----------|-------------------|-------|
| OpenShift | 4.18.x | Kubernetes 1.31 |
| OpenShift Pipelines | 1.17.x, 1.18.x | Tekton Pipeline v0.65.0 |
| Kubernetes (vanilla) | 1.30, 1.31 | With Tekton Pipelines installed |

### Tekton/OpenShift Pipelines Compatibility

This release is built against **Tekton Pipeline v0.65.0**, which corresponds to:
- **OpenShift Pipelines 1.17** (OpenShift 4.17)
- **OpenShift Pipelines 1.18** (OpenShift 4.18)

> **Note:** OpenShift Pipelines 1.20 (Tekton v0.68.0) support is available in the `release-4.20` branch and will be the default in a future release. See [Issue #4](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/4) for upgrade timeline.

### Known Security Advisories

The following CVEs exist in transitive dependencies of Tekton Pipeline v0.65.0 and will be resolved when upgrading to OpenShift Pipelines 1.20:

| CVE | Severity | Package | Fixed In |
|-----|----------|---------|----------|
| CVE-2024-45338 | HIGH | golang.org/x/net v0.28.0 | v0.33.0+ |
| CVE-2025-22869 | MEDIUM | golang.org/x/crypto v0.26.0 | v0.31.0+ |
| CVE-2025-22866 | MEDIUM | golang.org/x/crypto v0.26.0 | v0.31.0+ |

These vulnerabilities are in the operator binary only and do not affect runtime behavior. The `release-4.20` branch resolves all CVEs.

## 📦 Installation

### Prerequisites
- OpenShift 4.18.x cluster
- OpenShift Pipelines 1.17 or 1.18 (for Tekton build strategy)
- Cluster admin access for CRD installation
- `oc` CLI tool installed

### Install the Operator

```bash
# Clone the repository
git clone https://github.com/tosin2013/jupyter-notebook-validator-operator.git
cd jupyter-notebook-validator-operator

# Checkout the release-4.18 branch
git checkout release-4.18

# Install CRDs
make install

# Deploy the operator
make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.18
```

### Verify Installation

```bash
# Check operator pod
oc get pods -n jupyter-notebook-validator-operator-system

# Check CRD
oc get crd notebookvalidationjobs.mlops.mlops.dev
```

## 🚀 Quick Start

### Example 1: Basic Notebook Validation

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: basic-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
    path: examples/basic-notebook.ipynb
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
    serviceAccountName: default
```

### Example 2: Validation with S2I Build

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: s2i-build-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
    path: examples/ml-notebook.ipynb
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
    serviceAccountName: default
    buildConfig:
      enabled: true
      strategy: s2i
      baseImage: quay.io/jupyter/minimal-notebook:latest
      requirementsFile: requirements.txt
      timeout: "15m"
```

### Example 3: Model-Aware Validation with KServe

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: kserve-model-validation
spec:
  notebook:
    git:
      url: https://github.com/example/ml-notebooks.git
      ref: main
    path: examples/kserve-model.ipynb
  podConfig:
    containerImage: quay.io/jupyter/tensorflow-notebook:latest
    serviceAccountName: default
  modelValidation:
    enabled: true
    platform: kserve
    phase: both
    targetModels:
      - name: iris-model
        namespace: ml-models
```

## 📊 Architecture

### Build Integration Workflow

```
┌─────────────────────────────────────────────────────────────┐
│ NotebookValidationJob Created                               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 1: Model Validation (if enabled)                      │
│ - Detect model serving platform                            │
│ - Validate model deployment                                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Build Integration (if enabled)                     │
│ - Detect available build strategies                        │
│ - Create build (S2I or Tekton)                            │
│ - Wait for build completion                               │
│ - Get built image reference                               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Create Validation Pod                              │
│ - Use built image (if build succeeded)                     │
│ - Fall back to spec image (if build failed)               │
│ - Execute notebook with Papermill                          │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Collect Results                                    │
│ - Parse execution results                                  │
│ - Compare with golden notebook (if specified)             │
│ - Update job status                                        │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 Configuration

### Build Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable custom image building |
| `strategy` | string | `"s2i"` | Build strategy (s2i, tekton) |
| `baseImage` | string | `"quay.io/jupyter/minimal-notebook:latest"` | Base image for build |
| `requirementsFile` | string | `"requirements.txt"` | Path to requirements file |
| `timeout` | string | `"15m"` | Build timeout duration |
| `autoGenerateRequirements` | bool | `false` | Auto-generate requirements.txt |
| `fallbackStrategy` | string | `"warn"` | Action when requirements missing |

### Model Validation Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable model validation |
| `platform` | string | `""` | Model serving platform (auto-detect if empty) |
| `phase` | string | `"both"` | Validation phase (clean, existing, both) |
| `targetModels` | array | `[]` | Models to validate against |
| `predictionValidation` | object | `nil` | Prediction validation config |

## 📈 Metrics

The operator exposes Prometheus metrics at `:8080/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `notebookvalidationjob_reconciliation_duration_seconds` | Histogram | Reconciliation duration |
| `notebookvalidationjob_validations_total` | Counter | Total validations by status |
| `notebookvalidationjob_git_clone_duration_seconds` | Histogram | Git clone duration |
| `notebookvalidationjob_active_pods` | Gauge | Active validation pods |
| `notebookvalidationjob_reconciliation_errors_total` | Counter | Reconciliation errors |
| `notebookvalidationjob_pod_creations_total` | Counter | Pod creations by status |

## 🔒 Security

### RBAC Permissions

The operator requires the following permissions:

- **Core Resources**: pods, secrets, configmaps, events
- **OpenShift Build**: buildconfigs, builds, imagestreams
- **Tekton**: pipelines, pipelineruns, taskruns
- **Model Serving**: inferenceservices, servingruntimes, deployments
- **Custom Resources**: notebookvalidationjobs

### Security Context

All validation pods run with:
- `runAsNonRoot: true`
- `allowPrivilegeEscalation: false`
- Dropped capabilities: `ALL`
- OpenShift-assigned UID from namespace range

## 🐛 Known Issues

1. **Build Timeout**: Very large builds may exceed the default 15-minute timeout. Increase the timeout in `buildConfig.timeout`.
2. **Papermill Installation**: Some base images may have permission issues installing Papermill. Use a custom image with Papermill pre-installed.
3. **Model Platform Detection**: Auto-detection may fail in clusters with multiple model serving platforms. Specify the platform explicitly.

## 🔄 Upgrade Path

### From v1.0.0-ocp4.18 to v1.0.0-ocp4.20

When OpenShift Pipelines 1.20 is available in your cluster:

1. **Check OpenShift Pipelines version**:
   ```bash
   oc get csv -n openshift-pipelines | grep pipelines
   ```

2. **Switch to release-4.20 branch**:
   ```bash
   git checkout release-4.20
   ```

3. **Redeploy the operator**:
   ```bash
   make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:v1.0.0-ocp4.20
   ```

The upgrade resolves all known CVEs and adds support for Tekton Pipeline v0.68.0 features.

## 📚 Documentation

- [Implementation Plan](docs/IMPLEMENTATION-PLAN.md)
- [Integration Testing Guide](docs/INTEGRATION_TESTING.md)
- [E2E Testing Guide](docs/E2E_TESTING.md)
- [ADR Documentation](docs/adrs/)

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📝 License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- OpenShift Build API team
- Tekton Pipelines community
- Jupyter Project
- Papermill maintainers

## 📞 Support

- **Issues**: https://github.com/tosin2013/jupyter-notebook-validator-operator/issues
- **Discussions**: https://github.com/tosin2013/jupyter-notebook-validator-operator/discussions

---

**Full Changelog**: https://github.com/tosin2013/jupyter-notebook-validator-operator/compare/v0.1.0...v1.0.0-ocp4.18

