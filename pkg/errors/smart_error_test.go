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

package errors

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSmartError(t *testing.T) {
	originalErr := errors.New("original error")
	smartErr := NewSmartError(CategoryRBAC, "TEST_CODE", "Test message", originalErr)

	assert.Equal(t, CategoryRBAC, smartErr.Category)
	assert.Equal(t, "TEST_CODE", smartErr.Code)
	assert.Equal(t, "Test message", smartErr.Message)
	assert.Equal(t, SeverityError, smartErr.Severity)
	assert.Equal(t, originalErr, smartErr.OriginalError)
	assert.False(t, smartErr.Retryable)
}

func TestSmartError_Error(t *testing.T) {
	smartErr := NewSmartError(CategoryRBAC, "RBAC_ERROR", "Permission denied", nil)
	expected := "[RBAC] RBAC_ERROR: Permission denied"
	assert.Equal(t, expected, smartErr.Error())
}

func TestSmartError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	smartErr := NewSmartError(CategoryRBAC, "TEST", "Test", originalErr)

	assert.Equal(t, originalErr, smartErr.Unwrap())
	assert.True(t, errors.Is(smartErr, originalErr))
}

func TestSmartError_ChainedMethods(t *testing.T) {
	smartErr := NewSmartError(CategoryTekton, "TEKTON_ERR", "Pipeline error", nil).
		WithRootCause("Task not found").
		WithImpact("Build cannot proceed").
		WithActions("Delete pipeline", "Recreate pipeline").
		WithReferences("ADR-028", "ADR-030").
		WithSeverity(SeverityCritical).
		WithRetryable(true)

	assert.Equal(t, "Task not found", smartErr.RootCause)
	assert.Equal(t, "Build cannot proceed", smartErr.Impact)
	assert.Equal(t, []string{"Delete pipeline", "Recreate pipeline"}, smartErr.Actions)
	assert.Equal(t, []string{"ADR-028", "ADR-030"}, smartErr.References)
	assert.Equal(t, SeverityCritical, smartErr.Severity)
	assert.True(t, smartErr.Retryable)
}

func TestSmartError_UserFriendlyMessage(t *testing.T) {
	smartErr := NewSmartError(CategoryRBAC, "RBAC_ERR", "Permission denied", nil)
	assert.Equal(t, "RBAC: Permission denied", smartErr.UserFriendlyMessage())
}

func TestSmartError_DetailedMessage(t *testing.T) {
	smartErr := NewSmartError(CategoryRBAC, "RBAC_ERR", "Permission denied", nil).
		WithRootCause("Missing tasks permission").
		WithImpact("Build cannot run").
		WithActions("Fix RBAC", "Restart operator").
		WithReferences("ADR-028")

	detailed := smartErr.DetailedMessage()

	assert.Contains(t, detailed, "Permission denied")
	assert.Contains(t, detailed, "RBAC")
	assert.Contains(t, detailed, "Missing tasks permission")
	assert.Contains(t, detailed, "Build cannot run")
	assert.Contains(t, detailed, "Fix RBAC")
	assert.Contains(t, detailed, "ADR-028")
}

func TestAnalyzeError_RBAC(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedCode   string
		expectedCat    ErrorCategory
		containsAction string
	}{
		{
			name:           "forbidden tasks",
			err:            errors.New("forbidden: cannot create tasks in namespace"),
			expectedCode:   "RBAC_PERMISSION_DENIED",
			expectedCat:    CategoryRBAC,
			containsAction: "tasks",
		},
		{
			name:           "forbidden pipelines",
			err:            errors.New("forbidden: cannot create pipelines"),
			expectedCode:   "RBAC_PERMISSION_DENIED",
			expectedCat:    CategoryRBAC,
			containsAction: "pipeline",
		},
		{
			name:           "unauthorized",
			err:            errors.New("unauthorized: invalid token"),
			expectedCode:   "RBAC_PERMISSION_DENIED",
			expectedCat:    CategoryRBAC,
			containsAction: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := AnalyzeError(tt.err)

			assert.NotNil(t, smartErr)
			assert.Equal(t, tt.expectedCat, smartErr.Category)
			assert.Equal(t, tt.expectedCode, smartErr.Code)
			if tt.containsAction != "" {
				found := false
				for _, action := range smartErr.Actions {
					if strings.Contains(strings.ToLower(action), tt.containsAction) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected action containing '%s'", tt.containsAction)
			}
		})
	}
}

