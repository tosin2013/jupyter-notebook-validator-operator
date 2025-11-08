# ADR 005: RBAC and Service Account Model

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator requires specific permissions to function correctly within a Kubernetes/OpenShift cluster. As an operator that creates and manages pods, watches custom resources, and updates status, we must carefully design the RBAC (Role-Based Access Control) model to:

1. Follow the principle of least privilege
2. Enable all required operator functionality
3. Prevent unauthorized access to cluster resources
4. Support both namespace-scoped and cluster-scoped deployments
5. Align with security best practices and compliance requirements

### Current Requirements

The operator must be able to:
- **Watch and manage** NotebookValidationJob custom resources
- **Create and manage** pods for notebook execution
- **Read** ConfigMaps and Secrets (for Git credentials, configuration)
- **Update** status subresources of NotebookValidationJob CRs
- **List and watch** pods to monitor validation job progress
- **Delete** completed validation pods (cleanup)

### Security Considerations

1. **Namespace Isolation**: Users should be able to run the operator in their namespace without cluster-admin privileges
2. **Secret Access**: Operator needs read access to secrets but should not have write access
3. **Pod Security**: Validation pods should run with minimal privileges
4. **Status Updates**: Only the operator should update CR status, not end users
5. **Audit Trail**: All operator actions should be auditable via Kubernetes RBAC

### Available Options

#### Option 1: Cluster-Admin Privileges
- **Pros**: Simplest to implement, no permission issues
- **Cons**: Violates least privilege, security risk, not acceptable for production

#### Option 2: Namespace-Scoped Roles Only
- **Pros**: Maximum isolation, users can install in their namespace
- **Cons**: Cannot watch CRDs cluster-wide, limits multi-namespace support

#### Option 3: Cluster-Scoped Roles with Minimal Permissions
- **Pros**: Supports all deployment modes, follows least privilege
- **Cons**: Requires cluster-admin to install, more complex RBAC setup

#### Option 4: Hybrid Model (Cluster + Namespace Roles)
- **Pros**: Flexible deployment, supports both scoped and cluster-wide modes
- **Cons**: Most complex to implement and document

## Decision

We will implement a **Hybrid RBAC Model** with the following components:

### Service Accounts

#### 1. Operator Service Account
- **Name**: `jupyter-notebook-validator-operator`
- **Purpose**: Run the operator controller
- **Scope**: Cluster-wide or namespace-scoped (configurable)

#### 2. Validation Pod Service Account
- **Name**: `jupyter-notebook-validator-runner`
- **Purpose**: Run validation pods
- **Scope**: Namespace-scoped
- **Permissions**: Minimal (read-only access to ConfigMaps/Secrets)

### RBAC Roles

#### ClusterRole: Operator Core Permissions
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: jupyter-notebook-validator-operator
rules:
  # CRD management
  - apiGroups: ["mlops.dev"]
    resources: ["notebookvalidationjobs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # Status subresource (separate permission)
  - apiGroups: ["mlops.dev"]
    resources: ["notebookvalidationjobs/status"]
    verbs: ["get", "update", "patch"]
  
  # Pod management for validation
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # Pod logs for result collection
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
  
  # ConfigMap access for configuration
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
  
  # Secret access for Git credentials (read-only)
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
  
  # Events for status reporting
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]
```

#### Role: Namespace-Scoped Operator Permissions
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-validator-system
rules:
  # Same as ClusterRole but namespace-scoped
  # Used when operator runs in single-namespace mode
  - apiGroups: ["mlops.dev"]
    resources: ["notebookvalidationjobs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  - apiGroups: ["mlops.dev"]
    resources: ["notebookvalidationjobs/status"]
    verbs: ["get", "update", "patch"]
  
  - apiGroups: [""]
    resources: ["pods", "configmaps", "secrets", "events"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
```

#### ClusterRole: Validation Pod Permissions
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: jupyter-notebook-validator-runner
rules:
  # Minimal permissions for validation pods
  # Read-only access to ConfigMaps for configuration
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get"]
  
  # Read-only access to Secrets for Git credentials
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
```

### Role Bindings

#### ClusterRoleBinding: Operator (Cluster-Wide Mode)
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: jupyter-notebook-validator-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: jupyter-notebook-validator-operator
subjects:
  - kind: ServiceAccount
    name: jupyter-notebook-validator-operator
    namespace: jupyter-validator-system
```

#### RoleBinding: Operator (Namespace-Scoped Mode)
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jupyter-notebook-validator-operator
  namespace: jupyter-validator-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: jupyter-notebook-validator-operator
subjects:
  - kind: ServiceAccount
    name: jupyter-notebook-validator-operator
    namespace: jupyter-validator-system
