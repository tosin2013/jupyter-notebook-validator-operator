# ADR 015: Environment-Variable Pattern for Notebook Credentials

## Status
Accepted

## Context

ADR-014 established a multi-tier credential injection strategy. This ADR focuses on **Tier 1: Static Secrets in Environment Variables**, which is the on-ramp for most users.

### Problem

Without standardized environment variable naming conventions, users will:
- Use inconsistent naming (`AWS_KEY` vs `AWS_ACCESS_KEY_ID`)
- Create non-portable notebooks (hardcoded names)
- Struggle with documentation and examples
- Face integration issues with standard libraries (boto3, psycopg2)

### Industry Standards

Most libraries expect specific environment variable names:
- **AWS SDK (boto3)**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
- **PostgreSQL (psycopg2)**: `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`
- **MySQL**: `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`
- **OpenAI**: `OPENAI_API_KEY`
- **Hugging Face**: `HUGGINGFACE_TOKEN`

## Decision

We standardize environment variable naming conventions for common services, following industry standards where they exist.

### AWS S3 Credentials

**Standard Names**:
```yaml
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
  - name: AWS_REGION
    value: us-east-1
  - name: S3_BUCKET  # Optional: application-specific
    value: my-training-data
```

**Notebook Usage**:
```python
import boto3
import os

# boto3 automatically reads AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
s3_client = boto3.client('s3')

bucket = os.environ.get('S3_BUCKET', 'default-bucket')
s3_client.download_file(bucket, 'data.csv', '/tmp/data.csv')
```

### Database Credentials (PostgreSQL)

**Standard Names**:
```yaml
env:
  - name: DB_HOST
    valueFrom:
      secretKeyRef:
        name: database-credentials
        key: host
  - name: DB_PORT
    value: "5432"
  - name: DB_NAME
    valueFrom:
      secretKeyRef:
        name: database-credentials
        key: database
  - name: DB_USER
    valueFrom:
      secretKeyRef:
        name: database-credentials
        key: username
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: database-credentials
        key: password
```

**Notebook Usage**:
```python
import psycopg2
import os

conn = psycopg2.connect(
    host=os.environ['DB_HOST'],
    port=os.environ.get('DB_PORT', '5432'),
    database=os.environ['DB_NAME'],
    user=os.environ['DB_USER'],
    password=os.environ['DB_PASSWORD']
)
```

### Database Credentials (MySQL)

**Standard Names**:
```yaml
env:
  - name: MYSQL_HOST
    valueFrom:
      secretKeyRef:
        name: mysql-credentials
        key: host
  - name: MYSQL_PORT
    value: "3306"
  - name: MYSQL_DATABASE
    valueFrom:
      secretKeyRef:
        name: mysql-credentials
        key: database
  - name: MYSQL_USER
    valueFrom:
      secretKeyRef:
        name: mysql-credentials
        key: username
  - name: MYSQL_PASSWORD
    valueFrom:
      secretKeyRef:
        name: mysql-credentials
        key: password
```

### API Keys

**Standard Names**:
```yaml
env:
  # OpenAI
  - name: OPENAI_API_KEY
    valueFrom:
      secretKeyRef:
        name: api-keys
        key: openai-api-key
  
  # Hugging Face
  - name: HUGGINGFACE_TOKEN
    valueFrom:
      secretKeyRef:
        name: api-keys
        key: huggingface-token
  
  # MLflow
  - name: MLFLOW_TRACKING_URI
    value: https://mlflow.example.com
  - name: MLFLOW_USERNAME
    valueFrom:
      secretKeyRef:
        name: mlflow-credentials
        key: username
  - name: MLFLOW_PASSWORD
    valueFrom:
      secretKeyRef:
        name: mlflow-credentials
        key: password
```

