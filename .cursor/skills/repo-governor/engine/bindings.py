#!/usr/bin/env python3
"""Provider resolution and the permission chokepoint (ADR-021, implements ADR-005 rule 2).

Every adapter invocation the engine makes passes through `call()`. Nothing else
in `engine/` may spawn an adapter, and `conformance/bindings.py` asserts it by
refusing any `adapters/` path literal elsewhere in this package.

Why this module exists at all: ADR-005 rule 2 specified

    "the permission gate as a single chokepoint every adapter invocation passes
     through -- no adapter may be called except through it"

and it was never built. `engine/completion.py` spawned adapters directly, so
`permitted()` was consulted for exactly one write and for none of the reads.
The manifest was load-bearing for a single operation and decorative for the
rest. That is the gap this closes.

Three rules follow from it:

  1. The engine names ROLES, never adapters. A path literal here is a binding
     decision made in code, which is the thing INV-013 forbids.
  2. Configuration reaches an adapter from the manifest binding, never from
     engine code. `if adapter == "adapters/linear"` is the engine knowing a
     vendor, which is the abstraction leaking in the direction that matters.
  3. A denied or unbound call returns a typed envelope, not an exception
     (ADR-005 rule 5), so a permission shortfall reads as a disposition rather
     than a crash -- and never as a silent skip that makes the decision look
     complete when it is not.

Usage:  python3 engine/bindings.py <role> [verb]      explain a resolution
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import manifest as MF  # noqa: E402

# ROOT is where the ENGINE lives -- it resolves adapter paths and nothing else.
# TARGET is the repository being GOVERNED. Conflating the two meant every
# repo-local provider read the engine's own repository whatever it was pointed
# at, and said so with provenance that looked correct (#24, ADR-027).
ROOT = Path(__file__).resolve().parent.parent

# A subprocess verb maps to the permission verb it needs. `query` is a read;
# `write` is a write. Nothing infers a verb from the function name.
VERB_PERMISSION = {"query": "read", "write": "write", "describe": "read"}


# Target resolution lives in manifest.py, so the manifest and the adapters
# cannot disagree about which repository is being governed.
target = MF.target


def _subject(t):
    """A stable name for the governed repository, for provenance qualification."""
    p = subprocess.run(["git", "-C", str(t), "remote", "get-url", "origin"],
                       capture_output=True, text=True, timeout=20)
    url = p.stdout.strip()
    if p.returncode == 0 and url:
        slug = re.sub(r"^.*[:/]([^/:]+/[^/]+?)(?:\.git)?$", r"\1", url)
        if slug and slug != url:
            return slug
    return t.name

_DESCRIBE_CACHE: dict[str, dict] = {}


def _fail(role, fn, etype, message):
    """The same envelope shape an adapter returns, so callers need no special case."""
    return {"ok": False, "role": role, "function": fn,
            "error": {"type": etype, "message": message}}


def resolve(role, manifest=None):
    """Return (bindings, error_envelope). `bindings` is always a list."""
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return None, _fail(role, None, "MANIFEST_INVALID",
                               f"refusing to evaluate: {errs[0]}")
    b = (m.get("providers") or {}).get(role)
    if b is None:
        return None, _fail(role, None, "UNBOUND_ROLE",
                           f"role {role!r} has no binding in the manifest; "
                           "a detected, reachable provider absent from the manifest "
                           "has no role (INV-013)")
    return (b if isinstance(b, list) else [b]), None


def _env_for(binding):
    """Adapter configuration comes from the manifest, never from engine code.

    Two channels, both declared by a human:

      `env`                     verbatim adapter-specific variables
      REPO_GOVERNOR_BINDING     the whole binding as JSON, for adapters that
                                read structured config (admission signals, paths)

    The engine sets no adapter-specific variable of its own. That is what stops
    `if adapter == "adapters/linear"` growing back.
    """
    env = dict(os.environ)
    for k, v in (binding.get("env") or {}).items():
        env[str(k)] = str(v)
    env["REPO_GOVERNOR_BINDING"] = json.dumps(binding, sort_keys=True)
    t = target()
    # REPO_GOVERNOR_TARGET is deliberately NOT exported. It is the ENGINE's
    # input, not adapter configuration: no adapter reads it, and they resolve
    # the repository from `cwd` -- which _spawn sets -- and from
    # REPO_GOVERNOR_SUBJECT below, which _protocol.py does read.
    #
    # Exporting it anyway meant every grandchild inherited it. An adapter that
    # runs a command -- a command_exit acceptance criterion, or a hook firing
    # inside an adapter subprocess -- would run the engine again, and the
    # engine would resolve the INHERITED target rather than where it stands.
    # The hook then announced a repository it was not in, which is ADR-027's
    # failure through a channel ADR-027 did not consider (issue 54).
    #
    # A user who exports this in their own shell is still honoured. That is a
    # declaration, and it is how ADR-027 targeting is meant to work. What is
    # removed is the engine propagating its own input to processes that never
    # asked for it.
    env["REPO_GOVERNOR_SUBJECT"] = _subject(t)
    return env


def _spawn(binding, role, fn, kw, verb):
    # cwd is the GOVERNED repository, not the engine's install directory. Every
    # repo-local adapter resolves its default path relative to cwd, which is
    # correct; pinning cwd to ROOT overrode all seven of them at once (#24).
    args = [sys.executable, str(ROOT / binding["adapter"]), verb, role, fn]
    args += [f"{k}={v}" for k, v in kw.items()]
    p = subprocess.run(args, capture_output=True, text=True, cwd=str(target()),
                       env=_env_for(binding), timeout=310)
    try:
        return json.loads(p.stdout)
    except json.JSONDecodeError:
        return _fail(role, fn, "NON_JSON", (p.stdout or p.stderr)[:160])


def call(role, fn, kw=None, verb="query", manifest=None, binding=None):
    """Resolve the role, check the permission, invoke. The only way in.

    The permission check happens BEFORE the subprocess. A denied call must not
    reach the provider at all -- checking afterwards would make the denial
    advisory, and the provider would already have been contacted.
    """
    need = VERB_PERMISSION.get(verb)
    if need is None:
        return _fail(role, fn, "BAD_REQUEST", f"unknown adapter verb {verb!r}")

    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return _fail(role, fn, "MANIFEST_INVALID", f"refusing to evaluate: {errs[0]}")

    allowed, why = MF.permitted(m, role, need)
    if not allowed:
        return _fail(role, fn, "PERMISSION_DENIED",
                     f"{role}.{need} is not granted: {why}")

    if binding is None:
        bindings, err = resolve(role, m)
        if err:
            return err
        binding = bindings[0]

    return _spawn(binding, role, fn, kw or {}, verb)


def call_all(role, fn, kw=None, verb="query", manifest=None):
    """Invoke EVERY binding on a role. Returns (results, error_envelope).

    ADR-013 Implementation Plan step 4, which was never built. `call()` ends
    with `bindings[0]`, and that is correct for a single-valued role and
    silently discards evidence on a multi-valued one. The cost was not
    theoretical: `.repo-governor.json` binds two `decision_history` providers,
    the second one deliberately -- its own manifest comment says "GitHub
    already records decisions in stateReason and nothing was reading them" --
    and it had still never been read. §39 rediscovery was being decided against
    one of two stores.

    `call()` is deliberately NOT reimplemented in terms of this. A role with one
    binding must produce the byte-identical envelope it produced before (ADR-008
    C7), and the cheapest way to guarantee that is for the single path to be
    the same code it always was. Callers that want the union ask for it.

    Each result is `{"binding": <the binding>, "envelope": <adapter reply>}`.
    A provider that fails contributes its typed failure rather than vanishing:
    ADR-003 rule 6 forbids conflating absence with unknown, and a union that
    quietly drops an unreachable provider reports a smaller architecture as
    though it were the whole one.
    """
    need = VERB_PERMISSION.get(verb)
    if need is None:
        return None, _fail(role, fn, "BAD_REQUEST", f"unknown adapter verb {verb!r}")

    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return None, _fail(role, fn, "MANIFEST_INVALID", f"refusing to evaluate: {errs[0]}")

    # The permission check happens ONCE, before any subprocess, for the same
    # reason it does in call(): a denied call must not reach a provider at all.
    # Checking per binding would be the same rule applied N times, which reads
    # as thoroughness and is really an invitation to check it zero times on the
    # path someone adds later.
    allowed, why = MF.permitted(m, role, need)
    if not allowed:
        return None, _fail(role, fn, "PERMISSION_DENIED",
                           f"{role}.{need} is not granted: {why}")

    bindings, err = resolve(role, m)
    if err:
        return None, err
    return [{"binding": b, "envelope": _spawn(b, role, fn, kw or {}, verb)}
            for b in bindings], None


def describe(binding, use_cache=True):
    """Read an adapter's advertised contract. Cached per process."""
    key = binding["adapter"]
    if use_cache and key in _DESCRIBE_CACHE:
        return _DESCRIBE_CACHE[key]
    p = subprocess.run([sys.executable, str(ROOT / key), "describe"],
                       capture_output=True, text=True, cwd=str(target()),
                       env=_env_for(binding), timeout=60)
    try:
        d = json.loads(p.stdout)
    except json.JSONDecodeError:
        d = {}
    _DESCRIBE_CACHE[key] = d
    return d


