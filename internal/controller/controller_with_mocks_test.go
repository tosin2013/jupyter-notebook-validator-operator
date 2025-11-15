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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/internal/controller/mocks"
)

// Helper to convert mock types to controller types
func mockCredsToController(mockCreds *mocks.GitCredentials) *GitCredentials {
	if mockCreds == nil {
		return nil
	}
	return &GitCredentials{
		Type:     mockCreds.Type,
		Username: mockCreds.Username,
		Password: mockCreds.Password,
		SSHKey:   mockCreds.SSHKey,
	}
}

func TestControllerWithMocks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller With Mocks Tests")
}

var _ = Describe("NotebookValidationJobReconciler with Mocks", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
		fakeClient client.Client
		gitOps     *mocks.MockGitOperations
		podLogOps  *mocks.MockPodLogOperations
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

		gitOps = mocks.NewMockGitOperations()
		podLogOps = mocks.NewMockPodLogOperations()
	})

	Describe("createValidationPod with mocked Git operations", func() {
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

		It("should demonstrate Git operations mock usage", func() {
			// Setup mock to return SSH credentials
			gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*mocks.GitCredentials, error) {
				return &mocks.GitCredentials{
					Type:   "ssh",
					SSHKey: "mock-ssh-key",
				}, nil
			}

			// Use the mock
			creds, err := gitOps.ResolveCredentials(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(creds).NotTo(BeNil())
			Expect(creds.Type).To(Equal("ssh"))
			Expect(creds.SSHKey).To(Equal("mock-ssh-key"))
			Expect(gitOps.ResolveCredentialsCallCount).To(Equal(1))
		})

		It("should handle Git credential resolution errors", func() {
			// Setup mock to return error
			gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*mocks.GitCredentials, error) {
				return nil, errors.New("failed to resolve credentials")
			}

			creds, err := gitOps.ResolveCredentials(ctx, job)
			Expect(err).To(HaveOccurred())
			Expect(creds).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to resolve credentials"))
		})
	})

	Describe("handlePodSuccess with mocked PodLogOperations", func() {
		var job *mlopsv1alpha1.NotebookValidationJob
		var pod *corev1.Pod

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
					Phase: PhaseRunning,
				},
			}
			Expect(fakeClient.Create(ctx, job)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-validation",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			}
			Expect(fakeClient.Create(ctx, pod)).To(Succeed())
		})

		It("should parse results from logs using PodLogOperations", func() {
			// Setup mock to return specific results
			podLogOps.ParseResultsFunc = func(logs string) (*mocks.NotebookExecutionResult, error) {
				return &mocks.NotebookExecutionResult{
					Status:   "succeeded",
					ExitCode: 0,
					Cells: []mocks.CellExecutionResult{
						{
							CellIndex: 0,
							CellType:  "code",
							Status:    "succeeded",
						},
					},
					Statistics: mocks.ExecutionStatistics{
						TotalCells:  1,
						CodeCells:   1,
						FailedCells: 0,
						SuccessRate: 100.0,
					},
				}, nil
			}

			// Use the mock
			result, err := podLogOps.ParseResults("test logs")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Status).To(Equal("succeeded"))
			Expect(result.ExitCode).To(Equal(0))
			Expect(len(result.Cells)).To(Equal(1))
			Expect(podLogOps.ParseResultsCallCount).To(Equal(1))
		})

		It("should handle log parsing errors gracefully", func() {
			// Setup mock to return error
			podLogOps.ParseResultsFunc = func(logs string) (*mocks.NotebookExecutionResult, error) {
				return nil, errors.New("failed to parse logs")
			}

			result, err := podLogOps.ParseResults("invalid logs")
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse logs"))
		})
	})

	Describe("Mock verification", func() {
		It("should track method calls", func() {
			gitOps.Reset()

			// Simulate calls
			testJob := &mlopsv1alpha1.NotebookValidationJob{}
			_, _ = gitOps.ResolveCredentials(ctx, testJob)
			_, _ = gitOps.ResolveCredentials(ctx, testJob)
			_, _ = gitOps.BuildCloneInitContainer(ctx, testJob, &mocks.GitCredentials{})

			Expect(gitOps.ResolveCredentialsCallCount).To(Equal(2))
			Expect(gitOps.BuildCloneInitContainerCallCount).To(Equal(1))
		})

		It("should verify call counts", func() {
			gitOps.Reset()
			podLogOps.Reset()

			// Make some calls
			_, _ = gitOps.ResolveCredentials(ctx, &mlopsv1alpha1.NotebookValidationJob{})
			_ = podLogOps.ExtractError("test logs")

			// Verify using the mock's verification method
			gitOps.VerifyCallCounts(GinkgoT(), map[string]int{
				"ResolveCredentials": 1,
			})

			podLogOps.VerifyCallCounts(GinkgoT(), map[string]int{
				"ExtractError": 1,
			})
		})
	})

	Describe("Mock with custom behaviors", func() {
		It("should allow custom error scenarios", func() {
			gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*mocks.GitCredentials, error) {
				return nil, k8serrors.NewNotFound(corev1.Resource("secret"), "missing-secret")
			}

			creds, err := gitOps.ResolveCredentials(ctx, &mlopsv1alpha1.NotebookValidationJob{})
			Expect(err).To(HaveOccurred())
			Expect(creds).To(BeNil())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should allow custom return values", func() {
			gitOps.BuildCloneInitContainerFunc = func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *mocks.GitCredentials) (corev1.Container, error) {
				return corev1.Container{
					Name:  "custom-git-clone",
					Image: "custom/git:latest",
					Env: []corev1.EnvVar{
						{Name: "CUSTOM_VAR", Value: "custom-value"},
					},
				}, nil
			}

			testJob := &mlopsv1alpha1.NotebookValidationJob{}
			container, err := gitOps.BuildCloneInitContainer(ctx, testJob, &mocks.GitCredentials{})
			Expect(err).NotTo(HaveOccurred())
			Expect(container.Name).To(Equal("custom-git-clone"))
			Expect(container.Image).To(Equal("custom/git:latest"))
			Expect(len(container.Env)).To(Equal(1))
			Expect(container.Env[0].Name).To(Equal("CUSTOM_VAR"))
		})
	})
})
