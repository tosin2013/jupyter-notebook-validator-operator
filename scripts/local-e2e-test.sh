#!/bin/bash
set -e

# Local E2E Test Script
# Replicates GitHub Actions E2E workflow for local testing
# Usage: ./scripts/local-e2e-test.sh [tier1|tier2|tier3|all]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIER="${1:-all}"
TEST_NAMESPACE="e2e-tests"
OPERATOR_NAMESPACE="jupyter-notebook-validator-operator"
TEST_REPO="https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
IMAGE_TAG="local-e2e-$(git rev-parse --short HEAD)-$(date +%s)"
IMAGE="quay.io/takinosh/jupyter-notebook-validator-operator:${IMAGE_TAG}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Local E2E Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Test Tier: ${TEST_TIER}"
echo "Test Namespace: ${TEST_NAMESPACE}"
echo "Operator Namespace: ${OPERATOR_NAMESPACE}"
echo "Image: ${IMAGE}"
echo ""

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Check prerequisites
print_section "Checking Prerequisites"

if ! command -v oc &> /dev/null; then
    print_error "oc CLI not found. Please install OpenShift CLI."
    exit 1
fi
print_success "oc CLI found"

# Check for docker or podman
if command -v docker &> /dev/null; then
    CONTAINER_TOOL="docker"
    print_success "docker found"
elif command -v podman &> /dev/null; then
    CONTAINER_TOOL="podman"
    print_success "podman found (using as container tool)"
else
    print_error "Neither docker nor podman found. Please install one."
    exit 1
fi

if ! command -v make &> /dev/null; then
    print_error "make not found. Please install make."
    exit 1
fi
print_success "make found"

# Verify cluster access
print_section "Verifying Cluster Access"
if ! oc cluster-info &> /dev/null; then
    print_error "Cannot connect to OpenShift cluster. Please login first."
    exit 1
fi
print_success "Connected to OpenShift cluster"
oc cluster-info
oc version

# Create test namespace
print_section "Creating Test Namespace"
if oc get namespace ${TEST_NAMESPACE} &> /dev/null; then
    print_warning "Test namespace ${TEST_NAMESPACE} already exists. Deleting..."
    oc delete namespace ${TEST_NAMESPACE} --wait=true || true
    sleep 5
fi
oc create namespace ${TEST_NAMESPACE}
oc project ${TEST_NAMESPACE}
print_success "Test namespace created: ${TEST_NAMESPACE}"

# Build operator image
print_section "Building Operator Image"
echo "Building: ${IMAGE}"
echo "Using container tool: ${CONTAINER_TOOL}"
if ! make docker-build IMG="${IMAGE}" CONTAINER_TOOL="${CONTAINER_TOOL}"; then
    print_error "Failed to build operator image"
    exit 1
fi
print_success "Operator image built successfully"

# Push operator image
print_section "Pushing Operator Image to Quay.io"
echo "Checking Quay.io credentials..."
if [ -z "$QUAY_USERNAME" ] || [ -z "$QUAY_PASSWORD" ]; then
    print_warning "QUAY_USERNAME or QUAY_PASSWORD not set"
    print_warning "Attempting to use existing ${CONTAINER_TOOL} login..."
else
    echo "Logging in to Quay.io..."
    echo "$QUAY_PASSWORD" | ${CONTAINER_TOOL} login -u "$QUAY_USERNAME" --password-stdin quay.io
fi

echo "Pushing: ${IMAGE}"
if ! make docker-push IMG="${IMAGE}" CONTAINER_TOOL="${CONTAINER_TOOL}"; then
    print_error "Failed to push operator image"
    print_warning "You may need to set QUAY_USERNAME and QUAY_PASSWORD environment variables"
    exit 1
fi
print_success "Operator image pushed successfully"

# Install cert-manager
print_section "Installing cert-manager"
if oc get namespace cert-manager &> /dev/null; then
    print_warning "cert-manager already installed"
else
    echo "Installing cert-manager v1.13.3..."
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
    
    echo "Waiting for cert-manager to be ready..."
    kubectl wait --for=condition=available deployment/cert-manager \
        -n cert-manager --timeout=5m
    kubectl wait --for=condition=available deployment/cert-manager-webhook \
        -n cert-manager --timeout=5m
    kubectl wait --for=condition=available deployment/cert-manager-cainjector \
        -n cert-manager --timeout=5m
fi
print_success "cert-manager is ready"
kubectl get pods -n cert-manager

# Install operator
print_section "Installing Operator"
echo "Installing CRDs..."
make install

