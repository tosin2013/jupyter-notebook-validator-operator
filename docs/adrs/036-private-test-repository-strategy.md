# ADR-036: Private Test Repository Strategy

**Status**: Accepted  
**Date**: 2025-11-11  
**Authors**: Sophia (AI Assistant), User Feedback  
**Related**: ADR-033 (E2E Testing), ADR-034 (Dual Testing), ADR-035 (Test Tier Organization), ADR-020 (Git Authentication)

## Context

The Jupyter Notebook Validator Operator needs to validate git authentication features (HTTPS and SSH) for accessing private repositories. This requires a test repository with controlled access.

### Current Situation

**Test Repository**: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks` (private)

**Purpose**:
- Test git authentication (HTTPS with PAT, SSH with keys)
- Validate operator's ability to clone private repositories
- Ensure credentials are properly injected into validation pods
- Test error handling for authentication failures

### Problem Statement

We need a testing strategy that:
1. **Validates Authentication**: Tests both HTTPS and SSH git authentication
2. **Protects Test Data**: Keeps test notebooks private for security testing
3. **Enables Public Documentation**: Allows users to understand the testing approach
4. **Plans for Future**: Provides path to public test repository for community

### Authentication Testing Requirements

The operator supports two authentication methods (ADR-020):

#### HTTPS Authentication
```yaml
spec:
  notebook:
    git:
      url: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git
      credentialsSecret: git-https-credentials  # username + password/PAT
```

#### SSH Authentication
```yaml
spec:
  notebook:
    git:
      url: git@github.com:tosin2013/jupyter-notebook-validator-test-notebooks.git
      credentialsSecret: git-ssh-credentials  # ssh-privatekey
```

## Decision

Use a **private test repository** for authentication testing while documenting the approach for users who want to replicate the testing strategy with their own repositories.

### Strategy Components

#### 1. Private Test Repository (Current)
**Repository**: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks` (private)

**Purpose**:
- ✅ Test HTTPS authentication with Personal Access Token (PAT)
- ✅ Test SSH authentication with SSH keys
- ✅ Validate credential injection into validation pods
- ✅ Test authentication error handling
- ✅ Ensure operator works with private repositories

**Access Control**:
- Owner: tosin2013 (Tosin Akinosho)
- Visibility: Private
- Authentication: Required for all operations

#### 2. Public Documentation (Current)
**Location**: `docs/TESTING_WITH_PRIVATE_REPOS.md` (to be created)

**Content**:
- How to create your own private test repository
- How to configure git credentials for testing
- How to run authentication tests
- Troubleshooting authentication issues
- Example test notebooks structure

**Benefits**:
- Users can replicate testing with their own repos
- Clear documentation of authentication testing approach
- No dependency on specific private repository

#### 3. Public Test Repository (Future - Phase 3)
**Planned Repository**: `https://github.com/tosin2013/jupyter-notebook-validator-public-tests` (future)

**Purpose**:
- ✅ Test basic notebook execution (no authentication)
- ✅ Provide public examples for community
- ✅ Enable community contributions
- ✅ Demonstrate operator capabilities

**Scope**:
- Tier 1 notebooks only (simple validation)
- No authentication required
- Public examples and tutorials
- Community-contributed test cases

**Timeline**: Phase 3 (after authentication testing is stable)

### Implementation Details

#### Private Repository Structure
```
jupyter-notebook-validator-test-notebooks/ (PRIVATE)
├── notebooks/
│   ├── tier1-simple/              # Basic validation (4 notebooks)
│   ├── tier2-intermediate/        # Build integration (4+ notebooks)
│   └── tier3-complex/             # Model inference (5+ notebooks)
├── deployments/                   # Model deployment manifests
├── lib/                           # Shared Python libraries
├── scripts/                       # Test execution scripts
├── requirements.txt
└── README.md                      # Private repo documentation
```

#### Public Documentation Structure
```
jupyter-notebook-validator-operator/
└── docs/
    ├── TESTING_WITH_PRIVATE_REPOS.md    # NEW: Guide for users
    ├── AUTHENTICATION_TESTING.md        # NEW: Authentication test guide
    └── adrs/
        └── 036-private-test-repository-strategy.md
```

#### Authentication Testing Workflow

**Step 1: Create Git Credentials Secret (HTTPS)**
```bash
# Create Personal Access Token at: https://github.com/settings/tokens
# Scopes: repo (full control)

oc create secret generic git-https-credentials \
  --from-literal=username=tosin2013 \
  --from-literal=password=ghp_YOUR_TOKEN \
  -n e2e-tests
```

**Step 2: Create Git Credentials Secret (SSH)**
```bash
# Generate SSH key
ssh-keygen -t rsa -b 4096 -f ~/.ssh/jupyter_validator_key -N ""

# Add public key to GitHub: Settings → SSH and GPG keys
cat ~/.ssh/jupyter_validator_key.pub

# Create secret
oc create secret generic git-ssh-credentials \
  --from-file=ssh-privatekey=$HOME/.ssh/jupyter_validator_key \
  -n e2e-tests
```

**Step 3: Run Authentication Tests**
```bash
# Test HTTPS authentication
oc apply -f examples/notebookvalidationjob-https-auth.yaml

# Test SSH authentication
oc apply -f examples/notebookvalidationjob-ssh-auth.yaml

# Verify authentication worked
oc get notebookvalidationjob -n e2e-tests
oc logs -n e2e-tests <pod-name> -c git-clone
```

### User Documentation Approach

Users who want to test with their own private repositories can follow this guide:

#### Creating Your Own Private Test Repository

