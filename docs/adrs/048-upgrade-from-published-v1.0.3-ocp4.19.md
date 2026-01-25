# ADR-048: Upgrade from Published v1.0.3-ocp4.19

**Status**: Accepted  
**Date**: 2026-01-25  
**Deciders**: Tosin Akinosho  
**Context**: Operator Hub Bundle Upgrade Path Fix  
**Supersedes**: ADR-047

## Context and Problem Statement

The operator is currently published on OperatorHub.io with the following versions:
- `jupyter-notebook-validator-operator.v1.0.2`
- `jupyter-notebook-validator-operator.v1.0.3-ocp4.19` (replaces v1.0.2)

ADR-047 proposed versions 1.0.7/1.0.8/1.0.9 with `replaces: v1.0.3`, but **the published
CSV name includes the `-ocp4.19` suffix**. This means the proposed upgrade chain would
be broken because there is no `v1.0.3` (without suffix) in the published catalog.

### Published Upgrade Chain (Current)
```
v1.0.2 → v1.0.3-ocp4.19 (published on OperatorHub.io)
```

### ADR-047's Broken Proposal
```
v1.0.3 → v1.0.7 → v1.0.8 → v1.0.9  ❌ BROKEN (v1.0.3 doesn't exist)
```

## Decision Drivers

1. **Upgrade Path Must Connect**: New version must replace the *actual* published CSV name
2. **Simplicity**: Start with a single new version, add more later if needed
3. **Multi-OpenShift Support**: Single bundle supporting all OpenShift versions
4. **Clean Registry**: Remove confusing old images from Quay

## Considered Options

### Option 1: Fix replaces to v1.0.3-ocp4.19 (CHOSEN)
Create v1.0.4 with `replaces: jupyter-notebook-validator-operator.v1.0.3-ocp4.19`

**Pros:**
- Direct, correct upgrade path from published version
- Simple single version release
- Can add 1.0.5, 1.0.6, etc. later for new features

**Cons:**
- Ties upgrade chain to the OCP-suffixed version name

### Option 2: Use skipRange only
Use `skipRange: ">=1.0.0 <1.0.4"` without replaces field

**Pros:**
- Flexible, skips all old versions

**Cons:**
- Less explicit upgrade path
- May not be recommended by community-operators

### Option 3: Start fresh with 2.0.0
Major version bump to break from old naming

**Pros:**
- Clean slate

**Cons:**
- Existing users won't get automatic upgrades
- Loses continuity

## Decision Outcome

**Chosen Option**: Option 1 - Create v1.0.4 with explicit replaces

### New Version: 1.0.4

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: jupyter-notebook-validator-operator.v1.0.4
  annotations:
    com.redhat.openshift.versions: "v4.18-v4.20"
    containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.4
spec:
  replaces: jupyter-notebook-validator-operator.v1.0.3-ocp4.19
  version: 1.0.4
```

### Upgrade Chain (Corrected)
```
v1.0.2 → v1.0.3-ocp4.19 → v1.0.4 (NEW)
                              ↓
                          v1.0.5 (future)
                              ↓
                          v1.0.6 (future)
```

## Implementation Plan

### Phase 1: Quay Cleanup
Delete all images except:
- `jupyter-notebook-validator-operator:1.0.3-ocp4.19`
- `jupyter-notebook-validator-operator-bundle:v1.0.3-ocp4.19`

Script: `scripts/cleanup-quay-images.sh`

### Phase 2: Create v1.0.4 Bundle

1. Update CSV on main branch:
   - Name: `jupyter-notebook-validator-operator.v1.0.4`
   - Replaces: `jupyter-notebook-validator-operator.v1.0.3-ocp4.19`
   - containerImage: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.4`
   - OpenShift versions: `v4.18-v4.20` (broad support)

2. Build and push images via GitHub Actions:
   - Operator: `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.4`
   - Bundle: `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.4`

### Phase 3: Local Validation

Run pre-submission tests:
```bash
./scripts/full-pre-submission-test.sh --version 1.0.4
```

### Phase 4: Community Operators Submission

1. Update local `community-operators` fork:
   - Remove: `1.0.7/`, `1.0.8/`, `1.0.9/` directories
   - Add: `1.0.4/` directory with new bundle

2. Verify single version in submission

3. Submit PR to `k8s-operatorhub/community-operators`

## Versioning Strategy Going Forward

Future releases will follow simple semantic versioning:
- **1.0.4**: Multi-OpenShift support (4.18-4.20)
- **1.0.5**: Bug fixes, minor improvements
- **1.0.6**: New features
- **1.1.0**: Breaking changes or major new features

Each version will:
- Use clean version numbers (no OCP suffix)
- Declare OpenShift compatibility via `com.redhat.openshift.versions` annotation
- Follow consecutive `replaces` chain

## Consequences

### Positive
- ✅ Correctly connects to published OperatorHub version
- ✅ Simpler single-version release process
- ✅ Cleaner Quay registry (old images deleted)
- ✅ Clear path for future versions

### Negative
- ⚠️ Lost work on 1.0.7/1.0.8/1.0.9 bundles (can be reused later)
- ⚠️ Need to wait for CI to build new images

### Neutral
- 📝 Community operators submission will be simpler (one version)
- 📝 Future versions can add OpenShift-specific features if needed

## References

- OperatorHub.io: https://operatorhub.io/operator/jupyter-notebook-validator-operator
- ADR-047: Fix Bundle Versioning (superseded)
- Community Operators Repository: https://github.com/k8s-operatorhub/community-operators
