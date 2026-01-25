#!/bin/bash
set -e

# test-k8s-matrix.sh - Kubernetes Version Matrix Testing
#
# Purpose: Test operator bundles against multiple Kubernetes versions sequentially
# Based on: ADR-048 (Pre-Submission Validation Test Rig Strategy)
#
# This script simulates Community Operators CI by testing against the Kubernetes
# version matrix that maps to OpenShift 4.18, 4.19, and 4.20.
#
# Usage:
#   ./scripts/test-k8s-matrix.sh [BUNDLE_DIR] [VERSION]
#   ./scripts/test-k8s-matrix.sh ./bundle 1.0.7
#
# Environment Variables:
#   BUNDLE_DIR - Path to bundle directory (default: ./bundle)
#   VERSION - Bundle version (default: latest in bundle)
#   SKIP_CLEANUP - Set to true to keep clusters for debugging

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUNDLE_DIR="${1:-${BUNDLE_DIR:-${PROJECT_ROOT}/bundle}}"
VERSION="${2:-${VERSION:-}}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

# Kubernetes versions to test (mapping to OpenShift versions)
# These match what Community Operators CI tests against
declare -a K8S_VERSIONS=(
    "v1.29.0"  # Early compatibility check
    "v1.30.0"  # Early compatibility check
    "v1.31.0"  # OpenShift 4.18
    "v1.33.0"  # OpenShift 4.20
)

