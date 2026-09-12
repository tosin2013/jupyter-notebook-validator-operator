#!/usr/bin/env python3
"""ADR-041: Runtime instrumentation for Jupyter notebook cells.

Injects validation checks after each code cell to detect:
- None return values where a result is expected
- NaN values in DataFrames and numeric outputs
- Empty DataFrames or collections
- Type mismatches against expected output specs

The instrumented notebook is written to a new file for execution via Papermill.
The original notebook is never modified.

Usage:
    python validation_instrumenter.py <input_notebook> <output_notebook> [--level learning|development|staging|production] [--expected-outputs expected.json]

Exit codes:
    0 - Instrumentation succeeded
    1 - Instrumentation failed
    2 - Error reading or parsing the notebook
"""

import json
import sys
from copy import deepcopy
from pathlib import Path
from typing import Any

# Validation check template injected after each code cell.
# Uses f-string formatting for the cell index at instrumentation time.
RUNTIME_CHECK_TEMPLATE = '''
# === ADR-041 Runtime Validation (Cell {cell_idx}) ===
import sys as _val_sys
import warnings as _val_warnings

def _adr041_validate_cell_{cell_idx}():
    """Runtime validation for cell {cell_idx}."""
    _issues = []
    # Collect local variables from the previous cell execution
    _frame = _val_sys._getframe(1) if hasattr(_val_sys, '_getframe') else None
    if _frame is None:
        return _issues

    _locals = _frame.f_locals

    # Check for NaN values in pandas objects
    try:
        import pandas as _pd
        for _name, _val in _locals.items():
            if _name.startswith('_'):
                continue
            if isinstance(_val, _pd.DataFrame):
                _nan_count = _val.isna().sum().sum()
                if _nan_count > 0:
                    _issues.append({{
                        "cell": {cell_idx},
                        "severity": "warning",
                        "category": "nan-detected",
                        "message": f"DataFrame '{{_name}}' contains {{_nan_count}} NaN values",
                        "suggestion": f"Use {{_name}}.dropna() or {{_name}}.fillna(value) to handle missing data"
                    }})
                if len(_val) == 0:
                    _issues.append({{
                        "cell": {cell_idx},
                        "severity": "warning",
                        "category": "empty-dataframe",
                        "message": f"DataFrame '{{_name}}' is empty (0 rows)",
                        "suggestion": "Verify data loading and filtering steps"
                    }})
            elif isinstance(_val, _pd.Series):
                _nan_count = _val.isna().sum()
                if _nan_count > 0:
                    _issues.append({{
                        "cell": {cell_idx},
                        "severity": "warning",
                        "category": "nan-detected",
                        "message": f"Series '{{_name}}' contains {{_nan_count}} NaN values",
                        "suggestion": f"Use {{_name}}.dropna() or {{_name}}.fillna(value)"
                    }})
    except ImportError:
        pass  # pandas not available

    # Check for None values in result-like variables
    _result_names = [n for n in _locals if not n.startswith('_') and n not in ('In', 'Out', 'get_ipython', 'exit', 'quit')]
    for _name in _result_names:
        _val = _locals[_name]
        if _val is None:
            _issues.append({{
                "cell": {cell_idx},
                "severity": "warning",
                "category": "none-value",
                "message": f"Variable '{{_name}}' is None",
                "suggestion": f"Ensure '{{_name}}' is assigned a meaningful value"
            }})

    # Check for NaN in numpy/float values
    try:
        import math as _math
        for _name in _result_names:
            _val = _locals[_name]
            if isinstance(_val, float) and _math.isnan(_val):
                _issues.append({{
                    "cell": {cell_idx},
                    "severity": "warning",
                    "category": "nan-detected",
                    "message": f"Variable '{{_name}}' is NaN",
                    "suggestion": "Check computation for division by zero or invalid operations"
                }})
    except ImportError:
        pass

    if _issues:
        _val_warnings.warn(
            f"ADR-041 validation: {{len(_issues)}} issue(s) in cell {cell_idx}: "
            + "; ".join(i["message"] for i in _issues),
            RuntimeWarning,
            stacklevel=2,
        )
        # Store issues in a global list for post-execution collection
        if not hasattr(_val_sys, '_adr041_issues'):
            _val_sys._adr041_issues = []
        _val_sys._adr041_issues.extend(_issues)

    return _issues

_adr041_validate_cell_{cell_idx}()
# === End ADR-041 Runtime Validation ===
'''

# Collector cell injected at the end of the notebook
COLLECTOR_CELL = '''
# === ADR-041 Validation Results Collection ===
import sys as _val_sys
import json as _val_json

_adr041_all_issues = getattr(_val_sys, '_adr041_issues', [])
if _adr041_all_issues:
    print("ADR041_VALIDATION_RESULTS=" + _val_json.dumps(_adr041_all_issues))
    print(f"ADR-041: {len(_adr041_all_issues)} runtime validation issue(s) detected")
else:
    print("ADR041_VALIDATION_RESULTS=[]")
    print("ADR-041: All runtime validation checks passed")
# === End ADR-041 Collection ===
'''


