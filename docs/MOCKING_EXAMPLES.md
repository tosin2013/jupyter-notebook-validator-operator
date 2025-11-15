# Mocking Examples for External Dependencies

## Overview

This document provides practical examples of using mocks for external dependencies in the Jupyter Notebook Validator Operator.

## Mock Libraries Created

### 1. Git Operations Mock (`mocks/git_operations_mock.go`)

Mocks Git credential resolution and container building operations.

**Usage Example:**

```go
import "github.com/tosin2013/jupyter-notebook-validator-operator/internal/controller/mocks"

// Create a mock
gitOps := mocks.NewMockGitOperations()

// Customize behavior for SSH credentials
gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *NotebookValidationJob) (*mocks.GitCredentials, error) {
    return &mocks.GitCredentials{
        Type:   "ssh",
        SSHKey: "test-ssh-key",
    }, nil
}

// Use in test
creds, err := gitOps.ResolveCredentials(ctx, job)
Expect(err).NotTo(HaveOccurred())
Expect(creds.Type).To(Equal("ssh"))

// Verify it was called
Expect(gitOps.ResolveCredentialsCallCount).To(Equal(1))
```

**Error Scenario:**

```go
// Mock error case
gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *NotebookValidationJob) (*mocks.GitCredentials, error) {
    return nil, k8serrors.NewNotFound(corev1.Resource("secret"), "missing-secret")
}

creds, err := gitOps.ResolveCredentials(ctx, job)
Expect(err).To(HaveOccurred())
Expect(k8serrors.IsNotFound(err)).To(BeTrue())
```

### 2. Pod Log Operations Mock (`mocks/pod_log_operations_mock.go`)

Mocks pod log collection and parsing operations.

**Usage Example:**

```go
podLogOps := mocks.NewMockPodLogOperations()

// Mock successful log parsing
podLogOps.ParseResultsFunc = func(logs string) (*mocks.NotebookExecutionResult, error) {
    return &mocks.NotebookExecutionResult{
        Status:   "succeeded",
        ExitCode: 0,
        Cells: []mocks.CellExecutionResult{
            {
                CellIndex: 0,
                CellType:  "code",
                Status:    "succeeded",
            },
        },
        Statistics: mocks.ExecutionStatistics{
            TotalCells:  1,
            CodeCells:   1,
            FailedCells: 0,
            SuccessRate: 100.0,
        },
    }, nil
}

// Use in test
result, err := podLogOps.ParseResults("test logs")
Expect(err).NotTo(HaveOccurred())
Expect(result.Status).To(Equal("succeeded"))
```

**Error Extraction:**

```go
podLogOps.ExtractErrorFunc = func(logs string) string {
    return "ERROR: Notebook execution failed"
}

errorMsg := podLogOps.ExtractError("some logs with ERROR: Notebook execution failed")
Expect(errorMsg).To(ContainSubstring("ERROR"))
```

## Integration Patterns

### Pattern 1: Dependency Injection

For new code, inject dependencies through interfaces:

```go
type MyController struct {
    gitOps    GitOperations
    podLogOps PodLogOperations
}

func NewMyController(gitOps GitOperations, podLogOps PodLogOperations) *MyController {
    return &MyController{
        gitOps:    gitOps,
        podLogOps: podLogOps,
    }
}

// In tests
gitOps := mocks.NewMockGitOperations()
podLogOps := mocks.NewMockPodLogOperations()
controller := NewMyController(gitOps, podLogOps)
```

### Pattern 2: Wrapper Functions

For existing code, create wrapper functions that can be mocked:

```go
// Production code
func (r *Reconciler) doSomething() {
    gitOps := NewGitOperations(r)  // Real implementation
    creds, _ := gitOps.ResolveCredentials(ctx, job)
    // ...
}

// Test code
func TestDoSomething(t *testing.T) {
    gitOps := mocks.NewMockGitOperations()
    // Test with mock
}
```

## Call Verification

Mocks track method calls for verification:

```go
gitOps.Reset()  // Clear call counts

// Make calls
_, _ = gitOps.ResolveCredentials(ctx, job1)
_, _ = gitOps.ResolveCredentials(ctx, job2)
_, _ = gitOps.BuildCloneInitContainer(ctx, job, creds)

// Verify
Expect(gitOps.ResolveCredentialsCallCount).To(Equal(2))
Expect(gitOps.BuildCloneInitContainerCallCount).To(Equal(1))

// Or use verification helper
gitOps.VerifyCallCounts(GinkgoT(), map[string]int{
    "ResolveCredentials":      2,
    "BuildCloneInitContainer": 1,
})
```

## Benefits

1. **Fast Tests** - No network calls or external service dependencies
2. **Deterministic** - Tests produce consistent results
3. **Isolated** - Test one component at a time
4. **Flexible** - Easy to test edge cases and error scenarios
5. **Maintainable** - Changes to external services don't break tests

## Best Practices

1. **Reset mocks between tests** - Use `BeforeEach` to reset call counts
2. **Use default behaviors** - Mocks have sensible defaults for common cases
3. **Customize only what you need** - Override specific functions for test scenarios
4. **Verify important calls** - Use call tracking for critical operations
5. **Test error paths** - Mock error scenarios to test error handling

## Future Enhancements

Additional mocks can be created for:
- Model validation operations
- Platform detection
- Metrics collection
- File system operations (if needed)
