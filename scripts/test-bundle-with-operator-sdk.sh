#!/bin/bash
set -e

# test-bundle-with-operator-sdk.sh - Test local bundles using operator-sdk run bundle
# This is the OFFICIAL recommended approach from OLM documentation

VERSION="${1:-1.0.7}"
CLUSTER_NAME="sdk-test-${VERSION}"
COMMUNITY_OPS_DIR="/home/lab-user/community-operators"
OPERATOR_DIR="${COMMUNITY_OPS_DIR}/operators/jupyter-notebook-validator-operator"
BUNDLE_DIR="${OPERATOR_DIR}/${VERSION}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "========================================"
log_info "Bundle Test - Version ${VERSION}"
log_info "Using operator-sdk run bundle (OFFICIAL METHOD)"
echo "========================================"
echo ""

# Validate bundle directory exists
if [ ! -d "${BUNDLE_DIR}" ]; then
    log_error "Bundle directory not found: ${BUNDLE_DIR}"
    exit 1
fi

log_info "Bundle directory: ${BUNDLE_DIR}"
echo ""

# Step 1: Validate the bundle
log_info "Step 1: Validating bundle with operator-sdk..."
cd "${BUNDLE_DIR}"
operator-sdk bundle validate . --select-optional suite=operatorframework --optional-values=k8s-version=1.31

log_success "✅ Bundle validation passed!"
echo ""

# Step 2: Create Kind cluster if needed
if ! sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_info "Step 2: Creating Kind cluster: ${CLUSTER_NAME}"
    sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind create cluster \
        --name "${CLUSTER_NAME}" \
        --image kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865 \
        --wait 60s
    log_success "Cluster created"
else
    log_info "Step 2: Using existing cluster: ${CLUSTER_NAME}"
fi

# Set kubectl context
export KUBECONFIG="/root/.kube/config"
sudo kubectl config use-context "kind-${CLUSTER_NAME}"
echo ""

# Step 3: Install OLM
if sudo kubectl get deployment -n olm olm-operator &>/dev/null; then
    log_info "Step 3: OLM already installed"
else
    log_info "Step 3: Installing OLM..."
    sudo /usr/local/bin/operator-sdk olm install --version v0.28.0 --timeout 5m
    log_success "OLM installed"
fi
echo ""

# Step 4: Build and run bundle using operator-sdk
log_info "Step 4: Building and deploying bundle with operator-sdk run bundle..."
log_info "This will:"
log_info "  - Build the bundle image"
log_info "  - Push to local registry"
log_info "  - Create CatalogSource"
log_info "  - Create Subscription"
log_info "  - Install the operator"
echo ""

# Use operator-sdk run bundle with local directory
# This handles all the complexity of building, loading, and deploying
sudo /usr/local/bin/operator-sdk run bundle-upgrade \
    "${BUNDLE_DIR}" \
    --timeout 10m \
    --kubeconfig /root/.kube/config \
    --namespace operators || \
sudo /usr/local/bin/operator-sdk run bundle \
    "${BUNDLE_DIR}" \
    --timeout 10m \
    --kubeconfig /root/.kube/config \
    --namespace operators

log_success "✅ Operator deployed successfully!"
echo ""

# Step 5: Verify installation
log_info "Step 5: Verifying installation..."
echo ""
log_info "ClusterServiceVersions:"
sudo kubectl get csv -n operators
echo ""
log_info "Operator Pods:"
sudo kubectl get pods -n operators -l control-plane=controller-manager
echo ""

log_success "🎉 Bundle ${VERSION} test PASSED!"
echo ""
log_info "Bundle tested from: ${BUNDLE_DIR}"
log_info "Cluster: ${CLUSTER_NAME}"
echo ""
log_info "To cleanup:"
log_info "  sudo /usr/local/bin/operator-sdk cleanup jupyter-notebook-validator-operator --namespace operators --kubeconfig /root/.kube/config"
log_info "To delete cluster:"
log_info "  sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind delete cluster --name ${CLUSTER_NAME}"

