# Architecture Ratification Review — v0.6.0

**Prepared** 2026-09-04 · **Not ratified.** The acceptance line at the end is the
maintainer's to write (§68). Everything above it is evidence, and evidence is not
a decision.

## Why this review exists

v0.5.0 shipped under a recorded departure from the release condition, and that
review said what shipping it again would mean:

> A departure recorded once is an exception; recorded every release, it is the
> condition being repealed by habit.

v0.6.0 is that second release. This document exists so the second departure is
recorded deliberately rather than inherited by silence — the failure mode the
sentence was written against.

## Method

Unchanged from v0.1.0 and v0.5.0, and reproducible:

- **Runtime dependency** — is the ADR cited by `engine/`, or by an adapter this
  repository actually binds? Unbound adapters ship but govern nothing here.
- **Its own acceptance conditions** — what does the ADR say would settle it, and
  is that true today? *This is the half a count would skip.*

```bash
# the derived set, by the record's own method
python3 conformance/skill.py | grep -i proposed

# did the ADRs themselves move since the last release?
git diff --stat v0.5.0..HEAD -- docs/adrs/031-*.md docs/adrs/033-*.md
```

## The ledger

Six ADRs are `Proposed`. Two are runtime-dependent:

| ADR | cited by | own conditions |
|---|---|---|
| **ADR-031** | `engine/manifest.py`, `engine/status.py` | 0 of 4 met |
| **ADR-033** | `engine/onboard.py` | 1 of 3 met |

**ADR-020, ADR-029, ADR-030, ADR-032** — held cleanly, no engine or
bound-adapter citation. Undisturbed by this release.

## Findings

### G1 — The set did not grow, and that is a measurement rather than luck.

v0.6.0 bound an eighth adapter (`adapters/decision-history-file`). The bound
surface the derivation reads therefore grew, and the runtime-dependent
`Proposed` set stayed at exactly {031, 033}. The guard added after #153 and #183
is doing the work it was added for: the set is recomputed from the code on every
conformance run rather than asserted in prose that goes stale.

### G2 — Neither ADR moved. Verified, not assumed.

`git diff v0.5.0..HEAD` over both files is empty. ADR-031 still says *"`Proposed`
until all four are met. None is met today."* ADR-033 still meets 1 of 3.

Nothing in v0.6.0 was aimed at either, and nothing in v0.6.0 disturbed either.

### G3 — ADR-033's condition 3 is measurably worse than at v0.5.0.

Condition 3 asks that the `get_provenance` gap be resolved *"one way or the
other — either an engine consumer reads it, or §12 drops it"*, and states the
harm: *"while seven adapters implement a method nothing calls, a reader is
entitled to assume provenance is carried when it is not."*

Measured across the two releases:

| | bound adapters | of those, implementing `get_provenance` | engine callers |
|---|---|---|---|
| v0.5.0 | 7 | 4 | **0** |
| v0.6.0 | 8 | **5** | **0** |

Binding `adapters/decision-history-file` added a fifth bound implementer of a
method no engine module calls. The condition did not merely fail to advance; the
gap it names is one wider. That is small, and it is in the wrong direction, and a
review that reported "unchanged" would have missed it. **#222** is open on
exactly this gap and is unmilestoned.

### G4 — F3 is no longer a structural prediction. It is the operating state.

The v0.5.0 review set out three readings of the conflict and declined to choose:
the condition is right and the dependency is the problem; the condition is right
and a departure is the honest exception; or the condition needs amending. It
named the cost of the third: *"a condition amended the first time it binds is a
condition that never binds."*

A year of releases is not required to see which reading is being adopted in
practice. Two consecutive releases have taken reading 2 without anyone choosing
it, which is how reading 3 arrives without an ADR — the condition stops binding
because it is departed from routinely, not because anyone amended it.

**#223** is open to settle this and is unmilestoned. It has now been open across
one full release cycle.

## What ratification requires of the release, not just of the ADRs

The release condition has two halves, and only the first is contested:

> Every architecture decision the runtime depends on is Accepted, **and no
> Proposed ADR is silently treated as normative by that release.**

The second half is met. `docs/adrs/README.md` names the set; `conformance/skill.py`
derives it from the code and fails if any document claims the runtime depends on
none of them; `docs/releases/v0.6.0.md` states the departure and quotes the
warning. Nothing here is silent.

The first half is not met, for the second consecutive release.

## What is left to the maintainer

Four things this document deliberately does not do:

1. **Choose among F3's three readings.** That is #223, and it is an ADR-shaped
   decision with its own acceptance conditions (ADR-032), not a line in a review.
2. **Ratify ADR-031 or ADR-033.** Neither meets its own terms, and accepting a
   decision whose stated conditions are unmet would make the conditions
   decorative.
3. **Admit #222 or #223.** Admission is declared, never assumed (ADR-018), and
   both were filed by the agent that found them.
4. **Accept or refuse this departure.** A second departure is defensible on the
   record — the set has not grown, the disclosure is complete, and the blocking
   evidence is not on a timeline this project controls. It is defensible *once
   more*. The line below is where that judgement goes.

---

## Maintainer's acceptance

**Ratified 2026-09-04 by Tosin Akinosho (§68).** The departure from the release
condition is accepted for v0.6.0. ADR-031 and ADR-033 remain `Proposed` and
runtime-depended-on; the set has not grown since v0.5.0, and the disclosure
obligation in the condition's second half is met.

Accepted on the understanding that **#223 settles what the condition means before
v0.7.0**. A third consecutive departure would not be a third exception — it is the
condition being repealed by habit, which is the outcome the v0.5.0 review named
and this one was written to prevent. G3 is why that boundary is stated rather
than assumed: ADR-033's condition 3 moved in the wrong direction this release, so
"unchanged" is already not quite true.

*Recorded by the agent at the maintainer's explicit instruction in session, not
authored by it. §68 makes the judgement human; the transcription is not the act.*
