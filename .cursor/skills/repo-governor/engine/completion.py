#!/usr/bin/env python3
"""Compose STOP_COMPLETE from three providers (ADR-017, §40).

    authority   (roadmap provider)      is this work authorized?
  + criteria    (acceptance provider)   what counts as done?
  + evaluation  (repository provider)   is it actually done?
  = disposition

No single provider is asked a question it cannot answer. The engine composes.

Deterministic (ADR-002): given the same provider responses this returns the
same disposition every time. It performs no I/O of its own beyond invoking
adapters, and reads no clock.

Every provider is reached through `engine/bindings.py`, which resolves the role
from the manifest and checks the permission before spawning anything (ADR-021).
This module names roles and never adapters; that is what makes the providers
actually interchangeable rather than merely described as such.

Usage:  python3 engine/completion.py <authority-id>
"""

from __future__ import annotations

import hashlib
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import vocabulary as V  # noqa: E402
import manifest as MF  # noqa: E402
import bindings as B  # noqa: E402

ROOT = Path(__file__).resolve().parent.parent
from version import ENGINE_VERSION  # noqa: E402  -- one source, see engine/version.py
try:
    MF_HASH = hashlib.sha256((ROOT / ".repo-governor.json").read_bytes()).hexdigest()[:16]
except OSError:
    MF_HASH = None


# The profile assumed when the manifest cannot be read. LITE escalates nothing,
# so an unreadable manifest never invents a blocking unknown -- it degrades to
# the weakest policy rather than the strictest, which is the safe direction for
# a value the engine failed to learn.
DEFAULT_PROFILE = "GOVERNOR_LITE"


def _profile(manifest):
    """Which condition profile's policy decides what blocks here.

    ADR-006 rule 4 makes profiles declarative policy packs: `policies/*.json`
    names the reasons a profile escalates to blocking, and `vocabulary.classify`
    honours them. This module never asked, so `escalate_to_blocking` was
    unreachable from completion and EVERY profile behaved as GOVERNOR_LITE --
    including GOVERNOR_HIGH_ASSURANCE, which escalates four reasons. Governance
    was therefore weaker than the manifest declared, silently, on exactly the
    repositories that asked for the most (#189). `engine/retirement.py` had it
    right the whole time.
    """
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return DEFAULT_PROFILE
    return ((m or {}).get("condition") or {}).get("profile") or DEFAULT_PROFILE


def _classify(u, profile=DEFAULT_PROFILE):
    """Attach dimension and blocking from the closed vocabulary (gate 7).

    An adapter may not decide whether its own unknown blocks. It names a
    reason; the engine classifies. A reason outside the closed set raises,
    because silently accepting an unclassifiable unknown would let a
    provider bypass the blocking rule entirely.
    """
    dim, blocking, desc = V.classify(u["reason"], profile)
    return {**u, "dimension": dim, "blocking": blocking, "meaning": desc}


