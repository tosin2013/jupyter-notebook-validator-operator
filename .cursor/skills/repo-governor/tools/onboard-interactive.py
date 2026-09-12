#!/usr/bin/env python3
"""Ask the questions detection cannot answer, and emit a manifest that validates.

`engine/onboard.py --write` produces a CANDIDATES document: what was found on
disk, with evidence, and `PROVIDER_UNCONFIRMED` wherever it would have to guess.
That file is evidence, not a manifest -- renaming it yields
`UNSUPPORTED_VERSION: manifest version None`, which was the documented
instruction until 2026-08-19 and had never been run end to end.

The gap is not ceremony. A manifest declares **which provider is the roadmap
authority and what signal means admission** (ADR-018), and neither is visible on
the filesystem. Whether admission means a milestone, a label or a project column
is a fact about how a team works. Detection must not guess it; a person can just
say it. So: ask.

Nothing here binds. Output is `.repo-governor.proposed.json` in manifest shape,
which a human reviews and renames. The engine never reads a proposal, which is
why silent binding stays unimplementable rather than merely forbidden.

Usage:  python3 tools/onboard-interactive.py <repo-path>
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SKILL = HERE.parent
PROPOSAL = ".repo-governor.proposed.json"

# Roadmap providers with a shipped adapter. "Something else" is a first-class
# answer: a tracker we do not support is a contribution signal, not a dead end.
ROADMAP = [
    ("github-projects", "adapters/github-projects", "GitHub"),
    ("linear",          "adapters/linear",          "Linear (MCP transport)"),
    ("file-roadmap",    "adapters/file-roadmap",    "A file in the repository"),
]
GH_SIGNALS = [
    ("milestone",      "Milestone membership means admitted"),
    ("project_status", "A Project column/status means admitted"),
    ("label",          "A specific label means admitted"),
    ("none",           "Every open issue is admitted (rare -- says nothing is triage)"),
]


def ask(prompt, options, allow_other=True, unavailable=None):
    """options: list of (value, description). Returns a value, or None for other.

    `unavailable` maps value -> why. Those options KEEP THEIR POSITION and are
    refused if chosen. Removing them renumbers the list, and renumbering turned
    "picked an impossible option" into "picked a different option" on the very
    next run -- the same keystroke, a different meaning, and a manifest bound to
    the wrong signal without anyone noticing.
    """
    unavailable = unavailable or {}
    print(f"\n{prompt}")
    for i, (val, desc) in enumerate(options, 1):
        mark = f"   -- UNAVAILABLE: {unavailable[val]}" if val in unavailable else ""
        print(f"  {i}. {desc}{mark}")
    if allow_other:
        print(f"  {len(options) + 1}. Something else / not listed")
    while True:
        raw = input("> ").strip()
        if raw.isdigit():
            n = int(raw)
            if 1 <= n <= len(options):
                val = options[n - 1][0]
                if val in unavailable:
                    print(f"  {val} cannot work here: {unavailable[val]}.")
                    print("  Nothing would ever be admitted under it. Pick another.")
                    continue
                return val
            if allow_other and n == len(options) + 1:
                return None
        print("Pick a number.")


def repo_id(repo: Path):
    """owner/repo from the git remote. Never defaulted -- ADR-028 exists because
    an adapter once fell back to the author's repository."""
    try:
        url = subprocess.run(["git", "-C", str(repo), "remote", "get-url", "origin"],
                             capture_output=True, text=True).stdout.strip()
    except Exception:
        url = ""
    m = re.search(r"[:/]([^/:]+/[^/]+?)(?:\.git)?$", url)
    return m.group(1) if m else None


REPO = "tosin2013/repo-governor"
TEMPLATE = "adapter-request.yml"


def contribute(repo, what):
    print(f"\n  No adapter ships for {what}.")
    print("  That is a gap worth recording rather than a reason to stop -- the")
    print("  provider abstraction exists so a tracker can be added without")
    print("  touching the engine (ADR-003).\n")
    # The form asks what an adapter must be able to answer -- above all what
    # ADMITTED means in that system, which is the one thing detection can never
    # see (ADR-018). Someone who can answer it has most of the design already.
    print("  File it (a form asks the questions an adapter has to answer):")
    print(f"    https://github.com/{REPO}/issues/new?template={TEMPLATE}")
    print("\n  Or from the terminal:")
    print(f"    gh issue create --repo {REPO} --template {TEMPLATE}")
    print("\n  Or write one: adapters/_protocol.py is the contract, and every")
    print("  shipped adapter is a single file. Contribution guide:")
    # A URL, not a filename. install-skill.sh prunes CONTRIBUTING.md, so naming
    # the file would point at something the install does not carry -- and where
    # a host repository has its own, it would name the wrong one.
    print(f"    https://github.com/{REPO}/blob/main/CONTRIBUTING.md")



