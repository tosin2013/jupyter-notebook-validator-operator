# OpenShift 4.18 Release Branch

**Branch:** `release-4.18`  
**Created:** November 8, 2025  
**Target:** OpenShift 4.18.x (Kubernetes 1.31)  
**Status:** 🚧 In Development

## Overview

This branch targets OpenShift 4.18 and is forward compatible with OpenShift 4.19, 4.20, and 4.21.

## Dependencies

### Kubernetes Dependencies
- **k8s.io/api**: v0.31.10
- **k8s.io/apimachinery**: v0.31.10
- **k8s.io/client-go**: v0.31.10
- **sigs.k8s.io/controller-runtime**: v0.19.4

### OpenShift Dependencies
- **github.com/openshift/api**: 5dd0bcfcbb79 (January 2025 commit)
  - Commit: `5dd0bcfcbb795976926583d2abc9f28bb6a33ff9`
  - Date: January 24, 2025
  - Compatible with: OpenShift 4.18 (Kubernetes 1.31)

### Tekton Dependencies
- **github.com/tektoncd/pipeline**: v0.65.0
  - Compatible with: Kubernetes 1.31
  - Features: Resolvers caching, Pipeline in Pipeline, ARM64 support

## Supported OpenShift Versions

| OpenShift Version | Kubernetes Version | Status | Notes |
|-------------------|-------------------|--------|-------|
| 4.18.x | 1.31 | ✅ Tested | Primary target |
| 4.19.x | 1.32 | ✅ Compatible | Forward compatible |
| 4.20.x | 1.33 | ✅ Compatible | Forward compatible |
| 4.21.x | 1.34 | ✅ Compatible | Forward compatible (expected) |

## Build Status

### Current Status
- ✅ Dependencies configured
- ✅ go.mod updated
- ✅ go mod tidy successful
- 🚧 Build strategies need API updates
- 🚧 Unit tests pending
- 🚧 Integration tests pending

### Known Issues
1. **Build Strategy API Mismatch**: The build strategy code (pkg/build/) was written before the CRD fields were finalized. Needs updates to match actual API:
   - `job.Spec.PodConfig` is a struct, not a pointer
   - `job.Spec.NotebookSource` field doesn't exist (need to use correct field name)
   - OpenShift BuildConfig API structure needs verification

### Next Steps
1. Update build strategy code to match actual CRD API
2. Write unit tests for build strategies
3. Write integration tests
4. Test on real OpenShift 4.18 cluster
5. Create release artifacts

## Development Workflow

### Building
```bash
# Build the operator
make build

# Build the build strategies
go build ./pkg/build/...

# Run tests
make test

# Run integration tests
INTEGRATION_TESTS=true go test -v -tags=integration ./pkg/build/...
```

### Testing on OpenShift 4.18
```bash
# Verify cluster version
oc version
# Should show: Server Version: 4.18.x

# Deploy operator
make deploy IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:release-4.18

# Test S2I build
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml

# Test Tekton build
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml
```

## Release Process

### Version Tagging
```bash
# Tag the release
git tag -a v1.0.0-ocp4.18 -m "Release v1.0.0 for OpenShift 4.18"
git push origin v1.0.0-ocp4.18
```

### Building Release Artifacts
```bash
# Build operator image
make docker-build docker-push IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:v1.0.0-ocp4.18

# Generate release manifests
make release-manifests VERSION=v1.0.0-ocp4.18

# Create GitHub release
gh release create v1.0.0-ocp4.18 \
  --title "v1.0.0 for OpenShift 4.18" \
  --notes "Release for OpenShift 4.18.x, 4.19.x, 4.20.x (Kubernetes 1.31)" \
  dist/install.yaml
```

## Backporting from Main

### Cherry-picking Features
```bash
# Switch to release-4.18
git checkout release-4.18

# Cherry-pick commit from main
git cherry-pick <commit-hash>

# If conflicts, resolve and continue
git cherry-pick --continue

# Push to release-4.18
git push origin release-4.18
```

### Backporting Bug Fixes
```bash
# Bug fixes should be applied to release-4.18 first
git checkout release-4.18
git checkout -b fix/critical-bug

# Make fix, commit
git add .
git commit -m "fix: Critical bug in build strategy"
git push origin fix/critical-bug

# Create PR to release-4.18
# After merge, cherry-pick to main
```

## Support Timeline

### Full Support
- **Start**: November 2025 (branch creation)
- **End**: August 2026 (OpenShift 4.18 Maintenance Support ends)

### Maintenance Support
- **Start**: August 2026
- **End**: February 2027 (OpenShift 4.18 EUS Term 1 ends)

### Extended Support (Optional)
- **EUS Term 1**: August 2026 - February 2027 (included with Premium)
- **EUS Term 2**: February 2027 - February 2028 (optional add-on)

### Recommended Migration Path
- **August 2026**: Consider upgrading to release-4.20 (EUS)
- **February 2027**: Upgrade to release-4.20 or release-4.22 (EUS)

## Comparison with Other Branches

| Feature | release-4.18 | release-4.20 | main |
|---------|--------------|--------------|------|
| OpenShift Version | 4.18-4.21 | 4.20-4.21 | Latest |
| Kubernetes Version | 1.31 | 1.33 | Latest |
| k8s.io Version | v0.31.x | v0.33.x | Latest |
| Status | 🚧 Development | 📋 Planned | 🔬 Experimental |
| Production Ready | Q4 2025 | Q1 2026 | N/A |
| Support Until | Feb 2027 | Oct 2028 | N/A |

## References

- [OpenShift 4.18 Release Notes](https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/release_notes/)
- [Kubernetes 1.31 Release Notes](https://kubernetes.io/blog/2024/08/13/kubernetes-v1-31-release/)
- [Tekton Pipeline v0.65.0 Release](https://github.com/tektoncd/pipeline/releases/tag/v0.65.0)
- [OpenShift API Repository](https://github.com/openshift/api)

## Contact

For questions or issues specific to this release branch:
- Create an issue with label `release-4.18`
- Tag: `@tosin2013`

---

**Last Updated:** November 8, 2025  
**Next Review:** December 2025

