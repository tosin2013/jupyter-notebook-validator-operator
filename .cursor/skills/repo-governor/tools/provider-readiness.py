#!/usr/bin/env python3
"""Can this provider express the states the equivalence bar needs? (issue 41)

`tools/live-equivalence.py` answers issue #1 -- do two live providers normalize
to the same governance verdict. It can only answer it for states BOTH providers
actually hold, and a state present in one is counted SKIP, never agreement.

So a run can be attempted, take a live payload, and report `3 of 5, skipped: 2`
-- which fails issue 41's bar by design and tells you nothing you could not have
learned first. This asks the prerequisite: WHICH OF THE FIVE STATES CAN EACH
PROVIDER EXPRESS, AND WHICH DOES IT ACTUALLY HOLD?

THE SCENARIOS ARE IMPORTED, NEVER RESTATED. `live-equivalence.SCENARIOS` is the
single source for the five semantic states, and a second copy here would be the
defect this repository has fixed in imports.py, SUPERSEDED_RE, the census, the
reference-tier counts and ADR_DIRS -- five times, and once more the day the ADR
convention list was written out by hand.

THREE ANSWERS, NOT TWO. A status a team never created and a status that exists
with nothing in it are different facts with different fixes:

    absent     no such status -- the team must create one (or enable Triage)
    declared   the status exists and holds nothing -- put one item in it
    populated  ready

Collapsing them into "missing" is the absence-vs-unknown failure ADR-003 rule 6
forbids of every adapter, and issue 200 records the engine committing it.

WHO CAN FIX IT is reported separately from WHAT IS MISSING, and is the more
useful axis. `absent` needs a workflow state created -- a team setting with no
API surface here, so a human does it. `declared` needs an issue put into a
state that already exists, which is an ordinary write. Printing both as
"missing" leaves the reader to find out which they can automate one failed
script later.

TRUNCATION IS REPORTED, NOT HIDDEN. The GitHub side reads a bounded page. A
bucket absent from that page is reported as `not in the newest N`, never as
absent: `.github/project-config.json` already records this exact trap --
"gh project item-list REQUIRES --limit; without it the command returns 30 and
exit 0, so a truncated board is indistinguishable from a complete one".

Linear state arrives on stdin as an MCP payload, so no credential is needed here
or in the adapter (ADR-020, ADR-028). Nothing about the payload is written to
disk, and only status TYPE names and counts are printed -- never a title, never
an identifier (§51).

    <mcp list_issues output> | python3 tools/provider-readiness.py --github <owner/repo>
    python3 tools/provider-readiness.py --github <owner/repo> --github-only
    python3 tools/provider-readiness.py --self-test

To distinguish `absent` from `declared`, include the team's status vocabulary:

    {"issues": [...], "statuses": [<mcp list_issue_statuses output>]}

Without it every unpopulated type reports `absent-or-declared`, which is honest
and less useful.

Exit 0 when every scenario is live in both providers, 1 when not, 2 on usage.
"""

from __future__ import annotations

import importlib.util
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

_spec = importlib.util.spec_from_file_location("_le", ROOT / "tools" / "live-equivalence.py")
_le = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_le)

SCENARIOS = _le.SCENARIOS          # the five states, imported
bucket_github = _le.bucket_github  # the same bucketing the comparison uses

GH_PAGE = 300  # deeper than pick_github's 100: this is a census, not a sample.


def github_states(nwo, limit=GH_PAGE):
    """Which buckets this repository holds, and how deep we looked."""
    p = subprocess.run(["gh", "issue", "list", "--repo", nwo, "--state", "all",
                        "--limit", str(limit), "--json",
                        "number,state,stateReason,milestone,assignees"],
                       capture_output=True, text=True, timeout=120)
    if p.returncode != 0:
        return None, 0, (p.stderr or "").strip()[:200]
    issues = json.loads(p.stdout or "[]")
    counts = {}
    for i in issues:
        b = bucket_github(i)
        if b:
            counts[b] = counts.get(b, 0) + 1
    return counts, len(issues), None


def linear_states(payload):
    """(populated counts, declared type names or None) from an MCP payload."""
    if isinstance(payload, list):
        issues, statuses = payload, None
    else:
        issues = payload.get("issues") or []
        statuses = payload.get("statuses")
    counts = {}
    for i in issues:
        t = i.get("statusType")
        if t:
            counts[t] = counts.get(t, 0) + 1
    declared = None
    if statuses:
        declared = sorted({s.get("type") for s in statuses if s.get("type")})
    return counts, declared


# Who can close each gap. ONE function, used by the report and by the self-test:
# a control that re-implements the routing tests nothing about the routing
# (issue 187, where exactly that left a broken comparator green).
FIXER = {"populated": None, "declared": "scripted",
         "absent": "human", "absent-or-declared": "unknown"}


def who_fixes(state):
    """"human" when only a team setting closes it, "scripted" when a write does."""
    return FIXER[state]


def classify(ltype, counts, declared):
    if counts.get(ltype):
        return "populated", counts[ltype]
    if declared is None:
        return "absent-or-declared", 0
    return ("declared" if ltype in declared else "absent"), 0


