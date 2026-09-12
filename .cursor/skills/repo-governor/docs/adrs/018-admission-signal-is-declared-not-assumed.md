# 18. The Admission Signal Is Declared, Not Assumed

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Provider abstraction / roadmap authority

## Context

`adapters/github-projects` reads admission from Projects v2 board membership: an issue's `Status` column maps to `ADMITTED` or `AUTHORIZED`, and an issue not on a board yields `NOT_ON_BOARD` — a blocking unknown.

Running it against a real repository showed that assumption does not hold. `tosin2013/mcp-adr-analysis-server` has **380 issues and no Project at all** (`projectsV2: []`). Every issue in it therefore resolves to a blocking `UNKNOWN`, which is honest — board state genuinely cannot be read — and useless. The adapter is unusable on what is likely the majority case: a repository that tracks work in issues without a Project board.

But the repository is not ungoverned. It has a clear admission practice, visible in its data:

| Signal | Observed |
| --- | --- |
| Milestones | `v2.6` (6 of 29 open), `v3.0` (8 of 16 open) — release-targeted |
| Labels | `priority:high`, `bug`, `documentation`, `agentic-workflows` (55) |
| Closed reasons | **54 `NOT_PLANNED`**, 6 `COMPLETED` |
| Assignees | 24 open issues, all unassigned |

Milestone membership *is* the admission mechanism. "This issue is in `v3.0`" means the same thing "this issue is in the Ready column" means elsewhere. And `NOT_PLANNED` — 54 of 60 closed issues — is an unambiguous authority-withdrawn signal already handled correctly.

So the adapter was not wrong about GitHub. It was wrong to assume *one* GitHub convention.

### Why the obvious fix is the wrong fix

The tempting move is to make the adapter fall back: try Projects, then milestones, then labels. That would make it work everywhere with no configuration.

It would also be Repo Governor deciding what admission means in someone else's repository. A milestone is a *release target*; treating membership as authorization is an interpretation, and a defensible one, but not a fact. Baking a fallback chain in would be the same failure as ranking two roadmap candidates to resolve a conflict (ADR-013 rule 1) — an automatic resolution that silently confers authority, which INV-013 exists to prevent.

## Decision

**The admission signal is named in the manifest. The adapter reads what it is told to read, and refuses to guess.**

1. **A roadmap binding declares its `admission` model.** One of a closed set:

   ```json
   "roadmap_authority": {
     "type": "github",
     "adapter": "adapters/github-projects",
     "admission": { "signal": "milestone" }
   }
   ```

   | `signal` | Admitted when | Authorized when |
   | --- | --- | --- |
   | `project_status` | on a Project board | `Status` in the authorizing set |
   | `milestone` | assigned to any open milestone | assigned **and** the milestone is the current one, or an assignee exists |
   | `label` | carries the declared admission label | carries the declared authorization label |
   | `none` | never inferred | never inferred — every item is `NOT_ADMITTED` |

2. **No fallback chain, ever.** If the declared signal is unreadable, the result is a blocking `UNKNOWN` naming the signal that was expected. The adapter does not try the next-best thing. A repository whose admission practice changes must say so in the manifest, which is a reviewed commit.

3. **Absent declaration is not a default.** A binding with no `admission` block yields `ADMISSION_SIGNAL_UNDECLARED`, blocking. This is deliberately inconvenient: the alternative is the adapter picking, and the whole thesis is that it must not.

4. **Withdrawal is signal-independent.** Closed as `NOT_PLANNED` or `DUPLICATE` is `CANCELLED` regardless of which admission model is declared. Withdrawal is unambiguous in GitHub's own vocabulary; only *admission* is convention-dependent.

5. **The adapter is renamed in spirit if not in path.** It is a GitHub roadmap-authority adapter that supports several admission conventions, not a Projects adapter. `type` becomes `github`; the path stays for compatibility.

## Consequences

**Positive**

- The adapter becomes usable on the common case — issues plus milestones, no board — without inferring anything.
- The declaration is a small, reviewable line that makes an otherwise invisible interpretation explicit. A reader of the manifest can see *why* the engine thinks an issue is admitted.
- Extends to conventions we have not met: a new `signal` value is additive, and until one is declared nothing changes behaviour.
- Repo Governor's own repository is a live example. It has both a Project *and* a milestone, so it must declare which one governs — exactly the ambiguity this ADR forces into the open.

**Negative**

- **More configuration for a case users will feel should just work.** Someone with a milestone-driven repository must write a line to say so, and will reasonably ask why the tool cannot see what is obvious. §54 names configuration too complex for simple repositories as a failure condition, and this pushes in that direction.
- **The mapping within a signal is still interpretation.** Declaring `milestone` does not say whether an open milestone means *admitted* or *authorized*. Rule 1's table picks one reading; a project that disagrees has no way to express that yet, short of another signal type.
- **Four signals is not all of them.** Repositories use assignee, column position in a legacy board, an `epic:` label prefix, a linked discussion. Each addition is another closed-set entry and another thing to test.
- Onboarding detection can now propose an admission model, and proposing one is a whisker from assuming one. Detection must cite the evidence (`"repo has 2 open milestones and no Project"`) and still leave the choice to a human.

**Neutral**

- No change to `file-roadmap` or `linear`, which have explicit authority fields and need no convention.

## Domain Considerations

The `NOT_PLANNED` finding is worth keeping in view: 54 of 60 closed issues in the observed repository were closed as not-planned. Under INV-005 and §39, those are *recorded decisions* — work that was considered and declined. A rediscovery of any of them must resolve to `CAPTURE_ONLY` or `ROADMAP_REVIEW`, never `EXECUTE`. That is a large body of decision history sitting in a signal Repo Governor already reads correctly, and it is the strongest argument that GitHub issues can serve as a real roadmap authority rather than a degraded one.

There is also a self-referential point. This repository has a Project *and* a milestone, and until now nothing forced a statement of which is authoritative. That ambiguity was invisible while the adapter assumed Projects. Requiring the declaration surfaces it — which is the ADR working on its author before anyone else.

## Implementation Plan

1. Add the `admission` block to the manifest schema; closed `signal` enum; no default.
2. Implement `milestone` and `label` signals in `adapters/github-projects`, with `project_status` as the existing path.
3. Add `ADMISSION_SIGNAL_UNDECLARED` and `ADMISSION_SIGNAL_UNREADABLE` to the closed reason vocabulary, both blocking.
4. Extend Layer 2 with a scenario per signal, asserting equivalent dispositions from equivalent state expressed three ways.
5. Teach `engine/onboard.py` to propose an admission model with cited evidence, never to select one.
6. Declare this repository's own signal in `.repo-governor.json` and record why.

## Related Specification Sections

§11 RoadmapAuthorityProvider · §19 Provider Detection · §20 Provider Conflict Handling · §21 Repository Governance Manifest · §39 Rediscovered Work · §54 Failure Conditions · INV-005, INV-013

## Domain References

- Observed 2026-08-17 on `tosin2013/mcp-adr-analysis-server`: 380 issues, `projectsV2: []`, milestones `v2.6` and `v3.0`, 54 of 60 closed issues `NOT_PLANNED`
- ADR-013 rule 1 (no ranking, no automatic tie-break), ADR-010 (detection proposes, never binds)

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
