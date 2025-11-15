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

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestNotebookValidationJobController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NotebookValidationJob Controller Unit Tests")
}

var _ = Describe("NotebookValidationJobReconciler", func() {
	var (
		ctx        context.Context
		reconciler *NotebookValidationJobReconciler
		scheme     *runtime.Scheme
		fakeClient client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = mlopsv1alpha1.AddToScheme(scheme)

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&mlopsv1alpha1.NotebookValidationJob{}).
			Build()
		reconciler = &NotebookValidationJobReconciler{
			Client:     fakeClient,
			Scheme:     scheme,
			RestConfig: &rest.Config{},
		}
	})

	Describe("Reconcile", func() {
		Context("when resource is not found", func() {
			It("should return without error", func() {
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "non-existent",
						Namespace: "default",
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
			})
		})

		Context("when resource exists", func() {
			var job *mlopsv1alpha1.NotebookValidationJob

			BeforeEach(func() {
				job = &mlopsv1alpha1.NotebookValidationJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job",
						Namespace: "default",
					},
					Spec: mlopsv1alpha1.NotebookValidationJobSpec{
						Notebook: mlopsv1alpha1.NotebookSpec{
							Git: mlopsv1alpha1.GitSpec{
								URL: "https://github.com/test/repo.git",
								Ref: "main",
							},
							Path: "notebooks/test.ipynb",
						},
						PodConfig: mlopsv1alpha1.PodConfigSpec{
							ContainerImage:     "jupyter/scipy-notebook:latest",
							ServiceAccountName: "default",
						},
					},
				}
				Expect(fakeClient.Create(ctx, job)).To(Succeed())
			})

			It("should initialize status if phase is empty", func() {
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      job.Name,
						Namespace: job.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeTrue())

				// Verify status was initialized
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{
					Name:      job.Name,
					Namespace: job.Namespace,
				}, updatedJob)).To(Succeed())

				Expect(updatedJob.Status.Phase).To(Equal(PhasePending))
				Expect(updatedJob.Status.StartTime).NotTo(BeNil())
				Expect(updatedJob.Status.RetryCount).To(Equal(0))
				Expect(len(updatedJob.Status.Conditions)).To(BeNumerically(">", 0))
			})

			It("should not reconcile if phase is Succeeded", func() {
				job.Status.Phase = PhaseSucceeded
				Expect(fakeClient.Status().Update(ctx, job)).To(Succeed())

				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      job.Name,
						Namespace: job.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
			})

			It("should not reconcile if phase is Failed", func() {
				job.Status.Phase = PhaseFailed
				Expect(fakeClient.Status().Update(ctx, job)).To(Succeed())

				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      job.Name,
						Namespace: job.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
			})

			It("should fail if max retries exceeded", func() {
				job.Status.Phase = PhaseRunning
				job.Status.RetryCount = MaxRetries
				Expect(fakeClient.Status().Update(ctx, job)).To(Succeed())

				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      job.Name,
						Namespace: job.Namespace,
					},
				}

				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify phase was updated to Failed
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(fakeClient.Get(ctx, types.NamespacedName{
					Name:      job.Name,
					Namespace: job.Namespace,
				}, updatedJob)).To(Succeed())

				Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
			})
		})
	})

	Describe("updateJobPhase", func() {
		var job *mlopsv1alpha1.NotebookValidationJob

		BeforeEach(func() {
			job = &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/repo.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
					},
				},
				Status: mlopsv1alpha1.NotebookValidationJobStatus{
					Phase: PhasePending,
				},
			}
			Expect(fakeClient.Create(ctx, job)).To(Succeed())
		})

		It("should update phase to Running", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseRunning, "Starting validation")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())

			Expect(updatedJob.Status.Phase).To(Equal(PhaseRunning))
			Expect(updatedJob.Status.Message).To(Equal("Starting validation"))
		})

		It("should update phase to Succeeded", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseSucceeded, "Validation completed")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())

			Expect(updatedJob.Status.Phase).To(Equal(PhaseSucceeded))
			Expect(updatedJob.Status.Message).To(Equal("Validation completed"))
			Expect(updatedJob.Status.CompletionTime).NotTo(BeNil())
		})

		It("should update phase to Failed", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseFailed, "Validation failed")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())

			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
			Expect(updatedJob.Status.Message).To(Equal("Validation failed"))
			Expect(updatedJob.Status.CompletionTime).NotTo(BeNil())
		})

		It("should update conditions when phase changes", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseRunning, "Starting")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())

			// Check that conditions were updated
			Expect(len(updatedJob.Status.Conditions)).To(BeNumerically(">", 0))
			validationCondition := findCondition(updatedJob.Status.Conditions, ConditionTypeValidationComplete)
			Expect(validationCondition).NotTo(BeNil())
			Expect(validationCondition.Status).To(Equal(metav1.ConditionUnknown))
		})
	})

	Describe("createValidationPod", func() {
		var job *mlopsv1alpha1.NotebookValidationJob

		BeforeEach(func() {
			job = &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/repo.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
					},
				},
			}
			Expect(fakeClient.Create(ctx, job)).To(Succeed())
		})

		It("should create a validation pod with correct name", func() {
			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())
			Expect(pod.Name).To(Equal("test-job-validation"))
			Expect(pod.Namespace).To(Equal("default"))
		})

		It("should create pod with git clone init container", func() {
			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			Expect(len(pod.Spec.InitContainers)).To(BeNumerically(">=", 1))
			gitCloneContainer := pod.Spec.InitContainers[0]
			Expect(gitCloneContainer.Name).To(Equal("git-clone"))
			Expect(gitCloneContainer.Image).To(Equal("alpine/git:latest"))
		})

		It("should create pod with validator container", func() {
			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			Expect(len(pod.Spec.Containers)).To(Equal(1))
			validatorContainer := pod.Spec.Containers[0]
			Expect(validatorContainer.Name).To(Equal("validator"))
			Expect(validatorContainer.Image).To(Equal("jupyter/scipy-notebook:latest"))
		})

		It("should create pod with workspace volume", func() {
			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			Expect(len(pod.Spec.Volumes)).To(BeNumerically(">=", 1))
			workspaceVolume := findVolume(pod.Spec.Volumes, "workspace")
			Expect(workspaceVolume).NotTo(BeNil())
			Expect(workspaceVolume.EmptyDir).NotTo(BeNil())
		})

		It("should set owner reference on pod", func() {
			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			Expect(len(pod.OwnerReferences)).To(Equal(1))
			ownerRef := pod.OwnerReferences[0]
			Expect(ownerRef.Name).To(Equal(job.Name))
			Expect(ownerRef.Kind).To(Equal("NotebookValidationJob"))
		})

		It("should include golden notebook init container when specified", func() {
			job.Spec.GoldenNotebook = &mlopsv1alpha1.NotebookSpec{
				Git: mlopsv1alpha1.GitSpec{
					URL: "https://github.com/test/golden.git",
					Ref: "main",
				},
				Path: "golden.ipynb",
			}
			Expect(fakeClient.Update(ctx, job)).To(Succeed())

			pod, err := reconciler.createValidationPod(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			Expect(len(pod.Spec.InitContainers)).To(BeNumerically(">=", 2))
			goldenContainer := findInitContainer(pod.Spec.InitContainers, "golden-git-clone")
			Expect(goldenContainer).NotTo(BeNil())
		})
	})

	Describe("handleReconcileError", func() {
		var job *mlopsv1alpha1.NotebookValidationJob

		BeforeEach(func() {
			job = &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/repo.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
					},
				},
				Status: mlopsv1alpha1.NotebookValidationJobStatus{
					Phase:      PhaseRunning,
					RetryCount: 0,
				},
			}
			Expect(fakeClient.Create(ctx, job)).To(Succeed())
		})

		It("should handle transient errors with requeue", func() {
			err := k8serrors.NewServerTimeout(corev1.Resource("pod"), "get", 0)
			result, handleErr := reconciler.handleReconcileError(ctx, job, err)
			Expect(handleErr).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})

		It("should handle retriable errors with backoff", func() {
			err := k8serrors.NewInternalError(errors.New("internal error"))
			result, handleErr := reconciler.handleReconcileError(ctx, job, err)
			Expect(handleErr).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify retry count was incremented
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.RetryCount).To(Equal(1))
		})

		It("should mark job as failed when max retries exceeded", func() {
			job.Status.RetryCount = MaxRetries - 1
			Expect(fakeClient.Status().Update(ctx, job)).To(Succeed())

			err := k8serrors.NewInternalError(errors.New("internal error"))
			_, handleErr := reconciler.handleReconcileError(ctx, job, err)
			Expect(handleErr).NotTo(HaveOccurred())

			// Verify job was marked as failed
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
		})

		It("should handle terminal errors by marking job as failed", func() {
			err := k8serrors.NewBadRequest("bad request")
			_, handleErr := reconciler.handleReconcileError(ctx, job, err)
			Expect(handleErr).NotTo(HaveOccurred())

			// Verify job was marked as failed
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
		})

		It("should apply exponential backoff for retriable errors", func() {
			// First retry (retryCount starts at 0, becomes 1 after increment)
			job.Status.RetryCount = 0
			Expect(fakeClient.Status().Update(ctx, job)).To(Succeed())

			err := k8serrors.NewInternalError(errors.New("internal error"))
			result, handleErr := reconciler.handleReconcileError(ctx, job, err)
			Expect(handleErr).NotTo(HaveOccurred())
			// After first error, retryCount becomes 1, backoff = 1 << (1-1) = 1 minute
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			// Get updated job to check retry count
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(fakeClient.Get(ctx, types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			}, updatedJob)).To(Succeed())

			// Second retry (retryCount is now 1, becomes 2 after increment)
			updatedJob.Status.RetryCount = 1
			Expect(fakeClient.Status().Update(ctx, updatedJob)).To(Succeed())

			err2 := k8serrors.NewInternalError(errors.New("internal error"))
			result, handleErr = reconciler.handleReconcileError(ctx, updatedJob, err2)
			Expect(handleErr).NotTo(HaveOccurred())
			// After second error, retryCount becomes 2, backoff = 1 << (2-1) = 2 minutes
			Expect(result.RequeueAfter).To(Equal(2 * time.Minute))

			// Third retry (retryCount is now 2, becomes 3 after increment)
			updatedJob.Status.RetryCount = 2
			Expect(fakeClient.Status().Update(ctx, updatedJob)).To(Succeed())

			err3 := k8serrors.NewInternalError(errors.New("internal error"))
			result, handleErr = reconciler.handleReconcileError(ctx, updatedJob, err3)
			Expect(handleErr).NotTo(HaveOccurred())
			// After third error, retryCount becomes 3, backoff = 1 << (3-1) = 4 minutes
			Expect(result.RequeueAfter).To(Equal(4 * time.Minute))
		})
	})
})

// Helper functions
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func findVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == name {
			return &volumes[i]
		}
	}
	return nil
}

func findInitContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}
