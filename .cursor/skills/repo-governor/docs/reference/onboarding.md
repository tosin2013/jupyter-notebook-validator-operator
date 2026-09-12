# Onboarding, Manifest & Implementation Gate

> Extracted from PRD v0.2 §18–§22 and §57–§61 on 2026-08-17. Original section numbering preserved.
> **Normative.** §58–§60 are the acceptance fixtures for `RG-SIM-ONBOARDING-v0.1`, the gate blocking `IMPLEMENTATION_READY`.

---

## §18 — Repository Onboarding

Repository onboarding is a first-class product capability. Conceptual command: `repo-governor init`

```text
ATTACH REPOSITORY
        ↓
INSPECT
        ↓
ASSESS REPOSITORY CONDITION
        ↓
DETECT POSSIBLE PROVIDERS
        ↓
PROPOSE PROVIDER ROLES
        ↓
HUMAN ACCEPTS / MODIFIES
        ↓
WRITE GOVERNANCE MANIFEST
        ↓
VALIDATE PROVIDERS
        ↓
READY_FOR_GOVERNANCE
```

## §19 — Provider Detection

Repo Governor may detect possible provider implementations:

```text
.beeds/                                 → candidate ExecutionStateProvider
docs/adr/                               → candidate ArchitectureEvidenceProvider
renovate.json                           → candidate ChangeSignalProvider
GitHub repository + Projects metadata   → candidate RoadmapAuthorityProvider
```

Detection output must use `PROVIDER_DETECTED` or `PROVIDER_UNCONFIRMED`. **It must not silently assign authority** (INV-013).

> ADR-010 makes this structural: detection writes `.repo-governor.proposed.yaml`, which the engine never reads.

## §20 — Provider Conflict Handling

Repo Governor must detect ambiguous authority:

```text
Linear detected.
GitHub Projects detected.
Both contain roadmap-capable objects.
```

Result:

```text
PROVIDER_CONFLICT

Roadmap authority: UNRESOLVED

Required: Select one canonical roadmap authority provider.
```

No execution should begin until the conflict is resolved where roadmap authority is required. **No ranking or automatic tie-break** — see ADR-013.

## §21 — Repository Governance Manifest

Repository-local configuration artifact: **`.repo-governor.json`** (encoding set by ADR-015; the YAML below is illustrative)

```yaml
repo_governor:
  version: 1

repository:
  id: org/example-repo

condition:
  assessed: L3
  profile: GOVERNOR_FULL

providers:
  roadmap_authority:
    type: linear
    project: ENG

  architecture:
    - type: adr
      path: docs/adr

  execution:
    type: beads

  repository:
    type: git

  change_signals:
    - type: renovate

  retirement:
    - type: repository_analysis

  decision_history:
    type: configured

permissions:
  roadmap_authority:
    read: true
    write: false

  architecture:
    read: true
    write: false

  execution:
    read: true
    write: true
```

The manifest defines where Repo Governor retrieves each form of state; which providers are authoritative for which role; and what Repo Governor may read or write. **It must not duplicate provider data.**

> ADR-015 makes JSON canonical — a YAML-subset parser silently mis-typed 7 of 10 realistic values.
> ADR-004 adds `engine_min_version`, per-binding `contract_version`, and explicit `adapter` paths, and makes this the *sole* binding artifact. ADR-013 resolves the list-vs-scalar cardinality shown above.

## §22 — Permission Model

Every provider must expose explicit permissions.

Minimum: `read` · `write` · `execute`
Optional: `create` · `update` · `archive` · `comment` · `transition`

**Repo Governor must not infer write authority from available credentials** (INV-014). ADR-005 sets the default to deny and reserves `execute` as unimplemented at v1.

---

## §57 — Required Final Pre-Implementation Simulation

**Experiment ID:** `RG-SIM-ONBOARDING-v0.1`

Validate repository attachment, provider detection, provider selection, conflict handling, and governance-manifest generation before implementation begins.

## §58 — Onboarding Fixture A — Greenfield

Repository: Git only. No roadmap, no task tracker, no ADRs, no Renovate.

Expected:

```text
Repository condition:  L0
No providers silently assigned.
Roadmap authority:     manual / unresolved
Execution provider:    not required
Architecture:          UNKNOWN
Recommended:           minimal greenfield governance
```

**Pass requirement:** Repo Governor must not force unnecessary infrastructure.

## §59 — Onboarding Fixture B — Growing Repository

Repository contains: GitHub Projects, `docs/adr/`, Renovate, Git.

Expected detection:

```text
GitHub Projects → candidate RoadmapAuthorityProvider
ADRs            → candidate ArchitectureEvidenceProvider
Renovate        → candidate ChangeSignalProvider
Git             → RepositoryEvidenceProvider
```

Execution provider remains optional.

## §60 — Onboarding Fixture C — Complex Repository

Repository contains: Linear, GitHub Projects, Beads, ADRs, Renovate, Git.

Expected:

```text
ROADMAP PROVIDER CONFLICT

Candidates:              Linear, GitHub Projects
Execution candidate:     Beads
Architecture candidate:  ADRs
Change signal candidate: Renovate
```

Repo Governor must **stop onboarding** until one roadmap authority is explicitly selected. Example accepted configuration:

```text
Linear   → RoadmapAuthorityProvider
Beads    → ExecutionStateProvider
ADRs     → ArchitectureEvidenceProvider
Renovate → ChangeSignalProvider
Git      → RepositoryEvidenceProvider
```

Final state: `READY_FOR_GOVERNANCE`

## §61 — Implementation Gate

The specification may move to `IMPLEMENTATION_READY` only when:

| # | Condition | Status (2026-08-17) | Evidence |
| --- | --- | --- | --- |
| 1 | onboarding simulation passes | ✅ Met | `conformance/onboarding.py` 29/29 |
| 2 | provider conflicts handled deterministically | ✅ Met | fixture C halts, no ranking |
| 3 | low-complexity repositories operate without unnecessary providers | ✅ Met | fixture A: Git only, L0 |
| 4 | provider detection does not assign authority | ✅ Met | proposal written; loader never reads it |
| 5 | manifest semantics stable enough for implementation | ✅ Met | `conformance/manifest.py` 26/26 |
| 6 | at least one provider per required core role has a viable adapter contract | ✅ Met | 9 adapters, Layer 1 112/112 |
| 7 | `UNKNOWN` and failure behavior defined | ✅ Met | `conformance/vocabulary.py`, closed sets enforced |

**Met 2026-08-17**, each verified by `engine/completion.py GATE-N` returning `STOP_COMPLETE`, not by assertion. See the ADR index for what this does *not* mean.
