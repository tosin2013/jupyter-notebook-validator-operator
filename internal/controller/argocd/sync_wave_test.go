package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestSetSyncWaveAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name           string
		job            *mlopsv1alpha1.NotebookValidationJob
		phase          string
		wantComplete   string
		wantFailed     string
		wantCompletion bool
	}{
		{
			name: "succeeded with sync wave",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					Annotations: map[string]string{
						AnnotationSyncWave: "3",
					},
				},
			},
			phase:          "Succeeded",
			wantComplete:   "3",
			wantFailed:     "",
			wantCompletion: true,
		},
		{
			name: "failed with sync wave",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					Annotations: map[string]string{
						AnnotationSyncWave: "3",
					},
				},
			},
			phase:          "Failed",
			wantComplete:   "",
			wantFailed:     "3",
			wantCompletion: true,
		},
		{
			name: "no sync wave annotation",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-job",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			},
			phase:          "Succeeded",
			wantComplete:   "",
			wantFailed:     "",
			wantCompletion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.job).Build()

			err := SetSyncWaveAnnotations(context.Background(), fakeClient, tt.job, tt.phase)
			assert.NoError(t, err)

			// Fetch updated job
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(tt.job), updatedJob)
			assert.NoError(t, err)

			annotations := updatedJob.GetAnnotations()
			if tt.wantCompletion {
				assert.NotEmpty(t, annotations[AnnotationCompletionTime])
			}

			if tt.wantComplete != "" {
				assert.Equal(t, tt.wantComplete, annotations[AnnotationWaveComplete])
				assert.Empty(t, annotations[AnnotationWaveFailed])
			}

			if tt.wantFailed != "" {
				assert.Equal(t, tt.wantFailed, annotations[AnnotationWaveFailed])
				assert.Empty(t, annotations[AnnotationWaveComplete])
			}
		})
	}
}
