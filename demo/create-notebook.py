#!/usr/bin/env python3
"""Create a minimal test notebook for the VHS demo."""
import json
import sys

path = sys.argv[1] if len(sys.argv) > 1 else "/tmp/hello.ipynb"

nb = {
    "cells": [
        {
            "cell_type": "code",
            "execution_count": None,
            "metadata": {},
            "outputs": [],
            "source": ["print('Hello from Papermill!')\n", "assert 1 + 1 == 2, 'Math works'"],
        }
    ],
    "metadata": {
        "kernelspec": {"display_name": "Python 3", "language": "python", "name": "python3"},
        "language_info": {"name": "python", "version": "3.11.0"},
    },
    "nbformat": 4,
    "nbformat_minor": 5,
}

with open(path, "w") as f:
    json.dump(nb, f, indent=1)

print(f"Created {path} ({len(nb['cells'])} cell(s))")
