# ADR-023: Strategy for Source-to-Image (S2I) Build Integration on OpenShift

## Status
Proposed

## Date
2025-01-08

## Context

The Jupyter Notebook Validator Operator needs to execute notebooks that may have custom dependencies. On OpenShift, Security Context Constraints (SCC) prevent containers from running as specific UIDs and writing to certain directories, causing runtime `pip install` failures.

### Current Problem
- Standard Jupyter images expect to run as UID 1000 with `/home/jovyan` as the home directory
- OpenShift's `restricted-v2` SCC assigns random UIDs (e.g., 1000920000) to containers
- When pip tries to install packages to `/home/jovyan/.local`, it fails with "Permission denied" errors
- The operator currently attempts runtime pip installation, which fails on OpenShift

### Industry Patterns
Major cloud ML platforms solve this by pre-building container images with dependencies before execution:
- **Azure ML**: Environment class automatically builds Docker images from requirements.txt
- **AWS SageMaker**: Bring Your Own Container (BYOC) pattern
- **Google Vertex AI**: Recommends custom containers for production (faster startup, better security)

### OpenShift Capabilities
OpenShift provides native Source-to-Image (S2I) capability that can:
- Automatically build container images from source code
- Detect and install dependencies from requirements.txt
- Create immutable, reproducible images
- Integrate with OpenShift's image registry and build system

## Decision

Adopt OpenShift's Source-to-Image (S2I) as the **primary build mechanism** for creating notebook validation container images with custom dependencies.

### Implementation Approach

When a `NotebookValidationJob` is created with an optional `buildConfig` specification:

1. **Detect OpenShift** by checking for the `build.openshift.io` API group
2. **Create an S2I BuildConfig** that references the notebook's Git repository
3. **Use a Jupyter-compatible S2I builder image** as the base for dependency installation
4. **Trigger the build** and wait for its completion before launching the validation pod
5. **Use the resulting image** for the validation pod rather than installing dependencies at runtime

### CRD Schema Extension

```yaml
apiVersion: mlops.redhat.com/v1alpha1
kind: NotebookValidationJob
spec:
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"  # Optional: use pre-built image
    buildConfig:  # Optional: trigger S2I build
      enabled: true
      strategy: "s2i"  # Default strategy
      baseImage: "quay.io/jupyter/minimal-notebook:latest"
      autoGenerateRequirements: false  # See ADR-024
      requirementsFile: "requirements.txt"  # Path in git repo
```

### Opt-In Behavior

The `buildConfig` field is **optional**, allowing users to:
- **Option 1**: Provide a pre-built image via `containerImage` (no build)
- **Option 2**: Enable `buildConfig` to trigger automatic S2I builds
- **Option 3**: Use default image with runtime pip installation (current behavior, may fail on OpenShift)

## Consequences

### Positive Consequences

1. **Eliminates runtime pip failures** on OpenShift by moving dependency installation to build phase
2. **Immutable, reproducible images**: Dependencies are baked in and cannot drift at runtime
3. **Faster notebook validation**: No per-job installation overhead
4. **Improved security posture**: Dependencies installed at build time under controlled contexts
5. **Alignment with major ML platforms**: Follows Azure ML, SageMaker, Vertex AI patterns
6. **Leverages native OpenShift tooling**: No external CI/CD dependencies needed
7. **Image caching**: Built artifacts can be reused across multiple validation jobs
8. **Better error handling**: Build failures are clearer than runtime permission errors

### Negative Consequences

