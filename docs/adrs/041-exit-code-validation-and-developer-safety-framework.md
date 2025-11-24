# ADR 041: Exit Code Validation and Developer Safety Framework

## Status
Proposed

## Context

### Problem Statement
Notebooks can execute without raising exceptions but still **logically fail** due to silent errors. This results in **false positives** where validation reports "Succeeded" but the notebook produces incorrect or invalid results.

**Examples of Silent Failures**:

```python
# Cell 1: Load data
data = load_data("nonexistent_file.csv")
# Returns None instead of raising exception ❌

# Cell 2: Process data
result = data.mean()  # Silently fails, result = NaN

# Cell 3: Save result
save_result(result)  # Saves invalid result

# ❌ Validation reports "Succeeded" but notebook is BROKEN!
```

### Root Causes
1. **Missing error handling**: Functions return None instead of raising exceptions
2. **Disabled assertions**: No validation checks after data operations
3. **Silent NaN/Inf propagation**: Numeric errors don't cause failures
4. **No exit code enforcement**: Cells return success even when logic fails

### Impact by Developer Skill Level

| Developer Type | Risk | Example Issues |
|----------------|------|----------------|
| **Junior Developers** | 🔴 High | May not know proper error handling patterns |
| **Data Scientists** | 🟡 Medium | Focus on analysis, skip production practices |
| **ML Engineers** | 🟡 Medium | Skip validation during experimentation |
| **Domain Experts** | 🔴 High | Unfamiliar with software engineering conventions |

### User Feedback
From OPERATOR-FEEDBACK.md (OpenShift AI Ops Self-Healing Platform Team):

> **Enhancement #8: Exit Code Validation and Developer Safety Checks**
>
> **Priority**: 🔴 Critical
> **Complexity**: Medium
> **Impact**: Prevents false positives in validation results
>
> "Notebooks can execute without raising exceptions but still **logically fail**... Validation reports 'Succeeded' but notebook is broken!"

## Decision

We will implement a **multi-layered validation framework** that combines:
1. **Pre-execution linting** to detect common issues
2. **Runtime instrumentation** to catch silent failures
3. **Post-execution validation** to verify correctness
4. **Educational feedback** to help developers learn best practices

### Framework Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   VALIDATION PIPELINE                   │
└─────────────────────────────────────────────────────────┘
            │
            ├─▶ 1. PRE-EXECUTION LINTING
            │   ├─ Check for missing assertions
            │   ├─ Check for error handling
            │   ├─ Check for type hints
            │   └─ Detect anti-patterns
            │
            ├─▶ 2. RUNTIME INSTRUMENTATION
            │   ├─ Inject cell exit code checks
            │   ├─ Monitor stderr output
            │   ├─ Check for None returns
            │   └─ Detect NaN/Inf values
            │
            ├─▶ 3. POST-EXECUTION VALIDATION
            │   ├─ Verify expected output types
            │   ├─ Check output shapes/ranges
            │   ├─ Validate data quality
            │   └─ Assert final results
            │
            └─▶ 4. EDUCATIONAL FEEDBACK
                ├─ Provide helpful error messages
                ├─ Suggest best practices
                ├─ Link to documentation
                └─ Offer templates
```

### CRD API Changes

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-with-strict-checks
spec:
  notebook:
    path: "notebooks/02-anomaly-detection/01-isolation-forest.ipynb"

  # NEW: Validation configuration
  validationConfig:
    # Validation level (controls strictness)
    level: "production"  # "learning" | "development" | "staging" | "production"

    # Strict mode (enable all safety checks)
    strictMode: true  # Default: false for backward compatibility

    # Exit code enforcement
    requireExplicitExitCodes: true  # Fail if cells don't set exit codes
    failOnStderr: true              # Fail if stderr contains output
    failOnWarnings: false           # Fail even on warnings (production only)

    # Data quality checks
    checkOutputTypes: true          # Verify expected output types
    detectSilentFailures: true      # Check for None/NaN returns
    verifyAssertions: true          # Ensure assertions are present

    # Expected outputs (optional, for production)
    expectedOutputs:
      - cell: 5  # Cell index
        type: "pandas.DataFrame"
        shape: [null, 10]  # Any rows, 10 columns
        notEmpty: true

      - cell: 8
        type: "float"
        range: [0.7, 1.0]  # Model accuracy between 70-100%

    # Developer assistance
    educationalMode: true           # Provide helpful feedback
    provideExamples: true           # Show code examples in errors
    suggestBestPractices: true      # Suggest improvements
```

