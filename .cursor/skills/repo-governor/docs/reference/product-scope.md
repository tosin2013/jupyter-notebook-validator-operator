# Product Scope & Positioning

> Extracted from PRD v0.2 §1–§4, §7–§9, §66–§70 on 2026-08-17. Original section numbering preserved.

**Project Name:** Repo Governor · **Repository:** `repo-governor` · **Research Domain:** Governed Repository Evolution
**Artifact Type:** Public open-source AI governance skill · **Ownership Coordinate:** Tosin Open Source
**Human Owner / Final Acceptance Authority:** Tosin Akinosho

---

## §1 — Product Summary

Repo Governor is a **tool-independent governance skill for AI-assisted software development**.

It governs what AI coding agents are allowed to create, change, maintain, migrate, deprecate, retire, or stop working on within a software repository.

Repo Governor does not replace software-development tools. It connects to existing tools through provider interfaces and evaluates their state using a consistent governance protocol.

Example provider systems: Linear, Jira, Jira Product Discovery, GitHub Projects, Plane, YouTrack, Beads, Amber, GAAI, Backlog.md, ADRs, Mneme, OpenSpec, Spec Kit, Git, Renovate, Dependabot, static-analysis tools.

The central principle:

> **Providers supply state and evidence. Repo Governor determines what that state permits.**

## §2 — Product Definition

> **Repo Governor is a pluggable AI governance skill that determines what an AI coding agent is authorized to create, change, maintain, or retire in a repository, what it must capture or escalate instead, and when it must stop.**

It evaluates roadmap authority, architectural authority, execution state, repository evidence, external change signals, retirement evidence, prior decisions, lifecycle obligations, and repository complexity — returning a bounded governance disposition.

## §3 — Problem Statement

AI coding agents operate across repositories containing overlapping sources of information: roadmaps, backlogs, issues, milestones, task graphs, ADRs, architecture documents, specifications, source, tests, TODOs, feature flags, migration code, generated code, dependency manifests, historical branches, execution memory, and previous agent discoveries.

**These artifacts do not have equal authority.** An agent may nevertheless infer permission from information merely because it exists:

> "This TODO exists, so I should implement it."
> "This task is marked Ready in the execution tracker, so I should work on it."
> "A new major version was released, so I should upgrade."
> "This module has no static references, so I should delete it."
> "The current feature is complete, but I found three improvements, so I should continue."
> "This ADR exists, therefore it still constrains the implementation."
> "This repository is empty, therefore I am free to choose the architecture."

Resulting failures: unauthorized feature expansion, roadmap drift, duplicate work, architecture drift, stale execution, unnecessary dependency upgrades, premature architecture, accidental compatibility breaks, resurrection of previously rejected work, unsafe code retirement, agent work continuing after completion, increased human review burden.

## §4 — Research Thesis

AI-assisted repository development needs a governance layer spanning multiple independent control planes:

```text
                 GOVERNED REPOSITORY EVOLUTION

                         INTENT
                           │
                    PRODUCT AUTHORITY
                           │
                           ▼
                       ADMISSION
                           │
          ┌────────────────┼────────────────┐
          │                │                │
        CREATE            CHANGE           RETIRE
          │                │                │
       features        maintenance       deletion
       capabilities    dependencies      deprecation
       foundations     migrations        simplification
          │                │                │
          └────────────────┼────────────────┘
                           │
                    ARCHITECTURE
                           │
                     EXECUTION
                           │
                       EVIDENCE
                           │
                       OUTCOME
                           │
                         STOP
```

Repo Governor is the policy layer governing these transitions.

> §5 Core Governance Principle and §6 Core Governance Invariants live in [invariants.md](invariants.md).

## §7 — Product Goals

1. determine whether requested work is currently authorized;
2. establish the scope envelope around authorized work;
3. resolve architecture evidence;
4. inspect and recover execution state;
5. reconcile conflicting provider states;
6. preserve discoveries without promoting them automatically;
7. govern dependency and lifecycle change;
8. govern code retirement;
9. recognize previously rejected or deferred work;
10. enforce hard completion stops;
11. operate across repository complexity levels;
12. support interchangeable provider implementations;
13. preserve provenance for every material decision;
14. minimize unnecessary human escalation;
15. preserve a valid `UNKNOWN` state;
16. support repository onboarding without assuming tool choices.

