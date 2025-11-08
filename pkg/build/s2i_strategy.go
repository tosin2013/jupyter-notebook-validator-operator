package build

import (
	"context"
	"fmt"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	buildv1 "github.com/openshift/api/build/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// S2IStrategy implements the Strategy interface for OpenShift Source-to-Image builds
type S2IStrategy struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewS2IStrategy creates a new S2I build strategy
func NewS2IStrategy(client client.Client, scheme *runtime.Scheme) *S2IStrategy {
	return &S2IStrategy{
		client: client,
		scheme: scheme,
	}
}

// Name returns the strategy name
func (s *S2IStrategy) Name() string {
	return "s2i"
}

// Detect checks if S2I is available (OpenShift build API)
func (s *S2IStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
	// Check if BuildConfig CRD exists
	buildConfig := &buildv1.BuildConfig{}
	err := client.Get(ctx, types.NamespacedName{Name: "test", Namespace: "default"}, buildConfig)
	
	// If we get NotFound, the CRD exists but the resource doesn't - that's OK
	// If we get NoKindMatchError, the CRD doesn't exist
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		// Check if it's a "no kind match" error (CRD doesn't exist)
		if runtime.IsNotRegisteredError(err) {
			return false, nil
		}
		// Other errors might indicate the API is available but we can't access it
		return false, err
	}
	
	return true, nil
}

// CreateBuild creates an S2I build for the notebook
func (s *S2IStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
	// Check if BuildConfig is provided
	if job.Spec.PodConfig.BuildConfig == nil {
		return nil, fmt.Errorf("buildConfig is required")
	}

	buildConfig := job.Spec.PodConfig.BuildConfig

	// Generate build name
	buildName := fmt.Sprintf("%s-build", job.Name)

	// Get base image (use default if not specified)
	baseImage := buildConfig.BaseImage
	if baseImage == "" {
		baseImage = "quay.io/jupyter/minimal-notebook:latest"
	}

	// Create BuildConfig
	bc := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                                  job.Name,
				"mlops.redhat.com/notebook-validation": "true",
			},
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: job.Spec.Notebook.Git.URL,
						Ref: job.Spec.Notebook.Git.Ref,
					},
					// ContextDir is the directory containing the notebook
					// We'll use the directory part of the notebook path
					ContextDir: "",
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.SourceBuildStrategyType,
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind: "DockerImage",
							Name: baseImage,
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: fmt.Sprintf("%s:latest", buildName),
					},
				},
			},
		},
	}

	// Create the BuildConfig
	if err := s.client.Create(ctx, bc); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create BuildConfig: %w", err)
		}
	}

	// Trigger a build
	build := &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-1", buildName),
			Namespace: job.Namespace,
			Labels: map[string]string{
				"buildconfig": buildName,
			},
		},
		Spec: buildv1.BuildSpec{
			CommonSpec: buildv1.CommonSpec{
				Source:   bc.Spec.Source,
				Strategy: bc.Spec.Strategy,
				Output:   bc.Spec.Output,
			},
		},
	}

	if err := s.client.Create(ctx, build); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to trigger build: %w", err)
		}
	}

	now := time.Now()
	return &BuildInfo{
		Name:      build.Name,
		Status:    BuildStatusPending,
		Message:   "Build created and triggered",
		StartTime: &now,
	}, nil
}

// GetBuildStatus returns the current build status
func (s *S2IStrategy) GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error) {
	// Parse namespace from context or use default
	namespace := "default" // TODO: Get from context
	
	build := &buildv1.Build{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: buildName, Namespace: namespace}, build); err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	info := &BuildInfo{
		Name:    build.Name,
		Message: build.Status.Message,
	}

	// Map OpenShift build phase to our BuildStatus
	switch build.Status.Phase {
	case buildv1.BuildPhaseNew, buildv1.BuildPhasePending:
		info.Status = BuildStatusPending
	case buildv1.BuildPhaseRunning:
		info.Status = BuildStatusRunning
	case buildv1.BuildPhaseComplete:
		info.Status = BuildStatusComplete
		if build.Status.Output.To != nil {
			info.ImageReference = build.Status.Output.To.ImageDigest
		}
	case buildv1.BuildPhaseFailed, buildv1.BuildPhaseError:
		info.Status = BuildStatusFailed
	case buildv1.BuildPhaseCancelled:
		info.Status = BuildStatusCancelled
	default:
		info.Status = BuildStatusUnknown
	}

	if build.Status.StartTimestamp != nil {
		info.StartTime = &build.Status.StartTimestamp.Time
	}
	if build.Status.CompletionTimestamp != nil {
		info.CompletionTime = &build.Status.CompletionTimestamp.Time
	}

	return info, nil
}

// WaitForCompletion waits for the build to complete
func (s *S2IStrategy) WaitForCompletion(ctx context.Context, buildName string, timeout time.Duration) (*BuildInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for build to complete")
		case <-ticker.C:
			info, err := s.GetBuildStatus(ctx, buildName)
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

// GetBuildLogs returns the build logs
func (s *S2IStrategy) GetBuildLogs(ctx context.Context, buildName string) (string, error) {
	// TODO: Implement log streaming from OpenShift build
	// This requires using the OpenShift REST API or build client
	return "", fmt.Errorf("log streaming not yet implemented")
}

// DeleteBuild cleans up build resources
func (s *S2IStrategy) DeleteBuild(ctx context.Context, buildName string) error {
	namespace := "default" // TODO: Get from context
	
	// Delete Build
	build := &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: namespace,
		},
	}
	if err := s.client.Delete(ctx, build); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete build: %w", err)
	}

	// Delete BuildConfig
	buildConfig := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: namespace,
		},
	}
	if err := s.client.Delete(ctx, buildConfig); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete buildconfig: %w", err)
	}

	return nil
}

// ValidateConfig validates the S2I build configuration
func (s *S2IStrategy) ValidateConfig(config *mlopsv1alpha1.BuildConfigSpec) error {
	if config.BaseImage == "" {
		return fmt.Errorf("baseImage is required for S2I builds")
	}
	return nil
}

