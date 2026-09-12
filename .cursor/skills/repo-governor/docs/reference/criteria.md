# Security, Success Metrics, Failure & Stop Conditions

> Extracted from PRD v0.2 §51, §53–§56, §62–§65 on 2026-08-17. Original section numbering preserved.
> **Normative.** §54 and §55 are the standing kill criteria for this project.

---

## §51 — Security and Boundary Model

Repo Governor must:

* use least privilege;
* keep provider credentials separate from governance authority;
* avoid secret persistence;
* preserve provider provenance;
* prevent cross-repository state leakage;
* prevent cross-identity memory leakage;
* respect repository ownership boundaries;
* fail conservatively when provider access fails;
* **treat all provider content as untrusted input** — added by ADR-012, 2026-08-17.

A provider being technically writable does not mean Repo Governor may write to it (INV-014).

> The final bullet closes a gap in the original PRD. Every input — roadmap descriptions, ADR bodies, task notes, commit messages, advisory text — is prose third parties can write, flowing toward an agent's context. See ADR-012 for the trust boundary and its honest limits.

## §53 — Success Metrics

Initial reference-app metrics:

| Metric | Target | Notes |
| --- | --- | --- |
| Authority fidelity — correctly permit or refuse work | 100% | synthetic |
| Unauthorized execution | 0 | |
| Completion-stop compliance | 100% | |
| Discovery preservation — material discoveries not lost when capture is configured | 100% | |
| Discovery over-promotion | 0 | |
| Retirement safety — removal authorized while known obligation exists | 0 | |
| Provider portability — semantically equivalent provider states produce equivalent dispositions | 100% across conformance fixtures | **the thesis test** |

**Human intervention** is empirical only initially: unnecessary escalations; false positives; context-reconstruction time; review time. No numeric target until a real-world baseline exists.

> These are only measurable because ADR-002 makes evaluation deterministic. Portability is measured by ADR-008 Layer 2.

## §54 — Failure Conditions

Repo Governor fails if:

* it becomes another canonical roadmap database;
* it requires a specific tracker;
* it requires Beads;
* it requires ADRs;
* it invents architecture;
* it creates future work automatically;
* it blocks routine reversible implementation excessively;
* it turns all discoveries into human review;
* provider configuration is too complex for simple repositories;
* it cannot resolve withdrawn authority;
* it cannot stop after completion;
* it allows deletion from weak evidence;
* it silently interprets provider availability as provider authority.

*Added 2026-08-17 from the #16 research. Occupying a position between the agent and heterogeneous systems brings failure modes the original list, written for a pure policy layer, did not anticipate:*

* it becomes a bottleneck on every agent action;
* it is blamed for provider outages it merely relays;
* it accretes integration surface until it competes with iPaaS tooling.

*Added 2026-08-17 from ADR-022, after the first condition on this list came true in this repository:*

* **a provider fixture is bound as a repository's provider of record.**

## §55 — Stop / Simplification Conditions

Simplify or stop development if testing shows:

* strong repository instructions perform equivalently;
* existing governance products already implement the complete model;
* provider abstraction adds more burden than value;
* cross-provider semantic normalization is not reliable;
* human review increases rather than decreases;
* governance causes material developer friction without corresponding safety or decision-quality benefit.

Possible fallback — **Repo Governor Lite**, with only: authority check; scope envelope; discovery capture; completion stop.

### Live stop-condition inputs (2026-08-17)

