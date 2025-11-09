package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// NotebookExecutionResult represents the parsed results from the validation pod
type NotebookExecutionResult struct {
	Status                   string                `json:"status"`
	Error                    string                `json:"error"`
	ExitCode                 int                   `json:"exit_code"`
	NotebookPath             string                `json:"notebook_path"`
	ExecutionDurationSeconds int                   `json:"execution_duration_seconds"`
	Timestamp                string                `json:"timestamp"`
	Cells                    []CellExecutionResult `json:"cells"`
	Statistics               ExecutionStatistics   `json:"statistics"`
}

// CellExecutionResult represents a single cell's execution result
type CellExecutionResult struct {
	CellIndex      int      `json:"cell_index"`
	CellType       string   `json:"cell_type"`
	ExecutionCount *int     `json:"execution_count"`
	Status         string   `json:"status,omitempty"`
	Error          string   `json:"error,omitempty"`
	Traceback      []string `json:"traceback,omitempty"`
}

// ExecutionStatistics contains summary statistics
type ExecutionStatistics struct {
	TotalCells  int     `json:"total_cells"`
	CodeCells   int     `json:"code_cells"`
	FailedCells int     `json:"failed_cells"`
	SuccessRate float64 `json:"success_rate"`
}

// collectPodLogs retrieves logs from the validation pod using Kubernetes clientset
func (r *NotebookValidationJobReconciler) collectPodLogs(ctx context.Context, pod *corev1.Pod, containerName string) (string, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Collecting pod logs",
		"namespace", pod.Namespace,
		"pod", pod.Name,
		"container", containerName)

	// Use the REST config from the reconciler
	if r.RestConfig == nil {
		return "", fmt.Errorf("REST config is not available")
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(r.RestConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create clientset: %w", err)
	}

	// Set up pod log options
	podLogOpts := &corev1.PodLogOptions{
		Container: containerName,
	}

	// Get pod logs
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open log stream: %w", err)
	}
	defer podLogs.Close()

	// Read logs into buffer
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// parseResultsFromLogs extracts the results JSON from pod logs
func parseResultsFromLogs(logs string) (*NotebookExecutionResult, error) {
	// Look for the "Results Summary:" section in the logs
	// The JSON should be between "Results Summary:" and the end of logs

	// Find the JSON block in the logs
	jsonStartMarker := "Results Summary:"
	jsonStartIdx := strings.Index(logs, jsonStartMarker)
	if jsonStartIdx == -1 {
		return nil, fmt.Errorf("results summary not found in logs")
	}

	// Extract everything after the marker
	jsonSection := logs[jsonStartIdx+len(jsonStartMarker):]

	// Find the JSON object (starts with { and ends with })
	// Use a simple approach: find the first { and the last }
	firstBrace := strings.Index(jsonSection, "{")
	if firstBrace == -1 {
		return nil, fmt.Errorf("JSON start not found in results section")
	}

	lastBrace := strings.LastIndex(jsonSection, "}")
	if lastBrace == -1 {
		return nil, fmt.Errorf("JSON end not found in results section")
	}

	jsonStr := jsonSection[firstBrace : lastBrace+1]

	// Parse the JSON
	var result NotebookExecutionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse results JSON: %w", err)
	}

	return &result, nil
}

// parseGoldenNotebookFromLogs extracts the golden notebook JSON from pod logs
// Phase 3: Golden Notebook Comparison
func parseGoldenNotebookFromLogs(logs string) (*NotebookFormat, error) {
	// Look for the "Golden Notebook Summary:" section in the logs
	jsonStartMarker := "Golden Notebook Summary:"
	jsonStartIdx := strings.Index(logs, jsonStartMarker)
	if jsonStartIdx == -1 {
		// Golden notebook not found - this is OK if not specified
		return nil, nil
	}

	// Extract everything after the marker
	jsonSection := logs[jsonStartIdx+len(jsonStartMarker):]

	// Find the JSON object (starts with { and ends with })
	firstBrace := strings.Index(jsonSection, "{")
	if firstBrace == -1 {
		return nil, fmt.Errorf("JSON start not found in golden notebook section")
	}

	lastBrace := strings.LastIndex(jsonSection, "}")
	if lastBrace == -1 {
		return nil, fmt.Errorf("JSON end not found in golden notebook section")
	}

	jsonStr := jsonSection[firstBrace : lastBrace+1]

	// Parse the JSON
	var golden NotebookFormat
	if err := json.Unmarshal([]byte(jsonStr), &golden); err != nil {
		return nil, fmt.Errorf("failed to parse golden notebook JSON: %w", err)
	}

	return &golden, nil
}

