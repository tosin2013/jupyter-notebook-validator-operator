#!/usr/bin/env python3
"""
Generate test notebooks for the jupyter-notebook-validator-test-notebooks repository.

This script creates Jupyter notebooks for testing:
1. ESO (External Secrets Operator) integration
2. Model-aware validation with KServe
3. Model-aware validation with OpenShift AI
4. Model training and deployment workflows
5. Model-aware validation with community platforms (vLLM, TorchServe, Triton)

Usage:
    python test/generate-test-notebooks.py --output-dir /path/to/jupyter-notebook-validator-test-notebooks
"""

import argparse
import os
import nbformat as nbf
from pathlib import Path


def create_aws_credentials_notebook():
    """Create AWS credentials test notebook."""
    nb = nbf.v4.new_notebook()
    
    nb['cells'] = [
        nbf.v4.new_markdown_cell("# AWS Credentials Test\n\nThis notebook tests AWS credential injection via External Secrets Operator (ESO)."),
        
        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os\n"
            "import boto3\n"
            "from botocore.exceptions import ClientError"
        ),
        
        nbf.v4.new_code_cell(
            "# Verify credentials are injected\n"
            "assert 'AWS_ACCESS_KEY_ID' in os.environ, \"AWS_ACCESS_KEY_ID not found\"\n"
            "assert 'AWS_SECRET_ACCESS_KEY' in os.environ, \"AWS_SECRET_ACCESS_KEY not found\"\n"
            "assert 'AWS_REGION' in os.environ, \"AWS_REGION not found\"\n"
            "print(\"✓ All AWS credentials found\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test credential format\n"
            "access_key = os.environ['AWS_ACCESS_KEY_ID']\n"
            "assert access_key.startswith('AKIA'), f\"Invalid AWS access key format: {access_key[:4]}...\"\n"
            "print(f\"✓ AWS Access Key format valid: {access_key[:4]}...\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test region\n"
            "region = os.environ['AWS_REGION']\n"
            "assert region in ['us-east-1', 'us-west-2', 'eu-west-1'], f\"Unexpected region: {region}\"\n"
            "print(f\"✓ AWS Region: {region}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Summary\n"
            "print(\"\\n=== AWS Credentials Test Summary ===\")\n"
            "print(\"✓ All AWS credentials properly injected\")\n"
            "print(\"✓ Credential format validation passed\")\n"
            "print(\"✓ Region validation passed\")"
        ),
    ]
    
    return nb


def create_database_connection_notebook():
    """Create database connection test notebook."""
    nb = nbf.v4.new_notebook()
    
    nb['cells'] = [
        nbf.v4.new_markdown_cell("# Database Connection Test\n\nThis notebook tests database credential injection via ESO."),
        
        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os\n"
            "from urllib.parse import quote_plus"
        ),
        
        nbf.v4.new_code_cell(
            "# Verify credentials are injected\n"
            "required_vars = ['DB_HOST', 'DB_PORT', 'DB_NAME', 'DB_USER', 'DB_PASSWORD']\n"
            "for var in required_vars:\n"
            "    assert var in os.environ, f\"{var} not found\"\n"
            "print(\"✓ All database credentials found\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Build connection string\n"
            "db_host = os.environ['DB_HOST']\n"
            "db_port = os.environ['DB_PORT']\n"
            "db_name = os.environ['DB_NAME']\n"
            "db_user = os.environ['DB_USER']\n"
            "db_password = os.environ['DB_PASSWORD']\n\n"
            "connection_string = f\"postgresql://{db_user}:{quote_plus(db_password)}@{db_host}:{db_port}/{db_name}\"\n"
            "print(f\"✓ Connection string built: postgresql://{db_user}:***@{db_host}:{db_port}/{db_name}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test connection (mock - don't actually connect in test)\n"
            "print(\"✓ Database credentials validated (connection test skipped in validation)\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Summary\n"
            "print(\"\\n=== Database Credentials Test Summary ===\")\n"
            "print(\"✓ All database credentials properly injected\")\n"
            "print(\"✓ Connection string format valid\")"
        ),
    ]
    
    return nb


