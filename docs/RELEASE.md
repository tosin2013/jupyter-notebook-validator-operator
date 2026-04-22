# Operator Release Guide

This document is the single authoritative runbook for releasing the
Jupyter Notebook Validator Operator to OperatorHub and community operator
catalogs. Follow the numbered steps in order.

---

## Version History and Submission Status

| Version | OCP Stream | `replaces` | OperatorHub Status |
|---|---|---|---|
| v1.0.5 | 4.18+ | v1.0.4 | Submitted |
| v1.0.6 | 4.18+ | v1.0.5 | Submitted |
| v1.0.7 | 4.18 | v1.0.6 | **Pending PR — submit before v1.0.8** |
| v1.0.8 | 4.19 | v1.0.7 | **Never submitted — blocked on v1.0.7 chain** |
| v1.0.9 | 4.20 | v1.0.8 | Not yet released |
| v1.0.10 | 4.21 | v1.0.9 | Planned |

> **OCP-stream convention:** v1.0.7 → OCP 4.18 | v1.0.8 → OCP 4.19 |
> v1.0.9 → OCP 4.20 | v1.0.10 → OCP 4.21

The `replaces:` chain is enforced by OLM. A version cannot be submitted to
OperatorHub until the version it replaces has been accepted. See the
[Backlog section](#operatorhub-submission-backlog) for resolution order.

---

## Metadata Standards

All releases **MUST** use the following consistent metadata.

### Provider and Maintainer

```yaml
provider:
  name: Decision Crafters
  url: https://www.decisioncrafters.com/

maintainers:
  - name: Tosin Akinosho
    email: takinosh@redhat.com
```

### Git Author (required for DCO)

```bash
git config user.email "takinosh@redhat.com"
git config user.name "Tosin Akinosho"
```

### DCO Sign-off

All commits to community-operator repos **MUST** include a DCO sign-off:

```bash
git commit -s -m "operator jupyter-notebook-validator-operator (v1.0.X)"
```

The sign-off line must match the author email:

```
Signed-off-by: Tosin Akinosho <takinosh@redhat.com>
```

### Icon

Use the standard Decision Crafters icon from v1.0.3-ocp4.19. Do **not** use
personal GitHub avatars. The `base64data:` value begins with:

```
iVBORw0KGgoAAAANSUhEUgAAAIAAAACACAIAAABMXPacAAAABG...
```

---

## Pre-Release Checklist

Before generating the bundle or opening PRs, verify:

- [ ] `provider.name` is `Decision Crafters`
- [ ] `provider.url` is `https://www.decisioncrafters.com/`
- [ ] `maintainers[0].email` is `takinosh@redhat.com`
- [ ] `maintainers[0].name` is `Tosin Akinosho`
- [ ] `containerImage` annotation matches the image tag being released
- [ ] Icon matches the standard icon (from v1.0.3-ocp4.19)
- [ ] Git author email is `takinosh@redhat.com`
- [ ] All commits include DCO sign-off (`-s` flag)

Verification commands:

```bash
CSV=bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml

# Provider
grep -A2 "provider:" $CSV

# Maintainers
grep -A2 "maintainers:" $CSV

# containerImage annotation
grep "containerImage:" $CSV | head -1

# Icon (first 50 chars)
grep "base64data:" $CSV | sed 's/.*base64data: //' | cut -c1-50
```

---

## Release Runbook

Replace `VERSION` with the new version number (e.g. `1.0.9`) and `OCP_STREAM`
with the target OCP minor version (e.g. `4.20`) throughout.

### Step 1 — Create the release branch

```bash
VERSION=1.0.9
OCP_STREAM=4.20

git checkout main
git pull origin main
git checkout -b release-${OCP_STREAM}
git push -u origin release-${OCP_STREAM}
```

### Step 2 — Bump versions

```bash
# Makefile
sed -i "s/^VERSION ?= .*/VERSION ?= ${VERSION}/" Makefile

# Helm chart
CHART=helm/jupyter-notebook-validator-operator/Chart.yaml
sed -i "s/^version:.*/version: ${VERSION}/" $CHART
sed -i "s/^appVersion:.*/appVersion: \"${VERSION}\"/" $CHART

# Verify
grep "^VERSION" Makefile
grep "^version\|^appVersion" $CHART
```

### Step 3 — Regenerate manifests and DeepCopy code

```bash
make manifests generate
git diff --stat   # verify only expected generated files changed
```

### Step 4 — Generate the OLM bundle

```bash
make bundle

# Set the replaces chain — MUST point to the previous released version
PREV_VERSION=1.0.8
CSV=bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml

# Update replaces
sed -i "s|replaces:.*|replaces: jupyter-notebook-validator-operator.v${PREV_VERSION}|" $CSV

# Update containerImage annotation (must match the image tag you will push)
sed -i "s|containerImage:.*quay.io/takinosh/jupyter-notebook-validator-operator:.*|containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:${VERSION}|" $CSV

# Update OCP version range annotation
# Edit bundle/metadata/annotations.yaml to add:
#   com.redhat.openshift.versions: "v4.20-v4.22"
# (adjust range to match the OCP stream)
```

Verify the key CSV fields after editing:

```bash
grep -E "replaces:|containerImage:|com.redhat.openshift" $CSV
```

### Step 5 — Validate the bundle

```bash
operator-sdk bundle validate ./bundle \
  --select-optional suite=operatorframework
```

All checks must pass with no errors before proceeding.

### Step 6 — Build and push images

```bash
IMG=quay.io/takinosh/jupyter-notebook-validator-operator:${VERSION}
BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:v${VERSION}

# Operator image
make docker-build docker-push IMG=${IMG}

# Bundle image
make bundle-build bundle-push BUNDLE_IMG=${BUNDLE_IMG}

# Verify images are accessible
docker pull ${IMG}
docker pull ${BUNDLE_IMG}
```

### Step 7 — Tag the release and push

```bash
git add Makefile bundle/ helm/ config/
git commit -s -m "release: v${VERSION} for OCP ${OCP_STREAM}"
git push origin release-${OCP_STREAM}

# Tag triggers release.yml CI which builds and pushes images and creates
# the GitHub Release
git tag v${VERSION}
git push origin v${VERSION}
```

Monitor the **Release** workflow in GitHub Actions to confirm images are built
and the GitHub Release is created.

### Step 8 — Sync your community-operators fork

Repeat this for both fork repos before opening PRs.

```bash
# community-operators-prod fork
cd ~/forks/community-operators-prod
git fetch upstream
git checkout main
git rebase upstream/main
git push origin main

# community-operators fork
cd ~/forks/community-operators
git fetch upstream
git checkout main
git rebase upstream/main
git push origin main
```

### Step 9 — Submit to `redhat-openshift-ecosystem/community-operators-prod`

**One bundle per PR. Do NOT include multiple versions in a single PR.**

```bash
cd ~/forks/community-operators-prod

OPERATOR=jupyter-notebook-validator-operator

# Create a branch for this version
git checkout -b add-${OPERATOR}-${VERSION}

# Copy the bundle
mkdir -p operators/${OPERATOR}/${VERSION}
cp -r ~/jupyter-notebook-validator-operator/bundle/manifests \
       operators/${OPERATOR}/${VERSION}/
cp -r ~/jupyter-notebook-validator-operator/bundle/metadata \
       operators/${OPERATOR}/${VERSION}/

# Ensure ci.yaml exists in the operator directory
# (copy from a previous version if it exists)
ls operators/${OPERATOR}/ci.yaml || \
  cp operators/${OPERATOR}/$(ls operators/${OPERATOR}/ | head -1)/ci.yaml \
     operators/${OPERATOR}/ci.yaml 2>/dev/null || true

git add operators/${OPERATOR}/
git commit -s -m "operator ${OPERATOR} (${VERSION})"
git push origin add-${OPERATOR}-${VERSION}

# Open PR — title must match exactly
gh pr create \
  --repo redhat-openshift-ecosystem/community-operators-prod \
  --title "operator ${OPERATOR} (${VERSION})" \
  --body "Adding ${OPERATOR} version ${VERSION} which replaces v${PREV_VERSION}."
```

Wait for the PR to pass all automated checks and be reviewed/merged before
proceeding to Step 10.

### Step 10 — Submit to `k8s-operatorhub/community-operators`

Only after the `community-operators-prod` PR from Step 9 has **merged**:

```bash
cd ~/forks/community-operators

OPERATOR=jupyter-notebook-validator-operator

git checkout -b add-${OPERATOR}-${VERSION}

mkdir -p operators/${OPERATOR}/${VERSION}
cp -r ~/jupyter-notebook-validator-operator/bundle/manifests \
       operators/${OPERATOR}/${VERSION}/
cp -r ~/jupyter-notebook-validator-operator/bundle/metadata \
       operators/${OPERATOR}/${VERSION}/

ls operators/${OPERATOR}/ci.yaml || \
  cp operators/${OPERATOR}/$(ls operators/${OPERATOR}/ | head -1)/ci.yaml \
     operators/${OPERATOR}/ci.yaml 2>/dev/null || true

git add operators/${OPERATOR}/
git commit -s -m "operator ${OPERATOR} (${VERSION})"
git push origin add-${OPERATOR}-${VERSION}

gh pr create \
  --repo k8s-operatorhub/community-operators \
  --title "operator ${OPERATOR} (${VERSION})" \
  --body "Adding ${OPERATOR} version ${VERSION} which replaces v${PREV_VERSION}."
```

---

## OperatorHub Submission Backlog

As of April 2026, two versions are pending submission:

| Version | Blocked by | Action |
|---|---|---|
| v1.0.7 | Nothing — submit first | Open PR to `community-operators-prod` |
| v1.0.8 | v1.0.7 PR must merge first | Open PR after v1.0.7 merges |
| v1.0.9 | v1.0.8 PR must merge first | Do not submit until chain is clear |

The `replaces:` field in OLM enforces a linear upgrade chain. Submitting
v1.0.8 before v1.0.7 has merged will cause catalog validation errors. Work
through the backlog in order.

Related issues:
- [#22](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/22) — Resolve OperatorHub submission backlog (v1.0.7 and v1.0.8)
- [#39](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/39) — Automate OperatorHub bundle submission script

---

## Troubleshooting

### DCO Check Failed

```bash
# Rebase and add sign-off to the last commit
git rebase HEAD~1 --signoff
git push --force-with-lease origin your-branch
```

If multiple commits need sign-off:

```bash
git rebase -i HEAD~N   # mark commits as 'edit'
# For each commit:
git commit --amend --signoff --no-edit
git rebase --continue
```

### containerImage Annotation Mismatch

The `containerImage` annotation in the CSV must exactly match the pushed image tag:

```yaml
metadata:
  annotations:
    containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.9
```

Verify with:

```bash
grep "containerImage:" bundle/manifests/*.clusterserviceversion.yaml | head -1
```

### Multiple Bundles in community-operators-prod

If you see "The PR affects more than one bundle", you included more than one
version directory. Create a separate PR for each version. Only one
`operators/<name>/<version>/` directory per PR is allowed.

### Bundle Validation Errors

Run the full scorecard locally before opening the PR:

```bash
operator-sdk bundle validate ./bundle \
  --select-optional suite=operatorframework

operator-sdk scorecard bundle \
  --config tests/scorecard/config.yaml \
  --wait-time 120s
```

---

## Release Scripts

```bash
# Full pre-submission test (builds, validates, runs scorecard)
./scripts/full-pre-submission-test.sh

# Local E2E test
./scripts/local-e2e-test.sh [tier1|tier2|tier3|metrics|webhook|all|basic]

# Pre-submission validation only
./scripts/pre-submission-validate.sh
```

---

## See Also

- [CHANGELOG.md](../CHANGELOG.md) — per-version feature and fix log
- [docs/RELEASE-NOTES-v1.0.7.md](RELEASE-NOTES-v1.0.7.md) — v1.0.7 release notes
- [docs/CI_CLUSTER_SETUP.md](CI_CLUSTER_SETUP.md) — OpenShift cluster registration for E2E CI
- [#39](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/39) — OperatorHub submission automation script (in progress)
