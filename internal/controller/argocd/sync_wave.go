package argocd

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// AnnotationSyncWave is the ArgoCD sync wave annotation
	AnnotationSyncWave = "argocd.argoproj.io/sync-wave"
	// AnnotationBlockWave indicates which wave should be blocked until this job completes
	AnnotationBlockWave = "mlops.dev/block-wave"
	// AnnotationWaveComplete indicates the wave number that completed successfully
	AnnotationWaveComplete = "mlops.dev/wave-complete"
	// AnnotationWaveFailed indicates the wave number that failed
	AnnotationWaveFailed = "mlops.dev/wave-failed"
	// AnnotationCompletionTime is the RFC3339 timestamp when the job completed
	AnnotationCompletionTime = "mlops.dev/completion-time"
)

// SetSyncWaveAnnotations sets ArgoCD sync wave annotations on job completion
// This enables GitOps tools to coordinate deployment ordering
func SetSyncWaveAnnotations(ctx context.Context, c client.Client, obj client.Object, phase string) error {
	logger := log.FromContext(ctx)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Get the sync wave number from annotations
	syncWave := annotations[AnnotationSyncWave]
	if syncWave == "" {
		// No sync wave annotation, nothing to do
		logger.V(1).Info("No sync wave annotation found, skipping wave annotation logic")
		return nil
	}

	completionTime := time.Now().Format(time.RFC3339)

	// Set completion annotations based on phase
	if phase == "Succeeded" {
		annotations[AnnotationWaveComplete] = syncWave
		annotations[AnnotationCompletionTime] = completionTime
		// Remove wave-failed if it was set
		delete(annotations, AnnotationWaveFailed)
		logger.Info("Set sync wave completion annotation",
			"wave", syncWave,
			"completionTime", completionTime)
	} else if phase == "Failed" {
		annotations[AnnotationWaveFailed] = syncWave
		annotations[AnnotationCompletionTime] = completionTime
		// Remove wave-complete if it was set
		delete(annotations, AnnotationWaveComplete)
		logger.Info("Set sync wave failure annotation",
			"wave", syncWave,
			"completionTime", completionTime)
	}

	obj.SetAnnotations(annotations)

	// Update the object
	if err := c.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update sync wave annotations: %w", err)
	}

	return nil
}
