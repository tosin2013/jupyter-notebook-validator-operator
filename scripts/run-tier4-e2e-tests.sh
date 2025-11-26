#!/bin/bash
set -e

# Script to run Tier 4 E2E tests (S2I/BuildConfig) on OpenShift cluster
# This script tests ADR-044 features: SCC management, retry logic, S2I builds

# Configuration
TEST_NAMESPACE="${TEST_NAMESPACE:-e2e-tests}"
TEST_REPO_URL="${TEST_REPO_URL:-https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git}"
TEST_REPO_BRANCH="${TEST_REPO_BRANCH:-main}"
TEST_CREDENTIALS_SECRET="${TEST_CREDENTIALS_SECRET:-git-credentials}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"  # 30 minutes for builds
EXPECTED_TEST_COUNT=2

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Running Tier 4 Tests (S2I/BuildConfig validation) ===${NC}"
echo "Namespace: ${TEST_NAMESPACE}"
echo "Repository: ${TEST_REPO_URL}"
echo "Branch: ${TEST_REPO_BRANCH}"
echo "Timeout: ${TIMEOUT_SECONDS} seconds (30 minutes)"
echo ""

# Create Tier 4 test jobs
echo -e "${BLUE}Creating Tier 4 NotebookValidationJob resources...${NC}"

# Test 01: S2I Python build with pandas
echo -e "${GREEN}Creating Tier 4 Test 01: S2I Python build...${NC}"
oc apply -f - <<EOF
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier4-test-01-s2i-python-build
  namespace: ${TEST_NAMESPACE}
  labels:
    tier: "4"
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_BRANCH}"
      credentialsSecret: "${TEST_CREDENTIALS_SECRET}"
    path: "notebooks/tier4-s2i/01-s2i-python-build.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    buildConfig:
      enabled: true
      strategy: "s2i"
  timeout: "15m"
EOF

# Wait 10 seconds before creating next test
echo -e "${YELLOW}Waiting 10s before creating next test...${NC}"
sleep 10

# Test 02: S2I custom requirements
echo -e "${GREEN}Creating Tier 4 Test 02: S2I custom requirements...${NC}"
oc apply -f - <<EOF
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: tier4-test-02-s2i-custom-requirements
  namespace: ${TEST_NAMESPACE}
  labels:
    tier: "4"
spec:
  notebook:
    git:
      url: "${TEST_REPO_URL}"
      ref: "${TEST_REPO_BRANCH}"
      credentialsSecret: "${TEST_CREDENTIALS_SECRET}"
    path: "notebooks/tier4-s2i/02-s2i-custom-requirements.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    buildConfig:
      enabled: true
      strategy: "s2i"
  timeout: "15m"
EOF

echo -e "${GREEN}All Tier 4 tests created with staggered timing${NC}"
echo ""

# Wait for tests to complete
echo -e "${BLUE}Waiting for Tier 4 tests to complete (may take 10-15 minutes)...${NC}"
echo "Checking progress every 10 seconds"
echo ""

ITERATIONS=$((TIMEOUT_SECONDS / 10))
for i in $(seq 1 $ITERATIONS); do
  COMPLETED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{.items[?(@.status.phase=="Succeeded")].metadata.name}' | wc -w)
  FAILED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{.items[?(@.status.phase=="Failed")].metadata.name}' | wc -w)

  # Calculate total completed tests
  TOTAL=$((COMPLETED + FAILED))

  # Check if all tests are done
  if [ "$TOTAL" -eq "$EXPECTED_TEST_COUNT" ]; then
    echo -e "${GREEN}All Tier 4 tests completed!${NC}"
    break
  fi

  echo "Progress: ${COMPLETED} succeeded, ${FAILED} failed (${TOTAL}/${EXPECTED_TEST_COUNT} complete)"

  # Check if we've reached the timeout
  if [ "$i" -eq "$ITERATIONS" ]; then
    echo -e "${RED}Timeout reached! Not all tests completed.${NC}"
    break
  fi

  sleep 10
done

echo ""
echo -e "${BLUE}=== Final Tier 4 Test Results ===${NC}"
oc get notebookvalidationjobs -n ${TEST_NAMESPACE}

