#!/bin/bash
set -e

# Tier 2 E2E Tests - Build Integration Testing
# Tests S2I and Tekton build workflows according to ADR-023, ADR-027, ADR-028, ADR-031, ADR-033
#
# This script tests the COMPLETE build workflow:
# 1. S2I Build Strategy (OpenShift BuildConfig)
# 2. Tekton Build Strategy (OpenShift Pipelines)
#
# Both strategies should:
# - Clone git repository
# - Build custom image with dependencies
# - Push to registry
# - Use built image for validation (no git-clone init container)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
TEST_NAMESPACE="${TEST_NAMESPACE:-e2e-tests}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-jupyter-notebook-validator-operator}"

# Target test repository (public repository for E2E testing)
REPO_URL="${TEST_REPO:-https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git}"
REPO_REF="${TEST_REPO_REF:-main}"
NOTEBOOK_PATH="${NOTEBOOK_PATH:-notebooks/tier1-simple/01-hello-world.ipynb}"

# Use OpenShift S2I-enabled Jupyter image (ADR-023)
# These images have S2I scripts at /usr/libexec/s2i/assemble
BASE_IMAGE="${BASE_IMAGE:-image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-image-registry.openshift-image-registry.svc:5000}"

# Git credentials (REQUIRED - repository is private)
# The test repository is PRIVATE and requires authentication
# Use git-credentials secret (created by setup-git-credentials.sh)
GIT_CREDENTIALS_SECRET="${GIT_CREDENTIALS_SECRET:-git-credentials}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check oc CLI
    if ! command -v oc &> /dev/null; then
        print_error "oc CLI not found"
        exit 1
    fi
    print_success "oc CLI found"
    
    # Check cluster connection
    if ! oc cluster-info &> /dev/null; then
        print_error "Not connected to OpenShift cluster"
        exit 1
    fi
    print_success "Connected to OpenShift cluster"
    
    # Check if operator is deployed
    if ! oc get deployment notebook-validator-controller-manager -n "$OPERATOR_NAMESPACE" &> /dev/null; then
        print_error "Operator not deployed in namespace: $OPERATOR_NAMESPACE"
        exit 1
    fi
    print_success "Operator deployed in namespace: $OPERATOR_NAMESPACE"
    
    # Check if test namespace exists
    if ! oc get namespace "$TEST_NAMESPACE" &> /dev/null; then
        print_warning "Test namespace does not exist, creating: $TEST_NAMESPACE"
        oc create namespace "$TEST_NAMESPACE"
    fi
    print_success "Test namespace exists: $TEST_NAMESPACE"

    # Check if Git credentials secret exists (REQUIRED for private repo)
    if ! oc get secret "$GIT_CREDENTIALS_SECRET" -n "$TEST_NAMESPACE" &> /dev/null; then
        print_error "Git credentials secret not found: $GIT_CREDENTIALS_SECRET"
        print_info "The test repository is PRIVATE and requires authentication"
        print_info "Please run: ./scripts/setup-git-credentials.sh $TEST_NAMESPACE"
        print_info "You need to set GITHUB_USERNAME and GITHUB_TOKEN environment variables"
        exit 1
    fi
    print_success "Git credentials secret exists: $GIT_CREDENTIALS_SECRET"
    
    # Check for S2I BuildConfig API
    if oc api-resources | grep -q "buildconfigs"; then
        print_success "S2I BuildConfig API available"
    else
        print_warning "S2I BuildConfig API not available (tests will be skipped)"
    fi
    
    # Check for Tekton Pipeline API
    if oc api-resources | grep -q "tekton.dev"; then
        print_success "Tekton Pipeline API available"
        # Verify git-clone and buildah tasks exist
        if oc get task git-clone -n openshift-pipelines &> /dev/null && \
           oc get task buildah -n openshift-pipelines &> /dev/null; then
            print_success "Required Tekton tasks (git-clone, buildah) found in openshift-pipelines namespace"
        else
            print_warning "Required Tekton tasks not found in openshift-pipelines namespace"
        fi
    else
        print_warning "Tekton Pipeline API not available (tests will be skipped)"
    fi
}

