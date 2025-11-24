package build

import (
	"context"
	"fmt"
	"strings"
	"time"

	securityv1 "github.com/openshift/api/security/v1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// TektonConditionSucceeded is the Tekton condition type for successful completion
	TektonConditionSucceeded = "Succeeded"
	// TektonStrategyName is the name of the Tekton build strategy
	TektonStrategyName = "tekton"
)

// TektonStrategy implements the Strategy interface for Tekton Pipelines
type TektonStrategy struct {
	client    client.Client
	apiReader client.Reader // Non-cached client for SCC Gets
	scheme    *runtime.Scheme
}

// NewTektonStrategy creates a new Tekton build strategy
func NewTektonStrategy(client client.Client, apiReader client.Reader, scheme *runtime.Scheme) *TektonStrategy {
	return &TektonStrategy{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
	}
}

// Name returns the strategy name
func (t *TektonStrategy) Name() string {
	return TektonStrategyName
}

// Detect checks if Tekton is available in the cluster
func (t *TektonStrategy) Detect(ctx context.Context, client client.Client) (bool, error) {
	logger := log.FromContext(ctx)

	// Check if TaskRun CRD exists by trying to list TaskRuns
	// This is more reliable than trying to Get a specific resource
	taskRunList := &tektonv1.TaskRunList{}
	err := client.List(ctx, taskRunList)

	if err != nil {
		logger.V(1).Info("Tekton detection: error listing TaskRuns",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"isNotFound", errors.IsNotFound(err),
			"isNotRegistered", runtime.IsNotRegisteredError(err))

		// Check if it's a "no kind match" error (CRD doesn't exist)
		if runtime.IsNotRegisteredError(err) {
			logger.Info("Tekton not available: TaskRun CRD not registered")
			return false, nil
		}

		// Check for "no matches for kind" error (API not available)
		if strings.Contains(err.Error(), "no matches for kind") {
			logger.Info("Tekton not available: TaskRun API not found")
			return false, nil
		}

		// Other errors might indicate permission issues
		logger.Error(err, "Tekton detection failed with unexpected error")
		return false, err
	}

	logger.Info("Tekton available: TaskRun API detected", "taskRunCount", len(taskRunList.Items))
	return true, nil
}

// ensureBuildPVC creates a unique PVC for the build if it doesn't exist
// ADR-040: Use unique PVC per build to avoid workspace contention with ReadWriteOnce
func (t *TektonStrategy) ensureBuildPVC(ctx context.Context, namespace, pvcName string) error {
	logger := log.FromContext(ctx)

	// Check if PVC already exists
	existingPVC := &corev1.PersistentVolumeClaim{}
	err := t.client.Get(ctx, client.ObjectKey{
		Name:      pvcName,
		Namespace: namespace,
	}, existingPVC)

	if err == nil {
		// PVC already exists
		logger.V(1).Info("Build PVC already exists", "pvc", pvcName, "namespace", namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check build PVC: %w", err)
	}

	// Create PVC
	logger.Info("Creating build PVC", "pvc", pvcName, "namespace", namespace)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":     "jupyter-notebook-validator-operator",
				"app.kubernetes.io/component":      "tekton-build",
				"mlops.redhat.com/build-workspace": "true",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce, // RWO is sufficient since each build has its own PVC
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	if err := t.client.Create(ctx, pvc); err != nil {
		return fmt.Errorf("failed to create build PVC: %w", err)
	}

	logger.Info("Successfully created build PVC", "pvc", pvcName, "namespace", namespace)
	return nil
}

// ensureTasksInNamespace copies required Tekton Tasks from openshift-pipelines namespace to the target namespace
// This implements ADR-028: Copy Tasks to user namespace for RBAC simplicity and isolation
func (t *TektonStrategy) ensureTasksInNamespace(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// List of Tasks to copy from openshift-pipelines namespace
	requiredTasks := []string{
		"git-clone",
		"buildah",
	}

	sourceNamespace := "openshift-pipelines"

	for _, taskName := range requiredTasks {
		// Check if Task already exists in target namespace
		existingTask := &tektonv1.Task{}
		err := t.client.Get(ctx, client.ObjectKey{
			Name:      taskName,
			Namespace: namespace,
		}, existingTask)

		if err == nil {
			// Task exists, check if it's managed by us
			if existingTask.Labels["app.kubernetes.io/managed-by"] == "jupyter-notebook-validator-operator" {
				logger.V(1).Info("Task already exists and is managed by operator", "task", taskName, "namespace", namespace)
				// TODO: Check version and update if needed (Phase 3 of ADR-028)
			} else {
				logger.Info("Task exists but not managed by operator, skipping", "task", taskName, "namespace", namespace)
			}
			continue
		}

		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check task %s: %w", taskName, err)
		}

		// Task doesn't exist, copy it from openshift-pipelines namespace
		logger.Info("Copying Task from openshift-pipelines to user namespace", "task", taskName, "from", sourceNamespace, "to", namespace)

		// Get the source Task
		sourceTask := &tektonv1.Task{}
		if err := t.client.Get(ctx, client.ObjectKey{
			Name:      taskName,
			Namespace: sourceNamespace,
		}, sourceTask); err != nil {
			return fmt.Errorf("failed to get source task %s from %s: %w", taskName, sourceNamespace, err)
		}

		// Create a copy in the target namespace
		targetTask := &tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by":  "jupyter-notebook-validator-operator",
					"mlops.redhat.com/task-type":    taskName,
					"mlops.redhat.com/task-version": "1.0.0", // Version for future updates
					"mlops.redhat.com/copied-from":  sourceNamespace,
				},
				Annotations: map[string]string{
					"mlops.redhat.com/source-namespace": sourceNamespace,
					"mlops.redhat.com/source-task":      taskName,
					"mlops.redhat.com/copied-at":        time.Now().Format(time.RFC3339),
				},
			},
			Spec: sourceTask.Spec,
		}

		if err := t.client.Create(ctx, targetTask); err != nil {
			return fmt.Errorf("failed to create task %s in namespace %s: %w", taskName, namespace, err)
		}

		logger.Info("Successfully copied Task to user namespace", "task", taskName, "namespace", namespace)
	}

	return nil
}

