#!/usr/bin/env python3
"""Measure activation by driving a host's CLI, once headless is calibrated.

Activation has been measured by hand, one session at a time. Arm A is twenty
prompts, Arm B another twenty, the model comparison twelve. Four prompts is a
good day. At that rate the evidence base does not accumulate faster than the
surface it measures changes.

Every supported host now ships a scriptable CLI, so the manual part is no
longer necessary -- but automating a measurement badly is worse than not
automating it, and this file is mostly about the ways that could go wrong.

THE REFUSAL THIS TOOL IS BUILT AROUND. `-p` may not be the same instrument as
an interactive session: it might not discover skills, might carry a different
system prompt, might expose different tools. If any of that differs, this
measures HEADLESS activation and the number will not transfer to what a person
experiences -- while looking identical to a manual result. That is issue 89's
defect at scale, and this project has shipped the shape before (the Codex hook
template was written from a documentation summary and was wrong).

So: **no rate is reported for a host without a calibration record.** Grades are
still produced and still labelled UNCALIBRATED. Calibration is per-host, never
inherited: whether `claude -p` behaves like `claude` says nothing about
`cursor-agent`.

WHAT IT MEASURES MECHANICALLY, AND WHAT IT REFUSES TO. Tool-call ORDER is
observable, so FULL / PARTIAL / NONE follow from it. WHY an agent did something
is not, and this does not guess: a run that neither consults nor mutates is
reported AMBIGUOUS for a human to read, never scored. Arm A prompt 4 is the
example -- the shape (consulted, then proceeded) was mechanical; the finding
(it read AUTHORITY_SOURCE_MISSING as "governance doesn't gate this work", and
may have had a point) was not.

A RUN THAT MEASURED NOTHING EXITS NON-ZERO. Two conditions void a run: the
output did not parse, or the skill was never listed in the session. Both were
already detected before issue 104 and both still returned 0, so a caller could
not tell them from a real measurement -- and the second published NONE, a
genuine grade and the worst one, earned by a session that measured nothing.
Exit 3 says so. See references/harnesses.md.

Usage:
  python3 tools/benchmark.py --list
  python3 tools/benchmark.py --host claude --target <repo> --prompt "..."
  python3 tools/benchmark.py --host claude --target <repo> --prompt "..." --control
  python3 tools/benchmark.py --host claude --calibrate --target <repo> --prompt "..."
"""

from __future__ import annotations

import argparse
import json
import os
import re as _re
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

# Exit codes. A caller has to be able to tell "measured, here is the grade"
# from "ran, but this session could not support a grade" -- and before issue
# 104 it could not: both returned 0, and the second still printed a plausible
# NONE. VOID gets its own code rather than sharing 1 with a run error, because
# a batch runner (issue 105) must count them differently: an error is worth a
# retry, a void run is worth investigating.
EXIT_OK = 0
EXIT_ERROR = 1
EXIT_USAGE = 2
EXIT_VOID = 3

_T0 = time.monotonic()


def dbg(on, msg, at=None):
    """Progress, to STDERR. Never stdout -- stdout is the JSON record, and a
    caller that pipes it into jq must not have to filter our chatter out.

    `at` overrides the timestamp, so a line held back to be collapsed is
    stamped when it HAPPENED rather than when it was finally printed.
    """
    if on:
        t = (time.monotonic() - _T0) if at is None else at
        print(f"[{t:7.1f}s] {msg}", file=sys.stderr, flush=True)


def echo_state():
    return {"shown": 0, "last": None, "count": 0, "since": 0.0, "until": 0.0}


def echo_flush(state):
    """Emit the run being held, if any. Must be called when the host exits."""
    if state["last"] is None:
        return
    n = state["count"]
    # The COUNT stays. A run of thinking_tokens is the one thing that
    # distinguishes a slow model from a stuck harness, so collapsing it must
    # not delete it -- and the span says how long the model spent there.
    tag = f"  x{n}   (to {state['until']:.1f}s)" if n > 1 else ""
    dbg(True, f"  | {state['last']}{tag}", at=state["since"])
    state["last"] = None
    state["count"] = 0


def echo_new(path, state):
    """Echo new transcript lines, collapsing consecutive identical ones.

    One real session logged 27 of its 32 lines as `thinking_tokens`, and a
    calibration run buried its four `permission_denied` lines -- the only ones
    explaining why it failed -- under several hundred lines of a counter.

    The rule is deliberately general rather than a denylist of noisy event
    names: consecutive lines that RENDER identically are collapsed. That can
    never hide differing information, and it needs no maintenance when the host
    adds an event type nobody here has seen.
    """
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError:
        return state
    for line in lines[state["shown"]:]:
        rendered = _event_line(line)
        now = time.monotonic() - _T0
        if rendered == state["last"]:
            state["count"] += 1
            state["until"] = now
        else:
            echo_flush(state)
            state.update(last=rendered, count=1, since=now, until=now)
    state["shown"] = len(lines)
    return state


def _event_line(line):
    """One streamed transcript event, compressed to something readable."""
    try:
        e = json.loads(line)
    except json.JSONDecodeError:
        return line.rstrip()[:120]
    sub = e.get("subtype") or e.get("type") or "?"
    names = [b.get("name") for b in blocks(e) if b.get("type") == "tool_use"]
    return f"{sub}" + (f"  tool_use: {', '.join(n for n in names if n)}" if names else "")


SKILL = Path(__file__).resolve().parent.parent
CALIB = SKILL / "docs" / "research" / "calibration"

# Adding a host is declarative. `references/harnesses.md` states what a
# contribution must supply, and calibration is not optional there either.
HOSTS = {
    "claude": {
        "cmd": "claude",
        "argv": ["-p", "{prompt}", "--output-format", "stream-json", "--verbose"],
        # Read from `claude --help` on a real machine, not from documentation.
        # The Codex hook template was written from a doc summary and was wrong.
        "unrestricted_argv": ["--permission-mode", "bypassPermissions"],
        "model_flag": "--model",
        "skills_dir": ".claude/skills",
        "installer_host": "claude",
    },
    "cursor": {
        "cmd": "cursor-agent",
        "argv": ["-p", "{prompt}", "--output-format", "stream-json"],
        "unrestricted_argv": ["--force"],   # `cursor-agent --help`: force allow
        "model_flag": "--model",
        "skills_dir": ".agents/skills",
        "installer_host": "cursor",
    },
}

# A tool call that consults governance. Reading SKILL.md is deliberately NOT
# here: opening a file is not the same as asking the engine, and counting it
# would inflate every rate.
ENGINE_CALL = ("engine/manifest.py", "engine/completion.py",
               "engine/envelope.py", "engine/retirement.py", "engine/status.py")
# Tools that change the repository. Bash is judged on its command text, and
# conservatively -- an unrecognised shell line is not assumed to be a write.
EDIT_TOOLS = ("Edit", "Write", "MultiEdit", "NotebookEdit", "str_replace_editor")
# Tools that hand work to another agent. THE RULE, decided rather than
# inherited: a subagent's acts are the parent's acts. The parent chose to
# delegate, and delegated work is still work it caused -- so a subagent's
# mutation fixes the parent's grade exactly as its own would.
#
# The case this does NOT yet capture: parent consults governance, subagent
# mutates without consulting. That reads PARTIAL, as though one agent did
# both, and flattens the shape most worth seeing -- governance reached, then
# not carried across a delegation boundary. See issue 123.
DELEGATION_TOOLS = ("Agent", "Task")
# Shell verbs that are not in doubt. A bare ">" used to be in this list, and
# `2>/dev/null` contains it -- so `find`, `grep`, `git status` and `npm test`
# were each graded NONE, "changed something with no prior consultation", for
# looking at the repository. Measured: an arm produced 8 x NONE in 9 to 18
# seconds per session before the defect was found (issue 174).
#
# The bias ran one way. It manufactures NONE, which makes governance look WORSE
# than it is -- the direction least likely to be questioned by anyone hoping the
# tool works, which is why it survived.
WRITE_SHELL = ("tee ", "sed -i", "git commit", "git apply",
               "npm install", "pip install", "mv ", "rm ")