def writer_for(role, fn, manifest=None):
    """Pick a binding that actually advertises `fn` as a writer.

    Selection is by advertised capability, never by product name. The previous
    implementation asked whether the adapter type started with a particular
    vendor string, which meant a second writable backend could never be chosen
    and a renamed one would silently stop being. `describe` already reports a
    `writers` list gated on a real writability probe (#17); this reads it.
    """
    m = manifest
    if m is None:
        m, errs = MF.load()
        if errs:
            return None, _fail(role, fn, "MANIFEST_INVALID", f"refusing to evaluate: {errs[0]}")
    bindings, err = resolve(role, m)
    if err:
        return None, err
    for b in bindings:
        if fn in (describe(b).get("writers") or {}):
            return b, None
    names = ", ".join(b.get("type", "?") for b in bindings) or "none"
    return None, _fail(role, fn, "NO_WRITABLE_PROVIDER",
                       f"no binding for {role!r} advertises {fn!r} as a writer "
                       f"(bound: {names})")


def main(argv):
    if not argv:
        print(__doc__.strip().splitlines()[-1], file=sys.stderr)
        return 2
    role = argv[0]
    verb = argv[1] if len(argv) > 1 else "read"
    m, errs = MF.load()
    if errs:
        print(f"MANIFEST INVALID — {errs[0]}")
        return 1
    bindings, err = resolve(role, m)
    if err:
        print(f"{err['error']['type']}: {err['error']['message']}")
        return 1
    allowed, why = MF.permitted(m, role, verb)
    print(f"role       : {role}")
    print(f"bindings   : {', '.join(b['type'] for b in bindings)}")
    print(f"{verb:<11}: {'ALLOW' if allowed else 'DENY '} — {why}")
    for b in bindings:
        d = describe(b)
        w = ", ".join(d.get("writers") or {}) or "(none)"
        print(f"  {b['type']:<26} reachable={d.get('transport', {}).get('reachable')} writers={w}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
