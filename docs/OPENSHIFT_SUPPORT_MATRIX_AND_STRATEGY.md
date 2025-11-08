# OpenShift Support Matrix and Migration Strategy

**Date:** November 8, 2025  
**Status:** 🎯 STRATEGIC DECISION REQUIRED

## Executive Summary

Based on the official Red Hat OpenShift lifecycle data (as of November 8, 2025), here's the **critical strategic recommendation** for your Jupyter Notebook Validator Operator development:

**🎯 RECOMMENDATION: Target OpenShift 4.18 (EUS) with a migration plan to 4.20 (EUS) and 4.21**

## Current OpenShift Landscape (November 2025)

### Active Versions

| Version | Status | GA Date | Full Support Ends | Maintenance Ends | EUS Term 1 Ends | EUS Term 2 Ends |
|---------|--------|---------|-------------------|------------------|-----------------|-----------------|
| **4.20** | ✅ **Full Support** | Oct 21, 2025 | GA of 4.21 + 3mo | Apr 21, 2027 | Oct 21, 2027 | Oct 21, 2028 |
| **4.19** | ✅ **Full Support** | Jun 17, 2025 | Jan 21, 2026 | Dec 17, 2026 | N/A | N/A |
| **4.18** | ⚠️ **Maintenance** | Feb 25, 2025 | Sep 17, 2025 | Aug 25, 2026 | Feb 25, 2027 | Feb 25, 2028 |
| **4.17** | ⚠️ **Maintenance** | Oct 1, 2024 | May 25, 2025 | Apr 1, 2026 | N/A | N/A |
| **4.16** | ⚠️ **Maintenance** | Jun 27, 2024 | Jan 1, 2025 | Dec 27, 2025 | Jun 27, 2026 | Jun 27, 2027 |

### Key Observations

1. **4.20 is the LATEST** (released Oct 21, 2025 - 18 days ago)
2. **4.18 is in Maintenance Support** (Full Support ended Sep 17, 2025)
3. **4.20 is an EUS release** (even-numbered = Extended Update Support)
4. **4.18 is an EUS release** (even-numbered = Extended Update Support)
5. **4.21 is expected** in ~4 months (Feb 2026)

## Kubernetes Version Mapping

Based on research and OpenShift release patterns:

| OpenShift | Kubernetes | Go Version | k8s.io Version | Status |
|-----------|-----------|------------|----------------|--------|
| **4.18** | **1.31** | 1.21+ | **v0.31.x** | ✅ Maintenance |
| **4.19** | **1.32** | 1.23+ | **v0.32.x** | ✅ Full Support |
| **4.20** | **1.33** | 1.24+ | **v0.33.x** | ✅ Full Support |
| **4.21** | **1.34** (est) | 1.24+ | **v0.34.x** (est) | 🔮 Expected Feb 2026 |

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
- **Now**: Develop on 4.18
- **Feb 2026**: 4.21 releases, 4.18 feels dated
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
✅ **Full Support until ~May 2026** (4.21 GA + 3 months)  
✅ **EUS release** with long support (until Oct 2028 with EUS Term 2)  
✅ **Latest Kubernetes 1.33 APIs**  
✅ **Latest OpenShift features**  
✅ **Forward compatible** with 4.21  
✅ **EUS-to-EUS upgrade path** (4.20 → 4.22 → 4.24)  

**Cons:**
⚠️ May not work on 4.18/4.19 without modifications  
⚠️ Requires k8s.io v0.33.x (need to verify operator-sdk compatibility)  
⚠️ Newer = less battle-tested (but only 18 days old)  

**Timeline:**
- **Now**: Develop on 4.20
- **Feb 2026**: 4.21 releases, test compatibility
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
- **Now**: Develop on 4.19
- **Jan 2026**: Full Support ends
- **Feb 2026**: 4.21 releases
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

## 🎯 FINAL RECOMMENDATION

### **Develop on OpenShift 4.20 (EUS) with Multi-Version Support Strategy**

**Primary Target:** OpenShift 4.20 (Kubernetes 1.33, k8s.io v0.33.x)  
**Secondary Support:** OpenShift 4.19, 4.21 (when released)  
**Backward Compatibility:** Document 4.18 compatibility requirements

### Rationale

1. **Current Latest**: 4.20 is the current GA release (Oct 21, 2025)
2. **EUS Benefits**: Long support window (up to Oct 2028)
3. **Latest APIs**: Access to Kubernetes 1.33 and latest OpenShift features
4. **Forward Compatible**: Will work on 4.21 when it releases (Feb 2026)
5. **EUS Upgrade Path**: Clean EUS-to-EUS upgrades (4.20 → 4.22 → 4.24)
6. **Market Timing**: By the time your operator is production-ready, 4.20 will be the standard

### Implementation Strategy

**Phase 1: Development (Nov 2025 - Jan 2026)**
```bash
# Upgrade to k8s.io v0.33.x for OpenShift 4.20
go get k8s.io/api@v0.33.0
go get k8s.io/apimachinery@v0.33.0
go get k8s.io/client-go@v0.33.0

# Find compatible OpenShift API (Oct 2025 commit for 4.20)
go get github.com/openshift/api@<commit-hash>

# Find compatible Tekton Pipeline
go get github.com/tektoncd/pipeline@<version>

go mod tidy
```

**Phase 2: Testing (Jan 2026 - Mar 2026)**
- ✅ Test on OpenShift 4.20
- ✅ Test on OpenShift 4.19 (backward compatibility)
- ✅ Test on OpenShift 4.21 when released (Feb 2026)

**Phase 3: Production (Mar 2026+)**
- ✅ Release operator with support matrix: 4.19, 4.20, 4.21
- ✅ Document 4.18 compatibility requirements (if needed)

**Phase 4: Maintenance (2026-2028)**
- ✅ Monitor 4.21, 4.22 releases
- ✅ Test on new versions as they release
- ✅ Plan EUS-to-EUS upgrade (4.20 → 4.22) in 2027

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

