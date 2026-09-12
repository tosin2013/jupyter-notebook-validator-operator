#!/usr/bin/env python3
"""Evidence union for multi-valued roles — ADR-013 step 4 (issue 154).

`engine/bindings.py::call()` ended with `bindings[0]`. That is correct for a
single-valued role and silently discards evidence on a multi-valued one, so a
second `architecture` provider validated, appeared in `engine/status.py`, and
was never asked anything. The cost was not theoretical: this repository binds
two `decision_history` providers, the second deliberately, and it had never
been read.

The suite is ordered by what breaks worst if it is wrong.

  1. ONE PROVIDER IS UNCHANGED. The file-based adapter stays the default and
     almost every repository has exactly one, so a fan-out that reshapes the
     single-provider path breaks every existing install to serve a case nobody
     has yet. ADR-008 C7.
  2. EVERY STORE IS READ. The consequence half: §39 rediscovery is decided
     against decision_history, so a store that is never read is evidence that
     cannot stop work.
  3. DISJOINT EVIDENCE UNIONS AND DOES NOT ESCALATE. ADR-013's own Consequences
     warn that multi-valued architecture "brush[es] against §54's
     over-escalation failure condition". Different constraints are the normal
     case, not a finding. This is the check that stops the fix becoming the bug.
  4. CONTRADICTION ESCALATES, BOTH CITED, AND DOES NOT BLOCK. Rule 3 says the
     engine does not pick; rule 2 scopes halting to single-valued roles.

Usage:  python3 conformance/union.py
"""

from __future__ import annotations

import sys
from pathlib import Path

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("union")

ROOT = Path(__file__).resolve().parent.parent
FIX = ROOT / "conformance" / "fixtures" / "union"
sys.path.insert(0, str(ROOT / "engine"))

import bindings as B  # noqa: E402
import envelope as E  # noqa: E402
import manifest as MF  # noqa: E402

# The envelope's key set BEFORE the union existed, read off dc2c26a. Frozen on
# purpose: the union adds keys only when more than one provider is bound, and
# this is what catches a key leaking onto the single-provider path.
PRE_UNION_KEYS = {
    "authority_id", "authority", "required_outcome", "in_scope",
    "necessary_incidental_work", "non_goals", "architecture_constraints",
    "discovery_policy", "acceptance_conditions", "stop_condition",
    "unresolved", "thinness",
}


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def load(name):
    m, errs = MF.load(FIX / f"manifest-{name}.json")
    if errs:
        raise SystemExit(f"fixture manifest {name} is invalid: {errs}")
    return m


