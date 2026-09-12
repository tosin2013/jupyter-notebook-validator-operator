# 4. Governance Manifest as the Sole Binding Artifact

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Configuration & state

## Context

§21 proposes `.repo-governor.yaml` and sketches its shape, but calls the filename "provisional" and leaves open what the file *means* — whether it is a cache, a config convenience, or the authoritative record of which systems govern this repository.

That question is load-bearing for INV-013: "Provider detection does not establish provider authority. Only accepted configuration establishes its governance role." If detection results could reach the engine without passing through an accepted artifact, INV-013 is unenforceable. The manifest is where human acceptance is recorded, so it has to be the only path.

§21 also warns the manifest "must not duplicate provider data unnecessarily" — it binds, it does not mirror. A manifest that accumulated cached roadmap items would drift into being the canonical roadmap database that §54 names as a failure condition.

## Decision

**`.repo-governor.json`, committed to the repository, is the single artifact that binds providers to governance roles. Nothing else confers a binding.**

1. **Binding requires presence in the manifest.** A detected, reachable, credentialed provider that is absent from the manifest has no governance role. The engine will not consult it.
2. **The manifest holds bindings and permissions only** — never provider state. No cached issues, no mirrored ADR text, no status snapshots. If the engine needs state, it queries the adapter.
3. **Committed and human-readable.** Governance configuration is reviewed like code, through pull request. This is what "human accepts" in §18's onboarding lifecycle means concretely, and it gives every binding change a reviewer and a timestamp for free.
4. **Schema-versioned.** `repo_governor.version` gates parsing. An engine encountering a version it does not implement refuses to evaluate rather than guessing — consistent with ADR-002's conservative default.
5. **No secrets, ever.** The manifest names a provider type and its coordinates. Credentials come from the environment. This is enforced by validation, not convention: a manifest containing anything resembling a token fails to load (ADR-005).
6. **Absent manifest → `AUTHORITY_SOURCE_MISSING`.** Not an error, not a default configuration. A repository without a manifest is un-onboarded, and the correct response is to run onboarding (ADR-010), not to infer sensible defaults.

Extension to the §21 sketch — three fields the draft omits:

```yaml
repo_governor:
  version: 1
  engine_min_version: 0.1.0     # refuse evaluation below this

providers:
  roadmap_authority:
    type: linear
    project: ENG
    contract_version: 1          # which role contract this adapter satisfies
    adapter: adapters/linear     # explicit path; no implicit resolution
```

`contract_version` lets role contracts evolve independently (ADR-003). `adapter` is explicit because implicit adapter discovery would reintroduce exactly the silent-binding problem INV-013 forbids.

> **Encoding decided by ADR-015.** The canonical file is `.repo-governor.json`, read with `json.loads`. Every rule in this ADR is unchanged; only the encoding is settled. YAML examples below remain illustrative.

## Consequences

**Positive**

- INV-013 becomes structurally true: there is one code path from configuration to evaluation, and it reads this file.
- Git history of the manifest is a governance audit log — who granted which system authority, and when — at no additional cost.
- Keeping state out means the manifest stays small enough to read in a review, which is what makes the review meaningful.

**Negative**

- Manual editing is required to bind a provider. Onboarding proposes; a human commits. This is friction by design, and §54 warns that excessive friction for simple repositories is a failure condition — mitigated by keeping the L0/L1 manifest to a handful of lines (ADR-006).
- A committed manifest is world-readable in a public repository. Project keys and board names leak. Acceptable — these are not secrets — but it must be documented, not discovered.
- Monorepos with per-package governance are unsupported at v1. Deferred deliberately rather than designed speculatively.

## Domain Considerations

The manifest is untrusted input from the engine's perspective in one specific sense: on a pull request from an outside contributor, a modified manifest could rebind roadmap authority to an attacker-controlled adapter. Validation must reject adapter paths outside the repository, and the human review of manifest changes is a real control, not a formality. ADR-012 covers the wider trust boundary.

## Implementation Plan

1. Write the JSON Schema for manifest v1; make the schema the specification, with §21's YAML as illustration.
2. Implement a loader that validates, then fails closed on version mismatch, unknown roles, or secret-shaped values.
3. Implement `repo-governor validate` to check manifest ↔ adapter agreement (does the named adapter actually satisfy the declared `contract_version`?).
4. Define the minimal L0/L1 manifest and hold it to ≤ 15 lines.
5. Add a migration path stub for v1 → v2 before v1 ships, so the first schema change is not an emergency.

## Related Specification Sections

§18 Repository Onboarding · §21 Repository Governance Manifest · §22 Permission Model · §42 Onboarding Dispositions · §54 Failure Conditions

## Domain References

- INV-013, INV-014
- `docs/research/2026-08-17-external-landscape.md` §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
