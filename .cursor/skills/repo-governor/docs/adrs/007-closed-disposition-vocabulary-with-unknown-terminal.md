# 7. Closed Disposition Vocabulary with UNKNOWN as a Terminal Outcome

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Policy engine / output contract

## Context

§41 defines twelve governance dispositions in three groups, and §42 adds seven onboarding dispositions. INV-012 requires `UNKNOWN` to be a valid governance outcome. §54 sets the countervailing constraint: the product fails if "it turns all discoveries into human review."

Two design questions are unresolved. First, whether the vocabulary is closed — can a profile or provider introduce a new disposition? Second, what `UNKNOWN` actually means to the agent that receives it, since an outcome the agent does not know how to act on is functionally a crash.

The guardrail literature supplies the framing: separate verdict from action, and set a confidence threshold below which the agent escalates. `UNKNOWN` is that threshold made explicit rather than left to the model's discretion.

## Decision

**The disposition vocabulary is closed at twenty values — twelve governance, eight onboarding. Every evaluation terminates in exactly one, and `UNKNOWN` is a legitimate terminal state carrying required diagnostic payload.**

> *Amended 2026-08-17.* Originally nineteen. Implementing gate 7 showed §42 had no value for *detection complete, proposal written, awaiting acceptance* — precisely where ADR-010's mandatory human step sits — so `PROPOSAL_READY` was added. The count changed because the spec had a hole, not because the set is open.

1. **Closed set, engine-owned.** Profiles select which dispositions are *reachable*; they cannot define new ones. Providers never emit dispositions — they emit typed state, and the engine rules. This is what makes cross-provider portability (§53) testable: two providers can be compared only if the output alphabet is fixed.

2. **Exactly one disposition per evaluation.** The top-level decision is singular. Per-discovery dispositions nest beneath it, as in §44's worked example where the top level is `CONTINUE` and `DISC-88` carries its own `CAPTURE_ONLY`.

3. **`UNKNOWN` must be actionable.** An `UNKNOWN` that says only "unknown" is a defect. Every `UNKNOWN` carries:

   ```yaml
   decision: UNKNOWN
   unknowns:
     - dimension: authority          # which of the seven questions failed
       reason: PROVIDER_UNREACHABLE  # typed, enumerated
       provider: linear
       resolution: |                 # what a human can do about it
         Verify LINEAR_API_KEY is set, or bind a manual roadmap provider.
       blocking: true                # does this prevent EXECUTE?
   ```

   The engine distinguishes *unresolvable* (evidence genuinely absent) from *unavailable* (provider down). Both yield `UNKNOWN`; the resolution differs, and conflating them wastes human attention.

4. **Non-blocking unknowns do not force review.** This is the direct answer to §54's over-escalation failure condition. An unresolved retirement signal on a module unrelated to the authorized scope is recorded as a non-blocking unknown and does not gate `CONTINUE`. Only unknowns on the evaluation's critical path block.

5. **`STOP_COMPLETE` is terminal and non-overridable.** When acceptance conditions are satisfied, no discovery, no matter how compelling, converts to `EXECUTE` within the same authorization (INV-009, §40). Discoveries are enumerated with `CAPTURE_ONLY` dispositions and the evaluation ends.

6. **Onboarding dispositions are a separate alphabet.** §42's seven values belong to a different state machine with a different consumer (a human running onboarding, not an agent mid-task). They never appear in a governance decision, and the two sets never mix.

## Consequences

**Positive**

- A closed alphabet makes ADR-008's golden fixtures possible: expected outputs are enumerable.
- Structured, typed unknowns turn INV-012 from a philosophical stance into a usable protocol — the agent can branch on `blocking`, and the human gets a resolution path.
- The blocking/non-blocking split is the specific mechanism that keeps INV-012 from colliding with §54.

**Negative**

- Twenty dispositions is a large surface for agents to handle correctly. Real-world agent behavior will collapse them — most agents will treat every non-`EXECUTE` value as "stop and ask." That erodes the vocabulary's value in practice, and only real-world observation will show how badly.
- Closing the set means an unanticipated governance situation has no precise expression and must be forced into `UNKNOWN` or `CONFLICT`. This is deliberate: an extensible vocabulary would destroy portability, which is a core thesis claim. Extension requires a version bump and a deliberate decision.
- Calibrating blocking vs non-blocking is judgment encoded as policy, and getting it wrong in either direction hits a named failure condition — too blocking hits §54's over-escalation, too permissive hits §53's unauthorized-execution target of zero.

## Domain Considerations

Cedar's default-deny discipline applies to the tie-break rule: where the engine cannot determine whether a disposition should be `EXECUTE` or something more conservative, it resolves conservatively. The asymmetry is real — a wrongly-blocked change costs a human a minute, a wrongly-permitted deletion can cost a release.

The `CONFLICT` disposition deserves emphasis because §38's authority-versus-execution case is the single most likely real-world trigger: a cancelled roadmap item whose execution tracker still says `READY`. The engine must return `AUTHORITY_WITHDRAWN` there, not `CONFLICT` — roadmap admission status governs (§38). `CONFLICT` is reserved for genuine ambiguity between peers, such as two bound roadmap authorities disagreeing (ADR-013).

## Implementation Plan

1. Define the nineteen dispositions as a closed enum with documented semantics in `references/dispositions.md`.
2. Define the typed `unknown` reason enum; keep it closed for the same reason as the disposition set.
3. Implement the blocking/non-blocking classifier as an explicit per-dimension policy, profile-configurable.
4. Write a decision-table test: every reachable combination of authority × architecture × execution state maps to exactly one expected disposition.
5. Publish an agent-side handling guide in `SKILL.md` — what to do on receiving each disposition — since a vocabulary nobody handles correctly is decoration.

## Related Specification Sections

§38 Authority vs Execution Conflict · §40 Completion Firewall · §41 Governance Dispositions · §42 Onboarding Dispositions · §43 Required Governance Output · §53 Success Metrics · §54 Failure Conditions · INV-009, INV-012

## Domain References

- [Guardrails for Autonomous AI Agents: Production Safety 2026](https://khimananda.com/blog/guardrails-for-autonomous-ai-agents)
- [Balancing speed and safety: A control framework for AI coding agents — AWS](https://aws.amazon.com/blogs/security/balancing-speed-and-safety-a-control-framework-for-ai-coding-agents/)
- `docs/research/2026-08-17-external-landscape.md` §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
