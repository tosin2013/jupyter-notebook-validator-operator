package build

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	securityv1 "github.com/openshift/api/security/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// S2IStrategyName is the name of the S2I build strategy
	S2IStrategyName = "s2i"
)

// S2IStrategy implements the Strategy interface for OpenShift Source-to-Image builds
type S2IStrategy struct {
	client    client.Client
	apiReader client.Reader // Non-cached client for SCC Gets
	scheme    *runtime.Scheme
}

// NewS2IStrategy creates a new S2I build strategy
func NewS2IStrategy(client client.Client, apiReader client.Reader, scheme *runtime.Scheme) *S2IStrategy {
	return &S2IStrategy{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
	}
}

// Name returns the strategy name
func (s *S2IStrategy) Name() string {
	return S2IStrategyName
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

// ensureBuildServiceAccount ensures that a ServiceAccount exists for S2I builds
// ADR-039 (adapted for S2I): Automatic SCC Management for S2I Builds
func (s *S2IStrategy) ensureBuildServiceAccount(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// Step 1: Ensure builder ServiceAccount exists
	sa := &corev1.ServiceAccount{}
	err := s.client.Get(ctx, client.ObjectKey{
		Name:      "builder",
		Namespace: namespace,
	}, sa)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check builder ServiceAccount: %w", err)
	}

	if errors.IsNotFound(err) {
		// ServiceAccount doesn't exist, create it
		logger.Info("Creating builder ServiceAccount", "namespace", namespace)
		newSA := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "builder",
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "jupyter-notebook-validator-operator",
					"app.kubernetes.io/component":  "s2i-build",
				},
			},
		}

		if err := s.client.Create(ctx, newSA); err != nil {
			return fmt.Errorf("failed to create builder ServiceAccount: %w", err)
		}
		logger.Info("Successfully created builder ServiceAccount", "namespace", namespace)
	} else {
		logger.V(1).Info("builder ServiceAccount already exists", "namespace", namespace)
	}

	// Step 2: Automatically grant pipelines-scc to the ServiceAccount for builds
	// ADR-039 (adapted): Operator should automatically configure SCC for S2I builds
	// SECURITY: Use pipelines-scc (not anyuid) for better security posture
	// pipelines-scc is more restrictive (SETFCAP only) while still supporting builds
	if err := s.grantSCCToServiceAccount(ctx, namespace, "builder", "pipelines-scc"); err != nil {
		// Log warning but don't fail - this might be a Kubernetes cluster without SCCs
		logger.Info("Failed to grant SCC (might be Kubernetes without OpenShift SCCs)",
			"error", err,
			"namespace", namespace,
			"serviceAccount", "builder",
			"scc", "pipelines-scc")
		logger.Info("If on OpenShift, manually grant SCC: oc adm policy add-scc-to-user pipelines-scc -z builder -n " + namespace)
	}

	return nil
}

// grantSCCToServiceAccount grants a SecurityContextConstraint to a ServiceAccount
// This automates the manual "oc adm policy add-scc-to-user" command
func (s *S2IStrategy) grantSCCToServiceAccount(ctx context.Context, namespace, serviceAccount, sccName string) error {
	logger := log.FromContext(ctx)

	// Get the SCC using APIReader (non-cached) to avoid triggering watch/list attempts
	// Since we only need to Get specific SCCs by name, we don't need caching
	scc := &securityv1.SecurityContextConstraints{}
	err := s.apiReader.Get(ctx, client.ObjectKey{Name: sccName}, scc)
	if err != nil {
		if errors.IsNotFound(err) {
			// SCC doesn't exist - likely Kubernetes without OpenShift
			return fmt.Errorf("SCC %s not found (Kubernetes cluster?): %w", sccName, err)
		}
		return fmt.Errorf("failed to get SCC %s: %w", sccName, err)
	}

	// Check if ServiceAccount already has the SCC
	serviceAccountUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceAccount)
	for _, user := range scc.Users {
		if user == serviceAccountUser {
			logger.V(1).Info("ServiceAccount already has SCC",
				"namespace", namespace,
				"serviceAccount", serviceAccount,
				"scc", sccName)
			return nil
		}
	}

	// Add ServiceAccount to SCC users
	logger.Info("Granting SCC to ServiceAccount",
		"namespace", namespace,
		"serviceAccount", serviceAccount,
		"scc", sccName)

	scc.Users = append(scc.Users, serviceAccountUser)

	if err := s.client.Update(ctx, scc); err != nil {
		return fmt.Errorf("failed to update SCC %s: %w", sccName, err)
	}

	logger.Info("Successfully granted SCC to ServiceAccount",
		"namespace", namespace,
		"serviceAccount", serviceAccount,
		"scc", sccName)

	return nil
}

