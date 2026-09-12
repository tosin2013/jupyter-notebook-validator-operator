# Bugs

Something is broken. A defect is the lane where governance feels most like bureaucracy — the fix is right there — and the lane where skipping it does the most quiet damage, because "fixing the bug" is the most common disguise for unauthorized scope.

**Lane:** Reliability / Defect. A found defect is a **defect candidate**; urgency can accelerate admission, it cannot replace admission.

## Prompt recipes

When the assistant finds a bug while doing other work:

> Capture that defect as a candidate: symptoms, reproduction, suspected cause, blast radius. **Do not fix it** — it is not the work you are authorized for. If you believe it blocks issue N, say so explicitly and I will decide whether to admit it.

When you are handing over a bug to fix:

> Issue N is a defect. Check its authority with Repo Governor first, then fix **the defect only**: the failing behaviour, its test, nothing else. Improvements you notice nearby are discoveries — capture them, do not make them. If the fix genuinely cannot land without touching something else, stop and tell me what and why.

For a production-severity defect, where the pressure to skip is highest:

> Treat this as urgent *candidate* work: give me the minimal diagnosis and the smallest safe fix as a proposal. I will admit it immediately if it holds up — urgency changes how fast I decide, not who decides.

## What the engine will say

A defect with an admitted, assigned issue reads `CONTINUE` like any other authorized work. An unfiled or unmilestoned defect reads `UNKNOWN` / `NOT_ADMITTED` — which is correct and is the point: *broken* is a fact about the code, *authorized to fix* is a fact about the roadmap, and the engine only ever reports the second. When the fix is done, [finishing-work](finishing-work.md) applies unchanged — bugs get no exemption from the completion firewall.

## The forbidden shortcut

**"While I'm in here."** The defect authorizes the defect. The three adjacent improvements the fix revealed are `DISCOVERED → EXECUTING` wearing a repair jacket — each needs its own admission, and a bug fix that arrives bundled with unrequested improvements is worse than either alone, because reviewing it means untangling authority after the fact.
