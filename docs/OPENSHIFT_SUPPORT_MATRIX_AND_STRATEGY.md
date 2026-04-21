# OpenShift Support Matrix and Migration Strategy

**Date:** April 21, 2026 (updated — original November 8, 2025)
**Status:** Active

## Executive Summary

Based on the official Red Hat OpenShift lifecycle data (updated April 2026 — OCP 4.21 is now GA):

**🎯 ACTIVE SUPPORT WINDOW: OCP 4.19 / 4.20 / 4.21 (rolling 3-version window; 4.18 moved to Maintenance-only)**

## Current OpenShift Landscape (April 2026)

### Active Versions

| Version | Status | GA Date | Full Support Ends | Maintenance Ends | EUS Term 1 Ends | EUS Term 2 Ends |
|---------|--------|---------|-------------------|------------------|-----------------|-----------------|
| **4.21** | ✅ **Full Support** | Apr 2026 | GA of 4.22 + 3mo | ~Oct 2027 | N/A | N/A |
| **4.20** | ✅ **Full Support** | Oct 21, 2025 | GA of 4.21 + 3mo | Apr 21, 2027 | Oct 21, 2027 | Oct 21, 2028 |
| **4.19** | ⚠️ **Maintenance** | Jun 17, 2025 | Jan 21, 2026 | Dec 17, 2026 | N/A | N/A |
| **4.18** | ⚠️ **Maintenance / EUS** | Feb 25, 2025 | Sep 17, 2025 | Aug 25, 2026 | Feb 25, 2027 | Feb 25, 2028 |
| **4.17** | ❌ **End of Life** | Oct 1, 2024 | May 25, 2025 | Apr 1, 2026 | N/A | N/A |

### Key Observations

1. **4.21 is the LATEST** (released April 2026 — GA)
2. **4.18 is in Maintenance/EUS Support** — operator support will be dropped when 4.22 ships
3. **4.20 is an EUS release** (even-numbered = Extended Update Support)
4. **4.18 is an EUS release** (even-numbered = Extended Update Support)
5. **Rolling 3-version active window: 4.19 / 4.20 / 4.21**

## Kubernetes Version Mapping

Based on research and OpenShift release patterns:

| OpenShift | Kubernetes | Go Version | k8s.io Version | Status |
|-----------|-----------|------------|----------------|--------|
| **4.18** | **1.31** | 1.21+ | **v0.31.x** | ✅ Maintenance |
| **4.19** | **1.32** | 1.23+ | **v0.32.x** | ✅ Full Support |
| **4.20** | **1.33** | 1.24+ | **v0.33.x** | ✅ Full Support |
| **4.21** | **1.34** | 1.24+ | **v0.34.x** | ✅ **GA (April 2026)** |

## Extended Update Support (EUS) Strategy

### What is EUS?

- **EUS releases**: All even-numbered releases (4.16, 4.18, 4.20, 4.22...)
- **Standard support**: 18 months (6 months Full + 12 months Maintenance)
- **EUS Term 1**: Additional 6 months (optional, included with Premium subscriptions)
- **EUS Term 2**: Additional 12 months (optional add-on)
- **Total EUS lifecycle**: Up to 36 months (3 years)

### EUS Benefits

1. **Longer support window**: 24-36 months vs 18 months
2. **Easier upgrades**: EUS-to-EUS upgrade paths (4.18 → 4.20 → 4.22)
3. **Stable production**: Fewer forced upgrades
4. **Worker node flexibility**: Streamlined worker node upgrades

## 🎯 STRATEGIC OPTIONS ANALYSIS

### Option 1: Target 4.18 (EUS) - CONSERVATIVE ⚠️

**Approach:**
- Develop on OpenShift 4.18
- Use k8s.io v0.31.x
- Support 4.18, 4.19, 4.20

**Pros:**
✅ Widest compatibility (3 versions)  
✅ EUS release with long support (until Feb 2028 with EUS Term 2)  
✅ Proven stable platform  

**Cons:**
❌ Already in Maintenance Support (Full Support ended Sep 2025)  
❌ Using older Kubernetes 1.31 APIs  
❌ Missing latest OpenShift features  
❌ Will need upgrade soon to stay current  

**Timeline:**
- **Nov 2025**: Initial development on 4.18
- **Apr 2026**: 4.21 GA — 4.18 now clearly dated; rolling window shifts to 4.19/4.20/4.21
- **Aug 2026**: 4.18 Maintenance Support ends
- **Feb 2027**: 4.18 EUS Term 1 ends (if purchased)
- **Feb 2028**: 4.18 EUS Term 2 ends (if purchased)

