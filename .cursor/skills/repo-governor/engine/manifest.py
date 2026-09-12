#!/usr/bin/env python3
"""Governance manifest loader (ADR-004, ADR-005, ADR-013, ADR-015, ADR-016).

Loads `.repo-governor.json` and fails closed. The manifest is the SOLE
artifact that binds providers to governance roles (INV-013) — a detected,
reachable, credentialed provider absent from this file has no role and is
never consulted.

Rules enforced here, beyond the schema:

  version gate     an unimplemented version refuses to evaluate, not guesses
  cardinality      single-valued authority roles vs multi-valued evidence roles
  no secrets       a token-shaped value fails the load outright
  adapter paths    must resolve inside the repository
  deny by default  unlisted role, unlisted verb, malformed block => deny
  execute reserved rejected at v1

Usage:  python3 engine/manifest.py [path]        validate and summarise
        python3 engine/manifest.py --explain <role> <verb>
"""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from jsonschema_mini import validate  # noqa: E402

# ROOT is where the ENGINE lives. It resolves the schema and adapter paths, and
# nothing else. The repository being GOVERNED is a separate question (ADR-027) --
# conflating them made every repo-local provider read the engine's own
# repository whatever it was pointed at, with provenance that looked correct.
ROOT = Path(__file__).resolve().parent.parent
SCHEMA = ROOT / "schemas" / "manifest-v1.json"


def _git_toplevel(start):
    import subprocess  # noqa: PLC0415 -- only needed on this path
    try:
        p = subprocess.run(["git", "-C", str(start), "rev-parse", "--show-toplevel"],
                           capture_output=True, text=True, timeout=20)
    except (FileNotFoundError, OSError):
        return None
    return Path(p.stdout.strip()) if p.returncode == 0 and p.stdout.strip() else None


def target():
    """The repository under governance.

    Declared by `REPO_GOVERNOR_TARGET`; otherwise the git repository enclosing
    the working directory; otherwise the working directory itself. Never the
    engine's install directory, unless that is genuinely where you are standing.

    A governance manifest describes the repository it sits in (ADR-004), so the
    manifest is read from here too. Reading the engine's manifest while
    governing somewhere else binds providers nobody declared for that place.
    """
    declared = os.environ.get("REPO_GOVERNOR_TARGET")
    if declared:
        start = Path(declared).resolve()
        return (_git_toplevel(start) or start).resolve()
    start = Path.cwd()
    resolved = (_git_toplevel(start) or start).resolve()
    return _escape_install(resolved)


# Skills directories, by host. `docs/installation.md` carries the full table.
SKILL_ROOTS = (".agents", ".claude", ".cursor", ".codex")


def _escape_install(repo):
    """An installed copy of this engine is not the repository under governance.

    Installing the skill clones this repository into `<target>/.agents/skills/
    repo-governor`. That clone is a git repository, and it ships a manifest
    binding `tosin2013/repo-governor` -- so an agent whose working directory
    lands inside it resolves the INSTALL as the target and answers questions
    about Repo Governor while standing in somebody else's project.

    Observed live: asked about issue 8 in an unrelated repository, the engine
    reported that repository's issue 8 as complete, citing acceptance criteria
    that belong to this one. It is not a hypothetical off-by-one; it produced a
    confident, fully-cited answer to a question nobody asked.

    So: when the resolved repository sits under a `<host>/skills/` path, keep
    walking outward to the repository that CONTAINS the install. ADR-027 said
    the governed repository is not the install directory; this is the case
    where detection could not tell them apart on its own.
    """
    parts = repo.parts
    for i in range(len(parts) - 1, 0, -1):
        if parts[i] == "skills" and parts[i - 1] in SKILL_ROOTS:
            outer = Path(*parts[:i - 1])
            return (_git_toplevel(outer) or outer).resolve()
    return repo

SUPPORTED_VERSIONS = (1,)

# ADR-013 cardinality. Roles answering "is this authorized?" are single-valued:
# two sources of authorization is no source of authorization.
SCALAR_ROLES = ("roadmap_authority", "execution", "repository", "acceptance_criteria")
ARRAY_ROLES = ("architecture", "change_signals", "retirement", "decision_history")

