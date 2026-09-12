package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newTestJob creates a minimal NotebookValidationJob for testing.
func newTestJob(notebookPath, baseImage string, buildConfig *mlopsv1alpha1.BuildConfigSpec) *mlopsv1alpha1.NotebookValidationJob {
	if buildConfig == nil {
		buildConfig = &mlopsv1alpha1.BuildConfigSpec{
			Enabled:                  true,
			AutoGenerateRequirements: true,
			BaseImage:                baseImage,
		}
	}
	return &mlopsv1alpha1.NotebookValidationJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "default"},
		Spec: mlopsv1alpha1.NotebookValidationJobSpec{
			Notebook: mlopsv1alpha1.NotebookSpec{
				Git:  mlopsv1alpha1.GitSpec{URL: "https://github.com/example/repo.git", Ref: "main"},
				Path: notebookPath,
			},
			PodConfig: mlopsv1alpha1.PodConfigSpec{
				ContainerImage: "test-image:latest",
				BuildConfig:    buildConfig,
			},
		},
	}
}

// writeFile creates a file with the given content inside a temp directory.
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", full, err)
	}
}

// --- Fallback chain tests (ADR-038) ---

func TestGenerateDockerfile_NotebookDirectoryRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notebooks/tier2/requirements.txt", "pandas==2.0\n")
	writeFile(t, dir, "notebooks/tier2/analysis.ipynb", "{}")

	job := newTestJob("notebooks/tier2/analysis.ipynb", "python:3.11", nil)
	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if result.UsingExistingDockerfile {
		t.Error("expected generated Dockerfile, got existing")
	}
	if !strings.Contains(result.Source, "notebook-directory") {
		t.Errorf("Source = %q, want substring 'notebook-directory'", result.Source)
	}
	if !strings.Contains(result.Content, "COPY notebooks/tier2/requirements.txt") {
		t.Errorf("Dockerfile missing COPY for notebook-dir requirements.txt:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "FROM python:3.11") {
		t.Errorf("Dockerfile missing base image:\n%s", result.Content)
	}
}

func TestGenerateDockerfile_TierDirectoryRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notebooks/requirements.txt", "scikit-learn>=1.0\n")
	writeFile(t, dir, "notebooks/tier2/analysis.ipynb", "{}")

	job := newTestJob("notebooks/tier2/analysis.ipynb", "", nil)
	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !strings.Contains(result.Source, "tier-directory") {
		t.Errorf("Source = %q, want substring 'tier-directory'", result.Source)
	}
	if result.RequirementsFile == "" {
		t.Error("RequirementsFile should not be empty")
	}
}

func TestGenerateDockerfile_RepositoryRootRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "numpy>=1.24\ntorch>=2.0\n")
	writeFile(t, dir, "analysis.ipynb", "{}")

	job := newTestJob("analysis.ipynb", "python:3.11", nil)
	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !strings.Contains(result.Source, "repository-root") {
		t.Errorf("Source = %q, want substring 'repository-root'", result.Source)
	}
	if !strings.Contains(result.Content, "COPY requirements.txt") {
		t.Errorf("Dockerfile missing COPY for root requirements.txt:\n%s", result.Content)
	}
}

func TestGenerateDockerfile_ExplicitRequirementsFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deps/custom-reqs.txt", "flask>=3.0\n")
	writeFile(t, dir, "app.ipynb", "{}")

	job := newTestJob("app.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: true,
		BaseImage:                "python:3.11",
		RequirementsFile:         "deps/custom-reqs.txt",
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !strings.Contains(result.Source, "explicit-path") {
		t.Errorf("Source = %q, want substring 'explicit-path'", result.Source)
	}
	if !strings.Contains(result.Content, "COPY deps/custom-reqs.txt") {
		t.Errorf("Dockerfile missing COPY for explicit requirements:\n%s", result.Content)
	}
}

func TestGenerateDockerfile_ExistingDockerfile(t *testing.T) {
	dir := t.TempDir()
	dockerfileContent := "FROM custom-image:latest\nRUN echo hello\n"
	writeFile(t, dir, "Dockerfile", dockerfileContent)
	writeFile(t, dir, "notebook.ipynb", "{}")

	job := newTestJob("notebook.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: true,
		BaseImage:                "python:3.11",
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !result.UsingExistingDockerfile {
		t.Error("expected UsingExistingDockerfile=true")
	}
	if !strings.Contains(result.Source, "existing Dockerfile") {
		t.Errorf("Source = %q, want substring 'existing Dockerfile'", result.Source)
	}
	if result.Content != dockerfileContent {
		t.Errorf("Content = %q, want %q", result.Content, dockerfileContent)
	}
}

func TestGenerateDockerfile_BareBaseImage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notebook.ipynb", "{}")

	job := newTestJob("notebook.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: true,
		BaseImage:                "python:3.11",
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if result.UsingExistingDockerfile {
		t.Error("expected generated Dockerfile, not existing")
	}
	if !strings.Contains(result.Source, "base image only") {
		t.Errorf("Source = %q, want substring 'base image only'", result.Source)
	}
	if !strings.Contains(result.Content, "FROM python:3.11") {
		t.Errorf("Dockerfile missing base image:\n%s", result.Content)
	}
	if result.RequirementsFile != "" {
		t.Errorf("RequirementsFile should be empty, got %q", result.RequirementsFile)
	}
}

// --- Edge case tests ---

func TestGenerateDockerfile_AutoGenerateDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "pandas\n")
	writeFile(t, dir, "notebook.ipynb", "{}")

	job := newTestJob("notebook.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: false,
		PreferDockerfile:         true,
		BaseImage:                "python:3.11",
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	// With auto disabled + preferDockerfile, should fall back to base-image-only
	// since there's no Dockerfile on disk
	if result.UsingExistingDockerfile {
		t.Error("expected generated base-image Dockerfile (no Dockerfile exists on disk)")
	}
	if result.RequirementsFile != "" {
		t.Errorf("requirements.txt should be ignored when auto-detect is disabled, got %q", result.RequirementsFile)
	}
}

