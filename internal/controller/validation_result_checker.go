package controller

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// ADR-041 severity constants for validation issues.
const (
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// ValidationIssue represents a single runtime validation issue collected from
// ADR-041 instrumentation.
type ValidationIssue struct {
	Cell       int    `json:"cell"`
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ValidationCheckResult holds the aggregated result of all post-execution checks.
type ValidationCheckResult struct {
	Passed              bool                                    `json:"passed"`
	TotalIssues         int                                     `json:"totalIssues"`
	BlockingIssues      int                                     `json:"blockingIssues"`
	Issues              []ValidationIssue                       `json:"issues"`
	EducationalFeedback []mlopsv1alpha1.EducationalFeedbackItem `json:"educationalFeedback,omitempty"`
}

// ParseRuntimeIssues extracts ADR-041 validation issues from pod log output.
// The instrumenter embeds a JSON array prefixed with "ADR041_VALIDATION_RESULTS=".
func ParseRuntimeIssues(logOutput string) ([]ValidationIssue, error) {
	const marker = "ADR041_VALIDATION_RESULTS="
	idx := strings.Index(logOutput, marker)
	if idx < 0 {
		return nil, nil
	}

	jsonStart := idx + len(marker)
	// Find the end of the JSON array (next newline or end of string)
	jsonEnd := strings.IndexByte(logOutput[jsonStart:], '\n')
	var jsonStr string
	if jsonEnd < 0 {
		jsonStr = logOutput[jsonStart:]
	} else {
		jsonStr = logOutput[jsonStart : jsonStart+jsonEnd]
	}
	jsonStr = strings.TrimSpace(jsonStr)

	var issues []ValidationIssue
	if err := json.Unmarshal([]byte(jsonStr), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse ADR-041 validation results: %w", err)
	}
	return issues, nil
}

// CheckExpectedOutputs validates cell outputs against the expected output specifications
// defined in the NotebookValidationJob spec.
func CheckExpectedOutputs(
	results []mlopsv1alpha1.CellResult,
	specs []mlopsv1alpha1.ExpectedOutputSpec,
) []ValidationIssue {
	var issues []ValidationIssue

	for _, spec := range specs {
		if spec.Cell < 0 || spec.Cell >= len(results) {
			issues = append(issues, ValidationIssue{
				Cell:     spec.Cell,
				Severity: SeverityError,
				Category: "invalid-spec",
				Message:  fmt.Sprintf("cell index %d is out of range (notebook has %d cells)", spec.Cell, len(results)),
			})
			continue
		}

		cellResult := results[spec.Cell]

		// Check NotEmpty
		if spec.NotEmpty && isOutputEmpty(cellResult) {
			issues = append(issues, ValidationIssue{
				Cell:       spec.Cell,
				Severity:   SeverityError,
				Category:   "empty-output",
				Message:    fmt.Sprintf("cell %d output is empty but notEmpty was specified", spec.Cell),
				Suggestion: "Ensure the cell produces output (print, display, or return a value)",
			})
		}

		// Check MinValue / MaxValue for numeric outputs
		if spec.MinValue != "" || spec.MaxValue != "" {
			numericIssues := checkNumericRange(spec, cellResult)
			issues = append(issues, numericIssues...)
		}

		// Check Shape
		if spec.Shape != "" {
			shapeIssues := checkShape(spec, cellResult)
			issues = append(issues, shapeIssues...)
		}
	}

	return issues
}

// CheckValidationResults is the main entry point for post-execution validation.
// It combines runtime issues from instrumentation with expected-output checks and
// applies the configured validation level.
func CheckValidationResults(
	job *mlopsv1alpha1.NotebookValidationJob,
	cellResults []mlopsv1alpha1.CellResult,
	logOutput string,
) (*ValidationCheckResult, error) {
	level := "development"
	if job.Spec.ValidationConfig != nil && job.Spec.ValidationConfig.Level != "" {
		level = job.Spec.ValidationConfig.Level
	}

	// Collect runtime issues from instrumentation
	runtimeIssues, err := ParseRuntimeIssues(logOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse runtime issues: %w", err)
	}

	// Collect expected-output issues
	var specIssues []ValidationIssue
	if job.Spec.ValidationConfig != nil && len(job.Spec.ValidationConfig.ExpectedOutputs) > 0 {
		specIssues = CheckExpectedOutputs(cellResults, job.Spec.ValidationConfig.ExpectedOutputs)
	}

	allIssues := append(runtimeIssues, specIssues...)

	// Apply level-based filtering
	blockingIssues := filterBlockingIssues(allIssues, level)

	// Generate educational feedback
	var feedback []mlopsv1alpha1.EducationalFeedbackItem
	if shouldGenerateFeedback(job) {
		feedback = generateFeedback(allIssues, level)
	}

	return &ValidationCheckResult{
		Passed:              len(blockingIssues) == 0,
		TotalIssues:         len(allIssues),
		BlockingIssues:      len(blockingIssues),
		Issues:              allIssues,
		EducationalFeedback: feedback,
	}, nil
}

// filterBlockingIssues returns only the issues that are blocking for the given level.
func filterBlockingIssues(issues []ValidationIssue, level string) []ValidationIssue {
	threshold := severityThreshold(level)
	var blocking []ValidationIssue
	for _, issue := range issues {
		if severityValue(issue.Severity) >= threshold {
			blocking = append(blocking, issue)
		}
	}
	return blocking
}

func severityThreshold(level string) int {
	switch level {
	case "learning":
		return severityValue(SeverityError)
	case "development":
		return severityValue(SeverityWarning)
	case "staging":
		return severityValue(SeverityWarning)
	case "production":
		return severityValue(SeverityInfo)
	default:
		return severityValue(SeverityWarning)
	}
}

func severityValue(severity string) int {
	switch severity {
	case SeverityInfo:
		return 0
	case SeverityWarning:
		return 1
	case SeverityError:
		return 2
	default:
		return 0
	}
}

func shouldGenerateFeedback(job *mlopsv1alpha1.NotebookValidationJob) bool {
	if job.Spec.ValidationConfig == nil {
		return false
	}
	if job.Spec.ValidationConfig.EducationalMode {
		return true
	}
	return job.Spec.ValidationConfig.Level == "learning"
}

func generateFeedback(issues []ValidationIssue, level string) []mlopsv1alpha1.EducationalFeedbackItem {
	feedback := make([]mlopsv1alpha1.EducationalFeedbackItem, 0, len(issues))
	for _, issue := range issues {
		item := mlopsv1alpha1.EducationalFeedbackItem{
			Cell:     issue.Cell,
			Severity: issue.Severity,
			Category: issue.Category,
			Message:  issue.Message,
		}
		if issue.Suggestion != "" {
			item.Suggestion = issue.Suggestion
		}
		// Add level-specific context
		if level == "learning" {
			item.Suggestion = addLearningContext(item)
		}
		feedback = append(feedback, item)
	}
	return feedback
}

func addLearningContext(item mlopsv1alpha1.EducationalFeedbackItem) string {
	base := item.Suggestion
	switch item.Category {
	case "nan-detected":
		return base + "\n\n💡 NaN (Not a Number) values often appear when data is missing or after invalid " +
			"operations like dividing by zero. Use df.isna().sum() to check how many NaN values exist."
	case "none-value":
		return base + "\n\n💡 None usually means a function didn't return a value, or an assignment failed silently. " +
			"Check that the previous operation completed successfully."
	case "silent-failure":
		return base + "\n\n💡 Silent failures are bugs that don't raise exceptions but produce wrong results. " +
			"Adding assert statements and explicit return values helps catch them early."
	case "empty-output":
		return base + "\n\n💡 An empty output may indicate that a filter removed all data, or the data source is empty. " +
			"Verify your data pipeline step by step."
	default:
		return base
	}
}

// isOutputEmpty checks if a cell result has no meaningful output.
func isOutputEmpty(result mlopsv1alpha1.CellResult) bool {
	return strings.TrimSpace(result.Output) == ""
}

// checkNumericRange validates that numeric outputs are within the expected range.
func checkNumericRange(spec mlopsv1alpha1.ExpectedOutputSpec, result mlopsv1alpha1.CellResult) []ValidationIssue {
	var issues []ValidationIssue
	output := strings.TrimSpace(result.Output)
	if output == "" {
		return issues
	}

	val, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return issues
	}

	if math.IsNaN(val) {
		issues = append(issues, ValidationIssue{
			Cell:       spec.Cell,
			Severity:   SeverityError,
			Category:   "nan-detected",
			Message:    fmt.Sprintf("cell %d output is NaN", spec.Cell),
			Suggestion: "Check computation for division by zero or invalid operations",
		})
		return issues
	}

	if spec.MinValue != "" {
		minVal, err := strconv.ParseFloat(spec.MinValue, 64)
		if err == nil && val < minVal {
			issues = append(issues, ValidationIssue{
				Cell:       spec.Cell,
				Severity:   SeverityError,
				Category:   "out-of-range",
				Message:    fmt.Sprintf("cell %d output %.6g is below minimum %s", spec.Cell, val, spec.MinValue),
				Suggestion: "Verify the computation produces values in the expected range",
			})
		}
	}

	if spec.MaxValue != "" {
		maxVal, err := strconv.ParseFloat(spec.MaxValue, 64)
		if err == nil && val > maxVal {
			issues = append(issues, ValidationIssue{
				Cell:       spec.Cell,
				Severity:   SeverityError,
				Category:   "out-of-range",
				Message:    fmt.Sprintf("cell %d output %.6g exceeds maximum %s", spec.Cell, val, spec.MaxValue),
				Suggestion: "Verify the computation produces values in the expected range",
			})
		}
	}

	return issues
}

// checkShape validates output dimensions against the expected shape spec.
// Shape format: "rows,cols" where -1 means any dimension.
func checkShape(spec mlopsv1alpha1.ExpectedOutputSpec, result mlopsv1alpha1.CellResult) []ValidationIssue {
	var issues []ValidationIssue
	output := strings.TrimSpace(result.Output)
	if output == "" {
		return issues
	}

	// Parse expected shape
	parts := strings.Split(spec.Shape, ",")
	if len(parts) != 2 {
		return issues
	}

	expectedRows, errR := strconv.Atoi(strings.TrimSpace(parts[0]))
	expectedCols, errC := strconv.Atoi(strings.TrimSpace(parts[1]))

	// Look for shape info in the output (e.g., "(100, 10)")
	if shapeStr := extractShapeFromOutput(output); shapeStr != "" {
		shapeParts := strings.Split(shapeStr, ",")
		if len(shapeParts) >= 1 {
			actualRows, err := strconv.Atoi(strings.TrimSpace(shapeParts[0]))
			if err == nil && errR == nil && expectedRows >= 0 && actualRows != expectedRows {
				issues = append(issues, ValidationIssue{
					Cell:       spec.Cell,
					Severity:   SeverityError,
					Category:   "shape-mismatch",
					Message:    fmt.Sprintf("cell %d output has %d rows, expected %d", spec.Cell, actualRows, expectedRows),
					Suggestion: "Check data filtering and transformation steps",
				})
			}
		}
		if len(shapeParts) >= 2 {
			actualCols, err := strconv.Atoi(strings.TrimSpace(shapeParts[1]))
			if err == nil && errC == nil && expectedCols >= 0 && actualCols != expectedCols {
				issues = append(issues, ValidationIssue{
					Cell:       spec.Cell,
					Severity:   SeverityError,
					Category:   "shape-mismatch",
					Message:    fmt.Sprintf("cell %d output has %d columns, expected %d", spec.Cell, actualCols, expectedCols),
					Suggestion: "Check column selection and data transformations",
				})
			}
		}
	}

	return issues
}

// extractShapeFromOutput attempts to find a shape tuple in the output string.
// Looks for patterns like "(100, 10)" or "shape: (100, 10)".
func extractShapeFromOutput(output string) string {
	// Find pattern like (N, M)
	start := strings.Index(output, "(")
	if start < 0 {
		return ""
	}
	end := strings.Index(output[start:], ")")
	if end < 0 {
		return ""
	}
	inner := output[start+1 : start+end]
	// Validate it looks like dimensions
	parts := strings.Split(inner, ",")
	if len(parts) < 2 {
		return ""
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(strings.TrimSpace(p)); err != nil {
			return ""
		}
	}
	return inner
}
