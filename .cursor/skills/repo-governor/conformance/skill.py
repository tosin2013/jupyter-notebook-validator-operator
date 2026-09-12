#!/usr/bin/env python3
"""The shipped agent surface is checked, not just written.

`SKILL.md` and `AGENTS.md` are the two files an agent actually reads, and they
drift from the code silently. Both defects this suite exists for were shipped:

  1. every command in SKILL.md was a bare relative path (`python3 engine/...`),
     which cannot work from the repository being governed. The obvious repair --
     cd into the skill so the paths resolve -- makes the engine govern the skill
     directory, which is the defect ADR-027 had just fixed.

  2. SKILL.md told the agent to invoke an adapter directly, bypassing the
     permission chokepoint ADR-021 had just been accepted for.

Neither was a stale citation, so a citation checker would have missed both. What
catches them is asking whether the documented commands work in the posture an
agent would use them, and whether the entry points they name exist.

The half this cannot check is a document quietly contradicting a decision it
never names -- that is a reading, and ADR-002 keeps readings out of the engine.

Usage:  python3 conformance/skill.py
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path

import _preflight  # noqa: E402 -- names an absent adapter dependency
import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("skill")


ROOT = Path(__file__).resolve().parent.parent
SKILL = ROOT / "SKILL.md"
AGENTS = ROOT / "AGENTS.md"

CMD_RE = re.compile(r"```bash\n(.*?)```", re.S)
ADR_RE = re.compile(r"ADR-(\d{3})")
ENTRY_RE = re.compile(r"engine/([a-z_]+)\.py")


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"    {detail}" if detail and not ok else ""))
    return 0 if ok else 1


def adr_status(num):
    for p in ROOT.glob(f"docs/adrs/{num}-*.md"):
        m = re.search(r"^\*\*Status\*\*:\s*(\w+)", p.read_text(), re.M)
        return m.group(1) if m else "UNSTATED"
    return None


def main():
    fails = 0
    skill = SKILL.read_text()
    agents = AGENTS.read_text() if AGENTS.exists() else ""

    print("The agent surface exists\n")
    fails += check("SKILL.md is present", SKILL.exists())
    fails += check("AGENTS.md is present — governance is pushed, not only pulled", AGENTS.exists())
    fails += check("CLAUDE.md points at AGENTS.md rather than duplicating it",
                   (ROOT / "CLAUDE.md").exists() and "AGENTS.md" in (ROOT / "CLAUDE.md").read_text())

    print("\nSKILL.md teaches an invocation that works from the governed repository\n")

    # 1. A bare relative engine path only resolves inside this checkout, which is
    #    the one repository you are almost never governing.
    bare = [ln.strip() for ln in skill.splitlines() if re.search(r"python3 engine/", ln)]
    fails += check("no bare `python3 engine/` path in SKILL.md", not bare, str(bare[:2]))

    # 2. Nor a bare adapter path -- that would also bypass the ADR-021 gate.
    bare_ad = [ln.strip() for ln in skill.splitlines() if re.search(r"python3 adapters/", ln)]
    fails += check("no direct adapter invocation in SKILL.md", not bare_ad, str(bare_ad[:2]))

    # 3. It must say how to locate the engine, or `$RG` is an unanswered question.
    fails += check("SKILL.md explains how to locate the skill directory",
                   "RG=" in skill and "base directory" in skill.lower())

    print("\nEvery entry point the surface names exists and runs\n")

    named = sorted(set(ENTRY_RE.findall(skill + agents)))
    missing = [n for n in named if not (ROOT / "engine" / f"{n}.py").exists()]
    fails += check(f"all {len(named)} named engine entry points exist", not missing, str(missing))

    # 4. Run the ones that need no argument, in the posture the docs describe:
    #    standing in a repository, engine addressed by absolute path.
    for entry, expect in (("manifest", "MANIFEST VALID"),):
        p = subprocess.run([sys.executable, str(ROOT / "engine" / f"{entry}.py")],
                           capture_output=True, text=True, cwd=str(ROOT), timeout=120)
        fails += check(f"engine/{entry}.py runs and reports {expect!r}",
                       expect in p.stdout, p.stdout[:100] or p.stderr[:100])

    # 5. And from a repository that is NOT governed, the documented failure is
    #    the one the surface promises -- naming the path it looked in.
    with tempfile.TemporaryDirectory() as td:
        # REPO_GOVERNOR_TARGET must be cleared. It declares the governed
        # repository (ADR-027) and outranks cwd -- so when this suite runs
        # THROUGH the engine, which sets it, the inherited value made an
        # ungoverned directory report MANIFEST VALID and this check fail. The
        # check was correct; its environment was not isolated from its caller.
        import os as _os2
        clean = {k: v for k, v in _os2.environ.items() if k != "REPO_GOVERNOR_TARGET"}
        p = subprocess.run([sys.executable, str(ROOT / "engine" / "manifest.py")],
                           capture_output=True, text=True, cwd=td, timeout=120, env=clean)
        out = p.stdout + p.stderr
        fails += check("an ungoverned repository yields AUTHORITY_SOURCE_MISSING naming its path",
                       "AUTHORITY_SOURCE_MISSING" in out and td in out, out[:120])

    print("\nNo third-party workspace content is carried in this public repository\n")

    # §51 forbids cross-repository leakage, and this repository is public. The
    # rule was asserted all through the Linear work and was ALREADY broken --
    # ADR-016 quoted a real issue identifier from a private workspace, found
    # only when the check was finally run rather than repeated. Synthetic
    # prefixes (ENG-, SIM-) are fine; a real workspace prefix is not.
    import subprocess as _sp3
    leaked = _sp3.run(["git", "grep", "-lIE", r"\bTOS-[0-9]{2,}"],
                      capture_output=True, text=True, cwd=str(ROOT)).stdout.split()
    fails += check("no real Linear workspace identifier is committed", not leaked, str(leaked))

    print("\nAny stated ADR count matches the ledger\n")

    # A count is a duplicated derivable fact, and duplicated state eventually
    # disagrees: AGENTS.md said "21 of 26 Accepted" after the ledger said 23,
    # because the line predated a ratification pass and nothing recomputed it.
    # Any "N of M Accepted"-shaped claim on the surface must match a recount
    # of the ledger itself -- or better, not exist at all.
    statuses = {}
    for f in sorted((ROOT / "docs" / "adrs").glob("[0-9][0-9][0-9]-*.md")):
        m = re.search(r"^\*\*Status\*\*:\s*(\w+)", f.read_text(), re.M)
        st = m.group(1) if m else "UNSTATED"
        statuses[st] = statuses.get(st, 0) + 1
    accepted, total = statuses.get("Accepted", 0), sum(statuses.values())
    count_docs = [("README.md", (ROOT / "README.md").read_text()),
                  ("AGENTS.md", agents),
                  ("docs/adrs/README.md", (ROOT / "docs" / "adrs" / "README.md").read_text())]
    for doc, text in count_docs:
        # A finditer that matches nothing runs its body zero times and the
        # suite goes green having checked NOTHING. That is the defect class
        # this whole file exists to catch, one level up: if a document is
        # rephrased, or a pattern rots, the check stops looking and never says
        # so. AGENTS.md deliberately carries no ADR count, so the floor is
        # per-document rather than global.
        seen_claims = 0
        for m in re.finditer(r"(\d+)\s+of\s+(\d+)\s+ADRs?[^.\n]{0,40}Accepted|"
                             r"(\d+)\s+ADRs?\s+—\s+\*\*(\d+)\s+Accepted|"
                             r"\*\*(\d+)\s+of\s+(\d+)\s+ADRs are Accepted|"
                             r"(\d+)\s+architectural decisions\s+\((\d+)\s+Accepted\)", text):
            nums = [g for g in m.groups() if g]
            claim = tuple(int(n) for n in nums)
            ok_claim = accepted in claim and total in claim
            seen_claims += 1
            fails += check(f"{doc} count claim {claim} matches ledger ({accepted} of {total})",
                           ok_claim, f"ledger says {accepted} Accepted of {total}")
        # README.md and docs/adrs/README.md both state a count today. If either
        # stops matching, that is a pattern that went blind, not a document
        # that got simpler -- and it must be reported as a failure.
        if doc != "AGENTS.md":
            fails += check(f"{doc} still states an ADR count this suite can read",
                           seen_claims > 0,
                           "found 0 count claims -- either the claim was removed "
                           "or the pattern stopped matching it; both need a human")

    print("\nThe decision index accounts for every decision\n")

    # A reader who finds ADR-024 and ADR-027 and nothing between has to guess
    # whether two decisions were deleted -- which would be the INV-005 failure
    # this project exists to prevent, reported by absence rather than by error.
    index = (ROOT / "docs" / "adrs" / "README.md").read_text()
    files = sorted(p.name[:3] for p in (ROOT / "docs" / "adrs").glob("[0-9][0-9][0-9]-*.md"))
    unindexed = [n for n in files if f"({n}-" not in index and f"~~{n}~~" not in index]
    fails += check(f"all {len(files)} ADR files appear in the index", not unindexed, str(unindexed))

    lo, hi = int(files[0]), int(files[-1])
    gaps = [f"{n:03d}" for n in range(lo, hi + 1) if f"{n:03d}" not in files]
    undocumented = [g for g in gaps if f"~~{g}~~" not in index]
    fails += check(f"every gap in the sequence is explained ({len(gaps)} gap(s))",
                   not undocumented, f"unexplained: {undocumented}")

    print("\nThe surface does not cite decisions that have moved\n")

    # One definition, in tools/_surface.py, shared with tools/surface-diff.py.
    # It was inline here and the diff needed the same list; a second copy of
    # "the agent surface" would drift from the first, which is the defect three
    # checks in this file already exist to catch.
    import sys as _sys
    _sys.path.insert(0, str(ROOT / "tools"))
    import _surface as _S
    surface = [(f, (ROOT / f).read_text(encoding="utf-8"))
               for f in _S.files_at("HEAD") if (ROOT / f).is_file()]

    # The workflow pages teach humans what to ask, so they are agent surface too
    # (issue 29's lesson: the instructions ARE the product). Same rules: no bare
    # relative engine path a reader could copy into the wrong repository.
    for doc, text in surface:
        if doc.startswith("docs/workflows/"):
            bare_wf = [ln.strip() for ln in text.splitlines()
                       if re.search(r"python3 (engine|adapters)/", ln)]
            fails += check(f"{doc} has no bare relative invocation", not bare_wf, str(bare_wf[:1]))

    for doc, text in surface:
        for num in sorted(set(ADR_RE.findall(text))):
            st = adr_status(num)
            if st is None:
                fails += check(f"{doc} cites ADR-{num}, which exists", False, "no such ADR")
            elif st == "Superseded":
                fails += check(f"{doc} does not cite superseded ADR-{num}", False,
                               "superseded decisions must not be cited as current")
            elif st == "Proposed":
                # Citing an unaccepted decision is allowed only where the text
                # says so -- otherwise the release treats it as normative.
                marked = re.search(r"ADR-" + num + r"[^\n]{0,80}(Proposed|experimental|not implemented)",
                                   text, re.I) or re.search(
                    r"(Proposed|experimental|held)[^\n]{0,80}ADR-" + num, text, re.I)
                fails += check(f"{doc} marks proposed ADR-{num} as unaccepted", bool(marked),
                               "cited without saying it is Proposed")
            elif st == "Accepted":
                # The mirror, and the one ratification itself creates: a document
                # that called a decision `Proposed` becomes wrong the moment the
                # decision is accepted. Staleness runs in both directions and
                # only one of them is obvious.
                stale = re.search(r"ADR-" + num + r"[^\n]{0,60}`?Proposed`?", text) or re.search(
                    r"`?Proposed`?[^\n]{0,60}ADR-" + num, text)
                fails += check(f"{doc} does not still call accepted ADR-{num} proposed", not stale,
                               "ratification made this line stale")

    # --- the README's install instructions must be the ones that work --------
    # For 83 commits the README said "Clone it into a skills directory". That
    # drags this repository's own AGENTS.md, CLAUDE.md and .repo-governor.json
    # into someone else's project, and the manifest makes the engine resolve the
    # INSTALL directory as the repository under governance (ADR-027) -- it then
    # answers confidently about the wrong project. tools/install-skill.sh was
    # written to prevent exactly that and the README never mentioned it.
    rd = (ROOT / "README.md").read_text()
    fails += check("README has an Install section", "\n## Install" in rd)
    fails += check("README install uses the script", "install-skill.sh" in rd,
                   "a plain clone installs this repo's manifest into someone else's repo")
    bad = re.search(r"[Cc]lone (it |this )?into a skills director", rd)
    fails += check("README does not tell people to clone into a skills directory",
                   not bad, bad.group(0) if bad else "")
    for tool in ("onboard-interactive.py", "selftest.py"):
        fails += check(f"README points at {tool}", tool in rd)

    print("\nA result can say whether it is still comparable\n")

    # Every activation result carries a commit and nothing said whether it
    # still counted after a release. The question was answered once by hashing
    # trees by hand, which nobody repeats -- so the fallback was treating every
    # result as stale on every release, discarding most of the evidence base
    # (issue 100).
    import sys as _sy
    _sy.path.insert(0, str(ROOT / "tools"))
    import _surface as _SF

    fails += check("the surface is defined once and shared",
                   (ROOT / "tools" / "_surface.py").is_file()
                   and "_surface" in (ROOT / "conformance" / "skill.py").read_text(encoding="utf-8"),
                   "a second copy of 'the agent surface' drifts from the first")
    here = _SF.files_at("HEAD")
    fails += check(f"it resolves to real files ({len(here)})",
                   len(here) >= 4 and "SKILL.md" in here,
                   f"{here[:5]} -- an empty surface would report every release comparable")
    fails += check("it is glob-based, so a new workflow page is surface the day it lands",
                   any("*" in g for g in _SF.SURFACE_GLOBS),
                   "a hardcoded file list would be wrong exactly when it mattered")

    # The distinction the tool exists for, driven through the real function on
    # synthetic content rather than asserted about prose.
    import tempfile as _tf, subprocess as _sp
    with _tf.TemporaryDirectory() as td:
        r = Path(td) / "r"
        r.mkdir()
        _sp.run(["git", "init", "-q", str(r)], capture_output=True)

        def commit(desc, body, tag):
            (r / "SKILL.md").write_text(f"---\nname: x\ndescription: {desc}\n---\n\n{body}\n")
            _sp.run(["git", "-C", str(r), "add", "-A"], capture_output=True)
            _sp.run(["git", "-C", str(r), "-c", "user.email=t@t", "-c", "user.name=t",
                     "commit", "-q", "-m", tag], capture_output=True)
            _sp.run(["git", "-C", str(r), "tag", tag], capture_output=True)

        commit("decide authorization", "one", "t1")
        commit("decide authorization", "two", "t2")     # body only
        commit("something else entirely", "two", "t3")  # description

        real = _SF._git
        _SF._git = lambda *a, **k: _sp.run(["git", "-C", str(r), *a],
                                           capture_output=True, text=True).stdout
        try:
            body_only = _SF.compare("t1", "t2")
            fails += check("a body change leaves activation rates comparable",
                           body_only["activation_comparable"] is True,
                           "the description is the surface a model judges; the body is "
                           "what it reads afterwards")
            fails += check("and marks grades as possibly shifted",
                           body_only["grades_comparable"] is False)
            desc = _SF.compare("t2", "t3")
            fails += check("a DESCRIPTION change makes activation results stale",
                           desc["activation_comparable"] is False,
                           "this is the case the whole tool exists to catch")
            same = _SF.compare("t1", "t1")
            fails += check("control: a ref against itself is comparable both ways",
                           same["activation_comparable"] and same["grades_comparable"],
                           "otherwise the checks above pass for any pair at all")
        finally:
            _SF._git = real

    print("\nThe two activation instruments cannot be reported as one number\n")

    # selftest runs four prompts with a parameterized first; the protocol runs
    # twenty per arm with fixed wording. The intake form asked only for "the
    # rate", so a 3/3 and a 3/20 landed in the same field and nothing
    # downstream could tell them apart -- while those results ARE the evidence
    # base for issues 5, 42 and the model comparison (issue 89).
    form = ROOT / ".github" / "ISSUE_TEMPLATE" / "activation-result.yml"
    fails += check("the activation-result form exists", form.is_file())
    if form.is_file():
        f = form.read_text(encoding="utf-8")
        fails += check("the form asks which instrument was run",
                       "id: instrument" in f,
                       "without it a self-test score and a protocol rate pool into "
                       "one field and the difference is unrecoverable")
        fails += check("and requires it",
                       f.split("id: instrument", 1)[1].split("- type:", 1)[0].count("required: true") == 1,
                       "an optional instrument field is the same defect with extra steps")
        fails += check("it offers both instruments as choices",
                       "selftest.py" in f and "Arm A" in f and "Arm B" in f)
        # The surface issue 86 missed: the hook precondition reached the
        # recording sheet, the protocol and selftest, but not the form -- which
        # is the intake for the people least likely to know the rule exists.
        fails += check("the form carries the no-hook precondition",
                       "No hook was installed" in f,
                       "issue 86 fixed the surfaces the maintainer reads and missed "
                       "the one strangers read")

    st = (ROOT / "tools" / "selftest.py").read_text(encoding="utf-8")
    fails += check("selftest says its score is not a protocol rate",
                   "NOT A PROTOCOL RATE" in st,
                   "a user scoring 3/3 has every reason to report it as one")

    print("\nAn integration contract exists, and leads with the rule that matters\n")

    # The product is tool-independent by thesis, which only holds if somebody
    # else can wire in a tracker without touching the engine. The contract was
    # scattered across storage-backends.md (decision_history-shaped), ADR-003,
    # ADR-008 and the suites, and the single most important rule -- that an
    # integration carries governance through rather than reinventing it -- was
    # written down nowhere.
    integ = ROOT / "references" / "integrations.md"
    fails += check("references/integrations.md exists", integ.is_file())
    if integ.is_file():
        text = integ.read_text(encoding="utf-8")
        fails += check("it states the pass-through rule",
                       "does not reinvent it" in text.lower()
                       or "carries governance through" in text.lower(),
                       "an integration that reinterprets state is the failure Layer 2 "
                       "exists to catch, and it must be named first")
        fails += check("it names the 'task says READY' failure specifically",
                       "Ready, so I should work on it" in text,
                       "the abstract rule is forgettable; the concrete failure is not")
        fails += check("it cites INV-002 (admission is not authorization)",
                       "INV-002" in text)
        fails += check("it names both conformance layers as the bar",
                       "layer1.py" in text and "layer2.py" in text,
                       "a contract checked by review rather than by suite is a "
                       "contract nobody outside this repository can meet")
        fails += check("it describes all three integration tiers",
                       all(k in text for k in ("filesystem", "Dolt", "Beads")))
        # This asserted "not admitted" until #196 was decided GO and the
        # adapter shipped. A suite that encodes a VERDICT goes wrong the moment
        # the verdict changes, and it went wrong here in the safe direction --
        # it failed rather than vouching for a claim that had become false.
        #
        # What survives is the RULE, which is not about Beads: a documented
        # contract must never read as a decision to build it, and a no-go must
        # stay a successful outcome for the next tracker. Assert that instead
        # of the verdict, so this check does not need revisiting per adapter.
        fails += check("it states the necessity rule that governs a new tracker",
                       "no-go" in text.lower() and "necessary" in text.lower(),
                       "documenting a contract must not read as deciding to build it, and "
                       "the rule has to outlive whichever way any one decision went")
        fails += check("providers.md points at it",
                       "integrations.md" in (ROOT / "references" / "providers.md").read_text(encoding="utf-8"),
                       "the roles table must not be the only thing an implementer finds")

    print("\nThe lanes point at the governed repository's own conventions\n")

    # Repo Governor rules on authority and says nothing about house style --
    # which is correct, and was silently total: no workflow page mentioned
    # CONTRIBUTING.md, AGENTS.md, commits or pull requests at all. An agent
    # got a correct verdict and no signal to branch, to run the repository's
    # own checks, or to respect its commit rules. Every one of those happened
    # HERE only because this repository's AGENTS.md points at its
    # CONTRIBUTING.md, and that chain exists nowhere else.
    for page in ("starting-work.md", "finishing-work.md"):
        text = (ROOT / "docs" / "workflows" / page).read_text(encoding="utf-8")
        fails += check(f"{page} names the governed repository's CONTRIBUTING.md",
                       "CONTRIBUTING.md" in text,
                       "the lane that starts or lands work must say whose rules apply")
        fails += check(f"{page} names its AGENTS.md too",
                       "AGENTS.md" in text,
                       "not every harness loads it automatically")
        fails += check(f"{page} says the repository is the GOVERNED one, not the skill",
                       "governed" in text.lower(),
                       "an install carries files with the same names; naming which "
                       "repository is meant is the whole point (ADR-027)")

    # --- the install instructions must name a tag that exists -------------
    #
    # Three places carry the version and NOTHING asserted they agree:
    # engine/version.py's ENGINE_VERSION, and a `git clone --branch vX.Y.Z` in
    # both README.md and docs/installation.md. A README telling a new user to
    # clone a tag that was never cut is a first-run failure, and it is the same
    # shape as the ADR-convention and suite-count drift this suite already
    # catches -- two places that must agree, with nothing asserting it.
    #
    # DERIVED from ENGINE_VERSION, never restated. A check that hard-codes the
    # version is the third place to forget at the next release.
    print("\nThe install instructions name the version the engine reports\n")
    import re as _vre
    vsrc = (ROOT / "engine" / "version.py").read_text()
    m = _vre.search(r'ENGINE_VERSION\s*=\s*"([^"]+)"', vsrc)
    fails += check("engine/version.py declares a version this suite can read", bool(m),
                   "without it the checks below pass vacuously")
    if m:
        ver = m.group(1)
        for doc in ("README.md", "docs/installation.md"):
            t = (ROOT / doc).read_text()
            tags = set(_vre.findall(r"--branch\s+v([0-9]+\.[0-9]+\.[0-9]+)", t))
            fails += check(f"{doc} clones v{ver}, the version the engine reports",
                           tags == {ver} if tags else True,
                           f"names {sorted(tags)}; engine reports {ver} — a clone "
                           "instruction pointing at a tag that does not exist is a "
                           "first-run failure")

    # --- issue 187: counts in the tiers nothing recounted ---------------------
    #
    # The count checks above cover six named files. docs/reference/, references/
    # and docs/workflows/ were outside all of them, and two claims in that zone
    # were wrong: "the seven roles" in the section map that resolves 640+ §NN
    # citations, and "nineteen values" for a vocabulary of 12 + 8.
    #
    # THE HARD PART IS NOT FINDING NUMBERS, IT IS NOT FLAGGING HISTORY.
    # docs/reference/criteria.md:146 says "seven" and is CORRECT -- it is an
    # amendment note recording what the set was before ADR-017 added an eighth
    # role. A check that greps number words and compares them to the code
    # demands that history be rewritten to match the present.
    #
    # That is this repository's oldest defect wearing a new hat: imports.py
    # refusing to grep source for imports because a string in a comment is not
    # an import; SUPERSEDED_RE matching prose ABOUT supersession. Text about a
    # fact is not the fact.
    #
    # So the rule is NOUN ADJACENCY: only a number bound directly to a noun the
    # engine can count is checked. "eight roles" is a claim; "this minimum set
    # is seven" is prose about a claim, and is left alone. Anything ambiguous
    # goes unchecked rather than being flagged -- unchecked is not wrong, and a
    # check that fails on correct history teaches people to weaken it.
    print("\nCounts in the reference tiers agree with the engine\n")

    # Loaded BY PATH, not by name. `import manifest` resolves to
    # conformance/manifest.py -- the suite, not the engine -- because
    # conformance/ is first on sys.path. It failed loudly here; the same
    # collision silently rebound a name in two suites once (issue 67).
    import importlib.util as _ilu  # noqa: PLC0415
    _spec = _ilu.spec_from_file_location("_engine_manifest", ROOT / "engine" / "manifest.py")
    _MF = _ilu.module_from_spec(_spec)
    _sys_path_added = str(ROOT / "engine")
    if _sys_path_added not in sys.path:
        sys.path.insert(0, _sys_path_added)
    _spec.loader.exec_module(_MF)
    NUMWORDS = {"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
                "seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11,
                "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15,
                "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19,
                "twenty": 20}
    # Only nouns the ENGINE can count. Deliberately excludes "categories" and
    # "values": both appear in immutable ADR history, and a rule that relies on
    # a path exclusion to avoid flagging them is one refactor from breaking.
    COUNTABLE = {
        "roles": len(_MF.SCALAR_ROLES) + len(_MF.ARRAY_ROLES),
        "invariants": len(set(re.findall(
            r"INV-0\d\d", (ROOT / "docs" / "reference" / "invariants.md").read_text()))),
    }
    CLAIM = re.compile(r"\*{0,2}(" + "|".join(NUMWORDS) + r")\*{0,2}\s+("
                        + "|".join(COUNTABLE) + r")\b", re.I)

    TIERS = ("docs/reference/*.md", "references/*.md", "docs/workflows/*.md")
    by_tier = {g: sorted(ROOT.glob(g)) for g in TIERS}
    tiers = sorted(f for fs in by_tier.values() for f in fs)
    # EACH tier, not the total. A single broken glob still left 16 of 26 files,
    # which cleared a >= 15 threshold and hid the breakage.
    empty = [g for g, fs in by_tier.items() if len(fs) < 3]
    fails += check(f"every reference tier was actually scanned ({len(tiers)} files)",
                   not empty,
                   f"matched almost nothing: {empty} -- a glob that finds no files "
                   "passes every check below vacuously")

    def bad_claims(text, where):
        """(mismatches, claims_seen) for one blob. ONE comparator, two callers.

        The control below feeds this the same function the file scan uses. An
        earlier version re-implemented the comparison inline, so a mutation
        disabling the real loop left the control passing -- a control that tests
        a COPY of the logic tests nothing about the logic.
        """
        bad, n = [], 0
        for i, line in enumerate(text.splitlines(), 1):
            for m in CLAIM.finditer(line):
                n += 1
                got, noun = NUMWORDS[m.group(1).lower()], m.group(2).lower()
                if got != COUNTABLE[noun]:
                    bad.append(f"{where}:{i} says {got} {noun}, "
                               f"engine says {COUNTABLE[noun]}")
        return bad, n

    wrong, seen = [], 0
    for f in tiers:
        b, n = bad_claims(f.read_text(), f.relative_to(ROOT))
        wrong += b
        seen += n
    fails += check(f"every bound count in the reference tiers is right ({seen} claim(s))",
                   not wrong, "; ".join(wrong))
    # Without this the check above passes by matching nothing -- and the zone is
    # consistent today, so a broken matcher and a working one look identical.
    fails += check("and the matcher found claims to check", seen >= 3,
                   f"only {seen} bound count(s) matched; the pattern has probably stopped "
                   "matching rather than the claims having gone away")

    # POSITIVE CONTROL. Every real claim in this zone is correct today, so a
    # comparator that never reports a mismatch looks exactly like one that
    # works -- a mutation disabling the comparison survived until this existed.
    # Feed it a claim that is known wrong and require it to say so.
    probe_n = COUNTABLE["roles"] + 1
    probe_word = [w for w, v in NUMWORDS.items() if v == probe_n]
    caught, _ = bad_claims(f"the {probe_word[0]} roles this repository does not have",
                           "<control>") if probe_word else ([], 0)
    fails += check("control: a deliberately wrong claim IS reported",
                   bool(caught),
                   f"a synthetic '{probe_n} roles' claim was not flagged, so the "
                   "comparison above proves nothing about the real files")

    print("\nThe conformance suite set is one fact, not six\n")

    # Same defect as the ADR counts above, different noun. The suite count has
    # been published as 7, 10, 11 and "eleven" simultaneously while disk held
    # 12. A count is a DERIVABLE fact, and a derivable fact restated in prose
    # is duplicated state that eventually disagrees.
    #
    # Four sources must agree, not two. Checking the filesystem against prose
    # alone would go vacuous the moment the pasted loops are replaced by a call
    # to tools/run-conformance.sh -- the runner deletes the very enumeration
    # the check inspects. So the RUNNER's own arrays are the third source, and
    # they are what catches a new suite file nobody wired in.
    #
    # The fourth is the LIVE WORKFLOW, which names its suites in its own step
    # title. Added after adding `roadmap` to LIVE left that title reading
    # "(hooks, install)" -- a job whose name lies about what it ran, which is
    # the failure the workflow's own header comment warns about: "a job named
    # only for `hooks` reporting `FAILED: install` teaches whoever sees it that
    # live red means someone moved a card".
    truth = {f.stem for f in sorted((ROOT / "conformance").glob("[a-z]*.py"))
             if "sys.exit(main(" in f.read_text(encoding="utf-8")}

    # Floor control. If the glob breaks, every regex below also finds nothing
    # and the whole section would pass having established nothing.
    fails += check(f"the suite set was actually derived ({len(truth)} found)",
                   len(truth) >= 8 and "layer1" in truth,
                   f"{sorted(truth)} -- a broken glob reads as a clean repository")

    runner = ROOT / "tools" / "run-conformance.sh"
    fails += check("a single runner exists for CI and contributors to share",
                   runner.is_file(),
                   "without one, every document pastes its own loop and they drift")
    if runner.is_file():
        rsrc = runner.read_text(encoding="utf-8")
        declared = set()
        for arr in re.finditer(r"^(?:HERMETIC|LIVE)=\(([^)]*)\)", rsrc, re.M):
            declared |= set(arr.group(1).split())
        fails += check("the runner names every suite on disk, and no others",
                       declared == truth,
                       f"runner-only={sorted(declared - truth)} "
                       f"disk-only={sorted(truth - declared)} -- a suite the runner "
                       "does not name never runs, and nothing else would notice")

    # The live workflow's step title is the fourth source. It is prose about a
    # set the runner owns, so it drifts the moment LIVE changes.
    wf = ROOT / ".github" / "workflows" / "conformance-live.yml"
    if runner.is_file() and wf.is_file():
        live = set()
        m = re.search(r"^LIVE=\(([^)]*)\)", rsrc, re.M)
        if m:
            live = set(m.group(1).split())
        wsrc = wf.read_text(encoding="utf-8")
        title = re.search(r"name:\s*live conformance\s*\(([^)]*)\)", wsrc)
        named = {t.strip() for t in title.group(1).split(",")} if title else set()
        fails += check("the live workflow's step names exactly the LIVE suites",
                       bool(live) and named == live,
                       f"LIVE={sorted(live)} step-title={sorted(named)} -- a job "
                       "whose name lies about what it ran is how a live failure "
                       "gets read as the wrong kind of failure")

    # Any loop still pasted into prose is a second source of truth. The runner
    # replaced them; one surviving is drift waiting to happen.
    for doc in ("AGENTS.md", "CONTRIBUTING.md", "docs/installation.md",
                "docs/adrs/README.md", ".github/PULL_REQUEST_TEMPLATE.md"):
        f = ROOT / doc
        if not f.is_file():
            continue
        text = f.read_text(encoding="utf-8")
        stale = re.search(r"for s in\s+layer1[^\n]*", text)
        fails += check(f"{doc} pastes no suite loop of its own",
                       not stale,
                       (stale.group(0)[:80] if stale else "") +
                       " -- call tools/run-conformance.sh instead")

    # Adapters drift exactly the same way, and did: adding one made three
    # README claims stale at once (12 adapters, 12 provider adapters, 136
    # checks) and nothing noticed. Same derivation shape as above -- recount
    # ground truth, then check every prose claim against it -- deliberately
    # sharing this section rather than adding a second implementation, since
    # two of "recount, then compare" would themselves eventually disagree.
    adapters = {p.name for p in (ROOT / "adapters").iterdir()
                if p.is_file() and not p.name.startswith("_")
                and not p.name.endswith(".pyc")}
    fails += check(f"the adapter set was actually derived ({len(adapters)} found)",
                   len(adapters) >= 8 and "git" in adapters,
                   f"{sorted(adapters)} -- a broken glob reads as a clean repository")

    # RATIFICATION-v0.1.0.md states the release condition:
    #
    #   Every architecture decision the runtime depends on is Accepted, and no
    #   Proposed ADR is silently treated as normative by that release.
    #
    # docs/adrs/README.md asserted the first half -- "none is referenced by the
    # runtime" -- and it had been false since #153 and #183, through four
    # releases, with every conformance run green. The condition is checkable
    # and nothing was checking it; that is the same shape as the adapter count
    # below and the contract-check count above.
    #
    # Derived by the record's OWN method: cited by engine/, or by an adapter
    # THIS repository binds. Unbound adapters ship but govern nothing here.
    bound = set()
    for _k, _v in (json.loads((ROOT / ".repo-governor.json").read_text(encoding="utf-8"))
                   .get("providers") or {}).items():
        if _k.startswith("$"):
            continue
        for _b in (_v if isinstance(_v, list) else [_v]):
            bound.add(_b["adapter"])
    fails += check(f"the bound-adapter set was derived ({len(bound)})", len(bound) >= 3,
                   "an empty set makes every ADR read as runtime-independent")

    proposed, dependent = [], []
    for adr in sorted((ROOT / "docs" / "adrs").glob("[0-9]*.md")):
        body = adr.read_text(encoding="utf-8")
        st = re.search(r"^\*\*Status\*\*:\s*(\S+)", body, re.M)
        if not (st and st.group(1).strip("*").startswith("Proposed")):
            continue
        num = re.match(r"(\d+)", adr.name).group(1)
        proposed.append(num)
        cited = any(f"ADR-{num}" in f.read_text(encoding="utf-8")
                    for f in (ROOT / "engine").glob("*.py"))
        cited = cited or any(f"ADR-{num}" in (ROOT / a).read_text(encoding="utf-8")
                             for a in bound)
        if cited:
            dependent.append(num)
    fails += check(f"proposed ADRs were actually found ({len(proposed)})", bool(proposed),
                   "a status regex that matches nothing reads as an all-Accepted repository")

    # THE SAME FALSE SENTENCE LIVED IN TWO FILES. The guard below reads
    # docs/adrs/README.md, and the top-level README carried its own copy of
    # "none referenced by the runtime" -- which stayed green because nothing
    # read that file for this claim. A guard scoped to one of two places that
    # must agree is the defect it was written against, one level up.
    #
    # So: no document may assert the runtime depends on none of them while the
    # derived set is non-empty. A blunt check, on purpose.
    NO_DEP = re.compile(r"none[^.\n]{0,40}referenced by the runtime", re.I)
    liars = [d for d in ("README.md", "AGENTS.md", "SKILL.md", "docs/adrs/README.md",
                         "docs/installation.md", "CONTRIBUTING.md")
             if (ROOT / d).is_file() and NO_DEP.search((ROOT / d).read_text(encoding="utf-8"))]
    fails += check("no document claims the runtime depends on no proposed ADR",
                   not (dependent and liars),
                   f"{liars} say none is referenced by the runtime, while "
                   f"{sorted(dependent)} are")

    idx = (ROOT / "docs" / "adrs" / "README.md").read_text(encoding="utf-8")
    m_dep = re.search(r"Runtime-dependent and still `Proposed`:\s*([^*]+)", idx)
    fails += check("the ADR index states which proposed ADRs the runtime depends on",
                   bool(m_dep),
                   "the release condition is about exactly this set; a reader cannot "
                   "check a claim the index does not make")
    if m_dep:
        claimed = sorted(set(re.findall(r"ADR-(\d+)", m_dep.group(1))))
        fails += check(f"and that list matches the code ({', '.join(dependent) or 'none'})",
                       claimed == sorted(dependent),
                       f"index says {claimed}, code says {sorted(dependent)} -- "
                       "RATIFICATION-v0.1.0.md: every architecture decision the runtime "
                       "depends on is Accepted. A new name here is a ratification "
                       "question, not a documentation edit.")
    # The adapter COUNT was guarded; the contract-check count beside it was
    # not, and had been stale through two suite growths (149 claimed, 184 then
    # 199 actual) with every run green. A number nothing recomputes is a
    # number that drifts silently -- the same shape as the adapter claim two
    # lines down, which is why it gets the same treatment rather than a
    # comment asking the next person to remember.
    L1_CLAIM = re.compile(r"(\d+)\s+contract checks\b", re.I)
    l1_claims = []
    for doc in ("README.md", "AGENTS.md", "docs/installation.md"):
        f = ROOT / doc
        if f.is_file():
            l1_claims += [(doc, int(m.group(1))) for m in L1_CLAIM.finditer(
                f.read_text(encoding="utf-8"))]
    if l1_claims:
        p1 = subprocess.run([sys.executable, str(ROOT / "conformance" / "layer1.py")],
                            capture_output=True, text=True, cwd=str(ROOT), timeout=300)
        m1 = re.search(r"executed=(\d+)", p1.stdout)
        fails += check("layer1's own check count was actually read",
                       bool(m1), f"no CONFORMANCE-COUNT in layer1 output: {p1.stdout[-160:]!r}")
        if m1:
            real = int(m1.group(1))
            # THE COUNT IS NOT ENVIRONMENT-INDEPENDENT, and the first version of
            # this guard assumed it was. Without `dolt` on PATH the dolt
            # adapter's probe fails, five capability checks do not run, and
            # layer1 reports 194 rather than 199 -- so CI's `static` job, which
            # exists precisely because it "needs no adapter dependency at all",
            # failed on a claim that is true on a developer machine.
            #
            # The honest repair is not to weaken the check. It is to say in the
            # README which environment the number describes, and to compare
            # exactly THERE. Where a dependency is absent the number is a
            # ceiling the environment cannot reach, and that is still a real
            # assertion: a claim smaller than what actually ran is wrong in
            # either environment.
            absent = _preflight.missing()
            for doc, n in l1_claims:
                if absent:
                    fails += check(
                        f"{doc}: '{n} contract checks' is a ceiling this environment "
                        f"({real} ran, {', '.join(sorted(absent))} absent) does not exceed",
                        real <= n,
                        f"layer1 ran {real} with {sorted(absent)} missing, which is MORE "
                        f"than the {n} claimed for a complete environment")
                else:
                    fails += check(f"{doc}: '{n} contract checks' matches the {real} "
                                   "layer1 runs", n == real, f"layer1 runs {real}")

    ADAPTER_CLAIM = re.compile(r"(\d+)\s+(?:provider\s+)?adapters?\b", re.I)
    adapter_claims = 0
    for doc in ("README.md", "AGENTS.md", "CONTRIBUTING.md", "docs/installation.md"):
        f = ROOT / doc
        if not f.is_file():
            continue
        for m in ADAPTER_CLAIM.finditer(f.read_text(encoding="utf-8")):
            adapter_claims += 1
            fails += check(f"{doc}: '{m.group(0).strip()}' matches the {len(adapters)} on disk",
                           int(m.group(1)) == len(adapters),
                           f"disk has {len(adapters)}")
    fails += check("at least one document states an adapter count",
                   adapter_claims > 0,
                   "found 0 -- either every count was removed or the pattern went blind")

    # Counts, anchored on the word so that "ADR-011" is not read as an 11.
    WORDS = {"seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11,
             "twelve": 12, "thirteen": 13}
    COUNT = re.compile(r"(\d+|" + "|".join(WORDS) + r")\s+(?:conformance\s+)?suites?\b"
                       r"|(\d+)\s+of\s+(\d+)\s+conformance\s+suites?", re.I)
    # A count claim the pattern cannot see is indistinguishable from no claim
    # at all, and that is how CONTRIBUTING.md came to say "must be green,
    # 10/10" eighteen lines after saying 16/16, with this check green.
    #
    # TWO GENERAL FIXES WERE TRIED AND BOTH FAILED, recorded so the next person
    # does not spend the afternoon rediscovering it:
    #
    #   Every bare N/N -> three false positives in README.md immediately. This
    #   repository writes activation rates that way ("20/20 on one host and 0/2
    #   on another"), and the exemption list it needed wants exactly the
    #   maintenance the phrasing list wants. Same defect, opposite costume.
    #
    #   N/N near context words -> docs/adrs/README.md writes 29/29 and 112/112
    #   about ASSERTIONS INSIDE a suite, and the word "conformance" is in the
    #   file path beside them. Broadening to "green|pass" made it worse.
    #
    # N/N genuinely means three things here: suites, assertions within a suite,
    # and activation rates. No rule over prose separates them, so the
    # enumeration stays -- with the phrasing that actually went stale added,
    # and with the blind spot stated rather than implied.
    RATIO_PHRASINGS = (
        r"(\d+)\s*/\s*(\d+)\s+pass",
        r"Expect\s+\**(\d+)\s*/\s*(\d+)",
        r"green\**\W{0,4}(\d+)\s*/\s*(\d+)",   # "must be green, 16/16," -- the stale one
    )
    RATIO = re.compile("|".join(RATIO_PHRASINGS), re.I)

    total_claims = 0
    for doc in ("README.md", "AGENTS.md", "CONTRIBUTING.md", "docs/installation.md",
                "docs/adrs/README.md", ".github/PULL_REQUEST_TEMPLATE.md"):
        f = ROOT / doc
        if not f.is_file():
            continue
        text = f.read_text(encoding="utf-8")
        claims = []
        for m in COUNT.finditer(text):
            raw = [g for g in m.groups() if g]
            # "4 of 12 conformance suites fail" -- the TOTAL is the last number.
            val = raw[-1]
            n = WORDS.get(val.lower(), None) if not val.isdigit() else int(val)
            if n is not None:
                claims.append((m.group(0).strip(), n))
        for m in RATIO.finditer(text):
            nums = [g for g in m.groups() if g]
            claims.append((m.group(0).strip(), int(nums[-1])))
        for label, n in claims:
            total_claims += 1
            fails += check(f"{doc}: '{label}' matches the {len(truth)} on disk",
                           n == len(truth),
                           f"disk has {len(truth)}: {sorted(truth)}")
        # Two claims in ONE file contradicting each other is what happened, and
        # no cross-document check would have caught it however good the
        # pattern: both were compared against disk, and only one was wrong.
        distinct = {n for _, n in claims}
        fails += check(f"{doc}: states one suite count, not several",
                       len(distinct) <= 1,
                       f"claims {sorted(distinct)} in the same file: "
                       f"{[c for c, _ in claims]}")
    fails += check("at least one document still states a suite count",
                   total_claims > 0,
                   "found 0 -- either every count was removed, or the pattern went "
                   "blind; a check that matches nothing is not a passing check")

    print(f"\n{'AGENT SURFACE: CONFORMANT' if not fails else f'AGENT SURFACE: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
