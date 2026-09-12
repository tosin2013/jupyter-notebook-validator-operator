"""What an agent reads. One definition, shared by the suite and the diff.

`conformance/skill.py` had this list inline and `tools/surface-diff.py` needed
the same one. A second copy of "the agent surface" would drift from the first,
which is the defect three checks in this repository already exist to catch --
so it lives here and both import it.

THE DISTINCTION THAT MATTERS. Not every change invalidates every result.

  description   the ACTIVATION surface. Activation is the model judging this
                text against a task, so changing it makes every activation
                rate stale.
  body          SKILL.md's body, references/, docs/workflows/ -- what an agent
                reads AFTER activating. Changing it may shift GRADES, the
                FULL/PARTIAL boundary, while activation rates stay comparable.

Everything else -- adapters, engine internals, conformance, CI, docs/research/
-- cannot affect either.
"""

from __future__ import annotations

import re
import subprocess

# Globs, not a file list: a workflow page added later is agent surface the day
# it lands, and a definition that had to be edited to notice would be wrong
# exactly when it mattered.
SURFACE_GLOBS = ("SKILL.md", "AGENTS.md", "references/*.md", "docs/workflows/*.md")

DESCRIPTION_RE = re.compile(r"^description:\s*(.+?)\s*$", re.M | re.S)


def _git(*args, ref=None):
    p = subprocess.run(["git", *args], capture_output=True, text=True)
    return p.stdout if p.returncode == 0 else ""


def files_at(ref):
    """Agent-surface paths present at `ref`, sorted."""
    import fnmatch
    listing = _git("ls-tree", "-r", "--name-only", ref).splitlines()
    return sorted(f for f in listing
                  if any(fnmatch.fnmatch(f, g) for g in SURFACE_GLOBS))


def read_at(ref, path):
    return _git("show", f"{ref}:{path}")


def description_at(ref):
    """The `description:` frontmatter of SKILL.md at `ref`, or None.

    Returns None rather than "" when it cannot be found: a missing description
    and an empty one are different facts, and treating the first as the second
    would report "unchanged" across a ref where SKILL.md did not exist.
    """
    text = read_at(ref, "SKILL.md")
    if not text:
        return None
    head = text.split("---", 2)
    body = head[1] if len(head) > 2 else text
    m = DESCRIPTION_RE.search(body)
    return m.group(1).strip() if m else None


def compare(ref_a, ref_b):
    """What moved between two refs, and what it means for a recorded result."""
    da, db = description_at(ref_a), description_at(ref_b)
    fa, fb = set(files_at(ref_a)), set(files_at(ref_b))

    changed, added, removed = [], sorted(fb - fa), sorted(fa - fb)
    for f in sorted(fa & fb):
        if read_at(ref_a, f) != read_at(ref_b, f):
            changed.append(f)

    return {
        "ref_a": ref_a, "ref_b": ref_b,
        "description_readable": da is not None and db is not None,
        "description_changed": (da != db),
        "changed": changed, "added": added, "removed": removed,
        # A description change makes activation rates stale. Everything else
        # leaves them comparable and may only shift grades.
        "activation_comparable": (da is not None and db is not None and da == db),
        "grades_comparable": not (changed or added or removed),
    }
