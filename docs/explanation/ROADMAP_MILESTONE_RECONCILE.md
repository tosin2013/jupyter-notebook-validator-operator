# Milestone Reconciliation: v1.0.9 / v1.0.10 and the 4.20-4.22 Window

**Date**: 2026-09-16 (updated from 2026-09-12)
**Authority**: ADR-056 (Accepted), issue #45
**Supersedes**: The v1.0.7=4.18, v1.0.8=4.19, v1.0.9=4.20, v1.0.10=4.21 sequential stream model

## Background

ADR-056 certifies a rolling three-minor window: **OpenShift 4.20, 4.21, and 4.22**.
OCP 4.18 is dropped from Tier 1 (maintenance ended 2026-08-25). OCP 4.19 is best-effort.

The existing milestones encoded a one-version-per-release model that no longer matches the certified window. OCP 4.21 is the middle of the window and does not need its own release -- v1.0.9 already claims `com.redhat.openshift.versions: v4.20-v4.22`.

## Milestone state after reconciliation

| Milestone | Description | Open | Closed | Notes |
|---|---|---|---|---|
| v1.0.9 | OCP 4.20 branch release (first in the ADR-056 window) | 0 | 33 | **Complete.** Released to OperatorHub. |
| v1.0.10 | OCP 4.22 release (k8s.io v0.35.x upgrade + release-4.22 branch) | 13 | 0 | **Repurposed from "4.21 stream".** |
| v1.1.0-backlog | Next minor version / long-tail ecosystem items | 6 | 0 | Unchanged. |
| ADR alignment and roadmapping | Governance work | 2 | 6 | #44 closed, #45 closing with this update. |

## Reconciliation decisions

### v1.0.9 -- COMPLETE

v1.0.9 was released as the OCP 4.20 branch release. All 33 issues are closed:
- `release-4.20` branch created, OLM bundle generated, images pushed to Quay.io
- OperatorHub submitted to both `community-operators-prod` and `community-operators`
- E2E validated on a live ROSA HCP cluster running OCP 4.20
- Bundle annotation: `com.redhat.openshift.versions: v4.20-v4.22`

### v1.0.10 -- Repurposed as OCP 4.22 Release

**Changed from "4.21 branch release" to "OCP 4.22 release".**

OCP 4.21 is the middle of the certified 4.20-4.22 window. The v1.0.9 bundle already covers 4.21 via the `v4.20-v4.22` OLM range. A separate 4.21 release is unnecessary.

v1.0.10 will instead serve as the OCP 4.22 release vehicle, carrying:
- **k8s.io client library upgrade** from v0.32.x to v0.35.x (K8s 1.35, matching OCP 4.22)
- **controller-runtime upgrade** from v0.19.x to v0.21.x
- **ENVTEST upgrade** from 1.31 to 1.35
- **release-4.22 branch** and OLM bundle generation
- **E2E testing** on an OCP 4.22 cluster

Issues in v1.0.10:

| Issue | Category | Rationale |
|---|---|---|
| #28 | Testing | Verify ADR-026 + ADR-020: Smart pod recovery + model-aware validation. |
| #29 | Testing | Verify ADR-042: Tekton git credentials auto-sync smoke test. |
| #30 | Docs | AWS Secrets Manager and IRSA credential injection examples. |
| #31 | Docs | AWS EBS and EFS StorageClass examples for PVC provisioning. |
| #32 | UX | Add NotebookValidationJob printer columns for `kubectl get`. |
| #92 | Docs | Document OLM v1 GA status and operator's OLM Classic strategy. |
| NEW | Architecture | ADR-059: k8s.io client library upgrade to v0.35.x. |
| NEW | Implementation | feat: Upgrade k8s.io to v0.35.x, controller-runtime to v0.21.x. |
| NEW | Release | Create release-4.22 branch and generate v1.0.10 OLM bundle. |
| NEW | Testing | E2E: Verify operator on OCP 4.22 cluster. |
| NEW | Release | OperatorHub: Submit v1.0.10 bundle to both upstream repos. |

### v1.1.0-backlog

**Unchanged.** No issues are window-related. All six are feature work or ecosystem items (#33-#38).

### ADR alignment and roadmapping

| Issue | Status | What was done |
|---|---|---|
| #43 | Closed | ADR governance process documented |
| #44 | Closed | ADR-056 ratified, ADR-002 superseded, CSV annotation updated via #16 |
| #45 | Closing | This reconciliation document (updated for 4.22) |
| #46 | Closed | ADR audit: 26 blank statuses, 12 missing index rows, dialects fixed |
| #47 | Closed | Docs reorganized into Diataxis subfolders, README rewritten |
| #48 | Closed | Broken cross-links fixed after reorganization |
| #49 | Closed | Stale OCP 4.18/4.19 references updated in guide docs |
| #93 | Closed | VHS terminal demo GIFs for data scientist quick-start |

### Issue #16 disposition

**Closed on v1.0.9.** #16 created the `release-4.20` branch and generated the v1.0.9 OLM bundle. The bundle annotation `com.redhat.openshift.versions: v4.20-v4.22` covers the full certified window. OperatorHub submission was completed via #39.

### release-4.18 and release-4.19 branches

**Frozen.** No new certified claims. Branch history is preserved. No deletion or force-push.

## What this document does NOT authorize

- Moving issues between milestones (human GitHub act)
- Closing issues as NOT_PLANNED
- Changing code, Helm, or the OLM bundle
- Editing milestone titles or descriptions on GitHub

These actions require a human to perform them on GitHub after reviewing this reconciliation.
