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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationConfigSpec_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *ValidationConfigSpec
		expected ValidationConfigSpec
	}{
		{
			name:   "nil config",
			config: nil,
			expected: ValidationConfigSpec{
				Level:                    "",
				StrictMode:               false,
				FailOnStderr:             false,
				FailOnWarnings:           false,
				DetectSilentFailures:     nil,
				RequireExplicitExitCodes: false,
				CheckOutputTypes:         false,
				VerifyAssertions:         false,
			},
		},
		{
			name:   "empty config",
			config: &ValidationConfigSpec{},
			expected: ValidationConfigSpec{
				Level:                    "",
				StrictMode:               false,
				FailOnStderr:             false,
				FailOnWarnings:           false,
				DetectSilentFailures:     nil,
				RequireExplicitExitCodes: false,
				CheckOutputTypes:         false,
				VerifyAssertions:         false,
			},
		},
		{
			name: "development level config",
			config: &ValidationConfigSpec{
				Level:      "development",
				StrictMode: false,
			},
			expected: ValidationConfigSpec{
				Level:      "development",
				StrictMode: false,
			},
		},
		{
			name: "production level config",
			config: &ValidationConfigSpec{
				Level:          "production",
				StrictMode:     true,
				FailOnStderr:   true,
				FailOnWarnings: true,
			},
			expected: ValidationConfigSpec{
				Level:          "production",
				StrictMode:     true,
				FailOnStderr:   true,
				FailOnWarnings: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				// Test nil case
				var config ValidationConfigSpec
				assert.Equal(t, tt.expected.Level, config.Level)
				assert.Equal(t, tt.expected.StrictMode, config.StrictMode)
				assert.Equal(t, tt.expected.FailOnStderr, config.FailOnStderr)
				assert.Equal(t, tt.expected.FailOnWarnings, config.FailOnWarnings)
			} else {
				assert.Equal(t, tt.expected.Level, tt.config.Level)
				assert.Equal(t, tt.expected.StrictMode, tt.config.StrictMode)
				assert.Equal(t, tt.expected.FailOnStderr, tt.config.FailOnStderr)
				assert.Equal(t, tt.expected.FailOnWarnings, tt.config.FailOnWarnings)
			}
		})
	}
}

func TestValidationConfigSpec_LevelValidation(t *testing.T) {
	validLevels := []string{"learning", "development", "staging", "production"}

	for _, level := range validLevels {
		t.Run("valid_level_"+level, func(t *testing.T) {
			config := &ValidationConfigSpec{
				Level: level,
			}
			assert.Equal(t, level, config.Level)
		})
	}
}

func TestValidationConfigSpec_StrictModeImplications(t *testing.T) {
	tests := []struct {
		name           string
		strictMode     bool
		failOnStderr   bool
		failOnWarnings bool
		description    string
	}{
		{
			name:           "strict mode with all failures enabled",
			strictMode:     true,
			failOnStderr:   true,
			failOnWarnings: true,
			description:    "production-ready validation",
		},
		{
			name:           "relaxed mode for development",
			strictMode:     false,
			failOnStderr:   false,
			failOnWarnings: false,
			description:    "development-friendly validation",
		},
		{
			name:           "stderr only mode",
			strictMode:     false,
			failOnStderr:   true,
			failOnWarnings: false,
			description:    "catch errors but allow warnings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ValidationConfigSpec{
				StrictMode:     tt.strictMode,
				FailOnStderr:   tt.failOnStderr,
				FailOnWarnings: tt.failOnWarnings,
			}

			assert.Equal(t, tt.strictMode, config.StrictMode)
			assert.Equal(t, tt.failOnStderr, config.FailOnStderr)
			assert.Equal(t, tt.failOnWarnings, config.FailOnWarnings)
		})
	}
}

func TestExpectedOutputSpec(t *testing.T) {
	tests := []struct {
		name     string
		output   ExpectedOutputSpec
		expected ExpectedOutputSpec
	}{
		{
			name: "dataframe output spec",
			output: ExpectedOutputSpec{
				Cell:     5,
				Type:     "pandas.DataFrame",
				Shape:    "-1,10",
				NotEmpty: true,
			},
			expected: ExpectedOutputSpec{
				Cell:     5,
				Type:     "pandas.DataFrame",
				Shape:    "-1,10",
				NotEmpty: true,
			},
		},
		{
			name: "numeric output spec with range",
			output: ExpectedOutputSpec{
				Cell:     8,
				Type:     "float",
				MinValue: "0.7",
				MaxValue: "1.0",
			},
			expected: ExpectedOutputSpec{
				Cell:     8,
				Type:     "float",
				MinValue: "0.7",
				MaxValue: "1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Cell, tt.output.Cell)
			assert.Equal(t, tt.expected.Type, tt.output.Type)
			assert.Equal(t, tt.expected.Shape, tt.output.Shape)
			assert.Equal(t, tt.expected.MinValue, tt.output.MinValue)
			assert.Equal(t, tt.expected.MaxValue, tt.output.MaxValue)
			assert.Equal(t, tt.expected.NotEmpty, tt.output.NotEmpty)
		})
	}
}

func TestNotebookValidationJobSpec_WithValidationConfig(t *testing.T) {
	tests := []struct {
		name              string
		spec              NotebookValidationJobSpec
		expectValidation  bool
		expectedStrictMod bool
	}{
		{
			name: "no validation config",
			spec: NotebookValidationJobSpec{
				Notebook: NotebookSpec{
					Git:  GitSpec{URL: "https://github.com/example/repo.git", Ref: "main"},
					Path: "notebook.ipynb",
				},
				PodConfig: PodConfigSpec{
					ContainerImage: "jupyter/minimal-notebook:latest",
				},
			},
			expectValidation:  false,
			expectedStrictMod: false,
		},
		{
			name: "with validation config - strict mode",
			spec: NotebookValidationJobSpec{
				Notebook: NotebookSpec{
					Git:  GitSpec{URL: "https://github.com/example/repo.git", Ref: "main"},
					Path: "notebook.ipynb",
				},
				PodConfig: PodConfigSpec{
					ContainerImage: "jupyter/minimal-notebook:latest",
				},
				ValidationConfig: &ValidationConfigSpec{
					Level:      "production",
					StrictMode: true,
				},
			},
			expectValidation:  true,
			expectedStrictMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasValidationConfig := tt.spec.ValidationConfig != nil
			assert.Equal(t, tt.expectValidation, hasValidationConfig)

			if hasValidationConfig {
				assert.Equal(t, tt.expectedStrictMod, tt.spec.ValidationConfig.StrictMode)
			}
		})
	}
}
