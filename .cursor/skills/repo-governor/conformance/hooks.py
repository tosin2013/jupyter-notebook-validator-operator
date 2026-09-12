#!/usr/bin/env python3
"""The hook is exercised on real payloads, because its defects only appear there.

Two were found this way while it was being written, and neither would have
survived contact with a fixture that shared the implementation's assumptions:

  1. the disposition regex missed every real payload, because hosts nest the
     engine's stdout as an ESCAPED JSON string -- `\\"disposition\\"`, not
     `"disposition"`. The capture silently recorded `null`.

  2. worse, a null disposition fell through to a scope check that returned
     "outside declared in_scope" for every path. A real compiled envelope for a
     real GitHub issue has `in_scope: []`, so that branch produced a confident
     refusal with a fabricated reason on every write. In blocking mode it would
     have blocked all editing while stating something untrue.

The second is why this suite asserts what the hook must NOT say as carefully as
what it must. A governance tool that refuses confidently for the wrong reason is
worse than one that stays quiet.

Usage:  python3 conformance/hooks.py
"""

from __future__ import annotations

import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile
from pathlib import Path

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("hooks")

ROOT = Path(__file__).resolve().parent.parent
HOOK = ROOT / "tools" / "hooks" / "governance-hook.py"
TEMPLATES = ROOT / "tools" / "hooks"

REFUSAL = ("NO_EXECUTION_AUTHORITY", "AUTHORITY_WITHDRAWN", "CONFLICT", "UNKNOWN")


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"    {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def run(moment, payload, *flags, cwd=None, env=None):
    e = None
    if env:
        import os
        e = {**os.environ, **env}
    p = subprocess.run([sys.executable, str(HOOK), moment, *flags],
                       input=json.dumps(payload), capture_output=True,
                       text=True, cwd=cwd or str(ROOT), env=e)
    try:
        out = json.loads(p.stdout) if p.stdout.strip() else {}
    except json.JSONDecodeError:
        out = {"__unparseable__": p.stdout}
    return p.returncode, out, p.stderr


