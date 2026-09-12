# 17. Completion Evidence from a Repo-Local Acceptance Artifact

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Domain model / completion
**Resolves**: [#15](https://github.com/tosin2013/repo-governor/issues/15) · **Unblocks**: [#13](https://github.com/tosin2013/repo-governor/issues/13) (gate 7)

## Context

§40 defines the completion firewall: when acceptance conditions are satisfied, the disposition is `STOP_COMPLETE`. INV-009 makes stopping mandatory, and it is one of the four always-on invariants — active at every profile including L0. ADR-014 makes the firewall unconditional.

Layer 2 conformance then produced an awkward result. Neither `github-projects` nor `linear` can supply machine-checkable acceptance conditions; both return `ACCEPTANCE_UNSTATED` for every item and honestly advertise `acceptance_conditions: false`. Only `file-roadmap` — the synthetic baseline — can supply them.

```
[completion_verifiable]
    file-roadmap     {"acceptance_conditions": [{"check": "tests_pass", "satisfied": true}],
                      "machine_checkable": true}
    github-projects  {"__unknown__": true, "reason": "ACCEPTANCE_UNSTATED"}
    linear           {"__unknown__": true, "reason": "ACCEPTANCE_UNSTATED"}
```

Per ADR-007 the engine must not declare completion it cannot verify. Correct — and it means that against every tracker anyone actually uses, the product's clearest demonstrable behaviour degrades to `UNKNOWN`.

### The question was aimed at the wrong source

#15 asked where to obtain acceptance conditions *from the tracker*, and answered "nowhere". Research (`docs/research/2026-08-17-transport-capability-completion.md` §4) says trackers were never the right place to look:

> Completion bars are becoming first-class, versioned artifacts — acceptance-criteria files that live in the repo next to CI config, reused across sessions and harnesses the way test suites already are.

Standard practice already separates two things this project has been conflating. **Acceptance criteria** are per-work-item conditions. **Definition of done** is the team-wide quality bar. Trackers carry the first badly and the second not at all. CI and the repository carry both well, and have for years.

## Decision

**Acceptance criteria are a declared repo-local artifact. Completion is verified against repository evidence. `STOP_COMPLETE` is composed by the engine from three sources, never read from one.**

| Question | Source | Role |
| --- | --- | --- |
| Is this work authorized? | tracker | `RoadmapAuthorityProvider` |
| What counts as done? | `.repo-governor/acceptance/<authority-id>.json` | `AcceptanceCriteriaProvider` |
| Is it actually done? | tests, CI status, merged PR, files present | `RepositoryEvidenceProvider` |
| Therefore: stop? | — | engine composes |

1. **A new provider role, `AcceptanceCriteriaProvider`.** Not a function on the architecture provider. Architecture evidence answers *how work must be built*; acceptance criteria answer *how you know it is finished*. Collapsing them would let an ADR's presence imply a completion bar, which is INV-004 in a different costume.

2. **Criteria are declared, never inferred.** The engine reads criteria that a human wrote and committed. It does not derive them from commit messages, PR titles, issue prose, or heuristics over repository state. Inferring a completion bar would manufacture completion authority exactly as §37 forbids manufacturing architectural authority — and it is the same failure mode ADR-012 rejects for provider prose.

3. **Criteria key to the authority ID.** `.repo-governor/acceptance/ENG-142.json` references the tracker's identifier. This references tracker state without duplicating it, so §54's roadmap-database prohibition is respected — the file says nothing about whether ENG-142 is authorized, only what finishing it means.

4. **Each criterion names a check the repository provider can evaluate.**

   ```json
   {
     "$comment": "Acceptance criteria for ENG-142. Declared, not inferred (ADR-017).",
     "authority_id": "ENG-142",
     "criteria": [
       { "check": "tests_pass",   "target": "conformance/layer1.py" },
       { "check": "file_exists",  "target": "adapters/beads" },
       { "check": "command_exit", "target": "python3 conformance/layer2.py" }
     ]
   }
   ```

   The `check` vocabulary is closed, like the disposition and error vocabularies. An unrecognised check is `MALFORMED_SOURCE`, never assumed satisfied.

5. **Absence degrades, it does not fail.** No acceptance artifact for an authority item yields a non-blocking `UNKNOWN` and no `STOP_COMPLETE`. Work proceeds under `CONTINUE`; the firewall simply never fires. This is honest, and it matches how architecture `UNKNOWN` already behaves.

6. **A tracker that *can* supply acceptance conditions still may.** `file-roadmap` does. Where both a tracker and a repo artifact supply criteria for the same item, disagreement is `CONFLICT` — not a precedence rule (ADR-013 rule 3).

7. **Satisfaction is evaluated by the repository provider, not by the criteria provider.** The acceptance provider says what must be true; the repository provider says whether it is. Keeping "the bar" and "the measurement" in separate roles is the same separation that keeps roadmap authority away from execution state.

## Consequences

**Positive**

- `STOP_COMPLETE` becomes derivable on real repositories using real trackers, which it currently is not. That restores the product's most demonstrable behaviour.
- Nothing is asked of trackers that trackers cannot do. GitHub Projects and Linear stay useful for authority — the thing they *are* good at — and the completion gap stops being their fault.
- Criteria live in version control, so they are reviewed, diffed, and attributable like tests. That is materially better than a tracker custom field, which has no review workflow.
- Rides an existing industry direction rather than inventing a mechanism.
- Fixes the root cause of #2 as well: envelope thinness on the acceptance axis was the same problem viewed from a different angle.

**Negative**

- **A new artifact for humans to maintain**, and §54 names configuration too complex for simple repositories as a failure condition. Mitigated by rule 5 — a repository that declares nothing simply never gets `STOP_COMPLETE` — but every repository that *wants* the firewall now has per-work-item files to write. That is real adoption cost and the strongest argument against.
- **Criteria drift.** A file written when work started may not describe what finishing came to mean, and nothing detects that. Stale criteria could fire the firewall early, which is worse than not firing at all. **Observed within hours of implementation:** GATE-6's criteria required `file_exists adapters/beads`, but §62 permits *Beads or a synthetic execution provider*. The criterion was mis-specified, the engine correctly returned `CONTINUE`, and amending it was legitimate — but amending criteria to make work pass is precisely how a completion firewall is defeated. The amendment is recorded in the artifact with its justification. **Partly closed 2026-08-17** by [`engine/amendments.py`](../../engine/amendments.py) and enforcement in the adapter:

* An amendment must carry a **resolvable citation** — `§NN`, `ADR-NNN`, `INV-NNN`, a repository path, or a commit sha. Free prose is unfalsifiable; `§62 permits a synthetic provider` can be checked against §62.
* The `effect` is declared from a closed set, and **loosening amendments** (`removed`, `weakened`) are always reported separately. Adding criteria needs no defence; removing them does.
* Amendment history is append-only and **counted**, so repeated amendment of the same gate is visible even when each one is individually defensible — the pattern is the signal.
* The adapter **refuses to serve criteria** whose amendment lacks a citation, returning `MALFORMED_SOURCE`, so the engine yields `UNKNOWN` rather than a bar that may have been edited to fit.

What this still does not do, and cannot: judge whether the cited section actually *supports* the amendment. That is a reading, and ADR-002 keeps readings out of the engine. The claim is made checkable and the pattern visible; a human still decides whether the argument is good. Both existing amendments were converted to the structured form and audited.
- **An eighth provider role**, against a PRD that already worried seven was a lot for an MVP. This one is small and file-backed, but it widens the surface.
- **The check vocabulary is a slow-growing dependency.** `tests_pass` and `file_exists` are easy; anything real will want conditional and compound checks, and that path ends at a small expression language nobody planned.
- Rule 6's `CONFLICT` on disagreement is correct and will be annoying, since a tracker field and a repo file drifting apart is a likely everyday occurrence.

**Neutral**

- Does not change ADR-014's firewall semantics at all. `STOP_COMPLETE` remains unconditional and non-overridable once criteria are satisfied; only the source of the criteria is settled.

## Domain Considerations

The decisive design point is rule 2. There is an attractive shortcut where the engine notices tests passing and the PR merged and concludes the work is done. That would make the firewall work everywhere with zero configuration — and it would be the product violating its own central principle. Repository evidence is evidence of repository state, not of intent (INV-003). Tests passing means tests pass; it does not mean someone accepted this as the definition of finished. The whole thesis is that information does not acquire authority by existing, and a green build is information.

So the cost in rule 5 is deliberate. A repository that declares no completion bar gets no completion stop, and Repo Governor says `UNKNOWN` rather than guessing. That is the honest degradation, and it is the same shape as every other `UNKNOWN` in the system.

## Implementation Plan

1. Define the `AcceptanceCriteriaProvider` role contract and add it to `references/providers.md` and §10's role list.
2. Define the closed `check` vocabulary; start with `tests_pass`, `file_exists`, `command_exit` and refuse to grow it without an ADR.
3. Implement `adapters/acceptance-file` reading `.repo-governor/acceptance/`; it must pass Layer 1 like any other adapter.
4. Extend `adapters/git` with the evaluation side of those checks.
5. Implement the composition in the engine: authority + criteria + evidence → `STOP_COMPLETE` | `CONTINUE` | `UNKNOWN`.
6. Add a Layer 2 scenario asserting the composition is equivalent across roadmap providers — the criteria source is the same file in every case, so this should now pass where `completion_verifiable` is currently a capability gap.
7. Write acceptance criteria for this repository's own gate conditions, and see whether maintaining them is tolerable. If it is not, that is evidence for §55.

## Related Specification Sections

§11 RoadmapAuthorityProvider · §14 RepositoryEvidenceProvider · §31 ScopeEnvelope · §40 Completion Firewall · §41 Dispositions · §54 Failure Conditions · §55 Stop Conditions · INV-003, INV-009

## Domain References

- `docs/research/2026-08-17-transport-capability-completion.md` §4
- [Define Done, Not Effort: Prompts That Make Agents Verify](https://www.digitalapplied.com/blog/define-done-acceptance-criteria-agent-prompts-2026)
- [Acceptance criteria vs. definition of done — TheServerSide](https://www.theserverside.com/tip/Acceptance-criteria-vs-definition-of-done-Whats-the-difference)
- [Definition of Done — Atlassian](https://www.atlassian.com/agile/project-management/definition-of-done)
- ADR-007 (must not declare unverifiable completion), ADR-014 (firewall), ADR-013 rule 3 (conflict, not precedence)

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
