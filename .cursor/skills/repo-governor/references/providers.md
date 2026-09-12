# Providers — operational reference

**Measuring activation on a new host?** [`harnesses.md`](harnesses.md) is the contract for adding one to the benchmark, and why calibration is per-host.

**Writing one?** [`integrations.md`](integrations.md) is the contract an integration must satisfy, how it is checked, and the rule that shapes the rest: an integration carries governance through rather than reinventing it.

Eight roles, each answering a different governance question. Normative contracts and
conformance minimums: [`docs/reference/provider-roles.md`](../docs/reference/provider-roles.md).

| Role | Question | Shipped adapter |
|---|---|---|
| `roadmap_authority` | Is this admitted and still authorized? | `file-roadmap`, `linear`, `github-projects` |
| `architecture` | What constrains how it must be built? | `adr` |
| `execution` | What is the state beneath the authority? | `execution-file`, `beads` |
| `repository` | What is actually here? | `git` |
| `change_signals` | What changed outside? | `change-signals-file` |
| `retirement` | What obligations block removal? | `retirement-analysis` |
| `decision_history` | What was already decided? | built-in log |
| `acceptance_criteria` | What counts as done? | `acceptance-file` |

Keeping these apart is the point: it stops *"the task says READY"* implying *"the work is authorized"*.

**One role answers its question and nothing acts on the answer.** `architecture` is
**reported, not consulted**: its decisions land in the compiled envelope and move one of
five thinness dimensions, and no disposition reads them. A repository can bind thirty
Accepted ADRs and no verdict changes.

That is worth stating because the binding implies otherwise — the adapter reads four
status dialects, onboarding reports `PROVIDER_DETECTED` with its evidence, and the
question above promises a constraint. `engine/status.py` now prints the same caveat
beside the role. Whether it *should* reach a disposition is
[#143](https://github.com/tosin2013/repo-governor/issues/143), open, and the obstacle is
measured rather than assumed: `get_constraints` returns ADR titles, and 0 of 8 realistic
discovery targets matched any of this repository's Accepted titles.

## Wire protocol

```bash
$ADAPTER describe                                  # capabilities + contract version
$ADAPTER query <role> <function> [k=v ...]         # typed response on stdout
$ADAPTER query <role> <fn> ... --input -           # option C -- EXPERIMENTAL, see below
```

Any language. The engine never imports a provider SDK. How an adapter reaches its
backend — HTTP, CLI, file, MCP — is invisible to the engine (ADR-016).

**`--input -` is experimental.** It rests on [ADR-020](../docs/adrs/020-agent-supplied-transport-with-adapter-as-normalizer.md), which is `Proposed` and not accepted, and the engine never uses it — only `conformance/transport.py` does. The authority-boundary question it raises is open ([#20](https://github.com/tosin2013/repo-governor/issues/20)). Build against it only if you are prepared for the contract to change.

## Rules an adapter must honour

- **Advertise honestly.** A claimed capability must be exercisable by a probe. Unreachable transport → advertise nothing.
- **Fail typed.** An unreachable backend returns `PROVIDER_UNAVAILABLE`, never a plausible empty result.
- **Distinguish absence from unknown.** "No such item" is `NOT_FOUND`; "could not determine" is an `unknown`.
- **Cite everything.** A fact without provenance is treated as unknown, not as true.
- **Never coerce.** A value outside a closed vocabulary is `MALFORMED_SOURCE`.

- **Answer about the checked-out revision, and say so.** A repo-local provider reads the working tree it is pointed at, so its answer is a function of the checkout. Most read paths that do not move between branches — `adapters/adr` reads `docs/adrs/`, `adapters/openspec` reads `openspec/` — and the variance is invisible. `adapters/speckit` reads `specs/<feature>/`, which Spec Kit populates **per feature branch**, so it is the first provider where the property is visible rather than theoretical. It is a property of the class, not that adapter's defect — ADR-033, **Proposed**: [a repo-local provider answers about the checked-out revision](../docs/adrs/033-repo-local-providers-answer-about-the-checked-out-revision.md).

  The engine may not consult a method whose answer varies with the checkout until the recorded provenance names the revision. `conformance/bindings.py` fails if it starts to.

Verify with `python3 conformance/layer1.py`.

## Cardinality

Single-valued: `roadmap_authority`, `execution`, `repository`, `acceptance_criteria`.
Multi-valued: `architecture`, `change_signals`, `retirement`, `decision_history`.

Two candidates for a single-valued role halt onboarding. No ranking is applied.

### Which architecture provider do I bind?

Two ship, and they answer different questions. A repository that has both should
bind both — neither is a substitute for the other.

| | answers | shipped as |
|---|---|---|
| **ADRs** | **what was decided.** Immutable: a decision is superseded by a new one, never edited. True regardless of what is being built now. | `adapters/adr` |
| **OpenSpec** | **what is being built.** `changes/<id>/` is in flight; it moves to `changes/archive/<id>/` when it lands, and `specs/` holds what the system must currently satisfy. | `adapters/openspec` |
| **Spec Kit** | **what the system must satisfy, and what each feature specifies.** `.specify/memory/constitution.md` holds the binding principles; `specs/<feature>/` holds the specifications. | `adapters/speckit` |

`adapters/adr` returns empty for `get_specs` by declaration and names OpenSpec as
the resolution; `adapters/openspec` returns empty for `get_superseded_decisions`
by declaration, because **archiving a change is completion, not supersession**;
`adapters/speckit` returns empty for both decision functions, because Spec Kit
records specifications and keeps no decision ledger. Each states what it cannot
supply rather than guessing.

Two things a spec provider will not do. **Task completion is never authority** —
`tasks.md` checkboxes are execution state, and INV-002 keeps that separate from
whether work is authorized. And **an unfilled template is not a constraint**:
10.2% of measured Spec Kit constitutions still carry the shipped placeholders,
and reading their articles would assert an architecture nobody wrote (§37).

You should not have to work this out. `engine/onboard.py` detects both and
proposes what it finds — it never binds (ADR-010), so the choice stays yours,
but it is made from evidence rather than from memory.

**What the choice costs you is a report, not a decision.** No disposition
consults `architecture` today, and `engine/status.py` says so where an operator
reads it. Binding the wrong one, or neither, changes what you are told and
changes no verdict.

**Every binding on a multi-valued role is queried, and the evidence is unioned**
(ADR-013 rule 3). Two `architecture` providers contribute their constraints
together, attributed per provider; two `decision_history` stores are both read,
so a decision recorded in either one is found. Nothing is ranked and nothing is
picked.

Disagreement is narrow on purpose. Providers holding **different** constraints
are the normal case — that is what "evidence accumulates" means, and escalating
on it would be §54's over-escalation failure condition. What escalates is one
id held as an *active* decision by one provider and as *superseded* by another:
that is `ARCHITECTURE_REVIEW` with both cited, and it does not block, because
ADR-013 rule 2 scopes halting to single-valued roles.

What this cannot see: *Accepted here, Rejected there*. `get_constraints` drops a
Rejected decision and `get_active_decisions` filters it out, so no function in
the role's contract reports an id whose status is neither active nor superseded.
