#!/usr/bin/env python3
"""Amendment integrity for acceptance criteria (ADR-017 open weakness).

The completion firewall has an obvious defeat: amend the criteria until the
work passes. ADR-017 named this as a known negative and admitted there was
"no mechanism that distinguishes a legitimate correction from a convenient
one". This is that mechanism, or as much of one as is actually checkable.

Three rules, all mechanical:

  1. An amendment must cite something VERIFIABLE. Free prose is
     unfalsifiable; "§62 permits a synthetic provider" can be checked
     against §62. Three citation forms resolve:

         §NN / ADR-NNN / INV-NNN   a specification section that exists
         path/to/file              an artifact in the repository
         <commit-sha>              a commit that resolves

     Not every legitimate amendment has a spec basis — GATE-1's came from
     discovering that committed fixtures leaked the parent repository's git
     remote, which is an implementation finding, not a reading. Artifact and
     commit citations exist so honest corrections of that kind are
     expressible without weakening the rule to "write something".

  2. A LOOSENING amendment — one that removes or weakens a criterion — is
     held to a higher bar than a tightening one, and is always reported.
     Adding criteria needs no defence; removing them does.

  3. Amendment history is append-only and counted. Repeated amendment of
     the same gate is visible even when each one is individually defensible,
     because the pattern is the signal.

What this deliberately does NOT do: judge whether the cited section actually
supports the amendment. That is a reading, and ADR-002 keeps readings out of
the engine. It makes the claim checkable and the pattern visible; a human
still decides whether the argument is good.

Usage:  python3 engine/amendments.py [--audit]
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ACCEPTANCE = ROOT / ".repo-governor" / "acceptance"
REFERENCE = ROOT / "docs" / "reference"
ADRS = ROOT / "docs" / "adrs"

SECTION_RE = re.compile(r"§(\d{1,2})")
ADR_RE = re.compile(r"ADR-(\d{3})")
INV_RE = re.compile(r"INV-(\d{3})")

LOOSENING = ("removed", "weakened")
TIGHTENING = ("added", "strengthened")
EFFECTS = LOOSENING + TIGHTENING + ("corrected",)


def known_sections():
    found = set()
    for p in REFERENCE.glob("*.md"):
        for m in re.findall(r"^## §(\d{1,2})", p.read_text(errors="ignore"), re.M):
            found.add(int(m))
    return found


def known_adrs():
    return {int(p.name[:3]) for p in ADRS.glob("[0-9]*.md")}


def known_invariants():
    text = (REFERENCE / "invariants.md").read_text(errors="ignore")
    return {int(m) for m in re.findall(r"INV-(\d{3})", text)}


def resolve(cite, secs, adrs, invs):
    """Does this citation point at something that exists?"""
    import subprocess
    m = SECTION_RE.fullmatch(cite) or SECTION_RE.match(cite)
    if m and cite.startswith("§"):
        n = int(m.group(1))
        return (n in secs, f"§{n}" + ("" if n in secs else " does not exist"))
    m = ADR_RE.match(cite)
    if m:
        n = int(m.group(1))
        return (n in adrs, f"ADR-{n:03d}" + ("" if n in adrs else " does not exist"))
    m = INV_RE.match(cite)
    if m:
        n = int(m.group(1))
        return (n in invs, f"INV-{n:03d}" + ("" if n in invs else " does not exist"))
    # A repository artifact.
    if "/" in cite or "." in cite:
        if (ROOT / cite).exists():
            return (True, cite)
        return (False, f"artifact {cite!r} does not exist in the repository")
    # A commit.
    if re.fullmatch(r"[0-9a-f]{7,40}", cite):
        r = subprocess.run(["git", "-C", str(ROOT), "cat-file", "-e", cite + "^{commit}"],
                           capture_output=True)
        if r.returncode == 0:
            return (True, f"commit {cite}")
        return (False, f"commit {cite!r} does not resolve")
    return (False, f"{cite!r} is not a recognisable citation "
                   "(expected §NN, ADR-NNN, INV-NNN, a repo path, or a commit sha)")


def audit():
    """Return (findings, records). A finding is a rule violation."""
    secs, adrs, invs = known_sections(), known_adrs(), known_invariants()
    findings, records = [], []

    for p in sorted(ACCEPTANCE.glob("*.json")):
        d = json.loads(p.read_text())
        wid = d.get("authority_id", p.stem)
        amendments = d.get("amendments")

        # Legacy shape: a single free-prose amendment_reason with no citation.
        if amendments is None and (d.get("amended") or d.get("amendment_reason")):
            findings.append((wid, "UNSTRUCTURED_AMENDMENT",
                             "amended with free prose and no citation; cannot be checked"))
            records.append((wid, 1, "legacy", []))
            continue

        if not amendments:
            records.append((wid, 0, "-", []))
            continue

        for i, a in enumerate(amendments):
            where = f"{wid} amendment {i + 1}"
            effect = a.get("effect")
            if effect not in EFFECTS:
                findings.append((wid, "BAD_EFFECT",
                                 f"{where}: effect {effect!r} not in {list(EFFECTS)}"))
            cites = a.get("cites") or []
            if not cites:
                findings.append((wid, "NO_CITATION",
                                 f"{where}: no citation. Free prose is unfalsifiable."))
            for c in cites:
                ok, detail = resolve(c, secs, adrs, invs)
                if not ok:
                    findings.append((wid, "UNRESOLVABLE_CITATION", f"{where}: {detail}"))
            if effect in LOOSENING and not cites:
                findings.append((wid, "UNJUSTIFIED_LOOSENING",
                                 f"{where}: removes or weakens a criterion with no citation"))
        loosenings = [a for a in amendments if a.get("effect") in LOOSENING]
        records.append((wid, len(amendments),
                        "loosening" if loosenings else "tightening",
                        [c for a in amendments for c in (a.get("cites") or [])]))

    return findings, records


def main(argv):
    findings, records = audit()

    print("Acceptance-criteria amendment audit\n")
    print(f"  {'work item':<12} {'amendments':>10}  {'direction':<11} citations")
    for wid, n, direction, cites in records:
        flag = "  <-- loosening" if direction == "loosening" else ""
        print(f"  {wid:<12} {n:>10}  {direction:<11} {', '.join(cites) or '-'}{flag}")

    if findings:
        print(f"\n{len(findings)} finding(s):\n")
        for wid, kind, detail in findings:
            print(f"  [{kind}] {detail}")
    else:
        print("\nno findings")

    total_amended = sum(1 for _, n, _, _ in records if n)
    print(f"\n{total_amended}/{len(records)} acceptance artifacts have been amended")
    print("AMENDMENT INTEGRITY: " + ("OK" if not findings else f"VIOLATIONS ({len(findings)})"))
    return 0 if not findings else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
