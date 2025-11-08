package controller

import (
	"testing"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// TestCompareNotebooks_ExactMatch tests exact match comparison
func TestCompareNotebooks_ExactMatch(t *testing.T) {
	// Create two identical notebooks
	executed := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
			{
				CellType: "markdown",
				Source:   "# Test Notebook",
			},
		},
	}

	golden := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
			{
				CellType: "markdown",
				Source:   "# Test Notebook",
			},
		},
	}

	config := ComparisonConfig{
		Strategy: "exact",
	}

	result := compareNotebooks(executed, golden, config)

	if result.Result != "matched" {
		t.Errorf("Expected result 'matched', got '%s'", result.Result)
	}

	if result.MatchedCells != 2 {
		t.Errorf("Expected 2 matched cells, got %d", result.MatchedCells)
	}

	if result.MismatchedCells != 0 {
		t.Errorf("Expected 0 mismatched cells, got %d", result.MismatchedCells)
	}

	if len(result.Diffs) != 0 {
		t.Errorf("Expected 0 diffs, got %d", len(result.Diffs))
	}
}

// TestCompareNotebooks_Mismatch tests notebooks with differences
func TestCompareNotebooks_Mismatch(t *testing.T) {
	executed := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
		},
	}

	golden := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Goodbye, World!\n",
					},
				},
			},
		},
	}

	config := ComparisonConfig{
		Strategy: "exact",
	}

	result := compareNotebooks(executed, golden, config)

	if result.Result != "failed" {
		t.Errorf("Expected result 'failed', got '%s'", result.Result)
	}

	if result.MatchedCells != 0 {
		t.Errorf("Expected 0 matched cells, got %d", result.MatchedCells)
	}

	if result.MismatchedCells != 1 {
		t.Errorf("Expected 1 mismatched cell, got %d", result.MismatchedCells)
	}

	if len(result.Diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(result.Diffs))
	}
}

// TestCompareNotebooks_ExtraCell tests notebooks with extra cells
func TestCompareNotebooks_ExtraCell(t *testing.T) {
	executed := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
			{
				CellType:       "code",
				ExecutionCount: intPtr(2),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Extra cell\n",
					},
				},
			},
		},
	}

	golden := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
		},
	}

	config := ComparisonConfig{
		Strategy: "exact",
	}

	result := compareNotebooks(executed, golden, config)

	if result.Result != "failed" {
		t.Errorf("Expected result 'failed', got '%s'", result.Result)
	}

	if result.MatchedCells != 1 {
		t.Errorf("Expected 1 matched cell, got %d", result.MatchedCells)
	}

	if result.MismatchedCells != 1 {
		t.Errorf("Expected 1 mismatched cell, got %d", result.MismatchedCells)
	}

	if len(result.Diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(result.Diffs))
	}

	if result.Diffs[0].DiffType != "extra_cell" {
		t.Errorf("Expected diff type 'extra_cell', got '%s'", result.Diffs[0].DiffType)
	}
}

// TestCompareNotebooks_MissingCell tests notebooks with missing cells
func TestCompareNotebooks_MissingCell(t *testing.T) {
	executed := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
		},
	}

	golden := &NotebookFormat{
		Cells: []NotebookCell{
			{
				CellType:       "code",
				ExecutionCount: intPtr(1),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Hello, World!\n",
					},
				},
			},
			{
				CellType:       "code",
				ExecutionCount: intPtr(2),
				Outputs: []CellOutput{
					{
						OutputType: "stream",
						Text:       "Missing cell\n",
					},
				},
			},
		},
	}

	config := ComparisonConfig{
		Strategy: "exact",
	}

	result := compareNotebooks(executed, golden, config)

	if result.Result != "failed" {
		t.Errorf("Expected result 'failed', got '%s'", result.Result)
	}

	if result.MatchedCells != 1 {
		t.Errorf("Expected 1 matched cell, got %d", result.MatchedCells)
	}

	if result.MismatchedCells != 1 {
		t.Errorf("Expected 1 mismatched cell, got %d", result.MismatchedCells)
	}

	if len(result.Diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(result.Diffs))
	}

	if result.Diffs[0].DiffType != "missing_cell" {
		t.Errorf("Expected diff type 'missing_cell', got '%s'", result.Diffs[0].DiffType)
	}
}