def report(lcounts, ldeclared, gcounts, gseen, github_only):
    runnable = 0
    print("Can the equivalence bar run here?\n")
    print(f"  {'semantic state':34} {'linear':22} github")
    for meaning, ltype, _expect, gh_state in SCENARIOS:
        gn = (gcounts or {}).get(gh_state, 0)
        gtxt = f"{gn} issue(s)" if gn else f"none in the newest {gseen}"
        if github_only:
            ltxt, ok = "-- not asked --", bool(gn)
        else:
            state, n = classify(ltype, lcounts, ldeclared)
            ltxt = f"{state}" + (f" ({n})" if n else "")
            ok = bool(gn) and state == "populated"
        runnable += 1 if ok else 0
        print(f"  {meaning[:33]:34} {ltxt:22} {gtxt}")
    total = len(SCENARIOS)
    print(f"\n  runnable: {runnable} of {total}")
    if runnable < total:
        # WHO CAN FIX IT is a separate axis from WHAT IS MISSING, and the more
        # useful one. Creating a workflow state is a team setting with no API
        # surface here; putting an issue INTO a state that already exists is an
        # ordinary write. Printing both as "missing" leaves the reader to
        # discover which of the two they can automate, one failed script later.
        human, scripted, unknown = [], [], []
        for meaning, ltype, _e, gh_state in SCENARIOS:
            gn = (gcounts or {}).get(gh_state, 0)
            if not gn:
                scripted.append(f"github: no issue is {gh_state!r} within the newest "
                                f"{gseen} -- create one, or search deeper")
            if github_only:
                continue
            state, _n = classify(ltype, lcounts, ldeclared)
            fixer = who_fixes(state)
            if fixer == "scripted":
                scripted.append(f"linear: status type {ltype!r} exists and holds "
                                f"nothing -- create an issue and set it to that state")
            elif fixer == "human":
                human.append(f"linear: this team has no status of type {ltype!r}. "
                             f"Add one in Team settings -> Workflow"
                             + (" (Triage is a team feature, enable it there)"
                                if ltype == "triage" else ""))
            elif fixer == "unknown":
                unknown.append(f"linear: nothing of type {ltype!r} present -- include "
                               f"list_issue_statuses output to tell 'absent' from "
                               f"'declared', which decides who can fix it")
        if human:
            print("\n  A HUMAN MUST DO THESE -- no API here reaches a team's workflow "
                  "settings:")
            for line in human:
                print(f"    {line}")
        if scripted:
            print("\n  These can be scripted:")
            for line in scripted:
                print(f"    {line}")
        if unknown:
            print("\n  Cannot tell who fixes these without the status vocabulary:")
            for line in unknown:
                print(f"    {line}")
    return 0 if runnable == len(SCENARIOS) else 1


def self_test():
    fails = 0

    def check(label, ok, detail=""):
        nonlocal fails
        print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
        fails += 0 if ok else 1

    check("the five scenarios are imported, not restated",
          SCENARIOS is _le.SCENARIOS and len(SCENARIOS) == 5,
          "a local copy of the table is a second source of truth")
    # The three-way answer is the point of the tool.
    check("a populated type reports populated",
          classify("backlog", {"backlog": 3}, ["backlog"]) == ("populated", 3))
    check("a declared-but-empty type is NOT reported absent",
          classify("canceled", {}, ["canceled", "backlog"]) == ("declared", 0),
          "a status that exists with nothing in it needs an issue, not a status")
    check("an absent type is NOT reported declared",
          classify("triage", {}, ["canceled", "backlog"]) == ("absent", 0),
          "a team with no triage status needs one created")
    check("without the vocabulary the two are not guessed apart",
          classify("triage", {}, None) == ("absent-or-declared", 0),
          "guessing here would invent a fact the payload does not carry")
    # Absence of evidence is not evidence of absence (ADR-003 rule 6).
    check("a populated type is populated whether or not it was declared",
          classify("started", {"started": 1}, None) == ("populated", 1))
    # WHO fixes it, routed through the same function the report uses.
    check("a missing status is a human's job, not a script's",
          who_fixes("absent") == "human",
          "creating a workflow state is a team setting with no API surface here")
    check("an empty-but-declared status is scriptable",
          who_fixes("declared") == "scripted",
          "putting an issue into a state that exists is an ordinary write")
    check("who-fixes-it covers every classify() answer",
          set(FIXER) == {"populated", "declared", "absent", "absent-or-declared"},
          "a state with no fixer would be printed as nothing at all")
    print(f"\nPROVIDER-READINESS SELF-TEST: {'PASS' if not fails else f'FAIL ({fails})'}")
    return 1 if fails else 0


def main(argv):
    if "--self-test" in argv:
        return self_test()
    if "--github" not in argv:
        print(__doc__.strip().split("\n\n")[-2], file=sys.stderr)
        return 2
    nwo = argv[argv.index("--github") + 1]
    github_only = "--github-only" in argv

    lcounts, ldeclared = {}, None
    if not github_only:
        raw = sys.stdin.read().strip()
        if not raw:
            print("no MCP payload on stdin; pass --github-only to skip Linear.",
                  file=sys.stderr)
            return 2
        try:
            lcounts, ldeclared = linear_states(json.loads(raw))
        except (json.JSONDecodeError, AttributeError) as e:
            print(f"stdin is not an MCP payload: {e}", file=sys.stderr)
            return 2

    gcounts, gseen, err = github_states(nwo)
    if gcounts is None:
        print(f"could not read {nwo}: {err}", file=sys.stderr)
        return 2
    return report(lcounts, ldeclared, gcounts, gseen, github_only)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