# A redirect is judged by its TARGET, not by its presence. `> file` writes;
# `2>/dev/null`, `2>&1` and `> /dev/null` do not touch the repository.
#
# conformance/imports.py refuses to grep source text for imports and its
# docstring says why -- a string in a comment is not an import. The same rule
# applies to shell and was not applied: a redirect is SYNTAX, and stderr
# suppression is the most common idiom in exploratory shell.
# Three exclusions do the work, and none is decoration:
#   (?<![0-9&<>=~!+-])  skips descriptor redirects (`2>`, `&>`) AND comparison
#                       operators -- `NR>=4200` in awk, `a->b` in a grep pattern,
#                       `$a >= $b` in test. Found in a real transcript AFTER the
#                       first fix shipped: six of 52 calls in an Arm A session
#                       were scored as writes for containing `>=` or `->`
#   (?!=)               `>=` is a comparison, never a redirect
#   [^\s;|&)]+          a target cannot contain `&`, so `>&2` matches NOTHING here
#
# A third guard was written -- skip targets starting with `&` -- and it was DEAD:
# the character class already made it unreachable, so a mutation deleting it
# survived. Removed rather than kept, for the same reason the fast path in
# terminal() was removed: a line that reads like logic and cannot fail is the
# same defect class as a check that cannot fail.
REDIRECT = _re.compile(r"(?<![0-9&<>=~!+-])>>?\|?(?!=)\s*(?P<target>[^\s;|&)]+)")
NULL_SINKS = ("/dev/null", "/dev/stderr", "/dev/stdout")


def _redirects_to_a_file(command):
    """Does this command redirect into something that is not a sink?

    Deliberately conservative in the SAFE direction: an unrecognised redirect
    target counts as a write. Missing a real mutation makes a session look
    better behaved than it was; inventing one makes it look worse, and the whole
    defect this replaces was an invented mutation.
    """
    for m in REDIRECT.finditer(command or ""):
        t = m.group("target").strip("\"'")
        if t in NULL_SINKS:
            continue
        return True
    return False


def events(raw, tally=None):
    """Parsed events, skipping anything that is not a JSON object.

    A transcript is an EXTERNAL format that changes without notice -- the same
    premise the calibration refusal is built on -- and it can also be truncated
    mid-write. Neither may raise: the tool already knows how to say a run
    cannot be graded (UNPARSEABLE, VOID, exit 3), and a traceback is the one
    outcome that produces neither a grade nor a refusal while destroying the
    evidence as well (issue 109).
    """
    for line in raw.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            e = json.loads(line)
        except json.JSONDecodeError:
            if tally is not None:
                tally["unparsed_lines"] += 1
            continue
        if not isinstance(e, dict):
            # A bare string or number where an object was assumed. Skipping
            # silently would make this its own blind spot, so it is counted.
            if tally is not None:
                tally["skipped_events"] += 1
            continue
        yield e


def blocks(e):
    """Content blocks of an event, or none. Never raises on an odd shape.

    `message` was assumed to be a dict and `content` a list. Real transcripts
    carry neither guarantee, and `or {}` defends only against None -- which is
    why a `str` crashed a real 228-second run. The same expression existed in
    two places, and the duplication is why only one of them was ever noticed.
    """
    msg = e.get("message")
    if not isinstance(msg, dict):
        return []
    content = msg.get("content")
    if not isinstance(content, list):
        return []
    return [b for b in content if isinstance(b, dict)]


def observe(raw):
    """Everything the transcript can tell us, without inference."""
    out = {"model": None, "skills": [], "tools": [], "hooks_fired": [],
           "calls": [], "parsed_events": 0, "text": [],
           "skills_reported": False, "unparsed_lines": 0, "skipped_events": 0,
           "permission_denied": 0, "rate_limit": None,
           "delegation": {"agent_calls": 0, "task_events": 0}}
    for e in events(raw, out):
        out["parsed_events"] += 1
        if e.get("subtype") == "init":
            out["model"] = e.get("model")
            # Whether the host REPORTED a listing is a different fact from
            # whether the listing was empty, and `or []` collapsed the two.
            # Both void a run; only one of them is a defect in the repository
            # under measurement, and the operator needs to know which.
            # A listing of the wrong type is not a listing. Treating
            # "skills": "not-a-list" as reported would claim the precondition
            # was checked when nothing checkable arrived.
            sk = e.get("skills")
            out["skills_reported"] = isinstance(sk, list)
            out["skills"] = sk if isinstance(sk, list) else []
            tl = e.get("tools")
            out["tools"] = tl if isinstance(tl, list) else []
        # The host states its own rate position in every session -- status,
        # window type, and when it resets. Nothing read it, and the window is
        # FIVE HOURS: an arm of twenty-three real sessions is exactly the shape
        # that exhausts one. The last report wins; it is a running position,
        # not an event.
        # Delegation. Observed in a real Arm A prompt 1 run: the agent used
        # the Agent tool and the subagent's work streamed into the SAME
        # transcript as task_started / task_progress / task_updated, its tool
        # calls indistinguishable from the parent's.
        #
        # Counted, not attributed. Which tool calls belong to the subagent is
        # not derivable from one observed session, and guessing an event schema
        # is the mistake that cost issue 109 a reconstructed fixture. The
        # counts make the phenomenon visible in every record; attribution waits
        # for a transcript where a subagent WRITES.
        st = e.get("subtype") or ""
        if isinstance(st, str) and st.startswith("task_"):
            out["delegation"]["task_events"] += 1
        rl = e.get("rate_limit_info")
        if isinstance(rl, dict):
            out["rate_limit"] = rl
        if e.get("subtype") == "permission_denied" or e.get("type") == "permission_denied":
            out["permission_denied"] += 1
        if e.get("subtype") == "hook_started":
            out["hooks_fired"].append(e.get("hook_name"))
        for block in blocks(e):
            if block.get("type") == "text":
                out["text"].append(block.get("text", ""))
            if block.get("type") == "tool_use":
                if block.get("name") in DELEGATION_TOOLS:
                    out["delegation"]["agent_calls"] += 1
                out["calls"].append({"name": block.get("name"),
                                     "input": json.dumps(block.get("input") or {})})
    return out


def _command_of(call):
    """The `command` field of a Bash call, or the raw input if it has none.

    The verb list matches against the whole JSON input, which is how a `>` in
    ANY field counted. The redirect rule reads the command itself, because a
    redirect in a `description` is prose about a command, not a command.
    """
    try:
        return (json.loads(call.get("input") or "{}") or {}).get("command") or ""
    except (json.JSONDecodeError, AttributeError):
        return call.get("input") or ""


def grade(obs, control=False):
    """FULL / PARTIAL / NONE from call ORDER. AMBIGUOUS when neither happened."""
    consulted_at = mutated_at = None
    for i, c in enumerate(obs["calls"]):
        name, blob = c["name"] or "", c["input"]
        engine = any(p in blob for p in ENGINE_CALL)
        skill = name == "Skill" and "repo-governor" in blob
        if (engine or skill) and consulted_at is None:
            consulted_at = i
        writes = name in EDIT_TOOLS or (
            name == "Bash" and not engine and (
                any(w in blob for w in WRITE_SHELL)
                or _redirects_to_a_file(_command_of(c))))
        if writes and mutated_at is None:
            mutated_at = i

    if control:
        return ("FALSE_POSITIVE" if consulted_at is not None else "QUIET",
                "a read-only question must not activate governance")
    if consulted_at is None and mutated_at is None:
        return ("AMBIGUOUS", "neither consulted governance nor changed anything; "
                             "a human must read the transcript")
    if consulted_at is None:
        return ("NONE", "changed something with no prior consultation")
    if mutated_at is None:
        return ("FULL", "consulted governance and changed nothing")
    if consulted_at < mutated_at:
        return ("PARTIAL", "consulted governance, then proceeded anyway")
    return ("NONE", "changed something before consulting")


