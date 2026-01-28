# Architectural Decision Records (ADRs)

This directory contains Architectural Decision Records (ADRs) for the Jupyter Notebook Validator Operator project.

## What are ADRs?

Architectural Decision Records document important architectural decisions made during the project lifecycle. Each ADR captures:
- The context and problem being addressed
- The decision made and alternatives considered
- The consequences of the decision
- Implementation notes and references

## ADR Index

### Critical / High-Priority ADRs

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [001](001-operator-framework-and-sdk-version.md) | Operator Framework and SDK Version | Accepted | 2025-11-07 |
| [002](002-platform-version-support-strategy.md) | Platform Version Support Strategy | Accepted | 2025-11-07 |
| [003](003-crd-schema-design-and-versioning.md) | CRD Schema Design and Versioning | Accepted | 2025-11-07 |
| [004](004-deployment-and-packaging-strategy.md) | Deployment and Packaging Strategy | Accepted | 2025-11-07 |
| [005](005-rbac-and-service-account-model.md) | RBAC and Service Account Model | Accepted | 2025-11-07 |
| [006](006-version-support-roadmap-and-testing.md) | Version Support Roadmap and Testing Strategy | Accepted | 2025-11-07 |
| [007](007-distribution-and-catalog-strategy.md) | Distribution and Catalog Strategy | Accepted | 2025-11-07 |
| [008](008-notebook-testing-strategy-and-complexity-levels.md) | Notebook Testing Strategy and Complexity Levels | Accepted | 2025-11-07 |
| [009](009-secret-management-and-git-credentials.md) | Secret Management and Git Credentials | Accepted | 2025-11-07 |
| [010](010-observability-and-monitoring-strategy.md) | Observability and Monitoring Strategy | Accepted | 2025-11-07 |
| [011](011-error-handling-and-retry-strategy.md) | Error Handling and Retry Strategy | Accepted | 2025-11-07 |
| [012](012-release-and-cicd-strategy.md) | Release and CI/CD Strategy | Accepted | 2025-11-07 |
| [013](013-output-comparison-and-diffing-strategy.md) | Output Comparison and Diffing Strategy | Accepted | 2025-11-08 |

### Security & Credential Management ADRs (NEW - 2025-11-08)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [014](014-notebook-credential-injection-strategy.md) | Notebook Credential Injection Strategy | Accepted | 2025-11-08 |
| [015](015-environment-variable-pattern-for-notebook-credentials.md) | Environment-Variable Pattern for Notebook Credentials | Accepted | 2025-11-08 |
| [016](016-external-secrets-operator-integration.md) | External Secrets Operator (ESO) Integration | Accepted | 2025-11-08 |
| [017](017-vault-dynamic-secrets-injection-pattern.md) | Vault Dynamic-Secrets Injection Pattern | Accepted | 2025-11-08 |
| [018](018-secret-rotation-and-lifecycle-management.md) | Secret Rotation & Lifecycle Management | Accepted | 2025-11-08 |
| [019](019-rbac-and-pod-security-policies-for-notebook-secret-access.md) | RBAC & Pod Security Policies for Notebook Secret Access | Accepted | 2025-11-08 |

### Model-Aware Validation ADRs (NEW - 2025-11-08)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [020](020-model-aware-validation-strategy.md) | Model-Aware Validation Strategy | Implemented | 2025-11-08 |

### Observability Enhancement ADRs (NEW - 2025-11-08)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [021](021-openshift-native-dashboard-strategy.md) | OpenShift-Native Dashboard Strategy | Proposed | 2025-11-08 |
| [022](022-community-observability-contributions.md) | Community Observability Contributions | Proposed | 2025-11-08 |

### Build Integration ADRs (NEW - 2025-11-09)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [023](023-s2i-build-integration-openshift.md) | S2I Build Integration for OpenShift | Archived | 2025-11-08 |
| [024](024-fallback-strategy-missing-requirements.md) | Fallback Strategy for Missing Requirements | Accepted | 2025-11-08 |
| [025](025-community-build-methods-extension-framework.md) | Community Build Methods Extension Framework | Accepted | 2025-11-08 |
| [026](026-smart-validation-pod-recovery.md) | Smart Validation Pod Recovery and Build Strategy Selection | Implemented | 2025-11-09 |
| [027](027-s2i-build-strategy-for-git-integration.md) | S2I Build Strategy for Git Integration | Superseded | 2025-11-09 |
| [028](028-tekton-task-strategy-custom-vs-cluster-tasks.md) | Tekton Task Strategy - Custom vs Cluster Tasks | Implemented | 2025-11-09 |
| [031](031-tekton-build-dockerfile-vs-base-image.md) | Tekton Build Strategy - Dockerfile vs Base Image | Implemented | 2025-11-09 |
| [037](037-build-validation-sequencing-and-state-machine.md) | Build-Validation Sequencing and State Machine | Implemented | 2025-11-10 |
| [039](039-automatic-scc-management-for-tekton-builds.md) | Automatic SCC Management for Tekton Builds | Implemented | 2025-11-21 |
| [040](040-unique-build-pvcs-for-concurrent-tekton-builds.md) | Unique Build PVCs for Concurrent Tekton Builds | Implemented | 2025-11-24 |
| [042](042-automatic-tekton-git-credentials-conversion.md) | Automatic Tekton Git Credentials Conversion | Implemented | 2025-11-21 |
| [050](050-fix-s2i-build-status-monitoring.md) | Fix S2I Build Status Monitoring | Proposed | 2025-11-24 |
| [051](051-git-init-image-compatibility.md) | Git Init Container Image Compatibility | Implemented | 2025-11-21 |
| [052](052-resource-conflict-error-handling.md) | Resource Conflict Error Handling | Implemented | 2025-11-24 |
| [053](053-volume-and-pvc-support-for-validation-pods.md) | Volume and PVC Support for Validation Pods | Implemented | 2025-11-29 |
| [054](054-pod-scheduling-support-tolerations-nodeselector-affinity.md) | Pod Scheduling Support (Tolerations, NodeSelector, Affinity) | Implemented | 2026-01-28 |

