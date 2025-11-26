/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var notebookvalidationjoblog = logf.Log.WithName("notebookvalidationjob-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NotebookValidationJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-mlops-mlops-dev-v1alpha1-notebookvalidationjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=mlops.mlops.dev,resources=notebookvalidationjobs,verbs=create;update,versions=v1alpha1,name=mnotebookvalidationjob.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &NotebookValidationJob{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *NotebookValidationJob) Default(ctx context.Context, obj runtime.Object) error {
	job, ok := obj.(*NotebookValidationJob)
	if !ok {
		return fmt.Errorf("expected a NotebookValidationJob object but got %T", obj)
	}
	notebookvalidationjoblog.Info("default", "name", job.Name, "namespace", job.Namespace)

	// Convert credentials array to envFrom (syntactic sugar)
	// ADR-014: Simplified credential injection pattern
	// The credentials field is a convenient shorthand that gets converted to envFrom with secretRef
	if len(job.Spec.PodConfig.Credentials) > 0 {
		notebookvalidationjoblog.Info("converting credentials to envFrom",
			"name", job.Name,
			"namespace", job.Namespace,
			"credentials", job.Spec.PodConfig.Credentials)

		// Convert each credential secret name to an envFrom entry
		for _, secretName := range job.Spec.PodConfig.Credentials {
			job.Spec.PodConfig.EnvFrom = append(job.Spec.PodConfig.EnvFrom, EnvFromSource{
				SecretRef: &SecretEnvSource{
					Name: secretName,
				},
			})
		}

		// Clear the credentials field after conversion to avoid confusion
		// The field is only used for input; envFrom is the canonical representation
		job.Spec.PodConfig.Credentials = nil

		notebookvalidationjoblog.Info("credentials converted to envFrom",
			"name", job.Name,
			"namespace", job.Namespace,
			"envFromCount", len(job.Spec.PodConfig.EnvFrom))
	}

	// Set default ServiceAccount to "default" if not specified
	// This allows the validation pod to run in any namespace without requiring
	// manual ServiceAccount creation. Users can override this by specifying
	// a custom ServiceAccount in the spec.
	//
	// Design rationale (see research on OpenTelemetry Operator, Istio, Vault patterns):
	// - Using "default" SA is the simplest approach for cross-namespace operation
	// - Avoids requiring users to manually create ServiceAccounts in every namespace
	// - Follows the principle of least surprise - "default" SA exists in all namespaces
	// - Users can grant additional permissions to "default" SA if needed
	// - Future enhancement: implement annotation-based injection for custom SAs
	if job.Spec.PodConfig.ServiceAccountName == "" {
		notebookvalidationjoblog.Info("injecting default ServiceAccount",
			"name", job.Name,
			"namespace", job.Namespace,
			"serviceAccount", "default")
		job.Spec.PodConfig.ServiceAccountName = "default"
	}

	// Set default timeout if not specified
	if job.Spec.Timeout == "" {
		job.Spec.Timeout = "30m"
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-mlops-mlops-dev-v1alpha1-notebookvalidationjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=mlops.mlops.dev,resources=notebookvalidationjobs,verbs=create;update,versions=v1alpha1,name=vnotebookvalidationjob.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &NotebookValidationJob{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job, ok := obj.(*NotebookValidationJob)
	if !ok {
		return nil, fmt.Errorf("expected a NotebookValidationJob object but got %T", obj)
	}
	notebookvalidationjoblog.Info("validate create", "name", job.Name, "namespace", job.Namespace)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	job, ok := newObj.(*NotebookValidationJob)
	if !ok {
		return nil, fmt.Errorf("expected a NotebookValidationJob object but got %T", newObj)
	}
	notebookvalidationjoblog.Info("validate update", "name", job.Name, "namespace", job.Namespace)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	job, ok := obj.(*NotebookValidationJob)
	if !ok {
		return nil, fmt.Errorf("expected a NotebookValidationJob object but got %T", obj)
	}
	notebookvalidationjoblog.Info("validate delete", "name", job.Name, "namespace", job.Namespace)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
