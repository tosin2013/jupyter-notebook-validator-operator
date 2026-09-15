# Jupyter Notebook Validator - OpenShift Operator Deployment
#
# This demo shows a platform engineer deploying the operator to an
# OpenShift cluster and running a validation job end-to-end.
#
# Render: ./demo/render.sh demo/openshift-deploy.tape
# Requires: oc (logged in), make, go

Output "demo/openshift-deploy.gif"

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 30ms
Set Padding 15
Set CursorBlink false

# Title
Type "echo '=== Jupyter Notebook Validator: OpenShift Deployment ==='"
Enter
Sleep 1s

# Step 1: Verify cluster
Type "echo '--- Step 1: Verify OpenShift cluster ---'"
Enter
Sleep 500ms

Type "oc whoami"
Enter
Sleep 1s

Type "oc version"
Enter
Sleep 2s

# Step 2: Install CRDs
Type "echo '--- Step 2: Install CRDs ---'"
Enter
Sleep 500ms

Type "make install"
Enter
Sleep 15s

# Step 3: Deploy operator
Type "echo '--- Step 3: Deploy operator ---'"
Enter
Sleep 500ms

Type "make deploy IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest"
Enter
Sleep 15s

# Step 4: Wait for operator
Type "echo '--- Step 4: Verify operator is running ---'"
Enter
Sleep 500ms

Type "oc get pods -n jupyter-notebook-validator-operator"
Enter
Sleep 3s

# Step 5: Submit a validation job
Type "echo '--- Step 5: Submit a NotebookValidationJob ---'"
Enter
Sleep 500ms

Type "oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob.yaml"
Enter
Sleep 3s

# Step 6: Watch status
Type "echo '--- Step 6: Check validation status ---'"
Enter
Sleep 500ms

Type "oc get notebookvalidationjobs"
Enter
Sleep 5s

Type "oc get pods -l app=notebook-validation"
Enter
Sleep 3s

# Final
Type "echo ''"
Enter
Type "echo 'Operator deployed and notebook validated on OpenShift.'"
Enter
Type "echo 'Docs: docs/how-to/E2E_ROSA_RUNBOOK.md'"
Enter
Sleep 3s