### Option 2: Target 4.20 (EUS) - RECOMMENDED ✅

**Approach:**
- Develop on OpenShift 4.20 (current latest)
- Use k8s.io v0.33.x
- Support 4.20, 4.21 (when released)
- Provide backward compatibility guidance for 4.18/4.19

**Pros:**
✅ **Current latest release** (Oct 2025)  
✅ **Full Support until ~Jul 2026** (4.21 GA Apr 2026 + 3 months)  
✅ **EUS release** with long support (until Oct 2028 with EUS Term 2)  
✅ **Latest Kubernetes 1.33 APIs**  
✅ **Latest OpenShift features**  
✅ **Forward compatible** with 4.21 (now GA)  
✅ **EUS-to-EUS upgrade path** (4.20 → 4.22 → 4.24)  

**Cons:**
⚠️ May not work on 4.18/4.19 without modifications  
⚠️ Requires k8s.io v0.33.x (need to verify operator-sdk compatibility)  
⚠️ Newer = less battle-tested (but only 18 days old)  

**Timeline:**
- **Oct 2025**: Develop on 4.20
- **Apr 2026**: 4.21 GA — 4.20 compatibility confirmed
- **Apr 2027**: 4.20 Maintenance Support ends
- **Oct 2027**: 4.20 EUS Term 1 ends
- **Oct 2028**: 4.20 EUS Term 2 ends

### Option 3: Target 4.19 - MIDDLE GROUND ⚖️

**Approach:**
- Develop on OpenShift 4.19
- Use k8s.io v0.32.x
- Support 4.19, 4.20, 4.21

**Pros:**
✅ Full Support until Jan 2026  
✅ Kubernetes 1.32 APIs  
✅ Forward compatible with 4.20 and 4.21  
✅ More battle-tested than 4.20 (5 months old)  

**Cons:**
❌ **NOT an EUS release** (odd-numbered)  
❌ **Shorter support window** (18 months total)  
❌ **Maintenance Support ends Dec 2026** (13 months from now)  
❌ Not the latest  
❌ Will need upgrade to 4.20 or 4.21 soon  

**Timeline:**
- **Jun 2025**: Develop on 4.19
- **Jan 2026**: Full Support ends
- **Apr 2026**: 4.21 GA released
- **Dec 2026**: 4.19 Maintenance Support ends (FORCED UPGRADE)

## 📊 Comparison Matrix

| Criteria | 4.18 (EUS) | 4.19 | 4.20 (EUS) |
|----------|------------|------|------------|
| **Current Status** | ⚠️ Maintenance | ✅ Full Support | ✅ Full Support |
| **Kubernetes** | 1.31 | 1.32 | 1.33 |
| **k8s.io Version** | v0.31.x | v0.32.x | v0.33.x |
| **EUS Release** | ✅ Yes | ❌ No | ✅ Yes |
| **Max Support** | 36 months | 18 months | 36 months |
| **Support Ends** | Feb 2028 | Dec 2026 | Oct 2028 |
| **Latest Features** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Stability** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Future-Proof** | ⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Upgrade Path** | 4.20→4.22 | 4.20→4.21 | 4.22→4.24 |

## 🎯 UPDATED RECOMMENDATION (April 2026)

### **Active Support Window: OCP 4.19 / 4.20 / 4.21**

**Primary Target:** OpenShift 4.20 (Kubernetes 1.33, k8s.io v0.33.x) — EUS, long support  
**Full Support:** OpenShift 4.21 (Kubernetes 1.34, k8s.io v0.34.x) — newest GA  
**Maintenance Support:** OpenShift 4.19 — active but entering end-of-life window  
**Dropped:** 4.18 removed from active CI; maintained via EUS track only for existing users

### Rationale

1. **4.21 is now GA** (released April 2026) — rolling window shifts to 4.19/4.20/4.21
2. **EUS Benefits**: 4.20 has long support (up to Oct 2028 with EUS Term 2)
3. **OCP-stream versioning**: v1.0.8 → OCP 4.19+, v1.0.9 → OCP 4.20+, v1.0.10 → OCP 4.21+
4. **EUS Upgrade Path**: Clean EUS-to-EUS upgrades (4.20 → 4.22 → 4.24)

### Implementation Strategy (Current)

