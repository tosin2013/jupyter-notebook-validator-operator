# ADR-031: Tekton Build Strategy - Dockerfile vs Base Image Support

**Status**: ✅ **IMPLEMENTED** (Phase 1 + Phase 2 + Git Credentials Fix)
**Date**: 2025-11-09
**Authors**: Sophia (AI Assistant), User Feedback
**Related**: ADR-028 (Tekton Task Strategy), ADR-027 (S2I Build Strategy), ADR-009 (Secret Management)
**Implementation**: Commits `3c95bc7` (Phase 1), `7d4fbd8` (Phase 2), `2f0ce75` (Git Credentials)

## Context

During Tekton build implementation, we discovered that the buildah Task expects a `DOCKERFILE` parameter pointing to a Dockerfile in the repository. However, our current implementation assumes users provide a `baseImage` and we build directly from it.

**User Insight**: "What if someone does not have a base image yet and has a docker file we may need to support both"

This raises a critical question: Should we support **both** Dockerfile-based builds AND base-image-only builds?

### Current Implementation

```yaml
spec:
  podConfig:
    buildConfig:
      enabled: true
      strategy: "tekton"
      baseImage: "quay.io/jupyter/minimal-notebook:latest"  # Base image approach
      autoGenerateRequirements: false
      requirementsFile: "requirements.txt"
```

**Problem**: Buildah Task expects `DOCKERFILE` parameter, but we're passing `BUILDER_IMAGE` which doesn't exist.

### Two Build Scenarios

#### Scenario 1: Base Image Only (Quick Start)
**User has**: Just a notebook and requirements.txt  
**User wants**: Quick validation with standard Jupyter image  
**Build approach**: Generate Dockerfile dynamically from baseImage  

**Example**:
```dockerfile
# Auto-generated Dockerfile
FROM quay.io/jupyter/minimal-notebook:latest
COPY requirements.txt /tmp/
RUN pip install -r /tmp/requirements.txt
COPY notebooks/ /notebooks/
```

**Benefits**:
- ✅ Simple for users (no Dockerfile needed)
- ✅ Fast iteration
- ✅ Good for CI/CD and quick validation
- ✅ Matches S2I behavior (baseImage + source)

#### Scenario 2: Custom Dockerfile (Production)
**User has**: Custom Dockerfile in repo  
**User wants**: Full control over build process  
**Build approach**: Use existing Dockerfile  

**Example**:
```dockerfile
# User's custom Dockerfile
FROM registry.access.redhat.com/ubi9/python-39:latest
RUN yum install -y gcc python3-devel
COPY requirements.txt /tmp/
RUN pip install --no-cache-dir -r /tmp/requirements.txt
COPY notebooks/ /notebooks/
RUN useradd -m jupyter
USER jupyter
```

**Benefits**:
- ✅ Full control over build environment
- ✅ Custom base images (UBI, Alpine, etc.)
- ✅ Complex dependency chains
- ✅ Production-grade builds
- ✅ Security hardening (non-root users, minimal layers)

## Decision

**Support BOTH approaches using buildah Task with Dockerfile generation**:

1. **If Dockerfile exists in repo**: Use it directly with buildah (Scenario 2)
2. **If only baseImage specified**: Generate Dockerfile, then use buildah (Scenario 1)
3. **If both exist**: Dockerfile takes precedence (explicit over implicit)

**Key Insight** (User Feedback): We don't need a custom Tekton Task! We can use the standard buildah Task for BOTH scenarios by generating a Dockerfile when only baseImage is provided. This is simpler and more maintainable than creating a custom Task.

### API Design

```yaml
spec:
  podConfig:
    buildConfig:
      enabled: true
      strategy: "tekton"
      
      # Option 1: Base Image (auto-generate Dockerfile)
      baseImage: "quay.io/jupyter/minimal-notebook:latest"
      
      # Option 2: Custom Dockerfile (use existing)
      dockerfile: "Dockerfile"  # Path relative to repo root
      
      # Option 3: Both (Dockerfile takes precedence)
      baseImage: "quay.io/jupyter/minimal-notebook:latest"  # Fallback
      dockerfile: "Dockerfile"  # Preferred
      
      # Common settings
      autoGenerateRequirements: false
      requirementsFile: "requirements.txt"
      context: "."  # Build context directory
```

