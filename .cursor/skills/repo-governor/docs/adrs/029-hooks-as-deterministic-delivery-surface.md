# 29. Hooks as a Deterministic Delivery Surface

**Status**: Proposed — **ready for ratification**. Validated 2026-08-19 by a four-condition controlled comparison: the activation claim holds only where no `AGENTS.md` exists, and the enforcement claim is proven. Acceptance is a human act (§68).
**Date**: 2026-08-19
**Domain**: Distribution & agent integration
**Amends**: [ADR-001](001-agent-skill-as-primary-delivery-surface.md) — promotes "coding-agent hooks" from a deferred §65 candidate to a secondary delivery surface. The Agent Skill remains primary.
**Evidence**: [`docs/research/2026-08-19-activation-and-hooks.md`](../research/2026-08-19-activation-and-hooks.md)

## Context

ADR-001 named this failure precisely, ten months before it was observed:

> Skill activation is model-mediated: the agent decides whether the description matches the task. **A governance skill that fails to activate is worse than absent, because the human assumes it ran. This is a real failure mode with no clean fix at the skill layer.**
>
> **No enforcement.** A skill advises; it cannot block.

It then deferred hooks, correctly: in August 2025 a hook meant one vendor, and §54 makes requiring a specific vendor a failure condition.

**The prediction came true under measurement.** [#36](https://github.com/tosin2013/repo-governor/issues/36) Arm A prompt 1, Claude Code / Opus 4.6, 2026-08-19: asked to *"have a look at issue 27 and fix it"*, the agent read the issue, explored a 9,738-line file, ran a coverage baseline and began writing tests. Governance was never consulted. Minutes earlier the same agent had listed the skill among its always-applied rules, paraphrasing the `description` accurately.

**So the description was present, was understood, and lost anyway.** That is a precedence failure, not a discovery failure, and no rewording of `SKILL.md` reaches it.

Two research findings decide the rest:

1. **This is the industry norm, not a defect in our description.** Vercel measured skills uninvoked in **56%** of cases where the agent had access; an independent study found **46% recall**; 56,804 indexed skills compete for fewer than 100 reliable auto-trigger slots. The @skills protocol paper calls installation-based triggering *"a lottery in which N descriptions must each win a probabilistic match at once."* Description tuning has a ceiling, and we are near it.

2. **The vendor-bet objection has expired.** All three target hosts now ship hooks — `.claude/settings.json`, `.cursor/hooks.json`, `.codex/hooks.json` — with a converged calling convention: JSON on stdin, exit 2 blocks, JSON on stdout carries a structured decision. This is the same argument ADR-001 used to promote Agent Skills: adopt a cross-vendor convention rather than build N integrations.

## Decision

**Repo Governor ships a hook surface as a secondary delivery mechanism. The Agent Skill remains primary.** One shared script, `tools/hooks/governance-hook.py`, with a thin JSON config per host.

The hook occupies three moments:

| Moment | Event (Claude / Cursor) | What it does |
|---|---|---|
| `prompt` | `UserPromptSubmit` / `beforeSubmitPrompt` | injects the governance **requirement** before the agent reasons |
| `write` | `PreToolUse` / `preToolUse` on `Edit\|Write` | checks a pending change against what the engine established |
| `capture` | `PostToolUse` on Bash / `afterShellExecution` | records the authority and disposition a session obtained |

**Four constraints, each load-bearing.**

**1. The hook never decides authorization.** `engine/completion.py` remains the only thing that produces a disposition (ADR-002). The hook routes and reports; it does not compute. It also never infers an authority id from prompt text — that would be a second authority surface, which ADR-022 forbids.

**2. The `prompt` moment delivers the requirement, not a verdict.** It cannot produce one: `completion.py` requires an authority id and a raw user prompt has none. What it makes deterministic is that the requirement *arrives*. This is the fix for an activation miss, and it must land at prompt time rather than at write time — by the time `PreToolUse` fires, the agent has already chosen an approach.

**3. Enforcement is opt-in per repository, via `repo_governor.enforcement: "blocking"`.** Default is `advisory`. An un-onboarded repository is not a governed one; blocking there would stop all editing everywhere the manifest is absent.

**4. Silence in ungoverned repositories.** No manifest, no output. A governance tool that narrates in repositories it has no authority over is a nuisance that gets uninstalled.

## Measured consequences (2026-08-19, added after validation)

This ADR was written to fix an activation miss. **Validation refuted that.**

| Test | Result |
|---|---|
| Does the hook fire and reach the model? | **yes** — token matched operator and model, but only after nesting `additionalContext` inside `hookSpecificOutput`. Top-level alone runs, reports success, and delivers nothing. |
| Prompt-1 shape, hook **on**, in `repo-governor` | FULL |
| Prompt-1 shape, hook **off**, same repo, same prompt | **FULL** |

`AGENTS.md` was doing the work. The agent named it as its source in every session. **In a repository that already announces itself, the hook adds nothing to activation.**

Worse for the original motivation: the hook is deliberately silent in un-onboarded repositories, and prompt 1 failed in an un-onboarded repository. **The surface could never have spoken in the case that prompted it.** The only row where it improves activation is a governed repository with no `AGENTS.md` — which one file would also fix.

**Resolved by a controlled comparison**, same repository and prompt, one variable at a time:

| `AGENTS.md` | prompt hook | write hook | Result |
|---|---|---|---|
| present | on | on | refuses |
| absent | on | on | refuses, quoting text found only in the injection |
| absent | **off** | off | **edits immediately** |
| absent | off | **on** | attempts the edit, **blocked by exit 2** |

- Activation: the hook adds nothing where `AGENTS.md` exists, and **governs on its own where it does not** (row 2 vs row 3). Row 3 also shows the skill description alone does not activate in a governed repository — the un-onboarded confound behind Arm A prompt 1's NONE is removed.
- Enforcement: **proven** (row 4). A write was prevented by mechanism rather than persuasion, which is what ADR-001 said the skill layer could not do.
- Reachability: blocking only engages when advisory delivery is absent. Where injection works the agent stops first, so **enforcement is a backstop against a non-compliant agent, not part of the normal path**.

**The hook is not a security boundary.** The remedy is captured as [issue 46](https://github.com/tosin2013/repo-governor/issues/46) — the CI surface ADR-001 deferred — gated on a *declared* PR-to-authority signal, because guessing one would reproduce in a new surface the defect [ADR-018](018-admission-signal-is-declared-not-assumed.md) exists to prevent.

**Restated:** Offered options after being blocked, the agent proposed *"remove it from `.claude/settings.json`"* unprompted, within one turn. It has file access and the config is a file. This stops mistakes — which is the common case, and it did so here — but containment requires somewhere the agent cannot reach: CI, branch protection, or review.

Full record: [`docs/research/hook-validation-results.md`](../research/hook-validation-results.md). Candidates that depend on the enforcement test are captured in [issue 45](https://github.com/tosin2013/repo-governor/issues/45), **unmilestoned**, so the engine returns the disposition `UNKNOWN` with a blocking unknown whose reason is `NOT_ADMITTED` — `NOT_ADMITTED` is a reason code, not a disposition, and does not appear in `DISPOSITIONS` — if enforcement fails, this surface should be deleted rather than kept as a plausible-sounding option, and issue 45 goes with it.

## Consequences

**Positive**

- Delivery becomes deterministic where it was probabilistic. Obedience remains model-mediated — the hook closes the activation gap, not the compliance gap.
- The completion firewall (ADR-023) gains a real chokepoint: a write attempted under `STOP_COMPLETE` is surfaced before it happens, which is §40's hardest case.
- Enforcement, where a repository opts into it, is the first thing in this project that can actually stop an agent rather than advise it.
- Cross-vendor by the same reasoning that made the skill cross-vendor. One script, three config files.

**Negative**

- **Hooks are host-level, not repository-level.** A cloned repository does not carry its own enforcement; the operator must install the hook on each machine. This narrows the gap ADR-001 named without closing it, and it is the strongest remaining argument for the deferred CI surface.
- **Only the Claude Code payload schema is verified.** Cursor's event names and exit semantics are confirmed from its docs; its stdin field names are not. The Codex template is a best guess — no Codex host has ever been available to this project ([#37](https://github.com/tosin2013/repo-governor/issues/37), [#42](https://github.com/tosin2013/repo-governor/issues/42)). Both templates say so in their own `$comment`.
- **A fourth surface to keep consistent.** `SKILL.md`, `AGENTS.md`, the engine and now the hook can drift apart. `conformance/hooks.py` is the mitigation and is mutation-tested.
- **Session state on disk.** `.repo-governor/sessions/` is per-machine, per-conversation, and gitignored. It is a cache, not evidence; the decision log (ADR-009/ADR-019) remains the record.

**Neutral**

- Reversible. Deleting the config files removes the surface; the skill and engine are untouched.

## What this deliberately does not do

**Scope enforcement by file path was designed, built, and removed.** `engine/envelope.py` can classify a target against `in_scope`/`non_goals`, so checking a pending write against the compiled envelope looked correct. Compiling a real envelope for a real GitHub issue returns:

```
in_scope: []    non_goals: []    required_outcome: null
```

GitHub issues declare no structured file scope. Every path would classify as *"outside declared in_scope"* — a confident refusal with a fabricated reason, on every write. In blocking mode it would have blocked all editing while stating something untrue.

**A governance tool that refuses confidently for the wrong reason is worse than one that stays quiet.** Path-level scope needs providers that declare paths; it is not available from the roadmap providers bound today, and asserting it would be exactly the "information acquires authority by existing" error the whole project refuses. `conformance/hooks.py` now asserts the hook makes no scope claim.

## Related Specification Sections

§54 Failure Conditions · §63 MVP Requirements · §64 MVP Non-Commitments · §65 Future Candidate Capabilities · §40 Completion

## Domain References

- [Hooks reference — Claude Code Docs](https://code.claude.com/docs/en/hooks)
- [Hooks — Cursor Docs](https://cursor.com/docs/hooks)
- [@skills: Attention Is All You Have (arXiv 2608.12610)](https://arxiv.org/abs/2608.12610)
- [`docs/research/2026-08-19-activation-and-hooks.md`](../research/2026-08-19-activation-and-hooks.md)

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map)._
