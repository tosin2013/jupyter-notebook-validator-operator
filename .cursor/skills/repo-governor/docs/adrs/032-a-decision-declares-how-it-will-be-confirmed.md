# ADR-032 — A decision declares how it will be confirmed; the engine does not police it

**Status**: Proposed

**Date**: 2026-08-24

**Resolves**: [issue 141](https://github.com/tosin2013/repo-governor/issues/141)

## Context

An ADR here can be `Accepted`, its stated obligations never built, and every check in the
repository stays green. Measured on this ledger on 2026-08-24
([research](../research/2026-08-24-architecture-decisions-and-release-readiness.md)):

| | |
|---|---|
| Accepted ADRs with a `## Consequences` section | 23 of 23 |
| ADRs with anything `Confirmation`-shaped | **0** |
| post-acceptance obligations across them | **267** |
| obligations carrying any discharge marker | **1** |

ADR-011 is the worked example. `## Decision` item 6 (no network in the engine) and
`## Implementation Plan` item 4 (a stdlib allowlist check) are both enforced by
`conformance/imports.py`. `## Decision` item 5 — *static typing via annotations checked in
CI* — has never been enforced, and the engine carries no annotations for a checker to
read. One decision, three obligations, two mechanically enforced and one invisible, with
nothing able to tell them apart.

The two that were discharged were discharged because a person remembered. That is a
mechanism, and it leaves no evidence either way.

Three external constraints bound any response, and all three were established before this
decision was written:

1. **The concept already exists.** MADR carries an optional `Confirmation` section for
   exactly this — *"it is not enough to decide; design and implementation actions are
   required to bring the AD to life."* It needs adopting, not inventing.
2. **Only structural obligations are mechanically confirmable.** Architecture testing
   enforces form, never substance. *"Publish the portability number as a standing claim"*
   is checkable for presence and not for truth.
3. **An Accepted ADR is immutable.** Discharge state therefore cannot live in the ADR
   body — which forecloses the obvious design and explains why this repository's single
   hand-rolled discharge marker, a strikethrough inside ADR-011, should not have been
   written there.

## Decision

**A decision declares how it will be confirmed. Nothing in the engine measures whether it
was.**

1. **ADRs from 032 onward carry a `## Confirmation` section**, stating for each obligation
   how anyone would know it had been discharged — naming a suite, a file, a command, or
   saying plainly that confirmation is a human judgement with no artefact.

2. **Confirmation is a claim about checkability, not a promise of enforcement.** An ADR may
   honestly say *"no mechanical confirmation is possible for this."* That is a better
   record than silence and is the expected answer for substantive obligations.

3. **Obligations are classified where they are written**: *structural* (a check can read
   it), *procedural* (a pipeline step either exists or does not), *substantive* (a claim
   about the world). A confirmation that does not say which kind it is will not survive
   review.

4. **Discharge state is not recorded in the ADR.** Immutability forbids it. If discharge
   ever needs recording, it lives outside and joins on the ADR id — the shape
   `.repo-governor/acceptance/<id>.json` already uses.

5. **The engine gains nothing.** `architecture` acquires no capability, `get_constraints`
   keeps its two states, no disposition consults decision debt, and no release check reads
   the ledger. A release continues to measure the tag, the versions, conformance, and the
   artifact.

## Consequences

1. **The cheap half of the problem is addressed and the expensive half is refused.** The
   root cause is that decisions do not say how they would be confirmed. Fixing that costs a
   heading. Building a mechanism to police 267 existing obligations costs a provider
   capability, a domain relation between decisions and releases that does not exist, and a
   maintenance surface — on evidence that amounts to one undischarged item.

2. **Existing ADRs are not retrofitted.** Immutability forbids editing them, and a
   retrofit would be 267 judgements made in bulk by whoever happened to be holding the
   pen. ADR-011 §5 stays undischarged and now stays *visibly* undischarged, in the research
   note rather than in the ADR.

3. **This buys visibility, not enforcement, and may therefore change nothing.** An author
   who would not have discharged an obligation may write a Confirmation section saying so
   and still not discharge it. That is a real risk and is the first acceptance condition.

4. **§54 is respected.** *"Blocks routine reversible implementation excessively"* is a
   standing failure condition. A mechanism firing on 267 items would meet it. A heading
   does not.

5. **Issue 46 stays deferred and issue 134 stays separate.** Nothing here gates a merge or
   a release, because that needs a pull-request-to-authority signal ADR-018 forbids
   assuming. Nothing here reports across a milestone either; that is 134's territory and
   this decision does not claim it.

## Confirmation

*This ADR is the first to carry this section, which is the point.*

| obligation | kind | how anyone would know |
|---|---|---|
| Decision 1 — ADRs from 032 onward carry `## Confirmation` | structural | a `conformance/skill.py` assertion over `docs/adrs/[0-9]*.md` with number ≥ 032. **Not yet written** — see acceptance condition 2 |
| Decision 4 — discharge state is not in the ADR body | structural | the same suite: no ADR contains a strikethrough or *done* marker on an obligation. ADR-011 violates this today and is grandfathered by immutability |
| Decision 5 — the engine gains nothing | structural | `git diff` touches no file under `engine/` or `adapters/`; `adapters/adr::get_constraints` still returns exactly `DEFINED` / `INFERRED` / `UNKNOWN` |
| Decision 2 and 3 — confirmations are honest and classified | **substantive** | **No mechanical confirmation is possible.** A check can see that a section exists and cannot see whether its content is true. Human review at ADR time, and nothing else |

That last row is the decision's own warning applied to itself: two of its five clauses
cannot be confirmed by anything but a reader.

## Acceptance conditions

`Proposed` until all four are met. **One is met today.**

1. **A decision written under this rule is shown to change what happens.** At least one
   ADR after 032 states a confirmation and that confirmation is *run* — not merely
   written. If authors write the section and never act on it, this bought a heading and
   the decision should be rejected rather than quietly kept.

2. **The structural half is enforced mechanically.** `conformance/skill.py` asserts that
   ADRs numbered ≥ 032 carry `## Confirmation`. Until that exists, this ADR asks for a
   convention and relies on memory — which is the failure it was written about. *(Not
   met.)*

3. **Measured on repositories this project does not own.** Whether obligations are stated
   consistently enough elsewhere for the concept to transfer at all. `adapters/adr` reads
   92% of *statuses* across 439 real ADRs; nothing suggests obligations are stated more
   consistently than statuses. Same bar as ADR-024 and ADR-030, precedent in
   [issue 39](https://github.com/tosin2013/repo-governor/issues/39). *(Not met.)*

4. **The refusal in Decision 5 survives contact with a real want.** If a release ever
   needs to consult decision debt, this ADR is wrong and should be superseded rather than
   amended. Recording the refusal is what makes that visible when it happens. *(Met — the
   refusal is recorded here, and issues 46 and 134 remain the places it would be
   revisited.)*

## Related Specification Sections

- **§8** — Repo Governor is not a static-analysis engine. A decision-debt checker is close
  enough to that line to need saying.
- **§54** — failure conditions; *blocks routine reversible implementation excessively*.
- **§65** — release governance is a Future Candidate Capability, and *"none are authorized
  merely by inclusion here."*
- **ADR-003** — provider roles and contracts; untouched by this decision, deliberately.
- **ADR-008** — conformance suites; where acceptance condition 2 would land.
- **ADR-018** — the admission signal is declared, never assumed; why no release gate.