func TestAnalyzeError_ResourceNotFound(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
		expectedCat  ErrorCategory
	}{
		{
			name:         "task not found",
			err:          errors.New("task git-clone not found in namespace"),
			expectedCode: "RESOURCE_NOT_FOUND",
			expectedCat:  CategoryResource,
		},
		{
			name:         "clustertask not found",
			err:          errors.New("clustertasks.tekton.dev 'git-clone' not found"),
			expectedCode: "RESOURCE_NOT_FOUND",
			expectedCat:  CategoryResource,
		},
		{
			name:         "generic not found",
			err:          errors.New("configmap not found"),
			expectedCode: "RESOURCE_NOT_FOUND",
			expectedCat:  CategoryResource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := AnalyzeError(tt.err)

			assert.NotNil(t, smartErr)
			assert.Equal(t, tt.expectedCat, smartErr.Category)
			assert.Equal(t, tt.expectedCode, smartErr.Code)
			assert.True(t, smartErr.Retryable)
		})
	}
}

func TestAnalyzeError_Tekton(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "param mismatch",
			err:          errors.New("Pipeline failed: missing values for these params: [URL]"),
			expectedCode: "TEKTON_PARAM_MISMATCH",
		},
		{
			name:         "pipeline invalid",
			err:          errors.New("Pipeline can't be run due to invalid spec"),
			expectedCode: "TEKTON_PIPELINE_INVALID",
		},
		{
			name:         "generic tekton",
			err:          errors.New("PipelineRun failed with timeout"),
			expectedCode: "TEKTON_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := AnalyzeError(tt.err)

			assert.NotNil(t, smartErr)
			assert.Equal(t, CategoryTekton, smartErr.Category)
			assert.Equal(t, tt.expectedCode, smartErr.Code)
		})
	}
}

func TestAnalyzeError_Authentication(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "git auth failed",
			err:          errors.New("authentication failed for git repository"),
			expectedCode: "GIT_AUTH_FAILED",
		},
		{
			name:         "git permission denied",
			err:          errors.New("permission denied (publickey) for git"),
			expectedCode: "GIT_AUTH_FAILED",
		},
		{
			name:         "generic auth",
			err:          errors.New("invalid credentials for API"),
			expectedCode: "AUTH_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := AnalyzeError(tt.err)

			assert.NotNil(t, smartErr)
			assert.Equal(t, CategoryAuthentication, smartErr.Category)
			assert.Equal(t, tt.expectedCode, smartErr.Code)
			assert.False(t, smartErr.Retryable)
		})
	}
}

func TestAnalyzeError_Configuration(t *testing.T) {
	err := errors.New("validation failed: invalid spec.notebook.path")
	smartErr := AnalyzeError(err)

	assert.NotNil(t, smartErr)
	assert.Equal(t, CategoryConfiguration, smartErr.Category)
	assert.Equal(t, "CONFIG_ERROR", smartErr.Code)
}

func TestAnalyzeError_Network(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "connection refused",
			err:  errors.New("connection refused to registry.example.com"),
		},
		{
			name: "timeout",
			err:  errors.New("timeout waiting for response"),
		},
		{
			name: "network unreachable",
			err:  errors.New("network is unreachable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smartErr := AnalyzeError(tt.err)

			assert.NotNil(t, smartErr)
			assert.Equal(t, CategoryNetwork, smartErr.Category)
			assert.Equal(t, "NETWORK_ERROR", smartErr.Code)
			assert.True(t, smartErr.Retryable)
		})
	}
}

func TestAnalyzeError_Unknown(t *testing.T) {
	err := errors.New("some random error that doesn't match patterns")
	smartErr := AnalyzeError(err)

	assert.NotNil(t, smartErr)
	assert.Equal(t, CategoryUnknown, smartErr.Category)
	assert.Equal(t, "UNKNOWN_ERROR", smartErr.Code)
}

func TestAnalyzeError_Nil(t *testing.T) {
	smartErr := AnalyzeError(nil)
	assert.Nil(t, smartErr)
}

func TestIsRetryable(t *testing.T) {
	retryableErr := NewSmartError(CategoryNetwork, "NET", "Network error", nil).WithRetryable(true)
	nonRetryableErr := NewSmartError(CategoryRBAC, "RBAC", "Permission denied", nil).WithRetryable(false)
	regularErr := errors.New("regular error")

	assert.True(t, IsRetryable(retryableErr))
	assert.False(t, IsRetryable(nonRetryableErr))
	assert.False(t, IsRetryable(regularErr))
}

func TestGetCategory(t *testing.T) {
	smartErr := NewSmartError(CategoryTekton, "TEKTON", "Tekton error", nil)
	regularErr := errors.New("regular error")

	assert.Equal(t, CategoryTekton, GetCategory(smartErr))
	assert.Equal(t, CategoryUnknown, GetCategory(regularErr))
}
