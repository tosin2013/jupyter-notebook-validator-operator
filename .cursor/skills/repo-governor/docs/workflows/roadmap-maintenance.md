# Roadmap maintenance

Reconciling the roadmap with reality: candidates that accumulated, work that finished, authority that should be withdrawn. This page is different from the others in one honest way: **Repo Governor does not mutate the roadmap.** Every lifecycle change below is a human act; the engine's role is to tell you what state things are in and to verify what you did.

**Lane:** the human side of all six. The provider (GitHub, Linear, Jira) stays canonical — Repo Governor never keeps a second copy of the roadmap (ADR-022), and that survives any future write capability.

## What is a weak act and what is a strong one

| Act | Kind | Who |
|---|---|---|
| file an unmilestoned issue for a discovery | capture knowledge | anyone, anytime |
| file a maintenance or retirement candidate | capture knowledge | anyone, anytime |
| attach a milestone (**admission**) | mutate lifecycle | human decision |
| assign / clear to execute (**authorization**) | mutate lifecycle | human decision |
| close as `NOT_PLANNED` (**withdrawal**) | mutate lifecycle | human decision |
| close as `COMPLETED` | mutate lifecycle | human, ideally on an engine `STOP_COMPLETE` |

Capture is cheap and safe precisely because it confers nothing. The bottom four change what agents are *allowed to do*, which is why they require an existing authorized decision — and why filing an issue is never admission, however good the issue.

## Prompt recipes

The reconcile survey — read-only by construction:

> Reconcile this repository's governance state with Repo Governor: for every open issue, report its disposition — authorized, admitted-only, not admitted, or complete-but-open. Flag closed issues whose engine verdict disagrees with their closure reason. **Report only. Do not change milestones, labels, assignments, or issue state.**

Before you perform admissions yourself:

> Here are the candidates I am considering admitting: N, M. For each, tell me what admission would make executable, what its acceptance criteria would need to be, and whether anything already recorded (a prior deferral, a reversal condition) bears on it. **The decision is mine; give me the evidence.**

After you have made changes by hand:

> I milestoned N and closed M as NOT_PLANNED. Re-evaluate both and confirm the engine now reads N as admitted and M as a recorded rejection. Report any surprises.

That last step closes a real loop: a `NOT_PLANNED` closure is read back by the decision-history provider as a recorded `REJECTED` decision — your manual act and its governance receipt use the same vocabulary from opposite ends.

## The future, honestly labelled

Whether Repo Governor should ever perform the strong acts itself — close a withdrawn issue, attach an admission the human already decided — is **open research, not a feature**: the governed-writeback question. The boundary it must hold: *write capability permits execution of an already-authorized transition; it never confers authority to choose the transition.* Until that work produces an Accepted ADR, every mutation on this page is yours.

## The forbidden shortcut

**Tidying as deciding.** Bulk-milestoning candidates to clean up the backlog *is admission* — of everything touched, all at once, without evaluating any of it. A board made tidy by mass-admission is a roadmap that no longer means anything; the untidy backlog of honest `NOT_ADMITTED` candidates is the system working.
