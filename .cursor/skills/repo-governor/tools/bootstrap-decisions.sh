#!/usr/bin/env bash
# Create the decision-history store a fresh clone does not carry.
#
# `.repo-governor/decisions-db/` is gitignored: it is a Dolt database, and a
# database is not source. But the manifest BINDS it, so without this a fresh
# clone reports PROVIDER_UNAVAILABLE and four conformance suites fail -- which
# looks like broken code and is actually a missing setup step (#35).
#
# A repository that declares a dependency should supply its bootstrap. This is
# that. Idempotent: safe to run on an existing store.
set -euo pipefail

DB="${REPO_GOVERNOR_DECISIONS_DB:-.repo-governor/decisions-db}"

if ! command -v dolt >/dev/null 2>&1; then
  echo "dolt is not installed. It is an ADAPTER dependency, never an engine one" >&2
  echo "(ADR-011 rule 4). Install it, or unbind decision_history in the manifest" >&2
  echo "and accept that INV-005 is unenforceable here." >&2
  exit 1
fi

if [ -d "$DB/.dolt" ]; then
  echo "decision store already present at $DB"
else
  mkdir -p "$DB"
  # `dolt init` fails with "empty ident name not allowed" unless given both.
  # Values are local bookkeeping, not identity -- Dolt records the committer.
  ( cd "$DB" && dolt init --name "repo-governor" --email "repo-governor@localhost" >/dev/null )
  echo "initialised decision store at $DB"
fi

dolt --data-dir "$DB" sql -q "
CREATE TABLE IF NOT EXISTS decisions (
  decision_id        VARCHAR(64) PRIMARY KEY,
  authority_id       VARCHAR(128) NOT NULL,
  disposition        VARCHAR(32)  NOT NULL,
  reason             TEXT,
  reversal_condition TEXT,
  engine_version     VARCHAR(32),
  manifest_hash      VARCHAR(64),
  snapshot_sha256    VARCHAR(64),
  typed_facts        JSON,
  redacted           BOOLEAN,
  fields_redacted    JSON,
  INDEX (authority_id)
)" >/dev/null

echo "schema ready. verify with:  python3 engine/manifest.py --validate"
