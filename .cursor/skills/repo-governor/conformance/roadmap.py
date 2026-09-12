#!/usr/bin/env python3
"""Authorized work has a stop condition (issue 134).

Milestone membership is admission here and an assignee is authorization
(`adapters/github-projects:340-364`). So a milestoned, assigned, open issue is
**cleared to execute**. If it has no acceptance record, nothing states what
would finish it: `engine/completion.py` reads `NO_CRITERIA_DECLARED`, and issue
134 says what that means -- such work "can never be finished, only abandoned".

THE STATE THIS SUITE EXISTS FOR was live when it was written. Issue 1, the
project's own thesis question, was milestoned and assigned with no bar: cleared
to execute with no reachable `STOP_COMPLETE`.

WHY ONLY MILESTONED **AND** ASSIGNED. Admitted-but-unassigned with no bar is
fine -- nobody is working it, and demanding a bar at admission would put
friction on a human act that ADR-018 deliberately keeps human. The dangerous
state is narrower than "no bar": it is *authorized* with no bar.

WHY IT IS LIVE, NOT HERMETIC. The answer depends on a tracker this suite does
not control, exactly like `hooks`. Someone assigning an issue turns it red with
no code change, which is the point -- and why `tools/run-conformance.sh
--hermetic` exists and why CI runs the live set as its own workflow.

NO NETWORK IS UNRESOLVED, NEVER SATISFIED. A suite that passes because it could
not ask reports safety it never established (ADR-007). `conformance/install.py`
takes the same line about an unreachable remote.

ONE COMPARATOR, TWO CALLERS. `unbarred()` is used by the real scan and by the
controls. Issue 187's control re-implemented its comparison inline, so breaking
the real loop left the control green -- a control that tests a *copy* of the
logic tests nothing about the logic.

Reports issue NUMBERS only, never titles or bodies (section 51).

Usage:  python3 conformance/roadmap.py
"""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))

import _count as _CNT  # noqa: E402 -- alias avoids `C`, bound to `completion`
# in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("roadmap")

BARS = ROOT / ".repo-governor" / "acceptance"
REPO = "tosin2013/repo-governor"


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def authorized_open():
    """Open issues that are milestoned AND assigned, or None if unreachable.

    None is not an empty set. An unreachable tracker and a tracker with no
    authorized work are different facts -- ADR-003 rule 6's distinction,
    applied to a network call.
    """
    try:
        p = subprocess.run(
            ["gh", "issue", "list", "--repo", REPO, "--state", "open", "--limit", "300",
             "--json", "number,milestone,assignees"],
            capture_output=True, text=True, timeout=120)
    except (OSError, subprocess.TimeoutExpired):
        return None
    if p.returncode != 0:
        return None
    try:
        rows = json.loads(p.stdout)
    except json.JSONDecodeError:
        return None
    return {r["number"] for r in rows if r.get("milestone") and r.get("assignees")}


def unbarred(numbers, bar_dir=BARS):
    """THE comparator. Which of these authority ids have no acceptance record."""
    return sorted(n for n in numbers if not (bar_dir / f"{n}.json").is_file())


def main():
    fails = 0
    print("Authorized work has a stop condition\n")

    live = authorized_open()

    # An unreachable tracker is a failure, not a skip. The alternative is a
    # green run over a question nobody asked.
    fails += check("the roadmap authority answered", live is not None,
                   "gh could not be reached, or returned no parseable result. "
                   "This suite is LIVE; run ./tools/run-conformance.sh --hermetic "
                   "to exclude it, but do not read this as a pass.")
    if live is None:
        print(f"\nROADMAP: NON-CONFORMANT ({fails})")
        return 1

    # A query returning nothing passes every check below it vacuously. This
    # repository has shipped that defect often enough to check for it by name.
    fails += check(f"and it returned authorized work to check ({len(live)} issues)",
                   bool(live),
                   "zero milestoned+assigned open issues -- either the roadmap is "
                   "genuinely empty or the query went blind; both need a human")

    missing = unbarred(live)
    fails += check("every authorized issue declares what would finish it",
                   not missing,
                   f"AUTHORIZED with no acceptance record: {missing}. Each is cleared "
                   f"to execute with STOP_COMPLETE unreachable -- write "
                   f".repo-governor/acceptance/<n>.json, or remove the milestone if "
                   f"the work is not committed.")

    # Controls. Both call `unbarred`, the same function the scan above used.
    print("\nControls\n")

    sentinel = 999999999
    fails += check("control: an authorized issue with no record IS reported",
                   unbarred({sentinel}) == [sentinel],
                   "the comparator did not flag a number that has no bar file; "
                   "every result above is meaningless")

    barred = sorted(int(f.stem) for f in BARS.glob("*.json") if f.stem.isdigit())
    fails += check(f"control: an issue that HAS a record is not reported ({len(barred)} on disk)",
                   bool(barred) and unbarred(set(barred[:5])) == [],
                   "the comparator flags issues that do have bars, so it would "
                   "report everything and prove nothing")

    print(f"\nROADMAP: {'CONFORMANT' if not fails else f'NON-CONFORMANT ({fails})'}")
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
