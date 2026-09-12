# 21. Every Provider Is Resolved Through the Manifest

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Engine architecture / authorization
**Implements**: [ADR-005](005-deny-by-default-permission-model.md) rule 2
**Resolves**: issue 21

## Context

ADR-005 rule 2 specified an implementation step in plain terms:

> Implement the permission gate as a single chokepoint every adapter invocation passes through — **no adapter may be called except through it.**

It was never built. `engine/completion.py` spawned adapters directly through a local `call()` helper that consulted nothing:

```python
def call(adapter, role, fn, kw, env_extra=None, verb="query"):
    p = subprocess.run(args, ...)      # no permission check
```

`MF.permitted` was consulted in exactly one place — the decision-history write — and nowhere on the read path. The manifest was load-bearing for one operation out of every call the engine made. For reads it was decorative.

Alongside that, the engine had accumulated knowledge it has no business holding:

| Location | Hardcoded |
|---|---|
| `completion.py:91` | `adapters/acceptance-file` |
| `completion.py:110` | `adapters/git` |
| `completion.py:224` | `adapters/file-roadmap` as the default roadmap |
| `completion.py:228-231` | fixture environment variables selected by adapter name — `if adapter == "adapters/linear"` |
| `completion.py:191` | decision-history writer chosen by `type.startswith("decision-history-dolt")` |

The last one is the sharpest. `describe` had already gained a `writers` list gated on a real writability probe (ADR-020, issue 17) — the mechanism to choose a backend by what it can actually do existed and was not used. Selection by product name meant a second writable backend could never be chosen, and a renamed one would silently stop being.

So the architecture was provider-oriented and the decision engine was not. Providers were interchangeable in the design and named in the code.

## Decision

**Every adapter invocation the engine makes resolves the role through the manifest and passes the permission gate first. `engine/bindings.py` is the only module in `engine/` that may spawn an adapter.**

1. **The engine names roles, never adapters.** A path literal in the decision path is a binding decision made in code, which is exactly what INV-013 forbids the rest of the system from doing.

2. **The permission check happens before the subprocess.** Checking afterwards would make denial advisory — the provider would already have been contacted. A denied call must not reach the provider at all.

3. **Denial is a typed envelope, not an exception** (ADR-005 rule 5), so a permission shortfall reads as a disposition rather than a crash, and never as a silent skip that makes a decision look complete when it is not.

4. **Configuration reaches an adapter from the manifest binding.** A new optional `env` object per binding carries adapter-specific variables, and `REPO_GOVERNOR_BINDING` carries the whole binding as JSON for adapters that read structured config such as a declared admission signal. **The engine sets no adapter-specific variable of its own.** That is what stops `if adapter == "adapters/linear"` growing back.

5. **Writer selection is by advertised capability.** `writer_for(role, fn)` picks a binding whose `describe` reports `fn` among its writers. A function nothing advertises is refused up front rather than sent to the first binding and failing deep inside the adapter.

6. **Binding is not granting.** A bound, reachable, credentialed provider with no permission block is denied. This is INV-014 at the engine boundary.

7. **The property is tested, not documented.** `conformance/bindings.py` fails the build if any of the above regresses — including a check that the decision path contains no `adapters/` literal, and that exactly one module spawns an adapter.

## Consequences

**Positive**

- The manifest becomes what ADR-004 says it is: the sole artifact that binds providers to roles. Until now that was true of the *description* and false of the *execution*.
- Rebinding a role is a manifest edit. The roadmap move in ADR-022 required no engine change, which is the first real evidence that the provider abstraction is load-bearing rather than decorative.
- Renaming an adapter no longer changes behaviour.

**Negative**

- One more indirection between the decision logic and the provider. Accepted: the indirection is the point, and `bindings.py` is small enough to read in one sitting.
- Adapter configuration now lives in two places during migration — binding `env` for engine-driven calls, and direct environment variables for conformance suites, which deliberately drive adapters without a manifest. The suites are testing the adapter contract, not the binding layer, so this is a real distinction rather than duplication.
- **One provider call still bypasses the gate:** `_repo_is_public()` shells out to `gh repo view` to pick the redaction default. It is a host query rather than an adapter invocation and no role serves repository visibility, but it is an exception and is named here rather than left to be discovered.

**Not addressed**

Whether the permission gate should also bound what the *host* can reach on the engine's behalf. Under agent-supplied transport (ADR-020) the manifest bounds what Repo Governor asks for and not what the host could supply. Issue 20.

## Related Specification Sections

§21 Repository Governance Manifest · §22 Permission Model · §51 Security and Boundary Model · INV-013 · INV-014
