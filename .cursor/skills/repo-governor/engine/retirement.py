#!/usr/bin/env python3
"""Retirement obligations, composed across every bound provider (§18, ADR-021).

    obligations (retirement providers)   what blocks removal?
  + repository evidence                  is it still referenced?
  = REMOVAL_READY or a typed refusal

Why this exists: `retirement` was the only bound role with no engine entry
point, so `SKILL.md` told the agent to invoke `adapters/retirement-analysis`
directly. That adapter checks no permission, so the shipped skill documented a
bypass of ADR-021's chokepoint -- an accepted rule contradicted by the file
users actually read.

`retirement` is multi-valued (ADR-013), so this queries EVERY bound provider
and reports each. Obligations accumulate: one provider finding nothing does not
clear an asset, and taking the first answer would let a narrow provider
overrule a broad one.

Usage:  python3 engine/retirement.py <asset-path>
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import bindings as B  # noqa: E402
import manifest as MF  # noqa: E402
import vocabulary as V  # noqa: E402


def evaluate(asset, manifest=None):
    """Return the retirement disposition for one asset."""
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return {"decision": "UNKNOWN", "asset": asset,
                    "unknowns": [{"dimension": "provider", "reason": "MANIFEST_UNREADABLE",
                                  "detail": errs[0], "blocking": True}],
                    "provenance": []}

    bindings, err = B.resolve("retirement", m)
    if err:
        # An unbound role is a typed disposition, not a crash and not a pass.
        # Nothing bound means nothing checked, which must never read as clear.
        return {"decision": "UNKNOWN", "asset": asset,
                "unknowns": [{"dimension": "retirement", "reason": "NO_RETIREMENT_EVIDENCE",
                              "detail": err["error"]["message"],
                              "resolution": "Bind a retirement provider, or accept that removal "
                                            "cannot be cleared here.",
                              "blocking": True}],
                "provenance": []}

    unknowns, provenance, results = [], [], []
    for b in bindings:
        r = B.call("retirement", "obligation_check", {"asset": asset}, manifest=m, binding=b)
        name = b.get("type", b.get("adapter", "?"))
        if not r.get("ok"):
            unknowns.append({"dimension": "retirement", "reason": r["error"]["type"],
                             "detail": f"{name}: {r['error']['message']}", "blocking": True})
            results.append({"provider": name, "clear": None})
            continue
        if r.get("unknown"):
            u = r["unknown"]
            dim, blocking, desc = V.classify(u["reason"], m["condition"]["profile"])
            unknowns.append({**u, "dimension": dim, "blocking": blocking, "meaning": desc,
                             "provider": name})
            results.append({"provider": name, "clear": None})
            continue
        provenance += r.get("provenance", [])
        value = r.get("value") or {}
        # The provider states its own disposition. Anything other than an
        # explicit REMOVAL_READY is treated as not clear -- including a value
        # this engine does not recognise, which must never read as permission.
        results.append({"provider": name,
                        "clear": value.get("disposition") == "REMOVAL_READY", **value})

    blocking = [u for u in unknowns if u.get("blocking")]
    not_clear = [x for x in results if not x["clear"]]

    if blocking or not_clear or not results:
        # ADR-007: an obligation that cannot be resolved is not an obligation
        # that is absent. Static analysis alone can never reach REMOVAL_READY,
        # and no provider answering is the least clear state of all.
        decision = "RETIREMENT_REVIEW"
    else:
        decision = "REMOVAL_READY"

    return {"decision": decision, "asset": asset, "providers": results,
            "unknowns": unknowns, "provenance": provenance}


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    print(json.dumps(evaluate(argv[0]), indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