def instrument_notebook(
    input_path: str,
    output_path: str,
    level: str = "development",
    expected_outputs: list[dict[str, Any]] | None = None,
) -> dict[str, Any]:
    """Instrument a notebook with runtime validation checks.

    Returns metadata about the instrumentation.
    """
    with open(input_path, encoding="utf-8") as f:
        notebook = json.load(f)

    instrumented = deepcopy(notebook)
    cells = instrumented.get("cells", [])
    new_cells = []
    instrumented_count = 0

    for i, cell in enumerate(cells):
        new_cells.append(cell)

        if cell.get("cell_type") != "code":
            continue

        source = "".join(cell.get("source", []))
        if not source.strip():
            continue

        check_code = RUNTIME_CHECK_TEMPLATE.format(cell_idx=i)

        # Add expected-output checks if specified for this cell
        if expected_outputs:
            for spec in expected_outputs:
                if spec.get("cell") == i:
                    check_code += _generate_output_check(i, spec)

        check_cell = {
            "cell_type": "code",
            "metadata": {"tags": ["adr041-validation"], "adr041_injected": True},
            "source": check_code.strip().split("\n"),
            "execution_count": None,
            "outputs": [],
        }
        # Wrap source lines so each ends with \n (nbformat convention)
        check_cell["source"] = [line + "\n" for line in check_code.strip().split("\n")]
        if check_cell["source"]:
            check_cell["source"][-1] = check_cell["source"][-1].rstrip("\n")

        new_cells.append(check_cell)
        instrumented_count += 1

    # Add collector cell at the end
    collector = {
        "cell_type": "code",
        "metadata": {"tags": ["adr041-collector"], "adr041_injected": True},
        "source": [line + "\n" for line in COLLECTOR_CELL.strip().split("\n")],
        "execution_count": None,
        "outputs": [],
    }
    if collector["source"]:
        collector["source"][-1] = collector["source"][-1].rstrip("\n")
    new_cells.append(collector)

    instrumented["cells"] = new_cells

    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(instrumented, f, indent=1)

    return {
        "inputNotebook": input_path,
        "outputNotebook": output_path,
        "level": level,
        "instrumentedCells": instrumented_count,
        "totalCells": len(cells),
    }


def _generate_output_check(cell_idx: int, spec: dict[str, Any]) -> str:
    """Generate additional output validation code for a cell."""
    checks = []

    if spec.get("type"):
        expected_type = spec["type"]
        checks.append(f'''
    # Type check for cell {cell_idx}
    for _name in _result_names:
        _val = _locals.get(_name)
        if _val is not None and type(_val).__name__ != "{expected_type}" and type(_val).__module__ + "." + type(_val).__name__ != "{expected_type}":
            _val_warnings.warn(f"ADR-041: Variable '{{_name}}' has type {{type(_val).__name__}}, expected {expected_type}", RuntimeWarning)
''')

    if spec.get("notEmpty"):
        checks.append(f'''
    # NotEmpty check for cell {cell_idx}
    for _name in _result_names:
        _val = _locals.get(_name)
        if _val is not None and hasattr(_val, '__len__') and len(_val) == 0:
            _val_warnings.warn(f"ADR-041: Variable '{{_name}}' is empty but notEmpty was expected", RuntimeWarning)
''')

    return "\n".join(checks)


def main() -> int:
    if len(sys.argv) < 3:
        print(
            "Usage: validation_instrumenter.py <input_notebook> <output_notebook> "
            "[--level LEVEL] [--expected-outputs FILE]",
            file=sys.stderr,
        )
        return 2

    input_path = sys.argv[1]
    output_path = sys.argv[2]
    level = "development"
    expected_outputs = None

    i = 3
    while i < len(sys.argv):
        if sys.argv[i] == "--level" and i + 1 < len(sys.argv):
            level = sys.argv[i + 1]
            i += 2
        elif sys.argv[i] == "--expected-outputs" and i + 1 < len(sys.argv):
            with open(sys.argv[i + 1], encoding="utf-8") as f:
                expected_outputs = json.load(f)
            i += 2
        else:
            i += 1

    if not Path(input_path).is_file():
        print(f"Error: notebook not found: {input_path}", file=sys.stderr)
        return 2

    try:
        result = instrument_notebook(input_path, output_path, level, expected_outputs)
    except (json.JSONDecodeError, KeyError) as e:
        print(f"Error instrumenting notebook: {e}", file=sys.stderr)
        return 1

    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
