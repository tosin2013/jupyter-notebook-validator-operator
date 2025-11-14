# ADR-035: Test Tier Organization and Scope

**Status**: Accepted  
**Date**: 2025-11-11  
**Authors**: Sophia (AI Assistant), User Feedback  
**Related**: ADR-033 (E2E Testing), ADR-034 (Dual Testing Strategy), ADR-036 (Private Test Repository)

## Context

The test notebooks repository contains various types of notebooks for testing different operator capabilities. We need a clear organization strategy to ensure systematic testing and maintainability.

### Current Situation

**Test Repository**: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks` (private)

**Current Structure** (Disorganized):
```
notebooks/
├── tier1-simple/          # ✅ 4 notebooks (well-organized)
├── tier2-intermediate/    # ⚠️  Empty
└── tier3-complex/         # ⚠️  Empty

model-training/            # ⚠️  Should be tier2
├── train-sentiment-model.ipynb

model-validation/          # ⚠️  Should be tier3
├── kserve/model-inference-kserve.ipynb
└── openshift-ai/sentiment-analysis-test.ipynb

eso-integration/           # ⚠️  Should be tier3
├── aws-credentials-test.ipynb
├── database-connection-test.ipynb
└── mlflow-tracking-test.ipynb

deployments/               # ✅ Model deployment manifests
lib/                       # ✅ Shared Python libraries
golden/                    # ✅ Golden outputs (Phase 3)
scripts/                   # ✅ Test execution scripts
```

### Problem Statement

The current structure has several issues:
1. **Unclear Tier Boundaries**: Notebooks scattered across multiple directories
2. **No Clear Testing Scope**: Unclear what each tier tests
3. **Difficult to Maintain**: Hard to find and update test notebooks
4. **Inconsistent Naming**: No naming convention for test notebooks
5. **Missing Documentation**: No clear guide on what belongs in each tier

## Decision

Reorganize test notebooks into **three clear tiers** based on complexity, execution time, and infrastructure requirements.

### Tier Definitions

#### Tier 1: Simple Validation (< 30 seconds)
**Purpose**: Basic notebook execution and validation  
**Environment**: Kind + OpenShift  
**Infrastructure**: None (no builds, no models, no external secrets)  
**Execution Time**: < 30 seconds per notebook

**Test Scope**:
- ✅ Basic Python execution
- ✅ Print statements and assertions
- ✅ Simple mathematical operations
- ✅ Basic pandas data validation
- ✅ Error handling and error display
- ✅ Git clone from private repository
- ✅ Notebook cell execution order

**Notebooks**:
```
notebooks/tier1-simple/
├── 01-hello-world.ipynb           # Basic print and assertions
├── 02-basic-math.ipynb             # Mathematical operations
├── 03-data-validation.ipynb        # Simple pandas validation
└── 04-error-test.ipynb             # Error handling test
```

#### Tier 2: Intermediate Complexity (1-5 minutes)
**Purpose**: Build integration and dependency management  
**Environment**: OpenShift only  
**Infrastructure**: S2I/Tekton builds, custom images  
**Execution Time**: 1-5 minutes per notebook

**Test Scope**:
- ✅ S2I build integration
- ✅ Tekton build integration
- ✅ Custom requirements.txt installation
- ✅ Model training with small datasets
- ✅ Data preprocessing and feature engineering
- ✅ Model evaluation and metrics
- ✅ Visualization with matplotlib/seaborn

**Notebooks** (Reorganized):
```
notebooks/tier2-intermediate/
├── 01-train-sentiment-model.ipynb      # MOVE from model-training/
├── 02-data-preprocessing.ipynb         # NEW (to be created)
├── 03-feature-engineering.ipynb        # NEW (to be created)
└── 04-model-evaluation.ipynb           # NEW (to be created)
```

**Build Testing**:
- Each notebook tested with **both S2I and Tekton** builds
- Validates custom requirements.txt installation
- Ensures built images work correctly

#### Tier 3: Complex Integration (5-30 minutes)
**Purpose**: Model inference and external integrations  
**Environment**: OpenShift only  
**Infrastructure**: Deployed models (KServe/OpenShift AI), External Secrets Operator  
**Execution Time**: 5-30 minutes per notebook

**Test Scope**:
- ✅ KServe InferenceService integration
- ✅ OpenShift AI model serving
- ✅ External Secrets Operator (AWS, database, MLflow)
- ✅ Model discovery and inference
- ✅ Multi-notebook workflows
- ✅ Performance testing

**Notebooks** (Reorganized):
```
notebooks/tier3-complex/
├── 01-model-inference-kserve.ipynb         # MOVE from model-validation/kserve/
├── 02-sentiment-analysis-test.ipynb        # MOVE from model-validation/openshift-ai/
├── 03-aws-credentials-test.ipynb           # MOVE from eso-integration/
├── 04-database-connection-test.ipynb       # MOVE from eso-integration/
└── 05-mlflow-tracking-test.ipynb           # MOVE from eso-integration/
```

**Infrastructure Requirements**:
- Deployed models via `deployments/setup-models.sh`
- External Secrets Operator installed
- AWS credentials, database credentials, MLflow tracking server

### Naming Convention

**Format**: `{tier}-{number}-{descriptive-name}.ipynb`

**Examples**:
- `tier1-01-hello-world.ipynb`
- `tier2-01-train-sentiment-model.ipynb`
- `tier3-01-model-inference-kserve.ipynb`

**Benefits**:
- Clear tier identification
- Sortable by execution order
- Descriptive names for easy discovery

### Test Execution Strategy

| Tier | Kind | OpenShift | Build Tests | Model Tests | External Secrets |
|------|------|-----------|-------------|-------------|------------------|
| **Tier 1** | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ❌ No |
| **Tier 2** | ❌ No | ✅ Yes | ✅ Yes (S2I + Tekton) | ❌ No | ❌ No |
| **Tier 3** | ❌ No | ✅ Yes | ❌ No | ✅ Yes | ✅ Yes |

### Directory Structure (Proposed)

```
jupyter-notebook-validator-test-notebooks/
├── notebooks/
│   ├── tier1-simple/              # Kind + OpenShift (4 notebooks)
│   ├── tier2-intermediate/        # OpenShift only (4+ notebooks)
│   └── tier3-complex/             # OpenShift only (5+ notebooks)
│
├── deployments/                   # Model deployment manifests
│   ├── setup-models.sh            # Deploy test models
│   ├── fraud-detection-model.yaml
│   ├── sentiment-analysis-model.yaml
│   └── servingruntime-sklearn.yaml
│
├── lib/                           # Shared Python libraries
│   └── model_discovery.py         # Model discovery helper
│
├── golden/                        # Golden notebook outputs (Phase 3)
│   ├── tier1-simple/
│   ├── tier2-intermediate/
│   └── tier3-complex/
│
├── scripts/                       # Test execution scripts
│   ├── run-tier1-tests.sh         # Run Tier 1 tests
│   ├── run-tier2-tests.sh         # Run Tier 2 tests (with builds)
│   └── run-tier3-tests.sh         # Run Tier 3 tests (with models)
│
├── docs/                          # Documentation
│   └── MODEL_DISCOVERY.md         # Model discovery guide
│
├── requirements.txt               # Python dependencies
└── README.md                      # Repository documentation
```

### Migration Plan

**Phase 1: Reorganize Existing Notebooks**
1. Move `model-training/train-sentiment-model.ipynb` → `notebooks/tier2-intermediate/01-train-sentiment-model.ipynb`
2. Move `model-validation/kserve/model-inference-kserve.ipynb` → `notebooks/tier3-complex/01-model-inference-kserve.ipynb`
3. Move `model-validation/openshift-ai/sentiment-analysis-test.ipynb` → `notebooks/tier3-complex/02-sentiment-analysis-test.ipynb`
4. Move `eso-integration/*.ipynb` → `notebooks/tier3-complex/03-05-*.ipynb`
5. Delete empty `model-training/`, `model-validation/`, `eso-integration/` directories

**Phase 2: Create New Tier 2 Notebooks**
6. Create `notebooks/tier2-intermediate/02-data-preprocessing.ipynb`
7. Create `notebooks/tier2-intermediate/03-feature-engineering.ipynb`
8. Create `notebooks/tier2-intermediate/04-model-evaluation.ipynb`

**Phase 3: Update Documentation**
9. Update `README.md` with new structure
10. Update test scripts to reference new paths
11. Update operator documentation

## Consequences

### Positive

- ✅ **Clear Organization**: Easy to find and understand test notebooks
- ✅ **Systematic Testing**: Clear progression from simple to complex
- ✅ **Maintainability**: Easy to add new tests to appropriate tier
- ✅ **Documentation**: Self-documenting structure
- ✅ **Scalability**: Easy to expand each tier independently
- ✅ **Consistent Naming**: Predictable notebook names

### Negative

- ❌ **Migration Effort**: Need to reorganize existing notebooks
- ❌ **Breaking Changes**: Existing references need updating
- ❌ **Learning Curve**: Team needs to learn new structure

### Neutral

- ⚠️ **Tier Boundaries**: Some notebooks may fit multiple tiers
- ⚠️ **Maintenance**: Need to keep structure consistent over time

## Alternatives Considered

### Alternative 1: Keep Current Structure
- **Pros**: No migration effort
- **Cons**: Confusing, hard to maintain, unclear boundaries
- **Rejected**: Does not solve the organization problem

### Alternative 2: Single Flat Directory
- **Pros**: Simple, no hierarchy
- **Cons**: Difficult to navigate with many notebooks, no clear testing scope
- **Rejected**: Does not scale

### Alternative 3: Feature-Based Organization
- **Pros**: Organized by feature (builds, models, secrets)
- **Cons**: Unclear execution order, difficult to run all tests
- **Rejected**: Tier-based organization is clearer

## Implementation Plan

### Phase 1: Repository Reorganization (Week 1)
1. Create new tier directories
2. Move existing notebooks to appropriate tiers
3. Update notebook paths in test scripts
4. Delete empty directories
5. Update README.md

### Phase 2: New Notebook Creation (Week 2)
6. Create Tier 2 data preprocessing notebook
7. Create Tier 2 feature engineering notebook
8. Create Tier 2 model evaluation notebook
9. Test all notebooks locally

### Phase 3: Documentation and Validation (Week 3)
10. Update operator documentation
11. Update ADR-033 with new structure
12. Run full test suite on OpenShift
13. Validate all tiers pass

## Verification

### Success Criteria
- [ ] All notebooks organized into appropriate tiers
- [ ] Test scripts updated and working
- [ ] Documentation updated
- [ ] All tests pass on OpenShift
- [ ] Tier 1 tests pass on Kind

### Testing
```bash
# Verify Tier 1 (Kind + OpenShift)
./scripts/run-tier1-tests.sh

# Verify Tier 2 (OpenShift only, with builds)
./scripts/run-tier2-tests.sh

# Verify Tier 3 (OpenShift only, with models)
./scripts/run-tier3-tests.sh
```

## References

- [Test Notebooks Repository](https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks)
- ADR-033: End-to-End Testing Against Live OpenShift Cluster
- ADR-034: Dual Testing Strategy with Kind and OpenShift
- ADR-036: Private Test Repository Strategy

## Notes

- Tier boundaries based on execution time and infrastructure requirements
- Each tier should be independently testable
- Notebooks should be self-contained (no dependencies between notebooks)
- Use `lib/` for shared code to avoid duplication
- Golden outputs (Phase 3) will use same tier structure

