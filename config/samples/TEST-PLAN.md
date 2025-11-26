# Jupyter Notebook Validator Operator - Test Plan

## Test Repository
All tests use: `https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git`

## Test Categories

### Tier 1: Simple Validation Tests (No Build)
Basic notebook execution without build integration.

| Test | Notebook | Purpose | Sample File |
|------|----------|---------|-------------|
| Hello World | `notebooks/tier1-simple/01-hello-world.ipynb` | Basic execution | `mlops_v1alpha1_notebookvalidationjob.yaml` |
| Basic Math | `notebooks/tier1-simple/02-basic-math.ipynb` | Computation validation | `test-basic-math.yaml` |
| Data Validation | `notebooks/tier1-simple/03-data-validation.ipynb` | Data processing | `test-data-validation.yaml` |
| Error Test | `notebooks/tier1-simple/04-error-test.ipynb` | Failure handling | `test-error-handling.yaml` |

### Tier 2: Build Integration Tests (S2I)
Test S2I build strategy with requirements.txt.

| Test | Notebook | Requirements | Sample File |
|------|----------|--------------|-------------|
| S2I Basic | `notebooks/tier1-simple/01-hello-world.ipynb` | Root `requirements.txt` | `mlops_v1alpha1_notebookvalidationjob_s2i.yaml` |
| S2I Auto-gen | `notebooks/tier1-simple/02-basic-math.ipynb` | Auto-generated | `mlops_v1alpha1_notebookvalidationjob_s2i_autogen.yaml` |

### Tier 3: Model Training Tests
Test notebooks that train ML models from scratch.

| Test | Notebook | Purpose | Sample File |
|------|----------|---------|-------------|
| Sentiment Model | `model-training/train-sentiment-model.ipynb` | Train sentiment analysis model | `model-training-job.yaml` |

### Tier 4: Model Validation Tests
Test notebooks that validate deployed models.

| Test | Notebook | Service Type | Sample File |
|------|----------|--------------|-------------|
| KServe Inference | `model-validation/kserve/model-inference-kserve.ipynb` | KServe | `model-validation-kserve.yaml` |
| OpenShift AI | `model-validation/openshift-ai/sentiment-analysis-test.ipynb` | OpenShift AI | `model-validation-openshift-ai.yaml` |

### Tier 5: ESO Integration Tests
Test External Secrets Operator integration.

| Test | Notebook | Secret Type | Sample File |
|------|----------|-------------|-------------|
| AWS Credentials | `eso-integration/aws-credentials-test.ipynb` | AWS | `test-eso-aws.yaml` |
| Database Connection | `eso-integration/database-connection-test.ipynb` | Database | `mlops_v1alpha1_notebookvalidationjob_database.yaml` |
| MLflow Tracking | `eso-integration/mlflow-tracking-test.ipynb` | MLflow | `test-eso-mlflow.yaml` |

### Tier 6: Golden Notebook Comparison
Test golden notebook comparison feature.

| Test | Notebook | Golden Notebook | Sample File |
|------|----------|-----------------|-------------|
| Golden Comparison | `notebooks/tier1-simple/01-hello-world.ipynb` | `golden/tier1-simple/01-hello-world.ipynb` | `mlops_v1alpha1_notebookvalidationjob_golden.yaml` |

## Test Execution Order

### Phase 1: Basic Validation (No Build)
```bash
# Test 1: Hello World
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob.yaml

# Test 2: Basic Math
oc apply -f config/samples/test-basic-math.yaml

# Test 3: Data Validation
oc apply -f config/samples/test-data-validation.yaml
```

### Phase 2: Build Integration (S2I)
```bash
# Ensure ServiceAccount exists
oc create serviceaccount notebook-validator-jupyter-notebook-validator-runner -n default

# Test 4: S2I Basic Build
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml

# Test 5: S2I Auto-gen Requirements
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i_autogen.yaml
```

### Phase 3: Model Training
```bash
# Test 6: Train Sentiment Model
oc apply -f config/samples/model-training-job.yaml
```

### Phase 4: Model Validation
```bash
# Test 7: KServe Model Inference (requires KServe deployed)
oc apply -f config/samples/model-validation-kserve.yaml

# Test 8: OpenShift AI Model Validation (requires RHOAI)
oc apply -f config/samples/model-validation-openshift-ai.yaml
```

### Phase 5: ESO Integration
```bash
# Test 9: AWS Credentials (requires ESO)
oc apply -f config/samples/test-eso-aws.yaml

# Test 10: Database Connection (requires ESO)
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_database.yaml
```

### Phase 6: Golden Comparison
```bash
# Test 11: Golden Notebook Comparison
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_golden.yaml
```

## Monitoring Tests

### Watch Job Status
```bash
oc get notebookvalidationjob -w
```

### Check Build Status (for S2I tests)
```bash
oc get builds -w
oc get buildconfigs
```

### Check Pod Logs
```bash
# Get validation pod name
POD=$(oc get pods -l mlops.redhat.com/notebook-validation=true --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')

# Watch logs
oc logs -f $POD
```

### Check Operator Logs
```bash
oc logs -n jupyter-notebook-validator-operator deployment/notebook-validator-controller-manager -c manager -f
```

## Success Criteria

### Tier 1 Tests
- ✅ Pod created successfully
- ✅ Notebook executed without errors
- ✅ Status shows "Complete"
- ✅ Execution time recorded

### Tier 2 Tests (S2I)
- ✅ BuildConfig created
- ✅ Build triggered and completed
- ✅ Image pushed to registry
- ✅ Validation pod uses built image
- ✅ Notebook executed successfully

### Tier 3 Tests (Model Training)
- ✅ Model trained from scratch
- ✅ Training metrics recorded
- ✅ Model artifacts saved

### Tier 4 Tests (Model Validation)
- ✅ Model service detected
- ✅ Inference requests successful
- ✅ Validation results recorded

### Tier 5 Tests (ESO)
- ✅ Secrets retrieved from external source
- ✅ Secrets mounted correctly
- ✅ Notebook can access secrets

### Tier 6 Tests (Golden)
- ✅ Both notebooks executed
- ✅ Outputs compared
- ✅ Differences reported (if any)

## Cleanup

```bash
# Delete all test jobs
oc delete notebookvalidationjob --all -n default

# Delete builds
oc delete builds --all -n default
oc delete buildconfigs --all -n default

# Delete test pods
oc delete pods -l mlops.redhat.com/notebook-validation=true -n default
```

## Next Steps

1. ✅ Update all sample files to use test repository
2. ⏳ Create missing sample files for new tests
3. ⏳ Run Tier 1 tests (basic validation)
4. ⏳ Fix S2I detection issue
5. ⏳ Run Tier 2 tests (S2I builds)
6. ⏳ Run remaining tiers as dependencies are met