# Findings that describe a CONFIGURATION GAP rather than a broken binding.
# Closed set, deliberately: `--validate` treats any other kind as blocking, so
# forgetting to list one costs a false refusal and never a false green
# (issue 180).
ADVISORY_FINDINGS = ("REQUIRED_ROLE_UNBOUND", "ARTIFACT_UNTRACKED",
                     "TRACKEDNESS_UNKNOWN", "MANIFEST_UNTRACKED",
                     "HISTORY_NOT_PORTABLE", "PORTABILITY_UNDECLARED")

# MANIFEST_UNTRACKED is advisory, and the banner is qualified instead.
#
# It was blocking first, which broke onboarding: a manifest is legitimately
# uncommitted between the moment the operator promotes the onboarding
# proposal to a manifest and the moment they commit it, and
# --validate is what the operator runs IN that window. Refusing there would
# make the tool fail at the one moment it is meant to help.
#
# The repo's own taxonomy decides it. An uncommitted manifest is a
# CONFIGURATION GAP, not a broken binding -- every binding answers, on this one
# machine -- and ADR-031 fixes the posture for gaps: "Report the fact, state the
# consequence, block nothing." So it reports and does not refuse. What it must
# never do is print READY_FOR_GOVERNANCE unqualified, because that is the false
# green that let a repository lose its authority to a decommissioned host; see
# the banner below.

VERBS = ("read", "write", "create", "update", "archive", "comment", "transition")
RESERVED_VERBS = ("execute",)

# ADR-005 rule 3: credentials never live here. Detection is structural, not advisory.
SECRET_PREFIXES = ("ghp_", "gho_", "ghs_", "github_pat_", "lin_api_", "xox",
                   "sk-", "AKIA", "glpat-", "-----BEGIN")
SECRET_KEYS = re.compile(r"(token|secret|password|passwd|api_?key|credential)", re.I)


class ManifestError(ValueError):
    pass


def _looks_secret(key, value):
    if isinstance(value, str):
        if any(value.startswith(p) for p in SECRET_PREFIXES):
            return f"value at {key} starts with a known credential prefix"
        # High-entropy long opaque string in a config file is a smell -- but a
        # PATH is not opaque. `adapters/decision-history-github` is 32 chars and
        # matched the original pattern, which is the kind of false positive that
        # gets a security check switched off. Credentials do not contain path
        # separators or dots; require their absence before flagging.
        if (len(value) >= 32 and "/" not in value and "." not in value
                and re.fullmatch(r"[A-Za-z0-9+/=_-]{32,}", value)):
            return f"value at {key} looks like an opaque credential ({len(value)} chars)"
    if SECRET_KEYS.search(str(key)):
        return f"key {key!r} names a credential; credentials come from the environment"
    return None


def _scan_secrets(node, path="$"):
    found = []
    if isinstance(node, dict):
        for k, v in node.items():
            hit = _looks_secret(f"{path}.{k}", v)
            if hit:
                found.append(hit)
            found += _scan_secrets(v, f"{path}.{k}")
    elif isinstance(node, list):
        for i, v in enumerate(node):
            found += _scan_secrets(v, f"{path}[{i}]")
    return found