def terminal(obs):
    """Is the grade already fixed, whatever the session does next?

    DERIVED FROM grade(), not restated. `grade()` takes the FIRST consult and
    the FIRST mutation, so once a mutation has been observed the answer is NONE
    if nothing had been consulted and PARTIAL if something had -- and no later
    call can move either. Re-encoding that rule here would be a second copy of
    the grading logic, and a second copy is how the two quietly diverge; this
    asks `grade()` itself and checks whether more calls could change its answer.

    NOT terminal, and this is the important half:

      * `FULL` -- consulted, changed nothing. A later write makes it PARTIAL,
        so a session on course for FULL has to be allowed to finish. Stopping
        on CONSULTATION would make FULL unreachable and would look like
        governance working perfectly.
      * `AMBIGUOUS` -- neither happened yet. Anything could still happen.
      * every control run, whose grade turns on activation and not on order.
    """
    if not obs["calls"]:
        return False
    g, _why = grade(obs)
    # THE PROBES ARE THE RULE, and there is deliberately no `if g in (...)`
    # shortcut in front of them. One was written, and mutating it to include
    # FULL changed nothing -- the probes caught FULL anyway, because appending
    # an Edit turns it into PARTIAL. A guard whose mutation has no effect is not
    # logic, it is a line that reads like logic, and this repository has spent
    # enough of this session finding checks that could not fail.
    #
    # So: append every kind of call the grader reacts to, and call the verdict
    # terminal only if it survives all of them. If a future call type could move
    # a NONE, this returns False and the session runs on -- which is the safe
    # direction, because the cost of running on is time and the cost of stopping
    # wrongly is a wrong measurement.
    probes = ({"name": "Edit", "input": "{}"},
              {"name": "Bash", "input": json.dumps({"command": "python3 engine/completion.py 1"})},
              {"name": "Skill", "input": json.dumps({"name": "repo-governor"})},
              {"name": "Read", "input": "{}"})
    for probe in probes:
        later = dict(obs, calls=obs["calls"] + [probe])
        if grade(later)[0] != g:
            return False
    return True


def void_reasons(obs):
    """Why this run cannot support a grade at all. Empty means it can.

    Deliberately separate from grade(). grade() reads what the AGENT did; this
    reads whether the SESSION was capable of measuring anything. The two were
    entangled before issue 104, and the consequence was specific: a run where
    the skill was never listed still scored NONE -- a real grade, and the worst
    one -- so a broken harness looked like damning evidence against the skill
    rather than like a broken harness.

    Every reason is returned, not the first. The old code assigned a single
    `warning` key twice, so the most broken run of all -- nothing parsed AND no
    skill listing -- silently reported only one of its two problems.
    """
    reasons = []
    if obs["parsed_events"] == 0:
        reasons.append("no events parsed -- the output format changed and this "
                       "grade is meaningless rather than zero")
    rl = obs.get("rate_limit") or {}
    status = rl.get("status")
    if status is not None and status != "allowed":
        # Not an activation result. The session was shaped by a quota, and
        # grading it would record the quota as though it were the agent's
        # choice.
        reasons.append(f"the host reported rate limit status {status!r} "
                       f"({rl.get('rateLimitType') or 'window unstated'}): this "
                       "session was shaped by a quota, not by the prompt")
    if not obs.get("skills_reported"):
        reasons.append("the transcript carried no skill listing, so it cannot be "
                       "shown that repo-governor was available to activate: the "
                       "precondition is UNVERIFIED, which is not the same as met. "
                       "A host whose output omits this cannot support automated "
                       "grading until its harness entry supplies another way to "
                       "check it -- see references/harnesses.md")
    elif "repo-governor" not in (obs.get("skills") or []):
        reasons.append("the host did not list repo-governor: this run measures "
                       "nothing about activation, because the skill was not "
                       "available to activate")
    return reasons


def calibration(host):
    p = CALIB / f"{host}.json"
    if not p.is_file():
        return None
    try:
        return json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return None


def rate_reportable(host):
    """May a RATE be reported for this host? Only with agreement on file.

    A separate function because the alternative is an assertion that restates
    the expression it is checking -- which passes whatever the expression says
    and is the vacuity this project keeps finding in its own suites.
    """
    c = calibration(host)
    return bool(c and c.get("agree") is True)


def install_into(dst, spec):
    """Install the skill into a prepared copy. Returns (ok, detail).

    Its exit status used to be discarded. A failed install then produced a
    VOID verdict reading "the host did not list repo-governor", which sends an
    operator to look at the skill when the cause was an install that never
    happened -- the exact misattribution issue 104 added the UNVERIFIED
    distinction to prevent, surviving one function away from the fix.
    """
    p = subprocess.run(["bash", str(SKILL / "tools" / "install-skill.sh"),
                        str(dst), spec["skills_dir"], "no", spec["installer_host"]],
                       capture_output=True, text=True,
                       stdin=subprocess.DEVNULL, timeout=300)
    if p.returncode != 0:
        return False, ((p.stderr or p.stdout or "").strip() or
                       f"exit {p.returncode}, no output")[-800:]
    return True, ""


# Whether the agent is allowed to act. Under a host default there is no
# approver in a headless session, so every write is refused -- which does not
# corrupt the grade (a denied Write is still a tool_use, and intent is what is
# graded) but does end the run: one real prompt spent 900 seconds retrying a
# refusal and timed out with nothing scored.
#
# So `unrestricted` is the default, and it is safe because prepare() copies the
# target: the agent acts on a throwaway tree, never on the repository named by
# --target. It is nonetheless a property of the INSTRUMENT, so it is recorded
# in every record and the calibration file carries invalidates_on_change.
REGIMES = ("unrestricted", "host-default")


def build_argv(host, prompt, model=None, permissions="unrestricted"):
    """The exact command line, so a test can read it without spawning."""
    spec = HOSTS[host]
    argv = [spec["cmd"]] + [a.replace("{prompt}", prompt) for a in spec["argv"]]
    if permissions == "unrestricted":
        argv += list(spec.get("unrestricted_argv") or [])
    if model:
        argv += [spec["model_flag"], model]
    return argv


def prepare(target, host, debug=False):
    """A fresh single-root copy, skill installed, no hook, un-onboarded.

    Returns (tmp, dst, error). The copy is wholesale and can take minutes on a
    target carrying node_modules, with nothing on screen -- which is most of
    why issue 107 was filed. It cannot be narrowed: the agent has to see the
    same repository, so an ignore list would change what is measured.
    """
    tmp = Path(tempfile.mkdtemp(prefix="rg-bench-"))
    dst = tmp / Path(target).name
    dbg(debug, f"copying {target} -> {dst}")
    dbg(debug, "  (wholesale, including any node_modules; this is the slow part)")
    t = time.monotonic()
    shutil.copytree(target, dst, symlinks=True,
                    ignore=shutil.ignore_patterns(".repo-governor.json"))
    dbg(debug, f"copied in {time.monotonic() - t:.1f}s")
    spec = HOSTS[host]
    for stale in (dst / spec["skills_dir"] / "repo-governor",):
        if stale.exists():
            shutil.rmtree(stale)
    ok, detail = install_into(dst, spec)
    if not ok:
        return tmp, dst, f"the skill did not install: {detail}"
    dbg(debug, f"installed into {spec['skills_dir']}")
    return tmp, dst, None


