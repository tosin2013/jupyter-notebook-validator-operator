# Helm Chart Sync Strategy

This document explains how the Helm chart stays in sync with the Kubernetes manifests in `config/`.

## The Challenge

The operator uses **two packaging formats**:

1. **Kustomize** (`config/` directory) - Used by Operator SDK, OLM bundles
2. **Helm** (`helm/` directory) - Used for Helm-based deployments

Both need to stay in sync, but Helm charts must be **self-contained** (can't reference files outside their directory).

## The Solution: Automated Sync

### 1. **Makefile Targets** (Manual Sync)

```bash
# Sync CRDs from config/crd/bases/ to helm/crds/
make helm-sync-crds

# Sync all resources (CRDs + observability)
make helm-sync

# Test Helm chart after sync
make helm-test
```

### 2. **Git Pre-Commit Hook** (Automatic Sync)

**Enable once:**
```bash
git config core.hooksPath .githooks
```

**What it does:**
- Detects CRD changes in `config/crd/bases/`
- Automatically runs `make helm-sync-crds`
- Adds synced files to your commit

**Example workflow:**
```bash
# 1. Edit CRD
vim config/crd/bases/mlops.mlops.dev_notebookvalidationjobs.yaml

# 2. Stage changes
git add config/crd/bases/mlops.mlops.dev_notebookvalidationjobs.yaml

# 3. Commit (hook automatically syncs to helm/crds/)
git commit -m "feat: update CRD with new field"

# 4. Hook output:
# 🔍 Checking if CRDs need to be synced to Helm chart...
# 📝 CRD files changed, syncing to Helm chart...
# ✅ CRDs synced to Helm chart and added to commit
```

### 3. **GitHub Actions** (CI/CD Validation)

The Helm validation workflow (`.github/workflows/helm-validation.yaml`) will:
- Check if CRDs are in sync
- Fail the build if they're out of sync
- Provide instructions to run `make helm-sync-crds`

## File Mapping

| Source (config/) | Destination (helm/) | Sync Method |
|------------------|---------------------|-------------|
| `config/crd/bases/*.yaml` | `helm/crds/*.yaml` | `make helm-sync-crds` |
| `config/prometheus/monitor.yaml` | `helm/templates/servicemonitor.yaml` | Manual template |
| `config/prometheus/alerting-rules.yaml` | `helm/templates/alerting-rules.yaml` | Manual template |
| `config/monitoring/openshift-console/*.yaml` | `helm/templates/dashboards/*.yaml` | Manual template |
| `config/manager/manager.yaml` | `helm/templates/deployment.yaml` | Manual template |
| `config/rbac/*.yaml` | `helm/templates/*.yaml` | Manual template |

## Why Not Symlinks?

Symlinks don't work because:
1. `helm package` doesn't follow symlinks
2. Breaks when chart is distributed
3. Not portable across operating systems

## Why Not Kustomize in Helm?

Helm and Kustomize serve different purposes:
- **Kustomize**: Overlay-based, used by Operator SDK
- **Helm**: Template-based, used for user installations

Mixing them creates complexity without benefits.

## Best Practices

### For Developers

1. **Always sync before committing:**
   ```bash
   make helm-sync
   git add helm/
   git commit -m "sync: update Helm chart"
   ```

2. **Test after syncing:**
   ```bash
   make helm-test
   ```

3. **Enable pre-commit hook:**
   ```bash
   git config core.hooksPath .githooks
   ```

### For CI/CD

1. **Validate sync in PR checks:**
   ```yaml
   - name: Check Helm sync
     run: |
       make helm-sync-crds
       git diff --exit-code helm/crds/
   ```

2. **Auto-sync in release workflow:**
   ```yaml
   - name: Sync and package Helm chart
     run: |
       make helm-sync
       make helm-package
   ```

## Troubleshooting

### CRDs out of sync

**Error:**
```
Error: CRDs in helm/crds/ are out of sync with config/crd/bases/
```

**Fix:**
```bash
make helm-sync-crds
git add helm/crds/
git commit -m "sync: update Helm CRDs"
```

### Pre-commit hook not running

**Check if enabled:**
```bash
git config core.hooksPath
# Should output: .githooks
```

**Enable:**
```bash
git config core.hooksPath .githooks
```

### Manual sync needed

If you don't want to use hooks:
```bash
# Before every commit that changes CRDs
make helm-sync-crds
git add helm/crds/
```

## Future Improvements

1. **Automated sync in GitHub Actions** - Auto-commit synced files in PRs
2. **Validation in CI** - Fail builds if sync is needed
3. **Sync observability resources** - Automate ServiceMonitor, dashboards sync
4. **Version checking** - Ensure Chart.yaml version matches operator version

