package controller

// Container and pod constants
const (
	// GitCloneContainerName is the standard name for git-clone init containers
	GitCloneContainerName = "git-clone"

	// PodStatusFailed represents a failed pod status
	PodStatusFailed = "failed"
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
