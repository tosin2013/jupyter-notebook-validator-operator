# 11. Python Stdlib-Only Engine with Language-Agnostic Adapters

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Runtime & packaging

## Context

ADR-001 ships Repo Governor as an Agent Skill whose `scripts/` directory contains the deterministic engine from ADR-002. ADR-003 defines adapters as subprocesses speaking JSON. Neither picks a language, and the choice is constrained more tightly than it first appears.

The constraints:

- **No install step.** Distribution is `git clone` into a skills directory. Anything requiring `pip install`, `npm install`, or a build breaks the property that makes skill distribution work across 26+ hosts.
- **Runs wherever agents run.** Skill scripts execute via bash in the agent's environment. That environment is not controlled by us and varies by host.
- **Deterministic and testable.** ADR-002 requires a pure function with unit-testable predicates.
- **Auditable.** This is a public governance artifact whose credibility depends on people reading it. Dependencies are audit surface.

## Decision

**The engine is Python 3.11+ using the standard library only. Adapters may be written in any language and communicate as subprocesses over JSON.**

1. **Python 3.11+, zero third-party dependencies.** Present by default on macOS and virtually all Linux, and already the ambient language of the agent tooling ecosystem. 3.11 is the floor for `tomllib` and mature `dataclasses` behaviour. Any dependency that seems necessary is a signal to reduce scope instead.

2. **JSON, not YAML.** PyYAML is not stdlib and would be the single dependency that breaks the no-install property. **Resolved by ADR-015: JSON is canonical.** A spike showed a 143-line YAML subset parser silently mis-types 7 of 10 realistic manifest values — `engine_min_version: 1.0` becomes a float, `type: no` becomes `False` — which is disqualifying under ADR-002. The engine reads `.repo-governor.json` with `json.loads` and ships no YAML parser.

3. **Adapters are language-agnostic subprocesses.** The engine invokes `$ADAPTER describe` and `$ADAPTER query ...`, reading JSON from stdout. It never imports a provider SDK. This is what lets a Go Beads adapter, a shell Git adapter, and a Python ADR adapter coexist without the core taking on any of their dependencies — and it is the mechanism by which §54's "must not require a specific tracker" holds at the code level rather than only in principle.

4. **Adapters may have dependencies; the engine may not.** A Linear adapter needing an HTTP client is fine — it is optional, separately installed, and only by someone who chose to bind Linear. The engine's dependency count stays zero.

5. **Typed boundaries.** `dataclasses` plus JSON Schema validation at every I/O boundary. Static typing via annotations checked in CI, not enforced at runtime.

6. **No network access in the engine.** All external I/O happens in adapters. This makes ADR-002's purity claim mechanically checkable — the engine imports no socket-capable module — and confines the credential surface to adapters, supporting §51.

## Consequences

**Positive**

- Clone-and-run works on every host with a modern Python, which is nearly all of them.
- Zero dependencies means zero supply-chain surface in the component that makes governance decisions. For a security-adjacent public project this is worth real inconvenience.
- The subprocess boundary is a natural process-isolation boundary: an adapter crash is a typed failure, not an engine crash.
- Third-party adapter authors are unconstrained in language choice, which materially lowers the barrier to the provider ecosystem the thesis needs.

**Negative**

- JSON is worse to hand-edit than YAML — no comments, mandatory quoting. ADR-015 accepted this cost because onboarding generates the manifest rather than humans authoring it from scratch. The ergonomic loss is real and lands on occasional edits.
- Stdlib-only means writing things that libraries do better — JSON Schema validation in particular. Expect a few hundred lines of infrastructure that a dependency would have supplied.
- Subprocess-per-query has real overhead (tens of milliseconds each). Fine at one evaluation per repository; would need a batch protocol if usage patterns change.
- Windows support is untested. Bash-invoked scripts and subprocess semantics differ. Out of scope for MVP, but it should be stated rather than discovered.
- Python version skew across hosts is a live risk. Some environments still ship 3.9. The engine should fail with a clear version message rather than a syntax error.

## Domain Considerations

The zero-dependency stance is a governance argument as much as a technical one. A tool that rules on what agents may change to a repository, while itself pulling a transitive dependency tree, has an awkward story. Being auditable in an afternoon is part of the product.

The engine/adapter split also matches ADR-005's permission model at the process level: adapters are the only components that touch credentials, so the permission gate sits exactly at the process boundary rather than somewhere inside a shared address space.

## Implementation Plan

1. ~~Decide manifest canonical format~~ — **done, ADR-015: JSON canonical.**
2. Implement a minimal JSON Schema validator covering the subset the schemas use.
3. Implement the adapter subprocess protocol with typed errors and timeouts.
4. Add a CI check asserting the engine imports nothing outside a stdlib allowlist, and nothing network-capable.
5. Add a Python version guard producing an actionable message below 3.11.
6. Test on Claude Code, Codex, and Cursor environments (ADR-001 step 5).

## Related Specification Sections

§51 Security and Boundary Model · §54 Failure Conditions · §62 Initial Implementation Boundary · §63 MVP Requirements · §64 MVP Non-Commitments

## Domain References

- [Claude Code Skills Complete Guide — creating, testing, distributing](https://hidekazu-konishi.com/entry/claude_code_skills_complete_guide.html)
- [Agent Skills — Claude Platform Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
- `docs/research/2026-08-17-external-landscape.md` §1

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
