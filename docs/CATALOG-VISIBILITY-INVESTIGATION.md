# Catalog Visibility Investigation - March 19, 2026

## Summary

**Status:** ✅ Investigation Complete  
**Operator Status:** ✅ Fully Functional on OKD 4.22  
**Catalog Visibility:** ❌ Not visible in OKD OperatorHub (root cause identified)

---

## Why Operator Doesn't Appear in OKD Catalog

The operator is **NOT visible** in OKD OperatorHub because:

1. **Only version 1.0.2 exists in community-operators-prod**
   - OKD pulls from `redhat-openshift-ecosystem/community-operators-prod`
   - Only v1.0.2 was submitted (outdated)
   
2. **Newer versions only in k8s-operatorhub/community-operators**
   - PR #7235 merged versions 1.0.7, 1.0.8, 1.0.9 to Kubernetes OperatorHub
   - These appear on OperatorHub.io but NOT in OpenShift/OKD

3. **No PR submitted to community-operators-prod for newer versions**
   - Current bundle is v1.0.6
   - Need to submit to community-operators-prod for OKD visibility

---

## Validation Results ✅

### Operator Deployment: SUCCESS
- Deployed on OKD 4.22.0-okd-scos.ec.9
- Both containers running (2/2 READY)
- Webhooks operational with cert-manager
- Metrics proxy working correctly

### Test Validation: SUCCESS
- Test notebook: `notebooks/tier1-simple/01-hello-world.ipynb`
- Status: **Succeeded**
- Execution time: ~103 seconds
- Repository: https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks

**Conclusion:** Operator is production-ready on OKD 4.22

---

## Critical Fixes Applied ⚠️

### 1. kube-rbac-proxy Image Fix (CRITICAL)
**Problem:** `gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0` doesn't exist  
**Solution:** Updated to `quay.io/brancz/kube-rbac-proxy:v0.18.0`  
**Status:** ✅ Fixed and pushed to all branches (main, release-4.18, release-4.19, release-4.20)

### 2. Channel Mismatch Fix
**Problem:** Bundle used `alpha` channel, catalog used `stable`  
**Solution:** Updated bundle metadata to `stable` channel  
**Status:** ✅ Fixed

---

## PR Status

### k8s-operatorhub/community-operators
- **Status:** ✅ MERGED
- **PR:** #7235
- **Versions:** 1.0.7, 1.0.8, 1.0.9
- **Result:** Available on OperatorHub.io

### redhat-openshift-ecosystem/community-operators-prod
- **Status:** ❌ NOT SUBMITTED (only v1.0.2 exists)
- **Versions Needed:** 1.0.6 or 1.0.7/1.0.8/1.0.9
- **Result:** NOT available in OKD OperatorHub

---

## Next Steps to Make Operator Visible in OKD

### Option 1: Submit v1.0.6 (RECOMMENDED)
**Pros:**
- Already tested and working on OKD 4.22
- Includes all critical fixes
- Simpler upgrade path

**Steps:**
```bash
# 1. Rebuild bundle with fixes
make bundle-build bundle-push BUNDLE_IMG=quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6

# 2. Fork and clone community-operators-prod
gh repo fork redhat-openshift-ecosystem/community-operators-prod
git clone https://github.com/YOUR_USERNAME/community-operators-prod
cd community-operators-prod

# 3. Create branch
git checkout -b add-jupyter-validator-1.0.6

# 4. Copy bundle
mkdir -p operators/jupyter-notebook-validator-operator/1.0.6
cp -r /path/to/bundle/* operators/jupyter-notebook-validator-operator/1.0.6/

# 5. Commit with DCO
git add operators/jupyter-notebook-validator-operator/1.0.6/
git commit -s -m "operator jupyter-notebook-validator-operator (1.0.6)"

# 6. Push and create PR
git push origin add-jupyter-validator-1.0.6
gh pr create --repo redhat-openshift-ecosystem/community-operators-prod \
  --title "operator jupyter-notebook-validator-operator (1.0.6)" \
  --body "Add version 1.0.6 with kube-rbac-proxy fix and stable channel"
```

### Option 2: Rebuild and Submit v1.0.7/1.0.8/1.0.9
**Pros:**
- Matches versions in k8s-operatorhub
- Consistent across both catalogs

**Cons:**
- Requires rebuilding all three bundles with kube-rbac-proxy fix
- More complex upgrade chain
- Need to update CSV replaces fields

---

## Pre-Submission Checklist

Before submitting to community-operators-prod:

- [ ] Bundle includes kube-rbac-proxy fix (quay.io/brancz/kube-rbac-proxy:v0.18.0)
- [ ] Channel set to `stable` in bundle/metadata/annotations.yaml
- [ ] Provider metadata correct (Decision Crafters)
- [ ] Maintainer email: takinosh@redhat.com
- [ ] Commit signed with DCO (`git commit -s`)
- [ ] Bundle validated: `opm alpha bundle validate`
- [ ] CSV replaces field correct
- [ ] Container images pushed to Quay.io
- [ ] Icon included in CSV

---

## Files Modified (Already Committed)

1. `config/default/manager_auth_proxy_patch.yaml` - kube-rbac-proxy fix
2. `bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml` - kube-rbac-proxy fix
3. `bundle/metadata/annotations.yaml` - channel fix (alpha → stable)

**Git Commits:**
- main: 8f2c219
- release-4.18: 1707f74
- release-4.19: f53f59b
- release-4.20: 6646ecb

---

## References

- **Operator Repository:** https://github.com/tosin2013/jupyter-notebook-validator-operator
- **k8s-operatorhub PR:** https://github.com/k8s-operatorhub/community-operators/pull/7235
- **community-operators-prod:** https://github.com/redhat-openshift-ecosystem/community-operators-prod
- **Detailed Findings:** `/root/.windsurf/plans/operator-catalog-investigation-findings.md`
- **ADR-007:** Distribution and Catalog Strategy
- **RELEASE.md:** Release procedures

---

## Contact

**Maintainer:** Tosin Akinosho <takinosh@redhat.com>  
**Provider:** Decision Crafters  
**Repository:** https://github.com/tosin2013/jupyter-notebook-validator-operator
