# Contributing New Model Serving Platforms

This guide provides step-by-step technical instructions for adding support for new model serving platforms to the Jupyter Notebook Validator Operator.

## Overview

The operator uses a plugin-based architecture for model serving platforms. Adding a new platform requires:

1. **Platform Definition** - Register the platform in the detector
2. **CRD Detection** - Define which CRDs indicate the platform is installed
3. **Health Check Logic** - Implement model health checking
4. **Documentation** - Create user documentation and examples
5. **Tests** - Add unit and integration tests

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  NotebookValidationJob                       │
│  spec.modelValidation.platform: "your-platform"             │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Platform Detector                            │
│  pkg/platform/detector.go                                    │
│  - Detects platform via CRDs                                 │
│  - Validates platform availability                           │
│  - Checks model health                                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  Model Resolver                              │
│  pkg/platform/model_resolver.go                              │
│  - Parses model references                                   │
│  - Handles cross-namespace lookups                           │
└─────────────────────────────────────────────────────────────┘
```

## Step 1: Add Platform Definition

Edit `pkg/platform/detector.go` to add your platform to the `platformDefinitions` map:

```go
// In the init() function or as a package-level variable
var platformDefinitions = map[Platform]PlatformDefinition{
    // Existing platforms...
    
    // Add your platform
    PlatformYourPlatform: {
        Name:        "Your Platform Name",
        Description: "Brief description of the platform",
        CRDs: []string{
            "your-resource.your-api-group.io",  // Required CRD
        },
        APIGroup:    "your-api-group.io",
        APIVersion:  "v1",
        ResourceKind: "YourResource",
    },
}
```

### Platform Constant

Add a constant for your platform:

```go
const (
    // Existing platforms...
    PlatformYourPlatform Platform = "your-platform"
)
```

### Example: Adding Ollama Support

```go
const (
    PlatformOllama Platform = "ollama"
)

var platformDefinitions = map[Platform]PlatformDefinition{
    PlatformOllama: {
        Name:        "Ollama",
        Description: "Local LLM serving platform",
        CRDs:        []string{}, // Ollama doesn't use CRDs, deployed as Deployment
        APIGroup:    "apps",
        APIVersion:  "v1",
        ResourceKind: "Deployment",
    },
}
```

## Step 2: Implement Platform Detection

If your platform uses CRDs, the existing `checkCRDs()` method handles detection automatically. For platforms without CRDs (like Ollama), add custom detection logic:

```go
// In detector.go

func (d *Detector) detectYourPlatform(ctx context.Context) bool {
    // Option 1: Check for specific CRDs
    if d.checkCRDs(ctx, []string{"your-resource.your-api-group.io"}) {
        return true
    }
    
    // Option 2: Check for specific deployments/services
    // Use the Kubernetes client to query for known resources
    
    return false
}
```

## Step 3: Implement Model Health Checking

Add logic to check if models are healthy on your platform. Update the `CheckModelHealth()` method:

```go
func (d *Detector) CheckModelHealth(ctx context.Context, platform Platform, modelName, namespace string) (bool, string, error) {
    switch platform {
    // Existing cases...
    
    case PlatformYourPlatform:
        return d.checkYourPlatformModelHealth(ctx, modelName, namespace)
    
    default:
        return false, "", fmt.Errorf("unsupported platform: %s", platform)
    }
}

func (d *Detector) checkYourPlatformModelHealth(ctx context.Context, modelName, namespace string) (bool, string, error) {
    // Query your platform's API to check model health
    // Return: (isHealthy, statusMessage, error)
    
    // Example for a Deployment-based platform:
    deployment := &appsv1.Deployment{}
    err := d.client.Get(ctx, client.ObjectKey{
        Namespace: namespace,
        Name:      modelName,
    }, deployment)
    if err != nil {
        return false, "", err
    }
    
    if deployment.Status.ReadyReplicas > 0 {
        return true, "Model deployment is ready", nil
    }
    return false, "Model deployment not ready", nil
}
```

## Step 4: Implement Model Availability Check

Add logic to check if a model exists:

```go
func (d *Detector) CheckModelAvailability(ctx context.Context, platform Platform, modelName, namespace string) (bool, error) {
    switch platform {
    // Existing cases...
    
    case PlatformYourPlatform:
        return d.checkYourPlatformModelAvailability(ctx, modelName, namespace)
    
    default:
        return false, fmt.Errorf("unsupported platform: %s", platform)
    }
}

func (d *Detector) checkYourPlatformModelAvailability(ctx context.Context, modelName, namespace string) (bool, error) {
    // Check if the model resource exists
    // Return: (exists, error)
    
    // Example:
    resource := &unstructured.Unstructured{}
    resource.SetGroupVersionKind(schema.GroupVersionKind{
        Group:   "your-api-group.io",
        Version: "v1",
        Kind:    "YourModelResource",
    })
    
    err := d.client.Get(ctx, client.ObjectKey{
        Namespace: namespace,
        Name:      modelName,
    }, resource)
    
    if err != nil {
        if errors.IsNotFound(err) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}
```

## Step 5: Add Custom Platform Support

For platforms that don't have built-in support, users can use the `customPlatform` field. Ensure your platform works with this pattern:

```yaml
modelValidation:
  enabled: true
  platform: your-platform
  customPlatform:
    apiGroup: your-api-group.io
    resourceType: yourresources
    healthCheckEndpoint: "http://{{.ModelName}}.{{.Namespace}}:8080/health"
    predictionEndpoint: "http://{{.ModelName}}.{{.Namespace}}:8080/predict"
```

## Step 6: Add Unit Tests

Create tests in `pkg/platform/detector_test.go`:

```go
func TestYourPlatformDetection(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(*fake.Client)
        expected bool
    }{
        {
            name: "platform detected when CRDs present",
            setup: func(c *fake.Client) {
                // Setup mock CRDs
            },
            expected: true,
        },
        {
            name: "platform not detected when CRDs missing",
            setup: func(c *fake.Client) {
                // No setup
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := fake.NewClientBuilder().Build()
            tt.setup(client)
            
            detector := NewDetector(client, nil)
            result := detector.detectYourPlatform(context.Background())
            
            if result != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, result)
            }
        })
    }
}

