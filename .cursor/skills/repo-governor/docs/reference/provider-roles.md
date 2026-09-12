# Provider Roles & Conformance

> Extracted from PRD v0.2 §10–§17 and §45–§50 on 2026-08-17. Original section numbering preserved.
> **Normative.** Cardinality per role is set by ADR-013; the adapter wire protocol by ADR-003.

---

## §10 — Provider Model

Repo Governor treats all external systems as pluggable providers. Eight categories (seven original, plus one added by ADR-017):

1. `RoadmapAuthorityProvider`
2. `ArchitectureEvidenceProvider`
3. `ExecutionStateProvider`
4. `RepositoryEvidenceProvider`
5. `ChangeSignalProvider`
6. `RetirementEvidenceProvider`
7. `DecisionHistoryProvider`
8. `AcceptanceCriteriaProvider` *(added by ADR-017, 2026-08-17 — what counts as done; distinct from architecture evidence, which is how work must be built)*

## §11 — RoadmapAuthorityProvider

**Purpose:** determine what work has been admitted and whether execution is currently authorized.

```text
get_work(id)
get_status(id)
get_authority(id)
get_scope(id)
get_acceptance_conditions(id)
get_non_goals(id)
get_parent_or_goal(id)
get_decision_history(id)
```

A valid implementation must answer: Does the work item exist? Is it admitted? Is execution authorized? Has authority been withdrawn? What is the accepted outcome? What is explicitly outside scope? What completes the work?

Potential implementations: Linear, Jira, Jira Product Discovery, Plane, GitHub Projects, YouTrack, file/manual provider.

## §12 — ArchitectureEvidenceProvider

**Purpose:** determine architecture constraints relevant to authorized work.

```text
get_active_decisions(scope)
get_superseded_decisions(scope)
get_specs(scope)
get_constraints(scope)
get_provenance(scope)
```

Potential implementations: ADR directories, Mneme, OpenSpec, Spec Kit, architecture policy files, custom providers, none.

Allowed result: `DEFINED` · `INFERRED` · `UNKNOWN`

## §13 — ExecutionStateProvider

**Purpose:** represent detailed execution state beneath an authorized work item.

```text
find_execution_root(authority_id)
get_tasks(root)
get_dependencies(root)
get_completed_work(root)
get_active_work(root)
get_failures(root)
get_discoveries(root)
get_handoff_state(root)
get_execution_history(root)
```

Potential implementations: Beads, Amber, GAAI, Backlog.md, other agent task systems, none.

**Repo Governor must not require an execution provider for low-complexity repositories.**

## §14 — RepositoryEvidenceProvider

**Purpose:** represent what is actually present in the repository — Git state, files, source, tests, schemas, interfaces, manifests, migrations, configuration, entry points, generated artifacts, feature flags.

**The repository provider must not establish roadmap authority** (INV-003).

## §15 — ChangeSignalProvider

**Purpose:** surface technical ecosystem changes — dependency updates, security advisories, EOL notices, platform/runtime changes, API deprecations, compatibility changes.

Potential implementations: Renovate, Dependabot, lifecycle APIs, vendor lifecycle sources, custom providers.

Signal state: `SIGNAL` · `IMPACT_ASSESSED` · `WATCH` · `NO_ACTION` · `CHANGE_CANDIDATE`

## §16 — RetirementEvidenceProvider

**Purpose:** provide evidence relevant to removing repository assets.

```text
static_references(asset)
dynamic_references(asset)
public_contracts(asset)
tests(asset)
configuration_references(asset)
migration_obligations(asset)
release_obligations(asset)
architecture_references(asset)
runtime_usage(asset)
```

Potential implementations: static-analysis tools, repository analysis, telemetry providers, custom scripts, language-specific analyzers.

## §17 — DecisionHistoryProvider

**Purpose:** resolve prior accepted decisions relevant to current work. Must support: accepted, deferred, rejected, superseded, reversal conditions, evidence lineage.

```yaml
feature: organization_rbac

previous_decision:
  disposition: DEFERRED
  reason: no validated organization-level requirement
  reversal_condition: validated multi-user organization requirement
```

**Rediscovery must not erase prior decision state.**

> ADR-009 makes the repository-local decision log a built-in implementation of this role, so no external system is required.

---

## §45 — Tool-Independent Provider Conformance

Every provider implementation must pass conformance tests. A provider must advertise: supported contract version; supported capabilities; persistence semantics; read/write permissions; failure behavior; provenance quality.

```yaml
provider:
  type: execution
  name: beads
  contract_version: 1

capabilities:
  dependencies: true
  handoff_state: true
  persistence: true
  discoveries: true

permissions:
  read: true
  write: true
```

## §46 — Roadmap Provider Conformance

1. work lookup; 2. current status retrieval; 3. authority resolution; 4. cancellation detection; 5. scope retrieval; 6. acceptance-condition retrieval; 7. provenance preservation.

## §47 — Execution Provider Conformance

1. map execution root to authority item; 2. list current tasks; 3. preserve dependency graph; 4. retrieve completed work; 5. retrieve failed attempts; 6. retrieve discoveries; 7. preserve handoff state; 8. accurately advertise persistence.

## §48 — Architecture Provider Conformance

1. current decision detection; 2. superseded decision detection; 3. relevant-scope lookup; 4. provenance; 5. valid `UNKNOWN` result.

## §49 — Change-Signal Provider Conformance

1. surface signal; 2. distinguish source date/version; 3. preserve provenance; 4. avoid representing signal as required work.

## §50 — Retirement Provider Conformance

1. static references; 2. known dynamic references; 3. public-contract awareness where available; 4. obligation reporting; 5. uncertainty reporting; 6. no autonomous deletion.

> ADR-008 adds two layers on top of these minimums: honest capability advertisement, typed failure, absence-vs-unknown distinction, mandatory provenance (Layer 1); and cross-provider equivalence fixtures (Layer 2 — the thesis test).
