# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Jupyter Notebook Validator Operator is a Kubernetes-native operator that automates Jupyter Notebook validation in MLOps workflows. Built with Operator SDK and Go, it orchestrates notebook execution in isolated pods, performs golden notebook comparisons, integrates with Git repositories, and validates against deployed ML models.

**Key Technologies**: Go 1.22+, Operator SDK v1.37.0, controller-runtime, OpenShift 4.18+, Kubernetes 1.31+

## Common Development Commands

### Building and Testing

```bash
# Build the operator binary
make build

# Run tests (unit + integration, excludes e2e)
make test

# Run E2E tests
make test-e2e

# Format code
make fmt

# Run static analysis
make vet

# Run linter
make lint

# Fix linting issues automatically
make lint-fix

# Generate CRD manifests and DeepCopy code
make manifests generate
```

### Running Locally

```bash
# Run operator locally (requires kubeconfig)
make run

# Install CRDs into cluster
make install

# Uninstall CRDs from cluster
make uninstall

# Deploy operator to cluster
make deploy IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:tag

# Undeploy operator from cluster
make undeploy
```

### Container Images

```bash
# Build container image
make docker-build IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:tag

# Push container image
make docker-push IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:tag

# Build and push multi-arch images
make docker-buildx IMG=quay.io/yourrepo/jupyter-notebook-validator-operator:tag
```

### Helm Chart Operations

```bash
# Sync CRDs to Helm chart
make helm-sync-crds

# Lint Helm chart
make helm-lint

# Render Helm templates (dry-run)
make helm-template

# Install Helm chart
make helm-install

# Upgrade Helm chart
make helm-upgrade

# Uninstall Helm chart
make helm-uninstall

# Package Helm chart for distribution
make helm-package
```

### Testing with Sample CRs

```bash
# Apply a basic validation job
kubectl apply -f config/samples/mlops_v1alpha1_notebookvalidationjob.yaml

# Check job status
kubectl get notebookvalidationjobs
kubectl describe notebookvalidationjob <name>

# Watch validation pod
kubectl get pods -w
kubectl logs <pod-name>
```

## Architecture Overview

### Core Components

1. **API Types** (`api/v1alpha1/`)
   - `NotebookValidationJob`: Main CRD that defines notebook validation specifications
   - Defines notebook source (Git), pod configuration, golden notebook for comparison, model validation settings, and credential injection
   - Webhook validation for CRD fields and sensible defaults

2. **Controller** (`internal/controller/`)
   - `notebookvalidationjob_controller.go`: Main reconciliation loop
   - Manages the validation workflow: Git clone → Pod creation → Execution → Result collection → Status updates
   - Orchestrates helper modules for specialized functionality

3. **Helper Modules** (`internal/controller/*_helper.go`)
   - `git_helper.go`: Clones Git repositories with HTTPS/SSH credentials, supports private repos
   - `papermill_helper.go`: Creates validation pods that execute notebooks with Papermill
   - `comparison_helper.go`: Compares notebook outputs with golden notebooks (cell-by-cell, metrics, tolerances)
   - `model_validation_helper.go`: Validates notebooks against deployed ML models (KServe, OpenShift AI, vLLM, etc.)
   - `pod_log_helper.go`: Collects and parses pod logs for debugging
   - `build_integration_helper.go`: Integrates with build systems (S2I, Tekton) for dependency resolution
   - `pod_failure_analyzer.go`: Analyzes pod failures with context-aware error messages

4. **Build Strategies** (`pkg/build/`)
   - Pluggable interface for different build backends
   - `s2i_strategy.go`: OpenShift Source-to-Image builds
   - `tekton_strategy.go`: Tekton Pipeline builds
   - `openshiftai.go`: OpenShift AI workbench integration
   - Auto-detects requirements.txt and builds custom images with dependencies

5. **Platform Detection** (`pkg/platform/`)
   - `detector.go`: Auto-detects ML model serving platforms (KServe, OpenShift AI, vLLM, TorchServe, TensorFlow Serving, Triton, Ray Serve, Seldon, BentoML)
   - Discovers model endpoints for validation

