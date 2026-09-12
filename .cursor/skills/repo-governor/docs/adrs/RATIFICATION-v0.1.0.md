# Architecture Ratification Review — v0.1.0

**Prepared** 2026-08-17 · **Ratified 2026-08-17 by Tosin Akinosho (§68).**

> **Outcome.** 23 of 26 Accepted. ADR-014 `Superseded`, split into ADR-023 (Accepted) and ADR-024 (held). ADR-020 and ADR-024 remain `Proposed` and **neither is referenced by the v0.1.0 runtime**, which is what makes holding them consistent with the release condition below.
>
> Two decisions were added *after* the first ratification and ratified in a second pass: **ADR-027** and **ADR-028**, both produced by fixing issues 24 and 26. The runtime depends on both — 027 in `engine/bindings.py`, `engine/manifest.py` and `adapters/_protocol.py`; 028 in the two GitHub-backed adapters — so under the condition stated here they could not be held. That is the condition working: it identified the gate rather than being satisfied by a count.
>
> The three documentation defects F1–F3 were fixed before the tag, and a fourth class was found afterwards and fixed under issue 29 — the shipped skill taught an invocation the engine had stopped supporting. `conformance/skill.py` now checks the agent surface on every run, in both directions of staleness.

**Acceptance authority:** Tosin Akinosho (§68). The review below is the evidence; the ratification recorded above is the decision. They are deliberately separate documents in one file — an agent produced the first and a human made the second.

## Why this was not "accept all 22"

Ratifying the architecture because the implementation works reverses the governance relationship:

```
implementation exists  →  therefore architecture must be accepted
```

which is `persistence ≠ authority` wearing a different hat — the exact inference this product exists to refuse. INV-013 says a detected, reachable provider has no role until the manifest binds it. The same applies here: a decision that is implemented, tested and shipped still has no architectural authority until a human accepts it.

So the release condition is not a count. It is:

> **Every architecture decision the v0.1.0 runtime depends on is Accepted, and no Proposed ADR is silently treated as normative by that release.**

That condition is checkable, and checking it found three defects that a category-based review would have missed.

## Method

Classification is derived from evidence, not from judgment about importance.

- **Runtime dependency** — is the ADR cited by `engine/` or by an adapter this repository actually binds? (Unbound adapters ship but govern nothing here.)
- **Mechanism exercised** — does the shipping engine reach the code path the ADR decides, or does the path exist only under test?
- **Position in the graph** — does the ADR stand alone, or does it amend/resolve another?

```bash
# reproduce the dependency map
grep -ohE 'ADR-0[0-9]{2}' engine/*.py | sort | uniq -c | sort -rn
jq -r '.providers|to_entries[]|select(.key|startswith("$")|not)|.value|
       if type=="array" then .[].adapter else .adapter end' .repo-governor.json \
  | xargs cat | grep -ohE 'ADR-0[0-9]{2}' | sort -u
```

## Findings that change the answer

### F1 — The engine never emits `EXECUTE`

Verified: no module under `engine/` produces it. The reachable governance dispositions are

```
STOP_COMPLETE  CONTINUE  UNKNOWN  AUTHORITY_WITHDRAWN  NO_EXECUTION_AUTHORITY
```

Five of the twelve the closed vocabulary defines. The shipping engine answers *"is this finished?"* and *"is authority absent or withdrawn?"* It never issues the affirmative *"you may proceed."*

That is a defensible v0.1.0 scope — the completion firewall is the harder half and it demonstrably works — but **`README.md` currently lists `EXECUTE` first among the dispositions "a deterministic engine returns."** That sentence claims a capability the release does not have.

### F2 — ADR-014 is half-shipped, and that makes #2 unanswerable

ADR-014 bundles two decisions: the **ScopeEnvelope** and the **completion firewall**. The firewall shipped (`engine/completion.py`). The envelope did not. Of ADR-014's own implementation plan, steps 1, 2, 5 and 6 are unbuilt:

| Step | State |
|---|---|
| 1. `ScopeEnvelope` schema + compiler | **not built** — the engine never calls `get_scope` |
| 2. non-goal matching, necessity classifier | **not built** |
| 3. completion firewall | shipped and verified |
| 4. §40 / §44 as tests | partial |
| 5. discovery capture to `.repo-governor/discoveries/` | **not built** — the directory does not exist |
| 6. measure envelope thinness | **not possible** — see below |

Consequence: **issue #2 is not merely unmeasured, it is unanswerable as written.** You cannot measure the thinness of an envelope when nothing compiles one. What §55 records as evidence for #2 — *"both real tracker adapters return `NON_GOALS_UNSTATED` for every item"* — is a fact about adapter *inputs*, not about a compiled envelope. That is still a real and discouraging signal, but it is not the measurement #2 asks for.

