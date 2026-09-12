# Invariants — operational reference

Fourteen invariants. Four are always on; ten activate with the governance profile.
Normative text and enforcement mapping: [`docs/reference/invariants.md`](../docs/reference/invariants.md).

This file answers *"what do I do when I hit this?"* rather than *"what does the spec say?"*.

## Always on, every profile

| | Rule | What it forbids in practice |
|---|---|---|
| **INV-001** | Discovery confers no authority | Implementing a TODO you found. Fixing an unrelated bug "while you're in there". Acting on your own good idea. |
| **INV-009** | Completed scope means stop | One more small improvement after acceptance conditions pass. |
| **INV-010** | No illegal transitions | `DISCOVERED → EXECUTING`, `VERSION_SIGNAL → UPGRADE`, `SUSPECTED_OBSOLETE → DELETE`. |
| **INV-012** | `UNKNOWN` is valid | Resolving an unknown by assuming the safe-looking answer. |

## Profile-gated

| | Rule | Bites when |
|---|---|---|
| INV-002 | Execution state confers no roadmap authority | A tracker says `READY` but the roadmap item was cancelled |
| INV-003 | Repository evidence is not product intent | Code, branches and TODOs look like a plan |
| INV-004 | Architecture constrains, does not authorize | An accepted ADR describes a feature nobody admitted |
| INV-005 | Persistence confers no authority | Durable agent memory preserves withdrawn work |
| INV-006 | External change is a signal, not work | A new major version appears |
| INV-007 | Apparent obsolescence confers no deletion authority | Static analysis finds zero references |
| INV-008 | Superseded decisions do not constrain | An old ADR contradicts a newer accepted one |
| INV-011 | Empty repository ≠ unlimited authority | Greenfield, no constraints visible |
| INV-013 | Detection ≠ provider authority | A tracker is present and reachable but unbound |
| INV-014 | Capability ≠ permission | A credential is in the environment |

## The two you will actually argue with

**INV-001** feels wrong in the moment. You have found a real bug, the fix is three lines, and capturing it feels like bureaucracy. Capture it anyway — the disposition is `CAPTURE_ONLY` and that is a complete outcome, not a deferral.

**INV-007** produces `RETIREMENT_REVIEW` on assets with zero references, which reads as a false positive. It is not. Dynamic loading, runtime usage, public contracts and migration obligations are invisible to static analysis, so "no references found" and "safe to delete" are different claims.
