---
name: repo-governor
description: Determine whether an AI coding agent is authorized to create, change, maintain, or retire something in this repository, and when it must stop. Use before implementing a feature, refactoring, upgrading a dependency, deleting code, acting on a TODO or discovery, or when asked whether work is authorized, in scope, or complete.
---

Do not decide authorization yourself. Run the engine and obey its disposition.

Repo Governor answers one question — *is this work currently authorized, and what may be done under that authorization?* — by reconciling state from bound providers. The answer is computed by a deterministic program, not inferred from this prose. Where this file and the code disagree, **the code is authoritative**.

## The rule everything rests on

> **Information may justify a decision. Information does not acquire authority merely by existing.**

A TODO, a `READY` task, a new dependency release, an unused-looking module, a green build — each is *evidence*. None is permission.

## Where to run it

**Stand in the repository you are governing. Invoke the engine by its full path.**

The engine governs the repository you are standing in, not the directory it lives in — so `cd`-ing into this skill to make a relative path work would govern the skill instead of your repository. That is a real defect this project shipped and fixed (ADR-027); do not recreate it.

You need the skill's own location. Take it, in this order:

1. the base directory your host gave you when it loaded this skill — most hosts state it;
2. the directory this `SKILL.md` was read from;
3. failing both, **ask**. Do not guess a path, and do not `cd` here to avoid the question.

```bash
RG=/path/to/repo-governor          # the skill's directory, from above
test -f "$RG/SKILL.md" || echo "RG is not the skill directory: '$RG'"
cd /path/to/the/repository/you/are/governing
```

**Check it.** An unset `$RG` expands to nothing, so `"$RG/engine/manifest.py"` becomes `/engine/manifest.py` — which fails as a missing *file* rather than a missing *variable*, and reads like the engine is not installed. That happened in a real session, and the agent carried on past it.

Every command below assumes `$RG` is set and that you are standing in the target repository.

## Before anything else

```bash
python3 "${RG:?set RG to the skill directory — see above}/engine/manifest.py"   # is this repository governed?
```

- **`MANIFEST VALID`** → governed. Continue.
- **`AUTHORITY_SOURCE_MISSING`** → not onboarded. It names the path it looked in — check that it is the repository you meant. Run `python3 "$RG/engine/onboard.py" .` and stop; binding requires a human.
- **`MANIFEST INVALID`** → refuse to evaluate. Report the errors. Do not guess.

A GitHub-backed role must declare its repository in the manifest binding — `env.REPO_GOVERNOR_GH_REPO` — because identity is never defaulted (ADR-028). An adapter that cannot tell which repository it is reading refuses rather than guessing.

## Ask the engine

```bash
python3 "$RG/engine/completion.py" <work-id>
```

Returns JSON with a `decision`. Obey it:

| Decision | What you do |
|---|---|
| `CONTINUE` | Work is authorized and unfinished. Proceed **within scope**. |
| `STOP_COMPLETE` | Acceptance conditions are satisfied. **Stop.** Capture discoveries; do not continue. |
| `NO_EXECUTION_AUTHORITY` | Admitted to the roadmap but not cleared to execute. Do not start. |
| `AUTHORITY_WITHDRAWN` | Cancelled or rejected. Stop, even if a task tracker says `READY`. |
| `UNKNOWN` | Read `unknowns[]`. If any has `blocking: true`, stop and report it. Non-blocking unknowns do not gate work. |
| `CONFLICT` | Two providers disagree as peers. Stop; a human selects. |

Every `unknown` carries `reason`, `dimension`, `blocking`, and a human-readable `resolution`. Report the resolution rather than working around it.

### Declaring a completion bar

`STOP_COMPLETE` is only reachable when the work item has a bar — one file at `.repo-governor/acceptance/<id>.json`. Scaffold it; do not hand-write the first one:

```bash
python3 "$RG/engine/acceptance.py" <id> --template
```

A criterion is exactly **`{check, target}`**. `check` is one of `tests_pass`, `file_exists`, `command_exit`; `target` is the path for `file_exists` and the command string for the other two. Any other key — `command`, `description` — is refused as `MALFORMED_SOURCE`, because a key that is never read makes the bar claim something it does not check. Prose goes in `$comment`.

```json
{ "authority_id": "42",
  "criteria": [ { "check": "file_exists", "target": "adapters/foo" },
                { "check": "command_exit", "target": "python3 conformance/foo.py" } ] }
```

The full contract, including `covers` for a bar that deliberately leaves part of the item out, is `schemas/acceptance-v1.json`. **Never write a criterion you have not seen fail** — an empty or unfalsifiable bar reads `NO_CRITERIA_DECLARED`, never satisfied (§40).

## Linear with MCP transport

When the manifest declares `roadmap_authority` as Linear with `"transport": {"kind": "mcp"}`, the engine cannot reach Linear on its own — the engine never calls MCP (ADR-016). The **agent** bridges the gap by fetching from Linear MCP and supplying the data to the engine through the environment.

**Step 1 — fetch.** Call the Linear MCP server's `list_issues` tool with `fields: ["id", "title", "status", "statusType"]`. These four fields are required; a payload missing any of them is refused as `MALFORMED_SOURCE`.

**Step 2 — cache.** Write the JSON response to a temporary file.

**Step 3 — evaluate.** Run `completion.py` with `REPO_GOVERNOR_LINEAR_FIXTURE` pointing at that file:

```bash
REPO_GOVERNOR_LINEAR_FIXTURE=/tmp/linear-issues.json python3 "$RG/engine/completion.py" <work-id>
```

The adapter reads the fixture, normalizes the MCP payload (same code path `tools/live-equivalence.py` uses), and returns a typed verdict. No `LINEAR_API_KEY` is needed.