// ensurePipelineServiceAccount ensures the pipeline ServiceAccount exists in the namespace
// The buildah task requires privileged access, so we automatically grant pipelines-scc
// This implements ADR-039: Automatic SCC Management for Tekton Builds
func (t *TektonStrategy) ensurePipelineServiceAccount(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx)

	// Step 1: Ensure pipeline ServiceAccount exists
	sa := &corev1.ServiceAccount{}
	err := t.client.Get(ctx, client.ObjectKey{
		Name:      "pipeline",
		Namespace: namespace,
	}, sa)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check pipeline ServiceAccount: %w", err)
	}

	if errors.IsNotFound(err) {
		// ServiceAccount doesn't exist, create it
		logger.Info("Creating pipeline ServiceAccount", "namespace", namespace)
		newSA := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pipeline",
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "jupyter-notebook-validator-operator",
					"app.kubernetes.io/component":  "tekton-build",
				},
			},
		}

		if err := t.client.Create(ctx, newSA); err != nil {
			return fmt.Errorf("failed to create pipeline ServiceAccount: %w", err)
		}
		logger.Info("Successfully created pipeline ServiceAccount", "namespace", namespace)
	} else {
		logger.V(1).Info("pipeline ServiceAccount already exists", "namespace", namespace)
	}

	// Step 2: Automatically grant pipelines-scc to the ServiceAccount
	// ADR-039: Operator should automatically configure SCC for builds
	if err := t.grantSCCToServiceAccount(ctx, namespace, "pipeline", "pipelines-scc"); err != nil {
		// Log warning but don't fail - this might be a Kubernetes cluster without SCCs
		logger.Info("Failed to grant SCC (might be Kubernetes without OpenShift SCCs)",
			"error", err,
			"namespace", namespace,
			"serviceAccount", "pipeline",
			"scc", "pipelines-scc")
		logger.Info("If on OpenShift, manually grant SCC: oc adm policy add-scc-to-user pipelines-scc -z pipeline -n " + namespace)
	}

	return nil
}

// grantSCCToServiceAccount grants a SecurityContextConstraint to a ServiceAccount
// This automates the manual "oc adm policy add-scc-to-user" command
func (t *TektonStrategy) grantSCCToServiceAccount(ctx context.Context, namespace, serviceAccount, sccName string) error {
	logger := log.FromContext(ctx)

	// Get the SCC using APIReader (non-cached) to avoid triggering watch/list attempts
	// Since we only need to Get specific SCCs by name, we don't need caching
	scc := &securityv1.SecurityContextConstraints{}
	err := t.apiReader.Get(ctx, client.ObjectKey{Name: sccName}, scc)
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

	if err := t.client.Update(ctx, scc); err != nil {
		return fmt.Errorf("failed to update SCC %s: %w", sccName, err)
	}

	logger.Info("Successfully granted SCC to ServiceAccount",
		"namespace", namespace,
		"serviceAccount", serviceAccount,
		"scc", sccName)

	return nil
}

