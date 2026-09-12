# Discovering work mid-task

The assistant is working on an authorized item and notices something else — a useful capability, an obvious refactor, a gap worth investigating. This is the moment governance exists for: the discovery is real, the value may be real, and **none of that is authority**.

**Lanes:** Product / Capability (a capability worth having → candidate, never automatically admitted) and Research Candidate (uncertainty worth investigating → a question to file, not an implementation to start).

## Prompt recipes

Set the expectation when the task begins, not after the temptation arrives:

> While working on issue N, capture any newly discovered features, defects, dependency concerns, architecture questions, or retirement candidates **separately** — as notes or draft issues for my review. **Do not implement any of them**, however small, unless Repo Governor resolves separate execution authority for that item.

When the assistant proposes an improvement mid-stream:

> Capture that as a candidate with enough context that someone could evaluate it later — what you found, why it matters, what it would take. Then return to issue N. **Do not add it to a milestone, and do not treat filing it as permission to start.**

For a genuine open question rather than a buildable thing:

> File that as a research candidate: state the question, what evidence would answer it, and what decision it blocks. **Not an implementation task** — the output of research is an answer, usually an ADR, not code.

## What the engine will say

`CAPTURE_ONLY` is the *correct, complete* outcome for a discovery — not a deferral, not a consolation prize. The engine now rules on discoveries directly (`engine/envelope.py --discovery`), so ask it rather than deciding yourself. ADR-024 is `Accepted` — its last acceptance condition, a measurement on repositories this project does not own, was answered across six of them — and INV-001 is now enforced rather than merely followed: an unsubstantiated necessity claim fails closed, and once acceptance conditions are satisfied nothing converts to execution at all. A clean `completion.py` run on issue N still says nothing about the thing discovered along the way; that is a separate question with a separate answer.

## The forbidden shortcut

**`DISCOVERED → EXECUTING`.** The most tempting transition in the entire model, taken in the moment of "it's three lines and I'm already here." The fix being small changes the diff, not the authority. Capturing it is the whole job; a captured discovery that waits is the system working.
