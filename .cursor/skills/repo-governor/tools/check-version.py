#!/usr/bin/env python3
"""Do the version strings in this tree agree with the tag being released?

`engine_version` lands in every decision record (ADR-009). A release whose
recorded engine version does not match its tag produces an evidence chain that
cites an engine nobody can check out -- which defeats the point of recording it.

Two DIFFERENT facts are checked, and collapsing them would be wrong:

  ENGINE_VERSION      what this engine IS. Must equal the tag exactly.
  engine_min_version  the oldest engine that can read a given manifest. Must be
                      <= the tag, and the two places that state it must agree
                      with each other. It is a compatibility floor, not a
                      version, so requiring equality would force a bump on
                      every release and make the field meaningless.

Every site is asserted to MATCH ITS PATTERN before its value is compared. A
pattern that stops matching -- because a constant moved or a file was
reformatted -- must fail loudly. Silently finding nothing to compare and
reporting success is how a version check goes blind, and this repository has
shipped that shape often enough to stop assuming it will not.

Usage:  python3 tools/check-version.py v0.2.0
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# path -> (regex with one capture group, what the value means)
EXACT = {
    "engine/version.py": (r'^ENGINE_VERSION\s*=\s*"([^"]+)"', "the engine's own version"),
}
# The version a READER is told to install. Not decoration: the README pinned
# v0.1.0 while v0.2.1 was Latest, across two releases, because pinning a
# literal closed the "install is not a release" gap and opened a worse one --
# unpinned it merely tracked main, pinned it is confidently wrong and rots
# silently. This is the same class as ENGINE_VERSION, so it is checked the
# same way rather than watched.
INSTALL_PIN = {
    "README.md": (r"--branch v([0-9]+\.[0-9]+\.[0-9]+)",
                  "the version the README tells people to clone"),
    "docs/installation.md": (r"--branch v([0-9]+\.[0-9]+\.[0-9]+)",
                             "the version the installation guide tells people to clone"),
}

FLOOR = {
    ".repo-governor.json": (r'"engine_min_version"\s*:\s*"([^"]+)"',
                            "this repository's own manifest floor"),
    "tools/onboard-interactive.py": (r'"engine_min_version"\s*:\s*"([^"]+)"',
                                     "the floor written into every proposal it generates"),
}


def parse(v):
    m = re.fullmatch(r"(\d+)\.(\d+)\.(\d+)", v)
    return tuple(int(x) for x in m.groups()) if m else None


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}"
          + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def read(rel, pattern):
    """(value, error). A pattern that matches nothing is an error, never a skip."""
    p = ROOT / rel
    if not p.is_file():
        return None, f"{rel} does not exist"
    hits = re.findall(pattern, p.read_text(encoding="utf-8"), re.M)
    if not hits:
        return None, (f"{rel}: the pattern matched nothing -- the constant moved, or the "
                      "file was reformatted, and this check went blind rather than failing")
    if len(set(hits)) > 1:
        return None, f"{rel}: states {sorted(set(hits))} in the same file"
    return hits[0], None


def main(argv):
    if not argv:
        print("usage: check-version.py <tag>", file=sys.stderr)
        return 2
    tag = argv[0].lstrip("v")
    want = parse(tag)
    fails = 0
    print(f"Release {argv[0]}\n")
    if not want:
        return check("the tag is a semantic version", False, f"{argv[0]!r}") or 1

    for rel, (pat, meaning) in EXACT.items():
        got, err = read(rel, pat)
        if err:
            fails += check(f"{rel} states a version", False, err)
            continue
        fails += check(f"{rel} ({meaning}) is {tag}", got == tag,
                       f"states {got!r}, the tag says {tag!r} -- decision records would "
                       "cite an engine that is not this one")

    for rel, (pat, meaning) in INSTALL_PIN.items():
        got, err = read(rel, pat)
        if err:
            # A pattern that matches nothing is a check that went blind, not a
            # document that got simpler.
            fails += check(f"{rel} names an install version", False, err)
            continue
        fails += check(f"{rel} ({meaning}) pins {tag}", got == tag,
                       f"tells readers to install v{got} while releasing v{tag} -- "
                       "a stale pin is worse than none, because it looks deliberate")

    floors = {}
    for rel, (pat, meaning) in FLOOR.items():
        got, err = read(rel, pat)
        if err:
            fails += check(f"{rel} states engine_min_version", False, err)
            continue
        floors[rel] = got
        v = parse(got)
        fails += check(f"{rel} ({meaning}) floor {got} is not above the release", 
                       bool(v) and v <= want,
                       f"a manifest declaring it needs {got} cannot be read by {tag}")

    if len(floors) == 2:
        vals = set(floors.values())
        fails += check("both engine_min_version sites agree", len(vals) == 1,
                       f"{floors} -- new users would be handed a floor this repository "
                       "does not itself declare")

    print(f"\n{'VERSION: CONSISTENT' if not fails else f'VERSION: INCONSISTENT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
