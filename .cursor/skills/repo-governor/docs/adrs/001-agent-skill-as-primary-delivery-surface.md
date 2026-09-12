# 1. Agent Skill as the Primary Delivery Surface

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Distribution & agent integration
**Amended by**: [ADR-029](029-hooks-as-deterministic-delivery-surface.md) — "coding-agent hooks" is promoted from a deferred §65 candidate to a secondary delivery surface. The Agent Skill remains primary; the negative consequence below was measured, and the vendor-bet reasoning that deferred hooks expired when all three target hosts converged on one hook convention.

## Context

§1 defines Repo Governor as a "tool-independent governance skill for AI-assisted software development," and §54 makes it a failure condition if the product "requires a specific tracker." Tool independence therefore applies to the *agent host* as much as to the provider systems: a governance layer that only works inside one vendor's coding agent has not achieved the stated thesis.

PRD v0.2 did not commit to a delivery form. §65 lists "MCP interface," "coding-agent hooks," "IDE integration," and "CI enforcement" as undifferentiated future candidates. Something has to be primary.

External research (2026-08-17) changed the calculus. Anthropic released **Agent Skills as an open standard on 2025-12-18**, and it has since been adopted by 26+ platforms including Claude, OpenAI Codex, Gemini CLI, Cursor, and VS Code. A `SKILL.md` plus supporting files is now a portable artifact across most of the coding-agent market, with no per-vendor integration work.

Its three-tier progressive disclosure model is a direct structural match for §23–28's requirement that governance depth scale with repository condition:

| Tier | Loaded when | Repo Governor content |
| --- | --- | --- |
| 1 | Always (~100 tokens) | `name` + `description` — the skill announces itself |
| 2 | On activation (< 5k tokens) | Core invariants, disposition vocabulary, evaluation entry point |
| 3 | On demand | Profile policy packs, provider contracts, lifecycle state machines, conformance fixtures |

An L1 repository never pays for L4 policy detail. This is progressive disclosure and progressive governance solving the same problem with the same mechanism.

## Decision

**Repo Governor ships primarily as an Agent Skill conforming to the Agent Skills open standard.** The repository root artifact is `SKILL.md` with a supporting `references/`, `policies/`, and `scripts/` tree.

Layout:

```text
SKILL.md                      # tier 2 — invariants, dispositions, entry point
references/
  providers.md                # provider contracts (tier 3)
  lifecycles.md               # admission / maintenance / retirement state machines
  dispositions.md             # full disposition semantics
policies/
  greenfield.json             # GOVERNOR_GREENFIELD
  lite.json                   # GOVERNOR_LITE
  standard.json               # GOVERNOR_STANDARD
  full.json                   # GOVERNOR_FULL
  high-assurance.json         # GOVERNOR_HIGH_ASSURANCE
engine/
  completion.py               # deterministic engine (see ADR-002, ADR-011)
  onboard.py                  # repository attachment and detection
  manifest.py vocabulary.py amendments.py
adapters/                     # 10 provider adapters, any language
conformance/                  # 7 suites — the evidence behind every claim
```

> **Two deviations from this sketch, both recorded when the skill was built 2026-08-17.**
>
> *Policy packs are JSON, not YAML.* ADR-011 leaves the engine with no YAML parser and ADR-015 made JSON canonical after a spike showed a hand-rolled YAML subset silently mis-typed 7 of 10 realistic values. The packs are loaded by `engine/vocabulary.py` at import, so they are live configuration rather than documentation.
>
> *`scripts/` became `engine/` plus `adapters/`.* The sketch assumed two scripts; the implementation is an engine with a separate adapter layer, which ADR-003's subprocess protocol requires. `SKILL.md` references those paths directly.

Secondary surfaces — MCP server, standalone CLI, CI action — are explicitly deferred. They may later wrap the same `scripts/` core, but none is required for MVP (§63) and none may fork the policy logic.

`SKILL.md` must stay under 5,000 tokens. Content that exceeds the budget moves to tier 3 rather than being cut.

## Consequences

**Positive**

- Tool independence at the agent layer is achieved by adopting a standard, not by building N integrations.
- Distribution is `git clone` into a skills directory. No install step, no registry, no hosted service — consistent with §64's non-commitments.
- Progressive disclosure gives the profile model (ADR-006) a native enforcement mechanism: an L1 repository literally never loads L4 policy files.
- The skill is inspectable plain text, which suits a public open-source governance artifact where auditability is the point.

**Negative**

- The 5k-token tier-2 budget is a hard constraint. The full invariant set (INV-001…INV-014), seven provider contracts, three lifecycles, and the disposition vocabulary cannot coexist in the body. Tiering discipline becomes an ongoing maintenance cost.
- Skill activation is model-mediated: the agent decides whether the description matches the task. A governance skill that fails to activate is worse than absent, because the human assumes it ran. This is a real failure mode with no clean fix at the skill layer — mitigated in ADR-002 by making the deterministic script, not the prose, the thing that produces a verdict.

  > **Measured 2026-08-19.** Issue 36 Arm A prompt 1, Claude Code / Opus 4.6: the agent stated the skill's description among its always-applied rules, then read an issue, explored the source and began writing code without consulting governance. Independently, Vercel measured skills uninvoked in 56% of cases with access. This consequence was correct, and ADR-029 adds the deterministic delivery surface it says the skill layer cannot provide.
- No enforcement. A skill advises; it cannot block. Repo Governor returns a disposition and relies on the agent honoring it. Hard enforcement requires the deferred CI surface.
- Cross-platform fidelity varies. 26+ platforms claim support; behavior on tier-3 loading and script execution will differ. Conformance testing (ADR-008) must cover host variation, not just provider variation.

**Neutral**

- Reversible. If skill adoption regresses, the `scripts/` core is independently invocable and the skill becomes one wrapper among several.

## Domain Considerations

The April 2026 incident in which a Claude-powered coding agent deleted a production database after **reasoning past explicit safety rules in its prompt** is the governing cautionary result. It says a skill's Markdown body must not be the sole carrier of a hard rule. Prose in `SKILL.md` orients the agent; `scripts/evaluate.py` produces the verdict. ADR-002 makes this binding.

## Implementation Plan

1. Draft `SKILL.md` frontmatter; token-count the body against the 5k budget.
2. Move every invariant beyond the core four to `references/`.
3. Define the tier-3 loading triggers explicitly in the body (e.g. "for L3+ repositories, read `references/lifecycles.md`").
4. Validate activation reliability: 20 synthetic agent prompts, measure how often the skill is invoked when it should be.
5. Test tier-3 loading and script execution on at least three hosts (Claude Code, Codex, Cursor).

## Related Specification Sections

§1 Product Summary · §23–28 Repository Condition Model · §54 Failure Conditions · §63 MVP Requirements · §64 MVP Non-Commitments · §65 Future Candidate Capabilities

## Domain References

- [Agent Skills — Claude Platform Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
- [The Agent Skills Ecosystem in 2026](https://agentman.ai/blog/agent-skills-ecosystem-report-2026)
- [Agent Skills: Progressive Disclosure as a System Design Pattern](https://www.newsletter.swirlai.com/p/agent-skills-progressive-disclosure)
- `docs/research/2026-08-17-external-landscape.md` §1

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
