# ADR 059: Kubernetes Client Library Upgrade to v0.35.x for OCP 4.22

## Status
Accepted

Ratified 2026-09-16 by repository owner (tosin2013).

## Context

ADR-056 (Accepted) certifies OpenShift **4.20, 4.21, and 4.22** as the Tier 1 support window. However, ADR-056 explicitly states it "does not authorize a k8s.io, ENVTEST, or Operator SDK upgrade."

### Current dependency state

| Dependency | Current | Kubernetes | OCP match |
|---|---|---|---|
| `k8s.io/api` | v0.32.2 | 1.32 | 4.19 |
| `k8s.io/apimachinery` | v0.32.4 | 1.32 | 4.19 |
| `k8s.io/client-go` | v0.32.2 | 1.32 | 4.19 |
| `sigs.k8s.io/controller-runtime` | v0.19.4 | 1.31 | 4.18 |
| `ENVTEST_K8S_VERSION` | 1.31.0 | 1.31 | 4.18 |
| Go | 1.25 | — | — |

The operator binary installs and runs on OCP 4.22 today because Kubernetes APIs are forward-compatible. However, the client libraries lag three minor versions behind the certified ceiling (K8s 1.35 for OCP 4.22). This mismatch means:

1. **API surface**: New K8s 1.33–1.35 API types and fields are unavailable to the operator code.
2. **ENVTEST**: Tests run against K8s 1.31 binaries while the operator certifies 4.20–4.22 (K8s 1.33–1.35).
3. **Deprecation risk**: K8s libraries deprecate and remove APIs over ~4 minor versions; staying on v0.32 while targeting 1.35 stretches that window.

### Version mapping (verified from go.mod of each release)

| controller-runtime | k8s.io | Kubernetes | OCP |
|---|---|---|---|
| v0.19.4 (current) | v0.31.x | 1.31 | 4.18 |
| v0.20.x | v0.32.x | 1.32 | 4.19 |
| v0.21.x | v0.33.x | 1.33 | 4.20 |
| v0.22.x | v0.34.x | 1.34 | 4.21 |
| **v0.23.x** | **v0.35.x** | **1.35** | **4.22** |

The latest stable releases at time of writing:
- `k8s.io/api` v0.35.8
- `sigs.k8s.io/controller-runtime` v0.23.3

## Decision

Upgrade the Kubernetes client library stack to the v0.35.x family, aligning with OCP 4.22 / Kubernetes 1.35:

| Dependency | Current | Target |
|---|---|---|
| `k8s.io/api` | v0.32.2 | v0.35.8 |
| `k8s.io/apimachinery` | v0.32.4 | v0.35.x (latest patch) |
| `k8s.io/client-go` | v0.32.2 | v0.35.x (latest patch) |
| `k8s.io/apiextensions-apiserver` | v0.32.1 | v0.35.x (latest patch) |
| `sigs.k8s.io/controller-runtime` | v0.19.4 | v0.23.3 |
| `ENVTEST_K8S_VERSION` | 1.31.0 | 1.35.0 |
| `ENVTEST_VERSION` | release-0.19 | release-0.23 |
| Go | 1.25 | 1.25 (no change unless v0.35.x requires higher) |

### What this does not decide

- Operator SDK version upgrade (remains v1.37.0 unless incompatible with controller-runtime v0.23)
- CRD API version changes (stays `v1alpha1`)
- New feature work or behavioral changes to the operator
- `openshift/api` or `tektoncd/pipeline` dependency versions (updated as needed for compatibility)

## Consequences

### Positive

- Client libraries match the certified OCP 4.22 ceiling (K8s 1.35)
- ENVTEST runs against K8s 1.35, matching the highest certified platform
- Access to any new K8s 1.33–1.35 API types if needed in future work
- Reduced deprecation risk from running libraries 3 minors behind

### Negative

- Breaking API changes between k8s.io v0.32 and v0.35 may require code fixes
- controller-runtime v0.23 may introduce behavioral changes vs v0.19 (four minor jumps)
- Indirect dependencies (`openshift/api`, `tektoncd/pipeline`) may need compatible versions
- Brief CI instability while the upgrade is validated

### Risks

- **openshift/api compatibility**: The `github.com/openshift/api` module may not have a release compatible with k8s.io v0.35.x. If so, a replace directive or fork may be temporarily needed.
- **Tekton API compatibility**: `github.com/tektoncd/pipeline` v1.0.2 may conflict with k8s.io v0.35.x.
- **Operator SDK**: operator-sdk v1.37.0 was built for controller-runtime v0.19.x; `make bundle` and code generation may need a newer SDK.

## Implementation Notes

1. This ADR must be **Accepted** before implementation begins (#106).
2. Upgrade in a single PR to avoid partial-version mismatches.
3. Run `make manifests generate fmt vet lint test` after upgrade to catch breakages.
4. Validate on an OCP 4.22 cluster before merging.

## References

- ADR-056: OpenShift Support Window 4.20–4.22
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- GitHub issue #105: Architecture decision for this upgrade
- GitHub issue #106: Implementation of this upgrade
- [controller-runtime releases](https://github.com/kubernetes-sigs/controller-runtime/releases)
- [Kubernetes 1.35 release notes](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.35.md)

## Related ADRs

- ADR-002: Platform Version Support Strategy (Superseded by ADR-056)
- ADR-006: Version Support Roadmap and Testing Strategy
- ADR-029: Platform Version Dependency Review Process
- ADR-032: GitHub Actions CI Testing Against Kubernetes 1.31.10
- ADR-056: OpenShift Support Window 4.20–4.22

## Revision History

| Date | Author | Description |
|------|--------|-------------|
| 2026-09-16 | Tosin Akinosho / agent under #105 | Proposed; version mapping verified from go.mod |
