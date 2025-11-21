# ADR-042: Automatic Tekton Git Credentials Secret Conversion

## Status

Accepted

## Context

The Jupyter Notebook Validator Operator supports two build strategies: S2I (Source-to-Image) and Tekton Pipelines. When users provide Git credentials for cloning private repositories, the operator needs to handle different secret formats:

1. **Validation Pod**: Uses standard Kubernetes secret format with `username` and `password` keys
2. **Tekton git-clone Task**: Expects basic-auth workspace format with `.git-credentials` and `.gitconfig` files

### The Problem

Prior to this ADR, users needed to manually create TWO separate secrets:
- `git-credentials`: Standard format for validation pods
- `git-credentials-tekton`: Tekton format for build pipelines

This created a poor user experience:
1. E2E tests were failing because only the standard secret was created
2. Users would need to understand Tekton's specific credential format
3. Documentation burden increased
4. Error messages were confusing (build would fail with "secret not found")

### Root Cause Analysis

From the Tier 2 E2E test failure:
```yaml
status:
  buildStatus:
    message: 'Build creation failed: pipelinerun creation verification failed:
              PipelineRun.tekton.dev "tier2-test-01-sentiment-model-build" not found'
    phase: Failed
```

The PipelineRun referenced a non-existent secret `git-credentials-tekton`, causing immediate build failure.

## Decision

The operator will automatically create and manage Tekton-formatted Git credentials secrets from standard secrets.

### Implementation

1. **Automatic Secret Conversion**: When a Tekton build is triggered with `spec.notebook.git.credentialsSecret` specified, the operator will:
   - Check if `{credentialsSecret}-tekton` exists
   - If not, create it from the source secret
   - Convert username/password to Tekton's `.git-credentials` and `.gitconfig` format

2. **Secret Naming Convention**:
   - Source: `{name}` (e.g., `git-credentials`)
   - Target: `{name}-tekton` (e.g., `git-credentials-tekton`)

3. **Format Conversion**:
   ```yaml
   # Source Secret (Standard Format)
   apiVersion: v1
   kind: Secret
   metadata:
     name: git-credentials
   data:
     username: base64(username)
     password: base64(password)

   # Generated Tekton Secret
   apiVersion: v1
   kind: Secret
   metadata:
     name: git-credentials-tekton
     labels:
       app.kubernetes.io/managed-by: jupyter-notebook-validator-operator
     annotations:
       tekton.dev/git-0: https://github.com
   data:
     .git-credentials: base64("https://username:password@github.com\n")
     .gitconfig: base64("[credential]\n\thelper = store\n")
   ```

4. **Lifecycle Management**:
   - Secrets are created on-demand during `CreateBuild()`
   - Labeled with `app.kubernetes.io/managed-by: jupyter-notebook-validator-operator`
   - Annotated with source secret reference for tracking

5. **RBAC Update**:
   - Added `create` permission for secrets in ClusterRole

### Code Location

- **Implementation**: `pkg/build/tekton_strategy.go:ensureTektonGitCredentials()`
- **Integration**: Called from `CreateBuild()` before pipeline creation
- **RBAC**: `config/rbac/role.yaml`

## Consequences

### Positive

1. **Improved User Experience**: Users only need to create one secret
2. **E2E Test Fix**: Tier 2 tests will pass with standard secrets
3. **Reduced Documentation**: No need to explain Tekton credential format
4. **Better Error Messages**: Clear errors if source secret is malformed
5. **Consistency**: Same secret naming works across S2I and Tekton strategies

### Negative

1. **RBAC Scope**: Operator now needs `create` permission for secrets (previously only `get`, `list`)
2. **Secret Proliferation**: Each credential secret creates a derived `-tekton` secret
3. **Sync Challenges**: Source secret updates don't automatically propagate (TODO: implement sync)

### Neutral

1. **Backward Compatibility**: Existing manual `-tekton` secrets are preserved and not overwritten
2. **Multi-Provider Support**: Current implementation hardcodes `github.com` in annotations (works for most providers due to generic credential format)

## Alternatives Considered

### Alternative 1: Require Users to Create Both Secrets
**Rejected**: Poor user experience, error-prone, requires deep Tekton knowledge.

### Alternative 2: Use Tekton Annotations on Standard Secret
**Rejected**: Tekton git-clone task specifically requires `.git-credentials` file format, not the standard keys.

### Alternative 3: Custom Tekton Task
**Rejected**: Would require maintaining custom Tekton tasks instead of using upstream git-clone task.

## Implementation Notes

### Future Enhancements (TODOs)

1. **Secret Synchronization**: Watch source secret for changes and update Tekton secret
2. **Multi-Provider Support**: Extract Git host from `spec.notebook.git.url` for annotation
3. **Cleanup on Deletion**: Consider deleting `-tekton` secret when source secret is deleted
4. **SSH Key Support**: Extend to handle SSH-based authentication

### Testing Strategy

1. Unit tests for `ensureTektonGitCredentials()`
2. E2E test verification with Tier 2 builds
3. Manual testing with private repositories

### Migration Path

Existing deployments with manual `-tekton` secrets:
- Will continue to work
- Operator detects existing secrets and skips creation
- No breaking changes

## References

- Tekton Auth Documentation: https://tekton.dev/docs/pipelines/auth/
- GitHub Issue: Tier 2 E2E Test Failures
- Related ADRs:
  - ADR-028: Tekton Task Management
  - ADR-031: Dockerfile Generation and Custom Base Images
  - ADR-039: Automatic SCC Management for Tekton Builds

## Date

2025-11-21
