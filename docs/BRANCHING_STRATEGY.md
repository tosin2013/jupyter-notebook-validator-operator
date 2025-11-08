# Git Branching Strategy for Multi-Version OpenShift Support

**Date:** November 8, 2025  
**Decision:** Use Git branches to support multiple OpenShift versions simultaneously  
**Status:** ✅ RECOMMENDED APPROACH

## Overview

Instead of trying to support all OpenShift versions in a single codebase with complex dependency management, we'll use **Git branches** to maintain separate versions of the operator for different OpenShift releases.

## Branch Structure

```
main (development)
├── release-4.18 (OpenShift 4.18 - Kubernetes 1.31)
│   ├── k8s.io v0.31.x
│   ├── OpenShift API (Feb 2025 commit)
│   └── Tekton Pipeline v0.65.0
│
└── release-4.20 (OpenShift 4.20 - Kubernetes 1.33)
    ├── k8s.io v0.33.x
    ├── OpenShift API (Oct 2025 commit)
    └── Tekton Pipeline v0.66.0+
```

## Benefits of This Approach

### ✅ Advantages

1. **Clean Dependency Management**
   - Each branch has its own `go.mod` with compatible versions
   - No complex version conflicts or compatibility matrices
   - Clear separation of concerns

2. **Independent Development**
   - Develop features on `main` branch
   - Backport to `release-4.18` as needed
   - Forward-port to `release-4.20` when ready
   - Test each version independently

3. **Proven Pattern**
   - Used by Kubernetes, OpenShift, and many operators
   - Well-understood by the community
   - Supported by CI/CD tools

4. **Flexible Release Cadence**
   - Release 4.18 version when stable
   - Release 4.20 version when ready
   - No need to wait for all versions to be ready

5. **Easy Maintenance**
   - Bug fixes can be cherry-picked across branches
   - Security patches can be applied to all supported versions
   - Clear EOL path for old versions

### ⚠️ Considerations

1. **Multiple Releases to Maintain**
   - Need to maintain 2-3 active branches
   - Bug fixes may need to be applied to multiple branches
   - CI/CD needs to test all branches

2. **Documentation Overhead**
   - Need to document which branch supports which OpenShift version
   - Users need to know which version to use

3. **Cherry-picking Complexity**
   - Some features may not cherry-pick cleanly
   - API changes may require manual porting

## Implementation Plan

### Phase 1: Commit Current Documentation (NOW)

```bash
# Commit all documentation to main branch
git add docs/
git commit -m "docs: Add OpenShift version strategy and branching documentation

- Add OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md
- Add FINAL_STRATEGIC_DECISION.md
- Add NEXT_STEPS_ACTION_PLAN.md
- Add BRANCHING_STRATEGY.md
- Document decision to use branch-based multi-version support"

git push origin main
```

### Phase 2: Create release-4.18 Branch (THIS WEEK)

```bash
# Create release-4.18 branch from main
git checkout -b release-4.18
git push origin release-4.18

# Update dependencies for OpenShift 4.18
go get k8s.io/api@v0.31.10
go get k8s.io/apimachinery@v0.31.10
go get k8s.io/client-go@v0.31.10
go get sigs.k8s.io/controller-runtime@v0.19.4
go get github.com/openshift/api@5dd0bcfcbb795976926583d2abc9f28bb6a33ff9
go get github.com/tektoncd/pipeline@v0.65.0
go mod tidy

# Update documentation to reflect 4.18 target
echo "# OpenShift 4.18 Release Branch

This branch targets OpenShift 4.18 (Kubernetes 1.31).

## Dependencies
- k8s.io v0.31.10
- OpenShift API: 5dd0bcfcbb79 (Jan 2025)
- Tekton Pipeline: v0.65.0
- controller-runtime: v0.19.4

## Supported Versions
- OpenShift 4.18.x (tested)
- OpenShift 4.19.x (compatible)
- OpenShift 4.20.x (compatible)
" > docs/RELEASE_BRANCH_INFO.md

git add go.mod go.sum docs/RELEASE_BRANCH_INFO.md
git commit -m "chore: Configure dependencies for OpenShift 4.18 (Kubernetes 1.31)"
git push origin release-4.18
```

