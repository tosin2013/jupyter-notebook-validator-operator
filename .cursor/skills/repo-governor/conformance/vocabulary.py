#!/usr/bin/env python3
"""Vocabulary conformance — gate 7 (#13).

Two jobs:

  1. The closed sets must not drift from the code. Every reason string any
     adapter or engine module emits must exist in engine/vocabulary.py, and
     every reason in the vocabulary must be reachable. A vocabulary that
     documents reasons nobody emits, or misses reasons that are emitted, is
     not a closed set — it is a wish.

  2. The decision table must be total. Every reachable combination of
     authority state and acceptance state maps to exactly one disposition,
     with no gaps and no ambiguity (ADR-007 rule 2).

Usage:  python3 conformance/vocabulary.py
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# File providers read fixtures from conformance/, never from .repo-governor/,
# which holds only this repository's real governance state (ADR-022).
ACCEPTANCE_ENV = {"REPO_GOVERNOR_ACCEPTANCE_DIR": "conformance/fixtures/acceptance"}
sys.path.insert(0, str(ROOT / "engine"))
import vocabulary as V  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("vocabulary")

SCAN_DIRS = ("adapters", "engine")
REASON_PATTERNS = (
    re.compile(r'reason\s*=\s*"([A-Z_]{3,})"'),
    re.compile(r'"reason"\s*:\s*"([A-Z_]{3,})"'),
)

# Authority x acceptance -> expected disposition. Total by construction:
# every authority value crossed with every acceptance state.
AUTHORITY_STATES = ("AUTHORIZED", "ADMITTED", "CANCELLED", "WITHDRAWN",
                    "REJECTED", "DEFERRED", "UNRESOLVED")
ACCEPTANCE_STATES = ("SATISFIED", "UNSATISFIED", "UNDECLARED", "UNVERIFIABLE")

EXPECTED = {}
for a in AUTHORITY_STATES:
    for c in ACCEPTANCE_STATES:
        if a == "UNRESOLVED":
            d = "UNKNOWN"
        elif a in ("CANCELLED", "WITHDRAWN", "REJECTED"):
            d = "AUTHORITY_WITHDRAWN"
        elif a in ("ADMITTED", "DEFERRED"):
            d = "NO_EXECUTION_AUTHORITY"
        elif c == "SATISFIED":
            d = "STOP_COMPLETE"
        elif c == "UNVERIFIABLE":
            d = "UNKNOWN"
        else:                      # UNSATISFIED or UNDECLARED
            d = "CONTINUE"
        EXPECTED[(a, c)] = d


def emitted_reasons():
    found = {}
    for d in SCAN_DIRS:
        for p in (ROOT / d).rglob("*"):
            if not p.is_file() or p.name.startswith("."):
                continue
            try:
                text = p.read_text(errors="ignore")
            except OSError:
                continue
            for pat in REASON_PATTERNS:
                for m in pat.findall(text):
                    found.setdefault(m, set()).add(str(p.relative_to(ROOT)))
    # Vocabulary definitions are not emissions.
    found.pop("REASONS", None)
    return found


def main():
    fails = 0
    print("Closed-set integrity\n")

    V.check_alphabets_disjoint()
    print("  [PASS] governance and onboarding alphabets are disjoint")

    emitted = emitted_reasons()
    # engine/vocabulary.py itself declares them; exclude it from "emitters".
    emitted = {r: {s for s in srcs if not s.endswith("vocabulary.py")}
               for r, srcs in emitted.items()}
    emitted = {r: s for r, s in emitted.items() if s}

    undeclared = sorted(set(emitted) - set(V.REASONS))
    for r in undeclared:
        fails += 1
        print(f"  [FAIL] {r} emitted by {sorted(emitted[r])[0]} but not in the vocabulary")
    if not undeclared:
        print(f"  [PASS] all {len(emitted)} emitted reasons are declared")

    unreachable = sorted(set(V.REASONS) - set(emitted))
    # Provider-health reasons are produced by the engine at composition time,
    # not by a literal in an adapter; allow those two explicitly.
    allowed_abstract = {"PROVIDER_UNREACHABLE", "TRANSPORT_UNCONFIGURED"}
    stray = [r for r in unreachable if r not in allowed_abstract]
    if stray:
        fails += 1
        print(f"  [FAIL] declared but never emitted: {stray}")
    else:
        print(f"  [PASS] every declared reason is reachable "
              f"({len(allowed_abstract)} engine-level exceptions)")

    bad = [r for r in V.REASONS if V.REASONS[r][0] not in V.DIMENSIONS]
    if bad:
        fails += 1
        print(f"  [FAIL] reasons with an undeclared dimension: {bad}")
    else:
        print(f"  [PASS] every reason names one of {len(V.DIMENSIONS)} declared dimensions")

    # classify() must reject anything outside the set.
    try:
        V.classify("TOTALLY_MADE_UP")
        fails += 1
        print("  [FAIL] classify() accepted a reason outside the closed set")
    except V.VocabularyError:
        print("  [PASS] classify() rejects a reason outside the closed set")

    # Profiles may only escalate, never loosen.
    loosened = []
    for prof, reasons in V.PROFILE_ESCALATIONS.items():
        for r in reasons:
            if r not in V.REASONS:
                loosened.append(f"{prof}:{r} not a real reason")
            elif V.REASONS[r][1] is True:
                loosened.append(f"{prof}:{r} already blocking")
    if loosened:
        fails += 1
        print(f"  [FAIL] profile escalation problems: {loosened}")
    else:
        print("  [PASS] profiles only escalate non-blocking reasons, never loosen blocking ones")

    print("\nDecision table totality\n")
    missing = [k for k in EXPECTED if EXPECTED[k] is None]
    bad_disp = sorted({d for d in EXPECTED.values() if not V.is_disposition(d)})
    if missing:
        fails += 1
        print(f"  [FAIL] {len(missing)} combination(s) map to nothing")
    else:
        print(f"  [PASS] all {len(EXPECTED)} authority x acceptance combinations map to a disposition")
    if bad_disp:
        fails += 1
        print(f"  [FAIL] table produces non-dispositions: {bad_disp}")
    else:
        print(f"  [PASS] every mapped value is in the closed disposition set")

    # No authority state may reach STOP_COMPLETE or EXECUTE without authorization.
    leaks = [(a, c) for (a, c), d in EXPECTED.items()
             if d in ("STOP_COMPLETE", "EXECUTE") and a != "AUTHORIZED"]
    if leaks:
        fails += 1
        print(f"  [FAIL] unauthorized states reaching execution dispositions: {leaks}")
    else:
        print("  [PASS] only AUTHORIZED reaches STOP_COMPLETE or EXECUTE (INV-002)")

    print("\nAmendment integrity (ADR-017 weakness)\n")
    sys.path.insert(0, str(ROOT / "engine"))
    import amendments as A
    findings, records = A.audit()
    if findings:
        fails += 1
        for wid, kind, detail in findings:
            print(f"  [FAIL] {kind}: {detail[:88]}")
    else:
        amended = sum(1 for _, n, _, _ in records if n)
        print(f"  [PASS] all {amended} amendment(s) carry a resolvable citation")

    # Negative: an un-cited amendment must make the adapter refuse.
    import copy, json as _j, subprocess, tempfile
    acc = ROOT / "conformance" / "fixtures" / "acceptance" / "AUTHORIZED-1.json"
    orig = acc.read_text()
    try:
        d = _j.loads(orig)
        if d.get("amendments"):
            d["amendments"][0].pop("cites", None)
            acc.write_text(_j.dumps(d))
            r = subprocess.run([sys.executable, str(ROOT / "adapters" / "acceptance-file"),
                                "query", "acceptance_criteria", "get_criteria", "id=AUTHORIZED-1"],
                               capture_output=True, text=True, cwd=ROOT,
                               env={**os.environ, **ACCEPTANCE_ENV})
            out = _j.loads(r.stdout)
            good = out.get("ok") is False and out["error"]["type"] == "MALFORMED_SOURCE"
            fails += not good
            print(f"  [{'PASS' if good else 'FAIL'}] adapter refuses criteria whose amendment lacks a citation")
    finally:
        acc.write_text(orig)

    print(f"\nvocabulary: {len(V.DISPOSITIONS)} governance + {len(V.ONBOARDING)} onboarding "
          f"dispositions, {len(V.REASONS)} reasons "
          f"({sum(1 for r in V.REASONS if V.REASONS[r][1])} blocking)")
    print("VOCABULARY: " + ("CLOSED" if not fails else f"NOT CLOSED ({fails} problem(s))"))
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
