# ADR-030: Smart Error Messages and User Feedback

**Status**: Proposed  
**Date**: 2025-11-09  
**Deciders**: Development Team  
**Related**: ADR-028 (Tekton Task Strategy), ADR-029 (Platform Version Dependency Review)

## Context

During Tekton integration testing, we discovered critical UX issues with error reporting:

### Current Problems

1. **Silent Failures**: Pipeline/PipelineRun creation fails but operator logs "Build created successfully"
2. **Misleading Status**: Status shows "Running" when nothing is actually running
3. **No Root Cause**: Errors don't explain WHY something failed (e.g., RBAC permissions, missing Tasks)
4. **No Actionable Guidance**: Users don't know what to do to fix the problem
5. **Hidden Context**: Critical information buried in logs instead of surfaced in status

### Real Example from Testing

```
2025-11-09T17:26:50Z	INFO	Build created successfully	
2025-11-09T17:26:50Z	INFO	Build status updated successfully	{"status": "Running", "message": "Build created and started"}
```

**Reality**: No Pipeline or PipelineRun exists. Build is not running.

**User sees**: "Build created and started" ✅ (false positive)

**User needs to know**: 
- Pipeline creation failed due to missing RBAC permissions for Tasks
- Need to add `tasks` resource to ClusterRole
- Command to fix: `oc patch clusterrole ...`

### Philosophical Foundation (Sophia Framework)

From "The Pragmatic Coders" - methodological pragmatism requires:

1. **Explicit Fallibilism**: Acknowledge limitations and errors clearly
2. **Systematic Verification**: Provide ways to verify what actually happened
3. **Pragmatic Success Criteria**: Focus on what works and how to achieve it
4. **Error Architecture Awareness**: Distinguish between different types of errors

## Decision

Implement **Smart Error Messages** with three levels of intelligence:

### Level 1: Accurate Status Reporting ⭐ CRITICAL

**Principle**: Never report success when operation failed

**Implementation**:
```go
// BEFORE (WRONG):
logger.Info("Build created successfully", "buildName", buildName)
return &BuildInfo{Status: BuildStatusRunning}, nil

// AFTER (CORRECT):
buildInfo, err := t.createPipelineAndRun(ctx, job, buildName)
if err != nil {
    logger.Error(err, "Failed to create Pipeline/PipelineRun", "buildName", buildName)
    return nil, fmt.Errorf("failed to create Tekton build: %w", err)
}
logger.Info("Build created successfully", "buildName", buildName, "pipelineRun", buildInfo.Name)
return buildInfo, nil
```

**Verification**: Check that resource actually exists before reporting success

### Level 2: Root Cause Analysis ⭐ IMPORTANT

**Principle**: Explain WHY something failed, not just THAT it failed

**Implementation**:
```go
// Detect common failure patterns and provide specific messages
func analyzeError(err error) string {
    switch {
    case strings.Contains(err.Error(), "forbidden"):
        return "RBAC permission denied. Check ClusterRole has required permissions."
    case strings.Contains(err.Error(), "not found"):
        return "Resource not found. May need to be created first."
    case strings.Contains(err.Error(), "already exists"):
        return "Resource already exists. Consider using a different name or deleting the existing resource."
    default:
        return err.Error()
    }
}
```

**Error Categories**:
- **RBAC Errors**: Permission denied, forbidden
- **Resource Errors**: Not found, already exists
- **Configuration Errors**: Invalid spec, missing required fields
- **Platform Errors**: API not available, CRD not installed
- **Dependency Errors**: Missing prerequisite resources

### Level 3: Actionable Guidance ⭐ GAME CHANGER

**Principle**: Tell users HOW to fix the problem

