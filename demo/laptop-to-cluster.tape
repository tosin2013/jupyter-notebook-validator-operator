# Jupyter Notebook Validator - Laptop to Cluster
#
# This demo shows the full data scientist workflow:
# 1. Validate a notebook locally with Docker + Papermill
# 2. Submit the same notebook to OpenShift as a NotebookValidationJob
# 3. Watch the cluster validate it and show logs
#
# This bridges the data scientist and platform engineer paths.
#
# Render: ./demo/render.sh demo/laptop-to-cluster.tape
# Requires: docker, python3, oc (logged in), make

Output "demo/laptop-to-cluster.gif"

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
Type "echo '=== Laptop to Cluster: Full Validation Workflow ==='"
Enter
Sleep 1s

# --- PHASE 1: Local validation ---

Type "echo ''"
Enter
Type "echo '>>> PHASE 1: Validate locally (no cluster needed) <<<'"
Enter
Sleep 1s

Type "python3 demo/create-notebook.py /tmp/hello.ipynb"
Enter
Sleep 1s

Type "docker run --rm -v /tmp/hello.ipynb:/work/input.ipynb -v /tmp:/work/output quay.io/jupyter/scipy-notebook:latest bash -c 'pip install -q papermill && papermill /work/input.ipynb /work/output/hello-output.ipynb'"
Enter
Sleep 60s

Type "python3 demo/check-output.py /tmp/hello-output.ipynb"
Enter
Sleep 3s

# --- PHASE 2: Promote to cluster ---

Type "echo ''"
Enter
Type "echo '>>> PHASE 2: Promote to OpenShift cluster <<<'"
Enter
Sleep 1s

Type "oc whoami && oc version"
Enter
Sleep 2s

Type "echo 'Submitting NotebookValidationJob to the cluster...'"
Enter
Sleep 500ms

Type "oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob.yaml"
Enter
Sleep 3s

# --- PHASE 3: Watch cluster validation ---

Type "echo ''"
Enter
Type "echo '>>> PHASE 3: Watch cluster validation <<<'"
Enter
Sleep 1s

Type "oc get pods -n jupyter-notebook-validator-operator -l app=jupyter-notebook-validator -w"
Enter
Sleep 90s

Ctrl+C
Sleep 1s

# Check status
Type "oc get notebookvalidationjobs -n jupyter-notebook-validator-operator"
Enter
Sleep 3s

# Show logs
Type "echo '--- Validation pod logs ---'"
Enter
Sleep 500ms

Type "oc logs -n jupyter-notebook-validator-operator notebookvalidationjob-sample-validation --tail=30"
Enter
Sleep 5s

# Final
Type "echo ''"
Enter
Type "echo 'Local validation passed. Cluster validation Succeeded.'"
Enter
Type "echo 'Same notebook, same result, production-grade guardrails.'"
Enter
Sleep 3s
