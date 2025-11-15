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

package mocks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// NotebookExecutionResult represents parsed results from validation pod
type NotebookExecutionResult struct {
	Status                   string
	Error                    string
	ExitCode                 int
	NotebookPath             string
	ExecutionDurationSeconds int
	Timestamp                string
	Cells                    []CellExecutionResult
	Statistics               ExecutionStatistics
}

// CellExecutionResult represents a single cell's execution result
type CellExecutionResult struct {
	CellIndex      int
	CellType       string
	ExecutionCount *int
	Status         string
	Error          string
	Traceback      []string
}

// ExecutionStatistics contains summary statistics
type ExecutionStatistics struct {
	TotalCells  int
	CodeCells   int
	FailedCells int
	SuccessRate float64
}

// NotebookFormat represents notebook structure for comparison
type NotebookFormat struct {
	Cells []NotebookCell
}

// NotebookCell represents a notebook cell
type NotebookCell struct {
	CellType       string
	ExecutionCount *int
	Outputs        []CellOutput
}

// CellOutput represents cell output
type CellOutput struct {
	OutputType string
	Text       string
	Ename      string
	Evalue     string
	Traceback  []string
}

// PodLogOperations defines the interface for pod log collection and parsing
type PodLogOperations interface {
	CollectLogs(ctx context.Context, pod *corev1.Pod, containerName string) (string, error)
	ParseResults(logs string) (*NotebookExecutionResult, error)
	ParseGoldenNotebook(logs string) (*NotebookFormat, error)
	ExtractError(logs string) string
}

// MockPodLogOperations is a mock implementation of PodLogOperations for testing
type MockPodLogOperations struct {
	// CollectLogsFunc allows customizing the CollectLogs behavior
	CollectLogsFunc func(ctx context.Context, pod *corev1.Pod, containerName string) (string, error)

	// ParseResultsFunc allows customizing the ParseResults behavior
	ParseResultsFunc func(logs string) (*NotebookExecutionResult, error)

	// ParseGoldenNotebookFunc allows customizing the ParseGoldenNotebook behavior
	ParseGoldenNotebookFunc func(logs string) (*NotebookFormat, error)

	// ExtractErrorFunc allows customizing the ExtractError behavior
	ExtractErrorFunc func(logs string) string

	// Call tracking
	CollectLogsCallCount         int
	ParseResultsCallCount        int
	ParseGoldenNotebookCallCount int
	ExtractErrorCallCount        int
}

// NewMockPodLogOperations creates a new mock PodLogOperations with default behaviors
func NewMockPodLogOperations() *MockPodLogOperations {
	return &MockPodLogOperations{
		CollectLogsFunc: func(ctx context.Context, pod *corev1.Pod, containerName string) (string, error) {
			return "mock logs", nil
		},
		ParseResultsFunc: func(logs string) (*NotebookExecutionResult, error) {
			return &NotebookExecutionResult{
				Status:   "succeeded",
				ExitCode: 0,
			}, nil
		},
		ParseGoldenNotebookFunc: func(logs string) (*NotebookFormat, error) {
			return nil, nil
		},
		ExtractErrorFunc: func(logs string) string {
			return ""
		},
	}
}

// CollectLogs implements PodLogOperations
func (m *MockPodLogOperations) CollectLogs(ctx context.Context, pod *corev1.Pod, containerName string) (string, error) {
	m.CollectLogsCallCount++
	if m.CollectLogsFunc != nil {
		return m.CollectLogsFunc(ctx, pod, containerName)
	}
	return "", fmt.Errorf("CollectLogs not implemented")
}

// ParseResults implements PodLogOperations
func (m *MockPodLogOperations) ParseResults(logs string) (*NotebookExecutionResult, error) {
	m.ParseResultsCallCount++
	if m.ParseResultsFunc != nil {
		return m.ParseResultsFunc(logs)
	}
	return nil, fmt.Errorf("ParseResults not implemented")
}

// ParseGoldenNotebook implements PodLogOperations
func (m *MockPodLogOperations) ParseGoldenNotebook(logs string) (*NotebookFormat, error) {
	m.ParseGoldenNotebookCallCount++
	if m.ParseGoldenNotebookFunc != nil {
		return m.ParseGoldenNotebookFunc(logs)
	}
	return nil, nil
}

// ExtractError implements PodLogOperations
func (m *MockPodLogOperations) ExtractError(logs string) string {
	m.ExtractErrorCallCount++
	if m.ExtractErrorFunc != nil {
		return m.ExtractErrorFunc(logs)
	}
	return ""
}

// Reset resets all call counts
func (m *MockPodLogOperations) Reset() {
	m.CollectLogsCallCount = 0
	m.ParseResultsCallCount = 0
	m.ParseGoldenNotebookCallCount = 0
	m.ExtractErrorCallCount = 0
}

// VerifyCallCounts verifies that methods were called the expected number of times
func (m *MockPodLogOperations) VerifyCallCounts(t interface {
	Errorf(format string, args ...interface{})
}, expected map[string]int) {
	if expected["CollectLogs"] != m.CollectLogsCallCount {
		t.Errorf("CollectLogs called %d times, expected %d", m.CollectLogsCallCount, expected["CollectLogs"])
	}
	if expected["ParseResults"] != m.ParseResultsCallCount {
		t.Errorf("ParseResults called %d times, expected %d", m.ParseResultsCallCount, expected["ParseResults"])
	}
	if expected["ParseGoldenNotebook"] != m.ParseGoldenNotebookCallCount {
		t.Errorf("ParseGoldenNotebook called %d times, expected %d", m.ParseGoldenNotebookCallCount, expected["ParseGoldenNotebook"])
	}
	if expected["ExtractError"] != m.ExtractErrorCallCount {
		t.Errorf("ExtractError called %d times, expected %d", m.ExtractErrorCallCount, expected["ExtractError"])
	}
}
