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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestResolveGitCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		secret      *corev1.Secret
		expectError bool
		expected    *GitCredentials
	}{
		{
			name: "no credentials secret",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "",
						},
					},
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type: "none",
			},
		},
		{
			name: "SSH credentials",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "git@github.com:test/repo.git",
							Ref:               "main",
							CredentialsSecret: "ssh-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ssh-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ssh-privatekey": []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key\n-----END RSA PRIVATE KEY-----"),
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type:   "ssh",
				SSHKey: "-----BEGIN RSA PRIVATE KEY-----\ntest-key\n-----END RSA PRIVATE KEY-----",
			},
		},
		{
			name: "HTTPS credentials with password",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "https-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "https-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"password": []byte("testpass"),
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "testpass",
			},
		},
		{
			name: "HTTPS credentials with token",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "token-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "token-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"token":    []byte("ghp_token123"),
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "ghp_token123",
			},
		},
		{
			name: "HTTPS credentials missing password/token",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "invalid-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
				},
			},
			expectError: true,
		},
		{
			name: "secret not found",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "missing-secret",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "unrecognized credential format",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/repo.git",
							Ref:               "main",
							CredentialsSecret: "unknown-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unknown-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"unknown-key": []byte("value"),
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var k8sClient client.Client

			objects := []client.Object{tt.job}
			if tt.secret != nil {
				objects = append(objects, tt.secret)
			}

			k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()

			reconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			result, err := reconciler.resolveGitCredentials(ctx, tt.job)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			if tt.expected != nil {
				if result.Type != tt.expected.Type {
					t.Errorf("Type: got %s, want %s", result.Type, tt.expected.Type)
				}
				if result.Username != tt.expected.Username {
					t.Errorf("Username: got %s, want %s", result.Username, tt.expected.Username)
				}
				if result.Password != tt.expected.Password {
					t.Errorf("Password: got %s, want %s", result.Password, tt.expected.Password)
				}
				if result.SSHKey != tt.expected.SSHKey {
					t.Errorf("SSHKey: got %s, want %s", result.SSHKey, tt.expected.SSHKey)
				}
			}
		})
	}
}

func TestBuildGitCloneInitContainer(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		creds       *GitCredentials
		expectError bool
		validate    func(*testing.T, corev1.Container)
	}{
		{
			name: "anonymous clone",
			job: &mlopsv1alpha1.NotebookValidationJob{
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
				},
			},
			creds: &GitCredentials{
				Type: "none",
			},
			expectError: false,
			validate: func(t *testing.T, container corev1.Container) {
				if container.Name != "git-clone" {
					t.Errorf("Container name: got %s, want git-clone", container.Name)
				}
				if container.Image != "alpine/git:latest" {
					t.Errorf("Container image: got %s, want alpine/git:latest", container.Image)
				}
				if len(container.Env) == 0 {
					t.Error("Expected at least HOME env var")
				}
			},
		},
		{
			name: "HTTPS clone",
			job: &mlopsv1alpha1.NotebookValidationJob{
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
				},
			},
			creds: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "testpass",
			},
			expectError: false,
			validate: func(t *testing.T, container corev1.Container) {
				if container.Name != "git-clone" {
					t.Errorf("Container name: got %s, want git-clone", container.Name)
				}
			},
		},
		{
			name: "SSH clone",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					Notebook: mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "git@github.com:test/repo.git",
							Ref: "main",
						},
						Path: "notebooks/test.ipynb",
					},
				},
			},
			creds: &GitCredentials{
				Type:   "ssh",
				SSHKey: "test-ssh-key",
			},
			expectError: false,
			validate: func(t *testing.T, container corev1.Container) {
				if container.Name != "git-clone" {
					t.Errorf("Container name: got %s, want git-clone", container.Name)
				}
				// Check for SSH_PRIVATE_KEY env var
				foundSSHKey := false
				for _, env := range container.Env {
					if env.Name == "SSH_PRIVATE_KEY" {
						foundSSHKey = true
						if env.Value != "test-ssh-key" {
							t.Errorf("SSH_PRIVATE_KEY value: got %s, want test-ssh-key", env.Value)
						}
					}
				}
				if !foundSSHKey {
					t.Error("SSH_PRIVATE_KEY env var not found")
				}
			},
		},
		{
			name: "unsupported credential type",
			job: &mlopsv1alpha1.NotebookValidationJob{
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
				},
			},
			creds: &GitCredentials{
				Type: "unknown",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			reconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			result, err := reconciler.buildGitCloneInitContainer(ctx, tt.job, tt.creds)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestResolveGoldenGitCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		secret      *corev1.Secret
		expectError bool
		expected    *GitCredentials
	}{
		{
			name: "no golden notebook",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					GoldenNotebook: nil,
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type: "none",
			},
		},
		{
			name: "golden notebook with SSH credentials",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					GoldenNotebook: &mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "git@github.com:test/golden.git",
							Ref:               "main",
							CredentialsSecret: "golden-ssh-secret",
						},
						Path: "golden.ipynb",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "golden-ssh-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ssh-privatekey": []byte("golden-ssh-key"),
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type:   "ssh",
				SSHKey: "golden-ssh-key",
			},
		},
		{
			name: "golden notebook no credentials",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					GoldenNotebook: &mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL:               "https://github.com/test/golden.git",
							Ref:               "main",
							CredentialsSecret: "",
						},
						Path: "golden.ipynb",
					},
				},
			},
			expectError: false,
			expected: &GitCredentials{
				Type: "none",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var k8sClient client.Client

			objects := []client.Object{tt.job}
			if tt.secret != nil {
				objects = append(objects, tt.secret)
			}

			k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()

			reconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			result, err := reconciler.resolveGoldenGitCredentials(ctx, tt.job)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			if tt.expected != nil {
				if result.Type != tt.expected.Type {
					t.Errorf("Type: got %s, want %s", result.Type, tt.expected.Type)
				}
				if result.SSHKey != tt.expected.SSHKey {
					t.Errorf("SSHKey: got %s, want %s", result.SSHKey, tt.expected.SSHKey)
				}
			}
		})
	}
}

func TestBuildGoldenGitCloneInitContainer(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		creds       *GitCredentials
		expectError bool
	}{
		{
			name: "no golden notebook",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					GoldenNotebook: nil,
				},
			},
			creds: &GitCredentials{
				Type: "none",
			},
			expectError: true,
		},
		{
			name: "golden notebook with credentials",
			job: &mlopsv1alpha1.NotebookValidationJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: mlopsv1alpha1.NotebookValidationJobSpec{
					GoldenNotebook: &mlopsv1alpha1.NotebookSpec{
						Git: mlopsv1alpha1.GitSpec{
							URL: "https://github.com/test/golden.git",
							Ref: "main",
						},
						Path: "golden.ipynb",
					},
				},
			},
			creds: &GitCredentials{
				Type: "none",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			reconciler := &NotebookValidationJobReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			result, err := reconciler.buildGoldenGitCloneInitContainer(ctx, tt.job, tt.creds)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Name != "golden-git-clone" {
				t.Errorf("Container name: got %s, want golden-git-clone", result.Name)
			}
		})
	}
}
