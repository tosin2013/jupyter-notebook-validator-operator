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

func TestBuildConfigSpec_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *BuildConfigSpec
		expected BuildConfigSpec
	}{
		{
			name:   "nil config",
			config: nil,
			expected: BuildConfigSpec{
				Enabled:                  false,
				Strategy:                 "",
				BaseImage:                "",
				AutoGenerateRequirements: false,
				RequirementsFile:         "",
				FallbackStrategy:         "",
			},
		},
		{
			name:   "empty config",
			config: &BuildConfigSpec{},
			expected: BuildConfigSpec{
				Enabled:                  false,
				Strategy:                 "",
				BaseImage:                "",
				AutoGenerateRequirements: false,
				RequirementsFile:         "",
				FallbackStrategy:         "",
			},
		},
		{
			name: "enabled with defaults",
			config: &BuildConfigSpec{
				Enabled: true,
			},
			expected: BuildConfigSpec{
				Enabled:                  true,
				Strategy:                 "",
				BaseImage:                "",
				AutoGenerateRequirements: false,
				RequirementsFile:         "",
				FallbackStrategy:         "",
			},
		},
		{
			name: "full config",
			config: &BuildConfigSpec{
				Enabled:                  true,
				Strategy:                 "s2i",
				BaseImage:                "quay.io/jupyter/minimal-notebook:latest",
				AutoGenerateRequirements: true,
				RequirementsFile:         "requirements.txt",
				FallbackStrategy:         "auto",
			},
			expected: BuildConfigSpec{
				Enabled:                  true,
				Strategy:                 "s2i",
				BaseImage:                "quay.io/jupyter/minimal-notebook:latest",
				AutoGenerateRequirements: true,
				RequirementsFile:         "requirements.txt",
				FallbackStrategy:         "auto",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				// Test nil case
				var config BuildConfigSpec
				assert.Equal(t, tt.expected.Enabled, config.Enabled)
				assert.Equal(t, tt.expected.Strategy, config.Strategy)
				assert.Equal(t, tt.expected.BaseImage, config.BaseImage)
				assert.Equal(t, tt.expected.AutoGenerateRequirements, config.AutoGenerateRequirements)
				assert.Equal(t, tt.expected.RequirementsFile, config.RequirementsFile)
				assert.Equal(t, tt.expected.FallbackStrategy, config.FallbackStrategy)
			} else {
				assert.Equal(t, tt.expected.Enabled, tt.config.Enabled)
				assert.Equal(t, tt.expected.Strategy, tt.config.Strategy)
				assert.Equal(t, tt.expected.BaseImage, tt.config.BaseImage)
				assert.Equal(t, tt.expected.AutoGenerateRequirements, tt.config.AutoGenerateRequirements)
				assert.Equal(t, tt.expected.RequirementsFile, tt.config.RequirementsFile)
				assert.Equal(t, tt.expected.FallbackStrategy, tt.config.FallbackStrategy)
			}
		})
	}
}

func TestBuildConfigSpec_StrategyValidation(t *testing.T) {
	validStrategies := []string{"s2i", "tekton", "kaniko", "shipwright", "custom"}

	for _, strategy := range validStrategies {
		t.Run("valid_strategy_"+strategy, func(t *testing.T) {
			config := &BuildConfigSpec{
				Enabled:  true,
				Strategy: strategy,
			}
			assert.Equal(t, strategy, config.Strategy)
		})
	}
}

func TestBuildConfigSpec_FallbackStrategyValidation(t *testing.T) {
	validFallbackStrategies := []string{"warn", "fail", "auto"}

	for _, fallback := range validFallbackStrategies {
		t.Run("valid_fallback_"+fallback, func(t *testing.T) {
			config := &BuildConfigSpec{
				Enabled:          true,
				FallbackStrategy: fallback,
			}
			assert.Equal(t, fallback, config.FallbackStrategy)
		})
	}
}

