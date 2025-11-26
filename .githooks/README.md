# Git Hooks

This directory contains Git hooks to automate repetitive tasks.

## Installation

To enable these hooks, run:

```bash
git config core.hooksPath .githooks
```

## Available Hooks

### `pre-commit`

Automatically syncs CRDs from `config/crd/bases/` to `helm/crds/` when CRD files are modified.

**What it does:**
1. Detects if any CRD files in `config/crd/bases/` have changed
2. Runs `make helm-sync-crds` to sync them to the Helm chart
3. Adds the synced files to your commit

**Why this is needed:**
- Helm charts are self-contained and need their own copy of CRDs
- Keeps `config/` and `helm/` in sync automatically
- Prevents forgetting to sync CRDs before committing

## Manual Sync

If you prefer not to use hooks, you can manually sync with:

```bash
make helm-sync-crds
```

## Disabling Hooks

To disable hooks temporarily:

```bash
git commit --no-verify
```

To disable permanently:

```bash
git config --unset core.hooksPath
```

