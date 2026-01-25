#!/bin/bash
set -e

# test-local-bundles.sh - Test local bundles from community-operators directory
# This builds bundle images from /home/lab-user/community-operators and tests them in Kind

VERSION="${1:-1.0.7}"
CLUSTER_NAME="local-bundle-test-${VERSION}"
COMMUNITY_OPS_DIR="/home/lab-user/community-operators"
OPERATOR_DIR="${COMMUNITY_OPS_DIR}/operators/jupyter-notebook-validator-operator"
BUNDLE_DIR="${OPERATOR_DIR}/${VERSION}"
LOCAL_REGISTRY="localhost:5000"
BUNDLE_IMAGE="${LOCAL_REGISTRY}/jupyter-notebook-validator-operator-bundle:${VERSION}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

echo "========================================"
log_info "Local Bundle Test - Version ${VERSION}"
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
log_info "Bundle image: ${BUNDLE_IMAGE}"
echo ""

# Check if cluster exists
if ! sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_info "Creating Kind cluster: ${CLUSTER_NAME} (Kubernetes 1.31, Podman rootful)"
    
    # Create cluster with local registry
    cat <<EOF | sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind create cluster --name "${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5000"]
    endpoint = ["http://kind-registry:5000"]
EOF
    
    # Create local registry container
    log_info "Creating local registry..."
    if ! sudo podman ps -a --format '{{.Names}}' | grep -q '^kind-registry$'; then
        sudo podman run -d --restart=always -p "5000:5000" --name kind-registry registry:2
    fi
    
    # Connect registry to cluster network
    log_info "Connecting registry to cluster network..."
    sudo podman network connect "kind" kind-registry 2>/dev/null || true
    
    log_info "Installing OLM using operator-sdk..."
    # Use operator-sdk to install OLM (handles large CRDs better)
    if ! command -v operator-sdk &> /dev/null; then
        log_error "operator-sdk not found. Installing..."
        ARCH=$(uname -m | sed 's/x86_64/amd64/')
        OS=$(uname | tr '[:upper:]' '[:lower:]')
        curl -sL "https://github.com/operator-framework/operator-sdk/releases/download/v1.37.0/operator-sdk_${OS}_${ARCH}" -o /tmp/operator-sdk
        chmod +x /tmp/operator-sdk
        sudo mv /tmp/operator-sdk /usr/local/bin/operator-sdk
    fi

    # Install OLM using operator-sdk
    sudo operator-sdk olm install --version v0.28.0 --timeout 5m

    log_success "OLM installed"
else
    log_info "Using existing cluster: ${CLUSTER_NAME}"
fi

# Build bundle image from local directory
log_info "Building bundle image from local directory..."
cd "${BUNDLE_DIR}"

# Create Dockerfile for bundle
cat > Dockerfile.bundle <<EOF
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

log_info "Building with Podman..."
sudo podman build -f Dockerfile.bundle -t "${BUNDLE_IMAGE}" .

log_info "Pushing to local registry..."
sudo podman push "${BUNDLE_IMAGE}"

log_success "Bundle image built and pushed: ${BUNDLE_IMAGE}"
echo ""

# Create operators namespace
log_info "Creating operators namespace..."
sudo kubectl --context "kind-${CLUSTER_NAME}" create namespace operators --dry-run=client -o yaml | sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f -

# Create OperatorGroup
log_info "Creating OperatorGroup..."
cat <<EOF | sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: jupyter-validator-operatorgroup
  namespace: operators
spec:
  targetNamespaces:
    - operators
EOF

# Create CatalogSource
log_info "Creating CatalogSource for LOCAL version ${VERSION}..."
cat <<EOF | sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: jupyter-validator-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: ${BUNDLE_IMAGE}
  displayName: Jupyter Notebook Validator Operator (Local)
  publisher: Decision Crafters
EOF