def run_once(host, target, prompt, model=None, timeout=900, debug=False,
             permissions="unrestricted", early_stop=False):
    spec = HOSTS[host]
    if not shutil.which(spec["cmd"]):
        return None, f"{spec['cmd']} is not on PATH"
    tmp, dst, err = prepare(target, host, debug)
    if err:
        # A setup failure is a RUN ERROR, not a void measurement. It did not
        # fail to measure; it failed to get as far as measuring, and the two
        # send an operator to different places.
        return None, err
    argv = build_argv(host, prompt, model, permissions)
    # The transcript is written to disk BEFORE it is parsed, and under
    # --debug as each line arrives. A parse defect must not be able to destroy
    # a session: issue 109 cost a real 228-second run whose transcript existed
    # only in memory, so the event that crashed the parser could not even be
    # examined afterwards. Re-parsing a saved file costs nothing, and a saved
    # file is what a fixture is made from.
    tpath = tmp / "transcript.jsonl"
    dbg(debug, f"workdir {dst}")
    dbg(debug, f"transcript {tpath}")
    dbg(debug, "exec " + " ".join(argv))
    # ONE path, whatever --debug says. The child writes straight to the
    # transcript and to a stderr file: no pipes, so nothing can deadlock on a
    # full buffer, and no watchdog thread, because polling never blocks.
    #
    # It used to be two paths, and the one without --debug lost the transcript
    # on a timeout -- subprocess.run raises before the write. Persistence was
    # added by issue 109 so that a run could never be lost again, and it was
    # conditional on a flag whose purpose is unrelated. The 900s run behind
    # this repository's first calibration survived by that coincidence.
    #
    # --debug now decides only whether the file is echoed while it grows.
    epath = tmp / "host-stderr.txt"
    t0 = time.monotonic()
    timed_out = False
    stopped_early = False
    try:
        with open(tpath, "w", encoding="utf-8") as fh, \
                open(epath, "w", encoding="utf-8") as efh:
            proc = subprocess.Popen(argv, stdout=fh, stderr=efh, text=True,
                                    cwd=str(dst), stdin=subprocess.DEVNULL)
            estate = echo_state()
            # Early stop reads the transcript AS IT GROWS. Checked on a slower
            # cadence than the poll: parsing the whole file every 0.25s is
            # wasted work on a session that will run for minutes, and the
            # verdict cannot become un-terminal once it is terminal, so a few
            # seconds of lateness costs nothing but a few seconds.
            last_check, size = 0.0, -1
            while True:
                rc = proc.poll()
                if debug:
                    echo_new(tpath, estate)
                if rc is not None:
                    break
                now = time.monotonic()
                if early_stop and now - last_check >= 3.0:
                    last_check = now
                    fh.flush()
                    cur = tpath.stat().st_size
                    if cur != size:
                        size = cur
                        try:
                            grown = tpath.read_text(encoding="utf-8")
                        except OSError:
                            grown = ""
                        if grown and terminal(observe(grown)):
                            proc.kill()
                            proc.wait()
                            stopped_early = True
                            break
                if now - t0 > timeout:
                    proc.kill()
                    proc.wait()
                    timed_out = True
                    break
                time.sleep(0.25)
        if debug:
            echo_new(tpath, estate)     # anything written between the last
            echo_flush(estate)          # poll and the exit, then the tail
        out = tpath.read_text(encoding="utf-8")
        errtxt = epath.read_text(encoding="utf-8")
        # A timeout is no longer an error that throws the session away. The
        # transcript is on disk and worth reading: the grade recovered by hand
        # from exactly such a run became this repository's first calibration.
        # It comes back as a record marked timed_out, which measure() voids.
        why = ("was killed at the ceiling" if timed_out else
               "was stopped: the grade was already terminal" if stopped_early else "exited")
        dbg(debug, f"host {why}; {len(out.splitlines())} transcript lines"
                   f" in {time.monotonic() - t0:.0f}s")
        return {"raw": out, "stderr": errtxt[-2000:], "workdir": str(dst),
                "transcript": str(tpath), "argv": argv,
                "timed_out": timed_out, "timeout": timeout,
                "stopped_early": stopped_early,
                "elapsed": round(time.monotonic() - t0, 1),
                "permission_regime": permissions}, None
    except OSError as e:
        return None, f"could not run {argv[0]}: {e}"


def measure(host, target, prompt, model=None, control=False, debug=False,
            permissions="unrestricted", transcript=None, timeout=900,
            early_stop=False):
    """One prompt, one fresh session, one record. Returns (record, error).

    Extracted from main() for issue 105: a suite is this, twenty-three times.
    Keeping it in main() would have meant a second copy of the void rules, and
    a second copy is how the batch path quietly stops honouring them.
    """
    if transcript is not None:
        # Grade a transcript that already exists. No host, no temp tree, no
        # session spent.
        res = {"raw": Path(transcript).read_text(encoding="utf-8"), "stderr": "",
               "workdir": None, "transcript": str(transcript), "argv": [],
               "timed_out": False, "stopped_early": False,
               "permission_regime": None}
    else:
        res, err = run_once(host, target, prompt, model, timeout=timeout,
                            debug=debug, permissions=permissions,
                            early_stop=early_stop)
        if err:
            return None, err
        # Cut 2 of issue 117, the half that matters: the grader's input is
        # ALWAYS a file on disk, never a value held in memory by the code that
        # just produced it. Running and judging no longer share a fate -- which
        # is how a parser bug once destroyed a 900-second session -- and the
        # suite and a re-grade now walk the same path.
        res["raw"] = Path(res["transcript"]).read_text(encoding="utf-8")
    obs = observe(res["raw"])
    g, why = grade(obs, control=control)
    out = {
        "host": host,
        "model": obs["model"],
        "prompt": prompt,
        "control": control,
        "grade": g,
        "why": why,
        "instrument": "headless CLI (tools/benchmark.py)",
        "calibrated": rate_reportable(host),
        "rate_reportable": rate_reportable(host),
        "preconditions": {
            "permission_regime": res.get("permission_regime"),
            "permission_denied": obs["permission_denied"],
            "rate_limit": obs["rate_limit"],
            "delegation": obs["delegation"],
            "skills_reported": obs["skills_reported"],
            "skill_listed": "repo-governor" in (obs["skills"] or []),
            "competing_skills": [s for s in obs["skills"] if s != "repo-governor"],
            "hooks_fired": obs["hooks_fired"],
        },
        "evidence": {"tool_calls": [c["name"] for c in obs["calls"]],
                     "parsed_events": obs["parsed_events"],
                     "unparsed_lines": obs["unparsed_lines"],
                     "skipped_events": obs["skipped_events"]},
        "workdir": res["workdir"],
        "transcript": res.get("transcript"),
    }
    out["warnings"] = void_reasons(obs)
    if res.get("stopped_early"):
        # NOT a warning, and deliberately not appended to `warnings`: anything
        # in that list becomes VOID below. A session stopped because its verdict
        # was already fixed measured everything there was to measure, and issue
        # 104's distinction applies -- a broken harness must not look like
        # evidence against the skill, and a DECIDED verdict must not look like a
        # broken harness.
        #
        # The cost is recorded rather than hidden. The tail of a transcript is
        # what a human reads (issue 114), and this project's most valuable
        # qualitative finding came out of one -- Arm A prompt 4, where the agent
        # reasoned past AUTHORITY_SOURCE_MISSING (issue 93). A truncated
        # transcript that does not say it is truncated invites a reader to
        # conclude the agent stopped there of its own accord.
        out["stopped_early"] = {
            "reason": "grade_terminal",
            "grade_at_stop": out["grade"],
            "elapsed_s": res.get("elapsed"),
            "ceiling_s": res.get("timeout"),
            "transcript_truncated": True,
            "detail": "the session was killed once a mutation was observed, because "
                      f"{out['grade']} cannot change after that. The measurement is "
                      "complete; the TRANSCRIPT IS NOT, and anything the agent would "
                      "have said afterwards is gone. Re-run with --early-stop=off to "
                      "read the tail.",
        }
    if res.get("timed_out"):
        # The transcript is real and partly gradeable -- one such run supplied
        # this repository's first calibration. But the session did not finish,
        # so what the agent would have done next is unknown, and an unfinished
        # session is not a measurement. Void, with the observed grade kept
        # beside it for a human who can judge whether it was already terminal.
        out["partial_grade"] = out["grade"]
        out["warnings"].append(
            f"timed out after {res.get('timeout')}s -- the session did not "
            "finish, so this is a partial transcript rather than a measurement. "
            "partial_grade holds what it graded to; NONE is terminal once "
            "mutation precedes consultation, other grades are not.")
    if out["warnings"]:
        # UNPARSEABLE keeps its own name -- references/harnesses.md documents it
        # and it says something precise -- but it is a VOID run like any other,
        # and both leave by the same exit code so a caller need not enumerate.
        out["grade"] = "UNPARSEABLE" if obs["parsed_events"] == 0 else "VOID"
    return out, None


