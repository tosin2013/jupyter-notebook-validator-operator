#!/usr/bin/env python3
"""The engine's dependency surface (ADR-011 rules 1 and 4, plan item 4).

ADR-011 has said since it was Accepted: "Add a CI check asserting the engine
imports nothing outside a stdlib allowlist, and nothing network-capable." This
is that check, and it was never written.

THREE THINGS IT MUST NOT BE, each of which this repository has actually shipped:

1. **Not a grep.** `engine/envelope.py:193`, `engine/manifest.py`, `engine/
   onboard.py` and `engine/migrate.py` all import inside function bodies or on
   a comma line (`import json, sys`). A regex over `^import` misses every one
   and reports a clean engine. This walks the AST.

2. **Not satisfied by an allowlist entry.** A module may be listed and still
   not be stdlib -- adding `import yaml` and `"yaml": "we need it"` in the same
   commit would otherwise pass. Every allowed module is RESOLVED and its origin
   checked against the stdlib path, so the allowlist cannot certify itself.

3. **Not blind to the boundary the engine actually crosses.** `engine/
   onboard.py:113` does `spec.loader.exec_module(...)` on `adapters/adr`,
   loading adapter code INTO THE ENGINE PROCESS. Rule 4 explicitly permits
   adapters to carry dependencies. So the day an adapter the engine execs grows
   an HTTP client -- allowed by rule 4 -- rule 1 breaks, and an audit that
   stopped at `engine/*.py` would report a clean bill of health. This follows
   that edge.

An unaudited grant is also a failure: a module allowlisted and imported by
nothing is permission sitting there waiting to be used, and the list is
supposed to describe the engine's dependency surface, not bless a wishlist.

Usage:  python3 conformance/imports.py
"""

from __future__ import annotations

import ast
import importlib.util
import sys
import sysconfig
from pathlib import Path

import _count as _CNT  # noqa: E402 -- alias avoids `C`, already bound to
# `completion` in two suites, where the collision silently rebound it (issue 67).
_CNT.watch("imports")


ROOT = Path(__file__).resolve().parent.parent
ENGINE = ROOT / "engine"

# Module -> why the engine is allowed to import it. A permission with no stated
# reason is not auditable, so the value is not decoration.
ALLOWED = {
    "__future__":  "annotations, for forward references without typing",
    "hashlib":     "manifest hashing for the evidence chain (ADR-009)",
    "importlib":   "loading an adapter module in-process (see EXEC_EDGE below)",
    "json":        "the canonical manifest and adapter wire format (ADR-015)",
    "os":          "environment reads for REPO_GOVERNOR_* targeting (ADR-027)",
    "pathlib":     "filesystem paths",
    "re":          "status-dialect and vocabulary parsing",
    "subprocess":  "spawning adapters -- the ADR-003 protocol IS a subprocess",
    "sys":         "argv, exit codes, stderr",
}

# Checked BEFORE the allowlist so the failure names the actual sin. `subprocess`
# is deliberately absent: it is how ADR-003 adapters are spawned at all, and
# ADR-011 rule 4 puts network capability on the adapter side of that boundary.
NETWORK = {
    "socket", "ssl", "http", "urllib", "ftplib", "smtplib", "poplib", "imaplib",
    "nntplib", "telnetlib", "xmlrpc", "socketserver", "webbrowser", "asyncio",
    "requests", "httpx", "urllib3", "aiohttp",
}

STDLIB_DIR = Path(sysconfig.get_paths()["stdlib"]).resolve()


def check(label, ok, detail=""):
    print(f"  [{'PASS' if ok else 'FAIL'}] {label}" + (f"\n         {detail}" if detail else ""))
    return 0 if ok else 1


def imports_of(path: Path):
    """Top-level package of every import anywhere in the file, nested included."""
    found = set()
    for node in ast.walk(ast.parse(path.read_text(encoding="utf-8"), str(path))):
        if isinstance(node, ast.Import):
            found |= {a.name.split(".")[0] for a in node.names}
        elif isinstance(node, ast.ImportFrom):
            # A relative import (level > 0) names no top-level package.
            if node.level == 0 and node.module:
                found.add(node.module.split(".")[0])
    return found


def exec_edges(path: Path):
    """Files this module loads into its own process via exec_module/SourceFileLoader.

    Returns (resolved_paths, unresolvable_count). A call site whose target is
    not a string literal cannot be audited statically, and that is reported as
    a failure rather than passed over -- an edge the checker cannot see is
    exactly the case this function exists for.
    """
    src = path.read_text(encoding="utf-8")
    if "exec_module" not in src and "SourceFileLoader" not in src:
        return set(), 0
    tree = ast.parse(src, str(path))
    targets, opaque = set(), 0
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        name = getattr(node.func, "attr", None) or getattr(node.func, "id", None)
        if name not in ("SourceFileLoader", "exec_module", "spec_from_loader"):
            continue
        if name != "SourceFileLoader":
            continue
        # The real call site builds its path as
        #     Path(__file__).resolve().parent.parent / "adapters" / "adr"
        # so there is no single string literal to read. The segments ARE
        # literals, though, so the path is statically reconstructible -- try
        # every contiguous run of string constants in the call subtree against
        # the repository root. Requiring one literal argument would report this
        # edge as unauditable and be wrong; accepting any call without proof
        # would report it as audited and be worse.
        # Sorted by SOURCE POSITION, not ast.walk order. walk() is
        # breadth-first, and `parent / "adapters" / "adr"` nests as
        # BinOp(BinOp(x, "adapters"), "adr") -- so walk yields "adr" BEFORE
        # "adapters" and the reconstructed path is silently reversed. That
        # produced a path that did not exist, which this function would then
        # have reported as an unauditable edge: a true failure for a false
        # reason, which is worse than a clean miss.
        consts = [n.value for n in sorted(
            (n for n in ast.walk(node)
             if isinstance(n, ast.Constant) and isinstance(n.value, str)),
            key=lambda n: (n.lineno, n.col_offset))]
        hit = None
        for i in range(len(consts)):
            for j in range(len(consts), i, -1):
                cand = ROOT.joinpath(*consts[i:j])
                if cand.is_file():
                    hit = cand
                    break
            if hit:
                break
        if hit:
            targets.add(hit.resolve())
        else:
            opaque += 1
    return targets, opaque


