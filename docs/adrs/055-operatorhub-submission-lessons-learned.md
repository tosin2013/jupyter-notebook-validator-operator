# ADR-055: OperatorHub Submission Lessons Learned (v1.0.8)

**Status**: Accepted
**Date**: 2026-04-22
**Deciders**: Tosin Akinosho
**Context**: OperatorHub submission for v1.0.8 to
`redhat-openshift-ecosystem/community-operators-prod` (PR #9442) and
`k8s-operatorhub/community-operators` (PR #7940)

---

## Context and Problem Statement

The v1.0.8 submission to both OperatorHub repositories encountered a series of
pipeline failures that required multiple rounds of diagnosis and correction.
This ADR documents the root causes, decisions made, and the updated conventions
that govern all future submissions.

Three distinct classes of failure were encountered:

1. `check_replaces_availability` — persistent `KeyError` even after v1.0.7 was
   confirmed to exist upstream.
2. Channel trap — adding the bundle to both `stable` and `alpha` caused a
   secondary `check_replaces_availability` failure in the `alpha` channel.
3. Transient `ppc64le` IIB failure — intermittent Red Hat infrastructure issue.

Additionally, the `release.yml` workflow contained a `Generate OLM bundle`
step that regenerated `bundle/` on every tag push, silently overwriting the
manually corrected `replaces`, `skips`, and `com.redhat.openshift.versions`
values committed to the repository.

---

## Decision 1 — Use `olm.skipRange` instead of `spec.replaces`

### Findings

The operatorcert static test framework uses `Bundle.probe()` to determine
whether a directory in the operator tree is a valid bundle. `Bundle.probe()`
returns `False` if the directory does not contain a `manifests/` subdirectory.

v1.0.7 was submitted to the upstream repos with manifests placed directly in
the version root (no `manifests/` subdir). When the pipeline runs
`check_replaces_availability` for a bundle whose `spec.replaces` points to
v1.0.7, the framework calls `all_bundles()`, which skips v1.0.7 silently.
The resulting `ver_to_dir` dict does not contain `'1.0.7'`, causing a
`KeyError`.

The `check_replaces_availability` function returns immediately when
`spec.replaces` is absent:

```python
replaces = bundle.csv.get("spec", {}).get("replaces")
if not replaces:
    return
```

### Decision

Remove `spec.replaces` from the CSV and replace it with an
`olm.skipRange` annotation in `metadata.annotations`:

```yaml
metadata:
  annotations:
    olm.skipRange: ">=1.0.2 <1.0.8"
```

OLM honours `skipRange` for upgrade path decisions. The lower bound `1.0.2`
is the first version ever published to OperatorHub for this operator.

### Consequences

- `check_replaces_availability` passes for all future versions regardless of
  whether the predecessor was submitted with a flat or nested bundle structure.
- The `catalog/catalog.yaml` in-repo FBC has been updated to use `skipRange`
  entries in the `olm.channel` block for v1.0.8 onwards.
- `docs/RELEASE.md` Step 4 now uses `olm.skipRange` and explicitly prohibits
  `spec.replaces`.

---

## Decision 2 — Submit to `stable` channel only

### Findings

The v1.0.8 bundle was initially submitted with
`operators.operatorframework.io.bundle.channels.v1: stable,alpha` in an
attempt to fix a pre-existing dangling bundle (`v1.0.2` in `alpha` had no
successor). While `skips: [v1.0.2]` resolved the dangling-bundle check, it
introduced a new failure: `check_replaces_availability` runs once per channel,
and because `spec.replaces` pointed to v1.0.7 (stable-only), the check failed
when processing the `alpha` channel.

### Decision

Submit all future bundles to the `stable` channel only:

```yaml
operators.operatorframework.io.bundle.channels.v1: stable
operators.operatorframework.io.bundle.channel.default.v1: stable
```

The pre-existing dangling `v1.0.2` in `alpha` is a pre-submission artifact
that the pipeline does not flag when the PR does not touch the `alpha` channel.

### Consequences

- No new alpha channel entries will be created unless a deliberate alpha
  testing track is established with a complete, unbroken upgrade chain from
  `v1.0.2`.
- The `stable,alpha` guidance has been removed from `docs/RELEASE.md`.

---

## Decision 3 — Remove bundle regeneration from `release.yml`

### Findings

The `publish-olm-bundle` job in `.github/workflows/release.yml` contained a
`Generate OLM bundle` step that ran `make bundle` on every tag push. This step
regenerated `bundle/` from scratch, overwriting:

- `spec.replaces` corrections
- `olm.skipRange` annotations
- `com.redhat.openshift.versions` in `bundle/metadata/annotations.yaml`
- `channel` settings

The committed `bundle/` directory in the repository is the source of truth.
The workflow must only build and push the bundle image from the committed
content.

Additionally, the `IMG` variable passed to `make bundle` used the `v`-prefixed
tag (`v1.0.8`), causing `containerImage: v1.0.8` in the generated CSV —
inconsistent with the `1.0.8` (no prefix) used in the actual pushed image.

### Decision

Remove the `Generate OLM bundle` and `Validate bundle` steps from the
`publish-olm-bundle` job. The job now only runs:

```yaml
- name: Build and push bundle image
  run: |
    docker login -u ${{ secrets.QUAY_USERNAME }} -p ${{ secrets.QUAY_PASSWORD }} quay.io
    make bundle-build bundle-push \
      BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:v${{ steps.version.outputs.version }}
```

A `workflow_dispatch` trigger with a `version` input was also added to allow
manual bundle republishing without creating a new tag.

### Consequences

- All bundle metadata corrections must be made directly in the `bundle/`
  directory and committed before tagging.
- `make bundle` is only used locally during the release preparation step (Step
  4 of `docs/RELEASE.md`), never in CI.
- The `workflow_dispatch` mechanism provides a recovery path if the bundle
  image push fails (e.g. due to a registry outage).

---

## Decision 4 — OperatorHub PRs must be single-commit

### Findings

Both fork PR branches had 2 commits after the initial submission fix. The
`operator-hosted-pipeline` requires exactly one commit per PR. Having multiple
commits does not cause a hard CI failure but is flagged by reviewers and
violates the submission guidelines.

### Decision

All OperatorHub fork PR branches must contain exactly one DCO-signed commit
with the message format:

```
operator jupyter-notebook-validator-operator (VERSION)

Signed-off-by: Name <email>
```

If corrections are needed after the initial push, always amend and
force-push rather than adding a new commit:

```bash
git add -A
git commit --amend -s --no-edit
git push --force origin <branch>
```

### Consequences

- `docs/RELEASE.md` Steps 9 and 10 now include explicit squash instructions.
- The `--amend` workflow is documented as the standard correction procedure.

---

## Summary of All Changes Made for v1.0.8

| File | Change |
|---|---|
| `bundle/manifests/...clusterserviceversion.yaml` | Removed `spec.replaces`, added `olm.skipRange: ">=1.0.2 <1.0.8"` |
| `bundle/metadata/annotations.yaml` | Channel changed from `stable,alpha` to `stable` only; removed `skips: v1.0.2` |
| `.github/workflows/release.yml` | Removed `Generate OLM bundle` and `Validate bundle` steps; added `workflow_dispatch` |
| `catalog/catalog.yaml` | v1.0.8 channel entry changed from `replaces` to `skipRange`; v1.0.7 bundle image tag normalised to `v1.0.7` |
| `docs/RELEASE.md` | Version history updated; Step 4 rewritten for `olm.skipRange`; single-commit guidance added; Known Pipeline Pitfalls section added |

---

## References

- [operatorcert `core.py` — `Bundle.probe()`](https://github.com/redhat-openshift-ecosystem/operator-pipelines/blob/main/operatorcert/operator_repo/core.py)
- [OLM Update Graph documentation](https://olm.operatorframework.io/docs/concepts/olm-architecture/operator-catalog/creating-an-update-graph/)
- [community-operators-prod PR #9442](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/9442)
- [community-operators PR #7940](https://github.com/k8s-operatorhub/community-operators/pull/7940)
- ADR-047: Fix Bundle Versioning for Consecutive Upgrade Chain
- ADR-048: Upgrade from Published v1.0.3-ocp4.19
