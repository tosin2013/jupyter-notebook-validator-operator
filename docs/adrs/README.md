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
| [020](020-model-aware-validation-strategy.md) | Model-Aware Validation Strategy | Proposed | 2025-11-08 |

### Observability Enhancement ADRs (NEW - 2025-11-08)

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [021](021-openshift-native-dashboard-strategy.md) | OpenShift-Native Dashboard Strategy | Proposed | 2025-11-08 |
| [022](022-community-observability-contributions.md) | Community Observability Contributions | Proposed | 2025-11-08 |

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
- **Deprecated**: ADR is no longer recommended but kept for historical reference
- **Superseded**: ADR has been replaced by a newer decision (link to new ADR)

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

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [OpenShift Documentation](https://docs.openshift.com/)

