#!/bin/bash
set -e

# Script to test volume functionality on validation cluster
# Cluster: https://console-openshift-console.apps.cluster-hh8nc.hh8nc.sandbox5424.opentlc.com

# Configuration
VALIDATION_CLUSTER_API="https://api.cluster-hh8nc.hh8nc.sandbox5424.opentlc.com:6443"
VALIDATION_CLUSTER_TOKEN="sha256~sG2tYISqb1DjJzskmYHg93Hb5AUq29XgPTOw3H6I6ik"
TEST_NAMESPACE="${TEST_NAMESPACE:-jupyter-validator-volume-test}"
OPERATOR_IMAGE="${OPERATOR_IMAGE:-quay.io/takinosh/jupyter-notebook-validator-operator:1.0.4-ocp4.18}"
TEST_REPO_URL="https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
TEST_REPO_BRANCH="main"
TIMEOUT_SECONDS=600  # 10 minutes for build tests

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

wait_for_condition() {
    local resource=$1
    local condition=$2
    local timeout=$3
    local namespace=$4
    
    log_info "Waiting for $resource to be $condition (timeout: ${timeout}s)..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if oc get $resource -n $namespace &>/dev/null; then
            local status=$(oc get $resource -n $namespace -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
            log_info "Current status: $status"
            
            if [[ "$status" == "$condition" ]]; then
                log_success "$resource is $condition"
                return 0
            fi
        fi
        
        sleep 5
        elapsed=$((elapsed + 5))
    done
    
    log_error "Timeout waiting for $resource to be $condition"
    return 1
}

# Phase 1: Login to validation cluster
log_info "=== Phase 1: Logging into validation cluster ==="
oc login --token=$VALIDATION_CLUSTER_TOKEN --server=$VALIDATION_CLUSTER_API --insecure-skip-tls-verify=true

# Verify cluster version
CLUSTER_VERSION=$(oc version -o json | jq -r '.openshiftVersion')
log_success "Connected to OpenShift $CLUSTER_VERSION"

# Phase 2: Create test namespace
log_info "=== Phase 2: Creating test namespace ==="
oc new-project $TEST_NAMESPACE 2>/dev/null || oc project $TEST_NAMESPACE
log_success "Using namespace: $TEST_NAMESPACE"

# Check storage classes
log_info "Available storage classes:"
oc get storageclass

# Phase 3: Deploy operator
log_info "=== Phase 3: Deploying operator ==="
log_info "Installing CRDs..."
make install

log_info "Deploying operator with image: $OPERATOR_IMAGE"
make deploy IMG=$OPERATOR_IMAGE

# Wait for operator to be ready
log_info "Waiting for operator pod to be ready..."
sleep 10
oc wait --for=condition=ready pod -l control-plane=controller-manager \
    -n jupyter-notebook-validator-operator-system --timeout=120s || {
    log_error "Operator pod not ready"
    oc get pods -n jupyter-notebook-validator-operator-system
    oc logs -n jupyter-notebook-validator-operator-system -l control-plane=controller-manager --tail=50
    exit 1
}
log_success "Operator is ready"

# Phase 4: Test basic volume (no build)
log_info "=== Phase 4: Testing basic notebook (no volume needed) ==="
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: volume-test-basic
  namespace: $TEST_NAMESPACE
spec:
  notebook:
    git:
      url: $TEST_REPO_URL
      ref: $TEST_REPO_BRANCH
      path: notebooks/tier1-simple/01-hello-world.ipynb
  podConfig:
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"
EOF

log_info "Waiting for basic test to complete..."
sleep 5

# Check job status
oc get notebookvalidationjob volume-test-basic -n $TEST_NAMESPACE -o yaml

# Wait for completion
wait_for_condition "notebookvalidationjob/volume-test-basic" "Succeeded" 300 $TEST_NAMESPACE || {
    log_error "Basic test failed"
    oc describe notebookvalidationjob volume-test-basic -n $TEST_NAMESPACE
    POD_NAME=$(oc get pods -n $TEST_NAMESPACE -l job-name=volume-test-basic -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POD_NAME" ]; then
        log_info "Pod logs:"
        oc logs $POD_NAME -n $TEST_NAMESPACE
    fi
    exit 1
}

log_success "Basic test completed successfully"

# Check that no PVC was created (simple notebook doesn't need build)
PVC_COUNT=$(oc get pvc -n $TEST_NAMESPACE --no-headers 2>/dev/null | wc -l)
if [ "$PVC_COUNT" -eq 0 ]; then
    log_success "No PVC created (expected for simple notebook)"
else
    log_warning "Unexpected PVC created: $PVC_COUNT PVCs found"
    oc get pvc -n $TEST_NAMESPACE
fi

# Phase 5: Test Tekton build with volumes (tier4-s2i has requirements.txt)
log_info "=== Phase 5: Testing Tekton build with requirements.txt (requires PVC for workspace) ==="
cat <<EOF | oc apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: volume-test-tekton-s2i
  namespace: $TEST_NAMESPACE
spec:
  notebook:
    git:
      url: $TEST_REPO_URL
      ref: $TEST_REPO_BRANCH
      path: notebooks/tier4-s2i/01-s2i-python-build.ipynb
  buildConfig:
    strategy: tekton
    timeout: 15m
  podConfig:
    resources:
      requests:
        memory: "1Gi"
        cpu: "1000m"
      limits:
        memory: "2Gi"
        cpu: "2000m"
EOF

log_info "Waiting for Tekton build to start..."
sleep 10

# Check for Pipeline creation
log_info "Checking for Pipeline creation..."
oc get pipeline -n $TEST_NAMESPACE

# Check for PipelineRun
log_info "Checking for PipelineRun..."
oc get pipelinerun -n $TEST_NAMESPACE

# Check for PVC creation (Tekton workspace)
log_info "Checking for PVC creation..."
sleep 5
PVC_COUNT=$(oc get pvc -n $TEST_NAMESPACE --no-headers 2>/dev/null | wc -l)
if [ "$PVC_COUNT" -gt 0 ]; then
    log_success "PVC created for Tekton workspace: $PVC_COUNT PVC(s)"
    oc get pvc -n $TEST_NAMESPACE

    # Describe PVC to check status
    PVC_NAME=$(oc get pvc -n $TEST_NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    log_info "PVC details:"
    oc describe pvc $PVC_NAME -n $TEST_NAMESPACE

    # Check PVC status
    PVC_STATUS=$(oc get pvc $PVC_NAME -n $TEST_NAMESPACE -o jsonpath='{.status.phase}')
    if [ "$PVC_STATUS" == "Bound" ]; then
        log_success "PVC is Bound"
    else
        log_error "PVC is not Bound: $PVC_STATUS"
        oc get events -n $TEST_NAMESPACE --field-selector involvedObject.kind=PersistentVolumeClaim
        exit 1
    fi
else
    log_error "No PVC created for Tekton build"
    exit 1
fi

# Wait for Tekton build to complete
log_info "Waiting for Tekton build to complete (timeout: ${TIMEOUT_SECONDS}s)..."
wait_for_condition "notebookvalidationjob/volume-test-tekton-s2i" "Succeeded" $TIMEOUT_SECONDS $TEST_NAMESPACE || {
    log_error "Tekton build test failed"

    # Collect debugging information
    log_info "=== Debugging Information ==="

    log_info "NotebookValidationJob status:"
    oc describe notebookvalidationjob volume-test-tekton-s2i -n $TEST_NAMESPACE

    log_info "PipelineRun status:"
    oc get pipelinerun -n $TEST_NAMESPACE -o yaml

    log_info "TaskRun status:"
    oc get taskrun -n $TEST_NAMESPACE

    log_info "Pods:"
    oc get pods -n $TEST_NAMESPACE

    # Get logs from TaskRun pods
    TASKRUN_PODS=$(oc get pods -n $TEST_NAMESPACE -l tekton.dev/task -o jsonpath='{.items[*].metadata.name}')
    for pod in $TASKRUN_PODS; do
        log_info "Logs from $pod:"
        oc logs $pod -n $TEST_NAMESPACE --all-containers || true
    done

    log_info "PVC status:"
    oc get pvc -n $TEST_NAMESPACE
    oc describe pvc -n $TEST_NAMESPACE

    log_info "Events:"
    oc get events -n $TEST_NAMESPACE --sort-by='.lastTimestamp'

    exit 1
}

log_success "Tekton build test completed successfully"

# Phase 6: Test tier5-volumes (ML training with volumes)
log_info "=== Phase 6: Testing tier5-volumes (ML training pipeline with volumes) ==="
cat <<EOF | oc apply -f -
apiVersion: mlops.redhat.com/v1alpha1
kind: NotebookValidationJob
metadata:
  name: volume-test-ml-training
  namespace: $TEST_NAMESPACE
spec:
  notebook:
    git:
      url: $TEST_REPO_URL
      ref: $TEST_REPO_BRANCH
      path: notebooks/tier5-volumes/01-ml-training-pipeline-volumes.ipynb
  buildConfig:
    strategy: tekton
    timeout: 15m
  podConfig:
    resources:
      requests:
        memory: "2Gi"
        cpu: "1000m"
      limits:
        memory: "4Gi"
        cpu: "2000m"
EOF

log_info "Waiting for ML training volume test to complete..."
wait_for_condition "notebookvalidationjob/volume-test-ml-training" "Succeeded" $TIMEOUT_SECONDS $TEST_NAMESPACE || {
    log_error "ML training volume test failed"
    oc describe notebookvalidationjob volume-test-ml-training -n $TEST_NAMESPACE
    exit 1
}

log_success "ML training volume test completed successfully"

# Phase 7: Verify volume functionality
log_info "=== Phase 7: Verifying volume functionality ==="

# Check final PVC status
log_info "Final PVC status:"
oc get pvc -n $TEST_NAMESPACE

# Check if validation pod ran
VALIDATION_POD=$(oc get pods -n $TEST_NAMESPACE -l job-name=volume-test-tekton-s2i -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$VALIDATION_POD" ]; then
    log_success "Validation pod created: $VALIDATION_POD"
    log_info "Validation pod logs:"
    oc logs $VALIDATION_POD -n $TEST_NAMESPACE || true
else
    log_warning "No validation pod found"
fi

# Summary
log_info "=== Test Summary ==="
log_success "✅ Phase 1: Cluster access - PASSED"
log_success "✅ Phase 2: Namespace creation - PASSED"
log_success "✅ Phase 3: Operator deployment - PASSED"
log_success "✅ Phase 4: Basic notebook test (tier1) - PASSED"
log_success "✅ Phase 5: Tekton build with requirements.txt (tier4-s2i) - PASSED"
log_success "✅ Phase 6: ML training with volumes (tier5-volumes) - PASSED"
log_success "✅ Phase 7: Volume functionality verification - PASSED"

log_success "All volume tests passed on validation cluster!"

# Show all test results
log_info ""
log_info "=== All NotebookValidationJobs ==="
oc get notebookvalidationjob -n $TEST_NAMESPACE

log_info ""
log_info "=== All PVCs Created ==="
oc get pvc -n $TEST_NAMESPACE

# Cleanup prompt
log_info ""
log_info "To cleanup test resources, run:"
log_info "  oc delete project $TEST_NAMESPACE"
log_info "  make undeploy"