def _survey(rid):
    """Show which admission signals actually EXIST, with the operator's consent.

    Detection must never probe a remote (ADR-010 rule 4): a probe that succeeds
    because a token happens to be present is capability implying permission.
    This is a different act. A person is asking to look at their own repository,
    the result only informs THEIR choice, and nothing is bound or inferred from
    it -- the answer to the next question is still theirs to give.

    It exists because four consecutive onboarding attempts declared
    project_status on a repository with zero Projects. Every id then read
    NOT_ON_BOARD while validation stayed green. Asking someone to declare a
    signal without showing them which signals exist is a question with a trap in
    it.
    """
    if input(f"\nLook up which admission signals exist in {rid}? "
             "Runs `gh`, reads counts only, changes nothing. [y/N] ").strip().lower() \
            not in ("y", "yes"):
        return None
    try:
        # state=all, NOT the default. GitHub's milestones endpoint defaults to
        # state=open, and this survey WITHHOLDS an option whose count is zero --
        # `ask` refuses it if chosen. So a repository whose milestones are all
        # closed reported "none" and the operator was refused milestone
        # admission on a repository where it demonstrably works: the engine
        # reads `milestone` without consulting its state, so a closed milestone
        # confers admission today (issue 221). Measured on this repository while
        # writing this: 4 open, 6 closed, 10 total.
        ms = json.loads(subprocess.run(
            ["gh", "api", f"repos/{rid}/milestones?state=all", "--jq",
             "[.[] | {t:.title, o:.open_issues, c:.closed_issues, s:.state}]"],
            capture_output=True, text=True, timeout=25).stdout or "[]")
        pj = json.loads(subprocess.run(
            ["gh", "api", "graphql", "-f", "query=query{repository(owner:\"%s\","
             "name:\"%s\"){projectsV2(first:10){totalCount}}}" % tuple(rid.split("/", 1)),
             "--jq", ".data.repository.projectsV2.totalCount"],
            capture_output=True, text=True, timeout=25).stdout.strip() or "0")
        lb = json.loads(subprocess.run(
            ["gh", "api", f"repos/{rid}/labels", "--jq", "length"],
            capture_output=True, text=True, timeout=25).stdout.strip() or "0")
    except Exception as e:
        print(f"  lookup failed ({e}); choose from what you know.")
        return None

    print()
    if ms:
        names = ", ".join(m["t"] for m in ms[:4])
        live = sum(m["o"] for m in ms)
        opened = sum(1 for m in ms if m.get("s") == "open")
        print(f"  milestones : {len(ms)} ({opened} open, {len(ms) - opened} closed)"
              f" ({names}) -- {live} open issues carry one")
        if not opened:
            # Not withheld: admission does not consult milestone state today, so
            # this choice works. Said out loud because it is the one case where
            # what works now and what ADR-018 says are being reconciled (221).
            print("               every milestone is CLOSED. This still reads as"
                  " admitted today, because the signal does not check"
                  " state -- see issue 221.")
    else:
        print("  milestones : none  <-- option 1 cannot work here")
    print(f"  Projects   : {pj}" + ("" if pj else "  <-- option 2 cannot work here"))
    print(f"  labels     : {lb}" + (""    if lb else "  <-- option 3 cannot work here"))
    print("\n  Existing is not the same as MEANING admitted. A repository can have"
          "\n  milestones that are release buckets rather than admission. Only you"
          "\n  know which. This just rules out the ones that cannot work.")
    return {"milestone": len(ms), "project_status": int(pj), "label": int(lb)}



LEVELS = [("L0", "GOVERNOR_GREENFIELD"), ("L1", "GOVERNOR_LITE"),
          ("L2", "GOVERNOR_STANDARD"), ("L3", "GOVERNOR_FULL"),
          ("L4", "GOVERNOR_HIGH_ASSURANCE")]


