# 8. Fixture-Based Provider Conformance Suite as the Portability Guarantee

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Testing & verification

## Context

§45–50 requires every provider implementation to pass conformance tests and enumerates minimum tests per role. §53 sets the portability target: "semantically equivalent provider states should produce equivalent governance dispositions — 100% across conformance fixtures."

That metric is the product's central falsifiable claim. If Linear-as-roadmap-authority and GitHub-Projects-as-roadmap-authority produce different dispositions from equivalent state, the provider abstraction has failed and §55's stop condition ("cross-provider semantic normalization is not reliable") fires. Conformance testing is therefore not QA — it is the experiment that validates the thesis.

§45–50 lists *what* to test but not the mechanism, and does not address the harder half: proving equivalence *across* providers rather than correctness *within* one.

## Decision

**Conformance is a two-layer fixture suite: per-role contract tests that each adapter must pass, plus cross-provider equivalence tests that assert identical dispositions from semantically equivalent state.**

### Layer 1 — Contract conformance (per adapter)

Each adapter runs against a fixture set for its role, verifying §46–50's minimum tests plus:

- **Honest capability advertisement.** Every capability claimed in `describe` is exercised; a claim the adapter cannot serve is a failure (ADR-003).
- **Typed failure.** Unreachable backend produces a typed error, never a plausible-looking empty result. Silently returning "no ADRs found" when the directory is unreadable is the specific bug this catches, and it is the one that produces confidently wrong governance.
- **Absence vs unknown.** The adapter distinguishes "no such item" from "could not determine" (ADR-007).
- **Provenance.** Every returned fact carries a citation to its source.

### Layer 2 — Cross-provider equivalence (the thesis test)

A semantic scenario is defined once, in provider-neutral terms, then instantiated in each provider:

```yaml
scenario: authority_withdrawn_but_execution_ready
neutral_state:
  work_item: { id: W-1, admitted: true, authority: WITHDRAWN }
  execution_root: { parent: W-1, status: READY }
expect:
  decision: AUTHORITY_WITHDRAWN     # per §38
instantiations:
  linear:          { issue_state: Canceled }
  github_projects: { item_status: "Won't Do", closed: true }
  manual:          { authority: withdrawn }
```

The suite asserts every instantiation yields the identical decision. Divergence is a normalization defect and is reported as such — it is data for §55, not merely a broken build.

### Two rules learned from the first Layer 2 run *(added 2026-08-17)*

**Equivalence is asserted over disposition-relevant facts, not whole payloads.** The first run flagged a divergence because two providers returned different `reason` codes — `AUTHORITY_UNSTATED` vs `NOT_ON_BOARD` — while agreeing that the outcome was unknown-and-blocking. The dispositions were identical. A provider-specific reason is *diagnostic*, and a more specific one is better, so comparing whole payloads punishes exactly the behaviour we want. Equivalence compares a defined projection: `authority`, `admitted`, `__unknown__`, `blocking`, `__error__`. Reasons are reported, never compared.

**An honestly-advertised capability gap is not a normalization failure.** GitHub Projects cannot express machine-checkable acceptance conditions, and says so (`acceptance_conditions: false`). Scoring that as divergence would conflate "the abstraction leaks" with "this backend genuinely lacks the concept". Scenarios declare which capability they need; a provider advertising it false yields `CAPABILITY_GAP`. This is what makes ADR-003's honest advertisement pay off — the gap becomes recorded, bounded information instead of a mystery failure.

### Governing rules

1. **Fixtures are recorded, not live.** Conformance runs offline against captured adapter responses. Live API tests are a separate, non-gating suite. Determinism (ADR-002) is only assertable against fixed inputs.
2. **Synthetic adapters are first-class.** §62 permits synthetic providers for several roles at MVP. A synthetic adapter must pass Layer 1 identically to a real one — otherwise it proves nothing about the contract it is standing in for.
3. **Golden decision records.** Each scenario stores the full expected `GovernanceDecision`, not just the top-level disposition. Regressions in unknowns, provenance, or nested discovery dispositions are caught too.
4. **Host conformance is a third, smaller layer.** Per ADR-001, tier-3 file loading and script execution vary across the 26+ skill hosts. A minimal smoke suite runs on Claude Code, Codex, and Cursor.

## Consequences

**Positive**

- Converts §53's portability metric from an assertion into a number that can be produced on demand.
- Provides an early, honest read on §55's stop condition. If Layer 2 cannot be made to pass for two real trackers, that is decisive information and it arrives before the implementation investment compounds.
- Gives third-party adapter authors a definition of done that does not require reading the engine.

**Negative**

- Layer 2 is expensive and the hardest thing in the project. Defining "semantically equivalent" across trackers with genuinely different state models is the normalization problem, not a test for it — the fixtures make disagreement visible but do not resolve it.
- Recorded fixtures drift from live APIs. A provider changes its status model and the suite passes while production breaks. Requires periodic re-recording, which is unglamorous recurring maintenance that solo projects skip.
- Building conformance infrastructure before a second real provider exists risks over-engineering for a portability claim not yet exercised.

## Domain Considerations

Layer 2 should be built early, with the manual/file provider plus one real provider, rather than deferred until several adapters exist. Its value is diagnostic: it tells us whether the thesis holds while changing course is still cheap. Building it late inverts the incentive — by then the sunk cost argues against believing a failing result.

The equivalence scenarios should be seeded directly from §38, §39, §40, and §44, which are already written as neutral state → expected disposition pairs.

## Implementation Plan

1. Define the fixture format and the neutral scenario schema.
2. Implement the Layer 1 harness; run it against the Git adapter first.
3. Seed Layer 2 with the four worked examples in lifecycles.md and domain-model.md plus the three onboarding fixtures from §58–60.
4. Implement the manual/file roadmap provider and one real one; run Layer 2 across both. **This is the RG-SIM-ONBOARDING-v0.1 gate evidence** (§57, §61).
5. Wire conformance into CI; publish the portability number in the README as a standing claim.

## Related Specification Sections

§45–50 Conformance · §53 Success Metrics · §55 Stop Conditions · §57 Required Final Pre-Implementation Simulation · §58–60 Onboarding Fixtures · §61 Implementation Gate · §62 Initial Implementation Boundary

## Domain References

- §53 provider portability target
- `docs/research/2026-08-17-external-landscape.md` §4

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
