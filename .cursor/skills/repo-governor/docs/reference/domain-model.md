# Domain Model, Output Contract & Observability

> Extracted from PRD v0.2 §30–§32, §43–§44, §52 on 2026-08-17. Original section numbering preserved.

---

## §30 — Core Domain Objects

```text
RepositoryContext
AuthorityItem
ScopeEnvelope
ArchitectureConstraint
ExecutionRoot
Discovery
MaintenanceCandidate
RetirementCandidate
DecisionReference
EvidenceRecord
ProviderBinding
ProviderPermission
```

## §31 — ScopeEnvelope

Every executable `AuthorityItem` resolves to:

```yaml
authority_id:
required_outcome:
in_scope:
necessary_incidental_work:
non_goals:
architecture_constraints:
discovery_policy:
acceptance_conditions:
stop_condition:
```

The scope envelope answers:

> What may the coding agent do while satisfying this specific authorization?

> **ADR-014** makes this *compiled* from provider state at evaluation time, never hand-authored. Non-goals are hard boundaries; `necessary_incidental_work` is bounded by a necessity test (*is the authorized outcome unreachable without this?*), not by size.

## §32 — Discovery Model

Discovery types:

```text
POSSIBLE_FEATURE
BUG
TECHNICAL_DEBT
ARCHITECTURE_IMPLICATION
MAINTENANCE_SIGNAL
RETIREMENT_SIGNAL
RESEARCH_QUESTION
DUPLICATE
UNKNOWN
```

Default authority: `NONE`
Default disposition: `CAPTURE_ONLY`

— unless the discovery is proven necessary to satisfy current authorized scope (INV-001).

## §43 — Required Governance Output

Every material evaluation produces:

```yaml
decision:
repository:
profile:
actor:
requested_action:

authority:
architecture:
execution:
repository_evidence:
change_signals:
retirement_evidence:
decision_history:

scope:
discoveries:
conflicts:
unknowns:
required_reviews:
stop_condition:

provenance:
```

## §44 — Example Evaluation

Input:

```yaml
actor: coding-agent

requested_action:
  type: implement
  work_ref: ENG-142

providers:
  roadmap: linear
  execution: beads
  architecture: adr
  repository: git
```

Provider state:

```text
Linear:     ENG-142 ACTIVE
Beads:      ENG-142/root IN_PROGRESS
ADR:        ADR-0029 ACCEPTED
Discovery:  DISC-88 possible RBAC feature
```

Output:

```yaml
decision: CONTINUE

authority:
  status: AUTHORIZED

architecture:
  state: DEFINED
  constraints:
    - ADR-0029

execution:
  root: ENG-142/root

discoveries:
  DISC-88:
    disposition: CAPTURE_ONLY

stop_condition:
  acceptance_conditions_satisfied: false
```

> Implement as an end-to-end fixture (ADR-014 step 4).

## §52 — Observability

Every governance decision must be traceable. Minimum event:

```yaml
timestamp:
repository:
profile:
actor:
requested_action:
provider_bindings:
authority_result:
architecture_result:
execution_result:
decision:
evidence_refs:
unknowns:
```

> **ADR-009** extends this into an append-only, hash-chained decision log at `.repo-governor/decisions/`, adding `engine_version`, `manifest_hash`, and `provider_snapshots` so decisions are replayable. Discoveries live separately at `.repo-governor/discoveries/` so the decision log stays immutable.
