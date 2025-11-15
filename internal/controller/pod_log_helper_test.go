/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"
)

func TestParseResultsFromLogs(t *testing.T) {
	tests := []struct {
		name        string
		logs        string
		expectError bool
		expected    *NotebookExecutionResult
	}{
		{
			name:        "empty logs",
			logs:        "",
			expectError: true,
		},
		{
			name:        "no results summary",
			logs:        "some log output\nmore logs",
			expectError: true,
		},
		{
			name: "valid results summary",
			logs: `some log output
Results Summary:
{
  "status": "succeeded",
  "error": "",
  "exit_code": 0,
  "notebook_path": "/workspace/repo/test.ipynb",
  "execution_duration_seconds": 10,
  "timestamp": "2025-01-01T00:00:00Z",
  "cells": [
    {
      "cell_index": 0,
      "cell_type": "code",
      "execution_count": 1,
      "status": "succeeded"
    }
  ],
  "statistics": {
    "total_cells": 1,
    "code_cells": 1,
    "failed_cells": 0,
    "success_rate": 100.0
  }
}
more logs after`,
			expectError: false,
			expected: &NotebookExecutionResult{
				Status:                   "succeeded",
				Error:                    "",
				ExitCode:                 0,
				NotebookPath:             "/workspace/repo/test.ipynb",
				ExecutionDurationSeconds: 10,
				Timestamp:                "2025-01-01T00:00:00Z",
				Cells: []CellExecutionResult{
					{
						CellIndex:      0,
						CellType:       "code",
						ExecutionCount: intPtr(1),
						Status:         "succeeded",
					},
				},
				Statistics: ExecutionStatistics{
					TotalCells:  1,
					CodeCells:   1,
					FailedCells: 0,
					SuccessRate: 100.0,
				},
			},
		},
		{
			name: "failed results",
			logs: `Results Summary:
{
  "status": "failed",
  "error": "Notebook execution failed",
  "exit_code": 1,
  "notebook_path": "/workspace/repo/test.ipynb",
  "execution_duration_seconds": 5,
  "timestamp": "2025-01-01T00:00:00Z",
  "cells": [],
  "statistics": {
    "total_cells": 0,
    "code_cells": 0,
    "failed_cells": 0,
    "success_rate": 0.0
  }
}`,
			expectError: false,
			expected: &NotebookExecutionResult{
				Status:                   "failed",
				Error:                    "Notebook execution failed",
				ExitCode:                 1,
				NotebookPath:             "/workspace/repo/test.ipynb",
				ExecutionDurationSeconds: 5,
				Timestamp:                "2025-01-01T00:00:00Z",
				Cells:                    []CellExecutionResult{},
				Statistics: ExecutionStatistics{
					TotalCells:  0,
					CodeCells:   0,
					FailedCells: 0,
					SuccessRate: 0.0,
				},
			},
		},
		{
			name: "invalid JSON",
			logs: `Results Summary:
{
  "status": "succeeded",
  invalid json
}`,
			expectError: true,
		},
		{
			name: "no JSON braces",
			logs: `Results Summary:
some text without JSON`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResultsFromLogs(tt.logs)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			if tt.expected != nil {
				if result.Status != tt.expected.Status {
					t.Errorf("Status: got %s, want %s", result.Status, tt.expected.Status)
				}
				if result.Error != tt.expected.Error {
					t.Errorf("Error: got %s, want %s", result.Error, tt.expected.Error)
				}
				if result.ExitCode != tt.expected.ExitCode {
					t.Errorf("ExitCode: got %d, want %d", result.ExitCode, tt.expected.ExitCode)
				}
				if result.NotebookPath != tt.expected.NotebookPath {
					t.Errorf("NotebookPath: got %s, want %s", result.NotebookPath, tt.expected.NotebookPath)
				}
				if len(result.Cells) != len(tt.expected.Cells) {
					t.Errorf("Cells length: got %d, want %d", len(result.Cells), len(tt.expected.Cells))
				}
			}
		})
	}
}

func TestParseGoldenNotebookFromLogs(t *testing.T) {
	tests := []struct {
		name        string
		logs        string
		expectNil   bool
		expectError bool
	}{
		{
			name:      "no golden notebook section",
			logs:      "some log output",
			expectNil: true,
		},
		{
			name: "valid golden notebook",
			logs: `some logs
Golden Notebook Summary:
{
  "cells": [
    {
      "cell_type": "code",
      "execution_count": 1,
      "outputs": []
    }
  ]
}
more logs`,
			expectNil:   false,
			expectError: false,
		},
		{
			name: "invalid JSON",
			logs: `Golden Notebook Summary:
{
  invalid json
}`,
			expectNil:   false,
			expectError: true,
		},
		{
			name: "empty golden notebook",
			logs: `Golden Notebook Summary:
{
  "cells": []
}`,
			expectNil:   false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseGoldenNotebookFromLogs(tt.logs)

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil but got result: %+v", result)
				}
				return
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil but expected non-nil")
			}
		})
	}
}

func TestConvertExecutionResultToNotebookFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    *NotebookExecutionResult
		expected *NotebookFormat
	}{
		{
			name:  "nil input",
			input: nil,
			expected: &NotebookFormat{
				Cells: []NotebookCell{},
			},
		},
		{
			name: "empty cells",
			input: &NotebookExecutionResult{
				Cells: []CellExecutionResult{},
			},
			expected: &NotebookFormat{
				Cells: []NotebookCell{},
			},
		},
		{
			name: "code cell with error",
			input: &NotebookExecutionResult{
				Cells: []CellExecutionResult{
					{
						CellIndex:      0,
						CellType:       "code",
						ExecutionCount: intPtr(1),
						Status:         "failed",
						Error:          "Test error",
						Traceback:      []string{"line1", "line2"},
					},
				},
			},
			expected: &NotebookFormat{
				Cells: []NotebookCell{
					{
						CellType:       "code",
						ExecutionCount: intPtr(1),
						Outputs: []CellOutput{
							{
								OutputType: "error",
								Ename:      "ExecutionError",
								Evalue:     "Test error",
								Traceback:  []string{"line1", "line2"},
							},
						},
					},
				},
			},
		},
		{
			name: "code cell without error",
			input: &NotebookExecutionResult{
				Cells: []CellExecutionResult{
					{
						CellIndex:      0,
						CellType:       "code",
						ExecutionCount: intPtr(1),
						Status:         "succeeded",
					},
				},
			},
			expected: &NotebookFormat{
				Cells: []NotebookCell{
					{
						CellType:       "code",
						ExecutionCount: intPtr(1),
						Outputs:        []CellOutput{},
					},
				},
			},
		},
		{
			name: "markdown cell",
			input: &NotebookExecutionResult{
				Cells: []CellExecutionResult{
					{
						CellIndex: 0,
						CellType:  "markdown",
					},
				},
			},
			expected: &NotebookFormat{
				Cells: []NotebookCell{
					{
						CellType:       "markdown",
						ExecutionCount: nil,
						Outputs:        []CellOutput{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertExecutionResultToNotebookFormat(tt.input)

			if result == nil {
				t.Fatal("Result is nil")
			}

			if len(result.Cells) != len(tt.expected.Cells) {
				t.Errorf("Cells length: got %d, want %d", len(result.Cells), len(tt.expected.Cells))
			}

			for i, expectedCell := range tt.expected.Cells {
				if i >= len(result.Cells) {
					t.Errorf("Missing cell at index %d", i)
					continue
				}

				actualCell := result.Cells[i]
				if actualCell.CellType != expectedCell.CellType {
					t.Errorf("Cell %d type: got %s, want %s", i, actualCell.CellType, expectedCell.CellType)
				}

				if len(actualCell.Outputs) != len(expectedCell.Outputs) {
					t.Errorf("Cell %d outputs length: got %d, want %d", i, len(actualCell.Outputs), len(expectedCell.Outputs))
				}

				for j, expectedOutput := range expectedCell.Outputs {
					if j >= len(actualCell.Outputs) {
						t.Errorf("Cell %d missing output at index %d", i, j)
						continue
					}

					actualOutput := actualCell.Outputs[j]
					if actualOutput.OutputType != expectedOutput.OutputType {
						t.Errorf("Cell %d output %d type: got %s, want %s", i, j, actualOutput.OutputType, expectedOutput.OutputType)
					}
					if actualOutput.Ename != expectedOutput.Ename {
						t.Errorf("Cell %d output %d ename: got %s, want %s", i, j, actualOutput.Ename, expectedOutput.Ename)
					}
					if actualOutput.Evalue != expectedOutput.Evalue {
						t.Errorf("Cell %d output %d evalue: got %s, want %s", i, j, actualOutput.Evalue, expectedOutput.Evalue)
					}
				}
			}
		})
	}
}

func TestExtractErrorFromLogs(t *testing.T) {
	tests := []struct {
		name     string
		logs     string
		expected string
	}{
		{
			name:     "empty logs",
			logs:     "",
			expected: "Validation failed (see pod logs for details)",
		},
		{
			name:     "no error patterns",
			logs:     "normal log output\nmore logs",
			expected: "Validation failed (see pod logs for details)",
		},
		{
			name: "single ERROR pattern",
			logs: `some logs
ERROR: Something went wrong
more logs`,
			expected: "ERROR: Something went wrong",
		},
		{
			name: "multiple error patterns",
			logs: `some logs
ERROR: First error
Error: Second error
FAILED: Third error
more logs`,
			expected: "ERROR: First error\nError: Second error\nFAILED: Third error",
		},
		{
			name: "more than 5 error lines",
			logs: `ERROR: Error 1
Error: Error 2
FAILED: Error 3
Exception: Error 4
Traceback: Error 5
ERROR: Error 6
Error: Error 7`,
			expected: "ERROR: Error 1\nError: Error 2\nFAILED: Error 3\nException: Error 4\nTraceback: Error 5",
		},
		{
			name:     "error with whitespace",
			logs:     `  ERROR:   Padded error  `,
			expected: "ERROR:   Padded error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorFromLogs(tt.logs)
			if result != tt.expected {
				t.Errorf("extractErrorFromLogs() = %q, want %q", result, tt.expected)
			}
		})
	}
}
