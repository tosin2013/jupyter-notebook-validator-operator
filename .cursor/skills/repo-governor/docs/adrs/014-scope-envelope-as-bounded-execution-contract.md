# 14. ScopeEnvelope as a Bounded Execution Contract with a Completion Firewall

**Status**: Superseded
**Split** 2026-08-17 into [ADR-023 — The Completion Firewall](023-completion-firewall.md) (**Accepted**) and [ADR-024 — The ScopeEnvelope Is Compiled, Not Authored](024-scope-envelope-compiler.md) (**Proposed**).

> **Why this was split.** The [ratification review](RATIFICATION-v0.1.0.md) found that this ADR bundled two decisions and only one of them shipped. The completion firewall is implemented and verified; the envelope compiler was never built and the engine never calls `get_scope`. Accepting it whole would have ratified architecture that does not exist; holding it whole would have left a working, load-bearing mechanism unaccepted.
>
> This document is kept rather than rewritten. It is the record of the original bundled decision, and a recorded decision must survive rediscovery (INV-005) — which applies to this project's own decisions before it applies to anyone else's. Read ADR-023 and ADR-024 for what is currently in force.
**Date**: 2026-08-17
**Domain**: Policy engine / core domain model

## Context

§31 defines `ScopeEnvelope` as the answer to "what may the coding agent do while satisfying this specific authorization?" §40 defines the completion firewall: when acceptance conditions are met, the disposition is `STOP_COMPLETE` and discovered work becomes `CAPTURE_ONLY`. INV-009 makes stopping mandatory.

This is the most user-visible part of the product and the one most likely to be either useless or intolerable. Too loose and it permits the scope creep the product exists to prevent; too tight and every incidental necessary change — a helper function, a test fixture, an import — triggers review, which is §54's "blocks routine reversible implementation excessively" failure condition. §31 already anticipates this with `necessary_incidental_work`, but does not define how the boundary is drawn or who draws it.

OpenKedge (arXiv 2604.08601) supplies the pattern: approved intents "are compiled into execution contracts that strictly bound permitted actions, resource scope, and time." Compilation is the key idea — the envelope is derived from authority rather than authored separately, so it cannot drift from what was actually authorized.

## Decision

**The ScopeEnvelope is compiled from provider state at evaluation time, never hand-authored, and it is a bounded contract whose satisfaction terminates the authorization.**

1. **Compiled, not written.** The envelope is derived from the `RoadmapAuthorityProvider` (`get_scope`, `get_non_goals`, `get_acceptance_conditions`) plus applicable architecture constraints. A hand-maintained envelope would become a second roadmap artifact and drift — precisely the canonical-database failure in §54.

2. **Non-goals are hard boundaries; in-scope is a positive grant.** An action matching a non-goal is refused regardless of any other justification. An action outside in-scope and not incidental requires review. The asymmetry is intentional: explicit exclusion should be stronger than inferred inclusion.

3. **`necessary_incidental_work` is bounded by necessity, not by size.** The test is a dependency test, not a magnitude test: *is the authorized outcome unreachable without this?* A large but strictly required migration qualifies. A three-line unrelated cleanup does not. This is INV-001 applied inside execution rather than at admission, and it is the rule that keeps the envelope from making ordinary implementation painful.

4. **Envelopes carry no time bound at v1.** OpenKedge bounds contracts by time as well as scope; here, an envelope is invalidated by authority change, not by expiry. Every evaluation re-reads authority (INV-002, INV-005), so a withdrawn item yields `AUTHORITY_WITHDRAWN` on the next evaluation. Adding a TTL would create staleness without adding safety.

5. **The completion firewall is unconditional.** When acceptance conditions are satisfied, `STOP_COMPLETE` is returned and no discovery converts to execution within that authorization — including discoveries that are correct, small, and obviously beneficial. Continuing requires a *separate* authorized item, which means a new evaluation against a different authority. This is the single least negotiable behavior in the product, and it is where INV-009 either holds or the product does not work.

