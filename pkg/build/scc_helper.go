package build

import (
	"context"
	"fmt"

	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SCCHelper provides shared SCC management functionality for build strategies
// ADR-039: Automatic SCC Management for Build Strategies
// ADR-044: Extracted to eliminate code duplication between S2I and Tekton
type SCCHelper struct {
	client    client.Client
	apiReader client.Reader // Non-cached client for SCC Gets
}

// NewSCCHelper creates a new SCC helper
func NewSCCHelper(client client.Client, apiReader client.Reader) *SCCHelper {
	return &SCCHelper{
		client:    client,
		apiReader: apiReader,
	}
}

// EnsureServiceAccount ensures that a ServiceAccount exists in the namespace
// Creates the ServiceAccount if it doesn't exist, with appropriate labels
func (h *SCCHelper) EnsureServiceAccount(ctx context.Context, namespace, serviceAccountName string, labels map[string]string) error {
	logger := log.FromContext(ctx)

	sa := &corev1.ServiceAccount{}
	err := h.client.Get(ctx, client.ObjectKey{
		Name:      serviceAccountName,
		Namespace: namespace,
	}, sa)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check ServiceAccount %s: %w", serviceAccountName, err)
	}

	if errors.IsNotFound(err) {
		// ServiceAccount doesn't exist, create it
		logger.Info("Creating ServiceAccount",
			"serviceAccount", serviceAccountName,
			"namespace", namespace)

		newSA := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: namespace,
				Labels:    labels,
			},
		}

		if err := h.client.Create(ctx, newSA); err != nil {
			return fmt.Errorf("failed to create ServiceAccount %s: %w", serviceAccountName, err)
		}
		logger.Info("Successfully created ServiceAccount",
			"serviceAccount", serviceAccountName,
			"namespace", namespace)
	} else {
		logger.V(1).Info("ServiceAccount already exists",
			"serviceAccount", serviceAccountName,
			"namespace", namespace)
	}

	return nil
}

// GrantSCCToServiceAccount grants a SecurityContextConstraint to a ServiceAccount
// This automates the manual "oc adm policy add-scc-to-user" command
// ADR-044: Uses pipelines-scc for better security posture (SETFCAP only vs RunAsAny)
func (h *SCCHelper) GrantSCCToServiceAccount(ctx context.Context, namespace, serviceAccount, sccName string) error {
	logger := log.FromContext(ctx)

	// Get the SCC using APIReader (non-cached) to avoid triggering watch/list attempts
	// Since we only need to Get specific SCCs by name, we don't need caching
	scc := &securityv1.SecurityContextConstraints{}
	err := h.apiReader.Get(ctx, client.ObjectKey{Name: sccName}, scc)
	if err != nil {
		if errors.IsNotFound(err) {
			// SCC doesn't exist - likely Kubernetes without OpenShift
			return fmt.Errorf("SCC %s not found (Kubernetes cluster?): %w", sccName, err)
		}
		return fmt.Errorf("failed to get SCC %s: %w", sccName, err)
	}

	// Check if ServiceAccount already has the SCC
	serviceAccountUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount)
	for _, user := range scc.Users {
		if user == serviceAccountUser {
			logger.V(1).Info("ServiceAccount already has SCC",
				"namespace", namespace,
				"serviceAccount", serviceAccount,
				"scc", sccName)
			return nil
		}
	}

	// Add ServiceAccount to SCC users
	logger.Info("Granting SCC to ServiceAccount",
		"namespace", namespace,
		"serviceAccount", serviceAccount,
		"scc", sccName)

	scc.Users = append(scc.Users, serviceAccountUser)

	if err := h.client.Update(ctx, scc); err != nil {
		return fmt.Errorf("failed to update SCC %s: %w", sccName, err)
	}

	logger.Info("Successfully granted SCC to ServiceAccount",
		"namespace", namespace,
		"serviceAccount", serviceAccount,
		"scc", sccName)

	return nil
}

// EnsureBuildServiceAccountWithSCC is a convenience function that creates a ServiceAccount
// and grants it the specified SCC, with graceful degradation on non-OpenShift clusters
func (h *SCCHelper) EnsureBuildServiceAccountWithSCC(
	ctx context.Context,
	namespace, serviceAccountName, sccName string,
	labels map[string]string) error {

	logger := log.FromContext(ctx)

	// Step 1: Ensure ServiceAccount exists
	if err := h.EnsureServiceAccount(ctx, namespace, serviceAccountName, labels); err != nil {
		return err
	}

	// Step 2: Grant SCC to ServiceAccount
	// Log warning but don't fail - this might be a Kubernetes cluster without SCCs
	if err := h.GrantSCCToServiceAccount(ctx, namespace, serviceAccountName, sccName); err != nil {
		logger.Info("Failed to grant SCC (might be Kubernetes without OpenShift SCCs)",
			"error", err,
			"namespace", namespace,
			"serviceAccount", serviceAccountName,
			"scc", sccName)
		logger.Info(fmt.Sprintf("If on OpenShift, manually grant SCC: oc adm policy add-scc-to-user %s -z %s -n %s",
			sccName, serviceAccountName, namespace))
		// Don't return error - allow operation to continue on non-OpenShift clusters
	}

	return nil
}
