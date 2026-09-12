# OpenShift AI Integration Guide

## Overview

The Jupyter Notebook Validator Operator automatically integrates with OpenShift AI (formerly Red Hat OpenShift Data Science) to provide users with S2I-enabled Jupyter notebook images optimized for notebook validation workloads.

## Features

### Automatic ImageStream Discovery

When OpenShift AI is installed, the operator automatically:

1. **Detects OpenShift AI** - Checks for the `redhat-ods-applications` namespace
2. **Lists Available Images** - Discovers all S2I-enabled Jupyter images
3. **Provides Recommendations** - Suggests the best image for your workload
4. **Exposes Metadata** - Shows image descriptions, tags, and capabilities

### Available Images

OpenShift AI provides two primary S2I-enabled images:

#### 1. Minimal Python (`s2i-minimal-notebook`)
- **Display Name**: Minimal Python
- **Description**: Jupyter notebook image with minimal dependency set to start experimenting with Jupyter environment
- **Use Case**: Lightweight notebooks with custom dependencies
- **Image Reference**: `image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1`
- **Available Tags**: 1.2, 2023.1, 2023.2, 2024.1, 2024.2, 2025.1

#### 2. Standard Data Science (`s2i-generic-data-science-notebook`)
- **Display Name**: Standard Data Science
- **Description**: Jupyter notebook image with a set of data science libraries that advanced AI/ML notebooks will use as a base image
- **Use Case**: Data science workloads with pre-installed libraries
- **Image Reference**: `image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-generic-data-science-notebook:2025.1`
- **Available Tags**: 1.2, 2023.1, 2023.2, 2024.1, 2024.2, 2025.1

## User Workflow

### Step 1: Check Available Images

After creating a NotebookValidationJob with S2I enabled, check the status to see available images:

```bash
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus.availableImages}' | jq .
```

**Example Output**:
```json
[
  {
    "description": "Jupyter notebook image with minimal dependency set...",
    "displayName": "Minimal Python",
    "imageRef": "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1",
    "name": "s2i-minimal-notebook",
    "s2iEnabled": true,
    "tags": ["1.2", "2023.1", "2023.2", "2024.1", "2024.2", "2025.1"]
  },
  {
    "description": "Jupyter notebook image with a set of data science libraries...",
    "displayName": "Standard Data Science",
    "imageRef": "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-generic-data-science-notebook:2025.1",
    "name": "s2i-generic-data-science-notebook",
    "s2iEnabled": true,
    "tags": ["1.2", "2023.1", "2023.2", "2024.1", "2024.2", "2025.1"]
  }
]
```

### Step 2: Get Recommended Image

The operator automatically recommends the best image for your workload:

```bash
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus.recommendedImage}'
```

**Example Output**:
```
image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1
```

### Step 3: Choose Your Image

Create or update your NotebookValidationJob to use your preferred image:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation-job
  namespace: default
spec:
  notebook:
    git:
      url: "https://github.com/your-org/your-notebooks.git"
      ref: "main"
      credentialsSecret: "git-credentials"  # Optional for private repos
    path: "notebooks/my-notebook.ipynb"
  
  podConfig:
    buildConfig:
      enabled: true
      strategy: "s2i"
      # Choose your preferred OpenShift AI image
      baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
      # Or use the Standard Data Science image:
      # baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-generic-data-science-notebook:2025.1"
      autoGenerateRequirements: false
      requirementsFile: "requirements.txt"
      timeout: "15m"
    
    # Fallback image if build is disabled
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
      limits:
        memory: "2Gi"
        cpu: "2000m"
    
    serviceAccountName: "notebook-validator-jupyter-notebook-validator-runner"
  
  timeout: "30m"
```

## Image Selection Guide

### When to Use Minimal Python

✅ **Use `s2i-minimal-notebook` when**:
- You have a custom `requirements.txt` with specific dependencies
- You want a lightweight base image
- You need full control over installed packages
- Your notebooks have minimal dependencies

### When to Use Standard Data Science

✅ **Use `s2i-generic-data-science-notebook` when**:
- You need common data science libraries (pandas, numpy, scikit-learn, etc.)
- You want faster build times (libraries pre-installed)
- You're working with typical ML/AI workloads
- You want a standardized environment

## Build Process

### How S2I Builds Work

1. **Base Image Pull**: Operator pulls the selected OpenShift AI image
2. **Source Clone**: Git repository is cloned into the build
3. **Dependency Installation**: S2I assemble script installs dependencies from `requirements.txt`
4. **Image Build**: Custom image is built with your notebooks and dependencies
5. **Image Push**: Built image is pushed to OpenShift ImageStream
6. **Validation**: Validation pod uses the built image to execute notebooks

### Build Logs

Monitor build progress:

```bash
# List builds
oc get builds -n <namespace>

# Follow build logs
oc logs -f <build-name>-build -n <namespace>
```

### Build Status

Check build status in the job:

```bash
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus}' | jq .
```

## Troubleshooting

### Build Stuck in "New" Status

**Symptom**: Build shows status "New" and never starts

**Solution**: Manually trigger the build:
```bash
oc start-build <buildconfig-name> -n <namespace>
```

### S2I Assemble Script Fails

**Symptom**: Build fails with `/usr/libexec/s2i/assemble: No such file or directory`

**Cause**: Using a non-S2I image

**Solution**: Use an OpenShift AI S2I-enabled image:
- `s2i-minimal-notebook`
- `s2i-generic-data-science-notebook`

### Image Pull Errors

**Symptom**: Cannot pull OpenShift AI images

**Cause**: OpenShift AI not installed or images not available

**Solution**: 
1. Verify OpenShift AI is installed:
   ```bash
   oc get namespace redhat-ods-applications
   ```
2. Check available ImageStreams:
   ```bash
   oc get imagestreams -n redhat-ods-applications
   ```

## Advanced Configuration

### Using Specific Image Tags

You can specify a specific tag instead of `latest`:

```yaml
baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2024.2"
```

### Custom Requirements File

Specify a custom requirements file location:

```yaml
buildConfig:
  enabled: true
  strategy: "s2i"
  baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
  requirementsFile: "custom/path/requirements.txt"
```

### Build Timeout

Adjust build timeout for large dependency installations:

```yaml
buildConfig:
  enabled: true
  strategy: "s2i"
  baseImage: "image-registry.openshift-image-registry.svc:5000/redhat-ods-applications/s2i-minimal-notebook:2025.1"
  timeout: "30m"  # Increase for large builds
```

## Benefits of OpenShift AI Integration

1. **✅ Optimized Images**: Pre-configured for Jupyter notebook workloads
2. **✅ S2I Support**: Built-in S2I scripts for seamless builds
3. **✅ Security**: Red Hat-maintained and security-scanned images
4. **✅ Consistency**: Standardized environments across teams
5. **✅ Performance**: Optimized for OpenShift infrastructure
6. **✅ Support**: Enterprise support from Red Hat

## Next Steps

- [Build Strategies Guide](BUILD-STRATEGIES.md) - Learn about S2I vs Tekton
- [Git Authentication](GIT-AUTHENTICATION.md) - Configure private repository access
- [Samples](../config/samples/) - Example configurations

## References

- [OpenShift AI Documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_ai_self-managed)
- [Source-to-Image (S2I)](https://docs.openshift.com/container-platform/latest/openshift_images/using_images/using-s21-images.html)
- [ImageStreams](https://docs.openshift.com/container-platform/latest/openshift_images/image-streams-manage.html)

