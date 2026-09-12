# Architecture Ratification Review — v0.5.0

**Prepared** 2026-08-31 · **Ratified 2026-08-31 by Tosin Akinosho (§68).**

> **Decision.**
>
> **ADR-024 is `Accepted`.** Its four acceptance conditions are met, each verified
> against the file rather than asserted — see F1 below.
>
> **ADR-031 and ADR-033 remain `Proposed`, and they are runtime dependencies.**
> v0.5.0 therefore ships as a **recorded departure** from the release condition
> stated in `RATIFICATION-v0.1.0.md`. Neither can be accepted until measurements
> on repositories and hosts this project does not control are complete; holding
> the release on them indefinitely is the worse option. That is a judgement, made
> here rather than left implicit, which is the whole difference between this
> release and the four before it.
>
> **The F3 conflict stays open.** None of the three readings is adopted. A
> departure recorded once is an exception; recorded every release, it is the
> condition being repealed by habit, and the next reviewer should treat a second
> one as a signal to settle F3 rather than to repeat this line.

> **This document is the evidence half only.** An agent produced it. The
> acceptance decision belongs to the maintainer (§68) and is deliberately absent
> — v0.1.0 keeps the two in one file and separate, and this follows that.
>
> To ratify, add a line here naming who accepted what, on what date. To ship a
> departure instead, record that here in the same place, with its reason. A
> reader must be able to tell which happened without reading the git log.

## Why this review exists

`RATIFICATION-v0.1.0.md` states the release condition:

> **Every architecture decision the v0.1.0 runtime depends on is Accepted, and no
> Proposed ADR is silently treated as normative by that release.**

It has been applied **once**, at v0.1.0. Four releases — v0.2.0, v0.3.0, v0.4.0,
v0.4.1 — shipped without re-applying it, and none of their release notes mentions
ratification. During that window two held decisions acquired engine dependencies
and nobody recomputed the set, because nothing recomputed it.

`docs/adrs/README.md` asserted the condition still held. It had been false since
#153 and #183. That is now derived on every conformance run
(`conformance/skill.py`), so this review starts from a measured set rather than a
remembered one.

## Method

Unchanged from v0.1.0, and reproducible:

- **Runtime dependency** — is the ADR cited by `engine/`, or by an adapter this
  repository actually binds? Unbound adapters ship but govern nothing here.
- **Mechanism exercised** — does the shipping engine reach the code path the ADR
  decides, or does the path exist only under test?
- **Its own acceptance conditions** — what does the ADR itself say would settle
  it, and is that true today? *This is the half a count would skip.*

```bash
# the dependency map
grep -ohE 'ADR-0[0-9]{2}' engine/*.py | sort | uniq -c | sort -rn
jq -r '.providers|to_entries[]|select(.key|startswith("$")|not)|.value|
       if type=="array" then .[].adapter else .adapter end' .repo-governor.json \
  | xargs cat | grep -ohE 'ADR-0[0-9]{2}' | sort -u

# the same set, derived and asserted
python3 conformance/skill.py | grep 'list matches the code'
```

## The ledger

31 ADRs: **23 Accepted, 7 Proposed, 1 Superseded.**

Of the seven `Proposed`, four are held cleanly — **ADR-020, ADR-029, ADR-030,
ADR-032** are cited by no engine module and no bound adapter. Holding them is
consistent with the condition and nothing here disturbs them.

Three are not.

| ADR | site | mechanism exercised? |
|---|---|---|
| **024** Scope envelope compiler | `engine/envelope.py:2` | **Yes.** `envelope.py` *is* the compiler this ADR decides. Every discovery ruling runs through it. (`engine/status.py:64` also names 024, but only to explain why the ADR ledger is not scored — a reference, not a dependency.) |
| **031** `AUTHORITY_SOURCE_MISSING` obliges disclosure, not refusal | `engine/manifest.py:346` | **Yes.** The advisory/blocking split at `:353` is the rule this ADR states — an unmet `required_roles` reports and does not block. |
| **033** Repo-local providers answer about the checked-out revision | `engine/onboard.py:420` | **Yes.** Decides whether a `.beads/` store may be named with an adapter. Reached on every detection run against a repository that has one. Arrived with #216. |

## Findings that change the answer

### F1 — ADR-024 meets its own conditions. It is ratifiable.

Its four, each checked rather than asserted:

| condition | state |
|---|---|
| an envelope compiler exists and the engine consumes `get_scope` | **met** — `engine/envelope.py` |
| a discovery path can emit `CAPTURE_ONLY` through `decision_history` | **met** |
| issue 2 answered with a measurement on repositories this project does not own | **met** — closed COMPLETED 2026-08-18: *455 work items across 6 repositories*, five of them foreign |
| §40's worked example passes verbatim as an acceptance test | **met** — `conformance/envelope.py:43` runs `SECTION_40` verbatim; it is executed, not cited |

