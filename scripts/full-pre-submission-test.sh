#!/bin/bash
set -e

# full-pre-submission-test.sh - Full Pre-Submission Test for Community Operators
#
# Purpose: Comprehensive local testing that simulates community-operators CI
# Based on: ADR-047 (Bundle Versioning) and ADR-048 (Pre-Submission Validation)
#
# This script:
#   1. Installs prerequisites (Kind, operator-sdk, opm) if missing
#   2. Validates all bundle versions
#   3. Runs K8s version matrix tests (1.29, 1.30, 1.31, 1.33)
#   4. Tests OLM installation for each bundle version
#   5. Generates a comprehensive test report
#
# Usage:
#   ./scripts/full-pre-submission-test.sh                    # Run full test suite
#   ./scripts/full-pre-submission-test.sh --skip-install     # Skip dependency installation
#   ./scripts/full-pre-submission-test.sh --skip-cleanup     # Keep clusters for debugging
#   ./scripts/full-pre-submission-test.sh --version 1.0.9    # Test specific version only
#   ./scripts/full-pre-submission-test.sh --quick            # Quick test (single K8s version)

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMMUNITY_OPERATORS_DIR="${COMMUNITY_OPERATORS_DIR:-/home/lab-user/community-operators}"
OPERATOR_NAME="jupyter-notebook-validator-operator"

# Tool versions (match community-operators CI)
KIND_VERSION="${KIND_VERSION:-v0.23.0}"
OPERATOR_SDK_VERSION="${OPERATOR_SDK_VERSION:-v1.37.0}"
OPM_VERSION="${OPM_VERSION:-v1.43.0}"
OLM_VERSION="${OLM_VERSION:-v0.28.0}"

# Bundle versions to test
BUNDLE_VERSIONS=("1.0.4")

# K8s versions for matrix testing (maps to OpenShift versions)
K8S_VERSIONS=("v1.29.0" "v1.30.0" "v1.31.0" "v1.33.0")

# Bundle registry
BUNDLE_REGISTRY="${BUNDLE_REGISTRY:-quay.io/takinosh}"
BUNDLE_IMAGE_NAME="jupyter-notebook-validator-operator-bundle"

# Options
SKIP_INSTALL=false
SKIP_CLEANUP=false
QUICK_MODE=false
TEST_VERSION=""

# Results tracking
declare -A TEST_RESULTS
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_section() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-install)
            SKIP_INSTALL=true
            shift
            ;;
        --skip-cleanup)
            SKIP_CLEANUP=true
            shift
            ;;
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --version)
            TEST_VERSION="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-install    Skip installing prerequisites"
            echo "  --skip-cleanup    Keep Kind clusters after tests (for debugging)"
            echo "  --quick           Quick mode: test only k8s v1.31.0"
            echo "  --version VER     Test specific version only (1.0.7, 1.0.8, or 1.0.9)"
            echo "  --help, -h        Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  KIND_VERSION           Kind version (default: $KIND_VERSION)"
            echo "  OPERATOR_SDK_VERSION   operator-sdk version (default: $OPERATOR_SDK_VERSION)"
            echo "  OPM_VERSION            opm version (default: $OPM_VERSION)"
            echo "  COMMUNITY_OPERATORS_DIR  Path to community-operators (default: $COMMUNITY_OPERATORS_DIR)"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Quick mode uses single K8s version
if [ "$QUICK_MODE" = true ]; then
    K8S_VERSIONS=("v1.31.0")
    log_info "Quick mode enabled - testing only k8s v1.31.0"
fi

# Single version mode
if [ -n "$TEST_VERSION" ]; then
    BUNDLE_VERSIONS=("$TEST_VERSION")
    log_info "Testing single version: $TEST_VERSION"
fi

