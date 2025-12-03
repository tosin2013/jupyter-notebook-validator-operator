# Bundle Version Fix Plan

## Problem Summary

**Maintainer Feedback**: All three bundles (1.0.7-ocp4.18, 1.0.7-ocp4.19, 1.0.7-ocp4.20) are trying to replace the same previous version, creating conflicts and dangling bundles.

## Solution: Consecutive Semantic Versioning

Use different semantic versions for each OpenShift version to create a consecutive upgrade chain.

## Version Mapping

| Bundle Version | OpenShift Version | Operator Image Tag | Bundle Image Tag | Replaces |
|---------------|-------------------|-------------------|------------------|----------|
| **1.0.7** | 4.18+ | `1.0.7-ocp4.18` | `1.0.7` | `1.0.3` |
| **1.0.8** | 4.19+ | `1.0.8-ocp4.19` | `1.0.8` | `1.0.7` |
| **1.0.9** | 4.20+ | `1.0.9-ocp4.20` | `1.0.9` | `1.0.8` |

## Upgrade Chain

```
1.0.3 → 1.0.7 (OCP 4.18) → 1.0.8 (OCP 4.19) → 1.0.9 (OCP 4.20)
```

## Implementation Steps

### Step 1: Update release-4.18 Branch (Version 1.0.7)

```bash
git checkout release-4.18
```

**Changes Needed:**
- ✅ Makefile VERSION: Already `1.0.7`
- ✅ Operator image: Already `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7-ocp4.18`
- ✅ Bundle version: Already `1.0.7`
- ✅ Bundle image: Already `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7-ocp4.18`
- ❌ **Need to change bundle image tag**: `1.0.7-ocp4.18` → `1.0.7` (no OpenShift suffix)
- ❌ **Need to add replaces field** in CSV: `replaces: jupyter-notebook-validator-operator.v1.0.3`

**Actions:**
1. Rebuild bundle with tag `1.0.7` (no OpenShift suffix)
2. Add `replaces` field to CSV
3. Push new bundle image

### Step 2: Update release-4.19 Branch (Version 1.0.8)

```bash
git checkout release-4.19
```

**Changes Needed:**
- ❌ Makefile VERSION: Change `1.0.7` → `1.0.8`
- ❌ Operator image: Rebuild as `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.8-ocp4.19`
- ❌ Bundle version: Change to `1.0.8`
- ❌ Bundle image: Build as `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.8`
- ❌ **Add replaces field** in CSV: `replaces: jupyter-notebook-validator-operator.v1.0.7`

**Actions:**
1. Update Makefile VERSION to `1.0.8`
2. Rebuild operator image with tag `1.0.8-ocp4.19`
3. Generate bundle with version `1.0.8`
4. Add `replaces` field to CSV
5. Build bundle image with tag `1.0.8` (no OpenShift suffix)
6. Push both images

### Step 3: Update release-4.20 Branch (Version 1.0.9)

```bash
git checkout release-4.20
```

**Changes Needed:**
- ❌ Makefile VERSION: Change `1.0.7` → `1.0.9`
- ❌ Operator image: Rebuild as `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.9-ocp4.20`
- ❌ Bundle version: Change to `1.0.9`
- ❌ Bundle image: Build as `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.9`
- ❌ **Add replaces field** in CSV: `replaces: jupyter-notebook-validator-operator.v1.0.8`

**Actions:**
1. Update Makefile VERSION to `1.0.9`
2. Rebuild operator image with tag `1.0.9-ocp4.20`
3. Generate bundle with version `1.0.9`
4. Add `replaces` field to CSV
5. Build bundle image with tag `1.0.9` (no OpenShift suffix)
6. Push both images

### Step 4: Create FBC Catalog

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
  - name: jupyter-notebook-validator-operator.v1.0.7
  - name: jupyter-notebook-validator-operator.v1.0.8
    replaces: jupyter-notebook-validator-operator.v1.0.7
  - name: jupyter-notebook-validator-operator.v1.0.9
    replaces: jupyter-notebook-validator-operator.v1.0.8

---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.7
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.7

---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.8
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.8
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.8

---
schema: olm.bundle
name: jupyter-notebook-validator-operator.v1.0.9
package: jupyter-notebook-validator-operator
image: quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.9
properties:
  - type: olm.package
    value:
      packageName: jupyter-notebook-validator-operator
      version: 1.0.9
```

## Testing Plan

1. **Test on OpenShift 4.20.5 cluster** (current cluster)
   - Deploy version 1.0.9
   - Run tier5-volumes test
   - Verify Tekton v1 API support

2. **Test Upgrade Chain** (if possible)
   - Install 1.0.7
   - Upgrade to 1.0.8
   - Upgrade to 1.0.9

## Summary

**Key Changes:**
- Bundle image tags: Remove OpenShift suffix (`1.0.7-ocp4.18` → `1.0.7`)
- Operator image tags: Keep OpenShift suffix for clarity
- Use consecutive versions: 1.0.7, 1.0.8, 1.0.9
- Add `replaces` field to each CSV

**Result:**
- ✅ Consecutive upgrade chain
- ✅ No dangling bundles
- ✅ All versions propagate to Operator Hub
- ✅ Compatible with automation