6. **Logging & Security** (`pkg/logging/`)
   - `sanitize.go`: Sanitizes logs to remove credentials and sensitive data
   - Structured logging with credential scrubbing

### Reconciliation Workflow

The controller follows this workflow when reconciling a NotebookValidationJob:

1. **Initialization**: Set status to `Pending`, initialize conditions
2. **Git Clone**: Clone notebook repository (handle credentials if provided)
3. **Dependency Detection**: Check for requirements.txt, trigger build if needed
4. **Pod Creation**: Create validation pod with Papermill, inject credentials from Secrets/ESO
5. **Execution Monitoring**: Watch pod status, collect logs
6. **Result Collection**: Parse executed notebook outputs
7. **Golden Comparison** (optional): Compare outputs with golden notebook using configurable tolerances
8. **Model Validation** (optional): Validate against deployed ML models
9. **Status Update**: Set phase to `Succeeded`/`Failed`, update conditions and results

**Status Phases**: `Pending` → `Running` → `Succeeded`/`Failed`

### Key Features

- **Git Integration**: Clones notebooks from Git (HTTPS, SSH, git@host:path formats), supports private repos via Secrets
- **Credential Management**: Injects credentials via Kubernetes Secrets, External Secrets Operator (ESO), or Vault
- **Golden Notebook Comparison**: Cell-by-cell output comparison with configurable numeric tolerances, text matching strategies
- **Model-Aware Validation**: Discovers and validates against 9+ model serving platforms
- **Build System Integration**: Auto-detects requirements.txt, builds custom images with S2I or Tekton
- **Observability**: Prometheus metrics, structured logging, OpenShift Console dashboards
- **Security**: RBAC, Pod Security Standards, credential sanitization, audit logging

## Project Structure

```
.
├── api/v1alpha1/              # CRD type definitions and webhook validation
├── internal/controller/       # Main controller and helper modules
├── pkg/
│   ├── build/                 # Build strategy implementations (S2I, Tekton)
│   ├── logging/               # Log sanitization and security
│   └── platform/              # ML platform detection
├── config/
│   ├── crd/                   # Generated CRD manifests
│   ├── rbac/                  # RBAC roles and bindings
│   ├── samples/               # Example NotebookValidationJob CRs
│   ├── webhook/               # Webhook configurations
│   ├── monitoring/            # Prometheus ServiceMonitor, dashboards
│   └── tekton/                # Tekton pipeline definitions
├── helm/                      # Helm chart for Kubernetes deployment
├── test/
│   ├── e2e/                   # End-to-end tests (Ginkgo)
│   └── utils/                 # Test utilities
├── test-notebooks/            # Sample notebooks for testing (separate repo, included as subdir)
└── docs/                      # Architecture docs, ADRs, guides
```

## Testing Strategy

The operator uses a three-tier testing approach:

### Tier 1: Simple (< 30s)
- Basic Python execution (hello world, assertions)
- Fast unit tests, no external dependencies
- Run on every PR

### Tier 2: Intermediate (1-5 min)
- Data analysis workflows (pandas, numpy)
- Small datasets, CSV files
- Run on every PR

### Tier 3: Complex (5-15 min)
- Model training, ML pipelines
- Large datasets, model artifacts
- Run nightly or on-demand

### Test Execution

