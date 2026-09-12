# Dispositions and unknowns — operational reference

Closed sets, enforced by [`engine/vocabulary.py`](../engine/vocabulary.py).
Normative semantics: [`docs/reference/dispositions.md`](../docs/reference/dispositions.md).

## Governance dispositions (12)

**Execution** — `EXECUTE` · `CONTINUE` · `STOP_COMPLETE`
**Capture / review** — `CAPTURE_ONLY` · `ROADMAP_REVIEW` · `ARCHITECTURE_REVIEW` · `MAINTENANCE_REVIEW` · `RETIREMENT_REVIEW`
**Refusal / uncertainty** — `NO_EXECUTION_AUTHORITY` · `AUTHORITY_WITHDRAWN` · `CONFLICT` · `UNKNOWN`

Exactly one per evaluation. Per-discovery dispositions nest beneath it.

## Onboarding dispositions (8) — a separate alphabet

`PROVIDER_DETECTED` · `PROVIDER_UNCONFIRMED` · `PROVIDER_CONFIGURED` · `PROVIDER_UNAVAILABLE` · `PROVIDER_CONFLICT` · `AUTHORITY_SOURCE_MISSING` · `READY_FOR_GOVERNANCE` · `PROPOSAL_READY`

These never appear in a governance decision. Different state machine, different consumer.

## Handling `UNKNOWN`

Read `unknowns[]`. Each entry carries:

```json
{ "reason": "NO_CRITERIA_DECLARED", "dimension": "acceptance",
  "blocking": false, "resolution": "Declare criteria in ..., or accept no completion bar." }
```

One reason is worth knowing by name. **`BAR_COVERS_PART`** appears when every
declared criterion passed and the bar itself says it covers only part of the
item — `CONTINUE` with nothing failing, which is otherwise the most confusing
output the engine can produce. Two issues shipped with *"cut 1 only"* and
*"DECISION HALF ONLY"* written in a comment, and both read `STOP_COMPLETE` for
the whole item, because nothing mechanical reads prose. Declare the scope as
data (`covers`, with `declared` and `uncovered`) and completion becomes
unavailable until the uncovered part is split out or the bar is extended.

**Name where the split went, and the block discharges itself.** Adding
`split_to: ["156", "165"]` makes the engine evaluate those authorities instead
of reading the sentence beside them. Every one `STOP_COMPLETE` → the item may
complete, recorded as **`BAR_COVERS_PART_DISCHARGED`**. Any one outstanding →
`CONTINUE`, *naming which*, so a reader knows where to go.

**`split_to` records where work WENT, never what it waits ON.** That distinction
is load-bearing: a bar whose remainder was *deferred* — pending evidence, or
pending another run finishing — has not split anything, and naming its blocker
would make discharge mean "the thing I was waiting for happened" rather than
"the work was done somewhere else". Two bars in this repository carry no
`split_to` for exactly that reason and say so in `split_note`; using the field
as a dependency tracker would quietly turn every blocker into a discharge.

The refusal survives: a bar still cannot complete by deleting `covers`, and an
empty `split_to` is that with extra steps — `all([])` is `True`, so it is
rejected explicitly. A cycle and an over-deep chain are both refusals rather
than silent truncation, because a resolver that stops searching and answers
*"discharged"* has declared a completion it never established (ADR-007).

Without this the signal had no off switch: the engine told you to split the
uncovered half and then could not tell that you had, so the parent read
`CONTINUE` forever and a human had to arbitrate — which is the situation
`covers` was built to replace, one level up.

- **`blocking: true`** → stop and report the `resolution`.
- **`blocking: false`** → real uncertainty that does not gate this decision. Note it; continue.

The split is *"does the current decision depend on it"*, **not** *"how serious is it"*. That distinction is what keeps `UNKNOWN` from turning every discovery into human review.

## Blocking reasons (7)

`AUTHORITY_UNSTATED` · `NOT_ADMITTED` · `NOT_ON_BOARD` · `CHECK_TIMED_OUT` · `NOT_VISIBLE_TO_STATIC_ANALYSIS` · `PROVIDER_UNREACHABLE` · `TRANSPORT_UNCONFIGURED`

Everything else is non-blocking by default. A profile may **escalate** a non-blocking reason; nothing may loosen a blocking one.

Adapters name a reason; the **engine** classifies it. A reason outside the closed set raises.
