# AWS Storage Guide: EBS and EFS for Notebook Validation

This guide covers configuring AWS storage backends for notebook validation pods on ROSA and EKS.

## When to Use EBS vs EFS

| Feature | EBS (gp3-csi) | EFS (efs-sc) |
|---------|---------------|--------------|
| Access mode | ReadWriteOnce | ReadWriteMany |
| Use case | Single-pod builds and validation | Shared datasets, multi-pod access |
| Provisioning | Dynamic (WaitForFirstConsumer) | Dynamic or static |
| Performance | High IOPS, low latency | Throughput-optimized |
| Cost | Per-GB provisioned | Per-GB used (elastic) |
| AZ awareness | Single AZ (pod-collocated) | Multi-AZ |

### Choose EBS (gp3-csi) when

- Running a single build-then-validate pipeline per PVC
- Needing high IOPS for dependency installation (pip, conda)
- Budget-conscious (gp3 is cheaper per-GB than EFS)

### Choose EFS (efs-sc) when

- Multiple validation pods need to read the same dataset simultaneously
- Storing large model artifacts shared across jobs
- Running in a multi-AZ cluster where pods may schedule in different zones

## EBS Setup (gp3-csi)

### Prerequisites

On ROSA and EKS, the `gp3-csi` StorageClass is typically available by default:

```bash
kubectl get storageclass gp3-csi
```

### WaitForFirstConsumer Behavior

The `gp3-csi` StorageClass uses `volumeBindingMode: WaitForFirstConsumer`. This means the EBS volume is not provisioned until a pod claims it, and the volume is created in the same Availability Zone as the pod.

This is important for notebook validation because:
- The PVC remains `Pending` until the validation pod starts
- If you use node selectors or tolerations (ADR-054), the volume follows the pod's AZ

### Example

See [`config/samples/mlops_v1alpha1_notebookvalidationjob_aws_ebs_pvc.yaml`](../../config/samples/mlops_v1alpha1_notebookvalidationjob_aws_ebs_pvc.yaml).

## EFS Setup (efs-sc)

### Prerequisites

1. **Install the EFS CSI driver**:
   - **ROSA/OpenShift**: Install via OperatorHub (`AWS EFS CSI Driver Operator`)
   - **EKS**: Install as an EKS add-on or via Helm

2. **Create an EFS filesystem** in the same VPC as your cluster:
   ```bash
   aws efs create-file-system --performance-mode generalPurpose --throughput-mode elastic
   ```

3. **Create mount targets** in each subnet where your worker nodes run:
   ```bash
   aws efs create-mount-target \
     --file-system-id fs-0123456789abcdef0 \
     --subnet-id subnet-abc123 \
     --security-groups sg-xyz789
   ```

4. **Create the StorageClass**:
   ```yaml
   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: efs-sc
   provisioner: efs.csi.aws.com
   parameters:
     provisioningMode: efs-ap
     fileSystemId: fs-0123456789abcdef0
     directoryPerms: "700"
   ```

### ReadWriteMany for Multi-Pod Scenarios

EFS supports `ReadWriteMany`, enabling patterns like:

- **Shared test datasets**: Pre-load a large dataset once, validate multiple notebooks against it
- **Model artifact storage**: Store trained models in EFS, validate inference notebooks across teams
- **Concurrent Tekton builds** (ADR-040): Multiple builds can share intermediate artifacts

### Example

See [`config/samples/mlops_v1alpha1_notebookvalidationjob_aws_efs_pvc.yaml`](../../config/samples/mlops_v1alpha1_notebookvalidationjob_aws_efs_pvc.yaml).

## Related

- [ADR-040: Unique Build PVCs for Concurrent Tekton Builds](../adrs/040-unique-build-pvcs-for-concurrent-tekton-builds.md)
- [ADR-053: Volume and PVC Support for Validation Pods](../adrs/053-volume-and-pvc-support-for-validation-pods.md)
- [ADR-054: Pod Scheduling Support](../adrs/054-pod-scheduling-support-tolerations-nodeselector-affinity.md)
- [Notebook Credentials Guide](NOTEBOOK_CREDENTIALS_GUIDE.md) (for IRSA integration)