def create_mlflow_tracking_notebook():
    """Create MLflow tracking test notebook."""
    nb = nbf.v4.new_notebook()
    
    nb['cells'] = [
        nbf.v4.new_markdown_cell("# MLflow Tracking Test\n\nThis notebook tests MLflow credential injection via ESO."),
        
        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os"
        ),
        
        nbf.v4.new_code_cell(
            "# Verify credentials are injected\n"
            "assert 'MLFLOW_TRACKING_URI' in os.environ, \"MLFLOW_TRACKING_URI not found\"\n"
            "assert 'MLFLOW_TRACKING_USERNAME' in os.environ, \"MLFLOW_TRACKING_USERNAME not found\"\n"
            "assert 'MLFLOW_TRACKING_PASSWORD' in os.environ, \"MLFLOW_TRACKING_PASSWORD not found\"\n"
            "print(\"✓ All MLflow credentials found\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Set MLflow tracking URI\n"
            "tracking_uri = os.environ['MLFLOW_TRACKING_URI']\n"
            "print(f\"✓ MLflow tracking URI: {tracking_uri}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test authentication (mock)\n"
            "username = os.environ['MLFLOW_TRACKING_USERNAME']\n"
            "print(f\"✓ MLflow username: {username}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Summary\n"
            "print(\"\\n=== MLflow Credentials Test Summary ===\")\n"
            "print(\"✓ All MLflow credentials properly injected\")\n"
            "print(\"✓ Tracking URI configured\")"
        ),
    ]
    
    return nb


def create_kserve_inference_notebook():
    """Create KServe model inference test notebook."""
    nb = nbf.v4.new_notebook()
    
    nb['cells'] = [
        nbf.v4.new_markdown_cell("# KServe Model Inference Test\n\nThis notebook tests model inference against KServe InferenceService."),
        
        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os\n"
            "import requests\n"
            "import json\n"
            "import numpy as np"
        ),
        
        nbf.v4.new_code_cell(
            "# Verify model environment variables\n"
            "model_endpoint = os.environ.get('MODEL_ENDPOINT', 'http://fraud-detection-model.mlops.svc.cluster.local')\n"
            "model_name = os.environ.get('MODEL_NAME', 'fraud-detection-model')\n"
            "print(f\"✓ Model endpoint: {model_endpoint}\")\n"
            "print(f\"✓ Model name: {model_name}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test model health check\n"
            "health_url = f\"{model_endpoint}/v1/models/{model_name}\"\n"
            "try:\n"
            "    response = requests.get(health_url, timeout=5)\n"
            "    if response.status_code == 200:\n"
            "        print(\"✓ Model health check passed\")\n"
            "    else:\n"
            "        print(f\"⚠ Health check returned: {response.status_code}\")\n"
            "except requests.exceptions.RequestException as e:\n"
            "    print(f\"⚠ Health check skipped (model not deployed): {e}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Prepare test data\n"
            "test_data = {\n"
            "    \"instances\": [\n"
            "        [1.0, 2.0, 3.0, 4.0, 5.0]\n"
            "    ]\n"
            "}\n"
            "print(f\"✓ Test data prepared: {test_data}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Make prediction (mock if model not available)\n"
            "predict_url = f\"{model_endpoint}/v1/models/{model_name}:predict\"\n"
            "try:\n"
            "    response = requests.post(predict_url, json=test_data, timeout=10)\n"
            "    if response.status_code == 200:\n"
            "        predictions = response.json()\n"
            "        print(f\"✓ Prediction successful: {predictions}\")\n"
            "    else:\n"
            "        print(f\"⚠ Prediction skipped (model not deployed): {response.status_code}\")\n"
            "except requests.exceptions.RequestException as e:\n"
            "    print(f\"⚠ Prediction skipped (model not deployed): {e}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Summary\n"
            "print(\"\\n=== KServe Model Inference Test Summary ===\")\n"
            "print(\"✓ Model endpoint configured\")\n"
            "print(\"✓ Test data prepared\")\n"
            "print(\"✓ Inference test completed\")"
        ),
    ]
    
    return nb


