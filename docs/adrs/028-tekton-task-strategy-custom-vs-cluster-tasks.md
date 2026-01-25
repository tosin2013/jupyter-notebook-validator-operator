# ADR-028: Tekton Task Strategy - Custom Tasks vs Cluster Tasks

**Status**: Implemented
**Date**: 2025-11-09
**Updated**: 2026-01-24
**Authors**: Sophia (AI Assistant), Tosin Akinosho
**Related ADRs**: ADR-027 (S2I Build Strategy - Superseded), ADR-031 (Tekton Build - Primary)

## Context

We are implementing Tekton Pipelines support for building Jupyter notebook images as an alternative to OpenShift S2I builds. During implementation, we discovered a critical architectural decision: should we create custom Tekton Tasks specific to our operator, or should we rely on the cluster-provided Tasks in the `openshift-pipelines` namespace?

### Current Situation

**OpenShift Pipelines Installation**:
- OpenShift 4.18.21 includes Tekton Pipelines (OpenShift Pipelines Operator)
- Cluster-provided Tasks are installed in `openshift-pipelines` namespace
- Available Tasks: `git-clone`, `buildah`, `s2i-python`, `s2i-nodejs`, etc.
- Tasks are **NOT** ClusterTasks (cluster-scoped), they are namespace-scoped Tasks

**Current Implementation Issue**:
```yaml
# Current code tries to use ClusterTask
TaskRef:
  Name: "git-clone"
  Kind: "ClusterTask"  # ❌ FAILS - No ClusterTasks exist

# Actual resources in cluster
$ oc get clustertasks
No resources found

$ oc get tasks -n openshift-pipelines
NAME                      AGE
git-clone                 46h
buildah                   46h
s2i-python                46h
```

**Error from Previous Attempt**:
```
Pipeline default/notebookvalidationjob-tekton-sample-pipeline can't be Run; 
it contains Tasks that don't exist: Couldn't retrieve Task "git-clone": 
clustertasks.tekton.dev "git-clone" not found
```

### Strategic Question

**Should we**:
1. **Option A**: Create custom Tasks in the operator's namespace that are tailored to our specific needs?
2. **Option B**: Reference the cluster-provided Tasks from `openshift-pipelines` namespace?
3. **Option C**: Hybrid approach - custom Tasks that fall back to cluster Tasks?

## Decision Drivers

### 1. **Portability Across OpenShift Versions**
- OpenShift Pipelines versions change across releases
- Task APIs may evolve (v1beta1 → v1)
- Task parameters and workspaces may change

### 2. **Control and Customization**
- Jupyter notebook builds have specific requirements
- May need custom steps for notebook validation
- May need to integrate with OpenShift AI ImageStreams

### 3. **Maintenance Burden**
- Custom Tasks require ongoing maintenance
- Cluster Tasks are maintained by Red Hat
- Need to balance control vs maintenance

### 4. **User Experience**
- Users expect operator to "just work"
- Should not require manual Task installation
- Should work across different OpenShift versions

### 5. **CRD Research Best Practice** (from ADR-027)
- Always research actual cluster capabilities
- Use platform-provided features when appropriate
- Don't reinvent what platform provides

## Decision

**We will implement a HYBRID APPROACH (Option C)** with the following strategy:

### Phase 1: Copy Tasks to User's Namespace (IMMEDIATE) ⭐ **KEY DECISION**

**Critical Insight**: Tasks are namespace-scoped resources. To avoid cross-namespace reference issues and RBAC complexity, we will **copy custom Tasks to the user's target namespace** when creating Pipelines.

**Why This Matters**:
1. **RBAC Simplicity**: No cross-namespace permissions needed
2. **Isolation**: Each user namespace has its own Task copies
3. **Customization**: Users can modify Tasks in their namespace if needed
4. **Portability**: Works consistently across different OpenShift configurations
5. **No Shared State**: Avoids conflicts between different users/projects