def execution_evidence(authority_id, manifest=None):
    """What is happening beneath this authority item. EVIDENCE, never authority.

    ADR-013 makes execution a non-authoritative role and INV-002 says none of it
    confers roadmap authority. So this can enrich a disposition and must never
    change one: `_compose` reads it after the authority decision is already made.

    Why it exists at all: no engine module consulted this role, so the scenario
    the product was built for -- roadmap CANCELLED while subtasks run -- could
    neither fail nor pass, because nothing observed the contradiction (#34).
    Getting AUTHORITY_WITHDRAWN right by never looking is not the same as
    getting it right.

    An unbound role is reported as unbound. This repository deliberately binds
    no execution provider (ADR-022 rule 4), so that is the common path and it
    must read as absence of evidence, never as evidence of absence.
    """
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return {"state": "UNREADABLE", "detail": errs[0]}

    bindings, err = B.resolve("execution", m)
    if err:
        return {"state": "UNBOUND",
                "detail": "no execution provider is bound; nothing is known about work "
                          "beneath this item, which is not the same as knowing there is none"}

    out = {"state": "READ", "active": [], "completed": [], "discoveries": [],
           "unknowns": [], "provenance": []}
    for fn, key in (("get_active_work", "active"), ("get_completed_work", "completed"),
                    ("get_discoveries", "discoveries")):
        r = B.call("execution", fn, {"id": authority_id}, manifest=m)
        if not r.get("ok"):
            # NOT_FOUND means this item has no execution root -- an honest
            # absence. Anything else is a provider problem worth reporting.
            if r.get("error", {}).get("type") != "NOT_FOUND":
                out["unknowns"].append({"function": fn, "reason": r["error"]["type"]})
            continue
        if r.get("unknown"):
            out["unknowns"].append({"function": fn, "reason": r["unknown"]["reason"]})
            continue
        v = r.get("value") or {}
        out[key] = v.get(key) or v.get("discoveries") or []
        # The citations the provider already returned. This loop read `value`
        # and dropped the rest, so a verdict reporting work in flight cited
        # nothing for it -- while every other dimension in this module
        # accumulates (:206 authority, :243 criteria, :294 each check). #217.
        #
        # They ride HERE and not in the top-level chain on purpose. That chain
        # supports the disposition, and INV-002 is that execution state never
        # contributes to one. Merging them would put non-authoritative evidence
        # in the run of citations a reader takes as authority's.
        #
        # Execution evidence is also the least checkable dimension by nature:
        # every other one cites something inside the repository the reader is
        # standing in, while this comes from a tracker, or an export of one,
        # whose currency the verdict cannot state. The citation is the only
        # thing naming which source said so.
        out["provenance"] += r.get("provenance") or []
    if not any(out[k] for k in ("active", "completed", "discoveries")) and not out["unknowns"]:
        out["state"] = "NO_EXECUTION_ROOT"
    return out


# How far a covers chain may be followed before the resolver refuses. A parent
# split into a child that was itself split is ordinary; four levels of that is a
# roadmap problem, not a governance one. The bound is REPORTED when it is hit --
# a resolver that silently stops searching and answers "discharged" declares a
# completion it never established, which is ADR-007's line.
MAX_COVERS_DEPTH = 3


def _resolve_split(covers, manifest, seen, depth):
    """Did the uncovered half actually land? (discharged, note).

    `completion.py` has always told an author to "split the uncovered part into
    its own authority with its own bar" and then had no way to tell that they
    did, so the parent read CONTINUE forever. Six bars on disk carry a `covers`
    block and issue 117's says in as many words that it was split to issue 131
    BY HAND after already reading STOP_COMPLETE.

    THE REFUSAL SURVIVES. A bar still cannot complete by deleting `covers`; it
    has to NAME where the work went, and the names are checked by evaluating
    those authorities rather than by trusting the sentence next to them. An
    outstanding child, an unreachable one, a cycle or an over-deep chain all
    leave the parent incomplete.
    """
    names = covers.get("split_to")
    if not isinstance(names, list) or not names:
        # Missing, malformed, or EMPTY. `split_to: []` is deleting `covers` with
        # extra steps, and `all([])` is True -- the vacuous-truth shape this
        # repository keeps shipping. Absence of a pointer is not a discharge.
        return False, None
    if depth >= MAX_COVERS_DEPTH:
        return False, (f"covers chain exceeds {MAX_COVERS_DEPTH} levels at "
                       f"{names!r}; refusing rather than truncating the search")
    outstanding, notes = [], []
    for child in [str(n) for n in names]:
        if child in seen:
            return False, (f"covers chain returns to {child!r}, which is already being "
                           f"resolved ({' -> '.join(seen)}); a cycle cannot discharge "
                           "anything")
        kid = evaluate(child, manifest, _seen=seen + [child], _depth=depth + 1)
        d = kid.get("decision")
        if d != "STOP_COMPLETE":
            outstanding.append(f"{child} is {d}")
        notes.append(f"{child}={d}")
    if outstanding:
        return False, ("the uncovered half was split, and has not landed: "
                       + "; ".join(outstanding))
    return True, "the uncovered half was split and every part is complete: " + ", ".join(notes)


