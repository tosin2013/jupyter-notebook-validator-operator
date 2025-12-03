#!/bin/bash
# Create File-Based Catalog (FBC) for Jupyter Notebook Validator Operator
# Supports OpenShift 4.18, 4.19, and 4.20

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CATALOG_DIR="catalog"
OPERATOR_NAME="jupyter-notebook-validator-operator"
REGISTRY="quay.io/takinosh"

# Versions
VERSION_418="1.0.5-ocp4.18"
VERSION_419="1.0.6-ocp4.19"
VERSION_420="1.0.7-ocp4.20"

echo -e "${GREEN}=== Creating File-Based Catalog (FBC) ===${NC}"
echo ""

# Step 1: Create catalog directory
echo -e "${YELLOW}Step 1: Creating catalog directory...${NC}"
mkdir -p "$CATALOG_DIR"
echo -e "${GREEN}✓ Created $CATALOG_DIR/${NC}"
echo ""

# Step 2: Create catalog.yaml
echo -e "${YELLOW}Step 2: Creating catalog.yaml...${NC}"
cat > "$CATALOG_DIR/catalog.yaml" <<EOF
---
schema: olm.package
name: $OPERATOR_NAME
defaultChannel: stable
description: |
  Kubernetes operator for validating Jupyter notebooks in MLOps workflows.
  
  Features:
  - Git integration with credential support
  - Papermill notebook execution
  - Golden notebook comparison
  - Model validation (KServe, OpenShift AI, vLLM, etc.)
  - Build integration (S2I, Tekton)
  - External volume support (PVC, ConfigMap, Secret, EmptyDir)
  
  Supports OpenShift 4.18, 4.19, and 4.20.
---
schema: olm.channel
package: $OPERATOR_NAME
name: stable
entries:
  - name: $OPERATOR_NAME.v$VERSION_418
  - name: $OPERATOR_NAME.v$VERSION_419
    replaces: $OPERATOR_NAME.v$VERSION_418
  - name: $OPERATOR_NAME.v$VERSION_420
    replaces: $OPERATOR_NAME.v$VERSION_419
---
schema: olm.bundle
name: $OPERATOR_NAME.v$VERSION_418
package: $OPERATOR_NAME
image: $REGISTRY/$OPERATOR_NAME-bundle:$VERSION_418
properties:
  - type: olm.package
    value:
      packageName: $OPERATOR_NAME
      version: $VERSION_418
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
  - type: olm.csv.metadata
    value:
      annotations:
        com.redhat.openshift.versions: "v4.18"
---
schema: olm.bundle
name: $OPERATOR_NAME.v$VERSION_419
package: $OPERATOR_NAME
image: $REGISTRY/$OPERATOR_NAME-bundle:$VERSION_419
properties:
  - type: olm.package
    value:
      packageName: $OPERATOR_NAME
      version: $VERSION_419
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
  - type: olm.csv.metadata
    value:
      annotations:
        com.redhat.openshift.versions: "v4.19"
---
schema: olm.bundle
name: $OPERATOR_NAME.v$VERSION_420
package: $OPERATOR_NAME
image: $REGISTRY/$OPERATOR_NAME-bundle:$VERSION_420
properties:
  - type: olm.package
    value:
      packageName: $OPERATOR_NAME
      version: $VERSION_420
  - type: olm.gvk
    value:
      group: mlops.mlops.dev
      kind: NotebookValidationJob
      version: v1alpha1
  - type: olm.csv.metadata
    value:
      annotations:
        com.redhat.openshift.versions: "v4.20"
EOF

echo -e "${GREEN}✓ Created catalog.yaml${NC}"
echo ""

# Step 3: Create Dockerfile
echo -e "${YELLOW}Step 3: Creating Dockerfile...${NC}"
cat > "$CATALOG_DIR/Dockerfile" <<EOF
# File-Based Catalog (FBC) for Jupyter Notebook Validator Operator
FROM scratch

# Copy catalog definition
COPY catalog.yaml /configs/catalog.yaml

# FBC label
LABEL operators.operatorframework.io.index.configs.v1=/configs
EOF

echo -e "${GREEN}✓ Created Dockerfile${NC}"
echo ""

# Step 4: Create README
echo -e "${YELLOW}Step 4: Creating README...${NC}"
cat > "$CATALOG_DIR/README.md" <<EOF
# Jupyter Notebook Validator Operator Catalog

File-Based Catalog (FBC) for the Jupyter Notebook Validator Operator.

## Versions

- **v$VERSION_418**: OpenShift 4.18 (with volume support)
- **v$VERSION_419**: OpenShift 4.19
- **v$VERSION_420**: OpenShift 4.20

## Build Catalog

\`\`\`bash
podman build -f catalog/Dockerfile -t $REGISTRY/$OPERATOR_NAME-catalog:latest catalog/
podman push $REGISTRY/$OPERATOR_NAME-catalog:latest
\`\`\`

## Deploy Catalog

\`\`\`bash
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: $OPERATOR_NAME-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: $REGISTRY/$OPERATOR_NAME-catalog:latest
  displayName: Jupyter Notebook Validator Operator
  publisher: Community
  updateStrategy:
    registryPoll:
      interval: 10m
EOF
\`\`\`

## Upgrade Path

\`\`\`
v$VERSION_418 → v$VERSION_419 → v$VERSION_420
\`\`\`
EOF

echo -e "${GREEN}✓ Created README.md${NC}"
echo ""

# Summary
echo -e "${GREEN}=== FBC Catalog Created Successfully! ===${NC}"
echo ""
echo "Files created:"
echo "  - $CATALOG_DIR/catalog.yaml"
echo "  - $CATALOG_DIR/Dockerfile"
echo "  - $CATALOG_DIR/README.md"
echo ""
echo "Next steps:"
echo "  1. Build catalog image:"
echo "     podman build -f $CATALOG_DIR/Dockerfile -t $REGISTRY/$OPERATOR_NAME-catalog:latest $CATALOG_DIR/"
echo ""
echo "  2. Push catalog image:"
echo "     podman push $REGISTRY/$OPERATOR_NAME-catalog:latest"
echo ""
echo "  3. Deploy to cluster:"
echo "     See $CATALOG_DIR/README.md for deployment instructions"
echo ""

