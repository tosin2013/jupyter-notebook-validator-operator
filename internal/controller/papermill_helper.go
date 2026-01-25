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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mlopsv1alpha1 "github.com/tosin2013/jupyter-notebook-validator-operator/api/v1alpha1"
)

// buildPapermillValidationContainer creates the main validation container with Papermill
// Based on ADR-008: Notebook Testing Strategy
// containerImage parameter allows using a custom built image (Phase 4.5: S2I Build Integration)
func (r *NotebookValidationJobReconciler) buildPapermillValidationContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob, containerImage string) corev1.Container {
	logger := log.FromContext(ctx)

	notebookPath := job.Spec.Notebook.Path

	// ADR-019: Smart Validation Pod Recovery - Phase 3
	// When using built images (S2I/Tekton), notebooks are in /opt/app-root/src/
	// When using pre-built images with git-clone, notebooks are in /workspace/repo/
	var inputNotebook string
	if shouldSkipGitClone(containerImage, job.Spec.PodConfig.ContainerImage) {
		// Built image - notebooks are in S2I source directory
		inputNotebook = fmt.Sprintf("/opt/app-root/src/%s", notebookPath)
		logger.Info("Using built image notebook path", "path", inputNotebook)
	} else {
		// Pre-built image - notebooks are cloned to /workspace/repo/
		inputNotebook = fmt.Sprintf("/workspace/repo/%s", notebookPath)
		logger.Info("Using git-cloned notebook path", "path", inputNotebook)
	}

	outputNotebook := "/workspace/output.ipynb"
	resultsJSON := "/workspace/results.json"

	// Build Papermill execution script
	// This script:
	// 1. Installs Papermill if not present
	// 2. Executes the notebook with Papermill
	// 3. Captures execution results
	// 4. Generates structured JSON output for parsing
	// 5. Handles errors gracefully
	executionScript := fmt.Sprintf(`
#!/bin/bash
set -e
set -o pipefail  # Ensure pipeline failures are caught

echo "=========================================="
echo "Jupyter Notebook Validator - Papermill"
echo "=========================================="
echo "Input Notebook: %s"
echo "Output Notebook: %s"
echo "Results JSON: %s"
echo ""

# Function to log with timestamp
log() {
    echo "[$(date +'%%Y-%%m-%%d %%H:%%M:%%S')] $1"
}

# Function to handle errors with categorization
handle_error() {
    local exit_code=$1
    local error_msg="$2"
    local error_category="${3:-unknown}"
    log "ERROR: $error_msg"

    # Create error results JSON with category
    cat > %s <<EOF
{
  "status": "failed",
  "error": "$error_msg",
  "error_category": "$error_category",
  "exit_code": $exit_code,
  "notebook_path": "%s",
  "timestamp": "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)"
}
EOF
    exit $exit_code
}

# Check if notebook exists
log "Verifying notebook exists..."
if [ ! -f "%s" ]; then
    handle_error 1 "Notebook not found at path: %s" "configuration_error"
fi
log "✓ Notebook found"

# Install Papermill if not present
log "Checking Papermill installation..."
if ! python -c "import papermill" 2>/dev/null; then
    log "Installing Papermill..."
    log "Environment: HOME=$HOME, PYTHONUSERBASE=$PYTHONUSERBASE"
    log "User: $(id -u):$(id -g)"
    log "Writable check: $(test -w /workspace && echo 'YES' || echo 'NO')"

    # Run pip install and capture output
    pip install --user --no-cache-dir papermill nbformat nbconvert 2>&1 | tee /tmp/pip_install.log
    PIP_EXIT_CODE=$?

    # Check if pip reported errors (even if exit code is 0)
    if grep -q "ERROR:\|Permission denied\|Could not install" /tmp/pip_install.log; then
        log "ERROR: Pip installation failed. Log contents:"
        cat /tmp/pip_install.log
        handle_error 2 "Failed to install Papermill due to permission errors. " \
            "The container cannot write to the Python user site-packages directory. " \
            "SOLUTION: Use a custom container image with Papermill pre-installed. " \
            "See docs/ERROR_HANDLING_GUIDE.md for instructions." "dependency_install_failed"
    fi

    # Verify papermill was actually installed
    if ! python -c "import papermill" 2>/dev/null; then
        log "ERROR: Papermill import failed after pip install"
        handle_error 2 "Papermill installation appeared to succeed but the module cannot be imported. " \
            "This usually indicates a permission or path issue. " \
            "SOLUTION: Use a custom container image with Papermill pre-installed." "dependency_install_failed"
    fi

    log "✓ Papermill installed successfully"
else
    log "✓ Papermill already installed"
fi

# Check Python version
log "Python version: $(python --version)"
log "Papermill version: $(python -c 'import papermill; print(papermill.__version__)')"

# Execute notebook with Papermill
log ""
log "=========================================="
log "Executing Notebook with Papermill"
log "=========================================="

START_TIME=$(date +%%s)

# Run Papermill with detailed output
# --log-output: Log notebook output to console
# --progress-bar: Show progress
# --report-mode: Generate execution report
# Use 'python -m papermill' instead of 'papermill' to avoid PATH issues
# when papermill is installed with --user flag
if python -m papermill \
    "%s" \
    "%s" \
    --log-output \
    --progress-bar \
    --report-mode 2>&1 | tee /workspace/execution.log; then

    EXIT_CODE=0
    STATUS="succeeded"
    ERROR_MSG="None"
    ERROR_CATEGORY="none"
    log ""
    log "✓ Notebook execution completed successfully"
else
    EXIT_CODE=$?
    STATUS="failed"
    ERROR_CATEGORY="notebook_execution_failed"

    # Analyze the error log to provide better categorization
    if grep -q "PermissionError\|Permission denied" /workspace/execution.log; then
        ERROR_MSG="Notebook execution failed due to permission errors. Check that the container has write access to required directories."
        ERROR_CATEGORY="environment_setup_failed"
    elif grep -q "ModuleNotFoundError\|ImportError\|No module named" /workspace/execution.log; then
        ERROR_MSG="Notebook execution failed due to missing Python dependencies. Consider using a custom image with required packages pre-installed."
        ERROR_CATEGORY="dependency_install_failed"
    elif grep -q "NameError\|AttributeError\|TypeError" /workspace/execution.log; then
        ERROR_MSG="Notebook execution failed due to code errors. Review the notebook code for issues."
        ERROR_CATEGORY="notebook_execution_failed"
    else
        ERROR_MSG="Papermill execution failed with exit code $EXIT_CODE. Check logs for details."
        ERROR_CATEGORY="notebook_execution_failed"
    fi

    log ""
    log "✗ Notebook execution failed with exit code: $EXIT_CODE"
    log "Error category: $ERROR_CATEGORY"
fi

END_TIME=$(date +%%s)
DURATION=$((END_TIME - START_TIME))

log "Execution duration: ${DURATION}s"

# ADR-041: Exit Code Validation and Developer Safety Framework
# Apply validation config checks
log ""
log "=========================================="
log "ADR-041: Validation Safety Checks"
log "=========================================="
log "Validation Level: ${VALIDATION_LEVEL:-development}"
log "Strict Mode: ${VALIDATION_STRICT_MODE:-false}"
log "Fail on Stderr: ${VALIDATION_FAIL_ON_STDERR:-false}"
log "Fail on Warnings: ${VALIDATION_FAIL_ON_WARNINGS:-false}"
log "Detect Silent Failures: ${VALIDATION_DETECT_SILENT_FAILURES:-true}"

# Check for stderr output (ADR-041: failOnStderr)
if [ "${VALIDATION_FAIL_ON_STDERR}" = "true" ] && [ -f /workspace/execution.log ]; then
    # Check if there's any stderr content (warnings, errors printed to stderr)
    if grep -q "^\(WARNING\|Error\|ERROR\|Traceback\|DeprecationWarning\|FutureWarning\)" /workspace/execution.log; then
        if [ "$STATUS" = "succeeded" ]; then
            log "⚠️ ADR-041: Stderr output detected with failOnStderr=true"
            STATUS="failed"
            ERROR_MSG="Validation failed: stderr output detected (ADR-041 failOnStderr enabled)"
            ERROR_CATEGORY="validation_stderr_failure"
            EXIT_CODE=1
        fi
    fi
fi

# Check for Python warnings (ADR-041: failOnWarnings)
if [ "${VALIDATION_FAIL_ON_WARNINGS}" = "true" ] && [ -f /workspace/execution.log ]; then
    if grep -qE "(Warning|warning|WARN|UserWarning|DeprecationWarning|FutureWarning|RuntimeWarning)" /workspace/execution.log; then
        if [ "$STATUS" = "succeeded" ]; then
            log "⚠️ ADR-041: Python warnings detected with failOnWarnings=true"
            STATUS="failed"
            ERROR_MSG="Validation failed: Python warnings detected (ADR-041 failOnWarnings enabled)"
            ERROR_CATEGORY="validation_warning_failure"
            EXIT_CODE=1
        fi
    fi
fi

# Strict mode enforcement (ADR-041: strictMode)
if [ "${VALIDATION_STRICT_MODE}" = "true" ]; then
    log "🔒 ADR-041: Strict mode enabled - applying maximum validation rigor"
    # In strict mode, any stderr or warning is a failure
    if [ -f /workspace/execution.log ]; then
        STDERR_LINES=$(grep -c "^\(WARNING\|Error\|Traceback\)" /workspace/execution.log 2>/dev/null || echo "0")
        if [ "$STDERR_LINES" -gt 0 ] && [ "$STATUS" = "succeeded" ]; then
            log "⚠️ ADR-041: Strict mode detected $STDERR_LINES stderr/warning lines"
            STATUS="failed"
            ERROR_MSG="Validation failed in strict mode: found $STDERR_LINES stderr/warning lines"
            ERROR_CATEGORY="validation_strict_mode_failure"
            EXIT_CODE=1
        fi
    fi
fi

log "✓ ADR-041 safety checks completed"

# Parse notebook output for cell results
log ""
log "Parsing notebook results..."

# Extract cell execution results using Python
python3 <<PYTHON_SCRIPT
import json
import sys
from pathlib import Path

try:
    import nbformat
    
    # Read the executed notebook
    output_notebook = "%s"
    
    if not Path(output_notebook).exists():
        print("Warning: Output notebook not found, using input notebook")
        output_notebook = "%s"
    
    with open(output_notebook, 'r') as f:
        nb = nbformat.read(f, as_version=4)
    
    # Extract cell results
    results = {
        "status": "%s",
        "error": "%s",
        "error_category": "%s",
        "exit_code": %s,
        "notebook_path": "%s",
        "execution_duration_seconds": %s,
        "timestamp": "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)",
        "cells": []
    }
    
    for idx, cell in enumerate(nb.cells):
        cell_result = {
            "cell_index": idx,
            "cell_type": cell.cell_type,
            "execution_count": cell.get("execution_count"),
        }
        
        # For code cells, capture execution status
        if cell.cell_type == "code":
            # Check for errors in outputs
            has_error = False
            error_msg = None
            
            for output in cell.get("outputs", []):
                if output.get("output_type") == "error":
                    has_error = True
                    error_msg = output.get("evalue", "Unknown error")
                    cell_result["error"] = error_msg
                    cell_result["traceback"] = output.get("traceback", [])
            
            cell_result["status"] = "failed" if has_error else "succeeded"
        
        results["cells"].append(cell_result)
    
    # Count cell statistics
    total_cells = len(results["cells"])
    code_cells = sum(1 for c in results["cells"] if c["cell_type"] == "code")
    failed_cells = sum(1 for c in results["cells"] if c.get("status") == "failed")
    
    results["statistics"] = {
        "total_cells": total_cells,
        "code_cells": code_cells,
        "failed_cells": failed_cells,
        "success_rate": round((code_cells - failed_cells) / code_cells * 100, 2) if code_cells > 0 else 100.0
    }
    
    # Write results to JSON
    with open("%s", 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"✓ Parsed {total_cells} cells ({code_cells} code cells)")
    print(f"✓ Success rate: {results['statistics']['success_rate']}%%")
    
except Exception as e:
    print(f"Error parsing notebook: {e}", file=sys.stderr)
    # Create minimal results
    results = {
        "status": "failed",
        "error": f"Failed to parse notebook: {str(e)}",
        "error_category": "notebook_execution_failed",
        "exit_code": 1,
        "notebook_path": "%s",
        "timestamp": "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)"
    }
    with open("%s", 'w') as f:
        json.dump(results, f, indent=2)
    sys.exit(1)
PYTHON_SCRIPT

log ""
log "=========================================="
log "Validation Complete"
log "=========================================="
log "Status: $STATUS"
log "Results saved to: %s"
log ""

# Display results summary
if [ -f "%s" ]; then
    log "Results Summary:"
    cat %s | python3 -m json.tool
fi

# Parse golden notebook if it exists (Phase 3: Golden Notebook Comparison)
GOLDEN_NOTEBOOK_PATH="/workspace/golden/%s"
GOLDEN_JSON="/workspace/golden.json"

if [ -f "$GOLDEN_NOTEBOOK_PATH" ]; then
    log ""
    log "=========================================="
    log "Parsing Golden Notebook"
    log "=========================================="
    log "Golden notebook path: $GOLDEN_NOTEBOOK_PATH"

    python3 <<GOLDEN_PYTHON_SCRIPT
import json
import sys
from pathlib import Path

try:
    import nbformat

    golden_notebook = "$GOLDEN_NOTEBOOK_PATH"

    if not Path(golden_notebook).exists():
        print(f"Warning: Golden notebook not found at {golden_notebook}")
        sys.exit(0)

    with open(golden_notebook, 'r') as f:
        nb = nbformat.read(f, as_version=4)

    # Extract golden notebook structure
    golden_data = {
        "cells": []
    }

    for idx, cell in enumerate(nb.cells):
        cell_data = {
            "cell_type": cell.cell_type,
            "execution_count": cell.get("execution_count"),
            "metadata": cell.get("metadata", {}),
            "source": cell.get("source", ""),
            "outputs": []
        }

        # For code cells, capture outputs
        if cell.cell_type == "code":
            for output in cell.get("outputs", []):
                output_data = {
                    "output_type": output.get("output_type"),
                }

                if "text" in output:
                    output_data["text"] = output["text"]
                if "data" in output:
                    output_data["data"] = output["data"]
                if "execution_count" in output:
                    output_data["execution_count"] = output["execution_count"]
                if "name" in output:
                    output_data["name"] = output["name"]
                if "traceback" in output:
                    output_data["traceback"] = output["traceback"]
                if "ename" in output:
                    output_data["ename"] = output["ename"]
                if "evalue" in output:
                    output_data["evalue"] = output["evalue"]

                cell_data["outputs"].append(output_data)

        golden_data["cells"].append(cell_data)

    # Write golden notebook data to JSON
    with open("$GOLDEN_JSON", 'w') as f:
        json.dump(golden_data, f, indent=2)

    print(f"✓ Parsed golden notebook: {len(golden_data['cells'])} cells")

except Exception as e:
    print(f"Error parsing golden notebook: {e}", file=sys.stderr)
    sys.exit(0)  # Don't fail the validation if golden parsing fails
GOLDEN_PYTHON_SCRIPT

    if [ -f "$GOLDEN_JSON" ]; then
        log "✓ Golden notebook parsed successfully"
        log "Golden Notebook Summary:"
        cat $GOLDEN_JSON | python3 -m json.tool | head -n 50
    fi
else
    log "No golden notebook found at $GOLDEN_NOTEBOOK_PATH, skipping comparison"
fi

# Exit with appropriate code
exit $EXIT_CODE
`,
		inputNotebook, outputNotebook, resultsJSON,
		resultsJSON, notebookPath,
		inputNotebook, inputNotebook,
		inputNotebook, outputNotebook,
		outputNotebook, inputNotebook, `${STATUS}`, `${ERROR_MSG}`, `${ERROR_CATEGORY}`, `${EXIT_CODE}`, notebookPath, `${DURATION}`,
		resultsJSON, notebookPath, resultsJSON,
		resultsJSON, resultsJSON, resultsJSON,
		notebookPath)

	// Use the provided containerImage (may be built image or spec image)
	if containerImage == "" {
		containerImage = job.Spec.PodConfig.ContainerImage
	}

	// ADR-041: Build validation config environment variables
	validationEnvVars := buildValidationConfigEnvVars(job.Spec.ValidationConfig)

	container := corev1.Container{
		Name:  "validator",
		Image: containerImage,
		Command: []string{
			"/bin/bash",
			"-c",
			executionScript,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
			{
				// ADR-005: OpenShift Compatibility
				// Mount emptyDir at /home/jovyan to prevent permission errors
				// Jupyter containers expect this directory to exist and be writable
				Name:      "jovyan-home",
				MountPath: "/home/jovyan",
			},
		},
		Resources: convertResourceRequirements(job.Spec.PodConfig.Resources),
		Env: append(append(convertEnvVars(job.Spec.PodConfig.Env), validationEnvVars...),
			corev1.EnvVar{
				Name:  "HOME",
				Value: "/workspace",
			},
			corev1.EnvVar{
				Name:  "PYTHONUSERBASE",
				Value: "/workspace/.local",
			},
			corev1.EnvVar{
				Name:  "PIP_USER",
				Value: "1",
			},
			corev1.EnvVar{
				Name:  "PIP_NO_CACHE_DIR",
				Value: "1",
			},
		),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: boolPtr(true),
			// RunAsUser is intentionally omitted to allow OpenShift to assign a UID
			// from the namespace's allocated range (ADR-005: OpenShift Compatibility)
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}

	logger.Info("Built Papermill validation container", "notebook", notebookPath)
	return container
}