def _confirm_condition(level, profile, floors, ind):
    """Show the evidence, then let a person set the level. The script never decides.

    assess() is deliberately mechanical (ADR-002) and therefore crude: a floor
    fires on `"export "` appearing in any of 60 source files, which is true of
    essentially every TypeScript repository. Its own docstring says it reports a
    SUGGESTED level and a human decides -- but nothing in the flow ever asked.

    Printing the indicators is the point. A person can weigh them, and so can an
    agent running this on someone's behalf; neither can weigh a bare verdict.
    """
    print(f"\nCondition assessed: {level} / {profile}")
    if floors:
        print(f"  floor raised by : {', '.join(floors)}")
    interesting = [k for k, v in ind.items() if v not in (False, 0, None)]
    print(f"  indicators      : {', '.join(f'{k}={ind[k]}' for k in interesting) or 'none'}")
    if floors:
        print("\n  A floor means other people depend on this repository's shapes, so")
        print("  compatibility obligations exist. It may be RAISED but never lowered.")
    print("\n  Governance depth follows this (ADR-006): an L1 repository never loads")
    print("  L4 policy. Too high is friction; too low is a repository governed more")
    print("  loosely than its obligations warrant.")

    names = [l for l, _ in LEVELS]
    floor_at = names.index(level) if floors else 0
    allowed = names[floor_at:]
    if len(allowed) == 1:
        # The floor is already at the maximum. Offering a choice of one is not a
        # choice; say why there is nothing to decide and move on.
        print(f"\n  {level} is the highest level and the floor is already there,")
        print("  so there is nothing to choose. Recorded as assessed.")
        return level, profile
    print(f"\n  Accept {level}, or choose one of: {', '.join(allowed)}")
    raw = input(f"  [{level}] > ").strip().upper()
    if not raw:
        return level, profile
    if raw not in allowed:
        if raw in names:
            print(f"  {raw} is below the floor and cannot be set. Keeping {level}.")
        else:
            print(f"  Not a level. Keeping {level}.")
        return level, profile
    return raw, dict(LEVELS)[raw]


# Roles a proposal can bind without asking anyone to install anything. This
# table exists so the sweep is checkable rather than remembered.
#
# ISSUE 94 HAPPENED THREE TIMES. decision_history went unbound in every
# generated manifest because nothing proposed it, and CAPTURE_ONLY -- the
# default disposition -- had nowhere to record. That was fixed for one role and
# the class was never swept, so acceptance_criteria and architecture repeated
# it. The acceptance_criteria case is the worst of the three: an unbound role
# is refused at the binding layer with PERMISSION_DENIED, which completion.py
# turns into a BLOCKING UNKNOWN -- so STOP_COMPLETE was not merely unreachable,
# it failed as "something is broken" rather than as the honest CONTINUE with
# NO_CRITERIA_DECLARED that adapters/acceptance-file was built to give.
#
# `detect` names the role whose detection gates the entry, or None for roles
# that need no evidence because the adapter reads a directory this tool creates.
ZERO_INSTALL = {
    "acceptance_criteria": {"type": "acceptance-file",
                            "adapter": "adapters/acceptance-file",
                            "detect": None, "multi": False},
    "decision_history": {"type": "decision-history-file",
                         "adapter": "adapters/decision-history-file",
                         "detect": None, "multi": True},
    "architecture": {"type": "adr", "adapter": "adapters/adr",
                     "detect": "architecture", "multi": True},
}


# Which candidate fields become manifest fields, and which are detection's own
# bookkeeping. The partition is EXHAUSTIVE by assertion (conformance/onboarding
# .py): a field detection starts emitting must be classified here, or the check
# fails. Issue 199 was one field -- `path` -- silently absent from this list
# because there was no list: detection computed `docs/adr`, the evidence
# document recorded it, and the manifest writer read the candidate only as a
# boolean. The adapter then defaulted to `docs/adrs`, found nothing, and
# reported TRANSPORT_UNREACHABLE, which is what a broken adapter looks like.
CARRIED = ("type", "adapter", "path")
NOT_CARRIED = ("role", "disposition", "evidence", "not_evidence")


def _entry(cand, spec):
    """One provider entry, built from what detection FOUND, not from the table.

    The table names an adapter per role. That is right for the roles nothing
    detects -- their adapter reads a directory this tool creates -- and wrong
    for a detected role, where the table's answer and the evidence can differ:
    a repository carrying only `openspec/` was proposed `adapters/adr`, because
    the entry was assembled from a constant while the candidate that triggered
    it sat unread.
    """
    if cand is None:
        return {"type": spec["type"], "adapter": spec["adapter"], "contract_version": 1}
    out = {"type": cand.get("type", spec["type"]),
           "adapter": cand.get("adapter") or spec["adapter"],
           "contract_version": 1}
    for k in CARRIED:
        if k not in out and cand.get(k) is not None:
            out[k] = cand[k]
    return out