def evaluate(authority_id, manifest=None, _seen=None, _depth=0):
    """Return the full governance decision for the completion axis.

    Named by role throughout. Which adapter answers `roadmap_authority` is the
    manifest's business, not this function's.

    `_seen` and `_depth` exist only for `covers.split_to` resolution and are
    never supplied by a caller from outside this module.
    """
    _seen = _seen or [str(authority_id)]
    unknowns = []
    provenance = []
    profile = _profile(manifest)

    # 1. authority — is this authorized at all?
    auth = B.call("roadmap_authority", "get_authority", {"id": authority_id}, manifest=manifest)
    if not auth.get("ok"):
        return {"decision": "UNKNOWN", "authority_id": authority_id,
                "unknowns": [{"dimension": "authority", "reason": auth["error"]["type"],
                              "detail": auth["error"]["message"], "blocking": True}],
                "provenance": []}
    if auth.get("unknown"):
        u = _classify(auth["unknown"], profile)
        return {"decision": "UNKNOWN", "authority_id": authority_id,
                "unknowns": [u], "provenance": []}
    provenance += auth.get("provenance", [])
    authority = auth["value"]["authority"]
    if authority in ("CANCELLED", "WITHDRAWN", "REJECTED"):
        ex = execution_evidence(authority_id, manifest)
        out = {"decision": "AUTHORITY_WITHDRAWN", "authority_id": authority_id,
               "authority": authority, "unknowns": [], "provenance": provenance,
               "execution": ex}
        # The disposition does not change -- withdrawn is withdrawn. But work
        # running against withdrawn authority is the actionable fact, and an
        # engine that never looked could not report it.
        if ex.get("active"):
            out["execution_in_flight"] = [t.get("id") for t in ex["active"]]
            out["detail"] = (f"{len(ex['active'])} execution task(s) are in flight beneath an "
                             "item whose authority is withdrawn. They must stop; execution state "
                             "does not reinstate authority (INV-002).")
        return out
    if authority == "ADMITTED":
        return {"decision": "NO_EXECUTION_AUTHORITY", "authority_id": authority_id,
                "authority": authority,
                "detail": "Admitted to the roadmap but not cleared to execute (INV-002).",
                "unknowns": [], "provenance": provenance}

    # 2. criteria — what counts as done?
    crit = B.call("acceptance_criteria", "get_criteria", {"id": authority_id}, manifest=manifest)
    if not crit.get("ok"):
        return {"decision": "UNKNOWN", "authority_id": authority_id, "authority": authority,
                "unknowns": [{"dimension": "acceptance", "reason": crit["error"]["type"],
                              "detail": crit["error"]["message"], "blocking": True}],
                "provenance": provenance}
    if crit.get("unknown"):
        # No criteria declared => no completion bar => CONTINUE, not STOP.
        u = _classify(crit["unknown"], profile)
        unknowns.append(u)
        return {"decision": "CONTINUE", "authority_id": authority_id, "authority": authority,
                "stop_condition": {"acceptance_conditions_satisfied": "UNKNOWN"},
                "unknowns": unknowns, "provenance": provenance,
                "execution": execution_evidence(authority_id, manifest)}
    provenance += crit.get("provenance", [])
    criteria = crit["value"]["criteria"]

    # A DECLARED BUT EMPTY BAR IS NOT A SATISFIED BAR.
    #
    # Without this, `criteria: []` reaches the loop below, produces zero
    # results, and both "is anything unresolved?" and "is anything unmet?"
    # are false -- so the else branch declares STOP_COMPLETE by vacuous
    # quantification. Demonstrated on a live authority id: emptying the list
    # turned CONTINUE into STOP_COMPLETE with satisfied=true, on zero
    # evidence.
    #
    # Section 40 is the completion FIREWALL. A firewall that opens when handed
    # nothing to check is not one, and this is the exact shape an unedited
    # template would have had on first contact (issue 58).
    if not criteria:
        # Classified, not asserted. This dict carried "blocking": False as a
        # literal and returned before any classification, so GOVERNOR_FULL and
        # GOVERNOR_HIGH_ASSURANCE -- both of which escalate NO_CRITERIA_DECLARED
        # -- were silently overruled by a hardcoded value three lines below the
        # comment describing the firewall (#189).
        unknowns.append(_classify({
            "reason": "NO_CRITERIA_DECLARED",
            "detail": (f"The acceptance record for {authority_id} exists but declares no "
                       "criteria. An empty bar is not a met bar; nothing has been stated "
                       "that completion could be checked against."),
            "resolution": "Declare at least one criterion, or accept that this work has no "
                          "completion bar and will never read STOP_COMPLETE.",
        }, profile))
        return {"decision": "CONTINUE", "authority_id": authority_id, "authority": authority,
                "criteria": [],
                "stop_condition": {"acceptance_conditions_satisfied": "UNKNOWN"},
                "unknowns": unknowns, "provenance": provenance,
                "execution": execution_evidence(authority_id, manifest)}

    # 3. evaluation — is it actually done?
    results = []
    for c in criteria:
        ev = B.call("repository", "evaluate_check",
                    {"check": c["check"], "target": c["target"]}, manifest=manifest)
        if not ev.get("ok"):
            unknowns.append({"dimension": "evidence", "reason": ev["error"]["type"],
                             "detail": f"{c['check']} {c['target']}: {ev['error']['message']}",
                             "blocking": True})
            results.append({**c, "satisfied": None})
            continue
        if ev.get("unknown"):
            u = _classify(ev["unknown"], profile)
            unknowns.append(u)
            results.append({**c, "satisfied": None})
            continue
        provenance += ev.get("provenance", [])
        results.append({**c, "satisfied": ev["value"]["satisfied"]})

    unresolved = [r for r in results if r["satisfied"] is None]
    unmet = [r for r in results if r["satisfied"] is False]
    covers = (crit["value"] or {}).get("covers")

    if unresolved:
        # ADR-007: must not declare completion it cannot verify.
        decision, satisfied = "UNKNOWN", "UNKNOWN"
    elif unmet:
        decision, satisfied = "CONTINUE", False
    elif covers:
        # Every declared criterion is met, and the bar itself says it covers
        # only part of this authority. Completion is therefore not something
        # this bar can establish, whatever its criteria say -- UNLESS the bar
        # names where the uncovered half went and every part of it has landed.
        discharged, note = _resolve_split(covers, manifest, _seen, _depth)
        if discharged:
            decision, satisfied = "STOP_COMPLETE", True
            unknowns.append(_classify({
                "reason": "BAR_COVERS_PART_DISCHARGED",
                "detail": f"The bar covers {covers.get('declared')!r}. {note}",
                "resolution": "None needed; recorded so the discharge is visible.",
            }, profile))
        else:
            # CONTINUE rather than a new disposition: STOP_PARTIAL would need
            # section 32 and an ADR, and the safe direction needs no new
            # vocabulary. The work continues, which is true.
            decision, satisfied = "CONTINUE", False
            unknowns.append(_classify({
                "reason": "BAR_COVERS_PART",
                "detail": (f"The bar covers {covers.get('declared')!r}. NOT covered: "
                           f"{covers.get('uncovered')!r}. Every criterion passed, so the "
                           "covered part is done; the item is not."
                           + (f" {note}" if note else "")),
                "resolution": ("Split the uncovered part into its own authority with its "
                               "own bar and name it in 'split_to', or extend this bar to "
                               "cover it. Removing 'covers' without doing either would "
                               "declare completion the criteria do not establish, and an "
                               "empty 'split_to' is the same thing with extra steps."),
            }, profile))
    else:
        decision, satisfied = "STOP_COMPLETE", True

    out = {"decision": decision, "authority_id": authority_id, "authority": authority,
           "criteria": results,
           "stop_condition": {"acceptance_conditions_satisfied": satisfied},
           "unknowns": unknowns, "provenance": provenance,
           "execution": execution_evidence(authority_id, manifest)}
    # Scenario 3: work finished, and something was found along the way. The
    # discoveries are surfaced with the disposition rather than left for the
    # agent to remember -- but they are surfaced as CAPTURE_ONLY candidates, and
    # STOP_COMPLETE is unaffected by their existence (§40).
    disc = (out["execution"] or {}).get("discoveries") or []
    if disc and decision == "STOP_COMPLETE":
        out["captured"] = [{"id": d.get("id"), "type": d.get("type"),
                            "disposition": "CAPTURE_ONLY"} for d in disc]
        out["detail"] = (f"{len(disc)} discovery(ies) recorded beneath completed work. Each is "
                         "CAPTURE_ONLY: completion exhausts this authorization, and a discovery "
                         "made under it inherits no authority from it (INV-001, §40).")
    return out


