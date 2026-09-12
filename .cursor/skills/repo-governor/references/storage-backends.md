# Adding a storage backend

How to back a provider role with a different database. Written from building two
backends for `decision_history` — [Dolt](../adapters/decision-history-dolt) and
[GitHub](../adapters/decision-history-github) — so the pattern is described from
working code, not from a design sketch.

Decision recorded in [ADR-019](../docs/adrs/019-database-backed-decision-history.md).

## The one rule that shapes everything

**The engine carries no dependencies; adapters may.** ADR-011 rule 1 keeps
`engine/` on the Python standard library so `git clone` is the whole install.
Rule 4 permits an adapter to require anything, *because it is opt-in* — only
someone who binds it pays for it.

So a backend is a subprocess that speaks JSON, never a library the engine
imports.

## Use the CLI, not a driver

```python
subprocess.run(["dolt", "sql", "-q", query, "-r", "json"], ...)
```

Not `import mysql.connector`. A driver is a Python dependency and would leak
into the engine's environment; a CLI is a process boundary. This is the same
choice ADR-016 made for Beads — *prefer the interface a vendor built for
programs over the one they built for models* — and it extends cleanly:
`psql -t -A -c '…'`, `sqlite3 -json`, `duckdb -json`.

If a store has no machine-readable CLI output, that is a reason not to use it,
not a reason to add a driver.

## Measure the contract before writing against it

Every backend has surprises. Check these four things first and write them into
the adapter docstring, because the next person will not re-measure:

| Question | Dolt 2.3.0 | GitHub (`gh api graphql`) |
|---|---|---|
| What shape is success? | `{"rows": [ … ]}` | `{"data": {…}}` |
| Are JSON columns parsed or strings? | parsed | n/a |
| Do errors exit non-zero? | yes, exit 1 | yes |
| Is error output parseable? | **no — plain text** | yes, JSON |

That last row is why `_sql()` checks the exit code *before* parsing. Parsing
first would raise on the error path and lose the message — a real bug avoided
by measuring rather than assuming.

Initialization surprises count too: `dolt init` fails with `empty ident name
not allowed` unless given `--name` and `--email`.

## The four things every backend must do

### 1. Probe honestly

```python
def _probe():
    return shutil.which("dolt") is not None and (DB_DIR / ".dolt").is_dir()
```

Passed to `_protocol.main(probe=_probe)`. When it returns false, `describe`
advertises **no capabilities** — LSP's *missing means absent* rule. A missing
database must never read as an empty history.

### 2. Fail typed, never empty

An unreachable store returns `PROVIDER_UNAVAILABLE`. It does **not** return
zero rows. "The database is down" and "nothing was ever decided" are different
facts, and conformance check C3 exists to catch conflating them.

### 3. Distinguish absence from unknown

No rows for a known id is not an error — it is `NO_DECISION_RECORDED`,
non-blocking. Absence of a prior decision is not permission (INV-005), but it
does not stop work either.

### 4. Cite every fact

```python
cite("dolt", f"{DB_DIR}#decisions.{row['decision_id']}", "disposition")
```

Point at the underlying record, never at the query. A fact without provenance
is treated as unknown rather than as true.

## Refuse unsafe input; do not escape it

```python
SAFE_ID = re.compile(r"^[A-Za-z0-9._#-]{1,128}$")
```

Identifiers become SQL literals. Refusing anything outside a known-safe pattern
is checkable; escaping is a promise. `BAD_REQUEST` on a bad id, always.

## Say whether the store supplies the chain

ADR-009 specified hand-rolled hash chaining. Where a store provides history
natively, **do not reimplement it**:

```python
PROPERTIES = {"chain_supplied_by_store": True}
```

Dolt supplies `dolt_log` (commit hash, committer, timestamp) and
`dolt_history_decisions` (every revision of every row), so the chain is the
store's. A backend that cannot must implement chaining itself and advertise
`chain_supplied_by_store: False`, so the difference is visible rather than
assumed.

## Advertise gaps rather than papering over them

`decision-history-github` cannot express a reversal condition, because GitHub
has no such field. It says so:

```python
CAPABILITIES = {"reversal_condition": False}
```

and returns a typed blocking unknown explaining what is missing. Layer 2 then
scores this `CAPABILITY_GAP` rather than divergence — the abstraction is not
leaking, the backend genuinely lacks the concept.

Set `provenance_quality: "inferred"` where a value is derived rather than read.
GitHub's `stateReason` *implies* a decision; Dolt's `disposition` column *is*
one.

## Prove it

Add to `conformance/layer1.py`:

```python
"adapters/your-backend": {
    "role": "decision_history",
    "capability_fn": { … },          # every capability advertised true
    "break_env": { … },              # must yield PROVIDER_UNAVAILABLE
    "unknown_fn": ( … ),             # must yield a typed unknown
    "absence_fn": ( … ),             # must yield NOT_FOUND
},
```

Then add a Layer 2 scenario pairing it against an existing backend. **This is
the part that matters.** Two backends of genuinely different shape agreeing on
the same typed facts is the only real portability evidence; a second backend
you wrote to match the first proves only that you are consistent with yourself.

The `work_declined` scenario compares a SQL database against a REST API for
exactly this reason.

## Checklist

- [ ] CLI transport, no driver import
- [ ] Contract measured and written into the docstring
- [ ] `_probe()` gating capabilities
- [ ] Typed failure on unreachable
- [ ] Absence distinct from unknown
- [ ] Provenance on every fact, citing the record
- [ ] Identifiers refused, not escaped
- [ ] `chain_supplied_by_store` stated
- [ ] Gaps advertised false with a typed unknown
- [ ] Layer 1 entry, Layer 2 scenario against a different-shaped backend
