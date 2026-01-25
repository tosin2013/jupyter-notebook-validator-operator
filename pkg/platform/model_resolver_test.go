// Package platform provides model serving platform detection and validation
package platform

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// testLoggerKey is a custom type for context key to avoid staticcheck SA1029
type testLoggerKey struct{}

func TestResolveModelReference(t *testing.T) {
	// Set up a logger for tests
	logger := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	ctx = context.WithValue(ctx, testLoggerKey{}, logger)

	tests := []struct {
		name              string
		modelRef          string
		jobNamespace      string
		allowCrossNS      bool
		allowedNamespaces []string
		wantName          string
		wantNamespace     string
		wantErr           bool
	}{
		{
			name:          "simple model name",
			modelRef:      "fraud-detection-model",
			jobNamespace:  "mlops",
			allowCrossNS:  false,
			wantName:      "fraud-detection-model",
			wantNamespace: "mlops",
			wantErr:       false,
		},
		{
			name:          "model with same namespace prefix",
			modelRef:      "mlops/fraud-detection-model",
			jobNamespace:  "mlops",
			allowCrossNS:  false,
			wantName:      "fraud-detection-model",
			wantNamespace: "mlops",
			wantErr:       false,
		},
		{
			name:          "cross-namespace denied",
			modelRef:      "other-ns/fraud-detection-model",
			jobNamespace:  "mlops",
			allowCrossNS:  false,
			wantName:      "",
			wantNamespace: "",
			wantErr:       true,
		},
		{
			name:          "cross-namespace allowed",
			modelRef:      "ml-models/fraud-detection-model",
			jobNamespace:  "mlops",
			allowCrossNS:  true,
			wantName:      "fraud-detection-model",
			wantNamespace: "ml-models",
			wantErr:       false,
		},
		{
			name:              "cross-namespace with allowed list - valid",
			modelRef:          "ml-models/fraud-detection-model",
			jobNamespace:      "mlops",
			allowCrossNS:      true,
			allowedNamespaces: []string{"ml-models", "shared-models"},
			wantName:          "fraud-detection-model",
			wantNamespace:     "ml-models",
			wantErr:           false,
		},
		{
			name:              "cross-namespace with allowed list - denied",
			modelRef:          "restricted-ns/fraud-detection-model",
			jobNamespace:      "mlops",
			allowCrossNS:      true,
			allowedNamespaces: []string{"ml-models", "shared-models"},
			wantName:          "",
			wantNamespace:     "",
			wantErr:           true,
		},
		{
			name:          "empty model reference",
			modelRef:      "",
			jobNamespace:  "mlops",
			allowCrossNS:  false,
			wantName:      "",
			wantNamespace: "",
			wantErr:       true,
		},
		{
			name:          "invalid format - double slash",
			modelRef:      "ns//model",
			jobNamespace:  "mlops",
			allowCrossNS:  true,
			wantName:      "",
			wantNamespace: "",
			wantErr:       true,
		},
		{
			name:          "invalid format - trailing slash",
			modelRef:      "ns/",
			jobNamespace:  "mlops",
			allowCrossNS:  true,
			wantName:      "",
			wantNamespace: "",
			wantErr:       true,
		},
		{
			name:          "invalid format - leading slash",
			modelRef:      "/model",
			jobNamespace:  "mlops",
			allowCrossNS:  true,
			wantName:      "",
			wantNamespace: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewModelResolver(ModelResolutionConfig{
				AllowCrossNamespace: tt.allowCrossNS,
				DefaultNamespace:    tt.jobNamespace,
				AllowedNamespaces:   tt.allowedNamespaces,
			})

			ref, err := resolver.ResolveModelReference(ctx, tt.modelRef, tt.jobNamespace)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveModelReference() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveModelReference() unexpected error: %v", err)
				return
			}

			if ref.Name != tt.wantName {
				t.Errorf("ResolveModelReference() Name = %q, want %q", ref.Name, tt.wantName)
			}
			if ref.Namespace != tt.wantNamespace {
				t.Errorf("ResolveModelReference() Namespace = %q, want %q", ref.Namespace, tt.wantNamespace)
			}
		})
	}
}