def build_providers(choice, adapter, rid, admission, label, candidates):
    """The providers block, extracted so a test can read it without a terminal.

    A role appears here when a zero-install adapter can serve it AND, where the
    table says so, detection actually found it. Roles with neither -- execution,
    change_signals, retirement -- stay absent, which is INV-013 working rather
    than the same defect: a role nobody bound has no governance function.

    Detection proposes; only an accepted manifest binds (ADR-010 rule 1). But
    the proposal has to contain the thing for a human to accept it, and that is
    what was missing -- first the role itself, then (issue 199) the fields that
    make the role's binding work.
    """
    prov = {
        "repository": {"type": "git", "adapter": "adapters/git", "contract_version": 1},
        "roadmap_authority": {"type": choice, "adapter": adapter, "contract_version": 1},
    }

    for role, spec in ZERO_INSTALL.items():
        found = (candidates or {}).get(spec["detect"]) or [] if spec["detect"] else []
        if spec["detect"] and not found:
            continue
        # PROVIDER_DETECTED only. An UNCONFIRMED candidate is evidence that
        # something MIGHT be there -- `docs/decisions` holding zero readable
        # files is the common case -- and binding it produces a provider that
        # answers nothing. Detection already made that distinction; this is the
        # first place it is honoured.
        strong = [c for c in found if c.get("disposition") == "PROVIDER_DETECTED"]
        if spec["detect"] and not strong:
            continue
        if not spec["detect"]:
            prov[role] = [_entry(None, spec)] if spec["multi"] else _entry(None, spec)
        elif spec["multi"]:
            prov[role] = [_entry(c, spec) for c in strong]
        else:
            prov[role] = _entry(strong[0], spec)

    if admission:
        prov["roadmap_authority"]["admission"] = {"signal": admission}
    if choice == "github-projects":
        prov["roadmap_authority"]["transport"] = {"kind": "cli", "command": "gh"}
        # The adapter reads its configuration from this env block, not from the
        # fields above. Omitting it produced a manifest that validated as
        # READY_FOR_GOVERNANCE and then answered PROVIDER_UNAVAILABLE for every
        # id: "no repository declared ... an adapter that cannot tell which
        # repository it is reading must not answer" (ADR-028).
        env = {"REPO_GOVERNOR_GH_REPO": rid}
        if admission:
            env["REPO_GOVERNOR_GH_ADMISSION"] = admission
        if admission == "label":
            if not label:
                raise ValueError("A label signal needs the label named; it is never "
                                 "assumed (ADR-018).")
            env["REPO_GOVERNOR_GH_ADMISSION_LABEL"] = label
        prov["roadmap_authority"]["env"] = env
    return prov


