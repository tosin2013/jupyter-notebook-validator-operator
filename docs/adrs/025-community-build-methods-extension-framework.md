# ADR-025: Community-Contributed Build Methods and Extension Framework

## Status
Proposed

## Date
2025-01-08

## Context

While OpenShift S2I is the primary build mechanism for the operator (ADR-023), the Kubernetes ecosystem offers many alternative build tools:
- **Tekton Pipelines**: Cloud-native CI/CD framework
- **Shipwright**: Unified build API across different tools
- **Kaniko**: Build images without Docker daemon
- **Buildah**: OCI-compliant container image builder
- **BuildKit**: Modern Docker build backend

### Current Situation
- Different organizations have existing CI/CD pipelines
- Some users prefer specific build tools
- Vanilla Kubernetes clusters don't have OpenShift S2I
- Community wants to contribute alternative implementations

### Constraints
- S2I must remain the officially supported method
- Core operator should stay lightweight and maintainable
- Need clear extension points for community innovation
- Must maintain security and quality standards

### Goals
- Enable community contributions without bloating core
- Support diverse Kubernetes environments
- Provide clear extension framework
- Maintain consistent user experience

## Decision

Establish a **community contribution framework** for alternative build methods with clear boundaries between core and community support.

### Core Support: S2I Only

OpenShift S2I builds are the **only officially supported** build method in the core operator:
- Fully tested and documented
- Maintained by core team
- Included in release testing
- Covered by support SLAs

### Extension Interface: BuildStrategy

Define a pluggable `BuildStrategy` interface in the operator:

```go
// BuildStrategy defines the interface for different build backends
type BuildStrategy interface {
    // Name returns the strategy name (e.g., "s2i", "tekton", "kaniko")
    Name() string
    
    // Detect checks if this strategy is available in the cluster
    Detect(ctx context.Context, client client.Client) (bool, error)
    
    // CreateBuild creates a build for the given notebook validation job
    CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error
    
    // GetBuildStatus returns the current build status
    GetBuildStatus(ctx context.Context, buildName string) (BuildStatus, error)
    
    // GetBuiltImage returns the image reference after successful build
    GetBuiltImage(ctx context.Context, buildName string) (string, error)
    
    // DeleteBuild cleans up build resources
    DeleteBuild(ctx context.Context, buildName string) error
}

// BuildStatus represents the current state of a build
type BuildStatus struct {
    Phase      string    // Pending, Running, Succeeded, Failed
    Message    string    // Human-readable status message
    StartTime  time.Time
    CompletionTime time.Time
    Logs       string    // Build logs (if available)
}
```

### Community Directory Structure

```
docs/
├── adrs/
│   ├── 023-s2i-build-integration-openshift.md
│   ├── 024-fallback-strategy-missing-requirements.md
│   └── 025-community-build-methods-extension-framework.md
└── community/
    ├── README.md                          # Community contribution guide
    ├── build-strategies/
    │   ├── README.md                      # Build strategy overview
    │   ├── tekton/
    │   │   ├── README.md                  # Tekton implementation guide
    │   │   ├── implementation.go          # Reference implementation
    │   │   ├── pipeline.yaml              # Example Tekton Pipeline
    │   │   └── tests/                     # Test suite
    │   ├── kaniko/
    │   │   ├── README.md                  # Kaniko implementation guide
    │   │   ├── implementation.go          # Reference implementation
    │   │   ├── pod-template.yaml          # Example Kaniko pod
    │   │   └── tests/                     # Test suite
    │   └── shipwright/
    │       ├── README.md                  # Shipwright implementation guide
    │       ├── implementation.go          # Reference implementation
    │       ├── build.yaml                 # Example Shipwright Build
    │       └── tests/                     # Test suite
    └── CONTRIBUTING.md                    # Contribution guidelines
```

### CRD Schema Extension

```yaml
apiVersion: mlops.redhat.com/v1alpha1
kind: NotebookValidationJob
spec:
  podConfig:
    buildConfig:
      enabled: true
      strategy: "s2i"  # Options: "s2i" (core), "tekton", "kaniko", "shipwright" (community)
      strategyConfig:  # Strategy-specific configuration
        # For S2I (core)
        baseImage: "quay.io/jupyter/minimal-notebook:latest"
        
        # For Tekton (community)
        pipelineRef: "notebook-build-pipeline"
        
        # For Kaniko (community)
        executor: "gcr.io/kaniko-project/executor:latest"
        
        # For Shipwright (community)
        buildStrategyRef: "buildpacks-v3"
```

### Feature Flag System

```go
// Feature flags for build strategies
const (
    FeatureFlagS2I        = "build.strategy.s2i"        // Always enabled (core)
    FeatureFlagTekton     = "build.strategy.tekton"     // Community
    FeatureFlagKaniko     = "build.strategy.kaniko"     // Community
    FeatureFlagShipwright = "build.strategy.shipwright" // Community
)

// Enable community strategies via operator configuration
type OperatorConfig struct {
    EnabledBuildStrategies []string `json:"enabledBuildStrategies"`
}
```

### Validation Requirements

