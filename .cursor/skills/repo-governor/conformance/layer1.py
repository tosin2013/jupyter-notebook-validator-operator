#!/usr/bin/env python3
"""ADR-008 Layer 1 conformance — contract tests every adapter must pass.

Layer 1 asks "does this adapter honour the contract?", not "is it correct
about the world". Layer 2 (cross-provider equivalence — the thesis test,
issue #1) is separate and not implemented here.

Checks, from ADR-008:
  C1  describe is well-formed and declares role + contract_version
  C2  honest capability advertisement — every claimed capability is exercised
  C3  typed failure — an unreachable backend errors, never returns empty
  C4  absence vs unknown are distinct
  C5  provenance on every fact
  C6  unsupported function is rejected, not silently absorbed
  C7  determinism — same input, byte-identical output
  C8  an unreachable transport advertises NO capabilities (LSP missing=absent)
  C9  a reachable but UNWRITABLE transport advertises no writers (issue #17)

Usage:  python3 conformance/layer1.py [adapter ...]
"""

from __future__ import annotations

import json
import os
import subprocess
import tempfile
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import _preflight  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("layer1")

# adapter -> (role, probe args, capability->function map, env that breaks the backend)
SUITE = {
    "adapters/git": {
        "role": "repository",
        "probe": {},
        "capability_fn": {
            "git_state": ("get_state", {}),
            "file_listing": ("get_files", {}),
            "dependency_manifests": ("get_manifests", {}),
            "evaluate_check": ("evaluate_check", {"check": "file_exists", "target": "README.md"}),
        },
        "break_env": {"REPO_GOVERNOR_REPO": "/nonexistent-path-xyz"},
        "unknown_fn": ("get_entry_points", {}),
    },
    "adapters/adr": {
        "role": "architecture",
        "probe": {},
        "capability_fn": {
            "active_decisions": ("get_active_decisions", {}),
            "superseded_decisions": ("get_superseded_decisions", {}),
            "constraints": ("get_constraints", {}),
            "provenance": ("get_provenance", {}),
        },
        "break_env": {"REPO_GOVERNOR_ADR_DIR": "/nonexistent-path-xyz"},
        "unknown_fn": None,
    },
    "adapters/speckit": {
        "role": "architecture",
        "env": {"REPO_GOVERNOR_SPECKIT_DIR": "conformance/fixtures/speckit/authored/.specify"},
        "probe": {},
        "capability_fn": {
            "constraints": ("get_constraints", {}),
            "specs": ("get_specs", {}),
            "provenance": ("get_provenance", {}),
        },
        "break_env": {"REPO_GOVERNOR_SPECKIT_DIR": "/nonexistent-path-xyz"},
        "unknown_fn": None,
    },
    "adapters/openspec": {
        "role": "architecture",
        "env": {"REPO_GOVERNOR_OPENSPEC_DIR": "conformance/fixtures/openspec/full"},
        "probe": {},
        "capability_fn": {
            "active_decisions": ("get_active_decisions", {}),
            "completed_changes": ("get_completed_changes", {}),
            "constraints": ("get_constraints", {}),
            "specs": ("get_specs", {}),
            "provenance": ("get_provenance", {}),
        },
        "break_env": {"REPO_GOVERNOR_OPENSPEC_DIR": "/nonexistent-path-xyz"},
        "unknown_fn": None,
    },
    "adapters/file-roadmap": {
        "malformed": ("REPO_GOVERNOR_ROADMAP", '{"items":[{"id":"AUTHORIZED-1"}]}'),
        "role": "roadmap_authority",
        "probe": {"id": "AUTHORIZED-1"},
        "capability_fn": {
            "work_lookup": ("get_work", {"id": "AUTHORIZED-1"}),
            "status": ("get_status", {"id": "AUTHORIZED-1"}),
            "authority": ("get_authority", {"id": "AUTHORIZED-1"}),
            "scope": ("get_scope", {"id": "AUTHORIZED-1"}),
            "non_goals": ("get_non_goals", {"id": "AUTHORIZED-1"}),
            "acceptance_conditions": ("get_acceptance_conditions", {"id": "AUTHORIZED-1"}),
            "decision_history": ("get_decision_history", {"id": "RBAC-1"}),
            "cancellation_detection": ("get_authority", {"id": "CANCELLED-1"}),
        },
        "env": {"REPO_GOVERNOR_ROADMAP": "conformance/fixtures/roadmap.json"},
        "break_env": {"REPO_GOVERNOR_ROADMAP": "/nonexistent-path-xyz.json"},
        "unknown_fn": ("get_non_goals", {"id": "THIN-1"}),
        "absence_fn": ("get_work", {"id": "NO-SUCH-ITEM"}),
        "malformed_fn": ("get_authority", {"id": "BADAUTH-1"}),
    },
    "adapters/github-projects": {
        "role": "roadmap_authority",
        "probe": {"id": "1"},
        "capability_fn": {
            "work_lookup": ("get_work", {"id": "1"}),
            "status": ("get_status", {"id": "1"}),
            "authority": ("get_authority", {"id": "1"}),
            "cancellation_detection": ("get_authority", {"id": "3"}),
            "decision_history": ("get_decision_history", {"id": "3"}),
        },
        # Fixture mode by default so C7 determinism is assertable (ADR-008 rule 1).
        "env": {"REPO_GOVERNOR_GH_FIXTURE": "conformance/fixtures/github-projects.json",
                "REPO_GOVERNOR_GH_ADMISSION": "project_status"},
        "break_env": {"REPO_GOVERNOR_GH_FIXTURE": "/nonexistent-fixture-xyz.json"},
        "unknown_fn": ("get_non_goals", {"id": "1"}),
        "absence_fn": ("get_work", {"id": "9999"}),
    },
    "adapters/acceptance-file": {
        "role": "acceptance_criteria",
        "probe": {"id": "AUTHORIZED-1"},
        "capability_fn": {
            "criteria": ("get_criteria", {"id": "AUTHORIZED-1"}),
            "provenance": ("get_provenance", {}),
            "machine_checkable": ("get_criteria", {"id": "AUTHORIZED-1"}),
        },
        "env": {"REPO_GOVERNOR_ACCEPTANCE_DIR": "conformance/fixtures/acceptance"},
        "break_env": {"REPO_GOVERNOR_ACCEPTANCE_DIR": "/nonexistent-path-xyz"},
        "unknown_fn": ("get_criteria", {"id": "NOSUCH"}),
    },
    "adapters/execution-file": {
        "malformed": ("REPO_GOVERNOR_EXECUTION", '{"roots":"not-a-mapping"}'),
        "role": "execution",
        "probe": {"id": "AUTHORIZED-1"},
        "capability_fn": {
            "execution_root": ("find_execution_root", {"id": "AUTHORIZED-1"}),
            "tasks": ("get_tasks", {"id": "AUTHORIZED-1"}),
            "dependencies": ("get_dependencies", {"id": "AUTHORIZED-1"}),
            "completed_work": ("get_completed_work", {"id": "AUTHORIZED-1"}),
            "failures": ("get_failures", {"id": "AUTHORIZED-1"}),
            "discoveries": ("get_discoveries", {"id": "AUTHORIZED-1"}),
            "handoff_state": ("get_handoff_state", {"id": "AUTHORIZED-1"}),
        },
        "env": {"REPO_GOVERNOR_EXECUTION": "conformance/fixtures/execution.json"},
        "break_env": {"REPO_GOVERNOR_EXECUTION": "/nonexistent-path-xyz.json"},
        "unknown_fn": ("get_execution_history", {"id": "AUTHORIZED-1"}),
        "absence_fn": ("get_tasks", {"id": "NO-SUCH"}),
    },
    "adapters/beads": {
        # The malformed case is a status Beads does not emit today. The adapter
        # maps a CLOSED set and refuses anything else, because forwarding an
        # unrecognised status produces a task that is neither active nor
        # complete with nothing saying so -- an empty answer where an error
        # belongs.
        "malformed": ("REPO_GOVERNOR_BEADS",
                      '{"_type":"issue","id":"x","status":"snoozed","external_ref":"AUTHORIZED-1"}'),
        "role": "execution",
        "probe": {"id": "AUTHORIZED-1"},
        "capability_fn": {
            "execution_root": ("find_execution_root", {"id": "AUTHORIZED-1"}),
            "tasks": ("get_tasks", {"id": "AUTHORIZED-1"}),
            "dependencies": ("get_dependencies", {"id": "AUTHORIZED-1"}),
            "completed_work": ("get_completed_work", {"id": "AUTHORIZED-1"}),
            "discoveries": ("get_discoveries", {"id": "AUTHORIZED-1"}),
            "provenance": ("get_provenance", {}),
        },
        "env": {"REPO_GOVERNOR_BEADS": "conformance/fixtures/beads-issues.jsonl"},
        "break_env": {"REPO_GOVERNOR_BEADS": "/nonexistent-path-xyz.jsonl"},
        "unknown_fn": ("get_execution_history", {"id": "AUTHORIZED-1"}),
        "absence_fn": ("get_tasks", {"id": "NO-SUCH"}),
    },
    "adapters/change-signals-file": {
        "malformed": ("REPO_GOVERNOR_SIGNALS", '{"signals":"not-a-list"}'),
        "role": "change_signals",
        "probe": {},
        "capability_fn": {
            "signals": ("get_signals", {}),
            "signal_lookup": ("get_signal", {"id": "SIG-1"}),
            "source_dating": ("get_signal", {"id": "SIG-1"}),
        },
        "env": {"REPO_GOVERNOR_SIGNALS": "conformance/fixtures/signals.json"},
        "break_env": {"REPO_GOVERNOR_SIGNALS": "/nonexistent-path-xyz.json"},
        "unknown_fn": ("get_impact", {"id": "SIG-1"}),
        "absence_fn": ("get_signal", {"id": "NO-SUCH"}),
    },
    "adapters/retirement-analysis": {
        "role": "retirement",
        "probe": {"asset": "adapters/git"},
        "capability_fn": {
            "static_references": ("static_references", {"asset": "adapters/git"}),
            "tests": ("tests", {"asset": "adapters/git"}),
            "architecture_references": ("architecture_references", {"asset": "adapters/git"}),
            "obligation_check": ("obligation_check", {"asset": "adapters/git"}),
        },
        "break_env": {"REPO_GOVERNOR_REPO": "/nonexistent-path-xyz"},
        "unknown_fn": ("dynamic_references", {"asset": "adapters/git"}),
    },
    "adapters/decision-history-dolt": {
        "role": "decision_history",
        "probe": {"id": "RBAC-1"},
        "capability_fn": {
            "decisions": ("get_decisions", {"id": "RBAC-1"}),
            "disposition": ("get_disposition", {"id": "RBAC-1"}),
            "reversal_condition": ("get_reversal_condition", {"id": "RBAC-1"}),
            "provenance": ("get_provenance", {}),
            "revision_history": ("get_history", {"id": "RBAC-1"}),
        },
        "break_env": {"REPO_GOVERNOR_DECISIONS_DB": "/nonexistent-path-xyz"},
        "readonly_env": {"REPO_GOVERNOR_DECISIONS_DB": "conformance/fixtures/readonly-db"},
        "unknown_fn": ("get_disposition", {"id": "NEVER-DECIDED"}),
    },
    "adapters/decision-history-file": {
        "role": "decision_history",
        "probe": {"id": "RBAC-1"},
        "capability_fn": {
            "decisions": ("get_decisions", {"id": "RBAC-1"}),
            "disposition": ("get_disposition", {"id": "RBAC-1"}),
            "reversal_condition": ("get_reversal_condition", {"id": "RBAC-1"}),
            "provenance": ("get_provenance", {}),
            "revision_history": ("get_history", {"id": "RBAC-1"}),
        },
        "env": {"REPO_GOVERNOR_DECISIONS_DIR": "conformance/fixtures/decisions-file"},
        "break_env": {"REPO_GOVERNOR_DECISIONS_DIR": "/nonexistent-path-xyz"},
        "readonly_env": {"REPO_GOVERNOR_DECISIONS_DIR": "conformance/fixtures/readonly-decisions"},
        "unknown_fn": ("get_disposition", {"id": "NEVER-DECIDED"}),
    },
    "adapters/decision-history-github": {
        "role": "decision_history",
        "probe": {"id": "901"},
        "capability_fn": {
            "decisions": ("get_decisions", {"id": "901"}),
            "disposition": ("get_disposition", {"id": "901"}),
            "provenance": ("get_provenance", {}),
        },
        "env": {"REPO_GOVERNOR_GH_DECISIONS_FIXTURE": "conformance/fixtures/decision-history-github.json"},
        "break_env": {"REPO_GOVERNOR_GH_DECISIONS_FIXTURE": "/nonexistent-xyz.json"},
        "unknown_fn": ("get_disposition", {"id": "904"}),
        "absence_fn": ("get_decisions", {"id": "999999"}),
    },
    "adapters/linear": {
        "role": "roadmap_authority",
        "probe": {"id": "ENG-101"},
        "capability_fn": {
            "work_lookup": ("get_work", {"id": "ENG-101"}),
            "status": ("get_status", {"id": "ENG-101"}),
            "authority": ("get_authority", {"id": "ENG-101"}),
            "cancellation_detection": ("get_authority", {"id": "ENG-104"}),
            "decision_history": ("get_decision_history", {"id": "ENG-104"}),
            "admission_distinction": ("get_authority", {"id": "ENG-100"}),
        },
        "env": {"REPO_GOVERNOR_LINEAR_FIXTURE": "conformance/fixtures/linear.json"},
        "break_env": {"REPO_GOVERNOR_LINEAR_FIXTURE": "/nonexistent-fixture-xyz.json"},
        "unknown_fn": ("get_authority", {"id": "ENG-100"}),
        "absence_fn": ("get_work", {"id": "ENG-999"}),
    },
}


