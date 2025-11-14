#!/bin/bash
set -e

# test-local-kind.sh - Local Kind testing for Jupyter Notebook Validator Operator
#
# Purpose: Setup Kind cluster, deploy operator, run Tier 1 tests, cleanup
# Based on: ADR-032 (GitHub Actions CI), ADR-034 (Dual Testing Strategy)
# Tier: Tier 1 only (< 2 minutes total execution time)
#
# Features:
#   - Supports both Docker and Podman as container runtime
#   - Auto-installs Kind if not present
#   - Configures Kind to use Podman on RHEL/Fedora systems
#
# Usage:
#   ./scripts/test-local-kind.sh                    # Run full workflow
#   ./scripts/test-local-kind.sh --skip-cleanup     # Keep cluster for debugging
#   ./scripts/test-local-kind.sh --cleanup-only     # Only cleanup existing cluster
#   ./scripts/test-local-kind.sh --install-kind     # Only install Kind and exit

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-jupyter-validator-test}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.31.12}"
KIND_NODE_IMAGE="kindest/node:${KUBERNETES_VERSION}@sha256:0f5cc49c5e73c0c2bb6e2df56e7df189240d83cf94edfa30946482eb08ec57d2"
TEST_NAMESPACE="${TEST_NAMESPACE:-e2e-tests}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-jupyter-notebook-validator-operator}"
TEST_REPO_URL="${TEST_REPO_URL:-https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git}"
TEST_REPO_REF="${TEST_REPO_REF:-main}"
SKIP_CLEANUP=false
CLEANUP_ONLY=false
INSTALL_KIND_ONLY=false
PODMAN_ROOTFUL=false
KIND_VERSION="${KIND_VERSION:-v0.20.0}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-cleanup)
            SKIP_CLEANUP=true
            shift
            ;;
        --cleanup-only)
            CLEANUP_ONLY=true
            shift
            ;;
        --install-kind)
            INSTALL_KIND_ONLY=true
            shift
            ;;
        --podman-rootful)
            PODMAN_ROOTFUL=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-cleanup      Keep Kind cluster after tests (for debugging)"
            echo "  --cleanup-only      Only cleanup existing cluster and exit"
            echo "  --install-kind      Only install Kind and exit"
            echo "  --podman-rootful    Use Podman in rootful mode (requires sudo)"
            echo "  --help              Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  KIND_CLUSTER_NAME      Cluster name (default: jupyter-validator-test)"
            echo "  KIND_VERSION           Kind version (default: v0.20.0)"
            echo "  KUBERNETES_VERSION     Kubernetes version (default: v1.31.10)"
            echo "  TEST_NAMESPACE         Test namespace (default: e2e-tests)"
            echo "  OPERATOR_NAMESPACE     Operator namespace (default: jupyter-notebook-validator-operator)"
            echo "  TEST_REPO_URL          Test repository URL"
            echo "  TEST_REPO_REF          Test repository ref (default: main)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Install Kind if not present
install_kind() {
    log_info "Installing Kind ${KIND_VERSION}..."

    local kind_binary="/usr/local/bin/kind"
    local temp_dir=$(mktemp -d)

    # Download Kind binary
    log_info "Downloading Kind from GitHub..."
    curl -Lo "${temp_dir}/kind" "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"

    # Make executable
    chmod +x "${temp_dir}/kind"

    # Move to /usr/local/bin (requires sudo)
    if [ -w "/usr/local/bin" ]; then
        mv "${temp_dir}/kind" "$kind_binary"
    else
        log_info "Installing to /usr/local/bin requires sudo..."
        sudo mv "${temp_dir}/kind" "$kind_binary"
    fi

    # Verify installation
    if command -v kind &> /dev/null; then
        log_success "Kind installed successfully: $(kind version)"
    else
        log_error "Kind installation failed"
        exit 1
    fi

    # Cleanup
    rm -rf "$temp_dir"
}

# Detect container runtime (Docker or Podman)
detect_container_runtime() {
    if command -v docker &> /dev/null && docker info &> /dev/null 2>&1; then
        echo "docker"
    elif command -v podman &> /dev/null; then
        echo "podman"
    else
        echo "none"
    fi
}

