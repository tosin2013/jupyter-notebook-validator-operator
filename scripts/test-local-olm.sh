#!/bin/bash
set -e

# test-local-olm.sh - Local OLM testing for Jupyter Notebook Validator Operator
#
# Purpose: Test operator installation via OLM with all three bundle versions
# Tests: 1.0.7 (OpenShift 4.18+), 1.0.8 (OpenShift 4.19+), 1.0.9 (OpenShift 4.20+)
#
# Usage:
#   ./scripts/test-local-olm.sh                    # Test all versions
#   ./scripts/test-local-olm.sh --version 1.0.7    # Test specific version
#   ./scripts/test-local-olm.sh --skip-cleanup     # Keep cluster for debugging

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-jupyter-validator-olm-test}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.31.0}"
KIND_NODE_IMAGE="kindest/node:${KUBERNETES_VERSION}@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865"
TEST_NAMESPACE="${TEST_NAMESPACE:-olm-test}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-operators}"
SKIP_CLEANUP=false
TEST_VERSION=""
BUNDLE_REGISTRY="${BUNDLE_REGISTRY:-quay.io/takinosh}"
BUNDLE_NAME="jupyter-notebook-validator-operator-bundle"

# Bundle versions to test
declare -A BUNDLE_VERSIONS=(
    ["1.0.7"]="1.0.7"
    ["1.0.8"]="1.0.8"
    ["1.0.9"]="1.0.9"
)

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
        --version)
            TEST_VERSION="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-cleanup      Keep Kind cluster after tests (for debugging)"
            echo "  --version VERSION   Test specific version (1.0.7, 1.0.8, or 1.0.9)"
            echo "  --help              Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  KIND_CLUSTER_NAME      Cluster name (default: jupyter-validator-olm-test)"
            echo "  KUBERNETES_VERSION     Kubernetes version (default: v1.33.1)"
            echo "  TEST_NAMESPACE         Test namespace (default: olm-test)"
            echo "  OPERATOR_NAMESPACE     Operator namespace (default: operators)"
            echo "  BUNDLE_REGISTRY        Bundle registry (default: quay.io/takinosh)"
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi
    
    if ! command -v kind &> /dev/null; then
        missing_tools+=("kind")
    fi
    
    if ! command -v docker &> /dev/null && ! command -v podman &> /dev/null; then
        missing_tools+=("docker or podman")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    log_success "All prerequisites met"
}

# Cleanup existing cluster
cleanup_cluster() {
    log_info "Cleaning up existing Kind cluster: $CLUSTER_NAME"
    
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        kind delete cluster --name "$CLUSTER_NAME"
        log_success "Cluster deleted: $CLUSTER_NAME"
    else
        log_info "No existing cluster found: $CLUSTER_NAME"
    fi
}

# Create Kind cluster
create_cluster() {
    log_info "Creating Kind cluster: $CLUSTER_NAME (Kubernetes $KUBERNETES_VERSION)"
    
    kind create cluster \
        --name "$CLUSTER_NAME" \
        --image "$KIND_NODE_IMAGE" \
        --wait 60s
    
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
    log_success "Kind cluster created successfully"
}

# Install OLM
install_olm() {
    log_info "Installing Operator Lifecycle Manager (OLM)..."

    # Install OLM v0.30.0 (compatible with Kubernetes 1.33)
    kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.30.0/crds.yaml
    kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.30.0/olm.yaml

    # Wait for OLM to be ready
    log_info "Waiting for OLM to be ready..."
    kubectl wait --for=condition=Available --timeout=300s \
        -n olm deployment/catalog-operator \
        deployment/olm-operator

    log_success "OLM installed successfully"
}

# Create CatalogSource for testing
create_catalog_source() {
    local version=$1
    local catalog_name="jupyter-validator-catalog-${version}"

    log_info "Creating CatalogSource for version ${version}..."

    cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${catalog_name}
  namespace: olm
spec:
  sourceType: grpc
  image: ${BUNDLE_REGISTRY}/${BUNDLE_NAME}:${version}
  displayName: Jupyter Notebook Validator Operator (${version})
  publisher: Decision Crafters
  updateStrategy:
    registryPoll:
      interval: 10m
EOF

    # Wait for CatalogSource to be ready
    log_info "Waiting for CatalogSource to be ready..."
    local timeout=120
    local elapsed=0
    local interval=5

    while [ $elapsed -lt $timeout ]; do
        local state=$(kubectl get catalogsource "$catalog_name" -n olm -o jsonpath='{.status.connectionState.lastObservedState}' 2>/dev/null || echo "")

        if [ "$state" = "READY" ]; then
            log_success "CatalogSource is ready"
            return 0
        fi

        sleep $interval
        elapsed=$((elapsed + interval))
    done

    log_error "CatalogSource failed to become ready after ${timeout}s"
    kubectl get catalogsource "$catalog_name" -n olm -o yaml
    return 1
}

