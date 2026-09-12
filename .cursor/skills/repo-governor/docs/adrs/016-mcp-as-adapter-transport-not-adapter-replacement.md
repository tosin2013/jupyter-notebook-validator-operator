# 16. MCP as an Adapter Transport, Not an Adapter Replacement

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Amended by**: [ADR-020](020-agent-supplied-transport-with-adapter-as-normalizer.md) — transport may now live outside the adapter process entirely. "Transport is invisible to the engine" still holds.
**Date**: 2026-08-17
**Domain**: Provider abstraction / integration

## Context

Linear, Jira, Notion, Slack, GitHub and most other systems Repo Governor wants to read now ship official MCP servers. An agent host running Repo Governor often already has those connections configured and authenticated. The obvious question follows: should `adapters/linear` call `api.linear.app/graphql` directly, or should it — or the engine — just call the Linear MCP server?

The pull toward MCP is real and it is mostly about credentials. MCP servers handle OAuth flows, token refresh, and revocation. `adapters/linear` currently wants a raw `LINEAR_API_KEY` in the environment, which is the least pleasant part of its design and the part most likely to stop a user adopting it.

Against that sits ADR-012: provider content is untrusted input, and only *typed facts* may reach the engine. So the question is not "is MCP convenient" but "does MCP deliver typed facts".

### Evidence, measured this session

Two MCP tool calls, both from servers connected to this session:

```
adr-analysis.generate_adrs_from_prd  -> a prompt template instructing the
                                        caller to generate ADRs. Not ADRs.
deepwiki.read_wiki_structure         -> {"result": "<one markdown blob>"}
```

The second is representative. A single string field containing formatted prose, shaped for a language model to read. No typed fields, no provenance, nothing to index into.

Compare the transport `adapters/github-projects` actually uses:

```
gh api graphql -> {"data": {"repository": {"issues": {"nodes": [
                    {"number": 3, "state": "CLOSED", "stateReason": "COMPLETED",
                     "projectItems": {...}}]}}}}
```

Typed JSON. `state.type` is a field with a closed vocabulary, addressable and citable. That is what `_protocol.cite()` needs to produce provenance, and what the normalization maps in `adapters/linear` operate on.

MCP *can* return typed data — the protocol supports structured content, and some servers use it well. But it does not *guarantee* it. An adapter that receives a markdown blob and extracts `state.type` from it by pattern-matching is doing precisely what ADR-012 forbids: treating free text as a typed fact.

> **Measured against a first-party server, 2026-08-17.** The original claim that "the common case in the wild is prose" was drawn from two samples and was too broad. Linear's official MCP server returns **typed JSON**:
>
> ```json
> {"id":"ENG-299","title":"...","status":"Backlog","statusType":"backlog"}
> [{"type":"started","name":"In Review"},{"type":"backlog","name":"Backlog"}]
> ```
>
> `statusType` is exactly the field `adapters/linear` normalizes on, drawn from a closed vocabulary Linear controls. **MCP transport is viable for Linear**, and the rule in this ADR — use MCP only where the server returns structured content — selects it in rather than out. The rule was right; the empirical generalization behind it was not, and prose-returning servers (`deepwiki`, the `adr-analysis` server) are one class rather than the norm.

### Three further constraints

- **Determinism (ADR-002, conformance check C7).** A live MCP server is a network service with a version, a cache, and an LLM sometimes in the loop. Byte-identical output across runs is not guaranteed. Conformance already requires fixture mode for live backends; MCP does not change that, it just adds another live backend.
- **Availability.** MCP servers are configured per agent host, and interactively-authenticated ones are commonly absent in headless, CI, and cron contexts. ADR-001 defers CI enforcement as a secondary surface — but it will exist, and an adapter that only works when an MCP server happens to be connected is not tool-independent in the sense §54 requires.
- **Trust boundary.** An MCP server is a third party sitting between Repo Governor and the system of record, capable of rewriting what it relays. Its output is untrusted input in exactly the ADR-012 sense, and provenance must cite the underlying object, not the MCP tool call.

## Decision

**The adapter is the contract. MCP is one transport an adapter may use behind it. The engine never calls an MCP server.**

1. **No direct engine→MCP path.** The engine speaks only the ADR-003 subprocess protocol. Everything an adapter does to obtain state — HTTP, a CLI, a file, an MCP client — is behind that boundary and invisible to the engine, exactly as `gh` is today.

2. **An adapter may use MCP as its transport when, and only when, the server returns structured content.** The test is concrete: can the adapter address a typed field, with a closed or checkable vocabulary, without parsing prose? If the answer is "we'd regex the markdown", MCP is not a usable transport for that provider and the adapter uses the native API.