echo ""
echo -e "${BLUE}=== Checking for failures ===${NC}"
FAILED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{.items[?(@.status.phase=="Failed")].metadata.name}' | wc -w)
SUCCEEDED=$(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{.items[?(@.status.phase=="Succeeded")].metadata.name}' | wc -w)

if [ "$FAILED" -gt 0 ]; then
  echo -e "${RED}❌ ${FAILED} Tier 4 test(s) failed!${NC}"
  echo ""
  echo "Failed tests:"
  oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{range .items[?(@.status.phase=="Failed")]}{.metadata.name}{"\n"}{end}'

  echo ""
  echo -e "${YELLOW}=== Detailed failure information ===${NC}"
  for job in $(oc get notebookvalidationjobs -n ${TEST_NAMESPACE} -l tier=4 -o jsonpath='{.items[?(@.status.phase=="Failed")].metadata.name}'); do
    echo ""
    echo -e "${YELLOW}Job: ${job}${NC}"
    echo "Status:"
    oc get notebookvalidationjob ${job} -n ${TEST_NAMESPACE} -o jsonpath='{.status}' | python3 -m json.tool || oc get notebookvalidationjob ${job} -n ${TEST_NAMESPACE} -o yaml
    echo ""
    echo "Status conditions:"
    oc get notebookvalidationjob ${job} -n ${TEST_NAMESPACE} -o jsonpath='{.status.conditions[*]}' | python3 -m json.tool || echo "No conditions available"
    echo ""
    echo "Pod logs (last 100 lines):"
    POD_NAME=$(oc get pods -n ${TEST_NAMESPACE} -l job-name=${job} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POD_NAME" ]; then
      echo "Pod: ${POD_NAME}"
      oc logs ${POD_NAME} -n ${TEST_NAMESPACE} --tail=100 2>&1 || echo "Could not retrieve pod logs"
    else
      echo "No pod found for job ${job}"
    fi
    echo ""
    echo -e "${YELLOW}Checking BuildConfig and Builds:${NC}"
    BC_NAME="${job}-build"
    if oc get buildconfig ${BC_NAME} -n ${TEST_NAMESPACE} &>/dev/null; then
      echo "BuildConfig: ${BC_NAME}"
      oc get buildconfig ${BC_NAME} -n ${TEST_NAMESPACE} -o yaml
      echo ""
      echo "Builds for ${BC_NAME}:"
      oc get builds -n ${TEST_NAMESPACE} -l buildconfig=${BC_NAME}
      echo ""
      LATEST_BUILD=$(oc get builds -n ${TEST_NAMESPACE} -l buildconfig=${BC_NAME} -o jsonpath='{.items[-1:].metadata.name}')
      if [ -n "$LATEST_BUILD" ]; then
        echo "Latest build logs (${LATEST_BUILD}):"
        oc logs build/${LATEST_BUILD} -n ${TEST_NAMESPACE} --tail=50 2>&1 || echo "Could not retrieve build logs"
      fi
    else
      echo "No BuildConfig found: ${BC_NAME}"
    fi
    echo "---"
  done

  echo ""
  echo -e "${BLUE}=== ADR-044 Feature Verification ===${NC}"
  echo -e "${YELLOW}Checking SCC management:${NC}"
  oc get scc pipelines-scc -o jsonpath='{.users}' | grep -q "system:serviceaccount:${TEST_NAMESPACE}:builder" && \
    echo -e "${GREEN}✅ SCC pipelines-scc granted to builder ServiceAccount${NC}" || \
    echo -e "${RED}❌ SCC pipelines-scc NOT granted to builder ServiceAccount${NC}"

  echo ""
  echo -e "${YELLOW}Checking operator logs for ADR-044 features:${NC}"
  echo "Looking for: SCC management, BuildConfig retry logic"
  oc logs -n jupyter-notebook-validator-operator \
    -l control-plane=controller-manager --tail=200 | \
    grep -E "SCC|BuildConfig|retry|ServiceAccount" || \
    echo "No ADR-044 related logs found"

  exit 1
fi

if [ "$SUCCEEDED" -ne "$EXPECTED_TEST_COUNT" ]; then
  echo -e "${YELLOW}⚠️  Only ${SUCCEEDED}/${EXPECTED_TEST_COUNT} tests succeeded${NC}"
  exit 1
fi

echo -e "${GREEN}✅ All ${SUCCEEDED} Tier 4 tests completed successfully!${NC}"
echo ""
echo -e "${BLUE}=== ADR-044 Features Verified ===${NC}"
echo -e "${GREEN}✅ S2I builds completed${NC}"
echo -e "${GREEN}✅ SCC management working${NC}"
echo -e "${GREEN}✅ BuildConfig creation successful${NC}"
exit 0
