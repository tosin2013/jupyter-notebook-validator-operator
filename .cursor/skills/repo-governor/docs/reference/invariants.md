# Governance Invariants

> Extracted from PRD v0.2 §5–§6 on 2026-08-17. Original section numbering preserved.
> **Normative.** These are the rules ADR-002 requires to be implemented as executable predicates, not prose.

---

## §5 — Core Governance Principle

> **Information may justify a decision. Information does not acquire authority merely by existing.**

This principle applies equally to:

* repository code;
* roadmap items;
* execution tasks;
* architecture documents;
* dependency signals;
* static-analysis output;
* agent discoveries;
* persistent memory.

---

## §6 — Core Governance Invariants

### INV-001 — Discovery does not confer authority

A discovered feature, cleanup opportunity, refactor, dependency upgrade, architectural improvement, or technical possibility cannot automatically become executable work.

### INV-002 — Execution state does not confer roadmap authority

A task marked `READY`, `OPEN`, or `IN_PROGRESS` in an execution system does not independently establish that the parent work remains authorized.

### INV-003 — Repository evidence does not confer product intent

Code, TODO comments, incomplete modules, old feature flags, branches, and abandoned implementations are evidence of repository state, not current roadmap intent.

### INV-004 — Architecture authority constrains work but does not authorize it

An accepted ADR or specification may define how authorized work must be implemented. It does not independently authorize the work.

### INV-005 — Persistence does not confer authority

Durable execution state can preserve useful context. It can also preserve stale or withdrawn work. Persistent state must therefore always be resolved against current authority.

### INV-006 — External change is a signal, not automatically work

A dependency release, security advisory, runtime EOL, or external API change must be assessed before becoming an authorized maintenance item.

### INV-007 — Apparent obsolescence does not confer deletion authority

Unused-code detection, missing imports, low telemetry, or static-analysis output are retirement evidence only. Removal requires obligation checks.

### INV-008 — Superseded decisions do not constrain current work

Historical decisions remain provenance. Accepted successor decisions govern current architecture or policy.

### INV-009 — Completed authorized scope means stop

When accepted completion criteria are met, the AI agent must stop unless another separately authorized work item exists.

### INV-010 — Promotion requires a legal state transition

Illegal direct transitions include:

```text
DISCOVERED → EXECUTING
VERSION_SIGNAL → UPGRADE
SUSPECTED_OBSOLETE → DELETE
EXECUTION_READY → AUTHORIZED
```

### INV-011 — An empty repository does not confer unlimited authority

Greenfield repositories require explicit limits on initial architecture, dependencies, scope, and product assumptions.

### INV-012 — Unknown is a valid governance outcome

When authority, provenance, architecture, compatibility, or external obligations cannot be resolved, Repo Governor must return `UNKNOWN` or require review.

### INV-013 — Provider detection does not establish provider authority

Finding Linear metadata, GitHub Projects, Beads, ADRs, or another compatible system allows Repo Governor to propose a provider. Only accepted configuration establishes its governance role.

### INV-014 — Provider capability does not determine policy

A provider may technically support writes, status changes, or automation. That technical ability does not grant Repo Governor permission to use those capabilities.

---

## Implementation status

| Invariant | Enforced by | Always on? |
| --- | --- | --- |
| INV-001 | ADR-014 (envelope necessity test) | ✅ all profiles |
| INV-002 | ADR-003 (role separation) | profile-gated |
| INV-003 | ADR-003 (repository role cannot answer authority) | profile-gated |
| INV-004 | ADR-013 (architecture is multi-valued evidence) | profile-gated |
| INV-005 | ADR-009 (evidence chain) | profile-gated |
| INV-006 | lifecycles §34 | profile-gated |
| INV-007 | lifecycles §35–§36 | profile-gated |
| INV-008 | ADR-013 rule 4 | profile-gated |
| INV-009 | ADR-014 (completion firewall) | ✅ all profiles |
| INV-010 | ADR-002 (transition tables) | ✅ all profiles |
| INV-011 | repository-conditions §24 | L0 |
| INV-012 | ADR-007 (`UNKNOWN` terminal) | ✅ all profiles |
| INV-013 | ADR-010 (detection ≠ binding) | ✅ all profiles |
| INV-014 | ADR-005 (deny by default) | ✅ all profiles |

Per ADR-006, the four marked ✅ are the irreducible core carried in `SKILL.md` tier 2. The rest load with their profile.