### Implementation Strategy

#### Phase 1: Inline Dockerfile Generation (IMMEDIATE - SIMPLER!)

**User Insight**: "Do we need to create a custom tekton task? and use buildah when someone has a docker file"

**Answer**: NO! We can use a simple inline Task with a script step to generate the Dockerfile, then use buildah for both scenarios.

```go
// In tekton_strategy.go - Add inline Task for Dockerfile generation
func (t *TektonStrategy) createDockerfileGeneratorTask() tektonv1.PipelineTask {
    return tektonv1.PipelineTask{
        Name: "generate-dockerfile",
        TaskSpec: &tektonv1.EmbeddedTask{
            Params: []tektonv1.ParamSpec{
                {Name: "BASE_IMAGE", Type: tektonv1.ParamTypeString},
            },
            Workspaces: []tektonv1.WorkspaceDeclaration{
                {Name: "source"},
            },
            Steps: []tektonv1.Step{
                {
                    Name:  "check-dockerfile",
                    Image: "registry.access.redhat.com/ubi9/ubi-minimal:latest",
                    Script: `#!/bin/sh
set -e

# Check if Dockerfile already exists
if [ -f "$(workspaces.source.path)/Dockerfile" ] || [ -f "$(workspaces.source.path)/Containerfile" ]; then
    echo "✅ Dockerfile found in repository, using existing file"
    exit 0
fi

# Generate Dockerfile from baseImage
echo "📝 Generating Dockerfile from baseImage: $(params.BASE_IMAGE)"
cat > $(workspaces.source.path)/Dockerfile <<EOF
# Auto-generated by Jupyter Notebook Validator Operator
# ADR-031: Support both baseImage and custom Dockerfile
FROM $(params.BASE_IMAGE)

# Install dependencies if requirements file exists
COPY requirements.txt /tmp/ 2>/dev/null || true
RUN if [ -f /tmp/requirements.txt ]; then pip install --no-cache-dir -r /tmp/requirements.txt; fi

# Copy source code
COPY . /workspace/

# Set working directory
WORKDIR /workspace
EOF

echo "✅ Dockerfile generated successfully"
cat $(workspaces.source.path)/Dockerfile
`,
                },
            },
        },
    }
}
```

**Benefits of Inline Task**:
- ✅ No need to create/copy custom Task to namespace
- ✅ Simpler maintenance (code is in operator)
- ✅ Automatic Dockerfile detection (checks if exists first)
- ✅ Uses standard buildah Task for actual build
- ✅ Works for both scenarios seamlessly

#### Phase 2: Dockerfile Detection (SHORT-TERM)

```go
// Auto-detect Dockerfile in repository
func (t *TektonStrategy) detectDockerfile(ctx context.Context, repoURL string) (bool, string, error) {
    // Check common Dockerfile locations
    commonPaths := []string{
        "Dockerfile",
        "Containerfile",
        "docker/Dockerfile",
        ".devcontainer/Dockerfile",
    }
    
    // Use git-clone Task to fetch repo, then check for Dockerfile
    // If found, use it; otherwise generate from baseImage
    
    return false, "", nil
}
```

#### Phase 3: Advanced Features (FUTURE)

- Multi-stage Dockerfile generation
- Dockerfile templates for common scenarios
- Dockerfile validation and linting
- Build cache optimization

### Pipeline Changes

