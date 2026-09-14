# Build Strategy Implementation Status

**Date:** 2025-01-08  
**Status:** 🔄 IN PROGRESS - Framework Complete, Dependency Resolution Needed

## Overview

We've implemented a pluggable build strategy framework that supports multiple build backends (S2I, Tekton, Kaniko, Shipwright) as outlined in ADR-025. The framework is complete, but we're encountering Go module dependency conflicts that need to be resolved.

## ✅ Completed Work

### 1. Build Strategy Interface (`pkg/build/strategy.go`)

Created a comprehensive build strategy interface with:

- **`Strategy` Interface**: Defines methods all build strategies must implement
  - `Name()`: Returns strategy name
  - `Detect()`: Checks if strategy is available in cluster
  - `CreateBuild()`: Creates a build for a notebook
  - `GetBuildStatus()`: Returns current build status
  - `WaitForCompletion()`: Waits for build to complete
  - `GetBuildLogs()`: Returns build logs
  - `DeleteBuild()`: Cleans up build resources
  - `ValidateConfig()`: Validates build configuration

- **`BuildInfo` Struct**: Contains build information
  - Name, Status, Message, ImageReference
  - StartTime, CompletionTime, Logs

- **`BuildStatus` Enum**: Build status constants
  - Pending, Running, Complete, Failed, Cancelled, Unknown

- **`Registry` Type**: Manages available build strategies
  - `Register()`: Register a strategy
  - `Get()`: Get strategy by name
  - `List()`: List all strategies
  - `DetectAvailable()`: Auto-detect available strategies
  - `GetOrDetect()`: Get specified strategy or auto-detect

- **Error Types**: Custom errors for strategy management
  - `StrategyNotFoundError`
  - `StrategyNotAvailableError`
  - `NoStrategyAvailableError`

### 2. S2I Strategy Implementation (`pkg/build/s2i_strategy.go`)

Implemented OpenShift Source-to-Image strategy:

- **Detection**: Checks for `build.openshift.io` API group
- **Build Creation**: Creates BuildConfig and triggers Build
- **Status Monitoring**: Maps OpenShift build phases to BuildStatus
- **Resource Cleanup**: Deletes Build and BuildConfig
- **Validation**: Ensures baseImage is specified

**Key Features:**
- Uses OpenShift BuildConfig CRD
- Supports git source repositories
- Creates ImageStreamTag for output
- Handles build phase transitions

### 3. Tekton Strategy Implementation (`pkg/build/tekton_strategy.go`)

Implemented Tekton Pipelines strategy:

- **Detection**: Checks for `tekton.dev` API group
- **Build Creation**: Creates Pipeline with git-clone + buildah tasks
- **PipelineRun**: Orchestrates multi-step builds
- **Status Monitoring**: Extracts status from Tekton conditions
- **Resource Cleanup**: Deletes PipelineRun, TaskRun, and Pipeline

**Key Features:**
- Uses Tekton Pipeline and PipelineRun CRDs
- Supports git-clone ClusterTask for source fetching
- Uses buildah ClusterTask for image building
- Configurable registry via strategyConfig
- Workspace management with PVC templates

## ⚠️ Current Blocker: Dependency Conflicts

### Problem

We're encountering Go module version conflicts:

```
k8s.io/client-go@v0.32.8 requires k8s.io/api/networking/v1alpha1
  - This package was removed in k8s.io/api@v0.34.1

github.com/openshift/api requires go >= 1.24.0
  - Latest version incompatible with k8s.io/api@v0.29.2

github.com/tektoncd/pipeline@v1.6.0 requires newer k8s.io versions
  - Downgrading to v0.60.2 causes other conflicts

sigs.k8s.io/structured-merge-diff version mismatch
  - v4 vs v6 incompatibility
```

### Root Cause

The operator was originally built with k8s.io v0.29.2, but:
1. OpenShift API latest version requires Go 1.24+ and k8s.io v0.32+
2. Tekton Pipeline latest version requires k8s.io v0.32+
3. k8s.io v0.32+ removed networking/v1alpha1 package
4. Upgrading k8s.io breaks existing operator code

### Attempted Solutions

1. ✗ Upgrade to k8s.io v0.34.1 - breaks with missing networking/v1alpha1
2. ✗ Downgrade to k8s.io v0.29.2 - incompatible with OpenShift/Tekton APIs
3. ✗ Use OpenShift API v3.9.0+incompatible - retracted by module author
4. ⏳ Need to find compatible version matrix

