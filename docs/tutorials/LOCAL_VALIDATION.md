# Validate a Notebook Locally with Podman or Docker

Run a Jupyter notebook through Papermill on your laptop. No Kubernetes cluster, no kubectl, no Helm required.

## Prerequisites

- **Podman** or **Docker** installed on your machine.
- A Jupyter notebook (`.ipynb` file) you want to validate.

If you do not have a notebook handy, use one of the samples shipped with this project:

```bash
git clone https://github.com/tosin2013/jupyter-notebook-validator-operator.git
cd jupyter-notebook-validator-operator
```

The `config/samples/` directory contains example CRs that reference notebooks in the test repository at
`https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git`.

## Step 1: Choose a notebook

For this tutorial, create a minimal notebook to validate:

```bash
cat > hello.ipynb << 'NOTEBOOK'
{
 "cells": [
  {
   "cell_type": "code",
   "execution_count": null,
   "metadata": {},
   "outputs": [],
   "source": ["print('Hello from Papermill!')\n", "assert 1 + 1 == 2, 'Math is broken'"]
  }
 ],
 "metadata": {
  "kernelspec": {
   "display_name": "Python 3",
   "language": "python",
   "name": "python3"
  },
  "language_info": {
   "name": "python",
   "version": "3.11.0"
  }
 },
 "nbformat": 4,
 "nbformat_minor": 5
}
NOTEBOOK
```

## Step 2: Run the notebook with Podman (or Docker)

Execute the notebook inside a container. Replace `podman` with `docker` if you use Docker.

```bash
podman run --rm \
  -v "$(pwd)/hello.ipynb:/work/input.ipynb:Z" \
  -v "$(pwd):/work/output:Z" \
  quay.io/jupyter/scipy-notebook:latest \
  bash -c "pip install -q papermill && papermill /work/input.ipynb /work/output/hello-output.ipynb"
```

What this does:

1. Mounts `hello.ipynb` into the container as `/work/input.ipynb`.
2. Mounts the current directory as `/work/output` so the output notebook is written back to your host.
3. Installs Papermill inside the container (it is not included in the base image).
4. Executes the notebook cell-by-cell and writes the output to `hello-output.ipynb`.

## Step 3: Check the result

Open `hello-output.ipynb` in JupyterLab, VS Code, or any notebook viewer. Each cell should show its output and execution count. If a cell fails, Papermill stops execution and the error appears in the output notebook.

You can also inspect the output from the command line:

```bash
python3 -c "
import json, sys
nb = json.load(open('hello-output.ipynb'))
for i, cell in enumerate(nb['cells']):
    if cell['cell_type'] == 'code':
        outputs = ''.join(o.get('text', [''])[0] if isinstance(o.get('text'), list) else o.get('text', '') for o in cell['outputs'] if 'text' in o)
        status = 'PASS' if outputs or not cell['outputs'] else 'CHECK'
        print(f'Cell {i}: {status}  {outputs.strip()[:80]}')
"
```

## Step 4: Validate with a golden baseline (optional)

Save the output notebook as your golden reference:

```bash
cp hello-output.ipynb hello-golden.ipynb
```

After making changes to the notebook, re-run Step 2 and compare the output against the golden file:

```bash
python3 -c "
import json
golden = json.load(open('hello-golden.ipynb'))
current = json.load(open('hello-output.ipynb'))
for i, (g, c) in enumerate(zip(golden['cells'], current['cells'])):
    if g['cell_type'] == 'code':
        g_out = [o for o in g.get('outputs', [])]
        c_out = [o for o in c.get('outputs', [])]
        match = g_out == c_out
        print(f'Cell {i}: {\"MATCH\" if match else \"DIFF\"}')
        if not match:
            print(f'  Golden:  {g_out[:1]}')
            print(f'  Current: {c_out[:1]}')
"
```

When you move to a cluster, the operator performs this comparison automatically with configurable numeric tolerances.

## Step 5: Run a real data-science notebook

Try a more realistic notebook. Clone the test notebooks repository:

```bash
git clone https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
```

Run a tier-1 notebook:

```bash
podman run --rm \
  -v "$(pwd)/jupyter-notebook-validator-test-notebooks:/work/repo:Z" \
  -v "$(pwd):/work/output:Z" \
  quay.io/jupyter/scipy-notebook:latest \
  bash -c "pip install -q papermill && papermill /work/repo/notebooks/tier1-simple/01-hello-world.ipynb /work/output/01-hello-world-output.ipynb"
```

## What comes next

You have validated a notebook on your laptop with the same tool (Papermill) that the operator uses on a cluster.

When you are ready to validate notebooks in a production-like environment with GPU access, model endpoints, and automated golden comparison:

1. **Submit to a cluster:** See [Quick Start CI/CD](QUICK_START_CI_CD.md) for how to install the operator and submit a `NotebookValidationJob`.
2. **Configure golden comparison:** See [Golden Notebook Comparison](../how-to/GOLDEN_NOTEBOOK_COMPARISON.md) for tolerance settings.
3. **Validate against live models:** See [Model Discovery Guide](../how-to/MODEL_DISCOVERY_GUIDE.md) for auto-detecting model serving endpoints.
