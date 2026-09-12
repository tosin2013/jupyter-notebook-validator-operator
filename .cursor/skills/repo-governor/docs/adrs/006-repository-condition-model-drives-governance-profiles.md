# 6. Repository Condition Model Drives Governance Profiles

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Policy scaling

## Context

§23–28 defines five repository conditions (L0 greenfield through L4 mature/high-assurance) mapped to five governance profiles. §55 makes the stakes explicit: the project should stop or simplify if "provider configuration is too complex for simple repositories" or if "governance causes material developer friction without corresponding safety benefit."

So the profile model is not a convenience feature. It is the mechanism that keeps the product from failing its own stop conditions. A single governance depth cannot serve both a two-file greenfield repository and a mature library with supported release branches: applied uniformly, either the greenfield case drowns in ceremony or the mature case gets governance too weak to catch a breaking removal.

§23 is clear that "complexity is not determined by LOC alone" and lists twelve indicators, but does not say how they combine into a level, who decides, or whether the level can change.

## Decision

**Repository condition is assessed once during onboarding, recorded in the manifest, and human-owned. It selects a governance profile that determines which policy packs load and which providers are required.**

1. **Assessment proposes; the human decides.** Onboarding computes a suggested level from observable indicators and explains its reasoning. The human accepts or overrides, and the accepted value is written to `condition.assessed` in the manifest. An assessed level in the manifest is never silently recomputed — same principle as INV-013, applied to condition instead of providers.

2. **Indicators are evidence, not a score.** No weighted formula. Assessment reports which of §23's indicators are present (public contracts, migrations, plugin/dynamic loading, supported release branches, multiple agents, generated code…) and maps presence patterns to a suggested level. A weighted score would be false precision and would make disagreement unarguable; a list of observed indicators can be reasoned about.

3. **Certain indicators floor the level.** Public API surface, supported release branches, or generated consumers imply compatibility obligations that cannot be governed at `GOVERNOR_LITE` depth. These raise the floor to L4 regardless of size, and the floor may not be overridden downward — only the level above the floor is discretionary. This is the one place human override is constrained, because it is the case where being wrong removes something external consumers depend on.

4. **Profiles are declarative policy packs, loaded on demand.** Each of the five profiles is a YAML file under `policies/`. The profile determines which invariants are active beyond the always-on core, which provider roles are required, which lifecycles are enforced, and which dispositions may escalate to review. This is where ADR-001's tier-3 progressive disclosure pays off directly: an L1 repository never loads `high-assurance.yaml`.

5. **Four invariants are always on, at every level, including L0.** INV-001 (discovery confers no authority), INV-009 (completed scope means stop), INV-010 (no illegal transitions), INV-012 (`UNKNOWN` is valid). These are the irreducible core and appear in `SKILL.md` tier 2. Everything else is profile-gated.

6. **Level changes are manifest commits.** Re-assessment can be run at any time and will suggest a change, but it takes effect only when committed — reviewable like any other governance change.

## Consequences

**Positive**

- Directly answers §55's friction stop condition: an L0 repository needs Git, a manifest of about ten lines, and four invariants.
- Progressive disclosure and progressive governance use one mechanism, so there is no separate machinery to maintain.
- Profiles are data. Adding an organization policy pack later (§65) requires no engine change.

**Negative**

- Five levels is a lot of surface to specify, test, and document. Each profile multiplies the conformance matrix (ADR-008), and MVP will likely ship two or three well rather than five thinly.
- Repositories genuinely straddle levels. A monorepo with one mature public package and four internal ones has no correct single level; per-package governance is deferred (ADR-004) so such repositories must over-govern.
- Human override introduces a downgrade attack: someone annoyed by L4 friction sets L1 and loses retirement obligation checks. The floor rule blunts the worst case, but the residual risk is real and should be surfaced in the decision record whenever an override is active.

## Domain Considerations

The L0 case deserves particular attention because it inverts intuition. §24 identifies the risk as "AI mistakes absence of constraints for unlimited design authority" — INV-011. Governance need is high at *both* ends of the complexity range, which is the finding from §56's Simulation 2. L0 is not "governance off"; it is a different, small set of controls: architecture budget (§29), dependency restraint, and a hard first-feature scope.

## Implementation Plan

1. Write the five profile YAML files; define the schema before the content.
2. Implement indicator detection over repository evidence — public exports, migration directories, release branches, plugin registries, generated-code markers.
3. Implement the floor rule and make an active override explicit in every decision record it affects.
4. Build the L0 and L3 profiles first: they are the two extremes and will expose the most schema problems.
5. Cross-test: run one fixture repository under all five profiles and diff the dispositions, confirming depth actually varies as designed.

## Related Specification Sections

§23–28 Repository Condition Model · §29 Greenfield Architecture Budget · §55 Stop / Simplification Conditions · §56 Current Research Evidence (Simulation 2) · INV-011

## Domain References

- §56 Simulation 2 — governance need is high where constraints are absent and where obligations are numerous
- [Agent Skills: Progressive Disclosure as a System Design Pattern](https://www.newsletter.swirlai.com/p/agent-skills-progressive-disclosure)
- `docs/research/2026-08-17-external-landscape.md` §1

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