// convertExecutionResultToNotebookFormat converts execution results to NotebookFormat for comparison
// Phase 3: Golden Notebook Comparison
func convertExecutionResultToNotebookFormat(result *NotebookExecutionResult) *NotebookFormat {
	notebook := &NotebookFormat{
		Cells: make([]NotebookCell, 0, len(result.Cells)),
	}

	for _, cell := range result.Cells {
		notebookCell := NotebookCell{
			CellType:       cell.CellType,
			ExecutionCount: cell.ExecutionCount,
			Outputs:        []CellOutput{},
		}

		// For code cells with errors, add error output
		if cell.Status == "failed" && cell.Error != "" {
			errorOutput := CellOutput{
				OutputType: "error",
				Ename:      "ExecutionError",
				Evalue:     cell.Error,
				Traceback:  cell.Traceback,
			}
			notebookCell.Outputs = append(notebookCell.Outputs, errorOutput)
		}

		notebook.Cells = append(notebook.Cells, notebookCell)
	}

	return notebook
}

// updateJobStatusWithResults updates the NotebookValidationJob status with execution results
func (r *NotebookValidationJobReconciler) updateJobStatusWithResults(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	result *NotebookExecutionResult,
) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Updating job status with results",
		"namespace", job.Namespace,
		"name", job.Name,
		"status", result.Status,
		"cells", len(result.Cells),
		"successRate", result.Statistics.SuccessRate)

	// Convert execution results to CellResult format
	cellResults := make([]mlopsv1alpha1.CellResult, 0, len(result.Cells))
	for _, cell := range result.Cells {
		cellResult := mlopsv1alpha1.CellResult{
			CellIndex: cell.CellIndex,
		}

		// Map status
		if cell.Status == "succeeded" {
			cellResult.Status = "Success"
		} else if cell.Status == "failed" {
			cellResult.Status = "Failure"

			// Copy error message if present
			if cell.Error != "" {
				cellResult.ErrorMessage = cell.Error
			}

			// Copy traceback to output field for detailed debugging
			if len(cell.Traceback) > 0 {
				// Join traceback lines and truncate if too long
				tracebackStr := strings.Join(cell.Traceback, "\n")
				if len(tracebackStr) > 2000 {
					tracebackStr = tracebackStr[:2000] + "\n... (truncated)"
				}
				cellResult.Output = tracebackStr
			}
		} else if cell.CellType == "markdown" {
			cellResult.Status = "Skipped" // Markdown cells are not executed
		} else {
			cellResult.Status = "Success" // Default for code cells without explicit status
		}

		cellResults = append(cellResults, cellResult)
	}

	// Update job status
	job.Status.Results = cellResults

	// Update message with statistics
	message := fmt.Sprintf("Validation completed: %d/%d cells succeeded (%.1f%% success rate)",
		result.Statistics.CodeCells-result.Statistics.FailedCells,
		result.Statistics.CodeCells,
		result.Statistics.SuccessRate)

	if result.Status == "failed" {
		message = fmt.Sprintf("Validation failed: %s", result.Error)
	}

	job.Status.Message = message

	// Update the status
	if err := r.Status().Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	logger.Info("Job status updated successfully", "message", message)
	return nil
}

// extractErrorFromLogs extracts error messages from logs if validation failed
func extractErrorFromLogs(logs string) string {
	// Look for error patterns in logs
	errorPatterns := []string{
		"ERROR:",
		"Error:",
		"FAILED:",
		"Failed:",
		"Exception:",
		"Traceback",
	}

	lines := strings.Split(logs, "\n")
	var errorLines []string

	for _, line := range lines {
		for _, pattern := range errorPatterns {
			if strings.Contains(line, pattern) {
				errorLines = append(errorLines, strings.TrimSpace(line))
				break
			}
		}
	}

	if len(errorLines) > 0 {
		// Return first 5 error lines to avoid too much detail
		if len(errorLines) > 5 {
			errorLines = errorLines[:5]
		}
		return strings.Join(errorLines, "\n")
	}

	return "Validation failed (see pod logs for details)"
}

