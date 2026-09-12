#!/usr/bin/env python3
"""Onboarding simulation RG-SIM-ONBOARDING-v0.1 — gate 1 (#7).

Materializes fixtures A–C from `fixtures/onboarding.json` into isolated
temp repositories, runs `engine/onboard.py` against each, and asserts the
expectations in §58–§60.

Also validates gates 2, 3 and 4, which have no separate build:

    gate 2  fixture C emits PROVIDER_CONFLICT and halts (#8)
    gate 3  fixture A needs no providers beyond Git (#9)
    gate 4  detection proposes; the engine never reads the proposal (#10)

Usage:  python3 conformance/onboarding.py
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "engine"))
import onboard as O  # noqa: E402

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("onboarding")

SPEC = json.loads((ROOT / "conformance" / "fixtures" / "onboarding.json").read_text())


def materialize(name, fx, into: Path):
    repo = into / name
    repo.mkdir(parents=True)
    for rel, content in fx["files"].items():
        p = repo / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)
    subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
    if fx.get("remote"):
        subprocess.run(["git", "-C", str(repo), "remote", "add", "origin", fx["remote"]], check=True)
    return repo


class R:
    def __init__(self):
        self.rows = []

    def add(self, fx, check, ok, detail=""):
            self.rows.append((fx, check, ok, detail))

    def fails(self):
        return [r for r in self.rows if not r[2]]


def run_fixture(name, fx, tmp, rep):
    repo = materialize(name, fx, tmp)
    res = O.onboard(repo)
    exp = fx["expect"]
    got_roles = set(res["roles_detected"])

    if "condition" in exp:
        rep.add(name, f"condition == {exp['condition']}",
                res["condition"]["suggested"] == exp["condition"],
                f"got {res['condition']['suggested']} ({res['condition']['reason']})")
    if "profile" in exp:
        rep.add(name, f"profile == {exp['profile']}",
                res["condition"]["profile"] == exp["profile"],
                f"got {res['condition']['profile']}")

    rep.add(name, f"state == {exp['state']}", res["state"] == exp["state"],
            f"got {res['state']}")

    if "roles_detected" in exp:
        want = set(exp["roles_detected"])
        rep.add(name, f"roles == {sorted(want)}", got_roles == want,
                f"got {sorted(got_roles)}")

    for role in exp.get("must_not_detect", []):
        rep.add(name, f"does NOT detect {role}", role not in got_roles,
                f"unexpectedly detected {role}")

    for role, disp in exp.get("dispositions", {}).items():
        actual = [c["disposition"] for c in res["candidates"].get(role, [])]
        rep.add(name, f"{role} disposition {disp}", disp in actual, f"got {actual}")

    if exp.get("conflict_role"):
        cf = [c for c in res["conflicts"] if c["role"] == exp["conflict_role"]]
        rep.add(name, f"PROVIDER_CONFLICT on {exp['conflict_role']}", bool(cf),
                "no conflict raised")
        if cf:
            rep.add(name, "conflict names both candidates",
                    set(cf[0]["candidates"]) == set(exp["conflict_candidates"]),
                    f"got {cf[0]['candidates']}")
            rep.add(name, "no ranking applied (gate 2)",
                    "No ranking" in cf[0]["required"], "")

    # Every candidate must cite evidence (ADR-010 rule 3).
    uncited = [c["type"] for cs in res["candidates"].values() for c in cs if not c["evidence"]]
    rep.add(name, "every candidate cites evidence", not uncited, f"uncited: {uncited}")

    # Detection must never authenticate (ADR-010 rule 4).
    rep.add(name, "detection states it used no credentials",
            any("No credentials" in n for n in res["notes"]), "")

    # ISSUE 201. Detection may name an adapter only if that adapter can read the
    # evidence that triggered the detection. Two candidates broke it while
    # stating it in a comment three lines above -- a `.beads/` SQLite store
    # offered `adapters/execution-file`, a Renovate config offered
    # `adapters/change-signals-file`. Both carried an honest not_evidence line,
    # and a reader scanning the candidate row still got the wrong answer,
    # because the row is the part that looks authoritative.
    _named = [c for cs in res["candidates"].values() for c in cs if c.get("adapter")]
    _unnamed = [c for cs in res["candidates"].values() for c in cs if not c.get("adapter")]
    _ghost = [c["adapter"] for c in _named if not (ROOT / c["adapter"]).is_file()]
    rep.add(name, "every named adapter exists on disk", not _ghost, f"missing: {_ghost}")
    # A null adapter must never be silent. "Found, and nothing here reads it" is
    # useful; "found" with no adapter and no reason is a candidate a human
    # cannot act on either way.
    _mute = [c["type"] for c in _unnamed if not c.get("not_evidence")]
    rep.add(name, "a candidate naming no adapter says why", not _mute,
            f"silent: {_mute} -- naming no adapter is honest only when the reason is stated")

    return repo, res



def _parse_form(path):
    """(ok, detail). Needs a YAML parser; names the missing one rather than skipping.

    Same rule as conformance/_preflight.py: an absent dependency makes a verdict
    red WITH ITS CAUSE, never green by omission. A malformed issue form is
    exactly the defect worth catching -- GitHub does not reject it, it quietly
    serves a blank issue box instead, so the template looks installed while
    asking nothing.
    """
    import subprocess as sp
    try:
        import yaml
        d = yaml.safe_load(path.read_text())
    except ImportError:
        rb = sp.run(["ruby", "-ryaml", "-rjson", "-e",
                     f"print YAML.load_file({str(path)!r}).to_json"],
                    capture_output=True, text=True)
        if rb.returncode != 0:
            return False, ("no YAML parser available -- install pyyaml or ruby. "
                           "Skipping would report a malformed form as conformant.")
        d = json.loads(rb.stdout)
    except Exception as e:
        return False, f"YAML does not parse: {e}"

    for k in ("name", "description", "body"):
        if k not in d:
            return False, f"missing required top-level key: {k}"
    for i, b in enumerate(d["body"]):
        t = b.get("type")
        if t not in ("markdown", "input", "textarea", "dropdown", "checkboxes"):
            return False, f"field {i}: unknown type {t!r}"
        if t == "markdown":
            continue
        if not b.get("id"):
            return False, f"field {i} ({t}): no id"
        if not (b.get("attributes") or {}).get("label"):
            return False, f"field {i} ({t}): no attributes.label"
        if t == "dropdown" and not b["attributes"].get("options"):
            return False, f"dropdown {b['id']}: no options"
    return True, ""


def _src():
    return (ROOT / "engine" / "onboard.py").read_text()


def _detected_convs():
    """The conventions the DETECTION loop iterates, read from source.

    Asserting `for d in ADR_DIRS` textually is the only way to show the loop
    reads the shared list rather than a literal that happens to match it today.
    """
    import re
    m = re.search(r"# architecture — ADR directory[\s\S]{0,400}?\n    for d in ([^:]+):", _src())
    return list(O.ADR_DIRS) if m and m.group(1).strip() == "ADR_DIRS" else []


def _indicator_convs():
    """Same, for the architecture_history indicator."""
    import re
    m = re.search(r'ind\["architecture_history"\] = any\(\(repo / d\)\.is_dir\(\) for d in ([^)]+)\)', _src())
    return list(O.ADR_DIRS) if m and m.group(1).strip() == "ADR_DIRS" else []


def main():
    rep = R()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        repos = {}
        for name, fx in SPEC["fixtures"].items():
            repos[name], _ = run_fixture(name, fx, tmp, rep)

        # --- issue 161: detection and the difficulty indicator must agree ------
        #
        # engine/onboard.py decided two things about ADR evidence from two
        # different lists -- detection scanned five conventions, the
        # `architecture_history` indicator scanned three. So a repository using
        # `adrs/` or `doc/adr` was told it had "little architecture history" by
        # the same run that detected its architecture provider, and the
        # indicator raises the level from L1 to L2, which selects the profile,
        # which decides which roles manifest.py REQUIRES.
        #
        # DERIVED FROM O.ADR_DIRS, never restated. A check that hard-codes the
        # conventions is the third place to forget the next time one is added,
        # which is the defect it exists to catch, one layer up.
        levels, contradictions = {}, []
        for conv in O.ADR_DIRS:
            r = tmp / ("adrconv-" + conv.replace("/", "_"))
            (r / conv).mkdir(parents=True)
            # TWO decisions, so the provider reads PROVIDER_DETECTED rather than
            # PROVIDER_UNCONFIRMED. The contradiction is sharpest when detection
            # is confident and the indicator still says there is no history.
            for n, t in ((1, "First"), (2, "Second")):
                (r / conv / f"000{n}-x.md").write_text(
                    f"# {n}. {t} decision\n\n**Status**: Accepted\n")
            (r / "src").mkdir()
            for i in range(3):
                (r / "src" / f"m{i}.py").write_text("x=1\n")
            subprocess.run(["git", "-C", str(r), "init", "-q"], check=True)
            a = O.assess(r)
            levels[conv] = a["suggested"]
            found = "architecture" in O.detect(r)
            if found and not a["indicators"]["architecture_history"]:
                contradictions.append(conv)

        rep.add("issue-161", "every detected ADR convention raises the assessed level identically",
                len(set(levels.values())) == 1,
                f"{levels} — identical repositories differing only in directory name; "
                "the level selects the profile, and the profile decides which roles "
                "manifest.py requires")
        rep.add("issue-161", "no repository is told it has little architecture history "
                "while its architecture provider is detected",
                not contradictions,
                f"self-contradicting for {contradictions}")
        rep.add("issue-161", "detection and the difficulty indicator read the same list",
                # Both must be NON-EMPTY. `all()` over an empty list is True, so
                # the first draft of this check went green precisely when a site
                # stopped reading ADR_DIRS -- vacuous exactly where it mattered.
                bool(_detected_convs()) and bool(_indicator_convs())
                and set(_detected_convs()) == set(O.ADR_DIRS)
                and set(_indicator_convs()) == set(O.ADR_DIRS),
                "one of the two sites restates the conventions instead of reading ADR_DIRS; "
                "that is how they diverged in the first place")

        # The fix must not WIDEN. openspec/ is a separate judgement -- an OpenSpec
        # repository holds decisions-in-progress, not a decision ledger -- and
        # raising a repository's required roles for evidence no adapter can read
        # is the wrong order. It follows issue 155.
        r = tmp / "openspec-only"
        (r / "openspec" / "changes" / "add-thing").mkdir(parents=True)
        (r / "openspec" / "changes" / "add-thing" / "proposal.md").write_text("# p\n")
        (r / "src").mkdir()
        for i in range(3):
            (r / "src" / f"m{i}.py").write_text("x=1\n")
        subprocess.run(["git", "-C", str(r), "init", "-q"], check=True)
        rep.add("issue-161", "openspec/ alone does not raise the assessed level",
                O.assess(r)["suggested"] == "L1",
                "a spec workspace is intent declared, not decisions taken (issue 165)")

        # --- issue 165: the same refusal, for the OTHER spec provider ---------
        #
        # The exclusion held for openspec/ by test and for .specify/ by
        # accident. Both adapters exist now (PR 163, PR 168), so the original
        # justification -- that no adapter could read the evidence -- has
        # expired for both, and the decision was re-argued rather than flipped:
        # counting a spec workspace would move 18% of OpenSpec and 32% of Spec
        # Kit repositories, and the GREENFIELD -> STANDARD population is empty
        # repositories that ran a scaffold once (median 0 source files).
        sk = tmp / "speckit-only"
        (sk / ".specify" / "memory").mkdir(parents=True)
        (sk / ".specify" / "memory" / "constitution.md").write_text(
            "# C\n\n## Article I: Real\ntext\n")
        (sk / "specs" / "001-x").mkdir(parents=True)
        (sk / "specs" / "001-x" / "spec.md").write_text("# s\n")
        (sk / "src").mkdir()
        for i in range(3):
            (sk / "src" / f"m{i}.py").write_text("x=1\n")
        subprocess.run(["git", "-C", str(sk), "init", "-q"], check=True)
        rep.add("issue-165", "a Spec Kit workspace alone does not raise the assessed level",
                O.assess(sk)["suggested"] == "L1",
                "specs are intent declared; requiring a roadmap_authority for them aims "
                "§54 at the emptiest repositories in the sample")

        # THE REFUSAL MUST NOT OVER-REACH. Excluding spec workspaces must not
        # suppress a real ADR directory sitting beside one -- 6% of OpenSpec and
        # 5% of Spec Kit repositories carry both.
        both = tmp / "specs-and-adrs"
        (both / ".specify" / "memory").mkdir(parents=True)
        (both / ".specify" / "memory" / "constitution.md").write_text("# C\n\n## A\nx\n")
        (both / "openspec" / "changes" / "add-x").mkdir(parents=True)
        (both / "openspec" / "changes" / "add-x" / "proposal.md").write_text("# p\n")
        (both / "docs" / "adrs").mkdir(parents=True)
        for n in (1, 2):
            (both / "docs" / "adrs" / f"000{n}-d.md").write_text(
                f"# {n}. D\n\n**Status**: Accepted\n")
        (both / "src").mkdir()
        for i in range(3):
            (both / "src" / f"m{i}.py").write_text("x=1\n")
        subprocess.run(["git", "-C", str(both), "init", "-q"], check=True)
        a_both = O.assess(both)
        rep.add("issue-165", "a spec workspace beside real ADRs still raises the level",
                a_both["suggested"] == "L2" and a_both["indicators"]["architecture_history"],
                f"assessed {a_both['suggested']} — excluding spec workspaces must not "
                "suppress a genuine decision ledger that sits next to one")

        # --- issue 164: the API-surface floor reads structure, not substrings ---
        #
        # `public_api_surface` is a FLOOR indicator: ADR-006 rule 3 says it may
        # not be overridden downward, and it selects GOVERNOR_HIGH_ASSURANCE,
        # which engine/vocabulary.py requires five bound roles for. It used to
        # fire on the word `export ` appearing in a source file -- ordinary
        # module syntax in TypeScript, and silent about compatibility
        # obligations.
        #
        # Measured on 105 public repositories: it floored 100% of TypeScript
        # repositories, and 37 of 105 had no published package and no tags at
        # all. Replacing it changed the verdict for 52 of 105 -- half the
        # sample -- which is what "near-uncorrelated" means in practice.
        #
        # Every assertion below BUILDS A REPOSITORY and asks the engine. None
        # greps engine/onboard.py: a check that reads source text for the string
        # it disapproves of fails on the comment explaining the disapproval,
        # which happened in conformance/layer1.py during issue 155.
        def _mkrepo(name, files, tag=None):
            r = tmp / f"floor-{name}"
            for rel, content in files.items():
                q = r / rel
                q.parent.mkdir(parents=True, exist_ok=True)
                q.write_text(content)
            subprocess.run(["git", "-C", str(r), "init", "-q"], check=True)
            if tag:
                subprocess.run(["git", "-C", str(r), "add", "-A"], check=True,
                               capture_output=True)
                subprocess.run(["git", "-C", str(r), "-c", "user.email=t@e",
                                "-c", "user.name=t", "commit", "-qm", "x"],
                               check=True, capture_output=True)
                subprocess.run(["git", "-C", str(r), "tag", tag], check=True,
                               capture_output=True)
            return r

        TS = "export function widget() { return 1 }\n"

        # Exports and nothing else. No manifest, no tags: this repository does
        # not ship, and must not be told to bind five providers.
        only_exports = _mkrepo("only-exports", {"src/a.ts": TS, "src/b.ts": TS, "src/c.ts": TS})
        a = O.assess(only_exports)
        rep.add("issue-164",
                "the API-surface floor reads a declared package identity, not a source substring",
                not a["indicators"]["public_api_surface"],
                "a file containing the word 'export' is module syntax, not a compatibility "
                "obligation; this floored 100% of TypeScript repositories")
        rep.add("issue-164",
                "a repository that ships nothing is not floored to HIGH_ASSURANCE",
                a["profile"] != "GOVERNOR_HIGH_ASSURANCE",
                f"assessed {a['suggested']} / {a['profile']} — that profile requires five "
                "bound roles and manifest.py errors on each missing one (§54)")

        # A published identity. Without this the check above passes by never
        # firing at all, which removes the capability instead of correcting it.
        pkg = _mkrepo("published", {
            "package.json": '{"name": "widget", "version": "1.0.0", "main": "src/a.ts"}\n',
            "src/a.ts": TS})
        rep.add("issue-164", "a published package identity raises the floor",
                O.assess(pkg)["indicators"]["public_api_surface"],
                "a named, non-private manifest that declares an entry point is a claim "
                "that something outside this repository depends on it")

        # A name and a version and nothing else is `npm init` output, which every
        # application has. The B-growing fixture carries exactly that and expects
        # L2; floring it would tell ordinary applications they carry
        # compatibility obligations. Costs 3 of 49 on 105 real repositories.
        bare = _mkrepo("bare-manifest", {
            "package.json": '{"name": "app", "version": "1.0.0"}\n', "src/a.ts": TS})
        rep.add("issue-164", "a bare name-and-version manifest does not raise the floor",
                not O.assess(bare)["indicators"]["public_api_surface"],
                "npm init output is not a declaration that anything depends on this")

        # `"private": true` is honoured. A workspace root has a manifest and
        # ships nothing; treating every package.json as a publication would
        # reintroduce the false positives by a different route.
        # Carries an entry point TOO, so only the `private` guard can decline it.
        # The first version omitted `main`, which meant the strict entry-point
        # test already declined it and the private check was never exercised --
        # the assertion passed whether or not `private` was honoured, and a
        # mutation removing the guard survived.
        priv = _mkrepo("private-pkg", {
            "package.json": '{"name": "wsroot", "private": true, "main": "src/a.ts"}\n',
            "src/a.ts": TS})
        rep.add("issue-164", "a package marked private does not raise the floor",
                not O.assess(priv)["indicators"]["public_api_surface"],
                "'private': true says explicitly that this is not published")

        # Tags with no manifest. Measured near-independent from the package
        # signal (both=11, package-only=30, tags-only=11), so either alone
        # misses a real population.
        tagged = _mkrepo("tagged", {"main.go": "package main\n"}, tag="v1.0.0")
        rep.add("issue-164", "tags raise the floor even with no package manifest",
                O.assess(tagged)["indicators"]["public_api_surface"],
                "a project can tag releases without publishing a manifest")

        # A Go repository with one stray TypeScript file. The old scan took the
        # first sixty source files across ALL extensions, so this was floored on
        # a language the repository does not use -- Go has no `export` keyword,
        # and 11 of 14 Go repositories were floored anyway.
        mixed = _mkrepo("go-with-stray-ts", {
            "main.go": "package main\n", "internal/x.go": "package internal\n",
            "scripts/helper.ts": TS})
        rep.add("issue-164", "the floor scan is bounded by the languages the repository uses",
                not O.assess(mixed)["indicators"]["public_api_surface"],
                "a stray .ts file floored Go-dominant repositories, verified on a real one")

        # gate 4: proposal is written but the ENGINE never reads it.
        a = repos["A-greenfield"]
        subprocess.run([sys.executable, str(ROOT / "engine" / "onboard.py"), str(a), "--write"],
                       capture_output=True, check=True)
        wrote = (a / O.PROPOSAL).exists()
        rep.add("gate-4", "onboard --write emits a proposal", wrote, "")
        src = (ROOT / "engine" / "manifest.py").read_text()
        rep.add("gate-4", "manifest loader never reads the proposal",
                "proposed" not in src, "loader references the proposal file")
        # Renaming the proposal to the real name is the ONLY way to bind.
        rep.add("gate-4", "binding requires promotion by a human",
                O.PROPOSAL != ".repo-governor.json", "")

        # --- the proposal path is run end to end, because it never had been -------
        # Until 2026-08-19 both onboard.py's docstring and its emitted $comment said
        # "rename to .repo-governor.json and commit". Doing that yields
        # UNSUPPORTED_VERSION: the proposal is a candidates document with no version,
        # no providers block and no permissions. A whole documented workflow, wrong
        # at the last step, because nothing had ever executed it.
        import subprocess as _sp, tempfile as _tf, pathlib as _pl
        ROOT_ = Path(__file__).resolve().parent.parent
        with _tf.TemporaryDirectory() as td:
            tgt = _pl.Path(td) / "r"
            tgt.mkdir()
            _sp.run(["git", "init", "-q", str(tgt)], capture_output=True)
            _sp.run(["git", "-C", str(tgt), "remote", "add", "origin",
                     "https://github.com/acme/widget.git"], capture_output=True)
            _sp.run(["git", "-C", str(tgt), "-c", "user.email=t@t", "-c", "user.name=t",
                     "commit", "-q", "--allow-empty", "-m", "i"], capture_output=True)

            # 1. detection's own proposal must NOT masquerade as bindable
            _sp.run([sys.executable, str(ROOT_ / "engine" / "onboard.py"), str(tgt), "--write"],
                    capture_output=True)
            prop = tgt / ".repo-governor.proposed.json"
            rep.add("proposal-e2e", "detection writes a proposal", prop.exists())
            if prop.exists():
                body = prop.read_text()
                rep.add("proposal-e2e", "the proposal says renaming it does not bind",
                               "DOES NOT BIND" in body.upper(),
                               "it told people to rename it, and that produced an invalid manifest")
                prop.rename(tgt / ".repo-governor.json")
                r = _sp.run([sys.executable, str(ROOT_ / "engine" / "manifest.py"), "--validate"],
                            capture_output=True, text=True, cwd=str(tgt))
                rep.add("proposal-e2e", "a renamed detection proposal is correctly REJECTED",
                               "INVALID" in r.stdout, "it is evidence, not a manifest")
                (tgt / ".repo-governor.json").unlink()

            # 2. the interactive tool must produce something that actually validates
            r = _sp.run([sys.executable, str(ROOT_ / "tools" / "onboard-interactive.py"), str(tgt)],
                        input="1\nn\n1\n\n", capture_output=True, text=True)
            tool_out = r.stdout      # `r` is reused for --validate below
            # Withholding, not warning. The survey printed "option 2 cannot
            # work here" directly above the question and it was chosen anyway,
            # five times running. A warning the reader must notice is not a
            # control. The withholding path itself needs `gh`; what is checked
            # here is that it exists, and that declining the lookup does not
            # trigger it -- with no lookup we do not know which signals exist,
            # so withholding one would be the guess ADR-018 forbids.
            src_t2 = (ROOT_ / "tools" / "onboard-interactive.py").read_text()
            rep.add("proposal-e2e", "an unavailable signal is refused, not removed",
                    "UNAVAILABLE" in src_t2 and "Pick another" in src_t2,
                    "removing it renumbers the list, and the same keystroke then "
                    "selects a DIFFERENT signal -- observed on the very next run")
            # The survey WITHHOLDS a zero-count signal and `ask` refuses it if
            # chosen, so a wrong count does not mislead -- it refuses a correct
            # answer. GitHub's milestones endpoint defaults to state=open, so a
            # repository whose milestones are all closed counted zero and the
            # operator was refused milestone admission on a repository where it
            # works: the engine reads `milestone` without consulting its state
            # (issue 221). Measured on this repository: 4 open, 6 closed.
            #
            # Structural, because the behavioural version needs `gh` and a
            # repository with only closed milestones, and the hermetic suite has
            # neither. It asserts the query, which is where the defect was.
            rep.add("proposal-e2e", "the milestone survey is not blind to closed milestones",
                    "milestones?state=all" in src_t2,
                    "the endpoint defaults to state=open; a repository whose "
                    "milestones are all closed then counts zero and milestone "
                    "admission is REFUSED, not merely unadvertised")
            rep.add("proposal-e2e", "and it shows the open/closed split rather than a bare total",
                    'm.get("s") == "open"' in src_t2,
                    "'10 milestones' hides that none is open, which is the one "
                    "case where what works today and what ADR-018 says diverge")
            rep.add("proposal-e2e", "the option list is never rebuilt from counts",
                    "for v, d in GH_SIGNALS]" in src_t2,
                    "options must keep their positions whatever the counts are")
            rep.add("proposal-e2e", "declining the survey marks nothing unavailable",
                    "UNAVAILABLE" not in tool_out,
                    "with no lookup we do not know which exist")
            # Compared against the CALL site, not the def. _confirm_condition is
            # defined near the top of the file, so comparing against `def`
            # measured source order rather than the order a user experiences --
            # and failed on correct code.
            rep.add("proposal-e2e", "the label is asked when the label is chosen",
                    src_t2.index("Which label means admitted")
                    < src_t2.index("= _confirm_condition("),   # the CALL, not the def
                    "it was asked after the condition question, reading as a "
                    "continuation of something else")
            prop = tgt / ".repo-governor.proposed.json"
            # Captured BEFORE the rename below moves the file away. A later
            # `if prop.exists():` guard tested a path this line deletes, so
            # eight assertions after it never ran -- the suite reported PASS
            # over checks that were never executed, which is the vacuity this
            # repository keeps rediscovering, this time as unreachable code.
            made_proposal = prop.exists()
            rep.add("proposal-e2e", "onboard-interactive writes a proposal", made_proposal,
                    r.stderr[:120])
            if prop.exists():
                # The tool prints the store-creation steps, then the rename. The
                # stores are created explicitly rather than by the tool -- same
                # pattern as bootstrap-decisions.sh for Dolt, whose probe also
                # requires the database to exist. Skipping them here would test a
                # path the tool does not tell anyone to take.
                #
                # Both directories, because the proposal now binds both roles
                # (issue 144). Creating only the one the tool used to bind would
                # leave this check passing for a manifest a user cannot reproduce.
                for store in ("decisions", "acceptance"):
                    (tgt / ".repo-governor" / store).mkdir(parents=True, exist_ok=True)
                prop.rename(tgt / ".repo-governor.json")
                r = _sp.run([sys.executable, str(ROOT_ / "engine" / "manifest.py"), "--validate"],
                            capture_output=True, text=True, cwd=str(tgt))
                rep.add("proposal-e2e", "its output VALIDATES as a manifest",
                               "READY_FOR_GOVERNANCE" in r.stdout,
                               r.stdout.strip()[:160])
                m = json.loads((tgt / ".repo-governor.json").read_text())
                # Issue 56 shipped a zero-dependency decision store and nothing
                # proposed it, so CAPTURE_ONLY -- the default disposition --
                # still had nowhere to record on the path a user actually
                # takes. The adapter passing Layer 1 proved nothing about that.
                _dh = m["providers"].get("decision_history")
                rep.add("proposal-e2e", "the proposal binds decision_history",
                        bool(_dh),
                        "without it CAPTURE_ONLY has nowhere to record, which is the "
                        "state adapters/decision-history-file was built to end")
                rep.add("proposal-e2e", "bound to the backend that needs no binary",
                        bool(_dh) and "decision-history-file" in json.dumps(_dh),
                        "a default that requires dolt on PATH is not a default")
                rep.add("proposal-e2e", "and as a list, since the role is multi-valued",
                        isinstance(_dh, list),
                        "ADR-013: a second backend may bind alongside")
                # Issue 144: the same defect, twice more. An unbound
                # acceptance_criteria is refused at the binding layer with
                # PERMISSION_DENIED, which completion.py turns into a BLOCKING
                # UNKNOWN -- so STOP_COMPLETE was not merely unreachable, it
                # failed as "something is broken" rather than as the honest
                # CONTINUE with NO_CRITERIA_DECLARED that the adapter exists to
                # give. That is worse than the decision_history case above.
                _ac = (m.get("providers") or {}).get("acceptance_criteria")
                rep.add("proposal-e2e", "the proposal binds acceptance_criteria",
                        bool(_ac),
                        "unbound, STOP_COMPLETE is unreachable in every generated "
                        "manifest and the failure reads as a broken provider")
                rep.add("proposal-e2e", "bound to the backend that needs no binary",
                        bool(_ac) and "acceptance-file" in json.dumps(_ac),
                        "same argument as decision-history-file: a default that "
                        "requires an install is not a default")

                # The sweep, so this is not fixed a third time one role at a
                # time. Every zero-install role whose detection gate is
                # satisfied must reach the proposal; nothing outside the table
                # and the two always-bound roles may appear.
                # Loaded as a module rather than grepped: the table is data the
                # tool acts on, and a source-text check would pass on a table
                # that exists and is never read. imports.py refuses greps for
                # exactly this reason.
                import importlib.util as _ilu
                _sp_oi = _ilu.spec_from_file_location(
                    "_oi", ROOT_ / "tools" / "onboard-interactive.py")
                _oi = _ilu.module_from_spec(_sp_oi)
                _sp_oi.loader.exec_module(_oi)
                _spec = getattr(_oi, "ZERO_INSTALL", {})
                rep.add("proposal-e2e", "the zero-install table is declared, not implied",
                        bool(_spec),
                        "a sweep that reads no table is a sweep that checks nothing")
                _ungated = {r for r, v in _spec.items() if not v.get("detect")}
                rep.add("proposal-e2e",
                        "every ungated zero-install role reaches the proposal",
                        _ungated <= set(m.get("providers") or {}),
                        f"missing {sorted(_ungated - set(m.get('providers') or {}))}")
                # The gated case, driven directly. The end-to-end repository
                # above has no ADRs, so it can never exercise it -- a mutation
                # that skipped every gated role survived the whole suite until
                # these two existed, because "architecture absent" was the
                # correct answer for that fixture either way.
                _det = _oi.build_providers(
                    "github-projects", "adapters/github-projects", "o/r", None, None,
                    {"architecture": [{"type": "adr", "adapter": "adapters/adr",
                                       "disposition": "PROVIDER_DETECTED"}]})
                rep.add("proposal-e2e", "a DETECTED architecture provider is proposed",
                        "architecture" in _det,
                        "engine/onboard.py already prints PROVIDER_DETECTED with its "
                        "evidence; the proposal used to discard it, so a human never "
                        "saw the thing they were meant to accept (INV-013)")
                rep.add("proposal-e2e", "...as a list, since the role is multi-valued",
                        isinstance(_det.get("architecture"), list),
                        "ADR-013 puts architecture in ARRAY_ROLES")
                _und = _oi.build_providers(
                    "github-projects", "adapters/github-projects", "o/r", None, None, {})
                rep.add("proposal-e2e", "an UNDETECTED one is not",
                        "architecture" not in _und,
                        "proposing a role nothing detected is detection inventing "
                        "evidence, which is the opposite defect (ADR-010)")

                # ISSUE 199. Detection computed `docs/adr`, the evidence
                # document recorded it, and build_providers read the candidate
                # only as a boolean -- so the manifest bound adapters/adr with
                # no path, the adapter defaulted to `docs/adrs`, found nothing,
                # and reported TRANSPORT_UNREACHABLE. That is what a BROKEN
                # ADAPTER looks like, which is why the reporter read a correct
                # adapter as a defective one.
                _p = _oi.build_providers(
                    "github-projects", "adapters/github-projects", "o/r", None, None,
                    {"architecture": [{"type": "adr", "adapter": "adapters/adr",
                                       "path": "docs/adr",
                                       "disposition": "PROVIDER_DETECTED"}]})
                rep.add("proposal-e2e", "the detected path reaches the manifest",
                        (_p.get("architecture") or [{}])[0].get("path") == "docs/adr",
                        "a repository whose ADRs are not in docs/adrs binds an adapter "
                        "that finds nothing, and says so in the vocabulary of a broken "
                        "transport")
                # Control. Absence must stay absence: inventing a path would
                # bind every repository to a directory nobody detected.
                rep.add("proposal-e2e", "control: no detected path, no path key",
                        "path" not in (_det.get("architecture") or [{}])[0],
                        "a default written into the manifest is a claim detection "
                        "never made")
                # The table says adr. A repository carrying only openspec/ was
                # proposed adapters/adr because the entry was built from the
                # constant while the candidate that triggered it sat unread.
                _o = _oi.build_providers(
                    "github-projects", "adapters/github-projects", "o/r", None, None,
                    {"architecture": [{"type": "openspec", "adapter": "adapters/openspec",
                                       "disposition": "PROVIDER_DETECTED"}]})
                _oe = (_o.get("architecture") or [{}])[0]
                rep.add("proposal-e2e", "the adapter comes from the candidate, not the table",
                        _oe.get("adapter") == "adapters/openspec" and _oe.get("type") == "openspec",
                        f"got {_oe.get('type')}/{_oe.get('adapter')} -- ZERO_INSTALL names "
                        "one adapter per role, which is right only for roles nothing detects")
                # Control. UNCONFIRMED is detection saying "something might be
                # here" -- docs/decisions holding zero readable files is the
                # ordinary case. Binding it yields a provider that answers
                # nothing, which is the defect above wearing a different hat.
                _u = _oi.build_providers(
                    "github-projects", "adapters/github-projects", "o/r", None, None,
                    {"architecture": [{"type": "adr", "adapter": "adapters/adr",
                                       "path": "docs/decisions",
                                       "disposition": "PROVIDER_UNCONFIRMED"}]})
                rep.add("proposal-e2e", "control: an UNCONFIRMED candidate is not bound",
                        "architecture" not in _u,
                        "detection distinguishes DETECTED from UNCONFIRMED; the manifest "
                        "writer is the first place that distinction is honoured")
                # The partition is exhaustive against what detection REALLY
                # emits, derived by running it rather than by listing fields
                # here -- a hardcoded list is the second source of truth that
                # let `path` go missing with nothing to notice.
                _keys = set()
                for _cl in O.detect(ROOT_).values():
                    for _c in _cl:
                        _keys |= set(_c)
                _known = set(getattr(_oi, "CARRIED", ())) | set(getattr(_oi, "NOT_CARRIED", ()))
                rep.add("proposal-e2e", f"detection emitted fields to classify ({len(_keys)})",
                        bool(_keys),
                        "a detection run that yields no candidate fields makes the "
                        "partition check below vacuous")
                rep.add("proposal-e2e", "every field detection emits is carried or named",
                        _keys <= _known,
                        f"unclassified: {sorted(_keys - _known)} -- a field must be "
                        "carried into the manifest or listed as detection's own "
                        "bookkeeping, so the next one cannot go missing silently")

                # ISSUE 199, the half that cost the reporter the most time. With
                # no `path` the adapter defaults to docs/adrs, finds nothing, and
                # `--validate` said "advertises no capabilities; transport is not
                # configured" -- the vocabulary of a BROKEN ADAPTER for an adapter
                # that was working and pointed somewhere empty.
                import os as _os_, re as _re_, subprocess as _sub
                _tmp = tempfile.mkdtemp()
                _r = _sub.run([sys.executable, str(ROOT_ / "adapters" / "adr"), "describe"],
                              capture_output=True, text=True, cwd=_tmp,
                              env={**_os_.environ, "REPO_GOVERNOR_TARGET": _tmp})
                _d = json.loads(_r.stdout or "{}")
                _msg = ((_d.get("transport") or {}).get("detail") or "")
                shutil.rmtree(_tmp, ignore_errors=True)
                rep.add("proposal-e2e", "an unreachable ADR provider names the directory it tried",
                        "docs/adrs" in _msg,
                        f"got {_msg!r} -- a refusal that does not name the path it looked "
                        "in reads as a broken adapter, and was read that way")
                # The message lists the conventions. That list is O.ADR_DIRS and
                # must not become a second copy of it -- the defect this
                # repository has fixed in imports.py, SUPERSEDED_RE, the census
                # and the reference counts.
                # TOKENS, not substrings. `docs/adr` is a substring of
                # `docs/adrs`, so a containment test passes with the SINGULAR
                # form -- the one this whole issue is about -- deleted from the
                # message. Found by mutation; the first version of this check
                # was vacuous in exactly the case it exists for.
                _missing = [c for c in O.ADR_DIRS
                            if not _re_.search(rf"(?<![\w/]){_re_.escape(c)}(?![\w/])", _msg)]
                rep.add("proposal-e2e", "...and its convention list matches ADR_DIRS",
                        not _missing,
                        f"missing {_missing} -- the adapter "
                        "cannot import engine/onboard.py, so this assertion is the only "
                        "thing holding the two lists together")

                # ISSUE 201, the general property rather than the instance. A
                # candidate that NAMES an adapter is claiming that adapter can
                # serve it. Checked by spawning it against the repository that
                # triggered the detection -- not by grepping onboard.py, which
                # would pass on a table that exists and is never read.
                _fx = tempfile.mkdtemp()
                (Path(_fx) / ".beads").mkdir()
                (Path(_fx) / "docs" / "adr").mkdir(parents=True)
                (Path(_fx) / "docs" / "adr" / "001-t.md").write_text(
                    "# 1. T\n\n**Status**: Accepted\n", encoding="utf-8")
                (Path(_fx) / "renovate.json").write_text("{}", encoding="utf-8")
                _sub.run(["git", "init", "-q", "."], cwd=_fx, capture_output=True)
                _cands = O.detect(Path(_fx))
                _flat = [c for cs in _cands.values() for c in cs]
                rep.add("proposal-e2e", f"the over-promise fixture produced candidates ({len(_flat)})",
                        len(_flat) >= 3,
                        "a fixture that detects nothing makes every claim below vacuous")
                _bad = []
                for _c in _flat:
                    if not _c.get("adapter"):
                        continue
                    _env = {**_os_.environ, "REPO_GOVERNOR_TARGET": _fx}
                    _env.pop("REPO_GOVERNOR_BINDING", None)
                    if _c.get("path"):
                        _env["REPO_GOVERNOR_BINDING"] = json.dumps({"path": _c["path"]})
                    _rr = _sub.run([sys.executable, str(ROOT / _c["adapter"]), "describe"],
                                   capture_output=True, text=True, cwd=_fx, env=_env)
                    _tt = (json.loads(_rr.stdout or "{}").get("transport") or {})
                    if _tt.get("reachable") is not True:
                        _bad.append(f"{_c['type']}->{_c['adapter']}")
                rep.add("proposal-e2e", "every named adapter can read what its evidence cites",
                        not _bad,
                        f"unreachable against the repository that triggered them: {_bad} -- "
                        "detection may name an adapter only if that adapter can read the "
                        "evidence that triggered the detection")
                # The two instances, by name, because both were shipped and both
                # stated the rule they broke in a comment directly above.
                _ex = [c for c in _cands.get("execution", []) if c.get("type") == "beads"]
                rep.add("proposal-e2e", "a .beads store is detected and names no adapter",
                        bool(_ex) and not _ex[0].get("adapter"),
                        "adapters/execution-file reads .repo-governor/execution.json and "
                        "cannot open a SQLite store")
                _cs = _cands.get("change_signals", [])
                rep.add("proposal-e2e", "a renovate config is detected and names no adapter",
                        bool(_cs) and not any(c.get("adapter") for c in _cs),
                        "adapters/change-signals-file reads a local signals file, not a "
                        "Renovate config")

                # EVERY ARGUMENT build_providers RECEIVES IS BOUND ON EVERY PATH.
                # `label` was initialised only inside `if choice ==
                # "github-projects"` and passed unconditionally, so picking
                # Linear or file-roadmap raised UnboundLocalError -- the tool
                # could onboard a GitHub repository and nothing else. It shipped
                # in #145 and was found by an external user, not here, because
                # every assertion in this suite calls build_providers() directly
                # with "github-projects" and never drives main().
                #
                # Static, not driven: exercising the GitHub path end-to-end needs
                # a live `gh` probe, so a dynamic check would make this LIVE and
                # would still only cover the branch that already worked.
                import ast as _ast
                _src = (ROOT / "tools" / "onboard-interactive.py").read_text(encoding="utf-8")
                _tree = _ast.parse(_src)
                _main = next((n for n in _ast.walk(_tree)
                              if isinstance(n, _ast.FunctionDef) and n.name == "main"), None)
                rep.add("proposal-e2e", "onboard-interactive declares main()", _main is not None,
                        "the scope check below reads main(); without it, it checks nothing")
                if _main is not None:
                    _args = []
                    for _n in _ast.walk(_main):
                        if (isinstance(_n, _ast.Call) and isinstance(_n.func, _ast.Name)
                                and _n.func.id == "build_providers"):
                            _args = [a.id for a in _n.args if isinstance(a, _ast.Name)]
                    # Bound unconditionally = assigned in main's own body, or a
                    # parameter. An assignment nested inside if/for/try is not.
                    _bound = {t.id for st in _main.body if isinstance(st, (_ast.Assign, _ast.AugAssign))
                              for t in (st.targets if isinstance(st, _ast.Assign) else [st.target])
                              if isinstance(t, _ast.Name)}
                    _bound |= {a.arg for a in _main.args.args}
                    # Tuple unpacking at body level counts too.
                    for st in _main.body:
                        if isinstance(st, _ast.Assign):
                            for t in st.targets:
                                if isinstance(t, _ast.Tuple):
                                    _bound |= {e.id for e in t.elts if isinstance(e, _ast.Name)}
                    rep.add("proposal-e2e",
                            f"build_providers' arguments were found ({len(_args)})",
                            len(_args) >= 4,
                            "the call was not located, so the binding check below is vacuous")
                    _unbound = [a for a in _args if a not in _bound]
                    rep.add("proposal-e2e",
                            "every build_providers argument is bound on every path",
                            not _unbound,
                            f"conditionally bound: {_unbound} -- assigned inside a branch and "
                            "passed unconditionally, so a roadmap choice that skips that branch "
                            "raises UnboundLocalError before any manifest is written")
                shutil.rmtree(_fx, ignore_errors=True)

                _allowed = set(_spec) | {"repository", "roadmap_authority"}
                rep.add("proposal-e2e", "and nothing outside the table is proposed",
                        set(m.get("providers") or {}) <= _allowed,
                        f"unexpected {sorted(set(m.get('providers') or {}) - _allowed)}; "
                        "execution, change_signals and retirement staying unbound is "
                        "INV-013 working, not a defect to fix")

                _w = (m.get("permissions") or {}).get("decision_history", {})
                rep.add("proposal-e2e", "decision_history is the only write granted",
                        _w.get("write") is True
                        and not any(v.get("write") for k, v in m["permissions"].items()
                                    if isinstance(v, dict) and k != "decision_history"),
                        "recording a capture needs it; nothing else does, and a broader "
                        "grant would be an escalation buried in generated JSON")
                rep.add("proposal-e2e", "the write grant is explained where a reviewer meets it",
                        "write=true" in json.dumps(m["permissions"].get("$comment", "")),
                        "the schema closes `permission` to a verb set, so the reason "
                        "cannot sit beside the entry and must lead the block")

                rep.add("proposal-e2e", "it declares an admission signal (ADR-018)",
                               bool(m["providers"]["roadmap_authority"].get("admission", {}).get("signal")))
                perms = m.get("permissions") or {}
                rep.add("proposal-e2e", "it declares a permissions block at all",
                               bool(perms),
                               "manifest.py rejects a manifest without one")
                # Was "no write anywhere", written when the proposal bound no
                # role that needs one. decision_history does: CAPTURE_ONLY
                # cannot record without it (issue 94). Narrowed rather than
                # dropped, and narrowed in the strict direction -- the concern
                # was never write itself but an UNEXPLAINED escalation in
                # generated JSON, so this asserts the grant is confined to one
                # role and the checks above assert it is explained.
                stray = [k for k, v in perms.items()
                         if isinstance(v, dict) and v.get("write") and k != "decision_history"]
                rep.add("proposal-e2e",
                        "no write is granted outside decision_history (ADR-005)",
                        not stray,
                        f"{stray} -- a proposal that grants write asks a reviewer to "
                        "notice an escalation buried in generated JSON")
                # Two checks, because the generator emitted admission.signal --
                # which the adapter ignores -- and no env block at all,
                # producing a manifest that validated as READY_FOR_GOVERNANCE
                # and then answered PROVIDER_UNAVAILABLE for every id.
                #
                # 1. the vars whose ABSENCE makes the adapter refuse. Only these
                #    are required; the adapter reads three others that are
                #    optional, and demanding all five was this check's first,
                #    wrong version.
                import re as _re2
                adp = (ROOT_ / "adapters" / "github-projects").read_text()
                env_ = m["providers"]["roadmap_authority"].get("env", {})
                for var, why in (("REPO_GOVERNOR_GH_REPO",
                                  "an adapter that cannot tell which repository it "
                                  "reads must not answer (ADR-028)"),
                                 ("REPO_GOVERNOR_GH_ADMISSION",
                                  "the admission signal is declared, never assumed "
                                  "(ADR-018)")):
                    rep.add("proposal-e2e", f"the manifest supplies {var}",
                            bool(env_.get(var)), why)

                # 2. and nothing the adapter does not read -- generic drift catch
                #    in the other direction: a key nobody consumes is dead config
                #    that looks like configuration.
                known = set(_re2.findall(
                    r'os\.environ\.get\(\s*"(REPO_GOVERNOR_GH_[A-Z_]+)"', adp))
                stray = sorted(set(env_) - known)
                # Validation is not enough and the tool must say so. Three
                # onboarding attempts declared project_status on a repository
                # with no Project; --validate stayed green and every id read
                # NOT_ON_BOARD. Only asking a real question exposes that.
                # The survey is consent-gated: declining it must still produce a
                # working manifest. It informs the choice; it is not a step.
                # The condition must be the ASSESSED one. It was hardcoded to
                # The condition is ASSESSED and then CONFIRMED by a person. It
                # was hardcoded to L1/GOVERNOR_LITE with a comment telling the
                # reader to fix it by hand -- on a repository detection had put
                # at L4 because floor indicators fired. The engine says a floor
                # may not be overridden downward, so the tool was committing the
                # violation it printed a warning about two screens earlier.
                a = _sp.run([sys.executable, str(ROOT_ / "engine" / "onboard.py"),
                             str(tgt), "--json"], capture_output=True, text=True)
                want = json.loads(a.stdout)["condition"]
                rep.add("proposal-e2e", "the condition is assessed, not defaulted",
                        m["condition"]["assessed"] == want["suggested"],
                        f"proposal {m['condition']['assessed']}, "
                        f"assessed {want['suggested']}")
                rep.add("proposal-e2e", "the tool shows the indicators behind the level",
                        "indicators" in tool_out,
                        "a bare verdict cannot be weighed by a person or an agent")

                # Obligation indicators are reported, and are NOT floors. A
                # floor may not be lowered, so a missing licence would
                # permanently pin a repository to L4 for a reason unrelated to
                # how deeply its code needs governing.
                _ind = want.get("indicators") or {}
                rep.add("proposal-e2e", "license_present is reported as an indicator",
                        "license_present" in _ind,
                        "a repository with no licence grants nobody anything; its owner "
                        "should be told, and detection sees it for free")
                rep.add("proposal-e2e", "readme_present is reported as an indicator",
                        "readme_present" in _ind)
                rep.add("proposal-e2e", "neither obligation indicator raises the floor",
                        "license_present" not in O.FLOOR_INDICATORS
                        and "readme_present" not in O.FLOOR_INDICATORS,
                        "a floor cannot be lowered, so this would pin a repository to L4 "
                        "over a fact unrelated to its governance depth")

            # A floor may be raised, never lowered. Build a repo that floors at
            # L4 and try to force it down.
            fl = _pl.Path(td) / "floored"
            (fl / "src").mkdir(parents=True)
            _sp.run(["git", "init", "-q", str(fl)], capture_output=True)
            _sp.run(["git", "-C", str(fl), "remote", "add", "origin",
                     "https://github.com/acme/w.git"], capture_output=True)
            (fl / "src" / "a.ts").write_text("export const a = 1;\n")
            # Floored by a PUBLISHED IDENTITY, not by the word `export`. This
            # fixture exists to prove a floor cannot be overridden downward, so
            # it has to be floored by something that still floors (issue 164).
            (fl / "package.json").write_text('{"name": "floored", "version": "1.0.0", "main": "src/a.ts"}\n')
            _sp.run(["git", "-C", str(fl), "add", "-A"], capture_output=True)
            _sp.run(["git", "-C", str(fl), "-c", "user.email=t@t", "-c", "user.name=t",
                     "commit", "-q", "-m", "i"], capture_output=True)
            _sp.run([sys.executable, str(ROOT_ / "tools" / "onboard-interactive.py"),
                     str(fl)], input="1\nn\n1\nL0\n", capture_output=True, text=True)
            fp = fl / ".repo-governor.proposed.json"
            got_l = json.loads(fp.read_text())["condition"]["assessed"] if fp.exists() else None
            rep.add("proposal-e2e", "a floor cannot be overridden downward",
                    got_l == "L4",
                    f"asked for L0 on a floored repo and got {got_l!r}; "
                    "compatibility obligations do not go away by declaring a lower level")
            if fp.exists():
                rep.add("proposal-e2e", "the manifest names why the floor holds",
                        "floor" in json.dumps(json.loads(fp.read_text())["condition"]).lower(),
                        "a reviewer must see why the level cannot be lowered")

            if made_proposal:
                rep.add("proposal-e2e", "declining the signal survey still binds",
                        m["providers"]["roadmap_authority"]["admission"]["signal"]
                        == "milestone",
                        "the lookup is a convenience, not a required step")
                src_t = (ROOT_ / "tools" / "onboard-interactive.py").read_text()
                rep.add("proposal-e2e", "the roadmap option does not say 'Projects'",
                        '"GitHub"' in src_t and "GitHub issues / Projects" not in src_t,
                        "naming Projects there primed four consecutive operators to "
                        "pick 'Project column/status' at the NEXT question")
                # Behavioural: drive selftest against a repository that HAS a
                # hook. Grepping selftest for the word "hook" would pass on a
                # file that merely mentions it, and this precondition already
                # went unwritten once by depending on memory (issue 86).
                _hooked = _pl.Path(td) / "hooked"
                (_hooked / ".cursor").mkdir(parents=True)
                _sp.run(["git", "init", "-q", str(_hooked)], capture_output=True)
                (_hooked / ".cursor" / "hooks.json").write_text("{}")
                _st = _sp.run([sys.executable, str(ROOT_ / "tools" / "selftest.py"),
                               str(_hooked)], capture_output=True, text=True).stdout
                rep.add("proposal-e2e", "selftest detects an installed hook",
                        "hook is installed" in _st,
                        "a hooked repository measures DELIVERY, not activation, and "
                        "the operator must be told which number they are taking")
                rep.add("proposal-e2e", "and names which measurement the prompts make",
                        "DELIVERY, not activation" in _st)
                _clean = _pl.Path(td) / "unhooked"
                _clean.mkdir()
                _sp.run(["git", "init", "-q", str(_clean)], capture_output=True)
                _st2 = _sp.run([sys.executable, str(ROOT_ / "tools" / "selftest.py"),
                                str(_clean)], capture_output=True, text=True).stdout
                rep.add("proposal-e2e", "control: an unhooked repository measures activation",
                        "measure ACTIVATION" in _st2,
                        "otherwise the check above passes for any output at all")

                rep.add("proposal-e2e",
                        "the contribution pointer is a URL, not a pruned filename",
                        "blob/main/CONTRIBUTING.md" in src_t
                        and "See CONTRIBUTING.md" not in src_t,
                        "install-skill.sh prunes CONTRIBUTING.md, so naming the file "
                        "points at something the install does not carry -- and where the "
                        "host repository has its own, it names the wrong one")
                rep.add("proposal-e2e", "the survey warns existing != means admitted",
                        "not the same as MEANING admitted" in src_t,
                        "a repo can have milestones that are release buckets")

                rep.add("proposal-e2e", "the tool tells you to ask a real question",
                        "completion.py" in tool_out and "validation does not catch" in tool_out,
                        "--validate passes on a wrong admission signal (issue 51)")
                rep.add("proposal-e2e", "it says what a wrong signal looks like",
                        "NOT_ON_BOARD" in tool_out and "NOT_ADMITTED" in tool_out,
                        "the same reason for every id is the tell")

                rep.add("proposal-e2e", "the manifest sets no env the adapter ignores",
                        not stray, f"{stray} is dead config that reads as configuration")

                rep.add("proposal-e2e", "it reads the repository id from the remote",
                               m["repository"]["id"] == "acme/widget",
                               f"got {m['repository']['id']!r}")
                (tgt / ".repo-governor.json").unlink()

            # ADR-028 is only exercised where there IS no remote. The first
            # version of this check asserted the id on a repo that had one, so a
            # hardcoded fallback sailed straight through -- the default it
            # existed to forbid sat on a path the fixture never reached.
            bare = _pl.Path(td) / "bare"
            bare.mkdir()
            _sp.run(["git", "init", "-q", str(bare)], capture_output=True)
            # Feed VALID answers. With no remote the tool must ASK, so the first
            # answer is consumed as the id. A hardcoded fallback would skip the
            # question and stamp its own value instead -- which is precisely the
            # defect ADR-028 exists for, and precisely what an earlier version of
            # this check missed by asserting on a repo that had a remote.
            r = _sp.run([sys.executable, str(ROOT_ / "tools" / "onboard-interactive.py"),
                         str(bare)], input="me/mine\n1\nn\n1\n\n", capture_output=True, text=True)
            bp = bare / ".repo-governor.proposed.json"
            got = json.loads(bp.read_text())["repository"]["id"] if bp.exists() else None
            # The tool prints a URL. A URL that 404s is worse than no URL: it
            # tells someone their tracker is a gap AND that nobody wants to hear
            # about it. Check the template exists and that the two names agree.
            # Every issue form is parsed, not grepped. A malformed one does not
            # error -- GitHub serves a blank issue box instead, so it looks
            # installed while asking none of its questions.
            for name in ("adapter-request.yml", "activation-result.yml", "config.yml"):
                f = ROOT_ / ".github" / "ISSUE_TEMPLATE" / name
                rep.add("templates", f"{name} exists", f.exists())
                if f.exists() and name != "config.yml":
                    ok_f, why_f = _parse_form(f)
                    rep.add("templates", f"{name} is a well-formed issue form", ok_f, why_f)
                    rep.add("templates", f"{name} carries the public-repo rule (51)",
                            "51" in f.read_text() and "public" in f.read_text().lower())

            # The activation form must not invite a run that cannot be scored.
            av = (ROOT_ / ".github" / "ISSUE_TEMPLATE" / "activation-result.yml").read_text()
            for phrase, why in (
                ("One prompt per session", "batching measures persistence, not activation"),
                ("did not correct the agent", "a rescued miss is destroyed data"),
                ("never named Repo Governor", "naming it guarantees activation"),
                ("competing", "a rate without its field cannot be compared"),
            ):
                rep.add("templates", f"activation form asks about: {phrase}",
                        phrase.lower() in av.lower(), why)

            # The self-test must not become a mechanical checklist. Its whole
            # point is that the mechanical half proves nothing: a skill can be
            # installed, a manifest valid, and the model still never consult it.
            st = (ROOT_ / "tools" / "selftest.py")
            rep.add("templates", "selftest exists", st.exists())
            if st.exists():
                sb = st.read_text()
                rep.add("templates", "selftest says the mechanical half is insufficient",
                        "nowhere near sufficient" in sb,
                        "a green checklist that does not measure activation is the "
                        "exact failure ADR-001 names")
                rep.add("templates", "selftest carries a control prompt",
                        "CONTROL" in sb and "false positive" in sb,
                        "without one, a skill that fires on everything scores perfectly")
                rep.add("templates", "selftest tells the user what to do at <3/3",
                        "AGENTS.md" in sb and "hook" in sb,
                        "a diagnostic with no remedy is a complaint")
                rep.add("templates", "selftest links the report form",
                        "activation-result.yml" in sb)
                r_st = _sp.run([sys.executable, str(st), td], capture_output=True, text=True)
                rep.add("templates", "selftest runs on an ungoverned repo without crashing",
                        r_st.returncode == 0 and "not onboarded" in r_st.stdout,
                        r_st.stderr[:120])

            # The PR form must carry the two defects this repository actually has.
            pr = (ROOT_ / ".github" / "PULL_REQUEST_TEMPLATE.md").read_text()
            rep.add("templates", "PR template asks whether the test was mutated",
                    "mutated" in pr.lower(),
                    "PR 43 shipped a tautology that passed and looked like coverage")
            rep.add("templates", "PR template warns about closing keywords",
                    "closing verb" in pr.lower(),
                    "this repository has closed the same issue twice by accident")
            rep.add("templates", "PR template asks which authority admits the work",
                    "Authority:" in pr, "INV-002: admission is not authorization")

            tmpl = ROOT_ / ".github" / "ISSUE_TEMPLATE" / "adapter-request.yml"
            tool_src = (ROOT_ / "tools" / "onboard-interactive.py").read_text()
            rep.add("proposal-e2e", "the adapter-request template exists", tmpl.exists(),
                    "onboard-interactive.py links to it")
            rep.add("proposal-e2e", "the tool links to the template that exists",
                    tmpl.name in tool_src,
                    "a 404 tells someone their tracker is a gap and nobody wants to hear it")
            if tmpl.exists():
                body = tmpl.read_text()
                rep.add("proposal-e2e", "the template asks what ADMITTED means (ADR-018)",
                        "ADMITTED mean" in body,
                        "the one question detection can never answer")
                rep.add("proposal-e2e", "the template asks how work is WITHDRAWN",
                        "WITHDRAWN" in body,
                        "an agent working on withdrawn work is the failure this prevents")
                # PARSE it, not just grep it. A malformed issue form does not
                # error -- GitHub silently falls back to a blank issue box, so
                # the template appears to work while asking none of its
                # questions. String checks cannot see that.
                ok_parse, why = _parse_form(tmpl)
                rep.add("proposal-e2e", "the template is a well-formed GitHub issue form",
                        ok_parse, why)
                rep.add("proposal-e2e", "the template carries the public-repo warning",
                        "51" in body and "public" in body,
                        "requesters must not paste private workspace content")

            rep.add("proposal-e2e", "no remote: it asks rather than defaulting (ADR-028)",
                    got == "me/mine",
                    f"got {got!r} -- a value it was never told is a defaulted identity")


    cur = None
    for fx, check, ok, detail in rep.rows:
        if fx != cur:
            print(f"\n{fx}")
            cur = fx
        print(f"  [{'PASS' if ok else 'FAIL'}] {check}" + (f"\n         {detail}" if detail and not ok else ""))

    f = rep.fails()
    print(f"\n{len(rep.rows) - len(f)}/{len(rep.rows)} checks passed")
    print("RG-SIM-ONBOARDING-v0.1: " + ("PASS" if not f else f"FAIL ({len(f)})"))
    return 0 if not f else 1


if __name__ == "__main__":
    sys.exit(main())
