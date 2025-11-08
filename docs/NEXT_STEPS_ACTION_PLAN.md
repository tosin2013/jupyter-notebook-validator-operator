# Next Steps: Action Plan for Dependency Resolution

**Date:** November 8, 2025  
**Decision:** Target OpenShift 4.20 (EUS) with k8s.io v0.33.x  
**Status:** 🚀 READY TO EXECUTE

## TL;DR - What We Need to Do

1. Find OpenShift API commit compatible with k8s.io v0.33.x (OpenShift 4.20)
2. Find Tekton Pipeline version compatible with k8s.io v0.33.x
3. Update go.mod with these versions
4. Build and test the build strategies
5. Deploy to OpenShift 4.20 cluster

## Step-by-Step Action Plan

### Step 1: Research OpenShift API Compatible Version

**Goal:** Find an OpenShift API commit from Oct-Nov 2025 that uses k8s.io v0.33.x

**Commands to Run:**

```bash
# Clone OpenShift API repository (if not already cloned)
cd /tmp
git clone https://github.com/openshift/api
cd api

# Look for commits around OpenShift 4.20 release (Oct 21, 2025)
git log --since="2025-10-01" --until="2025-11-08" --oneline --all

# For each recent commit, check the go.mod file
git show <commit-hash>:go.mod | grep "k8s.io/api"

# Find a commit that has:
# k8s.io/api v0.33.0 (or v0.33.x)
# k8s.io/apimachinery v0.33.0
# k8s.io/client-go v0.33.0
```

**Expected Output:**
```
commit abc123def456... (example)
k8s.io/api v0.33.0
k8s.io/apimachinery v0.33.0
k8s.io/client-go v0.33.0
```

**Action:** Record the commit hash for use in go.mod

### Step 2: Research Tekton Pipeline Compatible Version

**Goal:** Find Tekton Pipeline version compatible with k8s.io v0.33.x

**Commands to Run:**

```bash
# Check Tekton Pipeline releases
cd /tmp
git clone https://github.com/tektoncd/pipeline
cd pipeline

# Look for recent releases
git tag --sort=-version:refname | head -20

# For each recent tag, check go.mod
git show v0.66.0:go.mod | grep "k8s.io/api"  # Example version
git show v0.65.0:go.mod | grep "k8s.io/api"
git show v0.64.0:go.mod | grep "k8s.io/api"

# Find a version that has k8s.io v0.33.x
```

**Alternative:** Check Tekton documentation for Kubernetes compatibility matrix

**Expected Output:**
```
v0.66.0 (example)
k8s.io/api v0.33.0
```

**Action:** Record the version for use in go.mod

### Step 3: Update go.mod

**Goal:** Update operator's go.mod with compatible versions

**File:** `/home/lab-user/jupyter-notebook-validator-operator/go.mod`

**Commands to Run:**

```bash
cd /home/lab-user/jupyter-notebook-validator-operator

# Upgrade Kubernetes dependencies to v0.33.x
go get k8s.io/api@v0.33.0
go get k8s.io/apimachinery@v0.33.0
go get k8s.io/client-go@v0.33.0

# Add OpenShift API (use commit hash from Step 1)
go get github.com/openshift/api@<commit-hash-from-step-1>

# Add Tekton Pipeline (use version from Step 2)
go get github.com/tektoncd/pipeline@<version-from-step-2>

# Clean up dependencies
go mod tidy

# Verify no conflicts
go mod verify
```

**Expected go.mod:**
```go
module github.com/tosin2013/jupyter-notebook-validator-operator

go 1.21

require (
    // Kubernetes 1.33 (OpenShift 4.20)
    k8s.io/api v0.33.0
    k8s.io/apimachinery v0.33.0
    k8s.io/client-go v0.33.0
    
    // OpenShift API (Oct 2025 commit for 4.20)
    github.com/openshift/api v0.0.0-<commit-hash>
    
    // Tekton Pipeline (compatible with k8s 1.33)
    github.com/tektoncd/pipeline v0.66.0  // Example version
    
    // ... other dependencies
)
```

### Step 4: Build and Test

**Goal:** Verify the build strategies compile and work

**Commands to Run:**

```bash
cd /home/lab-user/jupyter-notebook-validator-operator

# Build the build strategies
go build ./pkg/build/...

# If successful, build the entire operator
go build ./...

# Run unit tests
go test ./pkg/build/...

# Run all tests
go test ./...
```

**Expected Output:**
```
✅ Build successful
✅ Tests pass
```

**If Errors Occur:**
- Check for API changes in k8s.io v0.33.x
- Check for API changes in OpenShift API
- Check for API changes in Tekton Pipeline
- Update code to match new APIs

### Step 5: Deploy to OpenShift 4.20 Cluster

