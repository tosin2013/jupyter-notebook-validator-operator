#!/usr/bin/env python3
"""ScopeEnvelope compilation and the completion firewall (§31, §32, §40, ADR-024).

§40 says its worked example "should be implemented verbatim as an acceptance
test". This is that test, plus the classification paths around it.

The property that matters is not that discoveries are captured -- it is that
**nothing converts them to execution once the authorization is satisfied**,
including a discovery that is correct, small, obviously beneficial, and
genuinely necessary. A firewall with one exception is not a firewall.

Usage:  python3 conformance/envelope.py
"""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "engine"))
import envelope as E  # noqa: E402
import vocabulary as V  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("envelope")

# A rich envelope. Real trackers rarely supply this much -- which is issue #2,
# measured by `thinness` rather than assumed away.
RICH = {
    "authority_id": "SIM-1",
    "authority": "AUTHORIZED",
    "required_outcome": "Persistent state recovery",
    "in_scope": ["state/recovery.py", "state/snapshot.py"],
    "non_goals": ["Admin UI", "multi-team support"],
    "architecture_constraints": ["ADR-002"],
    "acceptance_conditions": [{"check": "tests_pass", "target": "state/"}],
}

# §40 verbatim: authorized work completed, three things discovered along the way.
SECTION_40 = [
    ("Admin UI", "POSSIBLE_FEATURE", "CAPTURE_ONLY"),
    ("Multi-team support", "POSSIBLE_FEATURE", "CAPTURE_ONLY"),
    # §40 permits CAPTURE_ONLY *or* MAINTENANCE_REVIEW for the refactor.
    ("Refactor", "TECHNICAL_DEBT", ("CAPTURE_ONLY", "MAINTENANCE_REVIEW")),
]


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"    {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def main():
    fails = 0
    print("§40 worked example, verbatim\n")

    for target, dtype, expect in SECTION_40:
        got = E.classify(RICH, dtype, target, completed=True)["disposition"]
        want = expect if isinstance(expect, tuple) else (expect,)
        fails += check(f"{target:<20} -> {got}", got in want, f"expected one of {want}")

    print("\nThe firewall admits no exception\n")

    # The whole point. Each of these would be a reasonable-sounding door.
    for label, kw in (
        ("a discovery claimed necessary", dict(claimed_necessary=True)),
        ("a discovery inside declared in_scope",
         dict(target="state/recovery.py", claimed_necessary=True)),
        ("a bug, which always feels urgent", dict(dtype="BUG")),
    ):
        args = {"dtype": "TECHNICAL_DEBT", "target": "something", **kw}
        got = E.classify(RICH, completed=True, **args)["disposition"]
        fails += check(f"{label} still captures", got == "CAPTURE_ONLY", f"got {got}")

    print("\nBefore completion: necessity is substantiated, never asserted\n")

    r = E.classify(RICH, "TECHNICAL_DEBT", "state/recovery.py cleanup", claimed_necessary=True)
    fails += check("necessary + inside in_scope may execute", r["disposition"] == "EXECUTE",
                   f"got {r['disposition']}")

    r = E.classify(RICH, "TECHNICAL_DEBT", "unrelated/module.py", claimed_necessary=True)
    fails += check("necessary claim OUTSIDE in_scope fails closed",
                   r["disposition"] == "CAPTURE_ONLY", f"got {r['disposition']}")

    r = E.classify(RICH, "POSSIBLE_FEATURE", "Admin UI redesign", claimed_necessary=True)
    fails += check("a declared non-goal outranks any necessity claim",
                   r["disposition"] == "CAPTURE_ONLY", f"got {r['disposition']}")

    thin = {**RICH, "in_scope": [], "non_goals": []}
    r = E.classify(thin, "TECHNICAL_DEBT", "anything", claimed_necessary=True)
    fails += check("with no declared in_scope, no necessity claim can be substantiated",
                   r["disposition"] == "CAPTURE_ONLY", f"got {r['disposition']}")

    print("\nVocabulary and routing\n")

    for dtype, want in (("ARCHITECTURE_IMPLICATION", "ARCHITECTURE_REVIEW"),
                        ("MAINTENANCE_SIGNAL", "MAINTENANCE_REVIEW"),
                        ("RETIREMENT_SIGNAL", "RETIREMENT_REVIEW"),
                        ("RESEARCH_QUESTION", "CAPTURE_ONLY")):
        got = E.classify(RICH, dtype, "x")["disposition"]
        fails += check(f"{dtype} routes to {want}", got == want, f"got {got}")

    bad = E.classify(RICH, "NOT_A_TYPE", "x")
    fails += check("a discovery type outside §32 is refused, not guessed",
                   bad.get("error") == "BAD_REQUEST")

    emitted = {E.classify(RICH, d, "x")["disposition"] for d in E.DISCOVERY_TYPES}
    emitted |= {"EXECUTE"}
    undeclared = sorted(d for d in emitted if d not in V.EXECUTION + V.REVIEW)
    fails += check("every disposition it can emit is in the closed vocabulary",
                   not undeclared, str(undeclared))

    print("\n§39: a decision already made is not re-made by rediscovery\n")

    # The engine wrote decisions and never read one back, so every rediscovery
    # looked new and an agent told no in one session could be told yes in the
    # next by the same engine (issue 95). `prior` is passed as DATA, so this
    # tests the rule without a provider -- ADR-021 keeps spawning in
    # bindings.py and classify() stays deterministic.
    fails += check("identity is stable across calls",
                   E.discovery_id("W-1", "BUG", "src/x.py")
                   == E.discovery_id("W-1", "BUG", "src/x.py"))
    fails += check("and distinguishes targets",
                   E.discovery_id("W-1", "BUG", "src/x.py")
                   != E.discovery_id("W-1", "BUG", "src/y.py"))
    fails += check("the target is hashed, never carried in the id",
                   "src" not in E.discovery_id("W-1", "BUG", "src/x.py"),
                   "§51 keeps repository content out of a public evidence chain, which "
                   "is why record() redacts the target")

    deferred = {"decision_id": "d-1", "disposition": "DEFERRED",
                "reason": "considered and not authorized", "reversal_condition": None}
    r = E.classify(RICH, "BUG", "src/x.py", prior=deferred)
    fails += check("a rediscovery stays CAPTURE_ONLY", r["disposition"] == "CAPTURE_ONLY")
    fails += check("and says it was decided before",
                   any("already decided" in x for x in r["reasons"]))
    fails += check("and reports the absence of a reversal condition as a gap",
                   any("no reversal condition" in x for x in r["reasons"]),
                   "silence about how a decision could be revisited is a gap in that "
                   "decision, not permission")

    # THE ONE THAT MATTERS. Necessity is the door §39 exists to close.
    nec = E.classify(RICH, "BUG", "state/recovery.py", claimed_necessary=True, prior=deferred)
    fails += check("a necessity claim does not reopen a recorded decision",
                   nec["disposition"] == "CAPTURE_ONLY",
                   "re-claiming necessity is exactly how a rediscovery would become "
                   "executable, which is what §39 forbids")

    # Positive control: without the prior, the same claim still reaches EXECUTE.
    # Without this the check above passes for any target at all.
    ctl = E.classify(RICH, "BUG", "state/recovery.py", claimed_necessary=True)
    fails += check("control: the same claim without a prior still EXECUTEs",
                   ctl["disposition"] == "EXECUTE",
                   f"got {ctl['disposition']} — the §39 check would then prove nothing")

    fails += check("an ACCEPTED prior does not block",
                   E.classify(RICH, "BUG", "src/x.py",
                              prior={"decision_id": "d-2", "disposition": "ACCEPTED"}
                              )["disposition"] == "CAPTURE_ONLY",
                   "only DEFERRED and REJECTED are withholdings; §39 is about those")
    fails += check("absence changes nothing",
                   E.classify(RICH, "BUG", "src/x.py", prior=None)["disposition"]
                   == E.classify(RICH, "BUG", "src/x.py")["disposition"],
                   "a repository with no decision history must behave as before")

    print("\nThe rule is WIRED, not merely present\n")

    # Everything above passes `prior` in by hand, so it tests the rule and not
    # the connection. Removing the read from main() would leave all of it
    # green -- which is precisely the defect this issue exists for: the
    # adapters implemented five read functions and the engine called none.
    # A rule nothing invokes is the same as no rule.
    import json as _j, os as _o, subprocess as _sp, sys as _s, tempfile as _tf
    with _tf.TemporaryDirectory() as td:
        r = Path(td) / "repo"
        (r / ".repo-governor" / "decisions").mkdir(parents=True)
        _sp.run(["git", "init", "-q", str(r)], capture_output=True)
        (r / "roadmap.json").write_text(_j.dumps({"items": {"W-1": {
            "title": "x", "status": "IN_PROGRESS", "authority": "AUTHORIZED",
            "admitted": True, "required_outcome": "x", "in_scope": ["src/"],
            "decision_history": []}}}))
        (r / ".repo-governor.json").write_text(_j.dumps({
            "repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
            "repository": {"id": "acme/w"},
            "condition": {"assessed": "L1", "profile": "GOVERNOR_LITE"},
            "permissions": {"repository": {"read": True, "write": False},
                            "roadmap_authority": {"read": True, "write": False},
                            "decision_history": {"read": True, "write": True}},
            "providers": {
                "repository": {"type": "git", "adapter": "adapters/git",
                               "contract_version": 1},
                "roadmap_authority": {"type": "file-roadmap",
                                      "adapter": "adapters/file-roadmap",
                                      "contract_version": 1,
                                      "env": {"REPO_GOVERNOR_ROADMAP": "roadmap.json"}},
                "decision_history": [{"type": "decision-history-file",
                                      "adapter": "adapters/decision-history-file",
                                      "contract_version": 1}]}}))
        env = dict(_o.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
        eng = str(ROOT / "engine" / "envelope.py")

        def run(*a):
            p_ = _sp.run([_s.executable, eng, "W-1", *a], capture_output=True,
                         text=True, cwd=str(r), env=env, timeout=300)
            try:
                return _j.loads(p_.stdout)
            except Exception:
                return {"_raw": p_.stdout[:200] + p_.stderr[:200]}

        first = run("--discovery", "BUG:src/x.py", "--record")
        fails += check("a first discovery records", (first.get("record") or {}).get("recorded") is True,
                       _j.dumps(first)[:200])
        again = run("--discovery", "BUG:src/x.py", "--necessary")
        fails += check("main() reads the record back on rediscovery",
                       again.get("prior_decision") is not None,
                       f"got {_j.dumps(again)[:200]} — the rule is present and unwired")
        fails += check("and the rediscovery is refused despite a necessity claim",
                       again.get("disposition") == "CAPTURE_ONLY")
        fresh = run("--discovery", "BUG:src/other.py", "--necessary")
        fails += check("control: a target never decided still EXECUTEs",
                       fresh.get("disposition") == "EXECUTE",
                       f"got {fresh.get('disposition')} — the check above would prove nothing")

    print("\nThinness is measured, not assumed away (issue #2)\n")

    full = E._thinness(RICH)
    fails += check("a fully-declared envelope reads FULL", full["verdict"] == "FULL",
                   str(full))
    bare = E._thinness({k: ([] if isinstance(v, list) else None) for k, v in RICH.items()})
    fails += check("an envelope no provider could fill reads THIN", bare["verdict"] == "THIN",
                   str(bare))

    print(f"\n{'ENVELOPE: CONFORMANT' if not fails else f'ENVELOPE: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
