# Credentials Field Guide

## Overview

The `credentials` field in `PodConfigSpec` provides a simplified, user-friendly way to inject secrets as environment variables into validation pods. It's syntactic sugar that automatically converts to the more verbose `envFrom` syntax.

## Why Use `credentials`?

### Before (verbose `envFrom` syntax):
```yaml
podConfig:
  envFrom:
    - secretRef:
        name: "aws-credentials"
    - secretRef:
        name: "database-credentials"
    - secretRef:
        name: "mlflow-credentials"
```

### After (simple `credentials` syntax):
```yaml
podConfig:
  credentials:
    - "aws-credentials"
    - "database-credentials"
    - "mlflow-credentials"
```

## How It Works

The operator automatically converts each credential name in the `credentials` list to an `envFrom` entry with a `secretRef`. This happens transparently during pod creation.

**Conversion Example:**
```yaml
credentials: ["aws-credentials"]
```

Is automatically converted to:
```yaml
envFrom:
  - secretRef:
      name: "aws-credentials"
```

## Usage Examples

### Example 1: Single Credential
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: aws-test
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/aws-test.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    credentials:
      - "aws-credentials"
```

### Example 2: Multiple Credentials
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: multi-service-test
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/integration-test.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
    credentials:
      - "aws-credentials"
      - "database-credentials"
      - "mlflow-credentials"
      - "api-keys"
```

### Example 3: Mixing `credentials` and `envFrom`

You can use both `credentials` (for simple secrets) and `envFrom` (for complex sources like ConfigMaps):

```yaml
podConfig:
  credentials:
    - "aws-credentials"
    - "database-credentials"
  envFrom:
    - configMapRef:
        name: "app-config"
  env:
    - name: "CUSTOM_VAR"
      value: "custom-value"
```

## Secret Format

The secrets referenced in the `credentials` field should contain key-value pairs that will be injected as environment variables:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: "AKIAIOSFODNN7EXAMPLE"
  AWS_SECRET_ACCESS_KEY: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  AWS_DEFAULT_REGION: "us-east-1"
```

All keys in the secret will be available as environment variables in the validation pod.

## Benefits

1. **Simplicity**: Reduces YAML verbosity for common use cases
2. **Readability**: Makes it immediately clear what credentials are being used
3. **Backwards Compatible**: Existing `envFrom` syntax continues to work
4. **Flexible**: Can be mixed with `envFrom` and `env` for complex scenarios

## When to Use Each Syntax

### Use `credentials` when:
- You're injecting secrets as environment variables
- You want simple, readable configuration
- All your credential sources are Kubernetes Secrets

### Use `envFrom` when:
- You need to inject ConfigMaps
- You need fine-grained control over secret injection
- You're using advanced features like `prefix` or `optional`

### Use `env` when:
- You need to inject individual keys from secrets/configmaps
- You need to set static values
- You need to use field references or resource field references

## Implementation Details

The conversion happens in the `createValidationPod` function in the controller:

1. The controller first processes explicit `envFrom` entries
2. Then it processes `credentials` entries, converting each to a `secretRef`
3. Both are combined into the pod's `envFrom` field
4. This ensures both syntaxes work together seamlessly

## See Also

- [Notebook Credentials Guide](NOTEBOOK_CREDENTIALS_GUIDE.md) - Comprehensive guide to credential management
- [Sample Manifests](../config/samples/) - Example configurations
- [API Reference](../api/v1alpha1/notebookvalidationjob_types.go) - Full API documentation