def load(path=None):
    """Return (manifest, errors). A non-empty error list means DO NOT evaluate."""
    p = Path(path) if path else (target() / ".repo-governor.json")
    if not p.exists():
        # Absent manifest is not an error; the repository is un-onboarded.
        return None, [f"AUTHORITY_SOURCE_MISSING: no manifest at {p}"]
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError as e:
        return None, [f"MALFORMED: {p} is not valid JSON: {e}"]

    # 1. version gate BEFORE schema, so a future version fails cleanly.
    ver = (data.get("repo_governor") or {}).get("version")
    if ver not in SUPPORTED_VERSIONS:
        return None, [f"UNSUPPORTED_VERSION: manifest version {ver!r}; "
                      f"this engine implements {list(SUPPORTED_VERSIONS)}. Refusing to guess."]

    errs = []

    # 2. schema
    schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
    errs += [f"SCHEMA: {e}" for e in validate(data, schema)]

    # 3. secrets — structural, not advisory
    errs += [f"SECRET: {s}" for s in _scan_secrets(data)]

    providers = data.get("providers") or {}

    # 4. cardinality (ADR-013)
    for role in SCALAR_ROLES:
        if role in providers and isinstance(providers[role], list):
            errs.append(f"CARDINALITY: {role!r} is single-valued; got a list. "
                        "Two sources of authorization is no source of authorization.")
    for role in ARRAY_ROLES:
        if role in providers and not isinstance(providers[role], list):
            errs.append(f"CARDINALITY: {role!r} is multi-valued; got a scalar.")

    # 5. repository role always required
    if "repository" not in providers:
        errs.append("MISSING_ROLE: 'repository' is the only always-required role (ADR-003 rule 4).")

    # 6. adapter paths must stay inside the repository
    for role, binding in providers.items():
        if role.startswith("$"):
            continue
        for b in (binding if isinstance(binding, list) else [binding]):
            if not isinstance(b, dict):
                continue
            adapter = b.get("adapter", "")
            resolved = (ROOT / adapter).resolve()
            if not str(resolved).startswith(str(ROOT)):
                errs.append(f"ADAPTER_PATH: {role} adapter {adapter!r} escapes the repository.")
            elif not resolved.exists():
                errs.append(f"ADAPTER_MISSING: {role} adapter {adapter!r} does not exist.")

    # 7. permissions: closed verb set, reserved verbs rejected
    for role, block in (data.get("permissions") or {}).items():
        if role.startswith("$"):
            continue  # $comment and friends are annotations, not roles
        if role not in providers:
            errs.append(f"PERMISSION_ORPHAN: permissions declared for unbound role {role!r}.")
        if not isinstance(block, dict):
            errs.append(f"PERMISSION_MALFORMED: {role!r} block is not an object; resolves to deny.")
            continue
        for verb in block:
            if verb.startswith("$"):
                continue  # annotation, not a verb
            if verb in RESERVED_VERBS:
                errs.append(f"PERMISSION_RESERVED: {verb!r} is reserved and unimplemented at v1 "
                            "(ADR-005 rule 6).")
            elif verb not in VERBS:
                errs.append(f"PERMISSION_UNKNOWN_VERB: {verb!r} on {role!r}; "
                            f"closed set is {list(VERBS)}.")

    return (None if errs else data), errs


def permitted(manifest, role, verb):
    """Deny by default. Absence is denial, never inheritance or inference."""
    if not manifest:
        return False, "no manifest loaded"
    if role not in (manifest.get("providers") or {}):
        return False, f"role {role!r} is not bound"
    block = (manifest.get("permissions") or {}).get(role)
    if not isinstance(block, dict):
        return False, f"no permission block for {role!r} — absence is denial"
    if verb not in block:
        return False, f"verb {verb!r} not granted on {role!r} — absence is denial"
    if block[verb] is not True:
        return False, f"verb {verb!r} explicitly denied on {role!r}"
    return True, f"granted in manifest permissions.{role}.{verb}"


def check_adapters(manifest):
    """ADR-004 step 3: does each adapter actually satisfy its declared contract?

    A manifest can declare contract_version 1 for an adapter that implements
    something else. The declaration is a claim; this verifies it.
    """
    # Spawning happens in bindings.py and nowhere else (ADR-021), so this asks
    # it rather than running the adapter itself. Imported here, not at module
    # scope, because bindings.py imports this module.
    import bindings as B  # noqa: PLC0415
    findings = []
    for role, binding in (manifest.get("providers") or {}).items():
        if role.startswith("$"):
            continue
        for b in (binding if isinstance(binding, list) else [binding]):
            adapter = b["adapter"]
            declared = b.get("contract_version")
            d = B.describe(b, use_cache=False)
            if not d:
                findings.append((role, adapter, "UNREACHABLE",
                                 "describe returned nothing parseable"))
                continue
            if d.get("role") != role:
                findings.append((role, adapter, "ROLE_MISMATCH",
                                 f"adapter serves {d.get('role')!r}, manifest binds it to {role!r}"))
            actual = d.get("contract_version")
            if declared is not None and actual != declared:
                findings.append((role, adapter, "CONTRACT_MISMATCH",
                                 f"manifest declares {declared}, adapter implements {actual}"))
            if not d.get("transport", {}).get("reachable", True):
                findings.append((role, adapter, "TRANSPORT_UNREACHABLE",
                                 d["transport"].get("detail")
                                 or "adapter advertises no capabilities; "
                                    "transport is not configured"))
            # A binding can be well-formed, reachable, and still answer nothing.
            # --validate reported READY_FOR_GOVERNANCE for exactly that, and it
            # is what a person runs to find out whether they onboarded
            # correctly -- so a green verdict there is worse than a red one,
            # because the next command says PROVIDER_UNAVAILABLE and they have
            # to work out which to believe (issue 51).
            #
            # The adapter answers this, not the engine: which variable it needs
            # is adapter-specific knowledge ADR-003 keeps out of here.
            if d.get("transport", {}).get("configured", True) is False:
                findings.append((role, adapter, "NOT_CONFIGURED",
                                 d["transport"].get("configured_detail")
                                 or "adapter reports it is not configured"))
            # Drift: the manifest grants a write the transport cannot serve.
            # Capabilities belong to (provider x transport), so a grant made when
            # the transport was writable can silently stop being honourable (#17).
            granted, _ = permitted(manifest, role, "write")
            if granted and d.get("transport", {}).get("writable") is False:
                findings.append((role, adapter, "WRITE_GRANTED_BUT_TRANSPORT_READONLY",
                                 "manifest grants write; the bound transport reports writable=false"))
    return findings


