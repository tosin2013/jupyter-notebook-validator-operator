# ADR-051: Git Init Container Image Compatibility

## Status
Implemented

**Date**: 2025-11-21
**Updated**: 2026-01-24
**Supersedes**: Previously numbered as ADR-042 (renumbered to resolve duplicate)

## Context
The operator uses a git-clone init container to fetch notebooks from Git repositories before validation. The choice of git init image affects compatibility with OpenShift's security context constraints and different execution modes.

### Current Implementation
The operator's `git_helper.go` uses bash scripts to clone repositories:
```bash
/bin/bash -c "git clone --depth 1 --branch ${ref} ${url} /workspace/repo"
```

### Image Compatibility Issue
Two different approaches exist for git cloning in containers:

1. **Standard bash + git** (alpine/git, bitnami/git):
   - Has `/bin/bash` and `git` command available
   - Works with our current bash script approach
   - Simpler, more portable

2. **Tekton git-init binary** (Red Hat pipelines-git-init-rhel8, Tekton git-init):
   - Uses a specialized `git-init` binary
   - Requires specific arguments: `-url`, `-revision`, `-path`, etc.
   - Does NOT have `/bin/bash` or standard `git` command
   - Example: `git-init -url=https://... -revision=main -path=/workspace/repo`

### Problem Discovered
When using `registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8`, the operator's bash-based approach fails with exit code 128 because:
- The image doesn't have `/bin/bash` in the expected location
- The image is designed to run the `git-init` binary directly, not bash scripts

### Research Findings
From examining the Tekton git-clone task on OpenShift 4.18:
```yaml
exec git-init \
    -url="${PARAMS_URL}" \
    -revision="${PARAMS_REVISION}" \
    -refspec="${PARAMS_REFSPEC}" \
    -path="${checkout_dir}" \
    -sslVerify="${PARAMS_SSL_VERIFY}" \
    -submodules="${PARAMS_SUBMODULES}" \
    -depth="${PARAMS_DEPTH}"
```

## Decision
Use a custom RHEL9-based git-init image (`quay.io/takinosh/git-init-rhel9:latest`) for the following reasons:

1. **Compatibility**: Works with our current bash-based implementation
2. **Red Hat Ecosystem**: Based on ubi9/ubi-minimal, aligned with OpenShift/RHEL
3. **Security**: Runs as non-root user (UID 1001, group 0), compatible with OpenShift SCC
4. **Controlled**: We maintain the image source and build process
5. **Automated Updates**: Dependabot keeps base image updated
6. **Portability**: Works on both OpenShift and vanilla Kubernetes

**Image Source**: https://github.com/tosin2013/git-init-rhel9
**Registry**: quay.io/takinosh/git-init-rhel9:latest

## Consequences

### Positive
- ✅ Fixes current tier1 test failures caused by incompatible git-init image
- ✅ No code changes required to git clone logic
- ✅ Works on OpenShift without requiring Red Hat registry authentication
- ✅ Simpler debugging - can exec into container and run git commands
- ✅ Consistent behavior across environments
- ✅ RHEL9-based, aligned with Red Hat ecosystem
- ✅ Automated security updates via Dependabot
- ✅ Full control over image source and build process

### Negative
- ⚠️ Requires maintaining separate git-init image repository
- ⚠️ Requires Quay.io for image hosting
- ⚠️ Not an official Red Hat supported image (but uses UBI9 base)

### Neutral
- 💡 Future enhancement: Support both modes (bash+git AND git-init binary)
- 💡 Could add auto-detection based on image capabilities
- 💡 GIT_INIT_IMAGE env var already supported for custom images

## Implementation

### Custom Git-Init Image
Created a separate repository with RHEL9-based Dockerfile:
- **Repository**: https://github.com/tosin2013/git-init-rhel9
- **Base Image**: registry.access.redhat.com/ubi9/ubi-minimal:latest
- **Installed Packages**: git, bash, openssh-clients, ca-certificates
- **User**: Non-root UID 1001, group 0 (OpenShift SCC compatible)
- **Working Directory**: /workspace with 1001:0 ownership and g+rwX permissions
- **GitHub Actions**: Automated build and push to Quay.io
- **Dependabot**: Automated base image security updates

### Update git_helper.go
```go
func getGitImage() string {
    // Priority 1: Check for manual override
    if gitImage := os.Getenv("GIT_INIT_IMAGE"); gitImage != "" {
        return gitImage
    }

    // Priority 2: Use custom RHEL9-based git-init image for all platforms
    // Built from: https://github.com/tosin2013/git-init-rhel9
    return "quay.io/takinosh/git-init-rhel9:latest"
}
```

### Configuration Override
Users can still use alternative images by setting:
```yaml
env:
  - name: GIT_INIT_IMAGE
    value: "alpine/git:latest"
```

Note: Tekton git-init binary images would require updating the git clone logic to use git-init binary arguments instead of bash scripts.

## Future Work

1. **Dual Mode Support**: Detect image type and use appropriate invocation method
2. **RHEL 9 Support**: Investigate pipelines-git-init-rhel9 compatibility
3. **Image Caching**: Consider bundling git in operator image to avoid pull
4. **Documentation**: Add troubleshooting guide for git clone failures

## References
- [Tekton git-init source](https://github.com/tektoncd/pipeline/tree/main/cmd/git-init)
- [Alpine Git image](https://hub.docker.com/r/alpine/git)
- [OpenShift Pipelines Tasks](https://github.com/tektoncd/catalog)
- Issue discovered during tier1 E2E test execution on OpenShift 4.18
