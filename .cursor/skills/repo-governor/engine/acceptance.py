#!/usr/bin/env python3
"""Show, or scaffold, the completion bar for one authority id (issue 58).

WHY. Nothing in this repository ever wrote an acceptance-criteria file. So in
every repository except this one `.repo-governor/acceptance/` is empty, and the
consequence is specific: STOP_COMPLETE is UNREACHABLE. `completion.py` answers
CONTINUE with a non-blocking NO_CRITERIA_DECLARED for every admitted item,
indefinitely, and the section 40 completion firewall -- "once acceptance
conditions are satisfied, nothing converts to execution" -- has no acceptance
conditions to be satisfied, so it never fires.

That is the product's headline behaviour. It works here because criteria were
hand-written; it worked nowhere else.

WHAT THIS DOES NOT DO, and must never do:

  It does not invent criteria. Inferring what "done" means from an issue's
  title or body is exactly the guess ADR-018 forbids for admission signals,
  applied to completion instead. A generated criterion that looks plausible
  and is wrong is worse than an empty one, because an empty one is honestly
  NO_CRITERIA_DECLARED while a wrong one produces a confident STOP_COMPLETE.

  It does not write unless asked. Creating a file in a repository is a change
  to that repository and deny-by-default applies (ADR-005) -- the same reason
  `onboard.py --write` sits behind a flag. Without --template this only reads.

  It does not overwrite. A bar that already exists is a bar somebody set, and
  replacing it with a skeleton would be editing the bar to fit the work.

Usage:
  python3 engine/acceptance.py <id>              report what exists  (READ ONLY)
  python3 engine/acceptance.py <id> --template   write a skeleton to fill in
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "engine"))
import bindings as B  # noqa: E402

CHECKS = ("tests_pass", "file_exists", "command_exit")


def path_for(wid):
    return B.target() / ".repo-governor" / "acceptance" / f"{wid}.json"


def template(wid):
    return {
        "$comment": (
            "TEMPLATE -- NOT a completion bar until a human edits it. The criteria array "
            "is deliberately empty, and an empty array reads NO_CRITERIA_DECLARED, never "
            "satisfied: completion.py and adapters/acceptance-file each refuse it "
            "independently. Nothing here was inferred from the work item; what 'done' "
            "means is not derivable from a title, and guessing it is how a confident "
            "wrong STOP_COMPLETE gets produced. Replace the array with conditions that "
            "would actually fail if the work were not finished."
        ),
        "$examples": [
            {"check": "file_exists", "target": "path/that/must/exist"},
            {"check": "command_exit", "target": "python3 tests/run.py"},
            {"check": "tests_pass", "target": "npm test"},
        ],
        # An author who knows their bar is partial needs somewhere to say so
        # that is not a comment. Two bars in this repository recorded exactly
        # that in prose and both read STOP_COMPLETE for the whole item, because
        # nothing mechanical reads prose (issue 133). Offered here, unset,
        # because most bars are not partial and a template must not suggest
        # that scoping one down is the normal move.
        "$covers_if_partial": {
            "$comment": ("Rename to 'covers' ONLY if this bar deliberately covers part "
                         "of the item. Completion then becomes unavailable until the "
                         "uncovered part is split out or this bar is extended. Both "
                         "fields are required: a partial bar that will not name what it "
                         "leaves out has recorded a boast, not a limit."),
            "declared": "the part this bar does cover",
            "uncovered": "the part it does NOT, and why it is not attempted here",
        },
        "authority_id": str(wid),
        "criteria": [],
    }


def verdict(wid):
    """What the engine says about this id right now. Read-only, in-process.

    Called rather than spawned. ADR-021 puts adapter spawning in bindings.py
    and nowhere else, and conformance/bindings.py enforces it by asserting no
    other engine module invokes a subprocess -- which caught the first version
    of this function shelling out to completion.py. Importing is also the
    cheaper answer: the manifest is already loaded.
    """
    try:
        import completion as C  # noqa: PLC0415 -- local, avoids a cycle at import time
        return C.evaluate(str(wid))
    except Exception:
        return None


def main(argv):
    if not argv or argv[0].startswith("-"):
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    wid = argv[0]
    write = "--template" in argv[1:]
    p = path_for(wid)

    if p.exists():
        data = json.loads(p.read_text(encoding="utf-8"))
        n = len(data.get("criteria") or [])
        print(f"{p}")
        print(f"  criteria declared: {n}")
        if n == 0:
            print("  An empty bar is not a met bar. This reads NO_CRITERIA_DECLARED and")
            print("  STOP_COMPLETE stays unreachable until a real condition is declared.")
        if write:
            # Never overwrite. A bar somebody set is not a skeleton to replace.
            print("\nRefusing to overwrite an existing acceptance record. Edit it, or "
                  "remove it first.", file=sys.stderr)
            return 1
        return 0

    print(f"No acceptance record for {wid} at {p}")
    d = verdict(wid)
    if d:
        print(f"  the engine currently answers: {d.get('decision')}")
        reasons = [u.get("reason") for u in (d.get("unknowns") or [])]
        if "NO_CRITERIA_DECLARED" in reasons:
            print("  STOP_COMPLETE is unreachable for this id: there is no bar to satisfy,")
            print("  so the completion firewall (section 40) cannot fire on it.")

    if not write:
        print(f"\nTo scaffold one:  python3 engine/acceptance.py {wid} --template")
        print("Writing is deliberate, not automatic: creating a file in a repository is a")
        print("change to it, and deny-by-default applies (ADR-005).")
        return 0

    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(template(wid), indent=2) + "\n", encoding="utf-8")
    print(f"\nwrote {p}")
    print("  It declares NOTHING yet, and that is deliberate -- the criteria array is")
    print("  empty and reads NO_CRITERIA_DECLARED. It is a skeleton, not a bar.")
    print("\n  Replace $examples with conditions that would FAIL if the work were not")
    print(f"  finished. Supported checks: {', '.join(CHECKS)}.")
    print(f"\n  Then: python3 engine/completion.py {wid}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
