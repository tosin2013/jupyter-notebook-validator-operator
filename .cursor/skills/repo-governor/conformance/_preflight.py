"""Name a missing adapter dependency, so a red suite means what it says.

Found while checking whether a second machine could pick this repository up.
Without `dolt` on PATH, `layer1.py` reports NON-CONFORMANT and `layer2.py`
reports NOT EQUIVALENT -- the thesis test, reading red -- when the fact of the
matter is that an adapter dependency is not installed. An operator who has
never seen these suites green has no way to tell those two apart, and the
second reading is the one that gets reported upward.

This does NOT skip anything. A suite that quietly excuses checks when a tool is
missing is how a harness goes vacuously green, which this project has already
had to fix five times. The checks still run and still fail; the failure just
stops being anonymous.

`bootstrap-decisions.sh` already refuses loudly on a missing `dolt`, so an
operator following the documented order learns before reaching here. This is
for the one who does not.
"""

from __future__ import annotations

import shutil

# Adapter dependencies (ADR-011 rule 4). The engine itself is stdlib-only
# (rule 1) and must never appear in this list.
DEPENDENCIES = {
    "dolt": "decision_history via adapters/dolt-decisions (INV-005)",
}


def missing():
    """Adapter dependencies absent from PATH, as {binary: what it serves}."""
    return {b: why for b, why in DEPENDENCIES.items() if not shutil.which(b)}


def banner():
    """Print a heads-up before the checks run. Returns what was missing."""
    absent = missing()
    for b, why in absent.items():
        print(f"[preflight] {b!r} is not on PATH -- serves {why}.")
    if absent:
        print("[preflight] Checks below still run. Failures naming these providers"
              "\n            are a missing dependency, not a conformance defect."
              "\n            Install, or run tools/bootstrap-decisions.sh for the"
              "\n            full message including the unbind option.\n")
    return absent


def attribute(absent):
    """Print after a FAILING verdict, so the cause travels with the result."""
    if absent:
        print(f"\nNote: {', '.join(sorted(absent))} absent from PATH. Re-check this"
              "\nverdict with the dependency installed before reporting it.")
