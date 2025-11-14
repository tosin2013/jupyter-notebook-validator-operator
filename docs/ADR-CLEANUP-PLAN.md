# ADR Cleanup Plan

**Date**: 2025-11-11  
**Status**: In Progress  
**Goal**: Ensure all ADRs are consistent, properly cross-referenced, and accurately reflect current architectural decisions

---

## Audit Results Summary

### Critical Issues Found

1. **Title Mismatches** (2 ADRs)
   - ADR-026: Title says "ADR-019" but file is "026-smart-validation-pod-recovery.md"
   - ADR-027: Title says "ADR-016" but file is "027-s2i-build-strategy-for-git-integration.md"

2. **Missing Status Fields** (2 ADRs)
   - ADR-012: release-and-cicd-strategy.md
   - ADR-020: model-aware-validation-strategy.md

3. **Missing Date Fields** (24 ADRs)
   - ADR-001 through ADR-027 (except 023, 028-036)

4. **Inconsistent Status Values**
   - Some use "## Status" format
   - Some use "**Status**:" format
   - Some have descriptive status (e.g., "✅ **IMPLEMENTED**")
   - Many have "UNKNOWN" status (no status field at all)

---

## Cleanup Actions

### Phase 1: Fix Critical Issues (Immediate)

#### 1.1 Fix Title Mismatches
- [ ] **ADR-026**: Change title from "ADR-019" to "ADR-026"
- [ ] **ADR-027**: Change title from "ADR-016" to "ADR-027"

#### 1.2 Add Missing Status Fields
- [ ] **ADR-012**: Add status field (likely "Accepted" based on implementation)
- [ ] **ADR-020**: Add status field (check if "Proposed" or "Accepted")

#### 1.3 Add Missing Date Fields
- [ ] **ADR-001 through ADR-022**: Add creation dates (estimate based on git history)
- [ ] **ADR-026, ADR-027**: Add creation dates

---

### Phase 2: Standardize Metadata Format

**Standard Format** (to be applied to all ADRs):
```markdown
# ADR-XXX: Title

**Status**: [Proposed | Accepted | Implemented | Deprecated | Superseded]  
**Date**: YYYY-MM-DD  
**Updated**: YYYY-MM-DD (if applicable)  
**Related**: ADR-XXX, ADR-YYY

## Context
...
```

**Actions**:
- [ ] Standardize all ADRs to use "**Status**:" format
- [ ] Ensure all dates are in YYYY-MM-DD format
- [ ] Add "**Updated**:" field where ADRs have been modified

---

### Phase 3: Establish Cross-References

#### 3.1 Core Infrastructure ADRs (Foundation)
- **ADR-001**: Operator Framework → Referenced by: ADR-003, ADR-004, ADR-005
- **ADR-002**: Platform Support → Referenced by: ADR-006, ADR-029, ADR-032
- **ADR-003**: CRD Schema → Referenced by: ADR-008, ADR-013, ADR-020
- **ADR-004**: Deployment Strategy → Referenced by: ADR-007, ADR-012
- **ADR-005**: RBAC Model → Referenced by: ADR-019, ADR-031, ADR-033

#### 3.2 Secret Management Cluster
- **ADR-009**: Secret Management (Core) → Referenced by: ADR-014, ADR-015, ADR-031
- **ADR-014**: Credential Injection → References: ADR-009, ADR-015
- **ADR-015**: Environment Variables → References: ADR-009, ADR-014
- **ADR-016**: ESO Integration → References: ADR-009, ADR-017, ADR-018
- **ADR-017**: Vault Integration → References: ADR-016
- **ADR-018**: Secret Rotation → References: ADR-016, ADR-017
- **ADR-019**: RBAC for Secrets → References: ADR-005, ADR-009

#### 3.3 Build Integration Cluster
- **ADR-023**: S2I Build Integration (Core) → Referenced by: ADR-024, ADR-027, ADR-031
- **ADR-024**: Fallback Strategy → References: ADR-023
- **ADR-025**: Extension Framework → References: ADR-023, ADR-031
- **ADR-027**: S2I Git Integration → References: ADR-009, ADR-023
- **ADR-028**: Tekton Task Strategy → References: ADR-031
- **ADR-031**: Tekton Build (Implemented) → References: ADR-023, ADR-027, ADR-028, ADR-009

