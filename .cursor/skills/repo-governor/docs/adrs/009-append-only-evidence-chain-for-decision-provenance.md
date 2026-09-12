# 9. Append-Only Evidence Chain for Decision Provenance

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md). · **Amended by [ADR-019](019-database-backed-decision-history.md), 2026-08-17**
**Date**: 2026-08-17
**Domain**: Observability & provenance

> **Amended by ADR-019.** This ADR specified JSON files under `.repo-governor/decisions/`, hash-chained by hand. It was never implemented that way. `decision_history` is now backed by a database — Dolt by default — because the role is append-only, unbounded, and queried by relationship, and because Dolt supplies the evidence chain natively via `dolt_log` and `dolt_history_decisions`. **The hand-specified hash chaining below is superseded, not reimplemented.**
>
> Two claims in this ADR no longer hold and are corrected rather than quietly contradicted:
>
> * *"enforceable with zero external dependencies"* — binding this role now requires installing a database. Defensible, because §54 prohibits requiring a specific third-party **tracker** for a role and `decision_history` is optional per profile, so `GOVERNOR_GREENFIELD` and `GOVERNOR_LITE` install nothing. But it is a real change to a stated rationale.
> * *"committed by default, gitignorable by choice"* — Dolt stores binary data and has its own remotes, so the store is gitignored. The distribution and pull-request-review benefits this ADR wanted from committing the log are given up.
>
> The redaction question this ADR left open is settled in ADR-019: hash plus typed facts plus explicit redaction markers, closing #4.

## Context

§7 goal 13 requires preserving "provenance for every material decision." §43 specifies the required governance output and §52 the minimum traceable event. What neither settles is where decision records live, whether they are durable, and whether they feed back into future evaluations.

This matters more than ordinary logging because of INV-005 and INV-008. Resolving "was this deferred before?" (§39) requires reading prior decisions. If decision history is only available through an external `DecisionHistoryProvider`, then repositories without one cannot enforce INV-005 at all — and §54 forbids requiring any specific external tool.

Prior art is directly applicable. OpenKedge (arXiv 2604.08601) records every stage of the mutation lifecycle in an **Intent-to-Execution Evidence Chain** as cryptographically linked lineage, producing "a verifiable decision trace" that makes mutation "deterministic, auditable, and replayable." The guardrail literature adds the operational argument: tamper-evident logging is standard remediation, and 58% of organizations reporting agent incidents needed five hours or more to detect and respond.

## Decision

**Every material evaluation appends a decision record to a repository-local, append-only log. The log is the built-in `DecisionHistoryProvider` and requires no external system.**

1. **Location and format.** `.repo-governor/decisions/` holds one JSON file per decision, named by timestamp and content hash. JSON, not YAML, because these are machine-written and machine-read; the human-facing rendering is generated.

2. **Chained records.** Each record carries the hash of its predecessor, so removal or edit of a historical decision is detectable. Not cryptographically signed at v1 — OpenKedge's ephemeral-identity enforcement is beyond scope for a repository skill — but tamper-*evident*, which is what the audit claim actually needs.

3. **Record contents** are §43's output plus what replay requires:

   ```yaml
   decision_id:
   timestamp:
   engine_version:              # replay needs this
   manifest_hash:               # which bindings were in force
   provider_snapshots:          # exact adapter responses consumed
   condition_override_active:   # per ADR-006
   decision:                    # the §43 governance output
   previous_decision_hash:
   ```

   Storing provider snapshots is what makes ADR-002's determinism useful: a decision can be re-run and the output must match byte for byte.

4. **Committed by default, gitignorable by choice.** Committing gives the log distribution, review, and durability for free, and makes decisions visible in pull requests. Repositories that find the churn unacceptable may gitignore the directory, accepting that INV-005 enforcement degrades to whatever an external `DecisionHistoryProvider` offers.

5. **The built-in provider is a floor, not a ceiling.** A configured external `DecisionHistoryProvider` is consulted *in addition*. Where they disagree, the disagreement is a `CONFLICT`, not a silent precedence rule.

6. **Deferral and rejection are decisions.** §39's rediscovery case only works if `DEFERRED` and its reversal condition were recorded when decided. `CAPTURE_ONLY` and every review disposition append records, not just `EXECUTE`.

7. **Discovery capture is separate.** Discoveries live in `.repo-governor/discoveries/` with their own lifecycle. Decisions reference them by ID. Conflating them would make the decision log mutable, which defeats its purpose.

## Consequences

**Positive**

- INV-005 and INV-008 become enforceable with zero external dependencies, which is what §54's "must not require ADRs / Beads / a tracker" demands.
- Replayability makes ADR-002's determinism claim auditable rather than asserted.
- A committed, human-readable log is a strong artifact for a public governance project — the decision trail is the product demonstrating itself.
- §53's "discovery preservation 100%" target gets a concrete storage mechanism.

**Negative**

- Repository churn. A decision per material evaluation across several agents produces real commit volume and pull-request noise. Mitigation — record only *material* evaluations, batch within a session, and generate a rolled-up summary — but this remains the most likely reason a team disables the feature.
- Cross-repository leakage risk. Decision records embed provider snapshots, which may contain roadmap item titles and internal project detail. In a public repository, committing them publishes that content. §51 requires preventing cross-repository state leakage; snapshot redaction must be configurable, and the default for public repositories needs deliberate thought before v1.
- Unbounded growth. Requires compaction, which introduces the question of what may be compacted without breaking the chain.
- Hash chaining without signing stops accident and casual edit, not a determined actor with commit access. The claim must be stated as tamper-evident, never tamper-proof.

## Domain Considerations

The redaction question is the sharpest open issue in this ADR and interacts with ADR-012. Provider snapshots are the highest-fidelity input to replay and the highest-risk content to commit. A reasonable default: store full snapshots for private repositories, store hashes plus typed facts for public ones, and make replay in the public case verify the hash rather than reproduce the content.

## Implementation Plan

1. Define the decision record JSON Schema; include `engine_version` and `manifest_hash` from the first version.
2. Implement the append-and-chain writer with hash verification on read.
3. Implement the built-in `DecisionHistoryProvider` reading the log; it must pass ADR-008 Layer 1 conformance like any other adapter.
4. Implement `repo-governor replay <decision_id>` and assert byte-identical reproduction in CI.
5. Design snapshot redaction and set the public-repository default before v1.
6. Define compaction: what may be collapsed, and how the chain survives it.

## Related Specification Sections

§7 Product Goals (13) · §17 DecisionHistoryProvider · §32 Discovery Model · §39 Rediscovered Work · §43 Required Governance Output · §51 Security and Boundary Model · §52 Observability · §53 Success Metrics · INV-005, INV-008

## Domain References

- [OpenKedge: Governing Agentic Mutation with Execution-Bound Safety and Evidence Chains — arXiv:2604.08601](https://arxiv.org/abs/2604.08601)
- [Decision Evidence Maturity Model for Agentic AI — arXiv:2605.04093](https://arxiv.org/pdf/2605.04093)
- [AI Agent Risks & Guardrails: 2026 Enterprise Security Guide](https://atlan.com/know/ai-agent-risks-guardrails/)
- `docs/research/2026-08-17-external-landscape.md` §3, §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
