#!/usr/bin/env python3
"""Provider pluggability is a checked property, not a described one (ADR-021).

The provider architecture was called complete while `engine/completion.py`
named five adapters directly and spawned every one of them without consulting
`permitted()`. Documentation cannot prevent that from growing back; a test can.

Each check below fails loudly if the engine reacquires knowledge it should not
have, or if the permission gate stops being a gate.

Usage:  python3 conformance/bindings.py
"""

from __future__ import annotations

import copy
import os as _os
import json
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import _preflight  # noqa: E402
sys.path.insert(0, str(ROOT / "engine"))
import bindings as B  # noqa: E402
import manifest as MF  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("bindings")

BASE = json.loads((ROOT / ".repo-governor.json").read_text())

# Only this module may spawn an adapter. `onboard.py` may NAME adapter paths --
# it proposes bindings for a human to accept (ADR-010) -- but it must never run
# one, and `completion.py` must not even name one.
SPAWNER = "bindings.py"
DECISION_PATH = ("completion.py",)


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"    {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def with_manifest(mutate):
    d = copy.deepcopy(BASE)
    mutate(d)
    f = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False)
    json.dump(d, f)
    f.close()
    m, errs = MF.load(f.name)
    Path(f.name).unlink()
    return m, errs


def main():
    absent = _preflight.banner()
    fails = 0
    print("Engine holds no adapter knowledge\n")

    # 1. The decision path names no adapter at all.
    for name in DECISION_PATH:
        text = (ROOT / "engine" / name).read_text()
        fails += check(f"engine/{name} contains no 'adapters/' literal",
                       "adapters/" not in text)

    # 2. Exactly one module spawns an adapter. Detected structurally: an adapter
    #    spawn is `sys.executable` pointed at a repository-relative path.
    spawners = []
    for p in sorted((ROOT / "engine").glob("*.py")):
        t = p.read_text()
        if "sys.executable" in t and "ROOT /" in t:
            spawners.append(p.name)
    fails += check(f"only {SPAWNER} spawns an adapter", spawners == [SPAWNER],
                   f"also spawning: {[s for s in spawners if s != SPAWNER]}")

    print("\nThe gate is a gate\n")

    # 3. A denied read returns a disposition and never reaches the provider.
    #    Asserted by pointing the binding at an adapter that does not exist: if
    #    the call were to spawn, it would fail with something other than
    #    PERMISSION_DENIED. Denial must win before resolution is even attempted.
    m, errs = with_manifest(lambda d: (
        d["permissions"]["architecture"].__setitem__("read", False),
        d["providers"]["architecture"][0].__setitem__("adapter", "adapters/adr")))
    r = B.call("architecture", "get_constraints", {"id": "x"}, manifest=m)
    fails += check("denied read returns PERMISSION_DENIED, not provider output",
                   r.get("error", {}).get("type") == "PERMISSION_DENIED",
                   json.dumps(r)[:120])

    # 4. Denial is a disposition, not an exception (ADR-005 rule 5).
    fails += check("denial is a typed envelope, not a raise", r.get("ok") is False)

    # 5. An unbound role is refused by name, not by crashing on a missing key.
    r = B.call("retirement_x", "anything", {}, manifest=BASE)
    fails += check("unbound role returns UNBOUND_ROLE or PERMISSION_DENIED",
                   r.get("error", {}).get("type") in ("UNBOUND_ROLE", "PERMISSION_DENIED"),
                   json.dumps(r)[:120])

    # 6. A role bound but never granted any verb is denied, so adding a binding
    #    is not itself a grant. This is INV-014 at the engine boundary: the
    #    provider is present and reachable, and still may not be used.
    m_ungranted, _ = with_manifest(lambda d: d["permissions"].pop("architecture", None))
    r = B.call("architecture", "get_constraints", {"id": "x"}, manifest=m_ungranted)
    fails += check("a bound role with no permission block is denied",
                   r.get("error", {}).get("type") == "PERMISSION_DENIED",
                   json.dumps(r)[:120])

    print("\nWriter selection is by capability, never by name\n")

    m, errs = MF.load()
    assert not errs, errs

    # 7. The chosen writer is one that ADVERTISES the function.
    b, err = B.writer_for("decision_history", "record_decision", m)
    ok = err is None and "record_decision" in (B.describe(b).get("writers") or {})
    fails += check("selected writer advertises record_decision", ok,
                   json.dumps(err or {})[:120])

    # 8. A function nothing advertises is refused, rather than sent to the first
    #    binding and failing deep inside the adapter.
    b2, err2 = B.writer_for("decision_history", "delete_everything", m)
    fails += check("unadvertised writer function is refused up front",
                   b2 is None and err2["error"]["type"] == "NO_WRITABLE_PROVIDER")

    # 9. Selection does not depend on the adapter's name. Renaming the type must
    #    not change the outcome -- this is the specific regression that the old
    #    `startswith("decision-history-dolt")` test would have produced.
    m_renamed, _ = with_manifest(
        lambda d: d["providers"]["decision_history"][0].__setitem__("type", "zzz-anonymous-store"))
    b3, err3 = B.writer_for("decision_history", "record_decision", m_renamed)
    fails += check("renaming the adapter type does not change selection",
                   err3 is None and b3 is not None and b3["adapter"] == b["adapter"],
                   json.dumps(err3 or {})[:120])

    print("\nConfiguration comes from the manifest, not from engine code\n")

    # 10. Binding `env` reaches the adapter process.
    m_env, _ = with_manifest(
        lambda d: d["providers"]["architecture"][0].__setitem__("env", {"RG_PROBE": "seen"}))
    env = B._env_for(m_env["providers"]["architecture"][0])
    fails += check("binding.env reaches the adapter environment", env.get("RG_PROBE") == "seen")

    # 11. The whole binding is offered as structured config, so an adapter can
    #     read a declared admission signal without the engine knowing its name.
    fails += check("REPO_GOVERNOR_BINDING carries the binding as JSON",
                   json.loads(env["REPO_GOVERNOR_BINDING"]).get("adapter") is not None)

    # 12. A credential cannot be smuggled in through the new field.
    _, errs_secret = with_manifest(
        lambda d: d["providers"]["architecture"][0].__setitem__(
            "env", {"TOKEN": "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123"}))
    fails += check("secret-shaped value in binding.env fails the load",
                   any("SECRET" in e for e in errs_secret), str(errs_secret)[:120])

    print("\nThe governed repository is where you point it\n")

    # 13. TARGET is not ROOT. `bindings.target()` must follow the declaration,
    #     not the engine's install path -- the defect that made every repo-local
    #     provider read this repository whatever it was pointed at (#24).
    import os as _os, subprocess as _sp, tempfile as _tf
    with _tf.TemporaryDirectory() as td:
        other = Path(td) / "other"
        (other / "docs" / "adrs").mkdir(parents=True)
        (other / "docs" / "adrs" / "001-only-one.md").write_text(
            "# 1. Only One\n\n**Status**: Accepted\n")
        _sp.run(["git", "init", "-q", str(other)], check=False, capture_output=True)
        _sp.run(["git", "-C", str(other), "remote", "add", "origin",
                 "https://github.com/elsewhere/other.git"], check=False, capture_output=True)
        (other / ".repo-governor.json").write_text(json.dumps({
            "repo_governor": {"version": 1},
            "repository": {"id": "elsewhere/other"},
            "condition": {"assessed": "L1", "profile": "GOVERNOR_LITE"},
            "providers": {
                "repository": {"type": "git", "adapter": "adapters/git", "contract_version": 1},
                "architecture": [{"type": "adr", "adapter": "adapters/adr", "contract_version": 1}]},
            "permissions": {"repository": {"read": True}, "architecture": {"read": True}}}))
        env0 = _os.environ.get("REPO_GOVERNOR_TARGET")
        _os.environ["REPO_GOVERNOR_TARGET"] = str(other)
        try:
            fails += check("target() follows the declaration, not the install path",
                           B.target() == other.resolve(), f"got {B.target()}")
            r = B.call("architecture", "get_provenance", {})
            n = (r.get("value") or {}).get("documents")
            fails += check("a foreign repository's own decisions are read", n == 1, f"read {n}")
            ref = (r.get("provenance") or [{}])[0].get("ref", "")
            fails += check("provenance names the governed repository",
                           ref.startswith("elsewhere/other//"), f"ref={ref!r}")
            # The decisive one: governing A must never cite B.
            fails += check("governing one repository never cites another",
                           "repo-governor" not in ref, f"ref={ref!r}")
        finally:
            if env0 is None:
                _os.environ.pop("REPO_GOVERNOR_TARGET", None)
            else:
                _os.environ["REPO_GOVERNOR_TARGET"] = env0

    # 14. No adapter names the author's repository. A default identity means a
    #     foreign repository gets confident answers about someone else's project
    #     (#26). A blunt grep on purpose -- a clever check is easier to defeat.
    named = [p.name for p in sorted((ROOT / "adapters").iterdir())
             if p.is_file() and "tosin2013" in p.read_text(errors="ignore")]
    fails += check("no adapter contains the author's repository slug", not named, str(named))

    print("\nDetection and execution are wired to the same value\n")

    import subprocess as _sp2, tempfile as _tf2
    with _tf2.TemporaryDirectory() as td:
        alt = Path(td) / "alt"
        (alt / "docs" / "adr").mkdir(parents=True)          # NOT docs/adrs
        (alt / "docs" / "adr" / "001-d.md").write_text("# 1. D\n\n**Status**: Accepted\n")
        (alt / "docs" / "adr" / "002-d.md").write_text("# 2. D\n\n**Status**: Accepted\n")

        # 15. Onboarding must FIND a non-default layout. `doc/adr` is adr-tools'
        #     default and `adrs/` is common; missing them reported real
        #     collections as no provider at all (#27).
        out = _sp2.run([sys.executable, str(ROOT / "engine" / "onboard.py"), str(alt)],
                       capture_output=True, text=True, cwd=str(ROOT), timeout=120).stdout
        fails += check("onboarding detects ADRs outside docs/adrs", "docs/adr:" in out, out[:120])

        # 16. And the path it detected must REACH the adapter. Detection wrote
        #     `path` into the proposed binding and nothing read it, so a detected
        #     provider governed as PROVIDER_UNAVAILABLE.
        r = _sp2.run([sys.executable, str(ROOT / "adapters" / "adr"),
                      "query", "architecture", "get_provenance"],
                     capture_output=True, text=True, cwd=str(alt), timeout=60,
                     env={**_os.environ,
                          "REPO_GOVERNOR_BINDING": json.dumps({"adapter": "adapters/adr",
                                                               "path": "docs/adr"})})
        got = json.loads(r.stdout)
        fails += check("a binding `path` reaches the adapter",
                       (got.get("value") or {}).get("documents") == 2, r.stdout[:120])

    # 17. Detection must not promise more than the adapter can read. It counted
    #     statuses with a substring test (20 of 22) where the adapter's parser
    #     read 2 -- ADR-010 stops detection assigning authority, not overstating
    #     capability.
    src = (ROOT / "engine" / "onboard.py").read_text()
    fails += check("detection counts statuses with the adapter's parser, not a substring test",
                   '"## Status" in' not in src and "_adr_status(" in src)

    # ADR-027, the case detection could not tell apart on its own. Installing the
    # skill clones this repository into <target>/.agents/skills/repo-governor.
    # That clone is a git repository; if it also carried a manifest, an agent
    # standing in it would resolve the INSTALL as the repository under
    # governance. Observed live: the engine reported another project's issue 8
    # complete, citing this repository's acceptance criteria.
    import tempfile
    for root in MF.SKILL_ROOTS:
        with tempfile.TemporaryDirectory() as tmp:
            inst = Path(tmp) / "outer" / root / "skills" / "repo-governor"
            inst.mkdir(parents=True)
            (Path(tmp) / "outer").mkdir(exist_ok=True)
            got = MF._escape_install(inst)
            fails += check(f"an install under {root}/skills resolves to the repo containing it",
                           got == (Path(tmp) / "outer").resolve(), f"got {got}")

    with tempfile.TemporaryDirectory() as tmp:
        plain = Path(tmp) / "ordinary-repo"
        plain.mkdir()
        fails += check("an ordinary repository is left alone",
                       MF._escape_install(plain.resolve()) == plain.resolve())

    print("\nThe engine does not hand adapters its own input\n")

    # Behavioural, not a source grep: call _env_for and look at what an adapter
    # would actually receive. REPO_GOVERNOR_TARGET is the engine's input; no
    # adapter reads it, and exporting it meant every grandchild inherited it --
    # so a hook firing inside an adapter subprocess ran the engine, which
    # resolved the INHERITED target and announced a repository it was not in.
    _b = m["providers"]["repository"]
    _env = B._env_for(_b)
    fails += check("REPO_GOVERNOR_TARGET is not exported to adapters",
                   "REPO_GOVERNOR_TARGET" not in _env,
                   "the engine's own input reaching a grandchild is issue 54")
    for keep in ("REPO_GOVERNOR_SUBJECT", "REPO_GOVERNOR_BINDING"):
        fails += check(f"{keep} IS still exported", keep in _env,
                       "adapters/_protocol.py reads it; removing it would break targeting "
                       "rather than fix a leak")

    # End to end on the real path: a command the engine runs through an adapter
    # must not see the variable. This is the leak as it actually occurred.
    import tempfile as _tf, os as _os
    with _tf.NamedTemporaryFile("w", suffix=".sh", delete=False) as fh:
        fh.write('echo "TARGET=${REPO_GOVERNOR_TARGET:-unset}"\n')
        _probe = fh.name
    try:
        r = B.call("repository", "evaluate_check",
                   {"check": "command_exit", "target": f"sh {_probe}"}, manifest=m)
        # The adapter reports only the exit code, so assert via a command that
        # FAILS when the variable is set -- the observable form of the leak.
        with open(_probe, "w") as fh:
            fh.write('[ -z "${REPO_GOVERNOR_TARGET:-}" ]\n')
        r2 = B.call("repository", "evaluate_check",
                    {"check": "command_exit", "target": f"sh {_probe}"}, manifest=m)
        got = (r2.get("value") or {}).get("satisfied")
        fails += check("a command run through an adapter does not inherit it", got is True,
                       "a command_exit criterion, or a hook, would resolve the wrong repository")
    finally:
        _os.unlink(_probe)

    # ADR-033 decision 3, the tripwire. A repo-local provider answers about the
    # CHECKED-OUT REVISION. adapters/speckit reads specs/<feature>/, which Spec
    # Kit populates per feature branch, so it can answer differently on two
    # checkouts of one repository. The obligation is: the day the engine reads a
    # branch-varying method, the recorded provenance has to name the revision,
    # and this is what makes that day visible. The check cannot say what the
    # right answer is when it fires -- only that ADR-033 is now due for a second
    # reading.
    #
    # IT USED TO SCAN FOR THE LITERAL PAIR `"architecture", "get_..."`, WHICH
    # HAD TWO HOLES.
    #
    # First, it read one role. Decision 3 is written about "a branch-varying
    # provider method", not an architecture one, so six roles were unguarded by
    # a check whose ADR calls it structural.
    #
    # Second, and worse, a text scan for adjacent literals cannot see dispatch
    # through a variable. `engine/completion.py` calls the execution role in a
    # loop -- `B.call("execution", fn, ...)` with `fn` bound per iteration -- so
    # the pair never appears adjacent and three engine calls were invisible to
    # a check that reported PASS. A scanner that silently skips what it cannot
    # parse is the failure this repository keeps finding: the method could not
    # have produced the answer it gave.
    #
    # So: AST, every role, and an unresolvable call site is REPORTED rather than
    # skipped.
    print("\nThe engine reads no unreviewed provider method, in any role (ADR-033)\n")

    import ast as _ast

    ROLES = {"roadmap_authority", "architecture", "execution", "repository",
             "change_signals", "retirement", "decision_history", "acceptance_criteria"}

    # What the engine reads today. SEEDING IS NOT REVIEW -- it is a baseline, so
    # that the NEXT method added is the thing that shows up. That was true of
    # the architecture set this replaces, and it is stated here rather than
    # implied.
    REVIEWED = {
        "roadmap_authority":   {"get_authority", "get_scope", "get_non_goals"},
        "architecture":        {"get_constraints", "get_active_decisions",
                                "get_superseded_decisions"},
        "acceptance_criteria": {"get_criteria"},
        "repository":          {"evaluate_check"},
        "decision_history":    {"get_decisions", "record_decision"},
        "retirement":          {"obligation_check"},
        "execution":           {"get_active_work", "get_completed_work", "get_discoveries"},
    }

    # Call sites whose method is not a literal. Each must be declared with the
    # methods reachable there, and those are checked against REVIEWED like any
    # other. A new dynamic site fails rather than passing unseen.
    DYNAMIC = {
        ("completion.py", "execution"): {"get_active_work", "get_completed_work",
                                         "get_discoveries"},
    }

    # Wrappers that take the role itself as an argument. Their own call sites
    # carry the literals and are matched above; the wrapper body is not a
    # distinct read.
    PASSTHROUGH = {"envelope.py:56"}

    resolved, sites, dynamic_seen, passthrough = {}, {}, set(), set()
    for f in sorted((ROOT / "engine").glob("*.py")):
        tree = _ast.parse(f.read_text(encoding="utf-8"))
        for node in _ast.walk(tree):
            if not isinstance(node, _ast.Call) or len(node.args) < 2:
                continue
            # Only actual provider invocations. `engine/onboard.py` calls
            # add("execution", "beads", ...) -- role first, PROVIDER TYPE
            # second -- which a role-name filter alone reads as a method call
            # and reports as an unreviewed method. Being wrong in that
            # direction is loud rather than silent, but it is still wrong.
            callee = node.func
            name = (callee.attr if isinstance(callee, _ast.Attribute)
                    else callee.id if isinstance(callee, _ast.Name) else "")
            if not (name == "call" or name.startswith("_ask")):
                continue
            a0 = node.args[0]
            if not (isinstance(a0, _ast.Constant) and a0.value in ROLES):
                # The ROLE itself is not a literal. `engine/envelope.py`'s
                # `_ask` wrapper is this: B.call(role, fn, ...) with both bound
                # by its caller. Harmless there because its call sites pass
                # literals and are matched above -- but a site like it outside a
                # wrapper would reach any role with any method, invisibly. Same
                # rule as unresolved methods: declared, or reported.
                if not isinstance(a0, _ast.Constant):
                    passthrough.add(f"{f.name}:{node.lineno}")
                continue
            role, a1 = a0.value, node.args[1]
            if isinstance(a1, _ast.Constant) and isinstance(a1.value, str):
                resolved.setdefault(role, set()).add(a1.value)
                sites.setdefault((role, a1.value), []).append(f"{f.name}:{node.lineno}")
            else:
                dynamic_seen.add((f.name, role))

    # FLOOR CONTROL. A walk that matches nothing makes every claim below
    # vacuous, and "the engine reads no unreviewed method" would be the loudest
    # possible false pass. Two floors, because the old check had one and still
    # missed a whole role: assert the resolved scan found calls, AND assert the
    # dynamic scan is working by requiring the known dynamic site to be seen.
    fails += check(f"the engine's provider calls were actually found "
                   f"({sum(len(v) for v in resolved.values())} across {len(resolved)} roles)",
                   len(resolved) >= 4,
                   "the walk matched almost nothing -- a broken scan reads as an engine "
                   "that consults no provider at all")
    fails += check("the scan sees dispatch through a variable, not only literals",
                   bool(dynamic_seen),
                   "the previous text scan could not, and three execution calls were "
                   "invisible to it while it reported PASS")

    unreviewed = {f"{role}.{fn}": sites[(role, fn)]
                  for role, fns in sorted(resolved.items())
                  for fn in sorted(fns - REVIEWED.get(role, set()))}
    fails += check("every engine provider call is a reviewed method",
                   not unreviewed,
                   f"unreviewed: {unreviewed}. A repo-local provider answers about the "
                   "checked-out revision; ADR-033 decision 3 requires the recorded "
                   "provenance to name it before the engine may read such a method. "
                   "Adding the method to REVIEWED is a decision, not a fix.")

    undeclared = sorted(dynamic_seen - set(DYNAMIC))
    fails += check("every dynamic call site is declared with the methods it reaches",
                   not undeclared,
                   f"undeclared dynamic dispatch: {undeclared}. The scanner cannot resolve "
                   "these, so they must be declared or they pass unseen.")
    undeclared_pt = sorted(passthrough - PASSTHROUGH)
    fails += check("every role-agnostic call site is a declared wrapper",
                   not undeclared_pt,
                   f"undeclared pass-through: {undeclared_pt}. A call whose ROLE is a "
                   "variable can reach any provider; it is a wrapper whose callers are "
                   "checked, or it is a hole.")
    fails += check("and the declared wrappers still exist",
                   passthrough >= PASSTHROUGH,
                   f"declared but not found: {sorted(PASSTHROUGH - passthrough)} -- a "
                   "declaration for a site that moved excuses nothing and hides the next one")

    for (fname, role), fns in sorted(DYNAMIC.items()):
        extra = sorted(fns - REVIEWED.get(role, set()))
        fails += check(f"{fname}'s dynamic {role} calls are reviewed methods",
                       not extra, f"unreviewed: {extra}")

    print(f"\n{'BINDINGS: CONFORMANT' if not fails else f'BINDINGS: NON-CONFORMANT ({fails})'}")
    if fails:
        _preflight.attribute(absent)
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
