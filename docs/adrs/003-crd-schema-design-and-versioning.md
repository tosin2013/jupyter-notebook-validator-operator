# ADR 003: CRD Schema Design and Versioning

## Status
Accepted

## Context

The NotebookValidationJob Custom Resource Definition (CRD) is the primary API contract between users and the Jupyter Notebook Validator Operator. The CRD design must:

1. Provide a clear, intuitive API for defining notebook validation jobs
2. Support evolution and backward compatibility as requirements change
3. Enable strong validation to prevent user errors
4. Align with Kubernetes API conventions and best practices
5. Support the validation workflow defined in the PRD

### Current Requirements (from PRD)

The CRD must support:
- **Notebook Source**: Git repository URL, ref (branch/tag/commit), path, and credentials
- **Golden Notebook**: Optional reference notebook for output comparison
- **Pod Configuration**: Container image, service account, resource requests/limits
- **Validation Rules**: Metadata validation, output comparison tolerance
- **Status Reporting**: Cell-by-cell results, overall status, error messages

### Technical Considerations

1. **API Versioning**: CRDs support multiple versions (v1alpha1, v1beta1, v1) with conversion
2. **Schema Validation**: OpenAPI v3 schema enables server-side validation
3. **Structural Schema**: Required for CRD v1, ensures consistent structure
4. **Defaulting**: Server-side defaulting reduces user burden
5. **Status Subresource**: Separates spec (desired state) from status (observed state)

### Available Options

#### Option 1: Single Version (v1alpha1) with No Conversion
- **Pros**: Simplest to implement, no conversion webhook needed
- **Cons**: Breaking changes require new CRD, no upgrade path for users

#### Option 2: Multi-Version with Conversion Webhooks
- **Pros**: Smooth upgrades, backward compatibility, follows K8s best practices
- **Cons**: Requires webhook infrastructure, more complex implementation

#### Option 3: Versioned CRDs (Separate CRDs per Version)
- **Pros**: Complete isolation between versions
- **Cons**: Users must migrate manually, no automatic conversion, operational complexity

## Decision

We will implement **Multi-Version CRD with Conversion Webhooks** using the following strategy:

### API Versioning Strategy
- **Initial Version**: `v1alpha1` (experimental, may have breaking changes)
- **Stable Version**: `v1beta1` (stable API, backward compatible changes only)
- **Production Version**: `v1` (fully stable, long-term support)

### CRD Configuration
- **API Group**: `mlops.dev`
- **Kind**: `NotebookValidationJob`
- **Versions**: Start with `v1alpha1`, add versions as API stabilizes
- **Storage Version**: Latest stable version (initially `v1alpha1`)
- **Served Versions**: All versions with conversion support

### Schema Design Principles
1. **Structural Schema**: Use OpenAPI v3 schema for all fields
2. **Required Fields**: Minimize required fields, use defaults where possible
3. **Validation**: Leverage CEL (Common Expression Language) for complex validation
4. **Immutability**: Mark fields as immutable where appropriate (e.g., notebook source)
5. **Status Subresource**: Separate spec and status for proper reconciliation

## Consequences

### Positive
- **Smooth Upgrades**: Users can upgrade operator without changing CRs
- **Backward Compatibility**: Old CRs continue to work with new operator versions
- **Strong Validation**: Server-side validation catches errors before reconciliation
- **API Evolution**: Can add features without breaking existing users
- **Kubernetes Native**: Follows K8s API conventions and best practices

### Negative
- **Webhook Infrastructure**: Requires webhook server and TLS certificates
- **Conversion Logic**: Must implement and test conversion between versions
- **Complexity**: More moving parts than single-version approach
- **Testing Burden**: Must test all version combinations

### Neutral
- **Version Lifecycle**: Must manage version deprecation and removal
- **Documentation**: Must document all versions and migration paths

## Implementation Notes

### CRD Structure (v1alpha1)

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: notebookvalidationjobs.mlops.dev
spec:
  group: mlops.dev
  names:
    kind: NotebookValidationJob
    listKind: NotebookValidationJobList
    plural: notebookvalidationjobs
    singular: notebookvalidationjob
    shortNames:
      - nvj
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required:
                - notebook
              properties:
                notebook:
                  type: object
                  required:
                    - git
                  properties:
                    git:
                      type: object
                      required:
                        - url
                        - path
                      properties:
                        url:
                          type: string
                          pattern: '^https?://.*'
                        ref:
                          type: string
                          default: "main"
                        path:
                          type: string
                        credentialsSecret:
                          type: string
                    golden:
                      type: object
                      properties:
                        git:
                          type: object
                          # Same structure as notebook.git
                podConfig:
                  type: object
                  properties:
                    containerImage:
                      type: string
                      default: "quay.io/jupyter/datascience-notebook:latest"
                    serviceAccountName:
                      type: string
                      default: "default"
                    resources:
                      type: object
                      properties:
                        requests:
                          type: object
                          properties:
                            cpu:
                              type: string
                              default: "500m"
                            memory:
                              type: string
                              default: "2Gi"
                        limits:
                          type: object
                          properties:
                            cpu:
                              type: string
                              default: "1"
                            memory:
                              type: string
                              default: "4Gi"
                timeoutSeconds:
                  type: integer
                  default: 3600
                  minimum: 60
                  maximum: 86400
            status:
              type: object
              properties:
                phase:
                  type: string
                  enum:
                    - Pending
                    - Running
                    - Succeeded
                    - Failed
                conditions:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                      status:
                        type: string
                      lastTransitionTime:
                        type: string
                        format: date-time
                      reason:
                        type: string
                      message:
                        type: string
                results:
                  type: array
                  items:
                    type: object
                    properties:
                      cellIndex:
                        type: integer
                      status:
                        type: string
                        enum:
                          - Success
                          - Failure
                      output:
                        type: string
                      error:
                        type: string
      subresources:
        status: {}
      additionalPrinterColumns:
        - name: Phase
          type: string
          jsonPath: .status.phase
        - name: Age
          type: date
          jsonPath: .metadata.creationTimestamp
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        service:
          name: jupyter-notebook-validator-webhook
          namespace: jupyter-notebook-validator-system
          path: /convert
      conversionReviewVersions:
        - v1
        - v1beta1
