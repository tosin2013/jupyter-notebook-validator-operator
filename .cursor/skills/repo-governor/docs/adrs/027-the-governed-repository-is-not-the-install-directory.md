# 27. The Governed Repository Is Not the Install Directory

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Engine architecture / portability
**Amends**: [ADR-021](021-every-provider-resolved-through-the-manifest.md)
**Resolves**: issue 24

## Context

[ADR-001](001-agent-skill-as-primary-delivery-surface.md) ships this as a skill cloned into a skills directory and pointed at whatever repository the agent is working in. Under that delivery model, the engine governed **its own** repository and said nothing about it.

```console
$ cd ~/workspaces/mcp-adr-analysis-server
$ python3 ~/workspace/repo-governor/engine/completion.py 22
{"decision":"CONTINUE","authority":"AUTHORIZED","provenance":["tosin2013/repo-governor#issues.22"]}
```

The target's issue 22 is a closed test-coverage task. It answered about this project's ratification gate — confidently, with provenance that read as correct.

### The mechanism, which was not what the issue first said

The issue was filed as *"ROOT is derived from the engine's install path, so the engine cannot be told which repository it governs."* True in outcome, wrong about the cause. Every file-backed adapter defaults to a **relative** path:

```
adr  docs/adrs        acceptance-file  .repo-governor/acceptance        git  .
```

Relative to the working directory — which is the correct design. A repo-local provider *should* read the repository it is invoked in. `engine/bindings.py` then overrode all seven at once:

```python
p = subprocess.run(args, ..., cwd=ROOT, ...)      # ROOT = the engine's own directory
```

The adapters were never wrong. The engine defeated them.

### The companion defect

Provenance from repo-local providers was repository-relative — `docs/adrs`, `filesystem:package.json`. The same check run against two repositories produced **opposite answers and byte-identical provenance**, so a recorded decision could not say what it was about. Fixing the answer without fixing the record would have left the audit trail wrong while the runtime was right, which is worse: a wrong run is noticed, a wrong record is trusted.

## Decision

**`ROOT` is where the engine lives. `TARGET` is what is governed. They are different questions and the engine must not answer one with the other.**

1. **`ROOT` resolves adapter paths and the schema. Nothing else.**

2. **`TARGET` is declared by `REPO_GOVERNOR_TARGET`, else the git repository enclosing the working directory, else the working directory itself.** Never the install directory unless that is genuinely where you are standing. Falling back to the enclosing repository is what makes "clone the skill, run it here" behave the way the delivery model implies.

3. **The manifest is read from `TARGET`.** A governance manifest describes the repository it sits in (ADR-004); reading the engine's manifest while governing elsewhere binds providers nobody declared for that place. An absent manifest at the target is `AUTHORITY_SOURCE_MISSING` — the repository is un-onboarded, which is a typed and honest answer.

4. **Adapters are spawned with `cwd = TARGET`.** This repairs all seven repo-local providers with one change, rather than threading a path through each. It is smaller than the plumbing the issue originally proposed **because the adapters were already correct.**

5. **Provenance is qualified with the governed repository.** `_protocol.cite` prefixes refs with `REPO_GOVERNOR_SUBJECT` — the origin remote slug, or the directory name when there is no remote. A decision record now names its subject.

6. **Target resolution lives in `manifest.py`**, and `bindings.py` delegates to it, so the manifest and the adapters cannot disagree about which repository is being governed.

## Consequences

**Positive**

- The delivery model in ADR-001 actually works. Verified: pointed at a foreign repository the engine reads **1** decision and cites `elsewhere/other//docs/adrs`; pointed at itself it reads **24** and cites `tosin2013/repo-governor//docs/adrs`.
- Standing in an un-onboarded repository now yields `AUTHORITY_SOURCE_MISSING` rather than a confident answer about somewhere else.
- Four checks in `conformance/bindings.py` fail the build if this regresses, including **"governing one repository never cites another"** — the property that actually matters.

**Negative**

- **`REPO_GOVERNOR_SUBJECT` is derived from the git remote**, so two clones of the same repository are indistinguishable in provenance, and a repository with no remote is identified by directory name. Good enough to tell two different projects apart, not good enough to tell two checkouts apart. Stated rather than solved.
- Resolving the target costs a `git rev-parse` per invocation. Negligible against the subprocess it precedes.
- **A manifest whose `repository.id` disagrees with the repository it sits in is not yet refused.** Copying a manifest into the wrong repository still governs. That is a fail-closed check worth adding and is not in this decision.

**What this does not fix**

The remote adapters do not take `cwd`, so they needed their own change — [ADR-028](028-provider-identity-is-never-defaulted.md). The two had to land together: an engine passing identity to adapters that still default, or adapters demanding identity the engine never sends, each leaves the hole open.

## Related Specification Sections

§9 Product Architecture · §21 Repository Governance Manifest · §51 Security and Boundary Model · INV-005 · INV-013