# Wait for CatalogSource
log_info "Waiting for CatalogSource to be ready..."
timeout=120
elapsed=0
while [ $elapsed -lt $timeout ]; do
    state=$(sudo kubectl --context "kind-${CLUSTER_NAME}" get catalogsource jupyter-validator-catalog -n olm -o jsonpath='{.status.connectionState.lastObservedState}' 2>/dev/null || echo "")
    if [ "$state" = "READY" ]; then
        log_success "CatalogSource is ready"
        break
    fi
    sleep 5
    elapsed=$((elapsed + 5))
done

if [ $elapsed -ge $timeout ]; then
    log_error "CatalogSource failed to become ready"
    sudo kubectl --context "kind-${CLUSTER_NAME}" get catalogsource jupyter-validator-catalog -n olm -o yaml
    sudo kubectl --context "kind-${CLUSTER_NAME}" get pods -n olm
    exit 1
fi

# Create Subscription
log_info "Creating Subscription for LOCAL version ${VERSION}..."
cat <<EOF | sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: operators
spec:
  channel: alpha
  name: jupyter-notebook-validator-operator
  source: jupyter-validator-catalog
  sourceNamespace: olm
  installPlanApproval: Automatic
  startingCSV: jupyter-notebook-validator-operator.v${VERSION}
EOF

# Wait for CSV installation
log_info "Waiting for operator installation..."
timeout=300
elapsed=0
while [ $elapsed -lt $timeout ]; do
    phase=$(sudo kubectl --context "kind-${CLUSTER_NAME}" get csv "jupyter-notebook-validator-operator.v${VERSION}" -n operators -o jsonpath='{.status.phase}' 2>/dev/null || echo "")

    case "$phase" in
        "Succeeded")
            log_success "✅ Operator installed successfully (LOCAL version ${VERSION})"

            # Check webhook annotation
            webhook_annotation=$(sudo kubectl --context "kind-${CLUSTER_NAME}" get csv "jupyter-notebook-validator-operator.v${VERSION}" -n operators -o jsonpath='{.metadata.annotations.operatorframework\.io/inject-webhook-certs}' 2>/dev/null || echo "")
            if [ "$webhook_annotation" = "true" ]; then
                log_success "  ✅ Webhook certificate injection annotation present"
            else
                log_error "  ❌ Webhook certificate injection annotation missing"
            fi

            # Check operator pod
            log_info "Checking operator pod..."
            sudo kubectl --context "kind-${CLUSTER_NAME}" get pods -n operators -l control-plane=controller-manager

            # Show CSV details
            log_info "CSV details:"
            sudo kubectl --context "kind-${CLUSTER_NAME}" get csv "jupyter-notebook-validator-operator.v${VERSION}" -n operators -o jsonpath='{.spec.replaces}' && echo "" || echo "  (no replaces field)"

            echo ""
            log_success "🎉 LOCAL Version ${VERSION} test PASSED!"
            echo ""
            log_info "Bundle tested from: ${BUNDLE_DIR}"
            exit 0
            ;;
        "Failed")
            log_error "❌ Operator installation failed"
            sudo kubectl --context "kind-${CLUSTER_NAME}" get csv "jupyter-notebook-validator-operator.v${VERSION}" -n operators -o yaml
            sudo kubectl --context "kind-${CLUSTER_NAME}" get pods -n operators
            exit 1
            ;;
        "")
            log_info "  Waiting for CSV... (${elapsed}s/${timeout}s)"
            ;;
        *)
            log_info "  CSV phase: ${phase} (${elapsed}s/${timeout}s)"
            ;;
    esac

    sleep 10
    elapsed=$((elapsed + 10))
done

log_error "❌ Operator installation timeout"
sudo kubectl --context "kind-${CLUSTER_NAME}" get csv -n operators
sudo kubectl --context "kind-${CLUSTER_NAME}" get subscription -n operators
sudo kubectl --context "kind-${CLUSTER_NAME}" get installplan -n operators
sudo kubectl --context "kind-${CLUSTER_NAME}" get pods -n olm
exit 1

