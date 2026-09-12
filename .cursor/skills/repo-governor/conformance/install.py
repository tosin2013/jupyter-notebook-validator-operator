#!/usr/bin/env python3
"""The install instructions name a tag that EXISTS (issue 178).

`conformance/skill.py` already checks that `git clone --branch vX.Y.Z` in the
install docs matches `ENGINE_VERSION`. That check cannot see the failure it was
written to prevent, and the gap was observed rather than imagined.

During the v0.4.0 release, between the version bump merging and the draft being
published, `main` told a new user to clone `v0.4.0` -- a tag that did not exist:

    fatal: Remote branch v0.4.0 not found in upstream origin

**The check was green throughout.** `README.md` and `ENGINE_VERSION` agreed
perfectly, and both were wrong.

INTERNAL CONSISTENCY IS NOT CORRECTNESS. A check comparing two files this
repository controls can only ever prove the first. Naming a tag that resolves is
a claim about the *remote*, so this suite asks the remote.

WHY IT IS LIVE, NOT HERMETIC. `tools/run-conformance.sh` splits HERMETIC from
LIVE, and `hooks` sits in LIVE because it depends on state the suite does not
control. A remote is the same kind of thing. The consequence is stated rather
than implied: **this does not run in the hermetic CI job**, and the moment it
would matter most is a release.

NO NETWORK IS UNRESOLVED, NEVER SATISFIED. A suite that passes because it could
not check reports safety it never established, which is ADR-007's line.
`conformance/layer1.py`'s `_preflight` names the missing binary and fails rather
than skipping quietly; this follows it.

Usage:  python3 conformance/install.py
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("install")

DOCS = ("README.md", "docs/installation.md")
BRANCH = re.compile(r"--branch\s+(v[0-9]+\.[0-9]+\.[0-9]+)")
# Overridable so the OFFLINE PATH CAN BE EXERCISED. Flipping that branch's
# verdict from False to True survived a mutation, because it only runs when the
# network is down and a normal run never reaches it -- an untested path guarding
# the exact failure this suite exists for.
REMOTE = os.environ.get("REPO_GOVERNOR_INSTALL_REMOTE",
                        "https://github.com/tosin2013/repo-governor")


# A remote that CANNOT resolve, chosen to fail instantly and offline. An
# https URL at a dead port does not fail fast -- git hangs on the connect --
# and the first version of this probe wedged the suite for minutes.
UNREACHABLE = "file:///nonexistent-repo-governor-probe.git"


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def declared():
    """Every `--branch vX.Y.Z` an install document tells a reader to clone."""
    out = {}
    for d in DOCS:
        for tag in BRANCH.findall((ROOT / d).read_text(encoding="utf-8")):
            out.setdefault(tag, []).append(d)
    return out


def remote_tags(url=REMOTE):
    """Tags on the remote, or None when it could not be reached.

    None is NOT an empty set. An unreachable remote and a remote with no tags
    are different facts with different consequences -- the same distinction
    ADR-003 rule 6 makes for providers, applied to a network call.
    """
    try:
        p = subprocess.run(["git", "ls-remote", "--tags", url],
                           capture_output=True, text=True, timeout=60)
    except (OSError, subprocess.TimeoutExpired):
        return None
    if p.returncode != 0:
        return None
    return {ln.rsplit("/", 1)[-1].removesuffix("^{}")
            for ln in p.stdout.splitlines() if "refs/tags/" in ln}


def main(argv):
    fails = 0
    print("The install instructions name a tag that exists\n")

    want = declared()
    fails += check("the install documents declare a branch to clone", bool(want),
                   f"no `--branch vX.Y.Z` found in {list(DOCS)} — this suite would "
                   "otherwise pass by having nothing to check")

    tags = remote_tags()
    if tags is None:
        # UNRESOLVED, not satisfied. This is the whole point of the suite: a
        # green run that could not reach the remote is the vacuous shape the
        # check being replaced already had.
        fails += check("the remote could be reached", False,
                       f"`git ls-remote --tags {REMOTE}` failed. This suite makes a claim "
                       "about the outside world and cannot make it offline. It is in the "
                       "LIVE set for that reason; it does not skip, because a suite that "
                       "passes when it could not check reports safety it never established")
        # The RETURN IS DERIVED FROM `fails`, not hardcoded to 1. It was
        # hardcoded, and a mutation flipping the check above from False to True
        # survived: the return did the failing and the assertion was decorative.
        # A verdict that cannot change the outcome is not a verdict.
        print(f"\nINSTALL: {'CONFORMANT' if not fails else f'NON-CONFORMANT ({fails})'}")
        return 0 if not fails else 1

    missing = {t: d for t, d in want.items() if t not in tags}
    fails += check("every branch named in the install docs resolves on the remote",
                   not missing,
                   f"{missing} — a reader following this gets 'fatal: Remote branch not "
                   f"found in upstream origin'. The remote has {len(tags)} tag(s)")

    # THE CONTROL. Without it the check above passes by asserting nothing --
    # which is exactly what the hermetic version does.
    fake = "v99.99.99-does-not-exist"
    fails += check("a tag that does not exist is reported, not passed over",
                   fake not in tags,
                   "the probe tag resolved, so the negative case proves nothing")

    if os.environ.get("REPO_GOVERNOR_INSTALL_CHILD"):
        # A SPAWNED CHILD NEVER SPAWNS. Without this the suite is one editing
        # mistake away from a fork bomb, and it got there: a stale restore put
        # back a hardcoded REMOTE while leaving the self-spawn, so every child
        # ignored the override, reached the probe, and spawned again. 466
        # processes before it was caught.
        #
        # The guard is STRUCTURAL, not a promise that the file stays correct.
        # Same reasoning as the cycle check in completion.py's split_to
        # resolution: anything that can call itself needs a reason it must stop,
        # and "the code is right" is not one.
        print(f"\nINSTALL: {'CONFORMANT' if not fails else f'NON-CONFORMANT ({fails})'}")
        return 0 if not fails else 1

    # The offline path, exercised END TO END rather than trusted. Asserting that
    # remote_tags() returns None is not enough: the mutation that flipped the
    # offline verdict from False to True survived it, because that branch never
    # runs while the network is up. So this SPAWNS THE SUITE against a remote
    # that cannot answer and requires it to exit non-zero.
    probe = subprocess.run(
        [sys.executable, str(Path(__file__).resolve())],
        capture_output=True, text=True, timeout=60,
        env=dict(os.environ, REPO_GOVERNOR_INSTALL_REMOTE=UNREACHABLE,
                 REPO_GOVERNOR_INSTALL_CHILD="1"))
    fails += check("no network is UNRESOLVED, never satisfied",
                   probe.returncode != 0 and "could be reached" in probe.stdout,
                   f"offline run exited {probe.returncode} — a suite that passes when it "
                   "could not reach the remote reports safety it never established")

    print(f"\n{'INSTALL: CONFORMANT' if not fails else f'INSTALL: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