// ensureTektonGitCredentials creates a Tekton-formatted Git credentials secret from a standard secret
// Tekton git-clone task expects .git-credentials and .gitconfig files for HTTPS authentication
// This converts the standard username/password format to Tekton's basic-auth workspace format
func (t *TektonStrategy) ensureTektonGitCredentials(ctx context.Context, namespace, sourceSecretName string) error {
	logger := log.FromContext(ctx)

	tektonSecretName := sourceSecretName + "-tekton"

	// Check if Tekton secret already exists
	existingSecret := &corev1.Secret{}
	err := t.client.Get(ctx, client.ObjectKey{
		Name:      tektonSecretName,
		Namespace: namespace,
	}, existingSecret)

	if err == nil {
		// Secret exists, check if it's managed by us
		if existingSecret.Labels["app.kubernetes.io/managed-by"] == "jupyter-notebook-validator-operator" {
			logger.V(1).Info("Tekton Git credentials secret already exists", "secret", tektonSecretName, "namespace", namespace)
			// TODO: Check if source secret has changed and update if needed
		} else {
			logger.Info("Tekton Git credentials secret exists but not managed by operator, skipping", "secret", tektonSecretName, "namespace", namespace)
		}
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check Tekton git credentials secret %s: %w", tektonSecretName, err)
	}

	// Tekton secret doesn't exist, create it from source secret
	logger.Info("Creating Tekton Git credentials secret from source secret", "source", sourceSecretName, "target", tektonSecretName, "namespace", namespace)

	// Get the source secret
	sourceSecret := &corev1.Secret{}
	if err := t.client.Get(ctx, client.ObjectKey{
		Name:      sourceSecretName,
		Namespace: namespace,
	}, sourceSecret); err != nil {
		return fmt.Errorf("failed to get source git credentials secret %s: %w", sourceSecretName, err)
	}

	// Extract username and password from source secret
	username, usernameExists := sourceSecret.Data["username"]
	if !usernameExists {
		return fmt.Errorf("source secret %s does not contain 'username' key", sourceSecretName)
	}

	var password []byte
	var passwordExists bool
	if password, passwordExists = sourceSecret.Data["password"]; !passwordExists {
		// Try 'token' as fallback
		if password, passwordExists = sourceSecret.Data["token"]; !passwordExists {
			return fmt.Errorf("source secret %s does not contain 'password' or 'token' key", sourceSecretName)
		}
	}

	// Create Tekton-formatted secret with .git-credentials and .gitconfig
	// Format documented at: https://tekton.dev/docs/pipelines/auth/

	// .git-credentials format: https://<username>:<password>@<hostname>
	// We'll use a generic format that works with most Git hosting providers
	gitCredentials := fmt.Sprintf("https://%s:%s@github.com\n", string(username), string(password))

	// .gitconfig to use the credentials helper
	gitConfig := `[credential]
	helper = store
`

	tektonSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonSecretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":   "jupyter-notebook-validator-operator",
				"app.kubernetes.io/component":    "tekton-build",
				"mlops.redhat.com/secret-type":   "git-credentials",
				"mlops.redhat.com/source-secret": sourceSecretName,
			},
			Annotations: map[string]string{
				"mlops.redhat.com/description":   "Tekton-formatted Git credentials converted from standard secret",
				"mlops.redhat.com/source-secret": sourceSecretName,
				"mlops.redhat.com/created-at":    time.Now().Format(time.RFC3339),
				"tekton.dev/git-0":               "https://github.com", // Tekton annotation for Git credentials
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			".git-credentials": []byte(gitCredentials),
			".gitconfig":       []byte(gitConfig),
		},
	}

	if err := t.client.Create(ctx, tektonSecret); err != nil {
		return fmt.Errorf("failed to create Tekton git credentials secret %s: %w", tektonSecretName, err)
	}

	logger.Info("Successfully created Tekton Git credentials secret", "secret", tektonSecretName, "namespace", namespace)
	return nil
}

