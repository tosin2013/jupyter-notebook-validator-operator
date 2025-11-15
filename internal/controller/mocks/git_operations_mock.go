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

package mocks

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// GitCredentials represents Git authentication credentials
type GitCredentials struct {
	Type     string // "https", "ssh", or "none"
	Username string // For HTTPS
	Password string // For HTTPS (token or password)
	SSHKey   string // For SSH (private key)
}

// GitOperations defines the interface for Git-related operations
type GitOperations interface {
	ResolveCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)
	ResolveGoldenCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)
	BuildCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)
	BuildGoldenCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)
}

// MockGitOperations is a mock implementation of GitOperations for testing
type MockGitOperations struct {
	// ResolveCredentialsFunc allows customizing the ResolveCredentials behavior
	ResolveCredentialsFunc func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)
	
	// ResolveGoldenCredentialsFunc allows customizing the ResolveGoldenCredentials behavior
	ResolveGoldenCredentialsFunc func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)
	
	// BuildCloneInitContainerFunc allows customizing the BuildCloneInitContainer behavior
	BuildCloneInitContainerFunc func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)
	
	// BuildGoldenCloneInitContainerFunc allows customizing the BuildGoldenCloneInitContainer behavior
	BuildGoldenCloneInitContainerFunc func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)
	
	// Call tracking
	ResolveCredentialsCallCount      int
	ResolveGoldenCredentialsCallCount int
	BuildCloneInitContainerCallCount  int
	BuildGoldenCloneInitContainerCallCount int
}

// NewMockGitOperations creates a new mock GitOperations with default behaviors
func NewMockGitOperations() *MockGitOperations {
	return &MockGitOperations{
		ResolveCredentialsFunc: func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
			return &GitCredentials{Type: "none"}, nil
		},
		ResolveGoldenCredentialsFunc: func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
			return &GitCredentials{Type: "none"}, nil
		},
		BuildCloneInitContainerFunc: func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
			return corev1.Container{
				Name:  "git-clone",
				Image: "alpine/git:latest",
			}, nil
		},
		BuildGoldenCloneInitContainerFunc: func(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
			return corev1.Container{
				Name:  "golden-git-clone",
				Image: "alpine/git:latest",
			}, nil
		},
	}
}

// ResolveCredentials implements GitOperations
func (m *MockGitOperations) ResolveCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	m.ResolveCredentialsCallCount++
	if m.ResolveCredentialsFunc != nil {
		return m.ResolveCredentialsFunc(ctx, job)
	}
	return &GitCredentials{Type: "none"}, nil
}

// ResolveGoldenCredentials implements GitOperations
func (m *MockGitOperations) ResolveGoldenCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	m.ResolveGoldenCredentialsCallCount++
	if m.ResolveGoldenCredentialsFunc != nil {
		return m.ResolveGoldenCredentialsFunc(ctx, job)
	}
	return &GitCredentials{Type: "none"}, nil
}

// BuildCloneInitContainer implements GitOperations
func (m *MockGitOperations) BuildCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	m.BuildCloneInitContainerCallCount++
	if m.BuildCloneInitContainerFunc != nil {
		return m.BuildCloneInitContainerFunc(ctx, job, creds)
	}
	return corev1.Container{Name: "git-clone"}, nil
}

// BuildGoldenCloneInitContainer implements GitOperations
func (m *MockGitOperations) BuildGoldenCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	m.BuildGoldenCloneInitContainerCallCount++
	if m.BuildGoldenCloneInitContainerFunc != nil {
		return m.BuildGoldenCloneInitContainerFunc(ctx, job, creds)
	}
	return corev1.Container{Name: "golden-git-clone"}, nil
}

// Reset resets all call counts
func (m *MockGitOperations) Reset() {
	m.ResolveCredentialsCallCount = 0
	m.ResolveGoldenCredentialsCallCount = 0
	m.BuildCloneInitContainerCallCount = 0
	m.BuildGoldenCloneInitContainerCallCount = 0
}

// VerifyCallCounts verifies that methods were called the expected number of times
func (m *MockGitOperations) VerifyCallCounts(t interface {
	Errorf(format string, args ...interface{})
}, expected map[string]int) {
	if expected["ResolveCredentials"] != m.ResolveCredentialsCallCount {
		t.Errorf("ResolveCredentials called %d times, expected %d", m.ResolveCredentialsCallCount, expected["ResolveCredentials"])
	}
	if expected["ResolveGoldenCredentials"] != m.ResolveGoldenCredentialsCallCount {
		t.Errorf("ResolveGoldenCredentials called %d times, expected %d", m.ResolveGoldenCredentialsCallCount, expected["ResolveGoldenCredentials"])
	}
	if expected["BuildCloneInitContainer"] != m.BuildCloneInitContainerCallCount {
		t.Errorf("BuildCloneInitContainer called %d times, expected %d", m.BuildCloneInitContainerCallCount, expected["BuildCloneInitContainer"])
	}
	if expected["BuildGoldenCloneInitContainer"] != m.BuildGoldenCloneInitContainerCallCount {
		t.Errorf("BuildGoldenCloneInitContainer called %d times, expected %d", m.BuildGoldenCloneInitContainerCallCount, expected["BuildGoldenCloneInitContainer"])
	}
}
