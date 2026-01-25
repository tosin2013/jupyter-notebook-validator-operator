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

// Package errors provides smart error handling with root cause analysis and actionable guidance.
// ADR-030: Smart Error Messages and User Feedback
package errors

import (
	"fmt"
	"strings"
)

// ErrorCategory represents the category of error for classification
type ErrorCategory string

const (
	// CategoryRBAC represents permission and RBAC errors
	CategoryRBAC ErrorCategory = "RBAC"
	// CategoryResource represents resource-related errors (not found, already exists)
	CategoryResource ErrorCategory = "Resource"
	// CategoryConfiguration represents configuration and validation errors
	CategoryConfiguration ErrorCategory = "Configuration"
	// CategoryPlatform represents platform availability errors
	CategoryPlatform ErrorCategory = "Platform"
	// CategoryDependency represents dependency and prerequisite errors
	CategoryDependency ErrorCategory = "Dependency"
	// CategoryTekton represents Tekton-specific errors
	CategoryTekton ErrorCategory = "Tekton"
	// CategoryBuild represents build-related errors
	CategoryBuild ErrorCategory = "Build"
	// CategoryNetwork represents network connectivity errors
	CategoryNetwork ErrorCategory = "Network"
	// CategoryAuthentication represents authentication errors
	CategoryAuthentication ErrorCategory = "Authentication"
	// CategoryUnknown represents unclassified errors
	CategoryUnknown ErrorCategory = "Unknown"
)

// ErrorSeverity represents the severity level of the error
type ErrorSeverity string

const (
	// SeverityCritical means the operation cannot proceed
	SeverityCritical ErrorSeverity = "Critical"
	// SeverityError means the operation failed but may be retryable
	SeverityError ErrorSeverity = "Error"
	// SeverityWarning means the operation succeeded but with issues
	SeverityWarning ErrorSeverity = "Warning"
	// SeverityInfo means informational message
	SeverityInfo ErrorSeverity = "Info"
)

// SmartError provides intelligent error handling with root cause analysis and actionable guidance.
// ADR-030: Smart Error Messages and User Feedback - Level 2 & 3
type SmartError struct {
	// Category classifies the error type (RBAC, Resource, Configuration, etc.)
	Category ErrorCategory `json:"category"`

	// Severity indicates the error severity level
	Severity ErrorSeverity `json:"severity"`

	// Code is a unique error code for programmatic handling
	Code string `json:"code"`

	// Message is the user-friendly error message
	Message string `json:"message"`

	// RootCause explains WHY the error occurred (technical details)
	RootCause string `json:"rootCause"`

	// Impact describes what this error means for the user
	Impact string `json:"impact"`

	// Actions provides specific steps to fix the problem
	Actions []string `json:"actions,omitempty"`

	// References provides links to documentation, ADRs, and examples
	References []string `json:"references,omitempty"`

	// Retryable indicates if the operation can be retried
	Retryable bool `json:"retryable"`

	// OriginalError is the underlying error
	OriginalError error `json:"-"`
}

// Error implements the error interface
func (e *SmartError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Category, e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *SmartError) Unwrap() error {
	return e.OriginalError
}