func TestResolveModelReferences(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	ctx = context.WithValue(ctx, testLoggerKey{}, logger)

	resolver := NewDefaultModelResolver("mlops")

	tests := []struct {
		name         string
		modelRefs    []string
		jobNamespace string
		wantCount    int
		wantErr      bool
	}{
		{
			name:         "multiple valid references",
			modelRefs:    []string{"model-a", "model-b", "model-c"},
			jobNamespace: "mlops",
			wantCount:    3,
			wantErr:      false,
		},
		{
			name:         "empty list",
			modelRefs:    []string{},
			jobNamespace: "mlops",
			wantCount:    0,
			wantErr:      false,
		},
		{
			name:         "mixed valid and invalid",
			modelRefs:    []string{"model-a", "other-ns/model-b", "model-c"},
			jobNamespace: "mlops",
			wantCount:    2, // model-a and model-c resolve, other-ns/model-b fails
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := resolver.ResolveModelReferences(ctx, tt.modelRefs, tt.jobNamespace)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveModelReferences() expected error but got none")
				}
			} else if err != nil {
				t.Errorf("ResolveModelReferences() unexpected error: %v", err)
			}

			if len(refs) != tt.wantCount {
				t.Errorf("ResolveModelReferences() returned %d refs, want %d", len(refs), tt.wantCount)
			}
		})
	}
}

func TestGroupByNamespace(t *testing.T) {
	refs := []*ModelReference{
		{Name: "model-a", Namespace: "ns1"},
		{Name: "model-b", Namespace: "ns1"},
		{Name: "model-c", Namespace: "ns2"},
		{Name: "model-d", Namespace: "ns3"},
	}

	grouped := GroupByNamespace(refs)

	if len(grouped) != 3 {
		t.Errorf("GroupByNamespace() returned %d namespaces, want 3", len(grouped))
	}

	if len(grouped["ns1"]) != 2 {
		t.Errorf("GroupByNamespace() ns1 has %d models, want 2", len(grouped["ns1"]))
	}

	if len(grouped["ns2"]) != 1 {
		t.Errorf("GroupByNamespace() ns2 has %d models, want 1", len(grouped["ns2"]))
	}
}

func TestGetUniqueNamespaces(t *testing.T) {
	refs := []*ModelReference{
		{Name: "model-a", Namespace: "ns1"},
		{Name: "model-b", Namespace: "ns1"},
		{Name: "model-c", Namespace: "ns2"},
	}

	namespaces := GetUniqueNamespaces(refs)

	if len(namespaces) != 2 {
		t.Errorf("GetUniqueNamespaces() returned %d namespaces, want 2", len(namespaces))
	}

	// Check both namespaces are present
	hasNS1, hasNS2 := false, false
	for _, ns := range namespaces {
		if ns == "ns1" {
			hasNS1 = true
		}
		if ns == "ns2" {
			hasNS2 = true
		}
	}

	if !hasNS1 || !hasNS2 {
		t.Errorf("GetUniqueNamespaces() missing expected namespaces: ns1=%v, ns2=%v", hasNS1, hasNS2)
	}
}

func TestModelReferenceFormatting(t *testing.T) {
	tests := []struct {
		name      string
		ref       *ModelReference
		want      string
		crossNS   bool
		jobNS     string
		wantCross bool
	}{
		{
			name:      "with namespace",
			ref:       &ModelReference{Name: "model", Namespace: "ns1"},
			want:      "ns1/model",
			crossNS:   true,
			jobNS:     "ns2",
			wantCross: true,
		},
		{
			name:      "same namespace",
			ref:       &ModelReference{Name: "model", Namespace: "ns1"},
			want:      "ns1/model",
			crossNS:   true,
			jobNS:     "ns1",
			wantCross: false,
		},
		{
			name:      "empty namespace",
			ref:       &ModelReference{Name: "model", Namespace: ""},
			want:      "model",
			crossNS:   true,
			jobNS:     "ns1",
			wantCross: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.FormatModelReference(); got != tt.want {
				t.Errorf("FormatModelReference() = %q, want %q", got, tt.want)
			}
			if got := tt.ref.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			if got := tt.ref.IsCrossNamespace(tt.jobNS); got != tt.wantCross {
				t.Errorf("IsCrossNamespace() = %v, want %v", got, tt.wantCross)
			}
		})
	}
}

func TestNewDefaultModelResolver(t *testing.T) {
	resolver := NewDefaultModelResolver("default-ns")

	if resolver.config.AllowCrossNamespace {
		t.Error("NewDefaultModelResolver() should not allow cross-namespace by default")
	}

	if resolver.config.DefaultNamespace != "default-ns" {
		t.Errorf("NewDefaultModelResolver() DefaultNamespace = %q, want %q", resolver.config.DefaultNamespace, "default-ns")
	}

	if len(resolver.config.AllowedNamespaces) != 0 {
		t.Errorf("NewDefaultModelResolver() AllowedNamespaces should be empty, got %v", resolver.config.AllowedNamespaces)
	}
}
