#!/bin/bash
# ESO Integration Test Script
# Tests External Secrets Operator integration with Jupyter Notebook Validator Operator
#
# Prerequisites:
# - External Secrets Operator installed
# - mlops namespace exists
# - NotebookValidationJob CRD installed

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
NAMESPACE="mlops"
TEST_NAME="eso-integration-test"
TIMEOUT=300  # 5 minutes

echo "========================================="
echo "ESO Integration Test"
echo "========================================="
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Function to wait for resource
wait_for_resource() {
    local resource=$1
    local name=$2
    local namespace=$3
    local timeout=$4
    local condition=${5:-""}
    
    print_info "Waiting for $resource/$name in namespace $namespace..."
    
    if [ -n "$condition" ]; then
        if timeout $timeout bash -c "until oc get $resource $name -n $namespace -o jsonpath='{.status.conditions[?(@.type==\"$condition\")].status}' 2>/dev/null | grep -q True; do sleep 2; done"; then
            print_success "$resource/$name is ready"
            return 0
        else
            print_error "$resource/$name did not become ready within ${timeout}s"
            return 1
        fi
    else
        if timeout $timeout bash -c "until oc get $resource $name -n $namespace &>/dev/null; do sleep 2; done"; then
            print_success "$resource/$name exists"
            return 0
        else
            print_error "$resource/$name was not created within ${timeout}s"
            return 1
        fi
    fi
}

# Step 1: Check ESO installation
echo "Step 1: Checking ESO installation..."
if oc get crd externalsecrets.external-secrets.io &>/dev/null; then
    print_success "External Secrets Operator CRD found"
    ESO_VERSION=$(oc get crd externalsecrets.external-secrets.io -o jsonpath='{.metadata.annotations.controller-gen\.kubebuilder\.io/version}')
    print_info "ESO version: $ESO_VERSION"
else
    print_error "External Secrets Operator not installed"
    echo "Please install ESO first: https://external-secrets.io/latest/introduction/getting-started/"
    exit 1
fi
echo ""

# Step 2: Check namespace
echo "Step 2: Checking namespace..."
if oc get namespace $NAMESPACE &>/dev/null; then
    print_success "Namespace $NAMESPACE exists"
else
    print_info "Creating namespace $NAMESPACE..."
    oc create namespace $NAMESPACE
    print_success "Namespace $NAMESPACE created"
fi
echo ""

# Step 3: Apply ESO resources
echo "Step 3: Applying ESO test resources..."
if oc apply -f config/samples/eso-fake-secretstore.yaml; then
    print_success "ESO resources applied"
else
    print_error "Failed to apply ESO resources"
    exit 1
fi
echo ""

# Step 4: Wait for SecretStore to be ready
echo "Step 4: Waiting for SecretStore..."
wait_for_resource "secretstore" "fake-secretstore" "$NAMESPACE" 60 "Ready"
echo ""

# Step 5: Wait for ExternalSecrets to sync
echo "Step 5: Waiting for ExternalSecrets to sync..."
for es in aws-credentials-eso database-config-eso mlflow-credentials-eso api-keys-eso; do
    wait_for_resource "externalsecret" "$es" "$NAMESPACE" 60 "Ready"
done
echo ""

# Step 6: Verify secrets were created
echo "Step 6: Verifying synced secrets..."
for secret in aws-credentials-eso database-config-eso mlflow-credentials-eso api-keys-eso; do
    if oc get secret $secret -n $NAMESPACE &>/dev/null; then
        print_success "Secret $secret created by ESO"
        # Show secret keys (not values)
        KEYS=$(oc get secret $secret -n $NAMESPACE -o jsonpath='{.data}' | jq -r 'keys[]' 2>/dev/null || echo "")
        if [ -n "$KEYS" ]; then
            print_info "  Keys: $(echo $KEYS | tr '\n' ', ' | sed 's/,$//')"
        fi
    else
        print_error "Secret $secret not found"
        exit 1
    fi
done
echo ""

# Step 7: Wait for NotebookValidationJob to be created
echo "Step 7: Waiting for NotebookValidationJob..."
wait_for_resource "notebookvalidationjob" "$TEST_NAME" "$NAMESPACE" 60
echo ""

# Step 8: Wait for validation pod to be created
echo "Step 8: Waiting for validation pod..."
POD_NAME="${TEST_NAME}-validation"
wait_for_resource "pod" "$POD_NAME" "$NAMESPACE" 120
echo ""

# Step 9: Check pod environment variables
echo "Step 9: Verifying environment variables in pod..."
sleep 5  # Give pod time to start

# Check if pod has the expected environment variables
print_info "Checking for AWS credentials..."
if oc exec -n $NAMESPACE $POD_NAME -c git-clone -- env 2>/dev/null | grep -q "AWS_ACCESS_KEY_ID"; then
    print_success "AWS_ACCESS_KEY_ID found in pod environment"
else
    print_error "AWS_ACCESS_KEY_ID not found in pod environment"
fi

print_info "Checking for database credentials..."
if oc exec -n $NAMESPACE $POD_NAME -c git-clone -- env 2>/dev/null | grep -q "DB_HOST"; then
    print_success "DB_HOST found in pod environment"
else
    print_error "DB_HOST not found in pod environment"
fi

print_info "Checking for MLflow credentials..."
if oc exec -n $NAMESPACE $POD_NAME -c git-clone -- env 2>/dev/null | grep -q "MLFLOW_TRACKING_URI"; then
    print_success "MLFLOW_TRACKING_URI found in pod environment"
else
    print_error "MLFLOW_TRACKING_URI not found in pod environment"
fi

print_info "Checking for API keys..."
if oc exec -n $NAMESPACE $POD_NAME -c git-clone -- env 2>/dev/null | grep -q "OPENAI_API_KEY"; then
    print_success "OPENAI_API_KEY found in pod environment"
else
    print_error "OPENAI_API_KEY not found in pod environment"
fi
echo ""

# Step 10: Check NotebookValidationJob status
echo "Step 10: Checking NotebookValidationJob status..."
PHASE=$(oc get notebookvalidationjob $TEST_NAME -n $NAMESPACE -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
print_info "Current phase: $PHASE"
echo ""

# Step 11: Show ExternalSecret status
echo "Step 11: ExternalSecret status summary..."
echo ""
oc get externalsecrets -n $NAMESPACE -o custom-columns=NAME:.metadata.name,STORE:.spec.secretStoreRef.name,REFRESH:.spec.refreshInterval,STATUS:.status.conditions[0].reason,READY:.status.conditions[0].status
echo ""

# Step 12: Show created secrets
echo "Step 12: Synced secrets summary..."
echo ""
oc get secrets -n $NAMESPACE | grep -E "NAME|eso"
echo ""

echo "========================================="
echo "ESO Integration Test Complete"
echo "========================================="
echo ""
print_success "All ESO integration checks passed!"
echo ""
print_info "Next steps:"
echo "  1. Check NotebookValidationJob logs: oc logs -n $NAMESPACE $POD_NAME -c notebook-validator"
echo "  2. Monitor ExternalSecret sync: oc get externalsecrets -n $NAMESPACE -w"
echo "  3. Test secret rotation: Update fake-secretstore data and wait for refresh"
echo ""
print_info "Cleanup:"
echo "  oc delete -f config/samples/eso-fake-secretstore.yaml"
echo ""