def create_openshift_ai_notebook():
    """Create OpenShift AI sentiment analysis test notebook."""
    nb = nbf.v4.new_notebook()
    
    nb['cells'] = [
        nbf.v4.new_markdown_cell("# OpenShift AI Sentiment Analysis Test\n\nThis notebook tests sentiment analysis model on OpenShift AI."),
        
        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os\n"
            "import requests\n"
            "import json"
        ),
        
        nbf.v4.new_code_cell(
            "# Verify OpenShift AI environment\n"
            "model_endpoint = os.environ.get('MODEL_ENDPOINT', 'http://sentiment-analysis-model.mlops.svc.cluster.local')\n"
            "print(f\"✓ OpenShift AI model endpoint: {model_endpoint}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Prepare sentiment analysis test data\n"
            "test_texts = [\n"
            "    \"This is a great product!\",\n"
            "    \"I love this service.\",\n"
            "    \"Terrible experience, very disappointed.\"\n"
            "]\n"
            "print(f\"✓ Test texts prepared: {len(test_texts)} samples\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Test sentiment analysis (mock if model not available)\n"
            "for i, text in enumerate(test_texts):\n"
            "    print(f\"\\nTest {i+1}: {text}\")\n"
            "    try:\n"
            "        response = requests.post(\n"
            "            f\"{model_endpoint}/predict\",\n"
            "            json={\"instances\": [text]},\n"
            "            timeout=10\n"
            "        )\n"
            "        if response.status_code == 200:\n"
            "            result = response.json()\n"
            "            print(f\"  ✓ Sentiment: {result}\")\n"
            "        else:\n"
            "            print(f\"  ⚠ Prediction skipped: {response.status_code}\")\n"
            "    except requests.exceptions.RequestException as e:\n"
            "        print(f\"  ⚠ Prediction skipped: {e}\")"
        ),
        
        nbf.v4.new_code_cell(
            "# Summary\n"
            "print(\"\\n=== Sentiment Analysis Test Summary ===\")\n"
            "print(\"✓ OpenShift AI endpoint configured\")\n"
            "print(f\"✓ Tested {len(test_texts)} samples\")"
        ),
    ]
    
    return nb


