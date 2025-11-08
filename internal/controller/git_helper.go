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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"github.com/tosin2013/jupyter-notebook-validator-operator/pkg/logging"
)

// GitCredentials holds parsed Git authentication credentials
type GitCredentials struct {
	Type     string // "https", "ssh", or "none"
	Username string // For HTTPS
	Password string // For HTTPS (token or password)
	SSHKey   string // For SSH (private key)
}

// resolveGitCredentials reads and parses Git credentials from a Kubernetes Secret
// Based on ADR-009: Secret Management and Git Credentials
func (r *NotebookValidationJobReconciler) resolveGitCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	logger := log.FromContext(ctx)

	// If no credentials secret specified, return empty credentials
	if job.Spec.Notebook.Git.CredentialsSecret == "" {
		logger.V(1).Info("No credentials secret specified, using anonymous access",
			"namespace", job.Namespace,
			"name", job.Name)
		return &GitCredentials{Type: "none"}, nil
	}

	logger.V(1).Info("Resolving Git credentials from secret",
		"namespace", job.Namespace,
		"name", job.Name,
		"secretName", job.Spec.Notebook.Git.CredentialsSecret)

	// Fetch the secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      job.Spec.Notebook.Git.CredentialsSecret,
		Namespace: job.Namespace,
	}

	if err := r.Get(ctx, secretName, secret); err != nil {
		logger.Error(err, "Failed to get credentials secret", "secretName", secretName.Name)
		return nil, fmt.Errorf("failed to get credentials secret %s: %w", secretName.Name, err)
	}

	logger.Info("Successfully retrieved credentials secret", "secretName", secretName.Name)

	// Determine credential type based on secret keys
	creds := &GitCredentials{}

	// Check for SSH key (ADR-009: SSH authentication)
	if sshKey, exists := secret.Data["ssh-privatekey"]; exists {
		creds.Type = "ssh"
		creds.SSHKey = string(sshKey)
		logger.Info("Using SSH authentication")
		return creds, nil
	}

	// Check for HTTPS credentials (ADR-009: HTTPS authentication)
	if username, exists := secret.Data["username"]; exists {
		creds.Type = "https"
		creds.Username = string(username)

		// Password or token
		if password, exists := secret.Data["password"]; exists {
			creds.Password = string(password)
		} else if token, exists := secret.Data["token"]; exists {
			creds.Password = string(token)
		} else {
			return nil, fmt.Errorf("HTTPS credentials require 'password' or 'token' key in secret")
		}

		logger.Info("Using HTTPS authentication", "username", sanitizeForLog(creds.Username))
		return creds, nil
	}

	// No recognized credential format
	return nil, fmt.Errorf("secret %s does not contain recognized credential format (ssh-privatekey or username+password/token)", secretName.Name)
}

