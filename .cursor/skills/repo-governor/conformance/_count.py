"""One number, reported the same way by every suite.

Issue 67. Eight assertions in `conformance/onboarding.py` were unreachable -- a
guard tested a path the code above it had renamed away -- and the suite
reported PASS throughout. Nothing counted how many assertions a suite actually
EXECUTED, so a suite running fewer than it contains read identically to one
running all of them.

WHY A PRINT WRAPPER, AND NOT AN EXPLICIT CALL. The first attempt put a
`tally(ok)` inside each suite's reporting helper. That works for the nine
suites that HAVE a helper, and not for the four that print `[PASS]` and
`[FAIL]` from separate branches -- `vocabulary.py` alone has nineteen such
literals. Instrumenting those by hand means ~thirty edits where a forgotten
branch silently under-counts, which is the same defect class this file exists
to close. Two bugs during that attempt were exactly that kind of slip: an
`import _count as C` silently rebound `C` in the two suites that already
import `completion` as `C`, and the first frame attribution reported one site
per suite because it found the counter's own call line.

So the count is taken from the thing every suite already does uniformly --
emitting a line that begins `[PASS]` or `[FAIL]` -- and a branch cannot be
forgotten because no branch is edited.

THE TRADE, STATED PLAINLY. This matches on the suite's own output prefix. A
suite that changed that prefix would stop counting silently, which is the
failure mode of every string-based check. `conformance/coverage.py` closes it
by failing any suite that reports `executed=0`: a suite asserting nothing is
either broken or lying, and both need a human.

TWO LIMITS FOUND BY MEASURING, NOT BY REASONING.

`executed` misses assertions printed by a SUBPROCESS: `layer2.py` runs
`tools/live-equivalence.py --self-test`, whose fourteen `[PASS]` lines go
straight to inherited stdout and never pass through this wrapper. Suites in
that position call `record()` explicitly.

`sites` is only meaningful for a suite that prints AT the assertion. `layer1`
and `onboarding` buffer rows and print them from one loop, so their site count
collapses to 1 while `executed` stays correct. That is a property of those
suites, not an error, and it is why nothing gates on `sites`.

WHAT THIS DOES NOT DO. It makes a DROP visible -- 90 assertions yesterday, 82
today means something stopped running. It does not prove every assertion in a
file is reachable, and it would NOT have found the original defect, whose
count was always 79 and never fell. Finding pre-existing dead code needs
per-site reachability against a known-complete site list, which nothing here
supplies. `sites()` records which lines fired so that work has somewhere to
start; nothing consumes it yet, and implying otherwise would be the vacuity
this file exists to prevent.
"""

from __future__ import annotations

import atexit
import builtins
import sys
from pathlib import Path

_executed = 0
_failed = 0
_lines: set = set()

MARKER = "CONFORMANCE-COUNT"

# Frames belonging to a reporting helper are not the assertion site; the site
# is whatever called them. Depth varies across the three reporting styles in
# use, so counting frames would be wrong for at least one of them.
_HELPERS = {"check", "add", "_print", "counting_print"}


def _record(ok):
    global _executed, _failed
    _executed += 1
    if not ok:
        _failed += 1
    try:
        f = sys._getframe(2)
        while f:
            p = Path(f.f_code.co_filename)
            if (p.parent.name == "conformance" and not p.name.startswith("_")
                    and f.f_code.co_name not in _HELPERS):
                _lines.add((p.name, f.f_lineno))
                break
            f = f.f_back
    except Exception:  # noqa: BLE001 -- counting must never break a suite
        pass


def record(ok):
    """Count an assertion a suite reports in its OWN shape.

    `conformance/layer2.py` scores scenarios as AGREE / DIVERGENCE / WRONG and
    never prints `[PASS]`, so the wrapper sees nothing to count. Rather than
    forcing every suite into one output vocabulary, a suite with a different
    shape says so here -- once, at its verdict site, where forgetting it would
    be as visible as forgetting to score the scenario at all.
    """
    _record(ok)


def counts():
    return {"executed": _executed, "failed": _failed, "sites": len(_lines)}


def sites():
    """(file, line) of every assertion that fired. For future reachability work."""
    return sorted(_lines)


def line(suite):
    """The declared, fixed-format line every suite ends with."""
    return f"{MARKER} suite={suite} executed={_executed} failed={_failed} sites={len(_lines)}"


def watch(suite):
    """Count every `[PASS]`/`[FAIL]` this process prints, and report at exit.

    Registered at exit rather than at the end of main() so the line is emitted
    even when a suite returns early or raises -- a suite that dies halfway
    should still say how far it got, since that is precisely the case where
    the count matters.
    """
    real = builtins.print

    def counting_print(*args, **kwargs):
        if args and isinstance(args[0], str):
            s = args[0].lstrip()
            if s.startswith("[PASS]"):
                _record(True)
            elif s.startswith("[FAIL]"):
                _record(False)
        return real(*args, **kwargs)

    builtins.print = counting_print
    atexit.register(lambda: real(line(suite)))
