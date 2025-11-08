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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NotebookValidationJobSpec defines the desired state of NotebookValidationJob
type NotebookValidationJobSpec struct {
	// Notebook specifies the notebook to validate
	// +kubebuilder:validation:Required
	Notebook NotebookSpec `json:"notebook"`

	// PodConfig specifies the pod configuration for validation execution
	// +kubebuilder:validation:Required
	PodConfig PodConfigSpec `json:"podConfig"`

	// GoldenNotebook specifies the golden notebook for comparison (optional)
	// +optional
	GoldenNotebook *NotebookSpec `json:"goldenNotebook,omitempty"`

	// ComparisonConfig specifies advanced comparison configuration (optional)
	// +optional
	ComparisonConfig *ComparisonConfigSpec `json:"comparisonConfig,omitempty"`

	// Timeout specifies the maximum execution time for the validation job
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="30m"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// ModelValidation specifies optional model-aware validation configuration
	// +optional
	ModelValidation *ModelValidationSpec `json:"modelValidation,omitempty"`
}

// NotebookSpec defines the notebook source
type NotebookSpec struct {
	// Git specifies the Git repository containing the notebook
	// +kubebuilder:validation:Required
	Git GitSpec `json:"git"`

	// Path is the path to the notebook file within the repository
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^.*\.ipynb$`
	Path string `json:"path"`
}

// GitSpec defines Git repository configuration
type GitSpec struct {
	// URL is the Git repository URL (supports https://, git://, ssh://, and git@host:path formats)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^((https?|git|ssh)://|git@).*$`
	URL string `json:"url"`

	// Ref is the Git reference (branch, tag, or commit SHA)
	// +kubebuilder:validation:Required
	Ref string `json:"ref"`

	// CredentialsSecret is the name of the Kubernetes Secret containing Git credentials
	// +optional
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

// PodConfigSpec defines the pod configuration for validation execution
type PodConfigSpec struct {
	// ContainerImage is the container image to use for validation
	// +kubebuilder:validation:Required
	ContainerImage string `json:"containerImage"`

	// Resources specifies the compute resources for the validation pod
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use for the validation pod
	// +kubebuilder:default="jupyter-notebook-validator-runner"
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Env specifies environment variables for the validation pod
	// +optional
	Env []EnvVar `json:"env,omitempty"`

	// EnvFrom specifies sources to populate environment variables in the validation pod
	// +optional
	EnvFrom []EnvFromSource `json:"envFrom,omitempty"`
}

// ResourceRequirements defines compute resource requirements
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed
	// +optional
	Limits map[string]string `json:"limits,omitempty"`

	// Requests describes the minimum amount of compute resources required
	// +optional
	Requests map[string]string `json:"requests,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	// Name is the environment variable name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Value is the environment variable value
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom specifies a source for the environment variable value
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource represents a source for an environment variable value
type EnvVarSource struct {
	// SecretKeyRef selects a key from a Secret
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`

	// ConfigMapKeyRef selects a key from a ConfigMap
	// +optional
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	// Name is the name of the Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the Secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// ConfigMapKeySelector selects a key from a ConfigMap
type ConfigMapKeySelector struct {
	// Name is the name of the ConfigMap
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the ConfigMap
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// EnvFromSource represents a source to populate environment variables
type EnvFromSource struct {
	// SecretRef references a Secret to populate environment variables
	// +optional
	SecretRef *SecretEnvSource `json:"secretRef,omitempty"`

	// ConfigMapRef references a ConfigMap to populate environment variables
	// +optional
	ConfigMapRef *ConfigMapEnvSource `json:"configMapRef,omitempty"`
}

// SecretEnvSource selects a Secret to populate environment variables
type SecretEnvSource struct {
	// Name is the name of the Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// ConfigMapEnvSource selects a ConfigMap to populate environment variables
type ConfigMapEnvSource struct {
	// Name is the name of the ConfigMap
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// ComparisonConfigSpec defines advanced comparison configuration
type ComparisonConfigSpec struct {
	// Strategy specifies the comparison strategy
	// +kubebuilder:validation:Enum=exact;normalized
	// +kubebuilder:default="normalized"
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// FloatingPointTolerance specifies the tolerance for floating-point comparisons
	// Stored as string to avoid float serialization issues across languages
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`
	// +kubebuilder:default="0.0001"
	// +optional
	FloatingPointTolerance *string `json:"floatingPointTolerance,omitempty"`

	// IgnoreTimestamps specifies whether to ignore timestamp differences
	// +kubebuilder:default=true
	// +optional
	IgnoreTimestamps *bool `json:"ignoreTimestamps,omitempty"`

	// IgnoreExecutionCount specifies whether to ignore execution count differences
	// +kubebuilder:default=true
	// +optional
	IgnoreExecutionCount *bool `json:"ignoreExecutionCount,omitempty"`

	// CustomTimestampPatterns specifies additional regex patterns for timestamp detection
	// +optional
	CustomTimestampPatterns []string `json:"customTimestampPatterns,omitempty"`

	// IgnoreOutputTypes specifies output types to ignore during comparison
	// +optional
	IgnoreOutputTypes []string `json:"ignoreOutputTypes,omitempty"`
}

// ModelValidationSpec defines model-aware validation configuration
type ModelValidationSpec struct {
	// Enabled specifies whether model validation is enabled
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Platform specifies the model serving platform
	// +kubebuilder:validation:Enum=kserve;openshift-ai;vllm;torchserve;tensorflow-serving;triton;ray-serve;seldon;bentoml;custom
	// +kubebuilder:default="kserve"
	// +optional
	Platform string `json:"platform,omitempty"`

	// Phase specifies which validation phase(s) to run
	// +kubebuilder:validation:Enum=clean;existing;both
	// +kubebuilder:default="both"
	// +optional
	Phase string `json:"phase,omitempty"`

	// TargetModels specifies the list of model names to validate against
	// +optional
	TargetModels []string `json:"targetModels,omitempty"`

	// PredictionValidation specifies prediction consistency validation configuration
	// +optional
	PredictionValidation *PredictionValidationSpec `json:"predictionValidation,omitempty"`

	// CustomPlatform specifies custom platform configuration for community platforms
	// +optional
	CustomPlatform *CustomPlatformSpec `json:"customPlatform,omitempty"`

	// Timeout specifies the maximum time for model validation
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// PredictionValidationSpec defines prediction consistency validation configuration
type PredictionValidationSpec struct {
	// Enabled specifies whether prediction validation is enabled
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// TestData specifies the test input data for prediction validation
	// +optional
	TestData string `json:"testData,omitempty"`

	// ExpectedOutput specifies the expected prediction output
	// +optional
	ExpectedOutput string `json:"expectedOutput,omitempty"`

	// Tolerance specifies the tolerance for floating-point prediction comparisons
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`
	// +kubebuilder:default="0.01"
	// +optional
	Tolerance string `json:"tolerance,omitempty"`
}

// CustomPlatformSpec defines custom platform configuration for community platforms
type CustomPlatformSpec struct {
	// APIGroup specifies the Kubernetes API group for the platform's CRDs
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`

	// ResourceType specifies the resource type for model resources
	// +optional
	ResourceType string `json:"resourceType,omitempty"`

	// HealthCheckEndpoint specifies the health check endpoint pattern
	// +optional
	HealthCheckEndpoint string `json:"healthCheckEndpoint,omitempty"`

	// PredictionEndpoint specifies the prediction endpoint pattern
	// +optional
	PredictionEndpoint string `json:"predictionEndpoint,omitempty"`
}

// NotebookValidationJobStatus defines the observed state of NotebookValidationJob
type NotebookValidationJobStatus struct {
	// Phase represents the current phase of the validation job
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the job's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// StartTime is when the validation started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the validation completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Results contains cell-by-cell execution results
	// +optional
	Results []CellResult `json:"results,omitempty"`

	// ValidationPodName is the name of the pod executing the validation
	// +optional
	ValidationPodName string `json:"validationPodName,omitempty"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`

	// RetryCount tracks the number of retry attempts
	// +optional
	RetryCount int `json:"retryCount,omitempty"`

	// LastRetryTime is when the last retry occurred
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`

	// ComparisonResult contains the golden notebook comparison result
	// +optional
	ComparisonResult *ComparisonResult `json:"comparisonResult,omitempty"`

	// ModelValidationResult contains the model validation result
	// +optional
	ModelValidationResult *ModelValidationResult `json:"modelValidationResult,omitempty"`
}

// ModelValidationResult represents the result of model-aware validation
type ModelValidationResult struct {
	// Phase indicates which validation phase was executed
	// +kubebuilder:validation:Enum=clean;existing;both
	Phase string `json:"phase"`

	// Platform indicates the detected or specified platform
	Platform string `json:"platform"`

	// PlatformDetected indicates whether the platform was auto-detected
	PlatformDetected bool `json:"platformDetected"`

	// CleanEnvironmentCheck contains Phase 1 validation results
	// +optional
	CleanEnvironmentCheck *CleanEnvironmentCheckResult `json:"cleanEnvironmentCheck,omitempty"`

	// ExistingEnvironmentCheck contains Phase 2 validation results
	// +optional
	ExistingEnvironmentCheck *ExistingEnvironmentCheckResult `json:"existingEnvironmentCheck,omitempty"`

	// Success indicates overall model validation success
	Success bool `json:"success"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`
}

// CleanEnvironmentCheckResult represents Phase 1 validation results
type CleanEnvironmentCheckResult struct {
	// PlatformAvailable indicates if the model serving platform is available
	PlatformAvailable bool `json:"platformAvailable"`

	// CRDsInstalled lists the detected CRDs
	// +optional
	CRDsInstalled []string `json:"crdsInstalled,omitempty"`

	// RBACPermissions indicates if required RBAC permissions are present
	RBACPermissions bool `json:"rbacPermissions"`

	// NetworkConnectivity indicates if network connectivity to model endpoints is available
	NetworkConnectivity bool `json:"networkConnectivity"`

	// RequiredLibraries lists the detected required libraries
	// +optional
	RequiredLibraries []string `json:"requiredLibraries,omitempty"`

	// Success indicates Phase 1 validation success
	Success bool `json:"success"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`
}

// ExistingEnvironmentCheckResult represents Phase 2 validation results
type ExistingEnvironmentCheckResult struct {
	// ModelsChecked lists the models that were validated
	// +optional
	ModelsChecked []ModelCheckResult `json:"modelsChecked,omitempty"`

	// PredictionValidation contains prediction consistency validation results
	// +optional
	PredictionValidation *PredictionValidationResult `json:"predictionValidation,omitempty"`

	// Success indicates Phase 2 validation success
	Success bool `json:"success"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`
}

// ModelCheckResult represents validation results for a single model
type ModelCheckResult struct {
	// ModelName is the name of the model
	ModelName string `json:"modelName"`

	// Available indicates if the model is available
	Available bool `json:"available"`

	// Healthy indicates if the model passed health checks
	Healthy bool `json:"healthy"`

	// Version is the detected model version
	// +optional
	Version string `json:"version,omitempty"`

	// ResourceStatus contains resource utilization information
	// +optional
	ResourceStatus *ModelResourceStatus `json:"resourceStatus,omitempty"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`
}

// ModelResourceStatus represents model resource utilization
type ModelResourceStatus struct {
	// CPUUsage is the CPU usage percentage
	// +optional
	CPUUsage string `json:"cpuUsage,omitempty"`

	// MemoryUsage is the memory usage
	// +optional
	MemoryUsage string `json:"memoryUsage,omitempty"`

	// GPUUsage is the GPU usage percentage
	// +optional
	GPUUsage string `json:"gpuUsage,omitempty"`
}

// PredictionValidationResult represents prediction consistency validation results
type PredictionValidationResult struct {
	// Success indicates if predictions matched expected output
	Success bool `json:"success"`

	// ActualOutput is the actual prediction output
	// +optional
	ActualOutput string `json:"actualOutput,omitempty"`

	// ExpectedOutput is the expected prediction output
	// +optional
	ExpectedOutput string `json:"expectedOutput,omitempty"`

	// Difference is the calculated difference between actual and expected
	// +optional
	Difference string `json:"difference,omitempty"`

	// Message provides a human-readable summary
	// +optional
	Message string `json:"message,omitempty"`
}

// CellResult represents the execution result of a single notebook cell
type CellResult struct {
	// CellIndex is the zero-based index of the cell
	// +kubebuilder:validation:Minimum=0
	CellIndex int `json:"cellIndex"`

	// Status is the execution status of the cell
	// +kubebuilder:validation:Enum=Success;Failure;Skipped
	Status string `json:"status"`

	// ExecutionTime is how long the cell took to execute
	// +optional
	ExecutionTime *metav1.Duration `json:"executionTime,omitempty"`

	// Output is the cell's stdout/stderr (truncated if too long)
	// +optional
	Output string `json:"output,omitempty"`

	// ErrorMessage is the error message if the cell failed
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ComparisonResult contains the golden notebook comparison result
type ComparisonResult struct {
	// Strategy used for comparison (exact, normalized, fuzzy, semantic)
	// +kubebuilder:validation:Enum=exact;normalized;fuzzy;semantic
	Strategy string `json:"strategy"`

	// Result is the overall comparison result (matched, failed, skipped)
	// +kubebuilder:validation:Enum=matched;failed;skipped
	Result string `json:"result"`

	// TotalCells is the total number of cells compared
	// +kubebuilder:validation:Minimum=0
	TotalCells int `json:"totalCells"`

	// MatchedCells is the number of cells that matched
	// +kubebuilder:validation:Minimum=0
	MatchedCells int `json:"matchedCells"`

	// MismatchedCells is the number of cells that did not match
	// +kubebuilder:validation:Minimum=0
	MismatchedCells int `json:"mismatchedCells"`

	// Diffs contains detailed diff information for mismatched cells
	// +optional
	Diffs []CellDiff `json:"diffs,omitempty"`
}

// CellDiff represents a difference between executed and golden notebook cells
type CellDiff struct {
	// CellIndex is the index of the cell (0-based)
	// +kubebuilder:validation:Minimum=0
	CellIndex int `json:"cellIndex"`

	// CellType is the type of cell (code, markdown)
	// +kubebuilder:validation:Enum=code;markdown
	CellType string `json:"cellType"`

	// DiffType describes the type of difference
	// +kubebuilder:validation:Enum=output_mismatch;execution_error;missing_cell;extra_cell
	DiffType string `json:"diffType"`

	// Expected is the expected output from golden notebook (truncated if too long)
	// +optional
	Expected string `json:"expected,omitempty"`

	// Actual is the actual output from executed notebook (truncated if too long)
	// +optional
	Actual string `json:"actual,omitempty"`

	// Diff is the unified diff format (truncated if too long)
	// +optional
	Diff string `json:"diff,omitempty"`

	// Severity indicates the importance of the difference
	// +kubebuilder:validation:Enum=minor;major;critical
	Severity string `json:"severity"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=nvj;nvjob
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Notebook",type=string,JSONPath=`.spec.notebook.path`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,priority=1

// NotebookValidationJob is the Schema for the notebookvalidationjobs API
type NotebookValidationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotebookValidationJobSpec   `json:"spec,omitempty"`
	Status NotebookValidationJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NotebookValidationJobList contains a list of NotebookValidationJob
type NotebookValidationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NotebookValidationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NotebookValidationJob{}, &NotebookValidationJobList{})
}
