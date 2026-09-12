# Architectural Decision Records — Repo Governor

Generated 2026-08-17 from PRD v0.2 (provider-oriented draft) using the MCP ADR Analysis Server's PRD-to-ADR protocol, informed by the external landscape research in [`../research/2026-08-17-external-landscape.md`](../research/2026-08-17-external-landscape.md).

**24 of 31 ADRs are Accepted.** 23 were ratified 2026-08-17 under [RATIFICATION-v0.1.0.md](RATIFICATION-v0.1.0.md); ADR-024 was ratified 2026-08-31 under [RATIFICATION-v0.5.0.md](RATIFICATION-v0.5.0.md), which also records why ADR-031 and ADR-033 were **not**. ADR-020, ADR-029, ADR-030, ADR-031, ADR-032 and ADR-033 are held `Proposed`; ADR-014 is `Superseded` by the 023/024 split.

> **Runtime-dependent and still `Proposed`: ADR-031, ADR-033.** This line used to assert that no held ADR was cited by the engine or a bound adapter — the release condition [RATIFICATION-v0.1.0.md](RATIFICATION-v0.1.0.md) states — *every architecture decision the runtime depends on is Accepted* — asserted as though it still held. It stopped holding when #153 and #183 gave ADR-024 and ADR-031 engine citations, and four releases shipped without anyone recomputing it. `conformance/skill.py` now derives this list from `engine/` and the adapters this repository binds, using the method that record itself prescribes, so it cannot go stale again. **ADR-033 joined the list with the Beads adapter (#216)**, which is the guard working rather than a documentation task: `engine/onboard.py` now applies ADR-033's rule to decide whether a `.beads/` store may be bound, so the runtime depends on a decision that is `Proposed`. Recording the fact is not ratifying it (§68). ADR-024 left this list on 2026-08-31 by being accepted, not by being reworded; ADR-031 and ADR-033 remain on it, and v0.5.0 ships as a departure recorded in that review. Per §61, the project cannot reach `IMPLEMENTATION_READY` until the `RG-SIM-ONBOARDING-v0.1` simulation passes.