Related: `CAPTURE_ONLY` is named in `SKILL.md` as the default disposition for discovery (INV-001), and **no engine code can produce it.** INV-001's enforcement is currently prose. ADR-002 opens by warning that *"an agent that reasons past prose is the documented April 2026 failure"* — the project has that failure mode live, against its own first invariant.

### F3 — ADR-020 is already normative in a shipped file

`references/providers.md` documents

```
$ADAPTER query <role> <fn> ... --input -           # option C: caller supplies raw
```

as a supported call form, with no indication it rests on a Proposed decision. `references/` ships as part of the skill. The engine never passes `--input` — the mechanism is exercised only by `conformance/transport.py` — so this is precisely "a Proposed ADR silently treated as normative by the release."

Two exits, and they are not equivalent: ratify ADR-020, or mark the documentation as experimental. The second is cheaper and matches the evidence, since #20 (the authority-boundary question ADR-020 created) is open.

## Proposed classification

### Class 1 — Foundational · ratify before v0.1.0 (11)

The base layer. Every other ADR assumes at least one of these, and the runtime cannot be described coherently without them.

| ADR | Decision | Runtime evidence |
|---|---|---|
| 001 | Agent Skill as primary surface | `SKILL.md` ships; cited in `engine/` |
| 002 | Deterministic engine, separate from model judgment | 4 citations; C7 byte-identical across runs |
| 003 | Provider roles with normalized contracts | the adapter protocol every adapter implements |
| 004 | Manifest as sole binding artifact | `engine/manifest.py`; 20 refusal cases |
| 005 | Deny-by-default permissions | 7 citations; now enforced at one chokepoint |
| 006 | Repository condition drives profiles | `engine/onboard.py`; profile packs |
| 007 | Closed disposition vocabulary, `UNKNOWN` terminal | `engine/vocabulary.py`; suite asserts closure |
| 010 | Detection separated from binding | 5 citations; onboarding simulation |
| 011 | Python stdlib-only engine | zero imports outside stdlib |
| 012 | Provider content is untrusted input | refusal paths in bound adapters |
| 013 | Single canonical authority per role | cardinality enforced at load |

### Class 2 — Derived or amending · ratifiable with their parent (9)

Each resolves or amends a Class 1 decision and has shipped, reproducible evidence. None can be accepted before its parent.

| ADR | Amends / resolves | Evidence |
|---|---|---|
| 008 | 003 | 7 suites; 136 Layer 1 checks |
| 009 | — (superseded in part by 019) | evidence-chain requirement |
| 015 | resolves 011's deferred choice | JSON canonical; schema enforced |
| 016 | constrains 003 | transport declared, never inferred |
| 017 | adds the 8th role | `STOP_COMPLETE` derivable; verified on 7 live issues |
| 018 | 003 | admission declared; `milestone` signal in use here |
| 019 | amends 009 | Dolt + GitHub backends; Layer 2 cross-store agreement |
| 021 | **implements 005 rule 2** | the chokepoint; `conformance/bindings.py` |
| 022 | amends §54 | roadmap rebound with no engine change |

**021 deserves a note.** It does not add architecture — it builds a rule ADR-005 stated and nobody implemented. Ratifying 005 while leaving 021 Proposed would accept a permission model whose only enforcement mechanism is unaccepted.

### Class 3 — Remain Proposed for v0.1.0 (2)

| ADR | Why |
|---|---|
| **014** | Half-shipped (F2). The firewall earned acceptance; the envelope has not been built. Ratifying it whole would accept architecture that does not exist. **Recommend splitting**: a completion-firewall ADR that can be ratified now, and a ScopeEnvelope ADR that stays Proposed with #2 attached to it. |
| **020** | Mechanism ships in adapters, engine never invokes it (F3), and the authority-boundary question it created (#20) is open. Keep Proposed and mark `references/providers.md` accordingly. |

## What ratification requires of the release, not just of the ADRs

Three documentation defects must be fixed or the release violates its own condition regardless of which ADRs are accepted:

1. `README.md` must stop listing `EXECUTE` among what the engine returns (F1).
2. `SKILL.md` must not present `CAPTURE_ONLY` as engine behaviour while no code produces it (F2).
3. `references/providers.md` must mark `--input -` as resting on a Proposed decision (F3).

## Recommended disposition

```
Class 1 (11)  →  ACCEPT           foundational; runtime depends on all of them
Class 2  (9)  →  ACCEPT           each after its parent; evidence reproducible
Class 3  (2)  →  REMAIN PROPOSED  014 pending a split, 020 pending #20
Docs          →  FIX F1, F2, F3   before any tag
```

**20 of 22 Accepted, 2 deliberately not** — and the two exceptions are the load-bearing part of the exercise, because they are the ones a count-based review would have swept in.

This review is evidence for a decision. It is not the decision.