```

#### RoleBinding: Validation Pods
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jupyter-notebook-validator-runner
  namespace: jupyter-validator-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: jupyter-notebook-validator-runner
subjects:
  - kind: ServiceAccount
    name: jupyter-notebook-validator-runner
    namespace: jupyter-validator-system
```

## Consequences

### Positive
- **Least Privilege**: Operator has only the permissions it needs
- **Flexible Deployment**: Supports both cluster-wide and namespace-scoped modes
- **Security Isolation**: Validation pods run with minimal permissions
- **Audit Trail**: All actions are logged via Kubernetes RBAC
- **Status Protection**: Status subresource prevents user tampering
- **Secret Safety**: Read-only access to secrets, no write permissions

### Negative
- **Installation Complexity**: Requires cluster-admin to create ClusterRole/ClusterRoleBinding
- **Namespace Proliferation**: Each namespace needs RoleBinding for validation pods
- **Permission Management**: Must update RBAC when adding new features
- **Documentation Burden**: Must document both deployment modes

### Neutral
- **Service Account Separation**: Operator and validation pods use different accounts
- **OpenShift Compatibility**: Works with OpenShift's SCC (Security Context Constraints)

## Implementation Notes

### Deployment Modes

#### Mode 1: Cluster-Wide (Recommended for Platform Teams)
```yaml
# Operator watches all namespaces
env:
  - name: WATCH_NAMESPACE
    value: ""  # Empty = all namespaces
```

#### Mode 2: Namespace-Scoped (Recommended for Self-Service)
```yaml
# Operator watches single namespace
env:
  - name: WATCH_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
```

#### Mode 3: Multi-Namespace (Advanced)
```yaml
# Operator watches specific namespaces
env:
  - name: WATCH_NAMESPACE
    value: "namespace1,namespace2,namespace3"
```

### Pod Security Context

```yaml
# Validation pod security context
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop:
      - ALL
```

### OpenShift Security Context Constraints (SCC)

```yaml
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: jupyter-notebook-validator-scc
allowPrivilegedContainer: false
allowHostDirVolumePlugin: false
allowHostNetwork: false
allowHostPorts: false
allowHostPID: false
allowHostIPC: false
readOnlyRootFilesystem: false
runAsUser:
  type: MustRunAsRange
  uidRangeMin: 1000
  uidRangeMax: 65535
seLinuxContext:
  type: MustRunAs
fsGroup:
  type: MustRunAs
supplementalGroups:
  type: RunAsAny
volumes:
  - configMap
  - downwardAPI
  - emptyDir
  - persistentVolumeClaim
  - projected
  - secret
users:
  - system:serviceaccount:jupyter-validator-system:jupyter-notebook-validator-runner
```

### RBAC Validation

```bash
# Check operator permissions
kubectl auth can-i create pods \
  --as=system:serviceaccount:jupyter-validator-system:jupyter-notebook-validator-operator

# Check validation pod permissions
kubectl auth can-i get secrets \
  --as=system:serviceaccount:jupyter-validator-system:jupyter-notebook-validator-runner

# Audit operator actions
kubectl get events --field-selector involvedObject.kind=NotebookValidationJob
```

### Permission Escalation Prevention

```go
// controllers/notebookvalidationjob_controller.go
func (r *NotebookValidationJobReconciler) createValidationPod(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) error {
    pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-validator", job.Name),
            Namespace: job.Namespace,
        },
        Spec: corev1.PodSpec{
            ServiceAccountName: "jupyter-notebook-validator-runner",  // Dedicated SA
            SecurityContext: &corev1.PodSecurityContext{
                RunAsNonRoot: pointer.Bool(true),
                RunAsUser:    pointer.Int64(1000),
                FSGroup:      pointer.Int64(1000),
            },
            Containers: []corev1.Container{
                {
                    Name:  "validator",
                    Image: job.Spec.PodConfig.ContainerImage,
                    SecurityContext: &corev1.SecurityContext{
                        AllowPrivilegeEscalation: pointer.Bool(false),
                        Capabilities: &corev1.Capabilities{
                            Drop: []corev1.Capability{"ALL"},
                        },
                    },
                },
            },
        },
    }
    
    return r.Create(ctx, pod)
}
```

## References

- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [OpenShift SCC](https://docs.openshift.com/container-platform/4.18/authentication/managing-security-context-constraints.html)
- [Operator SDK RBAC](https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/#rbac-markers)

## Related ADRs

- ADR 001: Operator Framework and SDK Version
- ADR 003: CRD Schema Design & Versioning
- ADR 004: Deployment & Packaging Strategy

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-07 | Team   | Initial decision |