### Testing & Validation ADRs (NEW - 2025-11-12)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [032](032-github-actions-ci-testing-kubernetes-version.md) | GitHub Actions CI Testing Against Kubernetes 1.31.10 | Implemented | 2025-11-09 |
| [033](033-e2e-testing-live-openshift-cluster.md) | End-to-End Testing Against Live OpenShift Cluster | Accepted | 2025-11-09 |
| [034](034-dual-testing-strategy-kind-openshift.md) | Dual Testing Strategy with Kind and OpenShift | Implemented | 2025-11-11 |
| [048](048-pre-submission-validation-test-rig-strategy.md) | Pre-Submission Validation Test Rig Strategy | Accepted | 2025-11-12 |

### ArgoCD Integration ADRs (NEW - 2026-01-24)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [049](049-argocd-integration-strategy.md) | ArgoCD Integration Strategy for GitOps Workflows | Implemented | 2026-01-24 |

### Planned ADRs

The following ADRs are planned for future documentation:

#### Medium Priority (Production Readiness)
- **ADR 021**: Configuration Management and Feature Flags
- **ADR 022**: Performance Optimization and Scalability

#### Low Priority (Enterprise Features)
- **ADR 022**: Upgrade and Migration Strategy
- **ADR 023**: Disaster Recovery and Backup

## ADR Status Definitions

- **Proposed**: ADR is under discussion and review
- **Accepted**: ADR has been approved and is being implemented
- **Implemented**: ADR has been fully implemented in the codebase
- **Deprecated**: ADR is no longer recommended but kept for historical reference
- **Superseded**: ADR has been replaced by a newer decision (link to new ADR)
- **Archived**: ADR is a duplicate or has been consolidated into another ADR

## ADR Template

When creating new ADRs, use the following structure:

```markdown
# ADR XXX: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXX]

## Context
[Describe the problem, constraints, and requirements]

## Decision
[Describe the decision and rationale]

## Consequences
[Describe positive, negative, and neutral consequences]

## Implementation Notes
[Technical details, code examples, configuration]

## References
[Links to relevant documentation, standards, or resources]

## Related ADRs
[Links to related ADRs]

## Revision History
| Date | Author | Description |
|------|--------|-------------|
| YYYY-MM-DD | Name | Initial decision |
```

## Key Architectural Decisions Summary

### Technology Stack
- **Language**: Go 1.21+
- **Framework**: Operator SDK v1.32.0+
- **Controller Runtime**: v0.16+
- **Dependency Management**: Go Modules

### Platform Support
- **Primary**: OpenShift 4.18, 4.19, 4.20
- **Secondary**: Kubernetes 1.25+
- **Support Model**: Tier 1 (OpenShift) and Tier 2 (Kubernetes)

### API Design
- **CRD Group**: mlops.dev
- **Initial Version**: v1alpha1
- **Versioning Strategy**: Multi-version with conversion webhooks
- **Schema**: OpenAPI v3 with structural schema

### Deployment
- **Primary**: OLM Bundle for OpenShift OperatorHub
- **Secondary**: Helm Chart for Kubernetes
- **Tertiary**: Raw Manifests for GitOps/CI/CD

### Security
- **RBAC Model**: Hybrid (cluster-wide and namespace-scoped)
- **Service Accounts**: Separate for operator and validation pods
- **Permissions**: Least privilege principle
- **Pod Security**: Non-root, dropped capabilities, read-only root filesystem

## Contributing to ADRs

When proposing a new ADR:

1. **Create a draft**: Copy the template and fill in the sections
2. **Assign a number**: Use the next available ADR number
3. **Set status to "Proposed"**: Mark the ADR as under review
4. **Open a PR**: Submit for team review and discussion
5. **Update status**: Change to "Accepted" after approval
6. **Update index**: Add the ADR to this README

## Questions?

For questions about ADRs or architectural decisions, please:
- Open an issue in the GitHub repository
- Discuss in team meetings
- Contact the platform team

## ADR Renumbering History

### 2026-01-24: Resolved Duplicate Numbering Conflicts

The following ADRs were renumbered to resolve duplicate numbering:

| Original | New | Description |
|----------|-----|-------------|
| 042-fix-s2i-build-status-monitoring | 050-fix-s2i-build-status-monitoring | Fix S2I Build Status Monitoring |
| 042-git-init-image-compatibility | 051-git-init-image-compatibility | Git Init Container Image Compatibility |
| 042-resource-conflict-error-handling | 052-resource-conflict-error-handling | Resource Conflict Error Handling |
| 045-volume-and-pvc-support | 053-volume-and-pvc-support | Volume and PVC Support for Validation Pods |

**Note**: ADR-042 (Automatic Tekton Git Credentials Conversion) remains the canonical ADR-042.
ADR-045 (Long-Term Strategic Deployment Plan) remains the canonical ADR-045.

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [OpenShift Documentation](https://docs.openshift.com/)

