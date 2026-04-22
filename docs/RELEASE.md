# Operator Release Guide

This document is the single authoritative runbook for releasing the
Jupyter Notebook Validator Operator to OperatorHub and community operator
catalogs. Follow the numbered steps in order.

---

## Version History and Submission Status

| Version | OCP Stream | Upgrade Strategy | OperatorHub Status |
|---|---|---|---|
| v1.0.5 | 4.18+ | `replaces: v1.0.4` | Submitted |
| v1.0.6 | 4.18+ | `replaces: v1.0.5` | Submitted |
| v1.0.7 | 4.18 | `replaces: v1.0.6` | Submitted (merged upstream) |
| v1.0.8 | 4.19 | `olm.skipRange: >=1.0.2 <1.0.8` | [community-operators-prod PR #9442](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/9442) / [community-operators PR #7940](https://github.com/k8s-operatorhub/community-operators/pull/7940) — pending maintainer merge |
| v1.0.9 | 4.20 | `olm.skipRange: >=1.0.2 <1.0.9` | Not yet released |
| v1.0.10 | 4.21 | `olm.skipRange: >=1.0.2 <1.0.10` | Planned |

> **OCP-stream convention:** v1.0.7 → OCP 4.18 | v1.0.8 → OCP 4.19 |
> v1.0.9 → OCP 4.20 | v1.0.10 → OCP 4.21
>
> **Upgrade strategy change at v1.0.8:** `spec.replaces` was replaced with
> `olm.skipRange` because v1.0.7 was submitted with a flat bundle structure
> (no `manifests/` subdirectory), making it invisible to the operatorcert
> `check_replaces_availability` check. See
> [Known Pipeline Pitfalls](#known-pipeline-pitfalls) for details.

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
```

After `make bundle`, manually correct the CSV and bundle metadata. **Do NOT
use `spec.replaces`** — use `olm.skipRange` instead. See
[Known Pipeline Pitfalls](#known-pipeline-pitfalls) for the reason.

```bash
CSV=bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml

# Remove spec.replaces if make bundle generated it
sed -i '/^  replaces:/d' $CSV

# Add olm.skipRange annotation to metadata.annotations
# (adjust lower bound to the first ever published version: 1.0.2)
python3 - <<'EOF'
import re, sys

path = "bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml"
version = "${VERSION}"   # replace with actual VERSION value

with open(path) as f:
    content = f.read()

skip_line = f'    olm.skipRange: ">=1.0.2 <{version}"\n'
# Insert after 'metadata:\n  annotations:\n'
content = re.sub(
    r'(metadata:\n  annotations:\n)',
    r'\1' + skip_line,
    content,
    count=1
)
with open(path, "w") as f:
    f.write(content)
print("olm.skipRange written")
EOF

# Update containerImage annotation (must match the image tag you will push)
sed -i "s|containerImage:.*quay.io/takinosh/jupyter-notebook-validator-operator:.*|containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:${VERSION}|" $CSV

# Update bundle/metadata/annotations.yaml — set channel and OCP range
# Edit manually or with sed:
sed -i "s|operators.operatorframework.io.bundle.channels.v1:.*|operators.operatorframework.io.bundle.channels.v1: stable|" \
    bundle/metadata/annotations.yaml
sed -i "s|com.redhat.openshift.versions:.*|com.redhat.openshift.versions: v${OCP_STREAM}-v4.22|" \
    bundle/metadata/annotations.yaml
```

Verify the key fields after editing:

```bash
grep -E "skipRange|containerImage:|com.redhat.openshift|channels" $CSV bundle/metadata/annotations.yaml
# spec.replaces must NOT appear:
grep "spec.replaces\|^  replaces:" $CSV && echo "ERROR: replaces still present" || echo "OK: no replaces"
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
  --body "Adding ${OPERATOR} version ${VERSION}."
```

> **OperatorHub requires exactly one commit per PR.** If you need to amend
> after pushing, squash before force-pushing — never add a second commit:
> ```bash
> git add -A
> git commit --amend -s --no-edit
> git push --force origin add-${OPERATOR}-${VERSION}
> ```
> If you already have multiple commits, squash them first:
> ```bash
> git reset --soft HEAD~N   # N = number of commits to collapse
> git commit -s -m "operator ${OPERATOR} (${VERSION})"
> git push --force origin add-${OPERATOR}-${VERSION}
> ```

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
  --body "Adding ${OPERATOR} version ${VERSION}."
```

> **OperatorHub requires exactly one commit per PR.** Apply the same
> squash procedure described in Step 9 if any amendments are needed.

---

## OperatorHub Submission Backlog

Current status as of April 2026:

| Version | community-operators-prod | community-operators |
|---|---|---|
| v1.0.7 | Merged ✓ | Merged ✓ |
| v1.0.8 | [PR #9442](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/9442) — CI passing, awaiting merge | [PR #7940](https://github.com/k8s-operatorhub/community-operators/pull/7940) — CI passing, awaiting merge |
| v1.0.9 | Not yet submitted — blocked until v1.0.8 merges | Not yet submitted |

Related issues:
- [#22](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/22) — Resolve OperatorHub submission backlog
- [#39](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/39) — Automate OperatorHub bundle submission script

---

## Known Pipeline Pitfalls

These issues were encountered during the v1.0.8 submission and are documented
here to prevent repeating them.

### Pitfall 1 — `check_replaces_availability` fails for flat-structure bundles

**Symptom:** The operatorcert static test reports `KeyError: '<version>'`
inside `check_replaces_availability`, even though the prior version exists in
the upstream repo.

**Root cause:** The `Bundle.probe()` function in the operatorcert framework
requires a `manifests/` subdirectory to consider a directory a valid bundle.
v1.0.7 was submitted to both upstream repos with manifests placed directly in
the version root (no `manifests/` subdir). As a result `all_bundles()` silently
skips v1.0.7, and any bundle that sets `spec.replaces: v1.0.7` gets a
`KeyError` during the check.

**Fix:** Use `olm.skipRange` in the CSV `metadata.annotations` instead of
`spec.replaces` in the CSV spec:

```yaml
metadata:
  annotations:
    olm.skipRange: ">=1.0.2 <1.0.8"
```

The `check_replaces_availability` function returns immediately when
`spec.replaces` is absent, so the broken lookup never happens. OLM still
honours the `skipRange` for upgrade path decisions.

**Future mitigation:** Always ensure bundle directories have the correct nested
structure (`manifests/` and `metadata/` subdirectories). Validate locally with:

```bash
operator-sdk bundle validate ./bundle
```

---

### Pitfall 2 — Channel trap: `stable,alpha` with a `stable`-only `replaces` target

**Symptom:** `check_replaces_availability` fails with `KeyError` even when
`spec.replaces` points to a version that appears to exist.

**Root cause:** The check runs once for every channel the submitted bundle is
declared in. If the bundle is in `stable,alpha` and the `replaces` target
(e.g. v1.0.7) is only in `stable`, the check fails when it processes the
`alpha` channel because v1.0.7 is not visible there.

**Fix:** Submit to `stable` only unless there is an explicit reason for
`alpha`. Set channels in `bundle/metadata/annotations.yaml`:

```yaml
operators.operatorframework.io.bundle.channels.v1: stable
operators.operatorframework.io.bundle.channel.default.v1: stable
```

Do not set `alpha` unless you are intentionally maintaining an alpha channel
with a full, unbroken upgrade chain from the oldest alpha bundle.

---

### Pitfall 3 — Transient `ppc64le` IIB build failure

**Symptom:** The `add-bundle-to-index` task in the `operator-hosted-pipeline`
fails with `IIB build failed` and `Reason: Failed to build the container image
on the arch ppc64le`. All static and certification tasks passed.

**Root cause:** Red Hat's IIB (Index Image Builder) service occasionally
encounters infrastructure failures on `ppc64le` build nodes. The failure is
unrelated to the bundle content.

**Fix:** Retry the pipeline. Because the GitHub token used for fork operations
may not have write access to comment on the upstream PR, the easiest retry
method is a no-op force-push:

```bash
cd ~/forks/community-operators-prod
git commit --amend --no-edit
git push --force origin add-jupyter-notebook-validator-operator-${VERSION}
```

This triggers a new pipeline run without changing any bundle content. If the
pipeline keeps failing on IIB for more than 2–3 retries, post a comment on the
upstream PR asking a maintainer to run `/pipeline restart operator-hosted-pipeline`.

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
