package argocd

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// LabelNotificationType is the label key for notification type
	LabelNotificationType = "mlops.dev/notification-type"
	// NotificationTypeValidationSuccess indicates a successful validation
	NotificationTypeValidationSuccess = "validation-success"
	// NotificationTypeValidationFailure indicates a failed validation
	NotificationTypeValidationFailure = "validation-failure"
)

// CreateNotificationEvent creates a Kubernetes Event with ArgoCD notification labels
// This enables ArgoCD notifications to trigger alerts based on notebook validation results
func CreateNotificationEvent(
	ctx context.Context,
	clientset kubernetes.Interface,
	scheme *runtime.Scheme,
	obj client.Object,
	eventType string,
	reason string,
	message string,
) error {
	logger := log.FromContext(ctx)

	// Determine notification type based on event type
	notificationType := NotificationTypeValidationFailure
	if eventType == corev1.EventTypeNormal {
		notificationType = NotificationTypeValidationSuccess
	}

	// Build structured message with metadata for notification parsing
	structuredMessage := buildStructuredMessage(obj, reason, message)

	// Create event with labels for ArgoCD notifications
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", obj.GetName()),
			Namespace:    obj.GetNamespace(),
			Labels: map[string]string{
				LabelNotificationType: notificationType,
			},
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      obj.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			Kind:            obj.GetObjectKind().GroupVersionKind().Kind,
			Namespace:       obj.GetNamespace(),
			Name:            obj.GetName(),
			UID:             obj.GetUID(),
			ResourceVersion: obj.GetResourceVersion(),
		},
		Type:    eventType,
		Reason:  reason,
		Message: structuredMessage,
		Source: corev1.EventSource{
			Component: "jupyter-notebook-validator-operator",
		},
		FirstTimestamp:      metav1.Now(),
		LastTimestamp:       metav1.Now(),
		Count:               1,
		ReportingController: "jupyter-notebook-validator-operator",
		ReportingInstance:   "controller",
	}

	// Create the event
	_, err := clientset.CoreV1().Events(obj.GetNamespace()).Create(ctx, event, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create notification event: %w", err)
	}

	logger.V(1).Info("Created notification event",
		"type", notificationType,
		"reason", reason,
		"namespace", obj.GetNamespace(),
		"name", obj.GetName())

	return nil
}

// buildStructuredMessage builds a structured message with metadata for notification parsing
func buildStructuredMessage(obj client.Object, reason, message string) string {
	annotations := obj.GetAnnotations()
	labels := obj.GetLabels()

	// Build structured message with key information
	structured := fmt.Sprintf("NotebookValidationJob '%s' %s\n\n", obj.GetName(), reason)
	structured += fmt.Sprintf("Message: %s\n", message)

	// Add sync wave information if available
	if wave := annotations["argocd.argoproj.io/sync-wave"]; wave != "" {
		structured += fmt.Sprintf("Sync Wave: %s\n", wave)
	}

	// Add part-of label if available (for Application identification)
	if partOf := labels["app.kubernetes.io/part-of"]; partOf != "" {
		structured += fmt.Sprintf("Application: %s\n", partOf)
	}

	// Add timestamp
	structured += fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339))

	// Add namespace
	structured += fmt.Sprintf("Namespace: %s\n", obj.GetNamespace())

	return structured
}
