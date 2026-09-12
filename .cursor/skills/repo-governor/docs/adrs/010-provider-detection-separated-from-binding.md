# 10. Provider Detection Strictly Separated from Provider Binding

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Onboarding

## Context

§18 makes repository onboarding "a first-class product capability" and §19 allows Repo Governor to detect candidate providers from filesystem and remote evidence — `.beads/`, `docs/adr/`, `renovate.json`, GitHub Projects metadata. INV-013 constrains the result: detection *proposes*; only accepted configuration establishes a governance role. §54 lists "silently interprets provider availability as provider authority" as a failure condition.

The temptation is strong and worth naming. Detecting `docs/adr/` and simply using it as the architecture provider would make onboarding a single command with no human step. That convenience is precisely the failure mode — it would mean a repository's governance bindings could change because someone created a directory.

## Decision

**Detection and binding are separate phases with a mandatory human step between them. Detection output is never consumed by the evaluation path.**

1. **Detection writes a proposal, not a manifest.** `onboard` produces `.repo-governor.proposed.yaml`, clearly named and never loaded by the engine. Promotion to `.repo-governor.yaml` is a human action — a rename plus review plus commit.

2. **Confidence is reported, never acted on.** Each detection carries `PROVIDER_DETECTED` (strong evidence: a valid `.beads/` database, a reachable Linear project) or `PROVIDER_UNCONFIRMED` (weak evidence: a `docs/adr/` directory holding three files that might be architecture notes). Both still require acceptance. Confidence informs the human's review; it never shortens the path.

3. **Detection is evidence-based and states its evidence.** Every proposal cites what was found and where, so the human can evaluate the inference rather than trust it:

   ```yaml
   candidate:
     role: architecture
     type: adr
     path: docs/adr
     disposition: PROVIDER_UNCONFIRMED
     evidence:
       - "docs/adr/ contains 12 files matching NNNN-*.md"
       - "9 of 12 contain a '## Status' heading"
     not_evidence:
       - "No ADR index or template found"
   ```

4. **Detection never authenticates or writes.** It reads the filesystem and public metadata. It does not use credentials to probe remote systems — a probe that succeeds because a token happens to be present is exactly the capability-implies-permission inference ADR-005 forbids. Reachability is verified after binding, during validation.

5. **No detection is a normal outcome.** §58's greenfield fixture requires that a Git-only repository produce no silent assignments, roadmap authority `manual / unresolved`, architecture `UNKNOWN`, and no forced infrastructure. Onboarding must be able to conclude "nothing to bind but Git" and treat that as success.

6. **Validation is a distinct third phase.** After binding, `validate` confirms each adapter is reachable, satisfies its declared `contract_version`, and honors declared permissions. Only then does the repository reach `READY_FOR_GOVERNANCE`. Binding a provider and reaching governance-ready are not the same event.

## Consequences

**Positive**

- INV-013 is enforced structurally: the engine has no code path that reads a detection result, so silent binding is not merely forbidden but unimplementable.
- Cited evidence makes onboarding reviewable. A human can disagree with a specific inference rather than accepting or rejecting a black box.
- Refusing to probe with credentials keeps detection safe to run on any repository, including ones the operator does not own.

**Negative**

- Onboarding is a multi-step process with a manual step in the middle, and §54 warns that configuration too complex for simple repositories is a failure condition. Mitigated by the L0/L1 case being genuinely tiny (ADR-006), but this is friction and should be measured, not assumed acceptable.
- Not probing with credentials weakens remote detection. GitHub Projects may be undetectable without an authenticated call, so it will often surface as "possible — verify manually" rather than a confident candidate. Correct, and less convenient.
- Two files with similar names (`.repo-governor.yaml` vs `.repo-governor.proposed.yaml`) invites confusion and mis-commits. Tooling should refuse to evaluate if only the proposed file exists, and say why.

## Domain Considerations

§60's complex fixture is the case this ADR must handle correctly: Linear *and* GitHub Projects both present, both roadmap-capable. Detection reports both as candidates with evidence and emits `PROVIDER_CONFLICT`; onboarding halts pending explicit selection. It must not rank them, prefer the more recently modified, or pick the one with better API coverage. Conflict resolution is ADR-013.

The trust boundary matters here too. Detection reads repository content that may be attacker-authored in a fork or pull request. A malicious `docs/adr/` could be crafted to look like a high-confidence architecture provider. Because binding requires human commit, the attack cannot complete — which is the practical payoff of the separation, not just a philosophical stance. See ADR-012.

## Implementation Plan

1. Implement detectors per role, each emitting cited evidence and a `PROVIDER_DETECTED` / `PROVIDER_UNCONFIRMED` disposition.
2. Implement proposal writing to `.repo-governor.proposed.yaml`; assert in tests that the engine never reads it.
3. Implement the three §58–60 onboarding fixtures as acceptance tests — greenfield, growing, complex.
4. Implement `validate` and the `READY_FOR_GOVERNANCE` transition.
5. Measure onboarding friction on the L0 fixture: time and number of human decisions to reach governance-ready. This is direct evidence for §55.

## Related Specification Sections

§18 Repository Onboarding · §19 Provider Detection · §20 Provider Conflict Handling · §42 Onboarding Dispositions · §54 Failure Conditions · §57 RG-SIM-ONBOARDING-v0.1 · §58–60 Onboarding Fixtures · §61 Implementation Gate · INV-013

## Domain References

- INV-013, INV-014
- [AI Agent Security Starts with Scope Control — Cloud Security Alliance](https://cloudsecurityalliance.org/blog/2026/05/12/ai-agent-security-starts-with-scope-control)
- `docs/research/2026-08-17-external-landscape.md` §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
