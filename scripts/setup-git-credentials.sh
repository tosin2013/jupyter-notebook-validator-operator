#!/bin/bash
set -e

# Setup Git Credentials for Private Repository Access
# Based on ADR-009: Secret Management and Git Credentials
# Based on ADR-016: External Secrets Operator Integration
# Based on ADR-031: Tekton Build Strategy

NAMESPACE="${1:-e2e-tests}"
GITHUB_USERNAME="${GITHUB_USERNAME:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

print_header "Git Credentials Setup for Private Repository"

# Check if credentials are provided
if [ -z "$GITHUB_USERNAME" ] || [ -z "$GITHUB_TOKEN" ]; then
    print_error "GitHub credentials not provided!"
    echo ""
    echo "Usage:"
    echo "  export GITHUB_USERNAME=your-github-username"
    echo "  export GITHUB_TOKEN=ghp_your_personal_access_token"
    echo "  ./scripts/setup-git-credentials.sh [namespace]"
    echo ""
    echo "To create a GitHub Personal Access Token:"
    echo "  1. Go to: https://github.com/settings/tokens"
    echo "  2. Click 'Generate new token (classic)'"
    echo "  3. Give it 'repo' scope for private repositories"
    echo "  4. Copy the token (starts with 'ghp_')"
    echo ""
    exit 1
fi

print_info "Namespace: $NAMESPACE"
print_info "GitHub Username: $GITHUB_USERNAME"
print_info "GitHub Token: ${GITHUB_TOKEN:0:10}... (masked)"

# Check if namespace exists
if ! oc get namespace "$NAMESPACE" &> /dev/null; then
    print_warning "Namespace does not exist, creating: $NAMESPACE"
    oc create namespace "$NAMESPACE"
fi

print_header "Creating Git Credentials Secrets"

# 1. Create secret for S2I builds (BuildConfig format)
print_info "Creating git-credentials secret for S2I builds..."
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: git-credentials
    mlops.dev/credential-type: s2i-basic-auth
type: kubernetes.io/basic-auth
stringData:
  username: "${GITHUB_USERNAME}"
  password: "${GITHUB_TOKEN}"
EOF

if [ $? -eq 0 ]; then
    print_success "S2I git-credentials secret created"
else
    print_error "Failed to create S2I git-credentials secret"
    exit 1
fi

# 2. Create secret for Tekton builds (basic-auth workspace format)
print_info "Creating git-credentials-tekton secret for Tekton builds..."
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials-tekton
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: git-credentials
    mlops.dev/credential-type: tekton-basic-auth
type: Opaque
stringData:
  .gitconfig: |
    [credential]
        helper = store
    [credential "https://github.com"]
        helper = store
  .git-credentials: |
    https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com
EOF

if [ $? -eq 0 ]; then
    print_success "Tekton git-credentials-tekton secret created"
else
    print_error "Failed to create Tekton git-credentials-tekton secret"
    exit 1
fi

print_header "Verification"

# Verify secrets were created
print_info "Verifying secrets..."
if oc get secret git-credentials -n "$NAMESPACE" &> /dev/null; then
    print_success "git-credentials secret exists"
else
    print_error "git-credentials secret not found"
    exit 1
fi

if oc get secret git-credentials-tekton -n "$NAMESPACE" &> /dev/null; then
    print_success "git-credentials-tekton secret exists"
else
    print_error "git-credentials-tekton secret not found"
    exit 1
fi

print_header "Summary"
print_success "Git credentials configured successfully!"
echo ""
echo "Secrets created in namespace: $NAMESPACE"
echo "  - git-credentials (for S2I builds)"
echo "  - git-credentials-tekton (for Tekton builds)"
echo ""
echo "Usage in NotebookValidationJob:"
echo ""
echo "  spec:"
echo "    notebook:"
echo "      git:"
echo "        url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
echo "        ref: main"
echo "        credentialsSecret: git-credentials  # For S2I builds"
echo "    podConfig:"
echo "      buildConfig:"
echo "        enabled: true"
echo "        strategy: s2i  # or tekton"
echo ""
echo "Note: For Tekton builds, the operator automatically uses git-credentials-tekton"
echo ""