All community build strategies must:
1. **Implement BuildStrategy interface** completely
2. **Pass security review** by core team
3. **Include comprehensive tests** (unit, integration, e2e)
4. **Provide clear documentation** with examples
5. **Handle errors gracefully** with actionable messages
6. **Support the same features** as S2I (requirements.txt, git repos, etc.)
7. **Be maintained** by community contributors

## Consequences

### Positive Consequences

1. **Encourages community innovation** and contributions
2. **Supports diverse Kubernetes environments** beyond OpenShift
3. **Allows organizations to use existing CI/CD tools**
4. **Keeps core operator lightweight** and maintainable
5. **Provides clear extension points** for advanced users
6. **Enables experimentation** with new build technologies
7. **Increases adoption** across different platforms
8. **Builds community engagement** and ownership

### Negative Consequences

1. **Community strategies may have varying quality** and maintenance
2. **Support burden** for community contributions
3. **Testing matrix expands** with each strategy
4. **Documentation must be maintained** for multiple approaches
5. **Users may be confused** about which strategy to use
6. **Fragmentation risk** if strategies diverge too much
7. **Security review overhead** for new contributions

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Community contributions become unmaintained | High | Clear labeling of community vs official; deprecation policy; community ownership model |
| Security vulnerabilities in community code | High | Mandatory security review; automated scanning; clear security guidelines |
| Incompatible changes in community build tools | Medium | Version pinning; compatibility testing; migration guides |
| Fragmentation of user base across strategies | Medium | Decision tree in docs; clear recommendations; feature parity matrix |
| Poor quality community contributions | Medium | Contribution guidelines; code review process; quality gates |
| Support requests for community strategies | Low | Clear support boundaries; community support channels; FAQ |

## Alternatives Considered

### 1. Hard-code support for multiple build tools in core
**Rejected**: Bloats codebase; increases maintenance burden; slows development; testing complexity.

### 2. No extension framework, S2I only
**Rejected**: Limits adoption on non-OpenShift clusters; ignores community needs; reduces flexibility.

### 3. Separate operators for each build strategy
**Rejected**: Fragments ecosystem; poor UX; duplicates code; confusing for users.

### 4. External webhook-based build triggers
**Rejected**: Adds complexity; security concerns; network dependencies; harder to debug.

### 5. Leave community contributions completely undocumented
**Rejected**: Poor discoverability; inconsistent quality; no guidance; wasted effort.

## Implementation Tasks

1. **Core Interface**
   - [ ] Define `BuildStrategy` interface
   - [ ] Implement S2I strategy using interface
   - [ ] Add strategy registry and discovery
   - [ ] Implement feature flag system

2. **Community Framework**
   - [ ] Create `docs/community/` directory structure
   - [ ] Write contribution guidelines
   - [ ] Create build strategy template
   - [ ] Add security review checklist

3. **Reference Implementations**
   - [ ] Implement Tekton reference strategy
   - [ ] Implement Kaniko reference strategy
   - [ ] Add comprehensive tests for each
   - [ ] Document integration patterns

4. **Documentation**
   - [ ] Create decision tree for strategy selection
   - [ ] Document each community strategy
   - [ ] Add troubleshooting guides
   - [ ] Create video tutorials

5. **Testing**
   - [ ] Unit tests for strategy interface
   - [ ] Integration tests for each strategy
   - [ ] E2E tests on different platforms
   - [ ] Performance benchmarks

6. **Community Engagement**
   - [ ] Create GitHub issues for strategy contributions
   - [ ] Set up community meetings
   - [ ] Create contributor recognition program
   - [ ] Establish support channels

## Strategy Selection Decision Tree

```
┌─────────────────────────────────────┐
│ Do you have OpenShift cluster?     │
└─────────────┬───────────────────────┘
              │
         ┌────┴────┐
         │   YES   │
         └────┬────┘
              │
         ┌────▼────────────────────────┐
         │ Use S2I (officially         │
         │ supported, recommended)     │
         └─────────────────────────────┘
              
         ┌────┴────┐
         │   NO    │
         └────┬────┘
              │
         ┌────▼────────────────────────┐
         │ Do you have Tekton?         │
         └─────────┬───────────────────┘
                   │
              ┌────┴────┐
              │   YES   │
              └────┬────┘
                   │
              ┌────▼────────────────────┐
              │ Use Tekton (community)  │
              └─────────────────────────┘
              
              ┌────┴────┐
              │   NO    │
              └────┬────┘
                   │
              ┌────▼────────────────────┐
              │ Use Kaniko (community)  │
              │ or provide pre-built    │
              │ images                  │
              └─────────────────────────┘
```

## Related ADRs

- **ADR-023**: Strategy for Source-to-Image (S2I) Build Integration on OpenShift (core strategy)
- **ADR-024**: Fallback Strategy for Notebooks Missing requirements.txt
- **ADR-022**: Community Observability Contributions (similar community framework)

## References

- [Tekton Pipelines](https://tekton.dev/)
- [Shipwright Build](https://shipwright.io/)
- [Kaniko](https://github.com/GoogleContainerTools/kaniko)
- [Buildah](https://buildah.io/)
- [BuildKit](https://github.com/moby/buildkit)
- [Operator SDK Extensibility Patterns](https://sdk.operatorframework.io/)