def _git_out(args, cwd):
    """Run git plumbing. Returns (rc, stdout); rc is None when git is unavailable."""
    import subprocess  # noqa: PLC0415 -- only needed on this path
    try:
        r = subprocess.run(["git", "-C", str(cwd), *args],
                           capture_output=True, text=True, timeout=20)
        return r.returncode, r.stdout
    except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
        return None, ""


def _artifact_classes(manifest, root, mp):
    """(label, kind, paths, why) for each class of artifact that carries governance.

    A class rather than a file, so a repository with 88 acceptance bars produces
    one finding naming a count instead of 88 findings nobody reads.
    """
    out = [("the manifest", "MANIFEST_UNTRACKED", [mp] if mp.exists() else [],
            "the manifest IS the authority (ADR-004): bindings, permissions, "
            "profile and the declared admission signal (ADR-018)")]

    bars = sorted((root / ".repo-governor" / "acceptance").glob("*.json"))
    out.append(("acceptance bars", "ARTIFACT_UNTRACKED", bars,
                "completion bars are repo-local by ADR-017; an untracked bar "
                "cannot be evaluated on any other checkout"))

    # Only when a file-backed decision store is actually bound. A store nobody
    # binds is not a governance artifact, and demanding it would be inventing a
    # requirement the manifest never made (INV-013).
    dh = (manifest.get("providers") or {}).get("decision_history") or []
    if any((b or {}).get("type") == "decision-history-file"
           for b in (dh if isinstance(dh, list) else [dh]) if isinstance(b, dict)):
        d = Path(os.environ.get("REPO_GOVERNOR_DECISIONS_DIR",
                                root / ".repo-governor" / "decisions"))
        f = Path(d) / "decisions.jsonl"
        out.append(("the decision log", "ARTIFACT_UNTRACKED", [f] if f.exists() else [],
                    "a decision history bound to a file backend is portable only "
                    "if the file is committed (ADR-019 rule 2)"))
    return out


