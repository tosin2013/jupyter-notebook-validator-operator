# ADR 038: Requirements.txt Auto-Detection and Dockerfile Generation Strategy

## Status
Proposed

## Context

### Problem Statement
Developers maintain `requirements.txt` for local notebook development, but the operator ignores this file and requires manual Dockerfile creation. This leads to:
- **Duplicate dependency management**: Maintaining both requirements.txt (local) and Dockerfile (operator)
- **Environment drift**: Local environment ≠ validation environment ≠ production environment
- **Manual synchronization errors**: Forgetting to update Dockerfile when requirements.txt changes
- **Increased onboarding friction**: Developers must learn Dockerfile syntax just to validate notebooks

**Current Developer Experience (BROKEN)**:
```bash
notebooks/02-anomaly-detection/
├── 01-isolation-forest-implementation.ipynb
├── requirements.txt  # ← Operator IGNORES this!
│   seaborn==0.12.2
│   joblib==1.3.2
│   scikit-learn==1.3.2
└── README.md

# Developer must ALSO create and maintain:
Dockerfile  # ← Duplicate dependency list!
FROM pytorch:2025.1
RUN pip install seaborn==0.12.2 joblib==1.3.2 scikit-learn==1.3.2
```

### Desired Developer Experience
```bash
notebooks/02-anomaly-detection/
├── 01-isolation-forest-implementation.ipynb
├── requirements.txt  # ← SINGLE source of truth
│   seaborn==0.12.2
│   joblib==1.3.2
│   scikit-learn==1.3.2
└── README.md

# Operator automatically:
# 1. Detects requirements.txt
# 2. Generates Dockerfile from it
# 3. Builds image with dependencies
# 4. Uses image for validation AND production
```

### User Feedback
From OPERATOR-FEEDBACK.md (OpenShift AI Ops Self-Healing Platform Team):

> **Enhancement #2: Auto-Detect requirements.txt**
>
> **Priority**: 🔴 Critical
> **Complexity**: Medium
> **Impact**: Enables standard Python workflow
>
> "Developers maintain `requirements.txt` for local development, but operator **ignores** it... Result: Drift between local dev and validation"

## Decision

We will implement **automatic detection and usage of requirements.txt files** with a fallback chain strategy.

### Detection Strategy

#### Fallback Chain
The operator will search for `requirements.txt` in this order:

1. **Notebook directory**: `notebooks/02-anomaly-detection/requirements.txt` (most specific)
2. **Tier directory**: `notebooks/requirements.txt` (shared across tier)
3. **Repository root**: `requirements.txt` (project-wide)
4. **Explicit path**: User-specified via `spec.podConfig.buildConfig.requirementsFile`
5. **Dockerfile**: Fall back to existing Dockerfile if no requirements.txt found
6. **Base image**: Use bare base image if no dependencies specified

#### Priority Rules
- If both requirements.txt and Dockerfile exist: **Warn user** and prefer requirements.txt by default
- User can override with `spec.podConfig.buildConfig.preferDockerfile: true`

### CRD API Changes

```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: validate-ml-notebook
spec:
  notebook:
    git:
      url: https://github.com/org/repo.git
    path: "notebooks/02-anomaly-detection/01-isolation-forest.ipynb"

  podConfig:
    buildConfig:
      enabled: true
      strategy: tekton  # or s2i

      # NEW: Auto-detection options
      autoGenerateRequirements: true  # Default: true (enable auto-detection)

      # NEW: Explicit path (optional)
      requirementsFile: "notebooks/02-anomaly-detection/requirements.txt"

      # NEW: Fallback chain (optional, for advanced use cases)
      requirementsSources:
        - "notebooks/02-anomaly-detection/requirements.txt"  # Try notebook-specific
        - "notebooks/requirements.txt"                       # Try tier-level
        - "requirements.txt"                                 # Fall back to root

      # NEW: Dockerfile preference (optional)
      preferDockerfile: false  # Default: false (prefer requirements.txt)

      # Existing fields
      baseImage: "quay.io/opendatahub/workbench-images:jupyter-datascience-ubi9-python-3.11-2025.1"
      dockerfile: "Dockerfile"  # Used as fallback
```

