#!/usr/bin/env python3
"""Repository onboarding — assess, detect, propose (spec §18–§20, ADR-006, ADR-010).

Three phases, deliberately separate:

    assess    repository condition L0–L4 -> suggested profile
    detect    candidate providers, each citing its evidence
    propose   write .repo-governor.proposed.json -- EVIDENCE for a human to read.
              NOT a bindable manifest; see tools/onboard-interactive.py.

The engine NEVER reads the proposal. Binding requires a human to promote it
to .repo-governor.json and commit (INV-013, ADR-010 rule 1). That separation
is why silent binding is unimplementable rather than merely forbidden.

Detection is filesystem-only. It does not authenticate or probe remote
systems — a probe that succeeds because a token happens to be present is
capability implying permission, which ADR-005 forbids (ADR-010 rule 4).

Usage:  python3 engine/onboard.py <repo-path> [--json] [--write]
"""

from __future__ import annotations

import json
import importlib.machinery
import re
import subprocess
import sys
from pathlib import Path

PROPOSAL = ".repo-governor.proposed.json"

# --- condition indicators (§23). Evidence, not a weighted score (ADR-006 rule 2).
# Certain indicators raise a FLOOR that may not be overridden downward (rule 3).
FLOOR_INDICATORS = ("public_api_surface", "release_branches", "generated_consumers")


def _git(repo, *args):
    try:
        p = subprocess.run(["git", "-C", str(repo), *args],
                           capture_output=True, text=True, timeout=30)
        return p.returncode, p.stdout
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return 127, ""


# Where ADR collections live, in one place because TWO places disagreed.
#
# Detection scanned five conventions and the difficulty indicator scanned three,
# so a repository using `adrs/` or `doc/adr` was told it had "little architecture
# history" by the same run that detected its architecture provider -- and the
# indicator is not cosmetic: it raises the assessed level from L1 to L2, which
# selects the profile, which decides which roles manifest.py REQUIRES. Renaming a
# directory changed which roles a repository had to bind (issue 161).
#
# `doc/adr` is adr-tools' default and `adrs/` is common at the repository root;
# missing them reported real collections as no provider at all (#27).
#
# SPEC WORKSPACES ARE DELIBERATELY ABSENT, and this is a decision with a reason
# rather than an omission. `openspec/` and `.specify/` are both readable now --
# adapters/openspec and adapters/speckit answer for them -- so the original
# justification, that no adapter could read the evidence, has expired. The
# exclusion was re-argued on the evidence instead of being quietly flipped
# (issue 165).
#
# ADRs are decisions TAKEN: durable, immutable, superseded rather than edited,
# existing only once something has been decided. Specs are intent DECLARED,
# and a scaffold emits them before anything exists at all. Those are not the
# same fact about a repository's history.
#
# Measured with the corrected floor from issue 164, counting a spec workspace
# would move 19 of 105 OpenSpec repositories (18%) and 13 of 40 Spec Kit
# repositories (32%). The transition that settles it is GREENFIELD -> STANDARD,
# 12 repositories across both tools:
#
#     source files, median            0
#     spec/change directories, median 1   (max 3)
#     planned but unbuilt             2 of 12
#
# That population is empty repositories that ran a scaffold once. Promoting them
# out of GREENFIELD would require each to bind a roadmap_authority, which is
# §54's "blocks routine reversible implementation excessively" aimed at the
# emptiest repositories in the sample.
#
# Two things would change this answer, and both are measurable rather than
# arguable: a spec COUNT threshold behaving differently from mere presence --
# 2 of 12 is too thin to design on -- and issue 150's demand evidence.
#
# `conformance/onboarding.py` asserts the exclusion for BOTH directories, so it
# cannot lapse for one of them by accident.
ADR_DIRS = ("docs/adr", "docs/adrs", "docs/decisions", "doc/adr", "adrs")

# A published identity per ecosystem. Structure, never prose about structure.
PACKAGE_MANIFESTS = ("package.json", "pyproject.toml", "go.mod", "Cargo.toml")


