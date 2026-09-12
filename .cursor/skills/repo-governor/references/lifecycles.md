# Lifecycles — operational reference

Three state machines. Normative diagrams: [`docs/reference/lifecycles.md`](../docs/reference/lifecycles.md).

## Admission — how work becomes executable

```
DISCOVERED → CAPTURED → CANDIDATE → EVALUATED → ADMITTED → AUTHORIZED → EXECUTING → EVIDENCE_READY → COMPLETE
```

**`DISCOVERED → EXECUTING` is forbidden.** This is the transition an agent takes without noticing: you find something, it is obviously worth doing, you do it. Admission is the missing step.

## Maintenance — how external change becomes work

```
EXTERNAL_SIGNAL → IMPACT_ASSESSED → NO_ACTION | WATCH → CHANGE_CANDIDATE → ADMITTED → AUTHORIZED → EXECUTING → VERIFIED
```

**A new version is not a reason to upgrade.** Impact assessment is a judgement about *this* repository; a feed cannot supply it.

## Retirement — how code gets removed

```
SUSPECTED_OBSOLETE → EVIDENCE_COLLECTED → RETIREMENT_CANDIDATE → OBLIGATION_CHECK
    → RETAIN | REVIEW | REMOVAL_READY → REMOVAL_AUTHORIZED → REMOVED → VERIFIED
```

`OBLIGATION_CHECK` examines dynamic loading, plugin registration, CLI use, configuration references, API exports, migration logic, release branches, compatibility promises, tests, architecture references, generated consumers and telemetry.

Unresolved on any dimension → `UNKNOWN` or `RETIREMENT_REVIEW`. Never removal.

## Two resolved cases worth knowing

**Roadmap cancelled, tracker says `READY`** → `AUTHORITY_WITHDRAWN`. Roadmap admission governs execution authorization. Not a `CONFLICT` — this case has a defined answer.

**Rediscovered work previously deferred** → `CAPTURE_ONLY` or `ROADMAP_REVIEW`, never `EXECUTE`, unless the recorded reversal condition is satisfied.