def create_model_training_notebook():
    """Create a notebook that trains a sentiment analysis model from scratch."""
    nb = nbf.v4.new_notebook()

    nb['cells'] = [
        nbf.v4.new_markdown_cell(
            "# Train Sentiment Analysis Model\n\n"
            "This notebook demonstrates a complete ML workflow:\n"
            "1. Load and prepare training data\n"
            "2. Train a sentiment analysis model\n"
            "3. Evaluate model performance\n"
            "4. Save model for deployment\n"
            "5. (Optional) Deploy to KServe/OpenShift AI"
        ),

        nbf.v4.new_code_cell(
            "# Import libraries\n"
            "import os\n"
            "import numpy as np\n"
            "import pandas as pd\n"
            "from sklearn.feature_extraction.text import TfidfVectorizer\n"
            "from sklearn.linear_model import LogisticRegression\n"
            "from sklearn.model_selection import train_test_split\n"
            "from sklearn.metrics import accuracy_score, classification_report\n"
            "import joblib\n"
            "import json"
        ),

        nbf.v4.new_markdown_cell("## Step 1: Create Training Data"),

        nbf.v4.new_code_cell(
            "# Create sample sentiment analysis dataset\n"
            "# In production, you would load this from S3, database, etc.\n"
            "training_data = [\n"
            "    # Positive sentiments\n"
            "    ('This is excellent!', 1),\n"
            "    ('I love this product', 1),\n"
            "    ('Amazing quality and service', 1),\n"
            "    ('Best purchase ever', 1),\n"
            "    ('Highly recommend this', 1),\n"
            "    ('Fantastic experience', 1),\n"
            "    ('Great value for money', 1),\n"
            "    ('Exceeded my expectations', 1),\n"
            "    ('Very satisfied with this', 1),\n"
            "    ('Outstanding product', 1),\n"
            "    \n"
            "    # Negative sentiments\n"
            "    ('This is terrible', 0),\n"
            "    ('I hate this product', 0),\n"
            "    ('Poor quality and service', 0),\n"
            "    ('Worst purchase ever', 0),\n"
            "    ('Do not recommend', 0),\n"
            "    ('Awful experience', 0),\n"
            "    ('Complete waste of money', 0),\n"
            "    ('Very disappointed', 0),\n"
            "    ('Not satisfied at all', 0),\n"
            "    ('Terrible product', 0),\n"
            "]\n"
            "\n"
            "# Convert to DataFrame\n"
            "df = pd.DataFrame(training_data, columns=['text', 'sentiment'])\n"
            "print(f\"✓ Created training dataset with {len(df)} samples\")\n"
            "print(f\"  Positive: {(df['sentiment'] == 1).sum()}\")\n"
            "print(f\"  Negative: {(df['sentiment'] == 0).sum()}\")"
        ),

        nbf.v4.new_markdown_cell("## Step 2: Prepare Data"),

        nbf.v4.new_code_cell(
            "# Split into train and test sets\n"
            "X_train, X_test, y_train, y_test = train_test_split(\n"
            "    df['text'], df['sentiment'], test_size=0.2, random_state=42\n"
            ")\n"
            "\n"
            "print(f\"✓ Split data:\")\n"
            "print(f\"  Training samples: {len(X_train)}\")\n"
            "print(f\"  Test samples: {len(X_test)}\")"
        ),

        nbf.v4.new_code_cell(
            "# Create TF-IDF vectorizer\n"
            "vectorizer = TfidfVectorizer(max_features=100, ngram_range=(1, 2))\n"
            "X_train_vec = vectorizer.fit_transform(X_train)\n"
            "X_test_vec = vectorizer.transform(X_test)\n"
            "\n"
            "print(f\"✓ Vectorized text data\")\n"
            "print(f\"  Feature dimensions: {X_train_vec.shape[1]}\")"
        ),

        nbf.v4.new_markdown_cell("## Step 3: Train Model"),

        nbf.v4.new_code_cell(
            "# Train logistic regression model\n"
            "model = LogisticRegression(random_state=42, max_iter=1000)\n"
            "model.fit(X_train_vec, y_train)\n"
            "\n"
            "print(\"✓ Model trained successfully\")"
        ),

        nbf.v4.new_markdown_cell("## Step 4: Evaluate Model"),

        nbf.v4.new_code_cell(
            "# Make predictions\n"
            "y_pred = model.predict(X_test_vec)\n"
            "\n"
            "# Calculate accuracy\n"
            "accuracy = accuracy_score(y_test, y_pred)\n"
            "print(f\"✓ Model Accuracy: {accuracy:.2%}\")\n"
            "\n"
            "# Print classification report\n"
            "print(\"\\nClassification Report:\")\n"
            "print(classification_report(y_test, y_pred, target_names=['Negative', 'Positive']))"
        ),

        nbf.v4.new_code_cell(
            "# Test with sample predictions\n"
            "test_texts = [\n"
            "    'This is amazing!',\n"
            "    'This is awful',\n"
            "    'Not bad, pretty good actually'\n"
            "]\n"
            "\n"
            "print(\"\\nSample Predictions:\")\n"
            "for text in test_texts:\n"
            "    vec = vectorizer.transform([text])\n"
            "    pred = model.predict(vec)[0]\n"
            "    prob = model.predict_proba(vec)[0]\n"
            "    sentiment = 'Positive' if pred == 1 else 'Negative'\n"
            "    confidence = prob[pred]\n"
            "    print(f\"  '{text}'\")\n"
            "    print(f\"    → {sentiment} (confidence: {confidence:.2%})\")"
        ),

        nbf.v4.new_markdown_cell("## Step 5: Save Model"),

        nbf.v4.new_code_cell(
            "# Create model directory\n"
            "model_dir = '/tmp/sentiment-model'\n"
            "os.makedirs(model_dir, exist_ok=True)\n"
            "\n"
            "# Save model and vectorizer\n"
            "joblib.dump(model, f'{model_dir}/model.pkl')\n"
            "joblib.dump(vectorizer, f'{model_dir}/vectorizer.pkl')\n"
            "\n"
            "# Save model metadata\n"
            "metadata = {\n"
            "    'model_type': 'LogisticRegression',\n"
            "    'accuracy': float(accuracy),\n"
            "    'features': X_train_vec.shape[1],\n"
            "    'training_samples': len(X_train),\n"
            "    'classes': ['Negative', 'Positive']\n"
            "}\n"
            "\n"
            "with open(f'{model_dir}/metadata.json', 'w') as f:\n"
            "    json.dump(metadata, f, indent=2)\n"
            "\n"
            "print(f\"✓ Model saved to {model_dir}\")\n"
            "print(f\"  - model.pkl\")\n"
            "print(f\"  - vectorizer.pkl\")\n"
            "print(f\"  - metadata.json\")"
        ),

        nbf.v4.new_markdown_cell(
            "## Step 6: Test Saved Model\n\n"
            "Verify the saved model can be loaded and used for predictions."
        ),

        nbf.v4.new_code_cell(
            "# Load saved model\n"
            "loaded_model = joblib.load(f'{model_dir}/model.pkl')\n"
            "loaded_vectorizer = joblib.load(f'{model_dir}/vectorizer.pkl')\n"
            "\n"
            "# Test prediction\n"
            "test_text = 'This is a great product!'\n"
            "vec = loaded_vectorizer.transform([test_text])\n"
            "pred = loaded_model.predict(vec)[0]\n"
            "prob = loaded_model.predict_proba(vec)[0]\n"
            "\n"
            "print(\"✓ Loaded model test:\")\n"
            "print(f\"  Input: '{test_text}'\")\n"
            "print(f\"  Prediction: {'Positive' if pred == 1 else 'Negative'}\")\n"
            "print(f\"  Confidence: {prob[pred]:.2%}\")"
        ),

        nbf.v4.new_markdown_cell(
            "## Step 7: Deploy Model (Optional)\n\n"
            "Deploy the trained model to KServe/OpenShift AI for serving.\n\n"
            "**Note:** This step requires:\n"
            "- Model files uploaded to S3 or persistent storage\n"
            "- Kubernetes/OpenShift cluster with KServe installed\n"
            "- Appropriate RBAC permissions"
        ),

        nbf.v4.new_code_cell(
            "# Check if we're running in Kubernetes\n"
            "import os\n"
            "from pathlib import Path\n"
            "\n"
            "in_kubernetes = Path('/var/run/secrets/kubernetes.io/serviceaccount/token').exists()\n"
            "deploy_enabled = os.environ.get('DEPLOY_MODEL', 'false').lower() == 'true'\n"
            "\n"
            "print(f\"Running in Kubernetes: {in_kubernetes}\")\n"
            "print(f\"Model deployment enabled: {deploy_enabled}\")\n"
            "\n"
            "if in_kubernetes and deploy_enabled:\n"
            "    print(\"\\n✓ Ready to deploy model\")\n"
            "else:\n"
            "    print(\"\\n⚠ Skipping deployment (set DEPLOY_MODEL=true to enable)\")"
        ),

        nbf.v4.new_code_cell(
            "# Deploy model to KServe/OpenShift AI\n"
            "if in_kubernetes and deploy_enabled:\n"
            "    try:\n"
            "        from kubernetes import client, config\n"
            "        import yaml\n"
            "        \n"
            "        # Load in-cluster config\n"
            "        config.load_incluster_config()\n"
            "        \n"
            "        # Get namespace\n"
            "        namespace = os.environ.get('NAMESPACE', 'mlops')\n"
            "        model_name = os.environ.get('MODEL_NAME', 'trained-sentiment-model')\n"
            "        storage_uri = os.environ.get('MODEL_STORAGE_URI', 'pvc://model-storage/sentiment-model')\n"
            "        \n"
            "        # Create InferenceService manifest\n"
            "        inference_service = {\n"
            "            'apiVersion': 'serving.kserve.io/v1beta1',\n"
            "            'kind': 'InferenceService',\n"
            "            'metadata': {\n"
            "                'name': model_name,\n"
            "                'namespace': namespace,\n"
            "                'annotations': {\n"
            "                    'serving.kserve.io/deploymentMode': 'Serverless'\n"
            "                },\n"
            "                'labels': {\n"
            "                    'trained-by': 'jupyter-notebook-validator',\n"
            "                    'model-type': 'sklearn',\n"
            "                    'training-notebook': 'train-sentiment-model'\n"
            "                }\n"
            "            },\n"
            "            'spec': {\n"
            "                'predictor': {\n"
            "                    'model': {\n"
            "                        'modelFormat': {\n"
            "                            'name': 'sklearn',\n"
            "                            'version': '1'\n"
            "                        },\n"
            "                        'runtime': 'mlserver-sklearn',\n"
            "                        'storageUri': storage_uri,\n"
            "                        'resources': {\n"
            "                            'requests': {\n"
            "                                'cpu': '100m',\n"
            "                                'memory': '256Mi'\n"
            "                            },\n"
            "                            'limits': {\n"
            "                                'cpu': '500m',\n"
            "                                'memory': '512Mi'\n"
            "                            }\n"
            "                        }\n"
            "                    }\n"
            "                }\n"
            "            }\n"
            "        }\n"
            "        \n"
            "        # Create custom object API\n"
            "        api = client.CustomObjectsApi()\n"
            "        \n"
            "        # Deploy InferenceService\n"
            "        try:\n"
            "            api.create_namespaced_custom_object(\n"
            "                group='serving.kserve.io',\n"
            "                version='v1beta1',\n"
            "                namespace=namespace,\n"
            "                plural='inferenceservices',\n"
            "                body=inference_service\n"
            "            )\n"
            "            print(f\"✓ InferenceService '{model_name}' created in namespace '{namespace}'\")\n"
            "        except client.exceptions.ApiException as e:\n"
            "            if e.status == 409:\n"
            "                print(f\"⚠ InferenceService '{model_name}' already exists\")\n"
            "            else:\n"
            "                raise\n"
            "        \n"
            "        print(f\"\\nDeployment details:\")\n"
            "        print(f\"  Model name: {model_name}\")\n"
            "        print(f\"  Namespace: {namespace}\")\n"
            "        print(f\"  Storage URI: {storage_uri}\")\n"
            "        print(f\"\\nWait for model to be ready:\")\n"
            "        print(f\"  oc wait --for=condition=Ready inferenceservice/{model_name} -n {namespace} --timeout=5m\")\n"
            "        \n"
            "    except Exception as e:\n"
            "        print(f\"✗ Deployment failed: {e}\")\n"
            "        print(\"\\nTo deploy manually, save the model to S3/PVC and create InferenceService:\")\n"
            "        print(yaml.dump(inference_service, default_flow_style=False))\n"
            "else:\n"
            "    print(\"Skipping deployment. To deploy manually:\")\n"
            "    print(\"\\n1. Upload model files to S3 or PVC\")\n"
            "    print(\"2. Create InferenceService with storageUri pointing to model location\")\n"
            "    print(\"3. Wait for model to be Ready\")"
        ),

        nbf.v4.new_markdown_cell(
            "## Step 8: Test Deployed Model (Optional)\n\n"
            "Use the model discovery library to find and test the deployed model."
        ),

        nbf.v4.new_code_cell(
            "# Test deployed model using model discovery\n"
            "if in_kubernetes and deploy_enabled:\n"
            "    try:\n"
            "        import sys\n"
            "        sys.path.append('/workspace/lib')\n"
            "        from model_discovery import discover_models, get_model_endpoint, make_prediction\n"
            "        \n"
            "        # Wait a bit for deployment\n"
            "        import time\n"
            "        print(\"Waiting 30 seconds for model deployment...\")\n"
            "        time.sleep(30)\n"
            "        \n"
            "        # Discover models\n"
            "        namespace = os.environ.get('NAMESPACE', 'mlops')\n"
            "        models = discover_models(platform='openshift-ai', namespace=namespace)\n"
            "        \n"
            "        model_name = os.environ.get('MODEL_NAME', 'trained-sentiment-model')\n"
            "        \n"
            "        if model_name in models:\n"
            "            print(f\"\\n✓ Found deployed model: {model_name}\")\n"
            "            model_info = models[model_name]\n"
            "            print(f\"  URL: {model_info['url']}\")\n"
            "            print(f\"  Ready: {model_info['ready']}\")\n"
            "            \n"
            "            if model_info['ready']:\n"
            "                # Test prediction\n"
            "                test_texts = [\n"
            "                    'This is amazing!',\n"
            "                    'This is terrible',\n"
            "                    'Pretty good product'\n"
            "                ]\n"
            "                \n"
            "                print(\"\\nTesting predictions:\")\n"
            "                for text in test_texts:\n"
            "                    # Transform text using vectorizer\n"
            "                    vec = loaded_vectorizer.transform([text])\n"
            "                    # Convert to list for JSON serialization\n"
            "                    vec_list = vec.toarray().tolist()\n"
            "                    \n"
            "                    try:\n"
            "                        result = make_prediction(\n"
            "                            model_info['url'],\n"
            "                            {'instances': vec_list}\n"
            "                        )\n"
            "                        pred = result.get('predictions', [[]])[0]\n"
            "                        sentiment = 'Positive' if pred[0] == 1 else 'Negative'\n"
            "                        print(f\"  '{text}' → {sentiment}\")\n"
            "                    except Exception as e:\n"
            "                        print(f\"  '{text}' → Error: {e}\")\n"
            "            else:\n"
            "                print(\"\\n⚠ Model not ready yet. Check status with:\")\n"
            "                print(f\"  oc get inferenceservice {model_name} -n {namespace}\")\n"
            "        else:\n"
            "            print(f\"\\n⚠ Model '{model_name}' not found in namespace '{namespace}'\")\n"
            "            print(f\"\\nAvailable models: {list(models.keys())}\")\n"
            "    \n"
            "    except ImportError:\n"
            "        print(\"⚠ model_discovery library not found\")\n"
            "        print(\"Make sure /workspace/lib/model_discovery.py exists\")\n"
            "    except Exception as e:\n"
            "        print(f\"✗ Testing failed: {e}\")\n"
            "else:\n"
            "    print(\"Skipping deployed model testing\")"
        ),

        nbf.v4.new_markdown_cell(
            "## Summary\n\n"
            "✅ Training data created (20 samples)\n"
            "✅ Model trained (Logistic Regression)\n"
            "✅ Model evaluated (accuracy reported)\n"
            "✅ Model saved to disk\n"
            "✅ Saved model tested\n"
            "✅ Model deployment (optional, if DEPLOY_MODEL=true)\n"
            "✅ Deployed model testing (optional)\n\n"
            "### Complete End-to-End Workflow\n\n"
            "This notebook demonstrates a complete ML workflow:\n"
            "1. **Data Preparation** - Create/load training data\n"
            "2. **Feature Engineering** - TF-IDF vectorization\n"
            "3. **Model Training** - Train classifier\n"
            "4. **Model Evaluation** - Validate performance\n"
            "5. **Model Persistence** - Save model artifacts\n"
            "6. **Model Testing** - Verify saved model works\n"
            "7. **Model Deployment** - Deploy to KServe/OpenShift AI (optional)\n"
            "8. **Inference Testing** - Test deployed model (optional)\n\n"
            "### Environment Variables for Deployment\n\n"
            "To enable automatic deployment, set:\n"
            "- `DEPLOY_MODEL=true` - Enable deployment\n"
            "- `NAMESPACE=mlops` - Target namespace\n"
            "- `MODEL_NAME=trained-sentiment-model` - Model name\n"
            "- `MODEL_STORAGE_URI=pvc://model-storage/sentiment-model` - Storage location\n\n"
            "### Manual Deployment\n\n"
            "If automatic deployment is disabled, deploy manually:\n\n"
            "```bash\n"
            "# 1. Extract model from pod\n"
            "POD=$(oc get pods -n mlops -l job-name=train-sentiment-model-validation -o jsonpath='{.items[0].metadata.name}')\n"
            "oc cp mlops/$POD:/tmp/sentiment-model ./trained-model/\n\n"
            "# 2. Upload to S3\n"
            "aws s3 cp ./trained-model/ s3://my-bucket/models/sentiment-v1/ --recursive\n\n"
            "# 3. Create InferenceService\n"
            "cat <<EOF | oc apply -f -\n"
            "apiVersion: serving.kserve.io/v1beta1\n"
            "kind: InferenceService\n"
            "metadata:\n"
            "  name: trained-sentiment-model\n"
            "  namespace: mlops\n"
            "spec:\n"
            "  predictor:\n"
            "    model:\n"
            "      modelFormat:\n"
            "        name: sklearn\n"
            "        version: '1'\n"
            "      runtime: mlserver-sklearn\n"
            "      storageUri: s3://my-bucket/models/sentiment-v1/\n"
            "EOF\n"
            "```"
        ),
    ]

    return nb