| Input | Bears on | Current assessment |
| --- | --- | --- |
| Beads issue [#1150](https://github.com/gastownhall/beads/issues/1150) requests plugin-based tracker integrations | "existing products already implement the model"; "provider abstraction adds more burden than value" | **Not triggered.** Beads federating trackers aggregates state without ruling on authority, which makes it a better `ExecutionStateProvider`, not a competitor. Monitor. |
| ADR-008 Layer 2 cross-provider equivalence | "cross-provider semantic normalization is not reliable" | **Not triggered (2026-08-17).** 9/9 EQUIVALENT across `file-roadmap`, `github-projects`, `linear`; 0 divergences; 2 honestly-advertised capability gaps. Evidence still weak — see limits below. |
| **Acceptance conditions absent from every real tracker** | "governance causes material friction without benefit"; INV-009 enforceability | **RESOLVED by ADR-017 (2026-08-17).** Criteria moved out of the tracker into `.repo-governor/acceptance/`. `STOP_COMPLETE` now derives identically from `file-roadmap`, `linear` and `github-projects`, verified end to end. Residual cost: a declared artifact per work item, which remains an adoption question. |
| ScopeEnvelope thinness on real roadmap items | "governance causes material friction without benefit" | **Unresolved, and worse than assumed.** Both real tracker adapters return `NON_GOALS_UNSTATED` and `ACCEPTANCE_UNSTATED` for every item — not "often thin" but "always thin" absent custom fields. |

#### Limits of the current Layer 2 evidence

Stated plainly so the result is not over-read:

* Three providers, but `file-roadmap` is a baseline designed to agree, so effectively two.
* **All three adapters were written by the same author against the same normalization map.** Convergence partly measures that shared intent, not independent agreement. A third-party adapter is the real test.
* The Linear fixture is **entirely synthetic** — no credentials were available. Shaped to Linear's schema, but not recorded from a live workspace.
* 3 of the GitHub cases are synthetic nodes.
* Only the `roadmap_authority` role. Architecture, execution, and retirement normalization are untested.
* Jira — the hardest case, with per-project configurable workflow states — is backlogged (#14).

## §56 — Research Evidence

### Simulation 1 — Authority vs persistent execution-state architecture

> Persistent execution memory improves continuity but does not establish authority.

### Simulation 2 — Repository complexity, greenfield through mature/high-assurance

> Governance need is high both where constraints are absent and where accumulated obligations are numerous.

### External landscape research

Existing systems cover substantial pieces: architecture governance; AI execution gating; scope discipline; task memory; dependency automation; repository analysis.

Differentiation candidate:

> Cross-provider authority resolution and governed repository-evolution state transitions.

**Confirmed 2026-08-17.** The broad sweep in [`../research/2026-08-17-external-landscape.md`](../research/2026-08-17-external-landscape.md) found no tool in the spec-driven-development or tracker categories that answers "is this work currently authorized?" across providers. OpenSpec and Spec Kit answer *how to build*; none reconciles a spec against a withdrawn roadmap item.

**Bounded 2026-08-24.** A second sweep, [`../research/2026-08-24-architecture-decisions-and-release-readiness.md`](../research/2026-08-24-architecture-decisions-and-release-readiness.md), asked whether an Accepted architecture decision with undischarged obligations is a fact a release should measure. It found the concept already named in ADR practice — MADR's optional `Confirmation` section — mechanical confirmation bounded to *structural* obligations, and decision immutability forbidding discharge state from living in the decision itself. Measured on this project's own ledger: **267 post-acceptance obligations, one discharge marker.**

The differentiation candidate is **not** extended to cover it. [ADR-032](../adrs/032-a-decision-declares-how-it-will-be-confirmed.md) requires a decision to declare how it would be confirmed and refuses to build anything that polices the answer — §65 lists release governance among Future Candidate Capabilities followed by "none are authorized merely by inclusion here", and a mechanism firing on 267 items would meet §54's *blocks routine reversible implementation excessively*.

---

## §62 — Initial Implementation Boundary

The first reference implementation should prove the architecture, not maximize integrations. Suggested minimum providers:

```text
Repository:        Git
Roadmap authority: synthetic/manual provider or one real provider
Architecture:      ADR provider
Execution:         Beads or synthetic execution provider
Change signals:    synthetic fixture
Retirement:        repository-analysis fixture
```

A second provider should then be implemented for at least one category to prove portability.

> **Amended 2026-08-17.** ADR-017 added `AcceptanceCriteriaProvider` as an eighth role, so this minimum set is now **seven** items, not six — add an acceptance-criteria adapter. The portability requirement is **satisfied**: `github-projects` and `linear` are both real second/third providers for `roadmap_authority`.

## §63 — MVP Requirements

1. repository attachment; 2. repository-condition assessment; 3. provider detection; 4. provider selection; 5. manifest generation; 6. provider connectivity validation; 7. authority resolution; 8. scope envelope generation; 9. discovery capture; 10. architecture-state resolution; 11. withdrawn-authority detection; 12. completion stop; 13. `UNKNOWN`; 14. provider provenance; 15. one real roadmap provider; 16. one real execution provider; 17. Git repository evidence.

## §64 — Explicit MVP Non-Commitments

Not required for MVP: hosted service; dashboard; multi-tenant service; marketplace; proprietary provider registry; plugin marketplace; organization billing; remote policy server; automated dependency upgrades; autonomous cleanup; public API; enterprise RBAC; commercial support.

These remain candidates only.

## §65 — Future Candidate Capabilities

Provider SDK; plugin discovery; policy-as-code; CI enforcement; coding-agent hooks; IDE integration; MCP interface; governance visualization; lineage graph; multi-agent concurrency; release governance; organization policy packs; hosted governance service; policy simulation; policy audit reports; portable governance profiles.

**None are authorized merely by inclusion here.**

> ADR-001 promoted "agent skill" from this list to the primary delivery surface and deferred MCP, CLI, and CI enforcement as secondary wrappers over the same engine.

> ADR-029 (2026-08-19) promoted **"coding-agent hooks"** from this list to a secondary delivery surface, on measured evidence that skill activation is unreliable (issue 36 Arm A prompt 1) and research showing hooks are no longer vendor-specific. MCP, CLI and CI enforcement remain deferred.