```

### Go Types (v1alpha1)

```go
// api/v1alpha1/notebookvalidationjob_types.go
package v1alpha1

import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NotebookValidationJobSpec defines the desired state
type NotebookValidationJobSpec struct {
    // Notebook defines the notebook to validate
    // +kubebuilder:validation:Required
    Notebook NotebookSource `json:"notebook"`

    // PodConfig defines the pod configuration for validation
    // +optional
    PodConfig *PodConfig `json:"podConfig,omitempty"`

    // TimeoutSeconds defines the maximum execution time
    // +kubebuilder:default:=3600
    // +kubebuilder:validation:Minimum:=60
    // +kubebuilder:validation:Maximum:=86400
    // +optional
    TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// NotebookSource defines where to fetch the notebook
type NotebookSource struct {
    // Git defines the Git repository source
    // +kubebuilder:validation:Required
    Git GitSource `json:"git"`

    // Golden defines an optional golden notebook for comparison
    // +optional
    Golden *GoldenNotebook `json:"golden,omitempty"`
}

// GitSource defines a Git repository source
type GitSource struct {
    // URL is the Git repository URL
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern:=`^https?://.*`
    URL string `json:"url"`

    // Ref is the branch, tag, or commit
    // +kubebuilder:default:="main"
    // +optional
    Ref string `json:"ref,omitempty"`

    // Path is the path to the notebook file
    // +kubebuilder:validation:Required
    Path string `json:"path"`

    // CredentialsSecret is the name of the secret containing Git credentials
    // +optional
    CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

// PodConfig defines pod configuration
type PodConfig struct {
    // ContainerImage is the container image to use
    // +kubebuilder:default:="quay.io/jupyter/datascience-notebook:latest"
    // +optional
    ContainerImage string `json:"containerImage,omitempty"`

    // ServiceAccountName is the service account to use
    // +kubebuilder:default:="default"
    // +optional
    ServiceAccountName string `json:"serviceAccountName,omitempty"`

    // Resources defines resource requirements
    // +optional
    Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// NotebookValidationJobStatus defines the observed state
type NotebookValidationJobStatus struct {
    // Phase represents the current phase
    // +optional
    Phase ValidationPhase `json:"phase,omitempty"`

    // Conditions represent the latest observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Results contains cell-by-cell validation results
    // +optional
    Results []CellResult `json:"results,omitempty"`
}

// ValidationPhase represents the validation phase
// +kubebuilder:validation:Enum:=Pending;Running;Succeeded;Failed
type ValidationPhase string

const (
    PhasePending   ValidationPhase = "Pending"
    PhaseRunning   ValidationPhase = "Running"
    PhaseSucceeded ValidationPhase = "Succeeded"
    PhaseFailed    ValidationPhase = "Failed"
)

// CellResult represents the result of validating a single cell
type CellResult struct {
    // CellIndex is the index of the cell
    CellIndex int `json:"cellIndex"`

    // Status is the validation status
    Status CellStatus `json:"status"`

    // Output is the cell output
    // +optional
    Output string `json:"output,omitempty"`

    // Error is the error message if validation failed
    // +optional
    Error string `json:"error,omitempty"`
}

// CellStatus represents cell validation status
// +kubebuilder:validation:Enum:=Success;Failure
type CellStatus string

const (
    CellSuccess CellStatus = "Success"
    CellFailure CellStatus = "Failure"
)
```

### Conversion Webhook

```go
// api/v1alpha1/notebookvalidationjob_conversion.go
package v1alpha1

import (
    "sigs.k8s.io/controller-runtime/pkg/conversion"
)

// Hub marks this version as a conversion hub
func (*NotebookValidationJob) Hub() {}

// ConvertTo converts this version to the Hub version
func (src *NotebookValidationJob) ConvertTo(dstRaw conversion.Hub) error {
    // Implement conversion logic when adding new versions
    return nil
}

// ConvertFrom converts from the Hub version to this version
func (dst *NotebookValidationJob) ConvertFrom(srcRaw conversion.Hub) error {
    // Implement conversion logic when adding new versions
    return nil
}
```

### Version Migration Path

1. **v1alpha1 → v1beta1**: Stabilize API, add validation rules
2. **v1beta1 → v1**: Production-ready, long-term support
3. **Deprecation**: Announce v1alpha1 deprecation when v1beta1 is released
4. **Removal**: Remove v1alpha1 support 6 months after v1 release

## References

- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [Kubebuilder CRD Generation](https://book.kubebuilder.io/reference/generating-crd.html)
- [OpenAPI v3 Schema](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation)

## Related ADRs

- ADR 001: Operator Framework and SDK Version
- ADR 002: Platform Version Support Strategy
- ADR 005: RBAC & Service Account Model

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial decision |

