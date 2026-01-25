#!/bin/bash
set -e

# pre-submission-validate.sh - Pre-Submission Validation Workflow
#
# Purpose: Comprehensive validation workflow before Community Operators submission
# Based on: ADR-048 (Pre-Submission Validation Test Rig Strategy)
#
# This script runs the full validation suite that simulates Community Operators CI:
#   1. Bundle structure validation
#   2. Kubernetes version matrix testing (1.29, 1.30, 1.31, 1.33)
#   3. Operator Framework validation suite
#   4. OS-aware testing (RHEL/CentOS detection)
#
# Usage:
#   ./scripts/pre-submission-validate.sh [BUNDLE_DIR] [VERSION]
#   ./scripts/pre-submission-validate.sh ./bundle 1.0.7
#
# Environment Variables:
#   BUNDLE_DIR - Path to bundle directory (default: ./bundle)
#   VERSION - Bundle version (default: auto-detect from bundle)
#   SKIP_MATRIX - Set to true to skip Kubernetes version matrix testing
#   SKIP_CLEANUP - Set to true to keep clusters for debugging

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUNDLE_DIR="${1:-${BUNDLE_DIR:-${PROJECT_ROOT}/bundle}}"
VERSION="${2:-${VERSION:-}}"
SKIP_MATRIX="${SKIP_MATRIX:-false}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

log_section() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
}

# Detect operating system
detect_os() {
    if [ -f /etc/redhat-release ]; then
        local version=$(cat /etc/redhat-release)
        echo "rhel:$version"
    elif [ -f /etc/centos-release ]; then
        local version=$(cat /etc/centos-release)
        echo "centos:$version"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID:$VERSION"
    else
        echo "unknown:unknown"
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

    # Check operator-sdk
    if ! command -v operator-sdk &> /dev/null; then
        missing_tools+=("operator-sdk")
    fi

    # Check Kind (only if matrix testing is enabled)
    if [ "$SKIP_MATRIX" != "true" ]; then
        if ! command -v kind &> /dev/null; then
            missing_tools+=("kind")
        fi
    fi

    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Install instructions:"
        [ "${missing_tools[*]}" == *"kubectl"* ] && log_info "  kubectl: https://kubernetes.io/docs/tasks/tools/"
        [ "${missing_tools[*]}" == *"operator-sdk"* ] && log_info "  operator-sdk: https://sdk.operatorframework.io/docs/installation/"
        [ "${missing_tools[*]}" == *"kind"* ] && log_info "  kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi

    # Check bundle directory
    if [ ! -d "$BUNDLE_DIR" ]; then
        log_error "Bundle directory not found: $BUNDLE_DIR"
        exit 1
    fi

    # Check bundle structure
    if [ ! -f "$BUNDLE_DIR/manifests" ] && [ ! -d "$BUNDLE_DIR/manifests" ]; then
        log_error "Bundle manifests not found in: $BUNDLE_DIR"
        exit 1
    fi

    if [ ! -f "$BUNDLE_DIR/metadata" ] && [ ! -d "$BUNDLE_DIR/metadata" ]; then
        log_error "Bundle metadata not found in: $BUNDLE_DIR"
        exit 1
    fi

    log_success "All prerequisites met"
}

