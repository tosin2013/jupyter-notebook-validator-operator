# Platform Compatibility Matrix

> **Requires:** Kubernetes or OpenShift cluster


**Last Updated**: 2026-09-17  
**Operator Version**: 1.0.10 (release-4.22 branch)

## Overview

This document tracks the compatibility of the Jupyter Notebook Validator Operator with different platform versions, APIs, and base images. It is updated as part of the dependency review process defined in [ADR-029](adrs/029-platform-version-dependency-review-process.md).

## Supported Platform Versions

| Operator Version | OpenShift      | Kubernetes     | Tekton Pipelines | OpenShift AI   | Status      |
|------------------|----------------|----------------|------------------|----------------|-------------|
| 0.1.0            | 4.18.21        | 1.31.10        | 1.17.0           | 2025.1         | ⚠️ Deprecated |
| 1.0.9            | 4.20           | 1.33           | 1.17.0           | 2025.1         | ✅ Tested   |
| 1.0.10           | 4.20-4.22      | 1.31-1.35      | 1.17+            | 2025.1+        | ✅ Current  |

**Legend**:
- ✅ Tested: Verified working on this exact version
- ✅ Expected: Should work based on API compatibility
- 🔄 Planned: Target for future release
- ⚠️ Deprecated: Still works but support ending soon
- ❌ Unsupported: Known not to work

## API Versions Used

### Core Kubernetes APIs

| API Group              | Version | First Used | Status     | Deprecated In | Removed In | Notes |
|------------------------|---------|------------|------------|---------------|------------|-------|
| core                   | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Pods, Services, Secrets |
| apps                   | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Deployments |
| batch                  | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Jobs |

### OpenShift APIs

| API Group              | Version | First Used | Status     | Deprecated In | Removed In | Notes |
|------------------------|---------|------------|------------|---------------|------------|-------|
| build.openshift.io     | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | BuildConfig, Build |
| image.openshift.io     | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | ImageStream |
| route.openshift.io     | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Routes (optional) |

### Tekton APIs

| API Group              | Version | First Used | Status     | Deprecated In | Removed In | Notes |
|------------------------|---------|------------|------------|---------------|------------|-------|
| tekton.dev (Pipeline)  | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Pipeline, PipelineRun |
| tekton.dev (Task)      | v1      | 0.1.0      | ✅ Stable  | N/A           | N/A        | Task, TaskRun |
| tekton.dev (Pipeline)  | v1beta1 | N/A        | ⚠️ Legacy  | Tekton 0.50   | Tekton 1.0 | Not used |

**Migration Notes**:
- Tekton v1beta1 → v1: Completed in operator 0.1.0
- No v1beta1 APIs are used in current implementation

## Base Image Versions

### Git Clone Images

| Platform   | Image                                                                                      | SHA256 Digest | Last Updated | CVE Status | Notes |
|------------|--------------------------------------------------------------------------------------------|---------------|--------------|------------|-------|
| OpenShift  | `registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8:latest`                  | `4fabae1312c1aaf8a57bd2de63bd040956faa0c728453f2a4b4002705fba0f0c` | 2025-11-07 | ✅ Clean | Preferred for OpenShift |
| Kubernetes | `gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.50.0`                | TBD           | TBD          | TBD        | Fallback for K8s |
| Universal  | `alpine/git:latest`                                                                        | TBD           | TBD          | TBD        | Alternative |

### Build Images

| Platform   | Image                                                                                      | SHA256 Digest | Last Updated | CVE Status | Notes |
|------------|--------------------------------------------------------------------------------------------|---------------|--------------|------------|-------|
| OpenShift  | `registry.redhat.io/rhel8/buildah:latest`                                                  | `6d2dcb651ba680cf4ec74331f8349dec43d071d420625a1703370acc8d984e9e` | 2025-11-07 | ✅ Clean | Preferred for OpenShift |
| Kubernetes | `quay.io/buildah/stable:latest`                                                            | TBD           | TBD          | TBD        | Fallback for K8s |

### OpenShift AI Images (S2I)

| Image Name                          | Tag    | Image Reference                                                                                    | Last Updated | Status     | Notes |
|-------------------------------------|--------|----------------------------------------------------------------------------------------------------|--------------|------------|-------|
| s2i-minimal-notebook                | 2025.1 | `image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1` | 2025-11-07 | ✅ Current | Minimal Python |
| s2i-generic-data-science-notebook   | 2025.1 | `image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-generic-data-science-notebook:2025.1` | 2025-11-07 | ✅ Current | Standard Data Science |

**Available Tags**: 1.2, 2023.1, 2023.2, 2024.1, 2024.2, 2025.1

## Go Module Dependencies

### Core Dependencies

| Module                                    | Version  | Last Updated | CVE Status | Notes |
|-------------------------------------------|----------|--------------|------------|-------|
| `github.com/openshift/api`                | v0.0.0   | 2025-11-09   | ✅ Clean   | OpenShift APIs |
| `github.com/tektoncd/pipeline`            | v0.63.0  | 2025-11-09   | ✅ Clean   | Tekton APIs |
| `sigs.k8s.io/controller-runtime`          | v0.19.0  | 2025-11-09   | ✅ Clean   | Kubernetes client |
| `k8s.io/api`                              | v0.31.1  | 2025-11-09   | ✅ Clean   | Kubernetes APIs |
| `k8s.io/apimachinery`                     | v0.31.1  | 2025-11-09   | ✅ Clean   | Kubernetes types |

