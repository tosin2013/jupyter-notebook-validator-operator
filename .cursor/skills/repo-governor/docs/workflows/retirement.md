# Retirement

Code looks dead — no references, a deprecated path, a feature flag nobody remembers. Deletion is the least reversible act an assistant performs, and unused-*looking* is a claim about what static analysis can see, not about what the code owes.

**Lane:** Retirement / Simplification. Old code becomes a **retirement candidate**; evidence is required, and the evidence bar is the highest of any lane. Lifecycle: `SUSPECTED_OBSOLETE → EVIDENCE_COLLECTED → RETIREMENT_CANDIDATE → OBLIGATION_CHECK → RETAIN | REVIEW | REMOVAL_READY → REMOVAL_AUTHORIZED → REMOVED → VERIFIED`.

## Prompt recipes

When something looks removable:

> That module looks unused. Run the retirement obligation check through Repo Governor (`engine/retirement.py`) and report every dimension — static references, dynamic loading, runtime usage, public contracts, migration obligations, tests. **Do not delete anything**, and treat zero static references as one data point, not a verdict.

When the check comes back `RETIREMENT_REVIEW` — which it will:

> File it as a retirement candidate carrying the obligation evidence: what is resolved and clear, what is unresolved and why static analysis cannot see it. **Removal needs its own authorization**; a candidate with honest unknowns is the complete outcome here.

When removal has actually been authorized:

> Issue N authorizes removing X. Remove **exactly X** — not its neighbours, not the cleanup it suggests — then verify: build, tests, and the obligation dimensions that were resolvable. Anything else you find worth removing is a new candidate, not an extension of this one.

## What the engine will say

`engine/retirement.py` queries **every** bound retirement provider — obligations accumulate, so one provider finding nothing clears nothing. `REMOVAL_READY` requires every dimension resolved *and* clear, and **static analysis alone can never reach it** (INV-007): dynamic loading, plugin registration, runtime usage, and compatibility promises are invisible to grep and return as blocking unknowns. A `RETIREMENT_REVIEW` on an asset with zero references is the correct, expected answer — not a false positive to argue with.

## The forbidden shortcut

**`SUSPECTED_OBSOLETE → DELETE`.** Absence of references is not absence of obligation — it is absence of *visible* references, reported by a tool that cannot see dynamic dispatch, reflection, external callers, or the promise someone made in a changelog. Deletion from weak evidence is a named product failure condition (§54), and it is the one mistake in this model that a revert does not always undo.
