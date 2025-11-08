// Package build provides build strategy interfaces and implementations
package build

import (
	"context"
	"time"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildStatus represents the status of a build
type BuildStatus string

const (
	// BuildStatusPending indicates the build is pending
	BuildStatusPending BuildStatus = "Pending"
	// BuildStatusRunning indicates the build is running
	BuildStatusRunning BuildStatus = "Running"
	// BuildStatusComplete indicates the build completed successfully
	BuildStatusComplete BuildStatus = "Complete"
	// BuildStatusFailed indicates the build failed
	BuildStatusFailed BuildStatus = "Failed"
	// BuildStatusCancelled indicates the build was cancelled
	BuildStatusCancelled BuildStatus = "Cancelled"
	// BuildStatusUnknown indicates the build status is unknown
	BuildStatusUnknown BuildStatus = "Unknown"
)

// BuildInfo contains information about a build
type BuildInfo struct {
	// Name is the build name
	Name string
	// Status is the current build status
	Status BuildStatus
	// Message provides additional information about the build
	Message string
	// ImageReference is the built image reference (available when complete)
	ImageReference string
	// StartTime is when the build started
	StartTime *time.Time
	// CompletionTime is when the build completed
	CompletionTime *time.Time
	// Logs contains build logs (if available)
	Logs string
}

// Strategy defines the interface for different build backends
// This interface allows pluggable build strategies (S2I, Tekton, Kaniko, etc.)
type Strategy interface {
	// Name returns the strategy name (e.g., "s2i", "tekton", "kaniko")
	Name() string

	// Detect checks if this strategy is available in the cluster
	Detect(ctx context.Context, client client.Client) (bool, error)

	// CreateBuild creates a build for the given notebook validation job
	CreateBuild(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) (*BuildInfo, error)

	// GetBuildStatus returns the current build status
	GetBuildStatus(ctx context.Context, buildName string) (*BuildInfo, error)

	// WaitForCompletion waits for the build to complete or timeout
	WaitForCompletion(ctx context.Context, buildName string, timeout time.Duration) (*BuildInfo, error)

	// GetBuildLogs returns the build logs
	GetBuildLogs(ctx context.Context, buildName string) (string, error)

	// DeleteBuild cleans up build resources
	DeleteBuild(ctx context.Context, buildName string) error

	// ValidateConfig validates the build configuration
	ValidateConfig(config *mlopsv1alpha1.BuildConfigSpec) error
}

// Registry manages available build strategies
type Registry struct {
	strategies map[string]Strategy
	client     client.Client
}

// NewRegistry creates a new build strategy registry
func NewRegistry(client client.Client) *Registry {
	return &Registry{
		strategies: make(map[string]Strategy),
		client:     client,
	}
}

// Register registers a build strategy
func (r *Registry) Register(strategy Strategy) {
	r.strategies[strategy.Name()] = strategy
}

// Get returns a build strategy by name
func (r *Registry) Get(name string) (Strategy, bool) {
	strategy, ok := r.strategies[name]
	return strategy, ok
}

// List returns all registered strategies
func (r *Registry) List() []Strategy {
	strategies := make([]Strategy, 0, len(r.strategies))
	for _, strategy := range r.strategies {
		strategies = append(strategies, strategy)
	}
	return strategies
}

// DetectAvailable returns all available strategies in the cluster
func (r *Registry) DetectAvailable(ctx context.Context) ([]Strategy, error) {
	available := make([]Strategy, 0)
	for _, strategy := range r.strategies {
		isAvailable, err := strategy.Detect(ctx, r.client)
		if err != nil {
			// Log error but continue checking other strategies
			continue
		}
		if isAvailable {
			available = append(available, strategy)
		}
	}
	return available, nil
}

// GetOrDetect returns the specified strategy or detects the best available one
func (r *Registry) GetOrDetect(ctx context.Context, name string) (Strategy, error) {
	// If name is specified, try to get it
	if name != "" {
		strategy, ok := r.Get(name)
		if !ok {
			return nil, &StrategyNotFoundError{Name: name}
		}
		// Verify it's available
		available, err := strategy.Detect(ctx, r.client)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, &StrategyNotAvailableError{Name: name}
		}
		return strategy, nil
	}

	// Auto-detect best available strategy
	available, err := r.DetectAvailable(ctx)
	if err != nil {
		return nil, err
	}
	if len(available) == 0 {
		return nil, &NoStrategyAvailableError{}
	}

	// Return first available strategy (priority order based on registration order)
	return available[0], nil
}

// StrategyNotFoundError indicates a strategy was not found
type StrategyNotFoundError struct {
	Name string
}

func (e *StrategyNotFoundError) Error() string {
	return "build strategy not found: " + e.Name
}

// StrategyNotAvailableError indicates a strategy is not available in the cluster
type StrategyNotAvailableError struct {
	Name string
}

func (e *StrategyNotAvailableError) Error() string {
	return "build strategy not available in cluster: " + e.Name
}

// NoStrategyAvailableError indicates no build strategy is available
type NoStrategyAvailableError struct{}

func (e *NoStrategyAvailableError) Error() string {
	return "no build strategy available in cluster"
}

