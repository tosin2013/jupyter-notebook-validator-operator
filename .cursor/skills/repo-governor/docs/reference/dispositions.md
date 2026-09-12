# Governance Dispositions

> Extracted from PRD v0.2 §41–§42 on 2026-08-17. Original section numbering preserved.
> **Normative.** ADR-007 closes this vocabulary, engine-owned: **twelve governance dispositions and eight onboarding dispositions**, which are separate alphabets and are not interchangeable. Profiles select which are *reachable*; they cannot define new ones. Providers never emit dispositions.

---

## §41 — Governance Dispositions

### Execution

```text
EXECUTE
CONTINUE
STOP_COMPLETE
```

### Capture / Review

```text
CAPTURE_ONLY
ROADMAP_REVIEW
ARCHITECTURE_REVIEW
MAINTENANCE_REVIEW
RETIREMENT_REVIEW
```

### Refusal / Uncertainty

```text
NO_EXECUTION_AUTHORITY
AUTHORITY_WITHDRAWN
CONFLICT
UNKNOWN
```

## §42 — Onboarding Dispositions

A **separate alphabet** with a different state machine and a different consumer (a human running onboarding, not an agent mid-task). These never appear in a governance decision, and the two sets never mix.

```text
PROVIDER_DETECTED
PROVIDER_UNCONFIRMED
PROVIDER_CONFIGURED
PROVIDER_UNAVAILABLE
PROVIDER_CONFLICT
AUTHORITY_SOURCE_MISSING
READY_FOR_GOVERNANCE
PROPOSAL_READY
```

> **`PROPOSAL_READY` added 2026-08-17 while implementing gate 7.** §42 had no value for *detection complete, proposal written, awaiting human acceptance*. `READY_FOR_GOVERNANCE` is the state after provider validation; `PROVIDER_CONFIGURED` is after binding. The gap sat exactly where ADR-010's mandatory human step lives, so onboarding had no honest way to report where it had stopped. This makes the onboarding alphabet eight values, and the total twenty rather than the nineteen ADR-007 states.

---

## Closed-set enforcement (gate 7)

The vocabularies live in [`engine/vocabulary.py`](../../engine/vocabulary.py) and are enforced, not merely documented:

* **Adapters name a reason; the engine classifies it.** An adapter does not decide whether its own unknown blocks — `engine/completion.py` looks the reason up and attaches `dimension` and `blocking`. A reason outside the closed set raises, because silently accepting an unclassifiable unknown would let a provider bypass the blocking rule entirely.
* **Profiles may escalate, never loosen.** `GOVERNOR_HIGH_ASSURANCE` makes `NON_GOALS_UNSTATED` and `ACCEPTANCE_UNSTATED` blocking; nothing may make a blocking reason non-blocking, or a profile could permit `EXECUTE` on evidence the engine could not resolve.
* **The sets cannot drift from the code.** [`conformance/vocabulary.py`](../../conformance/vocabulary.py) scans every adapter and engine module for emitted reason strings and fails if any is undeclared, or if any declared reason is never emitted.

## Semantics (ADR-007)

**Exactly one disposition per evaluation.** The top-level decision is singular; per-discovery dispositions nest beneath it.

**`UNKNOWN` must be actionable.** An `UNKNOWN` carrying only "unknown" is a defect. Required payload:

```yaml
decision: UNKNOWN
unknowns:
  - dimension: authority          # which of the seven questions failed
    reason: PROVIDER_UNREACHABLE  # typed, enumerated
    provider: linear
    resolution: |                 # what a human can do about it
      Verify LINEAR_API_KEY is set, or bind a manual roadmap provider.
    blocking: true                # does this prevent EXECUTE?
```

The engine distinguishes **unresolvable** (evidence genuinely absent) from **unavailable** (provider down). Both yield `UNKNOWN`; the resolution differs.

**Non-blocking unknowns do not force review.** Only unknowns on the evaluation's critical path block. This is what keeps INV-012 from colliding with the over-escalation failure condition ([criteria.md §54](criteria.md)).

**`STOP_COMPLETE` is terminal and non-overridable** (INV-009).

**Tie-break is conservative.** Where the engine cannot determine whether a disposition should be `EXECUTE` or something more conservative, it resolves conservatively. The asymmetry is real: a wrongly-blocked change costs a human a minute; a wrongly-permitted deletion can cost a release.
