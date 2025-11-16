package controller

// Container and pod constants
const (
	// GitCloneContainerName is the standard name for git-clone init containers
	GitCloneContainerName = "git-clone"
)

// Status constants
const (
	// StatusFailed represents a failed status (used for validation, pod, and cell status)
	StatusFailed = "failed"

	// StatusSucceeded represents a succeeded status
	StatusSucceeded = "succeeded"
)

// Git credential type constants
const (
	// GitCredTypeSSH represents SSH-based git authentication
	GitCredTypeSSH = "ssh"

	// GitCredTypeHTTPS represents HTTPS-based git authentication
	GitCredTypeHTTPS = "https"
)

// Notebook format constants
const (
	// NotebookFormatMarkdown represents markdown notebook format
	NotebookFormatMarkdown = "markdown"

	// NotebookFormatNormalized represents normalized notebook format
	NotebookFormatNormalized = "normalized"
)

// Default image constants
const (
	// DefaultJupyterImage is the default Jupyter notebook base image
	DefaultJupyterImage = "quay.io/jupyter/minimal-notebook:latest"
)
