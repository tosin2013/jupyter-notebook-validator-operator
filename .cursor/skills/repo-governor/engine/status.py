#!/usr/bin/env python3
"""What can governance actually answer in this repository, and what can it not?

There was no way to ask. `manifest.py --validate` answers a narrower question --
do the bound adapters satisfy their declared contracts -- and returns
READY_FOR_GOVERNANCE for a binding that is correctly formed and cannot answer
anything useful (issue 51). Issue 57.

The gap is bigger than validation, because THE ROLES THAT MAKE GOVERNANCE BITE
ARE THE ONES ONBOARDING DOES NOT BIND. `engine/onboard.py` detects four of the
eight roles the schema defines. A repository can be onboarded, validate green,
and still have no decision_history -- so CAPTURE_ONLY, the default disposition,
has nowhere to record -- and no acceptance criteria -- so STOP_COMPLETE is
unreachable and every verdict is CONTINUE, forever, non-blocking.

Every one of those is the system behaving correctly. None of them is visible.

FOUR THINGS THIS IS NOT:

  Not a score. A number invites optimising the number. This prints facts and
  lets a person weigh them, the same reasoning _confirm_condition uses for
  condition indicators.

  Not a second source of truth. Everything is derived from the manifest and
  from live `describe` calls. Nothing cached, nothing restated. This repository
  has recorded three separate times that a duplicated derivable fact eventually
  disagrees with its source.

  Not silently green. If it can inspect nothing it says so and exits non-zero,
  rather than printing an empty, healthy-looking report.

  Not a gate. It reports. Nothing keys off its output, and it decides nothing.

Usage:  python3 engine/status.py [repo-path]
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "engine"))

import bindings as B  # noqa: E402
import manifest as MF  # noqa: E402
import vocabulary as V  # noqa: E402

SCHEMA = ROOT / "schemas" / "manifest-v1.json"


def _architecture_detail(role, described):
    """What the architecture role actually holds, and what it does with it.

    A repository can bind thirty ADRs and see only `bound adapters/adr answers`,
    which says nothing about whether they were read or what they mean. That
    matters more now that onboarding proposes this role when it detects ADRs
    (issue 144): people will bind something whose value was invisible.

    NOT A SCORE. `engine/status.py` refuses those, and the reason applies with
    unusual force here: a Proposed ADR is not worse than an Accepted one.
    ADR-031 is correctly Proposed pending a measurement on a host this
    project does not own. (This named ADR-024 until 2026-08-31, when 024 was
    accepted -- the argument is unchanged and needed a live example, because an
    example that has since been accepted quietly makes the point read as
    historical.) A number would read that as debt and create pressure
    to accept decisions to raise it -- admission without authority, applied to
    an architecture ledger. Counts and named states carry the same information
    and cannot be optimised.

    The last line is the honest part. The role answers "what constrains how it
    must be built?" and no disposition consults the answer; it moves one
    thinness dimension in the compiled envelope and nothing else (issue 143).
    Saying so where an operator reads it beats leaving the binding to imply
    otherwise.
    """
    if role != "architecture" or not described:
        return []
    out = []
    con = B.call("architecture", "get_constraints", {"id": "-"})
    if not con.get("ok"):
        return [f"constraints unreadable: {(con.get('error') or {}).get('type')}"]
    state = (con.get("value") or {}).get("state")
    counts = {}
    act = B.call("architecture", "get_active_decisions", {})
    if act.get("ok"):
        for dec in (act.get("value") or {}).get("decisions") or []:
            word = (dec.get("status") or "?").split()[0]
            counts[word] = counts.get(word, 0) + 1
    sup = B.call("architecture", "get_superseded_decisions", {})
    if sup.get("ok"):
        n = (sup.get("value") or {}).get("count") or 0
        if n:
            counts["Superseded"] = n
    tally = ", ".join(f"{v} {k}" for k, v in sorted(counts.items())) or "no decisions read"
    out.append(f"{state} — {tally}")
    if con.get("unknown"):
        out.append(f"{con['unknown'].get('reason')}: {con['unknown'].get('detail', '')[:70]}")
    out.append("reported only; no disposition consults this")
    return out


def roles():
    """The eight roles, from the schema. Never a list restated here."""
    d = json.loads(SCHEMA.read_text(encoding="utf-8"))

    def find(o):
        if isinstance(o, dict):
            props = o.get("properties", {})
            if "providers" in props:
                return sorted(k for k in props["providers"].get("properties", {})
                              if not k.startswith("$"))
            for v in o.values():
                r = find(v)
                if r:
                    return r
        elif isinstance(o, list):
            for v in o:
                r = find(v)
                if r:
                    return r
        return None

    return find(d) or []


def describe(binding):
    try:
        return B.describe(binding, use_cache=False) or {}
    except Exception:
        return {}


def main(argv):
    target = Path(argv[0]).resolve() if argv else Path.cwd()

    # Declare the target so the manifest load and every adapter spawn agree on
    # which repository is being reported. `MF.load()` resolves through
    # `bindings.target()`, and `describe` spawns with `cwd=target()`; setting
    # only the argument would read one repository's manifest while probing
    # another's providers -- which is ADR-027's defect, reintroduced by a
    # report that looked like it took a path. Declared here, never inherited
    # (see issue 54 for the inherited case).
    os.environ["REPO_GOVERNOR_TARGET"] = str(target)

    m, errs = MF.load()
    if errs:
        print(f"MANIFEST INVALID ({len(errs)} error(s)) — nothing can be reported from it\n")
        for e in errs:
            print(f"  {e}")
        # Not silently green: an unreadable manifest is a status, and a
        # failing one.
        return 1

    all_roles = roles()
    if not all_roles:
        print("Could not read the role set from the schema. Refusing to report a "
              "partial picture as a complete one.", file=sys.stderr)
        return 1

    prov = {k: v for k, v in (m.get("providers") or {}).items() if not k.startswith("$")}

    print(f"Repo Governor — status")
    print(f"  repository : {m['repository']['id']}")
    print(f"  condition  : {m['condition']['assessed']} / {m['condition']['profile']}")
    print(f"  standing in: {target}")

    # ADR-006 line 18: the profile "determines which policy packs load and
    # which providers are required". The second half went unimplemented, so an
    # L4 repository could bind two of the five roles its profile demands and
    # hear nothing (issue 79). Unbound and unbound-but-required are different
    # facts and a reader needs to tell them apart.
    need = set(V.required_roles(m["condition"]["profile"]))
    print(f"\nROLES ({len(prov)} of {len(all_roles)} bound)\n")
    caps = {}
    for role in all_roles:
        b = prov.get(role)
        if not b:
            # INV-013: an unbound provider has no governance function. That is
            # an expected state and a reportable one, not an error.
            if role in need:
                print(f"  {role:<20} UNBOUND   — REQUIRED by {m['condition']['profile']} "
                      f"and not bound")
            else:
                print(f"  {role:<20} UNBOUND   — no governance function here (INV-013)")
            continue
        for one in (b if isinstance(b, list) else [b]):
            d = describe(one)
            reach = (d.get("transport") or {}).get("reachable")
            adv = {k for k, v in (d.get("capabilities") or {}).items() if v}
            caps.setdefault(role, set()).update(adv)
            state = "answers" if d and reach is not False else (
                "UNREACHABLE — advertises nothing" if d else "no parseable describe")
            print(f"  {role:<20} bound     {one['adapter']:<34} {state}")
            declared = {k for k, v in (d.get("capabilities") or {}).items() if v is False}
            if declared:
                print(f"  {'':<20}           cannot supply: {', '.join(sorted(declared))}")
            for line in _architecture_detail(role, d):
                print(f"  {'':<20}           {line}")

    # What follows is the useful half: not what is configured, but what can be
    # CONCLUDED. A disposition nothing can reach is a governance behaviour this
    # repository does not have, however green its manifest is.
    print("\nWHAT THIS REPOSITORY CAN CONCLUDE\n")
    ra = caps.get("roadmap_authority", set())
    verdicts = []

    verdicts.append(("CONTINUE", bool(prov.get("roadmap_authority")),
                     "needs a roadmap authority"))
    verdicts.append(("NO_EXECUTION_AUTHORITY", "authority" in ra,
                     "needs roadmap_authority to advertise 'authority'"))
    verdicts.append(("AUTHORITY_WITHDRAWN", "cancellation_detection" in ra,
                     "needs roadmap_authority to advertise 'cancellation_detection'"))

    acc_dir = target / ".repo-governor" / "acceptance"
    crit = sorted(p.stem for p in acc_dir.glob("*.json")) if acc_dir.is_dir() else []
    verdicts.append(("STOP_COMPLETE", bool(crit),
                     "no acceptance criteria declared anywhere — the completion firewall "
                     "(§40) cannot fire, and every verdict stays CONTINUE"))

    # CONFLICT was reported reachable whenever any role was multi-bound. It is
    # not reachable at all, from any manifest: no path in engine/ emits it as a
    # disposition, and a validated manifest cannot even hold the case it was
    # meant for -- ADR-013 rule 1 makes authority roles single-valued and
    # manifest.py rejects a list where a scalar is required, so two peer roadmap
    # authorities are a load-time CARDINALITY error, not a runtime verdict.
    # Disagreement between peers of a MULTI-valued role is ARCHITECTURE_REVIEW
    # (rule 3), which is a different word on purpose.
    #
    # This is the same defect the architecture caveat fixed in issue 143: a
    # surface claiming a capability the engine does not have. It was wrong in
    # both directions -- claiming the one nothing emits, omitting the one that is.
    verdicts.append(("CONFLICT", False,
                     "no engine path emits it. Two peers on a single-valued role "
                     "is a manifest CARDINALITY error at load time (ADR-013 rule 1), "
                     "and peers of a multi-valued role produce ARCHITECTURE_REVIEW"))

    arch = prov.get("architecture")
    arch_n = len(arch) if isinstance(arch, list) else (1 if arch else 0)
    verdicts.append(("ARCHITECTURE_REVIEW", True,
                     "always reachable: it is the §32 review lane for an "
                     "ARCHITECTURE_IMPLICATION discovery"))
    verdicts.append(("ARCHITECTURE_REVIEW \u00d72", arch_n > 1,
                     f"needs two architecture providers to disagree; {arch_n} bound"))

    dh = prov.get("decision_history")
    dh_write = bool(dh) and MF.permitted(m, "decision_history", "write")[0]
    verdicts.append(("CAPTURE_ONLY, recorded", dh_write,
                     "decision_history is unbound or not granted write, so "
                     "`envelope.py --record` has nowhere to write"))

    for name, ok, why in verdicts:
        if ok:
            print(f"  {name:<24} reachable")
        else:
            print(f"  {name:<24} NOT reachable — {why}")

    # Obligation facts, reported and not judged. These gate nothing: a
    # repository without a licence still gets verdicts, still reaches every
    # disposition it otherwise would, and is told rather than stopped. Section
    # 54 fails this product if it "blocks routine reversible implementation
    # excessively", and a licence is not an authority question.
    print("\nOBLIGATIONS THIS REPOSITORY CARRIES\n")
    lic = any((target / n).is_file() for n in
              ("LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "LICENCE.md", "COPYING"))
    rdm = any((target / n).is_file() for n in
              ("README.md", "README", "README.rst", "README.txt"))
    if lic:
        print("  licence   present")
    else:
        print("  licence   ABSENT — with no licence a repository is all-rights-reserved by")
        print("            default: nobody has a grant to use it and no contributor has terms")
        print("            to contribute under. That may be deliberate for an internal")
        print("            repository. It does not block any work, and nothing here gates on it.")
    print(f"  readme    {'present' if rdm else 'absent — a documentation gap, not an obligation'}")

    print("\nLOCAL EVIDENCE\n")
    print(f"  acceptance criteria present for: {', '.join(crit) if crit else '(none)'}")
    # Say what cannot be answered, and why, rather than omitting the question.
    # No roadmap adapter exposes an enumeration function -- every one takes an
    # id -- so "admitted work WITHOUT criteria" is not computable here. That is
    # a missing provider capability, not an oversight in this report, and
    # printing nothing would hide it.
    print("  admitted work lacking criteria: NOT COMPUTABLE — no roadmap adapter")
    print("    advertises an enumeration function (every one takes an id), so the")
    print("    set of admitted work cannot be listed and this cannot be subtracted")
    print("    from it. Check a specific id with: engine/completion.py <id>")

    missing = sorted(r for r in need if r not in prov)
    if missing:
        print(f"\n  {m['condition']['profile']} requires {len(need)} role(s); "
              f"{len(missing)} not bound: {', '.join(missing)}")
        print("  That is a configuration gap, not a verdict. Nothing here refuses to")
        print("  answer because of it, and the assessed level is a human's to change.")

    unbound = [r for r in all_roles if r not in prov]
    if unbound:
        print(f"\n  {len(unbound)} role(s) unbound: {', '.join(unbound)}")
        print("  engine/onboard.py proposes only roadmap_authority, execution, repository")
        print("  and acceptance_criteria, so the rest are bound by hand or not at all.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