### Dockerfile Generation Algorithm

```python
def generate_dockerfile(job, git_repo_path):
    """
    Generate Dockerfile from requirements.txt or fall back to existing Dockerfile.
    """
    base_image = job.spec.buildConfig.baseImage or "python:3.11-slim"
    notebook_path = job.spec.notebook.path
    notebook_dir = os.path.dirname(notebook_path)

    # Step 1: Try to find requirements.txt
    requirements_file = None

    if job.spec.buildConfig.requirementsFile:
        # Explicit path specified
        requirements_file = os.path.join(git_repo_path, job.spec.buildConfig.requirementsFile)
    elif job.spec.buildConfig.requirementsSources:
        # Try fallback chain
        for source in job.spec.buildConfig.requirementsSources:
            candidate = os.path.join(git_repo_path, source)
            if os.path.exists(candidate):
                requirements_file = candidate
                break
    else:
        # Auto-detection (default fallback chain)
        candidates = [
            os.path.join(git_repo_path, notebook_dir, "requirements.txt"),  # Notebook-specific
            os.path.join(git_repo_path, "notebooks", "requirements.txt"),   # Tier-level
            os.path.join(git_repo_path, "requirements.txt"),                # Repo root
        ]
        for candidate in candidates:
            if os.path.exists(candidate):
                requirements_file = candidate
                break

    # Step 2: Generate Dockerfile from requirements.txt
    if requirements_file and os.path.exists(requirements_file):
        relative_path = os.path.relpath(requirements_file, git_repo_path)
        return f"""
FROM {base_image}

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Copy and install project dependencies
COPY {relative_path} /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

# Set working directory
WORKDIR /workspace

# Health check (optional)
RUN python -c "import sys; print(f'Python {{sys.version}}')"
        """.strip()

    # Step 3: Fall back to existing Dockerfile
    dockerfile_path = job.spec.buildConfig.dockerfile or "Dockerfile"
    dockerfile_full_path = os.path.join(git_repo_path, dockerfile_path)

    if os.path.exists(dockerfile_full_path):
        with open(dockerfile_full_path, 'r') as f:
            return f.read()

    # Step 4: Fall back to base image only
    return f"""
FROM {base_image}

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

WORKDIR /workspace
    """.strip()
```

### Implementation in Go

```go
// internal/controller/dockerfile_generator.go
package controller

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// GenerateDockerfile generates a Dockerfile from requirements.txt or falls back to existing Dockerfile
func GenerateDockerfile(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (string, string, error) {
    baseImage := job.Spec.PodConfig.BuildConfig.BaseImage
    if baseImage == "" {
        baseImage = "quay.io/opendatahub/workbench-images:jupyter-datascience-ubi9-python-3.11-2025.1"
    }

    // Step 1: Find requirements.txt
    requirementsFile, source := findRequirementsFile(job, gitRepoPath)

    // Step 2: Generate from requirements.txt if found
    if requirementsFile != "" {
        relativePath, err := filepath.Rel(gitRepoPath, requirementsFile)
        if err != nil {
            return "", "", err
        }

        dockerfile := fmt.Sprintf(`FROM %s

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Copy and install project dependencies
COPY %s /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

# Set working directory
WORKDIR /workspace

# Health check
RUN python -c "import sys; print(f'Python {sys.version}')"
`, baseImage, relativePath)

        return dockerfile, fmt.Sprintf("generated from %s (%s)", relativePath, source), nil
    }

    // Step 3: Fall back to existing Dockerfile
    dockerfilePath := job.Spec.PodConfig.BuildConfig.Dockerfile
    if dockerfilePath == "" {
        dockerfilePath = "Dockerfile"
    }

    dockerfileFullPath := filepath.Join(gitRepoPath, dockerfilePath)
    if _, err := os.Stat(dockerfileFullPath); err == nil {
        content, err := os.ReadFile(dockerfileFullPath)
        if err != nil {
            return "", "", err
        }
        return string(content), fmt.Sprintf("using existing %s", dockerfilePath), nil
    }

    // Step 4: Fall back to base image only
    dockerfile := fmt.Sprintf(`FROM %s

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

