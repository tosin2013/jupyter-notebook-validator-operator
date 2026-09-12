# 2. Deterministic Policy Engine Separate from Model Judgment

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Policy engine / core architecture

## Context

§53 sets success metrics that only mean something if evaluation is repeatable: authority fidelity 100%, unauthorized execution 0, completion-stop compliance 100%, provider portability "100% across conformance fixtures." A metric like "semantically equivalent provider states produce equivalent governance dispositions" is unmeasurable if the disposition comes from a language model's in-context reasoning, because equivalent inputs will not reliably produce equal outputs.

ADR-001 makes the skill the delivery surface. A skill is Markdown that an agent reads. Left there, every invariant in §6 would be prose the model is asked to honor — and the April 2026 production-database deletion is the direct counter-evidence: that agent **had explicit safety rules and reasoned past them**. Consensus remediation across the 2026 guardrail literature is uniform: move "never do this" rules out of the prompt and into code, and separate verdict from action.

Policy-as-code prior art confirms the shape. AWS Cedar reached GA for agent authorization in Bedrock AgentCore on 2026-03-03, evaluating every request **outside the agent's code, outside the model's reasoning, and immune to prompt injection**, default-deny. OPA fills the same role in cloud-native agent stacks.

## Decision

**The governance disposition is computed by a deterministic program, not inferred by the model.**

Three rules bind:

1. **Pure function.** `evaluate(manifest, provider_state, requested_action) → GovernanceDecision`. Same inputs, same output, always. No network calls, no clock reads, no model calls inside the evaluation path.
2. **Separation of verdict from action.** Repo Governor returns a disposition. It never performs the create, change, or retire operation it is ruling on. The agent acts; the engine only rules.
3. **The model's role is bounded to translation.** The agent may map fuzzy natural-language state into typed provider facts on the way *in*, and may explain the decision in prose on the way *out*. It may not decide. Where translation is uncertain, the provider emits `UNKNOWN` and the engine handles it (INV-012).

Invariants INV-001 through INV-014 are implemented as executable checks with test coverage, not as bullet points the model is asked to remember. The `SKILL.md` prose describing them is documentation of the code, and the code is authoritative on conflict.

We do **not** adopt Cedar or OPA. Both evaluate a `principal → action → resource` tuple. Repo Governor evaluates a multi-provider state reconciliation producing a disposition plus unknowns plus provenance — a different shape, and pulling in a policy runtime would add a hard dependency to an artifact whose main virtue is that it clones and runs.

## Consequences

**Positive**

- §53 metrics become measurable. Portability can be asserted with golden fixtures (ADR-008) rather than argued.
- Prompt injection through provider content cannot flip a disposition, because the disposition is not produced by reading text (see ADR-012).
- Decisions are replayable: stored inputs plus engine version reproduce the exact output, which is what makes the evidence chain in ADR-009 worth keeping.
- Governance logic is unit-testable — unusual for an agent-facing artifact, and a significant credibility asset for a public governance project.

**Negative**

- Real repositories are messier than typed facts. Deterministic evaluation will return `UNKNOWN` more often than a model would, and §54 warns that turning all discoveries into human review is a failure condition. Calibration is a live risk that only real-world use resolves.
- The translation boundary is now the weak point. If the agent mis-maps "Won't Fix" onto `ACTIVE`, the engine faithfully computes the wrong answer. Provider conformance tests (ADR-008) must target this mapping specifically.
- Two artifacts must stay in sync: the prose in `SKILL.md` and the checks in code. Drift is a maintenance tax.

**Neutral**

- The engine is a small, dependency-light program. Scale is one repository per invocation; performance is not a design driver.

## Domain Considerations

Cedar's default-deny posture is worth importing even though the engine is not. Where authority cannot be resolved, the outcome is `NO_EXECUTION_AUTHORITY` or `UNKNOWN` — never `EXECUTE`. §51's "fail conservatively when provider access fails" is the same principle stated for a different failure mode.

The OpenKedge result (arXiv 2604.08601) reinforces the ordering: intents are "evaluated against deterministically derived system state ... prior to execution." Deterministic derivation comes first; the bounded contract is compiled from it.

## Implementation Plan

1. Specify `GovernanceDecision` as a JSON Schema matching §43's required output.
2. Implement INV-001…INV-014 as discrete predicate functions, one test file each.
3. Implement the three lifecycle state machines (§33, §34, §35) as explicit transition tables; reject illegal transitions structurally rather than by check (INV-010).
4. Add a determinism test: evaluate every fixture 100× and assert byte-identical output.
5. Version the engine; stamp the version into every decision record.

## Related Specification Sections

§5 Core Governance Principle · §6 Core Governance Invariants · §33–35 Lifecycles · §41 Governance Dispositions · §43 Required Governance Output · §51 Security and Boundary Model · §53 Success Metrics

## Domain References

- [Why Policy in Amazon Bedrock AgentCore chose Cedar](https://aws.amazon.com/blogs/security/why-policy-in-amazon-bedrock-agentcore-chose-cedar-for-securing-agentic-workflows/)
- [Balancing speed and safety: A control framework for AI coding agents — AWS](https://aws.amazon.com/blogs/security/balancing-speed-and-safety-a-control-framework-for-ai-coding-agents/)
- [Why Open Policy Agent is the Missing Guardrail for Your AI Agents](https://codilime.com/blog/why-use-open-policy-agent-for-your-ai-agents/)
- [OpenKedge — arXiv:2604.08601](https://arxiv.org/abs/2604.08601)
- `docs/research/2026-08-17-external-landscape.md` §2, §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