6. **Acceptance conditions must be machine-checkable or the envelope says so.** If `get_acceptance_conditions` returns prose no adapter can evaluate, the envelope records `acceptance_conditions_satisfied: UNKNOWN` rather than guessing. Per ADR-007 this is a non-blocking unknown for `CONTINUE` and a blocking one for `STOP_COMPLETE` — the engine must not declare completion it cannot verify, and must not block ongoing work it cannot disprove.

7. **Discoveries are captured with typed dispositions, never dropped.** §53 targets 100% discovery preservation. Each discovery in a `STOP_COMPLETE` decision is enumerated with its own disposition, persisted per ADR-009, and available to the next admission cycle.

## Consequences

**Positive**

- Compilation keeps the envelope truthful: it cannot describe more scope than the roadmap provider actually granted.
- The necessity test gives a principled answer to the friction problem — routine implementation work is permitted because it is required, not because it is small.
- The completion firewall is the product's clearest, most demonstrable behavior, and the easiest thing to show a skeptic.
- Deriving from providers means richer trackers automatically produce richer envelopes with no schema change here.

**Negative**

- Most roadmap items in the wild have no explicit non-goals and vague acceptance criteria. Compiled envelopes will frequently be thin, and a thin envelope governs weakly. Repo Governor will be most useful to teams whose trackers are already disciplined — which is arguably the wrong population, since undisciplined teams need it more. This is a genuine adoption risk and should be measured, not assumed away.
- The necessity test requires judgment at the point of application, and ADR-002 removed judgment from the engine. In practice the agent proposes an action, the engine rules on typed facts about it, and the classification quality depends on how faithfully the agent describes what it is about to do. That is a soft spot in an otherwise deterministic chain and should be stated honestly.
- The completion firewall will be experienced as annoying. An agent that stops with three obvious improvements identified but unexecuted feels worse than one that just does them — right up until the one time it deletes something it should not have. The firewall's value is invisible when it works.
- No time bound means a long-running session holds a stale envelope between evaluations. Mitigated by re-evaluating on material actions rather than caching per session.

## Domain Considerations

The `STOP_COMPLETE` behavior is the direct product analogue of the "separate verdict from action" guardrail principle. Repo Governor rules that the authorization is exhausted; it does not stop the agent. If the agent continues anyway, the decision record (ADR-009) shows that it did so against an explicit `STOP_COMPLETE` — which is the accountability property, and the honest limit of what a skill can enforce.

§40's worked example — authorized persistent state recovery completed, with Admin UI, multi-team support, and a refactor discovered — should be implemented verbatim as an acceptance test. It is the single clearest expression of what the product does.

## Implementation Plan

1. Define the `ScopeEnvelope` schema per §31 and the compiler that derives it from provider state.
2. Implement non-goal matching and the necessity classifier for incidental work.
3. Implement the completion firewall with the acceptance-condition checkability distinction from rule 6.
4. Implement §40's example as an acceptance test, and §44's as an end-to-end fixture.
5. Implement discovery capture with typed dispositions writing to `.repo-governor/discoveries/` (ADR-009).
6. Measure envelope thinness across real repositories — what fraction of roadmap items yield an envelope with explicit non-goals and checkable acceptance conditions? This is direct evidence for §55.

## Related Specification Sections

§30 Core Domain Objects · §31 ScopeEnvelope · §32 Discovery Model · §33 Feature Admission Lifecycle · §39 Rediscovered Work · §40 Completion Firewall · §44 Example Evaluation · §53 Success Metrics · §54 Failure Conditions · INV-001, INV-002, INV-009

## Domain References

- [OpenKedge: Governing Agentic Mutation with Execution-Bound Safety and Evidence Chains — arXiv:2604.08601](https://arxiv.org/abs/2604.08601)
- [Balancing speed and safety: A control framework for AI coding agents — AWS](https://aws.amazon.com/blogs/security/balancing-speed-and-safety-a-control-framework-for-ai-coding-agents/)
- [AI Agent Scope Creep: Expand the Mandate or Push Back?](https://accelate.ai/blog/ai-agent-scope-creep-mandate-expansion)
- `docs/research/2026-08-17-external-landscape.md` §3, §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
