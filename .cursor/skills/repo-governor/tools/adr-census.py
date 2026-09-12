#!/usr/bin/env python3
"""Measure ADR readability across repositories this project did not write.

Layer 1 gives `adapters/adr` 136 passing contract checks. Against real
repositories it once read 16% of decisions -- because a test cannot detect a
convention it shares with the implementation, and every fixture was written in
this project's own dialect. This tool is the counterweight: it reports coverage
over directories nobody here authored.

Read-only. No network. Prints paths, never contents.

Usage:  python3 tools/adr-census.py <dir> [<dir> ...]
"""

from __future__ import annotations

import collections
import importlib.machinery
import importlib.util
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
NUM = re.compile(r"^(?:adr[-_])?(\d{3,4})[-_.]", re.I)


def _adapter():
    spec = importlib.util.spec_from_loader(
        "_adr", importlib.machinery.SourceFileLoader("_adr", str(ROOT / "adapters" / "adr")))
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    adr = _adapter()
    total = read = skipped = 0
    reasons = collections.Counter()
    per_dir = []
    for d in argv:
        p = Path(d).expanduser()
        if not p.is_dir():
            continue
        n = r = 0
        for f in sorted(p.glob("*.md")):
            if not NUM.match(f.name):
                skipped += 1
                continue
            n += 1
            status, why = adr._status_of(f.read_text(errors="replace"))
            if status:
                r += 1
            else:
                reasons[why] += 1
        if n:
            per_dir.append((r, n, str(p)))
            total += n
            read += r

    for r, n, path in sorted(per_dir, key=lambda x: (x[0] / x[1], -x[1])):
        flag = "" if r == n else "   <-- incomplete"
        print(f"  {r:>4}/{n:<4} {100 * r // n:>3}%  {path}{flag}")
    if not total:
        print("no numbered decision files found")
        return 1
    print(f"\n  TOTAL {read}/{total} readable ({100 * read // total}%) "
          f"across {len(per_dir)} collection(s); {skipped} non-decision file(s) skipped")
    for k, v in reasons.most_common():
        print(f"    {v:>4}  {k}")
    return 0 if read == total else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