// TestGetComparisonConfig tests comparison configuration from spec
func TestGetComparisonConfig(t *testing.T) {
	// Test default config
	job := &mlopsv1alpha1.NotebookValidationJob{}
	config := getComparisonConfig(job)

	if config.Strategy != "normalized" {
		t.Errorf("Expected default strategy 'normalized', got '%s'", config.Strategy)
	}

	if config.NumericTolerance != 0.0001 {
		t.Errorf("Expected default tolerance 0.0001, got %f", config.NumericTolerance)
	}

	if !config.IgnoreTimestamps {
		t.Error("Expected IgnoreTimestamps to be true by default")
	}

	// Test with spec configuration
	exactStrategy := "exact"
	tolerance := "0.001"
	ignoreTS := false
	ignoreEC := false

	job.Spec.ComparisonConfig = &mlopsv1alpha1.ComparisonConfigSpec{
		Strategy:                exactStrategy,
		FloatingPointTolerance:  &tolerance,
		IgnoreTimestamps:        &ignoreTS,
		IgnoreExecutionCount:    &ignoreEC,
		CustomTimestampPatterns: []string{`\d{4}-\d{2}-\d{2}`},
	}
	config = getComparisonConfig(job)

	if config.Strategy != "exact" {
		t.Errorf("Expected strategy 'exact', got '%s'", config.Strategy)
	}

	if config.NumericTolerance != 0.001 {
		t.Errorf("Expected tolerance 0.001, got %f", config.NumericTolerance)
	}

	if config.IgnoreTimestamps {
		t.Error("Expected IgnoreTimestamps to be false")
	}

	if config.IgnoreExecutionCount {
		t.Error("Expected IgnoreExecutionCount to be false")
	}

	// Check custom timestamp patterns were added
	found := false
	for _, pattern := range config.TimestampPatterns {
		if pattern == `\d{4}-\d{2}-\d{2}` {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected custom timestamp pattern to be added")
	}
}

// TestNormalizeFloatingPoint tests floating-point normalization
func TestNormalizeFloatingPoint(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tolerance float64
		expected  string
	}{
		{
			name:      "Simple float with tolerance 0.0001",
			input:     "Result: 3.14159265",
			tolerance: 0.0001,
			expected:  "Result: 3.1416",
		},
		{
			name:      "Multiple floats",
			input:     "Values: 1.23456, 7.89012",
			tolerance: 0.001,
			expected:  "Values: 1.235, 7.890",
		},
		{
			name:      "Scientific notation",
			input:     "Value: 1.23e-5",
			tolerance: 0.0001,
			expected:  "Value: 0.0000",
		},
		{
			name:      "Negative numbers",
			input:     "Error: -0.00123",
			tolerance: 0.0001,
			expected:  "Error: -0.0012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFloatingPoint(tt.input, tt.tolerance)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestNormalizeOutput tests output normalization with all features
func TestNormalizeOutput(t *testing.T) {
	config := ComparisonConfig{
		Strategy:          "normalized",
		NumericTolerance:  0.001,
		IgnoreTimestamps:  true,
		TimestampPatterns: defaultTimestampPatterns,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Timestamp removal",
			input:    "Execution time: 2024-01-15T10:30:00",
			expected: "Execution time: [TIMESTAMP]",
		},
		{
			name:     "Float normalization",
			input:    "Accuracy: 0.95678",
			expected: "Accuracy: 0.957",
		},
		{
			name:     "Whitespace normalization",
			input:    "Result:   multiple    spaces",
			expected: "Result: multiple spaces",
		},
		{
			name:     "Combined normalization",
			input:    "Time: 2024-01-15 10:30:00, Accuracy: 0.95678",
			expected: "Time: [TIMESTAMP], Accuracy: 0.957",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeOutput(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
