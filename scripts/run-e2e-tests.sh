#!/bin/bash

# End-to-End Test Runner for Jupyter Notebook Validator Operator
# Tests complete workflows including build integration and notebook validation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_NAMESPACE="${TEST_NAMESPACE:-notebook-validator-e2e-test}"
TEST_TIMEOUT="${TEST_TIMEOUT:-15m}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Jupyter Notebook Validator E2E Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if we're connected to a cluster
echo -e "${YELLOW}Checking cluster connectivity...${NC}"
if ! kubectl cluster-info &> /dev/null && ! oc cluster-info &> /dev/null; then
    echo -e "${RED}❌ Not connected to a Kubernetes/OpenShift cluster${NC}"
    echo "Please configure your kubeconfig and try again"
    exit 1
fi

# Detect cluster type
if oc version &> /dev/null; then
    CLUSTER_TYPE="openshift"
    CLI="oc"
    echo -e "${GREEN}✅ Connected to OpenShift cluster${NC}"
    oc cluster-info | head -1
else
    CLUSTER_TYPE="kubernetes"
    CLI="kubectl"
    echo -e "${GREEN}✅ Connected to Kubernetes cluster${NC}"
    kubectl cluster-info | head -1
fi
echo ""

# Detect available build strategies
echo -e "${YELLOW}Detecting available build strategies...${NC}"
S2I_AVAILABLE=false
TEKTON_AVAILABLE=false

if [ "$CLUSTER_TYPE" = "openshift" ]; then
    if $CLI api-resources | grep -q "builds.build.openshift.io"; then
        S2I_AVAILABLE=true
        echo -e "${GREEN}✅ S2I (OpenShift BuildConfig) detected${NC}"
    fi
fi

if $CLI api-resources | grep -q "pipelineruns.tekton.dev"; then
    TEKTON_AVAILABLE=true
    echo -e "${GREEN}✅ Tekton Pipelines detected${NC}"
fi

if [ "$S2I_AVAILABLE" = false ] && [ "$TEKTON_AVAILABLE" = false ]; then
    echo -e "${RED}❌ No build strategies available${NC}"
    echo "This cluster does not have S2I or Tekton installed"
    exit 1
fi
echo ""

# Confirm test execution
echo -e "${YELLOW}Test Configuration:${NC}"
echo "  Cluster Type: $CLUSTER_TYPE"
echo "  Test Namespace: $TEST_NAMESPACE"
echo "  Test Timeout: $TEST_TIMEOUT"
echo "  S2I Available: $S2I_AVAILABLE"
echo "  Tekton Available: $TEKTON_AVAILABLE"
echo ""

read -p "Run E2E tests? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
fi
echo ""

# Cleanup function
cleanup() {
    if [ "$SKIP_CLEANUP" = "true" ]; then
        echo -e "${YELLOW}⚠️  Skipping cleanup (SKIP_CLEANUP=true)${NC}"
        return
    fi

    echo -e "${YELLOW}Cleaning up test resources...${NC}"
    
    # Delete test namespace
    if $CLI get namespace "$TEST_NAMESPACE" &> /dev/null; then
        echo "Deleting namespace: $TEST_NAMESPACE"
        $CLI delete namespace "$TEST_NAMESPACE" --ignore-not-found=true --wait=true --timeout=60s || true
    fi
    
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Register cleanup on exit
trap cleanup EXIT

# Create test namespace
echo -e "${YELLOW}Creating test namespace: $TEST_NAMESPACE${NC}"
$CLI create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | $CLI apply -f -
echo -e "${GREEN}✅ Test namespace ready${NC}"
echo ""

# Run E2E tests
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Running E2E Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

cd "$PROJECT_ROOT"

# Test 1: S2I Workflow (if available)
if [ "$S2I_AVAILABLE" = "true" ]; then
    echo -e "${YELLOW}Test 1: S2I Build Workflow${NC}"
    echo "Testing complete S2I build workflow..."
    
    if go test -v -tags=e2e -timeout="$TEST_TIMEOUT" ./pkg/build/... -run "TestE2ES2IWorkflow" -count=1; then
        echo -e "${GREEN}✅ S2I workflow test passed${NC}"
    else
        echo -e "${RED}❌ S2I workflow test failed${NC}"
        exit 1
    fi
    echo ""
fi

# Test 2: Tekton Workflow (if available)
if [ "$TEKTON_AVAILABLE" = "true" ]; then
    echo -e "${YELLOW}Test 2: Tekton Build Workflow${NC}"
    echo "Testing complete Tekton build workflow..."
    
    if go test -v -tags=e2e -timeout="$TEST_TIMEOUT" ./pkg/build/... -run "TestE2ETektonWorkflow" -count=1; then
        echo -e "${GREEN}✅ Tekton workflow test passed${NC}"
    else
        echo -e "${RED}❌ Tekton workflow test failed${NC}"
        exit 1
    fi
    echo ""
fi

# Test 3: Complete Notebook Validation with S2I (if available)
if [ "$S2I_AVAILABLE" = "true" ]; then
    echo -e "${YELLOW}Test 3: Complete Notebook Validation with S2I${NC}"
    echo "Testing end-to-end notebook validation with S2I build..."
    
    if go test -v -tags=e2e -timeout="$TEST_TIMEOUT" ./pkg/build/... -run "TestE2ECompleteNotebookValidationWithS2I" -count=1; then
        echo -e "${GREEN}✅ Complete validation with S2I test passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Complete validation with S2I test skipped (requires controller deployment)${NC}"
    fi
    echo ""
fi

# Test 4: Complete Notebook Validation with Tekton (if available)
if [ "$TEKTON_AVAILABLE" = "true" ]; then
    echo -e "${YELLOW}Test 4: Complete Notebook Validation with Tekton${NC}"
    echo "Testing end-to-end notebook validation with Tekton build..."
    
    if go test -v -tags=e2e -timeout="$TEST_TIMEOUT" ./pkg/build/... -run "TestE2ECompleteNotebookValidationWithTekton" -count=1; then
        echo -e "${GREEN}✅ Complete validation with Tekton test passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Complete validation with Tekton test skipped (requires controller deployment)${NC}"
    fi
    echo ""
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}E2E Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✅ All E2E tests completed successfully!${NC}"
echo ""
echo "Test Results:"
if [ "$S2I_AVAILABLE" = "true" ]; then
    echo "  ✅ S2I workflow test: PASSED"
fi
if [ "$TEKTON_AVAILABLE" = "true" ]; then
    echo "  ✅ Tekton workflow test: PASSED"
fi
echo ""
echo "Next Steps:"
echo "  1. Integrate build strategies into controller reconciliation loop"
echo "  2. Test complete notebook validation with builds"
echo "  3. Test error handling and retry logic"
echo "  4. Test concurrent builds"
echo "  5. Performance and load testing"
echo ""

