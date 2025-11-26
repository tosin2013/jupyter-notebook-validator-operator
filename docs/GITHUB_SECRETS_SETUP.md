# GitHub Secrets Setup Guide

This guide explains how to configure GitHub Secrets for the CI/CD pipeline (ADR-032 and ADR-033).

## Overview

The Jupyter Notebook Validator Operator uses GitHub Actions for two-tier testing:
- **Tier 1**: Unit & Integration tests on KinD (Kubernetes v1.31.10) - No secrets required
- **Tier 2**: E2E tests on live OpenShift 4.18 cluster - Requires OpenShift and registry secrets

---

## Required Secrets

| Secret Name | Purpose | Required For | Rotation Period |
|-------------|---------|--------------|-----------------|
| `OPENSHIFT_SERVER` | OpenShift API server URL | Tier 2 E2E tests | When cluster changes |
| `OPENSHIFT_TOKEN` | Service account token | Tier 2 E2E tests | 90 days |
| `QUAY_USERNAME` | Quay.io registry username | Image push | When credentials change |
| `QUAY_PASSWORD` | Quay.io registry password | Image push | 90 days |
| `TEST_REPO_TOKEN` | GitHub PAT for test notebooks | Test notebooks (if private) | 90 days |

### 1. OPENSHIFT_SERVER

**Purpose:** OpenShift API server URL for E2E testing

**Value:** `https://api.cluster-c4r4z.c4r4z.sandbox5156.opentlc.com:6443` (example)

**How to Get:**

```bash
# Login to your OpenShift cluster
oc login --server=https://api.cluster.example.com:6443 --token=<your-token>

# Get the API server URL
oc whoami --show-server
```

**How to Set:**

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `OPENSHIFT_SERVER`
5. Value: Your OpenShift API server URL
6. Click **Add secret**

