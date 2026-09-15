# Jupyter Notebook Validator - Local Validation with Docker/Podman
#
# This demo shows a data scientist validating a notebook on their laptop
# using Docker and Papermill. No Kubernetes cluster required.
#
# Mirrors: docs/tutorials/LOCAL_VALIDATION.md
#
# Render: ./demo/render.sh demo/local-validation.tape
# Requires: docker (or podman), python3

Output "demo/local-validation.gif"

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 30ms
Set Padding 15

# Title
Type "echo '=== Jupyter Notebook Validator: Local Quick Start ==='"
Enter
Sleep 1s

# Step 1: Create a test notebook
Type "echo '--- Step 1: Create a test notebook ---'"
Enter
Sleep 500ms

Type "python3 demo/create-notebook.py /tmp/hello.ipynb"
Enter
Sleep 1s

# Step 2: Run with Docker + Papermill
Type "echo '--- Step 2: Validate with Docker + Papermill ---'"
Enter
Sleep 500ms

Type "docker run --rm -v /tmp/hello.ipynb:/work/input.ipynb -v /tmp:/work/output quay.io/jupyter/scipy-notebook:latest bash -c 'pip install -q papermill && papermill /work/input.ipynb /work/output/hello-output.ipynb'"
Enter
Sleep 60s

# Step 3: Check the result
Type "echo '--- Step 3: Inspect results ---'"
Enter
Sleep 500ms

Type "python3 demo/check-output.py /tmp/hello-output.ipynb"
Enter
Sleep 3s

Type "echo 'Ready for cluster validation.'"
Enter
Type "echo 'Next: docs/tutorials/QUICK_START_CI_CD.md'"
Enter
Sleep 3s