# Test S2I Build Strategy
test_s2i_build() {
    print_header "Test 1: S2I Build Strategy (ADR-023, ADR-027)"

    local job_name="tier2-s2i-build-test"

    print_info "Creating NotebookValidationJob with S2I build..."
    print_info "Job name: $job_name"
    print_info "Repository: $REPO_URL"
    print_info "Notebook: $NOTEBOOK_PATH"
    print_info "Strategy: s2i"
    print_info "Base image: $BASE_IMAGE"

    # Build the YAML with optional credentials
    local yaml_content="apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: ${job_name}
  namespace: ${TEST_NAMESPACE}
  labels:
    tier: \"2\"
    test-type: \"build-integration\"
    build-strategy: \"s2i\"
spec:
  notebook:
    git:
      url: \"${REPO_URL}\"
      ref: \"${REPO_REF}\""

    # Add credentials secret if specified (for private repos)
    if [ -n "$GIT_CREDENTIALS_SECRET" ]; then
        print_info "Using Git credentials secret: $GIT_CREDENTIALS_SECRET"
        yaml_content="${yaml_content}
      credentialsSecret: \"${GIT_CREDENTIALS_SECRET}\""
    else
        print_info "No Git credentials (public repository)"
    fi

    yaml_content="${yaml_content}
    path: \"${NOTEBOOK_PATH}\"
  podConfig:
    buildConfig:
      enabled: true
      strategy: \"s2i\"
      baseImage: \"${BASE_IMAGE}\"
      autoGenerateRequirements: false
      requirementsFile: \"requirements.txt\"
      fallbackStrategy: \"warn\"
    containerImage: \"${BASE_IMAGE}\"  # Fallback only
    # serviceAccountName is automatically injected by mutating webhook (ADR-005)
  timeout: \"30m\"
  retryLimit: 1"

    echo "$yaml_content" | oc apply -f -

    
    if [ $? -ne 0 ]; then
        print_error "Failed to create S2I build job"
        return 1
    fi
    print_success "S2I build job created"
    
    # Wait for build to start
    print_info "Waiting for S2I build to start..."
    sleep 10
    
    # Check if BuildConfig was created
    local build_config_name="${job_name}-build"
    if oc get buildconfig "$build_config_name" -n "$TEST_NAMESPACE" &> /dev/null; then
        print_success "BuildConfig created: $build_config_name"
        
        # Get build status
        local build_name=$(oc get builds -n "$TEST_NAMESPACE" -l buildconfig="$build_config_name" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null)
        if [ -n "$build_name" ]; then
            print_info "Build started: $build_name"
            print_info "Monitoring build progress..."
            
            # Follow build logs (with timeout)
            timeout 600 oc logs -f "build/$build_name" -n "$TEST_NAMESPACE" 2>&1 | head -50 || true
        fi
    else
        print_warning "BuildConfig not found (may not be created yet)"
    fi
    
    # Wait for job completion
    wait_for_job_completion "$job_name" 1800  # 30 minutes timeout
    
    return $?
}

