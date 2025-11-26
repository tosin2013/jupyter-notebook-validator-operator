package build

import (
	"testing"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// ValidateConfigTestCase represents a test case for ValidateConfig
type ValidateConfigTestCase struct {
	Name        string
	Config      *mlopsv1alpha1.BuildConfigSpec
	ExpectError bool
}

// RunValidateConfigTests runs a set of ValidateConfig test cases
func RunValidateConfigTests(t *testing.T, strategy Strategy, tests []ValidateConfigTestCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := strategy.ValidateConfig(tt.Config)
			if tt.ExpectError && err == nil {
				t.Error("ValidateConfig() expected error, got nil")
			}
			if !tt.ExpectError && err != nil {
				t.Errorf("ValidateConfig() unexpected error = %v", err)
			}
		})
	}
}
