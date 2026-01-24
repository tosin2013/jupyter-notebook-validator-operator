package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestTriggerHandler_ExecuteTriggers_Integration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	tests := []struct {
		name            string
		job             *mlopsv1alpha1.NotebookValidationJob
		existingPods    []client.Object
		wantPodsDeleted bool
		wantError       bool
	}{
		{
			name: "restart InferenceService - deletes pods",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					Annotations: map[string]string{
						AnnotationOnSuccessTrigger: `
- apiVersion: serving.kserve.io/v1beta1
  kind: InferenceService
  name: my-model
  action: restart
`,
					},
				},
			},
			existingPods: []client.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-model-predictor-0",
						Namespace: "default",
						Labels: map[string]string{
							"serving.kserve.io/inferenceservice": "my-model",
						},
					},
				},
			},
			wantPodsDeleted: true,
			wantError:       false,
		},
		{
			name: "restart Deployment - deletes pods",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					Annotations: map[string]string{
						AnnotationOnSuccessTrigger: `
- apiVersion: apps/v1
  kind: Deployment
  name: my-deployment
  action: restart
`,
					},
				},
			},
			existingPods: []client.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment-pod-0",
						Namespace: "default",
						Labels: map[string]string{
							"app": "my-app",
						},
					},
				},
			},
			wantPodsDeleted: true,
			wantError:       false,
		},
		{
			name: "no triggers configured",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-job",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			},
			wantPodsDeleted: false,
			wantError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake client with existing pods
			objs := []client.Object{tt.job}
			objs = append(objs, tt.existingPods...)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

			handler := NewTriggerHandler(client)

			// Execute triggers
			err := handler.ExecuteTriggers(context.Background(), tt.job)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify pods were deleted if expected
				if tt.wantPodsDeleted {
					for _, pod := range tt.existingPods {
						podObj := pod.(*corev1.Pod)
						err := client.Get(context.Background(), client.ObjectKeyFromObject(podObj), &corev1.Pod{})
						assert.Error(t, err, "Pod should be deleted")
						assert.True(t, client.IgnoreNotFound(err) == nil, "Error should be NotFound")
					}
				}
			}
		})
	}
}

func TestStatusAggregator_UpdateApplicationStatus_Integration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name           string
		jobs           []*mlopsv1alpha1.NotebookValidationJob
		wantAnnotation bool
		wantError      bool
	}{
		{
			name: "aggregate status from multiple jobs",
			jobs: []*mlopsv1alpha1.NotebookValidationJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "job1",
						Namespace: "default",
						Labels: map[string]string{
							LabelPartOf: "my-app",
						},
					},
					Status: mlopsv1alpha1.NotebookValidationJobStatus{
						Phase: "Succeeded",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "job2",
						Namespace: "default",
						Labels: map[string]string{
							LabelPartOf: "my-app",
						},
					},
					Status: mlopsv1alpha1.NotebookValidationJobStatus{
						Phase: "Failed",
					},
				},
			},
			wantAnnotation: true,
			wantError:      false,
		},
		{
			name: "no parent application",
			jobs: []*mlopsv1alpha1.NotebookValidationJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "job1",
						Namespace: "default",
						Labels:    map[string]string{},
					},
					Status: mlopsv1alpha1.NotebookValidationJobStatus{
						Phase: "Succeeded",
					},
				},
			},
			wantAnnotation: false,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := make([]client.Object, len(tt.jobs))
			for i, job := range tt.jobs {
				objs[i] = job
			}

			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			aggregator := NewStatusAggregator(client)

			// Update status
			err := aggregator.UpdateApplicationStatus(context.Background(), "default")

			if tt.wantError {
				assert.Error(t, err)
			} else {
				// Status aggregation is optional - it's OK if Application doesn't exist
				// The function should not error in this case
				assert.NoError(t, err)
			}
		})
	}
}
