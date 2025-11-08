# ADR 007: Distribution and Catalog Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator needs a clear distribution strategy to reach both enterprise OpenShift users and the broader Kubernetes community. This ADR defines where and how the operator will be published, ensuring maximum discoverability and ease of installation.

### Target Audiences

1. **Enterprise OpenShift Users**
   - Access operators via OpenShift OperatorHub (built-in)
   - Expect certified, supported operators
   - Require enterprise-grade security and compliance

2. **OpenShift Community Users**
   - Access operators via community catalogs
   - Value open-source and community-driven projects
   - Need easy installation and updates

3. **Kubernetes Community Users**
   - Access operators via OperatorHub.io, Artifact Hub
   - Use Helm charts or raw manifests
   - Prefer cloud-agnostic solutions

4. **CI/CD and GitOps Users**
   - Automate operator deployment
   - Use declarative manifests
   - Integrate with ArgoCD, Flux, Tekton

### Distribution Channels Overview

| Channel | Audience | Format | Certification | Auto-Updates |
|---------|----------|--------|---------------|--------------|
| Red Hat Ecosystem Catalog | Enterprise OpenShift | OLM Bundle | Required | Yes (OLM) |
| OpenShift OperatorHub (Community) | OpenShift Community | OLM Bundle | Optional | Yes (OLM) |
| OperatorHub.io | Kubernetes Community | OLM Bundle | No | Yes (OLM) |
| Artifact Hub | Kubernetes Community | Helm Chart | No | No |
| GitHub Releases | All Users | Manifests | No | No |
| Quay.io / Docker Hub | All Users | Container Images | No | N/A |

## Decision

We will implement a **Multi-Channel Distribution Strategy** with phased rollout aligned to our version support roadmap (ADR 006).

### Phase 1: OpenShift Foundation (Months 1-3)

#### Primary Distribution: OpenShift OperatorHub (Community Catalog)

**Target**: OpenShift 4.18 users
**Format**: OLM Bundle
**Certification**: Community (not certified initially)

##### Implementation Steps

1. **Create OLM Bundle**
   ```bash
   # Generate bundle
   make bundle IMG=quay.io/your-org/jupyter-notebook-validator-operator:v0.1.0
   
   # Build and push bundle image
   make bundle-build bundle-push BUNDLE_IMG=quay.io/your-org/jupyter-notebook-validator-operator-bundle:v0.1.0
   ```