def _declares_package(path: Path):
    """Does this manifest declare a package the world can depend on?

    Not "does a manifest exist" -- a private workspace root has a package.json
    and ships nothing, which is why `"private": true` is honoured rather than
    ignored.
    """
    try:
        text = path.read_text(errors="ignore")
    except OSError:
        return False
    if path.name == "package.json":
        try:
            j = json.loads(text)
        except json.JSONDecodeError:
            return False
        if not j.get("name") or j.get("private"):
            return False
        # A name and a version alone is `npm init` output, which every
        # application has. Something that means to be DEPENDED ON says where it
        # starts or what it ships. On 105 real repositories this distinction
        # costs 3 of 49 -- it is close to free in coverage and it stops an
        # ordinary application being told it carries compatibility obligations.
        return any(k in j for k in
                   ("main", "exports", "bin", "files", "publishConfig", "module", "types"))
    if path.name == "pyproject.toml":
        return bool(re.search(r"^\[project\]", text, re.M) and re.search(r"^name\s*=", text, re.M))
    if path.name == "go.mod":
        return bool(re.search(r"^module\s+\S+", text, re.M))
    if path.name == "Cargo.toml":
        return bool(re.search(r"^\[package\]", text, re.M) and re.search(r"^name\s*=", text, re.M))
    return False


def _ships_something(repo: Path):
    """Compatibility obligations: a published package identity, or release tags.

    Kept to the repository root and one level down. Walking the whole tree would
    find every vendored dependency's manifest and floor everything.
    """
    for depth in ("", "*/"):
        for name in PACKAGE_MANIFESTS:
            for p in repo.glob(f"{depth}{name}"):
                if _declares_package(p):
                    return True
    rc, out = _git(repo, "tag")
    return rc == 0 and bool(out.strip())


