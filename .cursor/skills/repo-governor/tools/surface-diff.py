#!/usr/bin/env python3
"""Does a result taken at one commit still apply at another?

Every activation result carries a commit, and nothing said whether it still
counted after a release. The question was answered once, by hand, by hashing
trees -- which is not a thing anybody will do again, so the honest fallback is
treating every result as stale on every release, which throws away most of the
evidence base.

It matters more now that tools/benchmark.py makes runs cheap: cheap runs across
a release cadence produce a pile of results whose comparability nobody can
establish, and the failure is silent both ways -- stale results pooled with
fresh ones, or good results discarded because nobody could tell.

WHAT IT DOES NOT DO. It reports; it gates nothing. No release is refused and no
record is invalidated -- a person decides whether a result still counts.

And it makes no claim about BEHAVIOUR. That the description is byte-identical
says the activation surface did not move. It does not say a model would answer
the same way: models change under you, which is what issue 5 and the
pre-registered comparison exist to measure. This narrows what has to be
re-measured; it never says nothing does.

Usage:  python3 tools/surface-diff.py <ref-a> <ref-b>
        python3 tools/surface-diff.py v0.1.0 v0.2.3 --json
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import _surface as S  # noqa: E402


def main(argv):
    if len(argv) < 2:
        print(__doc__.strip().splitlines()[-2].strip(), file=sys.stderr)
        return 2
    a, b = argv[0], argv[1]
    r = S.compare(a, b)

    if not r["description_readable"]:
        print(f"cannot read SKILL.md's description at one or both of {a}, {b}.",
              file=sys.stderr)
        print("Refusing to report comparability from a surface it cannot see.",
              file=sys.stderr)
        return 1

    if "--json" in argv:
        print(json.dumps(r, indent=2, sort_keys=True))
        return 0

    print(f"agent surface: {a} -> {b}\n")
    d = "CHANGED" if r["description_changed"] else "unchanged"
    print(f"  description   {d:<10} -> activation rates "
          + ("are STALE; the surface a model judges has moved"
             if r["description_changed"] else "remain comparable"))

    moved = r["changed"] + r["added"] + r["removed"]
    if moved:
        print(f"  body          CHANGED    -> grades may shift (the FULL/PARTIAL "
              "boundary); activation unaffected")
        for f in r["changed"]:
            print(f"                  changed  {f}")
        for f in r["added"]:
            print(f"                  added    {f}")
        for f in r["removed"]:
            print(f"                  removed  {f}")
    else:
        print("  body          unchanged  -> grades remain comparable too")

    print()
    if r["activation_comparable"] and r["grades_comparable"]:
        print("  Results taken at the first ref remain comparable at the second.")
    elif r["activation_comparable"]:
        print("  Activation RATES remain comparable. Grades were taken against text")
        print("  that has since moved, so re-read a FULL/PARTIAL before pooling it.")
    else:
        print("  Activation results taken at the first ref are STALE. The description")
        print("  is the surface a model judges, and it has changed.")
    print("\n  This reports; it gates nothing. It also says nothing about whether a")
    print("  MODEL would answer the same way -- models change independently of this")
    print("  repository, which is what the activation programme exists to measure.")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
