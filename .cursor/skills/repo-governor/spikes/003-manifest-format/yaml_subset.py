"""Minimal YAML-subset parser — spike for issue #3.

Scope deliberately tiny: block mappings, block sequences, scalars,
comments. No anchors, aliases, flow style, multi-line scalars, multiple
documents, tags, or merge keys. Standard library only (ADR-011).

The point of this spike is to find out how much code a "small vendored
parser" actually is, and where it breaks.
"""

from __future__ import annotations


class YamlSubsetError(ValueError):
    pass


def _strip_comment(line: str) -> str:
    out, quote = [], None
    for ch in line:
        if quote:
            out.append(ch)
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
            out.append(ch)
        elif ch == "#":
            break
        else:
            out.append(ch)
    if quote:
        raise YamlSubsetError(f"unterminated quote: {line!r}")
    return "".join(out).rstrip()


def _scalar(tok: str):
    tok = tok.strip()
    if not tok:
        return None
    if len(tok) >= 2 and tok[0] == tok[-1] and tok[0] in "\"'":
        return tok[1:-1]
    low = tok.lower()
    if low in ("true", "yes", "on"):
        return True
    if low in ("false", "no", "off"):
        return False
    if low in ("null", "~"):
        return None
    try:
        return int(tok)
    except ValueError:
        pass
    try:
        return float(tok)
    except ValueError:
        pass
    return tok


def _tokenize(text: str):
    rows = []
    for n, raw in enumerate(text.splitlines(), 1):
        if "\t" in raw.expandtabs(1)[: len(raw) - len(raw.lstrip())]:
            raise YamlSubsetError(f"line {n}: tab in indentation")
        if "\t" in raw[: len(raw) - len(raw.lstrip(" \t"))]:
            raise YamlSubsetError(f"line {n}: tab in indentation")
        body = _strip_comment(raw)
        if not body.strip():
            continue
        indent = len(body) - len(body.lstrip(" "))
        rows.append((n, indent, body.strip()))
    return rows


def _split_kv(s: str, lineno: int):
    quote = None
    for i, ch in enumerate(s):
        if quote:
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
        elif ch == ":" and (i + 1 == len(s) or s[i + 1] in " "):
            return s[:i].strip(), s[i + 1 :].strip()
    raise YamlSubsetError(f"line {lineno}: expected 'key: value', got {s!r}")


def _parse_block(rows, i, indent):
    """Return (value, next_index) for the block starting at rows[i]."""
    if rows[i][2].startswith("- "):
        seq = []
        while i < len(rows) and rows[i][1] == indent and rows[i][2].startswith("- "):
            lineno, _, body = rows[i]
            inner = body[2:].strip()
            if not inner:
                raise YamlSubsetError(f"line {lineno}: empty sequence entry")
            try:
                k, v = _split_kv(inner, lineno)
            except YamlSubsetError:
                seq.append(_scalar(inner))
                i += 1
                continue
            # inline mapping start: "- type: adr" plus deeper siblings
            item = {}
            item_indent = indent + 2
            if v:
                item[k] = _scalar(v)
                i += 1
            else:
                i += 1
                if i < len(rows) and rows[i][1] > item_indent:
                    item[k], i = _parse_block(rows, i, rows[i][1])
                else:
                    item[k] = None
            while i < len(rows) and rows[i][1] == item_indent and not rows[i][2].startswith("- "):
                lineno2, _, body2 = rows[i]
                k2, v2 = _split_kv(body2, lineno2)
                if v2:
                    item[k2] = _scalar(v2)
                    i += 1
                else:
                    i += 1
                    if i < len(rows) and rows[i][1] > item_indent:
                        item[k2], i = _parse_block(rows, i, rows[i][1])
                    else:
                        item[k2] = None
            seq.append(item)
        return seq, i

    mapping = {}
    while i < len(rows) and rows[i][1] == indent:
        lineno, _, body = rows[i]
        if body.startswith("- "):
            break
        k, v = _split_kv(body, lineno)
        if k in mapping:
            raise YamlSubsetError(f"line {lineno}: duplicate key {k!r}")
        if v:
            mapping[k] = _scalar(v)
            i += 1
        else:
            i += 1
            if i < len(rows) and rows[i][1] > indent:
                mapping[k], i = _parse_block(rows, i, rows[i][1])
            elif i < len(rows) and rows[i][2].startswith("- ") and rows[i][1] == indent:
                mapping[k], i = _parse_block(rows, i, indent)
            else:
                mapping[k] = None
    return mapping, i


def loads(text: str):
    rows = _tokenize(text)
    if not rows:
        return {}
    if rows[0][1] != 0:
        raise YamlSubsetError("document must start at indent 0")
    value, i = _parse_block(rows, 0, 0)
    if i != len(rows):
        raise YamlSubsetError(f"line {rows[i][0]}: unexpected indentation")
    return value