**Implementation Strategy**:
```go
// When creating a Pipeline, first ensure Tasks exist in target namespace
func (t *TektonStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
    // Step 1: Copy required Tasks to user's namespace
    if err := t.ensureTasksInNamespace(ctx, job.Namespace); err != nil {
        return nil, fmt.Errorf("failed to ensure tasks: %w", err)
    }

    // Step 2: Create Pipeline (references Tasks in same namespace)
    pipeline := t.createBuildPipeline(job, buildConfig, imageRef)
    if err := t.client.Create(ctx, pipeline); err != nil {
        return nil, fmt.Errorf("failed to create pipeline: %w", err)
    }

    // Step 3: Create PipelineRun
    // ...
}

func (t *TektonStrategy) ensureTasksInNamespace(ctx context.Context, namespace string) error {
    logger := log.FromContext(ctx)

    // List of Tasks to copy
    requiredTasks := []string{
        "jupyter-notebook-git-clone",
        "jupyter-notebook-buildah",
    }

    for _, taskName := range requiredTasks {
        // Check if Task already exists in target namespace
        existingTask := &tektonv1.Task{}
        err := t.client.Get(ctx, client.ObjectKey{
            Name:      taskName,
            Namespace: namespace,
        }, existingTask)

        if err == nil {
            // Task exists, check if it needs updating
            if t.shouldUpdateTask(existingTask) {
                logger.Info("Updating Task in namespace", "task", taskName, "namespace", namespace)
                if err := t.updateTask(ctx, existingTask); err != nil {
                    return err
                }
            }
            continue
        }

        if !errors.IsNotFound(err) {
            return fmt.Errorf("failed to check task %s: %w", taskName, err)
        }

        // Task doesn't exist, copy it
        logger.Info("Copying Task to namespace", "task", taskName, "namespace", namespace)
        if err := t.copyTaskToNamespace(ctx, taskName, namespace); err != nil {
            return err
        }
    }

    return nil
}
```

**Task Definition** (stored in operator as template):
```yaml
# Custom Task: jupyter-notebook-git-clone
# Copied to: user's target namespace (e.g., default, my-project, etc.)
# Based on: openshift-pipelines/git-clone but with notebook-specific features
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: jupyter-notebook-git-clone
  # namespace: <dynamically set to user's namespace>
  labels:
    app.kubernetes.io/managed-by: jupyter-notebook-validator-operator
    mlops.redhat.com/task-type: git-clone
spec:
  description: Clone git repository with Jupyter notebook validation
  params:
    - name: url
      type: string
    - name: revision
      type: string
      default: main
    - name: subdirectory
      type: string
      default: ""
  workspaces:
    - name: output
    - name: ssh-directory
      optional: true
  steps:
    - name: clone
      image: registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8:latest
      script: |
        #!/bin/sh
        set -e
        # Clone repository
        git clone $(params.url) $(workspaces.output.path)/$(params.subdirectory)
        cd $(workspaces.output.path)/$(params.subdirectory)
        git checkout $(params.revision)

        # Jupyter-specific: Validate notebook structure
        if [ -d "notebooks" ]; then
          echo "✓ Found notebooks directory"
          find notebooks -name "*.ipynb" | wc -l | xargs echo "Found notebooks:"
        fi
```

### Phase 2: Cluster Task Detection and Fallback (FUTURE)

**Add intelligent Task resolution**:
1. Check if cluster Tasks are available in `openshift-pipelines` namespace
2. Use cluster Tasks if available and compatible
3. Fall back to custom Tasks if cluster Tasks are missing or incompatible