def assess(repo: Path):
    """Report observed indicators and a SUGGESTED level. A human decides."""
    ind = {}
    files = [p for p in repo.rglob("*") if p.is_file() and ".git/" not in str(p)]
    src = [p for p in files if p.suffix in (".py", ".js", ".ts", ".go", ".rs", ".java", ".rb")]
    ind["source_files"] = len(src)

    pkgs = [p for p in repo.rglob("package.json") if "node_modules" not in str(p)]
    pkgs += list(repo.rglob("pyproject.toml")) + list(repo.rglob("go.mod"))
    ind["packages"] = len(pkgs)
    ind["multi_package"] = len(pkgs) > 1

    ind["migrations"] = any(p.is_dir() and p.name in ("migrations", "migrate")
                            for p in repo.rglob("*"))
    ind["feature_flags"] = any("flag" in p.name.lower() for p in files)
    # Reads ADR_DIRS, the same list detection reads. Restating the conventions
    # here is what made the two disagree (issue 161).
    ind["architecture_history"] = any((repo / d).is_dir() for d in ADR_DIRS)
    ind["ci_workflows"] = len(list((repo / ".github" / "workflows").glob("*"))) if (
        repo / ".github" / "workflows").is_dir() else 0

    # Obligation presence. A repository with no licence is all-rights-reserved
    # by default: no consumer has a grant to use it and no contributor has
    # terms to contribute under. That is a fact about what the repository OWES
    # people, which is the category the floor indicators below exist for.
    #
    # Deliberately NOT a floor, and deliberately not a gate. A floor may not be
    # lowered, so a missing licence would permanently force L4 on a repository
    # for a reason unrelated to how deeply its code needs governing; and
    # refusing a verdict over it would be section 54's "blocks routine
    # reversible implementation excessively". Absence is also not a defect --
    # an internal repository may carry no licence on purpose. So: report it,
    # and let a person weigh it.
    ind["license_present"] = any((repo / n).is_file() for n in
                                 ("LICENSE", "LICENSE.md", "LICENSE.txt",
                                  "LICENCE", "LICENCE.md", "COPYING"))
    # Rides along on the same mechanism. It carries LESS weight than the
    # licence: a missing README is a documentation gap, not an obligation to
    # anyone. It is here because it costs one line, not because it is
    # equivalent -- and nothing should be inferred from it beyond presence
    # (section 8 excludes a static-analysis engine by name).
    ind["readme_present"] = any((repo / n).is_file() for n in
                                ("README.md", "README", "README.rst", "README.txt"))

    # Floor indicators.
    #
    # `public_api_surface` asks whether this repository carries COMPATIBILITY
    # OBLIGATIONS -- whether something outside it depends on what it exposes.
    # It used to ask whether the word `export ` appeared in the first 4000 bytes
    # of any of the first 60 source files, which is ordinary module syntax in
    # TypeScript and says nothing about obligations.
    #
    # Measured on 105 public repositories (issue 164):
    #
    #     `export ` grep            73/105   70%   -- 100% of TypeScript repos
    #     published package         41/105   39%
    #     git tags present          22/105   21%
    #     package OR tags           52/105   50%
    #
    # 37 of 105 were floored with NO published package and NO tags -- nothing
    # saying they ship anything to anyone -- and each was told to bind the five
    # roles GOVERNOR_HIGH_ASSURANCE requires. §54 names "blocks routine
    # reversible implementation excessively" as a failure condition. The test
    # was also wrong the other way: 12 repositories publish a package with no
    # `export ` in the scanned slice, so it was close to UNCORRELATED with what
    # it claimed to detect rather than merely over-eager.
    #
    # Both signals are kept because they are near-independent (both=11,
    # package-only=30, tags-only=11): a project can publish to a registry
    # without tagging, and tag releases without a manifest.
    #
    # This is the fifth time a substring test has stood in for a structural fact
    # here. `conformance/imports.py` refuses to grep source for imports and says
    # why in its docstring -- a string in a comment is not an import.
    ind["public_api_surface"] = _ships_something(repo)
    rc, out = _git(repo, "branch", "-r")
    ind["release_branches"] = sum(
        1 for b in out.splitlines() if any(k in b for k in ("release/", "v1.", "v2.", "stable"))) > 0
    ind["generated_consumers"] = any(
        p.is_dir() and p.name in ("generated", "gen", "clients") for p in repo.rglob("*"))

    floors = [k for k in FLOOR_INDICATORS if ind.get(k)]

    if floors:
        level, why = "L4", f"floor raised by {', '.join(floors)} — compatibility obligations exist"
    elif ind["multi_package"] or (ind["migrations"] and ind["architecture_history"]):
        level, why = "L3", "multiple packages or migrations plus architecture history"
    elif ind["migrations"] or ind["architecture_history"] or ind["ci_workflows"] > 1:
        level, why = "L2", "migrations, architecture history, or multiple workflows present"
    elif ind["source_files"] > 2:
        level, why = "L1", "one small package, little architecture history"
    else:
        level, why = "L0", "nearly empty repository, no architecture history"

    profile = {"L0": "GOVERNOR_GREENFIELD", "L1": "GOVERNOR_LITE", "L2": "GOVERNOR_STANDARD",
               "L3": "GOVERNOR_FULL", "L4": "GOVERNOR_HIGH_ASSURANCE"}[level]
    return {"suggested": level, "profile": profile, "reason": why,
            "floor": floors or None, "indicators": ind}


# --- detection (§19). Filesystem evidence only; never credentials. -----------
def _adr_status(path):
    """Read one decision's status with the ADAPTER's own parser.

    Importing it rather than reimplementing is the point: a detector with its own
    notion of "has a status" is a second parser that will disagree with the first
    (#27). If the adapter learns a dialect, detection learns it in the same commit.
    """
    import importlib.util
    global _ADR_MOD
    if "_ADR_MOD" not in globals() or _ADR_MOD is None:
        spec = importlib.util.spec_from_loader(
            "_adr_adapter",
            importlib.machinery.SourceFileLoader(
                "_adr_adapter",
                str(Path(__file__).resolve().parent.parent / "adapters" / "adr")))
        _ADR_MOD = importlib.util.module_from_spec(spec)
        try:
            spec.loader.exec_module(_ADR_MOD)
        except Exception:  # noqa: BLE001 -- detection must degrade, never crash
            _ADR_MOD = False
    if _ADR_MOD is False:
        return None
    try:
        # _status_of, not a named regex: the adapter's regexes are private detail
        # and renaming one silently broke this import once already. The function
        # is the contract.
        status, _why = _ADR_MOD._status_of(path.read_text(errors="ignore"))
        return status
    except (OSError, AttributeError):
        return None


def _cite(repo, rel, note):
    return f"{rel}: {note}"


