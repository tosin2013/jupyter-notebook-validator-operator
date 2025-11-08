# ADR 012: Release and CI/CD Strategy

**Status:** Accepted  
**Date:** 2025-11-07  
**Deciders:** Development Team  
**Technical Story:** Automated release pipeline with testing and container image publishing

---

## Context and Problem Statement

The Jupyter Notebook Validator Operator requires a robust CI/CD pipeline to:
1. Automate building and testing on every commit
2. Publish container images to a registry (Quay.io)
3. Create GitHub releases with proper versioning
4. Run comprehensive tests before deployment
5. Ensure security scanning and vulnerability detection
6. Support both development and production releases

**Key Requirements:**
- Automated builds on push to main branch
- Automated testing (unit, integration, e2e)
- Container image publishing to Quay.io
- Semantic versioning (SemVer)
- Security scanning (Trivy, Snyk)
- Multi-architecture support (amd64, arm64)
- Release notes generation
- OLM bundle publishing

---

## Decision Drivers

1. **Automation First** - Minimize manual release steps
2. **Quality Gates** - No releases without passing tests
3. **Security** - Scan all images for vulnerabilities
4. **Traceability** - Link commits to releases
5. **Speed** - Fast feedback on PRs and commits
6. **Reliability** - Reproducible builds
7. **OpenShift Compatibility** - Images must work on OpenShift

---

## Considered Options

### Option 1: GitHub Actions (SELECTED)
**Pros:**
- Native GitHub integration
- Free for public repositories
- Excellent ecosystem of actions
- Matrix builds for multi-arch
- Secrets management built-in
- Easy to configure and maintain

**Cons:**
- Vendor lock-in to GitHub
- Limited to 2000 minutes/month for private repos

### Option 2: GitLab CI/CD
**Pros:**
- Powerful pipeline features
- Built-in container registry
- Excellent Kubernetes integration

**Cons:**
- Requires GitLab migration
- More complex configuration
- Additional infrastructure

### Option 3: Jenkins
**Pros:**
- Highly customizable
- Self-hosted option
- Large plugin ecosystem

**Cons:**
- Requires infrastructure management
- Complex setup and maintenance
- Slower feedback loop

---

## Decision Outcome

**Chosen option:** GitHub Actions with Quay.io registry

### Rationale:
1. **Native Integration** - Already using GitHub for source control
2. **Simplicity** - Minimal configuration, maximum automation
3. **Cost** - Free for public repositories
4. **Ecosystem** - Rich marketplace of pre-built actions
5. **Security** - Built-in secrets management
6. **Speed** - Fast build times with caching

---

## Implementation Details

### 1. Container Registry Configuration

**Registry:** Quay.io  
**Organization:** `takinosh`  
**Repository:** `jupyter-notebook-validator-operator`  
**Image URL:** `quay.io/takinosh/jupyter-notebook-validator-operator`

**Authentication:**
- Username: Stored in GitHub Secrets as `QUAY_USERNAME`
- Password: Stored in GitHub Secrets as `QUAY_PASSWORD`
- Robot account: `takinosh+jupyter_notebook_validator_operator`
- Login command: `podman login -u='$QUAY_USERNAME' -p='$QUAY_PASSWORD' quay.io`

**Image Tags:**
- `latest` - Latest commit on main branch
- `v<semver>` - Release tags (e.g., `v0.1.0`, `v0.2.0`)
- `<git-sha>` - Commit-specific tags for traceability
- `dev-<branch>` - Development branch builds

### 2. Versioning Strategy

**Semantic Versioning (SemVer):** `MAJOR.MINOR.PATCH`

- **MAJOR** (v1.0.0): Breaking API changes, incompatible CRD schema changes
- **MINOR** (v0.1.0): New features, backward-compatible changes
- **PATCH** (v0.0.1): Bug fixes, security patches

**Version Sources:**
1. Git tags (e.g., `v0.1.0`)
2. `VERSION` file in repository root
3. Operator manifests (`config/manager/kustomization.yaml`)

**Pre-release Tags:**
- `v0.1.0-alpha.1` - Alpha releases (unstable)
- `v0.1.0-beta.1` - Beta releases (feature complete, testing)
- `v0.1.0-rc.1` - Release candidates (production-ready)

### 3. GitHub Actions Workflows

#### **Workflow 1: CI (Continuous Integration)**
**File:** `.github/workflows/ci.yml`  
**Trigger:** Push to any branch, Pull Requests

**Jobs:**
1. **Lint and Format**
   - Run `go fmt`
   - Run `go vet`
   - Run `golangci-lint`

2. **Unit Tests**
   - Run `make test`
   - Generate coverage report
   - Upload to Codecov

3. **Build**
   - Run `make build`
   - Verify binary creation

4. **CRD Validation**
   - Run `make manifests`
   - Validate CRD schema
   - Check for breaking changes

5. **Security Scan**
   - Run `gosec` for Go code
   - Run `trivy` for dependencies

#### **Workflow 2: Build and Push Image**
**File:** `.github/workflows/build-push.yml`  
**Trigger:** Push to main branch, Git tags

**Jobs:**
1. **Build Multi-Arch Image**
   - Build for `linux/amd64` and `linux/arm64`
   - Use `docker/build-push-action`
   - Cache layers for speed