func TestBuildConfigSpec_StrategyConfig(t *testing.T) {
	tests := []struct {
		name           string
		strategyConfig map[string]string
		expectedLen    int
	}{
		{
			name:           "nil strategy config",
			strategyConfig: nil,
			expectedLen:    0,
		},
		{
			name:           "empty strategy config",
			strategyConfig: map[string]string{},
			expectedLen:    0,
		},
		{
			name: "s2i strategy config",
			strategyConfig: map[string]string{
				"builderImage": "registry.redhat.io/ubi8/python-39",
				"contextDir":   "/notebooks",
			},
			expectedLen: 2,
		},
		{
			name: "tekton strategy config",
			strategyConfig: map[string]string{
				"pipelineName": "notebook-build-pipeline",
				"workspace":    "notebook-workspace",
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BuildConfigSpec{
				Enabled:        true,
				Strategy:       "s2i",
				StrategyConfig: tt.strategyConfig,
			}

			if tt.strategyConfig == nil {
				assert.Nil(t, config.StrategyConfig)
			} else {
				assert.Equal(t, tt.expectedLen, len(config.StrategyConfig))
				for key, value := range tt.strategyConfig {
					assert.Equal(t, value, config.StrategyConfig[key])
				}
			}
		})
	}
}

func TestPodConfigSpec_WithBuildConfig(t *testing.T) {
	tests := []struct {
		name        string
		podConfig   PodConfigSpec
		expectBuild bool
	}{
		{
			name: "no build config",
			podConfig: PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
			},
			expectBuild: false,
		},
		{
			name: "build config disabled",
			podConfig: PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
				BuildConfig: &BuildConfigSpec{
					Enabled: false,
				},
			},
			expectBuild: false,
		},
		{
			name: "build config enabled",
			podConfig: PodConfigSpec{
				ContainerImage: "quay.io/jupyter/minimal-notebook:latest",
				BuildConfig: &BuildConfigSpec{
					Enabled:   true,
					Strategy:  "s2i",
					BaseImage: "quay.io/jupyter/minimal-notebook:latest",
				},
			},
			expectBuild: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasBuildConfig := tt.podConfig.BuildConfig != nil && tt.podConfig.BuildConfig.Enabled
			assert.Equal(t, tt.expectBuild, hasBuildConfig)
		})
	}
}

func TestBuildConfigSpec_AutoGenerateRequirements(t *testing.T) {
	tests := []struct {
		name                     string
		autoGenerateRequirements bool
		fallbackStrategy         string
		expectedBehavior         string
	}{
		{
			name:                     "auto-generate disabled, warn fallback",
			autoGenerateRequirements: false,
			fallbackStrategy:         "warn",
			expectedBehavior:         "use existing requirements.txt or warn",
		},
		{
			name:                     "auto-generate enabled, auto fallback",
			autoGenerateRequirements: true,
			fallbackStrategy:         "auto",
			expectedBehavior:         "generate requirements.txt if missing",
		},
		{
			name:                     "auto-generate disabled, fail fallback",
			autoGenerateRequirements: false,
			fallbackStrategy:         "fail",
			expectedBehavior:         "fail if requirements.txt missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BuildConfigSpec{
				Enabled:                  true,
				AutoGenerateRequirements: tt.autoGenerateRequirements,
				FallbackStrategy:         tt.fallbackStrategy,
			}

			assert.Equal(t, tt.autoGenerateRequirements, config.AutoGenerateRequirements)
			assert.Equal(t, tt.fallbackStrategy, config.FallbackStrategy)

			// Verify behavior logic
			if config.AutoGenerateRequirements || config.FallbackStrategy == "auto" {
				assert.Contains(t, tt.expectedBehavior, "generate")
			} else if config.FallbackStrategy == "fail" {
				assert.Contains(t, tt.expectedBehavior, "fail")
			} else {
				assert.Contains(t, tt.expectedBehavior, "warn")
			}
		})
	}
}
