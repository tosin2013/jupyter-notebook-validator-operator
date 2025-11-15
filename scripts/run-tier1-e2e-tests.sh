#!/bin/bash
set -e

# Script to run Tier 1 E2E tests on OpenShift cluster
# This script creates NotebookValidationJob resources and waits for them to complete

# Configuration
TEST_NAMESPACE="${TEST_NAMESPACE:-jupyter-notebook-validator-e2e}"
TEST_REPO_URL="${TEST_REPO_URL:-https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git}"
TEST_REPO_BRANCH="${TEST_REPO_BRANCH:-main}"
TEST_CREDENTIALS_SECRET="${TEST_CREDENTIALS_SECRET:-git-credentials}"
TEST_CONTAINER_IMAGE="${TEST_CONTAINER_IMAGE:-quay.io/jupyter/minimal-notebook:latest}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-300}"  # 5 minutes
EXPECTED_TEST_COUNT=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Running Tier 1 Tests (Simple notebooks) ==="
echo "Namespace: ${TEST_NAMESPACE}"
echo "Repository: ${TEST_REPO_URL}"
echo "Branch: ${TEST_REPO_BRANCH}"
echo ""

# Create Tier 1 test jobs
echo "Creating NotebookValidationJob resources..."

# Test 1: Hello World
oc apply -f - <<EOF
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-test-01-hello-world
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_BRANCH}"
      credentialsSecret: "${TEST_CREDENTIALS_SECRET}"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: "${TEST_CONTAINER_IMAGE}"
  timeout: "5m"
EOF

# Test 2: Simple Math
oc apply -f - <<EOF
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-test-02-simple-math
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_BRANCH}"
      credentialsSecret: "${TEST_CREDENTIALS_SECRET}"
    path: "notebooks/tier1-simple/02-simple-math.ipynb"
  podConfig:
    containerImage: "${TEST_CONTAINER_IMAGE}"
  timeout: "5m"
EOF

# Test 3: Data Validation
oc apply -f - <<EOF
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier1-test-03-data-validation
  namespace: ${TEST_NAMESPACE}
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_BRANCH}"
      credentialsSecret: "${TEST_CREDENTIALS_SECRET}"
    path: "notebooks/tier1-simple/03-data-validation.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/scipy-notebook:latest"
  timeout: "5m"
EOF

echo ""
echo "Waiting for Tier 1 tests to complete..."
echo "Timeout: ${TIMEOUT_SECONDS} seconds"
echo ""

# Wait for tests to complete
ITERATIONS=$((TIMEOUT_SECONDS / 5))
for i in $(seq 1 $ITERATIONS); do
  COMPLETED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o jsonpath='{.items[?(@.status.phase=="Succeeded")].metadata.name}' | wc -w)
  FAILED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o jsonpath='{.items[?(@.status.phase=="Failed")].metadata.name}' | wc -w)
  
  # Calculate total completed tests
  TOTAL=$((COMPLETED + FAILED))
  
  # Check if all tests are done
  if [ "$TOTAL" -eq "$EXPECTED_TEST_COUNT" ]; then
    echo -e "${GREEN}All tests completed!${NC}"
    break
  fi
  
  echo "Progress: ${COMPLETED} succeeded, ${FAILED} failed (${TOTAL}/${EXPECTED_TEST_COUNT} complete)"
  
  # Check if we've reached the timeout
  if [ "$i" -eq "$ITERATIONS" ]; then
    echo -e "${RED}Timeout reached! Not all tests completed.${NC}"
    break
  fi
  
  sleep 5
done

echo ""
echo "=== Final Test Results ==="
oc get notebookvalidationjobs -n ${TEST_NAMESPACE}

echo ""
echo "=== Checking for failures ==="
FAILED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o jsonpath='{.items[?(@.status.phase=="Failed")].metadata.name}' | wc -w)
SUCCEEDED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o jsonpath='{.items[?(@.status.phase=="Succeeded")].metadata.name}' | wc -w)

if [ "$FAILED" -gt 0 ]; then
  echo -e "${RED}❌ ${FAILED} Tier 1 test(s) failed!${NC}"
  echo ""
  echo "Failed tests:"
  oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -o jsonpath='{range .items[?(@.status.phase=="Failed")]}{.metadata.name}{"\n"}{end}'
  exit 1
fi

if [ "$SUCCEEDED" -ne "$EXPECTED_TEST_COUNT" ]; then
  echo -e "${YELLOW}⚠️  Only ${SUCCEEDED}/${EXPECTED_TEST_COUNT} tests succeeded${NC}"
  exit 1
fi

echo -e "${GREEN}✅ All ${SUCCEEDED} Tier 1 tests completed successfully!${NC}"
exit 0

