#!/usr/bin/env bash
# Run the conformance suites, and FAIL when any of them fails.
#
# Why this exists. The loop it replaces was published in CONTRIBUTING.md,
# docs/installation.md and AGENTS.md as the gate to run before opening a PR:
#
#     python3 conformance/$s.py >/dev/null 2>&1 && echo PASS || echo FAIL
#
# It prints FAIL and exits 0. Always. "Expect 10/10" was enforced by the reader
# noticing a word go by. That is the exact defect this repository has fixed six
# times inside its own suites -- a check that cannot fail is worse than no
# check, because it reports safety it never established -- sitting in its own
# contributor instructions.
#
# One runner, called by CI and quoted by the docs, so the two cannot drift.
#
# Usage:
#   ./tools/run-conformance.sh                 all suites
#   ./tools/run-conformance.sh --hermetic      everything that needs no network
#   ./tools/run-conformance.sh envelope skill  just these

# NOT `set -e`. A failing suite must be RECORDED and the run continued, so one
# early failure does not hide five later ones. `set -e` here would report the
# first defect and call it the only one.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 1

# Run the suites in a clean environment, whoever invoked them.
#
# The engine no longer re-exports REPO_GOVERNOR_TARGET to adapters (issue 54),
# so the leak this originally guarded is closed at its source. It stays because
# a PERSON may have the variable exported in their own shell -- a legitimate
# declaration -- and `hooks` and `onboarding` assert the UNGOVERNED case, which
# any inherited target makes false.
#
# A harness that reports differently depending on who ran it is not a harness.
# Suites that genuinely need targeting set it themselves, per subprocess
# (conformance/bindings.py does exactly that), so scrubbing here hides nothing.
unset REPO_GOVERNOR_TARGET REPO_GOVERNOR_SUBJECT REPO_GOVERNOR_BINDING

# `install` is LIVE for a different reason than `hooks`: it asks a REMOTE
# whether the tag the README tells people to clone actually exists. That is a
# claim about the outside world, and a suite comparing two files in this
# repository can only ever prove they agree with each other -- which they did,
# perfectly, while both were wrong, during the v0.4.0 release (issue 178).
#
# `hooks` is last and deliberately separate: it runs engine/completion.py
# against the LIVE repository (conformance/hooks.py:163), so its verdict
# depends on issue 36's current milestone and assignee. Someone moving a card
# on the board would turn it red with no code change, which is why --hermetic
# exists and why CI runs it as a non-blocking job.
HERMETIC=(layer1 layer2 transport manifest onboarding vocabulary bindings skill envelope execution imports status acceptance coverage benchmark union)
LIVE=(hooks install roadmap)

case "${1:-}" in
  --hermetic) SUITES=("${HERMETIC[@]}"); shift ;;
  --live)     SUITES=("${LIVE[@]}");     shift ;;
  -h|--help)  sed -n '2,25p' "$0"; exit 0 ;;
  "")         SUITES=("${HERMETIC[@]}" "${LIVE[@]}") ;;
  *)          SUITES=("$@") ;;
esac

# A named suite that does not exist must be an error, not a silent skip. A
# runner that quietly ignores a typo reports "all pass" over a set the caller
# did not get -- vacuous green by a different route.
missing=0
for s in "${SUITES[@]}"; do
  [ -f "conformance/$s.py" ] || { echo "no such suite: conformance/$s.py" >&2; missing=1; }
done
[ "$missing" -eq 0 ] || exit 2

failed=()
for s in "${SUITES[@]}"; do
  printf '%-12s ' "$s"
  # Output is captured rather than discarded: on failure it is REPLAYED below.
  # `>/dev/null 2>&1` is what made the old loop useless to debug -- you learned
  # that something failed and nothing about what.
  out="$(python3 "conformance/$s.py" 2>&1)"; rc=$?
  # Every suite ends with a declared line saying how many assertions it
  # actually executed (issue 67). Read that marker rather than counting
  # [PASS] lines: a suite that changes its output format would silently
  # leave coverage, and the first attempt at this was contaminated within
  # one command by a line whose LABEL quoted "14/14 pass".
  count_line="$(printf '%s\n' "$out" | grep "^CONFORMANCE-COUNT" | tail -1)"
  executed="$(printf '%s' "$count_line" | sed -n 's/.*executed=\([0-9]*\).*/\1/p')"
  if [ -z "$count_line" ]; then
    echo "FAIL (no CONFORMANCE-COUNT line)"
    failed+=("$s")
    echo "  | the suite does not report how much it did; silence is"
    echo "  | indistinguishable from having nothing to report"
    continue
  fi
  if [ "${executed:-0}" -eq 0 ]; then
    echo "FAIL (executed 0 assertions)"
    failed+=("$s")
    echo "  | a suite asserting nothing is broken or lying; both need a human"
    continue
  fi
  if [ "$rc" -eq 0 ]; then
    echo "PASS  ($executed assertions)"
  else
    echo "FAIL (exit $rc)"
    failed+=("$s")
    printf '%s\n' "$out" | sed 's/^/  | /'
  fi
done

echo
if [ "${#failed[@]}" -eq 0 ]; then
  echo "${#SUITES[@]}/${#SUITES[@]} pass"
  exit 0
fi
echo "$(( ${#SUITES[@]} - ${#failed[@]} ))/${#SUITES[@]} pass -- FAILED: ${failed[*]}"
exit 1