def _assessed(repo: Path):
    """(level, profile, floors, indicators, candidates) from engine/onboard.py.

    Candidates come back from the SAME call rather than a second one. Detection
    already reports which roles it found and with what evidence; the proposal
    used to discard all of it except the condition, which is why a detected
    architecture provider was printed as PROVIDER_DETECTED and then left out of
    the manifest (issue 144).
    """
    r = subprocess.run(
        [sys.executable, str(SKILL / "engine" / "onboard.py"), str(repo), "--json"],
        capture_output=True, text=True)
    try:
        d = json.loads(r.stdout)
        c = d["condition"]
        return (c["suggested"], c["profile"], c.get("floor") or [],
                c.get("indicators") or {}, d.get("candidates") or {})
    except Exception:
        # No silent fallback to a level. A condition nobody assessed is not a
        # condition, and guessing one is the defect this function replaced.
        print("Could not read the assessed condition from engine/onboard.py.",
              file=sys.stderr)
        raise SystemExit(1)


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    repo = Path(argv[0]).resolve()
    if not (repo / ".git").exists():
        print(f"not a git repository: {repo}", file=sys.stderr)
        return 1
    if (repo / ".repo-governor.json").exists():
        print(f"{repo} is already onboarded. Refusing to overwrite a binding manifest.",
              file=sys.stderr)
        return 1

    print(f"Onboarding proposal for {repo}")
    print("This writes a PROPOSAL. Nothing governs until you review and rename it.")

    # Detection first: it answers what it can, and its evidence is worth seeing.
    print("\n--- detected ---")
    subprocess.run([sys.executable, str(SKILL / "engine" / "onboard.py"), str(repo)])

    rid = repo_id(repo)
    if not rid:
        print("\nCould not read owner/repo from the git remote.")
        rid = input("Repository id (owner/repo): ").strip()
        if not rid:
            print("A repository id is required; it is never defaulted (ADR-028).",
                  file=sys.stderr)
            return 1
    else:
        print(f"\nRepository id, from the git remote: {rid}")

    choice = ask("Which system is the ROADMAP AUTHORITY -- the one that says what "
                 "work is admitted?", [(v, d) for v, _, d in ROADMAP])
    if choice is None:
        contribute(repo, "roadmap_authority")
        return 1
    adapter = next(a for v, a, _ in ROADMAP if v == choice)

    # BOTH initialised before the branch, and for the same reason. `admission`
    # already was; `label` was not, and it is passed to build_providers()
    # unconditionally -- so choosing Linear or file-roadmap raised
    # UnboundLocalError and the tool could onboard a GitHub repository and
    # nothing else. Shipped in #145 and found by an external user, because no
    # test had ever driven onboarding with a non-GitHub roadmap: every
    # assertion in conformance/onboarding.py called build_providers() with
    # "github-projects", including the ones added for #199.
    admission = None
    label = ""
    if choice == "github-projects":
        counts = _survey(rid)
        # Do NOT offer a signal that provably cannot work. The survey printed
        # "Projects: 0 <-- option 2 cannot work here" directly above this
        # question and the option was chosen anyway, five times running. A
        # warning the reader must notice is not a control -- which is this
        # project's own thesis, applied to its own tooling.
        opts, unavail = GH_SIGNALS, {}
        if counts:
            opts = [(v, f"{d}   [{counts[v]} exist]" if counts.get(v) else d)
                    for v, d in GH_SIGNALS]
            unavail = {v: "none exist in this repository"
                       for v, _ in GH_SIGNALS if v != "none" and not counts.get(v)}
        admission = ask("What does ADMITTED mean in that repository? Detection cannot "
                        "see this, and guessing it is how a second roadmap of record "
                        "gets created (ADR-018).", opts, allow_other=False,
                        unavailable=unavail)
        # Asked HERE, while the choice is on screen. It used to be asked after
        # the condition question, three screens later, where it read as a
        # continuation of something else.
        if admission == "label":
            label = input("  Which label means admitted? ").strip()

    # Take the ASSESSED condition, never a default. This wrote "L1" /
    # GOVERNOR_LITE unconditionally, with a comment telling the reader to fix it
    # by hand -- on a repository detection had assessed L4 because three floor
    # indicators fired. The engine's own rule is that a floor "may not be
    # overridden downward", so the tool was committing the violation it printed
    # a warning about two screens earlier. The profile decides governance depth
    # (ADR-006); writing L1 over an L4 under-governs a repository that has
    # compatibility obligations.
    level, profile, floors, ind, candidates = _assessed(repo)
    level, profile = _confirm_condition(level, profile, floors, ind)
    try:
        prov = build_providers(choice, adapter, rid, admission, label, candidates)
    except ValueError as e:
        print(str(e), file=sys.stderr)
        return 1

    # Deny by default (ADR-005). Every bound role gets read; exactly one gets
    # write, and the reason is stated at the top of the block rather than
    # beside the entry -- the schema closes `permission` to a verb set, so a
    # per-entry comment would not validate, and a reviewer should meet the
    # escalation before the JSON that contains it rather than after.
    perms = {"$comment": (
        "Deny by default (ADR-005). Read-only, with ONE exception stated here so it "
        "is not an escalation buried in generated JSON: decision_history has "
        "write=true. CAPTURE_ONLY is the default disposition for every discovery, and "
        "`envelope.py --record` cannot persist one without it. Recording a decision is "
        "the only act this grant permits; the engine still rules and never changes the "
        "repository. Remove it and captures are refused, which is a defensible choice "
        "-- it just means nothing remembers what was decided.")}
    perms.update({role: {"read": True, "write": False} for role in prov})
    perms["decision_history"] = {"read": True, "write": True}

    out = {
        "$comment": ("PROPOSAL in manifest shape. Review every line, then rename to "
                     ".repo-governor.json and commit. The engine never reads this file "
                     "(ADR-010 rule 1). Roles here are the ones a zero-install adapter can "
                     "serve, plus any detection actually found. Roles absent -- execution, "
                     "change_signals, retirement -- have no governance function here however "
                     "reachable the system is (INV-013); add them as you bind them."),
        "repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
        "repository": {"id": rid},
        "condition": {
            "assessed": level, "profile": profile,
            "$comment": ("Assessed by engine/onboard.py, not defaulted."
                         + (f" Floor indicators present ({', '.join(floors)}); per the"
                            " engine's rule this may not be overridden downward."
                            if floors else
                            " No floor indicator; a human may raise it if the"
                            " repository's obligations exceed what is observable.")),
        },
        "permissions": perms,
        "providers": prov,
    }
    # Where the repository carries compatibility obligations, say what the
    # default store does NOT guarantee. Capability language, not a product
    # name: ADR-003 keeps adapter knowledge out of the engine and ADR-030 is
    # Proposed with four unmet conditions, the first being whether level
    # predicts this need at all. So this is a note a person reads, never a
    # rule that changes a binding, and nothing below is conditional on it.
    if floors:
        print(f"\n  This repository floors at {level} ({', '.join(floors)}), meaning other")
        print("  people depend on its shapes. The proposed decision store hand-rolls its")
        print("  chain: a rewrite is DETECTABLE there, not prevented, and it names no")
        print("  committer. A backend that supplies history natively gives a stronger")
        print("  guarantee. references/integrations.md lists what one must satisfy.")
        print("  Nothing here is blocked, and the binding above is unchanged.")

    p = repo / PROPOSAL
    p.write_text(json.dumps(out, indent=2) + "\n", encoding="utf-8")
    print(f"\nwrote {p}")
    print("\nVerify it before binding:")
    # The store is created by an explicit step, not by this tool. Same pattern
    # as tools/bootstrap-decisions.sh for Dolt, whose probe also requires the
    # database to exist: a store nobody asked for is still a change to the
    # repository (ADR-005), and this tool writes a proposal, not a store.
    # Without it --validate reports TRANSPORT_UNREACHABLE and
    # WRITE_GRANTED_BUT_TRANSPORT_READONLY, both correct and both confusing on
    # a freshly generated proposal.
    # One line per store, both for the same reason: the adapter's probe checks
    # the directory exists, so a proposal that binds the role without it
    # validates as TRANSPORT_UNREACHABLE -- correct, and confusing on a freshly
    # generated manifest.
    print(f"  mkdir -p {repo / '.repo-governor' / 'decisions'}   # the decision store")
    print(f"  mkdir -p {repo / '.repo-governor' / 'acceptance'}  # where completion bars live")
    # And say how to write one. This printed the mkdir and stopped, which told
    # an author where bars live and nothing about what goes in them -- a
    # governed repository then authored criteria with 'command' and
    # 'description' keys that are never read, and the engine crashed on the
    # missing 'target' instead of returning a disposition (issues 213, 214).
    # The scaffold has shown the correct shape all along; nothing pointed at it.
    print(f"\n  # A bar is {{check, target}} per criterion -- schemas/acceptance-v1.json.")
    print(f"  # Do not hand-write the first one; scaffold it and edit:")
    print(f"  python3 {SKILL / 'engine' / 'acceptance.py'} <id> --template")
    print(f"  cd {repo} && mv {PROPOSAL} .repo-governor.json")
    print(f"  python3 {SKILL / 'engine' / 'manifest.py'} --validate")
    print("\nIf validation fails, rename it back. Nothing governs until it passes.")
    # --validate does NOT catch a wrong admission signal, or a binding that
    # cannot answer at all (issue 51). Only asking a real question does. Three
    # separate onboarding attempts declared project_status on a repository with
    # no Project; every id then read NOT_ON_BOARD, and validation stayed green
    # throughout.
    print("\nTHEN ASK IT A REAL QUESTION -- validation does not catch a wrong signal:")
    print(f"  python3 {SKILL / 'engine' / 'completion.py'} <an open issue number>")
    print("\n  A real verdict (EXECUTE, CONTINUE, NO_EXECUTION_AUTHORITY,")
    print("  STOP_COMPLETE) means the binding works.")
    print("  UNKNOWN with the SAME reason for every id you try means the")
    print("  admission signal is probably wrong -- NOT_ON_BOARD when there is no")
    print("  Project, NOT_ADMITTED for everything when nothing carries the")
    print("  milestone or label you named. Re-run this tool and pick again.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