func TestYourPlatformModelHealth(t *testing.T) {
    // Add tests for model health checking
}
```

## Step 7: Create Sample Manifests

Add example configurations in `config/samples/community/`:

### `model-validation-your-platform.yaml`

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-your-platform-notebook
  namespace: ml-team
spec:
  notebook:
    git:
      url: https://github.com/example/ml-notebooks.git
      ref: main
      path: inference/your-platform-example.ipynb
  
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: model-validator-sa
  
  modelValidation:
    enabled: true
    platform: your-platform
    phase: both
    targetModels:
      - my-model
    timeout: "5m"
```

### `your-platform-rbac.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: your-platform-model-validator
  namespace: ml-team
rules:
  - apiGroups: ["your-api-group.io"]
    resources: ["yourresources"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["your-api-group.io"]
    resources: ["yourresources/status"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: your-platform-model-validator
  namespace: ml-team
subjects:
  - kind: ServiceAccount
    name: model-validator-sa
    namespace: ml-team
roleRef:
  kind: Role
  name: your-platform-model-validator
  apiGroup: rbac.authorization.k8s.io
```

## Step 8: Create Documentation

Create `docs/community/your-platform.md`:

```markdown
# Your Platform Integration Guide

## Overview

This guide explains how to use the Jupyter Notebook Validator Operator with Your Platform.

## Prerequisites

- Kubernetes 1.25+
- Your Platform installed and configured
- At least one model deployed

## Installation

### Install Your Platform

```bash
# Platform-specific installation commands
kubectl apply -f https://your-platform.io/install.yaml
```

### Deploy a Test Model

```yaml
apiVersion: your-api-group.io/v1
kind: YourModelResource
metadata:
  name: test-model
  namespace: ml-team
spec:
  # Your platform-specific configuration
```

## Usage

### Basic Model Validation

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-my-notebook
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      path: inference.ipynb
  modelValidation:
    enabled: true
    platform: your-platform
    targetModels:
      - test-model
```

### With Prediction Validation

```yaml
modelValidation:
  enabled: true
  platform: your-platform
  targetModels:
    - test-model
  predictionValidation:
    enabled: true
    testData: '{"input": [1.0, 2.0, 3.0]}'
    expectedOutput: '{"prediction": [0.95]}'
    tolerance: "0.05"
```

## Troubleshooting

### Model Not Found

Ensure the model is deployed in the correct namespace:

```bash
kubectl get yourresources -n ml-team
```

### Health Check Failing

Check the model's status:

```bash
kubectl describe yourresource test-model -n ml-team
```

## API Reference

| Field | Description | Default |
|-------|-------------|---------|
| `platform` | Set to `your-platform` | - |
| `targetModels` | List of model names to validate | - |
| `phase` | Validation phase: `clean`, `existing`, `both` | `both` |
```

## Step 9: Update Platform List

Update `docs/COMMUNITY_PLATFORMS.md` to change your platform status from "HELP WANTED" to "Contributed":

```markdown
### Your Platform
- **Status**: ✅ Community Contributed
- **Contributor**: @your-github-username
- **Documentation**: `docs/community/your-platform.md`
- **Example**: `config/samples/community/model-validation-your-platform.yaml`
```

## Step 10: Submit Pull Request

### PR Checklist

- [ ] Platform definition added to `pkg/platform/detector.go`
- [ ] Model health check implemented
- [ ] Model availability check implemented
- [ ] Unit tests added
- [ ] Sample manifests created in `config/samples/community/`
- [ ] Documentation created in `docs/community/`
- [ ] Updated `docs/COMMUNITY_PLATFORMS.md`
- [ ] All existing tests pass (`make test`)
- [ ] Code passes linting (`make lint`)

### PR Title Format

```
feat(platform): Add support for Your Platform model serving
```

### PR Description Template

```markdown
## Summary

Adds model validation support for Your Platform.

## Changes

- Added platform definition for Your Platform
- Implemented model health checking via [API/method]
- Added documentation and examples

## Testing

- Unit tests: `go test ./pkg/platform/... -v -run TestYourPlatform`
- Tested with Your Platform version X.Y.Z on Kubernetes 1.28

## Documentation

- `docs/community/your-platform.md` - User guide
- `config/samples/community/model-validation-your-platform.yaml` - Example

Closes #XXX
```

## Tips for Success

### 1. Start Simple

Begin with basic platform detection and model availability. Add health checking and prediction validation later.

### 2. Use Existing Platforms as Reference

Study how KServe and OpenShift AI are implemented:
- `pkg/platform/detector.go` - Detection logic
- `internal/controller/model_validation_helper.go` - Controller integration

### 3. Test Locally First

```bash
# Run unit tests
go test ./pkg/platform/... -v

# Run full test suite
make test

# Check linting
make lint
```

### 4. Ask for Help

- Open a draft PR early for feedback
- Join discussions in GitHub Issues
- Tag maintainers for guidance

## Questions?

- **GitHub Issues**: [Open an issue](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new)
- **Discussions**: [Community Discussions](https://github.com/tosin2013/jupyter-notebook-validator-operator/discussions)

---

**Thank you for contributing to the Jupyter Notebook Validator Operator!** 🎉