// NotebookResults holds parsed notebook execution results
type NotebookResults struct {
	Status                   string         `json:"status"`
	Error                    string         `json:"error,omitempty"`
	ExitCode                 int            `json:"exit_code"`
	NotebookPath             string         `json:"notebook_path"`
	ExecutionDurationSeconds int            `json:"execution_duration_seconds"`
	Timestamp                string         `json:"timestamp"`
	Cells                    []CellResult   `json:"cells"`
	Statistics               CellStatistics `json:"statistics"`
}

// CellResult holds individual cell execution results
type CellResult struct {
	CellIndex      int      `json:"cell_index"`
	CellType       string   `json:"cell_type"`
	ExecutionCount *int     `json:"execution_count,omitempty"`
	Status         string   `json:"status,omitempty"`
	Error          string   `json:"error,omitempty"`
	Traceback      []string `json:"traceback,omitempty"`
}

// CellStatistics holds aggregate cell statistics
type CellStatistics struct {
	TotalCells  int     `json:"total_cells"`
	CodeCells   int     `json:"code_cells"`
	FailedCells int     `json:"failed_cells"`
	SuccessRate float64 `json:"success_rate"`
}

// convertResourceRequirements converts custom ResourceRequirements to Kubernetes ResourceRequirements
func convertResourceRequirements(customResources *mlopsv1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	if customResources == nil {
		return corev1.ResourceRequirements{}
	}

	k8sResources := corev1.ResourceRequirements{
		Limits:   make(corev1.ResourceList),
		Requests: make(corev1.ResourceList),
	}

	// Convert limits
	for key, value := range customResources.Limits {
		quantity, err := resource.ParseQuantity(value)
		if err == nil {
			k8sResources.Limits[corev1.ResourceName(key)] = quantity
		}
	}

	// Convert requests
	for key, value := range customResources.Requests {
		quantity, err := resource.ParseQuantity(value)
		if err == nil {
			k8sResources.Requests[corev1.ResourceName(key)] = quantity
		}
	}

	return k8sResources
}