# Test Tekton Build Strategy
test_tekton_build() {
    print_header "Test 2: Tekton Build Strategy (ADR-028, ADR-031)"

    local job_name="tier2-tekton-build-test"

    print_info "Creating NotebookValidationJob with Tekton build..."
    print_info "Job name: $job_name"
    print_info "Repository: $REPO_URL"
    print_info "Notebook: $NOTEBOOK_PATH"
    print_info "Strategy: tekton"
    print_info "Base image: $BASE_IMAGE"

    # Build the YAML with optional credentials
    local yaml_content="apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: ${job_name}
  namespace: ${TEST_NAMESPACE}
  labels:
    tier: \"2\"
    test-type: \"build-integration\"
    build-strategy: \"tekton\"
spec:
  notebook:
    git:
      url: \"${REPO_URL}\"
      ref: \"${REPO_REF}\""

    # Add credentials secret if specified (for private repos)
    # Note: Tekton requires different secret format (ADR-031)
    if [ -n "$GIT_CREDENTIALS_SECRET" ]; then
        print_info "Using Git credentials secret: ${GIT_CREDENTIALS_SECRET}-tekton"
        print_warning "Tekton requires basic-auth format (.gitconfig + .git-credentials)"
        yaml_content="${yaml_content}
      credentialsSecret: \"${GIT_CREDENTIALS_SECRET}\""
    else
        print_info "No Git credentials (public repository)"
    fi

    yaml_content="${yaml_content}
    path: \"${NOTEBOOK_PATH}\"
  podConfig:
    buildConfig:
      enabled: true
      strategy: \"tekton\"
      baseImage: \"${BASE_IMAGE}\"
      autoGenerateRequirements: false
      requirementsFile: \"requirements.txt\"
      fallbackStrategy: \"warn\"
    containerImage: \"${BASE_IMAGE}\"  # Fallback only
    # serviceAccountName is automatically injected by mutating webhook (ADR-005)
  timeout: \"30m\"
  retryLimit: 1"

    echo "$yaml_content" | oc apply -f -
    
    if [ $? -ne 0 ]; then
        print_error "Failed to create Tekton build job"
        return 1
    fi
    print_success "Tekton build job created"
    
    # Wait for pipeline to start
    print_info "Waiting for Tekton pipeline to start..."
    sleep 10
    
    # Check if Pipeline was created
    local pipeline_name="${job_name}-pipeline"
    if oc get pipeline "$pipeline_name" -n "$TEST_NAMESPACE" &> /dev/null; then
        print_success "Pipeline created: $pipeline_name"
        
        # Get PipelineRun status
        local pipelinerun_name=$(oc get pipelineruns -n "$TEST_NAMESPACE" -l tekton.dev/pipeline="$pipeline_name" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null)
        if [ -n "$pipelinerun_name" ]; then
            print_info "PipelineRun started: $pipelinerun_name"
            print_info "Monitoring pipeline progress..."
            
            # Show pipeline status
            oc get pipelinerun "$pipelinerun_name" -n "$TEST_NAMESPACE" -o yaml | grep -A 20 "status:" || true
        fi
    else
        print_warning "Pipeline not found (may not be created yet)"
    fi
    
    # Wait for job completion
    wait_for_job_completion "$job_name" 1800  # 30 minutes timeout
    
    return $?
}

# Wait for job completion
wait_for_job_completion() {
    local job_name=$1
    local timeout=$2
    local elapsed=0
    local interval=15
    
    print_info "Waiting for job completion: $job_name (timeout: ${timeout}s)"
    
    while [ $elapsed -lt $timeout ]; do
        local phase=$(oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        
        case "$phase" in
            "Succeeded")
                print_success "Job succeeded: $job_name"
                print_info "Job details:"
                oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml | grep -A 30 "status:"
                return 0
                ;;
            "Failed")
                print_error "Job failed: $job_name"
                print_info "Job details:"
                oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
                return 1
                ;;
            "Running"|"Pending")
                print_info "  Status: $phase (${elapsed}s elapsed)"
                ;;
            *)
                print_info "  Status: $phase"
                ;;
        esac

        sleep $interval
        elapsed=$((elapsed + interval))
    done

    print_error "Timeout waiting for job: $job_name"
    oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o yaml
    return 1
}

