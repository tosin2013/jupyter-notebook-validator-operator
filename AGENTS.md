# Agent instructions

This repository is governed by Repo Governor.

Before implementing work from a GitHub issue, run the engine and follow its disposition:

```bash
python3 .cursor/skills/repo-governor/engine/completion.py <issue-number>
```

Admitted work is GitHub Issues that belong to a milestone. An issue with no milestone is not admitted.

The skill lives at `.cursor/skills/repo-governor/`. Cursor hooks in `.cursor/hooks.json` inject this requirement at prompt time and check writes; they are advisory, not blocking.

See `.repo-governor.json` for provider bindings.