// handlePodSuccess processes a successful pod and updates the job status
func (r *NotebookValidationJobReconciler) handlePodSuccess(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	pod *corev1.Pod,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing successful pod", "pod", pod.Name)

	// Collect logs from the validator container
	logs, err := r.collectPodLogs(ctx, pod, "validator")
	if err != nil {
		logger.Error(err, "Failed to collect pod logs")
		// Still mark as succeeded but with a warning message
		return r.updateJobPhase(ctx, job, PhaseSucceeded,
			fmt.Sprintf("Validation completed but failed to collect logs: %v", err))
	}

	// Parse results from logs
	result, err := parseResultsFromLogs(logs)
	if err != nil {
		logger.Error(err, "Failed to parse results from logs")
		// Still mark as succeeded but with a warning message
		return r.updateJobPhase(ctx, job, PhaseSucceeded,
			fmt.Sprintf("Validation completed but failed to parse results: %v", err))
	}

	// Update job status with results
	if err := r.updateJobStatusWithResults(ctx, job, result); err != nil {
		logger.Error(err, "Failed to update job status with results")
		return ctrl.Result{}, err
	}

	// Phase 3: Golden Notebook Comparison
	// Parse golden notebook if specified and perform comparison
	if job.Spec.GoldenNotebook != nil {
		logger.Info("Golden notebook specified, performing comparison")

		// Parse golden notebook from logs
		goldenNotebook, err := parseGoldenNotebookFromLogs(logs)
		if err != nil {
			logger.Error(err, "Failed to parse golden notebook from logs")
			// Don't fail the validation, just log the error
			job.Status.Message = fmt.Sprintf("%s (golden notebook parsing failed: %v)", job.Status.Message, err)
		} else if goldenNotebook != nil {
			logger.Info("Golden notebook parsed successfully", "cells", len(goldenNotebook.Cells))

			// Convert execution result to NotebookFormat for comparison
			executedNotebook := convertExecutionResultToNotebookFormat(result)

			// Get comparison configuration from annotations
			comparisonConfig := getComparisonConfig(job)

			// Perform comparison
			comparisonResult := compareNotebooks(executedNotebook, goldenNotebook, comparisonConfig)

			// Update job status with comparison result
			job.Status.ComparisonResult = comparisonResult

			// Update the job status
			if err := r.Status().Update(ctx, job); err != nil {
				logger.Error(err, "Failed to update job status with comparison result")
				return ctrl.Result{}, err
			}

			logger.Info("Comparison complete",
				"strategy", comparisonResult.Strategy,
				"result", comparisonResult.Result,
				"matched", comparisonResult.MatchedCells,
				"mismatched", comparisonResult.MismatchedCells)

			// If comparison failed, mark validation as failed
			if comparisonResult.Result == "failed" {
				logger.Info("Golden notebook comparison failed, marking validation as failed")
				// Record validation completion metric with failed status
				recordValidationComplete(job.Namespace, "failed")
				return r.updateJobPhase(ctx, job, PhaseFailed,
					fmt.Sprintf("Validation failed: golden notebook comparison failed (%d/%d cells matched)",
						comparisonResult.MatchedCells, comparisonResult.TotalCells))
			}
		} else {
			logger.Info("Golden notebook not found in logs, skipping comparison")
		}
	}

	// Determine final phase based on execution result
	finalPhase := PhaseSucceeded
	status := "succeeded"
	if result.Status == "failed" {
		finalPhase = PhaseFailed
		status = "failed"
	}

	// Record validation completion metric
	recordValidationComplete(job.Namespace, status)
	logger.Info("Recorded validation completion metric", "namespace", job.Namespace, "status", status)

	// Update phase and completion time
	return r.updateJobPhase(ctx, job, finalPhase, job.Status.Message)
}

