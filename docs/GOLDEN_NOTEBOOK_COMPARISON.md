# Golden Notebook Comparison

## Overview

The Jupyter Notebook Validator Operator supports **golden notebook comparison** to detect regressions in notebook execution. This feature compares the output of an executed notebook against a "golden" reference notebook to ensure consistency.

## Implementation Status

✅ **Phase 3 Complete** (2025-11-08)

- Golden notebook fetching via second init container
- Cell-by-cell output comparison
- Multiple comparison strategies (exact, normalized)
- Diff generation and reporting
- Status updates with comparison results

## Architecture

### Pod Structure

When a golden notebook is specified, the validation pod includes two init containers:

```yaml
initContainers:
  - name: git-clone
    # Clones the target notebook to /workspace/repo
  - name: golden-git-clone
    # Clones the golden notebook to /workspace/golden
containers:
  - name: validator
    # Executes notebook and performs comparison
```

### Comparison Flow

1. **Notebook Execution**: Target notebook is executed with Papermill
2. **Golden Parsing**: Golden notebook is parsed from `/workspace/golden`
3. **Cell-by-Cell Comparison**: Each cell's output is compared
4. **Diff Generation**: Differences are identified and categorized
5. **Status Update**: Comparison results are stored in CR status

## Usage

### Basic Example

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation-job
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/my-notebook.ipynb"
  
  goldenNotebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/my-notebook-golden.ipynb"
  
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
```

### Comparison Strategies

Configure comparison strategy via annotations:

```yaml
metadata:
  annotations:
    mlops.dev/comparison-strategy: "exact"  # or "normalized"
```

#### Available Strategies

1. **exact** (default)
   - Byte-for-byte comparison of outputs
   - Strictest comparison mode
   - Best for deterministic notebooks

2. **normalized** (Phase 3)
   - Ignores whitespace differences
   - Ignores timestamps (configurable patterns)
   - Best for notebooks with timestamps or formatting variations

3. **fuzzy** (Phase 4 - Planned)
   - Floating-point tolerance
   - Configurable epsilon for numeric comparisons
   - Best for notebooks with floating-point calculations

4. **semantic** (Phase 4 - Planned)
   - Semantic comparison of outputs
   - Understands data structures
   - Best for complex data outputs

### Comparison Configuration

Additional configuration via annotations:

```yaml
metadata:
  annotations:
    mlops.dev/comparison-strategy: "normalized"
    mlops.dev/numeric-tolerance: "0.001"
    mlops.dev/ignore-timestamps: "true"
    mlops.dev/ignore-execution-count: "true"
```

## Comparison Results

### Status Fields

Comparison results are stored in the CR status:

```yaml
status:
  comparisonResult:
    strategy: "exact"
    result: "matched"  # or "failed"
    totalCells: 5
    matchedCells: 5
    mismatchedCells: 0
    diffs: []
```

### Diff Format

When cells don't match, diffs are generated:

```yaml
diffs:
  - cellIndex: 2
    cellType: "code"
    diffType: "output_mismatch"
    expected: "Expected output"
    actual: "Actual output"
    severity: "major"
```

### Diff Types

- **output_mismatch**: Cell outputs don't match
- **extra_cell**: Executed notebook has extra cells
- **missing_cell**: Executed notebook is missing cells
- **cell_type_mismatch**: Cell types don't match

### Severity Levels

- **minor**: Whitespace or formatting differences
- **major**: Content differences
- **critical**: Structural differences (missing/extra cells)

## Creating Golden Notebooks

### Best Practices

1. **Execute Once**: Run the notebook once to generate expected outputs
2. **Review Outputs**: Manually verify all outputs are correct
3. **Commit to Git**: Store golden notebook in version control
4. **Update Regularly**: Update golden notebook when expected behavior changes

### Example Workflow

```bash
# 1. Execute notebook to generate outputs
jupyter nbconvert --execute --to notebook \
  --output my-notebook-golden.ipynb \
  my-notebook.ipynb

# 2. Review the golden notebook
jupyter notebook my-notebook-golden.ipynb

# 3. Commit to Git
git add my-notebook-golden.ipynb
git commit -m "Add golden notebook for regression testing"
git push
```

### Golden Notebook Naming Convention

Recommended naming pattern:

```
notebooks/
  my-notebook.ipynb          # Target notebook
  my-notebook-golden.ipynb   # Golden reference
```

## Authentication

Golden notebooks support the same authentication methods as target notebooks:

### HTTPS Authentication

```yaml
goldenNotebook:
  git:
    url: "https://github.com/myorg/notebooks.git"
    ref: "main"
    credentialsSecret: "git-credentials"
  path: "notebooks/golden.ipynb"
```

### SSH Authentication

```yaml
goldenNotebook:
  git:
    url: "git@github.com:myorg/notebooks.git"
    ref: "main"
    credentialsSecret: "git-ssh-credentials"
  path: "notebooks/golden.ipynb"
```

### Different Repositories

Golden notebooks can be in a different repository:

```yaml
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/my-notebook.ipynb"
  
  goldenNotebook:
    git:
      url: "https://github.com/myorg/golden-notebooks.git"
      ref: "main"
      credentialsSecret: "golden-git-credentials"
    path: "golden/my-notebook-golden.ipynb"
```

## Troubleshooting

### Golden Notebook Not Found

**Error**: `Golden notebook not found at path: notebooks/golden.ipynb`

**Solution**: Verify the path is correct and the file exists in the repository

```bash
git clone <golden-repo-url>
ls -la notebooks/golden.ipynb
```

### Comparison Failed

**Error**: `Validation failed: golden notebook comparison failed (3/5 cells matched)`

**Solution**: Check the diff details in the CR status:

```bash
kubectl get notebookvalidationjob my-job -o yaml | grep -A 20 comparisonResult
```

### Golden Notebook Parsing Failed

**Error**: `golden notebook parsing failed: invalid JSON`

**Solution**: Ensure the golden notebook is a valid Jupyter notebook:

```bash
jupyter nbconvert --to notebook --execute golden.ipynb
```

## Examples

### Example 1: Simple Comparison

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: simple-comparison
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/hello-world.ipynb"
  goldenNotebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/hello-world-golden.ipynb"
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
```

### Example 2: Normalized Comparison

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: normalized-comparison
  annotations:
    mlops.dev/comparison-strategy: "normalized"
    mlops.dev/ignore-timestamps: "true"
spec:
  notebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/data-analysis.ipynb"
  goldenNotebook:
    git:
      url: "https://github.com/myorg/notebooks.git"
      ref: "main"
    path: "notebooks/data-analysis-golden.ipynb"
  podConfig:
    containerImage: "jupyter/scipy-notebook:latest"
```

## Related Documentation

- [ADR-013: Output Comparison and Diffing Strategy](adrs/013-output-comparison-and-diffing-strategy.md)
- [ADR-008: Notebook Testing Strategy](adrs/008-notebook-testing-strategy-and-complexity-levels.md)
- [Testing Guide](TESTING_GUIDE.md)

## Future Enhancements

### Phase 4 (Planned)

- Fuzzy comparison with floating-point tolerance
- Semantic comparison for complex data structures
- Configurable comparison rules per cell
- Visual diff reports
- Comparison metrics and trends

### Phase 5 (Planned)

- Multiple golden notebooks (A/B testing)
- Golden notebook versioning
- Automatic golden notebook updates
- Comparison performance optimizations