# Configure Kind to use Podman
configure_kind_podman() {
    log_info "Configuring Kind to use Podman (rootless mode)..."

    # Enable Podman socket if not already running
    if ! systemctl --user is-active --quiet podman.socket; then
        log_info "Starting Podman socket..."
        systemctl --user start podman.socket
        systemctl --user enable podman.socket
    fi

    # Configure systemd for rootless Podman
    # Create systemd user directory if it doesn't exist
    mkdir -p ~/.config/systemd/user

    # Check if Delegate=yes is already set
    if ! systemctl --user show-environment | grep -q "DELEGATE=yes" 2>/dev/null; then
        log_info "Configuring systemd delegation for rootless Podman..."

        # Create or update user service override
        mkdir -p ~/.config/systemd/user/user@.service.d
        cat > ~/.config/systemd/user/user@.service.d/delegate.conf <<EOF
[Service]
Delegate=yes
EOF

        # Reload systemd
        systemctl --user daemon-reload

        log_info "Systemd delegation configured. You may need to log out and back in for changes to take full effect."
        log_warning "If cluster creation fails, try: sudo loginctl enable-linger $USER"
    fi

    # Enable lingering for user (allows user services to run without login)
    if ! loginctl show-user "$USER" | grep -q "Linger=yes"; then
        log_info "Enabling user lingering..."
        sudo loginctl enable-linger "$USER" 2>/dev/null || log_warning "Could not enable lingering (may require sudo)"
    fi

    # Set KIND_EXPERIMENTAL_PROVIDER environment variable
    export KIND_EXPERIMENTAL_PROVIDER=podman

    log_success "Kind configured to use Podman (rootless mode)"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing_tools=()

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi

    # Check container runtime (Docker or Podman)
    CONTAINER_RUNTIME=$(detect_container_runtime)
    if [ "$CONTAINER_RUNTIME" = "none" ]; then
        log_error "No container runtime found (Docker or Podman required)"
        log_info "Install instructions:"
        log_info "  Docker:  https://docs.docker.com/get-docker/"
        log_info "  Podman:  https://podman.io/getting-started/installation"
        exit 1
    else
        log_info "Container runtime detected: $CONTAINER_RUNTIME"

        # Configure Kind for Podman if needed
        if [ "$CONTAINER_RUNTIME" = "podman" ]; then
            configure_kind_podman
        fi
    fi

    # Check or install Kind
    if ! command -v kind &> /dev/null; then
        log_warning "Kind not found - installing..."
        install_kind
    else
        log_info "Kind version: $(kind version)"
    fi

    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Install instructions:"
        log_info "  kubectl: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi

    log_success "All prerequisites met"
}

# Cleanup existing cluster
cleanup_cluster() {
    log_info "Cleaning up existing Kind cluster: $CLUSTER_NAME"

    # Determine if we need sudo for Podman rootful mode
    local KIND_CMD="kind"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KIND_CMD="sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind"
    fi

    if $KIND_CMD get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        $KIND_CMD delete cluster --name "$CLUSTER_NAME"
        log_success "Cluster deleted: $CLUSTER_NAME"
    else
        log_info "No existing cluster found: $CLUSTER_NAME"
    fi
}

# Create Kind cluster
create_cluster() {
    log_info "Creating Kind cluster: $CLUSTER_NAME (Kubernetes $KUBERNETES_VERSION)"

    # Determine if we need sudo for Podman rootful mode
    local KIND_CMD="kind"
    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KIND_CMD="sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind"
        KUBECTL_CMD="sudo kubectl"
        log_info "Using Podman in rootful mode (with sudo)"
    fi

    # Create Kind cluster with specific Kubernetes version
    $KIND_CMD create cluster \
        --name "$CLUSTER_NAME" \
        --image "$KIND_NODE_IMAGE" \
        --wait 60s

    # Verify cluster is ready
    $KUBECTL_CMD cluster-info --context "kind-${CLUSTER_NAME}"

    # Verify Kubernetes version
    local k8s_version=$($KUBECTL_CMD version --short 2>/dev/null | grep "Server Version" | awk '{print $3}')
    log_info "Kubernetes version: $k8s_version"

    if [[ "$k8s_version" != "$KUBERNETES_VERSION" ]]; then
        log_warning "Kubernetes version mismatch: expected $KUBERNETES_VERSION, got $k8s_version"
    fi

    log_success "Kind cluster created successfully"
}

