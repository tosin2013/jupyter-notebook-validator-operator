#!/usr/bin/env python3
"""ADR-008 Layer 2 — cross-provider equivalence. THE THESIS TEST (issue #1).

A scenario is defined once in provider-neutral terms, then instantiated in
each roadmap provider. If semantically equivalent state does not produce
equivalent typed facts, the provider abstraction has failed and the stop
condition in reference/criteria.md §55 fires.

Divergence here is DATA, not a broken build. Report it as a finding.

Usage:  python3 conformance/layer2.py
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import _preflight  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("layer2")

PROVIDERS = {
    "file-roadmap": {
        "adapter": "adapters/file-roadmap",
        "env": {"REPO_GOVERNOR_ROADMAP": "conformance/fixtures/roadmap.json"},
    },
    "github-projects": {
        "adapter": "adapters/github-projects",
        "env": {"REPO_GOVERNOR_GH_FIXTURE": "conformance/fixtures/github-projects-scenarios.json",
                "REPO_GOVERNOR_GH_ADMISSION": "project_status"},
    },
    "dolt-decisions": {
        "adapter": "adapters/decision-history-dolt",
        "env": {"REPO_GOVERNOR_DECISIONS_DB": "conformance/fixtures/decisions-db"},
        "role": "decision_history",
    },
    "file-decisions": {
        "adapter": "adapters/decision-history-file",
        "env": {"REPO_GOVERNOR_DECISIONS_DIR": "conformance/fixtures/decisions-file"},
        "role": "decision_history",
    },
    "github-decisions": {
        "adapter": "adapters/decision-history-github",
        "env": {"REPO_GOVERNOR_GH_DECISIONS_FIXTURE": "conformance/fixtures/decision-history-github.json"},
        "role": "decision_history",
    },
    "linear": {
        "adapter": "adapters/linear",
        "env": {"REPO_GOVERNOR_LINEAR_FIXTURE": "conformance/fixtures/linear.json"},
    },
}

# Equivalence is asserted over DISPOSITION-RELEVANT facts only. A provider's
# `reason` code is diagnostic and SHOULD differ — a more specific reason is a
# better reason. Comparing whole payloads would punish that.
EQUIVALENCE_KEYS = ("authority", "admitted", "disposition", "state",
                    "__unknown__", "blocking", "__error__")

# Each scenario: what it means, what the engine must be able to conclude,
# and how to express it in each provider. `needs` names the capability the
# scenario requires, so an honestly-advertised gap is not scored as divergence.
# The four ADR dialects are separate "providers" for equivalence purposes: the
# same decision written four genuinely different ways must yield the same typed
# facts. This is a real portability test rather than a self-consistency one,
# because the four documents share no syntax -- only meaning.
for _d in ("heading", "inline", "bullet", "yaml"):
    PROVIDERS[f"adr-{_d}"] = {"adapter": "adapters/adr", "role": "architecture",
                              "env": {"REPO_GOVERNOR_ADR_DIR": f"conformance/fixtures/adrs/{_d}"}}

SCENARIOS = [
    {
        "id": "authority_withdrawn",
        "meaning": "Work was admitted, then cancelled. §38: roadmap admission governs.",
        "function": "get_authority",
        "expect": {"authority": "CANCELLED", "admitted": False},
        "in": {"file-roadmap": "CANCELLED-1", "github-projects": "101", "linear": "ENG-104"},
    },
    {
        "id": "authorized_executing",
        "meaning": "Admitted and cleared to execute.",
        "function": "get_authority",
        "expect": {"authority": "AUTHORIZED", "admitted": True},
        "in": {"file-roadmap": "AUTHORIZED-1", "github-projects": "102", "linear": "ENG-103"},
    },
    {
        "id": "admitted_not_authorized",
        "meaning": "On the roadmap but not cleared to execute. INV-002.",
        "function": "get_authority",
        "expect": {"authority": "ADMITTED", "admitted": True},
        "in": {"file-roadmap": "ADMITTED-1", "github-projects": "1", "linear": "ENG-101"},
    },
    {
        "id": "not_governable",
        "meaning": "No admission state readable. Must be UNKNOWN+blocking, never EXECUTE.",
        "function": "get_authority",
        "expect": {"__unknown__": True, "blocking": True},
        "in": {"file-roadmap": "NOAUTH-1", "github-projects": "103", "linear": "ENG-100"},
    },
    {
        "id": "absent_item",
        "meaning": "The work item does not exist. NOT_FOUND, not UNKNOWN.",
        "function": "get_work",
        "expect": {"__error__": "NOT_FOUND"},
        "in": {"file-roadmap": "NO-SUCH", "github-projects": "9999", "linear": "ENG-999"},
    },
    {
        "id": "thin_envelope",
        "meaning": "No non-goals declared. Non-blocking unknown (issue #2).",
        "function": "get_non_goals",
        "expect": {"__unknown__": True, "blocking": False},
        "in": {"file-roadmap": "THIN-1", "github-projects": "1", "linear": "ENG-101"},
    },
    {
        "id": "completed_work_authority",
        "meaning": ("Work finished. Authority stays AUTHORIZED — completion is a separate axis. "
                    "STOP_COMPLETE is composed by the engine from authority + acceptance (§40)."),
        "function": "get_authority",
        "expect": {"authority": "AUTHORIZED", "admitted": True},
        "in": {"file-roadmap": "DONE-1", "github-projects": "3", "linear": "ENG-105"},
    },
    {
        "id": "completion_verifiable",
        "meaning": "Can the provider supply machine-checkable acceptance? Required for STOP_COMPLETE.",
        "function": "get_acceptance_conditions",
        "needs": "acceptance_conditions",
        "expect": {},
        "in": {"file-roadmap": "DONE-1", "github-projects": "3", "linear": "ENG-105"},
    },
    {
        "id": "not_admitted",
        "meaning": ("Filed but never admitted. INV-002 needs this distinct from admitted-not-authorized. "
                    "Linear expresses it as triage; most trackers cannot."),
        "function": "get_authority",
        "needs": "admission_distinction",
        "expect": {"__unknown__": True, "blocking": True},
        "in": {"github-projects": "103", "linear": "ENG-100"},
    },
    {
        "id": "adr_dialect_equivalence",
        "meaning": ("The same decision, written in four status dialects that share no syntax. "
                    "16% of 439 real ADRs were readable before this; a dialect the parser misses "
                    "is a decision the engine cannot see (#28)."),
        "function": "get_constraints",
        "role": "architecture",
        "expect": {},
        "in": {"adr-heading": "x", "adr-inline": "x", "adr-bullet": "x", "adr-yaml": "x"},
    },
    {
        "id": "work_declined",
        "meaning": ("Work was considered and declined. INV-005: a recorded decision must survive "
                    "rediscovery. Two stores of genuinely different shape, one contract."),
        "function": "get_disposition",
        "role": "decision_history",
        "expect": {"disposition": "REJECTED"},
        "in": {"dolt-decisions": "DECLINED-1", "github-decisions": "901",
               "file-decisions": "DECLINED-1"},
    },
    {
        "id": "no_decision_recorded",
        "meaning": "Nothing was ever decided. Absence is not permission, and does not block.",
        "function": "get_disposition",
        "role": "decision_history",
        "expect": {"__unknown__": True, "blocking": False},
        "in": {"dolt-decisions": "NEVER-DECIDED", "github-decisions": "904",
               "file-decisions": "NEVER-DECIDED"},
    },
]


def query(pname, function, wid, role="roadmap_authority"):
    p = PROVIDERS[pname]
    role = p.get("role", role)
    env = dict(os.environ)
    env.update(p["env"])
    r = subprocess.run(
        [sys.executable, str(ROOT / p["adapter"]), "query", role, function, f"id={wid}"],
        capture_output=True, text=True, cwd=ROOT, env=env, timeout=30,
    )
    try:
        return json.loads(r.stdout)
    except json.JSONDecodeError:
        return {"ok": False, "error": {"type": "NON_JSON", "message": r.stdout[:80]}}


def advertises(pname, capability):
    p = PROVIDERS[pname]
    env = dict(os.environ)
    env.update(p["env"])
    r = subprocess.run([sys.executable, str(ROOT / p["adapter"]), "describe"],
                       capture_output=True, text=True, cwd=ROOT, env=env, timeout=30)
    try:
        return json.loads(r.stdout).get("capabilities", {}).get(capability, False) is True
    except json.JSONDecodeError:
        return False


def projection(obs):
    """Disposition-relevant facts only.

    `constraints` is compared by COUNT, not by value: each provider cites its own
    file paths, so raw comparison would always differ for reasons that carry no
    meaning. Comparing only `state` proved too coarse -- one decision flipping to
    Superseded left another Accepted, so the state stayed DEFINED and a genuine
    divergence passed.
    """
    out = {k: obs[k] for k in EQUIVALENCE_KEYS if k in obs}
    if isinstance(obs.get("constraints"), list):
        out["n_constraints"] = len(obs["constraints"])
    return out


def observe(resp):
    """Reduce a response to the typed facts an engine would act on."""
    if not resp.get("ok"):
        return {"__error__": resp.get("error", {}).get("type")}
    if resp.get("unknown"):
        return {"__unknown__": True, "blocking": resp["unknown"]["blocking"],
                "reason": resp["unknown"]["reason"]}
    return resp.get("value") or {}


def matches(obs, expect):
    if "__not__" in expect:
        return not all(obs.get(k) == v for k, v in expect["__not__"].items())
    return all(obs.get(k) == v for k, v in expect.items() if k != "reason")


def main():
    absent = _preflight.banner()

    # The live tool is the other half of this question. Its first versions
    # scored one-sided rows as AGREE and compared unknown *reasons*, undoing
    # the projection lesson this file exists to enforce. `--self-test` is the
    # gate that those cannot silently return; it needs no network. Run it
    # first, and flush, so a failure is not buried under buffered scenario
    # output.
    sys.stdout.flush()
    live = subprocess.run(
        [sys.executable, str(ROOT / "tools" / "live-equivalence.py"), "--self-test"],
        cwd=ROOT,
    )
    if live.returncode != 0:
        return 1

    # The readiness tool answers the PREREQUISITE to the live run -- which of
    # these five states each provider can express, and which it actually holds.
    # It imports SCENARIOS from live-equivalence rather than restating them, so
    # its self-test is also the check that the two have not drifted apart. Same
    # gate, same reason: it needs no network, and a tool that silently stopped
    # distinguishing "no such status" from "status with nothing in it" would
    # report a fix nobody can act on.
    sys.stdout.flush()
    ready = subprocess.run(
        [sys.executable, str(ROOT / "tools" / "provider-readiness.py"), "--self-test"],
        cwd=ROOT,
    )
    if ready.returncode != 0:
        return 1

    covered = diverged = mismatched = 0
    findings = []
    print("Layer 2 — cross-provider equivalence\n")

    for sc in SCENARIOS:
        present = {p: w for p, w in sc["in"].items() if w}
        print(f"[{sc['id']}]  {sc['meaning']}")
        if len(present) < 2:
            only = ", ".join(present) or "none"
            print(f"    SKIP  expressible in only: {only} — not an equivalence test\n")
            findings.append((sc["id"], "UNTESTABLE", f"only expressible in {only}"))
            continue

        covered += 1
        obs = {p: observe(query(p, sc["function"], w, sc.get("role", "roadmap_authority")))
               for p, w in present.items()}

        # An advertised capability gap is not a normalization failure (ADR-003).
        need = sc.get("needs")
        if need:
            lacking = [p for p in present if not advertises(p, need)]
            if lacking:
                for p, o in obs.items():
                    print(f"    {p:<16} {json.dumps(o, sort_keys=True)}")
                print(f"    CAPABILITY GAP  {lacking} advertise '{need}' = false, honestly\n")
                findings.append((sc["id"], "CAPABILITY_GAP",
                                 f"{lacking} cannot supply '{need}'; advertised false"))
                continue

        proj = {p: projection(o) for p, o in obs.items()}
        vals = list(proj.values())
        agree = all(v == vals[0] for v in vals)
        correct = all(matches(v, sc["expect"]) for v in vals)

        for p, o in obs.items():
            extra = f"   (reason: {o['reason']})" if "reason" in o else ""
            print(f"    {p:<16} {json.dumps(projection(o), sort_keys=True)}{extra}")
        # Scored here rather than by the print wrapper: this suite's verdicts
        # are AGREE / DIVERGENCE / WRONG, and its only [PASS] lines come from a
        # subprocess whose stdout never passes through the wrapper (issue 67).
        # Scenarios that hit the CAPABILITY GAP branch above `continue` before
        # reaching here and are deliberately NOT counted: an advertised gap is
        # not an equivalence assertion. That is why executed trails `tested` by
        # exactly the number of gaps reported, and it reconciles rather than
        # drifts.
        _CNT.record(bool(agree and correct))
        if agree and correct:
            print("    AGREE + CORRECT\n")
        elif not agree:
            diverged += 1
            print(f"    ** DIVERGENCE ** providers disagree on equivalent state\n")
            findings.append((sc["id"], "DIVERGENCE", json.dumps(proj, sort_keys=True)))
        else:
            mismatched += 1
            bad = [p for p, o in proj.items() if not matches(o, sc["expect"])]
            print(f"    ** WRONG ** agree with each other but violate expectation: {bad}")
            print(f"       expected {json.dumps(sc['expect'], sort_keys=True)}\n")
            findings.append((sc["id"], "WRONG", f"{bad} vs expected {sc['expect']}"))

    # ADR-019 rule 3: a backend that cannot supply history natively "must
    # implement chaining itself and SAY SO in its properties". That field is
    # the whole reason a file backend may stand beside Dolt. Assert the
    # difference is DECLARED -- not that the two adapters differ, which is
    # obvious, but that a consumer reading `describe` can tell which guarantee
    # it is getting. Equivalence of ANSWERS must not imply equivalence of
    # GUARANTEES, and nothing else in this suite would catch that.
    print("\nA weaker chain declares itself weaker (ADR-019 rule 3)\n")
    chains = {}
    for _n in ("dolt-decisions", "file-decisions"):
        _p = PROVIDERS[_n]
        _env = dict(os.environ); _env.update(_p["env"])
        _r = subprocess.run([sys.executable, str(ROOT / _p["adapter"]), "describe"],
                            capture_output=True, text=True, cwd=ROOT, env=_env, timeout=30)
        try:
            chains[_n] = json.loads(_r.stdout)["properties"].get("chain_supplied_by_store")
        except Exception:
            chains[_n] = "UNREADABLE"
    chain_ok = chains.get("dolt-decisions") is True and chains.get("file-decisions") is False
    print(f"  [{'PASS' if chain_ok else 'FAIL'}] chain_supplied_by_store distinguishes them: {chains}")
    if not chain_ok:
        mismatched += 1
        findings.append(("chain_guarantee", "WRONG",
                         "the two decision_history backends do not declare different chain "
                         "guarantees; a consumer cannot tell a store-supplied chain from a "
                         "hand-rolled one, which is exactly what ADR-019 rule 3 requires"))

    total = len(SCENARIOS)
    print("-" * 62)
    print(f"scenarios: {total}   tested: {covered}   untestable: {total - covered}")
    print(f"divergences: {diverged}   wrong-but-agreeing: {mismatched}")
    if findings:
        print("\nFINDINGS (input to criteria.md §55):")
        for sid, kind, detail in findings:
            print(f"  [{kind}] {sid}: {detail[:150]}")
    verdict = "EQUIVALENT" if not diverged and not mismatched else "NOT EQUIVALENT"
    print(f"\nLAYER 2: {verdict} across {covered} tested scenario(s)")
    if verdict != "EQUIVALENT":
        _preflight.attribute(absent)
    return 0 if verdict == "EQUIVALENT" else 1


if __name__ == "__main__":
    sys.exit(main())
