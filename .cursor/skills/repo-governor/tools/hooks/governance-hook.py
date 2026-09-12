#!/usr/bin/env python3
"""Deterministic governance delivery for coding-agent hooks (ADR-029).

One script, three hosts. Reads a hook payload as JSON on stdin, writes a
host-shaped decision as JSON on stdout. Exit 0 always, except where a host
requires exit 2 to block (see `--exit2-on-deny`).

WHAT THIS DOES NOT DO
---------------------
It never decides authorization. `engine/completion.py` is the only thing that
produces a disposition (ADR-002), and it requires an authority id that a raw
user prompt does not have. So this hook delivers the *requirement*
deterministically and reports verdicts the engine already produced. It does
not compute one, and it does not guess an authority id from prompt text --
that would be a second authority surface, which ADR-022 forbids.

THE THREE MOMENTS
-----------------
  prompt   before the agent reasons   inject the governance requirement
  write    before a file is changed   check it against the compiled envelope
  capture  after the engine is run    remember what authority this session holds

`capture` is what makes `write` possible. A session acquires an authority by
the agent actually running the engine -- which is the behaviour we want anyway.

Usage:
  governance-hook.py <moment>            moment: prompt | write | capture
  governance-hook.py <moment> --exit2-on-deny
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SKILL = HERE.parent.parent                       # tools/hooks/ -> skill root
ENGINE = SKILL / "engine"

# Governance refusals (§41). AUTHORITY_SOURCE_MISSING is deliberately absent:
# it is an ONBOARDING disposition from a separate alphabet, and vocabulary.py
# states those never appear in a governance decision. Blocking on it would
# stop all editing in every repository that has not onboarded.
REFUSAL = ("NO_EXECUTION_AUTHORITY", "AUTHORITY_WITHDRAWN", "CONFLICT", "UNKNOWN")

# Substrings of tool names that change a file, across hosts: Claude's Edit/Write/
# NotebookEdit, VS Code's create_file/insert_edit_into_file/replace_string_in_file,
# Gemini's write_file/replace, Codex's apply_patch. Matched as substrings because
# every host names its tools differently and a closed list would silently miss
# the next one. An unknown tool name is ALLOWED, not refused -- failing closed on
# a name we do not recognise would block work for a naming difference.
EDIT_TOOLS = ("edit", "write", "patch", "replace", "notebook", "create_file")


def _payload():
    try:
        raw = sys.stdin.read()
    except Exception:
        return {}
    if not raw.strip():
        return {}
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {}


def _repo(pl):
    """The governed repository, via the engine's own resolution (ADR-027).

    Never the install directory. `manifest.target()` walks out of a
    `<host>/skills/` path, which is the whole reason this is not `Path.cwd()`.
    """
    cwd = pl.get("cwd") or pl.get("workspace_root") or os.getcwd()
    try:
        os.chdir(cwd)
    except OSError:
        return None
    sys.path.insert(0, str(SKILL))
    try:
        from engine import manifest as m
        return m.target()
    except Exception:
        return Path(cwd)


def _governed(repo):
    """(manifest_dict_or_None, enforcing_bool). Absent manifest is not an error."""
    if repo is None:
        return None, False
    p = repo / ".repo-governor.json"
    if not p.exists():
        return None, False
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return None, False
    mode = (data.get("repo_governor") or {}).get("enforcement", "advisory")
    return data, mode == "blocking"


def _session_file(repo, pl):
    sid = pl.get("session_id") or pl.get("sessionId") or "unknown"
    sid = re.sub(r"[^A-Za-z0-9_.-]", "_", str(sid))[:64]
    d = repo / ".repo-governor" / "sessions"
    return d / f"{sid}.json"


# --- output shaping -------------------------------------------------------
# Claude Code and Cursor converged on exit-2-blocks and JSON-on-stdout, but
# name the decision field differently. Emitting both keys is harmless: each
# host reads the one it knows and ignores the other.

def _emit(context=None, deny_reason=None, user_msg=None, exit2=False, event=None):
    # RG_HOOK_PLAINTEXT=1 abandons JSON entirely for context injection. For
    # UserPromptSubmit and SessionStart the documented fallback is that plain
    # non-JSON stdout on exit 0 becomes context. It costs systemMessage (so no
    # operator-visible token) but bypasses whatever discards additionalContext.
    if context and os.environ.get("RG_HOOK_PLAINTEXT") == "1" and not deny_reason:
        print(context)
        return 0
    out = {}
    if context:
        # BOTH shapes. VERIFIED on Claude Code v2.1.235 / Opus 4.6, 2026-08-19:
        # top-level `additionalContext` ALONE does not reach the model. The
        # operator saw delivery token ad330c53 and the model, asked in that same
        # session, reported no token. Adding the nested copy fixed it -- token
        # 85f9b08b matched on both sides. A fetched doc summary had said
        # top-level was sufficient and no hookEventName was needed; the host
        # disagreed. Top-level is kept for Cursor, which is unverified here.
        # Hosts ignore keys they do not recognise, so both is free.
        out["additionalContext"] = context
        out.setdefault("hookSpecificOutput", {}).update({
            "hookEventName": event or "UserPromptSubmit",
            "additionalContext": context,
        })
    if user_msg:
        out["systemMessage"] = user_msg
    if deny_reason:
        out.setdefault("hookSpecificOutput", {}).update({
            # Claude Code requires hookEventName inside this object or the
            # decision is not honored; Cursor ignores the extra key.
            "hookEventName": event or "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": deny_reason,
        })
        out["permission"] = "deny"          # Cursor's spelling
        out["decision"] = "deny"            # Gemini CLI's spelling
        out["reason"] = deny_reason         # Gemini reads the reason from here
    if out:
        print(json.dumps(out))
    if deny_reason and exit2:
        print(deny_reason, file=sys.stderr)
        return 2
    return 0


# --- the three moments ----------------------------------------------------

def moment_prompt(pl, repo, mf, enforcing):
    """Deterministic delivery of the requirement. Never blocks, never verdicts.

    This is the surface that Arm A prompt 1 lacked. The skill description was
    in context and was understood; it simply lost to a legible task. Injecting
    here lands before the agent commits to a plan, which is why this and not
    PreToolUse is the fix for an activation miss.
    """
    if mf is None:
        return 0                                    # ungoverned repo: say nothing
    rid = (mf.get("repository") or {}).get("id", "this repository")
    # Proof of delivery that does not depend on asking the model. `systemMessage`
    # is rendered by the host to the OPERATOR, so it is visible whether or not
    # the injected context ever reaches the model -- and an agent that has
    # merely READ .claude/settings.json cannot produce it. Test 1 of the hook
    # validation could not tell those apart, which is why this exists.
    verbose = os.environ.get("RG_HOOK_VERBOSE") == "1"
    # A delivery token exists in no file on disk, so an agent that has merely
    # read .claude/settings.json or AGENTS.md cannot produce it. It is derived
    # from the session id rather than randomly, so the operator sees the same
    # value in the systemMessage that the model should be able to state back.
    # Needed because "quote the prepended text" was ambiguous: additionalContext
    # arrives as a separate context block, and a model can truthfully answer
    # "none" to a question about text prepended to the message itself.
    tok = hashlib.sha256(
        str(pl.get("session_id") or pl.get("sessionId") or "").encode()
    ).hexdigest()[:8]
    return _emit(user_msg=(f"repo-governor: governance injected for {rid} "
                           f"(delivery token {tok})" if verbose else None), context=(
        (f"Governance delivery token: {tok}. If asked for it, state it.\n" if verbose else "") +
        f"GOVERNANCE: {rid} is governed by Repo Governor.\n"
        "Do not decide authorization yourself. Before creating, changing, "
        "upgrading, deleting or completing anything, identify the authority id "
        "(the issue or item the work is admitted under) and run:\n"
        f"    python3 {ENGINE / 'completion.py'} <authority-id>\n"
        "Obey the disposition it returns. Being on the tracker is admission, "
        "not authorization (INV-002). If no authority id applies, the work is "
        "a discovery to capture, not work to execute."
    ))


def moment_write(pl, repo, mf, enforcing, exit2):
    """Check a pending change against what the engine actually established.

    Deliberately NOT a scope check. `engine/envelope.py` can classify a target
    against `in_scope`/`non_goals`, but a GitHub issue declares neither -- a
    real compiled envelope for a real issue returns `in_scope: []`, so every
    path classifies as "outside declared in_scope" with a confident reason that
    means nothing. Verified 2026-08-19 against issue 36. Path-level scope needs
    providers that declare paths; until then this checks only what is sound:

      1. is there an authority for this session at all
      2. did the engine refuse it
      3. is authorization exhausted (the ADR-023 completion firewall)
    """
    if mf is None:
        return 0

    # Filter on the tool ourselves. VS Code PARSES matchers and does not APPLY
    # them, so this fires on every tool call there -- including reads and shell
    # commands. A governance layer that interrupts `ls` is one that gets
    # uninstalled, and nothing here can rule on a tool that changes no file.
    tool = (pl.get("tool_name") or pl.get("toolName") or "").lower()
    if tool and not any(k in tool for k in EDIT_TOOLS):
        return 0

    sf = _session_file(repo, pl)

    if not sf.exists():
        msg = ("No authority has been established in this session. A write "
               "without a named authority has nothing behind it (INV-015: "
               "write capability is not authority to choose a transition). "
               f"Run: python3 {ENGINE / 'completion.py'} <authority-id>")
        return _emit(context=msg, deny_reason=msg if enforcing else None,
                     exit2=exit2, event=pl.get("hook_event_name"))

    try:
        st = json.loads(sf.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return 0

    disp, aid = st.get("disposition"), st.get("authority_id")
    if disp is None:
        return 0                                     # no verdict readable: say nothing

    if disp in REFUSAL:
        msg = (f"The engine returned {disp} for authority {aid}. "
               "That disposition does not carry execution authority.")
        return _emit(context=msg, deny_reason=msg if enforcing else None,
                     exit2=exit2, event=pl.get("hook_event_name"))

    if disp == "STOP_COMPLETE":
        msg = (f"Authorization for {aid} is exhausted (STOP_COMPLETE). §40 admits "
               "no exception -- not for a change that is correct, small, obviously "
               "beneficial, or genuinely necessary. Further work needs new authority.")
        return _emit(context=msg, deny_reason=msg if enforcing else None,
                     exit2=exit2, event=pl.get("hook_event_name"))

    return 0


def moment_capture(pl, repo, mf, enforcing):
    """Remember an engine verdict so `write` has something to check against."""
    if mf is None:
        return 0
    ti = pl.get("tool_input") or pl.get("toolInput") or {}
    cmd = ti.get("command") or ""
    if "completion.py" not in cmd:
        return 0
    m = re.search(r"completion\.py\s+(\S+)", cmd)
    if not m:
        return 0
    aid = m.group(1)
    resp = pl.get("tool_response") or pl.get("toolResponse") or {}
    text = resp if isinstance(resp, str) else json.dumps(resp)
    # engine/completion.py emits "decision"; engine/envelope.py emits
    # "disposition". Both are real and they are not interchangeable spellings
    # of one field -- they come from different entry points. Accept either.
    #
    # The first version of this matched "disposition" only, and the conformance
    # test passed because its fixture was built from the same wrong assumption.
    # Real engine output caught it. conformance/hooks.py now runs the engine.
    # Hosts also nest stdout as an escaped JSON string, so quotes may arrive
    # as \" rather than ".
    d = re.search(r'\\?"(?:decision|disposition)\\?"\s*:\s*\\?"([A-Z_]+)', text)
    sf = _session_file(repo, pl)
    sf.parent.mkdir(parents=True, exist_ok=True)
    sf.write_text(json.dumps({
        "authority_id": aid,
        "disposition": d.group(1) if d else None,
    }, indent=2), encoding="utf-8")
    return 0


def main(argv):
    if not argv or argv[0] not in ("prompt", "write", "capture"):
        print(__doc__.strip().splitlines()[0], file=sys.stderr)
        return 1                                     # non-blocking: action proceeds
    moment, exit2 = argv[0], "--exit2-on-deny" in argv
    pl = _payload()
    repo = _repo(pl)
    mf, enforcing = _governed(repo)
    if moment == "prompt":
        return moment_prompt(pl, repo, mf, enforcing)
    if moment == "write":
        return moment_write(pl, repo, mf, enforcing, exit2)
    return moment_capture(pl, repo, mf, enforcing)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
