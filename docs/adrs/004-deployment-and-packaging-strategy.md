# ADR 004: Deployment and Packaging Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must be packaged and deployed in a way that:

1. Provides excellent user experience for installation and upgrades
2. Integrates seamlessly with OpenShift's OperatorHub
3. Supports both enterprise (OpenShift) and community (Kubernetes) users
4. Enables automated lifecycle management (installation, upgrades, uninstallation)
5. Follows cloud-native best practices

### Current Requirements

- **Primary Target**: OpenShift 4.18+ with OperatorHub integration
- **Secondary Target**: Vanilla Kubernetes clusters
- **User Personas**:
  - **Platform Admins**: Install and manage operator cluster-wide
  - **Namespace Users**: Use operator in their namespaces
  - **CI/CD Systems**: Automate operator deployment in pipelines

### Technical Considerations

1. **Operator Lifecycle Manager (OLM)**: OpenShift's native operator management system
2. **Helm Charts**: Popular packaging format for Kubernetes applications
3. **Raw Manifests**: Simple YAML files for kubectl apply
4. **Container Registry**: Where operator images are stored and distributed

### Available Options

#### Option 1: OLM Bundle Only
- **Pros**: 
  - Native OpenShift integration
  - Automatic upgrades via OLM
  - Dependency management
  - OperatorHub catalog listing
- **Cons**: 
  - OLM not standard on vanilla Kubernetes
  - Requires OLM installation for K8s users
  - More complex packaging

#### Option 2: Helm Chart Only
- **Pros**: 
  - Works on any Kubernetes cluster
  - Popular and well-understood
  - Templating and values customization
  - Helm Hub listing
- **Cons**: 
  - Not native to OpenShift OperatorHub
  - Manual upgrade management
  - No automatic dependency resolution

#### Option 3: Raw Manifests Only
- **Pros**: 
  - Simplest approach
  - No additional tools required
  - Easy to understand and customize
- **Cons**: 
  - No templating or customization
  - Manual upgrade process
  - No dependency management
  - Poor user experience

#### Option 4: Hybrid Approach (OLM + Helm + Manifests)
- **Pros**: 
  - Best experience for each platform
  - Maximum flexibility for users
  - Supports all deployment scenarios
- **Cons**: 
  - Must maintain multiple packaging formats
  - Increased testing burden
  - More documentation required

## Decision

We will adopt a **Hybrid Packaging Strategy** with the following priority:

### Primary: OLM Bundle (OpenShift OperatorHub)
- **Target**: OpenShift 4.18+ users
- **Distribution**: Red Hat Ecosystem Catalog, OperatorHub.io
- **Format**: OLM bundle with ClusterServiceVersion (CSV)
- **Upgrades**: Automatic via OLM subscription channels

### Secondary: Helm Chart (Kubernetes Community)
- **Target**: Vanilla Kubernetes users, local development
- **Distribution**: Artifact Hub, GitHub Releases
- **Format**: Helm chart with customizable values
- **Upgrades**: Manual via `helm upgrade`

### Tertiary: Raw Manifests (Advanced Users)
- **Target**: CI/CD pipelines, GitOps workflows, advanced users
- **Distribution**: GitHub Releases, documentation
- **Format**: Kustomize-compatible YAML manifests
- **Upgrades**: Manual via `kubectl apply`

## Consequences

### Positive
- **Best-in-Class UX**: Each platform gets optimal deployment experience
- **Broad Adoption**: Supports OpenShift, Kubernetes, and GitOps workflows
- **Automatic Upgrades**: OLM users get seamless upgrades
- **Flexibility**: Users can choose deployment method that fits their workflow
- **Discoverability**: Listed in OperatorHub and Artifact Hub

### Negative
- **Maintenance Burden**: Must maintain three packaging formats
- **Testing Complexity**: Must test all deployment methods
- **Documentation Overhead**: Separate guides for each method
- **Release Coordination**: Must synchronize releases across formats

### Neutral
- **Tooling**: Operator SDK supports generating all formats
- **CI/CD**: Can automate packaging for all formats

## Implementation Notes

### OLM Bundle Structure

```
bundle/
├── manifests/
│   ├── jupyter-notebook-validator-operator.clusterserviceversion.yaml
│   ├── mlops.dev_notebookvalidationjobs.yaml  # CRD
│   └── operator_rbac.yaml
├── metadata/
│   └── annotations.yaml
├── tests/
│   └── scorecard/
└── Dockerfile  # Bundle image
```

### ClusterServiceVersion (CSV) Example

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: jupyter-notebook-validator-operator.v0.1.0
  namespace: placeholder