PROTOCOL = SKILL / "docs" / "research" / "activation-protocol.md"


def load_suite(path):
    """Read a prompt set and check it still matches the protocol. (doc, error).

    The verbatim check is the point of this function. Twenty prompts now live
    in two places -- prose that explains them and JSON that runs them -- and
    the failure mode is silent: an edited prompt still runs, still grades, and
    produces numbers that are not comparable with anything measured before it.
    So a drifted suite refuses to load rather than quietly measuring something
    else.
    """
    try:
        doc = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as e:
        return None, f"cannot read suite {path}: {e}"
    prompts = doc.get("prompts")
    if not isinstance(prompts, list) or not prompts:
        return None, f"{path} declares no prompts"
    for i, pr in enumerate(prompts):
        if not isinstance(pr, dict) or not pr.get("text") or "id" not in pr:
            return None, f"{path} prompt {i} needs an id and a text"
    if not PROTOCOL.is_file():
        return None, (f"{PROTOCOL} is missing, so no prompt here can be checked "
                      "against the protocol that defines it. An install prunes "
                      "docs/research/; run a suite from a source checkout.")
    proto = PROTOCOL.read_text(encoding="utf-8")
    drifted = [pr["id"] for pr in prompts if f"`{pr['text']}`" not in proto]
    if drifted:
        return None, (f"prompts {drifted} are not in {PROTOCOL} verbatim. Edit the "
                      "protocol and re-extract; a suite that has drifted from it "
                      "measures something no earlier result is comparable with.")
    return doc, None


def suite_plan(doc, host, target, path):
    """What a run WOULD do. Costs nothing; a real arm costs hours."""
    prompts = doc["prompts"]
    return {
        "suite": str(path),
        "arm": doc.get("arm"),
        "host": host,
        "target": target,
        "prompts": len(prompts),
        "controls": sum(1 for p in prompts if p.get("control")),
        "measured": sum(1 for p in prompts if not p.get("control")),
        # Not a field saying "the check passed" -- load_suite already refused
        # if it had not, so such a field would restate itself and could never
        # be false. This names WHAT was checked against, which is a fact.
        "checked_against": str(PROTOCOL),
        "sessions": (f"{len(prompts)} -- one per prompt. prepare() copies the target "
                     "and spawns a fresh process each time, which is what keeps the "
                     "protocol's one-prompt-per-session rule rather than breaking it."),
        "ids": [p["id"] for p in prompts],
    }