**Phase 1: Infrastructure (Apr 2026)**
- ✅ Standardize IMAGE_TAG_BASE to quay.io/takinosh/
- ✅ Upgrade GO_VERSION to 1.24 across all CI
- ✅ Add release-4.21 to bundle-validation.yaml triggers

**Phase 2: Feature Development (Apr - Jun 2026)**
- Implement issues #7 (ADR-021), #8 (ADR-022), #9 (ADR-030)
- Generate bundle for v1.0.8 (OCP 4.19 stream)
- Submit v1.0.7 OperatorHub PR first

**Phase 3: OCP 4.21 Track (Jun - Sep 2026)**
- Create release-4.21 branch
- Validate scorecard on OCP 4.21
- Release v1.0.10 on OCP 4.21 stream
- Plan EUS-to-EUS upgrade (4.20 → 4.22) in 2027

## Multi-Version Support Strategy

### Operator Support Matrix

```yaml
# Recommended in operator metadata
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  annotations:
    # Declare supported OpenShift versions
    com.redhat.openshift.versions: "v4.19-v4.21"
spec:
  minKubeVersion: 1.32.0  # Kubernetes 1.32 (OpenShift 4.19)
```

### Version Detection in Code

```go
// pkg/platform/detector.go
func (d *Detector) GetRecommendedBuildStrategy(ctx context.Context) (string, error) {
    info, err := d.GetOpenShiftInfo(ctx)
    if err != nil {
        return "", err
    }
    
    // Detect OpenShift version
    version, err := d.GetOpenShiftVersion(ctx)
    if err != nil {
        return "s2i", nil // Default to S2I
    }
    
    // Recommend strategy based on version
    switch {
    case version >= "4.20":
        // 4.20+ has latest Tekton
        return "tekton", nil
    case version >= "4.18":
        // 4.18-4.19 use S2I
        return "s2i", nil
    default:
        return "s2i", nil
    }
}
```

## Dependency Resolution Plan

### Step 1: Research Compatible Versions

```bash
# Find OpenShift API commit for 4.20 (Oct 2025)
# Look for commits around Oct 21, 2025 that use k8s.io v0.33.x
git clone https://github.com/openshift/api
cd api
git log --since="2025-10-01" --until="2025-11-01" --oneline
# Find commit with k8s.io v0.33.x in go.mod
```

### Step 2: Find Tekton Version

```bash
# Find Tekton Pipeline version compatible with k8s.io v0.33.x
# Check Tekton releases from Oct-Nov 2025
```

### Step 3: Update go.mod

```go
// go.mod
module github.com/tosin2013/jupyter-notebook-validator-operator

go 1.21

require (
    // Kubernetes 1.33 (OpenShift 4.20)
    k8s.io/api v0.33.0
    k8s.io/apimachinery v0.33.0
    k8s.io/client-go v0.33.0
    
    // OpenShift API (compatible with k8s 1.33 / OpenShift 4.20)
    github.com/openshift/api v0.0.0-<commit-hash> // Oct 2025 commit
    
    // Tekton Pipeline (compatible with k8s 1.33)
    github.com/tektoncd/pipeline v<version>
)
```

## Next Steps

1. ✅ **Immediate**: Research OpenShift API and Tekton versions for k8s.io v0.33.x
2. ✅ **This Week**: Update go.mod with compatible versions
3. ✅ **This Week**: Build and test pkg/build/ strategies
4. ✅ **Next Week**: Deploy to OpenShift 4.20 cluster for integration testing
5. ✅ **Dec 2025**: Test on OpenShift 4.19 for backward compatibility
6. ✅ **Feb 2026**: Test on OpenShift 4.21 when released

## References

- OpenShift Life Cycle: https://access.redhat.com/support/policy/updates/openshift
- OpenShift Product Life Cycles: https://access.redhat.com/product-life-cycles?product=Red%20Hat%20OpenShift%20Container%20Platform
- Kubernetes Version Skew: https://kubernetes.io/releases/version-skew-policy/
- Operator SDK: https://sdk.operatorframework.io/

## Conclusion

**Target OpenShift 4.20 (EUS)** is the strategic choice that balances:
- ✅ Current technology (latest GA release)
- ✅ Long-term support (EUS with 36-month lifecycle)
- ✅ Latest features (Kubernetes 1.33, OpenShift 4.20 capabilities)
- ✅ Future-proof (forward compatible with 4.21, EUS upgrade path to 4.22)
- ✅ Market timing (will be standard by production release)

This positions your operator for success through 2028 with minimal forced upgrades.

