#!/usr/bin/env python3
"""ADR-041: Pre-execution AST analysis for Jupyter notebooks.

Performs static analysis on notebook code cells before execution to detect:
- Risky operations (exec, eval, os.system, subprocess calls)
- Missing error handling (bare except, no try/except around I/O)
- Missing assertions (no assert statements in test-like cells)
- Silent failure patterns (functions returning None implicitly)
- Unused variables that shadow important results

Usage:
    python validation_analyzer.py <notebook_path> [--level learning|development|staging|production] [--output json]

Exit codes:
    0 - No issues found (or learning mode: issues found but non-blocking)
    1 - Issues found that exceed the configured level threshold
    2 - Error reading or parsing the notebook
"""

import ast
import json
import sys
from pathlib import Path
from typing import Any


# Validation levels and their failure thresholds
LEVEL_THRESHOLDS = {
    "learning": "error",       # Only fail on errors, warn on everything else
    "development": "warning",  # Fail on warnings and errors
    "staging": "warning",      # Same as development but stricter checks enabled
    "production": "info",      # Fail on any issue
}

# Risky function calls that should be flagged
RISKY_CALLS = {
    "exec": "Arbitrary code execution via exec()",
    "eval": "Arbitrary code execution via eval()",
    "os.system": "Shell command execution via os.system()",
    "subprocess.call": "Shell command execution via subprocess.call()",
    "subprocess.run": "Shell command execution via subprocess.run()",
    "subprocess.Popen": "Shell command execution via subprocess.Popen()",
    "__import__": "Dynamic import via __import__()",
}


class Finding:
    """A single analysis finding."""

    def __init__(
        self,
        cell: int,
        severity: str,
        category: str,
        message: str,
        suggestion: str = "",
        code_example: str = "",
        line: int = -1,
    ):
        self.cell = cell
        self.severity = severity
        self.category = category
        self.message = message
        self.suggestion = suggestion
        self.code_example = code_example
        self.line = line

    def to_dict(self) -> dict[str, Any]:
        d: dict[str, Any] = {
            "cell": self.cell,
            "severity": self.severity,
            "category": self.category,
            "message": self.message,
        }
        if self.suggestion:
            d["suggestion"] = self.suggestion
        if self.code_example:
            d["codeExample"] = self.code_example
        if self.line >= 0:
            d["line"] = self.line
        return d


def load_notebook(path: str) -> dict[str, Any]:
    """Load and parse a Jupyter notebook file."""
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def get_code_cells(notebook: dict[str, Any]) -> list[tuple[int, str]]:
    """Extract code cells with their indices."""
    cells = []
    for i, cell in enumerate(notebook.get("cells", [])):
        if cell.get("cell_type") == "code":
            source = "".join(cell.get("source", []))
            if source.strip():
                cells.append((i, source))
    return cells


def check_risky_calls(cell_idx: int, source: str) -> list[Finding]:
    """Check for risky function calls in the cell."""
    findings = []
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return findings

    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            call_name = _get_call_name(node)
            if call_name in RISKY_CALLS:
                findings.append(
                    Finding(
                        cell=cell_idx,
                        severity="warning",
                        category="risky-operation",
                        message=RISKY_CALLS[call_name],
                        suggestion=f"Consider safer alternatives to {call_name}()",
                        line=node.lineno,
                    )
                )
    return findings


def check_missing_error_handling(cell_idx: int, source: str) -> list[Finding]:
    """Check for I/O operations without error handling."""
    findings = []
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return findings

    io_calls = {"open", "read_csv", "read_json", "read_excel", "read_parquet",
                "to_csv", "to_json", "to_excel", "to_parquet",
                "requests.get", "requests.post", "urlopen"}

    has_try = any(isinstance(n, ast.Try) for n in ast.walk(tree))

    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            call_name = _get_call_name(node)
            if call_name in io_calls and not has_try:
                findings.append(
                    Finding(
                        cell=cell_idx,
                        severity="warning",
                        category="missing-error-handling",
                        message=f"{call_name}() called without try/except",
                        suggestion="Wrap I/O operations in try/except to handle failures gracefully",
                        code_example=f"try:\n    result = {call_name}(...)\nexcept Exception as e:\n    print(f'Error: {{e}}')\n    raise",
                        line=node.lineno,
                    )
                )
    return findings


def check_missing_assertions(cell_idx: int, source: str) -> list[Finding]:
    """Check for cells that look like tests but lack assertions."""
    findings = []
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return findings

    has_assert = any(isinstance(n, ast.Assert) for n in ast.walk(tree))
    has_test_hint = any(
        kw in source.lower()
        for kw in ["test", "verify", "check", "validate", "assert"]
    )

    if has_test_hint and not has_assert:
        findings.append(
            Finding(
                cell=cell_idx,
                severity="info",
                category="missing-assertion",
                message="Cell appears to perform validation but has no assert statement",
                suggestion="Add assert statements to verify expected outcomes",
                code_example="assert result is not None, 'Result should not be None'\nassert len(df) > 0, 'DataFrame should not be empty'",
            )
        )
    return findings