def main():
    fails = 0

    # ---------------------------------------------------------------- 1
    print("A single-provider role behaves exactly as it did before the union\n")
    single = load("single")
    env = E.compile_envelope("154", manifest=single)

    fails += check("the envelope grows no key when one provider is bound",
                   set(env) == PRE_UNION_KEYS,
                   f"unexpected={sorted(set(env) - PRE_UNION_KEYS)} "
                   f"missing={sorted(PRE_UNION_KEYS - set(env))}")

    # The union must agree with a DIRECT single call, not merely with itself.
    # Comparing the union against the union is the vacuous shape this repository
    # has shipped six times: a check that cannot fail.
    direct = B.call("architecture", "get_constraints", {"id": "154"}, manifest=single)
    direct_ids = [c["id"] for c in (direct.get("value") or {}).get("constraints") or []]
    fails += check("unioned constraints equal what one direct call returns, in order",
                   env["architecture_constraints"] == direct_ids and len(direct_ids) > 1,
                   f"union={env['architecture_constraints']} direct={direct_ids} -- "
                   "the fixture must hold MORE THAN ONE decision or this cannot "
                   "detect a reordering, which is what it is for")

    fails += check("call_all on a single binding returns exactly one result",
                   len((B.call_all("architecture", "get_constraints",
                                   {"id": "154"}, manifest=single)[0] or [])) == 1)

    # ---------------------------------------------------------------- 2
    print("\nEvery bound decision_history store is read, not just the first\n")
    disjoint = load("disjoint")
    # ARCHITECTURE_IMPLICATION, deliberately not BUG. `BUG` defaults to
    # CAPTURE_ONLY with or without a prior decision, so asserting CAPTURE_ONLY on
    # a BUG is green whether or not the second store was ever read -- the
    # vacuous shape this repository has shipped six times. ARCHITECTURE_IMPLICATION
    # routes to ARCHITECTURE_REVIEW, so §39 firing MOVES the disposition and the
    # assertion can fail.
    DTYPE = "ARCHITECTURE_IMPLICATION"
    want = E.discovery_id("154", DTYPE, "union-fixture-target")
    prior, _ = E.prior_decision("154", DTYPE, "union-fixture-target", manifest=disjoint)

    fails += check("a decision held ONLY in the second store is found",
                   bool(prior) and prior.get("decision_id") == want,
                   "prior_decision read bindings[0] and the record lives in bindings[1]; "
                   "before issue 154 this returned None and §39 could not fire")

    # The consequence, not just the read. C1 proves the store is reachable;
    # this proves the reading reaches a verdict.
    denv0 = E.compile_envelope("154", manifest=disjoint)
    ruled = E.classify(denv0, DTYPE, "union-fixture-target", prior=prior)
    fails += check("§39 fires on it: the rediscovery is CAPTURE_ONLY",
                   ruled["disposition"] == "CAPTURE_ONLY" and ruled["authority"] == "NONE",
                   f"got {ruled['disposition']}/{ruled['authority']}")
    # The control for the check above: without the prior record this same
    # discovery is ARCHITECTURE_REVIEW. If it were CAPTURE_ONLY either way the
    # assertion would be measuring nothing.
    unruled = E.classify(denv0, DTYPE, "union-fixture-target", prior=None)
    fails += check("...and it would be ARCHITECTURE_REVIEW without the record",
                   unruled["disposition"] == "ARCHITECTURE_REVIEW",
                   f"got {unruled['disposition']} -- the check above cannot fail "
                   "if the disposition is the same with and without the record")
    fails += check("the verdict cites the recorded decision",
                   (ruled.get("prior_decision") or {}).get("decision_id") == want)

    # A rediscovery the stores have never seen must be unaffected. Without this,
    # a prior_decision that returned a record for EVERYTHING would pass the two
    # checks above -- the failure mode is over-blocking, and it is invisible
    # unless the negative case is asserted.
    none_prior, _ = E.prior_decision("154", DTYPE, "never-decided", manifest=disjoint)
    fails += check("an undecided target still returns no prior decision",
                   none_prior is None,
                   "a store scan that matches anything would block all work and "
                   "pass every positive check above")

    # ---------------------------------------------------------------- 3
    print("\nDisjoint evidence unions, and does NOT escalate (§54)\n")
    denv = E.compile_envelope("154", manifest=disjoint)
    # Provider-major order, not globally sorted: adrs-x holds ADR-0001 and
    # ADR-0003, adrs-y holds ADR-0002. A union that sorted the result would put
    # ADR-0002 in the middle, which is why the fixture ids are out of sequence.
    fails += check("both providers contribute their constraints, in binding order",
                   denv["architecture_constraints"] == ["ADR-0001", "ADR-0003", "ADR-0002"],
                   f"got {denv['architecture_constraints']} -- bindings[0] alone "
                   "returns only adrs-x, and a sorted union returns 1,2,3")
    fails += check("each provider's evidence is attributed separately",
                   [e["provider"] for e in denv.get("architecture_evidence") or []]
                   == ["adr#0", "adr#1"])
    fails += check("DIFFERENT constraints are a union, not a finding",
                   "architecture_review" not in denv,
                   "escalating on differing constraint sets is §54's over-escalation "
                   "failure condition, and it is the normal case for a role whose "
                   "whole purpose is that evidence accumulates")

    # ---------------------------------------------------------------- 4
    print("\nContradiction escalates, cites both, and does not block\n")
    contra = load("contradiction")
    cenv = E.compile_envelope("154", manifest=contra)
    rev = cenv.get("architecture_review") or {}
    items = rev.get("contradictions") or []

    fails += check("an id active in one provider and superseded in another is a finding",
                   len(items) == 1 and items[0]["id"] == "ADR-0001",
                   f"got {items}")
    fails += check("the disposition is ARCHITECTURE_REVIEW",
                   rev.get("disposition") == "ARCHITECTURE_REVIEW")
    fails += check("BOTH providers are cited — the engine does not pick (rule 3)",
                   bool(items) and items[0].get("active_in") == ["adr#0"]
                   and items[0].get("superseded_in") == ["adr#1"],
                   f"got {items[0] if items else None}")
    fails += check("it does not block: rule 2 scopes halting to single-valued roles",
                   rev.get("blocking") is False)
    fails += check("the contradicting id still appears in the unioned constraints",
                   "ADR-0001" in cenv["architecture_constraints"],
                   "a finding must not silently drop the evidence it is about")

    # The narrowness is load-bearing and therefore asserted, not just documented.
    fails += check("no finding is raised for the disjoint case",
                   not (E.compile_envelope("154", manifest=disjoint)
                        .get("architecture_review")))

    print(f"\n{'STATUS: CONFORMANT' if not fails else f'STATUS: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