**Implementation**:
```go
// TaskResolver determines which Task to use
type TaskResolver struct {
    client client.Client
}

func (r *TaskResolver) ResolveTask(ctx context.Context, taskName string) (*tektonv1.TaskRef, error) {
    // Try cluster Task first
    if r.isClusterTaskAvailable(ctx, taskName) {
        return &tektonv1.TaskRef{
            Name: taskName,
            Kind: "Task",
            // Reference from openshift-pipelines namespace
            Resolver: "cluster",
            Params: []tektonv1.Param{
                {Name: "namespace", Value: tektonv1.ParamValue{StringVal: "openshift-pipelines"}},
            },
        }, nil
    }
    
    // Fall back to custom Task
    return &tektonv1.TaskRef{
        Name: fmt.Sprintf("jupyter-notebook-%s", taskName),
        Kind: "Task",
    }, nil
}
```

### Phase 3: Task Versioning and Update Strategy (FUTURE)

**Add Task version management**:
1. Add version label to Tasks (e.g., `mlops.redhat.com/task-version: "1.0.0"`)
2. Check Task version before copying
3. Update Tasks in user namespaces when operator is upgraded
4. Provide opt-out mechanism for users who customized Tasks

**Implementation**:
```go
func (t *TektonStrategy) shouldUpdateTask(existingTask *tektonv1.Task) bool {
    // Check if Task is managed by operator
    if existingTask.Labels["app.kubernetes.io/managed-by"] != "jupyter-notebook-validator-operator" {
        // User created their own Task, don't update
        return false
    }

    // Check version
    currentVersion := existingTask.Labels["mlops.redhat.com/task-version"]
    requiredVersion := "1.0.0" // From operator's Task template

    if currentVersion != requiredVersion {
        return true
    }

    return false
}
```

### Phase 4: OpenShift Version Detection (FUTURE)

**Add OpenShift version detection**:
1. Detect OpenShift Pipelines version
2. Select compatible Task API version
3. Adjust parameters based on version

## Base Image Strategy

### Critical Consideration: Papermill Dependency

**Problem**: Our validation workflow requires Python + Papermill to execute notebooks, but Tekton Tasks need different base images for different purposes:

1. **Git Clone Task**: Needs git binary
2. **Build Task**: Needs buildah/docker for container builds
3. **Validation Task**: Needs Python + Papermill + Jupyter dependencies

**Current Cluster Images** (OpenShift-specific):
- Git Clone: `registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8@sha256:4fabae...`
- Buildah: `registry.redhat.io/rhel8/buildah@sha256:6d2dcb...`

### Decision: Multi-Image Strategy with Platform Detection

**We will use different base images based on platform and task purpose**:

#### 1. Git Clone Task Images

**OpenShift (Preferred)**:
```yaml
image: registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8:latest
# Pros: Optimized for OpenShift, includes /ko-app/git-init binary
# Cons: OpenShift-specific, requires Red Hat registry access
```

**Kubernetes (Fallback)**:
```yaml
image: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.50.0
# Pros: Works on vanilla Kubernetes, official Tekton image
# Cons: Requires internet access to gcr.io
```

**Universal (Alternative)**:
```yaml
image: alpine/git:latest
# Pros: Small, widely available, works everywhere
# Cons: Need to implement git-init logic ourselves
```

#### 2. Build Task Images

**OpenShift (Preferred)**:
```yaml
image: registry.redhat.io/rhel8/buildah:latest
# Pros: Optimized for OpenShift, rootless support, SCC-compliant
# Cons: OpenShift-specific
```

**Kubernetes (Fallback)**:
```yaml
image: quay.io/buildah/stable:latest
# Pros: Works on vanilla Kubernetes, official buildah image
# Cons: May need privileged mode on some clusters
```

#### 3. Notebook Validation Task Images (NEW - Not in this ADR)

**Note**: Validation happens in the validation pod, not in Tekton Tasks. The built image from Tekton already contains Python + Papermill because we're building from Jupyter base images (e.g., `quay.io/jupyter/minimal-notebook` or OpenShift AI images).

**Tekton Tasks only build the image** - they don't execute notebooks.

### Platform Detection Strategy

