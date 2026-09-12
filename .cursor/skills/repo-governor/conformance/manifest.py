#!/usr/bin/env python3
"""Manifest loader conformance — gate 5 (#11).

The loader's value is entirely in what it REFUSES. Each case below mutates
the real manifest into something that must fail, and asserts the specific
error fires. A loader that accepts a bad manifest is worse than none: it
produces confident governance from a wrong binding.

Usage:  python3 conformance/manifest.py
"""

from __future__ import annotations

import copy
import json
import sys
import tempfile
import pathlib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "engine"))
import manifest as M  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("manifest")

BASE = json.loads((ROOT / ".repo-governor.json").read_text())


def mutate(fn):
    d = copy.deepcopy(BASE)
    fn(d)
    return d


def _drop_repo(d):
    del d["providers"]["repository"]
    d["permissions"].pop("repository", None)


CASES = [
    ("baseline is valid",              lambda d: None,                                    None),
    ("future version refuses",         lambda d: d["repo_governor"].__setitem__("version", 99), "UNSUPPORTED_VERSION"),
    ("scalar role given a list",       lambda d: d["providers"].__setitem__("roadmap_authority", [d["providers"]["roadmap_authority"]]), "CARDINALITY"),
    ("array role given a scalar",      lambda d: d["providers"].__setitem__("architecture", d["providers"]["architecture"][0]), "CARDINALITY"),
    ("repository role missing",        _drop_repo,                                        "MISSING_ROLE"),
    ("adapter escapes repository",     lambda d: d["providers"]["repository"].__setitem__("adapter", "../../../etc/passwd"), "ADAPTER"),
    ("adapter does not exist",         lambda d: d["providers"]["repository"].__setitem__("adapter", "adapters/nope"), "ADAPTER_MISSING"),
    ("reserved verb execute",          lambda d: d["permissions"]["repository"].__setitem__("execute", True), "PERMISSION_RESERVED"),
    ("unknown verb",                   lambda d: d["permissions"]["repository"].__setitem__("delete", True), "PERMISSION_UNKNOWN_VERB"),
    ("permissions for unbound role",   lambda d: d["permissions"].__setitem__("retirement_x", {"read": True}), "PERMISSION_ORPHAN"),
    ("malformed permission block",     lambda d: d["permissions"].__setitem__("repository", "yes"), "PERMISSION_MALFORMED"),
    ("github token in manifest",       lambda d: d["providers"]["repository"].__setitem__("project", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123"), "SECRET"),
    ("opaque credential-shaped value", lambda d: d["providers"]["repository"].__setitem__("project", "aGVsbG93b3JsZGhlbGxvd29ybGRoZWxsb3dvcmxk"), "SECRET"),
    ("key named api_key",              lambda d: d["providers"]["repository"].__setitem__("api_key", "x"), "SECRET"),
    ("unknown top-level property",     lambda d: d.__setitem__("roadmap", {"items": {}}), "SCHEMA"),
    ("bad condition level",            lambda d: d["condition"].__setitem__("assessed", "L9"), "SCHEMA"),
    ("profile not in enum",            lambda d: d["condition"].__setitem__("profile", "GOVERNOR_TURBO"), "SCHEMA"),
    ("binding without adapter",        lambda d: d["providers"]["repository"].pop("adapter"), "SCHEMA"),
    ("bad transport kind",             lambda d: d["providers"]["repository"].__setitem__("transport", {"kind": "carrier-pigeon"}), "SCHEMA"),
    # Issue 198. The hook compares `mode == "blocking"`, so a typo would fall
    # through to advisory and give a repository weaker governance than it
    # declared, silently -- 189's defect one layer up. An enum makes it an
    # error instead of a downgrade.
    ("enforcement not in enum",        lambda d: d["repo_governor"].__setitem__("enforcement", "blockign"), "SCHEMA"),
    ("contract_version zero",          lambda d: d["providers"]["repository"].__setitem__("contract_version", 0), "SCHEMA"),
]

# Deny-by-default: (role, verb, expected)
PERM_CASES = [
    ("repository", "read", True),
    ("repository", "write", False),          # explicitly false
    ("repository", "create", False),         # absent verb => deny
    ("roadmap_authority", "transition", False),
    ("decision_history", "read", True),      # bound as of ADR-019 (two backends)
    ("decision_history", "write", True),     # the ONE write granted anywhere (ADR-019)
    ("architecture", "write", False),        # every other role stays read-only
    ("nonexistent", "read", False),          # unbound role => deny; all 8 real roles are now bound
]


def main():
    fails = 0
    print("Manifest loader — refusal cases\n")
    for label, fn, expect in CASES:
        d = mutate(fn)
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
            json.dump(d, f)
            tmp = f.name
        m, errs = M.load(tmp)
        Path(tmp).unlink()
        if expect is None:
            ok = not errs and m is not None
            detail = "" if ok else f"unexpected: {errs[:2]}"
        else:
            ok = any(e.startswith(expect) or expect in e for e in errs) and m is None
            detail = "" if ok else f"expected {expect}, got {errs[:2] or 'no errors'}"
        fails += not ok
        print(f"  [{'PASS' if ok else 'FAIL'}] {label:<34}{'  ' + detail if detail else ''}")

    print("\nDeny by default\n")
    m, errs = M.load()
    assert not errs, errs
    for role, verb, expect in PERM_CASES:
        got, why = M.permitted(m, role, verb)
        ok = got == expect
        fails += not ok
        print(f"  [{'PASS' if ok else 'FAIL'}] {role}.{verb:<11} -> {'ALLOW' if got else 'DENY'}   {why[:56]}")

    # A binding can be well-formed, reachable, and answer nothing. --validate
    # said READY_FOR_GOVERNANCE for exactly that, and it is the last check
    # before someone trusts the answers -- so a green verdict there is worse
    # than a red one, because the very next command says PROVIDER_UNAVAILABLE
    # and the operator has to decide which to believe (issue 51).
    print("\nA binding that cannot answer is not READY_FOR_GOVERNANCE\n")
    import copy as _copy, subprocess as _sp, tempfile as _tf, os as _os
    extra = 0

    def _validate(manifest_obj):
        with _tf.TemporaryDirectory() as td:
            r = pathlib.Path(td) / "repo"
            r.mkdir()
            _sp.run(["git", "init", "-q", str(r)], capture_output=True)
            (r / ".repo-governor.json").write_text(json.dumps(manifest_obj))
            env = dict(_os.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
            return _sp.run([sys.executable, str(ROOT / "engine" / "manifest.py"), "--validate"],
                           capture_output=True, text=True, cwd=str(r), env=env, timeout=120).stdout

    base = json.loads((ROOT / ".repo-governor.json").read_text())
    stripped = _copy.deepcopy(base)
    stripped["providers"] = {k: v for k, v in base["providers"].items()
                             if k in ("repository", "roadmap_authority") or k.startswith("$")}
    stripped["permissions"] = {k: v for k, v in base["permissions"].items()
                               if k in ("repository", "roadmap_authority") or k.startswith("$")}
    kept = _copy.deepcopy(stripped)
    stripped["providers"]["roadmap_authority"] = dict(stripped["providers"]["roadmap_authority"])
    stripped["providers"]["roadmap_authority"].pop("env", None)

    out_bad = _validate(stripped)
    ok = "READY_FOR_GOVERNANCE" not in out_bad and "NOT_CONFIGURED" in out_bad
    extra += not ok
    print(f"  [{'PASS' if ok else 'FAIL'}] a binding with no declared identity is refused"
          + ("" if ok else f"\n         {out_bad.strip()[:180]}"))

    ok2 = "REPO_GOVERNOR_GH_REPO" in out_bad
    extra += not ok2
    print(f"  [{'PASS' if ok2 else 'FAIL'}] and the finding names the missing configuration"
          + ("" if ok2 else "\n         'not configured' without naming it is not actionable"))

    # Positive control. Without it, the check above passes for any manifest
    # that fails to validate for ANY reason.
    out_good = _validate(kept)
    ok3 = "READY_FOR_GOVERNANCE" in out_good
    extra += not ok3
    print(f"  [{'PASS' if ok3 else 'FAIL'}] control: the same binding WITH identity is ready"
          + ("" if ok3 else f"\n         {out_good.strip()[:180]}"))

    # --- issue 180: an unmet requirement is a finding, not a failure --------
    #
    # Two surfaces disagreed about one repository. --validate printed
    # PROVIDER_UNAVAILABLE and exited 1; engine/status.py printed "That is a
    # configuration gap, not a verdict. Nothing here refuses to answer because
    # of it" and exited 0. conformance/status.py:238-248 already asserted the
    # second position. Same family as issue 161 -- two places that must agree,
    # with nothing asserting it.
    print("\nAn unmet requirement is a finding; a broken binding is a failure\n")

    def _validate_rc(manifest_obj):
        """(stdout, returncode) -- the exit code is half the contract here."""
        with _tf.TemporaryDirectory() as td:
            r = pathlib.Path(td) / "repo"
            r.mkdir()
            _sp.run(["git", "init", "-q", str(r)], capture_output=True)
            (r / ".repo-governor.json").write_text(json.dumps(manifest_obj))
            env = dict(_os.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
            pr = _sp.run([sys.executable, str(ROOT / "engine" / "manifest.py"), "--validate"],
                         capture_output=True, text=True, cwd=str(r), env=env, timeout=120)
            return pr.stdout, pr.returncode

    # L4 wants five roles; only `repository` is bound. Nothing here is broken --
    # the binding that exists answers fine.
    gap = {"repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
           "repository": {"id": "tosin2013/repo-governor"},
           "condition": {"assessed": "L4", "profile": "GOVERNOR_HIGH_ASSURANCE"},
           "permissions": {"repository": {"read": True, "write": False}},
           "providers": {"repository": {"type": "git", "adapter": "adapters/git",
                                        "contract_version": 1}}}
    out_gap, rc_gap = _validate_rc(gap)
    okg = "READY_FOR_GOVERNANCE" in out_gap and rc_gap == 0
    extra += not okg
    print(f"  [{'PASS' if okg else 'FAIL'}] an unmet required role does not turn validation into a failure"
          + ("" if okg else f"\n         rc={rc_gap} {out_gap.strip()[-160:]}"))

    okr = "REQUIRED_ROLE_UNBOUND" in out_gap and "advisory" in out_gap
    extra += not okr
    print(f"  [{'PASS' if okr else 'FAIL'}] the advisory finding is still reported, not silently dropped"
          + ("" if okr else "\n         non-fatal must not mean invisible: the gap is real and a reader needs it"))

    # THE CONTROL, and the reason it is second rather than an afterthought:
    # without it the check above passes by accepting everything, which makes
    # --validate vacuous -- worse than the bug. L1 requires only `repository`,
    # so no required-role finding can fire and the ONLY finding is a real one.
    broke = {"repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
             "repository": {"id": "tosin2013/repo-governor"},
             "condition": {"assessed": "L1", "profile": "GOVERNOR_LITE"},
             "permissions": {"repository": {"read": True, "write": False},
                             "decision_history": {"read": True, "write": False}},
             "providers": {
               "repository": {"type": "git", "adapter": "adapters/git", "contract_version": 1},
               "decision_history": [{"type": "decision-history-dolt",
                                     "adapter": "adapters/decision-history-dolt",
                                     "contract_version": 1,
                                     "env": {"REPO_GOVERNOR_DECISIONS_DB": "/nonexistent-db"}}]}}
    out_broke, rc_broke = _validate_rc(broke)
    okb = "PROVIDER_UNAVAILABLE" in out_broke and rc_broke != 0
    extra += not okb
    print(f"  [{'PASS' if okb else 'FAIL'}] an adapter that cannot answer still fails validation"
          + ("" if okb else f"\n         rc={rc_broke} {out_broke.strip()[-160:]}"))

    # THE ACTUAL BUG, asserted directly rather than inferred from the two halves
    # above. One fixture through both surfaces; neither may contradict the other.
    with _tf.TemporaryDirectory() as td:
        r2 = pathlib.Path(td) / "repo"
        r2.mkdir()
        _sp.run(["git", "init", "-q", str(r2)], capture_output=True)
        (r2 / ".repo-governor.json").write_text(json.dumps(gap))
        env2 = dict(_os.environ); env2["REPO_GOVERNOR_TARGET"] = str(r2)
        st = _sp.run([sys.executable, str(ROOT / "engine" / "status.py"), str(r2)],
                     capture_output=True, text=True, cwd=str(ROOT), env=env2, timeout=120)
    oka = (rc_gap == 0) == (st.returncode == 0)
    extra += not oka
    print(f"  [{'PASS' if oka else 'FAIL'}] validate and status agree about one repository"
          + ("" if oka else f"\n         validate rc={rc_gap}, status rc={st.returncode} "
                            "-- an operator has to decide which to believe"))

    # --- governance that lives on one machine is not governance --------------
    #
    # A repository was governed for months on a host that was then deleted. Its
    # manifest had never been committed, so the replacement host read it as
    # un-onboarded while the old one had been reporting READY_FOR_GOVERNANCE
    # throughout. Onboarding says "rename and commit" in a $comment and a
    # docstring, and neither is a check -- a rule declared in one place and read
    # in none.
    #
    # Both directions, because a trackedness check passes trivially in the
    # repository that develops it: everything here is committed already.
    print("\nA manifest no other host can see is not unqualified governance\n")

    def _validate_tracked(commit, ignore=False):
        """Validate a manifest that is, or is not, in git's index."""
        with _tf.TemporaryDirectory() as td:
            r = pathlib.Path(td) / "repo"
            r.mkdir()
            _sp.run(["git", "init", "-q", str(r)], capture_output=True)
            (r / ".repo-governor.json").write_text(json.dumps(kept))
            if ignore:
                (r / ".gitignore").write_text(".repo-governor.json\n")
            if commit:
                _sp.run(["git", "-C", str(r), "add", "-f", ".repo-governor.json"],
                        capture_output=True)
            env = dict(_os.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
            pr = _sp.run([sys.executable, str(ROOT / "engine" / "manifest.py"), "--validate"],
                         capture_output=True, text=True, cwd=str(r), env=env, timeout=120)
            return pr.stdout, pr.returncode

    out_un, rc_un = _validate_tracked(commit=False)
    oku = "MANIFEST_UNTRACKED" in out_un and "ON THIS HOST ONLY" in out_un
    extra += not oku
    print(f"  [{'PASS' if oku else 'FAIL'}] an uncommitted manifest is reported and the green is qualified"
          + ("" if oku else f"\n         {out_un.strip()[-200:]}"))

    # Reported, not refused. ADR-031: a configuration gap is disclosed, never
    # blocked -- and the operator runs this BEFORE committing, by design.
    okn = rc_un == 0
    extra += not okn
    print(f"  [{'PASS' if okn else 'FAIL'}] and it does not refuse: disclosure, not refusal (ADR-031)"
          + ("" if okn else f"\n         rc={rc_un} -- validating before the commit is the documented flow"))

    # THE CONTROL. Without it the check above passes for a guard that fires on
    # every repository, which would make the qualifier meaningless noise.
    out_tr, _ = _validate_tracked(commit=True)
    okt = "MANIFEST_UNTRACKED" not in out_tr and "ON THIS HOST ONLY" not in out_tr
    extra += not okt
    print(f"  [{'PASS' if okt else 'FAIL'}] control: once tracked, the qualifier is gone"
          + ("" if okt else f"\n         {out_tr.strip()[-200:]}"))

    # An actively-ignored artifact is the worse case: `git add` alone does not
    # fix it, and saying only "not tracked" would send the reader to a repair
    # that silently does nothing.
    out_ig, _ = _validate_tracked(commit=False, ignore=True)
    oki = "EXCLUDES" in out_ig
    extra += not oki
    print(f"  [{'PASS' if oki else 'FAIL'}] a gitignored manifest is named as excluded, not merely untracked"
          + ("" if oki else f"\n         {out_ig.strip()[-200:]}"))

    # --- a history that cannot leave this machine ---------------------------
    #
    # The check is deliberately not "is Dolt bound": section 54 forbids
    # requiring a named third-party tool, so the property is one each backend
    # answers for itself. Both directions, because an advisory that fires for
    # every repository says nothing.
    print("\nA decision history that cannot survive a clone is reported\n")

    def _validate_dh(backends):
        base_dh = _copy.deepcopy(kept)
        base_dh["providers"]["decision_history"] = backends
        base_dh["permissions"]["decision_history"] = {"read": True, "write": True}
        with _tf.TemporaryDirectory() as td:
            r = pathlib.Path(td) / "repo"
            r.mkdir()
            _sp.run(["git", "init", "-q", str(r)], capture_output=True)
            (r / ".repo-governor.json").write_text(json.dumps(base_dh))
            _sp.run(["git", "-C", str(r), "add", ".repo-governor.json"], capture_output=True)
            (r / ".repo-governor" / "decisions").mkdir(parents=True, exist_ok=True)
            env = dict(_os.environ); env["REPO_GOVERNOR_TARGET"] = str(r)
            return _sp.run([sys.executable, str(ROOT / "engine" / "manifest.py"), "--validate"],
                           capture_output=True, text=True, cwd=str(r), env=env,
                           timeout=120).stdout

    _dolt_only = [{"type": "decision-history-dolt",
                   "adapter": "adapters/decision-history-dolt", "contract_version": 1,
                   "env": {"REPO_GOVERNOR_DECISIONS_DB":
                           str((ROOT / ".repo-governor" / "decisions-db").resolve())}}]
    _with_file = _dolt_only + [{"type": "decision-history-file",
                                "adapter": "adapters/decision-history-file",
                                "contract_version": 1}]

    out_np = _validate_dh(_dolt_only)
    okp = "HISTORY_NOT_PORTABLE" in out_np
    extra += not okp
    print(f"  [{'PASS' if okp else 'FAIL'}] a history bound only to a non-portable store is reported"
          + ("" if okp else f"\n         {out_np.strip()[-200:]}"))

    # THE CONTROL. ADR-019 rule 2 expects a portable backend bound ALONGSIDE a
    # local one; flagging that arrangement would punish the recommended fix.
    out_p = _validate_dh(_with_file)
    okp2 = "HISTORY_NOT_PORTABLE" not in out_p
    extra += not okp2
    print(f"  [{'PASS' if okp2 else 'FAIL'}] control: one portable backend alongside is enough"
          + ("" if okp2 else f"\n         {out_p.strip()[-200:]}"))

    # A DECLARED transport is not a reached one (issue 130). Declaring
    # transport.kind=mcp made the adapter report reachable and writable, which
    # silenced WRITE_GRANTED_BUT_TRANSPORT_READONLY for a transport that has no
    # write path at all -- the #17 guard, disarmed by adding a line to a
    # manifest. It also advertised a full capability set the adapter could not
    # exercise, which ADR-008 C2 forbids.
    print("\nA declared transport is not a reached one\n")
    mcp = _copy.deepcopy(kept)
    mcp["providers"]["roadmap_authority"] = {
        "type": "linear", "adapter": "adapters/linear",
        "transport": {"kind": "mcp", "server": "linear"}}
    mcp["permissions"]["roadmap_authority"] = {"read": True, "write": True}
    out_mcp = _validate(mcp)

    okm1 = "WRITE_GRANTED_BUT_TRANSPORT_READONLY" in out_mcp
    extra += not okm1
    print(f"  [{'PASS' if okm1 else 'FAIL'}] granting write over a declared MCP transport is caught"
          + ("" if okm1 else "\n         agent-supplied stdin is read-only by construction; "
                             "a grant that cannot be served must not validate clean"))

    # Issue 124's own requirement, kept as a control: the fix for THIS issue
    # must not quietly undo it by making the adapter unconfigured again.
    okm2 = "NOT_CONFIGURED" not in out_mcp
    extra += not okm2
    print(f"  [{'PASS' if okm2 else 'FAIL'}] and it is still not NOT_CONFIGURED (issue 124)"
          + ("" if okm2 else "\n         a declared transport IS configured; that was the "
                             "whole complaint, and this check exists so fixing 130 "
                             "cannot silently reopen 124"))

    okm3 = "--input" in out_mcp
    extra += not okm3
    print(f"  [{'PASS' if okm3 else 'FAIL'}] the unreachable finding says what would fix it"
          + ("" if okm3 else "\n         the engine's default message claims the transport is "
                             "not configured, which is false here"))

    # The engine must not have learned an adapter's variable names.
    src = (ROOT / "engine" / "manifest.py").read_text()
    ok4 = "REPO_GOVERNOR_GH_REPO" not in src and "LINEAR_API_KEY" not in src
    extra += not ok4
    print(f"  [{'PASS' if ok4 else 'FAIL'}] the engine names no adapter's configuration (ADR-003)"
          + ("" if ok4 else "\n         adapter-specific knowledge must stay in the adapter"))

    # ISSUE 198. docs/installation.md documented `enforcement` while the schema's
    # additionalProperties refused it, so a reader following the instructions
    # produced MANIFEST_INVALID and every verdict degraded to UNKNOWN. Following
    # the documentation turned a governed repository into an ungoverned one.
    #
    # Read FROM the document, never restated here: a copy of the example would
    # agree with itself while both drifted from the schema, which is the shape
    # issue 178 recorded when README.md and ENGINE_VERSION agreed perfectly and
    # were both wrong.
    import re as _re
    _schema = json.loads((ROOT / "schemas" / "manifest-v1.json").read_text(encoding="utf-8"))
    _allowed = set(_schema["properties"]["repo_governor"]["properties"])
    _docs = [d for d in ("README.md", "docs/installation.md", "AGENTS.md",
                         "CONTRIBUTING.md") if (ROOT / d).is_file()]
    _frags, _bad = 0, []
    for _d in _docs:
        for _b in _re.findall(r"```json\s*\n(.*?)\n```",
                              (ROOT / _d).read_text(encoding="utf-8"), _re.S):
            try:
                _obj = json.loads(_b)
            except ValueError:
                continue
            _rg = _obj.get("repo_governor")
            if not isinstance(_rg, dict):
                continue
            _frags += 1
            _bad += [f"{_d}: {k!r}" for k in _rg if k not in _allowed]
    # Floor control: a pattern that matches nothing passes the claim below
    # having read no documentation at all.
    okf = _frags > 0
    extra += not okf
    print(f"  [{'PASS' if okf else 'FAIL'}] the install docs state a manifest fragment to check ({_frags})"
          + ("" if okf else "\n         found none -- either the example was removed or the "
                            "scan stopped matching it; both need a human"))
    oke = not _bad
    extra += not oke
    print(f"  [{'PASS' if oke else 'FAIL'}] every documented repo_governor key is one the schema permits"
          + ("" if oke else f"\n         refused by the schema: {_bad} -- following the "
                            "documentation would produce MANIFEST_INVALID and degrade "
                            "every verdict to UNKNOWN"))

    fails += extra
    total = len(CASES) + len(PERM_CASES) + 9
    print(f"\n{total - fails}/{total} checks passed")
    print("MANIFEST LOADER: " + ("CONFORMANT" if not fails else f"NON-CONFORMANT ({fails})"))
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
