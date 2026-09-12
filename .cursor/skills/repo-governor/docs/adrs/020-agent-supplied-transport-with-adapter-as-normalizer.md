# 20. Agent-Supplied Transport With the Adapter as Normalizer

**Status**: Proposed
**Date**: 2026-08-17
**Domain**: Provider abstraction / transport
**Amends**: [ADR-003](003-seven-provider-roles-with-normalized-contracts.md), [ADR-016](016-mcp-as-adapter-transport-not-adapter-replacement.md)
**Resolves**: issue 18

## Context

Repo Governor ships as a skill (ADR-001), so the agent running it already holds MCP connections, CLI tools and API credentials. Issue 18 asked whether that makes adapter-side fetching redundant — and if so, whether the transport problem could be *deleted* rather than solved, taking #17's capability matrix and the whole credential surface with it.

Three positions were on the table:

| | who fetches | who normalizes | cost |
|---|---|---|---|
| **A** | adapter | adapter | credentials in every adapter; transport matrix multiplies |
| **B** | agent | agent | model is in the data path producing the typed facts the engine rules on |
| **C** | agent | adapter | — |

**B is refused.** ADR-002 keeps model judgment out of the ruling path. Under B the same provider state can normalize differently between runs, which makes ADR-008 C7 determinism and §53 portability unmeasurable rather than merely hard — and provenance becomes model-attested ("the agent says it cited `state.type`") rather than structural. The falsifiable claim in #1 would be lost. That is too high a price for deleting a credential problem.

C keeps what matters from both: the transport moves outside, normalization stays mechanical.

### What was measured before deciding

`--input` was implemented in three adapters and compared against their own fetching path:

```
adapters/linear                   <-> adapters/linear-fetch                  42 comparisons
adapters/github-projects          <-> captured GraphQL response              25 comparisons
adapters/decision-history-github  <-> captured GraphQL response               3 comparisons
                                                              70 byte-identical, 0 divergent
                                                12 substitute payloads refused
```

Byte-identical, not merely equivalent. If the same bytes come out regardless of who obtained the input, the engine cannot tell which option was used — which is the entire claim.

## Decision

**An adapter MAY accept raw provider output on stdin (`--input -`) and normalize it. This is an additional input path, not a replacement for fetching, and it applies only to remote providers.**

1. **`--input` is optional and additive.** Option A remains the default and remains supported. An adapter that implements `--input` must produce identical output either way; `conformance/transport.py` asserts this, and a divergence is a defect in the adapter, not a permitted variation.

2. **Scoped to remote providers.** `github-projects`, `linear` and `decision-history-github` implement it. The repo-local providers — `adr`, `git`, `file-roadmap`, `acceptance-file`, `change-signals-file`, `execution-file`, `retirement-analysis` — do not, and should not. They read the repository they are governing: no credentials to relocate, no transport matrix to collapse, nothing for the agent to supply that the adapter cannot read directly. Adding the path there would be a second code path bought with no benefit. **"The agent is the transport" is a claim about remote providers only.**

3. **The adapter verifies the payload rather than trusting it.** Supplied input is refused, not normalized, when it is not valid JSON, not the provider's response shape, or *not about the identifier that was requested*. The wrong-issue check exists only under option C — under A the adapter controlled the query, so the failure mode could not arise.

4. **One normalizer, many payload shapes; shapes are told apart structurally.** GitHub's list query returns `repository.issues.nodes` and its single-issue query returns `repository.issue`; both are accepted by the same normalizer, distinguished by inspecting the payload rather than by a transport flag. This is what stops option C reintroducing #17's per-transport matrix by the back door: the variance lives in the payload, and the payload describes itself.

5. **Absence within a page is not absence.** A list payload is a page. `NOT_FOUND` asserted from one is refused with instructions to supply a single-issue response — the same page-boundary trap that produced a false negative on a 380-issue repository.

6. **The fetcher half is a separable executable, not prose.** `adapters/linear-fetch` does I/O and nothing else. It is the runnable specification of what to fetch, so an agent substituting its own transport has something to match. ADR-012's discomfort with contracts that live in prose is answered by making the contract executable and then checking its output.

## Consequences

**Positive**

- Credentials leave the adapter for remote providers. The agent host holds them; Repo Governor never sees them, which strengthens §51 least privilege.
- The #17 capability matrix stops multiplying for adapters that take this path — there is one normalizer regardless of how the bytes arrived.
- Conformance gets easier, not harder: a pure function is trivially testable against recorded raw input, which is what fixtures already are.
- MCP-vs-CLI-vs-REST stops being an adapter concern for these three providers.

**Negative**

- Two input paths per remote adapter is more surface than one. Mitigated by `conformance/transport.py`, which fails the build if they ever disagree.
- **Freshness is not verifiable.** The adapter can prove the payload is correctly shaped and about the right identifier; it cannot prove it is current. A stale-but-valid payload normalizes cleanly. This is a real weakening versus option A and is not closed by this decision.
- ADR-005 says the manifest is the complete statement of what Repo Governor may do. Under option C some access happens through the agent's credentials, outside the manifest's account. The manifest still bounds what Repo Governor *asks for*; it no longer bounds what the host *could* supply. Recorded here rather than resolved — see issue 16.

**Amendments**

- **ADR-003** gains an optional input source in the adapter protocol. Role contracts, function names and the typed-fact vocabulary are unchanged, so this is an amendment and not the rewrite issue 18 question 5 asked about.
- **ADR-016** stands. "Transport is invisible to the engine" survives option C — it is *more* true, because transport may now be absent from the adapter entirely. What ADR-016 did not anticipate is the transport living outside the process, which is recorded here.

## Domain Considerations

**Untrusted input (ADR-012).** Option C widens the untrusted surface: the payload now arrives from the agent rather than from a provider the adapter contacted. The four refusal classes above are the mitigation, and `conformance/transport.py` carries substitute payloads — an LLM summary, the wrong query, non-JSON, the wrong issue — that must each be refused as `MALFORMED_SOURCE`. Refusing a plausible substitute is the property under test; accepting one is how the model gets back into the data path through a side door.

**What is deliberately not decided.** Whether the *engine* should invoke the option C path, and how the skill instructs an agent to produce the raw payload, are left open. The mechanism exists and is proven; the orchestration around it is issue 16's territory.