**Implementation**:
```go
// PlatformDetector determines which base images to use
type PlatformDetector struct {
    client client.Client
}

func (p *PlatformDetector) DetectPlatform(ctx context.Context) (string, error) {
    // Check for OpenShift-specific resources
    routeList := &routev1.RouteList{}
    if err := p.client.List(ctx, routeList); err == nil {
        return "openshift", nil
    }

    // Check for OpenShift API server
    discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
    if err == nil {
        apiList, err := discoveryClient.ServerGroups()
        if err == nil {
            for _, group := range apiList.Groups {
                if group.Name == "route.openshift.io" {
                    return "openshift", nil
                }
            }
        }
    }

    return "kubernetes", nil
}

func (p *PlatformDetector) GetGitCloneImage(platform string) string {
    switch platform {
    case "openshift":
        return "registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8:latest"
    case "kubernetes":
        return "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.50.0"
    default:
        return "alpine/git:latest"
    }
}

func (p *PlatformDetector) GetBuildahImage(platform string) string {
    switch platform {
    case "openshift":
        return "registry.redhat.io/rhel8/buildah:latest"
    case "kubernetes":
        return "quay.io/buildah/stable:latest"
    default:
        return "quay.io/buildah/stable:latest"
    }
}
```

### Image Selection in Task Templates

**Tasks will be parameterized with image selection**:
```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: jupyter-notebook-git-clone
spec:
  params:
    - name: gitInitImage
      type: string
      description: Git init image to use (platform-specific)
      default: "registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8:latest"
  steps:
    - name: clone
      image: $(params.gitInitImage)
      # ... rest of task
```

**Operator sets image based on platform detection**:
```go
func (t *TektonStrategy) createPipelineRun(job *mlopsv1alpha1.NotebookValidationJob, platform string) *tektonv1.PipelineRun {
    gitInitImage := t.platformDetector.GetGitCloneImage(platform)
    buildahImage := t.platformDetector.GetBuildahImage(platform)

    return &tektonv1.PipelineRun{
        Spec: tektonv1.PipelineRunSpec{
            Params: []tektonv1.Param{
                {Name: "git-init-image", Value: tektonv1.ParamValue{StringVal: gitInitImage}},
                {Name: "buildah-image", Value: tektonv1.ParamValue{StringVal: buildahImage}},
                // ... other params
            },
        },
    }
}
```

### Papermill Consideration

**Important**: Papermill is NOT needed in Tekton Tasks. Here's why:

1. **Tekton Tasks**: Build the container image from source code
   - Input: Git repository with notebooks
   - Output: Container image with notebooks baked in
   - Base image: Jupyter image (already has Python + Papermill)

2. **Validation Pod**: Executes the notebooks using Papermill
   - Input: Built container image (from Tekton)
   - Container: Uses the built image which already has Papermill
   - Execution: Papermill runs inside the validation container

**Flow**:
```
Git Repo → Tekton (git-clone + buildah) → Built Image (has Papermill) → Validation Pod (runs Papermill)
```

**No Papermill needed in Tekton Tasks** because:
- We're building FROM Jupyter images that already have Papermill
- Tekton just copies notebooks into the image
- Validation pod uses the built image to execute notebooks

### Image Registry Considerations

**OpenShift**:
- Internal registry: `image-registry.openshift-image-registry.svc:5000`
- Red Hat registry: `registry.redhat.io` (requires pull secret)
- Quay.io: `quay.io` (public images)

**Kubernetes**:
- Docker Hub: `docker.io` (rate limiting!)
- Quay.io: `quay.io` (recommended)
- GCR: `gcr.io` (for Tekton official images)

**Recommendation**: Use Quay.io for cross-platform compatibility, fall back to platform-specific registries when available.

## Consequences

### Positive

