# Mocking Strategy for External Dependencies

## Overview

This document describes the mocking strategy for external dependencies in the Jupyter Notebook Validator Operator. By using interfaces and mocks, we can test controller logic in isolation without requiring actual external services.

## Architecture

### Interface-Based Design

We use Go interfaces to abstract external dependencies:

1. **GitOperations** - Handles Git credential resolution and container building
2. **PodLogOperations** - Handles pod log collection and parsing
3. Future interfaces for other external dependencies

### Mock Implementation

Mocks are located in `internal/controller/mocks/` and provide:
- Default behaviors for common scenarios
- Customizable function hooks for test-specific behavior
- Call tracking for verification
- Reset functionality for test isolation

## Usage Examples

### Git Operations Mock

```go
import "github.com/tosin2013/jupyter-notebook-validator-operator/internal/controller/mocks"

// Create a mock
gitOps := mocks.NewMockGitOperations()

// Customize behavior
gitOps.ResolveCredentialsFunc = func(ctx context.Context, job *NotebookValidationJob) (*GitCredentials, error) {
    return &GitCredentials{
        Type:   "ssh",
        SSHKey: "test-key",
    }, nil
}

// Use in tests
creds, err := gitOps.ResolveCredentials(ctx, job)
Expect(err).NotTo(HaveOccurred())
Expect(creds.Type).To(Equal("ssh"))

// Verify calls
Expect(gitOps.ResolveCredentialsCallCount).To(Equal(1))
```

### Pod Log Operations Mock

```go
podLogOps := mocks.NewMockPodLogOperations()

// Customize log parsing
podLogOps.ParseResultsFunc = func(logs string) (*NotebookExecutionResult, error) {
    return &NotebookExecutionResult{
        Status:   "succeeded",
        ExitCode: 0,
        Cells: []CellExecutionResult{
            {CellIndex: 0, Status: "succeeded"},
        },
    }, nil
}

// Test error scenarios
podLogOps.ParseResultsFunc = func(logs string) (*NotebookExecutionResult, error) {
    return nil, errors.New("parse error")
}
```

## Benefits

1. **Fast Tests** - No network calls or external service dependencies
2. **Deterministic** - Tests produce consistent results
3. **Isolated** - Test one component at a time
4. **Flexible** - Easy to test edge cases and error scenarios
5. **Maintainable** - Changes to external services don't break tests

## Integration with Real Implementation

The real implementations (`gitOperationsImpl`, `podLogOperationsImpl`) wrap the actual controller methods, allowing seamless switching between mocks and real implementations:

```go
// Production code uses real implementation
gitOps := NewGitOperations(reconciler)

// Test code uses mocks
gitOps := mocks.NewMockGitOperations()
```

## Future Enhancements

1. **Model Validation Mock** - Mock platform detection and model validation
2. **Kubernetes Client Mock** - Enhanced mocking for complex K8s operations
3. **Metrics Mock** - Mock Prometheus metrics for testing
4. **File System Mock** - Mock file operations if needed

## Best Practices

1. **Use mocks for external dependencies** - Network calls, file I/O, external APIs
2. **Use fake clients for Kubernetes** - Already provided by controller-runtime
3. **Keep mocks simple** - Default behaviors should cover common cases
4. **Verify important calls** - Use call tracking for critical operations
5. **Reset between tests** - Always reset mocks in BeforeEach

## Example Test Structure

```go
var _ = Describe("Controller with Mocks", func() {
    var (
        gitOps    *mocks.MockGitOperations
        podLogOps *mocks.MockPodLogOperations
    )

    BeforeEach(func() {
        gitOps = mocks.NewMockGitOperations()
        podLogOps = mocks.NewMockPodLogOperations()
    })

    It("should handle Git errors", func() {
        gitOps.ResolveCredentialsFunc = func(...) (*GitCredentials, error) {
            return nil, errors.New("git error")
        }
        // Test error handling
    })
})
```
