# ADR 013: Output Comparison and Diffing Strategy

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator must compare executed notebook outputs against golden (reference) notebooks to detect regressions and validate correctness. This comparison is critical for:

1. **Regression Detection**: Identify when code changes produce different outputs
2. **Reproducibility Validation**: Ensure notebooks produce consistent results
3. **CI/CD Integration**: Automated validation in deployment pipelines
4. **Quality Assurance**: Verify notebook behavior matches expectations

### PRD Requirements

**AC-3 (for US-4):**
- The CRD spec includes a field `spec.goldenNotebook` to point to a golden version of the notebook
- The controller fetches both the target notebook and the golden notebook
- After executing the target notebook, the controller performs a diff on the cell outputs
- The validation `status` is marked as `Failed` if significant differences are found

**US-5**: "View structured, cell-by-cell results of validation runs, including error messages and output diffs"

### Challenges

1. **Non-Deterministic Outputs**: Timestamps, random numbers, memory addresses
2. **Floating-Point Precision**: Numerical differences due to rounding
3. **Output Format Variations**: Text, HTML, images, JSON, plots
4. **Cell Order Changes**: Notebooks may be refactored with different cell ordering
5. **Acceptable Differences**: Some changes are intentional and should not fail validation
6. **Performance**: Large notebooks with many cells and outputs

## Decision

We will implement a **Configurable Multi-Strategy Comparison System** with the following components:

### 1. Comparison Strategies

#### Strategy 1: Exact Match (Default)
- **Use Case**: Deterministic notebooks with predictable outputs
- **Behavior**: Cell outputs must match exactly (byte-for-byte)
- **Pros**: Simple, fast, no false positives
- **Cons**: Fails on any difference (timestamps, random values)

#### Strategy 2: Normalized Comparison
- **Use Case**: Notebooks with timestamps, whitespace variations
- **Behavior**: Normalize outputs before comparison
  - Strip leading/trailing whitespace
  - Normalize line endings (CRLF → LF)
  - Remove timestamps (configurable patterns)
  - Ignore execution counts
- **Pros**: Handles common non-deterministic elements
- **Cons**: May miss significant whitespace changes

#### Strategy 3: Fuzzy Numeric Comparison
- **Use Case**: Notebooks with floating-point calculations
- **Behavior**: Compare numeric values with tolerance
  - Absolute tolerance: `|a - b| < epsilon`
  - Relative tolerance: `|a - b| / max(|a|, |b|) < epsilon`
  - Default epsilon: `1e-6` (configurable)
- **Pros**: Handles floating-point precision issues
- **Cons**: Requires parsing numeric values from text

#### Strategy 4: Semantic Comparison (Future)
- **Use Case**: Notebooks with complex outputs (plots, DataFrames)
- **Behavior**: Compare semantic meaning, not exact representation
  - Image comparison (perceptual hash, SSIM)
  - DataFrame comparison (schema + values)
  - JSON comparison (structure + values)
- **Pros**: Most flexible, handles complex outputs
- **Cons**: Complex implementation, slower

### 2. Comparison Configuration

Users can configure comparison behavior via CRD annotations:

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: my-validation
  annotations:
    mlops.dev/comparison-strategy: "normalized"  # exact, normalized, fuzzy, semantic
    mlops.dev/numeric-tolerance: "1e-6"          # For fuzzy strategy
    mlops.dev/ignore-timestamps: "true"          # Remove timestamp patterns
    mlops.dev/ignore-execution-counts: "true"    # Ignore cell execution counts
    mlops.dev/timestamp-patterns: "\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}"  # Custom regex
spec:
  notebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: analysis.ipynb
  goldenNotebook:
    git:
      url: https://github.com/example/notebooks.git
      ref: main
      path: golden/analysis-golden.ipynb
  podConfig:
    containerImage: quay.io/jupyter/scipy-notebook:latest
    serviceAccountName: jupyter-notebook-validator-runner
```

### 3. Diff Generation

Generate structured diffs for failed comparisons:

```json
{
  "comparisonResult": "failed",
  "strategy": "normalized",
  "totalCells": 10,
  "matchedCells": 8,
  "mismatchedCells": 2,
  "diffs": [
    {
      "cellIndex": 3,
      "cellType": "code",
      "diffType": "output_mismatch",
      "expected": "Result: 42.123456",
      "actual": "Result: 42.123457",
      "diff": "- Result: 42.123456\n+ Result: 42.123457",
      "severity": "minor"
    },
    {
      "cellIndex": 7,
      "cellType": "code",
      "diffType": "output_mismatch",
      "expected": "Error: Division by zero",
      "actual": "Success: 100",
      "diff": "- Error: Division by zero\n+ Success: 100",
      "severity": "major"
    }
  ]
}
```

### 4. Status Reporting

Update `NotebookValidationJobStatus` with comparison results:

```go
type NotebookValidationJobStatus struct {
    // ... existing fields ...
    
    // ComparisonResult contains the golden notebook comparison result
    // +optional
    ComparisonResult *ComparisonResult `json:"comparisonResult,omitempty"`
}