**Goal:** Test the operator on a real OpenShift 4.20 cluster

**Prerequisites:**
- Access to OpenShift 4.20 cluster
- `oc` CLI configured

**Commands to Run:**

```bash
# Verify cluster version
oc version
# Should show: Server Version: 4.20.x

# Build operator image
make docker-build docker-push IMG=<your-registry>/jupyter-notebook-validator-operator:latest

# Deploy operator
make deploy IMG=<your-registry>/jupyter-notebook-validator-operator:latest

# Verify deployment
oc get pods -n jupyter-notebook-validator-operator-system

# Test S2I build strategy
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_s2i.yaml

# Test Tekton build strategy
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob_tekton.yaml

# Check build status
oc get notebookvalidationjobs
oc describe notebookvalidationjob <job-name>
```

## Fallback Plan: If k8s.io v0.33.x Doesn't Work

If we encounter issues with k8s.io v0.33.x (e.g., operator-sdk incompatibility), we have a fallback:

### Fallback Option 1: Use k8s.io v0.32.x (OpenShift 4.19)

```bash
# Target OpenShift 4.19 instead
go get k8s.io/api@v0.32.0
go get k8s.io/apimachinery@v0.32.0
go get k8s.io/client-go@v0.32.0

# Find compatible OpenShift API (Jun 2025 commit)
go get github.com/openshift/api@<commit-hash>

# Find compatible Tekton Pipeline
go get github.com/tektoncd/pipeline@<version>
```

**Pros:**
- OpenShift 4.19 is in Full Support until Jan 2026
- More battle-tested (5 months old)
- Forward compatible with 4.20 and 4.21

**Cons:**
- Not an EUS release (shorter support window)
- Maintenance Support ends Dec 2026

### Fallback Option 2: Use k8s.io v0.31.x (OpenShift 4.18)

```bash
# Target OpenShift 4.18 (EUS)
go get k8s.io/api@v0.31.4
go get k8s.io/apimachinery@v0.31.4
go get k8s.io/client-go@v0.31.4

# Find compatible OpenShift API (Feb 2025 commit)
go get github.com/openshift/api@<commit-hash>

# Find compatible Tekton Pipeline
go get github.com/tektoncd/pipeline@v0.65.0
```

**Pros:**
- EUS release with long support
- Most stable and battle-tested
- Widest compatibility

**Cons:**
- Already in Maintenance Support
- Using older Kubernetes 1.31 APIs

## Success Criteria

✅ **Step 1 Complete:** Found OpenShift API commit with k8s.io v0.33.x  
✅ **Step 2 Complete:** Found Tekton Pipeline version with k8s.io v0.33.x  
✅ **Step 3 Complete:** Updated go.mod, `go mod tidy` succeeds  
✅ **Step 4 Complete:** `go build ./pkg/build/...` succeeds  
✅ **Step 5 Complete:** Operator deploys to OpenShift 4.20 cluster  
✅ **Step 6 Complete:** S2I build strategy works on cluster  
✅ **Step 7 Complete:** Tekton build strategy works on cluster  

## Timeline

| Task | Duration | Status |
|------|----------|--------|
| Research OpenShift API version | 30 min | 📋 Pending |
| Research Tekton Pipeline version | 30 min | 📋 Pending |
| Update go.mod | 15 min | 📋 Pending |
| Build and test locally | 30 min | 📋 Pending |
| Deploy to OpenShift 4.20 | 1 hour | 📋 Pending |
| Integration testing | 2 hours | 📋 Pending |
| **Total** | **~5 hours** | |

## Questions to Answer

1. **Do we have access to an OpenShift 4.20 cluster?**
   - If yes: Proceed with Step 1-5
   - If no: Can we deploy one? Or use 4.19?

2. **What is our operator-sdk version?**
   - Check: `operator-sdk version`
   - Verify compatibility with k8s.io v0.33.x

3. **Do we need to upgrade operator-sdk?**
   - If operator-sdk is old, may need to upgrade first

4. **What is our current go.mod state?**
   - Current k8s.io version: v0.29.2
   - Need to upgrade to v0.33.0

## Ready to Execute?

**YES** - Let's start with Step 1: Research OpenShift API version

**NO** - What's blocking us?
- Need to verify OpenShift 4.20 cluster access?
- Need to check operator-sdk compatibility?
- Need to discuss strategy further?

## Next Command to Run

```bash
# Let's start by checking what OpenShift cluster we have access to
oc version

# And check our current operator-sdk version
operator-sdk version

# Then we can proceed with the research
```

Would you like me to:
1. **Start researching** OpenShift API and Tekton versions for k8s.io v0.33.x?
2. **Check cluster access** first to verify we have OpenShift 4.20?
3. **Use a fallback** and target 4.19 or 4.18 instead?

