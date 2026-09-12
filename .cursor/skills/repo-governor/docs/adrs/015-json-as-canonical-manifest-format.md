# 15. JSON as the Canonical Manifest Format

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Configuration & runtime
**Resolves**: [#3](https://github.com/tosin2013/repo-governor/issues/3) · **Unblocks**: [#11](https://github.com/tosin2013/repo-governor/issues/11) (gate 5)

## Context

ADR-011 commits the engine to Python standard library only, with zero third-party dependencies — the property that makes `git clone` into a skills directory sufficient to run. §21 sketches the governance manifest as YAML. PyYAML is not stdlib, so those two commitments collide.

ADR-011 left two options open and deferred the choice to an implementation spike: vendor a small YAML-subset parser, or make JSON canonical and treat YAML as convenience input.

The spike is in [`spikes/003-manifest-format/`](../../spikes/003-manifest-format/) and is runnable: `python3 spike.py`.

### What the spike found

A deliberately minimal subset parser — block mappings, block sequences, scalars, comments; no anchors, flow style, or multi-line scalars — came to **143 lines**. It parses the full reference manifest correctly and rejects tabs, duplicate keys, unterminated quotes, and missing colons.

That is the good news, and it is not the finding that matters. The parser is *self-consistent*; it is not *safe*. Against ten values a real manifest could plausibly contain, **seven are silently mis-typed**:

| Input | Parsed as | Should be | Consequence |
| --- | --- | --- | --- |
| `engine_min_version: 1.0` | `1.0` (float) | `"1.0"` | version comparison breaks |
| `contract_version: 1.0` | `1.0` (float) | `"1.0"` | contract match breaks |
| `type: no` | `False` | `"no"` | provider type becomes a boolean |
| `project: 0755` | `755` (int) | `"0755"` | leading zero lost from a key |
| `id: 1e5` | `100000.0` | `"1e5"` | identifier becomes a float |
| `project: ON` | `True` | `"ON"` | the Norway problem, on a project key |
| `adapter: {cmd: x}` | `"{cmd: x}"` | reject | flow mapping accepted as a *string* |

The last row is the worst of them. Unsupported syntax does not error — it is silently absorbed as a string, so an adapter path that looks structured becomes a meaningless scalar and the engine proceeds. Every other row is a type confusion that survives into evaluation.

These are not defects to be fixed. They are YAML 1.1 scalar-resolution semantics, faithfully implemented. Fixing them means diverging from YAML, at which point the file is no longer YAML and the familiarity argument for using it evaporates.

### Why this is disqualifying rather than merely unfortunate

ADR-002 requires a deterministic engine whose disposition is computed from typed facts. ADR-012 establishes that the worst failure mode is a confident wrong answer rather than a loud one. A config parser that turns `engine_min_version: 1.0` into a float and `type: no` into `False` produces exactly that: evaluation continues, the disposition looks authoritative, and the manifest did not say what the engine believes it said.

For a project whose entire premise is that information must not silently acquire authority, shipping a parser that silently changes what the configuration means is not a defensible trade for nicer syntax.

## Decision

**JSON is the canonical manifest format. The engine reads `.repo-governor.json` with `json.loads` and nothing else. No YAML parser ships in the engine, vendored or otherwise.**

1. **Canonical file is `.repo-governor.json`.** ADR-004's rules are unchanged — sole binding artifact, committed, schema-versioned, no secrets, no provider state. Only the encoding changes.

2. **Onboarding writes the manifest; humans rarely author it from scratch.** This is what makes the trade cheap. Per ADR-010, `onboard` already generates a proposal for a human to review and commit. A generator emits JSON as easily as YAML, so the ergonomic cost lands only on hand-editing — and hand-editing JSON that already exists is a small burden.

3. **Comments use `$comment` keys**, as JSON Schema defines and as `.github/project-config.json` already does in this repository. Machine-ignorable, human-readable, no parser extension.

4. **YAML is accepted as *input* only if a YAML parser is already present**, and never by the engine. If a user hand-writes `.repo-governor.yaml`, `onboard` may convert it to JSON when PyYAML happens to be installed, and must otherwise fail with a clear message rather than guessing. The engine never looks at the YAML file. This keeps the zero-dependency guarantee absolute while leaving a door open for users who want it.

5. **JSON Schema remains the specification**, unchanged from ADR-004. The schema was always JSON Schema; now the document it validates is JSON too, which removes a translation step from validation.

6. **The reference manifest in §21 is re-expressed in JSON.** The YAML in the spec becomes illustrative, with the JSON normative.

## Consequences

**Positive**

- The zero-dependency guarantee in ADR-011 holds absolutely, with no asterisk and no vendored code to audit.
- `json.loads` is one line against 143, and its failure mode is a hard parse error rather than a silent coercion — which is what ADR-002 needs.
- No type ambiguity: JSON distinguishes `"1.0"` from `1.0` and `"no"` from `false` structurally. The seven hazards cannot occur.
- Unsupported syntax is a parse error, not an absorbed string.
- One less thing to test. The subset parser would have needed its own conformance suite to stay trustworthy as the manifest grew.

**Negative**

- **JSON is worse to hand-edit.** No comments, mandatory quoting, trailing-comma errors, and no block structure to lean on. This is a real ergonomic loss and the main argument against.
- **`$comment` is a convention, not a comment.** It shows up in parsed output and every consumer must ignore it. Less pleasant than `#`.
- **YAML is what people expect** from a tool-config file in 2026. Some users will read JSON as a dated choice, and the reasoning here will not be visible to them at the point they form that impression. The `$comment` at the top of the file should link to this ADR.
- **The convenience-conversion path is conditional**, so two users on different machines get different behavior from the same `.repo-governor.yaml`. Documented, but genuinely inelegant.

**Neutral**

- Reversible at low cost. If the engine ever acquires a dependency budget, adding YAML input is additive — the JSON path stays as the canonical form.

## Domain Considerations

The generator-writes-it observation is what makes this decision easy rather than painful. Config formats are optimized for hand-authoring, and this manifest mostly is not hand-authored: onboarding produces it, review approves it, and edits are small and occasional. Optimizing its encoding for a parser that cannot lie is the better trade.

There is also a consistency argument. ADR-009 already stores decision records as JSON because they are machine-written and machine-read. The manifest is closer to that category than it first appears.

## Implementation Plan

1. Rename the canonical artifact to `.repo-governor.json` throughout ADR-004, ADR-010, ADR-013, and [`onboarding.md`](../reference/onboarding.md) §21.
2. Re-express the §21 reference manifest as JSON, keeping the YAML alongside as illustration.
3. Implement the loader over `json.loads` plus JSON Schema validation, failing closed on version mismatch, unknown roles, and secret-shaped values (ADR-005).
4. Implement `onboard`'s optional YAML→JSON conversion, gated on PyYAML being importable, with an explicit error when it is not.
5. Add the `$comment` convention to the schema and require the generator to emit a top-level one linking to this ADR.
6. Keep [`spikes/003-manifest-format/spike.py`](../../spikes/003-manifest-format/spike.py) runnable as the evidence for this decision.

## Related Specification Sections

§21 Repository Governance Manifest · §22 Permission Model · §51 Security and Boundary Model · §61 Implementation Gate (condition 5)

## Domain References

- [`spikes/003-manifest-format/`](../../spikes/003-manifest-format/) — the spike, runnable
- ADR-002 (determinism), ADR-004 (manifest), ADR-011 (stdlib-only), ADR-012 (silent wrong answers)
- [JSON Schema `$comment`](https://json-schema.org/draft/2020-12/json-schema-core#name-comments-with-comment)

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
