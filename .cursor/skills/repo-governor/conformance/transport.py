#!/usr/bin/env python3
"""Transport equivalence — issue #18 option C.

Asserts that an adapter which can both fetch for itself (option A) and accept
raw provider output on stdin (option C) produces byte-identical results either
way. If that holds, the transport can move outside the adapter without
changing what the engine sees — which is the whole claim of option C.

Usage:  python3 conformance/transport.py
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("transport")

ROOT = Path(__file__).resolve().parent.parent

SUITE = [
    {
        "adapter": "adapters/linear",
        "fetcher": "adapters/linear-fetch",
        "role": "roadmap_authority",
        "env": {"REPO_GOVERNOR_LINEAR_FIXTURE": "conformance/fixtures/linear.json"},
        "ids": ["ENG-100", "ENG-101", "ENG-102", "ENG-103", "ENG-104", "ENG-105", "ENG-106"],
        "functions": ["get_authority", "get_status", "get_work", "get_decision_history",
                      "get_non_goals", "get_parent_or_goal"],
        # Raw payloads that look plausible but are not the real thing. Each must
        # be refused, never normalized (#18 Q2).
        "probe_id": "ENG-102",
        "substitutes": [
            ('{"summary": "ENG-102 is ready to start"}', "an LLM summary"),
            ('{"data": {"other": []}}', "the wrong query"),
            ("not json at all", "non-JSON"),
            ('{"data": {"issues": {}}}', "truncated payload"),
        ],
    },
    {
        # No fetcher process: the fixture IS a captured GraphQL response, so
        # catting it is a faithful stand-in for whatever obtained it. That is
        # precisely option C's claim — the source of the bytes is irrelevant.
        "adapter": "adapters/github-projects",
        "fetcher_file": "conformance/fixtures/github-projects-scenarios.json",
        "role": "roadmap_authority",
        "env": {"REPO_GOVERNOR_GH_FIXTURE": "conformance/fixtures/github-projects-scenarios.json",
                "REPO_GOVERNOR_GH_ADMISSION": "project_status"},
        "ids": ["1", "3", "101", "102", "103"],
        "functions": ["get_authority", "get_status", "get_work", "get_non_goals",
                      "get_parent_or_goal"],
        "probe_id": "102",
        "substitutes": [
            ('{"summary": "issue 102 is authorized"}', "an LLM summary"),
            ('{"data": {"repository": {"pullRequests": {"nodes": []}}}}', "the wrong query"),
            ("not json at all", "non-JSON"),
            ('{"data": {"repository": {"issue": {"number": 7}}}}', "the wrong issue"),
        ],
    },
    {
        "adapter": "adapters/decision-history-github",
        "fetcher_file": "conformance/fixtures/decision-history-github-901.json",
        "role": "decision_history",
        "env": {"REPO_GOVERNOR_GH_DECISIONS_FIXTURE": "conformance/fixtures/decision-history-github.json"},
        "ids": ["901"],
        "functions": ["get_disposition", "get_decisions", "get_reversal_condition"],
        "probe_id": "901",
        "substitutes": [
            ('{"summary": "901 was declined"}', "an LLM summary"),
            ('{"repository": {"issue": {"number": 901}}}', "no data envelope"),
            ("not json at all", "non-JSON"),
            ('{"data": {"repository": {"issue": {"number": 902, "state": "CLOSED", '
             '"stateReason": "COMPLETED"}}}}', "the wrong issue"),
        ],
    },
]


def run(cmd, env_extra, stdin=None):
    env = dict(os.environ)
    env.update(env_extra)
    return subprocess.run(cmd, capture_output=True, text=True, cwd=ROOT,
                          env=env, input=stdin, timeout=30)


def main():
    failures = 0
    checks = 0
    for spec in SUITE:
        a_path = str(ROOT / spec["adapter"])
        env = spec["env"]
        if "fetcher" in spec:
            source = spec["fetcher"]
            raw = run([sys.executable, str(ROOT / spec["fetcher"])], env).stdout
        else:
            source = spec["fetcher_file"]
            raw = (ROOT / spec["fetcher_file"]).read_text()
        print(f"{spec['adapter']}  <->  {source}")

        before = failures
        for wid in spec["ids"]:
            for fn in spec["functions"]:
                base = [sys.executable, a_path, "query", spec["role"], fn, f"id={wid}"]
                a = run(base, env).stdout
                c = run(base + ["--input", "-"], env, stdin=raw).stdout
                checks += 1
                if a != c:
                    failures += 1
                    print(f"  [FAIL] {wid} {fn}: A and C differ")
        n = len(spec["ids"]) * len(spec["functions"])
        print(f"  [{'PASS' if failures == before else 'FAIL'}] {n} A/C comparisons byte-identical")

        # substitute payloads must be refused
        sub_ok = True
        for payload, label in spec["substitutes"]:
            base = [sys.executable, a_path, "query", spec["role"], spec["functions"][0],
                    f"id={spec['probe_id']}", "--input", "-"]
            out = run(base, env, stdin=payload).stdout
            if '"MALFORMED_SOURCE"' not in out:
                print(f"  [FAIL] accepted {label} instead of refusing it")
                sub_ok = False
                failures += 1
        if sub_ok:
            print(f"  [PASS] {len(spec['substitutes'])} substitute payloads refused as MALFORMED_SOURCE")

    # Two transports of the SAME provider, genuinely different payload shapes.
    # This is #18 Q1's real test: GitHub's list-vs-single responses share one
    # schema family, so agreeing there proved less than it looked. Linear MCP and
    # Linear GraphQL do not -- different field names, different nesting.
    print("\nadapters/linear  MCP shape  <->  GraphQL shape")
    gql = (ROOT / "conformance" / "fixtures" / "linear.json").read_text()
    mcp = (ROOT / "conformance" / "fixtures" / "linear-mcp.json").read_text()
    ids = ["ENG-100", "ENG-101", "ENG-103", "ENG-104", "ENG-105"]
    fns = ["get_authority", "get_status", "get_work", "get_non_goals", "get_decision_history"]
    a_path = str(ROOT / "adapters" / "linear")
    n = 0
    for wid in ids:
        for fn in fns:
            base = [sys.executable, a_path, "query", "roadmap_authority", fn, f"id={wid}", "--input", "-"]
            if run(base, {}, stdin=gql).stdout != run(base, {}, stdin=mcp).stdout:
                failures += 1
                print(f"  [FAIL] {wid} {fn}: the two transports disagree")
            n += 1
    print(f"  [{'PASS' if not failures else 'FAIL'}] {n} typed-fact comparisons byte-identical")

    # And the honest counterpart: where a transport genuinely CANNOT answer, it
    # must say so rather than fabricate equivalence. MCP names a parent only by
    # opaque uuid; GraphQL returns the human identifier. Verified real, not
    # hypothetical -- this is the first per-(provider x transport) capability
    # difference in the project, which #17 predicted and could not yet exhibit.
    g_par = ('{"data":{"issues":{"nodes":[{"identifier":"ENG-200","title":"c",'
             '"state":{"name":"Backlog","type":"backlog"},"project":null,'
             '"parent":{"identifier":"ENG-42"},"labels":{"nodes":[]}}]}}}')
    m_par = ('{"issues":[{"id":"ENG-200","title":"c","status":"Backlog","statusType":"backlog",'
             '"project":null,"parentId":"a3f1c8e2-7b04-4d19-9f2a-6c5e10bd77aa","labels":[]}]}')
    base = [sys.executable, a_path, "query", "roadmap_authority", "get_parent_or_goal",
            "id=ENG-200", "--input", "-"]
    g_out = json.loads(run(base, {}, stdin=g_par).stdout)
    m_out = json.loads(run(base, {}, stdin=m_par).stdout)
    ok_g = (g_out.get("value") or {}).get("parent") == "ENG-42"
    ok_m = (m_out.get("unknown") or {}).get("reason") == "PARENT_NOT_RESOLVABLE_ON_TRANSPORT"
    failures += not (ok_g and ok_m)
    print(f"  [{'PASS' if ok_g and ok_m else 'FAIL'}] a capability absent on one transport is "
          f"declared, never fabricated")

    print(f"\nTRANSPORT EQUIVALENCE: {'CONFIRMED' if not failures else f'BROKEN ({failures})'}")
    return 0 if not failures else 1


if __name__ == "__main__":
    sys.exit(main())
