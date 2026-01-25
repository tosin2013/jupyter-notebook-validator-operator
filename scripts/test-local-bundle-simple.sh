#!/bin/bash
set -e

# test-local-bundle-simple.sh - Test local bundles using operator-sdk run bundle
# This is the recommended approach for testing OLM bundles locally

VERSION="${1:-1.0.7}"
CLUSTER_NAME="bundle-test-${VERSION}"
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
log_info "Local Bundle Test - Version ${VERSION}"
log_info "Using operator-sdk run bundle"
echo "========================================"
echo ""

# Validate bundle directory exists
if [ ! -d "${BUNDLE_DIR}" ]; then
    log_error "Bundle directory not found: ${BUNDLE_DIR}"
    log_info "Available versions:"
    ls -1 "${OPERATOR_DIR}" 2>/dev/null || echo "  (none)"
    exit 1
fi

log_info "Bundle directory: ${BUNDLE_DIR}"
echo ""

# Check if cluster exists
if ! sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_info "Creating Kind cluster: ${CLUSTER_NAME} (Kubernetes 1.31)"
    sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind create cluster \
        --name "${CLUSTER_NAME}" \
        --image kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865 \
        --wait 60s
    
    log_success "Cluster created"
else
    log_info "Using existing cluster: ${CLUSTER_NAME}"
fi

# Set kubectl context
export KUBECONFIG="/root/.kube/config"
sudo kubectl config use-context "kind-${CLUSTER_NAME}"

# Install OLM using operator-sdk (if not already installed)
if sudo kubectl get deployment -n olm olm-operator &>/dev/null; then
    log_info "OLM is already installed, skipping installation"
else
    log_info "Installing OLM..."
    sudo /usr/local/bin/operator-sdk olm install --version v0.28.0 --timeout 5m
    log_success "OLM installed successfully"
fi
echo ""

# Build bundle image locally
log_info "Building bundle image from local directory..."
cd "${BUNDLE_DIR}"

# Use a local tag
BUNDLE_IMAGE="localhost/jupyter-notebook-validator-operator-bundle:${VERSION}"

# Build with Podman
sudo podman build -f - -t "${BUNDLE_IMAGE}" . <<'EOF'
FROM scratch
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=jupyter-notebook-validator-operator
LABEL operators.operatorframework.io.bundle.channels.v1=alpha
LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha
COPY manifests /manifests/
COPY metadata /metadata/
EOF

log_success "Bundle image built: ${BUNDLE_IMAGE}"
echo ""

# Load bundle image into Kind cluster
log_info "Loading bundle image into Kind cluster..."

# Tag the image for the Kind cluster's local registry
KIND_IMAGE="localhost/jupyter-notebook-validator-operator-bundle:1.0.7"

# Get the Kind container name
KIND_CONTAINER="bundle-test-${VERSION}-control-plane"

# Load image directly into Kind node using podman save and docker load
sudo podman save "${BUNDLE_IMAGE}" | sudo podman exec -i "${KIND_CONTAINER}" ctr -n k8s.io images import -

log_success "Bundle image loaded into cluster"
echo ""

# Run bundle using operator-sdk
log_info "Deploying operator using operator-sdk run bundle..."
log_info "This will create CatalogSource, Subscription, and install the operator"
echo ""

# Use operator-sdk run bundle with the local image
sudo /usr/local/bin/operator-sdk run bundle "${BUNDLE_IMAGE}" \
    --namespace operators \
    --timeout 10m \
    --kubeconfig /root/.kube/config

log_success "✅ Operator deployed successfully!"
echo ""

# Verify installation
log_info "Verifying installation..."
sudo kubectl get csv -n operators
sudo kubectl get pods -n operators

echo ""
log_success "🎉 LOCAL Bundle ${VERSION} test PASSED!"
echo ""
log_info "Bundle tested from: ${BUNDLE_DIR}"
log_info "Cluster: ${CLUSTER_NAME}"
echo ""
log_info "To cleanup: sudo /usr/local/bin/operator-sdk cleanup jupyter-notebook-validator-operator --namespace operators"
log_info "To delete cluster: sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind delete cluster --name ${CLUSTER_NAME}"

