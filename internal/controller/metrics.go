/*
Copyright 2024.

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

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Reconciliation duration histogram
	// Tracks how long reconciliation loops take
	reconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notebookvalidationjob_reconciliation_duration_seconds",
			Help:    "Duration of NotebookValidationJob reconciliation in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"namespace", "result"},
	)

	// Validation job counters
	// Tracks total number of validation jobs by outcome
	validationJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notebookvalidationjob_validations_total",
			Help: "Total number of notebook validations",
		},
		[]string{"namespace", "status"},
	)

	// Git clone duration histogram
	// Tracks Git clone performance by authentication type
	gitCloneDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notebookvalidationjob_git_clone_duration_seconds",
			Help:    "Duration of Git clone operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120},
		},
		[]string{"namespace", "auth_type"},
	)

	// Active pod gauge
	// Tracks number of active validation pods by phase
	activePods = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "notebookvalidationjob_active_pods",
			Help: "Number of active validation pods",
		},
		[]string{"namespace", "phase"},
	)

	// Reconciliation errors counter
	// Tracks reconciliation errors by type
	reconciliationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notebookvalidationjob_reconciliation_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"namespace", "error_type"},
	)

	// Pod creation counter
	// Tracks pod creation attempts
	podCreations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notebookvalidationjob_pod_creations_total",
			Help: "Total number of validation pod creation attempts",
		},
		[]string{"namespace", "result"},
	)

	// Model validation metrics (ADR-020: Model-Aware Validation)
	// Tracks model validation duration by platform
	modelValidationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notebookvalidationjob_model_validation_duration_seconds",
			Help:    "Duration of model validation operations in seconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"namespace", "platform", "result"},
	)

	// Model health checks counter
	// Tracks model health check attempts by platform and result
	modelHealthChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notebookvalidationjob_model_health_checks_total",
			Help: "Total number of model health checks",
		},
		[]string{"namespace", "platform", "status"},
	)

	// Prediction validations counter
	// Tracks prediction validation attempts and results
	predictionValidations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notebookvalidationjob_prediction_validations_total",
			Help: "Total number of prediction validations",
		},
		[]string{"namespace", "platform", "result"},
	)

	// Platform detection duration histogram
	// Tracks platform detection performance
	platformDetectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notebookvalidationjob_platform_detection_duration_seconds",
			Help:    "Duration of platform detection operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"namespace", "platform", "detected"},
	)
)

func init() {
	// Register metrics with controller-runtime's registry
	metrics.Registry.MustRegister(
		reconciliationDuration,
		validationJobsTotal,
		gitCloneDuration,
		activePods,
		reconciliationErrors,
		podCreations,
		modelValidationDuration,
		modelHealthChecks,
		predictionValidations,
		platformDetectionDuration,
	)
}

// recordReconciliationDuration records the duration of a reconciliation loop
func recordReconciliationDuration(namespace, result string, duration float64) {
	reconciliationDuration.WithLabelValues(namespace, result).Observe(duration)
}

// recordValidationComplete records a completed validation job
func recordValidationComplete(namespace, status string) {
	validationJobsTotal.WithLabelValues(namespace, status).Inc()
}

// setActivePods sets the number of active pods for a given phase
func setActivePods(namespace, phase string, count float64) {
	activePods.WithLabelValues(namespace, phase).Set(count)
}

// recordPodCreation records a pod creation attempt
func recordPodCreation(namespace, result string) {
	podCreations.WithLabelValues(namespace, result).Inc()
}

// recordModelValidationDuration records the duration of a model validation operation
func recordModelValidationDuration(namespace, platform, result string, duration float64) {
	modelValidationDuration.WithLabelValues(namespace, platform, result).Observe(duration)
}

// recordModelHealthCheck records a model health check attempt
// TODO(ADR-020): Implement model health check functionality
//
//nolint:unused // Reserved for ADR-020 Model-Aware Validation Strategy
func recordModelHealthCheck(namespace, platform, status string) {
	modelHealthChecks.WithLabelValues(namespace, platform, status).Inc()
}

// recordPredictionValidation records a prediction validation attempt
// TODO(ADR-020): Implement prediction validation functionality
//
//nolint:unused // Reserved for ADR-020 Model-Aware Validation Strategy
func recordPredictionValidation(namespace, platform, result string) {
	predictionValidations.WithLabelValues(namespace, platform, result).Inc()
}

// recordPlatformDetection records the duration of a platform detection operation
func recordPlatformDetection(namespace, platform string, detected bool, duration float64) {
	detectedStr := "false"
	if detected {
		detectedStr = "true"
	}
	platformDetectionDuration.WithLabelValues(namespace, platform, detectedStr).Observe(duration)
}
