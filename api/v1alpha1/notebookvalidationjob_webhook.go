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
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-mlops-mlops-dev-v1alpha1-notebookvalidationjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=mlops.mlops.dev,resources=notebookvalidationjobs,verbs=create;update,versions=v1alpha1,name=mnotebookvalidationjob.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NotebookValidationJob{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NotebookValidationJob) Default() {
	notebookvalidationjoblog.Info("default", "name", r.Name, "namespace", r.Namespace)

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
	if r.Spec.PodConfig.ServiceAccountName == "" {
		notebookvalidationjoblog.Info("injecting default ServiceAccount", 
			"name", r.Name, 
			"namespace", r.Namespace,
			"serviceAccount", "default")
		r.Spec.PodConfig.ServiceAccountName = "default"
	}

	// Set default timeout if not specified
	if r.Spec.Timeout == "" {
		r.Spec.Timeout = "30m"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-mlops-mlops-dev-v1alpha1-notebookvalidationjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=mlops.mlops.dev,resources=notebookvalidationjobs,verbs=create;update,versions=v1alpha1,name=vnotebookvalidationjob.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NotebookValidationJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateCreate() (admission.Warnings, error) {
	notebookvalidationjoblog.Info("validate create", "name", r.Name, "namespace", r.Namespace)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	notebookvalidationjoblog.Info("validate update", "name", r.Name, "namespace", r.Namespace)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NotebookValidationJob) ValidateDelete() (admission.Warnings, error) {
	notebookvalidationjoblog.Info("validate delete", "name", r.Name, "namespace", r.Namespace)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