## §8 — Non-Goals

Repo Governor is **not**: a project-management system; a roadmap database; an issue tracker; an execution tracker; a specification system; an ADR system; a dependency updater; a CI/CD engine; a source-control replacement; a static-analysis engine; a universal agent-memory system; an autonomous product manager; an autonomous architect; an autonomous cleanup tool; a deployment platform; a code-generation model; a hosted SaaS commitment.

Repo Governor should reuse existing tools rather than rebuild mature functionality.

## §9 — Product Architecture

```text
                       REPO GOVERNOR
                 policy + governance skill
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
  ROADMAP AUTHORITY   ARCHITECTURE        EXECUTION
      PROVIDER          PROVIDER           PROVIDER
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                  REPOSITORY EVIDENCE
                           │
                  CHANGE SIGNALS
                           │
                 RETIREMENT EVIDENCE
                           │
                  DECISION HISTORY
```

Repo Governor consumes provider responses through normalized contracts. The policy engine evaluates normalized state and produces a governance disposition.

> **Access modality, added 2026-08-17 from the #16 research.** The diagram above shows providers as uniform peers, which is false in one important way: they differ not only in *schema* but in *how they are reached*.
>
> | Provider | Modality |
> | --- | --- |
> | Linear | remote MCP, OAuth — often already connected in the agent host |
> | Beads | `bd` CLI, local, Git-backed |
> | ADRs, acceptance criteria | filesystem |
> | GitHub | `gh` CLI / GraphQL |
> | Renovate | config file + API |
>
> Normalizing across modalities is a different job from normalizing across schemas, and it places Repo Governor in a **middleware position**. ADR-003's adapter protocol is already that middleware bus; it was built before the position was named.
>
> The position is accepted; the **middleware mandate is refused**. Repo Governor normalizes access only as far as ruling requires, and never takes responsibility for integration as a product concern — §8 disclaims that, and §54 now names the failure modes the middle brings. The public positioning in §66 stays governance-first: "skill" keeps expectations right where "middleware" invites an integration product.

---

## §66 — Public Positioning

> **Repo Governor brings consistent governance to AI-assisted software development without forcing teams to replace their existing tools. Connect the systems you already use for roadmap, architecture, execution, source control, and maintenance; Repo Governor determines what agents are actually authorized to do.**

## §67 — Project Naming

**Public project name:** Repo Governor · **Git repository:** `repo-governor` · **Research concept:** Governed Repository Evolution

The project must not be named after a provider combination. Avoid `linear-beads-governor`, `beads-roadmap`, `jira-agent-governor`. Provider independence is part of the product thesis.

## §68 — Ownership

Repo Governor is a future public project under **Tosin Open Source**.

Decision Crafters retains research provenance; reference-application provenance; nonexclusive upstream use; and the ability to test, reference, or integrate the public project subject to its future license and separately accepted relationships.

Ownership does not imply current authorization to create or publish the repository.

## §69 — Product State

```text
QUESTION → SPECIFIED → ARCHITECTURE SIMULATED → COMPLEXITY SIMULATED
→ EXTERNAL LANDSCAPE VALIDATED → SCOPE BROADENED → PROVIDER MODEL DEFINED
→ PRD v0.2 → ONBOARDING SIMULATION REQUIRED → IMPLEMENTATION_READY
```

State as of 2026-08-17: **`IMPLEMENTATION_READY`.** `RG-SIM-ONBOARDING-v0.1` passes and all seven §61 gate conditions are met, each engine-verified. Superseded by ADR-001…ADR-017, all still `Proposed`. The gate says the architecture is stable enough to build on; it does not say the thesis holds — see open questions #1 and #2.

## §70 — Final Product Principle

> **Tools tell us what they know. Governance determines what that knowledge allows an agent to do.**

And its operational consequence:

> **AI may discover broadly, but it may execute only within explicitly resolved authority.**
