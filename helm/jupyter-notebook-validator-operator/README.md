# Jupyter Notebook Validator Operator Helm Chart

A Kubernetes operator for validating Jupyter notebooks with ML model training and validation capabilities.

## TL;DR

```bash
# Add the Helm repository (when published)
helm repo add jupyter-validator https://tosin2013.github.io/jupyter-notebook-validator-operator

# Install the operator
helm install jupyter-validator jupyter-validator/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace
```

## Introduction

This chart deploys the Jupyter Notebook Validator Operator on a Kubernetes or OpenShift cluster using Helm.

### Features

- ✅ **Notebook Validation**: Execute and validate Jupyter notebooks
- ✅ **Golden Notebook Comparison**: Compare outputs against reference notebooks
- ✅ **ML Model Training**: Train and validate machine learning models
- ✅ **Tekton Integration**: Build custom environments with Tekton Pipelines
- ✅ **Git Integration**: Clone notebooks from Git repositories (HTTP/SSH)
- ✅ **External Secrets**: Integrate with External Secrets Operator
- ✅ **Prometheus Metrics**: Monitor validation jobs
- ✅ **OpenShift Support**: Native OpenShift 4.18+ support

## Prerequisites

### Required
- Kubernetes 1.31+ or OpenShift 4.18+
- Helm 3.8+
- **cert-manager v1.13+** (required for webhook certificates)

### Optional
- Tekton Pipelines v0.65.0+ (for build features)
- External Secrets Operator v0.9.0+ (for secret management)
- Prometheus Operator (for metrics and alerts)

### Installing cert-manager

If cert-manager is not already installed:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Verify cert-manager is running
kubectl get pods -n cert-manager
```

For OpenShift, cert-manager is typically pre-installed. Verify with:

```bash
oc get pods -n cert-manager
```

## Installing the Chart

### Basic Installation

```bash
helm install jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace
```

### Installation with Custom Values

```bash
helm install jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace \
  --values custom-values.yaml
```

### OpenShift Installation

```bash
helm install jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace \
  --set openshift.enabled=true \
  --set openshift.scc.create=true
```

## Uninstalling the Chart

```bash
helm uninstall jupyter-validator --namespace jupyter-validator-system
```

**Note**: By default, CRDs are kept on uninstall to prevent data loss. To remove CRDs:

```bash
kubectl delete crd notebookvalidationjobs.mlops.mlops.dev
```

## Configuration

### Key Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Operator image repository | `quay.io/tosin2013/jupyter-notebook-validator-operator` |
| `image.tag` | Operator image tag | `""` (uses appVersion) |
| `crds.install` | Install CRDs with chart | `true` |
| `crds.keep` | Keep CRDs on uninstall | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `rbac.scope` | RBAC scope (cluster/namespace) | `cluster` |
| `tekton.enabled` | Enable Tekton integration | `true` |
| `prometheus.enabled` | Create ServiceMonitor | `false` |
| `openshift.enabled` | Enable OpenShift features | `false` |

### Resource Configuration

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### Validation Job Defaults

```yaml
validation:
  timeout: "30m"
  resources:
    limits:
      cpu: "2"
      memory: "4Gi"
    requests:
      cpu: "500m"
      memory: "1Gi"
  pythonVersion: "3.11"
```

### Prometheus Monitoring

```yaml
prometheus:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
```

### OpenShift Configuration

```yaml
openshift:
  enabled: true
  scc:
    create: true
    name: "jupyter-notebook-validator-scc"
  route:
    enabled: true
    host: "jupyter-validator.apps.cluster.example.com"
    tls:
      enabled: true
      termination: edge
```

## Working with Private Git Repositories

The operator supports both public and private Git repositories. For private repositories, you need to create a Kubernetes Secret with your Git credentials.

### Creating Git Credentials Secret

#### Option 1: HTTPS with Personal Access Token (Recommended)

```bash
# Create secret with GitHub Personal Access Token
kubectl create secret generic git-credentials \
  --from-literal=username=your-github-username \
  --from-literal=password=ghp_your_personal_access_token \
  -n default

# Or using a file
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: default
type: Opaque
stringData:
  username: "your-github-username"
  password: "ghp_your_personal_access_token"
EOF
```

**Creating a GitHub Personal Access Token:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select scopes: `repo` (for private repos) or `public_repo` (for public repos)
4. Copy the token and use it as the password

#### Option 2: SSH Key Authentication

```bash
# Create secret with SSH private key
kubectl create secret generic git-ssh-credentials \
  --from-file=ssh-privatekey=/path/to/id_rsa \
  --from-literal=known_hosts="$(ssh-keyscan github.com)" \
  -n default