3. **Transport is declared, not inferred.** The binding in `.repo-governor.json` names it:

   ```json
   "roadmap_authority": {
     "type": "linear",
     "adapter": "adapters/linear",
     "transport": { "kind": "mcp", "server": "linear" }
   }
   ```

   Defaulting to MCP because a server happens to be connected would be provider capability implying provider use — INV-014 in a different costume.

4. **Transport does not relax any contract.** An MCP-transport adapter passes the same Layer 1 conformance: honest capability advertisement, typed failure on an unreachable backend, absence distinct from unknown, mandatory provenance, byte-identical determinism in fixture mode. If routing through MCP makes a contract check fail, the transport is wrong, not the check.

5. **Provenance cites the underlying object.** `cite("linear", "linear#ENG-101", "state.type")`, never `cite("mcp", "tool_call_7")`. The MCP hop is an implementation detail of retrieval and must not degrade a citation into "some tool said so".

6. **Every adapter keeps a non-MCP path.** MCP transport may be preferred where available; it may not be the only way. This is what keeps CI, cron, and headless runs working, and it is the difference between MCP being an optimization and MCP being a dependency.

## Consequences

**Positive**

- Captures the genuine win — delegated OAuth, no raw API key in the environment — without paying for it in typed-fact integrity.
- The conformance suite already discriminates good transports from bad ones. An MCP server returning prose will fail C5 (provenance) and C2 (capability exercised) without anyone needing to argue about it.
- Transport stays swappable. If Linear's MCP server gains structured output next quarter, `adapters/linear` changes internally and the engine, conformance fixtures, and Layer 2 results are untouched.
- Preserves the property that makes Repo Governor auditable: one protocol into the engine, regardless of how many integration fashions come and go.

**Negative**

- Two code paths per adapter that offers both, which is more to maintain and more to test. Mitigated by conformance running against both, but it is real cost.
- The credential problem is only solved where MCP transport is actually viable. For providers whose MCP servers return prose, adapters still want API keys, and that remains the roughest edge in adoption.
- "Does this server return structured content?" is a judgement made per server, per version. It will drift, and there is no mechanism here that notices when it does beyond conformance failing later.
- A user with a working Linear MCP connection may reasonably expect Repo Governor to just use it, and will experience the declared-transport requirement as friction. The manifest line is small, but the surprise is not.

**Neutral**

- Nothing here forecloses an MCP *server* surface for Repo Governor itself — that is the separate, still-deferred item in §65. Consuming MCP and exposing MCP are different decisions.

## Domain Considerations

The Beads case is worth naming because it is likely to arrive first. Beads ships an MCP server *and* a Go library *and* a CLI, and it is the leading `ExecutionStateProvider` candidate. The CLI is the better transport for an adapter: `bd` emits JSON with stable fields, whereas the MCP server is built for agents to converse with. Same system, and the right choice is the machine-readable surface rather than the agent-readable one. That heuristic generalizes — **prefer the interface the vendor built for programs over the one they built for models.**

There is also an ordering point. MCP output shaped for LLM consumption is not a defect of MCP; it is what those tools are for. The mismatch is that Repo Governor is not an LLM consumer at the point where it matters. ADR-002 put the disposition in a deterministic program precisely so that model-shaped input could not reach it, and this decision is the same boundary drawn one layer out.

## Implementation Plan

1. Add `transport` to the manifest JSON Schema (ADR-004): `{kind: "native"|"cli"|"mcp", ...}`, defaulting to `native`.
2. Document the structured-content test in `references/providers.md` as the gate for choosing MCP transport.
3. Add a conformance check: an MCP-transport adapter must produce provenance citing the underlying object, not the tool call.
4. Build one MCP-transport adapter end-to-end to validate the design. **Not possible in the session that wrote this ADR** — no tracker MCP server was connected — so this step is genuinely unverified and should be done before the ADR is accepted.
5. When the Beads execution adapter is written, use the `bd` CLI, and record why in that adapter's docstring.

## Related Specification Sections

§10–17 Provider Model · §45–50 Conformance · §51 Security and Boundary Model · §54 Failure Conditions · §65 Future Candidate Capabilities

## Domain References

- ADR-002 (determinism), ADR-003 (adapter protocol), ADR-011 (zero dependencies), ADR-012 (untrusted input)
- Measured this session: `deepwiki.read_wiki_structure` → single prose string; `adr-analysis.generate_adrs_from_prd` → prompt template
- [Beads — MCP server, CLI, and Go library](https://deepwiki.com/steveyegge/beads) §8.1, §11.5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
