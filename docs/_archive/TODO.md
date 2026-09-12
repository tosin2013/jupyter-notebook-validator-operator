# Project TODO List

**Generated**: 2026-01-24
**Source**: ADR Compliance Review

## Unimplemented ADRs (GitHub Issues Created)

The following ADRs have "Proposed" status and need implementation:

| ADR | Title | GitHub Issue |
|-----|-------|--------------|
| ADR-021 | OpenShift-Native Dashboard Strategy | [#7](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/7) |
| ADR-022 | Community Observability Contributions | [#8](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/8) |
| ADR-030 | Smart Error Messages and User Feedback | [#9](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/9) |
| ADR-038 | Requirements.txt Auto-Detection | [#10](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/10) |
| ADR-041 | Exit Code Validation Framework | [#11](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/11) |

## ADR Compliance Tasks

### ADR-014: AWS Integration Gap

The ADR mentions AWS Secrets Manager integration patterns that are not yet fully documented or implemented.

- [ ] Add AWS Secrets Manager credential injection examples to documentation
- [ ] Document AWS IAM Roles for Service Accounts (IRSA) pattern for notebook credentials
- [ ] Add AWS-specific integration tests (optional, cloud-specific feature)
- [ ] Create sample NotebookValidationJob using AWS Secrets Manager

### ADR-040: AWS StorageClass Examples

The ADR references AWS EBS for PVC provisioning but examples are limited.

- [ ] Add AWS EBS StorageClass example for PVC provisioning to documentation
- [ ] Document AWS EFS StorageClass for ReadWriteMany access mode scenarios
- [ ] Create sample manifests for AWS-based storage configurations

## ADR Renumbering Completed (2026-01-24)

The following ADRs were renumbered to resolve duplicate numbering conflicts:

| Original | New | Description |
|----------|-----|-------------|
| 042-fix-s2i-build-status-monitoring | 050-fix-s2i-build-status-monitoring | Fix S2I Build Status Monitoring |
| 042-git-init-image-compatibility | 051-git-init-image-compatibility | Git Init Container Image Compatibility |
| 042-resource-conflict-error-handling | 052-resource-conflict-error-handling | Resource Conflict Error Handling |
| 045-volume-and-pvc-support | 053-volume-and-pvc-support | Volume and PVC Support for Validation Pods |

## Status Updates Completed (2026-01-24)

The following ADRs had their status updated during the compliance review:

| ADR | Previous Status | New Status | Reason |
|-----|-----------------|------------|--------|
| 028-tekton-task-strategy | Accepted | Implemented | Tekton namespace-scoped Tasks fully working |
| 032-github-actions-ci-testing | Proposed | Implemented | GitHub Actions CI pipeline verified |
| 034-dual-testing-strategy | Accepted | Implemented | Kind + OpenShift dual testing operational |
| 051-git-init-image-compatibility | Accepted | Implemented | Configurable git images in production |
| 052-resource-conflict-error-handling | Accepted | Implemented | Conflict handling in reconciliation loop |
| 053-volume-and-pvc-support | unknown | Implemented | PVC support in CRD verified |
| 050-fix-s2i-build-status-monitoring | Proposed | Implemented | Controller uses GetLatestBuild for S2I builds |

## References

- [ADR Directory](./adrs/)
- [ADR Compliance Review Report](./ADR_COMPLIANCE_REVIEW_2026-01-24.md) (generated)
