# Quick Start: CI/CD Testing

This guide helps you quickly set up and test the CI/CD workflows.

## Prerequisites

- GitHub repository with workflows enabled
- OpenShift 4.18+ cluster access
- Quay.io account

## Step 1: Configure GitHub Secrets (5 minutes)

### OpenShift Authentication

```bash
# 1. Login to your OpenShift cluster
oc login --server=https://api.cluster.example.com:6443 --token=<your-token>

# 2. Create service account
oc create serviceaccount ci-test-runner -n default
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:default:ci-test-runner

# 3. Generate token (90-day expiration)
oc create token ci-test-runner -n default --duration=8760h
# Save this token - you'll need it for OPENSHIFT_TOKEN

# 4. Get server URL
oc whoami --show-server
# Save this URL - you'll need it for OPENSHIFT_SERVER
```

### Add Secrets to GitHub

1. Go to your repository on GitHub
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret** for each:

| Secret Name | Value | Example |
|-------------|-------|---------|
| `OPENSHIFT_SERVER` | Your OpenShift API URL | `https://api.cluster.example.com:6443` |
| `OPENSHIFT_TOKEN` | Token from step 3 above | `eyJhbGciOiJSUzI1NiIsImtpZCI6...` |
| `QUAY_USERNAME` | Your Quay.io username | `takinosh+ci_test_runner` |
| `QUAY_PASSWORD` | Your Quay.io password/token | `eyJ...` |

## Step 2: Test Tier 1 Workflow (Automatic)

Tier 1 tests run automatically on every PR and push.

### Create Test PR

```bash
# 1. Create a test branch
git checkout -b test-ci-workflows

# 2. Make a small change
echo "# CI/CD Test" >> README.md

# 3. Commit and push
git add README.md
git commit -m "test: Trigger CI workflows"
git push origin test-ci-workflows

# 4. Create PR on GitHub
```

### Monitor Workflow

1. Go to **Actions** tab in GitHub
2. Find "CI - Unit & Integration Tests (Tier 1)"
3. Watch the workflow execute (~10 minutes)

**Expected Results:**
- ✅ Unit tests pass
- ✅ Integration tests pass on KinD (Kubernetes v1.31.10)
- ✅ Build verification passes
- ✅ Manifest validation passes

## Step 3: Test Tier 2 Workflow (Manual Trigger)

Tier 2 tests require the `e2e-test` label or manual trigger.

### Option A: Label-Based Trigger

1. Go to your PR on GitHub
2. Add label: `e2e-test`
3. Workflow will trigger automatically

### Option B: Manual Trigger

1. Go to **Actions** tab
2. Select "E2E - OpenShift Cluster Tests (Tier 2)"
3. Click **Run workflow**
4. Select branch and test tier
5. Click **Run workflow**

### Monitor Workflow

1. Watch workflow execute (~15 minutes)
2. Check logs for each step

**Expected Results:**
- ✅ OpenShift login successful
- ✅ Operator deployed
- ✅ Test notebooks executed
- ✅ All tests pass

## Step 4: Verify Results

### Check Test Results

```bash
# View workflow summary
# Go to Actions → Select workflow run → View summary

# Download artifacts
# Go to workflow run → Scroll to Artifacts → Download
```

### Check OpenShift Cluster

```bash
# Login to cluster
oc login --token="<OPENSHIFT_TOKEN>" --server="<OPENSHIFT_SERVER>"

# Check operator deployment
oc get pods -n jupyter-notebook-validator-operator

# Check test resources (if workflow is running)
oc get notebookvalidationjobs -n e2e-tests
oc get pods -n e2e-tests
```

## Troubleshooting

### Tier 1 Fails: "KinD cluster creation failed"

**Solution**: GitHub Actions runner may have resource constraints. Retry workflow.

### Tier 2 Fails: "OPENSHIFT_SERVER or OPENSHIFT_TOKEN not configured"

**Solution**: Verify secrets are configured:
```bash
# Check in GitHub UI
Settings → Secrets and variables → Actions
```

### Tier 2 Fails: "Login failed: Unauthorized"

**Solution**: Token may be expired. Regenerate:
```bash
oc create token ci-test-runner -n default --duration=8760h
# Update OPENSHIFT_TOKEN secret in GitHub
```

### Tier 2 Fails: "denied: access forbidden"

**Solution**: Quay.io credentials incorrect. Verify:
```bash
echo "<QUAY_PASSWORD>" | docker login -u "<QUAY_USERNAME>" --password-stdin quay.io
```

## Next Steps

### 1. Enable Branch Protection

```
Settings → Branches → Add rule
- Branch name pattern: main, release-*
- Require status checks to pass:
  ✓ Tier 1 CI Status
  ✓ Unit Tests
  ✓ Integration Tests
  ✓ Build Verification
  ✓ Manifest Validation
```

### 2. Set Up Notifications

```
Settings → Notifications
- Enable email notifications for workflow failures
- Set up Slack/Discord webhooks (optional)
```

### 3. Schedule Regular E2E Tests

Add to `.github/workflows/e2e-openshift.yaml`:
```yaml
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC
```

### 4. Monitor Token Expiration

Set calendar reminders:
- **60 days**: Warning - tokens expire in 30 days
- **90 days**: Rotate all tokens

## Workflow Files

- **Tier 1**: `.github/workflows/ci-unit-tests.yaml`
- **Tier 2**: `.github/workflows/e2e-openshift.yaml`

## Documentation

- [GitHub Secrets Setup Guide](./GITHUB_SECRETS_SETUP.md)
- [Integration Testing Guide](./INTEGRATION_TESTING.md)
- [ADR-032: GitHub Actions CI Testing](./adrs/032-github-actions-ci-testing-kubernetes-version.md)
- [ADR-033: E2E Testing on OpenShift](./adrs/033-e2e-testing-live-openshift-cluster.md)

## Support

For issues:
1. Check workflow logs in GitHub Actions
2. Review troubleshooting section above
3. Check [GitHub Actions documentation](https://docs.github.com/en/actions)
4. Open issue on GitHub repository

---

**Congratulations!** 🎉 Your CI/CD workflows are now configured and tested.

Next: Proceed to Phase 6 (Observability) or prepare for 4.18 release.