2. **Scan Image**
   - Run Trivy vulnerability scan
   - Fail on HIGH/CRITICAL vulnerabilities

3. **Push to Quay.io**
   - Tag with `latest`, `<git-sha>`, and `<version>` (if tag)
   - Push to `quay.io/takinosh/jupyter-notebook-validator-operator`

4. **Sign Image (Optional)**
   - Use Cosign for image signing
   - Verify signature in deployment

#### **Workflow 3: Release**
**File:** `.github/workflows/release.yml`  
**Trigger:** Git tag push (e.g., `v0.1.0`)

**Jobs:**
1. **Build Release Artifacts**
   - Build operator binary for multiple platforms
   - Generate Kustomize manifests
   - Create OLM bundle

2. **Create GitHub Release**
   - Generate release notes from commits
   - Upload artifacts (binaries, manifests, bundle)
   - Mark as pre-release if alpha/beta/rc

3. **Publish OLM Bundle**
   - Push bundle to OperatorHub.io (future)
   - Update catalog image

4. **Update Documentation**
   - Generate API docs
   - Update installation instructions

#### **Workflow 4: E2E Tests**
**File:** `.github/workflows/e2e.yml`  
**Trigger:** Manual, Scheduled (nightly)

**Jobs:**
1. **Setup Test Cluster**
   - Create Kind cluster
   - Install operator

2. **Run E2E Tests**
   - Deploy sample CRs
   - Verify validation workflow
   - Check status updates

3. **Cleanup**
   - Delete test cluster
   - Archive logs

### 4. Quality Gates

**Required Checks for Merge:**
- ✅ All unit tests pass
- ✅ Code coverage ≥ 70%
- ✅ No linting errors
- ✅ CRD validation passes
- ✅ Security scan passes (no HIGH/CRITICAL)
- ✅ Build succeeds

**Required Checks for Release:**
- ✅ All CI checks pass
- ✅ E2E tests pass
- ✅ Image vulnerability scan passes
- ✅ Manual approval (for production releases)

### 5. Security Considerations

**Secrets Management:**
- Store Quay.io credentials in GitHub Secrets (`QUAY_USERNAME` and `QUAY_PASSWORD`)
- Never commit credentials to repository
- Rotate credentials quarterly
- Use robot accounts with minimal permissions
- Username stored as secret for easy rotation and centralized management

**Image Security:**
- Run as non-root user (UID 1001)
- Drop all capabilities
- Read-only root filesystem
- No privileged containers
- Scan for vulnerabilities with Trivy

**Supply Chain Security:**
- Pin action versions with SHA
- Verify checksums of downloaded tools
- Use official base images only
- Sign images with Cosign (optional)

### 6. Rollback Strategy

**Automated Rollback:**
- If E2E tests fail after release, automatically revert
- Notify team via Slack/email

**Manual Rollback:**
1. Identify last known good version
2. Re-tag image: `docker tag quay.io/takinosh/jupyter-notebook-validator-operator:v0.1.0 quay.io/takinosh/jupyter-notebook-validator-operator:latest`
3. Update deployment: `kubectl set image deployment/controller-manager manager=quay.io/takinosh/jupyter-notebook-validator-operator:v0.1.0`
4. Create hotfix branch and fix issue

---

## Consequences

### Positive

1. **Automation** - Releases are fully automated, reducing human error
2. **Quality** - Multiple quality gates ensure stable releases
3. **Security** - Automated scanning catches vulnerabilities early
4. **Traceability** - Every image is linked to a commit and release
5. **Speed** - Fast feedback on PRs and commits
6. **Reliability** - Reproducible builds with caching

### Negative

1. **Complexity** - Multiple workflows to maintain
2. **Cost** - GitHub Actions minutes (mitigated by free tier)
3. **Vendor Lock-in** - Tied to GitHub ecosystem
4. **Learning Curve** - Team needs to learn GitHub Actions syntax

### Neutral

1. **Quay.io Dependency** - Requires Quay.io account and credentials
2. **Multi-Arch Builds** - Longer build times for arm64 support

---

## Compliance and Validation

### OpenShift Certification
- Images must pass Red Hat certification process
- Use UBI (Universal Base Image) for production
- Follow OpenShift best practices

### OperatorHub.io Publishing
- Create OLM bundle with proper metadata
- Pass OperatorHub.io validation
- Maintain bundle in community-operators repository

---

## Monitoring and Metrics

**CI/CD Metrics:**
- Build success rate
- Average build time
- Test pass rate
- Deployment frequency
- Mean time to recovery (MTTR)

**Image Metrics:**
- Image size
- Vulnerability count (by severity)
- Pull count from Quay.io
- Multi-arch usage

---

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Quay.io Documentation](https://docs.quay.io/)
- [Semantic Versioning](https://semver.org/)
- [Operator SDK Image Building](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#2-build-and-push-the-operator-image)
- [Trivy Security Scanner](https://github.com/aquasecurity/trivy)
- [Cosign Image Signing](https://github.com/sigstore/cosign)

---

## Related ADRs

- **ADR-001:** Operator Framework and SDK Version
- **ADR-004:** Deployment and Packaging Strategy
- **ADR-008:** Notebook Testing Strategy and Complexity Levels
- **ADR-010:** Observability and Monitoring Strategy

---

**Last Updated:** 2025-11-07  
**Next Review:** After first production release

