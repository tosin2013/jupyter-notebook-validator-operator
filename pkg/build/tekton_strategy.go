package build

import (
	"context"
	"fmt"
	"time"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonStrategy implements the Strategy interface for Tekton Pipelines
type TektonStrategy struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewTektonStrategy creates a new Tekton build strategy
func NewTektonStrategy(client client.Client, scheme *runtime.Scheme) *TektonStrategy {
	return &TektonStrategy{
		client: client,
		scheme: scheme,
	}
}

// Name returns the strategy name
func (t *TektonStrategy) Name() string {
	return "tekton"
}

// Detect checks if Tekton is available in the cluster
func (t *TektonStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
	// Check if TaskRun CRD exists
	taskRun := &tektonv1.TaskRun{}
	err := client.Get(ctx, types.NamespacedName{Name: "test", Namespace: "default"}, taskRun)

	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		if runtime.IsNotRegisteredError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateBuild creates a Tekton TaskRun for building the notebook image
func (t *TektonStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
	// Check if BuildConfig is provided
	if job.Spec.PodConfig.BuildConfig == nil {
		return nil, fmt.Errorf("buildConfig is required")
	}

	buildConfig := job.Spec.PodConfig.BuildConfig
	buildName := fmt.Sprintf("%s-build", job.Name)

	// Get registry configuration from strategyConfig or use defaults
	registry := "image-registry.openshift-image-registry.svc:5000"
	if val, ok := buildConfig.StrategyConfig["registry"]; ok {
		registry = val
	}

	imageRef := fmt.Sprintf("%s/%s/%s:latest", registry, job.Namespace, buildName)

	// Create a Pipeline with git-clone + buildah tasks
	pipeline := t.createBuildPipeline(job, buildConfig, imageRef)
	if err := t.client.Create(ctx, pipeline); err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	// Create PipelineRun
	pipelineRun := t.createPipelineRun(job, buildName, pipeline.Name)
	if err := t.client.Create(ctx, pipelineRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create pipelinerun: %w", err)
		}
	}

	now := time.Now()
	return &BuildInfo{
		Name:           pipelineRun.Name,
		Status:         BuildStatusPending,
		Message:        "Tekton pipeline created and triggered",
		ImageReference: imageRef,
		StartTime:      &now,
	}, nil
}

// createBuildPipeline creates a Tekton Pipeline with git-clone and buildah tasks
func (t *TektonStrategy) createBuildPipeline(job *mlopsv1alpha1.NotebookValidationJob, buildConfig *mlopsv1alpha1.BuildConfigSpec, imageRef string) *tektonv1.Pipeline {
	// Get base image (use default if not specified)
	baseImage := buildConfig.BaseImage
	if baseImage == "" {
		baseImage = "quay.io/jupyter/minimal-notebook:latest"
	}

	return &tektonv1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pipeline", job.Name),
			Namespace: job.Namespace,
		},
		Spec: tektonv1.PipelineSpec{
			Params: []tektonv1.ParamSpec{
				{Name: "git-url", Type: tektonv1.ParamTypeString},
				{Name: "git-revision", Type: tektonv1.ParamTypeString, Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "main"}},
				{Name: "image-reference", Type: tektonv1.ParamTypeString},
				{Name: "base-image", Type: tektonv1.ParamTypeString, Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: baseImage}},
			},
			Workspaces: []tektonv1.PipelineWorkspaceDeclaration{
				{Name: "shared-workspace"},
			},
			Tasks: []tektonv1.PipelineTask{
				{
					Name: "fetch-repository",
					TaskRef: &tektonv1.TaskRef{
						Name: "git-clone",
						Kind: "ClusterTask",
					},
					Params: []tektonv1.Param{
						{Name: "url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-url)"}},
						{Name: "revision", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-revision)"}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: "shared-workspace"},
					},
				},
				{
					Name: "build-image",
					TaskRef: &tektonv1.TaskRef{
						Name: "buildah",
						Kind: "ClusterTask",
					},
					RunAfter: []string{"fetch-repository"},
					Params: []tektonv1.Param{
						{Name: "IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.image-reference)"}},
						{Name: "BUILDER_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.base-image)"}},
						{Name: "CONTEXT", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "."}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "source", Workspace: "shared-workspace"},
					},
				},
			},
		},
	}
}

// createPipelineRun creates a PipelineRun for the build pipeline
func (t *TektonStrategy) createPipelineRun(job *mlopsv1alpha1.NotebookValidationJob, buildName, pipelineName string) *tektonv1.PipelineRun {
	// Get base image (use default if not specified)
	baseImage := "quay.io/jupyter/minimal-notebook:latest"
	if job.Spec.PodConfig.BuildConfig != nil && job.Spec.PodConfig.BuildConfig.BaseImage != "" {
		baseImage = job.Spec.PodConfig.BuildConfig.BaseImage
	}

	return &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                                  job.Name,
				"mlops.redhat.com/notebook-validation": "true",
			},
		},
		Spec: tektonv1.PipelineRunSpec{
			PipelineRef: &tektonv1.PipelineRef{
				Name: pipelineName,
			},
			Params: []tektonv1.Param{
				{Name: "git-url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: job.Spec.Notebook.Git.URL}},
				{Name: "git-revision", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: job.Spec.Notebook.Git.Ref}},
				{Name: "image-reference", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", job.Namespace, buildName)}},
				{Name: "base-image", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: baseImage}},
			},
			Workspaces: []tektonv1.WorkspaceBinding{
				{
					Name: "shared-workspace",
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: *resource.NewQuantity(1*1024*1024*1024, resource.BinarySI),
								},
							},
						},
					},
				},
			},
		},
	}
}

