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

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// GitOperations defines the interface for Git-related operations
// This allows us to mock Git operations in tests
type GitOperations interface {
	// ResolveCredentials resolves Git credentials from Kubernetes secrets
	ResolveCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)

	// ResolveGoldenCredentials resolves Git credentials for golden notebook
	ResolveGoldenCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error)

	// BuildCloneInitContainer builds an init container for Git clone operations
	BuildCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)

	// BuildGoldenCloneInitContainer builds an init container for golden notebook Git clone
	BuildGoldenCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error)
}

// gitOperationsImpl implements GitOperations using Kubernetes client
type gitOperationsImpl struct {
	reconciler *NotebookValidationJobReconciler
}

// NewGitOperations creates a new GitOperations implementation
func NewGitOperations(reconciler *NotebookValidationJobReconciler) GitOperations {
	return &gitOperationsImpl{
		reconciler: reconciler,
	}
}

// ResolveCredentials implements GitOperations
func (g *gitOperationsImpl) ResolveCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	return g.reconciler.resolveGitCredentials(ctx, job)
}

// ResolveGoldenCredentials implements GitOperations
func (g *gitOperationsImpl) ResolveGoldenCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	return g.reconciler.resolveGoldenGitCredentials(ctx, job)
}

// BuildCloneInitContainer implements GitOperations
func (g *gitOperationsImpl) BuildCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	return g.reconciler.buildGitCloneInitContainer(ctx, job, creds)
}

// BuildGoldenCloneInitContainer implements GitOperations
func (g *gitOperationsImpl) BuildGoldenCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	return g.reconciler.buildGoldenGitCloneInitContainer(ctx, job, creds)
}
