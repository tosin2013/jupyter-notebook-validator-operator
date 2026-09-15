# Jupyter Notebook Validator - Local Validation with Podman/Docker
#
# This demo shows a data scientist validating a notebook on their laptop
# using Podman (or Docker) and Papermill. No Kubernetes cluster required.
#
# Mirrors: docs/tutorials/LOCAL_VALIDATION.md
#
# Render: vhs demo/local-validation.tape
# Requires: podman (or docker), python3

Output demo/local-validation.gif
Output demo/local-validation.mp4

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
Type "# Jupyter Notebook Validator - Local Validation Quick Start"
Enter
Type "# Validate a notebook on your laptop with Podman + Papermill"
Enter
Sleep 1s

# Step 1: Create a minimal notebook
Type "echo '--- Step 1: Create a test notebook ---'"
Enter
Sleep 500ms

Type@30ms "cat > hello.ipynb << 'EOF'"
Enter
Type@20ms '{"cells":[{"cell_type":"code","execution_count":null,"metadata":{},"outputs":[],"source":["print(\"Hello from Papermill!\")\n","assert 1 + 1 == 2, \"Math is broken\""]}],"metadata":{"kernelspec":{"display_name":"Python 3","language":"python","name":"python3"},"language_info":{"name":"python","version":"3.11.0"}},"nbformat":4,"nbformat_minor":5}'
Enter
Type "EOF"
Enter
Sleep 1s

Type "echo 'Notebook created: hello.ipynb'"
Enter
Sleep 1s

# Step 2: Run with Podman
Type "echo '--- Step 2: Run notebook with Podman + Papermill ---'"
Enter
Sleep 500ms

Type@30ms 'podman run --rm \'
Enter
Type@30ms '  -v "$(pwd)/hello.ipynb:/work/input.ipynb:Z" \'
Enter
Type@30ms '  -v "$(pwd):/work/output:Z" \'
Enter
Type@30ms '  quay.io/jupyter/scipy-notebook:latest \'
Enter
Type@30ms '  bash -c "pip install -q papermill && papermill /work/input.ipynb /work/output/hello-output.ipynb"'
Enter
Sleep 15s

# Step 3: Check the result
Type "echo '--- Step 3: Check results ---'"
Enter
Sleep 500ms

Type@30ms "python3 -c \""
Enter
Type@20ms "import json"
Enter
Type@20ms "nb = json.load(open('hello-output.ipynb'))"
Enter
Type@20ms "for i, cell in enumerate(nb['cells']):"
Enter
Type@20ms "    if cell['cell_type'] == 'code':"
Enter
Type@20ms "        outputs = ''.join(o.get('text', [''])[0] if isinstance(o.get('text'), list) else o.get('text', '') for o in cell['outputs'] if 'text' in o)"
Enter
Type@20ms "        status = 'PASS' if outputs else 'CHECK'"
Enter
Type@20ms "        print(f'Cell {i}: {status}  {outputs.strip()[:80]}')"
Enter
Type@20ms "\""
Enter
Sleep 3s

# Final message
Type "echo ''"
Enter
Type "echo 'All cells passed. Ready for cluster validation.'"
Enter
Type "echo 'Next: docs/tutorials/QUICK_START_CI_CD.md'"
Enter
Sleep 3s