### Phase 3: Create release-4.20 Branch (REFERENCE - Q1 2026)

```bash
# Create release-4.20 branch from main
git checkout main
git checkout -b release-4.20
git push origin release-4.20

# Update dependencies for OpenShift 4.20
go get k8s.io/api@v0.33.0
go get k8s.io/apimachinery@v0.33.0
go get k8s.io/client-go@v0.33.0
go get sigs.k8s.io/controller-runtime@v0.20.x  # When available
go get github.com/openshift/api@<oct-2025-commit>
go get github.com/tektoncd/pipeline@v0.66.0  # Or compatible version
go mod tidy

# Update documentation to reflect 4.20 target
echo "# OpenShift 4.20 Release Branch

This branch targets OpenShift 4.20 (Kubernetes 1.33).

## Dependencies
- k8s.io v0.33.0
- OpenShift API: <commit> (Oct 2025)
- Tekton Pipeline: v0.66.0+
- controller-runtime: v0.20.x

## Supported Versions
- OpenShift 4.20.x (tested)
- OpenShift 4.21.x (compatible)
" > docs/RELEASE_BRANCH_INFO.md

git add go.mod go.sum docs/RELEASE_BRANCH_INFO.md
git commit -m "chore: Configure dependencies for OpenShift 4.20 (Kubernetes 1.33)"
git push origin release-4.20
```

### Phase 4: Update Main Branch (ONGOING)

```bash
# Main branch continues development
git checkout main

# Update README to document branching strategy
cat >> README.md << 'EOF'

## OpenShift Version Support

This operator uses a **branch-based versioning strategy** to support multiple OpenShift versions:

| Branch | OpenShift Version | Kubernetes Version | Status |
|--------|-------------------|-------------------|--------|
| `release-4.18` | 4.18.x, 4.19.x, 4.20.x | 1.31 | ✅ Stable |
| `release-4.20` | 4.20.x, 4.21.x | 1.33 | 🚧 Development |
| `main` | Development | Latest | 🔬 Experimental |

### Which Branch Should I Use?

- **Production on OpenShift 4.18**: Use `release-4.18` branch
- **Production on OpenShift 4.19**: Use `release-4.18` branch (forward compatible)
- **Production on OpenShift 4.20**: Use `release-4.18` branch (forward compatible) or `release-4.20` when stable
- **Development/Testing**: Use `main` branch

### Installation

```bash
# For OpenShift 4.18/4.19/4.20
kubectl apply -f https://github.com/tosin2013/jupyter-notebook-validator-operator/releases/download/v1.0.0-ocp4.18/install.yaml

# For OpenShift 4.20/4.21 (when available)
kubectl apply -f https://github.com/tosin2013/jupyter-notebook-validator-operator/releases/download/v1.0.0-ocp4.20/install.yaml
```
EOF

git add README.md
git commit -m "docs: Document branch-based OpenShift version support strategy"
git push origin main
```

## Development Workflow

### Feature Development

```bash
# Develop new features on main
git checkout main
git checkout -b feature/my-new-feature

# Make changes, commit, push
git add .
git commit -m "feat: Add new feature"
git push origin feature/my-new-feature

# Create PR to main
# After merge to main, decide if feature should be backported
```

### Backporting to release-4.18

```bash
# Cherry-pick feature to release-4.18
git checkout release-4.18
git cherry-pick <commit-hash>

# If conflicts, resolve and continue
git cherry-pick --continue

# Push to release-4.18
git push origin release-4.18
```

### Bug Fixes

```bash
# Fix bug on the oldest supported branch first
git checkout release-4.18
git checkout -b fix/critical-bug

# Make fix, commit
git add .
git commit -m "fix: Critical bug in build strategy"
git push origin fix/critical-bug

# Create PR to release-4.18
# After merge, cherry-pick to newer branches

git checkout release-4.20
git cherry-pick <commit-hash>
git push origin release-4.20

git checkout main
git cherry-pick <commit-hash>
git push origin main
```

## Release Process

### Releasing from release-4.18

