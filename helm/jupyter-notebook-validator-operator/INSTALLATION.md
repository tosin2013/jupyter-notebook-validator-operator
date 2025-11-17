# Jupyter Notebook Validator Operator - Installation Guide

This guide walks you through deploying the Jupyter Notebook Validator Operator to a fresh cluster using Helm.

## Prerequisites Checklist

Before installing, ensure you have:

- [ ] Kubernetes 1.31+ or OpenShift 4.18+ cluster access
- [ ] `kubectl` or `oc` CLI configured
- [ ] Helm 3.8+ installed
- [ ] cert-manager v1.13+ installed (see below)
- [ ] Cluster admin permissions (for CRD installation)

## Step 1: Verify cert-manager

The operator requires cert-manager for webhook certificates.

### Check if cert-manager is installed:

```bash
# For Kubernetes
kubectl get pods -n cert-manager

# For OpenShift
oc get pods -n cert-manager
```

### Install cert-manager (if not present):

```bash
# For Kubernetes
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-cainjector -n cert-manager
```

For OpenShift, cert-manager is typically pre-installed. If not, follow the [OpenShift cert-manager documentation](https://docs.openshift.com/container-platform/latest/security/cert_manager_operator/index.html).

## Step 2: Clone the Repository (or use Helm repo)

```bash
# Option A: Clone from GitHub
git clone https://github.com/tosin2013/jupyter-notebook-validator-operator.git
cd jupyter-notebook-validator-operator

# Option B: Add Helm repository (when published)
# helm repo add jupyter-validator https://tosin2013.github.io/jupyter-notebook-validator-operator
# helm repo update
```

## Step 3: Install the Operator

### For Kubernetes:

```bash
helm install jupyter-validator helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace \
  --set image.repository=quay.io/takinosh/jupyter-notebook-validator-operator \
  --set image.tag=latest \
  --wait \
  --timeout 5m
```

### For OpenShift:

```bash
helm install jupyter-validator helm/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace \
  --set openshift.enabled=true \
  --set prometheus.enabled=true \
  --set image.repository=quay.io/takinosh/jupyter-notebook-validator-operator \
  --set image.tag=latest \
  --wait \
  --timeout 5m
```

## Step 4: Verify Installation

### Check operator pod:

```bash
# For Kubernetes
kubectl get pods -n jupyter-validator-system

# For OpenShift
oc get pods -n jupyter-validator-system
```

Expected output:
```
NAME                                                              READY   STATUS    RESTARTS   AGE
jupyter-validator-notebook-validator-controller-manager-xxxxx     2/2     Running   0          2m
```

### Check CRDs:

```bash
kubectl get crd notebookvalidationjobs.mlops.mlops.dev
```

### Check webhooks:

```bash
kubectl get mutatingwebhookconfiguration,validatingwebhookconfiguration | grep jupyter
```

### Check cert-manager certificate:

```bash
kubectl get certificate,issuer -n jupyter-validator-system
```

Expected output:
```
NAME                                                                  READY   SECRET                  AGE
certificate.cert-manager.io/jupyter-validator-notebook-validator...  True    webhook-server-cert     2m

NAME                                                                  READY   AGE
issuer.cert-manager.io/jupyter-validator-notebook-validator...       True    2m
```

## Step 5: (Optional) Configure Git Credentials for Private Repositories

If you need to access private Git repositories, create a credentials secret:

```bash
# For HTTPS with Personal Access Token
kubectl create secret generic git-credentials \
  --from-literal=username=your-github-username \
  --from-literal=password=ghp_your_personal_access_token \
  -n default
```

See the [README.md](README.md#working-with-private-git-repositories) for detailed instructions on:
- Creating GitHub Personal Access Tokens
- SSH key authentication
- Using External Secrets Operator
- Best practices for credential management

## Step 6: Create Your First Validation Job

### For Public Repositories:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: hello-world-test
  namespace: default
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"

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
EOF
```

### For Private Repositories:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: private-repo-test
  namespace: default
spec:
  notebook:
    git:
      url: "https://github.com/your-org/private-notebooks.git"
      ref: "main"
      credentialsSecret: "git-credentials"  # Reference the secret created above
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
EOF
```

### Check validation job status:

```bash
kubectl get notebookvalidationjob hello-world-test -n default
kubectl describe notebookvalidationjob hello-world-test -n default
```

## Troubleshooting

### Operator pod not starting

Check logs:
```bash
kubectl logs -n jupyter-validator-system -l control-plane=controller-manager -c manager --tail=100
```

### Webhook errors

Verify cert-manager certificate:
```bash
kubectl get certificate -n jupyter-validator-system
kubectl describe certificate -n jupyter-validator-system
```

### Validation job fails

Check validation pod logs:
```bash
kubectl get pods -n default | grep validation
kubectl logs <validation-pod-name> -n default
```

## Uninstallation

```bash
# Uninstall the operator
helm uninstall jupyter-validator -n jupyter-validator-system

# Optional: Remove CRDs (this will delete all NotebookValidationJob resources!)
kubectl delete crd notebookvalidationjobs.mlops.mlops.dev

# Optional: Remove namespace
kubectl delete namespace jupyter-validator-system
```

## Next Steps

- Read the [Architecture Overview](../../docs/ARCHITECTURE_OVERVIEW.md)
- Explore [sample validation jobs](../../config/samples/)
- Set up [Prometheus monitoring](../../docs/OBSERVABILITY.md)
- Configure [Git credentials](../../docs/GIT_CREDENTIALS.md)

