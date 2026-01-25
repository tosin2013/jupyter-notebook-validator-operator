#!/bin/bash
set -e

# quick-olm-test.sh - Quick OLM test using published bundle images
# Tests version 1.0.7 first, then optionally 1.0.8 and 1.0.9

VERSION="${1:-1.0.7}"
CLUSTER_NAME="olm-test-${VERSION}"
BUNDLE_REGISTRY="quay.io/takinosh"
BUNDLE_NAME="jupyter-notebook-validator-operator-bundle"

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
log_info "Quick OLM Test - Version ${VERSION}"
echo "========================================"
echo ""

# Check if cluster exists
if ! sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_info "Creating Kind cluster: ${CLUSTER_NAME} (Kubernetes 1.31, Podman rootful)"
    sudo KIND_EXPERIMENTAL_PROVIDER=podman /usr/local/bin/kind create cluster \
        --name "${CLUSTER_NAME}" \
        --image kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865 \
        --wait 60s
    
    log_info "Installing OLM..."
    sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/crds.yaml
    sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/olm.yaml

    log_info "Waiting for OLM..."
    sudo kubectl --context "kind-${CLUSTER_NAME}" wait --for=condition=Available --timeout=300s -n olm deployment/catalog-operator deployment/olm-operator
    log_success "OLM installed"
else
    log_info "Using existing cluster: ${CLUSTER_NAME}"
fi

# Create operators namespace
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
log_info "Creating CatalogSource for version ${VERSION}..."
cat <<EOF | sudo kubectl --context "kind-${CLUSTER_NAME}" apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: jupyter-validator-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: ${BUNDLE_REGISTRY}/${BUNDLE_NAME}:${VERSION}
  displayName: Jupyter Notebook Validator Operator
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
    exit 1
fi

# Create Subscription
log_info "Creating Subscription for version ${VERSION}..."
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
            log_success "✅ Operator installed successfully (version ${VERSION})"

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

            echo ""
            log_success "🎉 Version ${VERSION} test PASSED!"
            exit 0
            ;;
        "Failed")
            log_error "❌ Operator installation failed"
            sudo kubectl --context "kind-${CLUSTER_NAME}" get csv "jupyter-notebook-validator-operator.v${VERSION}" -n operators -o yaml
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
exit 1

