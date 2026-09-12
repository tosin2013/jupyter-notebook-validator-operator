"""Shared adapter-protocol helpers for the Python adapters (ADR-003).

A Go or shell adapter would not import this — the protocol is the JSON on
stdout, not this module. It exists only to keep the Python adapters short.

Wire protocol
-------------
    $ADAPTER describe                       -> capability manifest
    $ADAPTER query <role> <function> [k=v]  -> typed response (read)
    $ADAPTER write <role> <function> [k=v]  -> typed response (mutate)

`query` and `write` are separate verbs so the read/write split ADR-005 cares
about is visible at the protocol level rather than buried in a function name.
An adapter that declares no writers rejects `write` outright. The ENGINE still
checks the manifest before calling either -- an adapter offering a writer is
capability, not permission (INV-014).

Every response is a single JSON object on stdout. Exit code 0 means "the
adapter ran"; it does NOT mean the query succeeded — check `ok`. This split
exists because ADR-008 requires an unreachable backend to produce a typed
failure rather than a plausible-looking empty result.

Determinism note (ADR-002): adapters do not stamp timestamps into provenance.
The engine stamps the decision record. Timestamps here would break the
byte-identical replay that ADR-009 depends on.
"""

from __future__ import annotations

import json
import os
import sys

CONTRACT_VERSION = 1

# Typed error vocabulary. Closed set — see ADR-007's reasoning for dispositions.
ERRORS = (
    "PROVIDER_UNAVAILABLE",     # backend unreachable / unreadable
    "NOT_FOUND",                # the object genuinely does not exist
    "UNSUPPORTED_FUNCTION",     # not part of this adapter's advertised capability
    "MALFORMED_SOURCE",         # backend reachable but its data is unparseable
    "BAD_REQUEST",              # caller error: missing/invalid argument
)