## 🎯 Next Steps

### Option 1: Resolve Dependency Conflicts (Recommended)

1. **Research Compatible Versions**
   - Find k8s.io version that works with both operator and build APIs
   - Check OpenShift API compatibility matrix
   - Check Tekton Pipeline compatibility matrix

2. **Update go.mod with Compatible Versions**
   ```bash
   # Example (versions TBD after research)
   go get k8s.io/api@v0.30.x
   go get k8s.io/apimachinery@v0.30.x
   go get k8s.io/client-go@v0.30.x
   go get github.com/openshift/api@vX.Y.Z
   go get github.com/tektoncd/pipeline@vX.Y.Z
   ```

3. **Test Build**
   ```bash
   go build ./pkg/build/...
   go test ./pkg/build/...
   ```

### Option 2: Separate Build Module (Alternative)

Create a separate Go module for build strategies:

```
jupyter-notebook-validator-operator/
├── go.mod (operator code, k8s.io v0.29.2)
└── pkg/build/
    ├── go.mod (build strategies, newer k8s.io)
    ├── strategy.go
    ├── s2i_strategy.go
    └── tekton_strategy.go
```

**Pros:**
- Isolates dependency conflicts
- Allows different k8s.io versions

**Cons:**
- More complex module management
- Requires careful interface design

### Option 3: Dynamic Loading (Advanced)

Use Go plugins or gRPC to load build strategies dynamically:

**Pros:**
- Complete isolation
- Can use different Go versions

**Cons:**
- Complex implementation
- Runtime overhead

## 📋 Implementation Checklist

### Phase 1: Dependency Resolution
- [ ] Research compatible version matrix
- [ ] Update go.mod with compatible versions
- [ ] Verify all packages build successfully
- [ ] Run go mod tidy
- [ ] Commit working go.mod and go.sum

### Phase 2: Testing
- [ ] Create unit tests for Strategy interface
- [ ] Create unit tests for S2I strategy
- [ ] Create unit tests for Tekton strategy
- [ ] Create integration tests on OpenShift cluster
- [ ] Verify builds work end-to-end

### Phase 3: Controller Integration
- [ ] Create `internal/controller/build_helper.go`
- [ ] Initialize build strategy registry in controller
- [ ] Register S2I and Tekton strategies
- [ ] Integrate with reconciliation loop
- [ ] Add build status to CR status

### Phase 4: Additional Strategies (Future)
- [ ] Implement Kaniko strategy
- [ ] Implement Shipwright strategy
- [ ] Implement custom strategy support
- [ ] Document community contribution process

## 📚 References

- **ADR-023**: S2I Build Integration on OpenShift
- **ADR-024**: Fallback Strategy for Missing requirements.txt
- **ADR-025**: Community-Contributed Build Methods Extension Framework
- **OpenShift Build API**: https://docs.openshift.com/container-platform/latest/cicd/builds/understanding-image-builds.html
- **Tekton Pipelines**: https://tekton.dev/docs/pipelines/
- **Go Modules**: https://go.dev/ref/mod

## 🔍 Dependency Research Notes

### OpenShift API Versions

- Latest: v0.0.0-20251106190826-ebe535b08719 (requires Go 1.24+)
- Need to find: Stable version compatible with k8s.io v0.29-v0.30

### Tekton Pipeline Versions

- Latest: v1.6.0 (requires k8s.io v0.32+)
- Downgraded: v0.60.2 (compatible with k8s.io v0.29)
- Need to verify: v0.60.2 API compatibility

### K8s.io Version Matrix

| Version | networking/v1alpha1 | OpenShift API | Tekton Pipeline |
|---------|---------------------|---------------|-----------------|
| v0.29.2 | ✅ Present          | ❓ Unknown    | ✅ v0.60.2      |
| v0.30.x | ❓ Unknown          | ❓ Unknown    | ❓ Unknown      |
| v0.32.8 | ❌ Removed          | ✅ Compatible | ✅ v1.6.0       |
| v0.34.1 | ❌ Removed          | ✅ Compatible | ✅ v1.6.0       |

**Action Required**: Fill in the unknown cells through research and testing.

## 💡 Recommendations

1. **Short Term**: Focus on resolving dependency conflicts with Option 1
2. **Medium Term**: Complete S2I and Tekton implementations
3. **Long Term**: Add Kaniko and Shipwright strategies as community contributions

The framework is solid and well-designed. Once dependencies are resolved, implementation can proceed quickly.

