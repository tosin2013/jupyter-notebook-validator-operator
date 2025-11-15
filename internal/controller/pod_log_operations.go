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
	"context"

	corev1 "k8s.io/api/core/v1"
)

// PodLogOperations defines the interface for pod log collection and parsing
// This allows us to mock pod log operations in tests
type PodLogOperations interface {
	// CollectLogs retrieves logs from a pod
	CollectLogs(ctx context.Context, pod *corev1.Pod, containerName string) (string, error)
	
	// ParseResults parses execution results from logs
	ParseResults(logs string) (*NotebookExecutionResult, error)
	
	// ParseGoldenNotebook parses golden notebook data from logs
	ParseGoldenNotebook(logs string) (*NotebookFormat, error)
	
	// ExtractError extracts error messages from logs
	ExtractError(logs string) string
}

// podLogOperationsImpl implements PodLogOperations
type podLogOperationsImpl struct {
	reconciler *NotebookValidationJobReconciler
}

// NewPodLogOperations creates a new PodLogOperations implementation
func NewPodLogOperations(reconciler *NotebookValidationJobReconciler) PodLogOperations {
	return &podLogOperationsImpl{
		reconciler: reconciler,
	}
}

// CollectLogs implements PodLogOperations
func (p *podLogOperationsImpl) CollectLogs(ctx context.Context, pod *corev1.Pod, containerName string) (string, error) {
	return p.reconciler.collectPodLogs(ctx, pod, containerName)
}

// ParseResults implements PodLogOperations
func (p *podLogOperationsImpl) ParseResults(logs string) (*NotebookExecutionResult, error) {
	return parseResultsFromLogs(logs)
}

// ParseGoldenNotebook implements PodLogOperations
func (p *podLogOperationsImpl) ParseGoldenNotebook(logs string) (*NotebookFormat, error) {
	return parseGoldenNotebookFromLogs(logs)
}

// ExtractError implements PodLogOperations
func (p *podLogOperationsImpl) ExtractError(logs string) string {
	return extractErrorFromLogs(logs)
}