func TestGenerateDockerfile_PreferDockerfileOverRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "pandas\n")
	dockerfileContent := "FROM my-prebuilt:latest\n"
	writeFile(t, dir, "Dockerfile", dockerfileContent)
	writeFile(t, dir, "notebook.ipynb", "{}")

	job := newTestJob("notebook.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: true,
		PreferDockerfile:         true,
		BaseImage:                "python:3.11",
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !result.UsingExistingDockerfile {
		t.Error("expected UsingExistingDockerfile=true when preferDockerfile is set")
	}
	if result.Content != dockerfileContent {
		t.Errorf("Content = %q, want existing Dockerfile content", result.Content)
	}
}

func TestGenerateDockerfile_FallbackPriority(t *testing.T) {
	dir := t.TempDir()
	// Place requirements.txt at all three levels
	writeFile(t, dir, "notebooks/tier2/requirements.txt", "# notebook-level\n")
	writeFile(t, dir, "notebooks/requirements.txt", "# tier-level\n")
	writeFile(t, dir, "requirements.txt", "# root-level\n")
	writeFile(t, dir, "notebooks/tier2/analysis.ipynb", "{}")

	job := newTestJob("notebooks/tier2/analysis.ipynb", "python:3.11", nil)
	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	// Most specific (notebook directory) should win
	if !strings.Contains(result.Source, "notebook-directory") {
		t.Errorf("Source = %q, expected notebook-directory to take priority", result.Source)
	}
}

func TestGenerateDockerfile_CustomRequirementsSources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config/prod-deps.txt", "gunicorn\n")
	writeFile(t, dir, "requirements.txt", "dev-only\n")
	writeFile(t, dir, "notebook.ipynb", "{}")

	job := newTestJob("notebook.ipynb", "python:3.11", &mlopsv1alpha1.BuildConfigSpec{
		Enabled:                  true,
		AutoGenerateRequirements: true,
		BaseImage:                "python:3.11",
		RequirementsSources:      []string{"config/prod-deps.txt", "requirements.txt"},
	})

	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	if !strings.Contains(result.Source, "custom-chain") {
		t.Errorf("Source = %q, want substring 'custom-chain'", result.Source)
	}
	if !strings.Contains(result.Content, "config/prod-deps.txt") {
		t.Errorf("Dockerfile should reference custom-chain file:\n%s", result.Content)
	}
}

// --- Path traversal tests ---

func TestValidateAndJoinPath_RejectsAbsolutePath(t *testing.T) {
	_, err := validateAndJoinPath("/base", "/etc/passwd")
	if err == nil {
		t.Error("expected error for absolute path, got nil")
	}
}

func TestValidateAndJoinPath_RejectsTraversal(t *testing.T) {
	_, err := validateAndJoinPath("/base", "../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestValidateAndJoinPath_AcceptsValidSubpath(t *testing.T) {
	path, err := validateAndJoinPath("/base", "subdir/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/base", "subdir", "file.txt")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

// --- Validation warning tests ---

func TestValidateDockerfileGeneration_BothExistWarning(t *testing.T) {
	dir := t.TempDir()
	// ValidateDockerfileGeneration derives gitRepoPath = Dir(Dir(RequirementsFile)).
	// For a file at sub/requirements.txt, gitRepoPath = dir, so it looks for dir/Dockerfile.
	writeFile(t, dir, "sub/requirements.txt", "pandas\n")
	writeFile(t, dir, "Dockerfile", "FROM base\n")
	writeFile(t, dir, "sub/notebook.ipynb", "{}")

	job := newTestJob("sub/notebook.ipynb", "python:3.11", nil)
	result, err := GenerateDockerfile(job, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile() error: %v", err)
	}

	warnings := ValidateDockerfileGeneration(job, result)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Both requirements.txt and Dockerfile exist") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about both files existing, got %v", warnings)
	}
}
