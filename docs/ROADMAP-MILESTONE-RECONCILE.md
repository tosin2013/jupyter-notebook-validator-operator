# Milestone Reconciliation: v1.0.9 / v1.0.10 and the 4.20-4.22 Window

**Date**: 2026-09-12
**Authority**: ADR-056 (Accepted), issue #45
**Supersedes**: The v1.0.7=4.18, v1.0.8=4.19, v1.0.9=4.20, v1.0.10=4.21 sequential stream model

## Background

ADR-056 certifies a rolling three-minor window: **OpenShift 4.20, 4.21, and 4.22**.
OCP 4.18 is dropped from Tier 1 (maintenance ended 2026-08-25). OCP 4.19 is best-effort.

The existing milestones encode a one-version-per-release model that no longer matches the certified window.

## Current milestone state

| Milestone | Description (before) | Open | Closed |
|---|---|---|---|
| v1.0.9 | OCP 4.20 stream (Kubernetes 1.33) | 7 | 7 |
| v1.0.10 | OCP 4.21 stream (Kubernetes 1.34) | 5 | 0 |
| v1.1.0-backlog | Next minor version / long-tail ecosystem items | 6 | 0 |
| ADR alignment and roadmapping | (governance work) | 3 | 4 |

## Reconciliation decisions

### v1.0.9 (OCP 4.20 branch release)

**Keep as-is for now.** v1.0.9 is the OCP 4.20 *branch* release. Issue #16 creates `release-4.20` and generates the OLM bundle. This is a concrete deliverable that predates ADR-056.

Issues that stay on v1.0.9:

| Issue | Rationale |
|---|---|
| #16 | Release: Create release-4.20 branch and v1.0.9 OLM bundle. The 4.20 branch is the first release in the new certified window. |
| #17 | Implement ADR-038: Requirements.txt auto-detection. Needed before release. |
| #18 | Implement ADR-041: Exit code validation. Needed before release. |
| #19 | Fix ADR-050: S2I build status monitoring. Bug fix for release. |
| #26 | Verify ADR-021: OpenShift Console dashboard smoke test. Validation for release. |
| #27 | Verify ADR-022: Community observability pipeline. Validation for release. |
| #39 | Automate OperatorHub bundle submission. Release tooling. |

**Recommended description update**: "OCP 4.20 branch release (first release in the ADR-056 certified window: 4.20/4.21/4.22)"

### v1.0.10 (OCP 4.21 continuation)

**Reframe from "4.21 stream" to "4.21 branch release".** OCP 4.21 is the middle of the certified window, not a separate product. v1.0.10 carries verification and documentation work that was deferred from v1.0.9.

Issues that stay on v1.0.10:

| Issue | Rationale |
|---|---|
| #28 | Verify ADR-026 + ADR-020: Smart pod recovery + model-aware validation. Testing. |
| #29 | Verify ADR-042: Tekton git credentials auto-sync. Testing. |
| #30 | Docs: AWS Secrets Manager and IRSA credential injection examples. Documentation gap. |
| #31 | Docs: AWS EBS and EFS StorageClass examples for PVC provisioning. Documentation gap. |
| #32 | Add NotebookValidationJob printer columns. UX improvement. |

**Recommended description update**: "OCP 4.21 branch release (second release in the ADR-056 certified window)"

### v1.1.0-backlog

**Unchanged.** No issues are window-related. All six are feature work or ecosystem items.

### ADR alignment and roadmapping

Governance and documentation work completed in this milestone:

| Issue | Status | What was done |
|---|---|---|
| #43 | Closed | ADR governance process documented |
| #44 | Open | ADR-056 ratified, ADR-002 superseded, OLM work split to #16 |
| #45 | Open | This reconciliation document |
| #46 | Closed | ADR audit: 26 blank statuses, 12 missing index rows, dialects fixed |
| #47 | Open | Docs reorganized into subfolders, README rewritten, versions updated |
| #48 | Closed | Broken cross-links fixed after reorganization |
| #49 | Closed | Stale OCP 4.18/4.19 references updated in guide docs |

### Issue #16 disposition

**Keep on v1.0.9.** #16 creates the `release-4.20` branch and generates the v1.0.9 OLM bundle. It is the vehicle for the first release in the new window. The `com.redhat.openshift.versions` annotation in the bundle will reflect `v4.20-v4.22` per ADR-056. #44 has a `split_to` dependency on #16 for the OLM annotation work.

### release-4.18 branch

**Freeze.** No new certified claims. The branch history is preserved. No deletion or force-push.

## What this document does NOT authorize

- Moving issues between milestones (human GitHub act)
- Closing issues as NOT_PLANNED
- Changing code, Helm, or the OLM bundle
- Editing milestone titles or descriptions on GitHub

These actions require a human to perform them on GitHub after reviewing this reconciliation.
