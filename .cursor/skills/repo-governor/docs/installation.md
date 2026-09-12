# Installing Repo Governor on a host

Repo Governor ships as an [Agent Skill](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) ([ADR-001](adrs/001-agent-skill-as-primary-delivery-surface.md)). There is no installer and no service — the skill is a directory, and installing it means putting that directory where your agent host looks for skills.

**Which directory that is depends on the host, and getting it wrong is silent.** A host that cannot see the skill behaves exactly like a host that saw it and decided not to use it. That distinction matters enough that the [activation protocol](research/activation-protocol.md) treats an unverified install as no measurement at all.

## Where to put it

`.agents/skills/` is read by more than one vendor and is the closest thing to a neutral location. This repository already uses the same pattern one level up: `AGENTS.md` holds the content and `CLAUDE.md` is a one-line pointer to it, because [ADR-001](adrs/001-agent-skill-as-primary-delivery-surface.md) makes tool-independence the thesis rather than a preference.

```bash
git clone --branch v0.6.0 https://github.com/tosin2013/repo-governor /tmp/repo-governor
/tmp/repo-governor/tools/install-skill.sh <target-repo>
```

**Use the script rather than cloning straight into the skills directory.** A plain clone puts this repository's own `AGENTS.md` inside your project, and it opens *"This repository is governed by Repo Governor"* — a true statement about Repo Governor and a false one about yours. Cursor was observed injecting that file as an always-on workspace rule from inside the skill directory of an unrelated project, which hands your agent our house rules and tells it your repository is governed by something it has not agreed to. The script clones and then removes the three paths that are correct here and wrong anywhere else; see `INSTALLED.md` in the result.

Then add whatever pointer your host needs beside it:

| Host | Reads | Status |
|---|---|---|
| **Claude Code** | `.claude/skills/<name>/SKILL.md` | **verified** — used throughout this repository's own development |
| **Cursor** | `.agents/skills/`, `.cursor/skills/`, `~/.agents/skills/`, `~/.cursor/skills/` | **verified** 2026-08-18 — `.agents/skills/repo-governor/SKILL.md` was listed by a fresh Cursor session with its description ([#38](https://github.com/tosin2013/repo-governor/issues/38)) |
| **Codex** | `.agents/skills/`, `.codex/skills/`, `~/.codex/skills/` | from vendor docs; **unverified here** |

Codex is still vendor documentation rather than knowledge; [#37](https://github.com/tosin2013/repo-governor/issues/37) is the run that settles it. Cursor was settled by [#38](https://github.com/tosin2013/repo-governor/issues/38) and the docs were right.

**`.claude/skills/` is wrong for Cursor and Codex**, which is worth stating plainly because it fails silently: the files are there, the host sees nothing, and the skill looks like it declined to activate rather than like it was never found.

For a host that only reads a vendor-specific path, point it at the one copy rather than keeping two:

```bash
mkdir -p .claude/skills
ln -s ../../.agents/skills/repo-governor .claude/skills/repo-governor
```

Two real copies drift, and a drifted skill fails in the way that is hardest to notice: it works, and it is answering from the wrong version.

## What a plain clone drags in, and why the script exists

Verified on Cursor, 2026-08-18. Asked which instruction files were applied to a workspace containing an unrelated project, it listed **six**, of which four came from installed skill copies:

```
<target>/.agents/skills/repo-governor/AGENTS.md      <- injected as an always-on rule
<target>/.agents/skills/repo-governor/CLAUDE.md      <- injected as an always-on rule
```

So on that host, installing Repo Governor by cloning applies **Repo Governor's own house rules to the user's repository**, including a first line asserting that repository is governed by us. Nobody asked for that, and it is invisible unless you think to ask the host what it loaded.

`tools/install-skill.sh` removes three paths after cloning:

| Removed | Why |
|---|---|
| `AGENTS.md` | the assertion above |
| `CLAUDE.md` | loader shim for it |
| `.claude/` | carries `github-project-release-manager`, an unrelated skill that recursive discovery offers as available — also observed on Cursor |

The engine still runs from the pruned copy; none of the removed files are read by it.

**For [measuring activation](research/activation-protocol.md) this is not merely untidy, it is fatal** — an Arm A target that has been told it is governed is Arm B. The protocol carries the test.

## Verify the host can actually see it

**Do this before concluding anything about whether the skill activates.** The install is not finished when the files are in place; it is finished when the host demonstrates it found them.

```bash
test -f .agents/skills/repo-governor/SKILL.md && echo "SKILL.md present"
head -5 .agents/skills/repo-governor/SKILL.md          # name + description frontmatter
```

Files being present is necessary and not sufficient. Then, in the host itself:

1. **Start a new session.** Skills are discovered when a session starts — Cursor documents this explicitly. A skill added mid-session is invisible until you open a new chat, and judging activation from that session measures nothing.
2. Ask the host to list the skills it can see.
3. Confirm `repo-governor` is among them, with its description.

If it is not listed, the path is wrong for that host. Fix the path before running anything that depends on the skill being available.

## Dependencies

The engine is Python stdlib only ([ADR-011](adrs/011-python-stdlib-only-engine-with-language-agnostic-adapters.md) rule 1). Adapters may carry dependencies (rule 4), and one of them matters:

| Tool | Needed for | If absent |
|---|---|---|
| `python3` **3.11+** | the engine ([ADR-011](adrs/011-python-stdlib-only-engine-with-language-agnostic-adapters.md) declares the floor) | nothing works |
| `git` | provider resolution, target detection | nothing works |
| `dolt` | `decision_history` via `adapters/dolt-decisions` | **4 of 19 conformance suites fail** |
| `gh`, authenticated | the GitHub roadmap provider | live GitHub queries fail; offline suites are unaffected |

**`dolt` is the one that misleads.** Without it, `layer1`, `layer2`, `bindings` and `execution` all fail — including the portability thesis test, which reports `NOT EQUIVALENT`. That reads like a real result and is not one. The suites print a preflight line naming the missing binary, and `tools/bootstrap-decisions.sh` refuses before you reach them, but never report a red verdict from a box without `dolt`.

```bash
./tools/bootstrap-decisions.sh
./tools/run-conformance.sh
```

Expect 19/19 from a fresh clone.

## Governing a repository other than this one

The skill governs the repository you are standing in, not the one it is installed from. `REPO_GOVERNOR_TARGET` overrides the detection ([ADR-027](adrs/027-the-governed-repository-is-not-the-install-directory.md)):

```bash
REPO_GOVERNOR_TARGET=/path/to/governed/repo python3 engine/completion.py <id>
```

Installing the skill into a target's `.agents/skills/` puts a full clone of this repository inside it — including this repository's `AGENTS.md`, which announces that *this* repo is governed. On a host that loads nested agent-instruction files, that statement leaks into the target. It is harmless in normal use and fatal to an Arm A activation measurement; the [activation protocol](research/activation-protocol.md) carries the check.

## Hooks — deterministic delivery (optional, [ADR-029](adrs/029-hooks-as-deterministic-delivery-surface.md))

**Optional, and most repositories should not install it.** Everything below is recorded in [`hook-validation-results.md`](research/hook-validation-results.md); read that before deciding.

The hook was built to fix an activation miss ([#36](https://github.com/tosin2013/repo-governor/issues/36) Arm A prompt 1 explored for twelve tool calls and then wrote test files into the repository, never consulting governance across all fifty-two; and the industry baseline is roughly **half** of skill invocations never firing). **Validation refuted that purpose.** Same repository, same prompt, hook on and hook off — both graded FULL, and the agent named `AGENTS.md` as its source every time.

| You have | Install the hook? |
|---|---|
| `AGENTS.md` (or `CLAUDE.md` pointing at it) | **No.** Proven sufficient on its own; the hook adds nothing to activation. |
| A governed repository with no always-on file | Adding `AGENTS.md` is simpler and does the same job. |
| An un-onboarded repository | **It cannot help.** The hook is silent without a manifest, by design. |
| A need to *stop* a write, not advise against one | **This is the only reason.** See *Blocking mode* — and note it is untested. |

`AGENTS.md` is prose: it cannot stop anything. `PreToolUse` with exit 2 can. That is the hook's one unrefuted justification, and it is why ADR-029 is still `Proposed`.

**If you want governance to activate reliably, write an `AGENTS.md`. That is the finding.**

| Host | Config file | Prompt-time event | Verified on a real host? |
|---|---|---|---|
| **Claude Code** | `.claude/settings.json` | `UserPromptSubmit` | **yes** — token matched operator and model, 2026-08-19 |
| Cursor | `.cursor/hooks.json` | `beforeSubmitPrompt` | no — [issue 50](https://github.com/tosin2013/repo-governor/issues/50) |
| Codex | `.codex/hooks.json` | **none exists** | no — [issue 47](https://github.com/tosin2013/repo-governor/issues/47) |
| Gemini CLI | `.gemini/settings.json` | `BeforeAgent` | no — [issue 48](https://github.com/tosin2013/repo-governor/issues/48) |
| VS Code / Copilot | `.github/hooks/*.json` | `UserPromptSubmit` | no — [issue 49](https://github.com/tosin2013/repo-governor/issues/49) |

**"Verified" means one specific thing**: someone installed it, asked the model for a delivery token with no tool calls, and the model stated the token the operator saw. Event names and exit codes taken from vendor documentation are *not* verification — on Claude Code the documented output shape was wrong, and every visible signal said the hook was working while the model received nothing.

Each unverified row has an issue explaining exactly what is unknown for that host and what to run. **A report that it does not work is more useful than another confirmation from the host we can already run.**

Codex documents **no prompt-submit event at all**, so only the write check installs there — the hook cannot deliver the requirement before the agent acts, and an `AGENTS.md` is the whole activation remedy on that host. Its project hooks also load only when `.codex/` is *trusted*, which otherwise looks identical to a hook doing nothing.

The installer offers it when the host is Claude Code and the target is **governed**:

```bash
tools/install-skill.sh <target> .claude/skills yes            # -> .claude/settings.json
tools/install-skill.sh <target> .cursor/skills yes            # -> .cursor/hooks.json
tools/install-skill.sh <target> .codex/skills  yes            # -> .codex/hooks.json
tools/install-skill.sh <target> .agents/skills yes cursor     # cross-vendor path:
                                                              # declare the harness
```

**The harness is declared, never inferred.** Naming `.claude/skills` declares it; the cross-vendor `.agents/skills` declares nothing, so the script asks — or, with no terminal, tells you how to declare it and installs the skill without a hook.

An earlier version guessed from whichever host directory the target happened to contain. A repository someone once opened in Cursor, whose owner runs Codex, would have received `.cursor/hooks.json` that Codex never reads — **indistinguishable from a hook that does not work**, which is the one failure this surface exists to make impossible. It is also the defect [ADR-028](adrs/028-provider-identity-is-never-defaulted.md) exists for: an identity guessed from incidental evidence.

Omit the third argument to be prompted; pass `no` to never ask. The host decides which config file is written, and the script uses the same template documented above rather than a second copy — a config the installer writes and one the docs describe must not be able to drift.

Defaults to **no**. Non-interactive runs never write. It merges into an existing `.claude/settings.json` rather than replacing it, and reports any of its own keys it overwrote. It installs **advisory** hooks only — blocking needs `enforcement: "blocking"` in the manifest *and* `--exit2-on-deny` on the write hook, neither of which the installer adds.

**It refuses outright in an un-onboarded repository**, because the hook cannot speak without a manifest and onboarding one purely to enable a hook ends any activation measurement running against it.

To configure it by hand instead:

```bash
RG=/absolute/path/to/installed/skill      # the directory containing SKILL.md

# Claude Code, project-scoped:
mkdir -p .claude
sed "s#RG_SKILL_DIR#$RG#g" "$RG/tools/hooks/claude.json" > /tmp/rg-hooks.json
# merge /tmp/rg-hooks.json into .claude/settings.json -- do not overwrite an
# existing settings file, it belongs to the project

# confirm it works before trusting it
echo "{\"session_id\":\"x\",\"cwd\":\"$PWD\"}" | python3 "$RG/tools/hooks/governance-hook.py" prompt
```

The last command prints an `additionalContext` block in a governed repository and **nothing at all** in an un-onboarded one. Silence there is correct, not a failure.

### What it does, and what it will not do

| Moment | Effect |
|---|---|
| every prompt | injects the requirement to run the engine before acting |
| before `Edit`/`Write` | reports if no authority was established, if the engine refused, or if authorization is exhausted (`STOP_COMPLETE`) |
| after a Bash call | records the authority id and disposition the engine returned |

It **never decides authorization** — `engine/completion.py` remains the only thing that produces a disposition — and it makes **no claim about file scope**. Roadmap providers do not declare paths; a compiled envelope for a real GitHub issue returns `in_scope: []`, so a path check there would refuse every write with a fabricated reason. See ADR-029's *What this deliberately does not do*.

### Blocking mode

Advisory by default: the hook speaks, the agent decides. To let it actually stop a write, the **governed repository** opts in:

```json
{ "repo_governor": { "version": 1, "enforcement": "blocking" } }
```

and the config adds `--exit2-on-deny` to the `write` hook. Both are required; the flag alone does nothing. Enforcement is per repository because an un-onboarded repository is not a governed one, and blocking there would stop all editing everywhere the manifest is absent.

### Proving the hook actually ran

An agent can read `.claude/settings.json` and describe the hooks convincingly without any of them having fired. Asking it what rules apply therefore does **not** distinguish delivery from inspection — observed 2026-08-19, where the agent listed all three hooks accurately after reading the config file, and the injected text never appeared.

Two checks that do discriminate:

```bash
# 1. operator-visible, independent of the model entirely
export RG_HOOK_VERBOSE=1     # host renders: "repo-governor: governance injected for <repo>"
```

```
# 2. ask the model for the delivery token, with no tool calls
Without reading any files or running any tools: is there a governance delivery
token in your context for this session? If so, state it.
```

In verbose mode the hook emits a token derived from the session id to **both** channels — the operator sees it in the `systemMessage` line, the model receives it in `additionalContext`. It exists in no file, so an agent that has read `.claude/settings.json` or `AGENTS.md` cannot produce it, and it changes every session so it cannot be memorised. Matching tokens prove end-to-end delivery.

> **Do not ask the model to "quote any text prepended to this message".** Tried 2026-08-19; the answer was "none" while the hook had demonstrably run. `additionalContext` arrives as a separate context block rather than as part of the user message, so "none" is a truthful answer to that question and tells you nothing. The token avoids the ambiguity by asking for a value instead of a description.

`RG_HOOK_VERBOSE` is off by default and stays silent in ungoverned repositories, verbose or not.

### If the model never receives the context

Observed 2026-08-19, Claude Code v2.1.235 / Opus 4.6: the operator saw `delivery token ad330c53` and the model, asked the same session, answered *"there is no governance delivery token in my context."* The hook ran; `systemMessage` arrived; `additionalContext` did not.

The hook now emits the context **twice** — top-level and nested inside `hookSpecificOutput` — because the two are documented differently and a live host disagreed with the summary we checked. Hosts ignore keys they do not recognise, so emitting both costs nothing and removes the guess.

If a token still does not reach the model, fall back to plain text:

```bash
export RG_HOOK_PLAINTEXT=1
```

For `UserPromptSubmit` and `SessionStart`, plain non-JSON stdout on exit 0 is added to context directly. This bypasses whatever discards `additionalContext`, at the cost of `systemMessage` — so there is no operator-visible token, and delivery is verified by asking the model to quote the `GOVERNANCE:` block instead. It stays silent in ungoverned repositories either way.

**The failure this guards against is specific: a hook that runs, reports success, and delivers nothing.** Without the token, the transcript looks identical to a working one.

## Onboarding a repository

`engine/onboard.py <repo> --write` surveys the filesystem and writes `.repo-governor.proposed.json`. That file is **evidence, not a manifest** — renaming it yields `UNSUPPORTED_VERSION: manifest version None`. That rename was the documented instruction until 2026-08-19 and had never been run end to end.

For a proposal that actually binds:

```bash
python3 tools/onboard-interactive.py /path/to/repo
```

It runs detection, shows the evidence, then asks the two things no amount of file-reading can answer:

- **which system is the roadmap authority** — GitHub, Linear, a file, or something else
- **what ADMITTED means there** — milestone, project column, label, or nothing

The second is [ADR-018](adrs/018-admission-signal-is-declared-not-assumed.md): the admission signal is *declared*, never assumed. Whether admission means a milestone or a label is a fact about how a team works, not about the repository. Guessing it produces an engine that governs confidently against the wrong roadmap — which has happened twice here ([ADR-022](adrs/022-repo-governor-does-not-own-roadmap-state.md), and [ADR-028](adrs/028-provider-identity-is-never-defaulted.md) where adapters defaulted to the author's repository).

Output is deny-by-default ([ADR-005](adrs/005-deny-by-default-authority-resolution.md)): every bound role gets read, nothing gets write. Verify before binding:

```bash
mv .repo-governor.proposed.json .repo-governor.json
python3 <skill>/engine/manifest.py --validate      # want: READY_FOR_GOVERNANCE
```

If it fails, rename it back. Nothing governs until it passes.

**If your tracker is not one we support**, pick *"Something else"* — the tool prints a `gh issue create` line for an adapter request and points at `adapters/_protocol.py`, which is the whole contract. Every shipped adapter is a single file, and the engine never changes to accommodate one ([ADR-003](adrs/003-seven-provider-roles-with-normalized-contracts.md)).

## Does it actually work on your harness?

Installing the skill does not mean it will fire. Activation is model-mediated: published benchmarks put roughly half of skill invocations at never firing, and this project measured **20/20 on one host and 0/2 on another** using the same skill, the same prompts and the same repository.

```bash
python3 tools/selftest.py [repo-path]
```

The mechanical half — skill present, manifest valid, always-on file present — is checked for you and **proves nothing on its own**. The half that decides it cannot be scripted: whether your model consults governance before acting is a fact about your model. So the tool prints four prompts (three that should activate, one control that should not), tells you what each is testing, and what to do about each score.

Ten minutes, run in your own agent. If it scores below 3/3 the remedy is ordered by what our data supports: **add an `AGENTS.md` first** — it was the strongest single predictor we measured and every harness reads it — then consider the hook, which changed nothing where such a file already existed.

**A low score is worth reporting.** A result from a model nobody here can run is worth more than another high score from one we already have: [report it](https://github.com/tosin2013/repo-governor/issues/new?template=activation-result.yml).