### Validation Levels

| Level | Strictness | Use Case | Checks |
|-------|------------|----------|--------|
| **learning** | Low | Beginners | Warnings only, no failures. Extensive educational feedback. |
| **development** | Medium | Active dev | Fail on obvious errors (None returns, NaN). Warn on missing assertions. |
| **staging** | High | Pre-production | Strict exit code enforcement. Require explicit error handling. |
| **production** | Maximum | Critical workloads | Full strictness. Require test coverage. Fail on warnings. |

### Implementation Components

#### 1. Pre-Execution Linting

```python
# internal/controller/validation_analyzer.py
import ast
import nbformat

class NotebookLinter:
    """Static analysis for notebooks to detect common issues."""

    def lint_notebook(self, notebook_path, config):
        """Run static analysis on notebook."""
        nb = nbformat.read(notebook_path, as_version=4)
        issues = []

        for idx, cell in enumerate(nb.cells):
            if cell.cell_type == "code":
                issues.extend(self.lint_cell(idx, cell.source, config))

        return issues

    def lint_cell(self, cell_idx, source, config):
        """Lint a single cell."""
        issues = []

        try:
            tree = ast.parse(source)
        except SyntaxError as e:
            return [{
                "severity": "error",
                "cell": cell_idx,
                "issue": f"Syntax error: {e}",
                "suggestion": "Fix syntax error before running notebook"
            }]

        # Check for missing error handling
        if self.has_risky_operations(tree) and not self.has_error_handling(tree):
            issues.append({
                "severity": "warning",
                "cell": cell_idx,
                "issue": "Cell has risky operations without error handling",
                "suggestion": "Add try/except blocks for file I/O, network calls, etc.",
                "example": """
try:
    data = pd.read_csv("data.csv")
    assert not data.empty, "Data is empty"
except FileNotFoundError:
    print("❌ Error: data.csv not found")
    sys.exit(1)
"""
            })

        # Check for missing assertions after data operations
        if self.has_data_operations(tree) and not self.has_assertions(tree):
            issues.append({
                "severity": "warning",
                "cell": cell_idx,
                "issue": "Cell processes data without validation checks",
                "suggestion": "Add assertions to validate data quality",
                "example": """
# After loading data
assert not data.empty, "Data is empty"
assert len(data) > 100, f"Expected at least 100 rows, got {len(data)}"

# After model training
assert model is not None, "Model training returned None"
assert accuracy > 0.7, f"Model accuracy too low: {accuracy}"
"""
            })

        return issues

    def has_risky_operations(self, tree):
        """Check if code has risky operations (file I/O, network, etc.)."""
        risky_functions = ['open', 'read_csv', 'requests.get', 'urlopen', 'load']
        for node in ast.walk(tree):
            if isinstance(node, ast.Call):
                if isinstance(node.func, ast.Name) and node.func.id in risky_functions:
                    return True
        return False

    def has_error_handling(self, tree):
        """Check if code has try/except blocks."""
        for node in ast.walk(tree):
            if isinstance(node, ast.Try):
                return True
        return False

    def has_assertions(self, tree):
        """Check if code has assertion statements."""
        for node in ast.walk(tree):
            if isinstance(node, ast.Assert):
                return True
        return False
```

#### 2. Runtime Instrumentation

