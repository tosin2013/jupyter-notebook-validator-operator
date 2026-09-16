package build

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestTektonGitCredentialsConversion verifies ADR-042: creating a git-credentials
// Secret and verifying that the TektonStrategy creates the corresponding Tekton
// annotation-based secret.
func TestTektonGitCredentialsConversion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-credentials",
			Namespace: "test-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("myuser"),
			"password": []byte("mytoken"),
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sourceSecret).
		Build()

	strategy := NewTektonStrategy(cl, cl, scheme)

	ctx := context.Background()
	err := strategy.ensureTektonGitCredentials(ctx, "test-ns", "git-credentials")
	if err != nil {
		t.Fatalf("ensureTektonGitCredentials failed: %v", err)
	}

	tektonSecretName := "git-credentials-tekton"
	tektonSecret := &corev1.Secret{}
	err = cl.Get(ctx, types.NamespacedName{
		Name:      tektonSecretName,
		Namespace: "test-ns",
	}, tektonSecret)
	if err != nil {
		t.Fatalf("expected Tekton secret %q to be created, got error: %v", tektonSecretName, err)
	}

	if tektonSecret.Annotations["tekton.dev/git-0"] == "" {
		t.Error("expected tekton.dev/git-0 annotation to be set")
	}

	if tektonSecret.Labels[LabelManagedBy] != LabelManagedByValue {
		t.Errorf("expected label %s=%s, got %q", LabelManagedBy, LabelManagedByValue, tektonSecret.Labels[LabelManagedBy])
	}
}

// TestTektonGitCredentials_SecretAlreadyExists verifies that re-running
// EnsureGitCredentials when the Tekton secret already exists does not fail.
func TestTektonGitCredentials_SecretAlreadyExists(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-credentials",
			Namespace: "test-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("myuser"),
			"password": []byte("mytoken"),
		},
	}

	tektonSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-credentials-tekton",
			Namespace: "test-ns",
			Labels: map[string]string{
				LabelManagedBy: LabelManagedByValue,
			},
			Annotations: map[string]string{
				"tekton.dev/git-0": "https://github.com",
			},
		},
		Type: corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			"username": []byte("myuser"),
			"password": []byte("mytoken"),
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sourceSecret, tektonSecret).
		Build()

	strategy := NewTektonStrategy(cl, cl, scheme)

	ctx := context.Background()
	err := strategy.ensureTektonGitCredentials(ctx, "test-ns", "git-credentials")
	if err != nil {
		t.Fatalf("ensureTektonGitCredentials with existing secret should not fail: %v", err)
	}
}

// TestTektonGitCredentials_WatchSyncGap documents the known gap in ADR-042:
// the watch-and-sync path (automatic update when source secret changes) is
// not yet wired up. This test exists to document the gap per #29.
func TestTektonGitCredentials_WatchSyncGap(t *testing.T) {
	t.Log("ADR-042 watch-and-sync gap: the TODO at tekton_strategy.go:252 indicates " +
		"that automatic updates to Tekton secrets when the source git-credentials " +
		"Secret changes are not yet implemented. The Tekton secret is created once " +
		"but not updated. See issue #29 for tracking.")
}
