# Repository Condition Model & Governance Profiles

> Extracted from PRD v0.2 §23–§29 on 2026-08-17. Original section numbering preserved.
> **Normative.** Assessment mechanics, the indicator floor rule, and profile loading are set by ADR-006.

---

## §23 — Repository Condition Model

Governance depth must scale with repository condition:

```text
L0 — GREENFIELD
L1 — SIMPLE
L2 — GROWING
L3 — COMPLEX
L4 — MATURE / HIGH-ASSURANCE
```

**Complexity is not determined by LOC alone.** Indicators include: module/service topology; dependency surface; age/history; public contracts; plugin/dynamic loading; migrations; generated code; number of agents; environment disposability; supported release branches; architecture history; compatibility obligations.

## §24 — L0 — Greenfield

**Characteristics:** new or nearly empty repository; few constraints; no architecture history; product boundary still forming.

**Primary risk:**

> AI mistakes absence of constraints for unlimited design authority. (INV-011)

**Primary controls:** minimum authorized foundation; architecture budget; dependency restraint; `UNDECIDED`; first-feature scope; hard stop.

**Profile:** `GOVERNOR_GREENFIELD`

## §25 — L1 — Simple

**Characteristics:** one application; limited modules; small dependency surface; little architecture history; one developer/agent.

**Primary controls:** authority; scope envelope; discovery capture; acceptance criteria; stop.

**Profile:** `GOVERNOR_LITE` — execution-state provider normally optional.

## §26 — L2 — Growing

**Adds:** more modules; feature flags; migrations; multiple active work items; occasional handoffs; emerging architecture history.

**Profile:** `GOVERNOR_STANDARD` — adds execution lineage; decision history; maintenance candidates; retirement candidates; architecture resolution.

## §27 — L3 — Complex

**Characteristics:** multiple services/packages; extensive dependency graph; dynamic/plugin behavior; several AI agents; ephemeral environments; architecture history; migrations; competing historical evidence.

**Profile:** `GOVERNOR_FULL` — adds durable execution state; agent-handoff recovery; superseded-decision resolution; authority/execution conflict resolution; lifecycle intelligence; stronger provenance.

## §28 — L4 — Mature / High-Assurance

**Characteristics:** public APIs; multiple supported release branches; external consumers; compatibility guarantees; generated clients; dynamic plugins; multiple runtimes; long migration history; multiple concurrent agents.

**Profile:** `GOVERNOR_HIGH_ASSURANCE` — adds release-coordinate awareness; migration obligation checks; public-contract analysis; branch/version-specific architecture; retirement proof; conservative escalation.

> **Floor rule (ADR-006).** Public API surface, supported release branches, or generated consumers raise the assessed level to L4 regardless of repository size, and that floor may not be overridden downward.

## §29 — Greenfield Architecture Budget

Architecture decisions classify as:

```text
LOCAL_REVERSIBLE_CHOICE
CONSEQUENTIAL_ARCHITECTURE_DECISION
```

Examples:

```text
Helper naming                → LOCAL_REVERSIBLE_CHOICE
Persistence architecture     → CONSEQUENTIAL_ARCHITECTURE_DECISION
Public API versioning model  → CONSEQUENTIAL_ARCHITECTURE_DECISION
Temporary test fixture       → LOCAL_REVERSIBLE_CHOICE
```

Consequential decisions require explicit authority or architecture review.

---

## Research basis

PRD §56 Simulation 2 examined repository complexity from greenfield through mature/high-assurance. Finding:

> Governance need is high both where constraints are absent and where accumulated obligations are numerous.

This is why L0 is not "governance off" — it is a different, small set of controls.