def _repo_is_public():
    """Drives the redaction default. Unknown visibility is treated as public --
    failing conservatively, per §51."""
    try:
        p = subprocess.run(["gh", "repo", "view", "--json", "visibility", "-q", ".visibility"],
                           capture_output=True, text=True, cwd=ROOT, timeout=20)
        if p.returncode == 0:
            return p.stdout.strip().upper() != "PRIVATE"
    except (FileNotFoundError, subprocess.TimeoutExpired):
        pass
    return True


def _redact(decision, provenance, public):
    """Hash plus typed facts plus explicit markers (ADR-019).

    Never a silent omission: an omitted snapshot is indistinguishable from
    there having been no snapshot, which is the absence-versus-unknown
    confusion ADR-003 rule 6 forbids in adapters.
    """
    snapshot = json.dumps(provenance, sort_keys=True)
    sha = hashlib.sha256(snapshot.encode()).hexdigest()
    facts = {k: decision.get(k) for k in ("decision", "authority") if decision.get(k) is not None}
    if not public:
        return sha, facts, False, None
    return sha, facts, True, ["provider_snapshots"]


def record(decision, manifest=None):
    """Append the decision to the bound decision-history store, if permitted.

    Returns a dict describing what happened. Recording NEVER changes the
    disposition -- it is a side effect after the fact, so ADR-002's pure
    evaluation is untouched.

    The decision id is a content hash, not a timestamp. ADR-002 forbids clock
    reads in the evaluation path, and a content-derived id also makes recording
    idempotent: re-evaluating an unchanged state rewrites the same row.
    """
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return {"recorded": False, "why": "manifest invalid; refusing to record"}

    # Which backend can be written to is a question for the adapter's advertised
    # writers, not for its name. `describe` gates that list on a real writability
    # probe (#17), so a reachable-but-read-only store is correctly not chosen.
    binding, err = B.writer_for("decision_history", "record_decision", m)
    if err:
        return {"recorded": False, "why": err["error"]["message"]}

    public = _repo_is_public()
    sha, facts, redacted, fields = _redact(decision, decision.get("provenance", []), public)
    body = json.dumps(decision, sort_keys=True)
    did = "d-" + hashlib.sha256(body.encode()).hexdigest()[:24]

    disp = {"STOP_COMPLETE": "ACCEPTED", "CONTINUE": "ACCEPTED",
            "AUTHORITY_WITHDRAWN": "CANCELLED", "NO_EXECUTION_AUTHORITY": "DEFERRED",
            "UNKNOWN": "DEFERRED"}.get(decision["decision"])
    if disp is None:
        return {"recorded": False, "why": f"no decision-history mapping for {decision['decision']}"}

    kw = {"decision_id": did, "authority_id": decision["authority_id"],
          "disposition": disp, "reason": f"engine decision {decision['decision']}",
          "engine_version": ENGINE_VERSION, "manifest_hash": MF_HASH or "",
          "snapshot_sha256": sha, "typed_facts": json.dumps(facts, sort_keys=True),
          "redacted": str(redacted).lower(),
          "fields_redacted": json.dumps(fields) if fields else ""}
    r = B.call("decision_history", "record_decision", kw, verb="write",
               manifest=m, binding=binding)
    if not r.get("ok"):
        return {"recorded": False, "why": r.get("error", {}).get("message", "write failed")}
    return {"recorded": True, "decision_id": did, "redacted": redacted,
            "fields_redacted": fields, "committed": False}


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    authority_id = argv[0]
    # No --roadmap flag and no adapter-specific environment. Which provider
    # answers a role is declared in the manifest; an override here would be a
    # second binding surface, which is the shadow-system shape ADR-022 forbids.
    result = evaluate(authority_id)
    if "--record" in argv:
        result["record"] = record(result)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
