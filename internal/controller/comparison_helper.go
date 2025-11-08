package controller

import (
	"fmt"
	"regexp"
	"strings"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// NotebookFormat represents a simplified Jupyter notebook structure
type NotebookFormat struct {
	Cells []NotebookCell `json:"cells"`
}

// NotebookCell represents a single cell in a Jupyter notebook
type NotebookCell struct {
	CellType       string                 `json:"cell_type"`
	ExecutionCount *int                   `json:"execution_count,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Outputs        []CellOutput           `json:"outputs,omitempty"`
	Source         interface{}            `json:"source,omitempty"`
}

// CellOutput represents the output of a notebook cell
type CellOutput struct {
	OutputType     string                 `json:"output_type"`
	Text           interface{}            `json:"text,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	ExecutionCount *int                   `json:"execution_count,omitempty"`
	Name           string                 `json:"name,omitempty"`
	Traceback      []string               `json:"traceback,omitempty"`
	Ename          string                 `json:"ename,omitempty"`
	Evalue         string                 `json:"evalue,omitempty"`
}

// ComparisonConfig holds configuration for notebook comparison
type ComparisonConfig struct {
	Strategy             string
	NumericTolerance     float64
	IgnoreTimestamps     bool
	IgnoreExecutionCount bool
	TimestampPatterns    []string
}

// Default timestamp patterns to ignore
var defaultTimestampPatterns = []string{
	`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, // ISO 8601
	`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, // Common format
	`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`, // US format
	`Execution time: \d+\.\d+s`,           // Execution time
	`Duration: \d+ms`,                     // Duration
	`\d+\.\d+s`,                           // Simple duration
	`\d{10,13}`,                           // Unix timestamp
}

// compareNotebooks compares executed notebook with golden notebook
func compareNotebooks(executed, golden *NotebookFormat, config ComparisonConfig) *mlopsv1alpha1.ComparisonResult {
	result := &mlopsv1alpha1.ComparisonResult{
		Strategy:   config.Strategy,
		TotalCells: len(executed.Cells),
		Diffs:      []mlopsv1alpha1.CellDiff{},
	}

	// Compare each cell
	for i, execCell := range executed.Cells {
		// Check if golden notebook has this cell
		if i >= len(golden.Cells) {
			result.MismatchedCells++
			result.Diffs = append(result.Diffs, mlopsv1alpha1.CellDiff{
				CellIndex: i,
				CellType:  execCell.CellType,
				DiffType:  "extra_cell",
				Actual:    fmt.Sprintf("Cell %d exists in executed notebook but not in golden", i),
				Severity:  "major",
			})
			continue
		}

		goldenCell := golden.Cells[i]

		// Compare cell outputs based on strategy
		if cellOutputsMatch(execCell, goldenCell, config) {
			result.MatchedCells++
		} else {
			result.MismatchedCells++
			diff := generateCellDiff(i, execCell, goldenCell, config)
			result.Diffs = append(result.Diffs, diff)
		}
	}

	// Check for missing cells in executed notebook
	if len(golden.Cells) > len(executed.Cells) {
		for i := len(executed.Cells); i < len(golden.Cells); i++ {
			result.MismatchedCells++
			result.Diffs = append(result.Diffs, mlopsv1alpha1.CellDiff{
				CellIndex: i,
				CellType:  golden.Cells[i].CellType,
				DiffType:  "missing_cell",
				Expected:  fmt.Sprintf("Cell %d exists in golden notebook but not in executed", i),
				Severity:  "major",
			})
		}
	}

	// Determine overall result
	if result.MismatchedCells == 0 {
		result.Result = "matched"
	} else {
		result.Result = "failed"
	}

	return result
}

// cellOutputsMatch checks if two cells have matching outputs
func cellOutputsMatch(exec, golden NotebookCell, config ComparisonConfig) bool {
	// If cell types don't match, cells don't match
	if exec.CellType != golden.CellType {
		return false
	}

	// For markdown cells, compare source
	if exec.CellType == "markdown" {
		return cellSourcesMatch(exec, golden, config)
	}

	// For code cells, compare outputs
	if len(exec.Outputs) != len(golden.Outputs) {
		return false
	}

	for i := range exec.Outputs {
		if !outputsMatch(exec.Outputs[i], golden.Outputs[i], config) {
			return false
		}
	}

	return true
}

// cellSourcesMatch compares cell source content
func cellSourcesMatch(exec, golden NotebookCell, config ComparisonConfig) bool {
	execSource := formatSource(exec.Source)
	goldenSource := formatSource(golden.Source)

	if config.Strategy == "normalized" {
		execSource = normalizeOutput(execSource, config)
		goldenSource = normalizeOutput(goldenSource, config)
	}

	return execSource == goldenSource
}

// outputsMatch compares two cell outputs
func outputsMatch(exec, golden CellOutput, config ComparisonConfig) bool {
	// Output types must match
	if exec.OutputType != golden.OutputType {
		return false
	}

	// Compare text output
	execText := formatOutputText(exec)
	goldenText := formatOutputText(golden)

	if config.Strategy == "normalized" {
		execText = normalizeOutput(execText, config)
		goldenText = normalizeOutput(goldenText, config)
	}

	return execText == goldenText
}

// formatSource converts source (string or []string) to a single string
func formatSource(source interface{}) string {
	switch v := source.(type) {
	case string:
		return v
	case []interface{}:
		var lines []string
		for _, line := range v {
			if str, ok := line.(string); ok {
				lines = append(lines, str)
			}
		}
		return strings.Join(lines, "")
	default:
		return fmt.Sprintf("%v", source)
	}
}

// formatOutputText extracts text from cell output
func formatOutputText(output CellOutput) string {
	if output.Text != nil {
		switch v := output.Text.(type) {
		case string:
			return v
		case []interface{}:
			var lines []string
			for _, line := range v {
				if str, ok := line.(string); ok {
					lines = append(lines, str)
				}
			}
			return strings.Join(lines, "")
		}
	}

	// Handle error outputs
	if len(output.Traceback) > 0 {
		return strings.Join(output.Traceback, "\n")
	}

	if output.Ename != "" {
		return fmt.Sprintf("%s: %s", output.Ename, output.Evalue)
	}

	return ""
}

// normalizeOutput normalizes output text for comparison
func normalizeOutput(output string, config ComparisonConfig) string {
	normalized := output

	// Remove timestamps if configured
	if config.IgnoreTimestamps {
		patterns := config.TimestampPatterns
		if len(patterns) == 0 {
			patterns = defaultTimestampPatterns
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			normalized = re.ReplaceAllString(normalized, "[TIMESTAMP]")
		}
	}

	// Apply floating-point tolerance if configured
	if config.NumericTolerance > 0 {
		normalized = normalizeFloatingPoint(normalized, config.NumericTolerance)
	}

	// Normalize whitespace
	normalized = strings.TrimSpace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return normalized
}

// normalizeFloatingPoint normalizes floating-point numbers in output for tolerance-based comparison
func normalizeFloatingPoint(output string, tolerance float64) string {
	// Pattern to match floating-point numbers
	floatPattern := regexp.MustCompile(`-?\d+\.\d+([eE][+-]?\d+)?`)

	return floatPattern.ReplaceAllStringFunc(output, func(match string) string {
		// Parse the float
		var f float64
		if _, err := fmt.Sscanf(match, "%f", &f); err != nil {
			return match // Return original if parsing fails
		}

		// Round to tolerance precision
		// For tolerance 0.0001, round to 4 decimal places
		precision := 0
		temp := tolerance
		for temp < 1.0 && precision < 10 {
			temp *= 10
			precision++
		}

		// Format with appropriate precision
		format := fmt.Sprintf("%%.%df", precision)
		return fmt.Sprintf(format, f)
	})
}

// generateCellDiff generates a diff for a mismatched cell
func generateCellDiff(index int, exec, golden NotebookCell, config ComparisonConfig) mlopsv1alpha1.CellDiff {
	diff := mlopsv1alpha1.CellDiff{
		CellIndex: index,
		CellType:  exec.CellType,
		DiffType:  "output_mismatch",
		Severity:  "major",
	}

	// Get expected and actual outputs
	if exec.CellType == "markdown" {
		diff.Expected = truncateString(formatSource(golden.Source), 500)
		diff.Actual = truncateString(formatSource(exec.Source), 500)
	} else {
		diff.Expected = truncateString(formatCellOutputs(golden.Outputs), 500)
		diff.Actual = truncateString(formatCellOutputs(exec.Outputs), 500)
	}

	// Generate unified diff
	diff.Diff = generateUnifiedDiff(diff.Expected, diff.Actual)

	return diff
}

// formatCellOutputs formats all outputs of a cell into a single string
func formatCellOutputs(outputs []CellOutput) string {
	var result []string
	for _, output := range outputs {
		text := formatOutputText(output)
		if text != "" {
			result = append(result, text)
		}
	}
	return strings.Join(result, "\n")
}

// generateUnifiedDiff generates a simple unified diff
func generateUnifiedDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff []string
	diff = append(diff, "--- expected")
	diff = append(diff, "+++ actual")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		if i < len(expectedLines) && i < len(actualLines) {
			if expectedLines[i] != actualLines[i] {
				diff = append(diff, fmt.Sprintf("- %s", expectedLines[i]))
				diff = append(diff, fmt.Sprintf("+ %s", actualLines[i]))
			}
		} else if i < len(expectedLines) {
			diff = append(diff, fmt.Sprintf("- %s", expectedLines[i]))
		} else {
			diff = append(diff, fmt.Sprintf("+ %s", actualLines[i]))
		}
	}

	return truncateString(strings.Join(diff, "\n"), 1000)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// getComparisonConfig extracts comparison configuration from job spec