def run(adapter, args, env_extra=None):
    env = dict(os.environ)
    if env_extra:
        env.update(env_extra)
    p = subprocess.run([sys.executable, str(ROOT / adapter), *args],
                       capture_output=True, text=True, cwd=ROOT, env=env, timeout=30)
    return p.returncode, p.stdout, p.stderr


def q(adapter, role, fn, kw, env_extra=None, base_env=None):
    args = ["query", role, fn] + [f"{k}={v}" for k, v in kw.items()]
    merged = dict(base_env or {})
    merged.update(env_extra or {})
    rc, out, err = run(adapter, args, merged or None)
    try:
        return json.loads(out), out, err
    except json.JSONDecodeError:
        return None, out, err


class Report:
    def __init__(self):
        self.rows = []

    def add(self, adapter, check, ok, detail=""):
            self.rows.append((adapter, check, ok, detail))

    def failures(self):
        return [r for r in self.rows if not r[2]]


def check_adapter(adapter, spec, rep):
    role = spec["role"]
    base = spec.get("env")

    # C1 describe well-formed
    rc, out, err = run(adapter, ["describe"], base)
    try:
        d = json.loads(out)
        good = (d.get("role") == role and isinstance(d.get("contract_version"), int)
                and isinstance(d.get("capabilities"), dict) and isinstance(d.get("functions"), list))
        rep.add(adapter, "C1 describe well-formed", good,
                "" if good else f"got {out[:120]}")
    except json.JSONDecodeError:
        rep.add(adapter, "C1 describe well-formed", False, f"not JSON: {out[:80]} {err[:80]}")
        return

    # C2 honest capability advertisement
    for cap, claimed in d["capabilities"].items():
        if claimed is not True:
            continue
        pair = spec["capability_fn"].get(cap)
        if pair is None:
            rep.add(adapter, f"C2 capability '{cap}' exercised", False,
                    "claimed true but no probe defined — untestable claim")
            continue
        fn, kw = pair
        r, raw, err = q(adapter, role, fn, kw, base_env=base)
        good = bool(r) and r.get("ok") is True and (r.get("value") is not None or r.get("unknown"))
        rep.add(adapter, f"C2 capability '{cap}' exercised", good,
                "" if good else f"{fn} -> {str(r)[:100]}")

    # C3 typed failure on unreachable backend
    fn, kw = next(iter(spec["capability_fn"].values()))
    r, raw, err = q(adapter, role, fn, kw, spec["break_env"], base_env=base)
    good = bool(r) and r.get("ok") is False and r.get("error", {}).get("type") == "PROVIDER_UNAVAILABLE"
    rep.add(adapter, "C3 unreachable backend -> typed failure", good,
            "" if good else f"expected PROVIDER_UNAVAILABLE, got {str(r)[:110]}")

    # C4 absence vs unknown are distinct
    if spec.get("absence_fn"):
        fn, kw = spec["absence_fn"]
        r, _, _ = q(adapter, role, fn, kw, base_env=base)
        good = bool(r) and r.get("ok") is False and r.get("error", {}).get("type") == "NOT_FOUND"
        rep.add(adapter, "C4a absence -> NOT_FOUND", good,
                "" if good else f"got {str(r)[:110]}")
    if spec.get("unknown_fn"):
        fn, kw = spec["unknown_fn"]
        r, _, _ = q(adapter, role, fn, kw, base_env=base)
        good = bool(r) and r.get("ok") is True and r.get("unknown") is not None \
            and {"reason", "detail", "resolution", "blocking"} <= set(r["unknown"])
        rep.add(adapter, "C4b unknown carries typed payload", good,
                "" if good else f"got {str(r)[:110]}")

    # C4c a malformed source value must not be silently coerced (the ADR-015 lesson)
    if spec.get("malformed_fn"):
        fn, kw = spec["malformed_fn"]
        r, _, _ = q(adapter, role, fn, kw, base_env=base)
        good = bool(r) and r.get("ok") is False and r.get("error", {}).get("type") == "MALFORMED_SOURCE"
        rep.add(adapter, "C4c malformed value -> MALFORMED_SOURCE", good,
                "" if good else f"got {str(r)[:110]}")

    # C5 provenance on every fact
    missing = []
    for cap, pair in spec["capability_fn"].items():
        fn, kw = pair
        r, _, _ = q(adapter, role, fn, kw, base_env=base)
        if r and r.get("ok") and r.get("value") is not None:
            if not r.get("provenance"):
                missing.append(fn)
            else:
                for c in r["provenance"]:
                    if not (c.get("source") and c.get("ref")):
                        missing.append(f"{fn}(malformed cite)")
    rep.add(adapter, "C5 every fact carries provenance", not missing,
            "" if not missing else f"missing on: {missing}")

    # C6 unsupported function rejected
    r, _, _ = q(adapter, role, "get_nonexistent_thing", {}, base_env=base)
    good = bool(r) and r.get("ok") is False and r.get("error", {}).get("type") == "UNSUPPORTED_FUNCTION"
    rep.add(adapter, "C6 unsupported function rejected", good,
            "" if good else f"got {str(r)[:110]}")

    # C8 unreachable transport must advertise nothing (issue #17)
    broken = dict(base or {}); broken.update(spec["break_env"])
    rc2, out2, _ = run(adapter, ["describe"], broken)
    try:
        d2 = json.loads(out2)
        good = d2.get("capabilities") == {} and d2.get("transport", {}).get("reachable") is False
        rep.add(adapter, "C8 unreachable transport claims nothing", good,
                "" if good else f"advertised {len(d2.get('capabilities', {}))} capabilities while unreachable")
    except json.JSONDecodeError:
        rep.add(adapter, "C8 unreachable transport claims nothing", False, "describe not JSON")

    # C9 writers are gated on writability, not merely on reachability (#17)
    if spec.get("readonly_env"):
        ro = dict(base or {}); ro.update(spec["readonly_env"])
        # Establish read-only HERE rather than relying on committed permissions.
        # git does not preserve directory modes, so the fixture cloned writable
        # and this check quietly tested nothing on every machine but the one it
        # was written on -- a fixture whose property does not survive the
        # transport that carries it.
        import os as _os9, stat as _st9
        _ro_dir = ROOT / list(spec["readonly_env"].values())[0]
        _restore = []
        if _ro_dir.is_dir():
            for _d in [_ro_dir, _ro_dir / ".dolt"]:
                if _d.is_dir():
                    _restore.append((_d, _d.stat().st_mode))
                    _os9.chmod(_d, _st9.S_IRUSR | _st9.S_IXUSR)
        rc3, out3, _ = run(adapter, ["describe"], ro)
        try:
            d3 = json.loads(out3)
            t = d3.get("transport", {})
            good = t.get("reachable") is True and t.get("writable") is False and d3.get("writers") == []
            for _d, _m in _restore:
                _os9.chmod(_d, _m)
            rep.add(adapter, "C9 unwritable transport advertises no writers", good,
                    "" if good else f"reachable={t.get('reachable')} writable={t.get('writable')} "
                                    f"writers={d3.get('writers')}")
        except json.JSONDecodeError:
            rep.add(adapter, "C9 unwritable transport advertises no writers", False, "describe not JSON")

    # C7 determinism
    fn, kw = next(iter(spec["capability_fn"].values()))
    _, a, _ = q(adapter, role, fn, kw, base_env=base)
    _, b, _ = q(adapter, role, fn, kw, base_env=base)
    _, c, _ = q(adapter, role, fn, kw, base_env=base)
    rep.add(adapter, "C7 byte-identical across runs", a == b == c,
            "" if a == b == c else "output varies between identical invocations")

    # C10 REACHABLE BUT STRUCTURALLY WRONG. Distinct from C3, which points the
    # adapter at a path that does not exist. This gives it a store that EXISTS,
    # parses as JSON, and has the wrong shape -- the gap between "unreachable"
    # and "reachable but unusable", which ADR-008 names as two cases and the
    # suite only tested one.
    #
    # It matters because of how the failure presents. file-roadmap raised
    # AttributeError on `items` as a list; the traceback reached the engine as
    # reason NON_JSON, sending a reader to look for a syntax error in a file
    # that is valid JSON. Worse, a crashed provider returns UNKNOWN, and
    # UNKNOWN satisfies any assertion written as `decision != SOMETHING` -- so
    # the crash quietly made a test pass elsewhere.
    mal = spec.get("malformed")
    if mal:
        var, content = mal
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as fh:
            fh.write(content)
            bad_path = fh.name
        try:
            fn, kw = spec["capability_fn"][next(iter(spec["capability_fn"]))]
            r, raw, _ = q(adapter, spec["role"], fn, kw, {var: bad_path}, base_env=base)
            crashed = "Traceback" in (raw or "")
            typed = bool(r) and not r.get("ok") and \
                (r.get("error") or {}).get("type") in ("MALFORMED_SOURCE", "PROVIDER_UNAVAILABLE")
            ok10 = typed and not crashed
            rep.add(adapter, "C10 malformed store -> typed error, not a crash", ok10,
                    "" if ok10 else ("traceback leaked to the caller: " + (raw or "")[:120]
                                     if crashed else
                                     f"got {json.dumps(r)[:120] if r else (raw or '')[:120]}"))
        finally:
            os.unlink(bad_path)


