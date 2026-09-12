# Finishing work

The acceptance conditions are met. This is the moment the product exists for, because it is the moment assistants handle worst: work rarely ends with nothing left over, and the leftovers feel like momentum. **Completion means stop — not "stop after this one small thing."**

**Lane:** the exit of every lane. The completion firewall (ADR-023, Accepted) is the least negotiable behaviour in the model.

## Prompt recipes

When the work seems done:

> Evaluate issue N with Repo Governor. If it reports `STOP_COMPLETE`, stop: summarize what was done, list every discovery you captured along the way as candidates, and **do not** start any of them — including the small ones, especially the small ones.

Setting the expectation at handoff, before momentum exists:

> When issue N reaches its acceptance conditions, stop there. Improvements you notice at the end are discoveries for my review, not a victory lap. Continuing needs a **separately authorized item**, which means a new evaluation against a different authority — not this one stretched.

When the engine will not certify completion:

> The evaluation returned `CONTINUE` with unmet criteria — list exactly which checks failed and what would satisfy them. **Do not amend the acceptance criteria to make the work pass.** If a criterion is genuinely wrong, propose the amendment with its citation and I will decide.

## What the engine will say

`STOP_COMPLETE` is composed, not asserted: roadmap authority + declared acceptance criteria + repository evidence, each from its own provider (ADR-023). Missing criteria yield `CONTINUE` with `NO_CRITERIA_DECLARED` — the engine refusing to certify a completion it cannot verify, which is a feature. Completion does **not** retract authority: finished work stays `AUTHORIZED`, because completion is a separate axis (§40) — so a `STOP_COMPLETE` is the engine telling you the authorization is *exhausted*, not that it never existed.

Know the honest limit: the engine rules; it does not restrain. An assistant that continues past `STOP_COMPLETE` leaves a decision record showing it did so against an explicit verdict — accountability, which is what a skill can enforce, rather than prevention, which it cannot.

## Finishing it and landing it are different questions

`STOP_COMPLETE` says the work is done. It says nothing about how this repository wants it delivered, and the engine has no opinion on that because no provider can supply one.

Before you commit, re-read the governed repository's `CONTRIBUTING.md` and `AGENTS.md` if it has them. What they ask for is usually specific and usually cheap: run these checks first, branch rather than push, phrase the commit this way, fill in this section of the pull-request template. Skipping it turns finished work into a review round-trip.

Note the asymmetry with everything else on this page. The verdict is composed from evidence and you may not argue with it; the conventions are prose and you simply follow them. Neither is optional, but only one of them is checkable, which is exactly why this step is easy to forget.

## The forbidden shortcut

**Two of them, and the second is subtler.** The obvious one: continuing past `STOP_COMPLETE` because the next thing is small — the specific failure this product was built to prevent. The subtle one: **amending the criteria until the work passes.** That is how a completion firewall is defeated from the inside, and it is why amendments must carry a resolvable citation and why loosening is reported separately (`engine/amendments.py`). The firewall's value is invisible when it works; it looks like lost momentum right up until the one time it wasn't.