// CreateBuild creates a Tekton TaskRun for building the notebook image
func (t *TektonStrategy) CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error) {
	logger := log.FromContext(ctx)

	// Check if BuildConfig is provided
	if job.Spec.PodConfig.BuildConfig == nil {
		return nil, fmt.Errorf("buildConfig is required")
	}

	// ADR-028: Ensure required Tasks exist in user's namespace before creating Pipeline
	logger.Info("Ensuring Tekton Tasks exist in namespace", "namespace", job.Namespace)
	if err := t.ensureTasksInNamespace(ctx, job.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure tasks in namespace: %w", err)
	}

	// Ensure pipeline ServiceAccount exists with proper SCC for buildah
	logger.Info("Ensuring pipeline ServiceAccount exists", "namespace", job.Namespace)
	if err := t.ensurePipelineServiceAccount(ctx, job.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure pipeline ServiceAccount: %w", err)
	}

	// Ensure Tekton-formatted Git credentials secret exists if credentials are specified
	if job.Spec.Notebook.Git.CredentialsSecret != "" {
		logger.Info("Ensuring Tekton-formatted Git credentials secret",
			"namespace", job.Namespace,
			"sourceSecret", job.Spec.Notebook.Git.CredentialsSecret)
		if err := t.ensureTektonGitCredentials(ctx, job.Namespace, job.Spec.Notebook.Git.CredentialsSecret); err != nil {
			return nil, fmt.Errorf("failed to ensure Tekton git credentials: %w", err)
		}
	}

	buildConfig := job.Spec.PodConfig.BuildConfig
	buildName := fmt.Sprintf("%s-build", job.Name)

	// ADR-040: Create unique PVC per build to avoid workspace contention
	// This allows concurrent builds without ReadWriteOnce limitations
	pvcName := fmt.Sprintf("%s-workspace", buildName)
	logger.Info("Creating unique PVC for build", "pvc", pvcName, "namespace", job.Namespace)
	if err := t.ensureBuildPVC(ctx, job.Namespace, pvcName); err != nil {
		return nil, fmt.Errorf("failed to ensure build PVC: %w", err)
	}

	// Get registry configuration from strategyConfig or use defaults
	registry := "image-registry.openshift-image-registry.svc:5000"
	if val, ok := buildConfig.StrategyConfig["registry"]; ok {
		registry = val
	}

	imageRef := fmt.Sprintf("%s/%s/%s:latest", registry, job.Namespace, buildName)

	// Create a Pipeline with git-clone + buildah tasks
	pipeline := t.createBuildPipeline(job, buildConfig)
	if err := t.client.Create(ctx, pipeline); err != nil {
		if !errors.IsAlreadyExists(err) {
			// ADR-030 Phase 1: Return error instead of continuing silently
			return nil, fmt.Errorf("failed to create pipeline: %w", err)
		}
		// Pipeline already exists, fetch it to ensure we have the latest version
		logger.V(1).Info("Pipeline already exists, fetching existing", "pipeline", pipeline.Name)
		existingPipeline := &tektonv1.Pipeline{}
		if err := t.client.Get(ctx, client.ObjectKey{Name: pipeline.Name, Namespace: job.Namespace}, existingPipeline); err != nil {
			return nil, fmt.Errorf("failed to get existing pipeline: %w", err)
		}
		pipeline = existingPipeline
	}

	// ADR-030 Phase 1: Verify Pipeline was actually created with retry
	// Kubernetes API may take a moment to reflect the created resource
	verifyPipeline := &tektonv1.Pipeline{}
	maxRetries := 5
	retryDelay := 100 * time.Millisecond
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
		}

		lastErr = t.client.Get(ctx, client.ObjectKey{Name: pipeline.Name, Namespace: job.Namespace}, verifyPipeline)
		if lastErr == nil {
			logger.Info("Pipeline verified successfully", "pipeline", pipeline.Name, "namespace", job.Namespace, "attempts", attempt+1)
			break
		}

		if !errors.IsNotFound(lastErr) {
			// Non-NotFound error, fail immediately
			return nil, fmt.Errorf("pipeline creation verification failed: %w", lastErr)
		}

		logger.V(1).Info("Pipeline not found yet, retrying", "attempt", attempt+1, "maxRetries", maxRetries)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("pipeline creation verification failed after %d retries: %w", maxRetries, lastErr)
	}

	// Create PipelineRun
	pipelineRun := t.createPipelineRun(job, buildName, pipeline.Name, pvcName)
	if err := t.client.Create(ctx, pipelineRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			// ADR-030 Phase 1: Return error instead of continuing silently
			return nil, fmt.Errorf("failed to create pipelinerun: %w", err)
		}
		// PipelineRun already exists, fetch it
		logger.V(1).Info("PipelineRun already exists, fetching existing", "pipelineRun", pipelineRun.Name)
		existingPipelineRun := &tektonv1.PipelineRun{}
		if err := t.client.Get(ctx, client.ObjectKey{Name: pipelineRun.Name, Namespace: job.Namespace}, existingPipelineRun); err != nil {
			return nil, fmt.Errorf("failed to get existing pipelinerun: %w", err)
		}
		pipelineRun = existingPipelineRun
	}

	// ADR-030 Phase 1: Verify PipelineRun was actually created with retry
	// Kubernetes API may take a moment to reflect the created resource
	verifyPipelineRun := &tektonv1.PipelineRun{}
	prMaxRetries := 5
	prRetryDelay := 100 * time.Millisecond
	var prLastErr error

	for attempt := 0; attempt < prMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(prRetryDelay)
			prRetryDelay *= 2 // Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
		}

		prLastErr = t.client.Get(ctx, client.ObjectKey{Name: pipelineRun.Name, Namespace: job.Namespace}, verifyPipelineRun)
		if prLastErr == nil {
			logger.Info("PipelineRun verified successfully", "pipelineRun", pipelineRun.Name, "namespace", job.Namespace, "attempts", attempt+1)
			break
		}

		if !errors.IsNotFound(prLastErr) {
			// Non-NotFound error, fail immediately
			return nil, fmt.Errorf("pipelinerun creation verification failed: %w", prLastErr)
		}

		logger.V(1).Info("PipelineRun not found yet, retrying", "attempt", attempt+1, "maxRetries", prMaxRetries)
	}

	if prLastErr != nil {
		return nil, fmt.Errorf("pipelinerun creation verification failed after %d retries: %w", prMaxRetries, prLastErr)
	}

	// ADR-030 Phase 1: Only report success after verification
	now := time.Now()
	logger.Info("Build created successfully", "buildName", buildName, "pipeline", pipeline.Name, "pipelineRun", pipelineRun.Name)
	return &BuildInfo{
		Name:           pipelineRun.Name,
		Status:         BuildStatusPending,
		Message:        "Tekton pipeline created and triggered",
		ImageReference: imageRef,
		StartTime:      &now,
	}, nil
}

