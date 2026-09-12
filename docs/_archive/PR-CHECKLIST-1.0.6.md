# PR Checklist Verification - v1.0.6

**Operator:** jupyter-notebook-validator-operator  
**Version:** 1.0.6  
**Date:** 2026-01-28  
**Author:** Tosin Akinosho <takinosh@redhat.com>

---

## Updates to Existing Operators

| Requirement | Status | Notes |
|-------------|--------|-------|
| `ci.yaml` exists | ✅ | Already present in `operators/jupyter-notebook-validator-operator/` |
| CSV has `replaces` property | ✅ | `replaces: jupyter-notebook-validator-operator.v1.0.5` |
| CSV in appropriate channel | ✅ | `alpha` channel in `annotations.yaml` |
| Tested OLM update | ✅ | Passed `full-pre-submission-test.sh --version 1.0.6` |
| Submission signed (DCO) | ✅ | `Signed-off-by: Tosin Akinosho <takinosh@redhat.com>` |

---

## Submission Rules

| Requirement | Status | Notes |
|-------------|--------|-------|
| Only modifies one operator | ✅ | Only `jupyter-notebook-validator-operator` |
| Operator you own | ✅ | Yes |
| Not renaming | ✅ | Same operator name |
| No files outside operator folder | ✅ | Only `1.0.6/` directory added |
| Single commit | ✅ | `operator jupyter-notebook-validator-operator (1.0.6)` |

---

## Operator Description

| Requirement | Status |
|-------------|--------|
| Description of managed application | ✅ Present |
| Features and capabilities | ✅ Present |
| Manual steps/prerequisites | ✅ Present |

---

## Operator Metadata

| Requirement | Status | Value |
|-------------|--------|-------|
| Human readable name | ✅ | "Jupyter Notebook Validator Operator" |
| 1-liner description | ✅ | "Validates Jupyter notebooks using Papermill" |
| Valid category | ✅ | "AI/Machine Learning" |
| Capability level | ✅ | "Basic Install" |
| Links (maintainer, source, docs) | ✅ | Present |
| Example CRDs | ✅ | Present in `alm-examples` annotation |
| Quadratic logo/icon | ✅ | Updated via `scripts/update-icon.sh` |

---

## Provider & Maintainer Information

| Field | Value |
|-------|-------|
| Provider Name | Decision Crafters |
| Provider URL | https://www.decisioncrafters.com/ |
| Maintainer Name | Tosin Akinosho |
| Maintainer Email | takinosh@redhat.com |

---

## Container Images

| Image | Tag |
|-------|-----|
| Operator | `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.6` |
| Bundle | `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.6` |

---

## PR Information

### community-operators-prod (OpenShift/OKD)

- **Repository:** redhat-openshift-ecosystem/community-operators-prod
- **Branch:** `add-jupyter-validator-1.0.6`
- **PR Title:** `operator jupyter-notebook-validator-operator (1.0.6)`
- **PR URL:** https://github.com/tosin2013/community-operators-prod/pull/new/add-jupyter-validator-1.0.6

### community-operators (Kubernetes OperatorHub)

- **Repository:** k8s-operatorhub/community-operators
- **Branch:** `add-jupyter-validator-1.0.6`
- **PR Title:** `operator jupyter-notebook-validator-operator (1.0.6)`
- **PR URL:** https://github.com/tosin2013/community-operators/pull/new/add-jupyter-validator-1.0.6

---

## Testing Verification

```bash
# Pre-submission test passed
./scripts/full-pre-submission-test.sh --version 1.0.6 --quick

# Results:
# ✅ OLM Install (K8s v1.31.0): PASSED
# ✅ K8s Matrix (v1.31.0): PASSED
# ⏭️ Bundle Validate: SKIPPED (expected - not yet in community-operators)
```

---

## Summary

**All checklist items satisfied.** ✅

This submission is ready for PR creation to both:
- `redhat-openshift-ecosystem/community-operators-prod`
- `k8s-operatorhub/community-operators`