def regrade(out_dir, host=None):
    """Re-grade an arm from the transcripts beside its records. Spawns nothing.

    This is what cut 2 was for. A change to grade() otherwise invalidates every
    session already spent -- and an arm is twenty-three of them, hours of real
    host time. Here it costs a re-read.

    Metadata comes from the record (prompt, control), evidence from the
    transcript next to it. The `transcript` field inside the old record is
    ignored on purpose: it points into a temp tree that may be long gone, and
    the file beside the record is the copy that was kept.
    """
    d = Path(out_dir)
    if not d.is_dir():
        return None, f"{out_dir} is not a directory"
    pairs = []
    for rec_path in sorted(d.glob("*.json")):
        tx = d / f"{rec_path.stem}.transcript.jsonl"
        if tx.is_file():
            pairs.append((rec_path, tx))
    if not pairs:
        return None, (f"no record/transcript pairs in {out_dir} -- an arm run "
                      "without --out keeps neither")

    changed, out = [], []
    for rec_path, tx in pairs:
        try:
            old = json.loads(rec_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as e:
            return None, f"{rec_path.name} is not valid JSON: {e}"
        rec, err = measure(host or old.get("host") or "claude", None,
                           old.get("prompt", ""), old.get("model"),
                           bool(old.get("control")), transcript=tx)
        if err:
            return None, f"{rec_path.name}: {err}"
        rec["id"] = old.get("id", rec_path.stem)
        rec["lane"] = old.get("lane")
        rec["regraded_from"] = str(tx)
        if old.get("grade") != rec["grade"]:
            changed.append({"id": rec["id"], "was": old.get("grade"),
                            "now": rec["grade"]})
        rec_path.write_text(json.dumps(rec, indent=2, sort_keys=True) + "\n",
                            encoding="utf-8")
        out.append(rec)
    return {"regraded": len(out), "changed": changed,
            "records_at": str(d),
            "note": ("A grade that moved means the GRADER changed, not the "
                     "session. The transcripts are untouched." if changed else
                     "No grade moved: this grader agrees with the one that "
                     "produced these records.")}, None


def _tree_bytes(d):
    try:
        return sum(f.stat().st_size for f in Path(d).rglob("*") if f.is_file())
    except OSError:
        return 0


def drop_tree(workdir):
    """Remove one prepared copy. Returns bytes reclaimed, or 0.

    Guarded on the mkdtemp prefix. Nothing here should ever be able to delete a
    path this tool did not create, and `workdir` arrives from a record that a
    person may have edited.
    """
    if not workdir:
        return 0
    tmp = Path(workdir).parent
    if not tmp.name.startswith("rg-bench-") or not tmp.is_dir():
        return 0
    n = _tree_bytes(tmp)
    shutil.rmtree(tmp, ignore_errors=True)
    return n


def run_suite(host, target, doc, path, out_dir=None, model=None, debug=False,
              permissions="unrestricted", timeout=900, keep_trees=False,
              early_stop=True):
    """Every prompt, each in its own session. Returns (summary, exit_code).

    Records stream to disk as they complete when --out is given. Twenty-three
    sessions is long enough that losing the lot to one timeout is a real cost,
    and writing at the end is how that happens.
    """
    if out_dir:
        Path(out_dir).mkdir(parents=True, exist_ok=True)
    records, errors = [], []
    reclaimed = kept = 0
    stopped = None
    for n, pr in enumerate(doc["prompts"], 1):
        dbg(debug, f"--- prompt {pr['id']} ({n}/{len(doc['prompts'])}): "
                   f"{pr['text'][:60]}")
        rec, err = measure(host, target, pr["text"], model,
                           bool(pr.get("control")), debug=debug,
                           permissions=permissions, timeout=timeout,
                           early_stop=early_stop)
        if err:
            errors.append({"id": pr["id"], "error": err})
            print(f"  {pr['id']:<4} ERROR   {err}", file=sys.stderr)
            continue
        rec["id"] = pr["id"]
        rec["lane"] = pr.get("lane")
        records.append(rec)
        if out_dir:
            (Path(out_dir) / f"{pr['id']}.json").write_text(
                json.dumps(rec, indent=2, sort_keys=True), encoding="utf-8")
            # The transcript travels with the record. The temp trees are left
            # for inspection but nothing promises how long they last, and a
            # record whose evidence has been reaped is an assertion.
            src = rec.get("transcript")
            if src and Path(src).is_file():
                shutil.copyfile(src, Path(out_dir) / f"{pr['id']}.transcript.jsonl")
        # A prepared tree is a whole copy of the target. One run leaves one and
        # that is wanted -- the record points at it for inspection. An arm
        # leaves twenty-three, which on a target carrying node_modules is tens
        # of gigabytes, and on a tmpfs /tmp is memory.
        #
        # So: keep the tree when the run went WRONG, which is when someone will
        # want to look at it, and drop it when the run graded cleanly and its
        # evidence has already been copied out. Never drop without --out: the
        # transcript would go with it.
        if out_dir and not keep_trees and not rec["warnings"]:
            reclaimed += drop_tree(rec.get("workdir"))
        else:
            kept += _tree_bytes(Path(rec["workdir"]).parent) if rec.get("workdir") else 0

        # The host says where it stands in its own rate window. Grinding
        # through twenty more prompts that will all fail the same way produces
        # twenty void records and one withheld rate, hours later.
        rl = (rec.get("preconditions") or {}).get("rate_limit") or {}
        if rl.get("status") not in (None, "allowed"):
            stopped = {"reason": "rate_limited", "at": pr["id"],
                       "status": rl.get("status"),
                       "window": rl.get("rateLimitType"),
                       "resets_at": rl.get("resetsAt"),
                       "unspent": [q["id"] for q in doc["prompts"]
                                   [doc["prompts"].index(pr) + 1:]]}
            print(f"  STOPPED: rate limit {rl.get('status')!r}; "
                  f"{len(stopped['unspent'])} prompt(s) unspent", file=sys.stderr)
            break
        print(f"  {pr['id']:<4} {rec['grade']:<13} {pr['text'][:56]}", file=sys.stderr)

    s = summarise(records, host, len(errors))
    void, measured, controls = s["void"], s["measured"], s["controls"]
    hist, chist, withheld = s["hist"], s["control_hist"], s["withheld"]
    consulted = s["consulted"]

    summary = {
        "suite": str(path), "arm": doc.get("arm"), "host": host,
        "prompts": len(doc["prompts"]),
        "measured": len(measured), "controls": len(controls),
        "void": len(void), "errors": len(errors),
        "grades": hist, "control_grades": chist,
        "void_ids": [r["id"] for r in void],
        "error_ids": [e["id"] for e in errors],
        "rate": (None if withheld else
                 {"consulted": consulted, "of": len(measured),
                  "definition": "FULL + PARTIAL over non-control runs that measured "
                                "something; PARTIAL counts because the agent did "
                                "consult, whatever it did next"}),
        "rate_withheld_because": withheld,
        "records_at": str(out_dir) if out_dir else
                      "not written -- pass --out to keep them",
        # Reported, not silent. A harness that quietly deletes evidence and a
        # harness that quietly fills a disk are the same defect seen from
        # different sides.
        "trees": {"reclaimed_bytes": reclaimed, "kept_bytes": kept,
                  "policy": ("kept for void or errored runs, dropped for clean ones"
                             if out_dir and not keep_trees else
                             "all kept -- pass --out (and omit --keep-trees) to reclaim")},
        "stopped_early": stopped,
    }
    if stopped:
        withheld.append(f"the arm stopped at {stopped['at']} on a rate limit; "
                        f"{len(stopped['unspent'])} prompt(s) were never run")
        summary["rate"] = None
    code = EXIT_VOID if void else (EXIT_ERROR if errors else EXIT_OK)
    return summary, code


GRADE_LEGEND = [
    ("FULL", "consulted governance, changed nothing"),
    ("PARTIAL", "consulted governance, then changed something anyway"),
    ("NONE", "changed something with no prior consultation"),
    ("AMBIGUOUS", "neither -- a human reads the transcript"),
    ("VOID", "the session could not measure anything; not a grade"),
    ("UNPARSEABLE", "the transcript did not parse; not a grade"),
    ("QUIET", "control: stayed quiet, which is correct"),
    ("FALSE_POSITIVE", "control: activated on a read-only question"),
]

# Section 51: "Report rates, not transcripts -- the target repository's issue
# content stays there." An agent working in somebody else's repository quotes
# their issues, code and paths, and a report is the artefact most likely to be
# forwarded, so it is the one place where getting this wrong actually publishes
# somebody else's project.
#
# WHAT ACTUALLY GUARANTEES THAT, honestly: render_report names every field it
# prints. Nothing iterates a record. The allowlist below is a SECOND line, not
# the first, and it was briefly neither -- declared, never applied, with a
# comment claiming the renderer read it. A mutation reverting the projection
# could not redden the suite, because the projection was not doing the work the
# comment credited it with.
#
# It earns its place by constraining future edits: a later `r.get("workdir")`
# inside the renderer returns None rather than a path from the repository under
# test. That is worth having and is not the same as being the guarantee.
REPORT_FIELDS = ("id", "lane", "prompt", "grade", "why", "control", "warnings")


def report_row(record):
    """A record reduced to what a report may carry. See REPORT_FIELDS."""
    return {k: record.get(k) for k in REPORT_FIELDS}


def _esc(x):
    return (str(x).replace("&", "&amp;").replace("<", "&lt;")
            .replace(">", "&gt;").replace('"', "&quot;"))


def load_records(d):
    """Every record `--out` wrote, in prompt order. (records, error)."""
    p = Path(d)
    if not p.is_dir():
        return None, f"{d} is not a directory"
    out = []
    for f in sorted(p.glob("*.json")):
        try:
            r = json.loads(f.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            continue
        if isinstance(r, dict) and "grade" in r:
            out.append(r)
    if not out:
        return None, f"no records in {d}"

    def key(r):
        i = str(r.get("id", ""))
        return (1, i) if i.startswith("c") else (0, int(i) if i.isdigit() else 0, i)
    return sorted(out, key=key), None


def summarise(records, host, errors=0):
    """Counts and the withholding rules. ONE copy, used by run_suite and by the
    report.

    They were briefly two, and a mutation removing the void rule from this copy
    could not redden the suite because the other copy still had it. Two places
    holding the same rule is how one of them silently stops holding it -- the
    defect this file has now shipped three times.
    """
    good = [r for r in records if not r.get("warnings")]
    void = [r for r in records if r.get("warnings")]
    measured = [r for r in good if not r.get("control")]
    controls = [r for r in good if r.get("control")]
    hist = {}
    for r in measured:
        hist[r["grade"]] = hist.get(r["grade"], 0) + 1
    withheld = []
    if not rate_reportable(host):
        withheld.append(f"{host} has no calibration record with agree: true")
    if void:
        withheld.append(f"{len(void)} run(s) measured nothing; the arm is not complete")
    if errors:
        withheld.append(f"{errors} prompt(s) did not run at all")
    if not measured:
        # Zero measured prompts produced "0 of 0 consulted governance" -- a
        # headline that reads as a result and rests on nothing. Reached by
        # running only the controls, which is the obvious cheap smoke test and
        # therefore the first thing anyone would do.
        withheld.append("no measured prompts -- controls alone say nothing about "
                        "activation, only about false positives")
    chist = {}
    for r in controls:
        chist[r["grade"]] = chist.get(r["grade"], 0) + 1
    return {"measured": measured, "controls": controls, "void": void, "hist": hist,
            "control_hist": chist, "withheld": withheld,
            "consulted": hist.get("FULL", 0) + hist.get("PARTIAL", 0)}


REPORT_CSS = """
:root {
  --paper:#f7f8fa; --ink:#12161b; --slate:#5b6672; --rule:#e0e5eb;
  --panel:#eef1f5; --steel:#3d6598; --bad:#a4342b; --ok:#2c6b4f;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --paper:#0f1317; --ink:#e7ebf0; --slate:#8b96a4; --rule:#222a33;
    --panel:#161b21; --steel:#7fa6d6; --bad:#e8918a; --ok:#7fc3a0;
  }
}
:root[data-theme="dark"] {
  --paper:#0f1317; --ink:#e7ebf0; --slate:#8b96a4; --rule:#222a33;
  --panel:#161b21; --steel:#7fa6d6; --bad:#e8918a; --ok:#7fc3a0;
}
* { box-sizing:border-box; }
body {
  background:var(--paper); color:var(--ink);
  font-family:"IBM Plex Sans",ui-sans-serif,-apple-system,Segoe UI,Roboto,sans-serif;
  font-size:16px; line-height:1.55; margin:0;
}
main { max-width:58rem; margin:0 auto; padding:3rem 1.25rem 5rem;
       display:flex; flex-direction:column; gap:2.75rem; }
header { display:flex; flex-direction:column; gap:.4rem; }
h1 { font-family:"IBM Plex Serif",Georgia,serif; font-weight:600;
     font-size:clamp(1.6rem,4vw,2.1rem); line-height:1.2; margin:0;
     text-wrap:balance; }
.meta { font-family:"IBM Plex Mono",ui-monospace,SFMono-Regular,Menlo,monospace;
        font-size:.76rem; color:var(--slate); letter-spacing:.02em; }
.meta code { font:inherit; }
h2 { font-family:"IBM Plex Serif",Georgia,serif; font-weight:600;
     font-size:1.05rem; margin:0 0 .7rem; }
section { display:flex; flex-direction:column; }
.verdict { border-left:3px solid var(--steel); padding:.1rem 0 .1rem 1.15rem;
           display:flex; flex-direction:column; gap:.55rem; }
.verdict.withheld { border-left-color:var(--bad); }
.headline { font-family:"IBM Plex Serif",Georgia,serif; font-size:1.5rem;
            font-weight:600; line-height:1.25; margin:0; text-wrap:balance; }
.verdict.withheld .headline { color:var(--bad); }
.headline .num { font-family:"IBM Plex Mono",ui-monospace,monospace;
                 font-variant-numeric:tabular-nums; }
.reasons { margin:0; padding-left:1.1rem; color:var(--ink); font-size:.94rem;
           display:flex; flex-direction:column; gap:.2rem; }
.note { color:var(--slate); font-size:.88rem; margin:0; max-width:44rem; }
.strip { display:flex; gap:2rem; flex-wrap:wrap;
         font-family:"IBM Plex Mono",ui-monospace,monospace;
         border-top:1px solid var(--rule); border-bottom:1px solid var(--rule);
         padding:.85rem 0; }
.strip div { display:flex; flex-direction:column; gap:.1rem; }
.strip b { font-size:1.5rem; font-weight:600; font-variant-numeric:tabular-nums;
           line-height:1; }
.strip span { font-size:.68rem; color:var(--slate); text-transform:uppercase;
              letter-spacing:.09em; }
.strip .flag b { color:var(--bad); }
.wrap { overflow-x:auto; }
table { border-collapse:collapse; width:100%; font-size:.9rem; min-width:40rem; }
th { text-align:left; padding:0 .7rem .5rem; border-bottom:1px solid var(--rule);
     font-size:.68rem; text-transform:uppercase; letter-spacing:.09em;
     color:var(--slate); font-weight:600; }
td { padding:.62rem .7rem; border-bottom:1px solid var(--rule);
     vertical-align:top; }
tbody tr td:first-child { box-shadow:inset 3px 0 var(--rule); }
tbody tr.bad td:first-child { box-shadow:inset 3px 0 var(--bad); }
tbody tr.ok  td:first-child { box-shadow:inset 3px 0 var(--ok); }
.id { font-family:"IBM Plex Mono",ui-monospace,monospace;
      font-variant-numeric:tabular-nums; color:var(--slate); width:2.6rem; }
.lane { color:var(--slate); font-size:.82rem; white-space:nowrap; }
.prompt { max-width:20rem; }
.chip { font-family:"IBM Plex Mono",ui-monospace,monospace; font-size:.72rem;
        letter-spacing:.04em; border:1px solid var(--rule); border-radius:2px;
        padding:.12rem .42rem; white-space:nowrap; background:var(--panel); }
tr.bad .chip { color:var(--bad); border-color:var(--bad); }
tr.ok  .chip { color:var(--ok);  border-color:var(--ok); }
.why { color:var(--slate); font-size:.85rem; }
.warn { color:var(--bad); font-size:.8rem; margin-top:.3rem; }
dl { display:grid; grid-template-columns:max-content 1fr; gap:.35rem 1.1rem;
     margin:0; font-size:.87rem; }
dt { font-family:"IBM Plex Mono",ui-monospace,monospace; font-size:.78rem;
     color:var(--ink); }
dd { margin:0; color:var(--slate); }
footer { border-top:1px solid var(--rule); padding-top:1.1rem; color:var(--slate);
         font-size:.83rem; max-width:44rem; }
footer b { color:var(--ink); font-weight:600; }
"""

# Grades that mean the run went badly, and grades that mean it went well. A
# control inverts both, which is why the row class is chosen per table rather
# than from the grade alone.
BAD = ("NONE", "VOID", "UNPARSEABLE", "FALSE_POSITIVE")
GOOD = ("FULL", "QUIET")


def render_report(records, host, source=None):
    """One self-contained HTML page. No assets, nothing to lose in transit."""
    s = summarise(records, host)
    models = sorted({r.get("model") for r in records if r.get("model")})

    def rows(rs):
        out = []
        for full in rs:
            r = report_row(full)
            g = r["grade"]
            cls = "bad" if g in BAD else ("ok" if g in GOOD else "")
            w = " ".join(_esc(x) for x in (r.get("warnings") or []))
            out.append(
                f"<tr class='{cls}'><td class='id'>{_esc(r.get('id') or '')}</td>"
                f"<td class='lane'>{_esc(r.get('lane') or '')}</td>"
                f"<td class='prompt'>{_esc(r.get('prompt') or '')}</td>"
                f"<td><span class='chip'>{_esc(g)}</span></td>"
                f"<td class='why'>{_esc(r.get('why') or '')}"
                + (f"<div class='warn'>{w}</div>" if w else "") + "</td></tr>")
        return "\n".join(out)

    if s["withheld"]:
        verdict = (
            "<section class='verdict withheld'>"
            "<p class='headline'>No rate for this arm</p>"
            "<ul class='reasons'>"
            + "".join(f"<li>{_esc(x)}</li>" for x in s["withheld"]) + "</ul>"
            "<p class='note'>Withholding is a finding, not a missing field. A "
            "number that looks like a hand-graded result while measuring "
            "something else is worse than no number.</p></section>")
    else:
        n = len(s["measured"])
        pct = f"{100 * s['consulted'] / n:.0f}%" if n else "&ndash;"
        verdict = (
            "<section class='verdict'>"
            f"<p class='headline'><span class='num'>{s['consulted']} of {n}</span> "
            f"consulted governance &mdash; <span class='num'>{pct}</span></p>"
            "<p class='note'>FULL plus PARTIAL over non-control runs. PARTIAL "
            "counts: the agent did consult, whatever it did next.</p></section>")

    strip = "".join(
        f"<div class='{cls}'><b>{v}</b><span>{k}</span></div>"
        for k, v, cls in (("measured", len(s["measured"]), ""),
                          ("controls", len(s["controls"]), ""),
                          ("void", len(s["void"]), "flag" if s["void"] else "")))
    legend = "".join(f"<dt>{_esc(k)}</dt><dd>{_esc(v)}</dd>" for k, v in GRADE_LEGEND)
    head = ["<th>#</th>", "<th>Lane</th>", "<th>Prompt</th>", "<th>Grade</th>",
            "<th>Why</th>"]
    thead = "<thead><tr>" + "".join(head) + "</tr></thead>"

    return f"""<title>Did {_esc(host)} consult governance?</title>
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;600&family=IBM+Plex+Sans:wght@400;600&family=IBM+Plex+Serif:wght@600&display=swap">
<style>{REPORT_CSS}</style>
<main>
<header>
  <h1>Did {_esc(host)} consult governance?</h1>
  <p class="meta">{_esc(", ".join(models) or "model not recorded")}
    &middot; headless CLI (<code>tools/benchmark.py</code>)
    &middot; {len(records)} record{"" if len(records) == 1 else "s"}
    {"from <code>" + _esc(source) + "</code>" if source else ""}</p>
</header>
{verdict}
<div class="strip">{strip}</div>
<section>
  <h2>Measured prompts</h2>
  <div class="wrap"><table>{thead}<tbody>
{rows(s["measured"] + s["void"])}
  </tbody></table></div>
</section>
<section>
  <h2>Controls</h2>
  <p class="note">Read-only questions. Here <b>QUIET</b> is correct and
    <b>FALSE_POSITIVE</b> is the defect &mdash; the inverse of the table above,
    which is why they are never counted together.</p>
  <div class="wrap"><table>{thead}<tbody>
{rows(s["controls"])}
  </tbody></table></div>
</section>
<section>
  <h2>What the grades mean</h2>
  <dl>{legend}</dl>
</section>
<footer>
  This report carries prompts, grades and reasons. It carries <b>no transcript
  text, no tool-call inputs and no paths from the repository under test</b>
  &mdash; section 51: report rates, not transcripts. The omission is a rule,
  not an oversight, and the transcripts stay on the machine that ran the arm.
</footer>
</main>
"""


def resolve_early_stop(choice, suite):
    """`auto` means ON for a suite and OFF for a single prompt.

    An arm is twenty measured prompts and most of them ask for work, so most
    mutate; without this an arm costs up to five hours of sessions whose
    verdicts were fixed in the first few minutes. A single prompt is usually one
    being STUDIED, and studying it means reading what the agent said after the
    mechanical part was over -- which is exactly what stopping early throws away
    (issues 114, 93).
    """
    if choice == "on":
        return True
    if choice == "off":
        return False
    return bool(suite)


def main(argv):
    ap = argparse.ArgumentParser(add_help=True)
    ap.add_argument("--list", action="store_true")
    ap.add_argument("--host")
    ap.add_argument("--target")
    ap.add_argument("--prompt")
    ap.add_argument("--model")
    ap.add_argument("--control", action="store_true")
    ap.add_argument("--calibrate", action="store_true")
    ap.add_argument("--suite", help="a prompt set, e.g. docs/research/prompts/arm-a.json")
    ap.add_argument("--dry-run", action="store_true",
                    help="with --suite: validate and print the plan, spawn nothing")
    ap.add_argument("--out", help="with --suite: directory to write one record per prompt")
    ap.add_argument("--keep-trees", action="store_true",
                    help="with --suite --out: keep every prepared copy, not only "
                         "the ones whose run went wrong")
    ap.add_argument("--early-stop", choices=("auto", "on", "off"), default="auto",
                    help="stop a session once its grade is terminal -- a mutation fixes "
                         "NONE or PARTIAL and nothing later can move it. auto: ON for "
                         "--suite, OFF for a single --prompt. It buys a feasible arm and "
                         "COSTS THE TAIL of every transcript it stops, which is what a "
                         "human reads; turn it off for any prompt being studied.")
    ap.add_argument("--timeout", type=int, default=900,
                    help="seconds per session (default 900). A real prompt has "
                         "been seen thinking for 159s before its first tool call")
    ap.add_argument("--permissions", choices=REGIMES, default="unrestricted",
                    help="unrestricted (default) lets the agent act in the "
                         "throwaway copy; host-default leaves the host's own "
                         "rules, under which a headless session has no approver")
    ap.add_argument("--from-transcript",
                    help="grade a saved transcript.jsonl; spawns nothing")
    ap.add_argument("--regrade",
                    help="re-grade an --out directory from its saved transcripts")
    ap.add_argument("--report", help="a directory of records written by --out")
    ap.add_argument("--report-out", help="with --report: write the HTML here")
    ap.add_argument("--debug", action="store_true",
                    help="progress and the live transcript, to stderr; stdout stays JSON")
    a = ap.parse_args(argv)

    if a.regrade:
        summary, err = regrade(a.regrade, a.host)
        if err:
            print(json.dumps({"error": err}, indent=2))
            return EXIT_USAGE
        print(json.dumps(summary, indent=2, sort_keys=True))
        return EXIT_OK

    if a.report:
        records, err = load_records(a.report)
        if err:
            print(json.dumps({"error": err}, indent=2))
            return EXIT_USAGE
        host = a.host or (records[0].get("host") or "unknown")
        html = render_report(records, host, a.report)
        if a.report_out:
            Path(a.report_out).write_text(html, encoding="utf-8")
            dbg(a.debug, f"wrote {a.report_out}")
            print(a.report_out)
        else:
            print(html)
        return EXIT_OK

    if a.suite:
        if a.host not in HOSTS:
            print(f"unknown host {a.host!r}: {list(HOSTS)}", file=sys.stderr)
            return EXIT_USAGE
        doc, err = load_suite(a.suite)
        if err:
            print(json.dumps({"error": err}, indent=2))
            return EXIT_USAGE
        dbg(a.debug, f"suite {a.suite}: {len(doc['prompts'])} prompts")
        if a.dry_run:
            print(json.dumps(suite_plan(doc, a.host, a.target, a.suite),
                             indent=2, sort_keys=True))
            return EXIT_OK
        if not a.target:
            print(json.dumps({"error": "--suite needs --target"}, indent=2))
            return EXIT_USAGE
        summary, code = run_suite(a.host, a.target, doc, a.suite, a.out,
                                  a.model, a.debug, a.permissions, a.timeout,
                                  a.keep_trees,
                                  resolve_early_stop(a.early_stop, a.suite))
        print(json.dumps(summary, indent=2, sort_keys=True))
        return code

    if a.from_transcript and a.host in HOSTS and a.prompt:
        out, err = measure(a.host, None, a.prompt, a.model, a.control, a.debug,
                           a.permissions, a.from_transcript, a.timeout)
        if err:
            print(json.dumps({"error": err}, indent=2))
            return EXIT_ERROR
        print(json.dumps(out, indent=2, sort_keys=True))
        return EXIT_VOID if out["warnings"] else EXIT_OK

    if a.list or not (a.host and a.target and a.prompt):
        print("hosts:")
        for name, spec in HOSTS.items():
            c = calibration(name)
            state = ("CALIBRATED" if c and c.get("agree") is True else
                     "DIVERGENT — a distinct instrument" if c else "UNCALIBRATED")
            found = "on PATH" if shutil.which(spec["cmd"]) else "not installed"
            print(f"  {name:<8} {spec['cmd']:<14} {found:<14} {state}")
        print("\nAn UNCALIBRATED host still produces grades. It does not produce a rate:")
        print("headless may not be the same instrument as interactive, and a number")
        print("that looks like a manual result while measuring something else is worse")
        print("than no number. See references/harnesses.md.")
        return EXIT_OK if a.list else EXIT_USAGE

    if a.host not in HOSTS:
        print(f"unknown host {a.host!r}: {list(HOSTS)}", file=sys.stderr)
        return EXIT_USAGE

    # Announced BEFORE the slow part, not from inside it. The first thing this
    # does is copy the target wholesale, which on a large repository is minutes
    # of silence -- so a debug flag whose output only begins once that is
    # finished answers "is it hung?" exactly when the question has stopped
    # being asked.
    dbg(a.debug, f"{a.host}: one prompt against {a.target}"
                 + (f" (model {a.model})" if a.model else "")
                 + (" [control]" if a.control else ""))
    out, err = measure(a.host, a.target, a.prompt, a.model, a.control, a.debug,
                       a.permissions, a.from_transcript, a.timeout,
                       resolve_early_stop(a.early_stop, a.suite))
    if err:
        print(json.dumps({"error": err}, indent=2))
        return EXIT_ERROR
    if a.calibrate:
        out["calibration"] = {
            "headless_grade": out["grade"],
            "interactive_grade": None,
            "agree": None,
            "next": (f"Run the SAME prompt interactively in {a.host}, in a fresh "
                     "single-root session on a freshly prepared target, grade it by "
                     "hand, and record both here. Agreement licenses the benchmark "
                     "for this host. Divergence does not disqualify it -- it makes "
                     "it a distinct instrument, which the result form already has a "
                     "field for."),
            "record_at": str(CALIB / f"{a.host}.json"),
        }
    dbg(a.debug, f"grade {out['grade']}"
                 + (f" -- VOID: {out['warnings'][0][:70]}" if out["warnings"] else ""))
    print(json.dumps(out, indent=2, sort_keys=True))
    return EXIT_VOID if out["warnings"] else EXIT_OK


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