> **`PRD.md` no longer exists.** Its normative content was extracted into [`../reference/`](../reference/) on 2026-08-17, with original § numbering preserved so every citation below still resolves. Start at the [section map](../reference/README.md#section-map).

## Index

| # | Decision | Domain | Settles |
| --- | --- | --- | --- |
| [001](001-agent-skill-as-primary-delivery-surface.md) | Agent Skill as the primary delivery surface | Distribution | Delivery form was undecided |
| [002](002-deterministic-policy-engine-separate-from-model-judgment.md) | Deterministic policy engine separate from model judgment | Policy engine | Makes §53 metrics measurable |
| [003](003-seven-provider-roles-with-normalized-contracts.md) | Seven provider roles with normalized contracts | Provider abstraction | Adapter wire protocol; partial support |
| [004](004-governance-manifest-as-sole-binding-artifact.md) | Governance manifest as the sole binding artifact | Configuration | What `.repo-governor.json` *means* |
| [005](005-deny-by-default-permission-model.md) | Deny-by-default permission model | Security | Default for unstated permissions |
| [006](006-repository-condition-model-drives-governance-profiles.md) | Repository condition drives governance profiles | Policy scaling | How L0–L4 is assessed and who decides |
| [007](007-closed-disposition-vocabulary-with-unknown-terminal.md) | Closed disposition vocabulary, `UNKNOWN` terminal | Output contract | What `UNKNOWN` means operationally |
| [008](008-fixture-based-provider-conformance-suite.md) | Fixture-based provider conformance suite | Testing | How portability is proven |
| [009](009-append-only-evidence-chain-for-decision-provenance.md) | Append-only evidence chain for provenance | Observability | Where decision history lives |
| [010](010-provider-detection-separated-from-binding.md) | Detection strictly separated from binding | Onboarding | Makes INV-013 structural |
| [011](011-python-stdlib-only-engine-with-language-agnostic-adapters.md) | Python stdlib-only engine, any-language adapters | Runtime | Language and dependency policy |
| [012](012-provider-content-treated-as-untrusted-input.md) | Provider content treated as untrusted input | Security | **Gap in §51** — prompt injection |
| [013](013-single-canonical-authority-per-role.md) | Single canonical authority per role, halt on conflict | Conflict resolution | Role cardinality; no tie-breaking |
| ~~014~~ | ~~ScopeEnvelope + completion firewall~~ — **superseded**, split into [023](023-completion-firewall.md) and [024](024-scope-envelope-compiler.md) | Domain model | Bundled two decisions; only one shipped |
| [015](015-json-as-canonical-manifest-format.md) | JSON as the canonical manifest format | Configuration | Closes ADR-011's deferred choice; unblocks gate 5 |
| [016](016-mcp-as-adapter-transport-not-adapter-replacement.md) | MCP as adapter transport, not replacement | Integration | Whether providers route via MCP servers |
| [017](017-completion-evidence-from-repo-local-acceptance-artifact.md) | Completion evidence from a repo-local acceptance artifact | Domain model | Makes `STOP_COMPLETE` derivable; unblocks gate 7 |
| [018](018-admission-signal-is-declared-not-assumed.md) | Admission signal is declared, not assumed | Provider abstraction | Makes GitHub usable without a Project; **found by real-data testing** |
| [019](019-database-backed-decision-history.md) | Database-backed decision history | Storage | Closes the last unbound role; amends ADR-009 |
| [020](020-agent-supplied-transport-with-adapter-as-normalizer.md) | Agent-supplied transport, adapter as normalizer | Transport | Moves credentials out of remote adapters; amends ADR-003 and ADR-016 |
| [021](021-every-provider-resolved-through-the-manifest.md) | Every provider resolved through the manifest | Engine architecture | Builds ADR-005 rule 2, specified and never implemented |
| [022](022-repo-governor-does-not-own-roadmap-state.md) | Repo Governor does not own roadmap state | Product boundary | **Section 54 reproduced inside the tool**; roadmap rebound to GitHub |
| [023](023-completion-firewall.md) | The completion firewall | Policy engine | Split from 014 — the half that shipped |
| [024](024-scope-envelope-compiler.md) | ScopeEnvelope is compiled, not authored | Policy engine | Split from 014 — **the half that did not**; held Proposed |
| ~~025~~ ~~026~~ | **never written.** Reserved in a plan that proposed a dialect ADR and an identity ADR; only the second was written, and it landed as 028. Nothing was deleted — recorded so a gap in the sequence does not read as two missing decisions. | — | — |
| [027](027-the-governed-repository-is-not-the-install-directory.md) | The governed repository is not the install directory | Portability | The engine governed **itself** whatever it was pointed at |
| [028](028-provider-identity-is-never-defaulted.md) | Provider identity is never defaulted | Safety | Adapters named the author's repository as a fallback |
| [029](029-hooks-as-deterministic-delivery-surface.md) | Hooks as a deterministic delivery surface | Distribution | The skill description was understood, and lost to a legible task anyway |
| [030](030-backend-recommendation-from-declared-capability.md) | Backend recommendation from declared capability | Providers | A level-to-product table would put adapter names in the engine and break on the third backend |
| [031](031-authority-source-missing-obliges-disclosure-not-refusal.md) | AUTHORITY_SOURCE_MISSING obliges disclosure, not refusal | Agent surface | A measured run reasoned past a bare imperative, and may have been right |
| [032](032-a-decision-declares-how-it-will-be-confirmed.md) | A decision declares how it will be confirmed | Architecture | 267 obligations across the Accepted ADRs, one discharge marker between them |
| [033](033-repo-local-providers-answer-about-the-checked-out-revision.md) | A repo-local provider answers about the checked-out revision | Providers | The branch-varying methods are the two the engine never calls |

## Dependency order

```text
001 (skill)  ──┬──> 006 (profiles) ──> 008 (conformance)
               │
002 (engine) ──┼──> 007 (dispositions) ──> 014 (envelope + firewall)
               │
003 (roles)  ──┼──> 004 (manifest) ──┬──> 005 (permissions)
               │                     ├──> 010 (detection)
               │                     └──> 013 (cardinality)
               │
011 (runtime) ─┘
012 (untrusted input) — cross-cutting, constrains 003, 004, 009, 010
015 (JSON manifest) — resolves 011's deferred choice; unblocks gate 5
016 (MCP transport) — constrains 003; engine never calls MCP directly
017 (acceptance artifact) — adds 8th role; unblocks gate 7, fixes #2 root cause
009 (evidence chain) — depends on 002, 004; enables INV-005, INV-008
```

Read 001 and 002 first. They are the two decisions everything else assumes.

## What changed relative to PRD v0.2

The ADRs are not a restatement. Six things are settled, added, or contradicted — all four textual changes are now folded into [`../reference/`](../reference/):

1. **Delivery form decided (001).** PRD v0.2 listed MCP, hooks, IDE, and CI as undifferentiated future candidates. Agent Skills becoming a 26-platform open standard on 2025-12-18 makes skill-first the tool-independent choice rather than a vendor bet. §65 updated.
2. **Determinism made binding (002).** The invariants read as prose. An agent that reasons past prose is the documented April 2026 failure. Invariants become executable predicates; `SKILL.md` documents the code rather than carrying the rule.
3. **New ADR with no PRD basis (012).** §51's security model did not address prompt injection, yet every input is text third parties can write. §51 amended with the missing bullet.
4. **§21 inconsistency resolved (013).** The manifest sketch treated some roles as lists and others as scalars without saying why. ADR-013 makes cardinality a rule derived from whether a role answers "is this authorized?" §21 annotated.
5. **A live stop condition identified.** Beads issue #1150 requests plugin-based tracker integrations. If delivered, it overlaps the provider-abstraction thesis. §55 now tracks it as a concrete input rather than an abstract risk. Current assessment: Beads federating trackers makes it a better `ExecutionStateProvider`, not a competitor, because it aggregates state without ruling on authority.
6. **Differentiation confirmed.** No tool in the spec-driven-development or tracker categories answers "is this work currently authorized?" across providers. OpenSpec and Spec Kit answer *how to build*; none reconciles a spec against a withdrawn roadmap item. The §56 differentiation candidate survives the sweep.

## Open questions

These are not deferred decisions — they are things no one can answer yet without evidence.

| # | Question | Where it bites | Resolved by |
| --- | --- | --- | --- |
| ~~1~~ | ~~Manifest canonical format~~ — **resolved by [ADR-015](015-json-as-canonical-manifest-format.md)**: JSON canonical. Spike showed a 143-line YAML subset silently mis-types 7/10 realistic values. | 011, 004 | ✅ [#3](https://github.com/tosin2013/repo-governor/issues/3) closed |
| 2 | Decision-record redaction default for public repositories | 009, 012 | Design decision before v1 |
| 3 | Does cross-provider semantic normalization actually work? | 003, 008 | ADR-008 Layer 2 with two real providers — **this is the thesis test** |
| 4 | How thin are compiled ScopeEnvelopes on real roadmap items? | 014 | Measurement across real repositories |
| 5 | Skill activation reliability across hosts | 001 | 20-prompt activation test on 3 hosts |
| 6 | Monorepo / per-package governance | 004, 006 | Deliberately deferred past v1 |

Questions 3 and 4 are the ones that could invalidate the product. Both should be answered early, while changing course is still cheap.

## Path to IMPLEMENTATION_READY

[§61](../reference/onboarding.md) lists seven gate conditions. Current coverage:

| Gate condition | Status | Evidence |
| --- | --- | --- |
| 1. Onboarding simulation passes | ✅ **Met** — `completion.py GATE-1` → `STOP_COMPLETE` | [`conformance/onboarding.py`](../../conformance/onboarding.py) **29/29**, `RG-SIM-ONBOARDING-v0.1: PASS`. Fixtures A–C materialized into isolated temp repos |
| 2. Provider conflicts handled deterministically | ✅ **Met** — `completion.py GATE-2` → `STOP_COMPLETE` | Fixture C raises `PROVIDER_CONFLICT` on `roadmap_authority`, names both candidates, applies no ranking |
| 3. Low-complexity repos need no extra providers | ✅ **Met** — `completion.py GATE-3` → `STOP_COMPLETE` | Fixture A detects only `repository`; condition L0; state `AUTHORITY_SOURCE_MISSING` |
| 4. Detection does not assign authority | ✅ **Met** — `completion.py GATE-4` → `STOP_COMPLETE` | Proposal written to `.repo-governor.proposed.json`; asserted the manifest loader never reads it; binding requires human promotion |
| 5. Manifest semantics stable | ✅ **Met** — verified by `engine/completion.py GATE-5` → `STOP_COMPLETE` | `schemas/manifest-v1.json` frozen; `engine/manifest.py` fails closed; [26/26 refusal cases](../../conformance/manifest.py); `--validate` reaches `READY_FOR_GOVERNANCE`; `engine/migrate.py` stub exists per ADR-004 step 5 |
| 6. One viable adapter per required core role | ✅ **Met** — verified by `engine/completion.py GATE-6` → `STOP_COMPLETE`, not by assertion | All 9 adapters pass [Layer 1](../../conformance/layer1.py) **112/112**; [Layer 2](../../conformance/layer2.py) 9/9 EQUIVALENT across 3 roadmap providers; criteria in `.repo-governor/acceptance/GATE-6.json` |
| 7. `UNKNOWN` and failure behavior defined | ✅ **Met** — `completion.py GATE-7` → `STOP_COMPLETE` | [`engine/vocabulary.py`](../../engine/vocabulary.py) closes 12 governance + 8 onboarding dispositions and 18 reasons; [`conformance/vocabulary.py`](../../conformance/vocabulary.py) proves the sets cannot drift from the code |

**All seven gate conditions are met**, each on `engine/completion.py`'s verdict rather than assertion. Per §61 the specification may move to `IMPLEMENTATION_READY`.

Reproduce the whole thing:

```bash
for n in 7 8 9 10 11 12 13; do python3 engine/completion.py $n; done   # GitHub issues, live
./tools/run-conformance.sh
```

The gate ids are GitHub issue numbers, not `GATE-N`. They were file-roadmap ids until ADR-022 moved the roadmap of record to GitHub.

### What `IMPLEMENTATION_READY` does not mean

The gate asks whether the architecture is stable enough to build on. It does not ask whether the product is worth building, and four things remain open:

* ~~**All 22 ADRs are still `Proposed`.**~~ **Resolved 2026-08-17.** 21 of 24 Accepted; ADR-020 and ADR-024 deliberately held. See [`RATIFICATION-v0.1.0.md`](RATIFICATION-v0.1.0.md).
* **#1 (cross-provider normalization) has weak evidence.** 9/9 scenarios agree, but all three adapters were written by the same author against the same map — that measures shared intent as much as portability. A third-party adapter is the real test.
* **#2 (envelope thinness) is not merely unmeasured — it is unanswerable as written.** ADR-014's envelope compiler was never built, so nothing compiles an envelope whose thinness could be measured. Found by the ratification sweep.
* **The criteria-drift weakness is unresolved.** Acceptance criteria were amended twice during implementation. Both amendments were justified and recorded, and nothing but that record distinguishes a legitimate correction from a convenient one.