1. **✅ Immediate Fix**: Custom Tasks solve the current ClusterTask issue
2. **✅ Portability**: Works across OpenShift versions without depending on cluster Tasks
3. **✅ Control**: Can optimize Tasks for Jupyter notebook builds
4. **✅ User Experience**: Operator installs everything needed
5. **✅ Future Flexibility**: Can add cluster Task detection later
6. **✅ RBAC Simplicity**: No cross-namespace permissions needed
7. **✅ Isolation**: Each namespace has its own Task copies
8. **✅ Customization**: Users can modify Tasks in their namespace
9. **✅ No Shared State**: Avoids conflicts between users/projects

### Negative

1. **⚠️ Maintenance**: Need to maintain custom Tasks
2. **⚠️ Duplication**: Some overlap with cluster Tasks
3. **⚠️ Updates**: Need to track Tekton API changes
4. **⚠️ Resource Duplication**: Tasks copied to each namespace (minimal overhead)
5. **⚠️ Update Propagation**: Need strategy for updating Tasks in user namespaces

### Neutral

1. **📊 Complexity**: Hybrid approach adds some complexity
2. **📊 Testing**: Need to test with and without cluster Tasks
3. **📊 Versioning**: Need to track Task versions for updates

## Implementation Plan

### Immediate (This Session)

1. ✅ Create ADR-028 documenting strategy with namespace copy approach and base image strategy
2. ⏳ Implement platform detection (OpenShift vs Kubernetes)
3. ⏳ Create custom `jupyter-notebook-git-clone` Task template with parameterized images
4. ⏳ Create custom `jupyter-notebook-buildah` Task template with parameterized images
5. ⏳ Implement `ensureTasksInNamespace()` function to copy Tasks
6. ⏳ Update `tekton_strategy.go` to:
   - Detect platform
   - Select appropriate base images
   - Copy Tasks before creating Pipeline
   - Pass image parameters to PipelineRun
7. ⏳ Add Task version tracking and update logic
8. ⏳ Test end-to-end Tekton build flow with Task copying on OpenShift
9. ⏳ Test on vanilla Kubernetes (if available)

### Short-Term (This Week)

1. Add cluster Task detection logic
2. Implement fallback mechanism
3. Add comprehensive logging for Task resolution
4. Document Task customization options

### Long-Term (Future)

1. Add OpenShift version detection
2. Implement version-specific Task selection
3. Add Task performance benchmarking
4. Consider contributing improvements back to OpenShift Pipelines

## Alternatives Considered

### Alternative 1: Only Use Cluster Tasks

**Pros**:
- No maintenance burden
- Automatically get updates from Red Hat

**Cons**:
- ❌ Doesn't work - cluster Tasks are namespace-scoped, not ClusterTasks
- ❌ No control over Task behavior
- ❌ Can't optimize for Jupyter notebooks
- ❌ Breaks if cluster Tasks change

**Rejected**: Current implementation doesn't work with this approach

### Alternative 2: Only Use Custom Tasks

**Pros**:
- Full control
- Consistent across environments
- Can optimize for our use case

**Cons**:
- ❌ High maintenance burden
- ❌ Miss out on Red Hat improvements
- ❌ Need to track Tekton API changes

**Rejected**: Too much maintenance burden

### Alternative 3: Require Users to Install Tasks

**Pros**:
- No operator complexity
- Users can customize

**Cons**:
- ❌ Poor user experience
- ❌ Violates "operator should just work" principle
- ❌ Increases support burden

**Rejected**: Violates user experience goals

## References

- [Tekton Tasks Documentation](https://tekton.dev/docs/pipelines/tasks/)
- [OpenShift Pipelines Documentation](https://docs.openshift.com/pipelines/latest/)
- [Tekton Task Resolution](https://tekton.dev/docs/pipelines/resolution/)
- ADR-027: S2I Build Strategy (CRD Research Best Practice)

## Notes

- This ADR follows the CRD research best practice from ADR-027
- Researched actual cluster capabilities before making decision
- Hybrid approach balances control vs maintenance
- Can evolve strategy based on real-world usage

