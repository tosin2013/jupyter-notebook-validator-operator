# ADR 001: Operator Framework and SDK Version

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator is a Kubernetes-native tool designed to automate the validation of Jupyter Notebooks within an MLOps workflow. As a Kubernetes operator, we need to select an appropriate framework and SDK that will:

1. Provide robust integration with the Kubernetes control plane
2. Support our target platforms (OpenShift 4.18, 4.19, 4.20, and latest Kubernetes)
3. Enable efficient development and maintenance
4. Ensure long-term API stability and community support
5. Align with Go language ecosystem best practices

### Current Environment
- **Target Platform**: OpenShift 4.18 (initial), with future support for 4.19, 4.20, and latest Kubernetes
- **Language**: Go (as specified in PRD)
- **Development Team**: Platform engineers familiar with Kubernetes and Go
- **Deployment Model**: Kubernetes operator pattern with CRD-based API

### Technical Requirements
- Must orchestrate notebook validation by creating and managing pods
- Must watch and reconcile custom resources (NotebookValidationJob CRD)
- Must integrate with Git repositories for notebook fetching
- Must parse and compare notebook outputs
- Must update CR status with detailed validation results

### Available Options

#### Option 1: Operator SDK v1.x (Latest Stable)
- **Pros**:
  - Long-term API stability and community support
  - Consistent project layout via `operator-sdk init`
  - Built-in controller-runtime integration (v0.12+)
  - Comprehensive tooling for scaffolding, testing, and deployment
  - Active maintenance and regular updates
  - Strong OpenShift integration and OLM support
  - Extensive documentation and examples
- **Cons**:
  - Must upgrade SDK to leverage new controller-runtime features
  - Some opinionated project structure
  - Learning curve for team members new to operator pattern

#### Option 2: Controller-Runtime Directly
- **Pros**:
  - Maximum flexibility in project structure
  - Direct access to latest controller-runtime features
  - Lighter dependency footprint
  - More control over reconciliation logic
- **Cons**:
  - More manual wiring and boilerplate code
  - No scaffolding tools for CRDs, webhooks, etc.
  - Steeper learning curve
  - More maintenance burden
  - Less standardized project structure

#### Option 3: Operator SDK v0.x (Legacy)
- **Pros**:
  - Mature and well-tested
  - Some existing examples and tutorials
- **Cons**:
  - Deprecated and approaching end-of-support
  - Uses older patterns and APIs
  - Limited future compatibility
  - Security vulnerabilities may not be patched
  - Not recommended for new projects

## Decision

We will adopt **Operator SDK v1.x (latest stable)** with Go Modules and the SDK's built-in controller-runtime v0.12+.

### Specific Choices
- **Operator SDK Version**: v1.32.0 or later (latest stable at project start)
- **Go Version**: 1.21+ (as required by Operator SDK v1.32+)
- **Controller-Runtime**: v0.16+ (bundled with Operator SDK)
- **Dependency Management**: Go Modules
- **Project Layout**: Standard Operator SDK layout with Kubebuilder markers

### Rationale

1. **API Stability**: Operator SDK v1.x provides stable APIs that align with our multi-version support strategy (OpenShift 4.18-4.20, K8s 1.25+)

2. **Productivity**: The SDK's scaffolding tools (`operator-sdk init`, `create api`, `create webhook`) will accelerate development and ensure consistency

3. **OpenShift Integration**: Native support for OLM (Operator Lifecycle Manager) bundles, which is critical for OpenShift OperatorHub deployment

4. **Community Support**: Large, active community with extensive documentation, examples, and troubleshooting resources

5. **Testing Tools**: Built-in support for envtest and e2e testing frameworks that align with our testing strategy

6. **Future-Proofing**: Regular updates and compatibility with latest Kubernetes APIs ensure longevity

## Consequences

### Positive
- **Standardized Structure**: Consistent project layout makes onboarding new developers easier
- **Reduced Boilerplate**: SDK handles much of the Kubernetes client setup and reconciliation scaffolding
- **OLM Integration**: Seamless path to OpenShift OperatorHub deployment
- **Testing Support**: Built-in testing utilities for unit, integration, and e2e tests
- **Documentation**: Extensive official documentation and community examples
- **Upgrade Path**: Clear upgrade path for SDK and controller-runtime versions

### Negative
- **SDK Dependency**: Must track and upgrade Operator SDK versions to access new features
- **Opinionated Structure**: Some flexibility is traded for standardization
- **Learning Curve**: Team members must learn Operator SDK conventions and patterns
- **Version Coordination**: Must coordinate SDK version with target Kubernetes/OpenShift versions

### Neutral
- **Go Modules**: Standard Go dependency management (already industry standard)
- **Kubebuilder Markers**: Must use code annotations for CRD generation (common practice)

## Implementation Notes

### Initial Setup
```bash
# Initialize operator project
operator-sdk init \
  --domain mlops.dev \
  --repo github.com/your-org/jupyter-notebook-validator-operator \
  --plugins go/v4

# Create API and controller
operator-sdk create api \
  --group mlops \
  --version v1alpha1 \
  --kind NotebookValidationJob \
  --resource --controller
```

### Project Structure
```
jupyter-notebook-validator-operator/
├── api/
│   └── v1alpha1/
│       ├── notebookvalidationjob_types.go
│       └── zz_generated.deepcopy.go
├── config/
│   ├── crd/
│   ├── manager/
│   ├── rbac/
│   └── samples/
├── controllers/
│   └── notebookvalidationjob_controller.go
├── hack/
├── main.go
├── Dockerfile
├── Makefile
└── go.mod
```

### Version Compatibility Matrix
| Operator SDK | Controller-Runtime | Kubernetes | OpenShift |
|--------------|-------------------|------------|-----------|
| v1.32.0+     | v0.16+            | 1.25-1.28  | 4.18-4.20 |

### Upgrade Strategy
- Monitor Operator SDK releases quarterly
- Test new SDK versions in development environment before upgrading
- Document breaking changes in ADRs
- Maintain compatibility with supported OpenShift/Kubernetes versions

## References

- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Controller-Runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [OpenShift Operator Lifecycle Manager](https://docs.openshift.com/container-platform/4.18/operators/understanding/olm/olm-understanding-olm.html)
- [Kubebuilder Book](https://book.kubebuilder.io/)

## Related ADRs

- ADR 002: Platform Version Support Strategy
- ADR 003: CRD Schema Design & Versioning
- ADR 004: Deployment & Packaging Strategy

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial decision |

