# OLM Strategy: Classic vs OLM v1

**Date**: 2026-09-16
**Status**: Active

## Summary

This operator uses **OLM Classic** (`registry+v1` bundle format) for distribution via OperatorHub.io and Red Hat's community-operators-prod. OLM v1 (GA since OCP 4.20) is a newer framework but is not used for this operator's distribution.

## What is OLM v1?

OLM v1 (Operator Lifecycle Manager v1) is the next-generation operator lifecycle framework in OpenShift. It became Generally Available in OCP 4.20 ([RHEA-2026:0449](https://access.redhat.com/errata/RHEA-2026:0449)).

### Key differences from OLM Classic

| Feature | OLM Classic | OLM v1 |
|---------|-------------|--------|
| API | `Subscription`, `CatalogSource` | `ClusterExtension` |
| Bundle format | `registry+v1` | `registry+v1` (same) |
| Catalog format | SQLite database | File-Based Catalog (FBC) |
| Console support | Full OperatorHub UI | CLI only (no Console UI in 4.20-4.22) |
| Bundle size | Constrained by etcd value size | No size constraint |
| Network policies | Not supported in bundles | Supported in bundles |
| Install modes | `OwnNamespace`, `SingleNamespace`, `MultiNamespace`, `AllNamespaces` | `AllNamespaces` only (4.20) |
| Upgrade control | `Subscription` with approval | `ClusterExtension` with version constraints |

### OLM v1 GA timeline

| OCP Version | OLM v1 Status |
|-------------|---------------|
| 4.18 | Technology Preview |
| 4.19 | Technology Preview |
| 4.20 | **GA** (ClusterExtension API) |
| 4.21 | GA (maturing) |
| 4.22 | GA (current EUS) |

## Why This Operator Uses OLM Classic

### 1. OperatorHub Distribution

Both OperatorHub.io and `community-operators-prod` (Red Hat's catalog) require OLM Classic bundles. There is no OLM v1 submission pipeline for community operators as of OCP 4.22.

### 2. OpenShift Console Integration

The OpenShift Console's OperatorHub UI only supports OLM Classic. OLM v1 operators must be installed via CLI (`oc apply -f clusterextension.yaml`), which increases friction for adoption.

### 3. Install Mode Flexibility

OLM Classic supports `OwnNamespace` and `SingleNamespace` install modes, which are important for multi-tenant clusters. OLM v1 in 4.20-4.22 only supports `AllNamespaces`.

### 4. Maturity

OLM Classic is a stable, well-understood framework. OLM v1, while GA, is still maturing. The `ClusterExtension` API may evolve in future OCP releases.

## Bundle Format Compatibility

The `registry+v1` bundle format used by this operator is compatible with both OLM Classic and OLM v1. If OLM v1 adoption increases and a submission pipeline becomes available, the same bundles can be used without format changes.

### Current bundle structure

```
bundle/
  manifests/
    jupyter-notebook-validator-operator.clusterserviceversion.yaml
    mlops.mlops.dev_notebookvalidationjobs.yaml
  metadata/
    annotations.yaml
  tests/
    scorecard/
      config.yaml
```

## Installing with OLM v1 (Manual)

While this operator is distributed via OLM Classic, users on OCP 4.20+ can install it using OLM v1 if they prefer CLI-based management:

```yaml
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: jupyter-notebook-validator-operator
spec:
  source:
    sourceType: Catalog
    catalog:
      packageName: jupyter-notebook-validator-operator
  install:
    namespace: jupyter-notebook-validator-operator-system
    serviceAccount:
      name: jupyter-notebook-validator-operator-installer
```

> **Note**: This requires a catalog source that includes this operator's bundle. The default Red Hat catalog includes operators from `community-operators-prod`.

## Future Considerations

- **OLM v1 Console support**: When the OpenShift Console adds OLM v1 support, the friction difference between Classic and v1 installation will decrease.
- **Community submission pipeline**: If `community-operators-prod` or OperatorHub.io adds an OLM v1 submission path, this operator could publish to both.
- **FBC migration**: The existing File-Based Catalog (FBC) support added in issue #40 already aligns with OLM v1's catalog format, easing future transition.

## References

- [OCP 4.20 Extensions overview](https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html/extensions/extensions-overview)
- [OLM v1 ClusterExtension API](https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/extensions/cluster-extensions)
- [OperatorHub.io](https://operatorhub.io/operator/jupyter-notebook-validator-operator)
- [ADR-007: Distribution and Catalog Strategy](../adrs/007-distribution-and-catalog-strategy.md)
- [Issue #40: FBC migration](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/40)
