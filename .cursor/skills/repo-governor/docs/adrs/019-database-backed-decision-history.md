# 19. Database-Backed Decision History

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Provider abstraction / storage
**Amends**: [ADR-009](009-append-only-evidence-chain-for-decision-provenance.md)

## Context

`decision_history` is the only provider role with no adapter. ADR-009 specified an append-only evidence chain of JSON files under `.repo-governor/decisions/`, hash-chained by hand. It was never built: the directory does not exist, no code references it, and the role is absent from the manifest's bindings. INV-005 and INV-008 both depend on it, so both are currently unenforceable.

When the question of a database came up, the first instinct was to refuse on ADR-011 grounds. That was wrong, and the error is worth naming: **ADR-011 rule 1 governs the engine**, and rule 4 says plainly *"Adapters may have dependencies; the engine may not."* Decision history is a provider, not engine internals. A database backend was always permitted.

It is also the role that most wants one. Decision history is append-only, grows without bound, and is queried by *relationship* — *"was this deferred before, and has its reversal condition been met?"* (§39). That is a query, not a file read. Grepping a directory answers it badly and worse over time.

### What Dolt actually supplies, measured

Verified against Dolt 2.3.0 before committing to the design:

```
dolt_log                    commit_hash, committer, message, date
dolt_history_decisions      every revision of every row, each with its commit_hash
decisions AS OF '<commit>'  time travel to any prior state
```

Concretely: a row moved `DEFERRED` → `REJECTED` across two commits, and `dolt_history_decisions` returned both revisions with distinct commit hashes, while `AS OF` the earlier commit returned `DEFERRED` where the current value is `REJECTED`.

That is ADR-009's evidence chain — append-only, tamper-evident, replayable — **supplied by the store rather than hand-written**. The hash chaining ADR-009 specified is not reimplemented; it is superseded.

## Decision

**`decision_history` is backed by a database. Which database is a binding choice, not a product choice.**

1. **The engine's dependency count stays zero.** The adapter shells out to `dolt sql -q '…' -r json`. No SQL driver, no Python package — the same CLI-over-library choice ADR-016 made for Beads, and the reason ADR-011's guarantee survives intact.

2. **The backend is pluggable, not prescribed.** Any store satisfying the role contract may back it. Dolt is the reference implementation because it supplies history natively; `references/storage-backends.md` documents what another store must provide. `decision_history` is multi-valued (ADR-013), so a second backend may bind alongside — which is how portability gets tested rather than asserted.

3. **Where the store supplies history, do not hand-roll it.** `dolt_log` and `dolt_history_decisions` *are* the chain. A backend that cannot supply history must implement chaining itself and say so in its `properties`, so the difference is visible rather than assumed.

4. **Absent binary means no capabilities.** The adapter probes for `dolt` and advertises an empty capability set when it is missing, per the LSP missing-means-absent rule already implemented in `_protocol.main(probe=…)`. A missing database is not a silent empty history.

5. **Snapshots are redacted by default in public repositories** — hash plus typed facts plus explicit redaction markers. See Domain Considerations.

## Consequences

**Positive**

- Closes the last unbound role and makes INV-005 and INV-008 enforceable for the first time.
- Queries that grep answers badly become SQL: *"every DEFERRED decision whose reversal condition is unmet"* is one statement.
- Tamper evidence, authorship and time travel are the store's problem, not ours. Less code, and code we would have written worse.
- Two backends for one role is the first portability evidence in this project not drawn from a single self-authored fixture shape — which is what #1 has been missing.

**Negative**

- **This gives up a stated ADR-009 benefit.** That ADR argued files made INV-005 and INV-008 *"enforceable with zero external dependencies, which is what §54 demands."* Binding this role now requires installing something. Defensible — §54 prohibits requiring a specific third-party *tracker* for a role, and `decision_history` is optional per profile — but it is a real change to a stated rationale, recorded here rather than left to be discovered.
- **Operational surface.** `dolt init` requires identity configuration (`--name`, `--email`) and fails with `empty ident name not allowed` without it. Someone will hit that on first run.
- **Two more things to keep working.** A second backend doubles the conformance matrix for this role.
- Dolt errors exit 1 with **non-JSON** output. The adapter must check the exit code before parsing, which is easy to get wrong and is now a Layer 1 concern.

**Neutral**

- `GOVERNOR_GREENFIELD` and `GOVERNOR_LITE` do not require `decision_history`, so a small repository installs nothing. §54's friction condition is respected by *not binding the role*, not by keeping a file fallback.

## Domain Considerations

**Redaction.** ADR-009 stores provider snapshots, and this repository is public. Researched practice converges on three things: content hashes rather than content, so a verifier can confirm an artifact matches an input without seeing it; **redaction markers rather than silent omission**, recording that something was redacted and what type; and data minimization with hard exclusion of credentials.

The markers point is the one worth stating. Silently omitting a snapshot is indistinguishable from there having been nothing to record — the same absence-versus-unknown confusion ADR-003 rule 6 forbids in adapters. The log must say *"a snapshot existed and was redacted"*, never leave a gap.

Default: `snapshot_sha256` plus `typed_facts` plus `redacted: true` and `fields_redacted`, with full snapshots only where the repository is private. Replay verifies the hash rather than reproducing content. This closes #4.

## Implementation Plan

1. `adapters/decision-history-dolt` — `get_decisions`, `get_disposition`, `get_reversal_condition`, `get_provenance`.
2. `adapters/decision-history-github` — `stateReason` as recorded decisions; the second backend.
3. `references/storage-backends.md` — what another store must satisfy.
4. Write side in `engine/completion.py`, with redaction.
5. Bind both in the manifest; add both to Layer 1; add a Layer 2 equivalence scenario.
6. Amend ADR-009: implemented, dependency changed, chain superseded.

## Related Specification Sections

§17 DecisionHistoryProvider · §39 Rediscovered Work · §51 Security and Boundary Model · §52 Observability · §54 Failure Conditions · INV-005, INV-008

## Domain References

- Measured against Dolt 2.3.0, 2026-08-17: `dolt_log`, `dolt_history_decisions`, `AS OF`, `-r json` → `{"rows": […]}`, errors exit 1 with non-JSON output
- [Dolt — version control features](https://docs.dolthub.com/sql-reference/version-control)
- [Audit Trails for Accountability in LLMs — arXiv:2601.20727](https://arxiv.org/pdf/2601.20727)
- [Inspectable AI for Science — arXiv:2604.11261](https://arxiv.org/pdf/2604.11261)
- ADR-011 rule 4 (adapters may have dependencies), ADR-016 (CLI over library), ADR-013 (multi-valued roles)

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
