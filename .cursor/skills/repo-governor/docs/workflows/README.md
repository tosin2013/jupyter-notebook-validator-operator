# Working with an assistant under Repo Governor

Prompt and workflow recipes for using a coding assistant in a governed repository. [`SKILL.md`](../../SKILL.md) tells the *assistant* how to run the engine; these pages tell the *human* what to ask for — and what to tell the assistant not to do.

The prompts here are **shapes, not incantations**. Adapt the wording; keep the constraints. Every recipe carries an explicit "do not" clause, because the failure mode these exist to prevent is an assistant doing something reasonable-looking without authority — and assistants are at their most dangerous when being helpful.

## The six lanes

Not everything is a feature. Treating every signal as roadmap input is how a roadmap becomes a landfill. Each page below serves one lane:

| Lane | Typical trigger | Default consequence | Page |
|---|---|---|---|
| **Product / Capability** | new feature, enhancement, discovered capability | candidate; never automatically admitted | [discovering-work](discovering-work.md) |
| **Compatibility / Dependency** | new dependency version, EOL, API/runtime change | signal → impact assessment | [dependency-updates](dependency-updates.md) |
| **Reliability / Defect** | bug, regression, operational defect | defect candidate; urgency can affect admission, not replace it | [bugs](bugs.md) |
| **Architecture / Platform** | ADR conflict, platform change, migration need | architecture review / candidate | [architecture-changes](architecture-changes.md) |
| **Retirement / Simplification** | old code, deprecated path, obsolete feature flag | retirement candidate; evidence required | [retirement](retirement.md) |
| **Research Candidate** | uncertainty requiring investigation | research candidate; not implementation | folded into [discovering-work](discovering-work.md) |

Plus the two bookends of any authorized task: [starting-work](starting-work.md) and [finishing-work](finishing-work.md), and the human-side page, [roadmap-maintenance](roadmap-maintenance.md).

## Signals are not authority

A signal — a new version, a discovered capability, a bug, an unused-looking module, a new ADR implication, an open question — creates or updates **candidate information**. It never changes what is admitted.

Roadmap authority changes only through the admission lifecycle ([`references/lifecycles.md`](../../references/lifecycles.md)):

```
DISCOVERED → CAPTURED → EVALUATED → ADMITTED → AUTHORIZED → EXECUTING → COMPLETED
```

or its one legal reversal, `AUTHORIZED → WITHDRAWN`.

The distinction matters most for anything that *writes*. Capturing knowledge (filing an unmilestoned issue, recording a candidate) is a weak act available broadly. Mutating lifecycle state (attaching a milestone, closing as `NOT_PLANNED` or `COMPLETED`) is a strong act that requires an existing authorized decision. Repo Governor currently performs **neither** on your behalf — roadmap mutation is manual, and the research toward changing that is tracked in the governed-writeback issue. See [roadmap-maintenance](roadmap-maintenance.md).

## What every page assumes

- The assistant has located the skill directory (`$RG`) and is standing in the repository under governance — [`SKILL.md`](../../SKILL.md) § *Where to run it*.
- Work items are the ids of the bound roadmap provider. In a repository using GitHub with the `milestone` admission signal, that means **issue numbers**, and an unmilestoned issue reads `NOT_ADMITTED`.
- Dispositions mean what [`references/dispositions.md`](../../references/dispositions.md) says they mean. These pages do not redefine them.
