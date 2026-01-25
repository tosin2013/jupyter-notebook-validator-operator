#!/bin/bash
# cleanup-quay-images.sh - Clean up old Quay images
#
# This script deletes old images from Quay.io to clean up the registry.
# You must be logged in to Quay: podman login quay.io
#
# Usage: ./scripts/cleanup-quay-images.sh [--dry-run]
#
# Images to KEEP:
#   - quay.io/takinosh/jupyter-notebook-validator-operator:1.0.3-ocp4.19
#   - quay.io/takinosh/jupyter-notebook-validator-operator-bundle:v1.0.3-ocp4.19

set -e

DRY_RUN=false
if [ "$1" = "--dry-run" ]; then
    DRY_RUN=true
    echo "DRY RUN MODE - No images will be deleted"
    echo ""
fi

REGISTRY="quay.io/takinosh"

# Operator images to delete
OPERATOR_TAGS=(
    "1.0.0-ocp4.19"
    "1.0.0-ocp4.20"
    "1.0.1-ocp4.18"
    "1.0.2-ocp4.18"
    "1.0.3-ocp4.18"
    "1.0.4-ocp4.18"
    "1.0.4-ocp4.19"
    "1.0.4-ocp4.20"
    "1.0.5-ocp4.18-volumes"
    "1.0.7-ocp4.18"
    "1.0.7-ocp4.19"
    "1.0.7-ocp4.20"
    "1.0.8-ocp4.19"
    "1.0.9-ocp4.20"
)

# Bundle images to delete
BUNDLE_TAGS=(
    "1.0.7"
    "1.0.7-ocp4.18"
    "1.0.7-ocp4.19"
    "1.0.7-ocp4.20"
    "1.0.8"
    "1.0.9"
    "v1.0.0-ocp4.19"
    "v1.0.0-ocp4.20"
    "v1.0.1-ocp4.18"
    "v1.0.2-ocp4.18"
    "v1.0.3-ocp4.18"
    "v1.0.4-ocp4.18"
    "v1.0.4-ocp4.19"
    "v1.0.4-ocp4.20"
)

echo "=============================================="
echo "Quay.io Image Cleanup"
echo "=============================================="
echo ""

# Delete operator images
echo "=== Deleting Operator Images ==="
for tag in "${OPERATOR_TAGS[@]}"; do
    img="${REGISTRY}/jupyter-notebook-validator-operator:${tag}"
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would delete: $img"
    else
        echo "Deleting: $img"
        skopeo delete "docker://${img}" 2>/dev/null && echo "  ✅ Deleted" || echo "  ⚠️  Failed or not found"
    fi
done

echo ""
echo "=== Deleting Bundle Images ==="
for tag in "${BUNDLE_TAGS[@]}"; do
    img="${REGISTRY}/jupyter-notebook-validator-operator-bundle:${tag}"
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would delete: $img"
    else
        echo "Deleting: $img"
        skopeo delete "docker://${img}" 2>/dev/null && echo "  ✅ Deleted" || echo "  ⚠️  Failed or not found"
    fi
done

echo ""
echo "=============================================="
echo "Cleanup complete!"
echo ""
echo "Images KEPT:"
echo "  - ${REGISTRY}/jupyter-notebook-validator-operator:1.0.3-ocp4.19"
echo "  - ${REGISTRY}/jupyter-notebook-validator-operator-bundle:v1.0.3-ocp4.19"
echo "=============================================="
