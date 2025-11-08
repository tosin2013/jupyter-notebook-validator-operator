#!/bin/bash
# Setup Real Models for Testing
# This script deploys real models to OpenShift for end-to-end testing

set -e

NAMESPACE="${NAMESPACE:-mlops}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=========================================="
echo "Setting up Real Models for Testing"
echo "=========================================="
echo "Namespace: ${NAMESPACE}"
echo "Project Root: ${PROJECT_ROOT}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo "Checking prerequisites..."
if ! command_exists oc; then
    echo "❌ Error: oc CLI not found. Please install OpenShift CLI."
    exit 1
fi

if ! oc whoami &>/dev/null; then
    echo "❌ Error: Not logged into OpenShift. Please run 'oc login' first."
    exit 1
fi

echo "✓ Prerequisites met"
echo ""

# Step 1: Create namespace if it doesn't exist
echo "Step 1: Creating namespace..."
if oc get namespace "${NAMESPACE}" &>/dev/null; then
    echo "✓ Namespace ${NAMESPACE} already exists"
else
    oc create namespace "${NAMESPACE}"
    echo "✓ Created namespace ${NAMESPACE}"
fi
echo ""

# Step 2: Create ServiceAccount and RBAC
echo "Step 2: Setting up RBAC..."
cat <<EOF | oc apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: model-validator-sa
  namespace: ${NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: model-validator-role
  namespace: ${NAMESPACE}
rules:
  - apiGroups: ["serving.kserve.io"]
    resources: ["inferenceservices", "servingruntimes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["services", "pods"]
    verbs: ["get", "list"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: model-validator-binding
  namespace: ${NAMESPACE}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: model-validator-role
subjects:
  - kind: ServiceAccount
    name: model-validator-sa
    namespace: ${NAMESPACE}
EOF
echo "✓ RBAC configured"
echo ""

# Step 3: Deploy ServingRuntime for sklearn models
echo "Step 3: Deploying ServingRuntime for sklearn models..."
cat <<EOF | oc apply -f -
---
apiVersion: serving.kserve.io/v1alpha1
kind: ServingRuntime
metadata:
  name: mlserver-sklearn
  namespace: ${NAMESPACE}
  labels:
    opendatahub.io/dashboard: "true"
spec:
  supportedModelFormats:
    - name: sklearn
      version: "1"
      autoSelect: true
  multiModel: false
  containers:
    - name: kserve-container
      image: docker.io/seldonio/mlserver:1.3.5-sklearn
      env:
        - name: MLSERVER_MODELS_DIR
          value: /mnt/models
        - name: MLSERVER_GRPC_PORT
          value: "9000"
        - name: MLSERVER_HTTP_PORT
          value: "8080"
        - name: MLSERVER_LOAD_MODELS_AT_STARTUP
          value: "false"
        - name: MLSERVER_MODEL_NAME
          value: model
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
  builtInAdapter:
    serverType: mlserver
    runtimeManagementPort: 8001
    memBufferBytes: 134217728
    modelLoadingTimeoutMillis: 90000
EOF
echo "✓ ServingRuntime deployed"
echo ""

# Step 4: Deploy OpenShift AI Sentiment Analysis Model
echo "Step 4: Deploying OpenShift AI Sentiment Analysis Model..."
cat <<EOF | oc apply -f -
---
# Simple sklearn-based sentiment model for testing
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: sentiment-analysis-model
  namespace: ${NAMESPACE}
  annotations:
    serving.kserve.io/deploymentMode: "Serverless"
    openshift.io/display-name: "Sentiment Analysis Model"
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
        version: "1"
      runtime: mlserver-sklearn
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "512Mi"
EOF
echo "✓ Sentiment analysis model deployed"
echo ""

# Step 5: Deploy KServe Fraud Detection Model
echo "Step 5: Deploying KServe Fraud Detection Model..."
cat <<EOF | oc apply -f -
---
# Fraud detection model using sklearn
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: fraud-detection-model
  namespace: ${NAMESPACE}
  annotations:
    serving.kserve.io/deploymentMode: "Serverless"
    openshift.io/display-name: "Fraud Detection Model"
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
        version: "1"
      runtime: mlserver-sklearn
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "512Mi"
EOF
echo "✓ Fraud detection model deployed"
echo ""

# Step 6: Wait for models to be ready
echo "Step 6: Waiting for models to be ready..."
echo "This may take a few minutes..."

# Wait for sentiment analysis model
echo -n "Waiting for sentiment-analysis-model..."
timeout 300 bash -c 'until oc get inferenceservice sentiment-analysis-model -n '"${NAMESPACE}"' -o jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}" 2>/dev/null | grep -q "True"; do echo -n "."; sleep 5; done' || {
    echo ""
    echo "⚠ Warning: sentiment-analysis-model did not become ready within 5 minutes"
    echo "   Check status with: oc describe inferenceservice sentiment-analysis-model -n ${NAMESPACE}"
}
echo " ✓"

# Wait for fraud detection model
echo -n "Waiting for fraud-detection-model..."
timeout 300 bash -c 'until oc get inferenceservice fraud-detection-model -n '"${NAMESPACE}"' -o jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}" 2>/dev/null | grep -q "True"; do echo -n "."; sleep 5; done' || {
    echo ""
    echo "⚠ Warning: fraud-detection-model did not become ready within 5 minutes"
    echo "   Check status with: oc describe inferenceservice fraud-detection-model -n ${NAMESPACE}"
}
echo " ✓"
echo ""

# Step 7: Display model information
echo "=========================================="
echo "Model Deployment Summary"
echo "=========================================="
echo ""

echo "InferenceServices:"
oc get inferenceservices -n "${NAMESPACE}" -o wide

echo ""
echo "Model Endpoints:"
SENTIMENT_URL=$(oc get inferenceservice sentiment-analysis-model -n "${NAMESPACE}" -o jsonpath='{.status.url}' 2>/dev/null || echo "Not available")
FRAUD_URL=$(oc get inferenceservice fraud-detection-model -n "${NAMESPACE}" -o jsonpath='{.status.url}' 2>/dev/null || echo "Not available")

echo "  Sentiment Analysis: ${SENTIMENT_URL}"
echo "  Fraud Detection:    ${FRAUD_URL}"

echo ""
echo "=========================================="
echo "✓ Setup Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Copy model_discovery.py to your test notebooks repository"
echo "2. Update test notebooks to use model discovery"
echo "3. Create NotebookValidationJob with modelValidation enabled"
echo ""
echo "Example NotebookValidationJob:"
echo "---"
echo "apiVersion: mlops.mlops.dev/v1alpha1"
echo "kind: NotebookValidationJob"
echo "metadata:"
echo "  name: test-sentiment-model"
echo "  namespace: ${NAMESPACE}"
echo "spec:"
echo "  notebook:"
echo "    git:"
echo "      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
echo "      ref: main"
echo "    path: model-validation/openshift-ai/sentiment-analysis-test.ipynb"
echo "  podConfig:"
echo "    containerImage: quay.io/jupyter/scipy-notebook:latest"
echo "    serviceAccountName: model-validator-sa"
echo "  modelValidation:"
echo "    enabled: true"
echo "    platform: openshift-ai"
echo "    targetModels:"
echo "      - sentiment-analysis-model"
echo ""

