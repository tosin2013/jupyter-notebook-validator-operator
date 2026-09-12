# Fixtures for the `union` suite (issue 154)

Two-provider fixtures for ADR-013's multi-valued roles. Three ADR directories,
each holding the fewest decisions that make one case. `adrs-x` deliberately
holds TWO: with one, a check that the single-provider order is preserved cannot
fail, because a one-element list cannot be reordered.

| directory | holds | used for |
|---|---|---|
| `adrs-x/` | `ADR-0001`, `ADR-0003` Accepted | the first architecture provider in every case |
| `adrs-y/` | `ADR-0002` Accepted | **disjoint** — union, and no finding |
| `adrs-w/` | `ADR-0001` Superseded | **contradicts x** — same id, active in one, superseded in the other |

`dh-a/` is an empty decision store and `dh-b/` holds one `REJECTED` record. The
record exists ONLY in `dh-b`, which is the point: before issue 154 the engine
read `bindings[0]` and could not see it.

**These are two instances of `adapters/adr` over different directories.** They
prove the fan-out works and they prove nothing about portability —
`docs/adrs/README.md:121` warns that adapters by one author "measure shared
intent as much as portability". A genuinely foreign second provider is issues
155 and 156.
