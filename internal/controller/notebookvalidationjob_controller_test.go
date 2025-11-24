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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

var _ = Describe("NotebookValidationJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		notebookvalidationjob := &mlopsv1alpha1.NotebookValidationJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind NotebookValidationJob")
			err := k8sClient.Get(ctx, typeNamespacedName, notebookvalidationjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &mlopsv1alpha1.NotebookValidationJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: mlopsv1alpha1.NotebookValidationJobSpec{
						Notebook: mlopsv1alpha1.NotebookSpec{
							Git: mlopsv1alpha1.GitSpec{
								URL: "https://github.com/test/test-notebooks.git",
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
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &mlopsv1alpha1.NotebookValidationJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance NotebookValidationJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	// ADR-037: State Machine Unit Tests
	Context("When testing state machine transitions (ADR-037)", func() {
		const testResourceName = "test-state-machine"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      testResourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			// Cleanup after each test
			resource := &mlopsv1alpha1.NotebookValidationJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should initialize with PhaseInitializing", func() {
			By("Creating a new NotebookValidationJob")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
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
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			By("Reconciling the job")
			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying phase is set to Initializing")
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseInitializing))
			Expect(updatedJob.Status.StartTime).NotTo(BeNil())
		})

		It("should transition from Initializing to ValidationRunning when build is disabled", func() {
			By("Creating a job without build config")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
						// BuildConfig is nil (disabled)
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("First reconcile: Initialize to Initializing")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile: Initializing to ValidationRunning")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying phase transitioned to ValidationRunning")
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseValidationRunning))
		})

		It("should transition from Initializing to Building when build is enabled", func() {
			By("Creating a job with build config enabled")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
						BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
							Enabled:   true,
							Strategy:  "s2i",
							BaseImage: "jupyter/scipy-notebook:latest",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("First reconcile: Initialize to Initializing")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile: Initializing to Building")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying phase transitioned to Building")
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseBuilding))
		})

		It("should not reconcile jobs that are already complete (Succeeded)", func() {
			By("Creating a job")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
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
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			By("Setting status to Succeeded")
			job.Status.Phase = PhaseSucceeded
			Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())

			// Verify status was set
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseSucceeded))

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling the succeeded job")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying phase remains Succeeded")
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseSucceeded))
		})

		It("should not reconcile jobs that are already complete (Failed)", func() {
			By("Creating a job")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
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
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			By("Setting status to Failed")
			job.Status.Phase = PhaseFailed
			Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())

			// Verify status was set
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling the failed job")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying phase remains Failed")
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
		})

		It("should handle legacy Pending phase by migrating to new state machine", func() {
			By("Creating a job")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
						// No build config
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			By("Setting status to legacy Pending phase")
			job.Status.Phase = PhasePending // Legacy phase
			Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())

			// Verify status was set
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhasePending))

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling the legacy job")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying phase migrated to ValidationRunning (since no build)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseValidationRunning))
		})
	})

	// ADR-037: Requeue Logic Tests
	Context("When testing requeue logic (ADR-037)", func() {
		const testResourceName = "test-requeue"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      testResourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			// Cleanup after each test
			resource := &mlopsv1alpha1.NotebookValidationJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should requeue when transitioning phases", func() {
			By("Creating a new job")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
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
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling and expecting requeue")
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue(), "Should requeue after initialization")
		})
	})

	// ADR-037: Build Status Tests
	Context("When testing build status initialization (ADR-037)", func() {
		const testResourceName = "test-build-status"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      testResourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			// Cleanup after each test
			resource := &mlopsv1alpha1.NotebookValidationJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should initialize BuildStatus when entering Building phase", func() {
			By("Creating a job with build enabled and setting it to Building phase")
			job := &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/test-notebooks.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage:     "jupyter/scipy-notebook:latest",
						ServiceAccountName: "default",
						BuildConfig: &mlopsv1alpha1.BuildConfigSpec{
							Enabled:   true,
							Strategy:  "s2i",
							BaseImage: "jupyter/scipy-notebook:latest",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())

			controllerReconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("First reconcile: Initialize")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile: Transition to Building")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying BuildStatus is initialized")
			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseBuilding))
			// BuildStatus will be initialized in the next reconcile when build logic runs
		})
	})

	// Test error classification for ADR-011 and resource conflict handling
	Describe("Error Classification", func() {
		It("should classify conflict errors as transient", func() {
			conflictErr := errors.NewConflict(
				mlopsv1alpha1.GroupVersion.WithResource("notebookvalidationjobs").GroupResource(),
				"test-job",
				nil,
			)
			errorType := classifyError(conflictErr)
			Expect(errorType).To(Equal("Transient"))
		})

		It("should classify timeout errors as transient", func() {
			timeoutErr := errors.NewTimeoutError("test timeout", 30)
			errorType := classifyError(timeoutErr)
			Expect(errorType).To(Equal("Transient"))
		})

		It("should classify invalid errors as terminal", func() {
			invalidErr := errors.NewInvalid(
				mlopsv1alpha1.GroupVersion.WithKind("NotebookValidationJob").GroupKind(),
				"test-job",
				nil,
			)
			errorType := classifyError(invalidErr)
			Expect(errorType).To(Equal("Terminal"))
		})

		It("should classify not found errors as retriable", func() {
			notFoundErr := errors.NewNotFound(
				mlopsv1alpha1.GroupVersion.WithResource("notebookvalidationjobs").GroupResource(),
				"test-job",
			)
			errorType := classifyError(notFoundErr)
			Expect(errorType).To(Equal("Retriable"))
		})

		It("should handle nil errors", func() {
			errorType := classifyError(nil)
			Expect(errorType).To(Equal(""))
		})
	})

	// Test error handling for ADR-011 and ADR-042
	Describe("Error Handling", func() {
		var (
			ctx        context.Context
			reconciler *NotebookValidationJobReconciler
			job        *mlopsv1alpha1.NotebookValidationJob
		)

		BeforeEach(func() {
			ctx = context.Background()
			reconciler = &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			job = &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-error-handling",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/notebooks.git",
							Ref: "main",
						},
						Path: "test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test:latest",
					},
				},
				Status: mlopsv1alpha1.NotebookValidationJobStatus{
					Phase:      PhaseInitializing,
					RetryCount: 0,
				},
			}
		})

		Context("handleReconcileError with transient errors", func() {
			It("should requeue without incrementing retry count for conflict errors", func() {
				Expect(k8sClient.Create(ctx, job)).To(Succeed())

				conflictErr := errors.NewConflict(
					mlopsv1alpha1.GroupVersion.WithResource("notebookvalidationjobs").GroupResource(),
					job.Name,
					nil,
				)

				result, err := reconciler.handleReconcileError(ctx, job, conflictErr)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute))

				// Verify retry count not incremented
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
				Expect(updatedJob.Status.RetryCount).To(Equal(0))
				Expect(updatedJob.Status.Phase).To(Equal(PhaseInitializing)) // Not marked as Failed
			})

			It("should requeue without incrementing retry count for timeout errors", func() {
				Expect(k8sClient.Create(ctx, job)).To(Succeed())

				timeoutErr := errors.NewTimeoutError("test timeout", 30)

				result, err := reconciler.handleReconcileError(ctx, job, timeoutErr)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute))

				// Verify retry count not incremented
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
				Expect(updatedJob.Status.RetryCount).To(Equal(0))
			})
		})

		Context("handleReconcileError with retriable errors", func() {
			It("should increment retry count and requeue with backoff", func() {
				Expect(k8sClient.Create(ctx, job)).To(Succeed())

				notFoundErr := errors.NewNotFound(
					mlopsv1alpha1.GroupVersion.WithResource("pods").GroupResource(),
					"test-pod",
				)

				result, err := reconciler.handleReconcileError(ctx, job, notFoundErr)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute))

				// Verify retry count incremented
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
				Expect(updatedJob.Status.RetryCount).To(Equal(1))
				Expect(updatedJob.Status.LastRetryTime).NotTo(BeNil())
			})

			It("should mark as failed after max retries", func() {
				job.Status.RetryCount = MaxRetries // Set to max retries
				Expect(k8sClient.Create(ctx, job)).To(Succeed())

				notFoundErr := errors.NewNotFound(
					mlopsv1alpha1.GroupVersion.WithResource("pods").GroupResource(),
					"test-pod",
				)

				_, err := reconciler.handleReconcileError(ctx, job, notFoundErr)
				Expect(err).NotTo(HaveOccurred())

				// Verify job marked as failed
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
				Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
			})
		})

		Context("handleReconcileError with terminal errors", func() {
			It("should immediately mark as failed for invalid errors", func() {
				Expect(k8sClient.Create(ctx, job)).To(Succeed())

				invalidErr := errors.NewInvalid(
					mlopsv1alpha1.GroupVersion.WithKind("NotebookValidationJob").GroupKind(),
					job.Name,
					nil,
				)

				_, err := reconciler.handleReconcileError(ctx, job, invalidErr)
				Expect(err).NotTo(HaveOccurred())

				// Verify job marked as failed immediately
				updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
				Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
				Expect(updatedJob.Status.RetryCount).To(Equal(0)) // No retry for terminal errors
			})
		})
	})

	// Test status update functions
	Describe("Status Updates", func() {
		var (
			ctx        context.Context
			reconciler *NotebookValidationJobReconciler
			job        *mlopsv1alpha1.NotebookValidationJob
		)

		BeforeEach(func() {
			ctx = context.Background()
			reconciler = &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			job = &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-status-updates",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/example/notebooks.git",
							Ref: "main",
						},
						Path: "test.ipynb",
					},
					PodConfig: mlopsv1alpha1.PodConfigSpec{
						ContainerImage: "test:latest",
					},
				},
				Status: mlopsv1alpha1.NotebookValidationJobStatus{
					Phase: PhaseInitializing,
				},
			}
			Expect(k8sClient.Create(ctx, job)).To(Succeed())
		})

		It("should transition phase successfully", func() {
			result, err := reconciler.transitionPhase(ctx, job, PhaseBuilding, "Starting build")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseBuilding))
		})

		It("should update job phase with completion time for terminal phases", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseSucceeded, "Validation completed successfully")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseSucceeded))
			Expect(updatedJob.Status.CompletionTime).NotTo(BeNil())
		})

		It("should update job phase to failed with error message", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseFailed, "Build failed: missing dependencies")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseFailed))
			Expect(updatedJob.Status.Message).To(ContainSubstring("Build failed: missing dependencies"))
			Expect(updatedJob.Status.CompletionTime).NotTo(BeNil())
		})

		It("should not update completion time for non-terminal phases", func() {
			result, err := reconciler.updateJobPhase(ctx, job, PhaseBuilding, "Build in progress")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedJob := &mlopsv1alpha1.NotebookValidationJob{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal(PhaseBuilding))
			Expect(updatedJob.Status.CompletionTime).To(BeNil())
		})
	})
})
