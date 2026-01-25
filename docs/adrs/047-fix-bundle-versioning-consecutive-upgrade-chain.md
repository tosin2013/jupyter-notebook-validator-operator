# ADR-047: Fix Bundle Versioning for Consecutive Upgrade Chain

**Status**: Superseded by ADR-048  
**Date**: 2025-12-03  
**Superseded Date**: 2026-01-25  
**Deciders**: Tosin Akinosho, Community Operators Maintainer Feedback  
**Context**: Operator Hub Bundle Submission

> **Note**: This ADR is superseded by ADR-048. The upgrade chain proposed here 
> (`replaces: v1.0.3`) does not connect to the actual published version on 
> OperatorHub.io which is `v1.0.3-ocp4.19`. See ADR-048 for the corrected approach.

## Context and Problem Statement

The community-operators maintainer identified a critical issue with our bundle versioning strategy:

> "All three bundles in the PR are replacing the same bundle 1.0.3-ocp4.19 - resulting in a situation when only the highest version 1.0.4-ocp4.20 is valid and propagated to the operator hub. The remaining bundle versions are dangling (1.0.4-ocp4.18 and 1.0.4-ocp4.19) as it claims to replace version which is already replaced by the newest bundle version 1.0.4-ocp4.20"

### Current (Broken) Approach

```
1.0.3-ocp4.19 ← 1.0.7-ocp4.18 (replaces 1.0.3-ocp4.19)
1.0.3-ocp4.19 ← 1.0.7-ocp4.19 (replaces 1.0.3-ocp4.19) ❌ CONFLICT
1.0.3-ocp4.19 ← 1.0.7-ocp4.20 (replaces 1.0.3-ocp4.19) ❌ CONFLICT
```

**Problem**: Multiple bundles trying to replace the same version creates:
- Dangling bundle versions (only highest version is valid)
- Broken upgrade automation in Operator Hub
- Only 1.0.7-ocp4.20 would be propagated, leaving 4.18 and 4.19 users stranded

## Decision Drivers

1. **Operator Hub Requirements**: Consecutive upgrade chain required
2. **Multi-Version Support**: Need to support OpenShift 4.18, 4.19, 4.20
3. **User Experience**: Clear upgrade path for users on different OpenShift versions
4. **Automation Compatibility**: Must work with Operator Hub automation

## Considered Options

### Option 1: Single Bundle with Multi-Version Support (REJECTED)
- Use one bundle that works across all OpenShift versions
- **Pros**: Simple upgrade chain
- **Cons**: 
  - Loses version-specific optimizations
  - Can't leverage version-specific features (e.g., Tekton v1 API in 4.20)
  - Doesn't match our branch structure

### Option 2: Consecutive Semantic Versioning (RECOMMENDED)
- Use different semantic versions for each OpenShift version
- Create consecutive upgrade chain

**Approach A: Increment Patch Version**
```
1.0.3-ocp4.19 → 1.0.7 (for 4.18) → 1.0.8 (for 4.19) → 1.0.9 (for 4.20)
```

**Approach B: Increment Minor Version**
```
1.0.3-ocp4.19 → 1.1.0 (for 4.18) → 1.2.0 (for 4.19) → 1.3.0 (for 4.20)
```

### Option 3: Drop OpenShift Version from Bundle Name (RECOMMENDED)
- Use pure semantic versioning without OpenShift suffix
- Document OpenShift compatibility in CSV metadata
```
1.0.3 → 1.0.7 → 1.0.8 → 1.0.9
```

## Decision Outcome

**Chosen Option**: **Option 2 (Approach A) + Option 3 Combined**

Use consecutive semantic versioning WITHOUT OpenShift suffix in bundle names:
- **1.0.7** - Supports OpenShift 4.18+ (baseline)
- **1.0.8** - Supports OpenShift 4.19+ (enhanced features)
- **1.0.9** - Supports OpenShift 4.20+ (Tekton v1 API)

### Upgrade Chain
```
1.0.3 → 1.0.7 → 1.0.8 → 1.0.9
```

### CSV Metadata
Each bundle CSV will specify OpenShift compatibility:
```yaml
metadata:
  annotations:
    com.redhat.openshift.versions: "v4.18"  # for 1.0.7
    com.redhat.openshift.versions: "v4.19"  # for 1.0.8
    com.redhat.openshift.versions: "v4.20"  # for 1.0.9
```

### Container Image Tags
Keep OpenShift version in image tags for clarity:
- Operator: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18`
- Bundle: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7`

## Implementation Plan

### Phase 1: Update Bundle Versions
1. **1.0.7** (release-4.18 branch)
   - Bundle name: `jupyter-notebook-validator-operator.v1.0.7`
   - Replaces: `jupyter-notebook-validator-operator.v1.0.3`
   - OpenShift: 4.18+

2. **1.0.8** (release-4.19 branch)
   - Bundle name: `jupyter-notebook-validator-operator.v1.0.8`
   - Replaces: `jupyter-notebook-validator-operator.v1.0.7`
   - OpenShift: 4.19+

3. **1.0.9** (release-4.20 branch)
   - Bundle name: `jupyter-notebook-validator-operator.v1.0.9`
   - Replaces: `jupyter-notebook-validator-operator.v1.0.8`
   - OpenShift: 4.20+

### Phase 2: Update FBC Catalog
```yaml
---
schema: olm.channel
package: jupyter-notebook-validator-operator
name: stable
entries:
  - name: jupyter-notebook-validator-operator.v1.0.7
  - name: jupyter-notebook-validator-operator.v1.0.8
    replaces: jupyter-notebook-validator-operator.v1.0.7
  - name: jupyter-notebook-validator-operator.v1.0.9
    replaces: jupyter-notebook-validator-operator.v1.0.8
```

## Consequences

### Positive
- ✅ Consecutive upgrade chain (no conflicts)
- ✅ All bundles propagate to Operator Hub
- ✅ Clear upgrade path for users
- ✅ Compatible with Operator Hub automation
- ✅ Semantic versioning reflects feature additions

### Negative
- ⚠️ Need to rebuild all three bundles with new versions
- ⚠️ Image tags still have OpenShift version (minor inconsistency)
- ⚠️ Users must understand version-to-OpenShift mapping

### Neutral
- 📝 Documentation must clearly map versions to OpenShift versions
- 📝 Release notes must explain version strategy

## References

- Community Operators Maintainer Feedback (2025-12-03)
- ADR-046: Consistent Semantic Versioning Across OpenShift Versions
- OLM Bundle Documentation: https://olm.operatorframework.io/docs/concepts/olm-architecture/operator-catalog/creating-an-update-graph/

