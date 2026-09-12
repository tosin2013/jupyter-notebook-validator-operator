# Installed as a skill

This is a clone of [Repo Governor](https://github.com/tosin2013/repo-governor)
with three paths removed by `tools/install-skill.sh`:

| Removed | Why |
|---|---|
| `AGENTS.md` | says *"this repository is governed by Repo Governor"* — true of Repo Governor, not of the repository you installed it into. Cursor injects nested `AGENTS.md` files as always-on workspace rules. |
| `CLAUDE.md` | loader shim for the above |
| `.claude/` | carries an unrelated skill that recursive skill discovery would offer |
| `.repo-governor.json`, `.repo-governor/` | bind and configure governance for *Repo Governor's own repository*. Left in place, an agent standing in this directory resolves the install as the repository under governance and answers questions about the wrong project. |
| `CONTRIBUTING.md` | Repo Governor's contribution rules — its conformance suites, its branch policy, its PR template. True of Repo Governor, false of the repository you installed it into, and grepping for contribution rules would otherwise turn up two files describing different projects. To contribute an adapter, see https://github.com/tosin2013/repo-governor/blob/main/CONTRIBUTING.md |
| `docs/research/` | this project's working notes. `SKILL.md` reads `docs/workflows/` and `docs/reference/` and never these. One of them is the protocol for measuring whether this skill activates — shipping it means an agent being measured can read the experiment it is part of. |

`git status` here shows them as deletions. That is expected. To update:

```sh
git -C . stash && git -C . pull && git -C . stash pop
```

The engine governs the repository you are standing in, not this directory
(`REPO_GOVERNOR_TARGET`, ADR-027).