# Create OperatorGroup
create_operator_group() {
    log_info "Creating OperatorGroup in namespace ${OPERATOR_NAMESPACE}..."

    kubectl create namespace "$OPERATOR_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: jupyter-validator-operatorgroup
  namespace: ${OPERATOR_NAMESPACE}
spec:
  targetNamespaces:
    - ${OPERATOR_NAMESPACE}
    - ${TEST_NAMESPACE}
EOF

    log_success "OperatorGroup created"
}

# Create Subscription
create_subscription() {
    local version=$1
    local catalog_name="jupyter-validator-catalog-${version}"
    local subscription_name="jupyter-notebook-validator-operator-${version}"

    log_info "Creating Subscription for version ${version}..."

    cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${subscription_name}
  namespace: ${OPERATOR_NAMESPACE}
spec:
  channel: alpha
  name: jupyter-notebook-validator-operator
  source: ${catalog_name}
  sourceNamespace: olm
  installPlanApproval: Automatic
  startingCSV: jupyter-notebook-validator-operator.v${version}
EOF

    log_success "Subscription created"
}

# Wait for operator installation
wait_for_operator() {
    local version=$1
    local csv_name="jupyter-notebook-validator-operator.v${version}"

    log_info "Waiting for operator installation (CSV: ${csv_name})..."

    local timeout=300
    local elapsed=0
    local interval=10

    while [ $elapsed -lt $timeout ]; do
        local phase=$(kubectl get csv "$csv_name" -n "$OPERATOR_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")

        case "$phase" in
            "Succeeded")
                log_success "✅ Operator installed successfully (version ${version})"

                # Verify webhook certificate injection
                local webhook_annotation=$(kubectl get csv "$csv_name" -n "$OPERATOR_NAMESPACE" -o jsonpath='{.metadata.annotations.operatorframework\.io/inject-webhook-certs}' 2>/dev/null || echo "")
                if [ "$webhook_annotation" = "true" ]; then
                    log_success "  ✅ Webhook certificate injection annotation present"
                else
                    log_warning "  ⚠️  Webhook certificate injection annotation missing"
                fi

                # Check operator pod
                log_info "Checking operator pod..."
                kubectl get pods -n "$OPERATOR_NAMESPACE" -l control-plane=controller-manager

                return 0
                ;;
            "Failed")
                log_error "❌ Operator installation failed"
                kubectl get csv "$csv_name" -n "$OPERATOR_NAMESPACE" -o yaml
                return 1
                ;;
            "")
                log_info "  Waiting for CSV to be created... (${elapsed}s/${timeout}s)"
                ;;
            *)
                log_info "  CSV phase: ${phase} (${elapsed}s/${timeout}s)"
                ;;
        esac

        sleep $interval
        elapsed=$((elapsed + interval))
    done

    log_error "❌ Operator installation timeout after ${timeout}s"
    kubectl get csv -n "$OPERATOR_NAMESPACE"
    kubectl get subscription -n "$OPERATOR_NAMESPACE"
    kubectl get installplan -n "$OPERATOR_NAMESPACE"
    return 1
}

# Run simple validation test
run_validation_test() {
    local version=$1

    log_info "Running validation test for version ${version}..."

    # Create test namespace
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # Create service account
    kubectl create serviceaccount jupyter-notebook-validator-runner \
        -n "$TEST_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Create test NotebookValidationJob
    local job_name="olm-test-${version}"

    cat <<EOF | kubectl apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: ${job_name}
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
    serviceAccountName: jupyter-notebook-validator-runner
  timeout: "5m"
EOF

    # Wait for job to complete
    local timeout=180
    local elapsed=0
    local interval=5

    while [ $elapsed -lt $timeout ]; do
        local phase=$(kubectl get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        case "$phase" in
            "Succeeded")
                log_success "✅ Validation test passed for version ${version}"
                return 0
                ;;
            "Failed")
                log_error "❌ Validation test failed for version ${version}"
                kubectl get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
                return 1
                ;;
            *)
                sleep $interval
                elapsed=$((elapsed + interval))
                ;;
        esac
    done

    log_error "❌ Validation test timeout for version ${version}"
    kubectl get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
    return 1
}