```go
// Update createBuildPipeline to handle both scenarios
func (t *TektonStrategy) createBuildPipeline(job *mlopsv1alpha1.NotebookValidationJob, buildConfig *mlopsv1alpha1.BuildConfig, imageRef string) *tektonv1.Pipeline {
    // Determine Dockerfile path
    dockerfilePath := "./Dockerfile"  // Default for generated
    if buildConfig.Dockerfile != "" {
        dockerfilePath = buildConfig.Dockerfile
    }
    
    return &tektonv1.Pipeline{
        // ... existing code ...
        Tasks: []tektonv1.PipelineTask{
            {
                Name: "fetch-repository",
                // ... git-clone task ...
            },
            {
                Name: "generate-dockerfile",  // NEW TASK
                TaskRef: &tektonv1.TaskRef{
                    Name: "generate-dockerfile",
                    Kind: tektonv1.NamespacedTaskKind,
                },
                When: []tektonv1.WhenExpression{
                    {
                        Input:    "$(params.dockerfile-exists)",
                        Operator: "notin",
                        Values:   []string{"true"},
                    },
                },
                Params: []tektonv1.Param{
                    {Name: "BASE_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.base-image)"}},
                },
                RunAfter: []string{"fetch-repository"},
            },
            {
                Name: "build-image",
                TaskRef: &tektonv1.TaskRef{
                    Name: "buildah",
                    Kind: tektonv1.NamespacedTaskKind,
                },
                RunAfter: []string{"generate-dockerfile"},
                Params: []tektonv1.Param{
                    {Name: "IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.image-reference)"}},
                    {Name: "DOCKERFILE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: dockerfilePath}},
                    {Name: "CONTEXT", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "."}},
                },
            },
        },
    }
}
```

## Consequences

### Positive

✅ **Flexibility**: Support both quick validation and production builds  
✅ **User Choice**: Users pick the approach that fits their workflow  
✅ **Backward Compatible**: Existing baseImage configs still work  
✅ **Progressive Enhancement**: Start simple, add Dockerfile later  
✅ **Matches S2I**: Similar to S2I's baseImage approach  
✅ **Production Ready**: Custom Dockerfiles for production deployments  

### Negative

⚠️ **Complexity**: Two code paths to maintain  
⚠️ **Testing**: Need to test both scenarios  
⚠️ **Documentation**: Need to explain both approaches  
⚠️ **Dockerfile Generation**: Need to handle edge cases (no requirements.txt, etc.)  

### Neutral

🔄 **Migration Path**: Users can start with baseImage, migrate to Dockerfile  
🔄 **Precedence Rules**: Clear rules when both are specified  

## Implementation Plan

### Phase 1: Fix Current Build (IMMEDIATE)
1. ✅ Fix parameter names (URL, REVISION) - DONE
2. 🔄 Generate Dockerfile from baseImage dynamically
3. 🔄 Pass DOCKERFILE parameter to buildah Task
4. 🔄 Test with auto-generated Dockerfile

### Phase 2: Add Dockerfile Support (SHORT-TERM)
1. Add `dockerfile` field to BuildConfig CRD
2. Implement Dockerfile detection logic
3. Add conditional Task execution (generate-dockerfile)
4. Test with custom Dockerfiles

### Phase 3: Advanced Features (FUTURE)
1. Dockerfile templates
2. Multi-stage builds
3. Build cache optimization
4. Dockerfile validation

## Testing Strategy

### Test Case 1: Base Image Only
```yaml
buildConfig:
  baseImage: "quay.io/jupyter/minimal-notebook:latest"
  # No dockerfile specified
```
**Expected**: Auto-generate Dockerfile, build succeeds

### Test Case 2: Custom Dockerfile
```yaml
buildConfig:
  dockerfile: "Dockerfile"
  # No baseImage needed
```
**Expected**: Use existing Dockerfile, build succeeds

### Test Case 3: Both Specified
```yaml
buildConfig:
  baseImage: "quay.io/jupyter/minimal-notebook:latest"
  dockerfile: "Dockerfile"
```
**Expected**: Use Dockerfile (precedence), baseImage ignored

### Test Case 4: Neither Specified
```yaml
buildConfig:
  # No baseImage or dockerfile
```
**Expected**: Error with clear message

## References

- ADR-028: Tekton Task Strategy
- ADR-027: S2I Build Strategy (uses baseImage approach)
- ADR-030: Smart Error Messages (applies to build errors)
- Buildah Task Documentation: https://hub.tekton.dev/tekton/task/buildah
- S2I Documentation: https://docs.openshift.com/container-platform/4.18/cicd/builds/understanding-image-builds.html

## Implementation Status

