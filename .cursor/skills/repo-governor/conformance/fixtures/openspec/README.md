# Fixtures for `adapters/openspec` (issue 155)

Three layouts, each the smallest thing that makes one measured case real.

| fixture | layout | the case it encodes |
|---|---|---|
| `full/` | `specs/`, `changes/`, `changes/archive/` | the complete layout — 58.3% have `specs/`, 60.1% have `changes/archive/` |
| `delta/` | `changes/` only | **41.7% of real OpenSpec repositories.** `get_specs` must answer a typed `UNKNOWN`, never an empty success |
| `loose/` | `changes/` with two loose files | **11.2%.** The files must be reported as skipped, not silently dropped |

`archive/` sits **inside `changes/`**, which is where the census found it.
A top-level `openspec/archive/` — the layout issue 155 was originally written
against — exists in 4.5% of repositories and is not modelled here.

Nothing under `full/changes/archive/` is a superseded decision. It is a
**completed** change, and `get_superseded_decisions` returns empty by
declaration for that reason.