**Step 1: Create Private Repository**
```bash
# On GitHub, create a new private repository
# Name: my-notebook-tests
# Visibility: Private
```

**Step 2: Add Test Notebooks**
```bash
git clone https://github.com/YOUR_USERNAME/my-notebook-tests.git
cd my-notebook-tests

# Create a simple test notebook
mkdir -p notebooks/tier1-simple
cat > notebooks/tier1-simple/01-hello-world.ipynb << 'EOF'
{
  "cells": [
    {
      "cell_type": "code",
      "execution_count": null,
      "metadata": {},
      "outputs": [],
      "source": ["print('Hello, World!')"]
    }
  ],
  "metadata": {},
  "nbformat": 4,
  "nbformat_minor": 5
}
EOF

git add .
git commit -m "Add test notebook"
git push origin main
```

**Step 3: Configure Authentication**
```bash
# Create HTTPS credentials
oc create secret generic git-https-credentials \
  --from-literal=username=YOUR_USERNAME \
  --from-literal=password=YOUR_PAT \
  -n YOUR_NAMESPACE
```

**Step 4: Test with Operator**
```yaml
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: test-private-repo
spec:
  notebook:
    git:
      url: https://github.com/YOUR_USERNAME/my-notebook-tests.git
      ref: main
      credentialsSecret: git-https-credentials
    path: notebooks/tier1-simple/01-hello-world.ipynb
  podConfig:
    containerImage: quay.io/jupyter/minimal-notebook:latest
```

### Future Public Repository Plan

**Phase 3 Goals**:
1. Create public test repository with Tier 1 notebooks
2. Enable community contributions
3. Provide public examples and tutorials
4. Maintain private repository for authentication testing

**Public Repository Scope**:
- ✅ Tier 1 notebooks (no authentication required)
- ✅ Public examples and tutorials
- ✅ Community-contributed test cases
- ❌ No authentication testing (use private repo)
- ❌ No sensitive data or credentials

**Benefits**:
- Community can contribute test cases
- Public examples for documentation
- No authentication barriers for basic testing
- Private repo still used for authentication testing

## Consequences

### Positive

- ✅ **Authentication Testing**: Validates git authentication features
- ✅ **Security**: Keeps test data private for security testing
- ✅ **User Documentation**: Clear guide for users to replicate testing
- ✅ **Future Flexibility**: Path to public repository for community
- ✅ **Controlled Access**: Owner controls test repository access
- ✅ **Real-World Testing**: Tests actual private repository scenarios

### Negative

- ❌ **Access Dependency**: Tests depend on private repository access
- ❌ **Credential Management**: Requires secure credential storage
- ❌ **Community Barrier**: Community cannot directly contribute to test repo
- ❌ **Documentation Overhead**: Need to document approach for users

### Neutral

- ⚠️ **Dual Repository Strategy**: Will maintain both private and public repos
- ⚠️ **Maintenance**: Need to keep both repos synchronized

## Alternatives Considered

### Alternative 1: Public Test Repository Only
- **Pros**: No authentication barriers, community contributions
- **Cons**: Cannot test authentication features, no private repo scenarios
- **Rejected**: Authentication testing is critical feature

### Alternative 2: Mock Git Server
- **Pros**: No external dependencies, full control
- **Cons**: Does not test real GitHub authentication, complex setup
- **Rejected**: Real-world testing is more valuable

### Alternative 3: Multiple Private Repositories
- **Pros**: Test different authentication scenarios
- **Cons**: Complex management, multiple credentials
- **Rejected**: Single private repo is sufficient

## Implementation Plan

### Phase 1: Current State (Completed)
1. ✅ Private test repository created
2. ✅ Test notebooks organized into tiers
3. ✅ Authentication testing working

### Phase 2: Documentation (Week 1)
4. Create `docs/TESTING_WITH_PRIVATE_REPOS.md`
5. Create `docs/AUTHENTICATION_TESTING.md`
6. Document credential creation process
7. Add troubleshooting guide

### Phase 3: Public Repository (Future)
8. Create public test repository
9. Migrate Tier 1 notebooks to public repo
10. Enable community contributions
11. Update documentation

## Verification

### Success Criteria
- [ ] HTTPS authentication tests pass
- [ ] SSH authentication tests pass
- [ ] User documentation complete
- [ ] Troubleshooting guide available
- [ ] CI/CD uses private repo successfully

### Testing
```bash
# Verify HTTPS authentication
oc apply -f examples/notebookvalidationjob-https-auth.yaml
oc get notebookvalidationjob test-https-auth -o jsonpath='{.status.phase}'

# Verify SSH authentication
oc apply -f examples/notebookvalidationjob-ssh-auth.yaml
oc get notebookvalidationjob test-ssh-auth -o jsonpath='{.status.phase}'

# Verify authentication error handling
oc apply -f examples/notebookvalidationjob-invalid-credentials.yaml
oc get notebookvalidationjob test-invalid-creds -o jsonpath='{.status.message}'
```

## References

- [Test Notebooks Repository](https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks) (private)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [GitHub SSH Keys](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)
- ADR-020: Git Authentication Support (HTTPS and SSH)
- ADR-033: End-to-End Testing Against Live OpenShift Cluster
- ADR-034: Dual Testing Strategy with Kind and OpenShift
- ADR-035: Test Tier Organization and Scope

## Notes

- Private repository is essential for authentication testing
- Users can replicate testing with their own private repositories
- Public repository planned for Phase 3 (community contributions)
- Credential rotation should be performed every 90 days
- CI/CD secrets should use GitHub Secrets for secure storage
- Consider adding authentication performance benchmarks