# Verify build artifacts
verify_build_artifacts() {
    local job_name=$1
    local strategy=$2

    print_header "Verifying Build Artifacts for $job_name"

    # Check if validation pod was created
    local pod_name=$(oc get pods -n "$TEST_NAMESPACE" -l notebookvalidationjob="$job_name" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null)

    if [ -z "$pod_name" ]; then
        print_warning "No validation pod found for job: $job_name"
        return 1
    fi

    print_success "Validation pod found: $pod_name"

    # Check if pod has init containers (should NOT have git-clone if build succeeded)
    local init_containers=$(oc get pod "$pod_name" -n "$TEST_NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null)

    if echo "$init_containers" | grep -q "git-clone"; then
        print_error "Pod has git-clone init container - build may have failed!"
        print_info "Init containers: $init_containers"
        return 1
    else
        print_success "Pod does NOT have git-clone init container - using built image ✓"
    fi

    # Check build status in job
    local build_status=$(oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.buildStatus}' 2>/dev/null)
    print_info "Build status: $build_status"

    # Check for built image reference
    local built_image=$(oc get notebookvalidationjob "$job_name" -n "$TEST_NAMESPACE" -o jsonpath='{.status.builtImage}' 2>/dev/null)
    if [ -n "$built_image" ]; then
        print_success "Built image: $built_image"
    else
        print_warning "No built image reference found in job status"
    fi

    return 0
}

# Cleanup test resources
cleanup_test_resources() {
    print_header "Cleaning Up Test Resources"

    # Delete NotebookValidationJobs
    print_info "Deleting NotebookValidationJobs..."
    oc delete notebookvalidationjobs -n "$TEST_NAMESPACE" -l tier=2 --wait=false 2>/dev/null || true

    # Delete BuildConfigs
    print_info "Deleting BuildConfigs..."
    oc delete buildconfigs -n "$TEST_NAMESPACE" -l tier=2 --wait=false 2>/dev/null || true

    # Delete Pipelines
    print_info "Deleting Tekton Pipelines..."
    oc delete pipelines -n "$TEST_NAMESPACE" -l tier=2 --wait=false 2>/dev/null || true

    # Delete PipelineRuns
    print_info "Deleting Tekton PipelineRuns..."
    oc delete pipelineruns -n "$TEST_NAMESPACE" -l tier=2 --wait=false 2>/dev/null || true

    # Delete validation pods
    print_info "Deleting validation pods..."
    oc delete pods -n "$TEST_NAMESPACE" -l tier=2 --wait=false 2>/dev/null || true

    print_success "Cleanup completed"
}

# Main execution
main() {
    print_header "Tier 2 E2E Tests - Build Integration"
    print_info "Testing S2I and Tekton build workflows per ADR-023, ADR-027, ADR-028, ADR-031, ADR-033"

    # Check prerequisites
    check_prerequisites

    # Track test results
    local s2i_result=0
    local tekton_result=0

    # Test S2I build
    if oc api-resources | grep -q "buildconfigs"; then
        test_s2i_build
        s2i_result=$?

        if [ $s2i_result -eq 0 ]; then
            verify_build_artifacts "tier2-s2i-build-test" "s2i"
        fi
    else
        print_warning "Skipping S2I tests - BuildConfig API not available"
    fi

    # Test Tekton build
    if oc api-resources | grep -q "tekton.dev"; then
        test_tekton_build
        tekton_result=$?

        if [ $tekton_result -eq 0 ]; then
            verify_build_artifacts "tier2-tekton-build-test" "tekton"
        fi
    else
        print_warning "Skipping Tekton tests - Pipeline API not available"
    fi

    # Summary
    print_header "Test Summary"

    if [ $s2i_result -eq 0 ]; then
        print_success "S2I Build Test: PASSED"
    else
        print_error "S2I Build Test: FAILED"
    fi

    if [ $tekton_result -eq 0 ]; then
        print_success "Tekton Build Test: PASSED"
    else
        print_error "Tekton Build Test: FAILED"
    fi

    # Cleanup (optional - comment out to keep resources for debugging)
    # cleanup_test_resources

    # Exit with failure if any test failed
    if [ $s2i_result -ne 0 ] || [ $tekton_result -ne 0 ]; then
        print_error "Some tests failed!"
        exit 1
    fi

    print_success "All Tier 2 build tests passed!"
    exit 0
}

# Run main function
main "$@"
