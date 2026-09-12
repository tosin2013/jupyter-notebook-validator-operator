# Dependency updates

A new version exists — a framework major, a security advisory, an EOL notice. The release is a **signal**, and a signal's first stop is impact assessment, not the package manifest.

**Lane:** Compatibility / Dependency. Signal → impact assessment → maybe a maintenance candidate. The lifecycle is `EXTERNAL_SIGNAL → IMPACT_ASSESSED → NO_ACTION | WATCH | CHANGE_CANDIDATE → ADMITTED → …` ([`references/lifecycles.md`](../../references/lifecycles.md)).

**Binding caveat, stated plainly:** this lane needs a bound `change_signals` provider (Renovate, Dependabot), and a repository may deliberately leave that role unbound — this one does (ADR-022 rule 4: better honestly empty than fixture-backed). With the role unbound, version signals arrive by human notice and the impact assessment is still the required step; only the automated intake is missing.

## Prompt recipes

> Framework X released version Y. Assess the impact **on this repository specifically**: what do we use that changed, what breaks, what do we gain, what is the migration cost? Report `NO_IMPACT`, `WATCH`, or a maintenance candidate with your evidence. **Do not upgrade anything.**

When the assessment says a candidate is warranted:

> Write the impact assessment up as a maintenance candidate — what changes, why now rather than later, the risk of deferring. **Admitting it to the roadmap is my decision.** Do not modify dependency files, lockfiles, or CI configuration.

For a security advisory, where urgency pressure is highest:

> Assess this advisory against our actual usage: are we on an affected version, is the vulnerable path reachable from our code, what is the minimal remediation? Urgency raises the priority of *admission* — it does not replace admission. Report; **do not patch until the work item is authorized.**

## What the engine will say

An upgrade attempted without an admitted work item reads `UNKNOWN` / `NOT_ADMITTED` like any other unadmitted work — the engine does not know or care that a release exists. `IMPACT_NOT_ASSESSED` is the non-blocking unknown that names this lane's missing step: whether a change matters *here* is a judgement about this repository, and no feed can supply it.

## The forbidden shortcut

**`VERSION_SIGNAL → UPGRADE`.** A release existing never means upgrade — that is inferring authority from an artifact's existence, the exact reasoning error on the product's first page. The same applies in reverse: an advisory existing never means patch-now-ask-later. It means *assess now*, and let urgency argue for fast admission on the evidence.