record_result() {
    local test_name="$1"
    local result="$2"
    TEST_RESULTS["$test_name"]="$result"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$result" = "PASSED" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# ============================================================================
# PHASE 1: Install Prerequisites
# ============================================================================

install_prerequisites() {
    log_section "Phase 1: Installing Prerequisites"

    if [ "$SKIP_INSTALL" = true ]; then
        log_info "Skipping installation (--skip-install)"
        return 0
    fi

    # Install Kind
    if ! command -v kind &>/dev/null && ! sudo /usr/local/bin/kind version &>/dev/null; then
        log_info "Installing Kind $KIND_VERSION..."
        curl -Lo /tmp/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"
        chmod +x /tmp/kind
        sudo mv /tmp/kind /usr/local/bin/kind
        log_success "Kind $KIND_VERSION installed"
    else
        log_info "Kind already installed: $(kind version 2>/dev/null || sudo /usr/local/bin/kind version 2>/dev/null)"
    fi

    # Install operator-sdk
    if ! command -v operator-sdk &>/dev/null; then
        log_info "Installing operator-sdk $OPERATOR_SDK_VERSION..."
        curl -LO "https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64"
        chmod +x operator-sdk_linux_amd64
        sudo mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk
        log_success "operator-sdk $OPERATOR_SDK_VERSION installed"
    else
        log_info "operator-sdk already installed: $(operator-sdk version | head -1)"
    fi

    # Install opm
    if ! command -v opm &>/dev/null; then
        log_info "Installing opm $OPM_VERSION..."
        curl -LO "https://github.com/operator-framework/operator-registry/releases/download/${OPM_VERSION}/linux-amd64-opm"
        chmod +x linux-amd64-opm
        sudo mv linux-amd64-opm /usr/local/bin/opm
        log_success "opm $OPM_VERSION installed"
    else
        log_info "opm already installed: $(opm version | head -1)"
    fi

    log_success "All prerequisites installed"
}

# ============================================================================
# PHASE 2: Bundle Validation
# ============================================================================

validate_bundles() {
    log_section "Phase 2: Bundle Validation"

    local bundle_dir="${COMMUNITY_OPERATORS_DIR}/operators/${OPERATOR_NAME}"

    if [ ! -d "$bundle_dir" ]; then
        log_warning "Community operators directory not found: $bundle_dir"
        log_info "Will test bundles from release branches instead"
        validate_bundles_from_branches
        return
    fi

    for version in "${BUNDLE_VERSIONS[@]}"; do
        local version_dir="$bundle_dir/$version"
        
        if [ ! -d "$version_dir" ]; then
            log_warning "Bundle version $version not found in community-operators"
            record_result "bundle-validate-$version" "SKIPPED"
            continue
        fi

        log_info "Validating bundle $version..."

        # Basic validation
        if operator-sdk bundle validate "$version_dir" 2>&1; then
            log_success "Basic validation passed for $version"
        else
            log_error "Basic validation failed for $version"
            record_result "bundle-validate-$version" "FAILED"
            continue
        fi

        # Operator Framework suite
        if operator-sdk bundle validate "$version_dir" --select-optional suite=operatorframework 2>&1; then
            log_success "Operator Framework validation passed for $version"
        else
            log_error "Operator Framework validation failed for $version"
            record_result "bundle-validate-$version" "FAILED"
            continue
        fi

        # Check replaces field
        if grep -q "replaces:" "$version_dir/manifests"/*.clusterserviceversion.yaml 2>/dev/null; then
            local replaces=$(grep "replaces:" "$version_dir/manifests"/*.clusterserviceversion.yaml | head -1)
            log_success "replaces field found: $replaces"
        else
            log_warning "No replaces field found in $version"
        fi

        record_result "bundle-validate-$version" "PASSED"
    done
}

validate_bundles_from_branches() {
    cd "$PROJECT_ROOT"
    
    for version in "${BUNDLE_VERSIONS[@]}"; do
        local branch=""
        case $version in
            "1.0.7") branch="release-4.18" ;;
            "1.0.8") branch="release-4.19" ;;
            "1.0.9") branch="release-4.20" ;;
        esac

        log_info "Validating bundle $version from branch $branch..."
        
        git checkout "$branch" 2>/dev/null || {
            log_error "Failed to checkout $branch"
            record_result "bundle-validate-$version" "FAILED"
            continue
        }

        if operator-sdk bundle validate ./bundle 2>&1 && \
           operator-sdk bundle validate ./bundle --select-optional suite=operatorframework 2>&1; then
            log_success "Bundle $version validated successfully"
            record_result "bundle-validate-$version" "PASSED"
        else
            log_error "Bundle $version validation failed"
            record_result "bundle-validate-$version" "FAILED"
        fi
    done

    git checkout main 2>/dev/null || true
}

# ============================================================================
# PHASE 3: K8s Matrix Test
# ============================================================================

run_k8s_matrix_test() {
    log_section "Phase 3: Kubernetes Version Matrix Test"

    for k8s_version in "${K8S_VERSIONS[@]}"; do
        log_info "Testing with Kubernetes $k8s_version..."
        
        local cluster_name="pre-submit-${k8s_version//./}"
        
        # Create cluster
        if ! create_kind_cluster "$cluster_name" "$k8s_version"; then
            log_error "Failed to create cluster for $k8s_version"
            record_result "k8s-matrix-$k8s_version" "FAILED"
            continue
        fi

        # Install OLM
        if ! install_olm "$cluster_name"; then
            log_error "Failed to install OLM on $k8s_version"
            cleanup_cluster "$cluster_name"
            record_result "k8s-matrix-$k8s_version" "FAILED"
            continue
        fi

        # Test each bundle version
        local all_passed=true
        for version in "${BUNDLE_VERSIONS[@]}"; do
            if ! test_olm_installation "$cluster_name" "$version"; then
                log_error "OLM test failed for $version on k8s $k8s_version"
                all_passed=false
            fi
        done

        # Cleanup
        if [ "$SKIP_CLEANUP" != true ]; then
            cleanup_cluster "$cluster_name"
        else
            log_info "Keeping cluster $cluster_name for debugging"
        fi

        if [ "$all_passed" = true ]; then
            log_success "All tests passed on Kubernetes $k8s_version"
            record_result "k8s-matrix-$k8s_version" "PASSED"
        else
            record_result "k8s-matrix-$k8s_version" "FAILED"
        fi
    done
}

create_kind_cluster() {
    local cluster_name="$1"
    local k8s_version="$2"
    local node_image="kindest/node:$k8s_version"

    log_info "Creating Kind cluster: $cluster_name (k8s $k8s_version)"

    # Delete existing cluster if any
    sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind delete cluster --name "$cluster_name" 2>/dev/null || true

    # Create cluster
    if sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind create cluster \
        --name "$cluster_name" \
        --image "$node_image" \
        --wait 120s 2>&1; then
        log_success "Cluster $cluster_name created"
        
        # Copy kubeconfig for access
        sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind get kubeconfig --name "$cluster_name" > /tmp/kubeconfig-$cluster_name
        chmod 644 /tmp/kubeconfig-$cluster_name
        export KUBECONFIG=/tmp/kubeconfig-$cluster_name
        
        # Verify cluster access
        log_info "Verifying cluster access..."
        kubectl cluster-info --kubeconfig "$KUBECONFIG" 2>&1 || true
        
        return 0
    else
        log_error "Failed to create cluster $cluster_name"
        return 1
    fi
}

install_olm() {
    local cluster_name="$1"
    local kubeconfig="/tmp/kubeconfig-$cluster_name"
    
    log_info "Installing OLM $OLM_VERSION on $cluster_name..."
    export KUBECONFIG="$kubeconfig"

    # Apply OLM CRDs (use server-side apply to handle large annotations)
    if ! kubectl apply --server-side -f \
        "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${OLM_VERSION}/crds.yaml" 2>&1; then
        log_error "Failed to apply OLM CRDs"
        return 1
    fi

    # Apply OLM (use server-side apply)
    if ! kubectl apply --server-side -f \
        "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${OLM_VERSION}/olm.yaml" 2>&1; then
        log_error "Failed to apply OLM"
        return 1
    fi

    # Wait for OLM
    log_info "Waiting for OLM to be ready..."
    if kubectl wait --for=condition=Available \
        --timeout=300s -n olm deployment/catalog-operator deployment/olm-operator 2>&1; then
        log_success "OLM installed successfully"
        
        # Delete operatorhubio-catalog to avoid resolution conflicts
        # This catalog can interfere with bundle installation
        log_info "Removing operatorhubio-catalog to avoid resolution conflicts..."
        kubectl delete catalogsource operatorhubio-catalog -n olm 2>/dev/null || true
        
        return 0
    else
        log_error "OLM failed to become ready"
        kubectl get pods -n olm 2>&1 || true
        return 1
    fi
}

test_olm_installation() {
    local cluster_name="$1"
    local version="$2"
    local bundle_img="${BUNDLE_REGISTRY}/${BUNDLE_IMAGE_NAME}:${version}"
    local kubeconfig="/tmp/kubeconfig-$cluster_name"

    log_info "Testing OLM installation for version $version using operator-sdk run bundle..."

    # Set KUBECONFIG
    export KUBECONFIG="$kubeconfig"

    # Create namespace
    kubectl create namespace operators --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null

    # Use operator-sdk run bundle to test the bundle
    # This creates an ephemeral catalog and installs the operator
    log_info "Running: operator-sdk run bundle $bundle_img --namespace operators"
    
    if operator-sdk run bundle "$bundle_img" \
        --namespace operators \
        --timeout 5m 2>&1; then
        
        log_success "✅ Bundle $version installed successfully!"
        
        # Verify CSV status
        local phase=$(kubectl get csv -n operators -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "")
        log_info "CSV phase: $phase"
        
        # Check operator pod
        log_info "Operator pods:"
        kubectl get pods -n operators -l control-plane=controller-manager 2>/dev/null || true
        
        record_result "olm-install-$version-k8s-${cluster_name##*-}" "PASSED"
        
        # Cleanup for next test
        log_info "Cleaning up bundle..."
        operator-sdk cleanup "${OPERATOR_NAME}" --namespace operators 2>/dev/null || true
        
        return 0
    else
        log_error "❌ Bundle $version installation failed"
        
        # Debug info
        kubectl get csv -n operators -o yaml 2>/dev/null || true
        kubectl get pods -n operators 2>/dev/null || true
        kubectl get catalogsource -n operators 2>/dev/null || true
        
        record_result "olm-install-$version-k8s-${cluster_name##*-}" "FAILED"
        
        # Cleanup
        operator-sdk cleanup "${OPERATOR_NAME}" --namespace operators 2>/dev/null || true
        
        return 1
    fi
}

cleanup_cluster() {
    local cluster_name="$1"
    log_info "Cleaning up cluster $cluster_name..."
    sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind delete cluster --name "$cluster_name" 2>/dev/null || true
    rm -f /tmp/kubeconfig-$cluster_name 2>/dev/null || true
}

# ============================================================================
# PHASE 4: Generate Report
# ============================================================================

generate_report() {
    log_section "Test Report"

    echo "=============================================="
    echo "  PRE-SUBMISSION TEST RESULTS"
    echo "=============================================="
    echo ""
    echo "Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "Tool Versions:"
    echo "  - Kind: $KIND_VERSION"
    echo "  - operator-sdk: $OPERATOR_SDK_VERSION"
    echo "  - opm: $OPM_VERSION"
    echo "  - OLM: $OLM_VERSION"
    echo ""
    echo "Tested Versions: ${BUNDLE_VERSIONS[*]}"
    echo "K8s Matrix: ${K8S_VERSIONS[*]}"
    echo ""
    echo "----------------------------------------------"
    echo "  INDIVIDUAL TEST RESULTS"
    echo "----------------------------------------------"
    
    for test_name in "${!TEST_RESULTS[@]}"; do
        local result="${TEST_RESULTS[$test_name]}"
        if [ "$result" = "PASSED" ]; then
            echo -e "  ${GREEN}✅${NC} $test_name: $result"
        elif [ "$result" = "SKIPPED" ]; then
            echo -e "  ${YELLOW}⏭️${NC} $test_name: $result"
        else
            echo -e "  ${RED}❌${NC} $test_name: $result"
        fi
    done

    echo ""
    echo "----------------------------------------------"
    echo "  SUMMARY"
    echo "----------------------------------------------"
    echo "  Total Tests: $TOTAL_TESTS"
    echo -e "  ${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "  ${RED}Failed: $FAILED_TESTS${NC}"
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}=============================================="
        echo "  ✅ ALL TESTS PASSED - READY FOR PR"
        echo "=============================================="
        echo ""
        echo "Your bundles are ready for community-operators submission!"
        echo ""
        echo "Next steps:"
        echo "  1. Create PR to k8s-operatorhub/community-operators"
        echo "  2. PR Title: operators $OPERATOR_NAME (${BUNDLE_VERSIONS[*]})"
        echo "=============================================="
        echo -e "${NC}"
        return 0
    else
        echo -e "${RED}=============================================="
        echo "  ❌ SOME TESTS FAILED"
        echo "=============================================="
        echo ""
        echo "Please fix the failing tests before submitting PR."
        echo "Run with --skip-cleanup to debug Kind clusters."
        echo "=============================================="
        echo -e "${NC}"
        return 1
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    log_section "Full Pre-Submission Test for Community Operators"
    log_info "Simulating community-operators CI locally"
    log_info "Project: $OPERATOR_NAME"
    echo ""

    local start_time=$(date +%s)

    # Phase 1: Install prerequisites
    install_prerequisites

    # Phase 2: Validate bundles
    validate_bundles

    # Phase 3: K8s matrix test
    run_k8s_matrix_test

    # Phase 4: Generate report
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    log_info "Total test duration: ${duration}s ($(( duration / 60 ))m $(( duration % 60 ))s)"
    echo ""

    generate_report
}

# Run main
main
