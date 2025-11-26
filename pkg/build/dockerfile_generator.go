package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// validateAndJoinPath safely joins a base path with a user-provided path,
// preventing path traversal attacks (G304 fix).
// It returns an error if the resulting path would escape the base directory.
func validateAndJoinPath(basePath, userPath string) (string, error) {
	// Clean the user path to normalize it
	cleanUserPath := filepath.Clean(userPath)

	// Reject absolute paths
	if filepath.IsAbs(cleanUserPath) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", userPath)
	}

	// Reject paths that start with ..
	if strings.HasPrefix(cleanUserPath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", userPath)
	}

	// Join the paths
	fullPath := filepath.Join(basePath, cleanUserPath)

	// Verify the result is still within the base directory
	// Use Clean on basePath to ensure consistent comparison
	cleanBase := filepath.Clean(basePath)
	if !strings.HasPrefix(filepath.Clean(fullPath), cleanBase+string(filepath.Separator)) &&
		filepath.Clean(fullPath) != cleanBase {
		return "", fmt.Errorf("path escapes base directory: %s", userPath)
	}

	return fullPath, nil
}

// DockerfileGenerationResult contains the generated Dockerfile and metadata
type DockerfileGenerationResult struct {
	// Content is the generated Dockerfile content
	Content string
	// Source describes where the Dockerfile came from
	Source string
	// RequirementsFile is the path to requirements.txt used (if any)
	RequirementsFile string
	// UsingExistingDockerfile indicates whether an existing Dockerfile was used
	UsingExistingDockerfile bool
}

