#!/usr/bin/env python3
"""Execution state stays subordinate to roadmap authority (§62, ADR-013, INV-002).

Stage E of the real-repository validation program. The property under test is
not that execution is read -- it is that reading it **changes nothing about
authority**. An execution provider that could promote work would be a second
roadmap, which is ADR-022's failure with a different label.

Before #34 no engine module consulted this role at all, so the scenario the
product exists for -- the tracker says READY while the roadmap says cancelled --
could neither pass nor fail. Getting `AUTHORITY_WITHDRAWN` right by never
looking is not the same as getting it right, and only one of those survives a
provider being bound.

Usage:  python3 conformance/execution.py
"""

from __future__ import annotations

import copy
import json
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import _preflight  # noqa: E402
sys.path.insert(0, str(ROOT / "engine"))
import completion as C  # noqa: E402
import manifest as MF  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("execution")

BASE = json.loads((ROOT / ".repo-governor.json").read_text())


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"    {detail}" if detail and not ok else ""))
    return 0 if ok else 1


# Two structurally different execution providers over the same scenario. One
# reads a hand-authored JSON map; the other reads a `bd export` JSONL and joins
# on Beads' own external_ref. INV-002 is a property of the ROLE, so a suite that
# only ever exercised one provider could not tell the property from that
# provider's behaviour.
PROVIDERS = {
    "execution-file": {
        "type": "execution-file", "adapter": "adapters/execution-file", "contract_version": 1,
        "env": {"REPO_GOVERNOR_EXECUTION": "conformance/fixtures/execution.json"}},
    "beads": {
        "type": "beads", "adapter": "adapters/beads", "contract_version": 1,
        "env": {"REPO_GOVERNOR_BEADS": "conformance/fixtures/beads-issues.jsonl"}},
}


def bound_manifest(with_execution=True, provider="execution-file"):
    """A manifest binding the file providers, so execution has something to read."""
    d = copy.deepcopy(BASE)
    d["providers"]["roadmap_authority"] = {
        "type": "file-roadmap", "adapter": "adapters/file-roadmap", "contract_version": 1,
        "env": {"REPO_GOVERNOR_ROADMAP": "conformance/fixtures/roadmap.json"}}
    d["providers"]["acceptance_criteria"]["env"] = {
        "REPO_GOVERNOR_ACCEPTANCE_DIR": "conformance/fixtures/acceptance"}
    if with_execution:
        d["providers"]["execution"] = PROVIDERS[provider]
        d["permissions"]["execution"] = {"read": True}
    f = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False)
    json.dump(d, f)
    f.close()
    m, errs = MF.load(f.name)
    Path(f.name).unlink()
    assert not errs, errs
    return m


