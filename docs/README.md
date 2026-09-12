# Documentation

Navigation index for the Jupyter Notebook Validator Operator documentation.

## Getting Started

| Document | Description |
|---|---|
| [Development](getting-started/DEVELOPMENT.md) | Local development setup, building, and running |
| [Namespace Setup](getting-started/NAMESPACE_SETUP.md) | Kubernetes namespace and RBAC configuration |
| [Quick Start CI/CD](getting-started/QUICK_START_CI_CD.md) | CI/CD pipeline setup for notebook validation |

## Guides

| Document | Description |
|---|---|
| [Notebook Credentials](guides/NOTEBOOK_CREDENTIALS_GUIDE.md) | Credential injection via Secrets, ESO, and Vault |
| [Credentials Field Guide](guides/CREDENTIALS-FIELD-GUIDE.md) | Quick reference for credential fields |
| [Model Discovery](guides/MODEL_DISCOVERY_GUIDE.md) | Auto-detect and validate against ML model endpoints |
| [Model Training Workflow](guides/MODEL_TRAINING_WORKFLOW.md) | End-to-end model training with notebook validation |
| [Multi-User Model Validation](guides/MODEL_VALIDATION_MULTI_USER.md) | Shared model validation across teams |
| [End-to-End ML Workflow](guides/END_TO_END_ML_WORKFLOW.md) | Complete MLOps pipeline integration |
| [Golden Notebook Comparison](guides/GOLDEN_NOTEBOOK_COMPARISON.md) | Regression testing with golden notebooks |
| [Error Handling](guides/ERROR_HANDLING_GUIDE.md) | Error handling patterns and retry strategies |
| [ArgoCD Integration](guides/ARGOCD_INTEGRATION.md) | GitOps workflow with ArgoCD |
| [OpenShift AI Integration](guides/OPENSHIFT-AI-INTEGRATION.md) | OpenShift AI workbench integration |

## Architecture

| Document | Description |
|---|---|
| [Architecture Overview](architecture/ARCHITECTURE_OVERVIEW.md) | System design and component overview |
| [Platform Compatibility](architecture/PLATFORM-COMPATIBILITY.md) | Supported Kubernetes and OpenShift versions |
| [OpenShift Support Matrix](architecture/OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md) | OCP version support strategy and matrix |
| [OpenShift Deployment](architecture/OPENSHIFT_DEPLOYMENT_SOLUTION.md) | Deployment patterns for OpenShift |
| [Build Strategy](architecture/BUILD_STRATEGY_IMPLEMENTATION.md) | S2I and Tekton build strategies |

## Testing

| Document | Description |
|---|---|
| [Testing Guide](testing/TESTING_GUIDE.md) | Test strategy, tiers, and execution |
| [E2E Testing](testing/E2E_TESTING.md) | End-to-end test setup and execution |
| [Integration Testing](testing/INTEGRATION_TESTING.md) | Integration test patterns |
| [Test Notebooks Guide](testing/TEST_NOTEBOOKS_GUIDE.md) | Writing and organizing test notebooks |

## Operations

| Document | Description |
|---|---|
| [Observability](operations/OBSERVABILITY.md) | Prometheus metrics and dashboards |
| [CI Cluster Setup](operations/CI_CLUSTER_SETUP.md) | OpenShift cluster registration for CI |
| [GitHub Secrets Setup](operations/GITHUB_SECRETS_SETUP.md) | Repository secrets for CI workflows |
| [Webhook Configuration](operations/WEBHOOK_CONFIGURATION.md) | Admission webhook setup |
| [Webhook Installation](operations/WEBHOOK-INSTALLATION-GUIDE.md) | Webhook installation guide |
| [Tekton Build Setup](operations/TEKTON_BUILD_SETUP.md) | Tekton pipeline configuration |
| [Release Process](operations/RELEASE.md) | Release workflow and OLM submission |
| [Branching Strategy](operations/BRANCHING_STRATEGY.md) | Git branching model |
| [Community Observability](operations/COMMUNITY_OBSERVABILITY.md) | Contributing dashboards and alerts |

## Community

| Document | Description |
|---|---|
| [Supported Platforms](community/COMMUNITY_PLATFORMS.md) | Model serving platforms and integrations |
| [Contributing Model Platforms](community/CONTRIBUTING_MODEL_PLATFORMS.md) | Adding new platform detectors |

## Other

| Document | Description |
|---|---|
| [ADRs](adrs/) | Architectural decision records |
| [Releases](releases/) | Release notes by version |
| [Dashboards](dashboards/) | Grafana dashboard contributions |
| [Archive](_archive/) | Internal notes, session artifacts, and investigation logs |
