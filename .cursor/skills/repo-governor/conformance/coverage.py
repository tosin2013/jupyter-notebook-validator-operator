#!/usr/bin/env python3
"""Every suite reports how much it did, and does it the same way (issue 67).

Eight assertions in `conformance/onboarding.py` were unreachable and the suite
reported PASS throughout, because nothing counted how many assertions a suite
EXECUTED. A suite running fewer than it contains read identically to one
running all of them.

This validates the MECHANISM. It deliberately does not re-run the other
suites: `tools/run-conformance.sh` already runs them once, reads the declared
`CONFORMANCE-COUNT` line, and fails a suite that emits none or reports zero.
Re-running thirteen suites from inside a fourteenth would double every CI run
to re-learn what the runner already knows.

The uninstrumented case is the one that matters. A suite that quietly stops
counting looks exactly like a suite with nothing to count, so silence is a
failure here rather than an absence.

Usage:  python3 conformance/coverage.py
"""

from __future__ import annotations

import ast
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CONF = ROOT / "conformance"

import _count as _CNT  # noqa: E402
_CNT.watch("coverage")


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def suites():
    """Same derivation the suite-count check uses: a file with sys.exit(main())."""
    return sorted(p for p in CONF.glob("[a-z]*.py")
                  if "sys.exit(main(" in p.read_text(encoding="utf-8"))


def main():
    fails = 0
    found = suites()

    print("Every suite is instrumented, and none opted out quietly\n")
    fails += check(f"the suite set was derived ({len(found)} found)",
                   len(found) >= 8 and any(p.name == "layer1.py" for p in found),
                   "a broken glob reads as a fully instrumented repository")

    for p in found:
        src = p.read_text(encoding="utf-8")
        fails += check(f"{p.name} installs the counter",
                       "_CNT.watch(" in src,
                       "without it the suite reports nothing and its silence is "
                       "indistinguishable from having nothing to report")

    print("\nThe counter counts what it claims\n")
    # Behavioural, not a source grep: run a throwaway suite-shaped script and
    # read the line it emits. A grep for `watch(` proves the call is written,
    # not that it works.
    probe = CONF / "_coverage_probe.py"
    probe.write_text(
        "import sys\n"
        f"sys.path.insert(0, {str(CONF)!r})\n"
        "import _count as C\n"
        "C.watch('probe')\n"
        "print('  [PASS] one')\n"
        "print('  [FAIL] two')\n"
        "print('  not an assertion at all')\n"
        "C.record(True)\n", encoding="utf-8")
    try:
        out = subprocess.run([sys.executable, str(probe)], capture_output=True,
                             text=True, timeout=60).stdout
    finally:
        probe.unlink(missing_ok=True)

    line = [l for l in out.splitlines() if l.startswith(_CNT.MARKER)]
    fails += check("a suite-shaped script emits exactly one count line", len(line) == 1, out[-200:])
    if line:
        fields = dict(kv.split("=", 1) for kv in line[0].split()[1:])
        fails += check("it counts [PASS] and [FAIL], and record()", fields.get("executed") == "3",
                       f"executed={fields.get('executed')}, expected 3")
        fails += check("it counts failures separately", fields.get("failed") == "1",
                       f"failed={fields.get('failed')}")
        fails += check("a non-assertion line is not counted", fields.get("executed") != "4",
                       "any printed line would inflate every suite's total")

    print("\nThe format is a contract, not prose\n")
    src = (CONF / "_count.py").read_text(encoding="utf-8")
    fails += check("the marker is declared in one place",
                   src.count('MARKER = "') == 1 and "CONFORMANCE-COUNT" in src)
    fails += check("the runner reads the marker rather than guessing",
                   _CNT.MARKER in (ROOT / "tools" / "run-conformance.sh").read_text(encoding="utf-8"),
                   "if the runner greps [PASS] lines instead, a suite that changes "
                   "its output silently leaves coverage")

    print(f"\n{'COVERAGE: CONFORMANT' if not fails else f'COVERAGE: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
