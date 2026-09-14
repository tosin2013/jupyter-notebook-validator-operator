# Documentation

Navigation index for the Jupyter Notebook Validator Operator, organized by [Diataxis](https://diataxis.fr/) quadrant.

---

## Tutorials

Learn by doing. Step-by-step lessons that take you from zero to a working result.

| Document | Audience | Description |
|----------|----------|-------------|
| [Local Validation](tutorials/LOCAL_VALIDATION.md) | Data scientist | Validate a notebook on your laptop with Podman/Docker. No cluster needed. |
| [Quick Start CI/CD](tutorials/QUICK_START_CI_CD.md) | Data scientist | Submit validation jobs to a Kubernetes cluster |
| [End-to-End ML Workflow](tutorials/END_TO_END_ML_WORKFLOW.md) | Data scientist | Train, deploy, and validate a model end-to-end |
| [Model Training Workflow](tutorials/MODEL_TRAINING_WORKFLOW.md) | Data scientist | Validate notebooks that train ML models |

## How-to guides

Task recipes. Each guide solves a specific problem.

| Document | Audience | Description |
|----------|----------|-------------|
| [Notebook Credentials](how-to/NOTEBOOK_CREDENTIALS_GUIDE.md) | Data scientist | Inject credentials via Secrets, ESO, and Vault |
| [Credentials Field Guide](how-to/CREDENTIALS-FIELD-GUIDE.md) | Data scientist | Quick reference for credential fields |
| [Model Discovery](how-to/MODEL_DISCOVERY_GUIDE.md) | Data scientist | Auto-detect and validate against ML model endpoints |
| [Golden Notebook Comparison](how-to/GOLDEN_NOTEBOOK_COMPARISON.md) | Data scientist | Regression testing with golden notebooks |
| [Error Handling](how-to/ERROR_HANDLING_GUIDE.md) | Data scientist | Error handling patterns and retry strategies |
| [Model Validation Multi-User](how-to/MODEL_VALIDATION_MULTI_USER.md) | Platform engineer | Shared model validation across teams |
| [Development](how-to/DEVELOPMENT.md) | Contributor | Local development setup, building, and running |
| [Namespace Setup](how-to/NAMESPACE_SETUP.md) | Platform engineer | Kubernetes namespace and RBAC configuration |
| [ArgoCD Integration](how-to/ARGOCD_INTEGRATION.md) | Platform engineer | GitOps workflow with ArgoCD |
| [OpenShift AI Integration](how-to/OPENSHIFT-AI-INTEGRATION.md) | Platform engineer | OpenShift AI workbench integration |
| [Tekton Build Setup](how-to/TEKTON_BUILD_SETUP.md) | Platform engineer | Tekton pipeline configuration |
| [Webhook Installation](how-to/WEBHOOK-INSTALLATION-GUIDE.md) | Platform engineer | Admission webhook setup |
| [Webhook Configuration](how-to/WEBHOOK_CONFIGURATION.md) | Platform engineer | Webhook configuration reference |
| [CI Cluster Setup](how-to/CI_CLUSTER_SETUP.md) | Platform engineer | OpenShift cluster registration for CI |
| [GitHub Secrets Setup](how-to/GITHUB_SECRETS_SETUP.md) | Platform engineer | Repository secrets for CI workflows |
| [Release Process](how-to/RELEASE.md) | Contributor | Release workflow and OLM submission |
| [Branching Strategy](how-to/BRANCHING_STRATEGY.md) | Contributor | Git branching model |
| [Community Observability](how-to/COMMUNITY_OBSERVABILITY.md) | Contributor | Contributing dashboards and alerts |
| [Dashboard Contributing](how-to/DASHBOARD_CONTRIBUTING.md) | Contributor | How to contribute Grafana/Console dashboards |
| [Contributing Model Platforms](how-to/CONTRIBUTING_MODEL_PLATFORMS.md) | Contributor | Adding new platform detectors |

## Reference

Technical descriptions of APIs, metrics, and test infrastructure.

| Document | Audience | Description |
|----------|----------|-------------|
| [Testing Guide](reference/TESTING_GUIDE.md) | Contributor | Test strategy, tiers, and execution |
| [E2E Testing](reference/E2E_TESTING.md) | Contributor | End-to-end test setup and execution |
| [Integration Testing](reference/INTEGRATION_TESTING.md) | Contributor | Integration test patterns |
| [Test Notebooks Guide](reference/TEST_NOTEBOOKS_GUIDE.md) | Data scientist | Writing and organizing test notebooks |
| [Observability](reference/OBSERVABILITY.md) | Platform engineer | Prometheus metrics and dashboards |
| [Platform Compatibility](reference/PLATFORM-COMPATIBILITY.md) | Platform engineer | Supported Kubernetes and OpenShift versions |
| [OpenShift Support Matrix](reference/OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md) | Platform engineer | OCP version support strategy and matrix |
| [Community Platforms](reference/COMMUNITY_PLATFORMS.md) | Data scientist | Model serving platforms and integrations |
| [Notebook Success Rate Dashboard](reference/notebook-success-rate.md) | Platform engineer | Dashboard specification |

## Explanation

Background, architecture, and design rationale.

| Document | Audience | Description |
|----------|----------|-------------|
| [Architecture Overview](explanation/ARCHITECTURE_OVERVIEW.md) | All | System design and component overview |
| [Build Strategy Implementation](explanation/BUILD_STRATEGY_IMPLEMENTATION.md) | Contributor | S2I and Tekton build strategies |
| [OpenShift Deployment](explanation/OPENSHIFT_DEPLOYMENT_SOLUTION.md) | Platform engineer | Deployment patterns for OpenShift |
| [Design Document](../DESIGN_DOC.md) | All | arc42 SDD: goals, building blocks, runtime, deployment |

## Other

| Document | Description |
|----------|-------------|
| [ADRs](adrs/) | Architectural decision records |
| [Releases](releases/) | Release notes by version |
| [Archive](_archive/) | Internal notes, session artifacts, and investigation logs |
