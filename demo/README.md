# Terminal Demo Recordings

This directory contains [VHS](https://github.com/charmbracelet/vhs) tape files that produce terminal demo GIFs for the project README.

## Demos

| Demo | Audience | Tape File | Output |
|------|----------|-----------|--------|
| Local Validation | Data scientists | `local-validation.tape` | `local-validation.gif` |
| OpenShift Deploy | Platform engineers | `openshift-deploy.tape` | `openshift-deploy.gif` |

## Prerequisites

### Install VHS

VHS requires Go and ffmpeg.

**macOS (Homebrew):**

```bash
brew install charmbracelet/tap/vhs ffmpeg
```

**Linux:**

```bash
go install github.com/charmbracelet/vhs@latest
sudo apt-get install -y ffmpeg
```

**Verify installation:**

```bash
vhs --version
ffmpeg -version
```

### Per-demo requirements

**Local Validation demo:**
- Podman or Docker installed
- Python 3 installed

**OpenShift Deploy demo:**
- `oc` CLI installed and logged into an OpenShift cluster
- Go 1.25+ installed
- `make` available
- Repository cloned locally

## Render the GIFs

From the repository root:

```bash
# Data scientist demo
vhs demo/local-validation.tape

# Platform engineer demo
vhs demo/openshift-deploy.tape
```

Each command produces a `.gif` and `.mp4` in the `demo/` directory.

## Editing a demo

1. Edit the `.tape` file in this directory.
2. Render with `vhs demo/<name>.tape` to preview.
3. Commit only the `.tape` file. GIF and MP4 files are gitignored.

## VHS syntax reference

- [VHS README](https://github.com/charmbracelet/vhs)
- `Type "text"` types text into the terminal
- `Enter` presses the Enter key
- `Sleep 2s` pauses for 2 seconds
- `Hide` / `Show` hides or shows recording (skip slow steps)
- `Set Theme "Catppuccin Mocha"` sets the color theme
- `Output demo/name.gif` defines the output file

## Notes

- GIF and MP4 files are not committed to the repository (too large for git).
- The README references GIFs at `demo/local-validation.gif` and `demo/openshift-deploy.gif`.
- To update the README GIFs, render locally and host the resulting files (or use a CI job to render and attach to releases).