This is the ADR the v0.1.0 review deliberately held, on the reasoning that
ADR-014 was half-shipped and #2 was therefore unanswerable. **Both facts have
changed.** The compiler was built and #2 was answered with the measurement its
condition names.

### F2 — ADR-031 and ADR-033 cannot be ratified, by their own terms.

**ADR-031** says it plainly: *"`Proposed` until all four are met. **None is met
today.**"* Condition 1 is *"Arm A completes, or is explicitly restarted"* — that
is **issue 36, open**. The remaining three need disclosure observed in practice,
a deliberate re-grade of every recorded `PARTIAL`, and a measurement on a host
this project does not own.

**ADR-033** meets 1 of 3. Condition 2 wants a second branch-varying provider;
`adapters/beads` reads a fixed path, so its variance is invisible in exactly the
sense decision 1 uses to exclude `adapters/adr` and `adapters/openspec` — it does
not qualify, and manufacturing one would be writing a provider to pass a test.
Condition 3 is the `get_provenance` gap, and **#216 moved it further from met**:
eight adapters now implement a method the runtime never calls.

Accepting either would be the inference the v0.1.0 review refuses in as many
words:

```
implementation exists  →  therefore architecture must be accepted
```

That is `persistence ≠ authority` wearing a different hat, and it is the exact
inference this product exists to reject. **A no-go on ratification is a
successful outcome of this review.**

### F3 — The condition and the ADRs' own conditions are now in conflict, and it is structural.

This is the finding worth the most attention, and it is not about v0.5.0.

The release condition requires every runtime-depended-on decision to be
Accepted. ADR-031 and ADR-033 cannot be accepted until measurements on
repositories and hosts **this project does not own** are done. Those measurements
are correctly specified — they are what keeps the decisions falsifiable — and
they are not on any timeline this project controls.

So the condition, as written, is satisfiable only by never letting the runtime
depend on a decision that awaits external evidence. That was true at v0.1.0
because the held ADRs were *"neither referenced by the v0.1.0 runtime"*. **It has
never been tested against a release where they were.** It is now.

Three readings, and choosing between them is not this document's to do:

1. **The condition is right and the dependency is the problem.** The runtime
   should not reach a path decided by an unratified ADR; the remedy is to remove
   or gate the dependency.
2. **The condition is right and a departure is the honest exception.** Ship, and
   record the departure and its reason here, where the next reviewer will find it.
3. **The condition needs amending.** It was written for a release with no such
   dependencies and does not say what to do when external evidence is the
   blocker. An amendment would itself be an architecture decision, with its own
   acceptance conditions.

Reading 3 has a cost worth naming: a condition amended the first time it binds is
a condition that never binds.

## Proposed classification

Recommendation only. The decision is the maintainer's (§68).

### Ratify before v0.5.0 (1)

- **ADR-024** — Scope envelope compiler. All four of its stated conditions are
  met, verified above. Its dependency is exercised on every discovery ruling.

### Remain `Proposed`, with the reason recorded (2)

- **ADR-031** — blocked on issue 36 and three external observations. Nothing in
  this release moves it.
- **ADR-033** — 1 of 3. Condition 3 is the only one reachable by code, and it is
  a real decision: either an engine consumer reads `get_provenance`, or `§12`
  drops it from the contract. Condition 2 is not something to manufacture.

Both are runtime dependencies. **Shipping v0.5.0 with them held is a departure
from the release condition**, and the point of saying so here is that it be a
decision rather than an oversight — which is what the last four releases had.

### Held cleanly, undisturbed (4)

**ADR-020, ADR-029, ADR-030, ADR-032** — no engine or bound-adapter citation.
ADR-029's own status line records it as *ready for ratification*; that is
available at any time and is not gated by this review.

## What ratification requires of the release, not just of the ADRs

`tools/check-version.py v0.5.0` fails three ways today:

```
[FAIL] engine/version.py (the engine's own version) is 0.5.0
[FAIL] README.md (the version the README tells people to clone) pins 0.5.0
[FAIL] docs/installation.md (the version the installation guide tells people to clone) pins 0.5.0
```

Its own message is the reason to care: *a stale pin is worse than none, because
it looks deliberate.* These are mechanical and must be fixed before the tag
whichever way the ratification goes.

Conformance is **19/19** at `d82bf7c`. Fourteen merges since v0.4.1.

## What is left to the maintainer

1. **ADR-024** — accept, or say why not.
2. **ADR-031 and ADR-033** — ratify nothing, and decide between the three
   readings in F3. Whichever is chosen, record it in this file.
3. **The acceptance line** at the top of this document.

Publishing a release remains a separate, public act, and is the maintainer's.