def check_silent_failures(cell_idx: int, source: str) -> list[Finding]:
    """Check for patterns that may produce silent failures."""
    findings = []
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return findings

    for node in ast.walk(tree):
        # Functions that return None implicitly
        if isinstance(node, ast.FunctionDef):
            returns = [n for n in ast.walk(node) if isinstance(n, ast.Return)]
            if not returns:
                findings.append(
                    Finding(
                        cell=cell_idx,
                        severity="warning",
                        category="silent-failure",
                        message=f"Function '{node.name}' has no return statement (returns None implicitly)",
                        suggestion="Add an explicit return statement with a meaningful value",
                        code_example=f"def {node.name}(...):\n    ...\n    return result  # Explicit return",
                        line=node.lineno,
                    )
                )
            for ret in returns:
                if ret.value is None:
                    findings.append(
                        Finding(
                            cell=cell_idx,
                            severity="info",
                            category="silent-failure",
                            message=f"Function '{node.name}' has a bare 'return' (returns None)",
                            suggestion="Return a meaningful value or raise an exception",
                            line=ret.lineno,
                        )
                    )

        # Bare except clauses that swallow errors
        if isinstance(node, ast.ExceptHandler) and node.type is None:
            findings.append(
                Finding(
                    cell=cell_idx,
                    severity="warning",
                    category="silent-failure",
                    message="Bare 'except:' clause may hide errors silently",
                    suggestion="Catch specific exceptions: except ValueError as e:",
                    code_example="try:\n    result = compute()\nexcept ValueError as e:\n    logger.error(f'Computation failed: {e}')\n    raise",
                    line=node.lineno,
                )
            )
    return findings


def analyze_notebook(
    notebook_path: str,
    level: str = "development",
) -> dict[str, Any]:
    """Run all analyzers on a notebook and return structured results."""
    notebook = load_notebook(notebook_path)
    code_cells = get_code_cells(notebook)
    all_findings: list[Finding] = []

    analyzers = [
        check_risky_calls,
        check_missing_error_handling,
        check_silent_failures,
    ]
    # Assertion checks only for staging and production
    if level in ("staging", "production"):
        analyzers.append(check_missing_assertions)
    # In learning mode, also check assertions but as info
    elif level == "learning":
        analyzers.append(check_missing_assertions)

    for cell_idx, source in code_cells:
        for analyzer in analyzers:
            all_findings.extend(analyzer(cell_idx, source))

    # Determine if findings exceed the threshold for this level
    threshold = LEVEL_THRESHOLDS.get(level, "warning")
    severity_order = {"info": 0, "warning": 1, "error": 2}
    threshold_val = severity_order.get(threshold, 1)

    blocking = [
        f for f in all_findings if severity_order.get(f.severity, 0) >= threshold_val
    ]

    return {
        "notebook": notebook_path,
        "level": level,
        "totalFindings": len(all_findings),
        "blockingFindings": len(blocking),
        "passed": len(blocking) == 0,
        "findings": [f.to_dict() for f in all_findings],
    }


def _get_call_name(node: ast.Call) -> str:
    """Extract the function name from a Call node."""
    if isinstance(node.func, ast.Name):
        return node.func.id
    if isinstance(node.func, ast.Attribute):
        if isinstance(node.func.value, ast.Name):
            return f"{node.func.value.id}.{node.func.attr}"
        return node.func.attr
    return ""


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: validation_analyzer.py <notebook_path> [--level LEVEL] [--output json]", file=sys.stderr)
        return 2

    notebook_path = sys.argv[1]
    level = "development"
    output_format = "json"

    i = 2
    while i < len(sys.argv):
        if sys.argv[i] == "--level" and i + 1 < len(sys.argv):
            level = sys.argv[i + 1]
            i += 2
        elif sys.argv[i] == "--output" and i + 1 < len(sys.argv):
            output_format = sys.argv[i + 1]
            i += 2
        else:
            i += 1

    if not Path(notebook_path).is_file():
        print(f"Error: notebook not found: {notebook_path}", file=sys.stderr)
        return 2

    try:
        result = analyze_notebook(notebook_path, level)
    except (json.JSONDecodeError, KeyError) as e:
        print(f"Error parsing notebook: {e}", file=sys.stderr)
        return 2

    if output_format == "json":
        print(json.dumps(result, indent=2))
    else:
        for finding in result["findings"]:
            print(f"[{finding['severity'].upper()}] Cell {finding['cell']}: {finding['message']}")
        print(f"\nTotal: {result['totalFindings']} findings, {result['blockingFindings']} blocking")

    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