2. **Submit to Community Operators**
   - Fork [community-operators](https://github.com/k8s-operatorhub/community-operators) repository
   - Add operator bundle to `operators/jupyter-notebook-validator-operator/`
   - Create pull request with bundle manifests
   - Pass automated CI checks
   - Await community review and approval

3. **Bundle Structure**
   ```
   operators/jupyter-notebook-validator-operator/
   ├── 0.1.0/
   │   ├── manifests/
   │   │   ├── jupyter-notebook-validator-operator.clusterserviceversion.yaml
   │   │   ├── mlops.dev_notebookvalidationjobs.yaml
   │   │   └── operator_rbac.yaml
   │   └── metadata/
   │       └── annotations.yaml
   └── jupyter-notebook-validator-operator.package.yaml
   ```

4. **Package Manifest**
   ```yaml
   # jupyter-notebook-validator-operator.package.yaml
   packageName: jupyter-notebook-validator-operator
   channels:
     - name: alpha
       currentCSV: jupyter-notebook-validator-operator.v0.1.0
   defaultChannel: alpha
   ```

#### Secondary Distribution: GitHub Releases

**Target**: Advanced users, CI/CD pipelines
**Format**: Raw Kubernetes manifests (Kustomize)

##### Release Assets
- `install.yaml` - Complete installation manifest
- `samples/` - Example NotebookValidationJob CRs
- `jupyter-notebook-validator-operator-v0.1.0.tar.gz` - Source code
- `CHANGELOG.md` - Release notes

#### Tertiary Distribution: Container Registry

**Target**: All users (dependency for other channels)
**Registry**: Quay.io (primary), Docker Hub (mirror)

##### Image Naming
- Operator: `quay.io/your-org/jupyter-notebook-validator-operator:v0.1.0`
- Bundle: `quay.io/your-org/jupyter-notebook-validator-operator-bundle:v0.1.0`

### Phase 2: OpenShift Expansion (Months 4-6)

#### Enhanced Distribution: Red Hat Ecosystem Catalog (Certification Track)

**Target**: Enterprise OpenShift users
**Format**: Certified OLM Bundle
**Certification**: Red Hat Certified Operator

##### Certification Requirements

1. **Technical Requirements**
   - Pass Red Hat Operator Certification tests
   - Support OpenShift 4.18, 4.19, 4.20
   - Follow OpenShift best practices
   - Security scanning (no critical vulnerabilities)
   - Documentation completeness

2. **Business Requirements**
   - Red Hat Partner Connect membership
   - Support agreement for certified operator
   - Legal agreements and licensing

3. **Certification Process**
   ```bash
   # Run certification preflight checks
   preflight check operator \
     quay.io/your-org/jupyter-notebook-validator-operator-bundle:v0.2.0 \
     --docker-config ~/.docker/config.json
   
   # Submit for certification
   # Follow Red Hat Partner Connect portal workflow
   ```

4. **Benefits of Certification**
   - Listed in Red Hat Ecosystem Catalog
   - Trusted by enterprise customers
   - Eligible for Red Hat support
   - Enhanced discoverability

#### Continued Distribution
- OpenShift OperatorHub (Community) - updated to v0.2.0
- GitHub Releases - updated with multi-version support
- Container Registry - updated images

### Phase 3: Kubernetes Community (Months 7-9)

#### Primary Distribution: OperatorHub.io

**Target**: Kubernetes community users
**Format**: OLM Bundle
**Certification**: Community

##### Implementation Steps

1. **Submit to OperatorHub.io**
   - Fork [community-operators](https://github.com/k8s-operatorhub/community-operators) repository
   - Add operator to `operators/` directory
   - Ensure compatibility with vanilla Kubernetes
   - Create pull request
   - Pass CI checks and community review

2. **Kubernetes-Specific Considerations**
   - Remove OpenShift-specific dependencies
   - Test on multiple K8s distributions (GKE, EKS, AKS, kind)
   - Provide clear installation instructions for vanilla K8s

#### Secondary Distribution: Artifact Hub (Helm Chart)

**Target**: Kubernetes users preferring Helm
**Format**: Helm Chart
**Repository**: Artifact Hub

##### Implementation Steps

1. **Create Helm Chart Repository**
   ```bash
   # Create GitHub Pages for Helm repo
   helm package charts/jupyter-notebook-validator-operator
   helm repo index . --url https://your-org.github.io/jupyter-notebook-validator-operator
   ```

2. **Publish to Artifact Hub**
   - Add repository to [Artifact Hub](https://artifacthub.io/)
   - Provide `artifacthub-repo.yml` metadata
   - Ensure chart passes Artifact Hub checks

3. **Chart Metadata**
   ```yaml
   # artifacthub-repo.yml
   repositoryID: <uuid>
   owners:
     - name: Your Organization
       email: platform@example.com
   ```

4. **Installation via Helm**
   ```bash
   # Add Helm repository
   helm repo add jupyter-validator https://your-org.github.io/jupyter-notebook-validator-operator
   
   # Install operator
   helm install jupyter-validator jupyter-validator/jupyter-notebook-validator-operator \
     --namespace jupyter-validator-system \
     --create-namespace
   ```

#### Tertiary Distribution: GitHub Marketplace

**Target**: GitHub users, CI/CD integrations
**Format**: GitHub Action (optional)

##### GitHub Action for Operator Deployment
```yaml
# .github/actions/deploy-operator/action.yml
name: 'Deploy Jupyter Notebook Validator Operator'
description: 'Deploy the operator to a Kubernetes cluster'
inputs:
  kubeconfig:
    description: 'Kubernetes config'
    required: true
  version:
    description: 'Operator version'
    required: false
    default: 'latest'
runs:
  using: 'composite'
  steps:
    - name: Deploy operator
      shell: bash
      run: |
        kubectl apply -f https://github.com/your-org/jupyter-notebook-validator-operator/releases/download/${{ inputs.version }}/install.yaml
```

## Consequences

### Positive
- **Maximum Reach**: Covers all major distribution channels
- **User Choice**: Users can choose their preferred installation method
- **Discoverability**: Listed in multiple catalogs and marketplaces
- **Enterprise Credibility**: Certification path for enterprise adoption
- **Community Engagement**: Open-source presence on OperatorHub.io and Artifact Hub
- **Automation-Friendly**: Supports GitOps and CI/CD workflows

### Negative
- **Maintenance Burden**: Must maintain multiple distribution formats
- **Certification Cost**: Red Hat certification requires resources and partnership
- **Synchronization**: Must keep all channels updated with each release
- **Documentation Overhead**: Installation guides for each channel
- **Support Complexity**: Different support expectations per channel

### Neutral
- **Phased Rollout**: Complexity increases gradually with each phase
- **Community Feedback**: Early channels provide feedback for later channels

## Implementation Notes

### Release Checklist

For each release, update all distribution channels:

```bash
# Phase 1 Release Checklist
□ Build and push operator image to Quay.io
□ Build and push bundle image to Quay.io
□ Update community-operators repository (OpenShift)
□ Create GitHub Release with manifests
□ Update documentation
□ Announce release (blog, social media)

# Phase 2 Release Checklist (adds)
□ Submit for Red Hat certification
□ Update certified operator bundle
□ Update Red Hat Ecosystem Catalog listing

# Phase 3 Release Checklist (adds)
□ Update OperatorHub.io (Kubernetes)
□ Package and publish Helm chart
□ Update Artifact Hub listing
□ Update GitHub Action (if applicable)
```

### Automation

```yaml
# .github/workflows/release.yml
name: Release Operator

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Build and push images
        run: |
          make docker-build docker-push IMG=${{ env.OPERATOR_IMG }}
          make bundle bundle-build bundle-push BUNDLE_IMG=${{ env.BUNDLE_IMG }}
      
      - name: Generate manifests
        run: kustomize build config/default > install.yaml
      
      - name: Create GitHub Release
        uses: actions/create-release@v1
        with:
          tag_name: ${{ github.ref }}
          release_name: Release ${{ github.ref }}
          body_path: CHANGELOG.md
          files: |
            install.yaml
            config/samples/*.yaml
      
      - name: Package Helm chart
        run: |
          helm package charts/jupyter-notebook-validator-operator
          helm repo index . --url https://your-org.github.io/jupyter-notebook-validator-operator
      
      - name: Update community-operators
        run: |
          # Automated PR to community-operators repository
          ./scripts/update-community-operators.sh
```

### Distribution Matrix

| Phase | OpenShift OperatorHub | Red Hat Catalog | OperatorHub.io | Artifact Hub | GitHub Releases |
|-------|----------------------|-----------------|----------------|--------------|-----------------|
| 1     | ✅ Community         | ❌              | ❌             | ❌           | ✅              |
| 2     | ✅ Community         | ✅ Certified    | ❌             | ❌           | ✅              |
| 3     | ✅ Community         | ✅ Certified    | ✅ Community   | ✅ Helm      | ✅              |

### Support Model by Channel

| Channel | Support Level | SLA | Support Contact |
|---------|--------------|-----|-----------------|
| Red Hat Catalog (Certified) | Enterprise | Per agreement | Red Hat Support |
| OpenShift OperatorHub (Community) | Community | Best effort | GitHub Issues |
| OperatorHub.io | Community | Best effort | GitHub Issues |
| Artifact Hub | Community | Best effort | GitHub Issues |
| GitHub Releases | Community | Best effort | GitHub Issues |

### Metrics and Monitoring

Track adoption across channels:

```yaml
metrics:
  downloads:
    - quay_io_pulls
    - github_release_downloads
    - helm_chart_downloads
  
  installations:
    - operatorhub_installs
    - certified_operator_installs
    - helm_installs
  
  engagement:
    - github_stars
    - github_issues
    - community_contributions
```

## References

- [OperatorHub.io Contribution Guide](https://operatorhub.io/contribute)
- [Red Hat Operator Certification](https://connect.redhat.com/en/partner-with-us/red-hat-openshift-operator-certification)
- [Artifact Hub Documentation](https://artifacthub.io/docs/)
- [OLM Bundle Format](https://olm.operatorframework.io/docs/tasks/creating-operator-bundle/)
- [Helm Chart Best Practices](https://helm.sh/docs/chart_best_practices/)

## Related ADRs

- ADR 004: Deployment and Packaging Strategy (defines packaging formats)
- ADR 006: Version Support Roadmap (defines phased rollout timeline)
- ADR 002: Platform Version Support Strategy (defines supported versions)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial distribution strategy |