def _q(fixture, fn):
    """One openspec query against a fixture directory."""
    env = dict(os.environ, REPO_GOVERNOR_OPENSPEC_DIR=f"conformance/fixtures/openspec/{fixture}")
    env.pop("REPO_GOVERNOR_BINDING", None)
    r = subprocess.run([sys.executable, str(ROOT / "adapters" / "openspec"),
                        "query", "architecture", fn],
                       capture_output=True, text=True, cwd=str(ROOT), env=env, timeout=60)
    return json.loads(r.stdout)


def _openspec_design(rep):
    """The decisions the census forced, asserted rather than left in comments.

    Protocol conformance above says the adapter is well-formed. It says nothing
    about whether the mapping from an OpenSpec tree onto the §12 contract is
    honest, and two of those choices are traps that a well-formed adapter would
    walk straight into (issue 155).
    """
    A = "adapters/openspec"

    # 41.7% of measured repositories have changes/ and no specs/. An empty
    # success would state "this architecture has no specs" as a FACT; the truth
    # is that specs are recorded elsewhere or not yet accumulated (ADR-003
    # rule 6, INV-012).
    d = _q("delta", "get_specs")
    u = d.get("unknown") or {}
    rep.add(A, "openspec: specs/ absent yields a typed UNKNOWN, not an empty success",
            d.get("value") is None and u.get("reason") == "NO_ARCHITECTURE_EVIDENCE"
            and u.get("blocking") is False,
            f"value={d.get('value')} unknown={u or None}")

    # THE TRAP. Archiving is completion. Reporting archived changes as
    # superseded would make every finished change look like a withdrawn
    # decision.
    d = _q("full", "get_superseded_decisions")
    v = d.get("value") or {}
    comp = (_q("full", "get_completed_changes").get("value") or {}).get("count")
    rep.add(A, "openspec: an archived change is completed, never superseded",
            v.get("count") == 0 and "supersession" in (v.get("note") or "") and comp == 1,
            f"superseded={v.get('count')} completed={comp} — the fixture has one "
            "archived change; it must appear as completed and not as superseded")

    # THE TRAP, END TO END. The unit check can pass while the union still
    # escalates, so this binds a real ADR provider beside a real OpenSpec one
    # over a fixture whose archived change is NAMED ADR-0001 -- the exact id the
    # ADR provider holds as Accepted. ADR-013 rule 3 escalates "active in one
    # provider, superseded in another"; nothing here is superseded, so nothing
    # may escalate.
    sys.path.insert(0, str(ROOT / "engine"))
    import envelope as _E
    import manifest as _MF
    m, errs = _MF.load(ROOT / "conformance" / "fixtures" / "openspec" / "manifest-with-adr.json")
    env = _E.compile_envelope("155", manifest=m) if not errs else {}
    provs = [e["provider"] for e in env.get("architecture_evidence") or []]
    rep.add(A, "openspec: a change and an ADR with the same id do not contradict",
            not errs and "architecture_review" not in env and len(provs) == 2
            and "ADR-0001" in env.get("architecture_constraints", []),
            f"errs={errs} providers={provs} "
            f"review={(env.get('architecture_review') or {}).get('contradictions')}")

    # 11.2% keep loose files beside the change directories. Dropping them
    # silently reports a smaller change set as the whole one (issue 25).
    d = _q("loose", "get_active_decisions")
    u = d.get("unknown") or {}
    rep.add(A, "openspec: loose files in changes/ are reported as skipped, not dropped",
            u.get("reason") == "ARCHITECTURE_PARTIALLY_READ" and "2 entr" in (u.get("detail") or "")
            and (_q("loose", "get_provenance").get("value") or {}).get("unreadable_entries") == 2,
            f"unknown={u or None} — the fixture has two loose files beside one change dir")

    # The corrected census: project.md is present in 20.7% of OpenSpec
    # repositories under an independent selector, not the 97.8% a self-selecting
    # sample first reported. Detection keyed on it would miss four in five.
    #
    # ASSERTED BEHAVIOURALLY. The first version of this check grepped the
    # adapter source for "project.md" and failed on the COMMENT saying the
    # adapter does not use it -- the same defect as conformance/imports.py
    # refusing to grep for imports, and as SUPERSEDED_RE matching prose about
    # supersession. A file that talks about a thing is not an instance of it.
    with tempfile.TemporaryDirectory() as td:
        no_pm = Path(td) / "no-project-md" / "openspec" / "changes" / "add-x"
        no_pm.mkdir(parents=True)
        (no_pm / "proposal.md").write_text("# p\n")
        only_pm = Path(td) / "only-project-md" / "openspec"
        only_pm.mkdir(parents=True)
        (only_pm / "project.md").write_text("# p\n")

        def probes(d):
            env = dict(os.environ, REPO_GOVERNOR_OPENSPEC_DIR=str(d))
            env.pop("REPO_GOVERNOR_BINDING", None)
            r = subprocess.run([sys.executable, str(ROOT / "adapters" / "openspec"), "describe"],
                               capture_output=True, text=True, cwd=str(ROOT), env=env, timeout=60)
            return bool(json.loads(r.stdout).get("capabilities"))

        without = probes(no_pm.parent.parent)
        only = probes(only_pm)
    rep.add(A, "openspec: detection keys on the directory, not on project.md",
            without and not only,
            f"changes/-without-project.md advertises={without} (must be True); "
            f"project.md-alone advertises={only} (must be False)")