def detect(repo: Path):
    """Return candidates per role, each with cited evidence and a disposition."""
    cands = {}

    def add(role, type_, adapter, disposition, evidence, not_evidence=None, **extra):
        """One candidate. `adapter=None` means: found, and nothing here reads it.

        DETECTION MAY NAME AN ADAPTER ONLY IF THAT ADAPTER CAN READ THE EVIDENCE
        THAT TRIGGERED THE DETECTION (#27, #201). Two candidates broke it while
        stating it in a comment three lines above: a `.beads/` SQLite store was
        offered `adapters/execution-file`, which reads a JSON file, and a
        Renovate config was offered `adapters/change-signals-file`, which reads
        a local signals file. Both were marked UNCONFIRMED with an explicit
        not_evidence line, so nothing was concealed -- but the mitigation is a
        sentence and the field is a machine-readable claim, and a reader
        scanning the candidate row gets the wrong answer from the part that
        looks authoritative. An external reporter did exactly that.

        A null adapter keeps the detection, which is useful -- the role IS
        present -- and drops the pointer that cannot be honoured. It must be
        accompanied by a not_evidence line saying so; conformance asserts it.
        """
        cands.setdefault(role, []).append({
            "role": role, "type": type_, "adapter": adapter,
            "disposition": disposition, "evidence": evidence,
            "not_evidence": not_evidence or [], **extra})

    # repository — Git
    rc, _ = _git(repo, "rev-parse", "--git-dir")
    if rc == 0:
        add("repository", "git", "adapters/git", "PROVIDER_DETECTED",
            [_cite(repo, ".git", "valid git repository")])

    # architecture — ADR directory. Conventions come from ADR_DIRS, which
    # `assess()` also reads, so detecting a provider and assessing the level
    # cannot disagree about what an ADR collection looks like (issue 161).
    for d in ADR_DIRS:
        p = repo / d
        if p.is_dir():
            md = sorted(p.glob("*.md"))
            # Accept `NNNN-title.md` and `adr-NNN-title.md`; both are common.
            import re as _re
            _num = _re.compile(r"^(?:adr[-_])?\d{3,4}[-_.]", _re.I)
            numbered = [f for f in md if _num.match(f.name)]
            # Count with the ADAPTER's parser, not a substring test. A substring
            # test said 20 of 22 files "declare a Status" on a real repository
            # where the adapter could read 2 -- detection promising a provider
            # the adapter cannot deliver (#27). ADR-010 stops detection assigning
            # AUTHORITY; it did not stop it overstating CAPABILITY.
            readable = [f for f in numbered if _adr_status(f) is not None]
            strong = len(numbered) >= 2 and len(readable) >= max(1, len(numbered) // 2)
            add("architecture", "adr", "adapters/adr",
                "PROVIDER_DETECTED" if strong else "PROVIDER_UNCONFIRMED",
                [_cite(repo, d, f"{len(numbered)} numbered file(s)"),
                 _cite(repo, d, f"{len(readable)} with a Status this adapter can read")],
                [] if strong else [_cite(repo, d, f"only {len(readable)} of {len(numbered)} "
                                                  "have a readable Status")],
                path=d)

    # architecture — OpenSpec. A SECOND provider for the same role, which
    # ADR-013 allows ("zero or many — constraints accumulate; ADRs and specs
    # coexist legitimately") and the union built for issue 154 makes real.
    #
    # Keys on the directory plus one half of the layout, NEVER on project.md.
    # A census on 2026-08-24 first reported project.md at 97.8% and that was a
    # selection artifact -- the sample had been found by searching for it. An
    # independent selector puts it at 20.7%, so keying detection there would
    # miss four repositories in five (issue 155).
    osp = repo / "openspec"
    if osp.is_dir():
        halves = [d for d in ("changes", "specs") if (osp / d).is_dir()]
        if halves:
            changes = osp / "changes"
            n = len([q for q in changes.iterdir() if q.is_dir() and q.name != "archive"]) \
                if changes.is_dir() else 0
            arch = (changes / "archive").is_dir()
            add("architecture", "openspec", "adapters/openspec",
                "PROVIDER_DETECTED",
                [_cite(repo, "openspec", f"holds {', '.join(halves)}"),
                 _cite(repo, "openspec/changes", f"{n} active change(s)")]
                + ([_cite(repo, "openspec/changes/archive", "completed changes present")]
                   if arch else []),
                path="openspec")

    # architecture — GitHub Spec Kit. Keys on `.specify/` and NEVER on `specs/`.
    #
    # A census on 2026-08-24 ran two selectors: `.specify/memory/constitution.md`
    # returned 399 repositories, `specs/**/plan.md` returned 230, and they
    # overlapped by 6. Testing the second for contamination found 35 of 70 --
    # 50% -- carry no `.specify/` at all, because `specs/` is an ordinary
    # directory name. Proposing an architecture provider on that would
    # manufacture architectural authority from a folder name (§37), which is
    # worse than not detecting at all (issue 156).
    spk = repo / ".specify"
    if spk.is_dir():
        con = spk / "memory" / "constitution.md"
        specs = repo / "specs"
        feats = len([q for q in specs.iterdir() if q.is_dir()]) if specs.is_dir() else 0
        ev = [_cite(repo, ".specify", "Spec Kit workspace")]
        if con.is_file():
            ev.append(_cite(repo, ".specify/memory/constitution.md", "constitution present"))
        if feats:
            ev.append(_cite(repo, "specs", f"{feats} feature specification(s)"))
        add("architecture", "speckit", "adapters/speckit",
            "PROVIDER_DETECTED" if con.is_file() else "PROVIDER_UNCONFIRMED",
            ev,
            [] if con.is_file() else
            [_cite(repo, ".specify/memory", "no constitution.md; constraints unreadable")],
            path=".specify")

    # change_signals — Renovate / Dependabot
    for f, t in (("renovate.json", "renovate"), (".renovaterc.json", "renovate"),
                 (".github/dependabot.yml", "dependabot")):
        if (repo / f).exists():
            # UNCONFIRMED, not DETECTED. The config proves the SERVICE is
            # configured; it does not prove any bound adapter can read it.
            # `change-signals-file` reads a local signals file, not a Renovate or
            # Dependabot config -- proposing it here is detection naming an
            # adapter that cannot serve the provider it detected (#27).
            add("change_signals", t, None, "PROVIDER_UNCONFIRMED",
                [_cite(repo, f, "configuration present")],
                [_cite(repo, f, f"no adapter here reads {t}; change-signals-file reads a local "
                                "signals file, so this binding needs a real adapter or a human "
                                "exporting signals into that file")])

    # execution — Beads. TWO candidates, because two different facts.
    #
    # The rule above is why they are separate. `adapters/beads` reads the JSONL
    # export, not the store; naming it for a bare `.beads/` directory would be
    # the same over-promise this rule was written against, just with a
    # different adapter in the field.
    if (repo / ".beads" / "issues.jsonl").exists():
        add("execution", "beads", "adapters/beads", "PROVIDER_DETECTED",
            [_cite(repo, ".beads/issues.jsonl", "beads JSONL export present; adapters/beads "
                                                "reads exactly this file")])
    elif (repo / ".beads").is_dir():
        # The store is there and nothing here can open it. An embedded Dolt
        # database is also not revision-scoped (ADR-033) and is routinely
        # gitignored, so binding it would answer about the machine rather than
        # the checked-out revision even if an adapter could read it.
        add("execution", "beads", None, "PROVIDER_UNCONFIRMED",
            [_cite(repo, ".beads/", "beads database directory present")],
            [_cite(repo, ".beads/", "no adapter here reads a beads database. `adapters/beads` "
                                    "reads .beads/issues.jsonl, which is absent: run "
                                    "`bd export --all > .beads/issues.jsonl` and track it, or "
                                    "leave this role unbound")])

    # roadmap_authority — Linear
    for f in (".linear.json", ".linear.yml", "linear.json"):
        if (repo / f).exists():
            add("roadmap_authority", "linear", "adapters/linear", "PROVIDER_DETECTED",
                [_cite(repo, f, "Linear configuration present")])

    # roadmap_authority — GitHub Projects.
    # Cannot be confirmed without an authenticated call, and ADR-010 rule 4
    # forbids probing with credentials during detection. So: UNCONFIRMED.
    rc, remotes = _git(repo, "remote", "-v")
    has_gh = "github.com" in remotes
    marker = (repo / ".github" / "PROJECTS.md").exists() or (repo / ".github").is_dir()
    if has_gh or marker:
        ev = []
        if has_gh:
            ev.append(_cite(repo, "git remote", "origin points at github.com"))
        if marker:
            ev.append(_cite(repo, ".github/", "GitHub metadata directory present"))
        add("roadmap_authority", "github-projects", "adapters/github-projects",
            "PROVIDER_UNCONFIRMED", ev,
            [_cite(repo, "-", "whether a Project exists cannot be seen without an "
                              "authenticated call, which detection must not make")])

    return cands


def onboard(repo: Path):
    condition = assess(repo)
    candidates = detect(repo)

    conflicts, halted = [], False
    for role in ("roadmap_authority", "execution", "repository", "acceptance_criteria"):
        if len(candidates.get(role, [])) > 1:
            conflicts.append({
                "role": role,
                "candidates": [c["type"] for c in candidates[role]],
                "disposition": "PROVIDER_CONFLICT",
                "required": f"Select one canonical {role} provider. No ranking is applied — "
                            "any automatic tie-break would silently confer authority (INV-013).",
            })
            if role == "roadmap_authority":
                halted = True

    roles_found = sorted(candidates)
    if halted:
        state = "PROVIDER_CONFLICT"
    elif "roadmap_authority" not in candidates:
        state = "AUTHORITY_SOURCE_MISSING"
    else:
        state = "PROPOSAL_READY"

    return {
        "repository": str(repo),
        "state": state,
        "condition": condition,
        "candidates": candidates,
        "conflicts": conflicts,
        "roles_detected": roles_found,
        "execution_required": False,
        "notes": [
            "Detection proposes; only an accepted manifest binds (INV-013).",
            "No credentials were used. Remote systems were not probed.",
        ] + ([ "No roadmap authority detected. A manual/file provider or explicit "
               "configuration is required before any EXECUTE disposition." ]
             if "roadmap_authority" not in candidates else []),
    }


def main(argv):
    if not argv:
        print("usage: onboard.py <repo-path> [--json] [--write]", file=sys.stderr)
        return 2
    repo = Path(argv[0]).resolve()
    if not repo.is_dir():
        print(f"FATAL: {repo} is not a directory", file=sys.stderr)
        return 2
    result = onboard(repo)

    if "--write" in argv:
        out = repo / PROPOSAL
        out.write_text(json.dumps(
            {"$comment": "EVIDENCE, not a manifest. The engine never reads this file, and "
                         "RENAMING IT DOES NOT BIND -- it carries no repo_governor.version, no "
                         "providers block and no permissions, so manifest.py rejects it with "
                         "UNSUPPORTED_VERSION. That rename was the documented instruction until "
                         "2026-08-19 and had never been run end to end. For a manifest-shaped "
                         "proposal, run tools/onboard-interactive.py: it asks the two things "
                         "detection cannot see -- which provider is the roadmap authority, and "
                         "what admission means there (ADR-018). Binding stays a human act "
                         "(ADR-010 rule 1).", **result}, indent=2) + "\n")
        print(f"wrote {out}")

    if "--json" in argv:
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0

    c = result["condition"]
    print(f"repository : {result['repository']}")
    print(f"STATE      : {result['state']}")
    print(f"condition  : {c['suggested']} / {c['profile']}  ({c['reason']})")
    if c["floor"]:
        print(f"  floor    : {', '.join(c['floor'])} — may not be overridden downward")
    print("candidates :")
    for role in sorted(result["candidates"]):
        for cand in result["candidates"][role]:
            print(f"  {role:<20} {cand['type']:<16} {cand['disposition']}")
            for e in cand["evidence"]:
                print(f"      evidence: {e}")
            for e in cand["not_evidence"]:
                print(f"      not evidence: {e}")
    if not result["candidates"]:
        print("  (none)")
    for cf in result["conflicts"]:
        print(f"\nPROVIDER_CONFLICT on {cf['role']}: {', '.join(cf['candidates'])}")
        print(f"  {cf['required']}")
    for n in result["notes"]:
        print(f"note: {n}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
