# 3. Seven Provider Roles with Normalized Contracts

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Amended by**: [ADR-020](020-agent-supplied-transport-with-adapter-as-normalizer.md) — the adapter protocol gains an optional raw-input source (`--input -`). Role contracts and the typed-fact vocabulary are unchanged.
**Date**: 2026-08-17
**Domain**: Provider abstraction

## Context

§10 names seven provider categories. PRD v0.2 did not settle three questions that block implementation: whether seven roles is the right decomposition, what the wire contract between engine and provider looks like, and how a provider that only partially satisfies a role is handled.

The decomposition is defensible because each role answers a *different governance question*, and §6's invariants depend on those questions staying separate:

| Role | Question | Invariant it protects |
| --- | --- | --- |
| `RoadmapAuthorityProvider` | Is this work admitted and still authorized? | INV-002 |
| `ArchitectureEvidenceProvider` | What constrains how it must be built? | INV-004, INV-008 |
| `ExecutionStateProvider` | What is the state of work beneath the authority? | INV-005 |
| `RepositoryEvidenceProvider` | What is actually in the repository? | INV-003 |
| `ChangeSignalProvider` | What changed outside the repository? | INV-006 |
| `RetirementEvidenceProvider` | What obligations block removal? | INV-007 |
| `DecisionHistoryProvider` | What was already decided about this? | INV-005, INV-008 |

Collapsing any two would let one system's state silently answer the other's question, which is the exact failure §5 exists to prevent. A tracker holding both roadmap items and agent tasks is the common case, and the role split is what stops "task is READY" from implying "work is authorized."

Landscape research found no open-source unified abstraction over Jira / Linear / GitHub Projects. Existing solutions are point-to-point synchronizers that copy state rather than normalize it. One adjacent effort matters: Beads issue #1150 requests plugin-based tracker integrations. If delivered, Beads federates trackers — which makes it a *better* `ExecutionStateProvider`, not a competitor, because it aggregates state without ruling on authority.

## Decision

**Seven provider roles, each with a versioned contract, addressed through a uniform adapter protocol.**

1. **One role per governance question.** A single external system may fill multiple roles, but each binding is declared separately in the manifest (ADR-004) and evaluated independently. GitHub may be `RoadmapAuthorityProvider` and `RepositoryEvidenceProvider` simultaneously; the engine treats those as two providers that happen to share a backend.

2. **Adapters are subprocesses speaking JSON over stdout.** The engine invokes an adapter with a typed request on stdin and reads a typed response on stdout. Contract:

   ```text
   $ADAPTER describe                     → capability + contract-version manifest
   $ADAPTER query <role> <function> --json  → typed response or typed error
   ```

   This keeps adapters language-agnostic. **How an adapter obtains state — HTTP, a CLI, a file, an MCP server — is behind this boundary and invisible to the engine (ADR-016).** A Go adapter for Beads, a Python adapter for ADRs, and a shell adapter for Git all satisfy the same protocol. It also matches ADR-011's stdlib-only constraint: the engine never imports a provider SDK.

3. **`describe` separates capabilities from properties.** *(Added 2026-08-17 from implementation.)* `capabilities` are claims a conformance probe MUST be able to exercise; `properties` are declarative traits no probe can reach — `persistence`, `provenance_quality`. The first Layer 1 run failed on exactly this: `persistence: true` was advertised as a capability with no possible probe, which makes an honest-advertisement check vacuous. Untestable claims now have their own field.

4. **Capability advertisement is mandatory and honest.** Per §45, every adapter's `describe` output declares contract version, supported capabilities, persistence semantics, permissions, failure behavior, and provenance quality. Partial implementations are legal and normal — an adapter that cannot answer `get_non_goals` says so, and the engine records the gap as an unknown rather than treating absence as an empty set.

5. **Missing role ≠ error.** Only `RepositoryEvidenceProvider` is always required. `RoadmapAuthorityProvider` is required for any `EXECUTE` disposition. The rest are optional and profile-dependent (ADR-006). An L1 repository with Git and a manual roadmap file is a complete, valid configuration.

6. **Absence and unknown are distinct.** "No `ExecutionStateProvider` is bound" is a configuration fact. "The bound provider could not answer" is an operational fact producing `UNKNOWN` (INV-012). They must never collapse into the same code path.

## Consequences

**Positive**

- The role split is what makes INV-002 mechanically enforceable rather than aspirational.
- Subprocess adapters mean third parties can write providers in any language without the core taking a dependency, which is the practical form of §54's "must not require a specific tracker."
- Honest capability advertisement turns partial support into recorded uncertainty instead of silent wrong answers.

**Negative**

- Seven roles is a lot of surface for an MVP. §62 already concedes this by suggesting synthetic fixtures for several roles; ADR-008 defines what "synthetic" must still prove.
- Subprocess-per-query is slow and chatty. Acceptable at one-evaluation-per-repository scale; would need batching if the surface ever moves into a hot loop.
- Semantic normalization across trackers is the genuine hard part and this ADR does not solve it — it only isolates it into adapters. §55 correctly lists "cross-provider semantic normalization is not reliable" as a stop condition. ADR-008's cross-provider equivalence fixtures are the test that will surface this early.

**Neutral**

- Beads #1150 is tracked as a live stop-condition input, not a threat. Revisit if Beads ships tracker federation *and* authority semantics.

## Domain Considerations

Provider content is untrusted input, not just data — roadmap descriptions and ADR bodies are attacker-writable prose that flows toward an agent's context. ADR-012 sets the trust boundary; adapters are where it is enforced.

Provenance quality is a first-class capability field because §53's portability metric is meaningless without it: two providers can agree on a disposition while one has an auditable citation and the other has a guess.

## Implementation Plan

1. Write JSON Schemas for the seven role contracts, versioned independently, in `references/providers.md` plus `schemas/`.
2. Implement the adapter subprocess protocol and a reference `git` adapter.
3. Implement adapters for the §62 minimum set: Git, manual/file roadmap, ADR directory, synthetic execution.
4. Implement one *second* provider in one category (GitHub Projects as a second roadmap authority) to prove portability, per §62.
5. Build the capability-advertisement validator: reject an adapter whose `describe` claims a capability its `query` cannot serve.

## Related Specification Sections

§10–17 Provider Model and roles · §45–50 Conformance · §54 Failure Conditions · §55 Stop Conditions · §62 Initial Implementation Boundary

## Domain References

- [Beads issue #1150 — plugin-based tracker integrations](https://github.com/gastownhall/beads/issues/1150)
- [steveyegge/beads — DeepWiki](https://deepwiki.com/steveyegge/beads)
- [Issue Trackers as AI Agent Infrastructure](https://www.mindstudio.ai/blog/issue-trackers-ai-agent-infrastructure-jira-linear)
- `docs/research/2026-08-17-external-landscape.md` §4

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
