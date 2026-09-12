# 22. Repo Governor Does Not Own Roadmap State

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Product boundary / provider abstraction
**Amends**: §54 Failure Conditions
**Resolves**: issue 19
**Depends on**: [ADR-021](021-every-provider-resolved-through-the-manifest.md)

## Context

§54's oldest failure condition is the first one on the list:

> Repo Governor fails if **it becomes another canonical roadmap database.**

It happened. Not as a risk, as an accomplished fact, inside the tool built to prevent it.

`.repo-governor/roadmap.json` was created as a fixture for `adapters/file-roadmap`. It was then bound as this repository's `roadmap_authority`, because the file provider was the only roadmap provider that existed when the gate conditions were written. Real gate items were added to it. The engine read it, and the engine is what produced gate verdicts — so the fixture quietly became the roadmap of record.

By the time it was noticed, **three surfaces disagreed**:

| Surface | GATE-1 … GATE-7 |
|---|---|
| `.repo-governor/roadmap.json` | all seven `IN_PROGRESS` |
| GitHub issues #7–#13 | all seven `CLOSED`, milestone closed at 10/10 |
| `engine/completion.py GATE-1` | `STOP_COMPLETE` |

The file had not been updated after the first gate closed. Every gate closure required updating two places and one of them stopped being updated, which is the failure mode shadow systems always have.

### Why the fixture could become the roadmap

Two causes, both structural rather than careless:

1. **A fixture id and a work id were the same string.** `GATE-6` named both a conformance scenario and a real gate condition. Nothing distinguished them, so nothing could warn.
2. **The engine hardcoded the file adapter** (ADR-021), so rebinding to GitHub was not a manifest edit — it was a code change nobody was going to make casually.

### Why it took this long to see

`github-projects` originally assumed Projects-v2 membership meant admission. This repository has a Project, so the binding *would* have worked — but ADR-018 showed that assumption is one convention among several, and until admission became declarable there was no honest way to bind GitHub at all.

## Decision

**Repo Governor does not own roadmap state. A file-backed roadmap provider is a conformance fixture and is never a repository's roadmap of record.**

1. **Roadmap authority binds to a real tracker.** GitHub, Linear, Jira — a system where the work is actually managed by the people managing it. This repository binds `adapters/github-projects` with `admission.signal = "milestone"`.

2. **Synthetic providers live under `conformance/fixtures/`, never under `.repo-governor/`.** `.repo-governor/` holds only real governance state: the decision store, and acceptance artifacts for real work items. A file provider bound to a fixture makes the fixture the repository's state, which is how this happened.

3. **Fixture ids name the state they demonstrate, never a work item.** `GATE-6` became `AUTHORIZED-1`. An id shared between a fixture and real work is the mechanism by which one becomes the other.

4. **A role with no real provider stays unbound.** `execution` and `change_signals` were bound to file adapters reading synthetic fixtures; they are now unbound, because this repository has no execution tracker and no dependency-signal source. **An unbound role yields a typed disposition, which is the honest answer. A fixture-backed one yields a confident wrong answer**, which is worse than no answer at all.

5. **§54 gains the rule this produced**: *Repo Governor fails if a provider fixture is ever bound as a repository's provider of record.*

## Consequences

**Positive**

- One roadmap. Gate state now has exactly one answer, and it is the one a human reading GitHub would give.
- The rebind required **no engine change** — it was a manifest edit, made possible by ADR-021. That is the first evidence that provider interchangeability is real rather than described.
- Acceptance criteria for the seven gates were re-keyed from `GATE-N` to their GitHub issue numbers and kept, amendment history intact. A recorded decision must survive rediscovery (INV-005); deleting them because their authority id changed would have been the same failure in a different costume.

**Negative**

- **Governance now requires network access and a GitHub token for this repository.** The file provider was offline. §54 forbids requiring a *specific* tracker as a product rule, and this is a per-repository binding choice rather than a product requirement — but the friction is real and lands on exactly the CI and headless cases ADR-016 rule 6 worries about.
- Under the `milestone` signal, an issue with no milestone reads `NOT_ADMITTED` and blocks. Verified: #6 and #14 read `NOT_ADMITTED`, which is correct — they are explicit deferrals — but it means unmilestoned work is ungovernable rather than merely untracked.
- **The milestone signal needed an authorization rule that survives completion.** Assignee alone made authority evaporate when work finished: closed issues typically have no assignee, so finished gates read `ADMITTED` — "not cleared to execute" — about work that was executed. Closure is now part of the milestone signal's own definition of authorization, stated in the adapter rather than assumed. This is not a fallback across signals, which ADR-018 forbids.

**What this does not fix**

`execution.json` and `signals.json` were fixtures bound as providers for the same reason the roadmap was. Unbinding them is correct but leaves this repository with no execution or change-signal evidence at all, so those governance axes are now honestly empty rather than dishonestly full.

## Domain Considerations

The failure is worth keeping visible rather than tidying away. A governance tool reproducing the exact anti-pattern it exists to prevent is the strongest available evidence that the anti-pattern is structural and not a discipline problem — it happened here despite the failure condition being written down, first on the list, by the same person who then walked into it. §54's conditions describe forces, not mistakes.

## Related Specification Sections

§8 Product Scope · §9 Product Architecture · §54 Failure Conditions · INV-005 · INV-013