// convertEnvVars converts custom EnvVar slice to Kubernetes EnvVar slice
func convertEnvVars(customEnvVars []mlopsv1alpha1.EnvVar) []corev1.EnvVar {
	if customEnvVars == nil {
		return nil
	}

	k8sEnvVars := make([]corev1.EnvVar, 0, len(customEnvVars))

	for _, customEnv := range customEnvVars {
		k8sEnv := corev1.EnvVar{
			Name:  customEnv.Name,
			Value: customEnv.Value,
		}

		// Convert ValueFrom if present
		if customEnv.ValueFrom != nil {
			k8sEnv.ValueFrom = &corev1.EnvVarSource{}

			if customEnv.ValueFrom.SecretKeyRef != nil {
				k8sEnv.ValueFrom.SecretKeyRef = &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: customEnv.ValueFrom.SecretKeyRef.Name,
					},
					Key: customEnv.ValueFrom.SecretKeyRef.Key,
				}
			}

			if customEnv.ValueFrom.ConfigMapKeyRef != nil {
				k8sEnv.ValueFrom.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: customEnv.ValueFrom.ConfigMapKeyRef.Name,
					},
					Key: customEnv.ValueFrom.ConfigMapKeyRef.Key,
				}
			}
		}

		k8sEnvVars = append(k8sEnvVars, k8sEnv)
	}

	return k8sEnvVars
}