spec:
  displayName: Jupyter Notebook Validator Operator
  description: |
    Kubernetes-native operator for validating Jupyter Notebooks in MLOps workflows.
  version: 0.1.0
  maturity: alpha
  provider:
    name: Your Organization
  maintainers:
    - name: Platform Team
      email: platform@example.com
  links:
    - name: Documentation
      url: https://github.com/your-org/jupyter-notebook-validator-operator
  icon:
    - base64data: <base64-encoded-icon>
      mediatype: image/png
  keywords:
    - jupyter
    - notebook
    - validation
    - mlops
  installModes:
    - type: OwnNamespace
      supported: true
    - type: SingleNamespace
      supported: true
    - type: MultiNamespace
      supported: false
    - type: AllNamespaces
      supported: true
  install:
    strategy: deployment
    spec:
      permissions: []
      clusterPermissions:
        - serviceAccountName: jupyter-notebook-validator-operator
          rules:
            - apiGroups: ["mlops.dev"]
              resources: ["notebookvalidationjobs"]
              verbs: ["*"]
            - apiGroups: [""]
              resources: ["pods", "configmaps", "secrets"]
              verbs: ["get", "list", "watch", "create", "update", "delete"]
      deployments:
        - name: jupyter-notebook-validator-operator
          spec:
            replicas: 1
            selector:
              matchLabels:
                name: jupyter-notebook-validator-operator
            template:
              metadata:
                labels:
                  name: jupyter-notebook-validator-operator
              spec:
                serviceAccountName: jupyter-notebook-validator-operator
                containers:
                  - name: operator
                    image: quay.io/your-org/jupyter-notebook-validator-operator:v0.1.0
                    command:
                      - /manager
                    env:
                      - name: WATCH_NAMESPACE
                        valueFrom:
                          fieldRef:
                            fieldPath: metadata.annotations['olm.targetNamespaces']
  customresourcedefinitions:
    owned:
      - name: notebookvalidationjobs.mlops.dev
        version: v1alpha1
        kind: NotebookValidationJob
        displayName: Notebook Validation Job
        description: Defines a notebook validation job
```

### Helm Chart Structure

```
charts/jupyter-notebook-validator-operator/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── serviceaccount.yaml
│   ├── role.yaml
│   ├── rolebinding.yaml
│   ├── crd.yaml
│   └── NOTES.txt
└── README.md
```

### Helm Values Example

```yaml
# values.yaml
replicaCount: 1

image:
  repository: quay.io/your-org/jupyter-notebook-validator-operator
  tag: v0.1.0
  pullPolicy: IfNotPresent

serviceAccount:
  create: true
  name: jupyter-notebook-validator-operator

rbac:
  create: true

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector: {}
tolerations: []
affinity: {}
```

### Kustomize Structure

```
config/
├── default/
│   ├── kustomization.yaml
│   └── manager_auth_proxy_patch.yaml
├── manager/
│   ├── kustomization.yaml
│   └── manager.yaml
├── rbac/
│   ├── kustomization.yaml
│   ├── role.yaml
│   ├── role_binding.yaml
│   └── service_account.yaml
├── crd/
│   ├── kustomization.yaml
│   └── bases/
│       └── mlops.dev_notebookvalidationjobs.yaml
└── samples/
    └── mlops_v1alpha1_notebookvalidationjob.yaml
```

### Distribution Channels

#### OLM Channels
```yaml
# bundle/metadata/annotations.yaml
annotations:
  operators.operatorframework.io.bundle.channels.v1: alpha,beta,stable
  operators.operatorframework.io.bundle.channel.default.v1: stable
```

- **alpha**: Experimental features, may have breaking changes
- **beta**: Stable API, release candidates
- **stable**: Production-ready, long-term support

### CI/CD Pipeline

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-publish:
    runs-on: ubuntu-latest
    steps:
      - name: Build operator image
        run: make docker-build docker-push IMG=${{ env.IMAGE }}
      
      - name: Build OLM bundle
        run: make bundle bundle-build bundle-push
      
      - name: Package Helm chart
        run: helm package charts/jupyter-notebook-validator-operator
      
      - name: Publish to Artifact Hub
        run: helm push jupyter-notebook-validator-operator-*.tgz oci://registry.example.com/charts
      
      - name: Generate raw manifests
        run: kustomize build config/default > release/install.yaml
      
      - name: Create GitHub Release
        uses: actions/create-release@v1
        with:
          files: |
            release/install.yaml
            jupyter-notebook-validator-operator-*.tgz
```

### Installation Documentation

#### OpenShift (OLM)
```bash
# Install via OperatorHub UI or CLI
oc create -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jupyter-notebook-validator-operator
  namespace: openshift-operators
spec:
  channel: stable
  name: jupyter-notebook-validator-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
EOF
```

#### Kubernetes (Helm)
```bash
# Add Helm repository
helm repo add jupyter-validator https://your-org.github.io/jupyter-notebook-validator-operator

# Install operator
helm install jupyter-validator jupyter-validator/jupyter-notebook-validator-operator \
  --namespace jupyter-validator-system \
  --create-namespace
```

#### Kubernetes (Kustomize)
```bash
# Install via kustomize
kubectl apply -k github.com/your-org/jupyter-notebook-validator-operator/config/default?ref=v0.1.0
```

## References

- [Operator Lifecycle Manager](https://olm.operatorframework.io/)
- [Operator SDK Bundle](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/)
- [Helm Charts](https://helm.sh/docs/topics/charts/)
- [Kustomize](https://kustomize.io/)
- [OperatorHub.io](https://operatorhub.io/)
- [Artifact Hub](https://artifacthub.io/)

## Related ADRs

- ADR 001: Operator Framework and SDK Version
- ADR 002: Platform Version Support Strategy
- ADR 008: CI/CD Pipeline Integration

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial decision |

