#!/bin/bash
# Script to update the icon base64 in CSV files
# Usage: ./update-icon.sh <csv-file-path>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ICON_FILE="$PROJECT_ROOT/images/small-logo-base64.txt"

if [ ! -f "$ICON_FILE" ]; then
    echo "ERROR: Icon file not found: $ICON_FILE"
    exit 1
fi

# Get the correct base64 icon
CORRECT_ICON=$(cat "$ICON_FILE")

# Function to update a CSV file
update_csv() {
    local csv_file="$1"
    
    if [ ! -f "$csv_file" ]; then
        echo "ERROR: CSV file not found: $csv_file"
        return 1
    fi
    
    echo "Updating icon in: $csv_file"
    
    # Create a temporary file
    local temp_file=$(mktemp)
    
    # Use awk to replace the base64data line
    awk -v new_icon="$CORRECT_ICON" '
    /^  - base64data:/ {
        print "  - base64data: " new_icon
        next
    }
    { print }
    ' "$csv_file" > "$temp_file"
    
    # Replace original with updated
    mv "$temp_file" "$csv_file"
    
    echo "✓ Updated: $csv_file"
}

# If no arguments, show usage
if [ $# -eq 0 ]; then
    echo "Usage: $0 <csv-file-path> [<csv-file-path2> ...]"
    echo ""
    echo "Updates the icon base64 in ClusterServiceVersion YAML files."
    echo "Uses the icon from: $ICON_FILE"
    echo ""
    echo "Examples:"
    echo "  $0 bundle/manifests/jupyter-notebook-validator-operator.clusterserviceversion.yaml"
    echo "  $0 /path/to/community-operators-prod/operators/jupyter-notebook-validator-operator/1.0.2/manifests/*.clusterserviceversion.yaml"
    exit 0
fi

# Update each provided file
for csv_file in "$@"; do
    update_csv "$csv_file"
done

echo ""
echo "Done! Verify the changes with:"
echo "  grep -A1 'icon:' <csv-file>"