// buildGitCloneInitContainer creates an init container for Git clone
// Based on ADR-009: Git clone with credentials
func (r *NotebookValidationJobReconciler) buildGitCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	logger := log.FromContext(ctx)

	gitURL := job.Spec.Notebook.Git.URL
	gitRef := job.Spec.Notebook.Git.Ref
	notebookPath := job.Spec.Notebook.Path

	logger.V(1).Info("Building Git clone init container",
		"namespace", job.Namespace,
		"name", job.Name,
		"gitURL", logging.SanitizeURL(gitURL),
		"gitRef", gitRef,
		"notebookPath", notebookPath,
		"credentialType", creds.Type)

	// Build Git clone command based on credential type
	var cloneCommand string

	switch creds.Type {
	case "none":
		// Anonymous clone
		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Cloning repository (anonymous)..."
			git clone --depth 1 --branch %s %s /workspace/repo
			echo "Clone successful"
			ls -la /workspace/repo
			echo "Verifying notebook exists: %s"
			if [ ! -f "/workspace/repo/%s" ]; then
				echo "ERROR: Notebook not found at path: %s"
				exit 1
			fi
			echo "Notebook found successfully"
		`, gitRef, gitURL, notebookPath, notebookPath, notebookPath)

	case "https":
		// HTTPS clone with credentials
		// Sanitize credentials for URL (ADR-009: credential sanitization)
		sanitizedURL := strings.Replace(gitURL, "https://", fmt.Sprintf("https://%s:%s@", creds.Username, creds.Password), 1)

		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Cloning repository (HTTPS)..."
			# Disable credential helper to avoid leaking credentials
			git config --global credential.helper ""
			# Clone with embedded credentials
			git clone --depth 1 --branch %s "%s" /workspace/repo 2>&1 | sed 's/%s/***REDACTED***/g'
			echo "Clone successful"
			ls -la /workspace/repo
			echo "Verifying notebook exists: %s"
			if [ ! -f "/workspace/repo/%s" ]; then
				echo "ERROR: Notebook not found at path: %s"
				exit 1
			fi
			echo "Notebook found successfully"
		`, gitRef, sanitizedURL, creds.Password, notebookPath, notebookPath, notebookPath)

	case "ssh":
		// SSH clone with private key
		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Setting up SSH authentication..."
			mkdir -p ~/.ssh
			chmod 700 ~/.ssh
			# Write SSH key from secret
			echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
			chmod 600 ~/.ssh/id_rsa
			# Disable host key checking (for automation)
			echo "StrictHostKeyChecking no" > ~/.ssh/config
			echo "UserKnownHostsFile /dev/null" >> ~/.ssh/config
			chmod 600 ~/.ssh/config
			echo "Cloning repository (SSH)..."
			git clone --depth 1 --branch %s %s /workspace/repo
			echo "Clone successful"
			ls -la /workspace/repo
			echo "Verifying notebook exists: %s"
			if [ ! -f "/workspace/repo/%s" ]; then
				echo "ERROR: Notebook not found at path: %s"
				exit 1
			fi
			echo "Notebook found successfully"
			# Clean up SSH key
			rm -f ~/.ssh/id_rsa
		`, gitRef, gitURL, notebookPath, notebookPath, notebookPath)

	default:
		return corev1.Container{}, fmt.Errorf("unsupported credential type: %s", creds.Type)
	}

	// Build init container
	initContainer := corev1.Container{
		Name:  "git-clone",
		Image: "alpine/git:latest",
		Command: []string{
			"/bin/sh",
			"-c",
			cloneCommand,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: boolPtr(true),
			// RunAsUser is intentionally omitted to allow OpenShift to assign a UID
			// from the namespace's allocated range (ADR-005: OpenShift Compatibility)
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		// Set HOME to writable directory for git config
		Env: []corev1.EnvVar{
			{
				Name:  "HOME",
				Value: "/tmp",
			},
		},
	}

	// Add SSH key as environment variable if using SSH
	if creds.Type == "ssh" {
		initContainer.Env = append(initContainer.Env, corev1.EnvVar{
			Name:  "SSH_PRIVATE_KEY",
			Value: creds.SSHKey,
		})
	}

	logger.Info("Built Git clone init container", "credentialType", creds.Type)
	return initContainer, nil
}

// resolveGoldenGitCredentials reads and parses Git credentials for the golden notebook
// Based on ADR-009: Secret Management and Git Credentials
func (r *NotebookValidationJobReconciler) resolveGoldenGitCredentials(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*GitCredentials, error) {
	logger := log.FromContext(ctx)

	// Check if golden notebook is specified
	if job.Spec.GoldenNotebook == nil {
		return &GitCredentials{Type: "none"}, nil
	}

	// If no credentials secret specified, return empty credentials
	if job.Spec.GoldenNotebook.Git.CredentialsSecret == "" {
		logger.Info("No golden notebook credentials secret specified, using anonymous access")
		return &GitCredentials{Type: "none"}, nil
	}

	// Fetch the secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      job.Spec.GoldenNotebook.Git.CredentialsSecret,
		Namespace: job.Namespace,
	}

	if err := r.Get(ctx, secretName, secret); err != nil {
		logger.Error(err, "Failed to get golden notebook credentials secret", "secretName", secretName.Name)
		return nil, fmt.Errorf("failed to get golden notebook credentials secret %s: %w", secretName.Name, err)
	}

	logger.Info("Successfully retrieved golden notebook credentials secret", "secretName", secretName.Name)

	// Determine credential type based on secret keys
	creds := &GitCredentials{}

	// Check for SSH key (ADR-009: SSH authentication)
	if sshKey, exists := secret.Data["ssh-privatekey"]; exists {
		creds.Type = "ssh"
		creds.SSHKey = string(sshKey)
		logger.Info("Using SSH authentication for golden notebook")
		return creds, nil
	}

	// Check for HTTPS credentials (ADR-009: HTTPS authentication)
	if username, exists := secret.Data["username"]; exists {
		creds.Type = "https"
		creds.Username = string(username)

		// Password or token
		if password, exists := secret.Data["password"]; exists {
			creds.Password = string(password)
		} else if token, exists := secret.Data["token"]; exists {
			creds.Password = string(token)
		} else {
			return nil, fmt.Errorf("HTTPS credentials require 'password' or 'token' key in secret")
		}

		logger.Info("Using HTTPS authentication for golden notebook", "username", sanitizeForLog(creds.Username))
		return creds, nil
	}

	// No recognized credential format
	return nil, fmt.Errorf("secret %s does not contain recognized credential format (ssh-privatekey or username+password/token)", secretName.Name)
}

// buildGoldenGitCloneInitContainer creates an init container for golden notebook Git clone
// Based on ADR-009: Git clone with credentials and ADR-013: Golden Notebook Comparison
func (r *NotebookValidationJobReconciler) buildGoldenGitCloneInitContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, creds *GitCredentials) (corev1.Container, error) {
	logger := log.FromContext(ctx)

	if job.Spec.GoldenNotebook == nil {
		return corev1.Container{}, fmt.Errorf("golden notebook not specified")
	}

	gitURL := job.Spec.GoldenNotebook.Git.URL
	gitRef := job.Spec.GoldenNotebook.Git.Ref
	notebookPath := job.Spec.GoldenNotebook.Path

	// Build Git clone command based on credential type
	var cloneCommand string

	switch creds.Type {
	case "none":
		// Anonymous clone
		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Cloning golden notebook repository (anonymous)..."
			git clone --depth 1 --branch %s %s /workspace/golden
			echo "Golden clone successful"
			ls -la /workspace/golden
			echo "Verifying golden notebook exists: %s"
			if [ ! -f "/workspace/golden/%s" ]; then
				echo "ERROR: Golden notebook not found at path: %s"
				exit 1
			fi
			echo "Golden notebook found successfully"
		`, gitRef, gitURL, notebookPath, notebookPath, notebookPath)

	case "https":
		// HTTPS clone with credentials
		// Sanitize credentials for URL (ADR-009: credential sanitization)
		sanitizedURL := strings.Replace(gitURL, "https://", fmt.Sprintf("https://%s:%s@", creds.Username, creds.Password), 1)

		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Cloning golden notebook repository (HTTPS)..."
			# Disable credential helper to avoid leaking credentials
			git config --global credential.helper ""
			# Clone with embedded credentials
			git clone --depth 1 --branch %s "%s" /workspace/golden 2>&1 | sed 's/%s/***REDACTED***/g'
			echo "Golden clone successful"
			ls -la /workspace/golden
			echo "Verifying golden notebook exists: %s"
			if [ ! -f "/workspace/golden/%s" ]; then
				echo "ERROR: Golden notebook not found at path: %s"
				exit 1
			fi
			echo "Golden notebook found successfully"
		`, gitRef, sanitizedURL, creds.Password, notebookPath, notebookPath, notebookPath)

	case "ssh":
		// SSH clone with private key
		cloneCommand = fmt.Sprintf(`
			set -e
			echo "Setting up SSH authentication for golden notebook..."
			mkdir -p ~/.ssh
			chmod 700 ~/.ssh
			# Write SSH key from secret
			echo "$SSH_PRIVATE_KEY" > ~/.ssh/id_rsa
			chmod 600 ~/.ssh/id_rsa
			# Disable host key checking (for automation)
			echo "StrictHostKeyChecking no" > ~/.ssh/config
			echo "UserKnownHostsFile /dev/null" >> ~/.ssh/config
			chmod 600 ~/.ssh/config
			echo "Cloning golden notebook repository (SSH)..."
			git clone --depth 1 --branch %s %s /workspace/golden
			echo "Golden clone successful"
			ls -la /workspace/golden
			echo "Verifying golden notebook exists: %s"
			if [ ! -f "/workspace/golden/%s" ]; then
				echo "ERROR: Golden notebook not found at path: %s"
				exit 1
			fi
			echo "Golden notebook found successfully"
			# Clean up SSH key
			rm -f ~/.ssh/id_rsa
		`, gitRef, gitURL, notebookPath, notebookPath, notebookPath)

	default:
		return corev1.Container{}, fmt.Errorf("unsupported credential type: %s", creds.Type)
	}

	// Build init container
	initContainer := corev1.Container{
		Name:  "golden-git-clone",
		Image: "alpine/git:latest",
		Command: []string{
			"/bin/sh",
			"-c",
			cloneCommand,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: boolPtr(true),
			// RunAsUser is intentionally omitted to allow OpenShift to assign a UID
			// from the namespace's allocated range (ADR-005: OpenShift Compatibility)
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		// Set HOME to writable directory for git config
		Env: []corev1.EnvVar{
			{
				Name:  "HOME",
				Value: "/tmp",
			},
		},
	}

	// Add SSH key as environment variable if using SSH
	if creds.Type == "ssh" {
		initContainer.Env = append(initContainer.Env, corev1.EnvVar{
			Name:  "SSH_PRIVATE_KEY",
			Value: creds.SSHKey,
		})
	}

	logger.Info("Built golden notebook Git clone init container", "credentialType", creds.Type)
	return initContainer, nil
}

// sanitizeForLog removes sensitive information from log messages
// Based on ADR-009: Log sanitization
// Deprecated: Use logging.SanitizeString instead
func sanitizeForLog(value string) string {
	return logging.SanitizeString(value)
}

// Helper functions for pointer types
func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}