// createBuildPipeline creates a Tekton Pipeline with git-clone and buildah tasks
func (t *TektonStrategy) createBuildPipeline(job *mlopsv1alpha1.NotebookValidationJob, buildConfig *mlopsv1alpha1.BuildConfigSpec) *tektonv1.Pipeline {
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
				// ADR-031 Phase 2: Add custom Dockerfile path parameter
				{Name: "dockerfile-path", Type: tektonv1.ParamTypeString, Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
				// ADR-038: Add notebook path for requirements.txt fallback chain detection
				{Name: "notebook-path", Type: tektonv1.ParamTypeString, Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
			},
			Workspaces: []tektonv1.PipelineWorkspaceDeclaration{
				{Name: "shared-workspace"},
				{Name: "git-credentials", Optional: true},
			},
			Tasks: []tektonv1.PipelineTask{
				{
					Name: "fetch-repository",
					TaskRef: &tektonv1.TaskRef{
						Name: "git-clone",
						// ADR-028: Use Task (not ClusterTask) - Tasks will be copied to user namespace
						// For now, reference Tasks in same namespace (will be copied by ensureTasksInNamespace)
						Kind: tektonv1.NamespacedTaskKind,
					},
					Params: []tektonv1.Param{
						// ADR-030: Use uppercase param names to match OpenShift Pipelines git-clone Task
						{Name: "URL", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-url)"}},
						{Name: "REVISION", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-revision)"}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: "shared-workspace"},
						// ADR-031: Use basic-auth workspace for HTTPS authentication
						// git-clone Task expects .gitconfig and .git-credentials files for HTTPS
						{Name: "basic-auth", Workspace: "git-credentials"},
					},
				},
				{
					Name: "generate-dockerfile",
					// ADR-031: Inline Task to generate Dockerfile from baseImage if not present
					// User insight: "Do we need to create a custom tekton task? and use buildah when someone has a docker file"
					// Answer: NO! Use inline Task with script, then buildah for both scenarios
					// Phase 2: Support custom Dockerfile path from CRD
					TaskSpec: &tektonv1.EmbeddedTask{
						TaskSpec: tektonv1.TaskSpec{
							Params: []tektonv1.ParamSpec{
								{Name: "BASE_IMAGE", Type: tektonv1.ParamTypeString},
								{Name: "DOCKERFILE_PATH", Type: tektonv1.ParamTypeString},
								{Name: "NOTEBOOK_PATH", Type: tektonv1.ParamTypeString, Default: &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
							},
							Workspaces: []tektonv1.WorkspaceDeclaration{
								{Name: "source"},
							},
							Steps: []tektonv1.Step{
								{
									Name:  "check-and-generate-dockerfile",
									Image: "registry.access.redhat.com/ubi9/ubi-minimal:latest",
									// Fix: Run as non-root user to comply with OpenShift restricted-v2 SCC
									SecurityContext: &corev1.SecurityContext{
										RunAsNonRoot: func() *bool { b := true; return &b }(),
										RunAsUser:    func() *int64 { uid := int64(65532); return &uid }(), // Standard non-root user
									},
									Script: `#!/bin/sh
set -e

# ADR-031 Phase 2: Check for custom Dockerfile path first
if [ -n "$(params.DOCKERFILE_PATH)" ]; then
    CUSTOM_DOCKERFILE="$(workspaces.source.path)/$(params.DOCKERFILE_PATH)"
    if [ -f "$CUSTOM_DOCKERFILE" ]; then
        echo "✅ Custom Dockerfile found at: $(params.DOCKERFILE_PATH)"
        # Copy to standard location for buildah
        if [ "$(params.DOCKERFILE_PATH)" != "Dockerfile" ]; then
            cp "$CUSTOM_DOCKERFILE" "$(workspaces.source.path)/Dockerfile"
            echo "📋 Copied custom Dockerfile to ./Dockerfile"
        fi
        exit 0
    else
        echo "⚠️  Custom Dockerfile specified but not found: $(params.DOCKERFILE_PATH)"
        echo "⚠️  Falling back to auto-generation from baseImage"
    fi
fi

# Check if Dockerfile already exists in standard locations
if [ -f "$(workspaces.source.path)/Dockerfile" ] || [ -f "$(workspaces.source.path)/Containerfile" ]; then
    echo "✅ Dockerfile found in repository, using existing file"
    exit 0
fi

# Generate Dockerfile from baseImage
echo "📝 Generating Dockerfile from baseImage: $(params.BASE_IMAGE)"

# ADR-038: Requirements.txt fallback chain detection
# Try to find requirements.txt in order of specificity:
# 1. Notebook directory (most specific)
# 2. Tier directory (notebooks/)
# 3. Repository root (project-wide)

REQUIREMENTS_FILE=""
REQUIREMENTS_LOCATION=""

# Extract notebook directory from NOTEBOOK_PATH parameter (if available)
# Format: "notebooks/02-anomaly-detection/notebook.ipynb" → "notebooks/02-anomaly-detection"
NOTEBOOK_DIR=$(dirname "$(params.NOTEBOOK_PATH)" 2>/dev/null || echo "")

# 1. Try notebook-specific requirements.txt
if [ -n "$NOTEBOOK_DIR" ] && [ -f "$(workspaces.source.path)/$NOTEBOOK_DIR/requirements.txt" ]; then
    REQUIREMENTS_FILE="$NOTEBOOK_DIR/requirements.txt"
    REQUIREMENTS_LOCATION="notebook directory ($NOTEBOOK_DIR)"
# 2. Try tier-level requirements.txt
elif [ -f "$(workspaces.source.path)/notebooks/requirements.txt" ]; then
    REQUIREMENTS_FILE="notebooks/requirements.txt"
    REQUIREMENTS_LOCATION="tier directory (notebooks/)"
# 3. Try repository root requirements.txt
elif [ -f "$(workspaces.source.path)/requirements.txt" ]; then
    REQUIREMENTS_FILE="requirements.txt"
    REQUIREMENTS_LOCATION="repository root"
fi

# Generate Dockerfile based on whether requirements.txt was found
if [ -n "$REQUIREMENTS_FILE" ]; then
    echo "📦 Found requirements.txt in $REQUIREMENTS_LOCATION"
    cat > $(workspaces.source.path)/Dockerfile <<EOF
# Auto-generated by Jupyter Notebook Validator Operator
# ADR-038: Requirements.txt auto-detection with fallback chain
# Source: $REQUIREMENTS_LOCATION
FROM $(params.BASE_IMAGE)

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Install dependencies from requirements.txt
COPY $REQUIREMENTS_FILE /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

# Copy source code to /opt/app-root/src/ (S2I standard location)
# This matches S2I behavior so validation pod can find notebooks
COPY . /opt/app-root/src/

# Set working directory
WORKDIR /opt/app-root/src

# Health check
RUN python -c "import sys; print(f'Python {sys.version}')" && \
    python -c "import papermill; print(f'Papermill {papermill.__version__}')"
EOF
else
    echo "📦 No requirements.txt found in any location, using base image only"
    cat > $(workspaces.source.path)/Dockerfile <<EOF
# Auto-generated by Jupyter Notebook Validator Operator
# ADR-038: No requirements.txt found, using base image only
FROM $(params.BASE_IMAGE)

# Install notebook execution tools
RUN pip install --no-cache-dir papermill nbformat

# Copy source code to /opt/app-root/src/ (S2I standard location)
# This matches S2I behavior so validation pod can find notebooks
COPY . /opt/app-root/src/

# Set working directory
WORKDIR /opt/app-root/src
EOF
fi

echo "✅ Dockerfile generated successfully"
cat $(workspaces.source.path)/Dockerfile
`,
								},
							},
						},
					},
					RunAfter: []string{"fetch-repository"},
					Params: []tektonv1.Param{
						{Name: "BASE_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.base-image)"}},
						{Name: "DOCKERFILE_PATH", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.dockerfile-path)"}},
						{Name: "NOTEBOOK_PATH", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.notebook-path)"}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "source", Workspace: "shared-workspace"},
					},
				},
				{
					Name: "build-image",
					TaskRef: &tektonv1.TaskRef{
						Name: "buildah",
						// ADR-028: Use Task (not ClusterTask) - Tasks will be copied to user namespace
						Kind: tektonv1.NamespacedTaskKind,
					},
					RunAfter: []string{"generate-dockerfile"},
					Params: []tektonv1.Param{
						{Name: "IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.image-reference)"}},
						// ADR-031: Use DOCKERFILE parameter (buildah expects path to Dockerfile)
						{Name: "DOCKERFILE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "./Dockerfile"}},
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
func (t *TektonStrategy) createPipelineRun(job *mlopsv1alpha1.NotebookValidationJob, buildName, pipelineName, pvcName string) *tektonv1.PipelineRun {
	// Get base image (use default if not specified)
	baseImage := "quay.io/jupyter/minimal-notebook:latest"
	if job.Spec.PodConfig.BuildConfig != nil && job.Spec.PodConfig.BuildConfig.BaseImage != "" {
		baseImage = job.Spec.PodConfig.BuildConfig.BaseImage
	}

	// ADR-031 Phase 2: Get custom Dockerfile path if specified
	dockerfilePath := ""
	if job.Spec.PodConfig.BuildConfig != nil && job.Spec.PodConfig.BuildConfig.Dockerfile != "" {
		dockerfilePath = job.Spec.PodConfig.BuildConfig.Dockerfile
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
			// ADR-031: Use pipeline ServiceAccount which has pipelines-scc for buildah
			// Let OpenShift assign fsGroup automatically based on namespace UID range
			TaskRunTemplate: tektonv1.PipelineTaskRunTemplate{
				ServiceAccountName: "pipeline",
			},
			Params: []tektonv1.Param{
				{Name: "git-url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: job.Spec.Notebook.Git.URL}},
				{Name: "git-revision", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: job.Spec.Notebook.Git.Ref}},
				{Name: "image-reference", Value: tektonv1.ParamValue{
					Type:      tektonv1.ParamTypeString,
					StringVal: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/%s:latest", job.Namespace, buildName),
				}},
				{Name: "base-image", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: baseImage}},
				// ADR-031 Phase 2: Pass custom Dockerfile path
				{Name: "dockerfile-path", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: dockerfilePath}},
				// ADR-038: Pass notebook path for requirements.txt fallback chain
				{Name: "notebook-path", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: job.Spec.Notebook.Path}},
			},
			Workspaces: func() []tektonv1.WorkspaceBinding {
				workspaces := []tektonv1.WorkspaceBinding{
					{
						Name: "shared-workspace",
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							// ADR-040: Use unique PVC per build to avoid workspace contention
							ClaimName: pvcName,
						},
					},
				}

				// Add Git credentials workspace if specified
				// ADR-031: Tekton requires different secret format than validation pod
				// - Tekton build: Uses git-credentials-tekton (basic-auth format with .gitconfig + .git-credentials)
				// - Validation pod: Uses git-credentials (standard format with username + password)
				if job.Spec.Notebook.Git.CredentialsSecret != "" {
					// Use Tekton-specific secret format for build
					tektonSecretName := job.Spec.Notebook.Git.CredentialsSecret + "-tekton"
					workspaces = append(workspaces, tektonv1.WorkspaceBinding{
						Name: "git-credentials",
						Secret: &corev1.SecretVolumeSource{
							SecretName: tektonSecretName,
						},
					})
				}

				return workspaces
			}(),
		},
	}
}