### Phase 1: ✅ COMPLETE (Commit `3c95bc7`)
- Inline `generate-dockerfile` TaskSpec in Pipeline
- Auto-generate Dockerfile from baseImage if not present
- Check standard locations (Dockerfile, Containerfile)
- Use buildah Task for actual build

### Phase 2: ✅ COMPLETE (Commit `7d4fbd8`)
- Added `Dockerfile` field to `BuildConfigSpec` CRD
- Added `dockerfile-path` parameter to Pipeline
- Updated inline task script to check custom path first
- Added fsGroup (65532) to PipelineRun for PVC permissions fix
- Created test samples:
  - `mlops_v1alpha1_notebookvalidationjob_tekton.yaml` (baseImage only)
  - `mlops_v1alpha1_notebookvalidationjob_tekton_custom_dockerfile.yaml` (custom Dockerfile)

**PVC Permissions Fix**: Added `TaskRunTemplate.PodTemplate.SecurityContext.FSGroup = 65532` to fix "Permission denied" errors when git-clone writes to PVC.

### Git Credentials Fix: ✅ COMPLETE (Commit `2f0ce75`)
**Problem**: git-clone Task was using `ssh-directory` workspace with username/password secret, causing authentication failure for HTTPS URLs.

**Root Cause**: BuildConfig and Tekton require different secret formats:
- **BuildConfig**: `kubernetes.io/basic-auth` with `username` + `password` keys
- **Tekton HTTPS**: `Opaque` secret with `.gitconfig` + `.git-credentials` files for `basic-auth` workspace
- **Tekton SSH**: `Opaque` secret with `ssh-privatekey` for `ssh-directory` workspace

**Solution**:
- Changed Pipeline workspace mapping from `ssh-directory` to `basic-auth`
- Created `git-credentials-tekton` secret in correct format
- Added ESO configuration template (`config/samples/eso-git-credentials-tekton.yaml`)
- Updated sample YAMLs to reference new secret

**ESO Integration**: ExternalSecret syncs from same credential source as BuildConfig but formats output for Tekton basic-auth workspace.

### Phase 3: 🔮 FUTURE
- Multi-stage Dockerfile generation
- Dockerfile templates for common scenarios
- Dockerfile validation and linting
- Build cache optimization

## Test Scenarios

### Test 1: BaseImage Only (Auto-Generate Dockerfile)
**File**: `config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml`

```yaml
buildConfig:
  enabled: true
  strategy: "tekton"
  baseImage: "quay.io/jupyter/minimal-notebook:latest"
  # No dockerfile specified - will auto-generate
```

**Expected Behavior**:
1. Pipeline creates inline `generate-dockerfile` task
2. Task checks for Dockerfile/Containerfile in repo
3. Not found → generates Dockerfile from baseImage
4. buildah builds image from generated Dockerfile
5. Validation pod runs with built image

### Test 2: Custom Dockerfile Path
**File**: `config/samples/mlops_v1alpha1_notebookvalidationjob_tekton_custom_dockerfile.yaml`

```yaml
buildConfig:
  enabled: true
  strategy: "tekton"
  dockerfile: "docker/Dockerfile.custom"
  baseImage: "quay.io/jupyter/minimal-notebook:latest"  # Fallback
```

**Expected Behavior**:
1. Pipeline creates inline `generate-dockerfile` task with DOCKERFILE_PATH parameter
2. Task checks custom path first: `docker/Dockerfile.custom`
3. If found → copies to `./Dockerfile` for buildah
4. If not found → falls back to standard locations or auto-generation
5. buildah builds image from Dockerfile
6. Validation pod runs with built image

## Notes

**User Feedback**: This ADR was created in response to excellent user insight: "What if someone does not have a base image yet and has a docker file we may need to support both"

**Follow-up Question**: "Do we need to create a custom tekton task? and use buildah when someone has a docker file"
**Answer**: NO! Use inline TaskSpec (simpler, more maintainable) + buildah for BOTH scenarios

**Sophia Framework Alignment**:
- **Pragmatic Success Criteria**: Support real-world use cases (both scenarios)
- **Explicit Fallibilism**: Acknowledge we can't predict all user needs
- **Systematic Verification**: Test both code paths thoroughly
- **Cognitive Systematization**: Clear rules for precedence and fallback