**Implementation**:
```go
type SmartError struct {
    Category    string   // "RBAC", "Resource", "Configuration", etc.
    Message     string   // Human-readable error message
    RootCause   string   // Technical root cause
    Impact      string   // What this means for the user
    Actions     []string // Specific steps to fix
    References  []string // Links to docs, ADRs, examples
}

// Example:
&SmartError{
    Category:  "RBAC",
    Message:   "Failed to create Tekton Pipeline: permission denied",
    RootCause: "ClusterRole 'notebook-validator-manager-role' missing 'tasks' resource permission",
    Impact:    "Tekton builds cannot run. Operator cannot copy Tasks to user namespace.",
    Actions: []string{
        "Add 'tasks' to ClusterRole resources",
        "Run: oc patch clusterrole notebook-validator-manager-role --type='json' -p='[{\"op\": \"add\", \"path\": \"/rules/-\", \"value\": {\"apiGroups\": [\"tekton.dev\"], \"resources\": [\"tasks\"], \"verbs\": [\"create\", \"delete\", \"get\", \"list\", \"patch\", \"update\", \"watch\"]}}]'",
        "Restart operator: oc rollout restart deployment/notebook-validator-controller-manager -n jupyter-notebook-validator-operator",
    },
    References: []string{
        "ADR-028: Tekton Task Strategy",
        "config/rbac/role.yaml",
    },
}
```

### Level 4: Status Conditions (Kubernetes Best Practice)

**Principle**: Use Kubernetes Conditions for structured status reporting

**Implementation**:
```go
// Add to NotebookValidationJob status
type NotebookValidationJobStatus struct {
    // ... existing fields ...
    
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Set conditions
meta.SetStatusCondition(&job.Status.Conditions, metav1.Condition{
    Type:    "BuildReady",
    Status:  metav1.ConditionFalse,
    Reason:  "RBACPermissionDenied",
    Message: "Failed to create Tekton Pipeline: ClusterRole missing 'tasks' resource permission. See status.buildStatus.error for fix instructions.",
})

meta.SetStatusCondition(&job.Status.Conditions, metav1.Condition{
    Type:    "ValidationReady",
    Status:  metav1.ConditionFalse,
    Reason:  "WaitingForBuild",
    Message: "Waiting for build to complete before starting validation",
})
```

## Implementation Phases

### Phase 1: Fix Silent Failures (IMMEDIATE) ⏰

**Priority**: CRITICAL  
**Timeline**: Current sprint

**Tasks**:
1. ✅ Add error checking after Pipeline/PipelineRun creation
2. ✅ Verify resources exist before reporting success
3. ✅ Return errors instead of logging and continuing
4. ✅ Update tests to expect errors

**Files to Update**:
- `pkg/build/tekton_strategy.go` - Add error checking
- `internal/controller/build_integration_helper.go` - Propagate errors
- `pkg/build/tekton_strategy_test.go` - Test error cases

### Phase 2: Root Cause Analysis (SHORT-TERM) 📊

**Priority**: HIGH  
**Timeline**: Next sprint

**Tasks**:
1. Create `pkg/errors/smart_error.go` with SmartError type
2. Add error categorization logic
3. Update all error returns to use SmartError
4. Add error analysis to status messages

**Error Categories to Implement**:
- RBAC errors (forbidden, unauthorized)
- Resource errors (not found, already exists)
- Configuration errors (invalid spec, validation failed)
- Platform errors (API unavailable, CRD missing)
- Dependency errors (prerequisite missing)

### Phase 3: Actionable Guidance (MEDIUM-TERM) 🎯

**Priority**: MEDIUM  
**Timeline**: Sprint +2

**Tasks**:
1. Build knowledge base of common errors and fixes
2. Add action recommendations to SmartError
3. Include relevant ADR/doc references
4. Add CLI command suggestions

**Knowledge Base Structure**:
```yaml
errors:
  - pattern: "forbidden.*tasks"
    category: RBAC
    actions:
      - "Add 'tasks' resource to ClusterRole"
      - "Command: oc patch clusterrole ..."
    references:
      - "ADR-028"
      - "config/rbac/role.yaml"
```

### Phase 4: Status Conditions (LONG-TERM) 📈

**Priority**: LOW  
**Timeline**: Sprint +3

**Tasks**:
1. Add Conditions field to CRD status
2. Implement condition management
3. Update reconciler to set conditions
4. Add condition-based alerting

## Consequences

### Positive ✅

1. **Better UX**: Users know exactly what's wrong and how to fix it
2. **Faster Debugging**: Root cause immediately visible
3. **Self-Service**: Users can fix common issues without support
4. **Reduced Support Load**: Fewer "why isn't this working?" questions
5. **Operational Excellence**: Aligns with Kubernetes best practices
6. **Methodological Pragmatism**: Explicit about failures and how to succeed