def main():
    fails = 0
    print("HOOK CONFORMANCE\n")

    fails += check("hook exists and is executable", HOOK.exists())
    if not HOOK.exists():
        return 1

    # --- silence where there is no authority to speak from -----------------
    with tempfile.TemporaryDirectory() as td:
        rc, out, _ = run("prompt", {"session_id": "t", "cwd": td}, cwd=td)
        fails += check("ungoverned repository: prompt moment says nothing",
                       rc == 0 and out == {}, f"rc={rc} out={out}")
        rc, out, _ = run("write", {"session_id": "t", "cwd": td,
                                   "tool_input": {"file_path": f"{td}/x.py"}}, cwd=td)
        fails += check("ungoverned repository: write moment never blocks",
                       rc == 0 and out == {}, f"rc={rc} out={out}")

    # --- the push surface, which is the whole point ------------------------
    rc, out, _ = run("prompt", {"session_id": "t", "cwd": str(ROOT)})
    ctx = out.get("additionalContext", "")
    fails += check("governed repository: prompt moment injects the requirement",
                   rc == 0 and bool(ctx))
    fails += check("injected text names the engine entry point",
                   "completion.py" in ctx)
    fails += check("injected text states admission is not authorization",
                   "INV-002" in ctx)
    fails += check("prompt moment never blocks, whatever it says",
                   rc == 0 and "permissionDecision" not in json.dumps(out))

    # --- operator-visible delivery proof -----------------------------------
    # An agent that has merely READ .claude/settings.json can describe the
    # hooks convincingly. systemMessage is rendered by the host to the
    # operator, so it distinguishes "the hook ran" from "the model inferred it".
    rc, out, _ = run("prompt", {"session_id": "t", "cwd": str(ROOT)})
    fails += check("delivery proof is off by default", "systemMessage" not in out)
    rc, out, _ = run("prompt", {"session_id": "t", "cwd": str(ROOT)},
                     env={"RG_HOOK_VERBOSE": "1"})
    fails += check("RG_HOOK_VERBOSE=1 emits an operator-visible systemMessage",
                   "systemMessage" in out and "governance injected" in out.get("systemMessage", ""),
                   f"got {out.get('systemMessage')!r}")
    # The token is the only discriminator that survives AGENTS.md being
    # auto-loaded: it exists in no file, so a model that states it can only
    # have received the injected context.
    import re as _re
    rc, out, _ = run("prompt", {"session_id": "abc123", "cwd": str(ROOT)},
                     env={"RG_HOOK_VERBOSE": "1"})
    sm = _re.search(r"delivery token (\w+)", out.get("systemMessage", "") or "")
    ac = _re.search(r"delivery token: (\w+)", out.get("additionalContext", "") or "")
    fails += check("verbose mode emits a delivery token to the operator", bool(sm))
    fails += check("verbose mode emits the SAME token to the model",
                   bool(ac) and bool(sm) and ac.group(1) == sm.group(1),
                   "operator and model must see one value or the test proves nothing")
    rc, out2, _ = run("prompt", {"session_id": "different", "cwd": str(ROOT)},
                      env={"RG_HOOK_VERBOSE": "1"})
    ac2 = _re.search(r"delivery token: (\w+)", out2.get("additionalContext", "") or "")
    fails += check("the token varies by session, so it cannot be memorised",
                   bool(ac2) and ac2.group(1) != ac.group(1))
    rc, out3, _ = run("prompt", {"session_id": "abc123", "cwd": str(ROOT)})
    fails += check("no token when RG_HOOK_VERBOSE is unset",
                   "token" not in json.dumps(out3).lower())

    with tempfile.TemporaryDirectory() as td:
        rc, out, _ = run("prompt", {"session_id": "t", "cwd": td}, cwd=td,
                         env={"RG_HOOK_VERBOSE": "1"})
        fails += check("verbose mode still says nothing in an ungoverned repository",
                       out == {}, f"got {out}")

    # --- both context shapes, because one alone did not reach the model ----
    rc, out, _ = run("prompt", {"session_id": "t", "cwd": str(ROOT),
                                "hook_event_name": "UserPromptSubmit"})
    hso = out.get("hookSpecificOutput") or {}
    fails += check("context is emitted at top level", bool(out.get("additionalContext")))
    fails += check("context is ALSO emitted inside hookSpecificOutput",
                   bool(hso.get("additionalContext")),
                   "top-level alone reached the operator but not the model, 2026-08-19")
    fails += check("nested block carries the event name it belongs to",
                   hso.get("hookEventName") == "UserPromptSubmit")
    fails += check("both copies are identical",
                   out.get("additionalContext") == hso.get("additionalContext"))

    # --- plaintext fallback -------------------------------------------------
    p = subprocess.run([sys.executable, str(HOOK), "prompt"],
                       input=json.dumps({"session_id": "t", "cwd": str(ROOT)}),
                       capture_output=True, text=True, cwd=str(ROOT),
                       env={**os.environ, "RG_HOOK_PLAINTEXT": "1"})
    fails += check("RG_HOOK_PLAINTEXT=1 emits bare text, not JSON",
                   p.returncode == 0 and p.stdout.strip().startswith("GOVERNANCE:")
                   and not p.stdout.strip().startswith("{"), repr(p.stdout[:60]))
    with tempfile.TemporaryDirectory() as td:
        p = subprocess.run([sys.executable, str(HOOK), "prompt"],
                           input=json.dumps({"session_id": "t", "cwd": td}),
                           capture_output=True, text=True, cwd=td,
                           env={**os.environ, "RG_HOOK_PLAINTEXT": "1"})
        fails += check("plaintext mode still silent in an ungoverned repository",
                       not p.stdout.strip(), repr(p.stdout[:60]))

    # --- capture: the escaped-quote defect ---------------------------------
    sess = ROOT / ".repo-governor" / "sessions"
    # The engine is RUN, not imitated. A hand-written fixture here passed while
    # the hook searched for "disposition" and completion.py emits "decision".
    real = subprocess.run([sys.executable, str(ROOT / "engine" / "completion.py"), "36"],
                          capture_output=True, text=True, cwd=str(ROOT)).stdout
    fails += check("engine produced output to capture from", bool(real.strip()))
    real_disp = json.loads(real).get("decision") if real.strip() else None

    for shape, label in (
        ({"stdout": real}, "real-engine-output-nested"),
        (real, "real-engine-output-plain"),
    ):
        sid = re.sub(r"[^A-Za-z0-9_.-]", "_", f"conf_{label}")
        run("capture", {"session_id": sid, "cwd": str(ROOT),
                        "tool_input": {"command": "python3 engine/completion.py 36"},
                        "tool_response": shape})
        f = sess / f"{sid}.json"
        got = json.loads(f.read_text())["disposition"] if f.exists() else None
        fails += check(f"capture reads the verdict from {label}",
                       got is not None and got == real_disp,
                       f"got {got!r}, engine said {real_disp!r}")

        # --- the fabricated-reason defect ----------------------------------
        rc, out, _ = run("write", {"session_id": sid, "cwd": str(ROOT),
                                   "tool_input": {"file_path": "README.md"}})
        said = json.dumps(out)
        fails += check(f"write under a refusal ({label}) names the real verdict",
                       real_disp in said if real_disp in REFUSAL else True)
        fails += check(f"write under a refusal ({label}) invents no scope claim",
                       "in_scope" not in said and "envelope" not in said,
                       "claimed a scope verdict the providers cannot support")
        if f.exists():
            f.unlink()

    # --- an unreadable verdict must produce silence, not a guess -----------
    sid = "conf_noverdict"
    (sess).mkdir(parents=True, exist_ok=True)
    (sess / f"{sid}.json").write_text(json.dumps({"authority_id": "36", "disposition": None}))
    rc, out, _ = run("write", {"session_id": sid, "cwd": str(ROOT),
                               "tool_input": {"file_path": "README.md"}})
    fails += check("no readable verdict: the hook stays quiet rather than guessing",
                   rc == 0 and out == {}, f"said {out}")
    (sess / f"{sid}.json").unlink()

    # --- advisory is the default, and it must not block --------------------
    sid = "conf_advisory"
    (sess / f"{sid}.json").write_text(json.dumps(
        {"authority_id": "36", "disposition": "NO_EXECUTION_AUTHORITY"}))
    rc, out, _ = run("write", {"session_id": sid, "cwd": str(ROOT),
                               "tool_input": {"file_path": "README.md"}}, "--exit2-on-deny")
    enforcing = (json.loads((ROOT / ".repo-governor.json").read_text())
                 .get("repo_governor", {}).get("enforcement", "advisory")) == "blocking"
    fails += check("advisory manifest: --exit2-on-deny alone does not block",
                   rc == 0 and "permissionDecision" not in json.dumps(out) if not enforcing else True,
                   f"rc={rc}; the flag must not override the manifest")

    # --- the completion firewall -------------------------------------------
    (sess / f"{sid}.json").write_text(json.dumps(
        {"authority_id": "5", "disposition": "STOP_COMPLETE"}))
    rc, out, _ = run("write", {"session_id": sid, "cwd": str(ROOT),
                               "tool_input": {"file_path": "README.md"}})
    fails += check("STOP_COMPLETE is surfaced before a write (ADR-023)",
                   "STOP_COMPLETE" in json.dumps(out))
    (sess / f"{sid}.json").unlink()

    # --- AUTHORITY_SOURCE_MISSING is not a governance refusal --------------
    # Tested by BEHAVIOUR, not by reading the constant. The first version of
    # this check compared the suite's own copy of REFUSAL against itself and
    # survived a mutation that added AUTHORITY_SOURCE_MISSING to the hook --
    # the same tautology shape as PR 43's `wrong == dict(wrong)`.
    src = HOOK.read_text()
    (sess / f"{sid}.json").write_text(json.dumps(
        {"authority_id": "36", "disposition": "AUTHORITY_SOURCE_MISSING"}))
    rc, out, _ = run("write", {"session_id": sid, "cwd": str(ROOT),
                               "tool_input": {"file_path": "README.md"}}, "--exit2-on-deny")
    said = json.dumps(out)
    fails += check("AUTHORITY_SOURCE_MISSING does not refuse a write",
                   rc == 0 and "permissionDecision" not in said
                   and "does not carry execution authority" not in said,
                   "an onboarding disposition treated as a governance refusal "
                   "would stop all editing in every un-onboarded repository")
    (sess / f"{sid}.json").unlink()

    # --- per-host templates -------------------------------------------------
    for host, ev in (("claude", "UserPromptSubmit"), ("cursor", "beforeSubmitPrompt"),
                     ("codex", None)):
        t = TEMPLATES / f"{host}.json"
        fails += check(f"{host} config template exists", t.exists())
        if t.exists():
            try:
                cfg = json.loads(t.read_text())
                fails += check(f"{host} template is valid JSON", True)
                fails += check(f"{host} template invokes the shared script",
                               "governance-hook.py" in json.dumps(cfg))
                if ev:
                    fails += check(f"{host} template registers its prompt-time event ({ev})",
                                   ev in json.dumps(cfg))
            except json.JSONDecodeError as e:
                fails += check(f"{host} template is valid JSON", False, str(e))

    # --- all three deny spellings, and edit-only filtering -------------------
    # Tested by EMITTING a denial and reading the keys, not by grepping source.
    # The grep version passed after the Gemini spelling was deleted, because a
    # COMMENT elsewhere contained the literal `"decision"`. Prose is not
    # behaviour, and this is the third check in this file to learn it.
    with tempfile.TemporaryDirectory() as td:
        blk = pathlib.Path(td) / "blocking"
        blk.mkdir()
        subprocess.run(["git", "init", "-q", str(blk)], capture_output=True)
        mf_ = json.loads((ROOT / ".repo-governor.json").read_text())
        mf_["repo_governor"]["enforcement"] = "blocking"
        (blk / ".repo-governor.json").write_text(json.dumps(mf_), encoding="utf-8")
        sd = blk / ".repo-governor" / "sessions"
        sd.mkdir(parents=True)
        (sd / "d.json").write_text(json.dumps(
            {"authority_id": "1", "disposition": "NO_EXECUTION_AUTHORITY"}))
        rc, den, _ = run("write", {"session_id": "d", "cwd": str(blk), "tool_name": "Edit",
                                   "tool_input": {"file_path": "x.py"}}, cwd=str(blk))
        for key, host in (("hookSpecificOutput", "Claude/VS Code"),
                          ("permission", "Cursor"),
                          ("decision", "Gemini")):
            fails += check(f"a denial carries the {host} field ({key})", key in den,
                           f"one script serves every host only if it speaks every "
                           f"dialect; got {sorted(den)}")
    rc, out, _ = run("write", {"session_id": "t", "cwd": str(ROOT), "tool_name": "Bash"})
    fails += check("a non-edit tool is ignored", rc == 0 and out == {},
                   "VS Code parses matchers without applying them, so this fires on "
                   "every tool there -- including reads")
    rc, out, _ = run("write", {"session_id": "t", "cwd": str(ROOT),
                               "toolName": "insert_edit_into_file"})
    fails += check("a camelCase VS Code edit tool is recognised", bool(out))

    # --- the docs must not outlive the evidence -----------------------------
    # installation.md said "install it when a missed activation matters" until
    # the control refuted exactly that. A section that recommends a surface
    # must keep the counter-evidence one click away, or the next reader gets
    # the claim without the refutation.
    inst = (ROOT / "docs" / "installation.md").read_text()
    hook_sec = inst[inst.index("## Hooks"):] if "## Hooks" in inst else ""
    fails += check("installation.md has a hook section", bool(hook_sec))
    fails += check("the hook section links to its validation results",
                   "hook-validation-results.md" in hook_sec,
                   "the section that recommends the surface must cite what testing it showed")
    fails += check("the hook section calls the surface optional",
                   "optional" in hook_sec.lower())
    # Every host with a shipped template must appear in the matrix, and every
    # unverified one must link the issue that says so. A template that ships
    # without a row is a host we quietly claim to support.
    for host, iss in (("Claude Code", None), ("Cursor", "50"), ("Codex", "47"),
                      ("Gemini", "48"), ("VS Code", "49")):
        fails += check(f"install docs list {host}", host in hook_sec)
        if iss:
            fails += check(f"{host} row links its unvalidated-host issue",
                           f"/issues/{iss}" in hook_sec,
                           "an unverified host without a way to report is a dead end")
    fails += check("docs define what 'verified' means",
                   "Verified\" means one specific thing" in hook_sec
                   or "means one specific thing" in hook_sec,
                   "vendor docs are not verification -- Claude Code's documented "
                   "output shape was wrong")

    # --- the installer may advise and offer; it may never write silently -----
    # Tested by RUNNING it, not by grepping. The first version grepped for
    # "settings.json" and fired on the comment explaining why we do not write
    # it. The rule is no SILENT write -- an explicit yes is consent, not
    # imposition, which is the same line SKILL.md draws for --write.
    inst_sh = (ROOT / "tools" / "install-skill.sh").read_text()
    fails += check("installer warns when the target has no AGENTS.md",
                   "no AGENTS.md" in inst_sh)

    installer = ROOT / "tools" / "install-skill.sh"
    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "governed"
        tgt.mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        (tgt / ".repo-governor.json").write_text(
            (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
        r = subprocess.run(["bash", str(installer), str(tgt), ".claude/skills"],
                           capture_output=True, text=True, stdin=subprocess.DEVNULL)
        wrote = (tgt / ".claude" / "settings.json").exists()
        fails += check("non-interactive install writes no settings.json",
                       r.returncode == 0 and not wrote,
                       "a silent host reconfiguration is the prune's own error")
        fails += check("non-interactive install says how to opt in",
                       "non-interactive" in r.stdout)

    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "governed"
        tgt.mkdir(); (tgt / ".claude").mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        (tgt / ".repo-governor.json").write_text(
            (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
        (tgt / ".claude" / "settings.json").write_text(
            json.dumps({"permissions": {"allow": ["Bash(ls:*)"]}}), encoding="utf-8")
        subprocess.run(["bash", str(installer), str(tgt), ".claude/skills", "yes"],
                       capture_output=True, text=True, stdin=subprocess.DEVNULL)
        got = json.loads((tgt / ".claude" / "settings.json").read_text())
        fails += check("explicit yes installs the hook",
                       "UserPromptSubmit" in got.get("hooks", {}))
        fails += check("explicit yes preserves the user's own settings",
                       "permissions" in got,
                       "merging must never replace a file the user owns")

    # Every host the templates cover must be installable by the script, and the
    # script must use the SHIPPED template rather than a second copy inline --
    # a config the installer writes and one the docs describe must not drift.
    # ev is the PROMPT-time event, or None where the host documents none.
    # Codex is None on purpose: it was shipped with Cursor's event names taken
    # from a search summary, and its own docs describe no prompt-submit event.
    # A template check that passes on the wrong event names is what let that
    # ship, so the expected names are pinned here rather than merely counted.
    # Pin the event names against the SOURCE templates. The per-host install
    # loop below reads the config from a `git clone`, so an uncommitted edit is
    # invisible to it -- it can only catch a wrong template AFTER it ships,
    # which is not a guard. Reverting codex.json to Cursor's event names in the
    # working tree left that loop entirely green.
    EXPECTED = {
        "claude": {"UserPromptSubmit", "PreToolUse", "PostToolUse"},
        "cursor": {"beforeSubmitPrompt", "preToolUse", "afterShellExecution"},
        "codex":  {"PreToolUse", "PostToolUse"},
        "gemini": {"BeforeAgent", "BeforeTool", "AfterTool"},
        "vscode": {"UserPromptSubmit", "PreToolUse", "PostToolUse"},
    }
    for host, want in EXPECTED.items():
        t = TEMPLATES / f"{host}.json"
        got = set(json.loads(t.read_text()).get("hooks", {})) if t.exists() else set()
        fails += check(f"{host} template declares exactly {sorted(want)}", got == want,
                       f"got {sorted(got)} -- the wrong Codex events shipped because "
                       f"nothing pinned them")

    for host, rel, ev in (("claude", ".claude/settings.json", "UserPromptSubmit"),
                          ("cursor", ".cursor/hooks.json", "beforeSubmitPrompt"),
                          ("codex",  ".codex/hooks.json",  None),
                          ("gemini", ".gemini/settings.json", "BeforeAgent"),
                          ("vscode", ".github/hooks/repo-governor.json", "UserPromptSubmit")):
        with tempfile.TemporaryDirectory() as td:
            tgt = pathlib.Path(td) / "g"
            tgt.mkdir()
            subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
            (tgt / ".repo-governor.json").write_text(
                (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
            r = subprocess.run(["bash", str(ROOT / "tools" / "install-skill.sh"),
                                str(tgt), f".{host}/skills", "yes"],
                               capture_output=True, text=True, stdin=subprocess.DEVNULL)
            cfg = tgt / rel
            fails += check(f"{host}: installer writes {rel}", cfg.exists(), r.stdout[-160:])
            if cfg.exists():
                got = json.loads(cfg.read_text())
                hooks_ = got.get("hooks", {})
                if ev:
                    fails += check(f"{host}: config registers {ev}", ev in hooks_)
                else:
                    fails += check(f"{host}: registers NO prompt-submit event",
                                   not any(k.lower().startswith(("userprompt", "beforesubmit",
                                                                 "beforeagent")) for k in hooks_),
                                   "this host documents none; inventing one is how the "
                                   "wrong template shipped")
                fails += check(f"{host}: has a pre-write event", any(
                    k.lower() in ("pretooluse", "pretooluse", "beforetool") for k in hooks_))
                fails += check(f"{host}: paths are substituted, no placeholder left",
                               "RG_SKILL_DIR" not in json.dumps(got))
            if host != "claude":
                fails += check(f"{host}: install warns the template is unverified",
                               "UNVERIFIED" in r.stdout,
                               "only the Claude payload schema has been confirmed on a host")
                fails += check(f"{host}: install prints the delivery check",
                               "RG_HOOK_VERBOSE" in r.stdout and "delivery token" in r.stdout,
                               "a hook that runs and delivers nothing looks exactly like "
                               "a model ignoring governance")

    # The host is DECLARED, never inferred. An earlier version sniffed the
    # target for a .cursor/ or .claude/ directory -- ADR-028's defect, an
    # identity guessed from incidental evidence. The consequence is worse than
    # usual here: a repo someone opened in Cursor once, whose owner runs Codex,
    # gets .cursor/hooks.json that Codex never reads, which is indistinguishable
    # from a hook that does not work.
    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "trap"
        (tgt / ".cursor").mkdir(parents=True)
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        (tgt / ".repo-governor.json").write_text(
            (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
        r = subprocess.run(["bash", str(ROOT / "tools" / "install-skill.sh"),
                            str(tgt), ".agents/skills", "yes"],
                           capture_output=True, text=True, stdin=subprocess.DEVNULL)
        fails += check("a stray .cursor/ does NOT imply the cursor host",
                       not (tgt / ".cursor" / "hooks.json").exists(),
                       "writing a config the host never reads looks exactly like "
                       "a hook that does not work")
        fails += check("undeclared host says how to declare it",
                       "harness" in r.stdout and "names no host" in r.stdout)

    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "declared"
        (tgt / ".cursor").mkdir(parents=True)
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        (tgt / ".repo-governor.json").write_text(
            (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
        subprocess.run(["bash", str(ROOT / "tools" / "install-skill.sh"),
                        str(tgt), ".agents/skills", "yes", "cursor"],
                       capture_output=True, text=True, stdin=subprocess.DEVNULL)
        fails += check("an explicitly declared host installs",
                       (tgt / ".cursor" / "hooks.json").exists())

    # What an install CARRIES, not only what it removes. The prune list is one
    # place a licence obligation could be dropped by accident, and until now
    # nothing looked. Apache-2.0 section 4(a) requires recipients get a copy of
    # the License and 4(d) requires the NOTICE travel with it, so their
    # SURVIVAL is as much a contract as CONTRIBUTING.md's removal.
    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "carry"
        tgt.mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        (tgt / ".repo-governor.json").write_text(
            (ROOT / ".repo-governor.json").read_text(), encoding="utf-8")
        subprocess.run(["bash", str(ROOT / "tools" / "install-skill.sh"),
                        str(tgt), ".agents/skills", "no"],
                       capture_output=True, text=True, stdin=subprocess.DEVNULL)
        dest = tgt / ".agents" / "skills" / "repo-governor"
        for keep in ("LICENSE", "NOTICE"):
            fails += check(f"the install carries {keep} (Apache-2.0 4a/4d)",
                           (dest / keep).exists(),
                           "a distribution without it is a licence violation, and the "
                           "prune list is where that would happen silently")
        fails += check("the install does NOT carry Repo Governor's CONTRIBUTING.md",
                       not (dest / "CONTRIBUTING.md").exists(),
                       "it states this project's branch policy and conformance counts, "
                       "which are false inside the repository it was installed into")
        fails += check("SKILL.md survives the prune", (dest / "SKILL.md").exists())

    # Onboarding advice must reach a target with NO recognisable host directory
    # and NO terminal. It was nested inside the host block and gated on a TTY,
    # so a plain repo installed non-interactively got no guidance at all -- and
    # the hook offer it was blocking on can never fire until onboarding happens.
    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "plain"
        tgt.mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        r = subprocess.run(["bash", str(ROOT / "tools" / "install-skill.sh"),
                            str(tgt), ".agents/skills"],
                           capture_output=True, text=True, stdin=subprocess.DEVNULL)
        fails += check("ungoverned target is told it is not onboarded",
                       "not onboarded" in r.stdout,
                       "no host dir and no TTY must not mean no guidance")
        fails += check("advice names onboard-interactive.py, not engine/onboard.py",
                       "onboard-interactive.py" in r.stdout,
                       "engine/onboard.py emits evidence; renaming it does not bind")
        fails += check("advice gives the bind step",
                       ".repo-governor.proposed.json" in r.stdout
                       and "--validate" in r.stdout)
        fails += check("advice warns the proposal is a governance signal",
                       "measurement" in r.stdout)

    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "noproposal"
        tgt.mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        r = subprocess.run(["bash", str(installer), str(tgt), ".claude/skills", "yes"],
                           capture_output=True, text=True, stdin=subprocess.DEVNULL)
        fails += check("installer never writes a manifest proposal unasked",
                       not (tgt / ".repo-governor.proposed.json").exists()
                       and not (tgt / ".repo-governor.json").exists(),
                       "binding is a human act; even proposing is a governance signal "
                       "in the repository root")
        fails += check("installer warns a proposal is itself a governance signal",
                       "governance" in r.stdout and "measurement" in r.stdout)

    with tempfile.TemporaryDirectory() as td:
        tgt = pathlib.Path(td) / "ungoverned"
        tgt.mkdir()
        subprocess.run(["git", "init", "-q", str(tgt)], capture_output=True)
        r = subprocess.run(["bash", str(installer), str(tgt), ".claude/skills", "yes"],
                           capture_output=True, text=True, stdin=subprocess.DEVNULL)
        fails += check("even 'yes' does not install into an un-onboarded repository",
                       not (tgt / ".claude" / "settings.json").exists()
                       and "SILENT" in r.stdout,
                       "the hook cannot speak without a manifest; installing it there "
                       "only ends an activation measurement")

    # --- stdlib only (ADR-011) ----------------------------------------------
    # Tested against sys.stdlib_module_names, not a hand-maintained allowlist.
    # The first version was an allowlist and failed on `hashlib` -- which is
    # stdlib. A list that must be edited whenever correct code changes is a
    # check that reports on its own maintenance, not on the property.
    mods = set()
    for l in src.splitlines():
        m = re.match(r"(?:from|import)\s+([A-Za-z_][\w.]*)", l.strip())
        if m:
            mods.add(m.group(1).split(".")[0])
    bad = sorted(mods - set(sys.stdlib_module_names) - {"engine"})
    fails += check("hook imports stdlib only (ADR-011)", not bad, str(bad))

    print(f"\n{'HOOKS: CONFORMANT' if not fails else f'HOOKS: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