# Kind node image mapping
# Note: Using version tags - Kind will resolve to the correct image
# For production use, consider pinning to specific SHA256 digests for reproducibility
declare -A K8S_IMAGES=(
    ["v1.29.0"]="kindest/node:v1.29.0"
    ["v1.30.0"]="kindest/node:v1.30.0"
    ["v1.31.0"]="kindest/node:v1.31.0"
    ["v1.33.0"]="kindest/node:v1.33.0"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Detect operating system
detect_os() {
    if [ -f /etc/redhat-release ]; then
        echo "rhel"
    elif [ -f /etc/centos-release ]; then
        echo "centos"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    else
        echo "unknown"
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing_tools=()

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi

    # Check container runtime
    CONTAINER_RUNTIME=$(detect_container_runtime)
    if [ "$CONTAINER_RUNTIME" = "none" ]; then
        log_error "No container runtime found (Docker or Podman required)"
        exit 1
    else
        log_info "Container runtime detected: $CONTAINER_RUNTIME"
    fi

    # Check Kind
    if ! command -v kind &> /dev/null; then
        log_error "Kind not found - please install Kind first"
        log_info "Install instructions: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi

    # Check operator-sdk
    if ! command -v operator-sdk &> /dev/null; then
        log_error "operator-sdk not found - please install operator-sdk first"
        log_info "Install instructions: https://sdk.operatorframework.io/docs/installation/"
        exit 1
    fi

    # Check bundle directory
    if [ ! -d "$BUNDLE_DIR" ]; then
        log_error "Bundle directory not found: $BUNDLE_DIR"
        exit 1
    fi

    log_success "All prerequisites met"
}

# Cleanup existing cluster
cleanup_cluster() {
    local cluster_name=$1
    log_info "Cleaning up Kind cluster: $cluster_name"

    if kind get clusters 2>/dev/null | grep -q "^${cluster_name}$"; then
        kind delete cluster --name "$cluster_name"
        log_success "Cluster deleted: $cluster_name"
    else
        log_info "No existing cluster found: $cluster_name"
    fi
}

# Test bundle against a specific Kubernetes version
test_bundle_version() {
    local k8s_version=$1
    local cluster_name="operator-test-${k8s_version//./-}"
    local k8s_image="${K8S_IMAGES[$k8s_version]}"

    echo ""
    echo "-------------------------------------------"
    log_info "Testing against Kubernetes $k8s_version..."
    echo "-------------------------------------------"

    # Cleanup any existing cluster with this name
    cleanup_cluster "$cluster_name"

    # Create Kind cluster with specific Kubernetes version
    log_info "Creating Kind cluster: $cluster_name"
    if [ -n "$k8s_image" ]; then
        kind create cluster --image "$k8s_image" --name "$cluster_name" --wait 60s
    else
        # Fallback: use version tag directly
        kind create cluster --image "kindest/node:$k8s_version" --name "$cluster_name" --wait 60s
    fi

    # Verify cluster is ready
    kubectl cluster-info --context "kind-${cluster_name}" > /dev/null 2>&1

    # Verify Kubernetes version
    local actual_version=$(kubectl version --short 2>/dev/null | grep "Server Version" | awk '{print $3}' || echo "unknown")
    log_info "Kubernetes version: $actual_version"

    # Install OLM (Operator Lifecycle Manager)
    log_info "Installing OLM..."
    operator-sdk olm install --version v0.28.0 --timeout 5m

    # Wait for OLM to be ready
    log_info "Waiting for OLM to be ready..."
    kubectl wait --for=condition=Available --timeout=300s \
        -n olm deployment/olm-operator \
        deployment/catalog-operator \
        deployment/packageserver || {
        log_warning "OLM may not be fully ready, continuing anyway..."
    }

    # Validate bundle with operator-sdk
    log_info "Validating bundle against Kubernetes $k8s_version..."
    cd "$BUNDLE_DIR"

    # Extract k8s version number for validation (e.g., v1.31.0 -> 1.31)
    local k8s_version_num=$(echo "$k8s_version" | sed 's/v//' | cut -d. -f1,2)

    if operator-sdk bundle validate . \
        --select-optional suite=operatorframework \
        --optional-values="k8s-version=${k8s_version_num}" \
        --kubeconfig "$(kind get kubeconfig-path --name "$cluster_name")"; then
        log_success "✅ Bundle validation passed for Kubernetes $k8s_version"
        cd "$PROJECT_ROOT"
        
        # Cleanup cluster unless SKIP_CLEANUP is set
        if [ "$SKIP_CLEANUP" != "true" ]; then
            cleanup_cluster "$cluster_name"
        else
            log_warning "Skipping cleanup (SKIP_CLEANUP=true) - cluster $cluster_name still exists"
        fi
        
        return 0
    else
        log_error "❌ Bundle validation failed for Kubernetes $k8s_version"
        cd "$PROJECT_ROOT"
        
        # Keep cluster for debugging unless explicitly cleaning up
        if [ "$SKIP_CLEANUP" != "true" ]; then
            log_warning "Keeping cluster $cluster_name for debugging"
            log_info "To cleanup manually: kind delete cluster --name $cluster_name"
        fi
        
        return 1
    fi
}

# Main execution
main() {
    echo "========================================"
    log_info "Kubernetes Version Matrix Testing"
    log_info "Based on: ADR-048 (Pre-Submission Validation Test Rig)"
    echo "========================================"
    echo ""

    # Detect OS
    local os_type=$(detect_os)
    log_info "Operating system: $os_type"

    # Check prerequisites
    check_prerequisites

    # Display configuration
    log_info "Bundle directory: $BUNDLE_DIR"
    log_info "Kubernetes versions to test: ${K8S_VERSIONS[*]}"
    log_info "Container runtime: $CONTAINER_RUNTIME"
    echo ""

    # Track results
    local failed_versions=()
    local passed_versions=()

    # Test each Kubernetes version sequentially
    for version in "${K8S_VERSIONS[@]}"; do
        if test_bundle_version "$version"; then
            passed_versions+=("$version")
        else
            failed_versions+=("$version")
        fi
    done

    # Summary
    echo ""
    echo "========================================"
    log_info "Test Summary"
    echo "========================================"
    log_info "Total versions tested: ${#K8S_VERSIONS[@]}"
    log_success "Passed: ${#passed_versions[@]}"
    
    if [ ${#failed_versions[@]} -gt 0 ]; then
        log_error "Failed: ${#failed_versions[@]}"
        log_error "Failed versions:"
        for v in "${failed_versions[@]}"; do
            echo "  - $v"
        done
        echo ""
        log_error "❌ Some Kubernetes versions failed validation"
        log_info "Review the errors above and fix bundle compatibility issues"
        exit 1
    else
        log_success "✅ All Kubernetes versions passed validation!"
        echo ""
        log_info "Your bundle is ready for Community Operators submission"
        exit 0
    fi
}

# Run main function
main