```python
# internal/controller/validation_instrumenter.py
import sys
import math
import numpy as np
import pandas as pd

def instrument_notebook(notebook_path, config, output_path):
    """Inject validation checks into notebook cells."""
    nb = nbformat.read(notebook_path, as_version=4)

    # Inject preamble
    preamble_cell = nbformat.v4.new_code_cell(source="""
import sys
import math
import warnings

# Validation configuration
_STRICT_MODE = {strict_mode}
_FAIL_ON_STDERR = {fail_on_stderr}
_DETECT_SILENT_FAILURES = {detect_silent_failures}

# Cell tracking
_cell_outputs = []

def _validate_cell_output(cell_idx, result):
    '''Validate cell output for common issues.'''
    if _DETECT_SILENT_FAILURES:
        # Check for None returns
        if result is None:
            msg = f"⚠️ Warning: Cell {{cell_idx}} returned None - potential silent failure"
            print(msg, file=sys.stderr)
            if _STRICT_MODE:
                sys.exit(1)

        # Check for NaN values in numeric results
        if isinstance(result, float) and math.isnan(result):
            msg = f"❌ Error: Cell {{cell_idx}} returned NaN"
            print(msg, file=sys.stderr)
            if _STRICT_MODE:
                sys.exit(1)

        # Check for NaN values in arrays
        if isinstance(result, np.ndarray):
            if np.isnan(result).any():
                msg = f"❌ Error: Cell {{cell_idx}} result contains NaN values"
                print(msg, file=sys.stderr)
                if _STRICT_MODE:
                    sys.exit(1)

        # Check for NaN values in DataFrames
        if isinstance(result, pd.DataFrame):
            if result.isnull().any().any():
                msg = f"⚠️ Warning: Cell {{cell_idx}} DataFrame contains NaN values"
                print(msg, file=sys.stderr)

    _cell_outputs.append({{
        'cell': cell_idx,
        'result': result,
        'type': type(result).__name__
    }})

    return result
""".format(
        strict_mode=config.get('strictMode', False),
        fail_on_stderr=config.get('failOnStderr', False),
        detect_silent_failures=config.get('detectSilentFailures', True)
    ))

    nb.cells.insert(0, preamble_cell)

    # Wrap each code cell with validation
    for idx, cell in enumerate(nb.cells[1:], start=1):  # Skip preamble
        if cell.cell_type == "code":
            original_source = cell.source

            instrumented_source = f"""
# Cell {idx} - Original code
_cell_start_{idx} = True
try:
    _result_{idx} = (
{indent_code(original_source, 8)}
    )
    _validate_cell_output({idx}, _result_{idx})
except Exception as e:
    print(f"❌ Error in cell {idx}: {{e}}", file=sys.stderr)
    if _STRICT_MODE:
        sys.exit(1)
    raise
"""
            cell.source = instrumented_source

    # Write instrumented notebook
    nbformat.write(nb, output_path)

def indent_code(code, spaces):
    """Indent code block."""
    return '\n'.join(' ' * spaces + line for line in code.split('\n'))
```

#### 3. Post-Execution Validation

```go
// internal/controller/validation_result_checker.go
func (r *Reconciler) ValidateNotebookResults(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, executedNotebook string) error {
    config := job.Spec.ValidationConfig
    if config == nil || !config.CheckOutputTypes {
        return nil  // Skip if not configured
    }

    // Parse executed notebook
    nb, err := parseNotebook(executedNotebook)
    if err != nil {
        return err
    }

    // Validate expected outputs
    for _, expectedOutput := range config.ExpectedOutputs {
        cellIdx := expectedOutput.Cell
        if cellIdx >= len(nb.Cells) {
            return fmt.Errorf("expected output for cell %d, but notebook only has %d cells", cellIdx, len(nb.Cells))
        }

        cell := nb.Cells[cellIdx]
        if len(cell.Outputs) == 0 {
            return fmt.Errorf("cell %d has no outputs (expected %s)", cellIdx, expectedOutput.Type)
        }

        // Check output type
        actualType := detectOutputType(cell.Outputs[0])
        if actualType != expectedOutput.Type {
            return fmt.Errorf("cell %d output type mismatch: expected %s, got %s", cellIdx, expectedOutput.Type, actualType)
        }

        // Check output shape (for DataFrames, arrays)
        if expectedOutput.Shape != nil {
            actualShape := extractOutputShape(cell.Outputs[0])
            if !shapeMatches(actualShape, expectedOutput.Shape) {
                return fmt.Errorf("cell %d output shape mismatch: expected %v, got %v", cellIdx, expectedOutput.Shape, actualShape)
            }
        }

        // Check output range (for numeric values)
        if expectedOutput.Range != nil {
            value := extractNumericValue(cell.Outputs[0])
            if value < expectedOutput.Range[0] || value > expectedOutput.Range[1] {
                return fmt.Errorf("cell %d output out of range: expected [%f, %f], got %f", cellIdx, expectedOutput.Range[0], expectedOutput.Range[1], value)
            }
        }

        // Check not empty (for DataFrames)
        if expectedOutput.NotEmpty {
            if isEmptyOutput(cell.Outputs[0]) {
                return fmt.Errorf("cell %d output is empty (expected non-empty)", cellIdx)
            }
        }
    }

    return nil
}
```

#### 4. Educational Feedback System

