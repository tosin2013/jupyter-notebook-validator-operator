#!/usr/bin/env python3
"""Layer 2 against two LIVE roadmap providers — issue #1's actual bar.

`conformance/layer2.py` runs both trackers on recorded fixtures, because ADR-008
rule 1 requires fixed inputs for determinism (C7). That is correct for a
regression gate and it is why Layer 2 alone cannot close #1: fixtures written by
one author against one mental model measure shared intent as much as
portability.

This runs the same equivalence question against real Linear and real GitHub. It
is deliberately NOT a conformance suite:

  * it is non-deterministic by construction -- the providers change under it;
  * a divergence here is a FINDING for §55, not a broken build;
  * it needs network and credentials the suites must never require.

Linear state arrives on stdin as an MCP payload, so no Linear credential is
needed by this tool or by the adapter (ADR-020, ADR-028). Nothing about the
payload is written to disk: this repository is public and §51 forbids carrying
another workspace's content into it.

    <mcp list_issues output> | python3 tools/live-equivalence.py --github <owner/repo>
    python3 tools/live-equivalence.py --github <owner/repo> --github-only
    python3 tools/live-equivalence.py --self-test

The MCP payload must include id, title, status, statusType — the same fields
`adapters/linear` requires. A payload missing any of them is a usage error
(exit 2), not a §55 divergence. The adapter refuses it as MALFORMED_SOURCE;
scoring that as disagreement would fire the stop condition because the caller
omitted a field.

Five scenarios. GitHub can now express all five live, including closed
NOT_PLANNED. Linear may still lack `triage` or `canceled`. A scenario
expressible in only one provider is observed against the expected map and
counted as SKIP, never as agreement -- the first version of this tool scored
one-sided rows as AGREE, which is a green number for a comparison it never
made.

Equivalence is over SEMANTIC STATE, not over the same work item. Layer 2 has
always worked this way -- `authority_withdrawn` uses different ids in each
provider. Two providers agreeing about one shared item would test id lookup;
agreeing about equivalent states is what tests normalization.

Agreement without correctness is not success. Two adapters that both map
withdrawal to ADMITTED would have printed AGREE. Layer 2 already refuses that
(`WRONG`); this tool now does too. Observe() stores `__unknown__: True` and
keeps the reason diagnostic, matching layer2.py -- comparing reason strings
punishes a more specific code, which is the better code.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# Same keys layer2.py projects over for disposition-relevant facts.
# `reason` is diagnostic and MUST NOT appear here.
COMPARE_KEYS = ("authority", "admitted", "__unknown__", "blocking", "__error__")

# Must stay identical to adapters/linear MCP_REQUIRED. The live tool used to
# forward a thin payload, receive MALFORMED_SOURCE, and score it as DIVERGENCE
# — a stop-condition input caused by the caller omitting `title`.
MCP_REQUIRED = ("id", "title", "status", "statusType")

# The mapping under test. Each row is one semantic state, expressed in whatever
# each provider natively uses to mean it. `expect` is the typed fact the engine
# must be able to conclude -- agreement between providers is not enough.
SCENARIOS = [
    ("admitted, not cleared to execute", "backlog",
     {"authority": "ADMITTED", "admitted": True},
     "milestoned, unassigned"),
    ("authorized and executing", "started",
     {"authority": "AUTHORIZED", "admitted": True},
     "milestoned + assigned"),
    ("finished; authority is a separate axis", "completed",
     {"authority": "AUTHORIZED", "admitted": True},
     "closed"),
    # `triage`, not None. A previous row carried `None`, which made `lid`
    # None unconditionally: a scenario that could never execute, reported
    # forever as "not expressible live". A permanently-skipped row reads like
    # missing data and is really a dead code path.
    ("not admitted at all", "triage",
     {"__unknown__": True, "blocking": True},
     "no milestone"),
    # The withdrawal case: the one that motivated the entire project.
    ("authority withdrawn", "canceled",
     {"authority": "CANCELLED", "admitted": False},
     "closed NOT_PLANNED"),
]


def observe(resp):
    """Reduce a response to the typed facts an engine would act on.

    Identical contract to conformance/layer2.py: an unknown is `__unknown__:
    True` plus a diagnostic `reason`. Storing the reason string in
    `__unknown__` made two equivalent unknowns diverge whenever the adapters
    named the gap differently.
    """
    if not resp.get("ok"):
        return {"__error__": resp.get("error", {}).get("type")}
    if resp.get("unknown"):
        return {"__unknown__": True, "blocking": resp["unknown"]["blocking"],
                "reason": resp["unknown"]["reason"]}
    return resp.get("value") or {}


def projection(obs):
    return {k: obs[k] for k in COMPARE_KEYS if k in obs}


def matches(obs, expect):
    return all(obs.get(k) == v for k, v in expect.items())


def score(lp, gp, expect):
    """Classify a compared pair of projections against the expected map.

    Extracted so the self-test can call the exact function `main()` uses,
    rather than testing helpers adjacent to the scoring line while leaving
    the line itself unexercised. That gap is how `expect` went unused and
    `agreement` was decided purely by `lp == gp` -- covered by construction,
    not verified by any check.

    "agree"   -- providers match AND the match is the expected value.
    "wrong"   -- providers match each other but NOT the expected value.
                 Agreement without correctness is not success: two adapters
                 that both mapped withdrawal to ADMITTED would print AGREE
                 under a same-only check.
    "diverge" -- providers disagree. A normalization failure, §55 input.
    """
    same = lp == gp
    correct = matches(lp, expect) and matches(gp, expect)
    if same and correct:
        return "agree"
    if not same:
        return "diverge"
    return "wrong"


def mcp_missing(issues):
    return sorted({f for n in issues for f in MCP_REQUIRED if f not in n})


def bucket_github(issue):
    """Which live-equivalence bucket a GitHub issue occupies, or None.

    NOT_PLANNED is matched first: bucketing it as merely `closed` would make
    the withdrawal scenario match a finished issue, which is the confusion
    the completion firewall exists to prevent. A milestone on a rejected
    issue does not turn rejection into completion.
    """
    ms = bool(issue.get("milestone"))
    closed = issue.get("state") == "CLOSED"
    assigned = bool(issue.get("assignees"))
    if closed and issue.get("stateReason") == "NOT_PLANNED":
        return "closed NOT_PLANNED"
    if not ms and not closed:
        return "no milestone"
    if ms and closed:
        return "closed"
    if ms and assigned and not closed:
        return "milestoned + assigned"
    if ms and not assigned and not closed:
        return "milestoned, unassigned"
    return None


def ask_linear(payload, wid):
    p = subprocess.run([sys.executable, str(ROOT / "adapters" / "linear"),
                        "query", "roadmap_authority", "get_authority", f"id={wid}", "--input", "-"],
                       input=payload, capture_output=True, text=True, cwd=ROOT, timeout=60)
    try:
        return observe(json.loads(p.stdout))
    except json.JSONDecodeError:
        return {"__error__": "NON_JSON"}


def ask_github(nwo, number):
    p = subprocess.run([sys.executable, str(ROOT / "adapters" / "github-projects"),
                        "query", "roadmap_authority", "get_authority", f"id={number}"],
                       capture_output=True, text=True, cwd=ROOT, timeout=90,
                       env={**os.environ, "REPO_GOVERNOR_GH_REPO": nwo,
                            "REPO_GOVERNOR_GH_ADMISSION": "milestone"})
    try:
        return observe(json.loads(p.stdout))
    except json.JSONDecodeError:
        return {"__error__": "NON_JSON"}


def pick_github(nwo):
    """Real issues occupying each semantic state. Chosen from live data, not fixed."""
    out = subprocess.run(["gh", "issue", "list", "--repo", nwo, "--state", "all", "--limit", "100",
                          "--json", "number,state,stateReason,milestone,assignees"],
                         capture_output=True, text=True, timeout=90).stdout
    issues = json.loads(out or "[]")
    picks = {}
    for i in issues:
        bucket = bucket_github(i)
        if bucket:
            picks.setdefault(bucket, i["number"])
    return picks


def _fmt(obs):
    extra = f"   (reason: {obs['reason']})" if "reason" in obs else ""
    return json.dumps(projection(obs), sort_keys=True) + extra


def self_test():
    """No network. The projection and bucket lessons, as checks rather than comments."""
    fails = 0

    def check(label, ok):
        nonlocal fails
        print(f"  [{'PASS' if ok else 'FAIL'}] {label}")
        if not ok:
            fails += 1

    print("live-equivalence instrument\n")
    check("NOT_PLANNED without a milestone is withdrawal",
          bucket_github({"number": 40, "state": "CLOSED", "stateReason": "NOT_PLANNED",
                         "milestone": None, "assignees": []}) == "closed NOT_PLANNED")
    check("NOT_PLANNED wins over a milestone — rejection is not completion",
          bucket_github({"number": 1, "state": "CLOSED", "stateReason": "NOT_PLANNED",
                         "milestone": {"title": "v1"}, "assignees": []}) == "closed NOT_PLANNED")
    check("COMPLETED closed milestoned is finished, not withdrawn",
          bucket_github({"number": 2, "state": "CLOSED", "stateReason": "COMPLETED",
                         "milestone": {"title": "v1"}, "assignees": []}) == "closed")
    check("open unmilestoned is not admitted",
          bucket_github({"number": 3, "state": "OPEN", "stateReason": None,
                         "milestone": None, "assignees": []}) == "no milestone")
    check("milestoned assigned is authorized",
          bucket_github({"number": 4, "state": "OPEN", "stateReason": None,
                         "milestone": {"title": "v1"},
                         "assignees": [{"login": "x"}]}) == "milestoned + assigned")
    check("milestoned unassigned is admitted-not-authorized",
          bucket_github({"number": 5, "state": "OPEN", "stateReason": None,
                         "milestone": {"title": "v1"}, "assignees": []}) == "milestoned, unassigned")

    unk = observe({"ok": True, "unknown": {"reason": "NOT_ADMITTED", "blocking": True}})
    check("unknown stores True, not the reason string",
          unk.get("__unknown__") is True and unk.get("reason") == "NOT_ADMITTED")
    check("projection drops reason so a more specific code is not a divergence",
          projection(unk) == {"__unknown__": True, "blocking": True})

    # These three exercise score() directly -- the function main() actually
    # calls to decide AGREE / WRONG / DIVERGENCE. A prior version of this
    # check compared a dict to a copy of itself (`wrong == dict(wrong)`),
    # which cannot fail, and never called the scoring path at all: proven by
    # mutation, forcing `correct = True` inside the old inlined logic left
    # this suite green. Calling score() closes that gap by construction --
    # any future rewrite of the scoring line runs through the same function.
    correct_withdrawal = {"authority": "CANCELLED", "admitted": False}
    wrong_but_agreeing = {"authority": "ADMITTED", "admitted": True}
    disagreeing = {"authority": "AUTHORIZED", "admitted": True}
    check("providers agreeing on the CORRECT value scores agree",
          score(correct_withdrawal, dict(correct_withdrawal), correct_withdrawal) == "agree")
    check("two providers agreeing on the WRONG map scores wrong, not agree",
          score(wrong_but_agreeing, dict(wrong_but_agreeing), correct_withdrawal) == "wrong")
    check("providers disagreeing with each other scores diverge",
          score(wrong_but_agreeing, disagreeing, correct_withdrawal) == "diverge")
    # `matches` is a SUBSET test, so two different projections can both satisfy
    # the same expectation by differing only on a key the expectation omits.
    # Without this case, `if same and correct` degrades to `if correct` with the
    # suite still green: the three cases above all fail `correct` when they
    # disagree, so the mutant reaches the right answer by accident. Here one
    # provider reports `blocking` and the other is silent -- a real divergence
    # that both sides map correctly.
    check("providers differing outside the expected keys still diverge",
          score({"authority": "CANCELLED", "admitted": False, "blocking": True},
                {"authority": "CANCELLED", "admitted": False},
                correct_withdrawal) == "diverge")
    check("a payload missing title is a usage error, not a divergence",
          mcp_missing([{"id": "x", "statusType": "backlog"}]) == ["status", "title"])

    print()
    print("LIVE-EQUIVALENCE SELF-TEST: " + ("PASS" if fails == 0 else f"FAIL ({fails})"))
    return 0 if fails == 0 else 1


def main(argv):
    if "--self-test" in argv:
        return self_test()
    if "--github" not in argv:
        print("Usage: python3 tools/live-equivalence.py --github owner/repo [--github-only]",
              file=sys.stderr)
        print("       python3 tools/live-equivalence.py --self-test", file=sys.stderr)
        return 2
    nwo = argv[argv.index("--github") + 1]
    github_only = "--github-only" in argv

    by_type = {}
    payload = ""
    if not github_only:
        payload = sys.stdin.read()
        try:
            issues = json.loads(payload)["issues"]
        except (json.JSONDecodeError, KeyError, TypeError):
            print("stdin is not an MCP issues payload", file=sys.stderr)
            return 2
        for i in issues:
            by_type.setdefault(i.get("statusType"), i.get("id"))
        missing = mcp_missing(issues)
        if missing:
            print(f"MCP payload is missing required field(s) {missing}. "
                  f"Request fields={list(MCP_REQUIRED)}.", file=sys.stderr)
            return 2
    gh_picks = pick_github(nwo)

    print(f"Linear: {len(by_type)} live state types {sorted(k for k in by_type if k) or '[]'}"
          + ("  (--github-only)" if github_only else ""))
    print(f"GitHub: {nwo}, states {sorted(gh_picks)}\n")

    agree = diverge = mismatched = skipped = errors = 0
    observed_ok = observed_wrong = 0
    for meaning, ltype, expect, gh_state in SCENARIOS:
        lid = by_type.get(ltype)
        gid = gh_picks.get(gh_state)
        lobs = ask_linear(payload, lid) if lid else None
        gobs = ask_github(nwo, gid) if gid else None

        # A scenario expressible in ONE provider is not an equivalence test.
        # Counting it as agreement is how a harness reports a green number for a
        # comparison it never made -- layer2.py already refuses this.
        # Observing the available side against `expect` is not that: it records
        # whether the live mapping holds, without claiming a comparison.
        if lobs is None and gobs is None:
            print(f"[SKIP] {meaning}\n       not expressible live: "
                  f"linear={ltype}:{lid} github={gh_state}:{gid}\n")
            skipped += 1
            continue
        if (lobs or {}).get("__error__") or (gobs or {}).get("__error__"):
            print(f"[{meaning}]")
            if lobs is not None:
                print(f"    linear  {ltype:<10} {_fmt(lobs)}")
            if gobs is not None:
                print(f"    github  {gh_state:<22} {_fmt(gobs)}")
            print("    ** PROVIDER ERROR ** not a normalization finding "
                  "(malformed input, missing credentials, or a transport failure)\n")
            errors += 1
            continue
        if lobs is None or gobs is None:
            side, obs, label = (("github", gobs, gh_state) if gobs is not None
                                else ("linear", lobs, ltype))
            ok = matches(projection(obs), expect)
            skipped += 1
            if ok:
                observed_ok += 1
            else:
                observed_wrong += 1
            print(f"[SKIP] {meaning}")
            print(f"    {side:<7} {label:<22} {_fmt(obs)}")
            print(f"    OBSERVED {'CORRECT' if ok else 'WRONG'}  "
                  f"(one provider; not an equivalence test)\n")
            continue

        lp, gp = projection(lobs), projection(gobs)
        print(f"[{meaning}]")
        print(f"    linear  {ltype:<10} {_fmt(lobs)}")
        print(f"    github  {gh_state:<22} {_fmt(gobs)}")
        result = score(lp, gp, expect)
        if result == "agree":
            agree += 1
            print("    AGREE + CORRECT\n")
        elif result == "diverge":
            diverge += 1
            print("    ** DIVERGENCE ** equivalent state, different typed facts\n")
        else:
            mismatched += 1
            print("    ** WRONG ** agree with each other but violate expectation")
            print(f"       expected {json.dumps(expect, sort_keys=True)}\n")

    print("-" * 62)
    compared = agree + diverge + mismatched
    print(f"live scenarios: {len(SCENARIOS)}   agree: {agree}   diverge: {diverge}   "
          f"wrong: {mismatched}   skipped: {skipped}   errors: {errors}")
    if errors:
        print("\nLIVE EQUIVALENCE: PROVIDER ERROR — not a §55 finding. "
              "Fix the payload or credentials and re-run.")
        return 2
    if github_only:
        n = observed_ok + observed_wrong
        print("\nGITHUB LIVE MAPPING: "
              + ("CORRECT " if not observed_wrong else "WRONG ")
              + f"across {observed_ok} of {n} expressible scenarios")
        print("Not an equivalence test — one provider matching its own map "
              "cannot close issue 1.")
        if observed_wrong:
            print("A WRONG live mapping is a §55 stop-condition input, not a build failure. Record it.")
        return 0 if not observed_wrong else 1

    scope = f"across {compared} of {len(SCENARIOS)} scenarios"
    if skipped:
        scope += f" ({skipped} not expressible in these providers"
        if observed_ok or observed_wrong:
            scope += f"; {observed_ok} observed CORRECT, {observed_wrong} WRONG"
        scope += ")"
    print("\nLIVE EQUIVALENCE: " + ("EQUIVALENT " if not diverge and not mismatched
                                    else "NOT EQUIVALENT ") + scope)
    if diverge or mismatched:
        print("A divergence is a §55 stop-condition input, not a build failure. Record it.")
    if observed_wrong:
        print("A one-sided WRONG is the same: the live map failed even without a peer.")
        return 1
    return 0 if not diverge and not mismatched else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