**Notebook Usage**:
```python
import openai
import os

# OpenAI library automatically reads OPENAI_API_KEY
openai.api_key = os.environ['OPENAI_API_KEY']

response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### Naming Convention Rules

1. **Use UPPERCASE**: All environment variables should be UPPERCASE
2. **Use UNDERSCORES**: Separate words with underscores (`AWS_ACCESS_KEY_ID`, not `AwsAccessKeyId`)
3. **Follow Industry Standards**: Use standard names where they exist (boto3, psycopg2)
4. **Prefix by Service**: Group related variables by service prefix (`AWS_*`, `DB_*`, `MYSQL_*`)
5. **Avoid Abbreviations**: Use full words (`DATABASE`, not `DB` - except where industry standard)
6. **Be Explicit**: `AWS_ACCESS_KEY_ID` is better than `AWS_KEY`

### Secret Structure

**Kubernetes Secret Template**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
type: Opaque
stringData:
  access-key-id: AKIAIOSFODNN7EXAMPLE
  secret-access-key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**Key Naming in Secrets**:
- Use **kebab-case** for secret keys (`access-key-id`, not `accessKeyId`)
- Use descriptive names (`access-key-id`, not `key1`)
- Match the environment variable name (lowercase, hyphens instead of underscores)

## Consequences

### Positive

1. **Consistency**: All users follow the same naming conventions
2. **Portability**: Notebooks work across different environments
3. **Compatibility**: Works with standard libraries (boto3, psycopg2)
4. **Documentation**: Easy to document and teach
5. **Discoverability**: Users can find examples easily

### Negative

1. **Rigidity**: Users must follow conventions (but can override if needed)
2. **Learning Curve**: Users need to learn the conventions

### Neutral

1. **No Code Changes**: Operator already supports arbitrary environment variables
2. **Optional**: Users can still use custom names if needed

## Implementation

### Phase 1: Documentation
- [x] Document naming conventions in this ADR
- [ ] Create comprehensive examples in credential guide
- [ ] Create secret templates for common services

### Phase 2: Examples
- [ ] Create example CRD manifests with standard names
- [ ] Create example notebooks using standard names
- [ ] Create secret templates

### Phase 3: Validation
- [ ] Test with real notebooks
- [ ] Verify compatibility with standard libraries
- [ ] Gather user feedback

## Examples

### Complete Example: S3 Data Pipeline

**1. Create Secret**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: notebook-validation
type: Opaque
stringData:
  access-key-id: AKIAIOSFODNN7EXAMPLE
  secret-access-key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

**2. Create NotebookValidationJob**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: s3-pipeline-validation
  namespace: notebook-validation
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: s3-data-pipeline.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
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
      - name: AWS_REGION
        value: us-east-1
      - name: S3_BUCKET
        value: my-training-data
```

**3. Notebook Code**:
```python
# Cell 1: Import libraries
import boto3
import pandas as pd
import os

# Cell 2: Connect to S3 (boto3 reads AWS_* env vars automatically)
s3_client = boto3.client('s3')
bucket = os.environ['S3_BUCKET']

# Cell 3: Load data
s3_client.download_file(bucket, 'train.csv', '/tmp/train.csv')
df = pd.read_csv('/tmp/train.csv')

# Cell 4: Process data
# ... training logic ...

# Cell 5: Save model
s3_client.upload_file('/tmp/model.pkl', bucket, 'models/model.pkl')
```

## Alternatives Considered

### Alternative 1: No Standardization
**Rejected**: Leads to inconsistency, confusion, and non-portable notebooks

### Alternative 2: Custom Naming Scheme
**Rejected**: Incompatible with standard libraries, higher learning curve

### Alternative 3: Only Use Industry Standards
**Rejected**: Some services don't have standards (e.g., generic databases)

## Related ADRs

- **ADR-014**: Notebook Credential Injection Strategy (overall strategy)
- **ADR-016**: External Secret Operator Integration (Tier 2)
- **ADR-017**: Vault Dynamic-Secrets Injection Pattern (Tier 3)
- **ADR-019**: RBAC & Pod Security Policies (access control)

## References

- [AWS SDK Environment Variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html)
- [PostgreSQL Environment Variables](https://www.postgresql.org/docs/current/libpq-envars.html)
- [MySQL Environment Variables](https://dev.mysql.com/doc/refman/8.0/en/environment-variables.html)
- [OpenAI Python Library](https://github.com/openai/openai-python)
- [Hugging Face Transformers](https://huggingface.co/docs/transformers/installation)
- [12-Factor App: Config](https://12factor.net/config)

## Revision History

- **2025-11-08**: Initial version (Tosin Akinosho)

