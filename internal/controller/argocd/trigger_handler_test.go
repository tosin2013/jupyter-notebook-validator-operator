package argocd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTriggers(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      []ResourceTrigger
		wantError bool
	}{
		{
			name: "valid single trigger",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  name: my-model
  namespace: default
  action: restart
`,
			want: []ResourceTrigger{
				{
					APIVersion: "serving.kserve.io/v1beta1",
					Kind:       "InferenceService",
					Name:       "my-model",
					Namespace:  "default",
					Action:     "restart",
				},
			},
			wantError: false,
		},
		{
			name: "valid multiple triggers",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  name: model1
  action: restart
- apiVersion: apps/v1
  kind: Deployment
  name: consumer
  action: refresh
`,
			want: []ResourceTrigger{
				{
					APIVersion: "serving.kserve.io/v1beta1",
					Kind:       "InferenceService",
					Name:       "model1",
					Action:     "restart",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "consumer",
					Action:     "refresh",
				},
			},
			wantError: false,
		},
		{
			name: "missing apiVersion",
			yaml: `
- kind: InferenceService
  name: my-model
  action: restart
`,
			wantError: true,
		},
		{
			name: "missing kind",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  name: my-model
  action: restart
`,
			wantError: true,
		},
		{
			name: "missing name",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  action: restart
`,
			wantError: true,
		},
		{
			name: "missing action",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  name: my-model
`,
			wantError: true,
		},
		{
			name: "invalid action",
			yaml: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  name: my-model
  action: invalid
`,
			wantError: true,
		},
		{
			name: "sync action",
			yaml: `
- apiVersion: argoproj.io/v1alpha1
  kind: Application
  name: my-app
  action: sync
`,
			want: []ResourceTrigger{
				{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "Application",
					Name:       "my-app",
					Action:     "sync",
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTriggers(tt.yaml)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseGroupVersion(t *testing.T) {
	tests := []struct {
		name      string
		apiVersion string
		wantGroup  string
		wantVersion string
		wantError  bool
	}{
		{
			name:       "core API group",
			apiVersion: "v1",
			wantGroup:  "",
			wantVersion: "v1",
			wantError:  false,
		},
		{
			name:       "group/version format",
			apiVersion: "apps/v1",
			wantGroup:  "apps",
			wantVersion: "v1",
			wantError:  false,
		},
		{
			name:       "full group/version",
			apiVersion: "serving.kserve.io/v1beta1",
			wantGroup:  "serving.kserve.io",
			wantVersion: "v1beta1",
			wantError:  false,
		},
		{
			name:       "empty string",
			apiVersion: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGroupVersion(tt.apiVersion)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantGroup, got.Group)
				assert.Equal(t, tt.wantVersion, got.Version)
			}
		})
	}
}
