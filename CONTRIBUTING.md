# Contributing to Jupyter Notebook Validator Operator

Thank you for your interest in contributing! This guide covers everything you need to get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Making Changes](#making-changes)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Reporting Issues](#reporting-issues)
- [Specialized Contributions](#specialized-contributions)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- **Go 1.22+** ([install guide](https://go.dev/doc/install))
- **Docker** or **Podman** (for building container images)
- **kubectl** or **oc** CLI
- **make**
- **A Kubernetes or OpenShift cluster** (for integration/e2e testing)

### Fork and Clone

1. Fork the repository on GitHub.
2. Clone your fork locally:

```bash
git clone https://github.com/<your-username>/jupyter-notebook-validator-operator.git
cd jupyter-notebook-validator-operator
```

3. Add the upstream remote:

```bash
git remote add upstream https://github.com/tosin2013/jupyter-notebook-validator-operator.git
```

## Development Environment

### Build the Operator

```bash
make build
```

### Run Tests

```bash
# Unit and integration tests
make test

# End-to-end tests (requires a running cluster)
make test-e2e
```

### Code Quality

Run all of the following before submitting a PR:

```bash
make fmt       # Format code
make vet       # Go static analysis
make lint      # Linter (golangci-lint)
make lint-fix  # Auto-fix linting issues
```

### Regenerate CRDs and DeepCopy

If you modify API types in `api/v1alpha1/`, regenerate the manifests:

```bash
make manifests generate
```

### Run Locally

```bash
# Install CRDs into your cluster
make install

# Run the operator against the cluster
make run
```

## Making Changes

### Branch Naming

Create a descriptive branch from `main`:

```
feature/short-description
fix/issue-number-description
docs/what-changed
```

### Commit Messages

Write clear, concise commit messages:

- Use the imperative mood ("Add feature" not "Added feature")
- Reference related issues when applicable (e.g., `Fixes #42`)
- Keep the subject line under 72 characters

### Code Style

- Follow standard Go conventions and idioms.
- All exported types and functions must have doc comments.
- Keep functions focused and testable.
- Add tests for new functionality.

## Submitting a Pull Request

1. Ensure your branch is up to date with `main`:

```bash
git fetch upstream
git rebase upstream/main
```

2. Verify that all checks pass locally:

```bash
make fmt vet lint test
```

3. Push your branch and open a PR against `main`.

4. In your PR description, include:
   - A summary of what changed and why.
   - The type of change (bug fix, feature, refactor, docs).
   - How you tested the changes.
   - Any related issue numbers.

5. A maintainer will review your PR. Please be responsive to feedback.

## Reporting Issues

- **Bugs**: Use the [Bug Report](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?template=bug_report.md) template.
- **Feature requests**: Use the [Feature Request](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?template=feature_request.md) template.

Include as much detail as possible: operator version, Kubernetes/OpenShift version, logs, and steps to reproduce.

## Community

For general questions, usage tips, and ideas that are not a bug or a tracked feature request, use [GitHub Discussions](https://github.com/tosin2013/jupyter-notebook-validator-operator/discussions).

### Maintainer note: GitHub “About” metadata

To sync the repository description, homepage URL, topics, and Discussions flag via the GitHub CLI (after `gh auth login`):

```bash
./scripts/update-github-repo-metadata.sh
```

## Specialized Contributions

### Adding Model Serving Platforms

The operator supports a plugin-based architecture for model serving platforms. For detailed instructions on adding support for new platforms (e.g., custom inference servers), see the [Contributing Model Platforms Guide](docs/CONTRIBUTING_MODEL_PLATFORMS.md).

### Helm Chart

Changes to the Helm chart in `helm/` should be validated with:

```bash
make helm-lint
make helm-template
```

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