# Install cert-manager
install_cert_manager() {
    log_info "Installing cert-manager for webhooks..."

    # Determine kubectl command based on rootful mode
    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    $KUBECTL_CMD apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

    # Wait for cert-manager to be ready
    log_info "Waiting for cert-manager to be ready..."
    $KUBECTL_CMD wait --for=condition=Available --timeout=300s \
        -n cert-manager deployment/cert-manager \
        deployment/cert-manager-cainjector \
        deployment/cert-manager-webhook

    log_success "cert-manager installed successfully"
}

# Deploy operator
deploy_operator() {
    log_info "Deploying Jupyter Notebook Validator Operator..."

    cd "$PROJECT_ROOT"

    # Determine kubectl command based on rootful mode
    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    # Build and load operator image into Kind
    log_info "Building operator image..."
    log_info "Container runtime: $CONTAINER_RUNTIME"
    log_info "Podman rootful: $PODMAN_ROOTFUL"

    # Determine container tool based on runtime and rootful mode
    if [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
        if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
            # For rootful mode, we need to build with sudo
            log_info "Building with sudo podman..."
            sudo podman build -t jupyter-notebook-validator-operator:test .

            # Remove any existing tar file
            sudo rm -f /tmp/operator-image.tar

            # Save image to tar file
            log_info "Saving image to tar file..."
            sudo podman save -o /tmp/operator-image.tar localhost/jupyter-notebook-validator-operator:test
            sudo chmod 644 /tmp/operator-image.tar

            # Load image directly into Kind node using ctr
            log_info "Loading image into Kind node..."
            sudo podman exec -i "$CLUSTER_NAME-control-plane" ctr -n k8s.io images import /dev/stdin < /tmp/operator-image.tar

            # Cleanup tar file
            sudo rm -f /tmp/operator-image.tar
        else
            # For rootless mode, use make with CONTAINER_TOOL
            log_info "Building with podman (rootless)..."
            CONTAINER_TOOL=podman make docker-build IMG=jupyter-notebook-validator-operator:test

            # Load image into Kind (image is in user's Podman storage)
            log_info "Loading image into Kind cluster (rootless mode)..."
            kind load docker-image jupyter-notebook-validator-operator:test --name "$CLUSTER_NAME"
        fi
    else
        # Docker mode
        log_info "Building with docker..."
        CONTAINER_TOOL=docker make docker-build IMG=jupyter-notebook-validator-operator:test

        # Load image into Kind (image is in Docker storage)
        log_info "Loading image into Kind cluster (docker mode)..."
        kind load docker-image jupyter-notebook-validator-operator:test --name "$CLUSTER_NAME"
    fi
    
    # Deploy operator using kustomize
    log_info "Deploying operator manifests..."
    cd config/manager && kustomize edit set image controller=jupyter-notebook-validator-operator:test
    cd "$PROJECT_ROOT"

    $KUBECTL_CMD apply -k config/default

    # Wait for operator to be ready
    log_info "Waiting for operator to be ready..."
    $KUBECTL_CMD wait --for=condition=Available --timeout=300s \
        -n "$OPERATOR_NAMESPACE" deployment/jupyter-notebook-validator-operator-controller-manager

    log_success "Operator deployed successfully"
}

# Setup test environment
setup_test_environment() {
    log_info "Setting up test environment..."

    # Determine kubectl command based on rootful mode
    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    # Create test namespace
    $KUBECTL_CMD create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | $KUBECTL_CMD apply -f -

    # Create git credentials secret (if credentials are available)
    if [ -n "$GIT_USERNAME" ] && [ -n "$GIT_TOKEN" ]; then
        log_info "Creating git credentials secret..."
        $KUBECTL_CMD create secret generic git-https-credentials \
            --from-literal=username="$GIT_USERNAME" \
            --from-literal=password="$GIT_TOKEN" \
            -n "$TEST_NAMESPACE" \
            --dry-run=client -o yaml | $KUBECTL_CMD apply -f -
    else
        log_warning "GIT_USERNAME or GIT_TOKEN not set - skipping git credentials"
        log_info "Set these environment variables to test private repository access"
    fi

    # Create service account for validation pods
    $KUBECTL_CMD create serviceaccount jupyter-notebook-validator-runner \
        -n "$TEST_NAMESPACE" \
        --dry-run=client -o yaml | $KUBECTL_CMD apply -f -

    log_success "Test environment setup complete"
}

# Run Tier 1 tests
run_tier1_tests() {
    log_info "Running Tier 1 tests (simple notebooks, < 30s each)..."

    # Determine kubectl command based on rootful mode
    local KUBECTL_CMD="kubectl"
    if [[ "$PODMAN_ROOTFUL" == "true" ]]; then
        KUBECTL_CMD="sudo kubectl"
    fi

    local tier1_notebooks=(
        "notebooks/tier1-simple/01-hello-world.ipynb"
        "notebooks/tier1-simple/02-basic-math.ipynb"
        "notebooks/tier1-simple/03-data-validation.ipynb"
    )

    local failed_tests=()
    local passed_tests=()

    for notebook in "${tier1_notebooks[@]}"; do
        local notebook_name=$(basename "$notebook" .ipynb)
        local job_name="tier1-${notebook_name}"

        log_info "Testing: $notebook"

        # Create NotebookValidationJob
        cat <<EOF | $KUBECTL_CMD apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: ${job_name}
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_REF}"
    path: "${notebook}"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
    serviceAccountName: jupyter-notebook-validator-runner
  timeout: "5m"
EOF
        
        # Wait for job to complete (max 2 minutes)
        local timeout=120
        local elapsed=0
        local interval=5
        
        while [ $elapsed -lt $timeout ]; do
            local phase=$($KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

            case "$phase" in
                "Succeeded")
                    log_success "✅ Test passed: $notebook"
                    passed_tests+=("$notebook")
                    break
                    ;;
                "Failed")
                    log_error "❌ Test failed: $notebook"
                    $KUBECTL_CMD get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
                    failed_tests+=("$notebook")
                    break
                    ;;
                *)
                    sleep $interval
                    elapsed=$((elapsed + interval))
                    ;;
            esac
        done
        
        if [ $elapsed -ge $timeout ]; then
            log_error "❌ Test timeout: $notebook"
            failed_tests+=("$notebook")
        fi
        
        echo ""
    done
    
    # Summary
    echo "========================================"
    log_info "Tier 1 Test Summary"
    echo "========================================"
    log_info "Total tests: ${#tier1_notebooks[@]}"
    log_success "Passed: ${#passed_tests[@]}"
    log_error "Failed: ${#failed_tests[@]}"
    echo ""
    
    if [ ${#failed_tests[@]} -gt 0 ]; then
        log_error "Failed tests:"
        for test in "${failed_tests[@]}"; do
            echo "  - $test"
        done
        return 1
    fi
    
    log_success "All Tier 1 tests passed!"
    return 0
}

# Main execution
main() {
    echo "========================================"
    log_info "Kind Local Testing - Tier 1"
    log_info "Kubernetes Version: $KUBERNETES_VERSION"
    log_info "Cluster Name: $CLUSTER_NAME"
    echo "========================================"
    echo ""

    # Install Kind only mode
    if [ "$INSTALL_KIND_ONLY" = true ]; then
        if ! command -v kind &> /dev/null; then
            install_kind
        else
            log_info "Kind already installed: $(kind version)"
        fi
        exit 0
    fi

    # Cleanup only mode
    if [ "$CLEANUP_ONLY" = true ]; then
        cleanup_cluster
        exit 0
    fi

    # Check prerequisites
    check_prerequisites
    
    # Cleanup existing cluster
    cleanup_cluster
    
    # Create new cluster
    create_cluster
    
    # Install cert-manager
    install_cert_manager
    
    # Deploy operator
    deploy_operator
    
    # Setup test environment
    setup_test_environment
    
    # Run Tier 1 tests
    if run_tier1_tests; then
        log_success "🎉 All tests passed!"
        TEST_RESULT=0
    else
        log_error "❌ Some tests failed"
        TEST_RESULT=1
    fi
    
    # Cleanup (unless --skip-cleanup)
    if [ "$SKIP_CLEANUP" = true ]; then
        log_warning "Skipping cleanup (--skip-cleanup flag set)"
        log_info "To cleanup manually, run: kind delete cluster --name $CLUSTER_NAME"
    else
        cleanup_cluster
    fi
    
    exit $TEST_RESULT
}

# Run main function
main