type ComparisonResult struct {
    // Strategy used for comparison
    Strategy string `json:"strategy"`
    
    // Result is the overall comparison result (matched, failed, skipped)
    Result string `json:"result"`
    
    // TotalCells is the total number of cells compared
    TotalCells int `json:"totalCells"`
    
    // MatchedCells is the number of cells that matched
    MatchedCells int `json:"matchedCells"`
    
    // MismatchedCells is the number of cells that did not match
    MismatchedCells int `json:"mismatchedCells"`
    
    // Diffs contains detailed diff information for mismatched cells
    // +optional
    Diffs []CellDiff `json:"diffs,omitempty"`
}

type CellDiff struct {
    // CellIndex is the index of the cell (0-based)
    CellIndex int `json:"cellIndex"`
    
    // CellType is the type of cell (code, markdown)
    CellType string `json:"cellType"`
    
    // DiffType describes the type of difference
    DiffType string `json:"diffType"`  // output_mismatch, execution_error, missing_cell
    
    // Expected is the expected output from golden notebook
    Expected string `json:"expected"`
    
    // Actual is the actual output from executed notebook
    Actual string `json:"actual"`
    
    // Diff is the unified diff format
    Diff string `json:"diff"`
    
    // Severity indicates the importance of the difference
    Severity string `json:"severity"`  // minor, major, critical
}
```

### 5. Implementation Phases

#### Phase 1: Exact Match (MVP)
- Implement exact byte-for-byte comparison
- Generate basic diffs for mismatched cells
- Update status with comparison results
- **Timeline**: Week 1

#### Phase 2: Normalized Comparison
- Implement whitespace normalization
- Implement timestamp removal (regex patterns)
- Implement execution count ignoring
- **Timeline**: Week 2

#### Phase 3: Fuzzy Numeric Comparison
- Parse numeric values from text outputs
- Implement tolerance-based comparison
- Handle scientific notation
- **Timeline**: Week 3

#### Phase 4: Semantic Comparison (Future)
- Image comparison (perceptual hash)
- DataFrame comparison
- JSON comparison
- **Timeline**: Future release

## Consequences

### Positive
- **Flexibility**: Multiple strategies support different use cases
- **Configurability**: Users can tune comparison behavior
- **Detailed Feedback**: Structured diffs help debug failures
- **Incremental Implementation**: Can start with simple exact match
- **Extensibility**: Easy to add new comparison strategies

### Negative
- **Complexity**: Multiple strategies increase code complexity
- **Configuration Burden**: Users must understand which strategy to use
- **Performance**: Complex strategies (fuzzy, semantic) are slower
- **False Positives**: Overly strict comparison may fail on acceptable differences
- **False Negatives**: Overly lenient comparison may miss real issues

### Neutral
- **Default Behavior**: Exact match is safe default, users opt-in to fuzzy matching
- **Annotation-Based Config**: Keeps CRD spec clean, but requires annotation knowledge
- **Diff Format**: Unified diff is standard but may be verbose for large outputs

## Implementation Notes

### Comparison Algorithm (Exact Match)

```go
func compareNotebooks(executed, golden *nbformat.Notebook) (*ComparisonResult, error) {
    result := &ComparisonResult{
        Strategy:   "exact",
        TotalCells: len(executed.Cells),
    }
    
    for i, execCell := range executed.Cells {
        if i >= len(golden.Cells) {
            result.Diffs = append(result.Diffs, CellDiff{
                CellIndex: i,
                DiffType:  "missing_cell",
                Severity:  "major",
            })
            continue
        }
        
        goldenCell := golden.Cells[i]
        
        if !cellOutputsMatch(execCell, goldenCell) {
            result.MismatchedCells++
            result.Diffs = append(result.Diffs, generateCellDiff(i, execCell, goldenCell))
        } else {
            result.MatchedCells++
        }
    }
    
    if result.MismatchedCells == 0 {
        result.Result = "matched"
    } else {
        result.Result = "failed"
    }
    
    return result, nil
}
```

### Timestamp Removal (Normalized Strategy)

```go
var defaultTimestampPatterns = []string{
    `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`,           // ISO 8601
    `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,           // Common format
    `\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`,           // US format
    `Execution time: \d+\.\d+s`,                      // Execution time
    `Duration: \d+ms`,                                // Duration
}

func normalizeOutput(output string, patterns []string) string {
    normalized := output
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        normalized = re.ReplaceAllString(normalized, "[TIMESTAMP]")
    }
    return strings.TrimSpace(normalized)
}
```

## References

- [nbformat Documentation](https://nbformat.readthedocs.io/)
- [nbdime - Jupyter Notebook Diff and Merge](https://nbdime.readthedocs.io/)
- [pytest-notebook](https://pytest-notebook.readthedocs.io/)
- [Unified Diff Format](https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html)
- [Perceptual Image Hashing](https://www.phash.org/)

## Related ADRs

- ADR 003: CRD Schema Design (defines NotebookValidationJob spec)
- ADR 006: Notebook Execution Strategy (defines Papermill integration)
- ADR 008: Notebook Testing Strategy (defines test tiers and golden notebooks)
- ADR 010: Observability and Monitoring (defines status reporting)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-08 | Team   | Initial output comparison strategy |

