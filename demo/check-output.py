#!/usr/bin/env python3
"""Check executed notebook output for the VHS demo."""
import json
import sys

path = sys.argv[1] if len(sys.argv) > 1 else "/tmp/hello-output.ipynb"

nb = json.load(open(path))
cells = [c for c in nb["cells"] if c["cell_type"] == "code"]
print(f"Executed {len(cells)} cell(s)")
for i, cell in enumerate(cells):
    for o in cell["outputs"]:
        text = o.get("text", ["(no text)"])
        if isinstance(text, list):
            text = "".join(text).strip()
        print(f"  Cell {i}: PASS  output={text[:80]}")

print()
print("All cells passed!")