def _sq(fixture, fn):
    env = dict(os.environ,
               REPO_GOVERNOR_SPECKIT_DIR=f"conformance/fixtures/speckit/{fixture}/.specify")
    env.pop("REPO_GOVERNOR_BINDING", None)
    r = subprocess.run([sys.executable, str(ROOT / "adapters" / "speckit"),
                        "query", "architecture", fn],
                       capture_output=True, text=True, cwd=str(ROOT), env=env, timeout=60)
    return json.loads(r.stdout)


def _speckit_design(rep):
    """The three refusals the census forced, asserted rather than commented."""
    A = "adapters/speckit"

    # §37. Spec Kit ships a constitution with named placeholders, and 10.2% of
    # real constitutions are still in that state. Reading their articles as
    # constraints asserts an architecture from a file nobody wrote.
    d = _sq("template", "get_constraints")
    u = d.get("unknown") or {}
    rep.add(A, "speckit: an unfilled template constitution is UNKNOWN, never DEFINED",
            d.get("value") is None and u.get("reason") == "STATUS_TEMPLATE_UNFILLED",
            f"value={d.get('value')} unknown={u or None}")

    # Without this the check above passes by never producing constraints at all.
    d = _sq("authored", "get_constraints")
    v = d.get("value") or {}
    rep.add(A, "speckit: an authored constitution yields constraints",
            v.get("state") == "DEFINED" and v.get("count") == 4,
            f"state={v.get('state')} count={v.get('count')}")

    # ADR-016 rule 2: a typed field addressable without parsing prose. 50 of 50
    # real constitutions carry `##` headings, median 4.
    ids = [c["id"] for c in v.get("constraints") or []]
    rep.add(A, "speckit: constraints come from headings, not from prose",
            any(i.startswith("Article I:") for i in ids) and "Governance" in ids,
            f"ids={ids}")

    # 23.6% of real repositories do not number their feature directories. A
    # filter on `NNN-` would report those as having no specs at all.
    d = _sq("authored", "get_specs")
    specs = (d.get("value") or {}).get("specs") or []
    unnumbered = [s["id"] for s in specs if not s["numbered"]]
    rep.add(A, "speckit: a spec directory without NNN- numbering is still read",
            len(specs) == 2 and unnumbered == ["checkout-flow"],
            f"specs={[s['id'] for s in specs]} unnumbered={unnumbered}")

    # INV-002. The fixture carries a COMPLETED checkbox in tasks.md on purpose:
    # an assertion that checkbox state never surfaces is worth nothing unless
    # there is a completed one to surface.
    blob = json.dumps([_sq("authored", fn) for fn in
                       ("get_constraints", "get_specs", "get_active_decisions",
                        "get_superseded_decisions")])
    rep.add(A, "speckit: tasks.md completion never becomes a constraint or a decision",
            "T001" not in blob and "tasks.md" not in blob,
            "checkbox state is EXECUTION state; references/integrations.md names the "
            "failure -- 'This task is marked Ready, so I should work on it'")

    # The mirror of adapters/adr::get_specs. Empty WITH a note; empty in silence
    # would report "no architectural decisions" as a fact about the repository.
    a = (_sq("authored", "get_active_decisions").get("value") or {})
    sup = (_sq("authored", "get_superseded_decisions").get("value") or {})
    rep.add(A, "speckit: Spec Kit supplies no decision ledger, and says so",
            a.get("count") == 0 and sup.get("count") == 0
            and "no decision ledger" in (a.get("note") or ""),
            f"active={a.get('count')} superseded={sup.get('count')} note={a.get('note')!r}")

    # THE CENSUS CORRECTION. 50% of repositories matching `specs/**/plan.md`
    # carry no `.specify/`. Behavioural, not a source grep -- a grep for
    # "specs/" would fail on the comment saying we do not key on it, which
    # happened in this very file during issue 155.
    with tempfile.TemporaryDirectory() as td:
        bare = Path(td) / "specs-only" / "specs" / "001-thing"
        bare.mkdir(parents=True)
        (bare / "spec.md").write_text("# s\n")
        env = dict(os.environ, REPO_GOVERNOR_SPECKIT_DIR=str(bare.parent.parent / ".specify"))
        env.pop("REPO_GOVERNOR_BINDING", None)
        r = subprocess.run([sys.executable, str(ROOT / "adapters" / "speckit"), "describe"],
                           capture_output=True, text=True, cwd=str(ROOT), env=env, timeout=60)
        claims = bool(json.loads(r.stdout).get("capabilities"))
    rep.add(A, "speckit: detection keys on .specify, never on specs/",
            not claims,
            "a repository with specs/ and no .specify/ is not a Spec Kit repository; "
            "50% of specs/**/plan.md matches are not")


def main(argv):
    absent = _preflight.banner()
    targets = argv or list(SUITE)
    rep = Report()
    for adapter in targets:
        if adapter not in SUITE:
            print(f"unknown adapter {adapter!r}", file=sys.stderr)
            return 2
        check_adapter(adapter, SUITE[adapter], rep)

    if "adapters/openspec" in targets:
        _openspec_design(rep)
    if "adapters/speckit" in targets:
        _speckit_design(rep)

    cur = None
    for adapter, check, ok, detail in rep.rows:
        if adapter != cur:
            print(f"\n{adapter}")
            cur = adapter
        mark = "PASS" if ok else "FAIL"
        print(f"  [{mark}] {check}" + (f"\n         {detail}" if detail else ""))

    fails = rep.failures()
    print(f"\n{len(rep.rows) - len(fails)}/{len(rep.rows)} checks passed")
    print("LAYER 1: " + ("CONFORMANT" if not fails else f"NON-CONFORMANT ({len(fails)} failures)"))
    if fails:
        _preflight.attribute(absent)
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