// GetBuildStatus returns the current build status for a Tekton TaskRun or PipelineRun
func (t *TektonStrategy) GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error) {
	// List all PipelineRuns with our label
	pipelineRunList := &tektonv1.PipelineRunList{}
	if err := t.client.List(ctx, pipelineRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return nil, fmt.Errorf("failed to list pipelineruns: %w", err)
	}

	// Find the PipelineRun with matching name
	for i := range pipelineRunList.Items {
		if pipelineRunList.Items[i].Name == buildName {
			return t.getPipelineRunStatus(&pipelineRunList.Items[i]), nil
		}
	}

	// Try TaskRuns
	taskRunList := &tektonv1.TaskRunList{}
	if err := t.client.List(ctx, taskRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return nil, fmt.Errorf("failed to list taskruns: %w", err)
	}

	// Find the TaskRun with matching name
	for i := range taskRunList.Items {
		if taskRunList.Items[i].Name == buildName {
			return t.getTaskRunStatus(&taskRunList.Items[i]), nil
		}
	}

	return nil, fmt.Errorf("build not found: %s", buildName)
}

// getPipelineRunStatus extracts status from a PipelineRun
func (t *TektonStrategy) getPipelineRunStatus(pr *tektonv1.PipelineRun) *BuildInfo {
	info := &BuildInfo{
		Name: pr.Name,
	}

	// Get status from conditions
	for _, condition := range pr.Status.Conditions {
		if condition.Type == "Succeeded" {
			switch condition.Status {
			case corev1.ConditionTrue:
				info.Status = BuildStatusComplete
				info.Message = condition.Message
			case corev1.ConditionFalse:
				info.Status = BuildStatusFailed
				info.Message = condition.Message
			case corev1.ConditionUnknown:
				info.Status = BuildStatusRunning
				info.Message = condition.Message
			}
		}
	}

	if pr.Status.StartTime != nil {
		info.StartTime = &pr.Status.StartTime.Time
	}
	if pr.Status.CompletionTime != nil {
		info.CompletionTime = &pr.Status.CompletionTime.Time
	}

	return info
}

// getTaskRunStatus extracts status from a TaskRun
func (t *TektonStrategy) getTaskRunStatus(tr *tektonv1.TaskRun) *BuildInfo {
	info := &BuildInfo{
		Name: tr.Name,
	}

	// Get status from conditions
	for _, condition := range tr.Status.Conditions {
		if condition.Type == "Succeeded" {
			switch condition.Status {
			case corev1.ConditionTrue:
				info.Status = BuildStatusComplete
				info.Message = condition.Message
			case corev1.ConditionFalse:
				info.Status = BuildStatusFailed
				info.Message = condition.Message
			case corev1.ConditionUnknown:
				info.Status = BuildStatusRunning
				info.Message = condition.Message
			}
		}
	}

	if tr.Status.StartTime != nil {
		info.StartTime = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		info.CompletionTime = &tr.Status.CompletionTime.Time
	}

	return info
}

// WaitForCompletion waits for the Tekton build to complete
func (t *TektonStrategy) WaitForCompletion(ctx context.Context, buildName string, timeout time.Duration) (*BuildInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for build to complete")
		case <-ticker.C:
			info, err := t.GetBuildStatus(ctx, buildName)
			if err != nil {
				return nil, err
			}

			switch info.Status {
			case BuildStatusComplete:
				return info, nil
			case BuildStatusFailed, BuildStatusCancelled:
				return info, fmt.Errorf("build failed: %s", info.Message)
			}
		}
	}
}

// GetBuildLogs returns the build logs from Tekton
func (t *TektonStrategy) GetBuildLogs(ctx context.Context, buildName string) (string, error) {
	// TODO: Implement log streaming from Tekton TaskRun/PipelineRun pods
	return "", fmt.Errorf("log streaming not yet implemented for Tekton")
}

// DeleteBuild cleans up Tekton build resources
func (t *TektonStrategy) DeleteBuild(ctx context.Context, buildName string) error {
	// List and delete PipelineRuns
	pipelineRunList := &tektonv1.PipelineRunList{}
	if err := t.client.List(ctx, pipelineRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return fmt.Errorf("failed to list pipelineruns: %w", err)
	}

	for i := range pipelineRunList.Items {
		if pipelineRunList.Items[i].Name == buildName {
			if err := t.client.Delete(ctx, &pipelineRunList.Items[i]); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete pipelinerun: %w", err)
			}
		}
	}

	// List and delete TaskRuns
	taskRunList := &tektonv1.TaskRunList{}
	if err := t.client.List(ctx, taskRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return fmt.Errorf("failed to list taskruns: %w", err)
	}

	for i := range taskRunList.Items {
		if taskRunList.Items[i].Name == buildName {
			if err := t.client.Delete(ctx, &taskRunList.Items[i]); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete taskrun: %w", err)
			}
		}
	}

	// List and delete Pipelines
	pipelineList := &tektonv1.PipelineList{}
	if err := t.client.List(ctx, pipelineList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	pipelineName := fmt.Sprintf("%s-pipeline", buildName)
	for i := range pipelineList.Items {
		if pipelineList.Items[i].Name == pipelineName {
			if err := t.client.Delete(ctx, &pipelineList.Items[i]); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete pipeline: %w", err)
			}
		}
	}

	return nil
}

// ValidateConfig validates the Tekton build configuration
func (t *TektonStrategy) ValidateConfig(config *mlopsv1alpha1.BuildConfigSpec) error {
	// BaseImage is optional - we have a default
	// No specific validation needed for Tekton
	return nil
}