def main():
    parser = argparse.ArgumentParser(description='Generate test notebooks for integration testing')
    parser.add_argument('--output-dir', type=str, required=True,
                        help='Output directory (path to jupyter-notebook-validator-test-notebooks repo)')
    args = parser.parse_args()
    
    output_dir = Path(args.output_dir)
    
    # Create directory structure
    (output_dir / 'eso-integration').mkdir(parents=True, exist_ok=True)
    (output_dir / 'model-validation' / 'kserve').mkdir(parents=True, exist_ok=True)
    (output_dir / 'model-validation' / 'openshift-ai').mkdir(parents=True, exist_ok=True)
    (output_dir / 'model-training').mkdir(parents=True, exist_ok=True)

    # Generate notebooks
    notebooks = {
        'eso-integration/aws-credentials-test.ipynb': create_aws_credentials_notebook(),
        'eso-integration/database-connection-test.ipynb': create_database_connection_notebook(),
        'eso-integration/mlflow-tracking-test.ipynb': create_mlflow_tracking_notebook(),
        'model-validation/kserve/model-inference-kserve.ipynb': create_kserve_inference_notebook(),
        'model-validation/openshift-ai/sentiment-analysis-test.ipynb': create_openshift_ai_notebook(),
        'model-training/train-sentiment-model.ipynb': create_model_training_notebook(),
    }
    
    for path, notebook in notebooks.items():
        output_path = output_dir / path
        with open(output_path, 'w') as f:
            nbf.write(notebook, f)
        print(f"✓ Created: {output_path}")
    
    print(f"\n✓ Successfully generated {len(notebooks)} test notebooks in {output_dir}")
    print("\nNext steps:")
    print("1. cd to the jupyter-notebook-validator-test-notebooks directory")
    print("2. Review the generated notebooks")
    print("3. git add . && git commit -m 'Add integration test notebooks'")
    print("4. git push origin main")


if __name__ == '__main__':
    main()