// GetBuildStatus returns the current build status for a Tekton TaskRun or PipelineRun
func (t *TektonStrategy) GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error) {
	logger := log.FromContext(ctx)

	// List all PipelineRuns with our label
	pipelineRunList := &tektonv1.PipelineRunList{}
	if err := t.client.List(ctx, pipelineRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		// ADR-030 Phase 1: Provide context about what failed
		return nil, fmt.Errorf("failed to list pipelineruns (check RBAC permissions for pipelineruns.tekton.dev): %w", err)
	}

	// Find the PipelineRun with matching name
	for i := range pipelineRunList.Items {
		if pipelineRunList.Items[i].Name == buildName {
			logger.V(1).Info("Found PipelineRun", "name", buildName)
			return t.getPipelineRunStatus(ctx, &pipelineRunList.Items[i]), nil
		}
	}

	// Try TaskRuns
	taskRunList := &tektonv1.TaskRunList{}
	if err := t.client.List(ctx, taskRunList, client.MatchingLabels{"mlops.redhat.com/notebook-validation": "true"}); err != nil {
		// ADR-030 Phase 1: Provide context about what failed
		return nil, fmt.Errorf("failed to list taskruns (check RBAC permissions for taskruns.tekton.dev): %w", err)
	}

	// Find the TaskRun with matching name
	for i := range taskRunList.Items {
		if taskRunList.Items[i].Name == buildName {
			logger.V(1).Info("Found TaskRun", "name", buildName)
			return t.getTaskRunStatus(&taskRunList.Items[i]), nil
		}
	}

	// ADR-030 Phase 1: Provide helpful error message with context
	logger.V(1).Info("Build not found", "buildName", buildName, "pipelineRunCount", len(pipelineRunList.Items), "taskRunCount", len(taskRunList.Items))
	return nil, fmt.Errorf("build not found: %s (searched %d pipelineruns and %d taskruns with label mlops.redhat.com/notebook-validation=true)",
		buildName, len(pipelineRunList.Items), len(taskRunList.Items))
}

