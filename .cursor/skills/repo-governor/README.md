# Repo Governor

**A tool-independent governance skill for AI-assisted software development.**

Repo Governor determines what an AI coding agent is authorized to create, change, maintain, or retire in a repository — what it must capture or escalate instead, and when it must stop.

It does not replace your existing tools. It connects to them through provider interfaces and evaluates their state under a consistent governance protocol.

> **Providers supply state and evidence. Repo Governor determines what that state permits.**

---

## Status: architecture ready, thesis under validation

These are two different claims and were previously reported as one.

| | | |
| --- | --- | --- |
| **Implementation architecture** | **READY** | 16 adapters, a deterministic engine, 19 conformance suites. `RG-SIM-ONBOARDING-v0.1` passes; all 7 gate conditions ([§61](docs/reference/onboarding.md)) are engine-verified against live GitHub issues. |
| **Core product thesis** | **UNDER VALIDATION** | Whether one governance layer can rule identically across genuinely different trackers. Not shown yet. |

The thesis bar is stated in [#1](https://github.com/tosin2013/repo-governor/issues/1) and is **not met**: Layer 2 has never run against two *live* providers. `github-projects` and `linear` both run on recorded fixtures there, so what it currently proves is that the normalizers are self-consistent — not that they agree about a real system. The one genuinely live pairing is `decision_history`, a Dolt database against a GitHub fixture.

That distinction matters because semantic normalization is what decides whether this is tool-independent or merely adapter-shaped.

```bash
python3 engine/onboard.py <repo>           # assess, detect, propose
python3 engine/manifest.py --validate      # bind and check
python3 engine/completion.py <work-id>     # govern
```

| | |
| --- | --- |
| Roadmap authority | GitHub issues, admission by milestone ([ADR-018](docs/adrs/018-admission-signal-is-declared-not-assumed.md), [ADR-022](docs/adrs/022-repo-governor-does-not-own-roadmap-state.md)) |
| Decisions | 31 ADRs — **24 Accepted** (23 on 2026-08-17, [024](docs/adrs/024-scope-envelope-compiler.md) on 2026-08-31), 6 `Proposed` ([020](docs/adrs/020-agent-supplied-transport-with-adapter-as-normalizer.md), [029](docs/adrs/029-hooks-as-deterministic-delivery-surface.md), [030](docs/adrs/030-backend-recommendation-from-declared-capability.md), [031](docs/adrs/031-authority-source-missing-obliges-disclosure-not-refusal.md), [032](docs/adrs/032-a-decision-declares-how-it-will-be-confirmed.md), [033](docs/adrs/033-repo-local-providers-answer-about-the-checked-out-revision.md)) — **two of which, 031 and 033, the runtime depends on**, recorded as a departure in [RATIFICATION-v0.5.0.md](docs/adrs/RATIFICATION-v0.5.0.md) and again in [RATIFICATION-v0.6.0.md](docs/adrs/RATIFICATION-v0.6.0.md) — and 1 `Superseded` ([014](docs/adrs/014-scope-envelope-as-bounded-execution-contract.md), split) |
| Open thesis risks | [#1](https://github.com/tosin2013/repo-governor/issues/1) normalization (fixtures only), [#2](https://github.com/tosin2013/repo-governor/issues/2) envelope thinness (measured: *always* thin on real trackers), [#5](https://github.com/tosin2013/repo-governor/issues/5) skill activation (unmeasured) |

---

## The problem

AI coding agents work across repositories full of overlapping information — roadmaps, issues, task graphs, ADRs, specs, source, TODOs, feature flags, dead branches, execution memory. **These artifacts do not have equal authority.** But an agent will infer permission from information merely because it exists:

> "This TODO exists, so I should implement it."
> "This task is marked Ready, so I should work on it."
> "A new major version was released, so I should upgrade."
> "This module has no static references, so I should delete it."
> "The feature is complete, but I found three improvements, so I should continue."

The result is unauthorized feature expansion, roadmap drift, unsafe retirement, and agents that don't stop when the work is done.

The governing principle:

> **Information may justify a decision. Information does not acquire authority merely by existing.**

## How it works

Eight pluggable provider roles, each answering a different governance question:

| Role | Question | Example implementations |
| --- | --- | --- |
| `RoadmapAuthorityProvider` | Is this work admitted and still authorized? | Linear, Jira, GitHub Projects, file |
| `ArchitectureEvidenceProvider` | What constrains how it must be built? | ADRs, OpenSpec, Spec Kit, Mneme |
| `ExecutionStateProvider` | What is the state of work beneath it? | Beads, Amber, GAAI, Backlog.md |
| `RepositoryEvidenceProvider` | What is actually in the repository? | Git |
| `ChangeSignalProvider` | What changed outside the repository? | Renovate, Dependabot |
| `RetirementEvidenceProvider` | What obligations block removal? | static analysis, telemetry |
| `DecisionHistoryProvider` | What was already decided about this? | built-in decision log |
| `AcceptanceCriteriaProvider` | What counts as done? | repo-local declared criteria |

Keeping these separate is the point — it's what stops *"the task says READY"* from implying *"the work is authorized."*

A deterministic engine reconciles their state and returns one bounded disposition from a closed vocabulary of twelve. It returns a verdict; it never performs the action.

**Since v0.1.0 the engine emits eleven of the twelve.** `engine/completion.py` answers *is this finished?* and *is authority absent or withdrawn?*; `engine/envelope.py` compiles the ScopeEnvelope and rules on discoveries, adding `CAPTURE_ONLY`, the four review lanes, and `EXECUTE` for work substantiated as necessary to the authorized outcome. Only `CONFLICT` remains unreachable — it needs two peer providers actually disagreeing.

The v0.1.0 tag itself emits five; that limitation is recorded in the [ratification review](docs/adrs/RATIFICATION-v0.1.0.md). [ADR-024](docs/adrs/024-scope-envelope-compiler.md) is `Accepted` as of 2026-08-31: the fourth condition — measuring envelope thinness on repositories this project does not own — was answered by [#2](https://github.com/tosin2013/repo-governor/issues/2) across six repositories, five of them foreign.

Governance depth scales with repository condition (L0 greenfield → L4 mature/high-assurance), so a two-file repository needs Git, a ten-line manifest, and four invariants — nothing more.

## Install

```bash
git clone --branch v0.6.0 https://github.com/tosin2013/repo-governor /tmp/rg
/tmp/rg/tools/install-skill.sh /path/to/your/repo .claude/skills
```

**Name a tag.** `install-skill.sh` clones the checkout it is run from, so the version you get is decided by the clone above, not by the installer. Without `--branch` you install whatever `main` is at that moment, which is why the [activation-result form](https://github.com/tosin2013/repo-governor/issues/new?template=activation-result.yml) has to ask for a commit SHA. The release job builds its published tarball by running this same script at the tag, so a `--branch` install and the release artifact are the same bytes by construction.

**Use the script rather than cloning into a skills directory directly.** A plain clone brings this repository's own `AGENTS.md`, `CLAUDE.md` and `.repo-governor.json` along with it — and that last one makes the engine resolve the *install directory* as the repository under governance, so it answers confidently about the wrong project ([ADR-027](docs/adrs/027-the-governed-repository-is-not-the-install-directory.md)). The script clones and then prunes. It also offers to configure a hook, and refuses to in a repository that has no manifest, where the hook would be silent anyway.

Skills are discovered at session start, so start a **new** session and confirm the host lists `repo-governor`.

Then bind providers to roles. Detection cannot see which system is your roadmap authority, or what *admitted* means in it — milestone, project column, label — so it asks:

```bash
python3 /tmp/rg/tools/onboard-interactive.py /path/to/your/repo
```

That writes a **proposal**. Review it, rename it to `.repo-governor.json`, and commit — binding is a human act, and the commit is what stops a crafted `docs/adr/` in a fork from binding itself ([ADR-010](docs/adrs/010-provider-detection-separated-from-binding.md)).

### Then find out whether it actually works

```bash
python3 /tmp/rg/tools/selftest.py /path/to/your/repo
```

Installing a skill does not mean it will fire. Published benchmarks put roughly half of skill invocations at never firing, and this project measured **20/20 on one host and 0/2 on another** with the same skill, prompts and repository. The self-test gives you four prompts to run in your own agent and tells you what each score means. **A low score is the useful result**, and there is a [form for reporting it](https://github.com/tosin2013/repo-governor/issues/new?template=activation-result.yml).

If it scores below 3/3: add an `AGENTS.md` saying the repository is governed and how to run the engine. That was the strongest single predictor measured, and every harness reads it. See [`docs/installation.md`](docs/installation.md) for the hook surface, which is optional and which most repositories should not install.

## Repository layout

```text
adapters/      16 provider adapters, subprocess protocol, any language
engine/        deterministic policy engine, Python stdlib only
conformance/   19 suites — the evidence behind every gate claim
schemas/       manifest v1 JSON Schema
docs/
  adrs/        31 architectural decisions (24 Accepted) + index + two ratification reviews
  reference/   normative specification, §1–§70, INV-001…INV-014
  research/    external landscape sweep + transport/capability research
```

| Suite | Asserts |
| --- | --- |
| `layer1.py` | 199 contract checks across 16 adapters, with every adapter dependency installed |
| `layer2.py` | cross-provider equivalence — **the thesis test** |
| `transport.py` | agent-as-transport produces byte-identical results (70 comparisons) |
| `bindings.py` | the engine holds no adapter knowledge, and the permission gate is a gate |
| `skill.py` | the agent surface teaches an invocation that works, and cites no moved decision |
| `envelope.py` | §40 verbatim — the completion firewall admits no exception |
| `execution.py` | execution state informs; it never grants authority |
| `manifest.py` | 28 checks — 20 refusal cases; the loader's value is what it rejects |
| `onboarding.py` | `RG-SIM-ONBOARDING-v0.1`, fixtures A–C |
| `vocabulary.py` | closed sets cannot drift from the code |
| `hooks.py` | the hook delivers a real verdict, and stays silent where it has none |
| `imports.py` | ADR-011 rule 1 — the engine's dependency surface, including adapter code it execs |
| `status.py` | the status view reports what it cannot answer, rather than omitting it |
| `acceptance.py` | an empty completion bar is never a met one — two guards, each tested alone |
| `coverage.py` | every suite reports how much it did, so a suite that stops asserting cannot look green |
| `benchmark.py` | the activation harness withholds a rate for a host it has not calibrated |

**Start here:**

- [ADR index](docs/adrs/README.md) — the decisions, dependency order, and open questions
- [ADR-001](docs/adrs/001-agent-skill-as-primary-delivery-surface.md) and [ADR-002](docs/adrs/002-deterministic-policy-engine-separate-from-model-judgment.md) — the two decisions everything else assumes
- [Invariants](docs/reference/invariants.md) — INV-001 … INV-014, the irreducible rules
- [Section map](docs/reference/README.md#section-map) — where any `§NN` citation resolves

## Design commitments

- **Ships as an [Agent Skill](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)** — an open standard adopted by 26+ platforms. Installed with [`tools/install-skill.sh`](tools/install-skill.sh) into a skills directory ([which one depends on the host](docs/installation.md)); no service, no registry, no vendor lock. ([ADR-001](docs/adrs/001-agent-skill-as-primary-delivery-surface.md))
- **Deterministic, not model-judged** — invariants are executable predicates, not prose an agent is asked to honor. Same inputs, same disposition, always. ([ADR-002](docs/adrs/002-deterministic-policy-engine-separate-from-model-judgment.md))
- **Zero dependencies** — Python stdlib only. A tool that rules on what agents may change shouldn't drag in a transitive dependency tree. Adapters may be written in any language. ([ADR-011](docs/adrs/011-python-stdlib-only-engine-with-language-agnostic-adapters.md))
- **Deny by default** — no permission is inferred from an available credential. ([ADR-005](docs/adrs/005-deny-by-default-permission-model.md))
- **Provider content is untrusted input** — roadmap text and issue bodies become typed facts and cited evidence, never instructions. ([ADR-012](docs/adrs/012-provider-content-treated-as-untrusted-input.md))
- **`UNKNOWN` is a valid answer** — carrying a typed reason and a resolution path. ([ADR-007](docs/adrs/007-closed-disposition-vocabulary-with-unknown-terminal.md))

## What this is not

Not a project-management system, roadmap database, issue tracker, execution tracker, spec system, ADR system, dependency updater, CI/CD engine, static-analysis engine, autonomous architect, or hosted service. It reuses existing tools rather than rebuilding mature functionality. ([§8](docs/reference/product-scope.md))

## Open questions

Two could invalidate the product, and both are being answered early while changing course is still cheap:

1. **Does cross-provider semantic normalization actually work?** If Linear and GitHub Projects produce different dispositions from equivalent state, the abstraction has failed. Tested by [ADR-008](docs/adrs/008-fixture-based-provider-conformance-suite.md) Layer 2.
2. **How thin are compiled scope envelopes on real roadmap items?** Most trackers lack explicit non-goals, and a thin envelope governs weakly.

Full list, plus the standing [stop conditions](docs/reference/criteria.md), in the [ADR index](docs/adrs/README.md#open-questions).

## License and ownership

Licensed under the **Apache License 2.0** — see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE). Chosen for its patent grant and explicit contribution terms, because [ADR-003](docs/adrs/003-seven-provider-roles-with-normalized-contracts.md)'s adapter protocol is designed to invite third-party adapters and those need terms.

Ownership and license are separate questions and were decided separately. Public open-source under **Tosin Open Source**. Human owner and final acceptance authority: Tosin Akinosho.

Decision Crafters retains research provenance, reference-application provenance, and nonexclusive upstream use, subject to this license and separately accepted relationships. ([§68](docs/reference/product-scope.md))

---

> **Tools tell us what they know. Governance determines what that knowledge allows an agent to do.**
>
> **AI may discover broadly, but it may execute only within explicitly resolved authority.**
