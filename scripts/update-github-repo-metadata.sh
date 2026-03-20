#!/usr/bin/env bash
# Update GitHub repository metadata (description, homepage, topics, Discussions).
# Requires: gh CLI (https://cli.github.com/) and authenticated session: gh auth login
#
# Usage:
#   ./scripts/update-github-repo-metadata.sh
#   ./scripts/update-github-repo-metadata.sh owner/other-repo
#
set -euo pipefail

REPO="${1:-tosin2013/jupyter-notebook-validator-operator}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found. Install via: https://cli.github.com/" >&2
  exit 1
fi

gh auth status >/dev/null 2>&1 || {
  echo "Not logged in to GitHub. Run: gh auth login" >&2
  exit 1
}

echo "Updating ${REPO}..."

gh repo edit "${REPO}" \
  --enable-discussions \
  --description "Kubernetes operator for Jupyter notebook validation in MLOps (Papermill, golden notebooks, model validation)" \
  --homepage "https://operatorhub.io/operator/jupyter-notebook-validator-operator"

# Add topics (safe to re-run; duplicates are ignored by GitHub)
gh repo edit "${REPO}" \
  --add-topic kubernetes \
  --add-topic operator \
  --add-topic jupyter-notebook \
  --add-topic mlops \
  --add-topic papermill \
  --add-topic openshift \
  --add-topic tekton \
  --add-topic notebook-validation \
  --add-topic operator-sdk \
  --add-topic kserve

echo "Done. Verify: gh repo view ${REPO}"
