# 5. Deny-by-Default Permission Model with No Capability Inference

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Security & authorization

## Context

§22 requires explicit per-provider permissions and states that "Repo Governor must not infer write authority from available credentials." INV-014 generalizes it: "A provider may technically support writes, status changes, or automation. That technical ability does not grant Repo Governor permission to use those capabilities."

This is the standard confused-deputy problem. A `LINEAR_API_KEY` in the environment grants the *process* write access to Linear. Nothing about its presence expresses a human intent that Repo Governor should write to Linear. Conflating capability with permission is how agents take actions nobody authorized — and the April 2026 database deletion is that failure at its logical end: the agent had the capability, inferred the permission, and executed.

§22 lists `read`, `write`, `execute` as the minimum and six optional verbs, but does not state the default or how an unstated permission resolves.

## Decision

**Every permission is denied unless explicitly granted in the manifest. Absence is denial, never inheritance and never inference.**

1. **Deny by default.** An unlisted provider, an unlisted verb, or a malformed permission block resolves to deny. There is no "sensible default" branch.
2. **Permissions are per role-binding, not per system.** If GitHub fills both `roadmap_authority` and `repository`, each gets its own permission block. Granting write to one grants nothing to the other.
3. **Credentials are never evidence of permission.** The engine does not check whether a token exists when deciding what is permitted. Permission is read from the manifest; the credential is only consulted at the moment a permitted operation executes.
4. **Read is the presumed grant; write is exceptional.** Repo Governor's entire purpose is to rule, not to act (ADR-002). The only write that MVP contemplates is discovery capture to an `ExecutionStateProvider` — §21's example grants exactly that and nothing else. Any other write grant should be treated as a design smell and challenged in review.
5. **Permission failures are dispositions, not exceptions.** If a governance path requires a write the manifest denies, the outcome is a disposition explaining the shortfall — not a crash, and not a silent skip that makes the decision look complete when it is not.
6. **`execute` is reserved and unimplemented at v1.** Listing it in §22 without semantics invites divergent interpretation. The loader rejects it until it is defined.

## Consequences

**Positive**

- INV-014 is enforced by the type system rather than by discipline.
- A reviewer reading the manifest sees the complete set of things Repo Governor may do to external systems. Nothing is implicit.
- Least privilege (§51) follows automatically from the default rather than requiring per-deployment care.

**Negative**

- Verbose manifests when many verbs are needed. Mitigated by the fact that read-only is the overwhelmingly common configuration, so real manifests stay short.
- Deny-by-default produces confusing failures for new users — a governance path silently unavailable because a verb was never granted. Error messages must name the exact missing grant and the manifest path to add it. This is a documentation and DX obligation, not an optional polish item.
- Any future automation feature (auto-capture, auto-transition) collides with this default and will require deliberate opt-in design rather than a config flag.

## Domain Considerations

Cedar's default-deny posture in Bedrock AgentCore is the reference implementation of this principle for agent authorization, and it holds for the same reason here: in an agentic system the cost asymmetry between a wrongly-denied read and a wrongly-permitted write is enormous.

Separating credential possession from permission grant also contains blast radius in the multi-repository case (§51: "prevent cross-repository state leakage"). One `GITHUB_TOKEN` may span many repositories; the manifest is per-repository, so the permission grant is per-repository even when the credential is not.

## Implementation Plan

1. Model permissions as an explicit closed enum; reject unknown verbs at load time.
2. Implement the permission gate as a single chokepoint every adapter invocation passes through — no adapter may be called except through it.
3. Implement secret-shaped-value detection in the manifest loader (entropy + known token prefixes); fail closed on match.
4. Write negative tests: for each verb, assert the operation fails when the grant is absent.
5. Write the error-message catalogue for denied operations, each naming the missing grant and its manifest location.

## Related Specification Sections

§21 Repository Governance Manifest · §22 Permission Model · §51 Security and Boundary Model · INV-014

## Domain References

- [Why Policy in Amazon Bedrock AgentCore chose Cedar](https://aws.amazon.com/blogs/security/why-policy-in-amazon-bedrock-agentcore-chose-cedar-for-securing-agentic-workflows/)
- [AI Agent Security Starts with Scope Control — Cloud Security Alliance](https://cloudsecurityalliance.org/blog/2026/05/12/ai-agent-security-starts-with-scope-control)
- `docs/research/2026-08-17-external-landscape.md` §2, §5

---

_`§NN` and `INV-NNN` citations above resolve against [`docs/reference/`](../reference/) — see its [section map](../reference/README.md#section-map). Original numbering from PRD v0.2, extracted 2026-08-17._