# Test single version
test_version() {
    local version=$1

    echo ""
    echo "========================================"
    log_info "Testing Version ${version}"
    echo "========================================"
    echo ""

    # Create CatalogSource
    if ! create_catalog_source "$version"; then
        log_error "Failed to create CatalogSource for version ${version}"
        return 1
    fi

    # Create Subscription
    if ! create_subscription "$version"; then
        log_error "Failed to create Subscription for version ${version}"
        return 1
    fi

    # Wait for operator installation
    if ! wait_for_operator "$version"; then
        log_error "Failed to install operator version ${version}"
        return 1
    fi

    # Run validation test
    if ! run_validation_test "$version"; then
        log_error "Validation test failed for version ${version}"
        return 1
    fi

    log_success "🎉 Version ${version} test passed!"
    return 0
}

# Cleanup operator installation
cleanup_operator() {
    local version=$1

    log_info "Cleaning up operator version ${version}..."

    # Delete Subscription
    kubectl delete subscription "jupyter-notebook-validator-operator-${version}" -n "$OPERATOR_NAMESPACE" --ignore-not-found=true

    # Delete CSV
    kubectl delete csv "jupyter-notebook-validator-operator.v${version}" -n "$OPERATOR_NAMESPACE" --ignore-not-found=true

    # Delete CatalogSource
    kubectl delete catalogsource "jupyter-validator-catalog-${version}" -n olm --ignore-not-found=true

    # Delete test resources
    kubectl delete notebookvalidationjob --all -n "$TEST_NAMESPACE" --ignore-not-found=true

    # Wait for cleanup
    sleep 10

    log_success "Cleanup complete for version ${version}"
}

# Main execution
main() {
    echo "========================================"
    log_info "OLM Local Testing - Jupyter Notebook Validator Operator"
    log_info "Kubernetes Version: $KUBERNETES_VERSION"
    log_info "Cluster Name: $CLUSTER_NAME"
    echo "========================================"
    echo ""

    # Check prerequisites
    check_prerequisites

    # Cleanup existing cluster
    cleanup_cluster

    # Create new cluster
    create_cluster

    # Install OLM
    install_olm

    # Create OperatorGroup
    create_operator_group

    # Determine which versions to test
    local versions_to_test=()
    if [ -n "$TEST_VERSION" ]; then
        if [ -z "${BUNDLE_VERSIONS[$TEST_VERSION]}" ]; then
            log_error "Invalid version: $TEST_VERSION"
            log_info "Valid versions: ${!BUNDLE_VERSIONS[@]}"
            exit 1
        fi
        versions_to_test=("$TEST_VERSION")
    else
        versions_to_test=("1.0.7" "1.0.8" "1.0.9")
    fi

    # Test each version
    local failed_versions=()
    local passed_versions=()

    for version in "${versions_to_test[@]}"; do
        if test_version "$version"; then
            passed_versions+=("$version")
        else
            failed_versions+=("$version")
        fi

        # Cleanup between tests (unless it's the last one and --skip-cleanup is set)
        if [ "$version" != "${versions_to_test[-1]}" ] || [ "$SKIP_CLEANUP" = false ]; then
            cleanup_operator "$version"
        fi
    done

    # Summary
    echo ""
    echo "========================================"
    log_info "Test Summary"
    echo "========================================"
    log_info "Total versions tested: ${#versions_to_test[@]}"
    log_success "Passed: ${#passed_versions[@]}"
    log_error "Failed: ${#failed_versions[@]}"
    echo ""

    if [ ${#failed_versions[@]} -gt 0 ]; then
        log_error "Failed versions:"
        for version in "${failed_versions[@]}"; do
            echo "  - $version"
        done
        TEST_RESULT=1
    else
        log_success "🎉🎉 ALL VERSIONS PASSED! 🎉🎉"
        TEST_RESULT=0
    fi

    # Cleanup cluster (unless --skip-cleanup)
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


