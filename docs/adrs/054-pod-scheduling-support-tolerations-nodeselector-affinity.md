# ADR-054: Pod Scheduling Support (Tolerations, NodeSelector, Affinity)

**Date**: 2026-01-28
**Status**: Implemented
**Deciders**: Development Team, User Community
**Technical Story**: Enable scheduling validation pods on specialized nodes (GPU, high-memory, spot instances)
**GitHub Issue**: [#13](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/13)

## Context and Problem Statement

Users running notebook validation on clusters with specialized nodes cannot schedule validation pods on those nodes because the CRD doesn't support Kubernetes pod scheduling fields. This is particularly problematic for:

1. **GPU Nodes**: ML training notebooks requiring GPU resources on tainted GPU nodes (e.g., `nvidia.com/gpu=True:NoSchedule`)
2. **High-Memory Nodes**: Data processing notebooks requiring dedicated high-memory nodes
3. **Spot/Preemptible Instances**: Cost-optimized workloads that can tolerate spot instance eviction
4. **Multi-Tenant Clusters**: Team-specific node pools with taints and labels

### Current State

The `PodConfigSpec` supports:
- `containerImage`, `env`, `envFrom`
- `resources` (CPU, memory, GPU requests/limits)
- `serviceAccountName`
- `volumeMounts`, `volumes` (ADR-053)
- `buildConfig`, `credentials`

**Missing**: Pod scheduling fields (`tolerations`, `nodeSelector`, `affinity`)

### User Story

> "When running notebook validation on clusters with specialized nodes that have taints (e.g., GPU nodes with `nvidia.com/gpu=True:NoSchedule`), validation pods cannot be scheduled because the CRD doesn't support specifying tolerations."

## Decision Drivers

- **GPU Workload Support**: ML notebooks requiring NVIDIA GPUs on tainted nodes
- **Enterprise Cluster Compatibility**: Multi-tenant clusters with team-specific node pools
- **Cost Optimization**: Scheduling on spot/preemptible instances
- **Kubernetes Native**: Use standard Kubernetes scheduling primitives
- **Parity with Industry Standards**: Similar operators (KServe, Kubeflow) support these fields
- **Backward Compatibility**: Existing NotebookValidationJobs must continue to work

## Considered Options

### Option 1: Tolerations Only

Add only `tolerations` field for basic taint toleration:

```yaml
spec:
  podConfig:
    tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
```

**Pros**:
- Simplest implementation
- Covers GPU node scheduling use case

**Cons**:
- No node targeting (only tolerates, doesn't prefer)
- No pod affinity/anti-affinity for spreading workloads
- Incomplete solution

### Option 2: Full Scheduling Support ✅ **Selected**

Add all three scheduling fields: `tolerations`, `nodeSelector`, `affinity`

```yaml
spec:
  podConfig:
    tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
    nodeSelector:
      nvidia.com/gpu.present: "true"
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
            - matchExpressions:
                - key: nvidia.com/gpu.present
                  operator: In
                  values: ["true"]
```

**Pros**:
- Complete Kubernetes scheduling support
- Enables all use cases (GPU, high-memory, spot, multi-tenant)
- Future-proof
- Industry standard

**Cons**:
- More complex API
- More types to maintain
- Higher implementation effort

### Option 3: Simplified Scheduling Profiles

Add a `schedulingProfile` enum with predefined configurations:

```yaml
spec:
  podConfig:
    schedulingProfile: gpu  # or: high-memory, spot, default
```

**Pros**:
- Very simple user experience
- No Kubernetes knowledge required

**Cons**:
- Not flexible enough for enterprise requirements
- Hardcoded assumptions about taint keys/values
- Non-standard pattern

## Decision Outcome

**Chosen option**: **Option 2 - Full Scheduling Support**

This provides complete flexibility while maintaining Kubernetes-native patterns that users are familiar with. Similar operators (KServe InferenceService, Kubeflow Notebooks) use the same approach.

## API Design

### New Types in `api/v1alpha1/notebookvalidationjob_types.go`

```go
// PodConfigSpec additions
type PodConfigSpec struct {
    // ... existing fields ...

    // Tolerations allow the validation pod to be scheduled onto nodes with matching taints
    // +optional
    Tolerations []Toleration `json:"tolerations,omitempty"`

    // NodeSelector is a map of {key,value} pairs for selecting nodes
    // +optional
    NodeSelector map[string]string `json:"nodeSelector,omitempty"`

    // Affinity specifies advanced scheduling rules including node affinity,
    // pod affinity, and pod anti-affinity
    // +optional
    Affinity *Affinity `json:"affinity,omitempty"`
}

// Toleration allows the pod to be scheduled onto nodes with matching taints
type Toleration struct {
    // Key is the taint key that the toleration applies to
    // +optional
    Key string `json:"key,omitempty"`

    // Operator represents a key's relationship to the value
    // +kubebuilder:validation:Enum=Exists;Equal
    // +kubebuilder:default="Equal"
    // +optional
    Operator string `json:"operator,omitempty"`

    // Value is the taint value the toleration matches to
    // +optional
    Value string `json:"value,omitempty"`

    // Effect indicates the taint effect to match
    // +kubebuilder:validation:Enum="";NoSchedule;PreferNoSchedule;NoExecute
    // +optional
    Effect string `json:"effect,omitempty"`

    // TolerationSeconds represents the period of time the toleration tolerates the taint
    // +optional
    TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// Affinity is a group of affinity scheduling rules
type Affinity struct {
    // NodeAffinity describes node affinity scheduling rules for the pod
    // +optional
    NodeAffinity *NodeAffinity `json:"nodeAffinity,omitempty"`

    // PodAffinity describes pod affinity scheduling rules
    // +optional
    PodAffinity *PodAffinity `json:"podAffinity,omitempty"`

    // PodAntiAffinity describes pod anti-affinity scheduling rules
    // +optional
    PodAntiAffinity *PodAntiAffinity `json:"podAntiAffinity,omitempty"`
}

// NodeAffinity defines node affinity scheduling rules
type NodeAffinity struct {
    // RequiredDuringSchedulingIgnoredDuringExecution specifies hard node constraints
    // +optional
    RequiredDuringSchedulingIgnoredDuringExecution *NodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

    // PreferredDuringSchedulingIgnoredDuringExecution specifies soft node preferences
    // +optional
    PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// ... additional types for NodeSelector, NodeSelectorTerm, NodeSelectorRequirement,
// PreferredSchedulingTerm, PodAffinity, PodAntiAffinity, PodAffinityTerm,
// WeightedPodAffinityTerm, LabelSelector, LabelSelectorRequirement
```

### Example Usage

#### Use Case 1: GPU Training Notebook

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: gpu-training-validation
spec:
  notebook:
    git:
      url: https://github.com/example/ml-notebooks.git
      ref: main
    path: notebooks/gpu-training.ipynb
  podConfig:
    containerImage: quay.io/jupyter/pytorch-notebook:cuda-latest
    resources:
      limits:
        nvidia.com/gpu: "1"
        memory: "16Gi"
    # Tolerate GPU node taints
    tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
    # Target GPU nodes
    nodeSelector:
      nvidia.com/gpu.present: "true"
  timeout: "2h"
```

#### Use Case 2: Spot Instance Scheduling

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: spot-instance-validation
spec:
  notebook:
    git:
      url: https://github.com/example/batch-notebooks.git
      ref: main
    path: notebooks/batch-processing.ipynb
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
    # Tolerate spot instance taints (AWS, Azure, GCP)
    tolerations:
      - key: kubernetes.io/spot
        operator: Exists
        effect: NoSchedule
      - key: kubernetes.azure.com/scalesetpriority
        operator: Equal
        value: "spot"
        effect: NoSchedule
      - key: cloud.google.com/gke-preemptible
        operator: Equal
        value: "true"
        effect: NoSchedule
    # Prefer spot nodes for cost savings
    affinity:
      nodeAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            preference:
              matchExpressions:
                - key: kubernetes.io/lifecycle
                  operator: In
                  values: ["spot", "preemptible"]
  timeout: "30m"
```

#### Use Case 3: Multi-Tenant Cluster with Pod Anti-Affinity

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: team-ml-validation
  namespace: team-ml
spec:
  notebook:
    git:
      url: https://github.com/team-ml/notebooks.git
      ref: main
    path: notebooks/model-evaluation.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    # Tolerate team-specific node taints
    tolerations:
      - key: team
        operator: Equal
        value: "ml"
        effect: NoSchedule
    # Target team-specific node pool
    nodeSelector:
      team: ml
      environment: production
    # Spread validation jobs across nodes
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: jupyter-notebook-validator
              topologyKey: kubernetes.io/hostname
  timeout: "45m"
```

## Implementation

### Controller Changes

The controller's `createValidationPod` function was updated to apply scheduling fields:

```go
// internal/controller/notebookvalidationjob_controller.go

// GitHub Issue #13: Pod Scheduling Support
// Add tolerations for scheduling on nodes with taints (e.g., GPU nodes)
if len(job.Spec.PodConfig.Tolerations) > 0 {
    logger.Info("Adding tolerations", "tolerationCount", len(job.Spec.PodConfig.Tolerations))
    pod.Spec.Tolerations = convertTolerations(job.Spec.PodConfig.Tolerations)
}

// Add nodeSelector for targeting specific node labels
if len(job.Spec.PodConfig.NodeSelector) > 0 {
    logger.Info("Adding nodeSelector", "nodeSelector", job.Spec.PodConfig.NodeSelector)
    pod.Spec.NodeSelector = job.Spec.PodConfig.NodeSelector
}

// Add affinity for advanced scheduling requirements
if job.Spec.PodConfig.Affinity != nil {
    logger.Info("Adding affinity rules")
    pod.Spec.Affinity = convertAffinity(job.Spec.PodConfig.Affinity)
}
```

### Conversion Functions

Helper functions convert custom types to Kubernetes types:

```go
// internal/controller/papermill_helper.go

func convertTolerations(customTolerations []mlopsv1alpha1.Toleration) []corev1.Toleration
func convertAffinity(customAffinity *mlopsv1alpha1.Affinity) *corev1.Affinity
func convertNodeAffinity(customNodeAffinity *mlopsv1alpha1.NodeAffinity) *corev1.NodeAffinity
func convertPodAffinity(customPodAffinity *mlopsv1alpha1.PodAffinity) *corev1.PodAffinity
func convertPodAntiAffinity(customPodAntiAffinity *mlopsv1alpha1.PodAntiAffinity) *corev1.PodAntiAffinity
// ... and supporting functions
```

### Unit Tests

Comprehensive unit tests were added:

- `TestConvertTolerations`: 7 test cases covering nil input, Exists/Equal operators, NoSchedule/NoExecute/PreferNoSchedule effects, TolerationSeconds
- `TestConvertAffinity`: 6 test cases covering nil input, node affinity (required/preferred), pod affinity, pod anti-affinity, combined affinities

## Consequences

### Positive

1. **GPU Workload Support**: ML notebooks can now run on tainted GPU nodes
2. **Enterprise Ready**: Full support for multi-tenant cluster configurations
3. **Cost Optimization**: Enables spot/preemptible instance scheduling
4. **Kubernetes Native**: Uses standard scheduling primitives
5. **Industry Parity**: Matches KServe, Kubeflow, and other ML operators
6. **Backward Compatible**: Existing jobs without scheduling fields continue to work

### Negative

1. **API Complexity**: Additional types to understand and maintain
2. **Documentation Burden**: More fields to document
3. **User Learning Curve**: Kubernetes scheduling knowledge required

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Invalid toleration causes pod stuck in Pending | Validation job fails | Add samples and documentation with common patterns |
| Conflicting affinity rules | Pod unschedulable | Document best practices for combining rules |
| nodeSelector too restrictive | No matching nodes | Warn users to verify nodes exist with required labels |
| TolerationSeconds misuse | Premature pod eviction | Document NoExecute effect and TolerationSeconds interaction |

## Security Considerations

1. **No Elevated Privileges**: Scheduling fields don't grant additional permissions
2. **Namespace Scoped**: Pod scheduling is limited to namespace boundaries
3. **RBAC Unchanged**: No additional RBAC permissions required for scheduling
4. **Taint Bypass**: Users can only schedule on nodes they have access to via tolerations

## Files Changed

1. `api/v1alpha1/notebookvalidationjob_types.go` - Added Toleration, Affinity, and supporting types
2. `api/v1alpha1/zz_generated.deepcopy.go` - Generated DeepCopy functions
3. `internal/controller/notebookvalidationjob_controller.go` - Apply scheduling fields to pod spec
4. `internal/controller/papermill_helper.go` - Added conversion functions
5. `internal/controller/papermill_helper_test.go` - Added unit tests
6. `config/crd/bases/mlops.mlops.dev_notebookvalidationjobs.yaml` - Updated CRD
7. `helm/jupyter-notebook-validator-operator/crds/` - Updated Helm CRD
8. `config/samples/mlops_v1alpha1_notebookvalidationjob_gpu_scheduling.yaml` - Sample manifests
9. `README.md` - Updated documentation

## Backport Status

This feature was backported to all supported release branches:
- `main` ✅
- `release-4.20` ✅
- `release-4.19` ✅
- `release-4.18` ✅

## References

- [GitHub Issue #13](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/13)
- [Kubernetes Taints and Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- [Kubernetes Node Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity)
- [Kubernetes Pod Affinity/Anti-Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity)
- [NVIDIA GPU Operator Taints](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/gpu-operator-rdma.html)
- [KServe InferenceService Scheduling](https://kserve.github.io/website/latest/modelserving/nodescheduling/inferenceservicenodescheduling/)
- **Related ADRs**:
  - ADR-053: Volume and PVC Support for Validation Pods
  - ADR-005: OpenShift Compatibility