def check_tracked(manifest, path=None):
    """Findings for governance artifacts git would not carry to another host.

    Present-on-one-machine is not governance. A manifest nobody committed is
    invisible to every other host -- that host reports AUTHORITY_SOURCE_MISSING
    and reads the repository as un-onboarded, while this surface reports
    READY_FOR_GOVERNANCE. Onboarding says "rename and commit" in a $comment and
    a docstring; neither is a check.

    Severity is deliberately uneven. An untracked MANIFEST blocks: a repository
    claiming READY_FOR_GOVERNANCE on a file no other host can see asserts
    something false, and the tempting repair -- re-onboarding -- is not recovery
    but re-founding, silently redeciding bindings, profile and admission.
    Untracked bars or decisions are advisory: still governed, just answering
    less on the next clone.
    """
    root = target()
    mp = Path(path) if path else (root / ".repo-governor.json")

    rc, _ = _git_out(["rev-parse", "--is-inside-work-tree"], root)
    if rc != 0:
        return [("(governance artifacts)", str(root), "TRACKEDNESS_UNKNOWN",
                 "not a git worktree, or git is unavailable, so whether these "
                 "artifacts reach another host cannot be determined. Unknown is "
                 "not tracked (ADR-003 rule 6).")]

    findings = []
    for label, kind, paths, why in _artifact_classes(manifest, root, mp):
        if not paths:
            continue
        rels = [os.path.relpath(p, root) for p in paths]
        _, listed = _git_out(["ls-files", "--", *rels], root)
        tracked = {ln for ln in listed.splitlines() if ln}
        missing = [r for r in rels if r not in tracked]
        if not missing:
            continue
        # An actively-ignored artifact is the worse case and gets said so: it is
        # not an oversight a `git add` fixes.
        irc, _ = _git_out(["check-ignore", "-q", "--", missing[0]], root)
        excluded = " A .gitignore rule EXCLUDES it, so `git add` alone will not fix this." if irc == 0 else ""
        shown = ", ".join(missing[:3]) + (f" (+{len(missing) - 3} more)" if len(missing) > 3 else "")
        findings.append(("(governance artifact)", label, kind,
                         f"{len(missing)} of {len(rels)} not tracked: {shown}. {why}. "
                         f"`git clone` does not carry it.{excluded}"))
    return findings


def check_portability(manifest):
    """Advisory when a repository's decision history cannot survive a clone.

    The question is deliberately not "is Dolt bound". Section 54 forbids
    requiring a specific third-party tool and ADR-019 rule 2 says the backend is
    pluggable, not prescribed, so the checkable property is one every backend
    answers for itself: does the history reach another host? Adapters advertise
    it in `properties.portable`; the engine reads roles and never adapter names
    (engine/bindings.py).

    Fires only when EVERY bound backend says no. One portable backend alongside
    a fast local one is the arrangement ADR-019 rule 2 recommends, not a defect.
    """
    dh = (manifest.get("providers") or {}).get("decision_history")
    if not dh:
        return []  # unbound: INV-013, not this check's business
    import bindings as B  # noqa: PLC0415 -- bindings imports this module

    verdicts = []
    for b in (dh if isinstance(dh, list) else [dh]):
        if not isinstance(b, dict):
            continue
        d = B.describe(b, use_cache=False) or {}
        verdicts.append(((d.get("properties") or {}).get("portable"), b.get("adapter")))
    if not verdicts or any(v is True for v, _ in verdicts):
        return []
    # `None` is not `False`: a backend that never declared the property has not
    # said it is unportable, and reporting it as such would be inventing an
    # answer (ADR-003 rule 6).
    if all(v is None for v, _ in verdicts):
        return [("decision_history", "(all backends)", "PORTABILITY_UNDECLARED",
                 "no bound backend declares `properties.portable`, so whether this "
                 "repository's decision history survives a clone is unknown.")]
    return [("decision_history", "(all backends)", "HISTORY_NOT_PORTABLE",
             "every bound backend declares portable=false, so the decision history "
             "lives only on this machine and a clone starts empty. Bind a portable "
             "backend alongside -- the role is multi-valued (ADR-013) and ADR-019 "
             "rule 2 expects exactly that pairing.")]