// UserFriendlyMessage returns a formatted message suitable for status fields
func (e *SmartError) UserFriendlyMessage() string {
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

// DetailedMessage returns a comprehensive message with all details
func (e *SmartError) DetailedMessage() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Error: %s\n", e.Message))
	sb.WriteString(fmt.Sprintf("Category: %s\n", e.Category))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", e.Severity))

	if e.RootCause != "" {
		sb.WriteString(fmt.Sprintf("\nRoot Cause:\n%s\n", e.RootCause))
	}

	if e.Impact != "" {
		sb.WriteString(fmt.Sprintf("\nImpact:\n%s\n", e.Impact))
	}

	if len(e.Actions) > 0 {
		sb.WriteString("\nActions to fix:\n")
		for i, action := range e.Actions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
		}
	}

	if len(e.References) > 0 {
		sb.WriteString("\nReferences:\n")
		for _, ref := range e.References {
			sb.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	return sb.String()
}

// NewSmartError creates a new SmartError with the given parameters
func NewSmartError(category ErrorCategory, code, message string, originalErr error) *SmartError {
	return &SmartError{
		Category:      category,
		Severity:      SeverityError,
		Code:          code,
		Message:       message,
		OriginalError: originalErr,
		Retryable:     false,
	}
}

// WithRootCause adds root cause information
func (e *SmartError) WithRootCause(rootCause string) *SmartError {
	e.RootCause = rootCause
	return e
}

// WithImpact adds impact information
func (e *SmartError) WithImpact(impact string) *SmartError {
	e.Impact = impact
	return e
}

// WithActions adds fix actions
func (e *SmartError) WithActions(actions ...string) *SmartError {
	e.Actions = actions
	return e
}

// WithReferences adds documentation references
func (e *SmartError) WithReferences(refs ...string) *SmartError {
	e.References = refs
	return e
}

// WithSeverity sets the severity level
func (e *SmartError) WithSeverity(severity ErrorSeverity) *SmartError {
	e.Severity = severity
	return e
}

// WithRetryable marks the error as retryable
func (e *SmartError) WithRetryable(retryable bool) *SmartError {
	e.Retryable = retryable
	return e
}

// AnalyzeError analyzes an error and returns a SmartError with appropriate categorization
// ADR-030: Level 2 - Root Cause Analysis
func AnalyzeError(err error) *SmartError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// RBAC Errors
	if strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "unauthorized") {
		return analyzeRBACError(err, errStr)
	}

	// Resource Errors
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "notfound") {
		return analyzeResourceNotFoundError(err, errStr)
	}

	if strings.Contains(errStr, "already exists") || strings.Contains(errStr, "alreadyexists") {
		return analyzeResourceExistsError(err, errStr)
	}

	// Tekton Errors
	if strings.Contains(errStr, "pipeline") || strings.Contains(errStr, "pipelinerun") ||
		strings.Contains(errStr, "task") || strings.Contains(errStr, "tekton") {
		return analyzeTektonError(err, errStr)
	}

	// Authentication Errors
	if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "credential") ||
		strings.Contains(errStr, "token") || strings.Contains(errStr, "permission denied") {
		return analyzeAuthenticationError(err, errStr)
	}

	// Configuration Errors
	if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "missing required") {
		return analyzeConfigurationError(err, errStr)
	}

	// Network Errors
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") || strings.Contains(errStr, "unreachable") {
		return analyzeNetworkError(err, errStr)
	}

	// Default unknown error
	return NewSmartError(CategoryUnknown, "UNKNOWN_ERROR", err.Error(), err).
		WithSeverity(SeverityError).
		WithImpact("The operation failed for an unclassified reason.").
		WithActions(
			"Check the operator logs for more details",
			"Review the NotebookValidationJob configuration",
			"Ensure all prerequisites are met",
		)
}

// analyzeRBACError provides detailed analysis for RBAC errors
func analyzeRBACError(err error, errStr string) *SmartError {
	smartErr := NewSmartError(CategoryRBAC, "RBAC_PERMISSION_DENIED", "Permission denied", err).
		WithSeverity(SeverityCritical)

	if strings.Contains(errStr, "tasks") {
		smartErr.Message = "Permission denied: Cannot access Tekton Tasks"
		smartErr.RootCause = "ClusterRole missing 'tasks' resource permission for tekton.dev API group"
		smartErr.Impact = "Tekton builds cannot run. Operator cannot copy Tasks to user namespace."
		smartErr.Actions = []string{
			"Add 'tasks' to ClusterRole resources in config/rbac/role.yaml",
			"Apply updated RBAC: kubectl apply -f config/rbac/role.yaml",
			"Or patch directly: kubectl patch clusterrole <role-name> --type='json' -p='[{\"op\": \"add\", \"path\": \"/rules/-\", \"value\": {\"apiGroups\": [\"tekton.dev\"], \"resources\": [\"tasks\"], \"verbs\": [\"create\", \"delete\", \"get\", \"list\", \"patch\", \"update\", \"watch\"]}}]'",
		}
		smartErr.References = []string{
			"ADR-028: Tekton Task Strategy",
			"config/rbac/role.yaml",
		}
	} else if strings.Contains(errStr, "pipeline") {
		smartErr.Message = "Permission denied: Cannot access Tekton Pipelines"
		smartErr.RootCause = "ClusterRole missing 'pipelines' or 'pipelineruns' resource permission"
		smartErr.Impact = "Tekton builds cannot be created or monitored."
		smartErr.Actions = []string{
			"Verify ClusterRole has permissions for pipelines and pipelineruns resources",
			"Check ServiceAccount binding to ClusterRole",
		}
		smartErr.References = []string{
			"ADR-028: Tekton Task Strategy",
			"config/rbac/role.yaml",
		}
	} else {
		smartErr.RootCause = "Insufficient permissions to perform the requested operation"
		smartErr.Impact = "The operation cannot proceed without proper authorization."
		smartErr.Actions = []string{
			"Review ClusterRole permissions",
			"Ensure ServiceAccount is properly bound",
			"Check namespace RBAC policies",
		}
	}

	return smartErr
}

