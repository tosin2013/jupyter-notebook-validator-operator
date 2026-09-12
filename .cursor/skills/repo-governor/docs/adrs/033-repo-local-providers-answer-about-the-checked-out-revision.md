# ADR-033 — A repo-local provider answers about the checked-out revision

**Status**: Proposed

**Date**: 2026-08-29

**Resolves**: [issue 173](https://github.com/tosin2013/repo-governor/issues/173)

**Bounds**: [ADR-027](027-the-governed-repository-is-not-the-install-directory.md) — which repository is governed; this ADR is about *which revision of it*

## Context

`adapters/speckit` reads `specs/<feature>/`, and Spec Kit populates that directory **per
feature branch**. So one repository, checked out twice, can yield two different
architectures. The adapter reports the fact in `get_provenance` and does nothing about it,
which is honest and is not a decision. Issue 173 asks whether the fact is acceptable and
offers three readings: the per-branch answer is correct; it is a provenance problem; the
provider should refuse off the default branch.

### What was measured

The engine calls exactly **three** methods on the `architecture` role, in two modules:

| call site | method |
|---|---|
| `engine/envelope.py:123`, `engine/status.py:84` | `get_active_decisions` |
| `engine/envelope.py:124`, `engine/status.py:89` | `get_superseded_decisions` |
| `engine/envelope.py:166`, `engine/status.py:79` | `get_constraints` |

For `adapters/speckit`, none of those three is branch-varying:

- `get_active_decisions` and `get_superseded_decisions` are **always empty by declaration**
  — `_no_ledger`, because Spec Kit supplies no decision ledger. That refusal is one of the
  three issue 156 declared.
- `get_constraints` reads `.specify/memory/constitution.md`, a fixed path.

The two methods that *are* branch-varying — `get_specs` and `get_provenance`, both reading
`specs/` — are called by **nothing outside `conformance/layer1.py` and
`conformance/bindings.py`**.

**So no engine consumer reads anything branch-varying today.** Two consequences follow, and
they pull in opposite directions: the problem is not live, and the honest caveat the adapter
emits in `get_provenance` reaches no consumer either, because the engine never calls that
method on any provider.

> **The first analysis of this was wrong and the error is worth keeping.** It reasoned that
> because `envelope.py` consults the `architecture` role and folds the result into the
> evidence chain, two branches must produce different facts under identical provenance. The
> premise is true; the conclusion does not follow, because it never asked *which methods*
> are branch-varying. That is this repository's most-repeated defect — a conclusion the
> method could not have produced — and it was caught by listing the call sites rather than
> by reasoning harder.

## Decision

**A repo-local provider answers about the checked-out revision. That is a property of the
class, and the engine may not consult a method whose answer varies with the checkout until
the recorded provenance names the revision.**

1. **The property is general, not a Spec Kit defect.** Every file-backed provider answers
   about the working tree it is pointed at. `adapters/adr` and `adapters/openspec` read
   paths that do not move between branches, so the variance is invisible; Spec Kit keys
   documents to feature branches *by design*, so it is the first provider where the property
   is visible. It belongs in `references/providers.md` as a property of repo-local
   providers, not as a Spec Kit note.

2. **The per-branch answer is correct and is not refused.** Reading 3 is rejected. A feature
   branch genuinely has constraints the default branch does not, and answering `UNKNOWN` off
   the default branch would discard true information — and would be the engine deciding
   which revision is "the" architecture, a judgement ADR-027 gives it no grounds to make.

3. **No engine consumer may read a branch-varying provider method unless the recorded
   provenance names the revision.** Today this is satisfied vacuously: no such consumer
   exists. It is stated as an obligation anyway, because vacuous satisfaction is exactly the
   condition that changes silently.

4. **ADR-027's revision gap is bounded, not closed.** ADR-027:68 already stated it — *"Good
   enough to tell two different projects apart, not good enough to tell two checkouts apart.
   Stated rather than solved."* This ADR does not supply revision identity. It prevents the
   gap being crossed without anyone noticing.

5. **`adapters/speckit` is unchanged.** It reports the fact correctly, and issue 173 puts
   changing it out of scope.

## Consequences

**Positive**

- The question is answered without new vocabulary, a provider capability, or an engine
  change. What it adds is one conformance assertion.
- The obligation in decision 3 is *structural* rather than aspirational: adding a fourth
  architecture method to the engine turns a suite red and puts this ADR back in front of
  whoever added it.

**Negative**

- **`get_provenance` remains dead weight.** Seven adapters implement it, `§12` specifies it,
  and no engine module calls it — so the branch Spec Kit carefully reports reaches no
  decision. This ADR does not fix that; it records that the qualification channel exists and
  is not connected, which is a worse state than it looks because the adapter's caveat reads
  like a mitigation.
- Decision 3 constrains a future author, and the tripwire cannot say what the *right* answer
  is when it fires — only that the question is now live.

**What this does not fix**

Nothing here lets a decision be reproduced against the revision it was recorded on
(ADR-009). That needs revision identity, which ADR-027 declined and this ADR does not
supply.

## Confirmation

| obligation | kind | how anyone would know |
|---|---|---|
| Decision 3 — no engine consumer reads a branch-varying method | **structural** | `conformance/bindings.py` scans `engine/*.py` for `("architecture", "<method>")` and fails if the set is not exactly `get_constraints`, `get_active_decisions`, `get_superseded_decisions`. **Written with this ADR and running.** |
| Decision 5 — the adapter is unchanged | **structural** | `git diff` touches no file under `adapters/` |
| Decision 1 — the property is stated as a class property | **procedural** | `references/providers.md` says it, or it does not |
| Decision 2 — the per-branch answer is the right call | **substantive** | **No mechanical confirmation is possible.** Whether refusing would have been better is a judgement about what governance owes a reader, and only a human reviewing the decision can make it |
| Decision 4 — the gap is bounded rather than closed | **substantive** | No check can show that a gap was *deliberately* left open rather than missed. The record is the confirmation |

## Acceptance conditions

`Proposed` until all three are met. **One is met today.**

1. **The tripwire has fired at least once, or been shown to fire.** A guard nobody has seen
   go red is a guard nobody has tested. Demonstrated by mutation when written; *(met)*.

2. **A second branch-varying provider exists.** The class claim in decision 1 rests on one
   observed instance. If no second provider ever exhibits it, this generalised from a sample
   of one and the honest revision is to scope it back to Spec Kit. *(Not met.)*

3. **The `get_provenance` gap is resolved one way or the other** — either an engine consumer
   reads it, or `§12` drops it. While seven adapters implement a method nothing calls, a
   reader is entitled to assume provenance is carried when it is not. *(Not met.)*

## Related Specification Sections

§10–§17 provider model and the eight roles · §12 `ArchitectureEvidenceProvider` ·
ADR-003 · ADR-009 · ADR-013 · ADR-027 · ADR-032
