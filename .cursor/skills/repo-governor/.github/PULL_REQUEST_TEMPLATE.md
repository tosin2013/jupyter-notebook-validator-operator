## What this changes

<!-- One or two sentences. The commit messages carry the reasoning; this is the summary. -->

## Which authority admits it

<!--
This repository governs itself. Work is admitted by milestone membership, not by
being a good idea (INV-002). Give the issue number, or say plainly that this is
unadmitted and why it should be taken anyway -- that is an allowed answer, not a
failure to comply.
-->

Authority: <!-- issue number, or "unadmitted, because ..." -->

```
python3 engine/completion.py <issue-number>
```

## Conformance

Every suite, from a clean checkout:

```bash
./tools/bootstrap-decisions.sh
./tools/run-conformance.sh
```

- [ ] 19/19 pass
- [ ] If any suite needs `dolt`, I installed it rather than unbinding the provider

## If you added or changed a test: did you break the code and watch it fail?

<!--
The single most useful box on this form.

PR 43's first fix asserted `wrong == dict(wrong)` -- a tautology. It passed, it
looked like coverage, and setting the value it was meant to check to a constant
left the suite green. It was caught by mutation, not by review.

Three checks written in this repository since have had the same shape: one
compared a constant to itself, one crashed instead of failing, and one asserted
a repository id on a fixture that could never exercise the default it forbade.

So: make the change the test exists to catch, run the suite, confirm it goes
red, then revert. Say what you broke and what failed.
-->

- [ ] I mutated the thing this test guards and the suite went red
- [ ] Not applicable — no tests changed

Mutation tried: <!-- e.g. "set enforcement to always-blocking -> 3 failures" -->

## Before you push

- [ ] **No closing verb next to a `#` reference in any commit message.** They fire from anywhere in the message, quoting does not escape them, and this repository has closed the same issue twice by accident — once in the commit documenting the trap. Write `issue 27`, not `#27`, near words like close/fix/resolve.
- [ ] Nothing private in the diff. This repository is public: rates and shapes, never workspace content (§51).
- [ ] If an ADR changed, its citation count and index entry still match (`python3 conformance/skill.py`).