1. **OpenShift coupling**: Ties the solution to OpenShift's S2I API, reducing portability to vanilla Kubernetes
2. **Prerequisite requirements**: Requires OpenShift cluster with BuildConfig APIs and registry access
3. **Additional operator complexity**: Build orchestration logic must be maintained and tested
4. **Longer time-to-first-validation**: Initial builds add latency before notebook execution
5. **Registry storage usage**: Builds and pushed images consume registry storage quotas
6. **Learning curve**: Users must understand S2I concepts and configuration

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Build failures harder to debug than runtime pip errors | Medium | Stream build logs through operator; surface clear error messages; link to troubleshooting docs |
| S2I builder images become outdated or incompatible | Medium | Define and version S2I builder image requirements; update in lockstep with Jupyter base images |
| Network policies block registry access during build/push | High | Document required network and registry permissions; provide guidance to cluster administrators |
| Users unaware of fallback options if S2I unavailable | Low | Document fallback to user-supplied pre-built images via optional `buildConfig` field |
| Build queue delays in multi-tenant clusters | Medium | Implement build timeout and retry logic; document expected build times |

## Alternatives Considered

### 1. Kaniko-based builds inside Kubernetes pods
**Rejected**: Requires privileged containers or complex RBAC configurations, which conflicts with security goals.

### 2. Buildah via local pod
**Rejected**: Similar security concerns as Kaniko; requires elevated privileges.

### 3. External CI pipelines (Tekton, GitHub Actions)
**Rejected for core**: Adds external dependencies and complexity. Documented as community contribution in ADR-025.

### 4. Runtime pip installation with custom base images (current approach)
**Rejected**: Fails on OpenShift due to SCC restrictions; poor user experience.

### 5. Require users to always provide pre-built images
**Rejected**: Poor user experience; high barrier to entry for notebook validation.

## Implementation Tasks

1. **CRD Schema Updates**
   - [ ] Add `buildConfig` field to `PodConfig` struct
   - [ ] Add validation for `buildConfig` fields
   - [ ] Update CRD documentation and examples

2. **Platform Detection**
   - [ ] Extend `pkg/platform/detector.go` to detect OpenShift
   - [ ] Check for `build.openshift.io` API group
   - [ ] Check for `image.openshift.io` API group

3. **Build Orchestration**
   - [ ] Implement S2I BuildConfig creation logic
   - [ ] Add build status monitoring and waiting
   - [ ] Stream build logs to operator events
   - [ ] Handle build failures with clear error messages

4. **Controller Integration**
   - [ ] Update reconciliation loop to check for `buildConfig`
   - [ ] Trigger builds before creating validation pods
   - [ ] Use built image reference for validation pod

5. **Documentation**
   - [ ] Create S2I prerequisites guide
   - [ ] Document BuildConfig configuration options
   - [ ] Add troubleshooting guide for build failures
   - [ ] Provide example NotebookValidationJob manifests

6. **Testing**
   - [ ] Unit tests for build orchestration logic
   - [ ] Integration tests on OpenShift cluster
   - [ ] E2E tests covering successful and failed builds
   - [ ] Performance tests for build caching

## Related ADRs

- **ADR-008**: Notebook Testing Strategy and Complexity Levels
- **ADR-011**: Error Handling and Retry Strategy
- **ADR-019**: RBAC and Pod Security Policies for Notebook Secret Access
- **ADR-021**: OpenShift-Native Dashboard Strategy
- **ADR-024**: Fallback Strategy for Notebooks Missing requirements.txt (companion ADR)
- **ADR-025**: Community-Contributed Build Methods and Extension Framework (companion ADR)

## References

- [OpenShift S2I Documentation](https://docs.openshift.com/container-platform/latest/openshift_images/using_images/using-s21-images.html)
- [jupyter-on-openshift Project](https://github.com/jupyter-on-openshift/jupyter-notebooks)
- [Azure ML Environment Class](https://learn.microsoft.com/en-us/azure/machine-learning/concept-environments)
- [AWS SageMaker BYOC](https://docs.aws.amazon.com/sagemaker/latest/dg/docker-containers.html)
- [Google Vertex AI Custom Containers](https://cloud.google.com/vertex-ai/docs/training/create-custom-container)
- [OpenShift Security Context Constraints](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)