#### 3.4 Testing Strategy Cluster
- **ADR-008**: Notebook Testing Strategy → Referenced by: ADR-032, ADR-033, ADR-035
- **ADR-032**: CI Testing (Kind) → References: ADR-002, ADR-008, ADR-034
- **ADR-033**: E2E Testing (OpenShift) → References: ADR-032, ADR-034, ADR-035, ADR-036
- **ADR-034**: Dual Testing Strategy → References: ADR-032, ADR-033, ADR-035
- **ADR-035**: Test Tier Organization → References: ADR-008, ADR-033, ADR-034, ADR-036
- **ADR-036**: Private Test Repository → References: ADR-009, ADR-033, ADR-034, ADR-035

#### 3.5 Observability Cluster
- **ADR-010**: Observability Strategy (Core) → Referenced by: ADR-021, ADR-022
- **ADR-021**: OpenShift Dashboard → References: ADR-010
- **ADR-022**: Community Contributions → References: ADR-010, ADR-021

#### 3.6 Validation & Recovery Cluster
- **ADR-011**: Error Handling → Referenced by: ADR-026, ADR-030
- **ADR-013**: Output Comparison → References: ADR-003, ADR-008
- **ADR-020**: Model-Aware Validation → References: ADR-008, ADR-036
- **ADR-026**: Smart Recovery → References: ADR-011, ADR-023, ADR-031
- **ADR-030**: Smart Error Messages → References: ADR-011, ADR-028, ADR-029

#### 3.7 Release & Distribution Cluster
- **ADR-006**: Version Roadmap → References: ADR-002, ADR-029
- **ADR-007**: Distribution Strategy → References: ADR-004
- **ADR-012**: CI/CD Strategy → References: ADR-004, ADR-032, ADR-033
- **ADR-029**: Dependency Review → References: ADR-002, ADR-006

---

### Phase 4: Identify Deprecated/Superseded ADRs

**Candidates for Deprecation**:
- None identified yet (all ADRs appear relevant)

**Candidates for "Superseded" Status**:
- **ADR-023** (S2I Build Integration): Partially superseded by ADR-031 (Tekton Build)
  - Action: Update status to "Accepted - Partially Superseded by ADR-031"
  - Reason: Tekton is now the primary build method, but S2I still supported

**Candidates for Status Update**:
- **ADR-001 through ADR-007**: Should be "Accepted" (foundational, implemented)
- **ADR-008 through ADR-019**: Should be "Accepted" (implemented or in use)
- **ADR-020 through ADR-022**: Should remain "Proposed" (future work)
- **ADR-023 through ADR-027**: Should be "Accepted" (implemented)
- **ADR-028**: Should be "Accepted" (decision made, referenced by ADR-031)
- **ADR-029**: Already "Accepted" ✅
- **ADR-030**: Should remain "Proposed" (future enhancement)
- **ADR-031**: Already "Implemented" ✅
- **ADR-032**: Should remain "Proposed" (in progress)
- **ADR-033 through ADR-036**: Already "Accepted" ✅

---

### Phase 5: Update Implementation Plan

**Actions**:
- [ ] Remove references to deprecated ADRs (if any)
- [ ] Update ADR status references in IMPLEMENTATION-PLAN.md
- [ ] Ensure all "Implemented" ADRs are marked as complete in the plan
- [ ] Add cross-reference links between related sections

---

## Execution Order

1. **Fix Critical Issues** (30 minutes)
   - Fix title mismatches in ADR-026 and ADR-027
   - Add missing status fields to ADR-012 and ADR-020
   
2. **Add Missing Dates** (1 hour)
   - Use git log to find creation dates for ADR-001 through ADR-027
   - Add date fields to all ADRs

3. **Standardize Format** (2 hours)
   - Apply standard metadata format to all 36 ADRs
   - Ensure consistent status values

4. **Add Cross-References** (3 hours)
   - Add "Related ADRs" sections to all ADRs
   - Update existing references to be consistent

5. **Update Statuses** (1 hour)
   - Review and update status for ADR-001 through ADR-032
   - Mark superseded ADRs appropriately

6. **Update Implementation Plan** (30 minutes)
   - Sync IMPLEMENTATION-PLAN.md with updated ADR statuses
   - Add cross-reference links

---

## Success Criteria

- ✅ All ADRs have correct title numbers
- ✅ All ADRs have Status and Date fields
- ✅ All ADRs use consistent metadata format
- ✅ All ADRs have "Related ADRs" sections with accurate cross-references
- ✅ All ADR statuses accurately reflect implementation state
- ✅ IMPLEMENTATION-PLAN.md is synchronized with ADR statuses
- ✅ No broken references between ADRs

---

## Next Steps

After cleanup is complete:
1. Create ADR index document (docs/adrs/INDEX.md) with relationship map
2. Add ADR compliance check to CI/CD pipeline
3. Document ADR maintenance process in CONTRIBUTING.md