def main():
    absent = _preflight.banner()
    fails = 0
    m = bound_manifest()

    print("Roadmap authority overrides execution state\n")

    r = C.evaluate("CANCELLED-1", manifest=m)
    fails += check("cancelled item with subtasks running is AUTHORITY_WITHDRAWN",
                   r["decision"] == "AUTHORITY_WITHDRAWN", r["decision"])
    fails += check("the in-flight work is REPORTED, not silently ignored",
                   bool(r.get("execution_in_flight")), str(r.get("execution")))
    fails += check("execution never appears as an authority value",
                   r.get("authority") == "CANCELLED", str(r.get("authority")))

    print("\nAbsent execution state invents nothing\n")

    r = C.evaluate("RBAC-1", manifest=m)
    fails += check("no execution root does not fabricate work",
                   (r.get("execution") or {}).get("state") == "NO_EXECUTION_ROOT",
                   str((r.get("execution") or {}).get("state")))
    fails += check("and does not change the disposition", r["decision"] in ("CONTINUE", "STOP_COMPLETE"),
                   r["decision"])

    print("\nCompletion plus a discovery beneath it\n")

    r = C.evaluate("AUTHORIZED-1", manifest=m)
    fails += check("finished work is STOP_COMPLETE", r["decision"] == "STOP_COMPLETE", r["decision"])
    cap = r.get("captured") or []
    fails += check("the discovery is surfaced with the decision", bool(cap), str(r.get("execution")))
    fails += check("and every one is CAPTURE_ONLY",
                   all(c["disposition"] == "CAPTURE_ONLY" for c in cap), str(cap))
    fails += check("completed subtasks are visible as evidence",
                   len((r.get("execution") or {}).get("completed") or []) > 0)

    print("\nAn unbound role is absence of evidence, not evidence of absence\n")

    r = C.evaluate("AUTHORIZED-1", manifest=bound_manifest(with_execution=False))
    ex = r.get("execution") or {}
    fails += check("unbound execution reports UNBOUND", ex.get("state") == "UNBOUND", str(ex))
    fails += check("and the disposition is unchanged by its absence",
                   r["decision"] == "STOP_COMPLETE", r["decision"])
    fails += check("no discoveries are claimed when nothing was read", not r.get("captured"))

    print("\nThe same subordination, through Beads (#196)\n")

    # #33 recorded a no-go on this adapter, and one of its load-bearing reasons
    # was that no engine module consulted the role -- so the scenario could
    # neither pass nor fail. #34 made the role consulted; this makes the
    # scenario run through a real tracker's own export rather than a file
    # written to make it pass.
    mb = bound_manifest(provider="beads")

    rb = C.evaluate("CANCELLED-1", manifest=mb)
    exb = rb.get("execution") or {}
    fails += check("cancelled item with Beads subtasks running is AUTHORITY_WITHDRAWN",
                   rb["decision"] == "AUTHORITY_WITHDRAWN", rb["decision"])
    fails += check("and the running subtask is VISIBLE, not merely overridden",
                   len(exb.get("active") or []) >= 1,
                   f"active={exb.get('active')} -- overriding evidence nobody read is "
                   "not the same as overriding evidence")

    ra = C.evaluate("AUTHORIZED-1", manifest=mb)
    exa = ra.get("execution") or {}
    fails += check("an authorized item reads its Beads root",
                   exa.get("state") == "READ" and len(exa.get("completed") or []) >= 1,
                   str(exa)[:200])
    fails += check("a discovered-from edge arrives as a CAPTURE_ONLY discovery",
                   any(x.get("default_disposition") == "CAPTURE_ONLY"
                       for x in (exa.get("discoveries") or [])),
                   f"discoveries={exa.get('discoveries')} -- \u00a732 and INV-001 mean a "
                   "discovery never arrives executable")

    # THE CONTROL, and the reason this section is worth its runtime. Both
    # providers must reach the SAME disposition from structurally different
    # sources. If they diverge, one of them is contributing authority, which is
    # exactly what INV-002 forbids and what a single-provider suite cannot see.
    mf = bound_manifest(provider="execution-file")
    for wid in ("CANCELLED-1", "AUTHORIZED-1"):
        a = C.evaluate(wid, manifest=mf)["decision"]
        b = C.evaluate(wid, manifest=mb)["decision"]
        fails += check(f"{wid}: both execution providers reach the same disposition ({a})",
                       a == b, f"execution-file={a} beads={b} -- an execution provider that "
                               "changes the disposition is a second roadmap")
    print("\nExecution evidence cites its source (#217)\n")

    # execution_evidence read `value` and dropped `provenance`, so a verdict
    # reporting work in flight cited nothing for it -- while every other
    # dimension in completion.py accumulates. Execution is the dimension where
    # that costs most: every other one cites something inside the repository
    # the reader is standing in, and this comes from a tracker whose currency
    # the verdict cannot state.
    #
    # TWO INPUTS, TWO OUTPUTS. A fix that unconditionally appends a citation
    # passes any check written only on the populated case, so the empty case is
    # asserted beside it. That is the same requirement #200 makes of the state
    # collapse, and for the same reason.
    mprov = bound_manifest()
    rp = C.evaluate("CANCELLED-1", manifest=mprov)
    ep = rp.get("execution") or {}
    cites = ep.get("provenance") or []
    fails += check("a populated execution block cites its source",
                   bool(cites) and all(c.get("source") and c.get("ref") for c in cites),
                   f"provenance={cites}")
    fails += check("and the citation names the provider that answered",
                   {c.get("source") for c in cites} == {"execution-file"},
                   f"sources={ {c.get('source') for c in cites} }")

    rn = C.evaluate("THIN-1", manifest=mprov)
    en = rn.get("execution") or {}
    fails += check("an item with no execution root cites nothing",
                   not (en.get("provenance") or []),
                   f"{en.get('provenance')} -- a fix that always appends would pass the "
                   "check above while citing evidence it never read")
    fails += check("control: the two cases really are different states",
                   ep.get("state") == "READ" and en.get("state") == "NO_EXECUTION_ROOT",
                   f"{ep.get('state')} vs {en.get('state')} -- if both were the same "
                   "state the pair above would prove nothing")

    print(f"\n{'EXECUTION SUBORDINATION: CONFORMANT' if not fails else f'EXECUTION SUBORDINATION: NON-CONFORMANT ({fails})'}")
    if fails:
        _preflight.attribute(absent)
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