### Testing Dependencies

| Module                                    | Version  | Last Updated | CVE Status | Notes |
|-------------------------------------------|----------|--------------|------------|-------|
| `github.com/onsi/ginkgo/v2`               | v2.20.2  | 2025-11-09   | ✅ Clean   | Test framework |
| `github.com/onsi/gomega`                  | v1.34.2  | 2025-11-09   | ✅ Clean   | Assertions |
| `github.com/stretchr/testify`             | v1.9.0   | 2025-11-09   | ✅ Clean   | Test utilities |

**Last Dependency Scan**: 2025-11-09  
**Scan Tool**: `go mod verify` + manual review  
**Next Scheduled Scan**: 2025-12-09 (monthly)

## Known Issues and Workarounds

### OpenShift 4.20-4.22

**Issue**: None currently known  
**Status**: ✅ Working

### Tekton Pipelines 1.17.0

**Issue**: ClusterTasks do not exist, Tasks are namespace-scoped  
**Workaround**: Copy Tasks to user namespace (implemented in ADR-028)  
**Status**: ✅ Resolved

### OpenShift AI 2025.1

**Issue**: None currently known  
**Status**: ✅ Working

## Testing Matrix

### Automated Testing

| Platform       | Version | Test Type        | Frequency | Status     | Last Run   |
|----------------|---------|------------------|-----------|------------|------------|
| OpenShift      | 4.20-4.22 | Unit Tests       | Per PR    | ✅ Passing | 2026-09-17 |
| OpenShift      | 4.20-4.22 | Integration      | Per PR    | ✅ Passing | 2026-09-17 |
| OpenShift      | 4.20-4.22 | E2E (S2I)        | Per PR    | ✅ Passing | 2026-09-17 |
| OpenShift      | 4.20-4.22 | E2E (Tekton)     | Manual    | 🔄 Pending | N/A        |
| Kubernetes     | 1.31-1.35 | Unit Tests       | Per PR    | ✅ Passing | 2026-09-17 |
| Kubernetes     | 1.31-1.35 | Integration      | Manual    | 🔄 Pending | N/A        |

### Manual Testing

| Scenario                                  | OpenShift 4.20-4.22 | Kubernetes 1.31-1.35 | Notes |
|-------------------------------------------|----------------|-----------------|-------|
| S2I Build with OpenShift AI               | ✅ Tested      | N/A             | Working perfectly |
| S2I Build with custom base image          | ✅ Tested      | N/A             | Working |
| Tekton Build with git-clone + buildah     | 🔄 In Progress | 🔄 Pending      | ADR-028 implementation |
| Validation pod with built image           | ✅ Tested      | 🔄 Pending      | 100% success rate |
| Validation pod with pre-built image       | ✅ Tested      | 🔄 Pending      | Working |

## Upgrade Path

### From 0.1.0 to 0.2.0 (Planned)

**Prerequisites**:
- OpenShift 4.20+ or Kubernetes 1.31+
- Tekton Pipelines 1.17+ (if using Tekton builds)

**Breaking Changes**: TBD

**Upgrade Steps**: TBD

**Rollback Procedure**: TBD

## Security Advisories

### Current Advisories

**None** - No active CVEs affecting the operator or its dependencies.

**Last Security Scan**: 2025-11-09  
**Next Scheduled Scan**: 2025-11-16 (weekly)

### Historical Advisories

**None** - No historical CVEs to report.

## Deprecation Notices

### Upcoming Deprecations

**None** - No deprecations planned for current APIs or dependencies.

### Past Deprecations

**None** - No past deprecations to report.

## Support Policy

### Version Support

- **Current Release**: Full support (bug fixes, security updates, new features)
- **Previous Release (N-1)**: Security updates and critical bug fixes only
- **Older Releases (N-2+)**: No support, upgrade recommended

### Platform Support

- **OpenShift**: Certified rolling window (currently 4.20, 4.21, 4.22 per ADR-056)
- **Kubernetes**: Corresponding versions (currently 1.31-1.35)
- **Tekton**: Current major version (e.g., 1.x)

## Contributing

### Reporting Compatibility Issues

If you encounter compatibility issues:

1. Check this document for known issues
2. Search existing GitHub issues
3. Create a new issue with:
   - Platform version (OpenShift/Kubernetes)
   - Operator version
   - Error messages and logs
   - Steps to reproduce

### Updating This Document

This document is updated:
- **Automatically**: When CI/CD detects dependency changes
- **Manually**: During release preparation
- **As Needed**: When compatibility issues are discovered

**Process**: See [ADR-029](adrs/029-platform-version-dependency-review-process.md)

## References

- [ADR-029: Platform Version Dependency Review Process](adrs/029-platform-version-dependency-review-process.md)
- [ADR-027: S2I Build Strategy](adrs/027-s2i-build-strategy-for-git-integration.md)
- [ADR-028: Tekton Task Strategy](adrs/028-tekton-task-strategy-custom-vs-cluster-tasks.md)
- [OpenShift Release Notes](https://docs.openshift.com/container-platform/4.22/release_notes/ocp-4-22-release-notes.html)
- [Kubernetes Release Notes](https://kubernetes.io/releases/)
- [Tekton Pipeline Releases](https://github.com/tektoncd/pipeline/releases)