// analyzeResourceNotFoundError provides detailed analysis for resource not found errors
func analyzeResourceNotFoundError(err error, errStr string) *SmartError {
	smartErr := NewSmartError(CategoryResource, "RESOURCE_NOT_FOUND", "Resource not found", err).
		WithSeverity(SeverityError)

	if strings.Contains(errStr, "task") && strings.Contains(errStr, "git-clone") {
		smartErr.Message = "Task 'git-clone' not found"
		smartErr.RootCause = "Task 'git-clone' not found in target namespace. May need to be copied from openshift-pipelines namespace."
		smartErr.Impact = "Pipeline cannot start. Task needs to be available in the namespace."
		smartErr.Actions = []string{
			"Operator should automatically copy Tasks (ADR-028)",
			"If automatic copy failed, check operator logs for RBAC errors",
			"Manual copy: kubectl get task git-clone -n openshift-pipelines -o yaml | kubectl apply -n <namespace> -f -",
		}
		smartErr.References = []string{
			"ADR-028: Tekton Task Strategy (namespace copy approach)",
		}
	} else if strings.Contains(errStr, "clustertask") {
		smartErr.Message = "ClusterTask not found (deprecated)"
		smartErr.RootCause = "Pipeline references ClusterTask which is deprecated. Should use namespace-scoped Task instead."
		smartErr.Impact = "Pipeline cannot run with ClusterTask references."
		smartErr.Actions = []string{
			"Delete the Pipeline and let operator recreate it with Task references",
			"Or update operator to latest version with ADR-028 fix",
			"Command: kubectl delete pipeline <name> -n <namespace>",
		}
		smartErr.References = []string{
			"ADR-028: Tekton Task Strategy",
		}
	} else {
		smartErr.RootCause = "The requested resource does not exist"
		smartErr.Impact = "The operation cannot proceed without the required resource."
		smartErr.Actions = []string{
			"Create the required resource first",
			"Check resource name and namespace are correct",
		}
	}

	smartErr.Retryable = true
	return smartErr
}

// analyzeResourceExistsError provides detailed analysis for resource already exists errors
func analyzeResourceExistsError(err error, errStr string) *SmartError {
	return NewSmartError(CategoryResource, "RESOURCE_EXISTS", "Resource already exists", err).
		WithSeverity(SeverityWarning).
		WithRootCause("A resource with the same name already exists in the namespace").
		WithImpact("The operation cannot create a duplicate resource.").
		WithActions(
			"Use a different resource name",
			"Delete the existing resource if no longer needed",
			"Or update the existing resource instead of creating",
		).
		WithRetryable(false)
}

