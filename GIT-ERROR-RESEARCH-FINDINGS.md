# Git Error Research Findings - November 2025

## 🔍 **Research Summary**

Based on external research of current best practices, common issues, and platform-specific solutions for git authentication errors in Kubernetes/OpenShift environments.

## 📊 **Key Findings**

### 1. Git Exit Codes

**Exit Code 2**: Generic git error, commonly authentication failure
- Missing credentials
- Invalid credentials
- Insufficient token permissions

**Exit Code 128**: Fatal git error, commonly repository access issues
- Repository not found
- SSH key revoked/missing
- Network connectivity problems
- Authentication required but not provided

### 2. GitHub Personal Access Tokens (2025)

**Two Types of PATs**:

1. **Classic Tokens** (Legacy, still supported)
   - Full repository access
   - Broad permissions
   - No expiration by default
   - **Deprecated** for new use cases

2. **Fine-Grained Tokens** (Recommended)
   - Repository-specific access
   - Granular permissions
   - Mandatory expiration (max 1 year)
   - Better security posture

**Required Permissions for Private Repos**:
- **Classic**: `repo` scope (full control)
- **Fine-Grained**: `Contents` (read) permission minimum

**Common Issues**:
- Token expiration (fine-grained tokens expire!)
- Insufficient scopes/permissions
- Two-factor authentication conflicts
- Organization SSO requirements

### 3. Kubernetes Secret Best Practices

**Format for Git Credentials**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
type: kubernetes.io/basic-auth
stringData:
  username: oauth2  # For GitHub PAT
  password: ghp_xxxxxxxxxxxx  # Your PAT
```

**Security Best Practices**:
- ✅ Use `kubernetes.io/basic-auth` type
- ✅ Store tokens as environment variables, not hardcoded
- ✅ Use fine-grained tokens with minimal permissions
- ✅ Set token expiration
- ✅ Rotate tokens regularly
- ❌ Never commit tokens to git
- ❌ Never use classic tokens for new projects

### 4. OpenShift Tekton Pipelines Authentication

**Official Documentation** (Red Hat OpenShift Pipelines 1.16):

**Secret Annotation Requirements**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  annotations:
    tekton.dev/git-0: https://github.com  # Domain annotation required!
type: kubernetes.io/basic-auth
stringData:
  username: oauth2
  password: ghp_xxxxxxxxxxxx
```

**Key Requirements**:
- Annotation key MUST begin with `tekton.dev/git-`
- Value is the URL of the host (e.g., `https://github.com`)
- Multiple annotations for multiple git hosts
- Secret must be in same namespace as Pipeline/PipelineRun

**Common Mistakes**:
- ❌ Missing `tekton.dev/git-` annotation
- ❌ Wrong annotation format
- ❌ Secret in different namespace
- ❌ Using SSH instead of HTTPS

### 5. OpenShift SCC and Git-Clone Init Containers

**Problem**: Git-clone init containers fail with permission denied

**Root Cause**: OpenShift's restricted SCC doesn't allow:
- Running as root
- Writing to certain directories
- Arbitrary user IDs

**Solution**: Use Tekton Pipelines
- Tekton uses `pipelines-scc` during build
- Git-clone happens during build phase (elevated permissions)
- Final image runs under restricted SCC
- No git-clone init container needed in validation pod

**From OpenShift 4.8/4.9 Release Notes**:
> "A non-root user is now added to the build-base image of pipelines so that git-init can clone repositories as a non-root user."

### 6. Common Error Patterns

**"Authentication Failed"**:
- Cause: Invalid or expired PAT
- Solution: Regenerate token with correct scopes

**"Repository not found"**:
- Cause: Private repo without credentials OR wrong URL
- Solution: Verify URL and add credentials secret

**"Permission denied (publickey)"**:
- Cause: Using SSH instead of HTTPS
- Solution: Use HTTPS with PAT

**"Could not read from remote repository"**:
- Cause: Missing credentials or insufficient permissions
- Solution: Check token scopes and secret configuration

## 🎯 **Recommendations for Error Messages**

### 1. Detect Token Type Issues
```
HINT: GitHub deprecated classic tokens. Use fine-grained tokens:
  https://github.com/settings/tokens?type=beta
```

### 2. Provide Tekton-Specific Guidance
```
For OpenShift Tekton Pipelines, ensure secret has annotation:
  annotations:
    tekton.dev/git-0: https://github.com
```

### 3. Link to Official Documentation
```
See: https://docs.redhat.com/en/documentation/red_hat_openshift_pipelines/1.16/html/securing_openshift_pipelines/authenticating-pipelines-repos-using-secrets
```

### 4. Explain SCC Context
```
OpenShift's restricted SCC prevents git-clone init containers from running.
Tekton builds use pipelines-scc during build, then produce restricted-SCC-compatible images.
```

## 📝 **Updated Error Message Template**

```
Git authentication failed (exit code 2). The git-clone init container cannot access the repository.

ROOT CAUSE: Git credentials are missing, invalid, or insufficient.

COMMON ISSUES (2025):
- GitHub fine-grained token expired (max 1 year)
- Token missing 'Contents' read permission
- Missing tekton.dev/git- annotation for Tekton
- Using classic token instead of fine-grained token

RECOMMENDED SOLUTION: Use Tekton build with properly configured git credentials.

Quick Fix:
  spec:
    notebook:
      git:
        url: "https://github.com/your-org/your-repo.git"
        ref: "main"
        credentialsSecret: "git-credentials"
    podConfig:
      buildConfig:
        enabled: true
        strategy: "tekton"

Create git-credentials secret (OpenShift Tekton):
  kubectl create secret generic git-credentials \
    --from-literal=username=oauth2 \
    --from-literal=password=ghp_xxxxxxxxxxxx \
    --dry-run=client -o yaml | \
  kubectl annotate -f - \
    tekton.dev/git-0=https://github.com \
    --local -o yaml | \
  kubectl apply -f -

Generate GitHub fine-grained token:
  1. Go to: https://github.com/settings/tokens?type=beta
  2. Select repositories
  3. Grant 'Contents' read permission
  4. Set expiration (max 1 year)

Why this works: Tekton clones during build with pipelines-scc, then validation pod uses built image (no git-clone init container).

See: https://docs.redhat.com/en/documentation/red_hat_openshift_pipelines/1.16/html/securing_openshift_pipelines/authenticating-pipelines-repos-using-secrets
```

## 🔗 **References**

1. **GitHub PAT Documentation**:
   - https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens

2. **OpenShift Pipelines Authentication**:
   - https://docs.redhat.com/en/documentation/red_hat_openshift_pipelines/1.16/html/securing_openshift_pipelines/authenticating-pipelines-repos-using-secrets

3. **Tekton Git Authentication**:
   - https://tekton.dev/docs/pipelines/auth/
   - https://tekton.dev/docs/how-to-guides/clone-repository/

4. **OpenShift SCC Documentation**:
   - https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/authentication_and_authorization/managing-security-context-constraints

## ✅ **Action Items**

1. ✅ **Implemented**: Basic git error detection (exit codes 2, 128)
2. ✅ **Implemented**: Comprehensive error messages with solutions
3. 🔄 **Next**: Add Tekton annotation guidance
4. 🔄 **Next**: Add fine-grained token recommendations
5. 🔄 **Next**: Link to official documentation
6. 📅 **Future**: Dynamic research-enhanced error analysis