**Security Notes:**
- This is not sensitive (it's a public URL)
- Stored as secret for easy configuration management
- Update when cluster changes

### 2. OPENSHIFT_TOKEN

**Purpose:** Service account token for OpenShift authentication

**Value:** Long-lived service account token (90-day expiration)

**How to Create:**

```bash
# Create service account for CI/CD
oc create serviceaccount ci-test-runner -n default

# Grant cluster-admin permissions (required for operator installation)
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:default:ci-test-runner

# Generate token with 90-day expiration (8760 hours)
oc create token ci-test-runner -n default --duration=8760h
```

**How to Set:**

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `OPENSHIFT_TOKEN`
5. Value: The token from the command above (starts with `eyJ...`)
6. Click **Add secret**

**Security Notes:**
- **CRITICAL**: This token has cluster-admin permissions
- Rotate every 90 days
- Never commit to repository
- Monitor usage in OpenShift audit logs

### 3. QUAY_USERNAME

**Purpose:** Quay.io robot account username for authentication

**Value:** `takinosh+jupyter_notebook_validator_operator` (example)

**How to Set:**

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `QUAY_USERNAME`
5. Value: `takinosh+jupyter_notebook_validator_operator`
6. Click **Add secret**

**Security Notes:**
- This is a robot account username (not sensitive, but good practice to keep as secret)
- Allows easy rotation without updating workflow files
- Keeps credentials centralized in GitHub Secrets

### 4. QUAY_PASSWORD

**Purpose:** Authenticate to Quay.io for pushing container images

**Value:** Robot account password for Quay.io

**How to Create Robot Account:**

1. Login to [Quay.io](https://quay.io)
2. Navigate to your organization/repository
3. Go to **Settings** → **Robot Accounts**
4. Click **Create Robot Account**
5. Name: `ci_test_runner`
6. Permissions: **Write** to repository
7. Save the generated token

**How to Set:**

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `QUAY_PASSWORD`
5. Value: Robot account token (starts with `eyJ...`)
6. Click **Add secret**

**Security Notes:**
- This is a robot account token, not a personal password
- Token has limited permissions (push to specific repository only)
- Token can be rotated without affecting personal account
- Never commit this token to the repository
- Rotate every 90 days

### 5. TEST_REPO_TOKEN (Optional)

**Purpose:** GitHub Personal Access Token for accessing private test notebooks repository

**Value:** GitHub PAT with `repo` scope

**When Required:**
- Only if test notebooks repository is private
- Not required for public repositories

**How to Create:**

1. Go to GitHub → **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
2. Click **Generate new token (classic)**
3. Name: `CI Test Notebooks Access`
4. Scopes: Select `repo` (Full control of private repositories)
5. Expiration: 90 days
6. Click **Generate token**
7. Save the token (starts with `ghp_...`)

**How to Set:**

1. Go to your GitHub repository
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `TEST_REPO_TOKEN`
5. Value: GitHub PAT
6. Click **Add secret**

**Security Notes:**
- Only required for private test repositories
- Use minimal scopes (only `repo`)
- Rotate every 90 days
- Never commit to repository

---

## Verification

After setting up the secret, verify it works:

### Test Locally (Optional)

```bash
# Set environment variables (for testing only, never commit these!)
export QUAY_USERNAME="takinosh+jupyter_notebook_validator_operator"
export QUAY_PASSWORD="NFRTXOQPWE2HWPESJJQRLRU89JKKRO1NMY24M6IYKVZ0ERSRRLI0XLIUFRDEYPW4"

# Test Quay.io login
echo "$QUAY_PASSWORD" | podman login -u="$QUAY_USERNAME" --password-stdin quay.io

# Pull existing image (if any)
podman pull quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Logout
podman logout quay.io

# Unset environment variables
unset QUAY_USERNAME QUAY_PASSWORD
```

### Test in GitHub Actions

1. Push a commit to the `main` branch
2. Go to **Actions** tab in GitHub
3. Watch the "Build and Push Image" workflow
4. Verify it completes successfully
5. Check Quay.io for the new image

---

## Quay.io Robot Account Details

**Username:** `takinosh+jupyter_notebook_validator_operator`  
**Repository:** `quay.io/takinosh/jupyter-notebook-validator-operator`  
**Permissions:** Write (push images)

**Robot Account Benefits:**
- Separate from personal account
- Can be revoked without affecting personal access
- Scoped to specific repository
- Audit trail for automated pushes

---

## Image Registry Information

### Public Image URL
```
quay.io/takinosh/jupyter-notebook-validator-operator
```

### Available Tags
- `latest` - Latest build from main branch
- `v0.1.0`, `v0.2.0`, etc. - Release versions
- `main-<sha>` - Commit-specific builds
- `dev-<branch>` - Development branch builds

### Pull Image
```bash
# Using Podman
podman pull quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Using Docker
docker pull quay.io/takinosh/jupyter-notebook-validator-operator:latest

# Using Kubernetes
kubectl set image deployment/controller-manager \
  manager=quay.io/takinosh/jupyter-notebook-validator-operator:latest
```

---

## CI/CD Workflow Overview

### Workflows Using Secrets

1. **`.github/workflows/build-push.yml`**
   - Triggers: Push to main, tags, manual
   - Actions: Build multi-arch image, scan, push to Quay.io
   - Uses: `QUAY_USERNAME` and `QUAY_PASSWORD` for authentication

2. **`.github/workflows/release.yml`**
   - Triggers: Git tag push (e.g., `v0.1.0`)
   - Actions: Build binaries, create release, push images
   - Uses: `QUAY_USERNAME` and `QUAY_PASSWORD` for bundle image push

### Workflow Permissions

The workflows use the following GitHub permissions:
- `contents: read` - Read repository code
- `contents: write` - Create releases (release workflow only)
- `packages: write` - Push container images

---

## Security Best Practices

### Secret Rotation

Rotate the `QUAY_PASSWORD` secret every 90 days:

1. Generate new robot account token in Quay.io
2. Update GitHub Secrets with new values:
   - `QUAY_USERNAME` (if robot account name changed)
   - `QUAY_PASSWORD` (new token)
3. Test workflows to ensure they still work
4. Revoke old token in Quay.io

### Access Control

- Only repository admins can view/edit secrets
- Secrets are not exposed in workflow logs
- Secrets are masked in GitHub Actions output
- Use separate secrets for different environments (dev/staging/prod)

### Audit

Monitor secret usage:
- Check GitHub Actions logs for authentication failures
- Review Quay.io audit logs for push activity
- Set up alerts for suspicious activity

---

## Troubleshooting

### Authentication Failed

**Error:** `Error: failed to authorize: failed to fetch anonymous token`

**Solution:**
1. Verify `QUAY_PASSWORD` secret is set correctly
2. Check robot account is not expired
3. Verify repository permissions in Quay.io
4. Test login locally with the same credentials

### Image Push Failed

**Error:** `Error: failed to push image`

**Solution:**
1. Check Quay.io repository exists
2. Verify robot account has write permissions
3. Check image size (Quay.io has limits)
4. Review Quay.io status page for outages

### Workflow Not Triggering

**Error:** Workflow doesn't run on push

**Solution:**
1. Check workflow file syntax (YAML validation)
2. Verify trigger conditions (branches, tags)
3. Check repository settings → Actions → General
4. Ensure Actions are enabled for the repository

---

## Additional Secrets (Future)

As the project grows, you may need additional secrets:

### CODECOV_TOKEN (Optional)
- Purpose: Upload test coverage to Codecov
- How to get: Sign up at codecov.io and link repository

### SLACK_WEBHOOK (Optional)
- Purpose: Send build notifications to Slack
- How to get: Create incoming webhook in Slack workspace

### SNYK_TOKEN (Optional)
- Purpose: Security scanning with Snyk
- How to get: Sign up at snyk.io and generate API token

---

## References

- [GitHub Secrets Documentation](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Quay.io Robot Accounts](https://docs.quay.io/glossary/robot-accounts.html)
- [Docker Login Action](https://github.com/docker/login-action)
- [ADR-012: Release and CI/CD Strategy](../adrs/012-release-and-cicd-strategy.md)

---

**Last Updated:** 2025-11-07  
**Maintained By:** Development Team