// buildValidationConfigEnvVars builds environment variables from ValidationConfig
// ADR-041: Exit Code Validation and Developer Safety Framework
func buildValidationConfigEnvVars(config *mlopsv1alpha1.ValidationConfigSpec) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	if config == nil {
		// Default values when no config specified
		envVars = append(envVars,
			corev1.EnvVar{Name: "VALIDATION_LEVEL", Value: "development"},
			corev1.EnvVar{Name: "VALIDATION_STRICT_MODE", Value: boolFalse},
			corev1.EnvVar{Name: "VALIDATION_FAIL_ON_STDERR", Value: boolFalse},
			corev1.EnvVar{Name: "VALIDATION_FAIL_ON_WARNINGS", Value: boolFalse},
			corev1.EnvVar{Name: "VALIDATION_DETECT_SILENT_FAILURES", Value: boolTrue},
		)
		return envVars
	}

	// Validation level
	level := config.Level
	if level == "" {
		level = "development"
	}
	envVars = append(envVars, corev1.EnvVar{Name: "VALIDATION_LEVEL", Value: level})

	// Strict mode
	strictMode := boolFalse
	if config.StrictMode {
		strictMode = boolTrue
	}
	envVars = append(envVars, corev1.EnvVar{Name: "VALIDATION_STRICT_MODE", Value: strictMode})

	// Fail on stderr
	failOnStderr := boolFalse
	if config.FailOnStderr {
		failOnStderr = boolTrue
	}
	envVars = append(envVars, corev1.EnvVar{Name: "VALIDATION_FAIL_ON_STDERR", Value: failOnStderr})

	// Fail on warnings
	failOnWarnings := boolFalse
	if config.FailOnWarnings {
		failOnWarnings = boolTrue
	}
	envVars = append(envVars, corev1.EnvVar{Name: "VALIDATION_FAIL_ON_WARNINGS", Value: failOnWarnings})

	// Detect silent failures (default true)
	detectSilentFailures := boolTrue
	if config.DetectSilentFailures != nil && !*config.DetectSilentFailures {
		detectSilentFailures = boolFalse
	}
	envVars = append(envVars, corev1.EnvVar{Name: "VALIDATION_DETECT_SILENT_FAILURES", Value: detectSilentFailures})

	return envVars
}