// getPipelineRunStatus extracts status from a PipelineRun
func (t *TektonStrategy) getPipelineRunStatus(ctx context.Context, pr *tektonv1.PipelineRun) *BuildInfo {
	logger := log.FromContext(ctx)
	info := &BuildInfo{
		Name: pr.Name,
	}

	// Get status from conditions
	for _, condition := range pr.Status.Conditions {
		if condition.Type == TektonConditionSucceeded {
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

	// ADR-031: Extract image reference from PipelineRun parameters
	// The image-reference parameter contains the built image location
	imageRefFound := false
	for _, param := range pr.Spec.Params {
		if param.Name == "image-reference" {
			info.ImageReference = param.Value.StringVal
			imageRefFound = true
			logger.V(1).Info("Extracted image reference from PipelineRun",
				"imageRef", info.ImageReference,
				"pipelineRun", pr.Name)
			break
		}
	}

	if !imageRefFound {
		// Log all parameter names for debugging
		paramNames := make([]string, len(pr.Spec.Params))
		for i, param := range pr.Spec.Params {
			paramNames[i] = param.Name
		}
		logger.Error(nil, "image-reference parameter not found in PipelineRun",
			"pipelineRun", pr.Name,
			"availableParams", paramNames)
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
		if condition.Type == TektonConditionSucceeded {
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

// GetLatestBuild returns the most recent PipelineRun for a Pipeline
func (t *TektonStrategy) GetLatestBuild(ctx context.Context, pipelineName string) (*BuildInfo, error) {
	logger := log.FromContext(ctx)

	// List all PipelineRuns for this Pipeline
	pipelineRunList := &tektonv1.PipelineRunList{}
	if err := t.client.List(ctx, pipelineRunList, client.MatchingLabels{
		"tekton.dev/pipeline":                  pipelineName,
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return nil, fmt.Errorf("failed to list pipelineruns: %w", err)
	}

	if len(pipelineRunList.Items) == 0 {
		return nil, fmt.Errorf("no pipelineruns found for Pipeline: %s", pipelineName)
	}

	logger.Info("Found PipelineRuns for Pipeline", "pipelineName", pipelineName, "count", len(pipelineRunList.Items))

	// Find the most recent PipelineRun (by creation timestamp)
	var mostRecent *tektonv1.PipelineRun
	for i := range pipelineRunList.Items {
		pr := &pipelineRunList.Items[i]
		if mostRecent == nil || pr.CreationTimestamp.After(mostRecent.CreationTimestamp.Time) {
			mostRecent = pr
		}
	}

	if mostRecent == nil {
		return nil, fmt.Errorf("no suitable pipelinerun found for Pipeline: %s", pipelineName)
	}

	logger.Info("Using most recent PipelineRun", "pipelineRunName", mostRecent.Name)
	return t.getPipelineRunStatus(ctx, mostRecent), nil
}

// TriggerBuild manually triggers a Tekton build (creates a new PipelineRun)
func (t *TektonStrategy) TriggerBuild(ctx context.Context, buildName string) error {
	// For Tekton, we would need to create a new PipelineRun from the Pipeline
	// This is more complex than S2I and would require the full job context
	// For now, return not implemented
	return fmt.Errorf("manual trigger not yet implemented for Tekton")
}

// GetImageFromImageStream checks ImageStream for recently pushed image (Tekton doesn't use ImageStreams)
func (t *TektonStrategy) GetImageFromImageStream(ctx context.Context, imageStreamName string) (string, error) {
	// Tekton doesn't use ImageStreams - it pushes directly to external registries
	return "", fmt.Errorf("ImageStream not applicable for Tekton strategy")
}

// CleanupOldBuilds removes old PipelineRuns to prevent resource accumulation
func (t *TektonStrategy) CleanupOldBuilds(ctx context.Context, pipelineName string, keepCount int) error {
	logger := log.FromContext(ctx)

	// List all PipelineRuns for this Pipeline
	pipelineRunList := &tektonv1.PipelineRunList{}
	if err := t.client.List(ctx, pipelineRunList, client.MatchingLabels{
		"tekton.dev/pipeline":                  pipelineName,
		"mlops.redhat.com/notebook-validation": "true",
	}); err != nil {
		return fmt.Errorf("failed to list pipelineruns: %w", err)
	}

	if len(pipelineRunList.Items) <= keepCount {
		logger.V(1).Info("No PipelineRuns to clean up", "pipelineName", pipelineName, "totalRuns", len(pipelineRunList.Items), "keepCount", keepCount)
		return nil
	}

	// Sort PipelineRuns by creation timestamp (newest first)
	runs := pipelineRunList.Items
	// Sort using a simple bubble sort since we can't import sort package easily
	for i := 0; i < len(runs)-1; i++ {
		for j := 0; j < len(runs)-i-1; j++ {
			if runs[j].CreationTimestamp.Before(&runs[j+1].CreationTimestamp) {
				runs[j], runs[j+1] = runs[j+1], runs[j]
			}
		}
	}

	// Delete old PipelineRuns (keep the most recent keepCount runs)
	runsToDelete := runs[keepCount:]
	deletedCount := 0

	for i := range runsToDelete {
		pr := &runsToDelete[i]
		// Don't delete running PipelineRuns
		for _, condition := range pr.Status.Conditions {
			if condition.Type == TektonConditionSucceeded && condition.Status == corev1.ConditionUnknown {
				logger.Info("Skipping running PipelineRun", "pipelineRunName", pr.Name)
				continue
			}
		}

		if err := t.client.Delete(ctx, pr); err != nil {
			if !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete old PipelineRun", "pipelineRunName", pr.Name)
				continue
			}
		}
		deletedCount++
		logger.V(1).Info("Deleted old PipelineRun", "pipelineRunName", pr.Name)
	}

	logger.Info("Cleaned up old PipelineRuns", "pipelineName", pipelineName, "deletedCount", deletedCount)
	return nil
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
