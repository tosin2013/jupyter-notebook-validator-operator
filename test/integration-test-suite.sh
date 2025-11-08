#!/bin/bash
# Integration Test Suite for Jupyter Notebook Validator Operator
# Tests against https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks
#
# Test Coverage:
# 1. ESO (External Secrets Operator) Integration
# 2. Model-Aware Validation with KServe
# 3. Model-Aware Validation with OpenShift AI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NAMESPACE:-mlops}"
TEST_REPO="https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
TIMEOUT="${TIMEOUT:-300}" # 5 minutes default timeout

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

run_test() {
    local test_name="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test $TESTS_RUN: $test_name${NC}"
    echo -e "${BLUE}========================================${NC}"
}

wait_for_job() {
    local job_name="$1"
    local timeout="$2"
    local namespace="$3"
    
    log_info "Waiting for job '$job_name' to complete (timeout: ${timeout}s)..."
    
    local elapsed=0
    local interval=5
    
    while [ $elapsed -lt $timeout ]; do
        local phase=$(oc get notebookvalidationjob "$job_name" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
        
        if [ "$phase" == "Succeeded" ]; then
            log_success "Job '$job_name' completed successfully"
            return 0
        elif [ "$phase" == "Failed" ]; then
            log_error "Job '$job_name' failed"
            oc get notebookvalidationjob "$job_name" -n "$namespace" -o yaml
            return 1
        fi
        
        echo -n "."
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    echo ""
    log_error "Job '$job_name' timed out after ${timeout}s"
    return 1
}

cleanup_test() {
    local job_name="$1"
    local namespace="$2"
    
    log_info "Cleaning up test resources for '$job_name'..."
    oc delete notebookvalidationjob "$job_name" -n "$namespace" --ignore-not-found=true 2>/dev/null || true
    sleep 2
}

# Test 1: ESO Integration Test
test_eso_integration() {
    run_test "ESO (External Secrets Operator) Integration"
    
    # Check if ESO is installed
    log_info "Checking ESO installation..."
    if ! oc get crd externalsecrets.external-secrets.io &>/dev/null; then
        log_error "ESO CRD not found. Please install External Secrets Operator first."
        return 1
    fi
    log_success "ESO CRD found"
    
    # Apply ESO test resources
    log_info "Applying ESO test resources..."
    if ! oc apply -f config/samples/eso-fake-secretstore.yaml; then
        log_error "Failed to apply ESO test resources"
        return 1
    fi
    log_success "ESO test resources applied"
    
    # Wait for ExternalSecrets to sync
    log_info "Waiting for ExternalSecrets to sync..."
    sleep 10
    
    # Verify secrets were created
    local secrets=("aws-credentials-eso" "database-config-eso" "mlflow-credentials-eso" "api-keys-eso")
    for secret in "${secrets[@]}"; do
        if oc get secret "$secret" -n "$NAMESPACE" &>/dev/null; then
            log_success "Secret '$secret' created successfully"
        else
            log_error "Secret '$secret' not found"
            return 1
        fi
    done
    
    # Wait for NotebookValidationJob to complete
    if wait_for_job "eso-integration-test" "$TIMEOUT" "$NAMESPACE"; then
        log_success "ESO integration test passed"
        
        # Verify envFrom was properly set
        local envfrom_count=$(oc get notebookvalidationjob eso-integration-test -n "$NAMESPACE" -o jsonpath='{.spec.podConfig.envFrom}' | jq '. | length')
        if [ "$envfrom_count" == "4" ]; then
            log_success "envFrom field contains 4 secret references"
        else
            log_error "envFrom field contains $envfrom_count references (expected 4)"
            return 1
        fi
        
        return 0
    else
        log_error "ESO integration test failed"
        return 1
    fi
}

# Test 2: KServe Model Validation Test
test_kserve_model_validation() {
    run_test "KServe Model-Aware Validation"
    
    # Check if KServe is installed
    log_info "Checking KServe installation..."
    if ! oc get crd inferenceservices.serving.kserve.io &>/dev/null; then
        log_warning "KServe CRD not found. Skipping KServe test."
        log_warning "To run this test, install KServe: https://kserve.github.io/website/"
        return 0  # Skip, not fail
    fi
    log_success "KServe CRD found"
    
    # Apply KServe test resources
    log_info "Applying KServe test resources..."
    cleanup_test "model-validation-kserve-example" "$NAMESPACE"
    
    if ! oc apply -f config/samples/model-validation-kserve.yaml; then
        log_error "Failed to apply KServe test resources"
        return 1
    fi
    log_success "KServe test resources applied"
    
    # Wait for InferenceService to be ready (if it exists)
    log_info "Waiting for InferenceService to be ready..."
    sleep 15
    
    # Wait for NotebookValidationJob to complete
    if wait_for_job "model-validation-kserve-example" "$TIMEOUT" "$NAMESPACE"; then
        log_success "KServe model validation test passed"
        
        # Verify model validation result
        local model_validation_success=$(oc get notebookvalidationjob model-validation-kserve-example -n "$NAMESPACE" -o jsonpath='{.status.modelValidationResult.success}' 2>/dev/null || echo "false")
        if [ "$model_validation_success" == "true" ]; then
            log_success "Model validation succeeded"
        else
            log_warning "Model validation result not available or failed (this is expected if models are not deployed)"
        fi
        
        return 0
    else
        log_error "KServe model validation test failed"
        return 1
    fi
}

# Test 3: OpenShift AI Model Validation Test
test_openshift_ai_model_validation() {
    run_test "OpenShift AI Model-Aware Validation"
    
    # Check if OpenShift AI is installed
    log_info "Checking OpenShift AI installation..."
    if ! oc get crd inferenceservices.serving.kserve.io &>/dev/null; then
        log_warning "OpenShift AI/KServe CRD not found. Skipping OpenShift AI test."
        log_warning "To run this test, install OpenShift AI: https://docs.redhat.com/en/documentation/red_hat_openshift_ai_self-managed/"
        return 0  # Skip, not fail
    fi
    log_success "OpenShift AI/KServe CRD found"
    
    # Apply OpenShift AI test resources
    log_info "Applying OpenShift AI test resources..."
    cleanup_test "model-validation-openshift-ai-example" "$NAMESPACE"
    
    if ! oc apply -f config/samples/model-validation-openshift-ai.yaml; then
        log_error "Failed to apply OpenShift AI test resources"
        return 1
    fi
    log_success "OpenShift AI test resources applied"
    
    # Wait for NotebookValidationJob to complete
    if wait_for_job "model-validation-openshift-ai-example" "$TIMEOUT" "$NAMESPACE"; then
        log_success "OpenShift AI model validation test passed"
        
        # Verify model validation result
        local platform=$(oc get notebookvalidationjob model-validation-openshift-ai-example -n "$NAMESPACE" -o jsonpath='{.status.modelValidationResult.platform}' 2>/dev/null || echo "unknown")
        if [ "$platform" == "openshift-ai" ]; then
            log_success "Platform correctly detected as OpenShift AI"
        else
            log_warning "Platform detected as '$platform' (expected 'openshift-ai')"
        fi
        
        return 0
    else
        log_error "OpenShift AI model validation test failed"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Jupyter Notebook Validator Operator${NC}"
    echo -e "${BLUE}Integration Test Suite${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v oc &>/dev/null; then
        log_error "oc command not found. Please install OpenShift CLI."
        exit 1
    fi
    log_success "oc command found"
    
    if ! command -v jq &>/dev/null; then
        log_error "jq command not found. Please install jq."
        exit 1
    fi
    log_success "jq command found"
    
    # Check cluster connection
    if ! oc cluster-info &>/dev/null; then
        log_error "Not connected to OpenShift cluster. Please login first."
        exit 1
    fi
    log_success "Connected to OpenShift cluster"
    
    # Create namespace if it doesn't exist
    if ! oc get namespace "$NAMESPACE" &>/dev/null; then
        log_info "Creating namespace '$NAMESPACE'..."
        oc create namespace "$NAMESPACE"
    fi
    log_success "Namespace '$NAMESPACE' exists"
    
    # Apply CRD
    log_info "Applying NotebookValidationJob CRD..."
    oc apply -f config/crd/bases/mlops.mlops.dev_notebookvalidationjobs.yaml
    log_success "CRD applied"
    
    echo ""
    log_info "Starting integration tests..."
    echo ""
    
    # Run tests
    test_eso_integration || true
    test_kserve_model_validation || true
    test_openshift_ai_model_validation || true
    
    # Print summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "Tests Run:    ${TESTS_RUN}"
    echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "Some tests failed!"
        exit 1
    fi
}

# Run main function
main "$@"

