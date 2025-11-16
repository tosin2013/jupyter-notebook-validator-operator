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
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

func TestGetGitImage(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expected    string
		description string
	}{
		{
			name: "manual override via GIT_INIT_IMAGE",
			envVars: map[string]string{
				"GIT_INIT_IMAGE": "custom/git:v1.0",
			},
			expected:    "custom/git:v1.0",
			description: "Should use GIT_INIT_IMAGE when set",
		},
		{
			name: "OpenShift platform via PLATFORM env",
			envVars: map[string]string{
				"PLATFORM": "openshift",
			},
			expected:    "registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8@sha256:4fabae1312c1aaf8a57bd2de63bd040956faa0c728453f2a4b4002705fba0f0c",
			description: "Should use OpenShift git-init image when PLATFORM=openshift",
		},
		{
			name: "OpenShift platform via OPENSHIFT_BUILD_NAMESPACE",
			envVars: map[string]string{
				"OPENSHIFT_BUILD_NAMESPACE": "my-namespace",
			},
			expected:    "registry.redhat.io/openshift-pipelines/pipelines-git-init-rhel8@sha256:4fabae1312c1aaf8a57bd2de63bd040956faa0c728453f2a4b4002705fba0f0c",
			description: "Should use OpenShift git-init image when OPENSHIFT_BUILD_NAMESPACE is set",
		},
		{
			name:        "default Kubernetes",
			envVars:     map[string]string{},
			expected:    "bitnami/git:latest",
			description: "Should use bitnami/git for vanilla Kubernetes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars
			os.Unsetenv("GIT_INIT_IMAGE")
			os.Unsetenv("PLATFORM")
			os.Unsetenv("OPENSHIFT_BUILD_NAMESPACE")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			result := getGitImage()
			if result != tt.expected {
				t.Errorf("%s: getGitImage() = %v, want %v", tt.description, result, tt.expected)
			}

			// Cleanup
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
		})
	}
}

func TestResolveGitCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name        string
		job         *mlopsv1alpha1.NotebookValidationJob
		secret      *corev1.Secret
		expected    *GitCredentials
		expectError bool
		description string
	}{
		{
			name: "no credentials secret specified",
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
			expected: &GitCredentials{
				Type: "none",
			},
			expectError: false,
			description: "Should return 'none' type when no credentials secret specified",
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
							CredentialsSecret: "git-ssh-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-ssh-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ssh-privatekey": []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----"),
				},
			},
			expected: &GitCredentials{
				Type:   "ssh",
				SSHKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----",
			},
			expectError: false,
			description: "Should parse SSH credentials correctly",
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
							CredentialsSecret: "git-https-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-https-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"password": []byte("testpass"),
				},
			},
			expected: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "testpass",
			},
			expectError: false,
			description: "Should parse HTTPS credentials with password",
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
							CredentialsSecret: "git-token-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-token-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"token":    []byte("ghp_testtoken123"),
				},
			},
			expected: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "ghp_testtoken123",
			},
			expectError: false,
			description: "Should parse HTTPS credentials with token",
		},
		{
			name: "missing password and token",
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
							CredentialsSecret: "git-invalid-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-invalid-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("testuser"),
				},
			},
			expected:    nil,
			expectError: true,
			description: "Should error when HTTPS credentials missing password/token",
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
							CredentialsSecret: "git-unknown-secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-unknown-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"somekey": []byte("somevalue"),
				},
			},
			expected:    nil,
			expectError: true,
			description: "Should error when secret has unrecognized format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with secret if provided
			var objs []runtime.Object
			if tt.secret != nil {
				objs = append(objs, tt.secret)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()

			reconciler := &NotebookValidationJobReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			ctx := context.Background()
			result, err := reconciler.resolveGitCredentials(ctx, tt.job)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if result == nil {
					t.Errorf("%s: expected result but got nil", tt.description)
					return
				}
				if result.Type != tt.expected.Type {
					t.Errorf("%s: Type = %v, want %v", tt.description, result.Type, tt.expected.Type)
				}
				if result.Username != tt.expected.Username {
					t.Errorf("%s: Username = %v, want %v", tt.description, result.Username, tt.expected.Username)
				}
				if result.Password != tt.expected.Password {
					t.Errorf("%s: Password = %v, want %v", tt.description, result.Password, tt.expected.Password)
				}
				if result.SSHKey != tt.expected.SSHKey {
					t.Errorf("%s: SSHKey = %v, want %v", tt.description, result.SSHKey, tt.expected.SSHKey)
				}
			}
		})
	}
}

func TestBuildGitCloneInitContainer(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = mlopsv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &NotebookValidationJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	job := &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Path: "notebooks/test.ipynb",
				Git: mlopsv1alpha1.GitSpec{
					URL: "https://github.com/test/repo.git",
					Ref: "main",
				},
			},
		},
	}

	tests := []struct {
		name            string
		creds           *GitCredentials
		expectError     bool
		validateCommand func(t *testing.T, command string)
		description     string
	}{
		{
			name: "anonymous clone",
			creds: &GitCredentials{
				Type: "none",
			},
			expectError: false,
			validateCommand: func(t *testing.T, command string) {
				if !contains(command, "Cloning repository (anonymous)") {
					t.Error("Command should contain anonymous clone message")
				}
				if !contains(command, "git clone --depth 1 --branch main") {
					t.Error("Command should contain git clone with depth and branch")
				}
				if !contains(command, "notebooks/test.ipynb") {
					t.Error("Command should verify notebook path")
				}
			},
			description: "Should generate anonymous clone command",
		},
		{
			name: "HTTPS clone",
			creds: &GitCredentials{
				Type:     "https",
				Username: "testuser",
				Password: "testpass",
			},
			expectError: false,
			validateCommand: func(t *testing.T, command string) {
				if !contains(command, "Cloning repository (HTTPS)") {
					t.Error("Command should contain HTTPS clone message")
				}
				if !contains(command, "testuser:testpass@") {
					t.Error("Command should embed credentials in URL")
				}
				if !contains(command, "***REDACTED***") {
					t.Error("Command should redact password in output")
				}
			},
			description: "Should generate HTTPS clone command with credentials",
		},
		{
			name: "SSH clone",
			creds: &GitCredentials{
				Type:   "ssh",
				SSHKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----",
			},
			expectError: false,
			validateCommand: func(t *testing.T, command string) {
				if !contains(command, "Setting up SSH authentication") {
					t.Error("Command should contain SSH setup message")
				}
				if !contains(command, "mkdir -p ~/.ssh") {
					t.Error("Command should create .ssh directory")
				}
			},
			description: "Should generate SSH clone command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			container, err := reconciler.buildGitCloneInitContainer(ctx, job, tt.creds)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if container.Name != "git-clone" {
					t.Errorf("%s: container name = %v, want git-clone", tt.description, container.Name)
				}
				if len(container.Command) == 0 {
					t.Errorf("%s: container command is empty", tt.description)
				}
				if tt.validateCommand != nil && len(container.Args) > 0 {
					tt.validateCommand(t, container.Args[len(container.Args)-1])
				}
			}
		})
	}
}

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "username",
			expected: "us***me", // First 2 and last 2 chars
		},
		{
			input:    "ab",
			expected: "***", // 4 chars or less
		},
		{
			input:    "a",
			expected: "***", // 4 chars or less
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "verylongusername",
			expected: "ve***me", // First 2 and last 2 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeForLog(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeForLog(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
