# ADR-030 — Backend recommendation is computed from declared capability, not from condition level

**Status**: Proposed

**Date**: 2026-08-19

## Context

[ADR-006](006-repository-condition-assessment-with-progressive-governance.md)
established progressive governance: condition level selects a profile, and the
profile determines which policy packs load and which provider **roles** are
required. Issue 79 implemented the second half — `required_roles` had been
declared in `policies/*.json` since the profiles were written and read by
nothing.

That leaves an obvious next question, raised by the maintainer: if level decides
which *roles* you need, should it also decide which *backend* fills one? A
simple repository is well served by the filesystem; a complex one — public API
surface, supported release branches, generated consumers — carries obligations
that seem to argue for a store with stronger guarantees.

The appeal is real. So is the trap.

## Decision

**A recommendation, where one is made at all, is computed from the capability an
adapter declares — never from a table mapping a condition level to a product
name. And the claim that level predicts the need is recorded as unmeasured
rather than encoded.**

1. **No product names in the engine.** A mapping like `L4 → Dolt` fails twice:
   it puts adapter knowledge in the engine, which
   [ADR-003](003-seven-provider-roles-with-normalized-contracts.md) forbids, and
   it stops working the moment somebody writes a third backend — the exact
   third-party integration the protocol exists to invite.

2. **The distinguishing property is already declared.**
   `adapters/decision-history-dolt` reports `chain_supplied_by_store: true`;
   `adapters/decision-history-file` reports `false`. That field exists because
   [ADR-019](019-append-only-decision-history-with-dolt.md) rule 3 requires a
   backend that cannot supply history natively to implement chaining itself
   **and say so**. A recommendation phrased as *"at this level you want a store
   that supplies its own chain; yours hand-rolls one"* is computable from
   `describe`, and a third backend inherits it for free.

3. **A recommendation is never a requirement.** §54 fails this product if
   "provider configuration is too complex for simple repositories". Same
   treatment as the licence indicator: report the fact, state the consequence,
   say plainly that nothing is blocked. An unmet recommendation must not change
   any disposition.

4. **No product is recommended before its necessity is argued.** The `execution`
   role is required at `GOVERNOR_FULL` and above, and `execution-file` fills it.
   Naming a specific execution tracker as the recommendation at L3 would admit
   that tracker through the back door, which is what the standing rule on the
   Stage E experiment exists to prevent. A role may be required; a product may
   not be recommended on the strength of anyone having enjoyed using it.

5. **The premise is a hypothesis and is not yet evidence.** That an L4
   repository needs a tamper-evident chain more than an L2 is plausible —
   compatibility obligations mean more consequential decisions, so evidence
   matters more — and has never been measured. This ADR records the reasoning;
   it does not license building a recommender.

## Acceptance conditions

This ADR stays `Proposed` until all four are met. None is met today.

1. **The axes are shown to be the right ones.** `chain_supplied_by_store` and
   `provenance_quality` exist because two backends happen to exist. Show they
   generalize, or find the properties that do.
2. **Level is shown to be the right predictor.** Test whether condition level,
   or the floor indicators it derives from — `public_api_surface`,
   `release_branches`, `generated_consumers` — actually correlates with
   governance value. The indicators name the obligation; the level is derived.
   The derived thing may be the worse signal.
3. **The recommendation is shown to be actionable in one step.** A person who
   reads it must be able to act on it immediately. If not, it is decoration,
   and decoration in a governance surface is worse than silence.
4. **Measured on repositories this project does not own.** The same bar
   [ADR-024](024-scope-envelope-compiler.md) was held to, and met on
   2026-08-31, for the same reason: a property observed only here is a property
   of this repository.

## Consequences

- Nothing changes today. `status.py` reports what is bound and what a profile
  requires; it recommends no backend, which is the correct behaviour while the
  premise is untested.
- If the conditions are met, the implementation is small — the properties are
  already declared and `status.py` already reads `describe`.
- If they are not met, that is a useful answer: it means backend choice is a
  judgement about a specific repository rather than a function of its
  assessed level, and the honest surface is the one that reports capability
  differences and lets a person decide.
- The risk this ADR is written to avoid is a plausible rule encoded early,
  which is harder to remove than to never add — the same failure shape as
  onboarding defaulting an admission signal.

## Related Specification Sections

§8 (scope), §23 (condition indicators), §54 (failure conditions).

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map)._
