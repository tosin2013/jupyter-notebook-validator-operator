package build

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	logger := log.FromContext(ctx)

	// Check if BuildConfig CRD exists by trying to list BuildConfigs
	// This is more reliable than trying to Get a specific resource
	buildConfigList := &buildv1.BuildConfigList{}
	err := client.List(ctx, buildConfigList)

	if err != nil {
		logger.V(1).Info("S2I detection: error listing BuildConfigs",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"isNotFound", errors.IsNotFound(err),
			"isNotRegistered", runtime.IsNotRegisteredError(err))

		// Check if it's a "no kind match" error (CRD doesn't exist)
		if runtime.IsNotRegisteredError(err) {
			logger.Info("S2I not available: BuildConfig CRD not registered")
			return false, nil
		}

		// Check for "no matches for kind" error (API not available)
		if strings.Contains(err.Error(), "no matches for kind") {
			logger.Info("S2I not available: BuildConfig API not found")
			return false, nil
		}

		// Other errors might indicate permission issues
		logger.Error(err, "S2I detection failed with unexpected error")
		return false, err
	}

	logger.Info("S2I available: BuildConfig API detected", "buildConfigCount", len(buildConfigList.Items))
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

	logger := log.FromContext(ctx)

	// Create ImageStream first (required for BuildConfig output)
	imageStream := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                                  job.Name,
				"mlops.redhat.com/notebook-validation": "true",
			},
		},
	}

	if err := s.client.Create(ctx, imageStream); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create ImageStream", "imageStreamName", buildName)
			return nil, fmt.Errorf("failed to create ImageStream: %w", err)
		}
		logger.Info("ImageStream already exists", "imageStreamName", buildName)
	} else {
		logger.Info("ImageStream created successfully", "imageStreamName", buildName)
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
				Source: func() buildv1.BuildSource {
					source := buildv1.BuildSource{
						Type: buildv1.BuildSourceGit,
						Git: &buildv1.GitBuildSource{
							URI: job.Spec.Notebook.Git.URL,
							Ref: job.Spec.Notebook.Git.Ref,
						},
						// ContextDir is the directory containing the notebook
						// We'll use the directory part of the notebook path
						ContextDir: "",
					}

					// Add Git credentials secret if specified
					if job.Spec.Notebook.Git.CredentialsSecret != "" {
						source.SourceSecret = &corev1.LocalObjectReference{
							Name: job.Spec.Notebook.Git.CredentialsSecret,
						}
					}

					return source
				}(),
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
	logger.Info("Creating BuildConfig", "buildConfigName", buildName, "gitURL", job.Spec.Notebook.Git.URL, "hasCredentials", job.Spec.Notebook.Git.CredentialsSecret != "")
	if err := s.client.Create(ctx, bc); err != nil {
		if !errors.IsAlreadyExists(err) {
			logger.Error(err, "Failed to create BuildConfig", "buildConfigName", buildName)
			return nil, fmt.Errorf("failed to create BuildConfig: %w", err)
		}
		logger.Info("BuildConfig already exists", "buildConfigName", buildName)
	} else {
		logger.Info("BuildConfig created successfully", "buildConfigName", buildName)
	}

	// Trigger a build using BuildRequest (proper OpenShift API)
	// This uses the BuildConfig's instantiate subresource which actually starts the build
	buildRequest := &buildv1.BuildRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: job.Namespace,
		},
	}

	logger.Info("Triggering build via BuildRequest", "buildConfigName", buildName)

	// Use SubResource to call the instantiate endpoint
	// This is equivalent to: oc start-build <buildconfig-name>
	build := &buildv1.Build{}
	if err := s.client.SubResource("instantiate").Create(ctx, bc, build, buildRequest); err != nil {
		logger.Error(err, "Failed to trigger build via instantiate", "buildConfigName", buildName)
		return nil, fmt.Errorf("failed to trigger build: %w", err)
	}

	logger.Info("Build created and triggered successfully", "buildName", build.Name, "buildConfigName", buildName)

	now := time.Now()
	return &BuildInfo{
		Name:      build.Name,
		Status:    BuildStatusPending,
		Message:   "Build created and started via BuildRequest",
		StartTime: &now,
	}, nil
}

// GetBuildStatus returns the current build status
func (s *S2IStrategy) GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error) {
	// List all builds with this name across namespaces
	buildList := &buildv1.BuildList{}
	if err := s.client.List(ctx, buildList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}

	// Find the build with matching name
	var build *buildv1.Build
	for i := range buildList.Items {
		if buildList.Items[i].Name == buildName {
			build = &buildList.Items[i]
			break
		}
	}

	if build == nil {
		return nil, fmt.Errorf("build not found: %s", buildName)
	}

	return s.buildInfoFromBuild(build), nil
}

// buildInfoFromBuild converts an OpenShift Build to BuildInfo
func (s *S2IStrategy) buildInfoFromBuild(build *buildv1.Build) *BuildInfo {
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

	return info
}

