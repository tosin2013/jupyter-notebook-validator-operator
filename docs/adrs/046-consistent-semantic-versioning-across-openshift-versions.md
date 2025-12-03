# ADR 046: Consistent Semantic Versioning Across OpenShift Versions

**Status**: Accepted  
**Date**: 2025-12-03  
**Deciders**: Development Team  
**Related**: FBC Multi-Version Upgrade Plan, ADR-045 (Long-term Strategic Deployment Plan)

## Context

We are implementing File-Based Catalogs (FBC) to support multiple OpenShift versions (4.18, 4.19, 4.20) with a single operator codebase. We need to establish a clear versioning strategy that:

1. Maintains consistency across OpenShift versions
2. Simplifies version management and upgrades
3. Follows semantic versioning principles
4. Makes it clear when operator functionality changes vs platform support changes

### Initial Approach (Rejected)

Our initial FBC structure used different semantic versions for each OpenShift version:
- `v1.0.5-ocp4.18` for OpenShift 4.18
- `v1.0.6-ocp4.19` for OpenShift 4.19
- `v1.0.7-ocp4.20` for OpenShift 4.20

**Problems with this approach:**
- Implies functional differences between versions when there are none
- Makes it unclear what version number represents (operator version vs platform version)
- Complicates upgrade planning (is 1.0.6 an upgrade from 1.0.5 or just a platform variant?)
- Breaks semantic versioning principles (patch versions should be compatible)

## Decision

**We will use consistent semantic versioning across all OpenShift versions, with the OpenShift version only appearing in container image tags.**

### Versioning Structure

```
Bundle Version: v1.0.7 (consistent across all platforms)
Bundle Names:
  - jupyter-notebook-validator-operator.v1.0.7-ocp4.18
  - jupyter-notebook-validator-operator.v1.0.7-ocp4.19
  - jupyter-notebook-validator-operator.v1.0.7-ocp4.20

Container Images:
  - quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18
  - quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.19
  - quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20

OLM Package Version: 1.0.7 (no OpenShift suffix)
```

### Upgrade Path

```
v1.0.7-ocp4.18 → v1.0.7-ocp4.19 → v1.0.7-ocp4.20
```

When we release version 1.0.8:
```
v1.0.8-ocp4.18 → v1.0.8-ocp4.19 → v1.0.8-ocp4.20
```

### FBC Catalog Structure

```yaml
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.7  # No OpenShift suffix
```

## Rationale

### 1. **Semantic Versioning Compliance**
- Version 1.0.7 represents the operator's functional version
- All OpenShift variants have the same functionality
- OpenShift version is a deployment detail, not a functional difference

### 2. **Clear Upgrade Semantics**
- Users understand that 1.0.7 → 1.0.8 is an operator upgrade
- OpenShift version changes (4.18 → 4.19) are platform upgrades
- No confusion about whether version numbers represent features or platforms

### 3. **Simplified Version Management**
- Single source of truth for operator version
- Easy to track what features are in what version
- Clear pattern for future releases

### 4. **Consistent Pattern for Future Releases**
- When we release 1.0.8, all platforms get 1.0.8
- When we add OpenShift 4.21 support, it gets the current operator version
- Pattern scales indefinitely

### 5. **OLM Best Practices**
- OLM `olm.package` version should represent functional version
- Platform-specific details belong in image tags and annotations
- Upgrade paths are clear and predictable

## Consequences

### Positive

1. **Clear Version Semantics**: Version numbers clearly represent operator functionality
2. **Simplified Upgrades**: Users know 1.0.7 → 1.0.8 is a functional upgrade
3. **Consistent Pattern**: Easy to apply to future releases
4. **Better Documentation**: Version numbers are self-explanatory
5. **Semantic Versioning**: Follows semver principles correctly

### Negative

1. **Bundle Name Redundancy**: Bundle names include OpenShift version but package version doesn't
2. **Migration Required**: Need to update existing 1.0.5 references to 1.0.7
3. **Documentation Updates**: All docs need to reflect new versioning

### Neutral

1. **Image Tags Still Platform-Specific**: Container images still have OpenShift version in tag
2. **FBC Complexity**: FBC still has three bundles, one per platform

## Implementation

### Phase 1: Update FBC Catalog (Completed)
- ✅ Updated `catalog/catalog.yaml` to use v1.0.7 for all bundles
- ✅ Changed `olm.package` version to `1.0.7` (no suffix)
- ✅ Kept OpenShift version in bundle names and image tags

### Phase 2: Update Documentation
- ✅ Update `docs/FBC-BUNDLE-STRUCTURE-EXPLAINED.md`
- ✅ Update `docs/FBC-MULTI-VERSION-UPGRADE-PLAN.md`
- ✅ Update phase guides (PHASE2, PHASE3, PHASE4)
- ✅ Create this ADR

### Phase 3: Build Process Updates
- Update Makefile VERSION to 1.0.7
- Update bundle CSV version to 1.0.7
- Build all three platform images with 1.0.7 tag

### Phase 4: Testing
- Test upgrade path: 1.0.7-ocp4.18 → 1.0.7-ocp4.19 → 1.0.7-ocp4.20
- Verify OLM recognizes version correctly
- Confirm no version conflicts

## Examples

### Current Release (1.0.7)
```bash
# Build all three platforms with same version
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.19
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.20
```

### Next Release (1.0.8)
```bash
# Same pattern, just increment version
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.8-ocp4.18
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.8-ocp4.19
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.8-ocp4.20
```

### Adding New Platform (4.21)
```bash
# Use current operator version
make docker-build IMG=quay.io/takinosh/jupyter-notebook-validator-operator:1.0.8-ocp4.21
```

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [OLM File-Based Catalogs](https://olm.operatorframework.io/docs/reference/file-based-catalogs/)
- [OLM Package Properties](https://olm.operatorframework.io/docs/reference/file-based-catalogs/#olmpackage)
- [FBC Multi-Version Upgrade Plan](../FBC-MULTI-VERSION-UPGRADE-PLAN.md)
- [FBC Bundle Structure Explained](../FBC-BUNDLE-STRUCTURE-EXPLAINED.md)

## Notes

This decision was made after reviewing the OLM documentation and recognizing that the `olm.package` version property should represent the operator's functional version, not the platform version. The OpenShift version is a deployment detail that belongs in image tags and bundle names, but not in the semantic version itself.

This pattern is consistent with how other multi-platform operators handle versioning in the Operator Hub.

