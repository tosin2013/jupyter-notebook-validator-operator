# Architecture changes

The work in front of you collides with a recorded decision — an ADR forbids the approach, or demands one nobody wants anymore, or two decisions turn out to conflict. The collision is real information. It is not permission to route around the decision.

**Lane:** Architecture / Platform. An ADR conflict, platform shift, or migration need becomes an **architecture review or candidate** — never an in-flight swerve.

## Prompt recipes

Before work that smells architectural:

> Before implementing issue N, read the architecture constraints through Repo Governor and list the Accepted decisions that bind this work. If the natural implementation conflicts with any of them, **stop and report the conflict** — do not pick a side, and do not design around the decision.

When a conflict surfaces mid-task:

> Write that up as an architecture review candidate: the decision in tension, what changed since it was accepted, the options with their costs. **Do not implement any option**, and do not edit the ADR — changing a decision's status is a ratification act, not an edit.

When the assistant proposes a new decision:

> Draft it as a `Proposed` ADR: context, decision, consequences — including the honest negative ones. **`Proposed` is not architecture anyone may rely on.** Do not update code, SKILL.md, or references to assume it, and do not mark it Accepted; ratification is mine (§68).

## What the engine will say

`get_constraints` reports the architecture state: `DEFINED` from Accepted decisions, `INFERRED` when only Proposed ones exist — which do **not** establish authoritative constraints — and `UNKNOWN` with `ARCHITECTURE_PARTIALLY_READ` (blocking) when more decision files could not be read than constraints were established. That last one means fix the reading problem first; governing on the readable minority would state an architecture that is mostly unexamined.

## The forbidden shortcut

**Implementing around a decision.** Code that quietly contradicts an Accepted ADR converts the decision ledger into fiction one commit at a time — and the drift is invisible precisely because the code never cites the decision it violates. This repository shipped that failure itself: an accepted rule about how adapters must be reached, contradicted by an instruction that never mentioned it. The status ladder is the only path: `Proposed → Accepted`, or `Accepted → Superseded` by a decision that says why.