```

#### Option 3: Using External Secrets Operator (Advanced)

If you have External Secrets Operator installed:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: git-credentials
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: git-credentials
    creationPolicy: Owner
  data:
  - secretKey: username
    remoteRef:
      key: git/credentials
      property: username
  - secretKey: password
    remoteRef:
      key: git/credentials
      property: token
```

### Using Credentials in NotebookValidationJob

Once you've created the secret, reference it in your NotebookValidationJob:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-private-repo
  namespace: default
spec:
  notebook:
    git:
      url: "https://github.com/your-org/private-notebooks.git"
      ref: "main"
      credentialsSecret: "git-credentials"  # Reference the secret
    path: "notebooks/analysis.ipynb"

  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"

  timeout: "10m"
```

### SSH URL Example

For SSH authentication:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-ssh-repo
  namespace: default
spec:
  notebook:
    git:
      url: "git@github.com:your-org/private-notebooks.git"
      ref: "main"
      credentialsSecret: "git-ssh-credentials"
    path: "notebooks/analysis.ipynb"

  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "1Gi"
        cpu: "1000m"

  timeout: "10m"
```

### Best Practices for Git Credentials

1. **Use Personal Access Tokens** instead of passwords for HTTPS
2. **Limit token scope** to only what's needed (e.g., `repo` for private repos)
3. **Rotate tokens regularly** and update secrets
4. **Use namespace-specific secrets** to limit access
5. **Consider External Secrets Operator** for centralized secret management
6. **Use SSH keys** for automated systems and CI/CD pipelines
7. **Never commit credentials** to Git repositories

### Troubleshooting Git Authentication

If your validation job fails with git clone errors:

```bash
# Check if secret exists
kubectl get secret git-credentials -n default

# Verify secret contents (base64 encoded)
kubectl get secret git-credentials -n default -o yaml

# Check validation pod logs
kubectl logs <validation-pod-name> -n default -c git-clone

# Common errors:
# - "Authentication failed": Wrong username/password or token
# - "Repository not found": Check URL and permissions
# - "Permission denied (publickey)": SSH key not configured correctly
```

## Examples

### Example 1: Basic Notebook Validation

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-basic-notebook
spec:
  notebookRepo: "https://github.com/example/notebooks.git"
  notebookPath: "examples/basic-analysis.ipynb"
  branch: "main"
```

### Example 2: Golden Notebook Comparison

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-with-golden
spec:
  notebookRepo: "https://github.com/example/notebooks.git"
  notebookPath: "examples/ml-training.ipynb"
  goldenNotebookPath: "examples/golden/ml-training-golden.ipynb"
  comparisonMode: "advanced"
```

### Example 3: Custom Build with Tekton

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-custom-build
spec:
  notebookRepo: "https://github.com/example/notebooks.git"
  notebookPath: "examples/custom-deps.ipynb"
  buildStrategy: "tekton"
  customDockerfile: |
    FROM python:3.11-slim
    RUN pip install tensorflow scikit-learn pandas
```

## Upgrading

### Upgrade to Latest Version

```bash
helm upgrade jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system
```

### Upgrade with New Values

```bash
helm upgrade jupyter-validator ./helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --values new-values.yaml
```

## Troubleshooting

### Check Operator Logs

```bash
kubectl logs -n jupyter-validator-system \
  -l control-plane=controller-manager \
  --tail=100
```

### Check Validation Job Status

```bash
kubectl get notebookvalidationjobs -A
kubectl describe notebookvalidationjob <name> -n <namespace>
```

### Check Tekton Pipeline Runs

```bash
kubectl get pipelineruns -n <namespace>
kubectl logs -n <namespace> <pipelinerun-name> --all-containers
```

## Documentation

- [Architecture Overview](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/docs/ARCHITECTURE_OVERVIEW.md)
- [Integration Testing](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/docs/INTEGRATION_TESTING.md)
- [ADRs](https://github.com/tosin2013/jupyter-notebook-validator-operator/tree/main/docs/adrs)
- [GitHub Secrets Setup](https://github.com/tosin2013/jupyter-notebook-validator-operator/blob/main/docs/GITHUB_SECRETS_SETUP.md)

## Support

- **Issues**: [GitHub Issues](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/tosin2013/jupyter-notebook-validator-operator/discussions)

## License

Apache License 2.0

## Maintainers

- Tosin Akinosho ([@tosin2013](https://github.com/tosin2013))

