"""Minimal JSON Schema validator — standard library only (ADR-011).

Covers exactly the subset `schemas/manifest-v1.json` uses:

    type, const, enum, pattern, minimum
    required, properties, additionalProperties (bool or schema)
    items, $ref (local, "#/$defs/..." and "#/properties/..."), $defs

Not a general implementation and does not try to be. ADR-011 accepted that
stdlib-only means writing a few hundred lines a dependency would have
supplied; keeping the covered subset small is what makes that safe. Any
schema keyword used but not implemented here raises, rather than being
silently ignored — an unenforced constraint is worse than a missing one.
"""

from __future__ import annotations

import re

SUPPORTED = {
    "$schema", "$id", "$comment", "title", "$defs", "$ref",
    "type", "const", "enum", "pattern", "minimum",
    "required", "properties", "additionalProperties", "propertyNames", "items",
}

TYPES = {
    "object": dict, "array": list, "string": str,
    "integer": int, "number": (int, float), "boolean": bool, "null": type(None),
}


class SchemaError(ValueError):
    """The schema itself uses something this validator does not implement."""


def _resolve(ref, root):
    if not ref.startswith("#/"):
        raise SchemaError(f"only local refs are supported, got {ref!r}")
    node = root
    for part in ref[2:].split("/"):
        part = part.replace("~1", "/").replace("~0", "~")
        if part not in node:
            raise SchemaError(f"unresolvable ref {ref!r}")
        node = node[part]
    return node


def _check_supported(schema):
    unknown = set(schema) - SUPPORTED
    if unknown:
        raise SchemaError(f"schema uses unimplemented keywords: {sorted(unknown)}")


def validate(instance, schema, root=None, path="$"):
    """Return a list of human-readable error strings. Empty means valid."""
    root = root if root is not None else schema
    if "$ref" in schema:
        return validate(instance, _resolve(schema["$ref"], root), root, path)
    _check_supported(schema)
    errs = []

    if "type" in schema:
        want = schema["type"]
        py = TYPES.get(want)
        if py is None:
            raise SchemaError(f"unknown type {want!r}")
        # bool is a subclass of int; keep them distinct.
        ok = isinstance(instance, py) and not (
            want in ("integer", "number") and isinstance(instance, bool)
        )
        if not ok:
            return [f"{path}: expected {want}, got {type(instance).__name__}"]

    if "const" in schema and instance != schema["const"]:
        errs.append(f"{path}: expected constant {schema['const']!r}, got {instance!r}")

    if "enum" in schema and instance not in schema["enum"]:
        errs.append(f"{path}: {instance!r} is not one of {schema['enum']}")

    if "pattern" in schema and isinstance(instance, str):
        if not re.search(schema["pattern"], instance):
            errs.append(f"{path}: {instance!r} does not match /{schema['pattern']}/")

    if "minimum" in schema and isinstance(instance, (int, float)):
        if instance < schema["minimum"]:
            errs.append(f"{path}: {instance} is below minimum {schema['minimum']}")

    if isinstance(instance, dict):
        for key in schema.get("required", []):
            if key not in instance:
                errs.append(f"{path}: missing required property {key!r}")
        # propertyNames constrains the KEYS. Used to keep `binding.env` to
        # shell-variable shape, so a manifest cannot smuggle arbitrary names
        # into an adapter's environment.
        if "propertyNames" in schema:
            for key in instance:
                errs += [e.replace(f"{path}.{key}", f"{path} key {key!r}", 1)
                         for e in validate(key, schema["propertyNames"], root, f"{path}.{key}")]
        props = schema.get("properties", {})
        for key, value in instance.items():
            if key in props:
                errs += validate(value, props[key], root, f"{path}.{key}")
            else:
                extra = schema.get("additionalProperties", True)
                if extra is False:
                    errs.append(f"{path}: unexpected property {key!r}")
                elif isinstance(extra, dict):
                    errs += validate(value, extra, root, f"{path}.{key}")

    if isinstance(instance, list) and "items" in schema:
        for i, item in enumerate(instance):
            errs += validate(item, schema["items"], root, f"{path}[{i}]")

    return errs
