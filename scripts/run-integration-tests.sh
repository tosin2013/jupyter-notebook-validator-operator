#!/bin/bash
# Integration test runner for Jupyter Notebook Validator Operator
# This script runs integration tests against a real OpenShift/Kubernetes cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_NAMESPACE="${TEST_NAMESPACE:-notebook-validator-integration-test}"
SKIP_S2I_BUILD="${SKIP_S2I_BUILD_TEST:-false}"
SKIP_TEKTON_BUILD="${SKIP_TEKTON_BUILD_TEST:-false}"
TEST_TIMEOUT="${TEST_TIMEOUT:-10m}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Jupyter Notebook Validator Operator${NC}"
echo -e "${BLUE}Integration Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

# Check if oc/kubectl is available
if command -v oc &> /dev/null; then
    CLI="oc"
    echo -e "${GREEN}✓${NC} OpenShift CLI (oc) found"
elif command -v kubectl &> /dev/null; then
    CLI="kubectl"
    echo -e "${GREEN}✓${NC} Kubernetes CLI (kubectl) found"
else
    echo -e "${RED}✗${NC} Neither oc nor kubectl found. Please install one."
    exit 1
fi

# Check cluster connectivity
echo -e "${YELLOW}Checking cluster connectivity...${NC}"
if $CLI cluster-info &> /dev/null; then
    echo -e "${GREEN}✓${NC} Connected to cluster"
    $CLI version --short 2>/dev/null || $CLI version
else
    echo -e "${RED}✗${NC} Cannot connect to cluster. Please check your kubeconfig."
    exit 1
fi

# Check current context
CURRENT_CONTEXT=$($CLI config current-context 2>/dev/null || echo "unknown")
echo -e "${BLUE}Current context:${NC} $CURRENT_CONTEXT"

# Detect cluster type
echo -e "${YELLOW}Detecting cluster type...${NC}"
if $CLI get clusterversion &> /dev/null; then
    CLUSTER_TYPE="OpenShift"
    CLUSTER_VERSION=$($CLI get clusterversion -o jsonpath='{.items[0].status.desired.version}' 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✓${NC} OpenShift cluster detected (version: $CLUSTER_VERSION)"
else
    CLUSTER_TYPE="Kubernetes"
    CLUSTER_VERSION=$($CLI version -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "unknown")
    echo -e "${GREEN}✓${NC} Kubernetes cluster detected (version: $CLUSTER_VERSION)"
fi

# Check for S2I availability (OpenShift only)
if [ "$CLUSTER_TYPE" = "OpenShift" ]; then
    echo -e "${YELLOW}Checking for S2I (Source-to-Image)...${NC}"
    if $CLI get buildconfig &> /dev/null; then
        echo -e "${GREEN}✓${NC} S2I is available"
        S2I_AVAILABLE=true
    else
        echo -e "${YELLOW}⚠${NC}  S2I not available or no permissions"
        S2I_AVAILABLE=false
        SKIP_S2I_BUILD="true"
    fi
else
    echo -e "${YELLOW}⚠${NC}  S2I not available (not an OpenShift cluster)"
    S2I_AVAILABLE=false
    SKIP_S2I_BUILD="true"
fi

# Check for Tekton availability
echo -e "${YELLOW}Checking for Tekton Pipelines...${NC}"
if $CLI get pipelinerun &> /dev/null; then
    echo -e "${GREEN}✓${NC} Tekton Pipelines is available"
    TEKTON_AVAILABLE=true
else
    echo -e "${YELLOW}⚠${NC}  Tekton Pipelines not available or no permissions"
    TEKTON_AVAILABLE=false
    SKIP_TEKTON_BUILD="true"
fi

# Create test namespace if it doesn't exist
echo -e "${YELLOW}Setting up test namespace...${NC}"
if $CLI get namespace $TEST_NAMESPACE &> /dev/null; then
    echo -e "${BLUE}ℹ${NC}  Test namespace already exists: $TEST_NAMESPACE"
else
    echo -e "${YELLOW}Creating test namespace: $TEST_NAMESPACE${NC}"
    $CLI create namespace $TEST_NAMESPACE
    echo -e "${GREEN}✓${NC} Test namespace created"
fi

# Set current namespace
$CLI config set-context --current --namespace=$TEST_NAMESPACE &> /dev/null

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Configuration${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Cluster Type:${NC}        $CLUSTER_TYPE"
echo -e "${BLUE}Cluster Version:${NC}     $CLUSTER_VERSION"
echo -e "${BLUE}Test Namespace:${NC}      $TEST_NAMESPACE"
echo -e "${BLUE}S2I Available:${NC}       $S2I_AVAILABLE"
echo -e "${BLUE}Tekton Available:${NC}    $TEKTON_AVAILABLE"
echo -e "${BLUE}Skip S2I Build:${NC}      $SKIP_S2I_BUILD"
echo -e "${BLUE}Skip Tekton Build:${NC}   $SKIP_TEKTON_BUILD"
echo -e "${BLUE}Test Timeout:${NC}        $TEST_TIMEOUT"
echo ""

# Confirm before running tests
read -p "$(echo -e ${YELLOW}Do you want to proceed with integration tests? [y/N]: ${NC})" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Tests cancelled.${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Running Integration Tests${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Export environment variables for tests
export SKIP_S2I_BUILD_TEST=$SKIP_S2I_BUILD
export SKIP_TEKTON_BUILD_TEST=$SKIP_TEKTON_BUILD

# Run integration tests
echo -e "${YELLOW}Running integration tests...${NC}"
echo ""

# Run tests with integration tag
if go test -v -tags=integration -timeout=$TEST_TIMEOUT ./pkg/build/... -count=1; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✓ All integration tests passed!${NC}"
    echo -e "${GREEN}========================================${NC}"
    TEST_RESULT=0
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}✗ Some integration tests failed${NC}"
    echo -e "${RED}========================================${NC}"
    TEST_RESULT=1
fi

# Cleanup option
echo ""
read -p "$(echo -e ${YELLOW}Do you want to clean up the test namespace? [y/N]: ${NC})" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Cleaning up test namespace...${NC}"
    $CLI delete namespace $TEST_NAMESPACE --ignore-not-found=true
    echo -e "${GREEN}✓${NC} Test namespace deleted"
else
    echo -e "${BLUE}ℹ${NC}  Test namespace preserved: $TEST_NAMESPACE"
    echo -e "${BLUE}ℹ${NC}  To clean up manually, run: $CLI delete namespace $TEST_NAMESPACE"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Integration Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Cluster:${NC}             $CLUSTER_TYPE $CLUSTER_VERSION"
echo -e "${BLUE}Test Namespace:${NC}      $TEST_NAMESPACE"
echo -e "${BLUE}Tests Run:${NC}           Integration tests"
if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${BLUE}Result:${NC}              ${GREEN}✓ PASSED${NC}"
else
    echo -e "${BLUE}Result:${NC}              ${RED}✗ FAILED${NC}"
fi
echo -e "${BLUE}========================================${NC}"

exit $TEST_RESULT