```bash
# Unit and integration tests (fast)
make test

# E2E tests (requires cluster)
make test-e2e

# Run specific test
go test -v ./internal/controller -run TestReconcile

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Important Design Patterns

### Helper Module Pattern
The controller delegates specialized tasks to helper modules:
- Each helper is a focused set of functions in `*_helper.go`
- Helpers are stateless and accept context + client
- Testable in isolation with mock clients

### Build Strategy Pattern
Build strategies implement a common interface:
```go
type Strategy interface {
    Name() string
    Detect(ctx, client) (bool, error)
    CreateBuild(ctx, job) (*BuildInfo, error)
    GetBuildStatus(ctx, buildName) (*BuildInfo, error)
    // ... more methods
}
```
This allows pluggable build backends without changing controller logic.

### Status Conditions
Use typed conditions for observability:
- `Ready`, `GitCloned`, `ValidationStarted`, `ValidationComplete`, `EnvironmentReady`
- Each condition has `Type`, `Status`, `Reason`, `Message`
- Follow Kubernetes conventions for condition updates

### Credential Injection
Credentials are injected via environment variables:
- Git credentials: `GIT_USERNAME`, `GIT_PASSWORD`, `GIT_SSH_KEY`
- Notebook credentials: User-defined via `spec.podConfig.credentialMappings`
- Logs are sanitized to remove credential values

## Model Validation

Model validation discovers and validates against deployed ML models:

1. **Platform Detection**: Auto-detect KServe, OpenShift AI, vLLM, etc.
2. **Model Discovery**: Find models by name, namespace, or labels
3. **Endpoint Construction**: Build model inference endpoints
4. **Notebook Execution**: Inject model info as environment variables
5. **Validation**: Notebooks make inference requests, assert on responses

Example CR with model validation:
```yaml
spec:
  modelValidation:
    enabled: true
    models:
      - name: my-model
        platform: kserve
        namespace: models
```

## Git Credentials

Git credentials are provided via Kubernetes Secrets:

### HTTPS Authentication
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
type: Opaque
stringData:
  username: myuser
  password: mytoken
```

### SSH Authentication
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
type: Opaque
stringData:
  ssh-privatekey: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
    -----END OPENSSH PRIVATE KEY-----
```

Reference in NotebookValidationJob:
```yaml
spec:
  notebook:
    git:
      url: https://github.com/org/repo.git
      ref: main
      credentialsSecret: git-credentials
```

## OpenShift vs Kubernetes Differences

The operator supports both OpenShift and vanilla Kubernetes:

- **OpenShift**: Uses S2I builds via BuildConfig, integrates with ImageStreams
- **Kubernetes**: Uses Tekton Pipelines for builds, or pre-built images
- Platform detection is automatic via API availability checks

## Debugging Tips

### Watch operator logs
```bash
kubectl logs -n jupyter-notebook-validator-operator-system deployment/jupyter-notebook-validator-operator-controller-manager -f
```

### Check validation pod logs
```bash
# Find the pod
kubectl get pods -l job-name=<notebookvalidationjob-name>

# View logs
kubectl logs <pod-name>
```

### Inspect CR status
```bash
kubectl get notebookvalidationjob <name> -o yaml
```

### Check for build errors (OpenShift)
```bash
# Check BuildConfig
oc get buildconfig

# Check builds
oc get builds

# View build logs
oc logs build/<build-name>
```

### Common Issues

1. **Git clone failures**: Check credentials secret exists and has correct keys (`username`/`password` or `ssh-privatekey`)
2. **Pod stuck in Pending**: Check resource quotas, node capacity
3. **Build failures**: Check requirements.txt syntax, ensure base image supports dependencies
4. **Model validation failures**: Verify model exists, check namespace, ensure network policies allow access

## Documentation References

- **Architecture**: `docs/ARCHITECTURE_OVERVIEW.md`
- **Testing**: `docs/TESTING_GUIDE.md`
- **Credentials**: `docs/NOTEBOOK_CREDENTIALS_GUIDE.md`
- **Model Validation**: `docs/MODEL_DISCOVERY_GUIDE.md`
- **ADRs**: `docs/adrs/` (architectural decision records)

## Development Workflow

1. Make changes to API types → Run `make manifests generate`
2. Update controller logic → Run `make fmt vet lint`
3. Add/update tests → Run `make test`
4. Test locally → Run `make install run` (or `make deploy`)
5. Build image → Run `make docker-build docker-push IMG=...`
6. Deploy to cluster → Run `make deploy IMG=...`
7. Test with samples → Apply CRs from `config/samples/`

## Release Management

The project uses release branches:
- `main`: Development branch
- `release-X.Y`: Stable release branches (e.g., `release-4.18` for OpenShift 4.18)

When creating PRs or commits:
- Target appropriate branch based on OpenShift/Kubernetes version
- Update relevant docs in `docs/` if architecture changes
- Add ADRs for significant design decisions in `docs/adrs/`