func getComparisonConfig(job *mlopsv1alpha1.NotebookValidationJob) ComparisonConfig {
	// Default configuration
	config := ComparisonConfig{
		Strategy:             "normalized", // Default to normalized for better UX
		NumericTolerance:     0.0001,       // Default tolerance
		IgnoreTimestamps:     true,
		IgnoreExecutionCount: true,
		TimestampPatterns:    defaultTimestampPatterns,
	}

	// If no comparison config specified, return defaults
	if job.Spec.ComparisonConfig == nil {
		return config
	}

	compConfig := job.Spec.ComparisonConfig

	// Read strategy from spec
	if compConfig.Strategy != "" {
		config.Strategy = compConfig.Strategy
	}

	// Read floating-point tolerance from spec
	if compConfig.FloatingPointTolerance != nil {
		// Parse string to float64
		if tolerance, err := parseFloat(*compConfig.FloatingPointTolerance); err == nil {
			config.NumericTolerance = tolerance
		}
	}

	// Read ignore-timestamps from spec
	if compConfig.IgnoreTimestamps != nil {
		config.IgnoreTimestamps = *compConfig.IgnoreTimestamps
	}

	// Read ignore-execution-count from spec
	if compConfig.IgnoreExecutionCount != nil {
		config.IgnoreExecutionCount = *compConfig.IgnoreExecutionCount
	}

	// Add custom timestamp patterns
	if len(compConfig.CustomTimestampPatterns) > 0 {
		config.TimestampPatterns = append(config.TimestampPatterns, compConfig.CustomTimestampPatterns...)
	}

	return config
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
