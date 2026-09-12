# Transcript fixtures

Every other fixture in this repository is hand-written by whoever wrote the
parser, which means only the shapes that person imagined are covered. That is
how issue 109 shipped: `observe()` handled `message` being `None` or a `dict`
because those were imagined, and crashed on a `str` because it was not.

## `claude-stream-json.jsonl`

**Real.** Captured with `claude -p "..." --output-format stream-json --verbose`
on 2026-08-20, Claude Code, `claude-opus-5[1m]`.

Scrubbed: `session_id`, `uuid`, `cwd`, `messaging_socket_path`, `apiKeySource`,
`memory_paths` replaced with `<scrubbed>`. Long `init` lists (`skills`,
`tools`, `slash_commands`, `agents`, `plugins`, `mcp_servers`, `capabilities`)
trimmed to their first three entries. **Types and structure are untouched** —
that is the whole value of the file.

It is genuinely a **VOID** transcript by issue 104's rule: the session ran in a
repository where `repo-governor` was not an installed skill, so the listing
does not contain it. Nothing was removed to make that true.

Confirmed from it: `init` **does** carry `skills` on this host, so the
precondition is verifiable here rather than permanently UNVERIFIED.

## `malformed.jsonl`

**Not real. Reconstructed from a traceback**, and that distinction matters.

A real calibration run died at 228 seconds on `AttributeError: 'str' object
has no attribute 'get'`, and the transcript that would have shown the exact
offending event was destroyed by the crash — which is the second defect issue
109 fixes. So these shapes are what the traceback *implies* plus the
neighbouring assumptions the same code makes.

**If a future run still crashes, this file was wrong, not the guard.** Capture
that transcript and add it here.

The last line is deliberately not JSON: a transcript can be truncated
mid-write, and the parser must skip it rather than die on it.