WORKDIR /workspace
`, baseImage)

    return dockerfile, "generated from base image (no dependencies)", nil
}

// findRequirementsFile searches for requirements.txt using the fallback chain
func findRequirementsFile(job *mlopsv1alpha1.NotebookValidationJob, gitRepoPath string) (string, string) {
    // Explicit path specified
    if job.Spec.PodConfig.BuildConfig.RequirementsFile != "" {
        candidate := filepath.Join(gitRepoPath, job.Spec.PodConfig.BuildConfig.RequirementsFile)
        if _, err := os.Stat(candidate); err == nil {
            return candidate, "explicit path"
        }
    }

    // Custom fallback chain
    if len(job.Spec.PodConfig.BuildConfig.RequirementsSources) > 0 {
        for _, source := range job.Spec.PodConfig.BuildConfig.RequirementsSources {
            candidate := filepath.Join(gitRepoPath, source)
            if _, err := os.Stat(candidate); err == nil {
                return candidate, "custom fallback chain"
            }
        }
    }

    // Default fallback chain
    notebookPath := job.Spec.Notebook.Path
    notebookDir := filepath.Dir(notebookPath)

    candidates := []struct {
        path   string
        source string
    }{
        {filepath.Join(gitRepoPath, notebookDir, "requirements.txt"), "notebook directory"},
        {filepath.Join(gitRepoPath, "notebooks", "requirements.txt"), "tier directory"},
        {filepath.Join(gitRepoPath, "requirements.txt"), "repository root"},
    }

    for _, candidate := range candidates {
        if _, err := os.Stat(candidate.path); err == nil {
            return candidate.path, candidate.source
        }
    }

    return "", ""
}
```

### Integration with Build Strategies

#### Tekton Pipeline Strategy

```go
// pkg/build/tekton_strategy.go
func (s *TektonStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
    // Generate Dockerfile from requirements.txt or fall back
    dockerfile, source, err := GenerateDockerfile(job, s.gitRepoPath)
    if err != nil {
        return nil, fmt.Errorf("failed to generate Dockerfile: %w", err)
    }

    log.Info("Dockerfile generated", "source", source)

    // Create ConfigMap with Dockerfile
    cm := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-dockerfile", job.Name),
            Namespace: job.Namespace,
        },
        Data: map[string]string{
            "Dockerfile": dockerfile,
        },
    }

    if err := s.client.Create(ctx, cm); err != nil {
        return nil, err
    }

    // Create Tekton PipelineRun that uses this Dockerfile
    // ... rest of Tekton logic
}
```

#### S2I Build Strategy (OpenShift)

```go
// pkg/build/s2i_strategy.go
func (s *S2IStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
    // Generate Dockerfile from requirements.txt
    dockerfile, source, err := GenerateDockerfile(job, s.gitRepoPath)
    if err != nil {
        return nil, fmt.Errorf("failed to generate Dockerfile: %w", err)
    }

    log.Info("Dockerfile generated for S2I build", "source", source)

    // Create BuildConfig with inline Dockerfile strategy
    bc := &buildv1.BuildConfig{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-build", job.Name),
            Namespace: job.Namespace,
        },
        Spec: buildv1.BuildConfigSpec{
            Source: buildv1.BuildSource{
                Type: buildv1.BuildSourceDockerfile,
                Dockerfile: &dockerfile,
                Git: &buildv1.GitBuildSource{
                    URI: job.Spec.Notebook.Git.URL,
                    Ref: job.Spec.Notebook.Git.Ref,
                },
            },
            Strategy: buildv1.BuildStrategy{
                Type: buildv1.DockerBuildStrategyType,
                DockerStrategy: &buildv1.DockerBuildStrategy{
                    DockerfilePath: ".", // Inline Dockerfile
                },
            },
            // ... rest of S2I logic
        },
    }

    if err := s.client.Create(ctx, bc); err != nil {
        return nil, err
    }

    return s.startBuild(ctx, bc)
}
```

## Consequences

