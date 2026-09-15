# Jupyter Notebook Validator - OpenShift Operator Deployment
#
# This demo shows a platform engineer deploying the operator to an
# OpenShift cluster and running a validation job end-to-end.
#
# Render: vhs demo/openshift-deploy.tape
# Requires: oc, make, go (connected to an OpenShift cluster)

Output demo/openshift-deploy.gif
Output demo/openshift-deploy.mp4

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 40ms
Set Padding 20

# Title banner
Type "# Jupyter Notebook Validator - OpenShift Operator Deployment"
Enter
Type "# Deploy the operator and validate a notebook on a live cluster"
Enter
Sleep 1s

# Step 1: Login to OpenShift
Type "echo '--- Step 1: Connect to OpenShift cluster ---'"
Enter
Sleep 500ms

Type@30ms "oc login --server=https://api.my-cluster.example.com:6443 -u admin"
Enter
Sleep 2s

Type "oc whoami"
Enter
Sleep 1s

Type "oc version --short"
Enter
Sleep 1s

# Step 2: Install CRDs
Type "echo '--- Step 2: Install CRDs ---'"
Enter
Sleep 500ms

Type@30ms "make install"
Enter
Sleep 3s

# Step 3: Deploy operator
Type "echo '--- Step 3: Deploy the operator ---'"
Enter
Sleep 500ms

Type@30ms "make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest"
Enter
Sleep 5s

# Step 4: Verify operator is running
Type "echo '--- Step 4: Verify operator pod ---'"
Enter
Sleep 500ms

Type@30ms "oc get pods -n jupyter-notebook-validator-operator"
Enter
Sleep 2s

# Step 5: Create a validation job
Type "echo '--- Step 5: Submit a NotebookValidationJob ---'"
Enter
Sleep 500ms

Type@30ms "cat <<'CR' | oc apply -f -"
Enter
Type@20ms "apiVersion: mlops.mlops.dev/v1alpha1"
Enter
Type@20ms "kind: NotebookValidationJob"
Enter
Type@20ms "metadata:"
Enter
Type@20ms "  name: demo-hello-world"
Enter
Type@20ms "  namespace: default"
Enter
Type@20ms "spec:"
Enter
Type@20ms "  notebook:"
Enter
Type@20ms "    git:"
Enter
Type@20ms "      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
Enter
Type@20ms "      ref: main"
Enter
Type@20ms "    path: notebooks/tier1-simple/01-hello-world.ipynb"
Enter
Type@20ms "  podConfig:"
Enter
Type@20ms "    containerImage: quay.io/jupyter/scipy-notebook:latest"
Enter
Type@20ms "  timeout: \"5m\""
Enter
Type@20ms "CR"
Enter
Sleep 2s

# Step 6: Watch the validation
Type "echo '--- Step 6: Watch validation progress ---'"
Enter
Sleep 500ms

Type@30ms "oc get notebookvalidationjobs demo-hello-world -w"
Enter
Sleep 10s

Ctrl+C
Sleep 500ms

# Step 7: Check final status
Type "echo '--- Step 7: Final status ---'"
Enter
Sleep 500ms

Type@30ms "oc get notebookvalidationjob demo-hello-world -o jsonpath='{.status.phase}'"
Enter
Sleep 1s

Type "echo ''"
Enter

Type@30ms "oc get notebookvalidationjob demo-hello-world -o jsonpath='{.status.conditions[*].message}'"
Enter
Sleep 2s

Type "echo ''"
Enter

# Final message
Type "echo ''"
Enter
Type "echo 'Notebook validation succeeded on OpenShift.'"
Enter
Type "echo 'Operator docs: docs/tutorials/QUICK_START_CI_CD.md'"
Enter
Type "echo 'E2E runbook:   docs/how-to/E2E_ROSA_RUNBOOK.md'"
Enter
Sleep 3s
