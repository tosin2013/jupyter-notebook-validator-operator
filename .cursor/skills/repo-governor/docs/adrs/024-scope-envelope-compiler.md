# 24. The ScopeEnvelope Is Compiled, Not Authored

**Status**: Accepted — ratified 2026-08-31 under RATIFICATION-v0.5.0.md. All four acceptance conditions below are met; the last, issue 2, was answered by measurement across six repositories on 2026-08-18.
**Ratification**: [RATIFICATION-v0.5.0.md](RATIFICATION-v0.5.0.md) — the link lives here and not on the Status line, because `adapters/adr` treats a `[` in a status value as an unfilled template (`PLACEHOLDER`), and a markdown link there made this project's own adapter read this project's own ratification as a template.
**Held at ratification**: 2026-08-17. Not accepted for v0.1.0 — see [RATIFICATION-v0.1.0.md](RATIFICATION-v0.1.0.md) finding F2. This describes architecture that **does not exist yet**.
**Date**: 2026-08-17
**Domain**: Policy engine / core domain model
**Splits**: [ADR-014](014-scope-envelope-as-bounded-execution-contract.md), with [ADR-023](023-completion-firewall.md)
**Tracked by**: issue 23 · **Blocks**: issue 2

## Context

ADR-014 bundled the ScopeEnvelope with the completion firewall. The firewall shipped. **This did not**, and nothing recorded the difference until the ratification review looked.

What is actually absent, verified:

```bash
grep -rn 'get_scope' engine/          # no results — the engine never asks for scope
grep -rn 'CAPTURE_ONLY' engine/*.py   # vocabulary definition only; no module emits it
ls .repo-governor/discoveries         # does not exist
```

The **inputs** exist and are unconsumed: `adapters/file-roadmap`, `adapters/linear` and `adapters/github-projects` all serve `get_scope`, `get_non_goals` and `get_parent_or_goal`. What is missing is the consumer.

Two consequences follow, and both are why this is held rather than quietly deferred:

1. **INV-001 is currently enforced by prose.** `SKILL.md` tells the operator that `CAPTURE_ONLY` is the default disposition for discovered work, and no code can produce it. [ADR-002](002-deterministic-policy-engine-separate-from-model-judgment.md) opens by arguing against exactly this — *"an agent that reasons past prose is the documented April 2026 failure"* — so the project has that failure mode live against its own first invariant. `SKILL.md` now says plainly that this one is on the operator, which is honest but is not enforcement.

2. **Issue 2 is unanswerable, not unmeasured.** It asks how thin compiled envelopes are on real roadmap items. Nothing compiles one, so there is no artifact to measure.

## Decision (proposed, not accepted)

**The ScopeEnvelope is compiled from provider state at evaluation time, never hand-authored.**

1. **Compiled, not written.** Derived from the roadmap provider (`get_scope`, `get_non_goals`, `get_acceptance_conditions`) plus applicable architecture constraints. A hand-maintained envelope becomes a second roadmap artifact and drifts — which is [ADR-022](022-repo-governor-does-not-own-roadmap-state.md)'s failure, and this project has now demonstrated that failure is structural rather than hypothetical.

2. **Non-goals are hard boundaries; in-scope is a positive grant.** An action matching a non-goal is refused regardless of any other justification. Explicit exclusion is stronger than inferred inclusion.

3. **`necessary_incidental_work` is bounded by necessity, not size.** A dependency test, not a magnitude test: *is the authorized outcome unreachable without this?* A large required migration qualifies; a three-line unrelated cleanup does not.

4. **Discoveries are captured with typed dispositions, never dropped.** Each discovery in a `STOP_COMPLETE` decision is enumerated with its own disposition and persisted through the `decision_history` provider. **This supersedes ADR-014's `.repo-governor/discoveries/` file sketch** — ADR-019 gave the role a real backend, so writing discoveries to a directory would reintroduce the file-as-state pattern ADR-022 forbids.

## Why this is not accepted

**The evidence points the wrong way and nobody has measured how wrong.**

Both real tracker adapters return `NON_GOALS_UNSTATED` and `ACCEPTANCE_UNSTATED` for **every** item — recorded in §55 as *"not 'often thin' but 'always thin'"*. If that generalizes, a compiler faithfully produces thin envelopes, and the question stops being *"how thin?"* and becomes **"is a thin envelope worth compiling at all?"** — a §55 stop-condition question rather than a measurement.

Accepting this ADR now would ratify a mechanism whose central premise is contradicted by the only evidence available. The honest sequence is: build the compiler, measure on real repositories (issue 2), then decide.

Rule 3 carries its own risk, stated in ADR-014 and unchanged: the necessity test requires judgment at the point of application, and ADR-002 removed judgment from the engine. In practice the agent describes an action, the engine rules on typed facts about it, and classification quality depends on how faithfully the agent describes what it is about to do. That is a soft spot in an otherwise deterministic chain.

## Acceptance conditions

Ratifiable when all of:

- an envelope compiler exists and the engine consumes `get_scope`;
- a discovery path can emit `CAPTURE_ONLY` through `decision_history`;
- issue 2 is answered with a measurement on repositories this project does not own;
- §40's worked example passes verbatim as an acceptance test.

## Related Specification Sections

§30 Core Domain Objects · §31 ScopeEnvelope · §32 Discovery Model · §39 Rediscovered Work · §53 Success Metrics · §54 Failure Conditions · INV-001
