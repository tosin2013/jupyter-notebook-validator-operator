# Troubleshooting Build Detection Issues

## Quick Diagnosis

### Check Job BuildStatus

```bash
# Get full build status
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus}' | jq

# Get just the error message
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus.message}'

# Get strategy and phase
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus.strategy}'
oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus.phase}'
```

### Check Operator Logs

```bash
# Recent logs with build-related messages
oc logs -n jupyter-notebook-validator-operator \
  deployment/notebook-validator-controller-manager \
  -c manager --tail=100 | grep -i "build\|s2i\|tekton\|detect"

# Follow logs in real-time
oc logs -n jupyter-notebook-validator-operator \
  deployment/notebook-validator-controller-manager \
  -c manager -f
```

## Common Issues and Solutions

### Issue 1: "Strategy not available: s2i"

**Symptoms**:
```yaml
buildStatus:
  phase: Failed
  strategy: s2i
  message: "Strategy not available: s2i. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster."
```

**Diagnosis**:
```bash
# Check if BuildConfig CRD exists
oc api-resources | grep -i buildconfig

# Check if OpenShift Build API is available
oc get crd buildconfigs.build.openshift.io

# Try to list BuildConfigs
oc get buildconfigs -A
```

**Solutions**:

1. **On OpenShift**: BuildConfig should be available by default
   ```bash
   # Verify OpenShift version
   oc version
   
   # Check if build operator is running
   oc get pods -n openshift-cluster-version
   ```

2. **On vanilla Kubernetes**: S2I is not available
   - Use Tekton strategy instead
   - Or use container image directly (no build)

3. **If BuildConfig exists but detection fails**: Check RBAC permissions
   ```bash
   # Check ClusterRole permissions
   oc get clusterrole notebook-validator-manager-role -o yaml | grep -A 10 "build.openshift.io"
   
   # Should see:
   # - apiGroups:
   #   - build.openshift.io
   #   resources:
   #   - buildconfigs
   #   - builds
   #   verbs:
   #   - create
   #   - delete
   #   - get
   #   - list
   #   - patch
   #   - update
   #   - watch
   ```

### Issue 2: "Strategy not available: tekton"

**Symptoms**:
```yaml
buildStatus:
  phase: Failed
  strategy: tekton
  message: "Strategy not available: tekton. This may indicate that the required CRDs (BuildConfig for S2I, Pipeline for Tekton) are not installed in the cluster."
```

**Diagnosis**:
```bash
# Check if Tekton CRDs exist
oc api-resources | grep tekton

# Check if TaskRun CRD exists
oc get crd taskruns.tekton.dev

# Try to list TaskRuns
oc get taskruns -A
```

**Solutions**:

1. **Install Tekton Pipelines**:
   ```bash
   # On OpenShift: Install OpenShift Pipelines Operator
   # Via OperatorHub or:
   oc apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
   ```

2. **Verify installation**:
   ```bash
   # Check Tekton pods
   oc get pods -n tekton-pipelines
   
   # Should see:
   # tekton-pipelines-controller
   # tekton-pipelines-webhook
   ```

3. **Check RBAC permissions**:
   ```bash
   oc get clusterrole notebook-validator-manager-role -o yaml | grep -A 10 "tekton.dev"
   ```

### Issue 3: "Strategy detection failed for s2i: [permission error]"

**Symptoms**:
```yaml
buildStatus:
  phase: Failed
  strategy: s2i
  message: "Strategy detection failed for s2i: buildconfigs.build.openshift.io is forbidden: User \"system:serviceaccount:default:notebook-validator-jupyter-notebook-validator-runner\" cannot list resource \"buildconfigs\" in API group \"build.openshift.io\" in the namespace \"default\""
```

**Diagnosis**:
```bash
# Check ServiceAccount
oc get sa notebook-validator-jupyter-notebook-validator-runner -n default

# Check RoleBindings
oc get rolebinding -n default | grep notebook-validator

# Check ClusterRoleBindings
oc get clusterrolebinding | grep notebook-validator
```

**Solutions**:

1. **Verify ClusterRoleBinding exists**:
   ```bash
   oc get clusterrolebinding notebook-validator-manager-rolebinding -o yaml
   ```

2. **Recreate RBAC if missing**:
   ```bash
   make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:release-4.18-<sha>
   ```

3. **Manual RBAC fix** (if needed):
   ```bash
   oc create clusterrolebinding notebook-validator-build-access \
     --clusterrole=notebook-validator-manager-role \
     --serviceaccount=default:notebook-validator-jupyter-notebook-validator-runner
   ```

### Issue 4: "no matches for kind BuildConfig"

**Symptoms**:
```yaml
buildStatus:
  phase: Failed
  strategy: s2i
  message: "Strategy detection failed for s2i: no matches for kind BuildConfig in group build.openshift.io"
```

**Diagnosis**:
```bash
# Check API groups
oc api-versions | grep build

# Check if build.openshift.io is available
oc api-resources --api-group=build.openshift.io
```

**Solutions**:

1. **You're on vanilla Kubernetes**: S2I is OpenShift-specific
   - Switch to Tekton strategy
   - Or use direct container image

2. **You're on OpenShift but Build API is disabled**:
   ```bash
   # Check cluster operators
   oc get clusteroperators
   
   # Look for issues with openshift-apiserver
   oc get clusteroperator openshift-apiserver -o yaml
   ```

## Verification Steps

### After Fixing Issues

1. **Delete and recreate the job**:
   ```bash
   oc delete notebookvalidationjob <job-name>
   oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml
   ```

2. **Watch the build status**:
   ```bash
   watch -n 2 'oc get notebookvalidationjob <job-name> -o jsonpath="{.status.buildStatus}" | jq'
   ```

3. **Check for successful detection**:
   ```bash
   # Should see in operator logs:
   # "S2I available: BuildConfig API detected"
   # or
   # "Tekton available: TaskRun API detected"
   
   oc logs -n jupyter-notebook-validator-operator \
     deployment/notebook-validator-controller-manager \
     -c manager --tail=50 | grep "available:"
   ```

4. **Verify build creation**:
   ```bash
   # For S2I
   oc get buildconfigs
   oc get builds
   
   # For Tekton
   oc get taskruns
   ```

## Getting Help

If you're still experiencing issues:

1. **Collect diagnostic information**:
   ```bash
   # Save to file for sharing
   {
     echo "=== Job Status ==="
     oc get notebookvalidationjob <job-name> -o yaml
     
     echo -e "\n=== BuildStatus ==="
     oc get notebookvalidationjob <job-name> -o jsonpath='{.status.buildStatus}' | jq
     
     echo -e "\n=== Operator Logs ==="
     oc logs -n jupyter-notebook-validator-operator \
       deployment/notebook-validator-controller-manager \
       -c manager --tail=100
     
     echo -e "\n=== API Resources ==="
     oc api-resources | grep -E "build|tekton"
     
     echo -e "\n=== RBAC ==="
     oc get clusterrole notebook-validator-manager-role -o yaml
     
   } > diagnostic-output.txt
   ```

2. **Check documentation**:
   - [Build Error Reporting](BUILD-ERROR-REPORTING.md)
   - [ADR-016: S2I Build Strategy](adr/ADR-016-S2I-Build-Strategy-for-Git-Integration.md)
   - [Git Clone Approaches](GIT-CLONE-APPROACHES.md)

3. **Open an issue**: Include the diagnostic output file