// CreateBuild creates an S2I build for the notebook
// ADR-038: Supports requirements.txt auto-detection with Docker build strategy
// ADR-039 (adapted): Automatic SCC management for S2I builds
// ADR-030 (adapted): Retry logic with exponential backoff for resource verification
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

	// ADR-039 (adapted): Ensure ServiceAccount exists and has required SCCs
	if err := s.ensureBuildServiceAccount(ctx, job.Namespace); err != nil {
		// Log warning but don't fail - SCC management might not be available on Kubernetes
		logger.Info("Failed to ensure build ServiceAccount (continuing anyway)",
			"error", err,
			"namespace", job.Namespace)
	}

	// ADR-038: Determine build strategy based on requirements.txt detection
	// For S2I, we generate an inline Dockerfile when AutoGenerateRequirements is true
	var inlineDockerfile string
	var buildStrategyType buildv1.BuildStrategyType
	var sourceStrategy *buildv1.SourceBuildStrategy
	var dockerStrategy *buildv1.DockerBuildStrategy

	if buildConfig.AutoGenerateRequirements && !buildConfig.PreferDockerfile {
		// Use Docker build strategy with generated Dockerfile
		// Note: We generate a simple Dockerfile that handles requirements.txt at build time
		buildStrategyType = buildv1.DockerBuildStrategyType
		inlineDockerfile = generateInlineDockerfile(job, baseImage)
		dockerStrategy = &buildv1.DockerBuildStrategy{}
		logger.Info("Using Docker build strategy with auto-generated Dockerfile",
			"buildName", buildName,
			"autoGenerate", buildConfig.AutoGenerateRequirements)
	} else {
		// Use traditional S2I source build strategy
		buildStrategyType = buildv1.SourceBuildStrategyType
		sourceStrategy = &buildv1.SourceBuildStrategy{
			From: corev1.ObjectReference{
				Kind: "DockerImage",
				Name: baseImage,
			},
		}
		logger.Info("Using S2I source build strategy",
			"buildName", buildName,
			"baseImage", baseImage)
	}

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

	// Create BuildConfig with ConfigChange trigger for automatic build start
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

					// ADR-038: Add inline Dockerfile if using Docker strategy
					if buildStrategyType == buildv1.DockerBuildStrategyType && inlineDockerfile != "" {
						source.Dockerfile = &inlineDockerfile
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
					Type:           buildStrategyType,
					SourceStrategy: sourceStrategy,
					DockerStrategy: dockerStrategy,
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: fmt.Sprintf("%s:latest", buildName),
					},
				},
			},
			// Add ConfigChange trigger to automatically start build on creation
			Triggers: []buildv1.BuildTriggerPolicy{
				{
					Type: buildv1.ConfigChangeBuildTriggerType,
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
		logger.Info("BuildConfig already exists, fetching existing", "buildConfigName", buildName)
		existingBC := &buildv1.BuildConfig{}
		if err := s.client.Get(ctx, client.ObjectKey{Name: buildName, Namespace: job.Namespace}, existingBC); err != nil {
			return nil, fmt.Errorf("failed to get existing BuildConfig: %w", err)
		}
		bc = existingBC
	} else {
		logger.Info("BuildConfig created successfully", "buildConfigName", buildName)
	}

	// ADR-030 (adapted): Verify BuildConfig was actually created with retry
	// Kubernetes API may take a moment to reflect the created resource
	verifyBC := &buildv1.BuildConfig{}
	maxRetries := 5
	retryDelay := 100 * time.Millisecond
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
		}

		lastErr = s.client.Get(ctx, client.ObjectKey{Name: buildName, Namespace: job.Namespace}, verifyBC)
		if lastErr == nil {
			logger.Info("BuildConfig verified successfully", "buildConfig", buildName, "namespace", job.Namespace, "attempts", attempt+1)
			break
		}

		if !errors.IsNotFound(lastErr) {
			// Non-NotFound error, fail immediately
			return nil, fmt.Errorf("buildconfig creation verification failed: %w", lastErr)
		}

		logger.V(1).Info("BuildConfig not found yet, retrying", "attempt", attempt+1, "maxRetries", maxRetries)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("buildconfig creation verification failed after %d retries: %w", maxRetries, lastErr)
	}

	// BuildConfig created with ConfigChange trigger - build will start automatically
	// The controller will use smart discovery (GetLatestBuild) to find the triggered build

	logger.Info("BuildConfig created with ConfigChange trigger - build will start automatically",
		"buildConfigName", buildName)

	now := time.Now()
	return &BuildInfo{
		Name:      buildName, // Return BuildConfig name, controller will find actual builds
		Status:    BuildStatusPending,
		Message:   "BuildConfig created with auto-trigger - build starting",
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
			// Get full image reference from ImageStream instead of just digest
			info.ImageReference = s.getFullImageReference(build)
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

// getFullImageReference constructs the full image reference from Build and ImageStream
// Returns: image-registry.openshift-image-registry.svc:5000/namespace/imagestream@sha256:...
func (s *S2IStrategy) getFullImageReference(build *buildv1.Build) string {
	// If no digest, return empty
	if build.Status.Output.To == nil || build.Status.Output.To.ImageDigest == "" {
		return ""
	}

	digest := build.Status.Output.To.ImageDigest

	// Get the BuildConfig name from build labels
	buildConfigName := build.Labels["buildconfig"]
	if buildConfigName == "" {
		// Fallback: just return the digest
		return digest
	}

	// Get the BuildConfig to find the output ImageStreamTag
	bc := &buildv1.BuildConfig{}
	if err := s.client.Get(context.Background(), client.ObjectKey{
		Name:      buildConfigName,
		Namespace: build.Namespace,
	}, bc); err != nil {
		// Fallback: just return the digest
		return digest
	}

	// Get the ImageStream name from BuildConfig output
	if bc.Spec.Output.To == nil || bc.Spec.Output.To.Name == "" {
		return digest
	}

	// Parse ImageStreamTag name (format: "imagestream:tag")
	imageStreamName := bc.Spec.Output.To.Name
	if idx := strings.Index(imageStreamName, ":"); idx != -1 {
		imageStreamName = imageStreamName[:idx]
	}

	// Get the ImageStream to find the docker image repository
	is := &imagev1.ImageStream{}
	if err := s.client.Get(context.Background(), client.ObjectKey{
		Name:      imageStreamName,
		Namespace: build.Namespace,
	}, is); err != nil {
		// Fallback: just return the digest
		return digest
	}

	// Get the docker image repository
	dockerRepo := is.Status.DockerImageRepository
	if dockerRepo == "" {
		return digest
	}

	// Construct full image reference: registry/namespace/imagestream@digest
	return fmt.Sprintf("%s@%s", dockerRepo, digest)
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

// generateInlineDockerfile generates an inline Dockerfile for S2I Docker build strategy
// ADR-038: Used when AutoGenerateRequirements is true
// This creates a simple Dockerfile that handles requirements.txt detection at build time
func generateInlineDockerfile(job *mlopsv1alpha1.NotebookValidationJob, baseImage string) string {
	// Determine potential requirements.txt locations based on notebook path
	notebookPath := job.Spec.Notebook.Path
	notebookDir := strings.TrimSuffix(notebookPath, "/"+strings.Split(notebookPath, "/")[len(strings.Split(notebookPath, "/"))-1])

	// ADR-038 fallback chain:
	// 1. Notebook directory: notebooks/02-anomaly-detection/requirements.txt
	// 2. Tier directory: notebooks/requirements.txt
	// 3. Repository root: requirements.txt

	dockerfile := fmt.Sprintf(`FROM %s

# ADR-038: Auto-generated Dockerfile with requirements.txt fallback chain
# This Dockerfile checks for requirements.txt in multiple locations and installs dependencies

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Copy project files
COPY . /workspace
WORKDIR /workspace

# ADR-038 Fallback chain: Install dependencies from requirements.txt if found
# 1. Try notebook-specific requirements.txt
# 2. Fall back to tier-level requirements.txt
# 3. Fall back to repository root requirements.txt
RUN if [ -f "%s/requirements.txt" ]; then \
        echo "Installing from notebook directory: %s/requirements.txt"; \
        pip install --no-cache-dir -r %s/requirements.txt; \
    elif [ -f "notebooks/requirements.txt" ]; then \
        echo "Installing from tier directory: notebooks/requirements.txt"; \
        pip install --no-cache-dir -r notebooks/requirements.txt; \
    elif [ -f "requirements.txt" ]; then \
        echo "Installing from repository root: requirements.txt"; \
        pip install --no-cache-dir -r requirements.txt; \
    else \
        echo "No requirements.txt found, using base image dependencies only"; \
    fi

# Health check
RUN python -c "import sys; print(f'Python {sys.version}')" && \
    python -c "import papermill; print(f'Papermill {papermill.__version__}')"
`, baseImage, notebookDir, notebookDir, notebookDir)

	return dockerfile
}