**Hooks and headless contexts** cannot see the agent's MCP session. If hooks are installed, the hook subprocess will report `PROVIDER_UNAVAILABLE` with a message naming the missing input. That is correct — hooks deliver the governance requirement; they do not compute verdicts. The agent's `completion.py` run is where the verdict is produced.

**Do not** default to MCP because a server happens to be connected (INV-014). The transport is declared in the manifest, not inferred.

## Four invariants that always apply

These hold at every profile, including a nearly empty repository. The other ten load with the governance profile — see `references/invariants.md`.

- **INV-001 — Discovery confers no authority.** Finding a bug, a refactor, a cleanup, or an obvious improvement does not make it executable work. Default disposition is `CAPTURE_ONLY`.
- **INV-009 — Completed scope means stop.** When acceptance conditions are met, stop. Not "stop after this one small thing."
- **INV-010 — No illegal transitions.** `DISCOVERED → EXECUTING`, `VERSION_SIGNAL → UPGRADE`, and `SUSPECTED_OBSOLETE → DELETE` are forbidden. Each needs admission first.
- **INV-012 — `UNKNOWN` is a valid answer.** Where authority, obligations, or compatibility cannot be resolved, the correct output is `UNKNOWN`. Do not resolve it by assuming.

## Discoveries

Anything you notice that is not the authorized work — a possible feature, a bug, technical debt, a retirement candidate — is a **discovery**. Record it; do not act on it.

`CAPTURE_ONLY` is the default and is a complete, correct outcome. Promoting a discovery requires separate admission through the roadmap provider.

Ask the engine rather than deciding yourself:

```bash
python3 "$RG/engine/envelope.py" <work-id> --discovery <TYPE>[:target] [--record]
```

It compiles the ScopeEnvelope from provider state and rules on the discovery. `--record` persists the capture through the decision-history provider, idempotently.

**Two limits worth knowing.** Necessity is a claim you make (`--necessary`) and the engine *substantiates against declared scope* — an unsupported claim fails closed to `CAPTURE_ONLY`, so it is not a password that unlocks work. And once acceptance conditions are satisfied, **nothing** converts to execution: not a bug, not a necessary change, not a three-line fix (§40). ADR-024 is `Accepted` — its last acceptance condition, a measurement on repositories this project does not own, was answered across six of them.

## Before deleting anything

```bash
python3 "$RG/engine/retirement.py" <path>
```

`REMOVAL_READY` requires every obligation dimension resolved and clear. Static analysis alone **can never reach it** — dynamic loading, runtime usage, public contracts and migration obligations are invisible to grep and return as blocking unknowns. A `RETIREMENT_REVIEW` on an asset with zero references is the correct, expected result, not a false positive.

## Onboarding a repository

```bash
python3 "$RG/engine/onboard.py" <path>            # assess condition, detect candidates — READ ONLY
```

**Stop there and report.** `--write` creates `.repo-governor.proposed.json` in the target, and creating a file in a repository is a change to that repository — deny-by-default applies to it like anything else. Run it only when a human asks for the proposal to be written:

```bash
python3 "$RG/engine/onboard.py" <path> --write    # only on request
```

"Proposing" names what the file means, not what writing it costs. An agent asked to fix a parser that answers by leaving a governance artifact in the repository root has changed the repository it just declined to change.

Detection **proposes**. It never binds. Promoting the proposal to `.repo-governor.json` is a human action, and the engine never reads the proposal file. Two candidates for a single-valued role produce `PROVIDER_CONFLICT` and onboarding halts — no ranking is applied, because any automatic tie-break would silently confer authority.

```bash
python3 "$RG/engine/manifest.py" --validate       # do bound adapters satisfy their declared contracts?
```

## When the situation matches a lane, read its page first

`docs/workflows/` carries per-situation recipes. They are written for the human, but **their do-not clauses bind you** — and a request rarely arrives labelled, so match the situation, not the wording:

| The human says something like | Read |
|---|---|
| "work on issue N", "can you just add…" | `docs/workflows/starting-work.md` |
| — or you noticed something worth doing mid-task | `docs/workflows/discovering-work.md` |
| "upgrade X", "there's a new version / CVE" | `docs/workflows/dependency-updates.md` |
| "fix this bug" | `docs/workflows/bugs.md` |
| "this conflicts with ADR-N", "change the design" | `docs/workflows/architecture-changes.md` |
| "delete this", "this looks unused" | `docs/workflows/retirement.md` |
| the work seems done | `docs/workflows/finishing-work.md` |
| "clean up the backlog", "reconcile the roadmap" | `docs/workflows/roadmap-maintenance.md` |

Each page names the forbidden shortcut its lane tempts. If the human's request *is* that shortcut — "just delete it", "just upgrade", "while you're in there" — the page's constraint still applies: say what the lane requires instead of silently complying or silently refusing.

## Load these only when you need them

| File | Read it when |
|---|---|
| `references/invariants.md` | all fourteen invariants, and which profile activates each |
| `references/dispositions.md` | full disposition and unknown-reason semantics |
| `references/providers.md` | writing an adapter, or a provider is behaving oddly |
| `references/lifecycles.md` | admission, maintenance or retirement state machines |
| `docs/reference/` | any `§NN` citation; start at its section map |

## What this never does

Repo Governor returns a **verdict**. It does not create, change, or delete anything, and it does not write to your tracker. It has no permission it was not explicitly granted in the manifest — an available credential grants nothing.

If it says stop, that is the product working. Continuing past `STOP_COMPLETE` because the next thing looks small is the specific failure this exists to prevent.
