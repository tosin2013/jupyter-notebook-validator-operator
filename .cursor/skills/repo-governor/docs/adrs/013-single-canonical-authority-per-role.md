# 13. Single Canonical Authority per Role, with Halt on Conflict

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Provider abstraction / conflict resolution

## Context

§20 requires detecting ambiguous authority and gives the worked case: Linear and GitHub Projects both present, both containing roadmap-capable objects. The expected result is `PROVIDER_CONFLICT`, roadmap authority `UNRESOLVED`, and a requirement to "select one canonical roadmap authority provider." §60's complex fixture makes this an acceptance test, and §61's implementation gate requires provider conflicts to be "handled deterministically."

Two questions remain open. Can a role have multiple providers at all? And what happens when providers that *are* legitimately plural disagree?

PRD v0.2 was not uniform on plurality. Its §21 manifest sketch shows `architecture`, `change_signals`, and `retirement` as YAML lists while `roadmap_authority` and `execution` are single values. That distinction is correct and worth making explicit rather than leaving as an artifact of the example.

## Decision

**Roles that answer "is this authorized?" are single-valued. Roles that supply accumulating evidence are multi-valued. Conflict in a single-valued role halts; conflict in a multi-valued role is recorded.**

| Role | Cardinality | Rationale |
| --- | --- | --- |
| `roadmap_authority` | exactly one | Two sources of authorization is no source of authorization |
| `execution` | zero or one | Must map to exactly one execution root per authority item |
| `repository` | exactly one | One repository, one Git |
| `architecture` | zero or many | Constraints accumulate; ADRs and specs coexist legitimately |
| `change_signals` | zero or many | Renovate and an advisory feed are complementary |
| `retirement` | zero or many | More evidence strictly improves obligation checks |
| `decision_history` | one or many | Built-in log (ADR-009) plus optional external |

**Rules**

1. **No ranking, no precedence, no tie-break for single-valued roles.** Detection reporting two roadmap candidates emits `PROVIDER_CONFLICT` and onboarding halts. The engine must not prefer the more recently modified, the more capable, or the more specific. Any automatic resolution is Repo Governor deciding who holds authority, which is precisely the thing it exists not to do (INV-013).

2. **Halt is scoped, not global.** An unresolved roadmap authority blocks any disposition requiring authorization — `EXECUTE`, `CONTINUE` — and yields `AUTHORITY_SOURCE_MISSING` or `CONFLICT`. It does not block onboarding of the other roles, and it does not block `CAPTURE_ONLY` for discoveries. Governance degrades in the affected dimension rather than failing whole.

3. **Multi-valued roles union their evidence, and disagreement is a finding.** Two architecture providers with contradicting constraints produce `ARCHITECTURE_REVIEW`, both constraints cited. The engine does not pick. This is INV-004 and INV-008 working as intended: contradiction between an accepted ADR and a current spec is real information a human should see, not noise to be resolved silently.

4. **Superseded resolution happens inside a provider, not across them.** INV-008's "superseded decisions do not constrain current work" is a within-provider lineage question — ADR-0012 superseded by ADR-0031 in the same directory. Cross-provider supersession is not modelled at v1; if an ADR and an OpenSpec change conflict, that is `ARCHITECTURE_REVIEW`, not an inferred lineage.

5. **§38 is not a conflict.** Roadmap says cancelled, execution says ready — this is a *resolved* case with a defined answer: `AUTHORITY_WITHDRAWN`. Roadmap admission governs execution authorization. `CONFLICT` is reserved for disagreement between peers of the same role, never between different roles where precedence is already defined.

## Consequences

**Positive**

- Makes §61's "provider conflicts handled deterministically" true by construction: halting is deterministic in a way that heuristic resolution never is.
- The cardinality table resolves the §21 inconsistency and gives the manifest schema (ADR-004) a clear rule.
- Scoped halt avoids the failure mode where one unresolved binding renders the whole skill unusable.

**Negative**

- Teams genuinely run two roadmap systems — a product tracker and an engineering tracker — and this design forces them to name one as authoritative. That may be an unwelcome organizational conversation, and some will decline and abandon the tool. It is the right constraint anyway: a team that cannot say where authorization lives does not have governable authorization, and pretending otherwise would produce confident wrong answers.
- Halting on conflict is friction at exactly the moment a user is trying to get started, and §60 makes it an MVP acceptance requirement. The error message and the resolution path have to be excellent, since this is a first-run experience for complex repositories.
- Multi-valued architecture providers make `ARCHITECTURE_REVIEW` likelier, brushing against §54's over-escalation failure condition. Only real use will show whether contradictions are common enough to be a problem.

## Domain Considerations

The unranked-halt rule is the load-bearing part. Every plausible tie-break — most recently updated, most complete API, most items — is a heuristic that would silently confer authority, and §54 names that as a failure condition in as many words. Halting is less convenient and is the only option consistent with the thesis.

Worth noting for later: an organization-level policy pack (§65) could pre-declare "Linear is always roadmap authority," which resolves the conflict by *prior human decision* rather than by inference. That is compatible with this ADR — the decision still comes from a human, just earlier — and is the right shape for a future extension.

## Implementation Plan

1. Encode cardinality per role in the manifest schema; reject a list where a scalar is required at load time.
2. Implement `PROVIDER_CONFLICT` detection for single-valued roles with both candidates and their evidence cited.
3. Implement scoped degradation: which dispositions remain reachable when a given role is unresolved.
4. Implement evidence union and contradiction detection for multi-valued roles.
5. Build §60's complex fixture as an acceptance test — conflict detected, onboarding halted, and after explicit selection, `READY_FOR_GOVERNANCE`.
6. Write the conflict-resolution guidance a user sees on halt, and test it on someone who has not read the specification.

## Related Specification Sections

§20 Provider Conflict Handling · §21 Repository Governance Manifest · §38 Authority vs Execution Conflict · §41–42 Dispositions · §54 Failure Conditions · §60 Onboarding Fixture C · §61 Implementation Gate · INV-004, INV-008, INV-013

## Domain References

- §60 Onboarding Fixture C — Complex Repository
- `docs/research/2026-08-17-external-landscape.md` §4

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
