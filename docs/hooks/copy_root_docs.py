"""MkDocs hook: copy root-level Markdown files into docs/ before build.

Root files (README.md, DESIGN_DOC.md, CONTRIBUTING.md, CHANGELOG.md,
CODE_OF_CONDUCT.md) live at the repository root for GitHub visibility.
This hook copies them into docs/ so MkDocs can include them in the nav
without maintaining duplicates.
"""

import shutil
from pathlib import Path

ROOT_FILES = {
    "README.md": "index.md",
    "DESIGN_DOC.md": "design-doc.md",
    "CONTRIBUTING.md": "contributing.md",
    "CHANGELOG.md": "changelog.md",
    "CODE_OF_CONDUCT.md": "code-of-conduct.md",
}


def on_pre_build(config, **kwargs):
    """Copy root Markdown files into the docs directory."""
    repo_root = Path(config["config_file_path"]).parent
    docs_dir = Path(config["docs_dir"])

    for src_name, dest_name in ROOT_FILES.items():
        src = repo_root / src_name
        dest = docs_dir / dest_name
        if src.exists():
            shutil.copy2(src, dest)


def on_post_build(config, **kwargs):
    """Remove copied files to keep the working tree clean."""
    docs_dir = Path(config["docs_dir"])
    for dest_name in ROOT_FILES.values():
        dest = docs_dir / dest_name
        if dest.exists():
            dest.unlink()