// GenerateDockerfile generates a Dockerfile from requirements.txt or falls back to existing Dockerfile
// ADR-038: Requirements.txt Auto-Detection and Dockerfile Generation Strategy
//
// Fallback chain:
// 1. Explicit RequirementsFile path
// 2. Custom RequirementsSources chain
// 3. Auto-detection: notebook-dir → tier-dir → repo-root
// 4. Existing Dockerfile
// 5. Base image only (no dependencies)
func GenerateDockerfile(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (*DockerfileGenerationResult, error) {
	// Default base image if not specified
	baseImage := job.Spec.PodConfig.BuildConfig.BaseImage
	if baseImage == "" {
		baseImage = "quay.io/opendatahub/workbench-images:jupyter-datascience-ubi9-python-3.11-2025.1"
	}

	// Check if auto-detection is disabled and PreferDockerfile is true
	if !job.Spec.PodConfig.BuildConfig.AutoGenerateRequirements && job.Spec.PodConfig.BuildConfig.PreferDockerfile {
		// Use existing Dockerfile only
		return useExistingDockerfile(job, gitRepoPath)
	}

	// Step 1: Find requirements.txt
	requirementsFile, source := findRequirementsFile(job, gitRepoPath)

	// Step 2: Check if both requirements.txt and Dockerfile exist
	if requirementsFile != "" {
		dockerfilePath, err := getDockerfilePath(job, gitRepoPath)
		dockerfileExists := err == nil && fileExists(dockerfilePath)

		if dockerfileExists && job.Spec.PodConfig.BuildConfig.PreferDockerfile {
			// User explicitly prefers Dockerfile
			return useExistingDockerfile(job, gitRepoPath)
		}

		// Generate Dockerfile from requirements.txt
		return generateFromRequirements(baseImage, requirementsFile, gitRepoPath, source)
	}

	// Step 3: Fall back to existing Dockerfile
	return useExistingDockerfile(job, gitRepoPath)
}

// findRequirementsFile finds requirements.txt using fallback chain
func findRequirementsFile(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (string, string) {
	// Disabled auto-detection
	if !job.Spec.PodConfig.BuildConfig.AutoGenerateRequirements {
		return "", ""
	}

	// Step 1: Explicit path specified
	if job.Spec.PodConfig.BuildConfig.RequirementsFile != "" {
		// #nosec G304 -- Path is validated by validateAndJoinPath to prevent traversal
		path, err := validateAndJoinPath(gitRepoPath, job.Spec.PodConfig.BuildConfig.RequirementsFile)
		if err == nil && fileExists(path) {
			return path, "explicit-path"
		}
		// Explicit path not found or invalid, log warning but continue with fallback
	}

	// Step 2: Custom fallback chain
	if len(job.Spec.PodConfig.BuildConfig.RequirementsSources) > 0 {
		for _, source := range job.Spec.PodConfig.BuildConfig.RequirementsSources {
			// #nosec G304 -- Path is validated by validateAndJoinPath to prevent traversal
			path, err := validateAndJoinPath(gitRepoPath, source)
			if err == nil && fileExists(path) {
				return path, "custom-chain"
			}
		}
	}

	// Step 3: Auto-detection fallback chain
	notebookPath := job.Spec.Notebook.Path
	// Validate notebook path first
	_, err := validateAndJoinPath(gitRepoPath, notebookPath)
	if err != nil {
		// Invalid notebook path, skip notebook-specific requirements
		notebookPath = ""
	}
	notebookDir := filepath.Dir(notebookPath)

	// Build candidates with validated paths
	var candidates []struct {
		path   string
		source string
	}

	// Notebook-specific (most specific) - only if notebook path is valid
	if notebookDir != "" && notebookDir != "." {
		if path, err := validateAndJoinPath(gitRepoPath, filepath.Join(notebookDir, "requirements.txt")); err == nil {
			candidates = append(candidates, struct {
				path   string
				source string
			}{path, "notebook-directory"})
		}
	}

	// Tier-level (notebooks directory)
	if path, err := validateAndJoinPath(gitRepoPath, "notebooks/requirements.txt"); err == nil {
		candidates = append(candidates, struct {
			path   string
			source string
		}{path, "tier-directory"})
	}

	// Repository root (project-wide)
	if path, err := validateAndJoinPath(gitRepoPath, "requirements.txt"); err == nil {
		candidates = append(candidates, struct {
			path   string
			source string
		}{path, "repository-root"})
	}

	for _, candidate := range candidates {
		if fileExists(candidate.path) {
			return candidate.path, candidate.source
		}
	}

	return "", ""
}

// generateFromRequirements generates Dockerfile content from requirements.txt
func generateFromRequirements(baseImage, requirementsFile, gitRepoPath, source string) (*DockerfileGenerationResult, error) {
	relativePath, err := filepath.Rel(gitRepoPath, requirementsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path for requirements.txt: %w", err)
	}

	// Ensure forward slashes for Dockerfile COPY command
	relativePath = filepath.ToSlash(relativePath)

	// Generate Dockerfile content
	dockerfile := fmt.Sprintf(`FROM %s

# ADR-038: Auto-generated Dockerfile from requirements.txt
# Source: %s (%s)

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Copy and install project dependencies
COPY %s /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

# Set working directory
WORKDIR /workspace

# Health check
RUN python -c "import sys; print(f'Python {{sys.version}}')"
`, baseImage, relativePath, source, relativePath)

	return &DockerfileGenerationResult{
		Content:                 dockerfile,
		Source:                  fmt.Sprintf("generated from %s (%s)", relativePath, source),
		RequirementsFile:        requirementsFile,
		UsingExistingDockerfile: false,
	}, nil
}

// useExistingDockerfile reads and returns existing Dockerfile or generates minimal one
func useExistingDockerfile(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (*DockerfileGenerationResult, error) {
	dockerfilePath, err := getDockerfilePath(job, gitRepoPath)
	if err != nil {
		// Invalid Dockerfile path, fall back to generating minimal Dockerfile
		dockerfilePath = ""
	}

	// Try to read existing Dockerfile
	if dockerfilePath != "" && fileExists(dockerfilePath) {
		// #nosec G304 -- Path is validated by getDockerfilePath using validateAndJoinPath
		content, err := os.ReadFile(dockerfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
		}

		relativePath, _ := filepath.Rel(gitRepoPath, dockerfilePath)
		return &DockerfileGenerationResult{
			Content:                 string(content),
			Source:                  fmt.Sprintf("existing Dockerfile (%s)", relativePath),
			RequirementsFile:        "",
			UsingExistingDockerfile: true,
		}, nil
	}

	// Generate minimal Dockerfile with base image only
	baseImage := job.Spec.PodConfig.BuildConfig.BaseImage
	if baseImage == "" {
		baseImage = "quay.io/opendatahub/workbench-images:jupyter-datascience-ubi9-python-3.11-2025.1"
	}

	dockerfile := fmt.Sprintf(`FROM %s

# ADR-038: Minimal Dockerfile (no requirements.txt found)

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Set working directory
WORKDIR /workspace
`, baseImage)

	return &DockerfileGenerationResult{
		Content:                 dockerfile,
		Source:                  "generated (base image only, no dependencies)",
		RequirementsFile:        "",
		UsingExistingDockerfile: false,
	}, nil
}

// getDockerfilePath returns the full path to Dockerfile with path traversal protection
func getDockerfilePath(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (string, error) {
	dockerfilePath := job.Spec.PodConfig.BuildConfig.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	// Validate the path to prevent traversal attacks
	return validateAndJoinPath(gitRepoPath, dockerfilePath)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ValidateDockerfileGeneration validates the generated Dockerfile
// ADR-038: Provides warnings if both requirements.txt and Dockerfile exist
func ValidateDockerfileGeneration(job *mlopsv1alpha1.NotebookValidationJob, result *DockerfileGenerationResult) []string {
	var warnings []string

	// Check if both requirements.txt and Dockerfile exist
	if result.RequirementsFile != "" && !result.UsingExistingDockerfile {
		// Generated from requirements.txt, check if Dockerfile also exists
		gitRepoPath := filepath.Dir(filepath.Dir(result.RequirementsFile))
		dockerfilePath, err := getDockerfilePath(job, gitRepoPath)

		if err == nil && fileExists(dockerfilePath) && !job.Spec.PodConfig.BuildConfig.PreferDockerfile {
			warnings = append(warnings, fmt.Sprintf(
				"Both requirements.txt and Dockerfile exist. Using requirements.txt by default. "+
					"Set spec.podConfig.buildConfig.preferDockerfile=true to use Dockerfile instead.",
			))
		}
	}

	// Check for large requirements files
	if result.RequirementsFile != "" {
		info, err := os.Stat(result.RequirementsFile)
		if err == nil && info.Size() > 100*1024 { // > 100KB
			warnings = append(warnings, fmt.Sprintf(
				"requirements.txt is unusually large (%d KB). Consider using a pre-built base image with common dependencies.",
				info.Size()/1024,
			))
		}
	}

	// Check for security issues in generated Dockerfile
	if strings.Contains(result.Content, "pip install") && !strings.Contains(result.Content, "--no-cache-dir") {
		warnings = append(warnings, "Generated Dockerfile should use 'pip install --no-cache-dir' to reduce image size")
	}

	return warnings
}
