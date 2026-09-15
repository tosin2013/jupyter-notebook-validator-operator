#!/bin/bash
# Render VHS tape files to GIF using a two-step process:
# 1. VHS captures terminal frames (text + cursor PNGs)
# 2. ffmpeg composites frames into a GIF
#
# This workaround is needed because VHS's built-in GIF creation
# silently fails in some headless environments.
#
# Usage: ./demo/render.sh demo/local-validation.tape
#        ./demo/render.sh demo/openshift-deploy.tape
#        ./demo/render.sh   (renders both)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"

render_tape() {
    local tape_file="$1"
    local base_name
    base_name=$(basename "$tape_file" .tape)
    local output_gif="$SCRIPT_DIR/${base_name}.gif"
    local frame_dir="/tmp/vhs-frames-${base_name}"

    echo "=== Rendering: $tape_file → $output_gif ==="

    # Clean previous frame dir
    rm -rf "$frame_dir"

    # Create a temporary tape that outputs frames instead of GIF
    local temp_tape
    temp_tape=$(mktemp /tmp/vhs-render-XXXX.tape)

    # Remove Output lines and add our frame output
    grep -v '^Output ' "$tape_file" > "$temp_tape"
    sed -i "1i Output \"${frame_dir}.png\"" "$temp_tape"

    echo "  Step 1: Capturing frames with VHS..."
    cd "$REPO_DIR"
    VHS_NO_SANDBOX=true vhs "$temp_tape" 2>&1 | grep -v "^$"

    rm -f "$temp_tape"

    # Check frames were captured
    local text_count cursor_count
    text_count=$(ls "${frame_dir}.png"/frame-text-*.png 2>/dev/null | wc -l)
    cursor_count=$(ls "${frame_dir}.png"/frame-cursor-*.png 2>/dev/null | wc -l)
    echo "  Captured: $text_count text frames, $cursor_count cursor frames"

    if [ "$text_count" -eq 0 ]; then
        echo "  ERROR: No frames captured. VHS recording failed."
        return 1
    fi

    # Extract dimensions from first frame
    local dims
    dims=$(ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0 "${frame_dir}.png/frame-text-00001.png" 2>/dev/null)
    local width height
    width=$(echo "$dims" | cut -d, -f1)
    height=$(echo "$dims" | cut -d, -f2)
    echo "  Frame size: ${width}x${height}"

    echo "  Step 2: Converting frames to GIF with ffmpeg..."
    ffmpeg -y \
        -r 50 -start_number 1 -i "${frame_dir}.png/frame-text-%05d.png" \
        -r 50 -start_number 1 -i "${frame_dir}.png/frame-cursor-%05d.png" \
        -filter_complex "
            [0][1]overlay[merged];
            [merged]fps=15,setpts=PTS/1.0[speed];
            [speed]split[plt_a][plt_b];
            [plt_a]palettegen=max_colors=256[plt];
            [plt_b][plt]paletteuse[palette]
        " \
        -map "[palette]" \
        "$output_gif" 2>&1 | tail -5

    local gif_size
    gif_size=$(du -h "$output_gif" | cut -f1)
    echo "  ✅ Created: $output_gif ($gif_size)"

    # Cleanup frames
    rm -rf "${frame_dir}.png"
    echo ""
}

cd "$REPO_DIR"

if [ $# -eq 0 ]; then
    echo "Rendering all demos..."
    for tape in "$SCRIPT_DIR"/*.tape; do
        render_tape "$tape"
    done
else
    for tape in "$@"; do
        render_tape "$tape"
    done
fi

echo "Done. GIF files are in demo/"
ls -lh "$SCRIPT_DIR"/*.gif 2>/dev/null || echo "(no GIFs found)"
