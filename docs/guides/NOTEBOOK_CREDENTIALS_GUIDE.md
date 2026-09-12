# Notebook Credentials Guide

**Version:** 1.0  
**Last Updated:** 2025-11-08  
**Status:** Production Ready

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Credential Injection Patterns](#credential-injection-patterns)
4. [AWS S3 Access](#aws-s3-access)
5. [Database Connections](#database-connections)
6. [API Key Injection](#api-key-injection)
7. [Multi-Service Examples](#multi-service-examples)
8. [External Secrets Operator (ESO)](#external-secrets-operator-eso)
9. [Vault Integration](#vault-integration)
10. [Security Best Practices](#security-best-practices)
11. [Troubleshooting](#troubleshooting)

## Overview

The Jupyter Notebook Validator Operator supports secure credential injection for notebooks that need to access external services during validation. This guide covers all supported patterns and best practices.

### Supported Services

- **Cloud Storage**: AWS S3, Azure Blob Storage, GCP Cloud Storage
- **Databases**: PostgreSQL, MySQL, MongoDB, Redis
- **APIs**: OpenAI, Hugging Face, MLflow, custom REST APIs
- **Model Registries**: MLflow, KServe, Seldon
- **Data Platforms**: Snowflake, Databricks, BigQuery

### Three-Tier Strategy

1. **Tier 1: Environment Variables** (Basic) - Kubernetes Secrets as env vars
2. **Tier 2: External Secrets Operator** (Recommended) - Cloud-native secret sync
3. **Tier 3: Vault Dynamic Secrets** (Advanced) - Short-lived credentials

## Quick Start

### Step 1: Create a Secret

```bash
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=AKIAIOSFODNN7EXAMPLE \
  --from-literal=secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  -n default
```

### Step 2: Reference in NotebookValidationJob

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-notebook-job
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/my-notebook.ipynb"
  
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: secret-access-key
```

### Step 3: Use in Notebook

```python
import boto3
import os

# Credentials automatically loaded from environment
s3 = boto3.client('s3')
s3.list_buckets()
```

## Credential Injection Patterns

### Pattern 1: Individual Environment Variables

Use `env` for individual credentials:

```yaml
spec:
  podConfig:
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
      - name: AWS_DEFAULT_REGION
        value: "us-east-1"  # Plain value for non-sensitive data
```

**When to use:**
- Few credentials needed
- Mix of secret and non-secret values
- Fine-grained control over variable names

### Pattern 2: Bulk Secret Loading

Use `envFrom` to load all keys from a Secret:

```yaml
spec:
  podConfig:
    envFrom:
      - secretRef:
          name: database-credentials
      - configMapRef:
          name: app-config
```

**When to use:**
- Many credentials from same source
- All keys should be environment variables
- Simpler configuration

### Pattern 3: Mixed Approach

Combine both patterns:

```yaml
spec:
  podConfig:
    envFrom:
      - secretRef:
          name: database-credentials
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
      - name: LOG_LEVEL
        value: "INFO"
```

## AWS S3 Access

### Creating AWS Credentials Secret

```bash
kubectl create secret generic aws-credentials \
  --from-literal=access-key-id=AKIA... \
  --from-literal=secret-access-key=wJalr... \
  --from-literal=region=us-east-1 \
  -n default
```

### NotebookValidationJob Configuration

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: s3-data-pipeline
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/s3-pipeline.ipynb"
  
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: access-key-id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: secret-access-key
      - name: AWS_DEFAULT_REGION
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: region
      - name: S3_BUCKET_NAME
        value: "my-data-bucket"
```

### Notebook Code (boto3)

```python
import boto3
import pandas as pd
import os

# Initialize S3 client (uses AWS_* environment variables)
s3 = boto3.client('s3')

# Get bucket name from environment
bucket = os.environ['S3_BUCKET_NAME']

# Download training data
s3.download_file(bucket, 'data/train.csv', 'train.csv')
df = pd.read_csv('train.csv')

# Train model
model = train_model(df)

# Upload model artifacts
s3.upload_file('model.pkl', bucket, 'models/model.pkl')
```

### Notebook Code (s3fs)

```python
import s3fs
import pandas as pd
import os

# Initialize S3 filesystem
fs = s3fs.S3FileSystem(
    key=os.environ['AWS_ACCESS_KEY_ID'],
    secret=os.environ['AWS_SECRET_ACCESS_KEY']
)

# Read data directly from S3
bucket = os.environ['S3_BUCKET_NAME']
with fs.open(f'{bucket}/data/train.csv', 'r') as f:
    df = pd.read_csv(f)

# Write results back to S3
with fs.open(f'{bucket}/results/output.csv', 'w') as f:
    df.to_csv(f, index=False)
```

## Database Connections

### PostgreSQL

#### Creating Database Secret

```bash
kubectl create secret generic postgres-credentials \
  --from-literal=DB_HOST=postgres.example.com \
  --from-literal=DB_PORT=5432 \
  --from-literal=DB_NAME=mlops_features \
  --from-literal=DB_USER=mlops_reader \
  --from-literal=DB_PASSWORD=secure-password \
  -n default
```

#### NotebookValidationJob Configuration

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: database-feature-engineering
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/feature-engineering.ipynb"
  
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
    envFrom:
      - secretRef:
          name: postgres-credentials
    env:
      - name: DB_SSL_MODE
        value: "require"
```

#### Notebook Code (psycopg2)

```python
import psycopg2
import pandas as pd
import os

# Connect to PostgreSQL
conn = psycopg2.connect(
    host=os.environ['DB_HOST'],
    port=os.environ['DB_PORT'],
    database=os.environ['DB_NAME'],
    user=os.environ['DB_USER'],
    password=os.environ['DB_PASSWORD'],
    sslmode=os.environ.get('DB_SSL_MODE', 'prefer')
)

# Query features
query = """
    SELECT user_id, feature1, feature2, feature3
    FROM features
    WHERE date >= '2024-01-01'
"""
df = pd.read_sql(query, conn)

# Process features
processed_df = process_features(df)

# Close connection
conn.close()
```

#### Notebook Code (SQLAlchemy)

```python
from sqlalchemy import create_engine
import pandas as pd
import os

# Create database URL
db_url = f"postgresql://{os.environ['DB_USER']}:{os.environ['DB_PASSWORD']}@{os.environ['DB_HOST']}:{os.environ['DB_PORT']}/{os.environ['DB_NAME']}"

# Create engine
engine = create_engine(db_url)

# Query with pandas
df = pd.read_sql_table('features', engine)

# Or use raw SQL
df = pd.read_sql_query("SELECT * FROM features WHERE date >= '2024-01-01'", engine)
```

### MySQL

```python
import mysql.connector
import pandas as pd
import os

# Connect to MySQL
conn = mysql.connector.connect(
    host=os.environ['DB_HOST'],
    port=int(os.environ['DB_PORT']),
    database=os.environ['DB_NAME'],
    user=os.environ['DB_USER'],
    password=os.environ['DB_PASSWORD']
)

# Query data
df = pd.read_sql("SELECT * FROM features", conn)
conn.close()
```

### MongoDB

```python
from pymongo import MongoClient
import os

# Connect to MongoDB
client = MongoClient(
    host=os.environ['MONGO_HOST'],
    port=int(os.environ['MONGO_PORT']),
    username=os.environ['MONGO_USER'],
    password=os.environ['MONGO_PASSWORD']
)

# Access database and collection
db = client[os.environ['MONGO_DATABASE']]
collection = db['features']

# Query documents
documents = list(collection.find({'date': {'$gte': '2024-01-01'}}))
```

## API Key Injection

### OpenAI API

#### Creating API Key Secret

```bash
kubectl create secret generic api-keys \
  --from-literal=openai=sk-proj-... \
  --from-literal=huggingface=hf_... \
  -n default
```

#### NotebookValidationJob Configuration

```yaml
spec:
  podConfig:
    env:
      - name: OPENAI_API_KEY
        valueFrom:
          secretKeyRef:
            name: api-keys
            key: openai
```

#### Notebook Code

```python
import openai
import os

# Set API key
openai.api_key = os.environ['OPENAI_API_KEY']

# Generate embeddings
response = openai.Embedding.create(
    input="Your text here",
    model="text-embedding-ada-002"
)
embeddings = response['data'][0]['embedding']
```

### Hugging Face

```python
from transformers import pipeline
import os

# Set token
hf_token = os.environ['HUGGINGFACE_TOKEN']

# Load model
classifier = pipeline(
    "sentiment-analysis",
    use_auth_token=hf_token
)

# Use model
result = classifier("I love this!")
```

### MLflow Tracking

#### Creating MLflow Secret

```bash
kubectl create secret generic mlflow-credentials \
  --from-literal=username=mlflow-user \
  --from-literal=password=mlflow-password \
  -n default

kubectl create configmap mlflow-config \
  --from-literal=MLFLOW_TRACKING_URI=https://mlflow.example.com \
  -n default
```

#### NotebookValidationJob Configuration

```yaml
spec:
  podConfig:
    envFrom:
      - configMapRef:
          name: mlflow-config
    env:
      - name: MLFLOW_TRACKING_USERNAME
        valueFrom:
          secretKeyRef:
            name: mlflow-credentials
            key: username
      - name: MLFLOW_TRACKING_PASSWORD
        valueFrom:
          secretKeyRef:
            name: mlflow-credentials
            key: password
```

#### Notebook Code

```python
import mlflow
import os

# Set tracking URI and credentials
mlflow.set_tracking_uri(os.environ['MLFLOW_TRACKING_URI'])
os.environ['MLFLOW_TRACKING_USERNAME'] = os.environ['MLFLOW_TRACKING_USERNAME']
os.environ['MLFLOW_TRACKING_PASSWORD'] = os.environ['MLFLOW_TRACKING_PASSWORD']

# Start experiment
mlflow.set_experiment("my-experiment")

with mlflow.start_run():
    # Log parameters
    mlflow.log_param("learning_rate", 0.01)
    
    # Train model
    model = train_model()
    
    # Log metrics
    mlflow.log_metric("accuracy", 0.95)
    
    # Log model
    mlflow.sklearn.log_model(model, "model")
```

## Multi-Service Examples

See `config/samples/mlops_v1alpha1_notebookvalidationjob_multi_service.yaml` for a complete example combining:
- AWS S3 for data storage
- PostgreSQL for feature store
- MLflow for experiment tracking
- OpenAI for embeddings
- Hugging Face for models

### End-to-End ML Pipeline

```python
import boto3
import psycopg2
import mlflow
import openai
import pandas as pd
import os
from sklearn.ensemble import RandomForestClassifier

# 1. Load data from S3
s3 = boto3.client('s3')
s3.download_file(os.environ['S3_BUCKET'], 'data/train.csv', 'train.csv')
df = pd.read_csv('train.csv')

# 2. Load features from database
conn = psycopg2.connect(
    host=os.environ['DB_HOST'],
    database=os.environ['DB_NAME'],
    user=os.environ['DB_USER'],
    password=os.environ['DB_PASSWORD']
)
features_df = pd.read_sql("SELECT * FROM features", conn)
conn.close()

# 3. Generate embeddings with OpenAI
openai.api_key = os.environ['OPENAI_API_KEY']
embeddings = []
for text in df['text']:
    response = openai.Embedding.create(input=text, model="text-embedding-ada-002")
    embeddings.append(response['data'][0]['embedding'])
df['embeddings'] = embeddings

# 4. Train model and track with MLflow
mlflow.set_tracking_uri(os.environ['MLFLOW_TRACKING_URI'])
mlflow.set_experiment("fraud-detection")

with mlflow.start_run():
    model = RandomForestClassifier()
    model.fit(df[['embeddings']], df['label'])

    mlflow.log_param("model_type", "RandomForest")
    mlflow.log_metric("accuracy", 0.95)
    mlflow.sklearn.log_model(model, "model")

# 5. Save results to S3
s3.upload_file('model.pkl', os.environ['S3_BUCKET'], 'models/fraud-detection.pkl')
```

## External Secrets Operator (ESO)

External Secrets Operator syncs secrets from external secret management systems (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager, Vault) into Kubernetes Secrets.

### Prerequisites

1. Install External Secrets Operator:

```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace
```

2. Configure cloud provider credentials (example for AWS):

```bash
kubectl create secret generic aws-secret-manager-credentials \
  --from-literal=access-key-id=AKIA... \
  --from-literal=secret-access-key=wJalr... \
  -n default
```

### AWS Secrets Manager Integration

#### Step 1: Create SecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: default
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: aws-secret-manager-credentials
            key: access-key-id
          secretAccessKeySecretRef:
            name: aws-secret-manager-credentials
            key: secret-access-key
```

#### Step 2: Create ExternalSecret

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: database-credentials
    creationPolicy: Owner
  data:
    - secretKey: DB_HOST
      remoteRef:
        key: prod/database/postgres
        property: host
    - secretKey: DB_USER
      remoteRef:
        key: prod/database/postgres
        property: username
    - secretKey: DB_PASSWORD
      remoteRef:
        key: prod/database/postgres
        property: password
```

#### Step 3: Use in NotebookValidationJob

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: notebook-with-eso
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/feature-engineering.ipynb"

  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
    envFrom:
      - secretRef:
          name: database-credentials  # Synced by ESO
```

### Azure Key Vault Integration

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: azure-keyvault
  namespace: default
spec:
  provider:
    azurekv:
      vaultUrl: "https://my-vault.vault.azure.net"
      authType: ServicePrincipal
      authSecretRef:
        clientId:
          name: azure-credentials
          key: client-id
        clientSecret:
          name: azure-credentials
          key: client-secret
      tenantId: "tenant-id-here"
```

### GCP Secret Manager Integration

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-manager
  namespace: default
spec:
  provider:
    gcpsm:
      projectID: "my-project-id"
      auth:
        secretRef:
          secretAccessKeySecretRef:
            name: gcp-credentials
            key: service-account-key
```

## Vault Integration

HashiCorp Vault provides dynamic, short-lived credentials with automatic rotation.

### Vault Agent Sidecar Pattern

#### Step 1: Configure Vault Kubernetes Auth

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc:443" \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
    token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token
```

#### Step 2: Create Vault Policy

```hcl
# database-read-policy.hcl
path "database/creds/readonly" {
  capabilities = ["read"]
}
```

```bash
vault policy write database-read database-read-policy.hcl
```

#### Step 3: Create Vault Role

```bash
vault write auth/kubernetes/role/notebook-validator \
    bound_service_account_names=jupyter-notebook-validator-runner \
    bound_service_account_namespaces=default \
    policies=database-read \
    ttl=1h
```

#### Step 4: Configure NotebookValidationJob with Vault Annotations

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: notebook-with-vault
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "notebook-validator"
    vault.hashicorp.com/agent-inject-secret-database: "database/creds/readonly"
    vault.hashicorp.com/agent-inject-template-database: |
      {{- with secret "database/creds/readonly" -}}
      export DB_USER="{{ .Data.username }}"
      export DB_PASSWORD="{{ .Data.password }}"
      {{- end }}
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/feature-engineering.ipynb"

  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
    serviceAccountName: "jupyter-notebook-validator-runner"
```

**Note:** Vault Agent sidecar automatically injects credentials and handles rotation.

## Security Best Practices

### 1. Use Least Privilege

**DO:**
- Create read-only database users for notebooks
- Use IAM roles with minimal permissions
- Restrict secret access with RBAC

**DON'T:**
- Use admin credentials in notebooks
- Grant broad permissions
- Share credentials across environments

### 2. Rotate Credentials Regularly

**Static Credentials:**
- Rotate quarterly at minimum
- Use automated rotation tools
- Track rotation in audit logs

**Dynamic Credentials:**
- Use Vault for short-lived credentials (TTL: 1-24 hours)
- Automatic rotation on each notebook run
- No manual rotation needed

### 3. Never Hardcode Credentials

**DO:**
```python
import os
api_key = os.environ['API_KEY']
```

**DON'T:**
```python
api_key = "sk-proj-abc123..."  # NEVER DO THIS
```

### 4. Use RBAC for Secret Access

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: notebook-secret-reader
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["aws-credentials", "database-credentials"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: notebook-secret-reader-binding
  namespace: default
subjects:
  - kind: ServiceAccount
    name: jupyter-notebook-validator-runner
    namespace: default
roleRef:
  kind: Role
  name: notebook-secret-reader
  apiGroup: rbac.authorization.k8s.io
```

### 5. Enable Audit Logging

Monitor secret access:
- Enable Kubernetes audit logs
- Track secret read operations
- Alert on suspicious access patterns

### 6. Encrypt Secrets at Rest

Ensure Kubernetes secrets are encrypted:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: <base64-encoded-secret>
      - identity: {}
```

### 7. Use Pod Security Standards

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: validation-pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: notebook
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
            - ALL
        readOnlyRootFilesystem: true
```

## Troubleshooting

### Issue: "Secret not found"

**Symptoms:**
```
Error: secrets "aws-credentials" not found
```

**Solutions:**
1. Verify secret exists:
   ```bash
   kubectl get secret aws-credentials -n default
   ```

2. Check namespace matches:
   ```bash
   kubectl get notebookvalidationjob my-job -o yaml | grep namespace
   ```

3. Create secret if missing:
   ```bash
   kubectl create secret generic aws-credentials \
     --from-literal=access-key-id=AKIA... \
     -n default
   ```

### Issue: "Permission denied" accessing secret

**Symptoms:**
```
Error: secrets "aws-credentials" is forbidden: User "system:serviceaccount:default:jupyter-notebook-validator-runner" cannot get resource "secrets"
```

**Solutions:**
1. Check RBAC permissions:
   ```bash
   kubectl auth can-i get secrets --as=system:serviceaccount:default:jupyter-notebook-validator-runner -n default
   ```

2. Create Role and RoleBinding (see Security Best Practices section)

### Issue: Environment variables not available in notebook

**Symptoms:**
```python
KeyError: 'AWS_ACCESS_KEY_ID'
```

**Solutions:**
1. Verify pod has environment variables:
   ```bash
   kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].env}'
   ```

2. Check secret key names match:
   ```bash
   kubectl get secret aws-credentials -o jsonpath='{.data}' | jq
   ```

3. Verify envFrom syntax:
   ```yaml
   envFrom:
     - secretRef:
         name: aws-credentials  # Correct
   # NOT:
   # envFrom:
   #   - secret: aws-credentials  # Wrong
   ```

### Issue: ESO ExternalSecret not syncing

**Symptoms:**
```
ExternalSecret status: SecretSyncedError
```

**Solutions:**
1. Check SecretStore status:
   ```bash
   kubectl get secretstore aws-secrets-manager -o yaml
   ```

2. Verify cloud provider credentials:
   ```bash
   kubectl get secret aws-secret-manager-credentials -o yaml
   ```

3. Check ESO logs:
   ```bash
   kubectl logs -n external-secrets-system deployment/external-secrets
   ```

4. Verify IAM permissions (AWS example):
   - `secretsmanager:GetSecretValue`
   - `secretsmanager:DescribeSecret`

### Issue: Vault Agent sidecar not injecting secrets

**Symptoms:**
- No Vault sidecar container in pod
- Secrets not available at expected path

**Solutions:**
1. Verify Vault annotations:
   ```bash
   kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations}' | jq
   ```

2. Check ServiceAccount has Vault role:
   ```bash
   vault read auth/kubernetes/role/notebook-validator
   ```

3. Verify Vault Agent Injector is running:
   ```bash
   kubectl get pods -n vault
   ```

4. Check Vault Agent logs:
   ```bash
   kubectl logs <pod-name> -c vault-agent
   ```

### Issue: Credentials work locally but not in operator

**Symptoms:**
- Notebook runs successfully locally
- Fails with authentication errors in operator

**Solutions:**
1. Check if notebook uses hardcoded credentials:
   ```python
   # Search for hardcoded values
   grep -r "AKIA" notebooks/
   grep -r "password.*=" notebooks/
   ```

2. Ensure notebook reads from environment:
   ```python
   import os
   # DO THIS:
   api_key = os.environ.get('API_KEY')
   # NOT THIS:
   api_key = "hardcoded-value"
   ```

3. Test environment variable availability:
   ```python
   import os
   print("Available env vars:", list(os.environ.keys()))
   ```

## Additional Resources

- [ADR-014: Notebook Credential Injection Strategy](adrs/014-notebook-credential-injection-strategy.md)
- [ADR-009: Secret Management Strategy](adrs/009-secret-management-strategy.md)
- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
- [External Secrets Operator Documentation](https://external-secrets.io/)
- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Sample Manifests](../config/samples/)

## Support

For issues or questions:
1. Check this troubleshooting guide
2. Review sample manifests in `config/samples/`
3. Check operator logs: `kubectl logs -n jupyter-notebook-validator-system deployment/jupyter-notebook-validator-controller-manager`
4. Open an issue on GitHub with:
   - NotebookValidationJob YAML
   - Pod logs
   - Error messages
   - Steps to reproduce

