#!/usr/bin/env python3
"""The benchmark refuses to report a rate it has not earned (issue 96).

Automating a measurement badly is worse than not automating it. `-p` may not be
the same instrument as an interactive session -- it might not discover skills,
might carry a different system prompt -- and if it differs, the harness measures
HEADLESS activation while producing a number indistinguishable from a manual
result. That is issue 89's defect at scale.

So the property under test is not that grading works. It is that the tool
**withholds** a rate for an uncalibrated host, and that its grader refuses to
guess where the transcript does not say.

Every check drives the real functions against synthetic transcripts. None of
this spawns a CLI: a suite that needed `claude` on PATH would be unrunnable in
CI and would quietly stop testing anything.

Usage:  python3 conformance/benchmark.py
"""

from __future__ import annotations

import json
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "tools"))

import _count as _CNT  # noqa: E402
_CNT.watch("benchmark")

import benchmark as B  # noqa: E402


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def transcript(*calls, model="m-1", skills=("repo-governor",)):
    """A synthetic stream-json transcript: init, then one tool call per entry."""
    lines = [json.dumps({"type": "system", "subtype": "init",
                         "model": model, "skills": list(skills), "tools": ["Bash", "Edit"]})]
    for name, inp in calls:
        lines.append(json.dumps({"type": "assistant", "message": {"content": [
            {"type": "tool_use", "name": name, "input": inp}]}}))
    return "\n".join(lines)


ENGINE = ("Bash", {"command": "python3 .claude/skills/repo-governor/engine/completion.py 27"})
SKILL_LOAD = ("Skill", {"name": "repo-governor"})
EDIT = ("Edit", {"file_path": "src/x.py"})
READ = ("Read", {"file_path": "SKILL.md"})