```bash
git checkout release-4.18

# Tag the release
git tag -a v1.0.0-ocp4.18 -m "Release v1.0.0 for OpenShift 4.18"
git push origin v1.0.0-ocp4.18

# Build and publish operator image
make docker-build docker-push IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:v1.0.0-ocp4.18

# Generate release artifacts
make release-manifests VERSION=v1.0.0-ocp4.18

# Create GitHub release
gh release create v1.0.0-ocp4.18 \
  --title "v1.0.0 for OpenShift 4.18" \
  --notes "Release for OpenShift 4.18.x, 4.19.x, 4.20.x (Kubernetes 1.31)" \
  dist/install.yaml
```

### Releasing from release-4.20

```bash
git checkout release-4.20

# Tag the release
git tag -a v1.0.0-ocp4.20 -m "Release v1.0.0 for OpenShift 4.20"
git push origin v1.0.0-ocp4.20

# Build and publish operator image
make docker-build docker-push IMG=quay.io/tosin2013/jupyter-notebook-validator-operator:v1.0.0-ocp4.20

# Generate release artifacts
make release-manifests VERSION=v1.0.0-ocp4.20

# Create GitHub release
gh release create v1.0.0-ocp4.20 \
  --title "v1.0.0 for OpenShift 4.20" \
  --notes "Release for OpenShift 4.20.x, 4.21.x (Kubernetes 1.33)" \
  dist/install.yaml
```

## CI/CD Configuration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Test

on:
  push:
    branches:
      - main
      - release-*
  pull_request:
    branches:
      - main
      - release-*

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        branch:
          - main
          - release-4.18
          - release-4.20
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ matrix.branch }}
      
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      
      - name: Test
        run: make test
      
      - name: Build
        run: make build
```

## Branch Lifecycle

### Active Branches

| Branch | Created | EOL | Status |
|--------|---------|-----|--------|
| `release-4.18` | Nov 2025 | Aug 2026 | ✅ Active |
| `release-4.20` | Q1 2026 | Oct 2028 | 🚧 Planned |
| `main` | Always | Never | 🔬 Development |

### EOL Process

When a branch reaches EOL:

1. **Announce EOL** (3 months before)
2. **Stop accepting new features** (1 month before)
3. **Security fixes only** (until EOL)
4. **Archive branch** (at EOL)
5. **Update documentation** (remove from supported list)

## Comparison with Other Approaches

### Branch-Based (CHOSEN) vs Single Branch

| Aspect | Branch-Based | Single Branch |
|--------|--------------|---------------|
| Dependency Management | ✅ Clean, isolated | ❌ Complex, conflicts |
| Testing | ✅ Independent per version | ⚠️ Must test all versions |
| Releases | ✅ Independent cadence | ❌ All versions together |
| Maintenance | ⚠️ Multiple branches | ✅ Single codebase |
| Backporting | ⚠️ Cherry-pick needed | ✅ Automatic |
| Complexity | ⚠️ Branch management | ❌ Dependency hell |

## Examples from the Ecosystem

### Kubernetes
- `release-1.31` branch for Kubernetes 1.31
- `release-1.32` branch for Kubernetes 1.32
- `release-1.33` branch for Kubernetes 1.33

### OpenShift
- `release-4.18` branch for OpenShift 4.18
- `release-4.19` branch for OpenShift 4.19
- `release-4.20` branch for OpenShift 4.20

### Tekton Pipeline
- `release-v0.65.x` branch for v0.65 releases
- `release-v0.66.x` branch for v0.66 releases
- `main` branch for development

## Conclusion

**Branch-based versioning is the RIGHT approach** for this operator because:

1. ✅ **Clean Dependencies**: Each branch has compatible versions
2. ✅ **Independent Testing**: Test each OpenShift version separately
3. ✅ **Flexible Releases**: Release when ready, not when all versions are ready
4. ✅ **Proven Pattern**: Used by Kubernetes, OpenShift, Tekton
5. ✅ **Easy Maintenance**: Cherry-pick bug fixes across branches

## Next Steps

1. ✅ **Commit Documentation** (NOW)
2. ✅ **Create release-4.18 Branch** (THIS WEEK)
3. ✅ **Configure Dependencies** (THIS WEEK)
4. ✅ **Test on OpenShift 4.18** (NEXT WEEK)
5. 🚧 **Create release-4.20 Branch** (Q1 2026)
6. 🚧 **Test on OpenShift 4.20** (Q1 2026)

**Ready to proceed?**

