# Repo Governor — Normative Reference

This directory holds the load-bearing specification content, extracted from **PRD v0.2** on 2026-08-17 and reorganized by topic. `PRD.md` was deleted after extraction; nothing normative was lost.

**Original § numbering is preserved throughout**, so every `§NN` and `INV-NNN` citation in the ADRs still resolves. Use the map below.

The ADRs in [`../adrs/`](../adrs/) are the *decisions*. These files are the *specification those decisions act on*. Where they disagree, the ADR is newer and wins — such points are annotated inline.

## Files

| File | Covers | Normative? |
| --- | --- | --- |
| [invariants.md](invariants.md) | §5–§6 — INV-001 … INV-014 | ✅ Yes |
| [product-scope.md](product-scope.md) | §1–§4, §7–§9, §66–§70 — definition, problem, thesis, goals, non-goals, positioning, ownership | Partly — §8 non-goals are binding |
| [provider-roles.md](provider-roles.md) | §10–§17, §45–§50 — the eight roles and their conformance minimums | ✅ Yes |
| [onboarding.md](onboarding.md) | §18–§22, §57–§61 — onboarding, detection, conflict, manifest, permissions, simulation, gate | ✅ Yes |
| [repository-conditions.md](repository-conditions.md) | §23–§29 — L0–L4, profiles, architecture budget | ✅ Yes |
| [domain-model.md](domain-model.md) | §30–§32, §43–§44, §52 — objects, ScopeEnvelope, discovery, output contract, observability | ✅ Yes |
| [lifecycles.md](lifecycles.md) | §33–§40 — admission, maintenance, retirement, resolution, firewall | ✅ Yes |
| [dispositions.md](dispositions.md) | §41–§42 — governance and onboarding vocabularies | ✅ Yes |
| [criteria.md](criteria.md) | §51, §53–§56, §62–§65 — security, metrics, failure, stop conditions, MVP | ✅ Yes |

## Section map

| § | Topic | File |
| --- | --- | --- |
| §1–§4 | Summary, definition, problem, thesis | product-scope.md |
| §5–§6 | Principle, INV-001 … INV-014 | **invariants.md** |
| §7–§9 | Goals, non-goals, architecture | product-scope.md |
| §10–§17 | Provider model and the eight roles | provider-roles.md |
| §18–§22 | Onboarding, detection, conflict, manifest, permissions | onboarding.md |
| §23–§29 | Repository conditions L0–L4, architecture budget | repository-conditions.md |
| §30–§32 | Domain objects, ScopeEnvelope, discovery model | domain-model.md |
| §33–§40 | Lifecycles, architecture resolution, conflict, firewall | lifecycles.md |
| §41–§42 | Governance and onboarding dispositions | dispositions.md |
| §43–§44 | Required output, example evaluation | domain-model.md |
| §45–§50 | Provider conformance minimums | provider-roles.md |
| §51 | Security and boundary model | criteria.md |
| §52 | Observability | domain-model.md |
| §53–§56 | Metrics, failure conditions, stop conditions, research evidence | criteria.md |
| §57–§61 | Onboarding simulation, fixtures A–C, implementation gate | onboarding.md |
| §62–§65 | Implementation boundary, MVP scope, non-commitments, future candidates | criteria.md |
| §66–§70 | Positioning, naming, ownership, state, final principle | product-scope.md |

## What the ADRs changed

The extraction is faithful, with four deliberate edits where an ADR supersedes the PRD:

1. **§51 gained a bullet** — "treat all provider content as untrusted input" (ADR-012). The original security model did not address prompt injection.
2. **§21's manifest gained fields** — `engine_min_version`, per-binding `contract_version`, explicit `adapter` paths (ADR-004), and its list-vs-scalar inconsistency is resolved by ADR-013's cardinality rule.
3. **§55 gained live stop-condition inputs** — Beads #1150, the Layer 2 normalization experiment, and envelope thinness are tracked concretely rather than abstractly.
4. **§65 lost "agent skill"** — promoted to the primary delivery surface by ADR-001; MCP, CLI, and CI enforcement stay deferred.

Two sections are now historical rather than forward-looking: §56 records that the external landscape sweep was completed and its differentiation candidate confirmed, and §69's product state reads *PRD v0.2 superseded by ADR-001…ADR-014*.

## Provenance

- Source: `PRD.md` v0.2 (provider-oriented draft), 37,366 bytes, deleted 2026-08-17 after extraction.
- Research informing the ADRs: [`../research/2026-08-17-external-landscape.md`](../research/2026-08-17-external-landscape.md)
- Per §68, Decision Crafters retains research provenance for this material.
