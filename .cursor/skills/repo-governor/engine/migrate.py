#!/usr/bin/env python3
"""Manifest migration. ADR-004 step 5: write the v1->v2 path BEFORE v1 ships,
so the first schema change is not an emergency.

There is no v2 yet. This exists so that adding one has an obvious home and a
tested shape, rather than being invented under pressure.

Usage:  python3 engine/migrate.py <manifest.json> [--to N]
"""
from __future__ import annotations
import json, sys
from pathlib import Path

MIGRATIONS = {}   # (from, to) -> callable(dict) -> dict


def register(frm, to):
    def deco(fn):
        MIGRATIONS[(frm, to)] = fn
        return fn
    return deco


def migrate(data, target):
    cur = (data.get("repo_governor") or {}).get("version")
    while cur != target:
        step = MIGRATIONS.get((cur, cur + 1))
        if step is None:
            raise SystemExit(f"no migration from v{cur} to v{cur + 1}")
        data = step(data)
        cur += 1
        data["repo_governor"]["version"] = cur
    return data


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    target = int(argv[argv.index("--to") + 1]) if "--to" in argv else 1
    data = json.loads(Path(argv[0]).read_text())
    cur = (data.get("repo_governor") or {}).get("version")
    if cur == target:
        print(f"already at v{target}; nothing to do")
        return 0
    print(json.dumps(migrate(data, target), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
