# Testing Strategy for Kubernetes Operator

## Mock Libraries and Fake Clients

### Current Approach: controller-runtime Fake Client

We are using **`sigs.k8s.io/controller-runtime/pkg/client/fake`**, which is the **official and recommended** fake client library from the Kubernetes SIG (Special Interest Group). This is the best choice for the following reasons:

1. **Official Support**: Maintained by the Kubernetes community and kept in sync with Kubernetes API versions
2. **Version Compatibility**: Fully compatible with Kubernetes v0.29.2 (which we're using)
3. **No Additional Dependencies**: Already included with `controller-runtime` (no need to add external libraries)
4. **Standard Practice**: Used by most Kubernetes operators in production

### Why Not Other Mock Libraries?

While there are other mock libraries available, they have limitations:

- **`github.com/golang/mock`** or **`github.com/uber-go/mock`**: These are generic mock generators, but they don't understand Kubernetes API semantics and would require extensive manual setup
- **Custom mocks**: Would need to be maintained separately and kept in sync with Kubernetes API changes
- **Third-party Kubernetes mocks**: Often outdated or not compatible with newer Kubernetes versions

### Using the Fake Client

The fake client from controller-runtime is used like this:

```go
import (
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
    "k8s.io/apimachinery/pkg/runtime"
)

// Setup
scheme := runtime.NewScheme()
_ = corev1.AddToScheme(scheme)
_ = mlopsv1alpha1.AddToScheme(scheme)

fakeClient := fake.NewClientBuilder().
    WithScheme(scheme).
    WithObjects(initialObjects...).  // Optional: pre-populate with objects
    Build()

reconciler := &NotebookValidationJobReconciler{
    Client: fakeClient,
    Scheme: scheme,
}
```

### Known Limitations

The fake client has some limitations (as documented in the official docs):

1. **No OpenAPI Validation**: Objects are not validated against OpenAPI schemas
2. **Limited Subresource Support**: Updating metadata and status in the same reconcile can be tricky
3. **Generation/ResourceVersion**: These fields don't behave exactly like a real API server
4. **No Error Injection**: Can't easily test specific error conditions

### When to Use Fake Client vs envtest

**Use Fake Client (Unit Tests)**:
- Testing business logic
- Testing helper functions
- Fast execution
- No external dependencies

**Use envtest (Integration Tests)**:
- Testing full reconciliation flow
- Testing with real API server behavior
- Testing webhooks
- Testing finalizers

### Example: Our Test Structure

```go
// Unit tests with fake client (fast, isolated)
var _ = Describe("NotebookValidationJobReconciler", func() {
    var fakeClient client.Client
    
    BeforeEach(func() {
        fakeClient = fake.NewClientBuilder().
            WithScheme(scheme).
            Build()
    })
    
    It("should create validation pod", func() {
        // Test with fake client
    })
})

// Integration tests with envtest (slower, more realistic)
var _ = Describe("NotebookValidationJob Integration", func() {
    var testEnv *envtest.Environment
    
    BeforeSuite(func() {
        testEnv = &envtest.Environment{
            CRDDirectoryPaths: []string{"../../config/crd/bases"},
        }
        cfg, _ := testEnv.Start()
        // Use real client with test API server
    })
})
```

### Best Practices

1. **Use fake client for unit tests** - Fast, isolated, no cluster needed
2. **Use envtest for integration tests** - More realistic, tests full flow
3. **Keep tests focused** - Test one thing at a time
4. **Test error paths** - Use fake client's limitations to test error handling
5. **Mock external dependencies** - For things like Git operations, use interfaces

### Conclusion

**We don't need additional mock libraries** - the fake client from controller-runtime is the standard, maintained, and compatible solution. It's already in our dependencies and works perfectly with Kubernetes v0.29.2.