// convertVolumes converts custom Volume slice to Kubernetes Volume slice
// ADR-045: Volume and PVC Support for Validation Pods
func convertVolumes(customVolumes []mlopsv1alpha1.Volume) []corev1.Volume {
	if customVolumes == nil {
		return nil
	}

	k8sVolumes := make([]corev1.Volume, 0, len(customVolumes))

	for _, customVol := range customVolumes {
		k8sVol := corev1.Volume{
			Name: customVol.Name,
		}

		// Handle PersistentVolumeClaim
		if customVol.PersistentVolumeClaim != nil {
			k8sVol.VolumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: customVol.PersistentVolumeClaim.ClaimName,
					ReadOnly:  customVol.PersistentVolumeClaim.ReadOnly,
				},
			}
		}

		// Handle ConfigMap
		if customVol.ConfigMap != nil {
			configMapVolSource := &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: customVol.ConfigMap.Name,
				},
				Optional: customVol.ConfigMap.Optional,
			}

			// Convert Items if present
			if len(customVol.ConfigMap.Items) > 0 {
				configMapVolSource.Items = make([]corev1.KeyToPath, 0, len(customVol.ConfigMap.Items))
				for _, item := range customVol.ConfigMap.Items {
					k8sItem := corev1.KeyToPath{
						Key:  item.Key,
						Path: item.Path,
					}
					if item.Mode != nil {
						k8sItem.Mode = item.Mode
					}
					configMapVolSource.Items = append(configMapVolSource.Items, k8sItem)
				}
			}

			// Convert DefaultMode if present
			if customVol.ConfigMap.DefaultMode != nil {
				configMapVolSource.DefaultMode = customVol.ConfigMap.DefaultMode
			}

			k8sVol.VolumeSource = corev1.VolumeSource{
				ConfigMap: configMapVolSource,
			}
		}

		// Handle Secret
		if customVol.Secret != nil {
			secretVolSource := &corev1.SecretVolumeSource{
				SecretName: customVol.Secret.SecretName,
				Optional:   customVol.Secret.Optional,
			}

			// Convert Items if present
			if len(customVol.Secret.Items) > 0 {
				secretVolSource.Items = make([]corev1.KeyToPath, 0, len(customVol.Secret.Items))
				for _, item := range customVol.Secret.Items {
					k8sItem := corev1.KeyToPath{
						Key:  item.Key,
						Path: item.Path,
					}
					if item.Mode != nil {
						k8sItem.Mode = item.Mode
					}
					secretVolSource.Items = append(secretVolSource.Items, k8sItem)
				}
			}

			// Convert DefaultMode if present
			if customVol.Secret.DefaultMode != nil {
				secretVolSource.DefaultMode = customVol.Secret.DefaultMode
			}

			k8sVol.VolumeSource = corev1.VolumeSource{
				Secret: secretVolSource,
			}
		}

		// Handle EmptyDir
		if customVol.EmptyDir != nil {
			emptyDirSource := &corev1.EmptyDirVolumeSource{}

			// Convert Medium if specified
			if customVol.EmptyDir.Medium != "" {
				emptyDirSource.Medium = corev1.StorageMedium(customVol.EmptyDir.Medium)
			}

			// Convert SizeLimit if specified
			if customVol.EmptyDir.SizeLimit != "" {
				quantity, err := resource.ParseQuantity(customVol.EmptyDir.SizeLimit)
				if err == nil {
					emptyDirSource.SizeLimit = &quantity
				}
			}

			k8sVol.VolumeSource = corev1.VolumeSource{
				EmptyDir: emptyDirSource,
			}
		}

		k8sVolumes = append(k8sVolumes, k8sVol)
	}

	return k8sVolumes
}

// convertVolumeMounts converts custom VolumeMount slice to Kubernetes VolumeMount slice
// ADR-045: Volume and PVC Support for Validation Pods
func convertVolumeMounts(customMounts []mlopsv1alpha1.VolumeMount) []corev1.VolumeMount {
	if customMounts == nil {
		return nil
	}

	k8sMounts := make([]corev1.VolumeMount, 0, len(customMounts))

	for _, customMount := range customMounts {
		k8sMount := corev1.VolumeMount{
			Name:      customMount.Name,
			MountPath: customMount.MountPath,
			SubPath:   customMount.SubPath,
			ReadOnly:  customMount.ReadOnly,
		}
		k8sMounts = append(k8sMounts, k8sMount)
	}

	return k8sMounts
}
