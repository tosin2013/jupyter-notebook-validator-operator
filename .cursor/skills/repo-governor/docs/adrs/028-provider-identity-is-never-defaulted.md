# 28. Provider Identity Is Never Defaulted

**Status**: Accepted
**Ratified**: 2026-08-17 by Tosin Akinosho (§68), under the [v0.1.0 architecture ratification review](RATIFICATION-v0.1.0.md).
**Date**: 2026-08-17
**Domain**: Provider abstraction / safety
**Extends**: [ADR-018](018-admission-signal-is-declared-not-assumed.md)
**Resolves**: issue 26
**Lands with**: [ADR-027](027-the-governed-repository-is-not-the-install-directory.md)

## Context

Two shipped adapters named the author's own repository as a fallback:

```python
NWO = os.environ.get("REPO_GOVERNOR_GH_REPO", "tosin2013/repo-governor")
```

`adapters/github-projects` and `adapters/decision-history-github`. On **any** other repository, absent explicit configuration, both answered about this project — with provenance that read as correct because it truthfully named the repository it had queried.

**This survived ADR-027's fix**, which is why it is a separate decision rather than a footnote to it. ADR-027 makes the engine *pass* an identity. A default fires whenever the identity is **not** passed: a direct adapter invocation, a conformance run, an integration written against the subprocess protocol, any path the engine does not thread. A default is not a resolution path — it is what happens when resolution does not occur.

The same shape one level down: `signal: label` was declarable while its *parameters* were not, so an adapter told to use labels picked `admitted` and `ready` in someone else's repository. ADR-018 refused exactly that reasoning for the signal and left the parameters guessing.

## Decision

**An adapter that cannot determine which system it is reading must fail, not guess.**

1. **No identity default.** Unset repository identity yields `PROVIDER_UNAVAILABLE` with a resolution naming the manifest field. This is the posture `adapters/decision-history-dolt` already took for a missing database, and the rule ADR-003 rule 6 states for absence versus unknown, applied to the *subject* of a query rather than its result.

2. **Declared parameters, not just declared signals.** If `signal: label` is declared, the label names must be declared too. A signal whose parameters are invented is a signal that was not really declared.

3. **The rule generalizes past these two adapters.** Any adapter reaching a system that could be one of several instances — a Jira site, a Linear workspace, a Postgres database — declares which one or refuses. Convenience defaults are acceptable for *behaviour*; they are never acceptable for *identity*, because a wrong behaviour is visible in the answer and a wrong identity is not.

4. **Checked bluntly.** `conformance/bindings.py` asserts no file under `adapters/` contains the author's repository slug — a plain substring test, deliberately. A clever check is easier to defeat and easier to argue with. This is why the comments explaining the removal do not quote the slug.

## Consequences

**Positive**

- The failure mode is now the correct one: a repository that has not declared its identity gets a typed unavailability, not a confident answer about a stranger's project.
- It composes with ADR-027. Together they are the whole of "Repo Governor can be pointed at a repository": `cwd` carries the repo-local providers, declared identity carries the remote ones.

**Negative**

- **Every manifest binding a GitHub-backed role must now declare `env.REPO_GOVERNOR_GH_REPO`.** This repository's own manifest was updated. Onboarding proposes bindings and should propose this too — it does not yet, so a freshly onboarded repository will need the field added by hand. Real friction, and the alternative is the defect.
- Conformance suites that drive adapters directly must set identity explicitly. Correct, and it makes the suites say what they are testing against.

**Not addressed**

Whether identity should be *verifiable* rather than merely declared — checking that the declared `owner/repo` matches the target's origin remote, so a copied manifest is refused rather than honoured. That is the fail-closed check ADR-027 also leaves open, and the two should be settled together.

## Related Specification Sections

§21 Repository Governance Manifest · §51 Security and Boundary Model · INV-013 · INV-014
