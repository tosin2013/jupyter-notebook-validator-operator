package controller

import (
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

const (
	// Build condition types
	ConditionTypeBuildStarted  = "BuildStarted"
	ConditionTypeBuildComplete = "BuildComplete"
	ConditionTypeBuildFailed   = "BuildFailed"

	// Build condition reasons
	ReasonBuildCreated      = "BuildCreated"
	ReasonBuildInProgress   = "BuildInProgress"
	ReasonBuildSucceeded    = "BuildSucceeded"
	ReasonBuildFailedReason = "BuildFailed"
	ReasonBuildTimeout      = "BuildTimeout"
	ReasonStrategyNotFound  = "StrategyNotFound"
	ReasonConfigInvalid     = "ConfigInvalid"
	ReasonBuildNotEnabled   = "BuildNotEnabled"

	// Build defaults
	DefaultBuildTimeout = 15 * time.Minute
)

// isBuildEnabled checks if build is enabled in the job spec
func isBuildEnabled(job *mlopsv1alpha1.NotebookValidationJob) bool {
	return job.Spec.PodConfig.BuildConfig != nil && job.Spec.PodConfig.BuildConfig.Enabled
}