### Negative ⚠️

1. **More Code**: Error handling logic increases codebase size
2. **Maintenance**: Knowledge base needs to be kept up-to-date
3. **Testing Complexity**: More error scenarios to test
4. **Localization**: Error messages harder to translate

### Risks 🔴

1. **Over-Engineering**: Could spend too much time on error messages
   - **Mitigation**: Implement incrementally, focus on common errors first
2. **Stale Guidance**: Fix instructions become outdated
   - **Mitigation**: Link to ADRs and docs instead of hardcoding
3. **Information Overload**: Too much detail confuses users
   - **Mitigation**: Tiered approach - summary + details on demand

## Examples

### Example 1: RBAC Permission Error

**Before**:
```
Build created successfully
Status: Running
Message: Build created and started
```

**After**:
```
Build failed: Permission denied
Category: RBAC
Root Cause: ClusterRole 'notebook-validator-manager-role' missing 'tasks' resource permission
Impact: Tekton builds cannot run. Operator cannot copy Tasks to user namespace.

Actions to fix:
1. Add 'tasks' to ClusterRole resources in config/rbac/role.yaml
2. Apply updated RBAC: oc apply -f config/rbac/role.yaml
3. Or patch directly: oc patch clusterrole notebook-validator-manager-role --type='json' -p='[...]'
4. Restart operator: oc rollout restart deployment/notebook-validator-controller-manager

References:
- ADR-028: Tekton Task Strategy
- config/rbac/role.yaml
```

### Example 2: Missing Task

**Before**:
```
Pipeline can't be Run; it contains Tasks that don't exist: Couldn't retrieve Task "git-clone"
```

**After**:
```
Build failed: Task not found
Category: Resource
Root Cause: Task 'git-clone' not found in namespace 'default'
Impact: Pipeline cannot start. Task needs to be copied from openshift-pipelines namespace.

Actions to fix:
1. Operator should automatically copy Tasks (ADR-028)
2. If automatic copy failed, check operator logs for RBAC errors
3. Manual copy: oc get task git-clone -n openshift-pipelines -o yaml | oc apply -n default -f -

References:
- ADR-028: Tekton Task Strategy (namespace copy approach)
- Operator logs: oc logs -n jupyter-notebook-validator-operator deployment/notebook-validator-controller-manager
```

## Compliance

### ADR-029: Platform Version Dependency Review

Smart error messages should include:
- Platform version compatibility information
- Links to PLATFORM-COMPATIBILITY.md
- Warnings about deprecated APIs
- Upgrade path suggestions

### ADR-028: Tekton Task Strategy

Error messages for Tekton builds should:
- Explain Task copying process
- Provide RBAC fix instructions
- Reference platform detection logic
- Link to base image documentation

## Monitoring and Metrics

Track error patterns to improve guidance:

```go
type ErrorMetrics struct {
    Category      string
    Count         int
    LastOccurred  time.Time
    FixAttempts   int
    FixSuccesses  int
}
```

**Metrics to Track**:
- Error frequency by category
- Time to resolution
- Self-service fix success rate
- Support ticket reduction

## References

- [Kubernetes API Conventions - Status](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)
- [Kubernetes Conditions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
- "The Pragmatic Coders" - Methodological Pragmatism Framework
- ADR-028: Tekton Task Strategy
- ADR-029: Platform Version Dependency Review Process

## Decision Outcome

**Chosen Option**: Implement all 4 levels incrementally

**Rationale**:
1. Level 1 (Accurate Status) is CRITICAL - fixes immediate false positives
2. Level 2 (Root Cause) is HIGH priority - dramatically improves debugging
3. Level 3 (Actionable Guidance) is MEDIUM priority - enables self-service
4. Level 4 (Status Conditions) is LOW priority - nice-to-have for advanced users

**Next Steps**:
1. Implement Phase 1 (Fix Silent Failures) immediately
2. Create `pkg/errors/smart_error.go` package
3. Update `tekton_strategy.go` to use SmartError
4. Add error knowledge base YAML file
5. Update documentation with common errors and fixes

