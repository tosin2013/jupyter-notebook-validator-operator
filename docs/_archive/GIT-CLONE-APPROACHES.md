# Git Clone Approaches for Notebook Validation

## Problem Statement

When running on OpenShift with security constraints, we need to clone Git repositories containing Jupyter notebooks. OpenShift enforces `runAsNonRoot` security context, which prevents using standard `alpine/git` images that run as root.

## Solution Approaches

### Approach 1: Non-Root Git Image (Quick Fix) ✅ IMPLEMENTED

**Status**: Implemented in commit `b84ffd5`

**How it works**:
- Use `bitnami/git:latest` instead of `alpine/git:latest`
- Bitnami images are designed to run as non-root users
- Compatible with OpenShift Security Context Constraints (SCC)
- Works with existing init container pattern

**Pros**:
- ✅ Quick fix - no architecture changes
- ✅ Works immediately
- ✅ OpenShift compatible
- ✅ Maintains separation of concerns (git clone vs execution)

**Cons**:
- ⚠️ Still requires init container overhead
- ⚠️ Git clone happens at runtime (slower pod startup)
- ⚠️ Network dependency at pod creation time

**Code Changes**:
```go
// internal/controller/git_helper.go
initContainer := corev1.Container{
    Name:  "git-clone",
    Image: "bitnami/git:latest",  // Changed from alpine/git:latest
    Command: []string{
        "/bin/bash",  // Changed from /bin/sh
        "-c",
        cloneCommand,
    },
    SecurityContext: &corev1.SecurityContext{
        RunAsNonRoot: boolPtr(true),
        AllowPrivilegeEscalation: boolPtr(false),
        Capabilities: &corev1.Capabilities{
            Drop: []corev1.Capability{"ALL"},
        },
    },
}
```

**Testing**:
```bash
# Test with basic validation
oc apply -f config/samples/test-basic-math.yaml

# Check pod status
oc get pods -w

# Verify git-clone init container succeeded
oc describe pod <pod-name>
```

---

### Approach 2: S2I Build Integration (Best Practice) 🎯 RECOMMENDED

**Status**: Available via `buildConfig.enabled: true`

**How it works**:
1. S2I builds a custom container image that includes:
   - Base Jupyter environment
   - Git repository cloned at build time
   - All dependencies from `requirements.txt` installed
   - Runs as non-root user by default
2. No init container needed - everything is baked into the image
3. Validation pod uses the built image directly

**Pros**:
- ✅ **Faster pod startup** - no git clone at runtime
- ✅ **No network dependency** at pod creation
- ✅ **Reproducible builds** - same image every time
- ✅ **Better security** - dependencies vetted at build time
- ✅ **OpenShift native** - uses OpenShift Build API
- ✅ **Eliminates init container complexity**

**Cons**:
- ⚠️ Requires S2I/Tekton available in cluster
- ⚠️ Build time overhead (first run only)
- ⚠️ More complex workflow

**Sample Configuration**:
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: test-s2i-build
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  
  podConfig:
    buildConfig:
      enabled: true
      strategy: "s2i"
      baseImage: "quay.io/jupyter/minimal-notebook:latest"
      requirementsFile: "requirements.txt"
      timeout: "15m"
    
    containerImage: "quay.io/jupyter/minimal-notebook:latest"  # Fallback
    serviceAccountName: "notebook-validator-jupyter-notebook-validator-runner"
```

**Workflow**:
```
1. Controller detects buildConfig.enabled: true
2. S2I BuildConfig created with:
   - Source: Git repository
   - Builder: Python S2I image
   - Output: Custom image with notebook + deps
3. Build triggered and monitored
4. Built image reference retrieved
5. Validation pod created using built image
6. Notebook executed (already present in image)
```

**Testing**:
```bash
# Test S2I build workflow
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml

# Watch build progress
oc get builds -w

# Check build logs
oc logs -f build/<build-name>

# Verify validation pod uses built image
oc get pod <validation-pod> -o yaml | grep image:
```

---

## Comparison Matrix

| Feature | Init Container (bitnami/git) | S2I Build |
|---------|------------------------------|-----------|
| **Pod Startup Time** | Slower (git clone at runtime) | Faster (pre-built) |
| **Network Dependency** | Required at pod creation | Only at build time |
| **OpenShift Compatible** | ✅ Yes | ✅ Yes |
| **Reproducibility** | ⚠️ Git ref can change | ✅ Image is immutable |
| **Dependency Management** | Runtime pip install | Build-time installation |
| **Security Scanning** | Limited | Full image scanning |
| **Complexity** | Low | Medium |
| **Best For** | Quick tests, development | Production, CI/CD |

---

## Recommendations

### For Development/Testing
Use **Approach 1** (bitnami/git init container):
- Quick iteration
- No build overhead
- Simple debugging

### For Production/CI/CD
Use **Approach 2** (S2I Build):
- Better performance
- Reproducible builds
- Enhanced security
- Proper dependency management

### Hybrid Approach
The operator supports both! Use `buildConfig.enabled: false` for quick tests, and `buildConfig.enabled: true` for production workloads.

---

## Future Enhancements

### Approach 3: Tekton Pipelines (Phase 4.5)
- More flexible than S2I
- Support for multi-stage builds
- Custom build steps
- Already implemented in `pkg/build/tekton_strategy.go`

### Approach 4: Pre-built Images
- Maintain a library of pre-built images
- Tag images by notebook repository + commit SHA
- Instant pod startup
- Requires image registry management

---

## Migration Path

If you're currently using init containers and want to migrate to S2I:

1. **Test S2I build** with a simple notebook:
   ```bash
   oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml
   ```

2. **Verify build completes** successfully:
   ```bash
   oc get builds
   oc logs -f build/<build-name>
   ```

3. **Update your NotebookValidationJob** to enable builds:
   ```yaml
   spec:
     podConfig:
       buildConfig:
         enabled: true
         strategy: "s2i"
   ```

4. **Monitor first build** - subsequent runs will reuse the image if source hasn't changed

---

## Troubleshooting

### Init Container Fails with "runAsNonRoot"
**Symptom**: `Error: container has runAsNonRoot and image will run as root`

**Solution**: Ensure using `bitnami/git:latest` not `alpine/git:latest`

### S2I Build Fails
**Symptom**: Build pod fails or times out

**Solutions**:
- Check build logs: `oc logs -f build/<build-name>`
- Verify base image is accessible
- Check requirements.txt syntax
- Increase build timeout in spec

### Git Clone Fails
**Symptom**: Init container or build fails to clone repository

**Solutions**:
- Verify Git URL is accessible
- Check credentials if private repo
- Verify branch/ref exists
- Check network policies

---

## References

- ADR-005: OpenShift Compatibility
- ADR-009: Git Integration with Credentials
- Phase 4.5: S2I Build Integration
- [Bitnami Git Container](https://github.com/bitnami/containers/tree/main/bitnami/git)
- [OpenShift S2I Documentation](https://docs.openshift.com/container-platform/latest/cicd/builds/understanding-image-builds.html)

