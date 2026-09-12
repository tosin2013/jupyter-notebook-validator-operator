#!/usr/bin/env python3
"""The status report tells the truth about what it cannot do (issue 57).

A report is easy to make green: print less. So these checks are mostly about
ABSENCE being reported rather than omitted -- the same distinction ADR-008 C4
makes for adapters, applied to the surface a human reads.

The one that matters most is the last: a repository with no acceptance criteria
must be TOLD that STOP_COMPLETE is unreachable. That is the condition under
which the completion firewall silently never fires, it is the state every
freshly onboarded repository is in, and a status view that omitted it would be
hiding the single most important thing about the repository it is describing.

Usage:  python3 conformance/status.py
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("status")


ROOT = Path(__file__).resolve().parent.parent
STATUS = ROOT / "engine" / "status.py"
SCHEMA = ROOT / "schemas" / "manifest-v1.json"


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def run(path, env_extra=None):
    env = dict(os.environ)
    env.pop("REPO_GOVERNOR_TARGET", None)
    env.update(env_extra or {})
    p = subprocess.run([sys.executable, str(STATUS), str(path)],
                       capture_output=True, text=True, cwd=str(ROOT), env=env, timeout=120)
    return p.returncode, p.stdout + p.stderr


def bare_repo(td, *, manifest=None, criteria=None):
    r = Path(td) / "repo"
    (r / ".repo-governor").mkdir(parents=True)
    subprocess.run(["git", "init", "-q", str(r)], capture_output=True)
    subprocess.run(["git", "-C", str(r), "remote", "add", "origin",
                    "https://github.com/acme/widget.git"], capture_output=True)
    (r / "roadmap.json").write_text(json.dumps(
        {"items": [{"id": "W-1", "title": "x", "status": "IN_PROGRESS",
                    "authority": "AUTHORIZED", "admitted": True,
                    "scope": [], "non_goals": [], "acceptance_conditions": []}]}))
    if manifest is not None:
        (r / ".repo-governor.json").write_text(json.dumps(manifest))
    for cid in (criteria or []):
        d = r / ".repo-governor" / "acceptance"
        d.mkdir(parents=True, exist_ok=True)
        (d / f"{cid}.json").write_text(json.dumps(
            {"authority_id": cid, "criteria": [{"check": "file_exists", "target": "README.md"}]}))
    return r


MINIMAL = {
    "repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
    "repository": {"id": "acme/widget"},
    "condition": {"assessed": "L1", "profile": "GOVERNOR_LITE"},
    "permissions": {"repository": {"read": True, "write": False},
                    "roadmap_authority": {"read": True, "write": False}},
    "providers": {
        "repository": {"type": "git", "adapter": "adapters/git", "contract_version": 1},
        "roadmap_authority": {"type": "file-roadmap", "adapter": "adapters/file-roadmap",
                              "contract_version": 1,
                              "env": {"REPO_GOVERNOR_ROADMAP": "roadmap.json"}},
    },
}


def main():
    fails = 0
    print("Every role the schema defines appears, bound or not\n")

    rc, out = run(ROOT)
    fails += check("it runs against this repository", rc == 0, out[-400:])

    schema_roles = sorted(
        k for k in json.loads(SCHEMA.read_text())["properties"]["providers"]["properties"]
        if not k.startswith("$"))
    fails += check(f"the schema still defines a role set ({len(schema_roles)})",
                   len(schema_roles) >= 6,
                   "if this is empty every check below passes having compared nothing")
    missing = [r for r in schema_roles if r not in out]
    fails += check("every schema role is named in the report", not missing,
                   f"{missing} — a role omitted from the report is a governance question "
                   "the reader does not know went unasked")

    print("\nUnbound is reported, not omitted\n")
    with tempfile.TemporaryDirectory() as td:
        r = bare_repo(td, manifest=MINIMAL)
        rc, out = run(r)
        fails += check("a two-role repository still reports", rc == 0, out[-300:])
        for role in ("decision_history", "acceptance_criteria", "architecture"):
            fails += check(f"{role} is named as unbound", f"{role}" in out and "UNBOUND" in out,
                           "silence about an unbound role reads as a bound one")
        fails += check("INV-013 is cited for the unbound state", "INV-013" in out,
                       "an unbound provider has no governance function; say why")

        # THE CHECK THIS SUITE EXISTS FOR.
        fails += check("STOP_COMPLETE is reported UNREACHABLE with no criteria",
                       "STOP_COMPLETE" in out and "NOT reachable" in out,
                       "every freshly onboarded repository is in this state, and a report "
                       "that omits it hides the completion firewall never firing")
        fails += check("the record path is reported unreachable with decision_history unbound",
                       "CAPTURE_ONLY" in out and "nowhere to write" in out,
                       "CAPTURE_ONLY is the DEFAULT disposition; if it cannot record, say so")

    print("\nDeclaring criteria changes the answer\n")
    with tempfile.TemporaryDirectory() as td:
        r = bare_repo(td, manifest=MINIMAL, criteria=["W-1"])
        rc, out = run(r)
        seg = out.split("WHAT THIS REPOSITORY CAN CONCLUDE")[-1].split("LOCAL EVIDENCE")[0]
        line = [l for l in seg.splitlines() if "STOP_COMPLETE" in l]
        fails += check("with criteria present, STOP_COMPLETE reads reachable",
                       bool(line) and "NOT reachable" not in line[0],
                       f"{line} — otherwise the check above passes for a reason unrelated "
                       "to criteria, and neither line means anything")

    # A disposition NOTHING EMITS must not be listed as reachable, and one that
    # is emitted must not be omitted. status.py claimed `CONFLICT reachable`
    # whenever any role was multi-bound; no path in engine/ emits that
    # disposition, and a validated manifest cannot even hold the case it was
    # meant for -- two peers on a single-valued role is a load-time CARDINALITY
    # error (ADR-013 rule 1). Meanwhile classify() returns ARCHITECTURE_REVIEW
    # for every ARCHITECTURE_IMPLICATION and status.py did not list it.
    #
    # Wrong in both directions, on the surface whose entire job is saying what
    # this repository can conclude. Same defect family as the architecture
    # caveat issue 143 fixed, and the reason it went unnoticed is that no suite
    # asserted the negative (issue 154).
    print("\nIt claims only dispositions the engine can actually emit\n")
    rc, out = run(ROOT)
    seg = out.split("WHAT THIS REPOSITORY CAN CONCLUDE")[-1].split("OBLIGATIONS")[0]
    conflict = [l for l in seg.splitlines() if l.strip().startswith("CONFLICT")]
    fails += check("CONFLICT is not claimed reachable",
                   bool(conflict) and "NOT reachable" in conflict[0],
                   f"{conflict} — nothing in engine/ emits it; grep for it and the only "
                   "hit is the line that claims it")
    # The EXACT label, not a prefix. The first draft used startswith() and was
    # satisfied by the neighbouring "ARCHITECTURE_REVIEW x2" line, so deleting
    # the one it was written about left it green -- a check passing for the
    # wrong reason, caught by mutating the thing it claimed to watch.
    arch_line = [l for l in seg.splitlines()
                 if l.split()[:1] == ["ARCHITECTURE_REVIEW"] and "\u00d7" not in l]
    fails += check("ARCHITECTURE_REVIEW is listed reachable, because classify() emits it",
                   bool(arch_line) and "NOT reachable" not in arch_line[0],
                   f"{arch_line} — omitting an emittable disposition understates the "
                   "repository the same way claiming an unemittable one overstates it")

    print("\nIt is not silently green\n")
    with tempfile.TemporaryDirectory() as td:
        r = bare_repo(td, manifest=None)          # no manifest at all
        rc, out = run(r)
        fails += check("an un-onboarded repository exits non-zero", rc != 0, f"rc={rc}")
        fails += check("and names what was missing", "AUTHORITY_SOURCE_MISSING" in out
                       or "MANIFEST INVALID" in out, out[-200:])
    with tempfile.TemporaryDirectory() as td:
        broken = dict(MINIMAL); broken["repo_governor"] = {"version": 99}
        r = bare_repo(td, manifest=broken)
        rc, out = run(r)
        fails += check("an unreadable manifest exits non-zero rather than reporting from it",
                       rc != 0, f"rc={rc}")

    print("\nObligation facts are reported, and gate nothing\n")

    with tempfile.TemporaryDirectory() as td:
        r = bare_repo(td, manifest=MINIMAL)          # no LICENSE, no README
        rc, out = run(r)
        fails += check("a repository with no licence is told so", "licence" in out
                       and "ABSENT" in out,
                       "all-rights-reserved by default is a fact its owner should know")
        fails += check("and told what that actually means", "grant" in out.lower(),
                       "'no licence' without the consequence is trivia")
        fails += check("and told it is not a blocker", "does not block" in out,
                       "an obligation note that reads like a refusal is over-escalation (section 54)")
        fails += check("the report still succeeds", rc == 0,
                       "a missing licence must not turn the report itself into a failure")

        # The load-bearing one: it must not gate. A repository with no licence
        # still gets verdicts.
        env = dict(os.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
        pr = subprocess.run([sys.executable, str(ROOT / "engine" / "completion.py"), "W-1"],
                            capture_output=True, text=True, cwd=str(r), env=env, timeout=300)
        try:
            got = json.loads(pr.stdout)
        except Exception:
            got = {}
        fails += check("a licence-less repository still receives a verdict",
                       got.get("decision") is not None,
                       f"{pr.stdout[:150]} -- section 54: it must not block routine work")
        fails += check("and the verdict names no licence reason",
                       "licen" not in json.dumps(got).lower(),
                       "an obligation indicator must not leak into a governance disposition")

        r2 = bare_repo(td + "/withlic", manifest=MINIMAL)
        (r2 / "LICENSE").write_text("Apache License\n")
        rc2, out2 = run(r2)
        fails += check("positive control: a licensed repository reads present",
                       "licence   present" in out2,
                       "otherwise the ABSENT check passes because it always says ABSENT")

    print("\nA profile's required roles are read, not merely declared\n")

    import sys as _s
    _s.path.insert(0, str(ROOT / "engine"))
    import vocabulary as _V

    fails += check("policies still declare required_roles",
                   bool(_V.required_roles("GOVERNOR_HIGH_ASSURANCE")),
                   "the mapping lived in policies/*.json unread for the life of the "
                   "project; an empty read here would restore that silently")
    fails += check("a heavier profile requires more than a lighter one",
                   len(_V.required_roles("GOVERNOR_HIGH_ASSURANCE"))
                   > len(_V.required_roles("GOVERNOR_LITE")),
                   "progressive governance (ADR-006) is the whole point")
    fails += check("an unknown profile requires nothing, and invents nothing",
                   _V.required_roles("NOT_A_PROFILE") == (),
                   "guessing a default would be the assumption ADR-018 forbids, "
                   "applied to profiles")

    with tempfile.TemporaryDirectory() as td:
        heavy = dict(MINIMAL)
        heavy["condition"] = {"assessed": "L4", "profile": "GOVERNOR_HIGH_ASSURANCE"}
        r = bare_repo(td, manifest=heavy)
        rc, out = run(r)
        fails += check("an L4 repository is told which required roles are unbound",
                       "REQUIRED by GOVERNOR_HIGH_ASSURANCE" in out,
                       "unbound and unbound-but-required are different facts")
        fails += check("and told it is a configuration gap, not a verdict",
                       "not a verdict" in out,
                       "an unmet requirement that reads like a refusal is the "
                       "over-escalation section 54 names")
        fails += check("the report still succeeds", rc == 0,
                       "an unmet requirement must not turn the report into a failure")

        # Positive control: a profile whose requirement IS met says nothing.
        light = dict(MINIMAL)
        light["condition"] = {"assessed": "L1", "profile": "GOVERNOR_LITE"}
        r2 = bare_repo(td + "/light", manifest=light)
        _rc2, out2 = run(r2)
        fails += check("control: a profile whose requirements are met reports none unbound",
                       "REQUIRED by GOVERNOR_LITE" not in out2,
                       "otherwise the check above passes for any repository at all")

    print("\nIt does not claim what no provider can supply\n")
    rc, out = run(ROOT)
    fails += check("it says admitted-work-without-criteria is NOT COMPUTABLE",
                   "NOT COMPUTABLE" in out,
                   "no roadmap adapter advertises enumeration; claiming the set, or "
                   "printing nothing, would both misrepresent that")

    print("\nA role that answers a question nothing acts on says so\n")
    # Issue 143. A repository can bind thirty ADRs and see only
    # "bound adapters/adr answers", which says nothing about whether they were
    # read or what they mean -- and the binding implies a constraint the engine
    # never consults. That matters more now that onboarding proposes this role
    # when it detects ADRs (issue 144).
    rc3, out3 = run(ROOT)
    fails += check("architecture reports the state it resolved",
                   "DEFINED" in out3,
                   "get_constraints answers DEFINED / INFERRED / UNKNOWN and the "
                   "operator was shown none of it")
    fails += check("...and the decisions it read, by status",
                   "Accepted" in out3 and "Proposed" in out3,
                   "counts and named states, never a ratio")
    fails += check("...and that no disposition consults them",
                   "no disposition consults this" in out3,
                   "the role answers 'what constrains how it must be built?' and "
                   "nothing reads the answer; leaving that unsaid lets the binding "
                   "imply governance it does not perform")

    # NOT A SCORE. The refusal is in status.py's own docstring -- "a number
    # invites optimising the number" -- and it bites hardest here: a Proposed
    # ADR is not worse than an Accepted one. ADR-031 is correctly Proposed
    # pending a measurement on a host this project does not own, and a
    # percentage would read that as debt. (ADR-024 was the example until it was
    # accepted on 2026-08-31.)
    import re as _re
    fails += check("no ratio or percentage is printed for the ledger",
                   not _re.search(r"architecture[\s\S]{0,220}?(\d+\s*%|\d+\s*/\s*\d+\s+(?:Accepted|ADRs))", out3),
                   "a score would create pressure to accept decisions to raise it, "
                   "which is admission without authority applied to an architecture "
                   "ledger")

    print(f"\n{'STATUS: CONFORMANT' if not fails else f'STATUS: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