def resolves_to_stdlib(mod):
    """(ok, detail). Built-in, or an origin under the interpreter's stdlib dir."""
    try:
        spec = importlib.util.find_spec(mod)
    except (ImportError, ValueError):
        return False, "not importable"
    if spec is None:
        return False, "no spec"
    if spec.origin in ("built-in", "frozen") or getattr(spec, "origin", None) is None:
        return True, str(spec.origin)
    origin = Path(spec.origin).resolve()
    if "site-packages" in origin.parts or "dist-packages" in origin.parts:
        return False, f"third-party: {origin}"
    try:
        origin.relative_to(STDLIB_DIR)
    except ValueError:
        return False, f"outside the stdlib: {origin}"
    return True, str(origin)


def main():
    fails = 0
    mods = sorted(ENGINE.glob("*.py"))
    local = {p.stem for p in mods}

    print("ADR-011 rule 1 -- the engine's own imports\n")

    # A scan that scanned nothing must never read as a clean engine. This is
    # the anti-vacuity floor: the check that the check ran.
    fails += check(f"the scan found engine modules to read ({len(mods)})", len(mods) >= 5,
                   "zero or few modules means the glob is broken, not that the engine is clean")

    observed = {}
    for p in mods:
        for m in imports_of(p):
            observed.setdefault(m, set()).add(p.name)
    external = {m: v for m, v in observed.items() if m not in local}

    fails += check(f"the scan found imports to audit ({len(external)} external)",
                   len(external) >= 5, "an empty import set means the AST walk is broken")

    # Positive control. `engine/envelope.py` imports hashlib INSIDE a function.
    # If this ever passes with a regex-shaped scanner, the scanner is broken --
    # so the control is the checker testing its own method, not the engine.
    ctl = imports_of(ENGINE / "envelope.py")
    fails += check("control: a function-body import is seen (envelope.py imports hashlib)",
                   "hashlib" in ctl,
                   "only an AST walk finds this; a grep over '^import' does not")

    print("\nNothing network-capable\n")
    for m in sorted(external):
        if m in NETWORK:
            fails += check(f"NETWORK_CAPABLE: {m} imported by {', '.join(sorted(observed[m]))}",
                           False, "ADR-011 rule 1: the engine reaches no network")
    fails += check("no engine module imports a network-capable stdlib package",
                   not (set(external) & NETWORK), str(sorted(set(external) & NETWORK)))

    print("\nEverything imported is allowlisted, and everything allowlisted is imported\n")
    unlisted = sorted(set(external) - set(ALLOWED))
    fails += check("no unlisted import", not unlisted,
                   f"{unlisted} -- add to ALLOWED with a reason, or remove the import")
    # The other direction. An entry nobody imports is an unaudited grant.
    unused = sorted(set(ALLOWED) - set(external))
    fails += check("no allowlist entry without an import", not unused,
                   f"{unused} -- the list describes the surface, it does not grant permission")

    print("\nAllowlisted means stdlib, verified against the interpreter\n")
    for m in sorted(set(ALLOWED) & set(external)):
        ok, detail = resolves_to_stdlib(m)
        if not ok:
            fails += check(f"NOT_STDLIB: {m}", False, detail)
    bad = [m for m in sorted(set(ALLOWED) & set(external)) if not resolves_to_stdlib(m)[0]]
    fails += check("every allowed module resolves inside the stdlib", not bad, str(bad))

    print("\nRule 4 boundary -- adapter code the engine loads into its own process\n")
    edges, opaque = set(), 0
    for p in mods:
        e, o = exec_edges(p)
        edges |= e
        opaque += o
    fails += check("every in-process load names a literal path", opaque == 0,
                   f"{opaque} exec_module/SourceFileLoader site(s) with a computed path; "
                   "an edge that cannot be read statically cannot be audited")
    fails += check("the exec boundary is found at all", bool(edges),
                   "engine/onboard.py loads adapters/adr in-process; finding none means "
                   "the edge detector stopped working, not that the edge went away")
    for f in sorted(edges):
        rel = f.relative_to(ROOT)
        seen = imports_of(f)
        # _protocol is a sibling module of the adapters, not a package on PATH.
        ext = {m for m in seen if m not in local and m != "_protocol"}
        offend = sorted((ext - set(ALLOWED)) | (ext & NETWORK))
        fails += check(f"{rel} (exec'd by the engine) stays inside the engine's allowlist",
                       not offend,
                       f"{offend} -- ADR-011 rule 4 lets ADAPTERS carry dependencies, but "
                       "this one is executed IN the engine process, where rule 1 applies")

    print(f"\n{'IMPORTS: CONFORMANT' if not fails else f'IMPORTS: NON-CONFORMANT ({fails})'}")
    return 0 if not fails else 1


if __name__ == "__main__":
    sys.exit(main())
