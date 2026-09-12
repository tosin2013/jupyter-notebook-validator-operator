# Lifecycles & Resolution Rules

> Extracted from PRD v0.2 §33–§40 on 2026-08-17. Original section numbering preserved.
> **Normative.** ADR-002 requires these to be implemented as explicit transition tables that reject illegal transitions structurally (INV-010).

---

## §33 — Feature Admission Lifecycle

```text
DISCOVERED
     ↓
CAPTURED
     ↓
CANDIDATE
     ↓
EVALUATED
     ↓
ADMITTED
     ↓
AUTHORIZED
     ↓
EXECUTING
     ↓
EVIDENCE_READY
     ↓
COMPLETE
```

Direct transition prohibited: `DISCOVERED → EXECUTING`

## §34 — Maintenance Lifecycle

```text
EXTERNAL_SIGNAL
      ↓
IMPACT_ASSESSED
      ↓
NO_ACTION / WATCH
      ↓
CHANGE_CANDIDATE
      ↓
ADMITTED
      ↓
AUTHORIZED
      ↓
EXECUTING
      ↓
VERIFIED
```

**New version alone does not authorize upgrade work** (INV-006).

## §35 — Retirement Lifecycle

```text
SUSPECTED_OBSOLETE
      ↓
EVIDENCE_COLLECTED
      ↓
RETIREMENT_CANDIDATE
      ↓
OBLIGATION_CHECK
      ↓
RETAIN / REVIEW / REMOVAL_READY
      ↓
REMOVAL_AUTHORIZED
      ↓
REMOVED
      ↓
VERIFIED
```

## §36 — Retirement Obligation Check

Before removal, Repo Governor checks available evidence for: dynamic loading; plugin registration; CLI use; configuration references; API exports; migration logic; supported release branches; compatibility promises; tests; architecture references; generated consumers; telemetry; external contracts.

If unresolved: `UNKNOWN` or `RETIREMENT_REVIEW`.

> This is the check the April 2026 production-database deletion lacked. See ADR-012 domain considerations.

## §37 — Architecture Resolution

Architecture state must resolve to `DEFINED` · `INFERRED` · `UNKNOWN`.

- **DEFINED** — accepted current architecture evidence exists.
- **INFERRED** — repository patterns exist but authoritative status is not established.
- **UNKNOWN** — insufficient evidence exists.

**Repo Governor must not manufacture architectural authority.**

## §38 — Authority vs Execution Conflict

```text
Roadmap provider:    FEATURE-81 = CANCELLED
Execution provider:  FEATURE-81/root = READY
```

Expected: `AUTHORITY_WITHDRAWN`

**Roadmap admission status governs execution authorization.** Per ADR-013 rule 5, this is a *resolved* case with a defined answer — not `CONFLICT`, which is reserved for disagreement between peers of the same role.

## §39 — Rediscovered Work

```text
Agent discovers RBAC opportunity.
```

Decision history says:

```text
RBAC previously deferred.
Reversal condition not satisfied.
```

Expected: `CAPTURE_ONLY` or `ROADMAP_REVIEW`. **Never `EXECUTE`** without new authority.

> Requires that `DEFERRED` and its reversal condition were recorded when decided — see ADR-009 rule 6.

## §40 — Completion Firewall

When acceptance conditions are satisfied: `STOP_COMPLETE`

```text
Authorized:  Persistent state recovery
Completed:   Acceptance tests passed
Discovered:  Admin UI
             Multi-team support
             Refactor
```

Expected:

```text
STOP_COMPLETE

Admin UI:     CAPTURE_ONLY
Multi-team:   CAPTURE_ONLY
Refactor:     CAPTURE_ONLY or MAINTENANCE_REVIEW
```

**Unconditional.** No discovery converts to execution within a satisfied authorization, including discoveries that are correct, small, and obviously beneficial (INV-009, ADR-014 rule 5). This example should be implemented verbatim as an acceptance test.