def main(argv):
    if argv and argv[0] == "--validate":
        m, errs = load()
        if errs:
            print("MANIFEST INVALID — cannot validate adapters\n")
            for e in errs:
                print(f"  {e}")
            return 1
        findings = check_adapters(m)
        # ADR-006 line 18: the profile determines which providers are REQUIRED.
        # The mapping has been in policies/*.json since the profiles were
        # written and was read by nothing (issue 79). --validate is what a
        # person runs to find out whether they onboarded correctly, so an
        # unmet requirement belongs here rather than only in a status view.
        #
        # A role bound to an adapter reporting configured=false does NOT count
        # as met: structurally present and unable to answer is the state issue
        # 51 was about.
        import vocabulary as _V  # noqa: PLC0415 -- local, avoids an import cycle
        unusable = {role for role, _a, kind, _d in findings if kind == "NOT_CONFIGURED"}
        bound = {r for r in (m.get("providers") or {}) if not r.startswith("$")} - unusable
        for role in sorted(set(_V.required_roles(m["condition"]["profile"])) - bound):
            findings.append((role, "(unbound)", "REQUIRED_ROLE_UNBOUND",
                             f"{m['condition']['profile']} requires this role; it is not "
                             "usable here. Bind it, or reassess the condition -- the level "
                             "is a human's to change (ADR-006 rule 1)."))
        # Governance that lives on one machine is not governance. Checked here
        # because --validate is what a person runs to find out whether they
        # onboarded correctly, and "did it reach the repository" is half of that.
        findings += check_tracked(m)
        findings += check_portability(m)
        for role, adapter, kind, detail in findings:
            print(f"  [{kind}] {role} -> {adapter}: {detail}")

        # An unmet requirement is a CONFIGURATION GAP, not a broken binding, and
        # the two were indistinguishable here: `findings` mixed an adapter that
        # cannot run with a profile wanting a role nobody bound, and any entry
        # at all flipped the banner and the exit code.
        #
        # engine/status.py:292 already says the right thing about the same fact
        # -- "That is a configuration gap, not a verdict. Nothing here refuses
        # to answer because of it" -- and returns 0, and
        # conformance/status.py:238-248 asserts it. This surface disagreed with
        # that one about one repository (issue 180). ADR-031 names the field:
        # "An unmet `required_roles` is a configuration gap, not a verdict...
        # Report the fact, state the consequence, block nothing."
        #
        # ADVISORY IS A CLOSED SET AND EVERYTHING ELSE BLOCKS. A kind nobody
        # listed here is treated as blocking, because a missed advisory is noise
        # and a missed blocker is a false green.
        advisory = [f for f in findings if f[2] in ADVISORY_FINDINGS]
        blocking = [f for f in findings if f[2] not in ADVISORY_FINDINGS]
        n = len([r for r in (m.get("providers") or {}) if not r.startswith("$")])
        state = "PROVIDER_UNAVAILABLE" if blocking else "READY_FOR_GOVERNANCE"
        # Qualify the green. A reader who sees READY_FOR_GOVERNANCE and stops
        # reading has been told the repository is governed; if the manifest is
        # uncommitted, that is true only of the machine they are sitting at.
        if any(f[2] == "MANIFEST_UNTRACKED" for f in advisory) and not blocking:
            state = "READY_FOR_GOVERNANCE ON THIS HOST ONLY"
        counts = f"{n} bindings, {len(blocking)} finding(s)"
        if advisory:
            counts += f", {len(advisory)} advisory"
        print(f"\n{state} ({counts})")
        if advisory and not blocking:
            print("  The advisory finding(s) above are a configuration gap, not a verdict.")
            print("  Nothing refuses to answer because of them, and the assessed level is")
            print("  a human's to change. engine/status.py says which dispositions the")
            print("  gap puts out of reach.")
        return 0 if not blocking else 1

    if argv and argv[0] == "--explain":
        if len(argv) < 3:
            print("usage: --explain <role> <verb>", file=sys.stderr)
            return 2
        m, errs = load()
        ok, why = permitted(m, argv[1], argv[2])
        print(f"{'ALLOW' if ok else 'DENY '}  {argv[1]}.{argv[2]}  — {why}")
        return 0

    m, errs = load(argv[0] if argv else None)
    if errs:
        print(f"MANIFEST INVALID ({len(errs)} error(s)) — refusing to evaluate\n")
        for e in errs:
            print(f"  {e}")
        return 1
    p = m["providers"]
    print("MANIFEST VALID")
    print(f"  repository : {m['repository']['id']}")
    print(f"  condition  : {m['condition']['assessed']} / {m['condition']['profile']}")
    print(f"  bindings   : {len(p)} role(s)")
    for role in sorted(k for k in p if not k.startswith("$")):
        b = p[role]
        names = ", ".join(x["type"] for x in (b if isinstance(b, list) else [b]))
        print(f"    {role:<20} {names}")
    print("  permissions:")
    for role in sorted(k for k in m.get("permissions", {}) if not k.startswith("$")):
        granted = [v for v, on in m["permissions"][role].items() if on is True]
        print(f"    {role:<20} {', '.join(granted) or '(none granted)'}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
