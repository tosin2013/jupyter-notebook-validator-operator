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
func (r *NotebookValidationJobReconciler) buildPapermillValidationContainer(ctx context.Context, job *mlopsv1alpha1.NotebookValidationJob) corev1.Container {
	logger := log.FromContext(ctx)

	notebookPath := job.Spec.Notebook.Path
	inputNotebook := fmt.Sprintf("/workspace/repo/%s", notebookPath)
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

# Function to handle errors
handle_error() {
    local exit_code=$1
    local error_msg="$2"
    log "ERROR: $error_msg"
    
    # Create error results JSON
    cat > %s <<EOF
{
  "status": "failed",
  "error": "$error_msg",
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
    handle_error 1 "Notebook not found at path: %s"
fi
log "✓ Notebook found"

# Install Papermill if not present
log "Checking Papermill installation..."
if ! python -c "import papermill" 2>/dev/null; then
    log "Installing Papermill..."
    pip install --quiet papermill nbformat nbconvert || handle_error 2 "Failed to install Papermill"
    log "✓ Papermill installed"
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
if papermill \
    "%s" \
    "%s" \
    --log-output \
    --progress-bar \
    --report-mode 2>&1 | tee /workspace/execution.log; then

    EXIT_CODE=0
    STATUS="succeeded"
    ERROR_MSG="None"
    log ""
    log "✓ Notebook execution completed successfully"
else
    EXIT_CODE=$?
    STATUS="failed"
    ERROR_MSG="Papermill execution failed with exit code $EXIT_CODE"
    log ""
    log "✗ Notebook execution failed with exit code: $EXIT_CODE"
fi

END_TIME=$(date +%%s)
DURATION=$((END_TIME - START_TIME))

log "Execution duration: ${DURATION}s"

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
		outputNotebook, inputNotebook, `${STATUS}`, `${ERROR_MSG}`, `${EXIT_CODE}`, notebookPath, `${DURATION}`,
		resultsJSON, notebookPath, resultsJSON,
		resultsJSON, resultsJSON, resultsJSON,
		notebookPath)

	container := corev1.Container{
		Name:  "validator",
		Image: job.Spec.PodConfig.ContainerImage,
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
		},
		Resources: convertResourceRequirements(job.Spec.PodConfig.Resources),
		Env: append(convertEnvVars(job.Spec.PodConfig.Env), corev1.EnvVar{
			Name:  "HOME",
			Value: "/workspace",
		}),
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