def emit(obj) -> None:
    json.dump(obj, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")


def ok(role, function, value, provenance, unknown=None):
    """A successful typed response.

    `provenance` is mandatory and must be non-empty (ADR-012): a fact with no
    citation is treated as unknown, not as true.
    """
    if not provenance:
        return fail(role, function, "MALFORMED_SOURCE", "fact produced without provenance")
    return {
        "ok": True,
        "contract_version": CONTRACT_VERSION,
        "role": role,
        "function": function,
        "value": value,
        "provenance": provenance,
        "unknown": unknown,
    }


def unknown(role, function, reason, detail, resolution, blocking=True):
    """Evidence could not be resolved. Distinct from absence (ADR-003 rule 5)."""
    return {
        "ok": True,
        "contract_version": CONTRACT_VERSION,
        "role": role,
        "function": function,
        "value": None,
        "provenance": [],
        "unknown": {
            "reason": reason,
            "detail": detail,
            "resolution": resolution,
            "blocking": blocking,
        },
    }


def fail(role, function, etype, message):
    assert etype in ERRORS, f"undeclared error type {etype}"
    return {
        "ok": False,
        "contract_version": CONTRACT_VERSION,
        "role": role,
        "function": function,
        "error": {"type": etype, "message": message},
    }


# The repository this invocation is governing. Set by engine/bindings.py from the
# resolved target (ADR-027). Absent when an adapter is driven directly, which is
# how the conformance suites run.
SUBJECT = os.environ.get("REPO_GOVERNOR_SUBJECT", "")


def binding():
    """The manifest binding for this invocation, or {} when driven directly."""
    try:
        b = json.loads(os.environ.get("REPO_GOVERNOR_BINDING", "") or "{}")
        return b if isinstance(b, dict) else {}
    except json.JSONDecodeError:
        return {}


def configured(env_var, binding_key, default):
    """Resolve a repo-local path: explicit env, then the manifest binding, then a default.

    `engine/onboard.py` detects where a provider's files actually live and writes
    that into the proposed binding as `path`. Nothing read it. A repository with
    149 decisions in `docs/adr` was detected correctly, bound correctly, and then
    read at `docs/adrs` -- which does not exist -- so it governed as
    PROVIDER_UNAVAILABLE (#27). Detection and execution were wired to different
    values, and each looked right on its own.

    Env wins over the binding so a conformance suite can drive an adapter at a
    fixture without a manifest; the binding wins over the default so a human's
    declared layout beats this project's convention.
    """
    explicit = os.environ.get(env_var)
    if explicit:
        return explicit
    return binding().get(binding_key) or default


def cite(source, ref, field=None):
    """One provenance entry. No timestamp — see module docstring.

    Refs from repo-local providers are repository-relative (`docs/adrs`,
    `filesystem:package.json`), which identifies nothing on its own. The same
    check run against two repositories produced opposite answers and
    byte-identical provenance (#24), so a decision record could not say which
    repository it was about. Qualify the ref with the governed repository
    whenever the engine has told us what that is.
    """
    if SUBJECT and not ref.startswith(SUBJECT + "//"):
        ref = f"{SUBJECT}//{ref}"
    c = {"source": source, "ref": ref}
    if field:
        c["field"] = field
    return c


def parse_args(argv):
    """`query <role> <function> [k=v ...]` -> (role, function, {k: v})."""
    if len(argv) < 2:
        return None, None, {}
    role, function = argv[0], argv[1]
    kw = {}
    for tok in argv[2:]:
        if "=" in tok:
            k, v = tok.split("=", 1)
            kw[k] = v
    return role, function, kw


def main(role, capabilities, functions, describe_extra=None, properties=None, probe=None,
         writers=None, writers_probe=None, transports=None, config_probe=None,
         transport_detail=None):
    """Standard entry point. `functions` maps name -> callable(kw) -> response.

    `capabilities` are claims that MUST be exercisable by a conformance probe
    (ADR-008 C2). `properties` are declarative traits that cannot be probed —
    persistence, provenance quality. Keeping them apart is what makes an
    honest-advertisement check meaningful rather than vacuous.

    `transports` maps a transport name to the capability set that transport can
    serve, for `describe --transport=<name>`. A capability set is a property of
    (provider x transport), not of the provider alone (issue #17). Omit it when
    every transport offers the same set, and say so rather than implying variance
    that does not exist.

    `writers_probe` gates the WRITE half separately. A reachable transport is not
    necessarily a writable one -- a read-only Dolt database is readable and
    advertising `record_decision` against it is the same dishonesty as
    advertising reads against an absent one.

    `probe` is a zero-arg callable returning whether the transport is usable.
    When it returns false, `describe` advertises NO capabilities. This follows
    LSP's rule that a missing property means absence of the capability: an
    adapter that cannot reach its backend can serve nothing, so it must claim
    nothing. Without this an unconfigured adapter advertises capabilities it
    will fail every query for (issue #17).
    """
    argv = sys.argv[1:]
    if not argv:
        emit(fail(role, "-", "BAD_REQUEST", "usage: describe | query <role> <function> [k=v]"))
        return 0
    if argv[0] == "describe":
        # `describe --transport=<name>` answers "what could THAT transport serve?"
        want = None
        for a in argv[1:]:
            if a.startswith("--transport="):
                want = a.split("=", 1)[1]
        if want is not None:
            known = transports or {}
            if want not in known:
                emit(fail(role, "describe", "BAD_REQUEST",
                          f"transport {want!r} is not one this adapter supports: "
                          f"{sorted(known) or '(none declared; capabilities do not vary by transport)'}"))
                return 0
            emit({"contract_version": CONTRACT_VERSION, "role": role,
                  "transport": {"name": want, "hypothetical": True},
                  "capabilities": known[want].get("capabilities", capabilities),
                  "writers": sorted(known[want].get("writers", writers or {})),
                  "properties": properties or {}})
            return 0
        reachable = True if probe is None else bool(probe())
        # (ok, detail) or a bare bool. The detail is what an operator needs:
        # "not configured" without naming the missing thing is not actionable.
        _cfg = (True, None) if config_probe is None else config_probe()
        configured, config_detail = _cfg if isinstance(_cfg, tuple) else (bool(_cfg), None)
        writable = reachable and (True if writers_probe is None else bool(writers_probe()))
        d = {
            "contract_version": CONTRACT_VERSION,
            "role": role,
            # LSP rule: missing means absent. Unreachable transport => no claims.
            "capabilities": capabilities if reachable else {},
            "properties": properties or {},
            "functions": sorted(functions),
            "writers": sorted(writers or {}) if writable else [],
            # `configured` is a THIRD axis, separate from reachable and
            # writable. A reachable transport with no identity declared can
            # answer nothing, and `--validate` reported READY_FOR_GOVERNANCE
            # for exactly that (issue 51). The adapter answers it, because
            # which variable it needs is adapter-specific knowledge ADR-003
            # keeps out of the engine. Default True: an adapter with nothing
            # to configure is configured.
            # `detail` explains an unreachable transport in the adapter's own
            # words. The engine's default message assumes unreachable implies
            # unconfigured, which is false for a declared agent-supplied
            # transport: configured by a human, and simply not usable from
            # this process.
            "transport": {"reachable": reachable, "writable": writable,
                          "configured": configured, "configured_detail": config_detail,
                          "detail": (transport_detail() if callable(transport_detail)
                                     else transport_detail) if not reachable else None,
                          "supports": sorted(transports or {})},
        }
        if describe_extra:
            d.update(describe_extra)
        emit(d)
        return 0
    if argv[0] not in ("query", "write"):
        emit(fail(role, "-", "BAD_REQUEST", f"unknown verb {argv[0]!r}"))
        return 0
    verb = argv[0]
    req_role, function, kw = parse_args(argv[1:])
    if req_role != role:
        emit(fail(role, function or "-", "BAD_REQUEST", f"this adapter serves role {role!r}, not {req_role!r}"))
        return 0
    table = (writers or {}) if verb == "write" else functions
    fn = table.get(function)
    if fn is None:
        if verb == "write" and not writers:
            emit(fail(role, function or "-", "UNSUPPORTED_FUNCTION",
                      "this adapter declares no writers; it is read-only"))
        else:
            emit(fail(role, function or "-", "UNSUPPORTED_FUNCTION", f"{function!r} not advertised"))
        return 0
    emit(fn(kw))
    return 0
