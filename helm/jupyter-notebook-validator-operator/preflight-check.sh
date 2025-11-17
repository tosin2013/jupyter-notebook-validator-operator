#!/bin/bash
# Preflight check script for Jupyter Notebook Validator Operator Helm installation
# This script verifies all prerequisites are met before installation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

echo "=========================================="
echo "Jupyter Notebook Validator Operator"
echo "Pre-flight Check"
echo "=========================================="
echo ""

# Function to print status
print_status() {
    local status=$1
    local message=$2
    
    if [ "$status" == "PASS" ]; then
        echo -e "${GREEN}✓${NC} $message"
        ((PASSED++))
    elif [ "$status" == "FAIL" ]; then
        echo -e "${RED}✗${NC} $message"
        ((FAILED++))
    elif [ "$status" == "WARN" ]; then
        echo -e "${YELLOW}⚠${NC} $message"
        ((WARNINGS++))
    fi
}

# Check 1: kubectl/oc CLI
echo "Checking CLI tools..."
if command -v kubectl &> /dev/null; then
    print_status "PASS" "kubectl CLI found"
    CLI="kubectl"
elif command -v oc &> /dev/null; then
    print_status "PASS" "oc CLI found (OpenShift)"
    CLI="oc"
else
    print_status "FAIL" "kubectl or oc CLI not found"
    CLI=""
fi
echo ""

# Check 2: Cluster connectivity
if [ -n "$CLI" ]; then
    echo "Checking cluster connectivity..."
    if $CLI cluster-info &> /dev/null; then
        print_status "PASS" "Connected to Kubernetes/OpenShift cluster"
        
        # Get cluster version
        if [ "$CLI" == "oc" ]; then
            VERSION=$($CLI version -o json 2>/dev/null | grep -o '"openshiftVersion":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
            echo "  OpenShift version: $VERSION"
        else
            VERSION=$($CLI version --short 2>/dev/null | grep "Server Version" | cut -d' ' -f3 || echo "unknown")
            echo "  Kubernetes version: $VERSION"
        fi
    else
        print_status "FAIL" "Cannot connect to cluster"
    fi
    echo ""
fi

# Check 3: Helm
echo "Checking Helm..."
if command -v helm &> /dev/null; then
    HELM_VERSION=$(helm version --short 2>/dev/null | cut -d'v' -f2 | cut -d'+' -f1)
    print_status "PASS" "Helm found (version: $HELM_VERSION)"
    
    # Check Helm version >= 3.8
    REQUIRED_VERSION="3.8.0"
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$HELM_VERSION" | sort -V | head -n1)" == "$REQUIRED_VERSION" ]; then
        print_status "PASS" "Helm version >= 3.8.0"
    else
        print_status "WARN" "Helm version < 3.8.0 (recommended: 3.8+)"
    fi
else
    print_status "FAIL" "Helm not found"
fi
echo ""

# Check 4: cert-manager
if [ -n "$CLI" ]; then
    echo "Checking cert-manager..."
    if $CLI get namespace cert-manager &> /dev/null; then
        print_status "PASS" "cert-manager namespace exists"
        
        # Check cert-manager pods
        CERT_MANAGER_PODS=$($CLI get pods -n cert-manager --no-headers 2>/dev/null | wc -l)
        if [ "$CERT_MANAGER_PODS" -gt 0 ]; then
            RUNNING_PODS=$($CLI get pods -n cert-manager --no-headers 2>/dev/null | grep -c "Running" || echo "0")
            if [ "$RUNNING_PODS" -eq "$CERT_MANAGER_PODS" ]; then
                print_status "PASS" "cert-manager pods running ($RUNNING_PODS/$CERT_MANAGER_PODS)"
            else
                print_status "WARN" "Some cert-manager pods not running ($RUNNING_PODS/$CERT_MANAGER_PODS)"
            fi
        else
            print_status "FAIL" "No cert-manager pods found"
        fi
        
        # Check cert-manager CRDs
        if $CLI get crd certificates.cert-manager.io &> /dev/null; then
            print_status "PASS" "cert-manager CRDs installed"
        else
            print_status "FAIL" "cert-manager CRDs not found"
        fi
    else
        print_status "FAIL" "cert-manager not installed (REQUIRED)"
        echo "  Install with: kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml"
    fi
    echo ""
fi

# Check 5: Cluster admin permissions
if [ -n "$CLI" ]; then
    echo "Checking permissions..."
    if $CLI auth can-i create customresourcedefinitions --all-namespaces &> /dev/null; then
        print_status "PASS" "Can create CRDs (cluster admin)"
    else
        print_status "FAIL" "Cannot create CRDs (cluster admin required)"
    fi
    
    if $CLI auth can-i create namespace &> /dev/null; then
        print_status "PASS" "Can create namespaces"
    else
        print_status "WARN" "Cannot create namespaces (may need to create jupyter-validator-system manually)"
    fi
    echo ""
fi

# Check 6: Optional components
echo "Checking optional components..."

# Tekton
if [ -n "$CLI" ] && $CLI get namespace tekton-pipelines &> /dev/null; then
    print_status "PASS" "Tekton Pipelines installed (optional)"
else
    print_status "WARN" "Tekton Pipelines not found (optional - needed for build features)"
fi

# Prometheus Operator
if [ -n "$CLI" ] && $CLI get crd servicemonitors.monitoring.coreos.com &> /dev/null; then
    print_status "PASS" "Prometheus Operator installed (optional)"
else
    print_status "WARN" "Prometheus Operator not found (optional - needed for metrics)"
fi

echo ""
echo "=========================================="
echo "Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
echo -e "${RED}Failed:${NC} $FAILED"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}❌ Pre-flight check FAILED${NC}"
    echo "Please fix the failed checks before installing the operator."
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Pre-flight check PASSED with warnings${NC}"
    echo "You can proceed with installation, but some features may not work."
    exit 0
else
    echo -e "${GREEN}✅ Pre-flight check PASSED${NC}"
    echo "All prerequisites met. You can proceed with installation."
    exit 0
fi