```yaml
# Status with educational feedback
status:
  phase: "Failed"
  message: "Validation failed: silent failures detected in cells 3 and 8"

  educationalFeedback:
    - issue: "Cell 3 returned None without explicit error"
      severity: "error"
      cell: 3
      explanation: |
        Your function load_data() returned None when the file wasn't found,
        but didn't raise an exception. This creates a "silent failure" where
        the notebook appears to succeed but produces invalid results.

      bestPractice: |
        Use explicit error handling:

        def load_data(path):
            if not os.path.exists(path):
                raise FileNotFoundError(f"Data file not found: {path}")
            return pd.read_csv(path)

      documentation: "https://docs.python.org/3/tutorial/errors.html"

    - issue: "No assertions found in data processing cells"
      severity: "warning"
      cell: 5
      suggestion: |
        Add data quality checks after loading:

        assert not data.empty, "Data is empty"
        assert len(data) > 100, f"Expected at least 100 rows, got {len(data)}"
        assert data['column'].notna().all(), "Missing values in critical column"
```

## Consequences

### Positive
- ✅ **Eliminates false positives**: Validation actually validates correctness
- ✅ **Improves notebook quality**: Developers learn best practices
- ✅ **Flexible strictness**: Adjust per team/environment (learning → production)
- ✅ **Educational**: Teaches proper error handling and validation
- ✅ **Production-ready**: Ensures notebooks are truly deployment-ready
- ✅ **Backward compatible**: Disabled by default (opt-in)

### Negative
- ⏱️ **Increased execution time**: Pre-execution linting adds ~2-5 seconds
- 🔄 **Complexity**: More validation logic to maintain
- 📚 **Documentation burden**: Need to explain validation levels and best practices

### Neutral
- 🔧 **Configuration options**: Many knobs to tune (can be overwhelming)
- 📊 **Status verbosity**: Educational feedback makes status longer

## Implementation Notes

### Testing Strategy

#### Unit Tests
- [ ] Test linter detects missing error handling
- [ ] Test linter detects missing assertions
- [ ] Test instrumentation injects validation correctly
- [ ] Test post-execution validator checks output types
- [ ] Test post-execution validator checks output ranges

#### Integration Tests
- [ ] Test strict mode catches None returns
- [ ] Test strict mode catches NaN values
- [ ] Test learning mode provides warnings only
- [ ] Test production mode fails on warnings
- [ ] Test educational feedback generation

#### E2E Tests
- [ ] Create notebook with silent failure (None return)
- [ ] Verify learning mode warns but doesn't fail
- [ ] Verify production mode fails
- [ ] Verify educational feedback in status
- [ ] Create notebook with proper error handling
- [ ] Verify passes in all modes

### Phased Rollout

#### Phase 1: Basic Exit Code Validation (Week 1)
- [ ] Implement runtime instrumentation for None/NaN detection
- [ ] Add `strictMode` flag (default: false)
- [ ] Add `failOnStderr` flag

#### Phase 2: Data Quality Checks (Week 2)
- [ ] Implement post-execution output validation
- [ ] Add `expectedOutputs` field
- [ ] Support DataFrame shape/type checks

#### Phase 3: Educational Mode (Week 3)
- [ ] Implement pre-execution linting
- [ ] Add educational feedback generation
- [ ] Create best practice templates

#### Phase 4: Validation Levels (Week 4)
- [ ] Implement level-based configuration
- [ ] Document level usage guidelines
- [ ] Create migration guide

### Migration Path

#### For Existing Users
1. **v0.1.x → v0.2.0**: No breaking changes (all features opt-in)
2. **Gradual Adoption**:
   - Start with `level: "learning"` (warnings only)
   - Move to `level: "development"` after fixing issues
   - Move to `level: "staging"` for pre-production
   - Move to `level: "production"` for critical workloads

## References

- [OPERATOR-FEEDBACK.md](../../OPERATOR-FEEDBACK.md) - Enhancement #8: Exit Code Validation
- [Python Error Handling](https://docs.python.org/3/tutorial/errors.html)
- [Pandas Data Validation](https://pandas.pydata.org/docs/reference/api/pandas.DataFrame.empty.html)
- [Jupyter Notebook Format](https://nbformat.readthedocs.io/)

## Related ADRs

- [ADR-008: Notebook Testing Strategy](008-notebook-testing-strategy-and-complexity-levels.md) - Test tiers
- [ADR-013: Output Comparison Strategy](013-output-comparison-and-diffing-strategy.md) - Golden notebook comparison
- [ADR-037: Build-Validation Sequencing](037-build-validation-sequencing-and-state-machine.md) - Validation phase

## Revision History

| Date | Author | Description |
|------|--------|-------------|
| 2025-11-20 | Claude Code | Initial proposal based on production feedback |