// GetLatestBuild returns the most recent build for a BuildConfig
// Prioritizes: Complete > Running > Pending > Failed
func (s *S2IStrategy) GetLatestBuild(ctx context.Context, buildConfigName string) (*BuildInfo, error) {
	logger := log.FromContext(ctx)

	// List all builds for this BuildConfig
	buildList := &buildv1.BuildList{}
	if err := s.client.List(ctx, buildList, client.MatchingLabels{
		"buildconfig":                          buildConfigName,
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}

	if len(buildList.Items) == 0 {
		return nil, fmt.Errorf("no builds found for BuildConfig: %s", buildConfigName)
	}

	logger.Info("Found builds for BuildConfig", "buildConfigName", buildConfigName, "count", len(buildList.Items))

	// Categorize builds by status
	var completedBuilds, runningBuilds, pendingBuilds, failedBuilds []*buildv1.Build

	for i := range buildList.Items {
		build := &buildList.Items[i]
		switch build.Status.Phase {
		case buildv1.BuildPhaseComplete:
			completedBuilds = append(completedBuilds, build)
		case buildv1.BuildPhaseRunning:
			runningBuilds = append(runningBuilds, build)
		case buildv1.BuildPhaseNew, buildv1.BuildPhasePending:
			pendingBuilds = append(pendingBuilds, build)
		case buildv1.BuildPhaseFailed, buildv1.BuildPhaseError, buildv1.BuildPhaseCancelled:
			failedBuilds = append(failedBuilds, build)
		}
	}

	// Priority: Complete > Running > Pending > Failed
	// Within each category, choose the most recent (by creation timestamp)
	var selectedBuild *buildv1.Build

	if len(completedBuilds) > 0 {
		selectedBuild = s.getMostRecentBuild(completedBuilds)
		logger.Info("Using completed build", "buildName", selectedBuild.Name)
	} else if len(runningBuilds) > 0 {
		selectedBuild = s.getMostRecentBuild(runningBuilds)
		logger.Info("Using running build", "buildName", selectedBuild.Name)
	} else if len(pendingBuilds) > 0 {
		selectedBuild = s.getMostRecentBuild(pendingBuilds)
		logger.Info("Using pending build", "buildName", selectedBuild.Name)
	} else if len(failedBuilds) > 0 {
		selectedBuild = s.getMostRecentBuild(failedBuilds)
		logger.Info("Using failed build", "buildName", selectedBuild.Name)
	}

	if selectedBuild == nil {
		return nil, fmt.Errorf("no suitable build found for BuildConfig: %s", buildConfigName)
	}

	return s.buildInfoFromBuild(selectedBuild), nil
}

// getMostRecentBuild returns the build with the most recent creation timestamp
func (s *S2IStrategy) getMostRecentBuild(builds []*buildv1.Build) *buildv1.Build {
	if len(builds) == 0 {
		return nil
	}

	mostRecent := builds[0]
	for _, build := range builds[1:] {
		if build.CreationTimestamp.After(mostRecent.CreationTimestamp.Time) {
			mostRecent = build
		}
	}
	return mostRecent
}

// TriggerBuild manually triggers a build that's stuck in New/Pending status
func (s *S2IStrategy) TriggerBuild(ctx context.Context, buildName string) error {
	logger := log.FromContext(ctx)

	// List all builds to find the one we want
	buildList := &buildv1.BuildList{}
	if err := s.client.List(ctx, buildList, client.MatchingLabels{
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return fmt.Errorf("failed to list builds: %w", err)
	}

	// Find the build
	var targetBuild *buildv1.Build
	for i := range buildList.Items {
		if buildList.Items[i].Name == buildName {
			targetBuild = &buildList.Items[i]
			break
		}
	}

	if targetBuild == nil {
		return fmt.Errorf("build not found: %s", buildName)
	}

	buildConfigName := targetBuild.Labels["buildconfig"]
	if buildConfigName == "" {
		return fmt.Errorf("build %s has no buildconfig label", buildName)
	}

	logger.Info("Triggering stuck build by updating build phase", "buildName", buildName, "buildConfigName", buildConfigName)

	// Update the build to trigger it (change phase from New to Pending)
	// This simulates what the build controller should do
	targetBuild.Status.Phase = buildv1.BuildPhasePending
	if err := s.client.Status().Update(ctx, targetBuild); err != nil {
		logger.Error(err, "Failed to update build status, build may need manual intervention")
		return fmt.Errorf("failed to trigger build: %w", err)
	}

	logger.Info("Build phase updated to Pending", "buildName", buildName)
	return nil
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

// GetImageFromImageStream checks ImageStream for recently pushed image
func (s *S2IStrategy) GetImageFromImageStream(ctx context.Context, imageStreamName string) (string, error) {
	logger := log.FromContext(ctx)

	// Get the ImageStream
	imageStreamList := &imagev1.ImageStreamList{}
	if err := s.client.List(ctx, imageStreamList, client.MatchingLabels{
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return "", fmt.Errorf("failed to list ImageStreams: %w", err)
	}

	// Find the ImageStream with matching name
	var foundImageStream *imagev1.ImageStream
	for i := range imageStreamList.Items {
		if imageStreamList.Items[i].Name == imageStreamName {
			foundImageStream = &imageStreamList.Items[i]
			break
		}
	}

	if foundImageStream == nil {
		return "", fmt.Errorf("ImageStream not found: %s", imageStreamName)
	}

	// Check if there are any tags
	if len(foundImageStream.Status.Tags) == 0 {
		return "", fmt.Errorf("ImageStream %s has no tags", imageStreamName)
	}

	// Get the latest tag (usually "latest" or the most recent)
	var latestTag *imagev1.NamedTagEventList
	for i := range foundImageStream.Status.Tags {
		tag := &foundImageStream.Status.Tags[i]
		if tag.Tag == "latest" {
			latestTag = tag
			break
		}
	}

	// If no "latest" tag, use the first tag with items
	if latestTag == nil {
		for i := range foundImageStream.Status.Tags {
			tag := &foundImageStream.Status.Tags[i]
			if len(tag.Items) > 0 {
				latestTag = tag
				break
			}
		}
	}

	if latestTag == nil || len(latestTag.Items) == 0 {
		return "", fmt.Errorf("ImageStream %s has no image items", imageStreamName)
	}

	// Get the most recent image
	latestImage := latestTag.Items[0]
	imageRef := fmt.Sprintf("%s@%s", foundImageStream.Status.DockerImageRepository, latestImage.Image)

	logger.Info("Found image in ImageStream", "imageStreamName", imageStreamName, "imageRef", imageRef)
	return imageRef, nil
}

// CleanupOldBuilds removes old builds to prevent resource accumulation
func (s *S2IStrategy) CleanupOldBuilds(ctx context.Context, buildConfigName string, keepCount int) error {
	logger := log.FromContext(ctx)

	// List all builds for this BuildConfig
	buildList := &buildv1.BuildList{}
	if err := s.client.List(ctx, buildList, client.MatchingLabels{
		"buildconfig":                          buildConfigName,
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return fmt.Errorf("failed to list builds: %w", err)
	}

	if len(buildList.Items) <= keepCount {
		logger.V(1).Info("No builds to clean up", "buildConfigName", buildConfigName, "totalBuilds", len(buildList.Items), "keepCount", keepCount)
		return nil
	}

	// Sort builds by creation timestamp (newest first)
	builds := buildList.Items
	sort.Slice(builds, func(i, j int) bool {
		return builds[i].CreationTimestamp.After(builds[j].CreationTimestamp.Time)
	})

	// Delete old builds (keep the most recent keepCount builds)
	buildsToDelete := builds[keepCount:]
	deletedCount := 0

	for i := range buildsToDelete {
		build := &buildsToDelete[i]
		// Don't delete running builds
		if build.Status.Phase == buildv1.BuildPhaseRunning {
			logger.Info("Skipping running build", "buildName", build.Name)
			continue
		}

		if err := s.client.Delete(ctx, build); err != nil {
			if !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete old build", "buildName", build.Name)
				continue
			}
		}
		deletedCount++
		logger.V(1).Info("Deleted old build", "buildName", build.Name)
	}

	logger.Info("Cleaned up old builds", "buildConfigName", buildConfigName, "deletedCount", deletedCount)
	return nil
}

// GetBuildLogs returns the build logs
func (s *S2IStrategy) GetBuildLogs(ctx context.Context, buildName string) (string, error) {
	// TODO: Implement log streaming from OpenShift build
	// This requires using the OpenShift REST API or build client
	return "", fmt.Errorf("log streaming not yet implemented")
}

// DeleteBuild cleans up build resources
func (s *S2IStrategy) DeleteBuild(ctx context.Context, buildName string) error {
	// List all builds with this name
	buildList := &buildv1.BuildList{}
	if err := s.client.List(ctx, buildList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return fmt.Errorf("failed to list builds: %w", err)
	}

	// Find and delete the build with matching name
	for i := range buildList.Items {
		if buildList.Items[i].Name == buildName {
			if err := s.client.Delete(ctx, &buildList.Items[i]); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete build: %w", err)
			}
			break
		}
	}

	// List and delete BuildConfigs
	buildConfigList := &buildv1.BuildConfigList{}
	if err := s.client.List(ctx, buildConfigList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		return fmt.Errorf("failed to list buildconfigs: %w", err)
	}

	for i := range buildConfigList.Items {
		if buildConfigList.Items[i].Name == buildName {
			if err := s.client.Delete(ctx, &buildConfigList.Items[i]); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete buildconfig: %w", err)
			}
			break
		}
	}

	return nil
}

// ValidateConfig validates the S2I build configuration
func (s *S2IStrategy) ValidateConfig(config *mlopsv1alpha1.BuildConfigSpec) error {
	// BaseImage is optional - we have a default
	// No specific validation needed for S2I
	return nil
}
