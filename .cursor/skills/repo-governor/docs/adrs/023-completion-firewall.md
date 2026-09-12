# 23. The Completion Firewall

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Policy engine / core domain model
**Splits**: [ADR-014](014-scope-envelope-as-bounded-execution-contract.md), with [ADR-024](024-scope-envelope-compiler.md)

## Context

ADR-014 bundled two decisions: the **ScopeEnvelope** and the **completion firewall**. The ratification review found that only one of them shipped. The firewall is implemented, exercised and verified; the envelope compiler was never built. Accepting ADR-014 whole would have ratified architecture that does not exist, and holding it whole would have left a working, load-bearing mechanism unaccepted.

This ADR carries the half that earned acceptance. [ADR-024](024-scope-envelope-compiler.md) carries the half that has not.

The split is recorded rather than performed silently: ADR-014 remains in place as the record of the original bundled decision, superseded by these two. A recorded decision must survive rediscovery (INV-005), and that applies to the project's own decisions before it applies to anyone else's.

## Decision

**When acceptance conditions are satisfied, the disposition is `STOP_COMPLETE`, and no discovery converts to execution within that authorization.**

1. **The firewall is unconditional.** Discoveries that are correct, small and obviously beneficial do not survive it. Continuing requires a *separate* authorized item, which means a new evaluation against a different authority. This is the least negotiable behavior in the product, and where INV-009 either holds or the product does not work.

2. **Completion is composed by the engine, never asked of one provider.** `authority` (roadmap) + `criteria` (acceptance) + `evaluation` (repository) → disposition. No provider is asked a question it cannot answer. This is what made `STOP_COMPLETE` derivable at all when no real tracker could supply machine-checkable acceptance conditions (ADR-017).

3. **Acceptance conditions must be machine-checkable or the engine says so.** Prose no adapter can evaluate yields `acceptance_conditions_satisfied: UNKNOWN`, never a guess. Per ADR-007 that is non-blocking for `CONTINUE` and blocking for `STOP_COMPLETE`: **the engine must not declare a completion it cannot verify, and must not block ongoing work it cannot disprove.**

4. **Authority is re-read on every evaluation; envelopes carry no time bound.** An authorization is invalidated by authority change, not by expiry. A withdrawn item yields `AUTHORITY_WITHDRAWN` on the next evaluation (INV-002, INV-005). A TTL would add staleness without adding safety.

5. **Completion does not retract authority.** Finished work stays `AUTHORIZED`; completion is a separate axis. A provider whose authorization signal disappears at completion is a provider whose signal is wrong — this is why the `milestone` admission signal counts closure as authorizing (ADR-022).

## Evidence at ratification

```
issues #7 … #13 (7 closed gate conditions, live GitHub)  ->  STOP_COMPLETE
conformance/layer2.py  completed_work_authority           ->  AUTHORIZED across 3 providers
conformance/layer1.py  C7 determinism                     ->  byte-identical across runs
```

`engine/completion.py` composes it; `engine/amendments.py` guards the obvious attack on it.

## Consequences

**Positive**

- The clearest, most demonstrable behavior in the product, and the easiest thing to show a skeptic.
- Deriving completion from three roles rather than one means a tracker that cannot express acceptance conditions does not prevent governing completion.

**Negative**

- **It will be experienced as annoying.** An agent that stops with three obvious improvements identified but unexecuted feels worse than one that just does them — right up until the once it deletes something it should not have. The firewall's value is invisible when it works.
- **Criteria drift is the way to defeat it.** Acceptance criteria were amended twice during this project's own implementation. Both were justified and recorded. `engine/amendments.py` requires every amendment to carry a resolvable citation and reports loosening separately, which raises the cost of a convenient amendment without eliminating it. **It cannot judge whether a cited section actually supports the amendment** — that is a reading, and ADR-002 keeps readings out of the engine. This is an accepted, open weakness, not a solved problem.
- **The firewall rules; it does not stop.** Repo Governor returns a verdict. An agent that continues anyway leaves a decision record showing it did so against an explicit `STOP_COMPLETE`, which is the accountability property and the honest limit of what a skill can enforce.

## Related Specification Sections

§40 Completion Firewall · §44 Example Evaluation · §53 Success Metrics · INV-002 · INV-005 · INV-009