echo "Deploying operator with webhook support..."
make deploy IMG="${IMAGE}"

echo "Waiting for operator to be ready..."
oc wait --for=condition=available deployment/notebook-validator-controller-manager \
    -n ${OPERATOR_NAMESPACE} --timeout=5m

echo "Verifying webhook certificate..."
oc get certificate -n ${OPERATOR_NAMESPACE}
oc get secret webhook-server-cert -n ${OPERATOR_NAMESPACE}

print_success "Operator deployed successfully"
oc get pods -n ${OPERATOR_NAMESPACE}

# Clone test notebooks
print_section "Cloning Test Notebooks Repository"
if [ -d "test-notebooks" ]; then
    print_warning "test-notebooks directory exists. Removing..."
    rm -rf test-notebooks
fi

git clone ${TEST_REPO} test-notebooks
cd test-notebooks
print_success "Test notebooks cloned successfully"
ls -la

# Run tests based on tier
print_section "Running Tests (Tier: ${TEST_TIER})"

run_tier1_tests() {
    echo -e "${BLUE}=== Running Tier 1 Tests (Simple notebooks) ===${NC}"
    if [ -f scripts/run-tier1-tests.sh ]; then
        bash scripts/run-tier1-tests.sh
    else
        print_warning "scripts/run-tier1-tests.sh not found"
        echo "Looking for tier1 notebooks..."
        if [ -d "notebooks/tier1-simple" ]; then
            for notebook in notebooks/tier1-simple/*.ipynb; do
                if [ -f "$notebook" ]; then
                    echo "Found: $notebook"
                fi
            done
        else
            print_warning "notebooks/tier1-simple directory not found"
        fi
    fi
    print_success "Tier 1 tests completed"
}

run_tier2_tests() {
    echo -e "${BLUE}=== Running Tier 2 Tests (Intermediate notebooks) ===${NC}"
    if [ -f scripts/run-tier2-tests.sh ]; then
        bash scripts/run-tier2-tests.sh
    else
        print_warning "Tier 2 tests not yet implemented"
    fi
    print_success "Tier 2 tests completed"
}

run_tier3_tests() {
    echo -e "${BLUE}=== Running Tier 3 Tests (Complex notebooks) ===${NC}"
    if [ -f scripts/run-tier3-tests.sh ]; then
        bash scripts/run-tier3-tests.sh
    else
        print_warning "Tier 3 tests not yet implemented"
    fi
    print_success "Tier 3 tests completed"
}

case ${TEST_TIER} in
    tier1|1)
        run_tier1_tests
        ;;
    tier2|2)
        run_tier2_tests
        ;;
    tier3|3)
        run_tier3_tests
        ;;
    all)
        run_tier1_tests
        run_tier2_tests
        run_tier3_tests
        ;;
    *)
        print_error "Invalid test tier: ${TEST_TIER}"
        echo "Usage: $0 [tier1|tier2|tier3|all]"
        exit 1
        ;;
esac

cd ..

# Collect test results
print_section "Collecting Test Results"
mkdir -p test-results/logs

echo "Getting NotebookValidationJobs..."
oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o yaml > test-results/notebookvalidationjobs.yaml || true

echo "Getting validation pod logs..."
for pod in $(oc get pods -n ${TEST_NAMESPACE} -l app=notebook-validation -o name 2>/dev/null); do
    pod_name=$(basename $pod)
    echo "  Collecting logs from: ${pod_name}"
    oc logs $pod -n ${TEST_NAMESPACE} > test-results/logs/${pod_name}.log 2>&1 || true
done

echo "Getting operator logs..."
oc logs -n ${OPERATOR_NAMESPACE} \
    -l control-plane=controller-manager --tail=500 > test-results/logs/operator.log 2>&1 || true

print_success "Test results collected in test-results/"

# Summary
print_section "Test Summary"
echo "Test results saved to: test-results/"
echo "  - notebookvalidationjobs.yaml"
echo "  - logs/*.log"
echo ""
echo "To view operator logs:"
echo "  cat test-results/logs/operator.log"
echo ""
echo "To view validation job status:"
echo "  oc get notebookvalidationjobs -n ${TEST_NAMESPACE}"
echo ""

print_success "Local E2E tests completed!"
echo ""
echo -e "${YELLOW}Note: Test namespace ${TEST_NAMESPACE} is still running.${NC}"
echo -e "${YELLOW}To clean up, run: oc delete namespace ${TEST_NAMESPACE}${NC}"

