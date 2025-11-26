# Development Guide

**Jupyter Notebook Validator Operator**  
**Last Updated:** 2025-11-10  
**Target Audience:** Developers, Contributors

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Setup](#local-development-setup)
3. [Local Testing with Kind](#local-testing-with-kind)
4. [Building and Running](#building-and-running)
5. [Testing Strategy](#testing-strategy)
6. [Troubleshooting](#troubleshooting)
7. [Contributing](#contributing)

---

## Prerequisites

### Required Tools

| Tool | Version | Purpose | Installation |
|------|---------|---------|--------------|
| **Go** | 1.21+ | Operator development | [golang.org](https://golang.org/doc/install) |
| **Operator SDK** | 1.32.0+ | Operator framework | [operatorframework.io](https://sdk.operatorframework.io/docs/installation/) |
| **kubectl** | 1.25+ | Kubernetes CLI | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| **Kind** | 0.20.0+ | Local Kubernetes clusters | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| **Docker** or **Podman** | 20.10+ / 4.0+ | Container runtime | [docker.com](https://docs.docker.com/get-docker/) / [podman.io](https://podman.io/getting-started/installation) |
| **kustomize** | 5.0+ | Kubernetes manifest management | [kustomize.io](https://kubectl.docs.kubernetes.io/installation/kustomize/) |

### Optional Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| **make** | Build automation | Usually pre-installed on Linux/macOS |
| **git** | Version control | [git-scm.com](https://git-scm.com/downloads) |
| **jq** | JSON processing | [stedolan.github.io/jq](https://stedolan.github.io/jq/download/) |

### Verify Installation

```bash
# Check Go version
go version  # Should be 1.21 or higher

# Check Operator SDK
operator-sdk version

# Check kubectl
kubectl version --client

# Check Kind
kind version

# Check container runtime (Docker or Podman)
docker --version && docker info  # Docker
# OR
podman --version && podman ps    # Podman

# Check kustomize
kustomize version
```

### Container Runtime: Docker vs Podman

The test script supports both Docker and Podman as container runtimes:

- **Docker**: Traditional container runtime, common on macOS and Windows
- **Podman**: Daemonless container runtime, common on RHEL/Fedora/OpenShift environments

**Podman Advantages:**
- ✅ No daemon required (more secure)
- ✅ Rootless containers by default
- ✅ Compatible with Docker CLI
- ✅ Native on RHEL/Fedora systems

The script automatically detects which runtime is available and configures Kind accordingly.

---

## Local Development Setup

### 1. Clone Repository

```bash
git clone https://github.com/tosin2013/jupyter-notebook-validator-operator.git
cd jupyter-notebook-validator-operator
```

### 2. Install Dependencies

```bash
# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### 3. Generate Code

```bash
# Generate CRD manifests, RBAC, and deepcopy code
make generate
make manifests
```

### 4. Build Operator

```bash
# Build operator binary
make build

# Build Docker image
make docker-build IMG=jupyter-notebook-validator-operator:dev
```

---

## Local Testing with Kind

**Based on:** ADR-032 (GitHub Actions CI), ADR-034 (Dual Testing Strategy)

### Features

- ✅ **Auto-installs Kind** if not present (no manual installation needed)
- ✅ **Supports Docker and Podman** (auto-detects container runtime)
- ✅ **Kubernetes v1.31.10** (matches OpenShift 4.18.21)
- ✅ **Fast execution** (< 2 minutes for Tier 1 tests)
- ✅ **Automatic cleanup** (or keep cluster for debugging)

### Quick Start

```bash
# Run full test workflow (< 2 minutes)
# Note: Kind will be auto-installed if not present
./scripts/test-local-kind.sh

# Install Kind only (without running tests)
./scripts/test-local-kind.sh --install-kind

# Keep cluster for debugging
./scripts/test-local-kind.sh --skip-cleanup

# Cleanup existing cluster
./scripts/test-local-kind.sh --cleanup-only
```

### What the Script Does

1. **Prerequisites Check**: Verifies Kind, kubectl, Docker are installed
2. **Cluster Creation**: Creates Kind cluster with Kubernetes v1.31.10
3. **cert-manager Installation**: Installs cert-manager for webhooks
4. **Operator Deployment**: Builds and deploys operator to Kind
5. **Test Execution**: Runs Tier 1 tests (simple notebooks)
6. **Cleanup**: Deletes Kind cluster (unless `--skip-cleanup`)

### Expected Output

```
========================================
[INFO] Kind Local Testing - Tier 1
[INFO] Kubernetes Version: v1.31.10
[INFO] Cluster Name: jupyter-validator-test
========================================

[INFO] Checking prerequisites...
[SUCCESS] All prerequisites met
[INFO] Creating Kind cluster: jupyter-validator-test (Kubernetes v1.31.10)
[SUCCESS] Kind cluster created successfully
[INFO] Installing cert-manager for webhooks...
[SUCCESS] cert-manager installed successfully
[INFO] Deploying Jupyter Notebook Validator Operator...
[SUCCESS] Operator deployed successfully
[INFO] Setting up test environment...
[SUCCESS] Test environment setup complete
[INFO] Running Tier 1 tests (simple notebooks, < 30s each)...
[INFO] Testing: notebooks/tier1-simple/01-hello-world.ipynb
[SUCCESS] ✅ Test passed: notebooks/tier1-simple/01-hello-world.ipynb
[INFO] Testing: notebooks/tier1-simple/02-basic-math.ipynb
[SUCCESS] ✅ Test passed: notebooks/tier1-simple/02-basic-math.ipynb
[INFO] Testing: notebooks/tier1-simple/03-data-validation.ipynb
[SUCCESS] ✅ Test passed: notebooks/tier1-simple/03-data-validation.ipynb

========================================
[INFO] Tier 1 Test Summary
========================================
[INFO] Total tests: 3
[SUCCESS] Passed: 3
[ERROR] Failed: 0

[SUCCESS] All Tier 1 tests passed!
[SUCCESS] 🎉 All tests passed!
[INFO] Cleaning up existing Kind cluster: jupyter-validator-test
[SUCCESS] Cluster deleted: jupyter-validator-test
```

### Execution Time

- **Total Time**: < 2 minutes
- **Cluster Setup**: ~30 seconds
- **Operator Deployment**: ~30 seconds
- **Test Execution**: ~30 seconds (3 notebooks × 10s each)
- **Cleanup**: ~10 seconds

### Environment Variables

```bash
# Customize cluster configuration
export KIND_CLUSTER_NAME="my-test-cluster"
export KUBERNETES_VERSION="v1.31.10"
export TEST_NAMESPACE="my-tests"
export OPERATOR_NAMESPACE="my-operator"

# Test private repository (optional)
export GIT_USERNAME="your-github-username"
export GIT_TOKEN="ghp_your_personal_access_token"
export TEST_REPO_URL="https://github.com/your-org/test-notebooks.git"
export TEST_REPO_REF="main"

# Run tests
./scripts/test-local-kind.sh
```

---

## Building and Running

### Build Operator Binary

```bash
# Build for current platform
make build

# Run locally (outside cluster)
make run
```

### Build Docker Image

```bash
# Build image
make docker-build IMG=jupyter-notebook-validator-operator:dev

# Push to registry (optional)
make docker-push IMG=quay.io/your-org/jupyter-notebook-validator-operator:dev
```

### Deploy to Kubernetes

```bash
# Deploy CRDs
make install

# Deploy operator
make deploy IMG=jupyter-notebook-validator-operator:dev

# Verify deployment
kubectl get deployment -n jupyter-notebook-validator-operator
kubectl get pods -n jupyter-notebook-validator-operator
```

### Undeploy

```bash
# Remove operator
make undeploy

# Remove CRDs
make uninstall
```

---

## Testing Strategy

### Test Tiers

| Tier | Environment | Execution Time | Infrastructure | Purpose |
|------|-------------|----------------|----------------|---------|
| **Tier 1** | Kind + OpenShift | < 30s | None | Fast feedback, basic validation |
| **Tier 2** | OpenShift only | 1-5 min | Build (S2I/Tekton) | Model training, dependencies |
| **Tier 3** | OpenShift only | 5-30 min | KServe/ESO/Models | Full integration, inference |

### Local Testing (Tier 1 Only)

```bash
# Run Tier 1 tests with Kind
./scripts/test-local-kind.sh

# Run unit tests
make test

# Run with coverage
go test -v -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

### OpenShift Testing (All Tiers)

```bash
# Run E2E tests on OpenShift
./scripts/run-e2e-tests.sh

# Run integration tests
./scripts/run-integration-tests.sh

# Run Tier 2 build tests
./scripts/tier2-build-tests.sh
```

---

## Troubleshooting

### Kind Cluster Issues

#### Problem: Kind cluster creation fails

```bash
# Check container runtime
docker info  # Docker
# OR
podman ps    # Podman

# Check Kind version
kind version  # Should be 0.20.0+

# Delete existing cluster
kind delete cluster --name jupyter-validator-test

# Try again
./scripts/test-local-kind.sh
```

#### Problem: Podman rootless mode requires Delegate=yes

If you see this error:
```
ERROR: failed to create cluster: running kind with rootless provider requires setting systemd property "Delegate=yes"
```

**Solution 1: Configure systemd delegation (recommended)**
```bash
# Run the configuration
./scripts/test-local-kind.sh --install-kind

# Log out and log back in (required for systemd changes)
# Then try again
./scripts/test-local-kind.sh
```

**Solution 2: Use Podman in rootful mode (works immediately)**
```bash
# Run Kind with Podman in rootful mode (recommended if delegation fails)
./scripts/test-local-kind.sh --podman-rootful

# Create Kind cluster with sudo
sudo KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster --name test
```

**Solution 3: Use OpenShift cluster instead**
```bash
# Skip Kind testing and use OpenShift directly
./scripts/run-e2e-tests.sh
```

#### Problem: Image not found in Kind

```bash
# Load image manually
kind load docker-image jupyter-notebook-validator-operator:test --name jupyter-validator-test

# Verify image is loaded
docker exec -it jupyter-validator-test-control-plane crictl images | grep jupyter
```

#### Problem: Operator pod not starting

```bash
# Check operator logs
kubectl logs -n jupyter-notebook-validator-operator deployment/jupyter-notebook-validator-operator-controller-manager

# Check pod events
kubectl describe pod -n jupyter-notebook-validator-operator -l control-plane=controller-manager

# Check image pull policy
kubectl get deployment -n jupyter-notebook-validator-operator -o yaml | grep imagePullPolicy
```

### Test Failures

#### Problem: Test timeout

```bash
# Increase timeout in test script
export TEST_TIMEOUT=300  # 5 minutes

# Check pod status
kubectl get pods -n e2e-tests

# Check pod logs
kubectl logs -n e2e-tests <pod-name>
```

#### Problem: Git authentication fails

```bash
# Verify credentials are set
echo $GIT_USERNAME
echo $GIT_TOKEN

# Check secret exists
kubectl get secret git-https-credentials -n e2e-tests

# Verify secret content
kubectl get secret git-https-credentials -n e2e-tests -o yaml
```

### Performance Issues

#### Problem: Tests are slow

```bash
# Check Docker resources
docker info | grep -A 5 "CPUs\|Total Memory"

# Increase Docker resources (Docker Desktop)
# Settings → Resources → Increase CPUs and Memory

# Use faster storage driver
# Docker Desktop → Settings → Docker Engine → "storage-driver": "overlay2"
```

---

## Contributing

### Development Workflow

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make Changes**
   ```bash
   # Edit code
   vim internal/controller/notebookvalidationjob_controller.go
   
   # Generate manifests
   make generate manifests
   ```

3. **Test Locally**
   ```bash
   # Run unit tests
   make test
   
   # Run Kind tests
   ./scripts/test-local-kind.sh
   ```

4. **Commit Changes**
   ```bash
   git add .
   git commit -m "feat: Add new feature"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feature/my-feature
   # Create PR on GitHub
   ```

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting: `go fmt ./...`
- Run linters: `golangci-lint run`
- Add comments for exported functions
- Write unit tests for new code

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: Add new feature
fix: Fix bug in controller
docs: Update documentation
test: Add unit tests
refactor: Refactor code
chore: Update dependencies
```

---

## Additional Resources

- [Operator SDK Documentation](https://sdk.operatorframework.io/docs/)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [ADR Documentation](docs/adrs/)
- [Integration Testing Guide](docs/INTEGRATION_TESTING.md)
- [E2E Testing Guide](docs/E2E_TESTING.md)

---

**Questions?** Open an issue on [GitHub](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues)