// handlePodFailure processes a failed pod and updates the job status
// Implements ADR-019: Smart Validation Pod Recovery with failure analysis
func (r *NotebookValidationJobReconciler) handlePodFailure(
	ctx context.Context,
	job *mlopsv1alpha1.NotebookValidationJob,
	pod *corev1.Pod,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing failed pod", "pod", pod.Name)

	// ADR-019: Analyze pod failure to determine recovery strategy
	analysis := analyzePodFailure(ctx, pod)
	logger.Info("Pod failure analysis complete",
		"reason", analysis.Reason,
		"isTransient", analysis.IsTransient,
		"shouldRetry", analysis.ShouldRetry,
		"failedContainer", analysis.FailedContainer,
		"isInitContainer", analysis.IsInitContainer,
		"isSCCViolation", analysis.IsSCCViolation,
		"isImageIssue", analysis.IsImageIssue,
		"suggestedAction", analysis.SuggestedAction)

	// Try to collect logs to get error details
	logs, err := r.collectPodLogs(ctx, pod, "validator")
	errorMsg := fmt.Sprintf("Validation pod failed: %s - %s", analysis.Reason, analysis.ErrorMessage)

	if err == nil {
		// Extract error from logs
		if extractedError := extractErrorFromLogs(logs); extractedError != "" {
			errorMsg = fmt.Sprintf("%s. Log error: %s", errorMsg, extractedError)
		}
	}

	// Increment retry count
	job.Status.RetryCount++
	job.Status.LastRetryTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, job); err != nil {
		return ctrl.Result{}, err
	}

	// Get recovery action based on failure analysis
	recoveryAction := getFailureRecoveryAction(analysis, job.Status.RetryCount)
	logger.Info("Determined recovery action", "action", recoveryAction, "retryCount", job.Status.RetryCount)

	// Handle recovery action
	switch recoveryAction {
	case "skip_init_container":
		logger.Info("Init container failure detected - will retry without init container (using built image)")
		// Delete the failed pod
		if err := r.Delete(ctx, pod); err != nil {
			logger.Error(err, "Failed to delete failed pod")
		}
		// Requeue to create a new pod (controller will skip init container for built images)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	case "retry_with_backoff":
		if job.Status.RetryCount < MaxRetries {
			// Exponential backoff: 1m, 2m, 4m
			backoff := time.Minute * time.Duration(1<<uint(job.Status.RetryCount-1))
			if backoff > 5*time.Minute {
				backoff = 5 * time.Minute
			}
			logger.Info("Retrying validation with backoff", "retryCount", job.Status.RetryCount, "backoff", backoff)
			// Delete the failed pod
			if err := r.Delete(ctx, pod); err != nil {
				logger.Error(err, "Failed to delete failed pod")
			}
			// Requeue with backoff
			return ctrl.Result{RequeueAfter: backoff}, nil
		}

	case "fallback_to_prebuilt_image":
		logger.Info("Image pull failures - will fallback to pre-built image on next retry")
		// Delete the failed pod
		if err := r.Delete(ctx, pod); err != nil {
			logger.Error(err, "Failed to delete failed pod")
		}
		// TODO: Implement fallback to spec.podConfig.containerImage
		// For now, retry with backoff
		return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil

	case "max_retries_exceeded", "do_not_retry":
		// Max retries reached or should not retry
		logger.Info("Max retries exceeded or retry not recommended", "retryCount", job.Status.RetryCount)
		// Record validation completion metric with failed status
		recordValidationComplete(job.Namespace, "failed")
		logger.Info("Recorded validation completion metric", "namespace", job.Namespace, "status", "failed")

		return r.updateJobPhase(ctx, job, PhaseFailed,
			fmt.Sprintf("Validation failed after %d retries: %s. Suggested action: %s",
				job.Status.RetryCount, errorMsg, analysis.SuggestedAction))

	default:
		// Unknown recovery action - use default retry logic
		if job.Status.RetryCount < MaxRetries {
			logger.Info("Using default retry logic", "retryCount", job.Status.RetryCount)
			// Delete the failed pod
			if err := r.Delete(ctx, pod); err != nil {
				logger.Error(err, "Failed to delete failed pod")
			}
			// Requeue to create a new pod
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	// Max retries reached, mark as failed
	// Record validation completion metric with failed status
	recordValidationComplete(job.Namespace, "failed")
	logger.Info("Recorded validation completion metric", "namespace", job.Namespace, "status", "failed")

	return r.updateJobPhase(ctx, job, PhaseFailed,
		fmt.Sprintf("Validation failed after %d retries: %s. Suggested action: %s",
			MaxRetries, errorMsg, analysis.SuggestedAction))
}
