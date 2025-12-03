# FBC Bundle Structure Explained

**References:**
- [File-Based Catalogs](https://olm.operatorframework.io/docs/reference/file-based-catalogs/)
- [Catalog Templates](https://olm.operatorframework.io/docs/reference/catalog-templates/)

## Understanding Bundle vs Operator Images

### Key Concepts

1. **Bundle Image** (`quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18`)
   - Contains operator metadata (CSV, CRDs, RBAC)
   - Defines what the operator does
   - Platform-specific (OpenShift 4.18, 4.19, 4.20)

2. **Operator Image** (`quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes`)
   - Contains the actual operator binary/code
   - Runs as a pod in the cluster
   - Platform-specific (different Kubernetes/OpenShift APIs)

3. **Related Images**
   - All container images needed by the operator at runtime
   - Includes operator image, bundle image, sidecars (kube-rbac-proxy)
   - Critical for disconnected/air-gapped installations

## Two Approaches to Multi-Version Support

### Approach 1: Separate Bundles per OpenShift Version (Current)

**Structure:**
```yaml
# Three separate bundles, one per OpenShift version
- Bundle v1.0.5-ocp4.18 → Operator v1.0.5-ocp4.18-volumes
- Bundle v1.0.6-ocp4.19 → Operator v1.0.6-ocp4.19
- Bundle v1.0.7-ocp4.20 → Operator v1.0.7-ocp4.20
```

**Pros:**
- Clear separation of OpenShift versions
- Easy to test each version independently
- Explicit upgrade path between versions
- Matches our current branch structure (release-4.18, release-4.19, release-4.20)

**Cons:**
- More bundle images to maintain
- More complex catalog structure

**Use Case:** When operator has significant differences between OpenShift versions (API changes, feature differences)

### Approach 2: Single Bundle with Multiple Operator Images

**Structure:**
```yaml
# One bundle with multiple operator images
- Bundle v1.0.7 → relatedImages:
    - Operator v1.0.7-ocp4.18
    - Operator v1.0.7-ocp4.19
    - Operator v1.0.7-ocp4.20
```

**Pros:**
- Single bundle to maintain
- Simpler catalog structure
- All platform variants in one place

**Cons:**
- Bundle must be compatible with all OpenShift versions
- Harder to test platform-specific behavior
- Less clear upgrade path

**Use Case:** When operator code is identical across OpenShift versions (only runtime differences)

## Our Current FBC Structure (Approach 1)

```yaml
---
schema: olm.package
name: jupyter-notebook-validator-operator
defaultChannel: stable

---
schema: olm.channel
package: jupyter-notebook-validator-operator
name: stable
entries:
  - name: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
  - name: jupyter-notebook-validator-operator.v1.0.6-ocp4.19
    replaces: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
  - name: jupyter-notebook-validator-operator.v1.0.7-ocp4.20
    replaces: jupyter-notebook-validator-operator.v1.0.6-ocp4.19

---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.5-ocp4.18
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18
relatedImages:
  - name: operator
    image: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes
  - name: bundle
    image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18
  - name: kube-rbac-proxy
    image: gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.5-ocp4.18
```

## Why We Chose Approach 1

1. **Branch Structure Alignment**: We have separate release branches (release-4.18, release-4.19, release-4.20)
2. **API Differences**: OpenShift 4.18, 4.19, 4.20 may have different Kubernetes APIs
3. **Independent Testing**: Each version can be tested independently on its target cluster
4. **Clear Upgrade Path**: Users upgrade from 4.18 → 4.19 → 4.20 as they upgrade their clusters
5. **Volume Feature**: We added volume support in 4.18, need to ensure it works across versions

## Alternative: Approach 2 (For Future Consideration)

If we wanted to use a single bundle with multiple operator images:

```yaml
---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.7
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7
relatedImages:
  - name: operator-ocp4.18
    image: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18
  - name: operator-ocp4.19
    image: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.19
  - name: operator-ocp4.20
    image: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20
  - name: bundle
    image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7
  - name: kube-rbac-proxy
    image: gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.7
```

**Note:** This would require:
- Single CSV that works across all OpenShift versions
- Runtime platform detection in the operator
- More complex deployment logic

## Container Image Naming Convention

### Current (Approach 1)
```
Bundle Images:
- quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.5-ocp4.18
- quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6-ocp4.19
- quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.20

Operator Images:
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5-ocp4.18-volumes
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6-ocp4.19
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20
```

### Alternative (Approach 2)
```
Bundle Image:
- quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7

Operator Images:
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.19
- quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20
```

## Recommendation

**Continue with Approach 1** for the following reasons:

1. ✅ Matches our current development workflow
2. ✅ Clear separation of concerns
3. ✅ Easier to debug platform-specific issues
4. ✅ Explicit upgrade path
5. ✅ Better for testing on different clusters

**Consider Approach 2** in the future if:
- Operator code becomes identical across OpenShift versions
- We want to simplify catalog maintenance
- We implement runtime platform detection

## References

- **FBC Documentation**: https://olm.operatorframework.io/docs/reference/file-based-catalogs/
- **Catalog Templates**: https://olm.operatorframework.io/docs/reference/catalog-templates/
- **olm.bundle Schema**: https://olm.operatorframework.io/docs/reference/file-based-catalogs/#olmbundle
- **relatedImages**: Used for disconnected installations and image mirroring

