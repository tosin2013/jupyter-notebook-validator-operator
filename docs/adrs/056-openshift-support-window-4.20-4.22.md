# ADR 056: OpenShift Support Window 4.20–4.22

## Status
Accepted

Ratified 2026-09-12 by repository owner (tosin2013). Supersedes [ADR-002](002-platform-version-support-strategy.md).

## Context

ADR-002 (Accepted, 2025-11-07) certifies OpenShift **4.18, 4.19, and 4.20** as Tier 1, with Kubernetes 1.25+ as best-effort. That window no longer matches Red Hat’s OpenShift lifecycle as of 2026-09-12.

### What changed

| OpenShift | Kubernetes (release notes) | GA | Phase on 2026-09-12 |
|-----------|----------------------------|----|---------------------|
| 4.18 | 1.31 | 2025-02-25 | EUS Term 1. Maintenance ended **2026-08-25** ([RHBA-2026:29706](https://access.redhat.com/errata/RHBA-2026:29706)) |
| 4.19 | 1.32 | 2025-06-17 | Maintenance (ends ~2026-12-17). No EUS |
| 4.20 | 1.33 | 2025-10-21 | Maintenance. EUS floor |
| 4.21 | 1.34 | 2026-02-03 ([RHBA-2026:1481](https://access.redhat.com/errata/RHBA-2026:1481)) | Maintenance (Full Support ended ~90 days after 4.22 GA) |
| 4.22 | 1.35 | 2026-06-09 ([RHEA-2026:0449](https://access.redhat.com/errata/RHEA-2026:0449)) | Full Support. EUS |
| 4.23 | — | Not GA (CI/nightlies only) | Do not certify |

This repository’s own April 2026 matrix already said operator support for 4.18 would be dropped when 4.22 shipped. Both events have happened.

Catalog metadata already claims `com.redhat.openshift.versions: v4.19-v4.22` while CSV prose and ADR-002 still describe 4.18–4.20. That disagreement is evidence for a supersede, not permission to silently rewrite ADR-002.

Policy: [OpenShift Container Platform Life Cycle](https://access.redhat.com/support/policy/updates/openshift) — about a four-month minor cadence, at least four minors supported at once, even minors are EUS, maintenance ends 18 months after GA.

This operator is community-distributed. Aligning the **certified window** to Red Hat’s calendar does not inherit Red Hat Operator life-cycle classifications.

### Tree today

`go.mod` still pins `k8s.io/api v0.31.10` (4.18-era APIs). Installability on a newer cluster is not the same as a client-library bump. This ADR does **not** authorize a `k8s.io`, ENVTEST, or Operator SDK upgrade.

## Decision

We will certify a rolling **three-minor** OpenShift window whose high end is the current EUS GA:

### Tier 1 — Certified

- **OpenShift 4.20, 4.21, and 4.22**
- OLM annotation: `com.redhat.openshift.versions: v4.20-v4.22`
- Kubernetes mapping: 1.33 / 1.34 / 1.35
- Testing, bug fixes, and support claims apply only to this window once this ADR is Accepted

### Drop from Tier 1

- **OpenShift 4.18** — drop from certified support. Remaining `release-4.18` history is frozen for new certified claims.

### Best-effort (not certified)

- **OpenShift 4.19** — still in Red Hat maintenance until ~2026-12-17, no EUS. No dedicated CI obligation.
- Upstream Kubernetes: best-effort on versions that correspond to the certified OpenShift minors (1.33–1.35), not the ADR-002 “1.25+” floor.

### Out of the window

- **OpenShift 4.23** and later — not certified until GA plus an assigned, tested follow-on ADR or issue.

### What this does not decide

- Client library or Operator SDK upgrades (`k8s.io` may remain v0.31.x until a later authorized issue).
- Creating `release-4.22` or reusing issue #16 (v1.0.9 / `release-4.20` bundle) as the 4.22 vehicle.
- OperatorHub republish, CSV marketing text, or README install tags (follow-on after ratification).
- ADR-006’s historical 4.18-first phased roadmap and ADR-046’s 4.18/4.19/4.20 bundle table — named as related drift; not rewritten here.

When 4.24 (next EUS) is GA, the expected roll is **4.22 / 4.23 / 4.24** via another Proposed ADR, not an in-place edit of this file.

## Consequences

### Positive

- Claims can match Red Hat’s current Full Support / Maintenance set instead of an EUS-only 4.18 floor.
- EUS-to-EUS cluster path 4.20 → 4.22 stays in the certified window.
- OLM range can be narrowed from the current `v4.19-v4.22` annotation to `v4.20-v4.22` after ratification.

### Negative

- Users remaining on 4.18 leave certified support while their clusters may still be on EUS Term 1.
- Three OpenShift versions still need CI if we honour “certified.”
- ADR-046’s per-stream bundle table and `origin/release-4.18` become stale (discovery for a later issue).
- Catalog already advertised 4.19–4.22; dropping 4.19 from certified is a claim change, not only a 4.18 drop.

### Neutral

- Lowest-common-denominator Kubernetes APIs in the operator binary may still be 1.31 until a separate bump.
- Odd-numbered 4.21 stays in the window as the middle minor, not as its own “stream” product.

## Implementation Notes

1. Ratification: human sets this ADR to **Accepted** and ADR-002 to **Superseded by ADR-056**.
2. Then issue #45 may reconcile GitHub milestones v1.0.9 / v1.0.10 with this window (do not bulk-move issues).
3. Follow-on (not this ADR): CSV description, README, `docs/OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md`, OLM annotation `v4.20-v4.22`.

## References

- [OpenShift Container Platform Life Cycle](https://access.redhat.com/support/policy/updates/openshift)
- [RHEA-2026:0449](https://access.redhat.com/errata/RHEA-2026:0449) (OCP 4.22.0)
- [RHBA-2026:1481](https://access.redhat.com/errata/RHBA-2026:1481) (OCP 4.21.0)
- [RHBA-2026:29706](https://access.redhat.com/errata/RHBA-2026:29706) (4.18 end of Maintenance)
- GitHub issues #43 (process), #44 (this ADR), #45 (milestone reconcile)

## Related ADRs

- ADR-002: Platform Version Support Strategy (to be superseded)
- ADR-006: Version Support Roadmap and Testing Strategy (4.18-first phases; related drift)
- ADR-007: Distribution and Catalog Strategy
- ADR-029: Platform Version Dependency Review Process
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- ADR-046: Multi-Version Bundle Strategy and Upgrade Chain (4.18/4.19/4.20 table; related drift)
- ADR-055: OperatorHub Submission Lessons Learned (`olm.skipRange`)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2026-09-12 | Tosin Akinosho / agent under #44 | Proposed window; does not ratify |