// analyzeTektonError provides detailed analysis for Tekton-specific errors
func analyzeTektonError(err error, errStr string) *SmartError {
	smartErr := NewSmartError(CategoryTekton, "TEKTON_ERROR", "Tekton pipeline error", err).
		WithSeverity(SeverityError)

	if strings.Contains(errStr, "missing values for these params") ||
		strings.Contains(errStr, "param") {
		smartErr.Code = "TEKTON_PARAM_MISMATCH"
		smartErr.Message = "Parameter mismatch between Pipeline and Task"
		smartErr.RootCause = "Pipeline parameters don't match Task parameter names. OpenShift Pipelines Tasks use UPPERCASE parameter names."
		smartErr.Impact = "Pipeline cannot run due to missing parameter values."
		smartErr.Actions = []string{
			"Check parameter names match between Pipeline and Task",
			"OpenShift Pipelines Tasks use UPPERCASE parameter names (URL, REVISION, etc.)",
			"Delete Pipeline and let operator recreate: kubectl delete pipeline <name>",
		}
		smartErr.References = []string{
			"ADR-028: Tekton Task Strategy",
			"OpenShift Pipelines documentation",
		}
	} else if strings.Contains(errStr, "can't be run") || strings.Contains(errStr, "cannot be run") {
		smartErr.Code = "TEKTON_PIPELINE_INVALID"
		smartErr.Message = "Pipeline configuration is invalid"
		smartErr.RootCause = "Pipeline has configuration issues preventing execution"
		smartErr.Impact = "Build cannot proceed until Pipeline is fixed."
		smartErr.Actions = []string{
			"Check PipelineRun logs for specific error",
			"Delete and recreate Pipeline: kubectl delete pipeline <name>",
			"Verify Tasks exist in namespace",
		}
	} else {
		smartErr.RootCause = "Tekton pipeline encountered an error"
		smartErr.Impact = "Build process failed."
		smartErr.Actions = []string{
			"Check PipelineRun status: kubectl get pipelinerun -n <namespace>",
			"View PipelineRun logs: kubectl logs <pipelinerun-pod>",
			"Check operator logs for details",
		}
	}

	smartErr.References = append(smartErr.References, "ADR-030: Smart Error Messages")
	return smartErr
}

// analyzeAuthenticationError provides detailed analysis for authentication errors
func analyzeAuthenticationError(err error, errStr string) *SmartError {
	smartErr := NewSmartError(CategoryAuthentication, "AUTH_ERROR", "Authentication failed", err).
		WithSeverity(SeverityCritical).
		WithRetryable(false)

	if strings.Contains(errStr, "git") || strings.Contains(errStr, "repository") {
		smartErr.Code = "GIT_AUTH_FAILED"
		smartErr.Message = "Git authentication failed"
		smartErr.RootCause = "Git credentials are missing, invalid, or insufficient"
		smartErr.Impact = "Cannot clone repository for validation."
		smartErr.Actions = []string{
			"Verify git credentials secret exists and is properly configured",
			"For GitHub: Ensure fine-grained token has 'Contents' read permission",
			"For OpenShift Tekton: Add tekton.dev/git-0 annotation to secret",
			"Create secret: kubectl create secret generic git-credentials --from-literal=username=oauth2 --from-literal=password=<token>",
		}
		smartErr.References = []string{
			"config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml",
			"ADR-042: Automatic Tekton Git Credentials Conversion",
		}
	} else {
		smartErr.RootCause = "Authentication credentials are invalid or expired"
		smartErr.Impact = "Cannot access protected resources."
		smartErr.Actions = []string{
			"Verify credentials are correct and not expired",
			"Check secret exists and is properly formatted",
			"Regenerate credentials if expired",
		}
	}

	return smartErr
}

// analyzeConfigurationError provides detailed analysis for configuration errors
func analyzeConfigurationError(err error, errStr string) *SmartError {
	return NewSmartError(CategoryConfiguration, "CONFIG_ERROR", "Configuration error", err).
		WithSeverity(SeverityError).
		WithRootCause("The provided configuration is invalid or incomplete").
		WithImpact("The operation cannot proceed with invalid configuration.").
		WithActions(
			"Review NotebookValidationJob spec for errors",
			"Check required fields are provided",
			"Validate field formats match expected patterns",
		).
		WithReferences(
			"config/samples/ for example configurations",
			"API documentation",
		)
}

// analyzeNetworkError provides detailed analysis for network errors
func analyzeNetworkError(err error, errStr string) *SmartError {
	return NewSmartError(CategoryNetwork, "NETWORK_ERROR", "Network connectivity issue", err).
		WithSeverity(SeverityError).
		WithRootCause("Cannot establish network connection to required service").
		WithImpact("The operation cannot proceed due to network issues.").
		WithActions(
			"Check network connectivity from cluster",
			"Verify service endpoints are accessible",
			"Check firewall rules and network policies",
			"Retry after network issue is resolved",
		).
		WithRetryable(true)
}

// IsRetryable checks if a SmartError is retryable
func IsRetryable(err error) bool {
	if smartErr, ok := err.(*SmartError); ok {
		return smartErr.Retryable
	}
	return false
}

// GetCategory extracts the category from a SmartError
func GetCategory(err error) ErrorCategory {
	if smartErr, ok := err.(*SmartError); ok {
		return smartErr.Category
	}
	return CategoryUnknown
}