### Positive
- ✅ **Standard Python workflow**: Developers use requirements.txt as they do locally
- ✅ **No environment drift**: Local, CI, and production all use same dependencies
- ✅ **Reduced maintenance**: Single source of truth for dependencies
- ✅ **Improved developer experience**: No Dockerfile knowledge required
- ✅ **Backward compatible**: Existing Dockerfiles still work as fallback
- ✅ **Flexible**: Supports per-notebook, per-tier, and project-wide dependencies

### Negative
- 🔄 **Increased complexity**: More detection logic in operator
- 📝 **Warning needed**: When both requirements.txt and Dockerfile exist
- 🔧 **Testing burden**: Must test all fallback scenarios

### Neutral
- 📚 **Documentation update**: Need to document fallback chain and best practices
- 🔄 **Migration path**: Gradual adoption (Dockerfile still works)

## Implementation Notes

### Testing Strategy

#### Unit Tests
- [ ] Test requirements.txt detection in notebook directory
- [ ] Test requirements.txt detection in tier directory
- [ ] Test requirements.txt detection in repo root
- [ ] Test explicit requirementsFile path
- [ ] Test custom requirementsSources fallback
- [ ] Test Dockerfile fallback when no requirements.txt
- [ ] Test base image fallback when neither exists

#### Integration Tests
- [ ] Build with requirements.txt only (no Dockerfile)
- [ ] Build with Dockerfile only (no requirements.txt)
- [ ] Build with both (verify preference and warning)
- [ ] Build with explicit requirementsFile
- [ ] Build with custom fallback chain

#### E2E Tests
- [ ] End-to-end: requirements.txt → build → validation → production workbench
- [ ] Multi-tier notebooks with different requirements.txt files
- [ ] Verify no environment drift between local and validation

### Developer Workflow Example

```bash
# 1. Developer creates notebook with dependencies
cd notebooks/02-anomaly-detection/
cat > requirements.txt << EOF
seaborn==0.12.2
joblib==1.3.2
scikit-learn==1.3.2
numpy==1.24.3
pandas==2.0.3
EOF

# 2. Test locally
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
jupyter notebook 01-isolation-forest-implementation.ipynb

# 3. Commit and push
git add requirements.txt 01-isolation-forest-implementation.ipynb
git commit -m "feat: add isolation forest notebook"
git push

# 4. Operator automatically:
#    - Detects requirements.txt in notebook directory
#    - Generates Dockerfile from it
#    - Builds image with dependencies
#    - Validates notebook with built image
#    - Pushes image to registry for production use
```

### Status Reporting

Add informational message to status to show which source was used:

```yaml
status:
  phase: Building
  message: "Building image from requirements.txt (notebook directory: notebooks/02-anomaly-detection/requirements.txt)"
  buildStatus:
    phase: Running
    dockerfileSource: "generated from notebooks/02-anomaly-detection/requirements.txt (notebook directory)"
```

### Warning for Conflicting Files

When both requirements.txt and Dockerfile exist:

```yaml
status:
  phase: Building
  conditions:
    - type: Warning
      status: "True"
      reason: "ConflictingDependencies"
      message: |
        Both requirements.txt and Dockerfile found. Using requirements.txt by default.
        To use Dockerfile instead, set spec.podConfig.buildConfig.preferDockerfile: true
```

## References

- [OPERATOR-FEEDBACK.md](../../OPERATOR-FEEDBACK.md) - Enhancement #2: Auto-Detect requirements.txt
- [Python Packaging Guide](https://packaging.python.org/en/latest/guides/writing-pyproject-toml/)
- [pip requirements.txt Format](https://pip.pypa.io/en/stable/reference/requirements-file-format/)

## Related ADRs

- [ADR-037: Build-Validation Sequencing](037-build-validation-sequencing-and-state-machine.md) - Build must complete before validation
- [ADR-039: Dependency Version Pinning](039-dependency-version-pinning-policy.md) - Enforce pinning in requirements.txt
- [ADR-040: Shared Image Strategy](040-shared-image-validation-production.md) - Use validated image in production

## Revision History

| Date | Author | Description |
|------|--------|-------------|
| 2025-11-20 | Claude Code | Initial proposal based on production feedback |
