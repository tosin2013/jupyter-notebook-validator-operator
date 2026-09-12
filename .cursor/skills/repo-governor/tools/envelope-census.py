#!/usr/bin/env python3
"""Measure whether real roadmap items carry the information a ScopeEnvelope needs.

Issue #2 asks how thin compiled envelopes are on real work items. Measuring what
the ADAPTER extracts would answer nothing: `github-projects` and `linear` both
return SCOPE_NOT_STRUCTURED / NON_GOALS_UNSTATED / ACCEPTANCE_UNSTATED
unconditionally, with no lookup at all. Running that over a thousand issues
returns 0% every time and restates the source code.

The question worth measuring is upstream of the adapter:

    Is the information PRESENT in real work items, in any form?

That distinguishes two very different problems:

  * absent from the source  -> thin envelopes are inherent to how teams write
                               tickets, and §55's "governance causes friction
                               without benefit" is live;
  * present but unparsed    -> an adapter capability gap, which is fixable.

Detection is deliberately GENEROUS -- prose phrases, not structured fields --
because the goal is an upper bound on what any adapter could ever extract. A
low number here is therefore strong evidence; a high number is only a hint.

Read-only. Prints counts and repository names, never issue content (§51).

Usage:  python3 tools/envelope-census.py <owner/repo> [<owner/repo> ...]
"""

from __future__ import annotations

import json
import re
import subprocess
import sys

# Generous on purpose: any of these phrases counts as the dimension being
# present, however informally. This measures the ceiling, not the adapter.
SIGNALS = {
    "non_goals": re.compile(r"non[- ]goals?|out of scope|not in scope|explicitly excluded", re.I),
    # Split deliberately. A task checklist is DECOMPOSITION, not a completion
    # bar -- conflating them inflated acceptance to 75% on the first run, almost
    # entirely from `- [ ]` boxes. The strong signal is a stated bar; the weak
    # one is a list of things to do.
    "acceptance_stated": re.compile(r"acceptance criteria|definition of done|done when|"
                                    r"success criteria", re.I),
    "checklist_only": re.compile(r"- \[[ x]\]", re.I),
    "in_scope": re.compile(r"in scope|scope:|deliverables?:", re.I),
}


def issues(nwo, limit):
    p = subprocess.run(["gh", "issue", "list", "--repo", nwo, "--state", "all",
                        "--limit", str(limit), "--json", "number,body"],
                       capture_output=True, text=True, timeout=180)
    if p.returncode != 0:
        return None
    try:
        return json.loads(p.stdout or "[]")
    except json.JSONDecodeError:
        return None


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    limit = 100
    repos = [a for a in argv if "/" in a]

    totals = {k: 0 for k in SIGNALS}
    totals["any"] = 0
    grand = 0
    rows = []

    for nwo in repos:
        data = issues(nwo, limit)
        if data is None:
            print(f"  {'?':>5}  {nwo}   (unreadable — skipped)")
            continue
        n = len(data)
        if not n:
            continue
        counts = {k: 0 for k in SIGNALS}
        any_n = 0
        for i in data:
            body = i.get("body") or ""
            hit = False
            for k, rx in SIGNALS.items():
                if rx.search(body):
                    counts[k] += 1
                    hit = True
            any_n += hit
        rows.append((nwo, n, counts, any_n))
        grand += n
        totals["any"] += any_n
        for k in SIGNALS:
            totals[k] += counts[k]

    if not grand:
        print("no issues read")
        return 1

    print(f"{'repository':<40} {'items':>6} {'non-goal':>9} {'accept':>7} {'checklist':>10} {'scope':>6}   verdict")
    for nwo, n, c, _ in sorted(rows, key=lambda r: -r[1]):
        # Per-repository verdict on the three tracker-supplied envelope dimensions.
        # A dimension counts as "carried here" only if a majority of items have
        # it -- one disciplined ticket does not make a disciplined tracker.
        carried = sum(1 for k in ("non_goals", "acceptance_stated", "in_scope")
                      if c[k] * 2 > n)
        verdict = ("THIN" if carried == 0 else "PARTIAL" if carried < 3 else "FULL")
        print(f"  {nwo:<38} {n:>6} {c['non_goals']:>9} {c['acceptance_stated']:>7} "
              f"{c['checklist_only']:>10} {c['in_scope']:>6}   {verdict}")

    print(f"\n  TOTAL {grand} work items across {len(rows)} repository(ies)")
    for k in ("non_goals", "acceptance_stated", "checklist_only", "in_scope"):
        pct = 100 * totals[k] // grand
        print(f"    {k:<12} present in {totals[k]:>4} ({pct:>3}%)")
    pct_any = 100 * totals["any"] // grand
    print(f"    {'any signal':<12} present in {totals['any']:>4} ({pct_any:>3}%)")

    print("\n  Detection is generous (prose phrases, not structured fields), so these are")
    print("  UPPER BOUNDS on what any adapter could extract. The adapters extract 0% today.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