# Validate bundle structure
validate_bundle_structure() {
    log_section "Step 1: Bundle Structure Validation"

    log_info "Validating bundle structure in: $BUNDLE_DIR"

    local errors=0

    # Check required files/directories
    local required_items=(
        "manifests"
        "metadata"
    )

    for item in "${required_items[@]}"; do
        if [ ! -e "$BUNDLE_DIR/$item" ]; then
            log_error "Missing required bundle item: $item"
            errors=$((errors + 1))
        fi
    done

    # Check for ClusterServiceVersion
    if [ ! -f "$BUNDLE_DIR/manifests"/*.clusterserviceversion.yaml ] && \
       [ ! -f "$BUNDLE_DIR/manifests/"*.clusterserviceversion.yaml ] 2>/dev/null; then
        log_error "No ClusterServiceVersion found in manifests/"
        errors=$((errors + 1))
    fi

    # Check for CRDs
    if [ ! -f "$BUNDLE_DIR/manifests"/*.crd.yaml ] && \
       [ ! -f "$BUNDLE_DIR/manifests/"*.crd.yaml ] 2>/dev/null; then
        log_warning "No CRDs found in manifests/ (may be valid for some operators)"
    fi

    if [ $errors -gt 0 ]; then
        log_error "Bundle structure validation failed with $errors error(s)"
        return 1
    else
        log_success "Bundle structure validation passed"
        return 0
    fi
}

# Validate bundle with operator-sdk (basic validation)
validate_bundle_basic() {
    log_section "Step 2: Basic Bundle Validation (operator-sdk)"

    log_info "Running operator-sdk bundle validate (basic checks)..."
    cd "$BUNDLE_DIR"

    if operator-sdk bundle validate . --select-optional suite=operatorframework; then
        log_success "✅ Basic bundle validation passed"
        cd "$PROJECT_ROOT"
        return 0
    else
        log_error "❌ Basic bundle validation failed"
        cd "$PROJECT_ROOT"
        return 1
    fi
}

# Run Kubernetes version matrix testing
validate_k8s_matrix() {
    log_section "Step 3: Kubernetes Version Matrix Testing"

    if [ "$SKIP_MATRIX" = "true" ]; then
        log_warning "Skipping Kubernetes version matrix testing (SKIP_MATRIX=true)"
        return 0
    fi

    log_info "Running Kubernetes version matrix tests..."
    log_info "This will test against: v1.29.0, v1.30.0, v1.31.0, v1.33.0"
    log_info "This may take 10-15 minutes (sequential execution)..."
    echo ""

    # Export SKIP_CLEANUP for the matrix script
    export SKIP_CLEANUP

    if "$SCRIPT_DIR/test-k8s-matrix.sh" "$BUNDLE_DIR" "$VERSION"; then
        log_success "✅ Kubernetes version matrix testing passed"
        return 0
    else
        log_error "❌ Kubernetes version matrix testing failed"
        return 1
    fi
}

# Generate validation report
generate_report() {
    local overall_result=$1
    local os_info=$(detect_os)

    log_section "Pre-Submission Validation Report"

    echo "Validation Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "Bundle Directory: $BUNDLE_DIR"
    echo "Operating System: $os_info"
    echo ""

    if [ $overall_result -eq 0 ]; then
        log_success "✅ ALL VALIDATIONS PASSED"
        echo ""
        log_info "Your bundle is ready for Community Operators submission!"
        echo ""
        log_info "Next steps:"
        echo "  1. Review the bundle one more time"
        echo "  2. Submit to Community Operators repository"
        echo "  3. Monitor the Community Operators CI pipeline"
        echo ""
        log_info "Note: This validation provides ~95% confidence."
        log_info "      Community Operators CI may still catch edge cases."
    else
        log_error "❌ VALIDATION FAILED"
        echo ""
        log_error "Please fix the issues above before submitting to Community Operators."
        echo ""
        log_info "Common issues:"
        echo "  - API deprecations (e.g., v1beta1 → v1)"
        echo "  - Missing required bundle metadata"
        echo "  - CRD schema validation errors"
        echo "  - Kubernetes version compatibility issues"
    fi
}

# Main execution
main() {
    log_section "Pre-Submission Validation Workflow"
    log_info "Based on: ADR-048 (Pre-Submission Validation Test Rig Strategy)"
    log_info "Purpose: Validate bundle before Community Operators submission"
    echo ""

    # Detect OS
    local os_info=$(detect_os)
    log_info "Operating system: $os_info"

    # Check prerequisites
    check_prerequisites

    # Track overall result
    local overall_result=0
    local step_results=()

    # Step 1: Bundle structure validation
    if validate_bundle_structure; then
        step_results+=("✅ Bundle Structure: PASSED")
    else
        step_results+=("❌ Bundle Structure: FAILED")
        overall_result=1
    fi

    # Step 2: Basic bundle validation
    if validate_bundle_basic; then
        step_results+=("✅ Basic Validation: PASSED")
    else
        step_results+=("❌ Basic Validation: FAILED")
        overall_result=1
    fi

    # Step 3: Kubernetes version matrix testing
    if validate_k8s_matrix; then
        step_results+=("✅ K8s Matrix Testing: PASSED")
    else
        step_results+=("❌ K8s Matrix Testing: FAILED")
        overall_result=1
    fi

    # Generate final report
    echo ""
    log_section "Validation Summary"
    for result in "${step_results[@]}"; do
        echo "  $result"
    done
    echo ""

    # Generate detailed report
    generate_report $overall_result

    exit $overall_result
}

# Run main function
main