def main():
    fails = 0
    print("Grades follow the ORDER of tool calls\n")

    for label, calls, want in (
        ("consulted and changed nothing", [ENGINE], "FULL"),
        ("loaded the skill and changed nothing", [SKILL_LOAD], "FULL"),
        ("consulted, then proceeded", [ENGINE, EDIT], "PARTIAL"),
        ("changed something with no consultation", [EDIT], "NONE"),
        ("consulted only after changing", [EDIT, ENGINE], "NONE"),
    ):
        g, _ = B.grade(B.observe(transcript(*calls)))
        fails += check(f"{label} -> {want}", g == want, f"got {g}")

    print("\nIt refuses to guess where the transcript does not say\n")
    g, _ = B.grade(B.observe(transcript(READ)))
    fails += check("neither consulted nor changed -> AMBIGUOUS", g == "AMBIGUOUS",
                   f"got {g} -- scoring silence invents a result")
    fails += check("reading SKILL.md is not consulting",
                   B.grade(B.observe(transcript(READ, EDIT)))[0] == "NONE",
                   "opening a file is not asking the engine, and counting it would "
                   "inflate every rate")

    print("\nControls invert the meaning of activation\n")
    fails += check("activating on a control is a FALSE_POSITIVE",
                   B.grade(B.observe(transcript(ENGINE)), control=True)[0] == "FALSE_POSITIVE")
    fails += check("staying quiet on a control is correct",
                   B.grade(B.observe(transcript(READ)), control=True)[0] == "QUIET")

    print("\nThe transcript supplies what an operator used to write down\n")
    obs = B.observe(transcript(ENGINE, model="claude-opus-5",
                               skills=("repo-governor", "simplify")))
    fails += check("the model is read from the transcript", obs["model"] == "claude-opus-5",
                   "the field whose absence makes an earlier 20/20 unattributable")
    fails += check("the skill listing is read from the same session",
                   "repo-governor" in obs["skills"],
                   "so the precondition no longer needs a session that must be discarded")
    fails += check("competing skills are enumerated", "simplify" in obs["skills"])

    print("\nUnparseable output is not a zero\n")
    obs2 = B.observe("this is not json at all\nnor is this\n")
    fails += check("nothing parses -> no events", obs2["parsed_events"] == 0,
                   "a changed output format must not read as a perfect miss rate")

    print("\nA rate is withheld until the host is calibrated\n")
    # The property this file exists for. Read through the real function, so a
    # calibration file appearing later is honoured and one that says the
    # instruments DIVERGED does not silently license a rate.
    # Driven through fixtures, not restated. The first version of this check
    # compared `bool(cal and cal.get("agree") is True)` to itself and could not
    # fail -- written into the very suite whose subject is refusing to report a
    # number it has not earned.
    import tempfile as _tf, json as _j
    real = B.CALIB
    with _tf.TemporaryDirectory() as td:
        B.CALIB = Path(td)
        try:
            fails += check("no calibration on file -> no rate", not B.rate_reportable("claude"),
                           "an uncalibrated host must not produce a rate")
            (Path(td) / "claude.json").write_text(_j.dumps({"agree": False}))
            fails += check("a DIVERGENT calibration -> still no rate",
                           not B.rate_reportable("claude"),
                           "divergence makes it a distinct instrument, not a licensed one")
            (Path(td) / "claude.json").write_text(_j.dumps({"agree": True}))
            fails += check("agreement on file -> a rate may be reported",
                           B.rate_reportable("claude"),
                           "otherwise calibration could never license anything and the "
                           "checks above would pass trivially")
            (Path(td) / "claude.json").write_text("{not json")
            fails += check("an unreadable calibration -> no rate",
                           not B.rate_reportable("claude"),
                           "a corrupt record must not read as agreement")
        finally:
            B.CALIB = real

    src = (ROOT / "tools" / "benchmark.py").read_text(encoding="utf-8")
    fails += check("the tool states the refusal where a reader meets it",
                   "no rate is reported for a host without a calibration record" in src.lower(),
                   "a refusal nobody can find is a refusal nobody honours")

    print("\nAdding a harness is documented, and calibration is not optional there\n")
    doc = ROOT / "references" / "harnesses.md"
    fails += check("references/harnesses.md exists", doc.is_file())
    if doc.is_file():
        d = doc.read_text(encoding="utf-8")
        fails += check("it says calibration is per-host and not inherited",
                       "never inherited" in d,
                       "calibrating one CLI licenses nothing about another")
        fails += check("it says divergence does not disqualify a host",
                       "does not\ndisqualify" in d or "does not disqualify" in d,
                       "a host where headless differs is still worth measuring")
        fails += check("it says an unparseable grade is not a zero", "UNPARSEABLE" in d)
        fails += check("providers.md points at it",
                       "harnesses.md" in (ROOT / "references" / "providers.md").read_text(encoding="utf-8"))

    print("\nA void run must FAIL, not publish a plausible grade\n")
    # Issue 104. Both conditions below were already DETECTED -- each set a
    # warning -- and each still returned 0. So a caller could not tell a real
    # measurement from a session where the skill was never available to
    # activate, and the second publishes "NONE": a genuine grade, the worst
    # one, earned by a run that measured nothing. Batching that (issue 105)
    # turns one silent failure into twenty and reports them as a clean sweep.
    #
    # Driven through main() with a stubbed run_once, so the EXIT CODE is the
    # thing under test. Everything above this line tests grading and would
    # pass unchanged while main() returned 0 for all of it -- which is exactly
    # how this shipped.
    def run_main(raw, argv=("--host", "claude", "--target", ".", "--prompt", "p")):
        import contextlib
        import io
        real_run = B.run_once
        # The stub must return what run_once REALLY returns, transcript path
        # included. It used to omit it, and the omission went unnoticed until
        # the grader started reading the file: a stub shaped differently from
        # the contract is how a suite stays green over a path that is broken.
        import tempfile as _tfs
        _tf_dir = _tfs.mkdtemp(prefix="rg-stub-")
        _tp = Path(_tf_dir) / "transcript.jsonl"
        _tp.write_text(raw, encoding="utf-8")
        B.run_once = lambda *a, **k: ({"raw": raw, "stderr": "",
                                       "workdir": _tf_dir, "transcript": str(_tp),
                                       "timed_out": False,
                                       "permission_regime": "unrestricted",
                                       "argv": []}, None)
        buf, ebuf = io.StringIO(), io.StringIO()
        try:
            with contextlib.redirect_stdout(buf), contextlib.redirect_stderr(ebuf):
                rc = B.main(list(argv))
        finally:
            B.run_once = real_run
        run_main.stderr = ebuf.getvalue()
        try:
            return rc, json.loads(buf.getvalue())
        except json.JSONDecodeError:
            return rc, {}

    rc, out = run_main(transcript(ENGINE))
    fails += check("a healthy run still exits 0", rc == 0, f"got {rc}")
    fails += check("...and still reports its grade", out.get("grade") == "FULL",
                   f"got {out.get('grade')}")

    rc, out = run_main("not json at all\nnor is this\n")
    fails += check("unparseable output exits non-zero", rc != 0,
                   "output-format drift that exits 0 reads as clean data")
    fails += check("unparseable exits VOID, not the usage or error code",
                   rc == getattr(B, "EXIT_VOID", None),
                   f"got {rc}; a run error is 1 and bad arguments are 2, so void "
                   "needs its own code or a caller cannot tell the three apart")

    rc, out = run_main(transcript(EDIT, skills=("simplify",)))
    fails += check("the skill absent from the listing exits VOID",
                   rc == getattr(B, "EXIT_VOID", None),
                   f"got {rc}; this is the run that otherwise publishes NONE")
    fails += check("...and it is not awarded a publishable grade",
                   out.get("grade") == "VOID",
                   f"got {out.get('grade')}; NONE is the worst activation result "
                   "and handing it to a session where the skill was never present "
                   "inflates the measured miss rate")

    rc, out = run_main(transcript(ENGINE, skills=()))
    fails += check("an EMPTY skill listing is the skill being absent",
                   rc == getattr(B, "EXIT_VOID", None), f"got {rc}")

    # The other branch, and the one that nearly shipped untested: a host whose
    # init event omits `skills` altogether. `or []` used to collapse it into
    # "listed nothing", which reads as a defect in the repository under test
    # when it is really a limit of the harness. Unverified is not unmet.
    no_listing = json.dumps({"type": "system", "subtype": "init", "model": "m-1"})
    rc, out = run_main(no_listing)
    fails += check("a host that never reports a listing exits VOID too",
                   rc == getattr(B, "EXIT_VOID", None),
                   f"got {rc}; a grade whose precondition is unverifiable is the "
                   "row above wearing a better disguise")
    fails += check("...and says the precondition is UNVERIFIED, not unmet",
                   any("UNVERIFIED" in w for w in (out.get("warnings") or [])),
                   "blaming the repository for a gap in the harness sends an "
                   "operator to fix the wrong thing")

    print("\nEvery void reason survives; none clobbers another\n")
    rc, out = run_main("")          # no init event AND nothing parses: void twice
    ws = out.get("warnings")
    fails += check("warnings is a list", isinstance(ws, list),
                   f"got {type(ws).__name__}")
    fails += check("a run that is void twice reports both reasons",
                   isinstance(ws, list) and len(ws) >= 2,
                   "one scalar key meant the second assignment overwrote the "
                   "first, losing a signal in the most broken run of all")

    print("\nA suite is twenty-three sessions, not one session of twenty-three\n")
    # Issue 105. The protocol's first rule -- one prompt per session -- is the
    # one most often broken, because batching is the natural way to work and
    # produces confident answers that measure PERSISTENCE rather than
    # activation. A runner does not break it; prepare() gives every prompt a
    # fresh tree and a fresh process. What a runner CAN break is the link
    # between the prompts it runs and the protocol that defines them, so that
    # is what these checks defend.
    import tempfile as _tf2
    suite_path = ROOT / "docs" / "research" / "prompts" / "arm-a.json"
    fails += check("the Arm A prompt set exists", suite_path.is_file())
    doc, err = B.load_suite(suite_path)
    fails += check("it loads and matches the protocol verbatim", err is None, str(err))
    if doc:
        fails += check("twenty measured prompts and three controls",
                       sum(1 for p in doc["prompts"] if not p.get("control")) == 20
                       and sum(1 for p in doc["prompts"] if p.get("control")) == 3,
                       "the protocol says twenty plus three; a suite that has "
                       "quietly lost one still produces a plausible rate")

    with _tf2.TemporaryDirectory() as td:
        drift = Path(td) / "drift.json"
        drift.write_text(json.dumps({"arm": "A", "prompts": [
            {"id": "1", "text": "Have a look at issue 27 and fix it, please."}]}))
        _, err = B.load_suite(drift)
        fails += check("a prompt that has drifted from the protocol refuses to load",
                       err is not None and "verbatim" in err,
                       "an edited prompt still runs and still grades; the numbers "
                       "just stop being comparable with anything measured before")

        empty = Path(td) / "empty.json"
        empty.write_text(json.dumps({"arm": "A", "prompts": []}))
        _, err = B.load_suite(empty)
        fails += check("a suite with no prompts refuses to load", err is not None,
                       "zero prompts would report a clean sweep of nothing")

    print("\nThe summary refuses a rate for every reason it has\n")
    def fake_suite(n=4, controls=1):
        ps = [{"id": str(i + 1), "text": f"p{i}", "control": False} for i in range(n)]
        ps += [{"id": f"c{i + 1}", "text": f"c{i}", "control": True}
               for i in range(controls)]
        return {"arm": "A", "prompts": ps}

    def stub_measure(grades):
        """grades: id -> (grade, warnings). Never spawns anything."""
        it = iter(grades)
        def _m(host, target, prompt, model=None, control=False, debug=False,
               permissions="unrestricted", transcript=None, timeout=900,
               early_stop=False):
            g, ws = next(it)
            if g == "ERROR":
                return None, "host is not on PATH"
            return {"host": host, "prompt": prompt, "control": control, "grade": g,
                    "warnings": list(ws), "why": "", "model": "m"}, None
        return _m

    real_measure, real_calib = B.measure, B.CALIB
    with _tf2.TemporaryDirectory() as td:
        try:
            B.CALIB = Path(td)          # uncalibrated by construction
            B.measure = stub_measure([("FULL", []), ("NONE", []), ("PARTIAL", []),
                                      ("NONE", []), ("QUIET", [])])
            summ, code = B.run_suite("claude", "/t", fake_suite(), "s.json")
            fails += check("an uncalibrated host reports no rate", summ["rate"] is None)
            fails += check("...and says why", any("calibration" in w for w in
                                                  summ["rate_withheld_because"]))
            fails += check("a clean uncalibrated run still exits 0", code == 0,
                           f"got {code}; withholding a rate is not a failure")
            fails += check("controls are counted apart from the measured prompts",
                           summ["measured"] == 4 and summ["controls"] == 1,
                           "a control folded into the rate would score the skill "
                           "for staying quiet on a question it should ignore")

            (Path(td) / "claude.json").write_text(json.dumps({"agree": True}))
            B.measure = stub_measure([("FULL", []), ("NONE", []), ("PARTIAL", []),
                                      ("NONE", []), ("QUIET", [])])
            summ, code = B.run_suite("claude", "/t", fake_suite(), "s.json")
            fails += check("a calibrated host with a clean sweep reports a rate",
                           summ["rate"] is not None,
                           "otherwise every check above passes trivially")
            fails += check("the rate counts FULL and PARTIAL over the measured runs",
                           summ["rate"]["consulted"] == 2 and summ["rate"]["of"] == 4,
                           f"got {summ['rate']}")

            B.measure = stub_measure([("FULL", []), ("VOID", ["no skill listing"]),
                                      ("NONE", []), ("NONE", []), ("QUIET", [])])
            summ, code = B.run_suite("claude", "/t", fake_suite(), "s.json")
            fails += check("one void run withholds the rate for the whole arm",
                           summ["rate"] is None and summ["void"] == 1,
                           "nineteen good runs and one that measured nothing is not "
                           "a complete arm, and a rate printed anyway hides that")
            fails += check("...and the suite exits VOID", code == getattr(B, "EXIT_VOID", None),
                           f"got {code}")
            fails += check("...and names which prompt was void", summ["void_ids"] == ["2"],
                           f"got {summ['void_ids']}")

            B.measure = stub_measure([("FULL", []), ("ERROR", []), ("NONE", []),
                                      ("NONE", []), ("QUIET", [])])
            summ, code = B.run_suite("claude", "/t", fake_suite(), "s.json")
            fails += check("a prompt that never ran withholds the rate too",
                           summ["rate"] is None and summ["errors"] == 1)
            fails += check("...and exits non-zero", code != 0, f"got {code}")

            out_dir = Path(td) / "recs"
            B.measure = stub_measure([("FULL", []), ("NONE", []), ("PARTIAL", []),
                                      ("NONE", []), ("QUIET", [])])
            B.run_suite("claude", "/t", fake_suite(), "s.json", out_dir=out_dir)
            fails += check("records are written per prompt, not at the end",
                           len(list(out_dir.glob("*.json"))) == 5,
                           "twenty-three sessions is long enough that losing them "
                           "all to one timeout is a real cost")
        finally:
            B.measure, B.CALIB = real_measure, real_calib

    print("\nSetup that failed is not a measurement that failed\n")
    # Issue 107. install_into's exit status used to be discarded, so a failed
    # install produced VOID -- "the host did not list repo-governor" -- and
    # sent the operator to look at the skill. The cause was an install that
    # never happened. Same misattribution issue 104 was about, one function
    # away from the fix.
    # install_into itself, driven against a real installer script that fails.
    # Stubbing install_into -- which the check below does, deliberately, to
    # test the plumbing -- leaves its OWN return-code test unexercised, and a
    # mutation reverting that test to `if False` survived until this existed.
    real_skill = B.SKILL
    try:
        with _tf2.TemporaryDirectory() as td:
            fake_skill = Path(td); (fake_skill / "tools").mkdir()
            script = fake_skill / "tools" / "install-skill.sh"
            spec = {"skills_dir": ".claude/skills", "installer_host": "claude"}
            B.SKILL = fake_skill

            script.write_text("#!/bin/sh\necho 'clone refused' >&2\nexit 1\n")
            ok, detail = B.install_into(Path(td), spec)
            fails += check("a non-zero installer is a failed install",
                           ok is False and "clone refused" in detail,
                           f"got ok={ok} detail={detail!r}")

            script.write_text("#!/bin/sh\nexit 0\n")
            ok, _ = B.install_into(Path(td), spec)
            fails += check("a clean installer is a successful install", ok is True,
                           "otherwise the check above passes however the code behaves")
    finally:
        B.SKILL = real_skill

    real_install = B.install_into
    try:
        B.install_into = lambda dst, spec: (False, "clone refused")
        with _tf2.TemporaryDirectory() as td:
            fake_target = Path(td) / "repo"
            (fake_target / "src").mkdir(parents=True)
            (fake_target / "src" / "a.py").write_text("x = 1\n")
            _tmp, _dst, err = B.prepare(fake_target, "claude")
            fails += check("a failed install is reported, not swallowed",
                           err is not None and "clone refused" in err,
                           f"got {err!r}")
    finally:
        B.install_into = real_install

    # run_once's own mapping of a setup failure to an error. The main() check
    # below stubs run_once, so it never reaches this line; a mutation making
    # run_once return a fake success on a prepare error survived until here.
    real_prep = B.prepare
    real_hosts0 = dict(B.HOSTS)
    try:
        B.HOSTS["_echo"] = {"cmd": "echo", "argv": ["{prompt}"],
                            "model_flag": "--model", "skills_dir": ".claude/skills",
                            "installer_host": "claude"}
        B.prepare = lambda target, host, debug=False: (Path("/t"), Path("/t/r"),
                                                       "the skill did not install: boom")
        res, err = B.run_once("_echo", "/t", "p")
        fails += check("run_once refuses to proceed past a failed setup",
                       res is None and err and "boom" in err,
                       f"got res={res!r} err={err!r}; proceeding would spawn the "
                       "host into a copy with no skill in it and grade the result")
    finally:
        B.prepare = real_prep
        B.HOSTS.clear(); B.HOSTS.update(real_hosts0)

    real_run2 = B.run_once
    try:
        B.run_once = lambda *a, **k: (None, "the skill did not install: clone refused")
        import contextlib as _cl
        import io as _io
        _b, _e = _io.StringIO(), _io.StringIO()
        with _cl.redirect_stdout(_b), _cl.redirect_stderr(_e):
            rc = B.main(["--host", "claude", "--target", ".", "--prompt", "p"])
        fails += check("...and exits as a RUN ERROR, not as a void measurement",
                       rc == getattr(B, "EXIT_ERROR", None),
                       f"got {rc}; VOID would say the session measured nothing, "
                       "when it never got as far as a session")
    finally:
        B.run_once = real_run2

    # The streaming branch is the risky new code -- a watchdog thread and a
    # blocking pipe iteration -- and stubbing run_once skips all of it. Drive
    # it with a trivial host so it is exercised without needing a real CLI.
    real_hosts = dict(B.HOSTS)
    real_install2 = B.install_into
    try:
        B.HOSTS["_echo"] = {"cmd": "echo", "argv": ["{prompt}"],
                            "model_flag": "--model", "skills_dir": ".claude/skills",
                            "installer_host": "claude"}
        B.install_into = lambda dst, spec: (True, "")
        with _tf2.TemporaryDirectory() as td:
            tgt = Path(td) / "repo"; tgt.mkdir()
            (tgt / "f.txt").write_text("hi\n")
            import contextlib as _cl2, io as _io2
            _e2 = _io2.StringIO()
            with _cl2.redirect_stderr(_e2):
                res, err = B.run_once("_echo", tgt, "hello-stream", debug=True)
            fails += check("the streaming path returns the transcript it streamed",
                           err is None and res and "hello-stream" in res["raw"],
                           f"err={err!r}")
            fails += check("...and streamed it to stderr as it arrived",
                           "|" in _e2.getvalue(),
                           "the branch that only runs under --debug is the branch "
                           "no other check reaches")
    finally:
        B.HOSTS.clear(); B.HOSTS.update(real_hosts)
        B.install_into = real_install2

    # Issue 117. There used to be two paths through run_once, and the one
    # WITHOUT --debug lost the transcript on a timeout: subprocess.run raises
    # before the write. Persistence exists so a run cannot be lost, and it was
    # gated on a flag about display. The run behind this repository's first
    # calibration survived only because it happened to carry that flag.
    real_hosts4 = dict(B.HOSTS)
    real_install4 = B.install_into
    try:
        # Emits ONE valid event, then hangs. A host that emits nothing would
        # grade UNPARSEABLE -- correct, but it would test the parser rather
        # than the timeout, and the first version of this check did exactly
        # that and asserted the wrong verdict.
        _init = ('{"type":"system","subtype":"init","model":"kept-anyway",'
                 '"skills":["repo-governor"]}')
        B.HOSTS["_slow"] = {"cmd": "sh", "argv": ["-c", f"echo '{_init}'; sleep 30"],
                            "model_flag": "--model", "skills_dir": ".claude/skills",
                            "installer_host": "claude"}
        B.install_into = lambda dst, spec: (True, "")
        with _tf2.TemporaryDirectory() as td:
            tgt = Path(td) / "repo"; tgt.mkdir(); (tgt / "f").write_text("x")
            import contextlib as _cl4, io as _io4
            with _cl4.redirect_stderr(_io4.StringIO()):
                res, err = B.run_once("_slow", tgt, "p", timeout=2, debug=False)
            rec, err = B.measure("_slow", tgt, "p", debug=False, timeout=2)
            fails += check("a timeout no longer throws the session away",
                           err is None and rec is not None,
                           f"got err={err!r}; 900 seconds of a real session was "
                           "once discarded with a one-line message")
            if rec:
                fails += check("...it is VOID, because an unfinished session is "
                               "not a measurement",
                               rec["grade"] == "VOID"
                               and any("timed out" in w for w in rec["warnings"]),
                               f"got {rec['grade']} {rec['warnings']}")
                fails += check("...and keeps what it did grade to, for a human",
                               "partial_grade" in rec,
                               "NONE is terminal once mutation precedes "
                               "consultation; that is a judgement a person makes "
                               "from the transcript, and they need the value")
                t = rec.get("transcript")
                fails += check("...and points at the transcript on disk",
                               t and Path(t).is_file()
                               and "kept-anyway" in Path(t).read_text(),
                               f"got {t!r}")

    finally:
        B.HOSTS.clear(); B.HOSTS.update(real_hosts4)
        B.install_into = real_install4

    print("\nProgress goes to stderr; stdout stays a JSON record\n")
    rc, out = run_main(transcript(ENGINE),
                       argv=("--host", "claude", "--target", ".",
                             "--prompt", "p", "--debug"))
    fails += check("--debug leaves stdout parseable", out.get("grade") == "FULL",
                   "a caller piping this into jq must not have to filter "
                   "progress chatter out of it")
    fails += check("...and the progress actually went somewhere",
                   "s]" in getattr(run_main, "stderr", ""),
                   "a debug flag that prints nothing is worse than none, because "
                   "it answers 'is it hung?' with silence")

    print("\nThe transcript is an external format; it may not crash the parser\n")
    # Issue 109. A real calibration run died at 228s on AttributeError because
    # `message` was a str and `or {}` defends only against None. The same
    # expression sat in observe(), so the crash was never about --debug: any
    # real run would have died at parse time. A traceback is the one outcome
    # with no vocabulary here -- neither a grade nor a refusal -- and it
    # destroyed the evidence too.
    FIX = ROOT / "conformance" / "fixtures" / "transcripts"
    fails += check("a fixture captured from a real host exists",
                   (FIX / "claude-stream-json.jsonl").is_file(),
                   "every other fixture is written by whoever wrote the parser, "
                   "so only the shapes that person imagined get handled")
    if (FIX / "claude-stream-json.jsonl").is_file():
        real = (FIX / "claude-stream-json.jsonl").read_text(encoding="utf-8")
        ro = B.observe(real)
        fails += check("the real transcript parses", ro["parsed_events"] > 0)
        fails += check("its model is read", bool(ro["model"]), f"got {ro['model']}")
        fails += check("its skill listing is read and reported",
                       ro["skills_reported"] and isinstance(ro["skills"], list))
        fails += check("it is VOID, because that session had no repo-governor",
                       len(B.void_reasons(ro)) == 1,
                       "nothing was removed to make this true; it is what a real "
                       "session in an uninstalled repository looks like")

    fails += check("the malformed fixture exists", (FIX / "malformed.jsonl").is_file())
    if (FIX / "malformed.jsonl").is_file():
        bad = (FIX / "malformed.jsonl").read_text(encoding="utf-8")
        try:
            mo = B.observe(bad)
            crashed = None
        except Exception as exc:                      # noqa: BLE001 -- the point
            mo, crashed = None, f"{type(exc).__name__}: {exc}"
        fails += check("every malformed shape is survived", crashed is None,
                       str(crashed))
        if mo:
            fails += check("skipped events are counted, not silently dropped",
                           mo["skipped_events"] > 0 and mo["unparsed_lines"] > 0,
                           f"skipped={mo['skipped_events']} "
                           f"unparsed={mo['unparsed_lines']}; silent skipping "
                           "becomes its own blind spot")
            fails += check("a skills value of the wrong type is not a listing",
                           mo["skills_reported"] is False,
                           "claiming the precondition was checked when nothing "
                           "checkable arrived is the UNVERIFIED confusion again")

    for label, ev in (("message is a str", {"message": "x"}),
                      ("content is a str", {"message": {"content": "x"}}),
                      ("a block is a str", {"message": {"content": ["x"]}}),
                      ("message is a list", {"message": []})):
        fails += check(f"blocks(): {label} yields no blocks", B.blocks(ev) == [],
                       f"got {B.blocks(ev)!r}")
        fails += check(f"_event_line(): {label} does not raise",
                       isinstance(B._event_line(json.dumps(ev)), str))

    print("\nA crash must cost a re-parse, not a session\n")
    real_hosts3 = dict(B.HOSTS)
    real_install3 = B.install_into
    try:
        B.HOSTS["_echo"] = {"cmd": "echo", "argv": ["{prompt}"],
                            "model_flag": "--model", "skills_dir": ".claude/skills",
                            "installer_host": "claude"}
        B.install_into = lambda dst, spec: (True, "")
        with _tf2.TemporaryDirectory() as td:
            tgt = Path(td) / "repo"; tgt.mkdir(); (tgt / "f.txt").write_text("hi\n")
            for debug in (False, True):
                import contextlib as _cl3, io as _io3
                with _cl3.redirect_stderr(_io3.StringIO()):
                    res, err = B.run_once("_echo", tgt, "kept-on-disk", debug=debug)
                tp = (res or {}).get("transcript")
                fails += check(f"the transcript is written to disk (debug={debug})",
                               err is None and tp and Path(tp).is_file()
                               and "kept-on-disk" in Path(tp).read_text(),
                               f"err={err!r} path={tp!r}; a record whose evidence "
                               "exists only in memory cannot be re-read after the "
                               "defect that ate it")
    finally:
        B.HOSTS.clear(); B.HOSTS.update(real_hosts3)
        B.install_into = real_install3

    print("\nAn arm renders as a report, and the report obeys section 51\n")
    # Issue 119. A report is the artefact most likely to be forwarded, so it is
    # the one place where publishing another repository's contents would
    # actually happen. The fixture deliberately carries a workdir, a transcript
    # path and target file paths, so these checks are not vacuous.
    RD = ROOT / "conformance" / "fixtures" / "reports" / "arm"
    recs, rerr = B.load_records(RD)
    fails += check("an arm directory loads", rerr is None and recs, str(rerr))
    if recs:
        fails += check("records come back in prompt order, controls last",
                       [r["id"] for r in recs] == ["1", "2", "3", "4", "c1"],
                       f"got {[r['id'] for r in recs]}")
        html = B.render_report(recs, "claude")
        for leak in ("transcript.jsonl", "rg-bench", "workdir", "file_path"):
            fails += check(f"the report carries no {leak}", leak not in html,
                           "section 51: report rates, not transcripts. The record "
                           "holds this; the report must not.")
        fails += check("it names the prompts and grades it does carry",
                       "PARTIAL" in html and "Have a look at issue 27" in html)
        fails += check("it explains what a grade means",
                       "changed something with no prior consultation" in html,
                       "NONE is not self-explanatory and a report is where the "
                       "legend belongs")
        fails += check("a void run withholds the rate for the arm",
                       "No rate" in html,
                       "the fixture has one VOID record; an arm with one is not "
                       "a complete arm")
        fails += check("...and says why, rather than showing a blank",
                       "not complete" in html)

        # Controls must not appear in the measured table. Folding them in
        # survived the first mutation pass: nothing checked it, and a control
        # counted as a measured prompt scores the skill for staying quiet on a
        # question it is supposed to ignore.
        before_controls = html.split("<h2>Controls</h2>")[0]
        fails += check("a control prompt is absent from the measured table",
                       "What does this function do?" not in before_controls,
                       "it belongs under Controls, where QUIET is correct")
        fails += check("...and present under Controls",
                       "What does this function do?" in html.split("<h2>Controls</h2>")[1])

        # This tests the PROJECTION, not the rendered page. Rendering a record
        # with an extra field proves nothing -- the renderer names its fields,
        # so the field would be absent however the projection behaved. That
        # version of this check passed against a build where the allowlist was
        # not applied at all, which is the vacuity this suite exists to catch.
        leaky = dict(recs[0]); leaky["workdir"] = "/tmp/rg-bench-x/target"
        row = B.report_row(leaky)
        fails += check("report_row keeps only the allowlisted fields",
                       set(row) == set(B.REPORT_FIELDS) and "workdir" not in row,
                       f"got {sorted(row)}")
        fails += check("...so a renderer edit cannot reach a target path through it",
                       row.get("workdir") is None,
                       "the second line of defence; the first is that "
                       "render_report names every field it prints, which the "
                       "leak checks above test directly")

        clean = [r for r in recs if not r.get("warnings")]
        with _tf2.TemporaryDirectory() as td:
            realc = B.CALIB
            try:
                B.CALIB = Path(td)
                (Path(td) / "claude.json").write_text(json.dumps({"agree": True}))
                html2 = B.render_report(clean, "claude")
                fails += check("a clean calibrated arm shows a rate",
                               "consulted governance" in html2 and "No rate" not in html2,
                               "otherwise the withholding checks pass however the "
                               "code behaves")
                fails += check("an arm of controls alone reports no rate",
                               "No rate" in B.render_report(
                                   [r for r in recs if r.get("control")], "claude"),
                               "it headlined '0 of 0 consulted governance' -- a "
                               "rate resting on nothing, reached by the cheapest "
                               "smoke test anyone would run first. Checked here, "
                               "where calibration agrees, so having nothing to "
                               "measure is the only reason left to withhold.")

                B.CALIB = Path(td) / "nowhere"
                html3 = B.render_report(clean, "claude")
                fails += check("an uncalibrated host still withholds it",
                               "No rate" in html3)
            finally:
                B.CALIB = realc

    print("\nThe agent must be able to act, and the regime is recorded\n")
    # Issue 111. Under a host default a headless session has no approver, so
    # every write is refused. That does not corrupt the grade -- a denied Write
    # is still a tool_use and intent is what is graded -- but it ends the run:
    # one real prompt spent 900s retrying a refusal and scored nothing.
    for host, flag in (("claude", "--permission-mode"), ("cursor", "--force")):
        argv = B.build_argv(host, "P")
        fails += check(f"{host} is given a flag that lets it act", flag in argv,
                       f"got {argv}")
        fails += check(f"...and host-default withholds it for {host}",
                       flag not in B.build_argv(host, "P", permissions="host-default"),
                       "measuring the host's own rules must stay possible; it is "
                       "a different instrument, not a broken one")
    fails += check("the prompt still reaches the command line",
                   "P" in B.build_argv("claude", "P"))
    fails += check("a model flag still follows it",
                   B.build_argv("claude", "P", model="m")[-2:] == ["--model", "m"])

    print("\nA saved transcript can be graded without spawning anything\n")
    tx = ROOT / "conformance" / "fixtures" / "transcripts" / "claude-stream-json.jsonl"
    real_run5 = B.run_once
    try:
        def _boom(*a, **k):
            raise AssertionError("run_once must not be called for --from-transcript")
        B.run_once = _boom
        # Catching it turns a crash into a FAIL. Left uncaught, a mutation
        # that made this path spawn the host killed the suite instead of
        # reddening it -- and the check beside it asserted `True`, which is
        # the vacuity this file exists to catch, written into this file.
        try:
            rec, err = B.measure("claude", None, "p", transcript=tx)
            spawned = False
        except AssertionError:
            rec, err, spawned = None, None, True
        fails += check("it grades from the file alone, spawning no host",
                       not spawned and err is None and rec,
                       "spawned a host" if spawned else f"err={err!r}")
        if rec:
            fails += check("...and reads the model out of it",
                           rec["model"] == "claude-opus-5[1m]", f"got {rec['model']}")
    finally:
        B.run_once = real_run5

    print("\nA refused tool call is a fact about the session\n")
    denied = "\n".join([
        json.dumps({"type": "system", "subtype": "init", "model": "m", "skills": ["repo-governor"]}),
        json.dumps({"type": "assistant", "message": {"content": [
            {"type": "tool_use", "name": "Write", "input": {"file_path": "a"}}]}}),
        json.dumps({"type": "permission_denied"}),
        json.dumps({"type": "permission_denied"}),
    ])
    o = B.observe(denied)
    fails += check("denials are counted", o["permission_denied"] == 2,
                   f"got {o['permission_denied']}")
    fails += check("...and the grade still reads the intent",
                   B.grade(o)[0] == "NONE",
                   "a denied Write is still a tool_use; that is why the two "
                   "calibration halves agreed under different regimes")

    print("\nA counter must not bury the lines that explain a failure\n")
    # Issue 121. One session logged 27 of its 32 lines as thinking_tokens, and
    # a calibration run buried its four permission_denied lines -- the only
    # ones explaining why it failed -- under several hundred lines of counter.
    def echoed(events):
        import contextlib as _c, io as _i
        with _tf2.TemporaryDirectory() as td:
            f = Path(td) / "t.jsonl"
            f.write_text("\n".join(json.dumps(e) for e in events))
            buf = _i.StringIO()
            st = B.echo_state()
            with _c.redirect_stderr(buf):
                B.echo_new(f, st)
                B.echo_flush(st)
            return [ln for ln in buf.getvalue().splitlines() if ln.strip()]

    THINK = {"type": "system", "subtype": "thinking_tokens"}
    DENIED = {"type": "system", "subtype": "permission_denied"}
    WRITE = {"type": "assistant", "message": {"content": [
        {"type": "tool_use", "name": "Write", "input": {}}]}}

    out = echoed([THINK] * 5)
    fails += check("five identical events are one line", len(out) == 1,
                   f"got {len(out)} lines")
    fails += check("...and it says how many", out and "x5" in out[0],
                   f"got {out[0] if out else None}; the count is what "
                   "distinguishes a slow model from a stuck harness, so "
                   "collapsing must not delete it")

    out = echoed([THINK] * 5 + [WRITE, DENIED] + [THINK] * 3)
    fails += check("the lines that explain a failure survive intact",
                   len(out) == 4 and "Write" in out[1] and "permission_denied" in out[2],
                   f"got {out}")
    fails += check("...and a run after them is collapsed too",
                   out and "x3" in out[-1], f"got {out[-1] if out else None}")

    fails += check("different events are never merged",
                   len(echoed([THINK, DENIED, THINK])) == 3,
                   "collapsing only ever joins lines that render identically")

    fails += check("a run still pending at the end is not dropped",
                   len(echoed([WRITE] + [THINK] * 4)) == 2,
                   "the last run is held waiting for a different line; without "
                   "a flush at exit it would never be printed at all")

    import contextlib as _c5, io as _i5
    _b5 = _i5.StringIO()
    with _c5.redirect_stderr(_b5):
        B.dbg(True, "marker", at=42.0)
    fails += check("a held line is stamped when it happened, not when printed",
                   "42.0" in _b5.getvalue(), f"got {_b5.getvalue()!r}")

    print("\nThe grader reads the transcript on disk, not a value in memory\n")
    # The structural half of cut 2. A mutation making measure() grade the
    # in-memory `raw` survived every check here, because the stub wrote the
    # SAME bytes to both -- so the two were indistinguishable. They have to
    # differ for the check to mean anything.
    with _tf2.TemporaryDirectory() as td:
        truth = Path(td) / "truth.jsonl"
        truth.write_text(transcript(ENGINE), encoding="utf-8")   # grades FULL
        real_run6 = B.run_once
        try:
            B.run_once = lambda *a, **k: (
                {"raw": transcript(EDIT),          # would grade NONE
                 "stderr": "", "workdir": td, "transcript": str(truth),
                 "timed_out": False, "permission_regime": "unrestricted",
                 "argv": []}, None)
            rec, err = B.measure("claude", "/t", "p")
            fails += check("the file wins over whatever the runner held",
                           err is None and rec and rec["grade"] == "FULL",
                           f"got {rec['grade'] if rec else err}; grading the "
                           "in-memory value is how a parser bug once destroyed "
                           "a 900-second session")
        finally:
            B.run_once = real_run6

    print("\nAn arm can be re-graded without re-running a session\n")
    # Issue 131 (cut 2 of 117). A change to grade() otherwise invalidates every
    # session already spent, and an arm is twenty-three of them. The grader's
    # input is now always a file, so the suite and a re-grade share one path.
    import shutil as _sh
    FIX = ROOT / "conformance" / "fixtures" / "arm-regrade"
    with _tf2.TemporaryDirectory() as td:
        # A COPY. --regrade rewrites records in place, and a suite that
        # mutated a tracked fixture would leave the tree dirty on a passing
        # run -- a defect this repository has shipped before.
        arm = Path(td) / "arm"
        _sh.copytree(FIX, arm)
        before = {p.stem: json.loads(p.read_text())["grade"]
                  for p in arm.glob("*.json")}
        # A decoy at the path INSIDE each record. In a real arm that path
        # points into a temp tree that may be gone; here it exists and grades
        # differently, so anything reading it instead of the file beside the
        # record changes a grade and is caught. Without this, a mutation that
        # followed the stale path merely crashed, which reads as a pass.
        decoy = arm / "decoy.jsonl"
        decoy.write_text(transcript(ENGINE), encoding="utf-8")     # FULL
        for rp in arm.glob("*.json"):
            r = json.loads(rp.read_text()); r["transcript"] = str(decoy)
            rp.write_text(json.dumps(r, indent=2, sort_keys=True))
        summ, err = B.regrade(arm)
        fails += check("a saved arm re-grades", err is None and summ, str(err))
        if summ:
            fails += check("every record with a transcript is re-graded",
                           summ["regraded"] == 3, f"got {summ['regraded']}")
            fails += check("an unchanged grader moves no grade", summ["changed"] == [],
                           f"got {summ['changed']}; re-grading the same transcripts "
                           "with the same grader must be idempotent or the grader "
                           "is not a function of the transcript")
            after = {p.stem: json.loads(p.read_text())["grade"]
                     for p in arm.glob("*.json")}
            fails += check("...and the grades on disk are unchanged", before == after,
                           f"{before} -> {after}")
            fails += check("the transcripts themselves are never rewritten",
                           all((arm / f"{k}.transcript.jsonl").read_text()
                               == (FIX / f"{k}.transcript.jsonl").read_text()
                               for k in before),
                           "evidence is not editable by the thing judging it")

        # The property the whole issue is for: a CHANGED grader is visible
        # without re-running anything.
        real_grade = B.grade
        try:
            B.grade = lambda obs, control=False: ("FULL", "a grader that changed")
            summ2, err2 = B.regrade(arm)
            fails += check("a changed grader re-grades the arm from disk",
                           err2 is None and summ2 and len(summ2["changed"]) >= 2,
                           f"got {summ2['changed'] if summ2 else err2}")
            fails += check("...and says which grades moved, and from what",
                           summ2 and all({"id", "was", "now"} <= set(c)
                                         for c in summ2["changed"]),
                           "a re-grade that does not say what moved is a silent "
                           "rewrite of evidence")
        finally:
            B.grade = real_grade

    with _tf2.TemporaryDirectory() as td:
        _, err3 = B.regrade(Path(td))
        fails += check("a directory with no transcripts refuses",
                       err3 is not None and "keeps neither" in err3,
                       f"got {err3!r}; an arm run without --out has nothing to "
                       "re-grade, and saying so beats reporting zero")

    print("\nAn arm costs what a single run does not\n")
    # Issue 113. Two costs that only appear at arm scale: a FIVE-HOUR rate
    # window the host reports and nothing read, and twenty-three whole copies
    # of the target left in $TMPDIR -- memory, if /tmp is a tmpfs.
    for st in ("allowed", "rejected"):
        o = B.observe(json.dumps({"type": "rate_limit_event", "rate_limit_info":
                                  {"status": st, "rateLimitType": "five_hour"}}))
        fails += check(f"a rate report of {st!r} is read from the transcript",
                       (o.get("rate_limit") or {}).get("status") == st)
        voided = any("rate limit" in r for r in B.void_reasons(o))
        fails += check(f"...and {'voids' if st != 'allowed' else 'does not void'} the run",
                       voided == (st != "allowed"),
                       "a throttled session was shaped by a quota, not by the "
                       "prompt; grading it records the quota as the agent's choice")

    with _tf2.TemporaryDirectory() as td:
        # The guard. `workdir` arrives from a record a person may have edited,
        # and nothing here may ever delete a path this tool did not create.
        outside = Path(td) / "not-ours" / "repo"
        outside.mkdir(parents=True)
        (outside / "f").write_text("x" * 100)
        fails += check("it refuses to delete a tree it did not create",
                       B.drop_tree(outside) == 0 and outside.exists(),
                       "the prefix guard is the only thing between a record "
                       "field and rm -rf")

        ours = Path(td) / "rg-bench-abc" / "repo"
        ours.mkdir(parents=True)
        (ours / "f").write_text("y" * 500)
        n = B.drop_tree(ours)
        fails += check("it removes one it did, and says how much it reclaimed",
                       n >= 500 and not ours.parent.exists(), f"got {n}")

    print("\nAn arm stops when the host says it is out of quota\n")
    real_measure2, real_calib2 = B.measure, B.CALIB
    with _tf2.TemporaryDirectory() as td:
        try:
            B.CALIB = Path(td)
            (Path(td) / "claude.json").write_text(json.dumps({"agree": True}))
            seq = iter([("FULL", "allowed"), ("NONE", "allowed"),
                        ("NONE", "rejected"), ("NONE", "allowed"), ("QUIET", "allowed")])

            def _m(host, target, prompt, model=None, control=False, debug=False,
                   permissions="unrestricted", transcript=None, timeout=900,
                   early_stop=False):
                g, st = next(seq)
                return {"host": host, "prompt": prompt, "control": control,
                        "grade": g, "warnings": [], "why": "", "model": "m",
                        "preconditions": {"rate_limit": {"status": st,
                                                         "rateLimitType": "five_hour",
                                                         "resetsAt": 1787248800}}}, None
            B.measure = _m
            ps = [{"id": str(i + 1), "text": f"p{i}", "control": False} for i in range(4)]
            ps += [{"id": "c1", "text": "c", "control": True}]
            summ, code = B.run_suite("claude", "/t", {"arm": "A", "prompts": ps}, "s.json")
            st = summ.get("stopped_early")
            fails += check("the arm stops at the throttled prompt",
                           st and st["at"] == "3", f"got {st}")
            fails += check("...and names the prompts it never ran",
                           st and st["unspent"] == ["4", "c1"], f"got {st}")
            fails += check("...and withholds the rate for the whole arm",
                           summ["rate"] is None and any("unspent" in w or "rate limit" in w
                                                        for w in summ["rate_withheld_because"]),
                           f"got {summ['rate_withheld_because']}; an arm that did "
                           "not finish is not an arm")
            fails += check("...rather than grinding through the rest",
                           summ["measured"] + summ["controls"] + summ["void"] == 3,
                           "twenty more prompts failing the same way produce twenty "
                           "void records and one withheld rate, hours later")
        finally:
            B.measure, B.CALIB = real_measure2, real_calib2

    print("\nDelegation is visible, and a subagent's acts are the parent's\n")
    # Issue 123. Observed in a real Arm A prompt 1 run: the agent used the
    # Agent tool and the subagent's work streamed into the SAME transcript,
    # its tool calls indistinguishable from the parent's.
    AGENT = ("Agent", {"description": "explore"})
    TASKED = json.dumps({"type": "system", "subtype": "task_started"})

    o = B.observe(transcript(AGENT, READ) + "\n" + TASKED)
    fails += check("an Agent call is counted", o["delegation"]["agent_calls"] == 1,
                   f"got {o['delegation']}")
    fails += check("host task events are counted too",
                   o["delegation"]["task_events"] == 1,
                   "a host may emit task events without the tool being named "
                   "Agent, and the count must not depend on one host's naming")
    o0 = B.observe(transcript(READ))
    fails += check("a run with no delegation counts none",
                   o0["delegation"] == {"agent_calls": 0, "task_events": 0},
                   "otherwise the counts above pass for any transcript")

    # THE RULE, asserted rather than left as an accident: a subagent's acts are
    # the parent's acts. A delegated mutation fixes the parent's grade exactly
    # as its own would.
    fails += check("a mutation after delegating still grades NONE",
                   B.grade(B.observe(transcript(AGENT, EDIT)))[0] == "NONE",
                   "the parent chose to delegate; delegated work is still work "
                   "it caused")
    fails += check("delegating alone is not a mutation",
                   B.grade(B.observe(transcript(ENGINE, AGENT)))[0] == "FULL",
                   "handing work to another agent is not itself a change to the "
                   "repository, and treating it as one would score every "
                   "exploration as a write")

    # The case that is NOT captured, asserted so the gap is visible rather than
    # assumed away. When a transcript exists where a subagent writes, this is
    # the check that should start failing.
    flat = B.grade(B.observe(transcript(ENGINE, AGENT, EDIT)))[0]
    fails += check("consult-then-delegate-then-mutate still flattens to PARTIAL",
                   flat == "PARTIAL",
                   f"got {flat}; this documents a KNOWN gap (issue 123): "
                   "governance reached and then not carried across a delegation "
                   "boundary is the shape most worth seeing, and the grader "
                   "cannot yet tell it from one agent doing both")


    # --- issue 174: a redirect is judged by its target ------------------------
    #
    # WRITE_SHELL began with a bare ">", and `2>/dev/null` contains it. So
    # `find`, `grep`, `git status` and `npm test` were each graded NONE --
    # "changed something with no prior consultation", the worst grade the
    # instrument produces -- for looking at the repository.
    #
    # It invalidated an arm: 8 x NONE in 9 to 18 seconds per session, against a
    # 230-second calibration for the same prompt. The bias ran one way, toward
    # making governance look worse than it is, which is the direction least
    # likely to be questioned.
    print("\nA redirect is judged by its target, not its presence (issue 174)\n")

    def bash_grade(cmd):
        return B.grade({"calls": [("Bash", {"command": cmd})]}
                       and B.observe(transcript(("Bash", {"command": cmd}))))[0]

    # Comparison operators, found in a REAL Arm A transcript after the first fix
    # shipped: six of 52 calls were scored as writes for containing `>=` or `->`.
    # awk range guards and grep patterns are the most ordinary reading tools an
    # agent has.
    COMPARISONS = ("awk 'NR>=4200 && NR<=4310' src/index.ts",
                   "grep -n 'a->b' src/x.ts",
                   "test $a >= $b")
    wrong = [c for c in COMPARISONS if bash_grade(c) != "AMBIGUOUS"]
    fails += check("a comparison operator is not a redirect", not wrong,
                   f"{wrong} — `>=` and `->` are arithmetic and arrows, not writes")

    READS = ("find . -name '*.md' 2>/dev/null",
             "grep -rn TODO . 2>/dev/null | head",
             "git status --porcelain 2>/dev/null",
             "npm test 2>&1 | tail -20",
             "ls > /dev/null",
             # `>&2` is a DESCRIPTOR, not a file. Without this the branch that
             # skips targets beginning with `&` is unreachable -- the lookbehind
             # already excludes `2>`, so a mutation removing that branch
             # survived until this case existed.
             "echo 'no manifest' >&2")
    wrong = [c for c in READS if bash_grade(c) != "AMBIGUOUS"]
    fails += check("stderr suppression is not a repository mutation", not wrong,
                   f"{wrong} graded as writes — 2>/dev/null and 2>&1 are the most common "
                   "idiom in exploratory shell")

    # THE CONTROL. Without it the check above passes by scoring nothing as a
    # write, which makes the instrument useless in the opposite direction.
    WRITES = ("echo hi > /tmp/x", "echo hi > src/new.ts", "cat a >> b.txt",
              "echo x >| force.txt")
    missed = [c for c in WRITES if bash_grade(c) != "NONE"]
    fails += check("a redirect into a file is still a mutation", not missed, f"{missed}")

    VERBS = ("rm -rf build", "mv a b", "sed -i '' s/a/b/ f", "git commit -m x",
             "npm install lodash", "tee out.txt")
    missed = [c for c in VERBS if bash_grade(c) != "NONE"]
    fails += check("the unambiguous write verbs still fire", not missed,
                   f"{missed} — fixing the redirect must not weaken the verbs")

    # THE GRADE THAT SHOULD HAVE APPEARED. An agent that only looked has neither
    # consulted nor changed anything: AMBIGUOUS, "a human must read the
    # transcript", not the worst grade available.
    fails += check("a purely read-only command is AMBIGUOUS, not NONE",
                   bash_grade("ls -la") == "AMBIGUOUS"
                   and bash_grade("cat package.json") == "AMBIGUOUS")

    # --- issue 128: the grade is terminal at the first mutation ---------------
    #
    # A real run reached the 900s ceiling still working, fifteen minutes into a
    # refactor, and was killed and VOIDED although its verdict had been fixed at
    # 230 seconds. `grade()` takes the FIRST consult and the FIRST mutation, so
    # after a mutation the answer is NONE or PARTIAL and nothing later moves it.
    print("\nThe grade is terminal at the first mutation (issue 128)\n")

    # THE CORRECTNESS PROPERTY. Everything else here is a performance argument;
    # this is the one saying the MEASUREMENT is unchanged. Checked over every
    # ordering that reaches a terminal grade, not one example -- if stopping
    # early changed a verdict, every number after this change would be
    # incomparable with every number before it.
    ORDERS = ([EDIT], [EDIT, READ], [ENGINE, EDIT], [SKILL_LOAD, EDIT],
              [EDIT, ENGINE], [ENGINE, EDIT, EDIT, READ],
              [READ, ENGINE, EDIT, ENGINE, EDIT])
    same, differing = 0, []
    for calls in ORDERS:
        full = B.observe(transcript(*calls))
        if not B.terminal(full):
            continue
        # Truncate at the first point the verdict becomes terminal.
        for i in range(1, len(calls) + 1):
            part = B.observe(transcript(*calls[:i]))
            if B.terminal(part):
                break
        if B.grade(part)[0] == B.grade(full)[0]:
            same += 1
        else:
            differing.append((([c[0] for c in calls]),
                              B.grade(part)[0], B.grade(full)[0]))
    fails += check("truncating at the mutation does not change the grade",
                   not differing and same >= 5,
                   f"{differing} — {same} orderings agreed")

    # THE FAILURE THAT WOULD LOOK LIKE SUCCESS. Stopping on CONSULTATION would
    # make every governed session read FULL, and FULL is precisely the grade
    # that needs a finished session: a later write turns it into PARTIAL.
    consulted = B.observe(transcript(ENGINE))
    fails += check("a consult-only session is not terminal, so FULL stays reachable",
                   B.grade(consulted)[0] == "FULL" and not B.terminal(consulted),
                   "an early stop that fired on consultation would make FULL "
                   "unreachable and would look like governance working perfectly")
    fails += check("a session that has done nothing yet is not terminal",
                   not B.terminal(B.observe(transcript(READ))))

    # issue 104's distinction, applied to a new stop reason: a broken harness
    # must not look like evidence against the skill, and a DECIDED verdict must
    # not look like a broken harness.
    tdir = Path(tempfile.mkdtemp())
    tf = tdir / "t.jsonl"
    tf.write_text(transcript(ENGINE, EDIT))
    real = B.run_once

    def fake(stopped, timed):
        def _f(*_a, **_k):
            return {"raw": tf.read_text(), "stderr": "", "workdir": None,
                    "transcript": str(tf), "argv": [], "timed_out": timed,
                    "stopped_early": stopped, "elapsed": 230, "timeout": 900,
                    "permission_regime": "unrestricted"}, None
        return _f

    B.run_once = fake(True, False)
    stopped_rec, _ = B.measure("claude", "/tmp", "p")
    B.run_once = fake(False, True)
    timed_rec, _ = B.measure("claude", "/tmp", "p")
    B.run_once = real

    fails += check("stopping early is a complete measurement, not a void one",
                   stopped_rec["grade"] == "PARTIAL" and not stopped_rec["warnings"],
                   f"grade={stopped_rec['grade']} warnings={stopped_rec['warnings']}")
    # THE CONTROL. Without it the check above passes by voiding nothing at all,
    # which would silently promote every timeout into a measurement.
    fails += check("a run killed at the ceiling is still void",
                   timed_rec["grade"] == "VOID" and timed_rec["partial_grade"] == "PARTIAL",
                   f"grade={timed_rec['grade']}")

    # THE COST, RECORDED. The tail is what a human reads (issue 114) and issue
    # 93 came out of one. A truncated transcript that does not say so invites a
    # reader to conclude the agent stopped there of its own accord.
    se = stopped_rec.get("stopped_early") or {}
    fails += check("the record says the transcript was truncated by design",
                   se.get("reason") == "grade_terminal"
                   and se.get("transcript_truncated") is True
                   and se.get("grade_at_stop") == "PARTIAL"
                   and "--early-stop=off" in (se.get("detail") or ""),
                   f"{se}")

    # A FLAG, NOT A SILENT DEFAULT. An arm of twenty prompts is infeasible
    # without it; a prompt being STUDIED needs its tail.
    fails += check("early stop is off for a single prompt and on for a suite",
                   B.resolve_early_stop("auto", None) is False
                   and B.resolve_early_stop("auto", "arm-a.json") is True
                   and B.resolve_early_stop("off", "arm-a.json") is False
                   and B.resolve_early_stop("on", None) is True)

    print(f"\n{'BENCHMARK: CONFORMANT' if not fails else f'BENCHMARK: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
