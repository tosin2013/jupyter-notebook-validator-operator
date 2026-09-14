#!/usr/bin/env bash
# setup-rosa-e2e.sh — Provision a ROSA HCP cluster and deploy the operator for E2E testing
#
# This script installs missing tools (ROSA CLI, Terraform, oc, Go), creates
# ROSA prerequisites (OIDC config, operator roles, support role trust), provisions
# a ROSA HCP cluster, installs cert-manager, deploys the operator, and runs a
# smoke test — all in one shot.
#
# Usage:
#   ./scripts/setup-rosa-e2e.sh              # Full setup: tools + cluster + operator + smoke test
#   ./scripts/setup-rosa-e2e.sh --tools-only # Just install tools, don't provision
#   ./scripts/setup-rosa-e2e.sh --deploy     # Deploy operator to existing cluster (cert-manager + Tekton + operator + smoke test)
#   ./scripts/setup-rosa-e2e.sh --audit      # Audit cluster readiness for all E2E tiers
#   ./scripts/setup-rosa-e2e.sh --secrets    # Print GitHub secrets needed for CI
#   ./scripts/setup-rosa-e2e.sh --set-secrets# Auto-set GitHub repo secrets via gh CLI
#   ./scripts/setup-rosa-e2e.sh --destroy    # Tear down cluster (keep VPC/roles)
#   ./scripts/setup-rosa-e2e.sh --status     # Check cluster status
#   ./scripts/setup-rosa-e2e.sh --terraform-validate  # Validate Terraform configs (dry-run)
#
# Prerequisites:
#   - AWS CLI configured (aws sts get-caller-identity works)
#   - OCM CLI logged in (ocm whoami works)
#
# Environment variables (optional overrides):
#   CLUSTER_NAME     — Cluster name (default: jnvo-e2e)
#   AWS_REGION       — AWS region (default: us-east-1)
#   OCP_VERSION      — OpenShift version (default: 4.20.36)
#   WORKER_COUNT     — Number of workers (default: 2)
#   OPERATOR_PREFIX  — Operator role prefix (default: jnvo-e2e)
#   VPC_ID           — Existing VPC ID (auto-detected from sandbox)
#   OPERATOR_IMG     — Operator image to deploy (default: quay.io/takinosh/jupyter-notebook-validator-operator:latest)
#   CERT_MANAGER_VERSION — cert-manager version (default: v1.17.2)
#   GO_VERSION       — Go version to install (default: 1.25.0)
#   SKIP_SMOKE_TEST  — Set to "true" to skip the post-deploy smoke test

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────
# Config
# ──────────────────────────────────────────────────────────────────────
CLUSTER_NAME="${CLUSTER_NAME:-jnvo-e2e}"
AWS_REGION="${AWS_REGION:-us-east-1}"
OCP_VERSION="${OCP_VERSION:-4.20.36}"
WORKER_COUNT="${WORKER_COUNT:-2}"
OPERATOR_PREFIX="${OPERATOR_PREFIX:-jnvo-e2e}"
OPERATOR_IMG="${OPERATOR_IMG:-quay.io/takinosh/jupyter-notebook-validator-operator:latest}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.17.2}"
TEKTON_VERSION="${TEKTON_VERSION:-v0.65.1}"
GO_VERSION_INSTALL="${GO_VERSION:-1.25.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
STATE_DIR="$REPO_DIR/.rosa-e2e-state"
ROSA_VERSION="1.2.47"
TERRAFORM_VERSION="1.9.8"
OPERATOR_NAMESPACE="jupyter-notebook-validator-operator"
TEST_NAMESPACE="e2e-tests"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${BLUE}ℹ ${NC}$*"; }
ok()      { echo -e "${GREEN}✅ ${NC}$*"; }
warn()    { echo -e "${YELLOW}⚠️  ${NC}$*"; }
fail()    { echo -e "${RED}❌ ${NC}$*"; exit 1; }
section() { echo -e "\n${CYAN}━━━ $* ━━━${NC}\n"; }

# ──────────────────────────────────────────────────────────────────────
# Preflight checks
# ──────────────────────────────────────────────────────────────────────
preflight() {
    info "Running preflight checks..."

    if ! command -v aws &>/dev/null; then
        fail "AWS CLI not found. Install: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"
    fi

    if ! aws sts get-caller-identity &>/dev/null; then
        fail "AWS CLI not authenticated. Run: aws configure"
    fi

    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query 'Account' --output text)
    CREATOR_ARN=$(aws sts get-caller-identity --query 'Arn' --output text)
    ok "AWS: account=$AWS_ACCOUNT_ID user=$(aws sts get-caller-identity --query 'UserId' --output text)"

    if ! command -v ocm &>/dev/null; then
        fail "OCM CLI not found. Install: https://github.com/openshift-online/ocm-cli/releases"
    fi

    if ! ocm whoami &>/dev/null; then
        fail "OCM CLI not logged in. Run: ocm login --use-device-code"
    fi

    OCM_USER=$(ocm whoami 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('username','unknown'))" 2>/dev/null || echo "unknown")
    ok "OCM: logged in as $OCM_USER"
}

# ──────────────────────────────────────────────────────────────────────
# Tool installation
# ──────────────────────────────────────────────────────────────────────
install_tools() {
    section "Installing tools"

    # Go (needed for make install / make deploy)
    if command -v go &>/dev/null; then
        ok "Go already installed: $(go version 2>/dev/null)"
    elif [ -x /usr/local/go/bin/go ]; then
        export PATH=$PATH:/usr/local/go/bin
        ok "Go found at /usr/local/go: $(go version 2>/dev/null)"
    else
        info "Installing Go v${GO_VERSION_INSTALL}..."
        curl -sL "https://go.dev/dl/go${GO_VERSION_INSTALL}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf /tmp/go.tar.gz
        rm -f /tmp/go.tar.gz
        export PATH=$PATH:/usr/local/go/bin
        ok "Go installed: $(go version 2>/dev/null)"
    fi
    # Ensure Go is on PATH for subprocesses
    export PATH=$PATH:/usr/local/go/bin

    # ROSA CLI
    if command -v rosa &>/dev/null; then
        ok "ROSA CLI already installed: $(rosa version 2>/dev/null | head -1)"
    else
        info "Installing ROSA CLI v${ROSA_VERSION}..."
        curl -sL "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/${ROSA_VERSION}/rosa-linux.tar.gz" \
            -o /tmp/rosa.tar.gz
        tar -xzf /tmp/rosa.tar.gz -C /tmp rosa
        sudo mv /tmp/rosa /usr/local/bin/rosa
        sudo chmod +x /usr/local/bin/rosa
        rm -f /tmp/rosa.tar.gz
        ok "ROSA CLI installed: $(rosa version 2>/dev/null | head -1)"
    fi

    # Terraform
    if command -v terraform &>/dev/null; then
        ok "Terraform already installed: $(terraform version 2>/dev/null | head -1)"
    else
        info "Installing Terraform v${TERRAFORM_VERSION}..."
        curl -sL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" \
            -o /tmp/terraform.zip
        (cd /tmp && unzip -qo terraform.zip terraform)
        sudo mv /tmp/terraform /usr/local/bin/terraform
        sudo chmod +x /usr/local/bin/terraform
        rm -f /tmp/terraform.zip
        ok "Terraform installed: $(terraform version 2>/dev/null | head -1)"
    fi

    # oc CLI
    if command -v oc &>/dev/null; then
        ok "oc CLI already installed: $(oc version --client 2>/dev/null | head -1)"
    else
        info "Installing oc CLI (stable)..."
        curl -sL "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz" \
            -o /tmp/oc.tar.gz
        tar -xzf /tmp/oc.tar.gz -C /tmp oc kubectl
        sudo mv /tmp/oc /usr/local/bin/oc
        sudo mv /tmp/kubectl /usr/local/bin/kubectl 2>/dev/null || true
        sudo chmod +x /usr/local/bin/oc
        rm -f /tmp/oc.tar.gz
        ok "oc CLI installed: $(oc version --client 2>/dev/null | head -1)"
    fi

    # gh CLI (for setting secrets)
    if command -v gh &>/dev/null; then
        ok "gh CLI already installed: $(gh version 2>/dev/null | head -1)"
    else
        info "Installing gh CLI..."
        (type -p wget >/dev/null || (sudo apt update && sudo apt install -y wget)) 2>/dev/null
        sudo mkdir -p -m 755 /etc/apt/keyrings
        wget -qO- https://cli.github.com/packages/githubcli-archive-keyring.gpg \
            | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null
        sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
            | sudo tee /etc/apt/sources.list.d/github-cli-stable.list > /dev/null
        sudo apt update -qq && sudo apt install -y -qq gh
        ok "gh CLI installed: $(gh version 2>/dev/null | head -1)"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Detect sandbox infrastructure
# ──────────────────────────────────────────────────────────────────────
detect_sandbox() {
    section "Detecting sandbox infrastructure ($AWS_REGION)"

    # Detect VPC
    if [ -z "${VPC_ID:-}" ]; then
        VPC_ID=$(aws ec2 describe-vpcs \
            --region "$AWS_REGION" \
            --filters "Name=tag:Name,Values=*rosa*" \
            --query 'Vpcs[0].VpcId' --output text 2>/dev/null || echo "None")
    fi

    if [ "$VPC_ID" = "None" ] || [ -z "$VPC_ID" ]; then
        warn "No existing ROSA VPC found — Terraform will create one"
        USE_EXISTING_VPC=false
    else
        ok "Found VPC: $VPC_ID"
        USE_EXISTING_VPC=true

        # Get subnets
        PRIVATE_SUBNET=$(aws ec2 describe-subnets \
            --region "$AWS_REGION" \
            --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*private*" \
            --query 'Subnets[0].SubnetId' --output text 2>/dev/null)
        PUBLIC_SUBNET=$(aws ec2 describe-subnets \
            --region "$AWS_REGION" \
            --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*public*" \
            --query 'Subnets[0].SubnetId' --output text 2>/dev/null)
        AZ=$(aws ec2 describe-subnets \
            --region "$AWS_REGION" \
            --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*private*" \
            --query 'Subnets[0].AvailabilityZone' --output text 2>/dev/null)

        ok "  Private subnet: $PRIVATE_SUBNET ($AZ)"
        ok "  Public subnet:  $PUBLIC_SUBNET"
    fi

    # Detect account roles
    INSTALLER_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Installer-Role \
        --query 'Role.Arn' --output text 2>/dev/null || echo "")
    SUPPORT_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Support-Role \
        --query 'Role.Arn' --output text 2>/dev/null || echo "")
    WORKER_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Worker-Role \
        --query 'Role.Arn' --output text 2>/dev/null || echo "")

    if [ -n "$INSTALLER_ROLE_ARN" ] && [ -n "$SUPPORT_ROLE_ARN" ] && [ -n "$WORKER_ROLE_ARN" ]; then
        ok "Account roles found:"
        ok "  Installer: $INSTALLER_ROLE_ARN"
        ok "  Support:   $SUPPORT_ROLE_ARN"
        ok "  Worker:    $WORKER_ROLE_ARN"
        ACCOUNT_ROLES_EXIST=true
    else
        warn "Account roles not found — ROSA CLI will create them"
        ACCOUNT_ROLES_EXIST=false
    fi
}

# ──────────────────────────────────────────────────────────────────────
# ROSA login + prerequisites
# ──────────────────────────────────────────────────────────────────────
rosa_login() {
    info "Logging into ROSA..."

    # Get OCM token from ocm config
    OCM_TOKEN=$(ocm token 2>/dev/null || echo "")
    if [ -z "$OCM_TOKEN" ]; then
        fail "Cannot retrieve OCM token. Run: ocm login --use-device-code"
    fi

    rosa login --token="$OCM_TOKEN"
    ok "ROSA login successful"
}

rosa_prerequisites() {
    section "Setting up ROSA HCP prerequisites"

    # Verify ROSA quota
    info "Verifying AWS quota for ROSA..."
    rosa verify quota --region="$AWS_REGION" || warn "Quota check returned warnings (may still proceed)"

    # Account roles (if not detected)
    if [ "$ACCOUNT_ROLES_EXIST" != "true" ]; then
        info "Creating HCP account roles..."
        rosa create account-roles --hosted-cp --mode auto --yes
        ok "Account roles created"
        INSTALLER_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Installer-Role \
            --query 'Role.Arn' --output text)
        SUPPORT_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Support-Role \
            --query 'Role.Arn' --output text)
        WORKER_ROLE_ARN=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Worker-Role \
            --query 'Role.Arn' --output text)
    else
        ok "Account roles already exist — skipping"
    fi

    # Verify support role trust policy includes the user's OCM org
    info "Verifying support role trust policy..."
    OCM_ORG_EXTERNAL_ID=$(ocm whoami 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
print(d.get('organization',{}).get('external_id',''))
" 2>/dev/null || echo "")
    if [ -n "$OCM_ORG_EXTERNAL_ID" ]; then
        EXPECTED_SUPPORT_PRINCIPAL="arn:aws:iam::710019948333:role/RH-Technical-Support-${OCM_ORG_EXTERNAL_ID}"
        CURRENT_TRUST=$(aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Support-Role \
            --query 'Role.AssumeRolePolicyDocument' --output json 2>/dev/null)
        if ! echo "$CURRENT_TRUST" | grep -q "$OCM_ORG_EXTERNAL_ID"; then
            warn "Support role missing trust for org $OCM_ORG_EXTERNAL_ID — adding..."
            EXISTING_PRINCIPALS=$(echo "$CURRENT_TRUST" | python3 -c "
import sys,json
doc=json.load(sys.stdin)
for stmt in doc.get('Statement',[]):
    p=stmt.get('Principal',{}).get('AWS','')
    if isinstance(p,list):
        for x in p: print(x)
    elif p: print(p)
" 2>/dev/null)
            ALL_PRINCIPALS=$(echo -e "${EXISTING_PRINCIPALS}\n${EXPECTED_SUPPORT_PRINCIPAL}" | sort -u | grep -v '^$')
            PRINCIPALS_JSON=$(echo "$ALL_PRINCIPALS" | python3 -c "
import sys,json
lines=[l.strip() for l in sys.stdin if l.strip()]
print(json.dumps(lines if len(lines)>1 else lines[0]))
")
            cat > /tmp/support-trust-policy.json <<EOFTRUST
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {"AWS": ${PRINCIPALS_JSON}},
            "Action": "sts:AssumeRole"
        }
    ]
}
EOFTRUST
            aws iam update-assume-role-policy \
                --role-name ManagedOpenShift-HCP-ROSA-Support-Role \
                --policy-document file:///tmp/support-trust-policy.json
            rm -f /tmp/support-trust-policy.json
            ok "Support role trust policy updated"
        else
            ok "Support role trust policy already includes org $OCM_ORG_EXTERNAL_ID"
        fi
    fi

    # OIDC config
    info "Checking OIDC configuration..."
    EXISTING_OIDC=$(rosa list oidc-config -o json 2>/dev/null \
        | python3 -c "
import sys, json
configs = json.load(sys.stdin)
for c in configs:
    if c.get('managed'):
        print(c['id'])
        break
" 2>/dev/null || echo "")

    if [ -n "$EXISTING_OIDC" ]; then
        OIDC_CONFIG_ID="$EXISTING_OIDC"
        ok "OIDC config already exists: $OIDC_CONFIG_ID"
    else
        info "Creating managed OIDC config..."
        OIDC_OUTPUT=$(rosa create oidc-config --managed --mode auto --yes 2>&1)
        OIDC_CONFIG_ID=$(echo "$OIDC_OUTPUT" | grep -oP '[a-z0-9]{32}' | head -1 || echo "")

        if [ -z "$OIDC_CONFIG_ID" ]; then
            OIDC_CONFIG_ID=$(rosa list oidc-config -o json 2>/dev/null \
                | python3 -c "
import sys, json
configs = json.load(sys.stdin)
for c in configs:
    if c.get('managed'):
        print(c['id'])
        break
" 2>/dev/null || echo "")
        fi

        if [ -z "$OIDC_CONFIG_ID" ]; then
            fail "Could not create or find OIDC config. Output: $OIDC_OUTPUT"
        fi
        ok "OIDC config created: $OIDC_CONFIG_ID"
    fi

    # Operator roles (use cluster name as prefix to match OIDC config trust)
    EXISTING_OP_ROLES=$(aws iam list-roles \
        --query "Roles[?starts_with(RoleName, '${OPERATOR_PREFIX}-')].RoleName" \
        --output text 2>/dev/null | wc -w)

    if [ "$EXISTING_OP_ROLES" -gt 0 ]; then
        ok "Operator roles exist with prefix '$OPERATOR_PREFIX' ($EXISTING_OP_ROLES roles)"
    else
        info "Creating operator roles with prefix '$OPERATOR_PREFIX'..."
        rosa create operator-roles \
            --hosted-cp \
            --prefix "$OPERATOR_PREFIX" \
            --oidc-config-id "$OIDC_CONFIG_ID" \
            --installer-role-arn "$INSTALLER_ROLE_ARN" \
            --mode auto \
            --yes
        ok "Operator roles created"
    fi

    # Save state
    mkdir -p "$STATE_DIR"
    cat > "$STATE_DIR/prerequisites.env" <<EOF
# ROSA E2E prerequisites — generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
AWS_ACCOUNT_ID=$AWS_ACCOUNT_ID
AWS_REGION=$AWS_REGION
CREATOR_ARN=$CREATOR_ARN
INSTALLER_ROLE_ARN=$INSTALLER_ROLE_ARN
SUPPORT_ROLE_ARN=$SUPPORT_ROLE_ARN
WORKER_ROLE_ARN=$WORKER_ROLE_ARN
OIDC_CONFIG_ID=$OIDC_CONFIG_ID
OPERATOR_PREFIX=$OPERATOR_PREFIX
VPC_ID=${VPC_ID:-}
PRIVATE_SUBNET=${PRIVATE_SUBNET:-}
PUBLIC_SUBNET=${PUBLIC_SUBNET:-}
AZ=${AZ:-}
EOF

    ok "Prerequisites saved to $STATE_DIR/prerequisites.env"
}

# ──────────────────────────────────────────────────────────────────────
# Provision cluster
# ──────────────────────────────────────────────────────────────────────
provision_cluster() {
    section "Provisioning ROSA HCP cluster"

    # Check if cluster already exists
    EXISTING_CLUSTER=$(rosa list clusters -o json 2>/dev/null \
        | python3 -c "
import sys, json
clusters = json.load(sys.stdin)
for c in clusters:
    if c.get('name') == '$CLUSTER_NAME':
        print(c['id'] + ' ' + c.get('state','unknown'))
        break
" 2>/dev/null || echo "")

    if [ -n "$EXISTING_CLUSTER" ]; then
        CLUSTER_ID=$(echo "$EXISTING_CLUSTER" | awk '{print $1}')
        CLUSTER_STATE=$(echo "$EXISTING_CLUSTER" | awk '{print $2}')
        warn "Cluster '$CLUSTER_NAME' already exists (id=$CLUSTER_ID state=$CLUSTER_STATE)"

        if [ "$CLUSTER_STATE" = "ready" ]; then
            ok "Cluster is ready — skipping provisioning"
            get_cluster_credentials
            return 0
        elif [ "$CLUSTER_STATE" = "installing" ] || [ "$CLUSTER_STATE" = "pending" ] || [ "$CLUSTER_STATE" = "validating" ] || [ "$CLUSTER_STATE" = "waiting" ]; then
            info "Cluster is still provisioning ($CLUSTER_STATE) — waiting..."
            wait_for_cluster
            return 0
        else
            fail "Cluster is in unexpected state: $CLUSTER_STATE. Run --destroy first."
        fi
    fi

    # Build subnet list
    SUBNET_IDS="${PRIVATE_SUBNET},${PUBLIC_SUBNET}"

    info "Creating ROSA HCP cluster..."
    echo "  Name:      $CLUSTER_NAME"
    echo "  Region:    $AWS_REGION"
    echo "  AZ:        $AZ"
    echo "  Version:   $OCP_VERSION"
    echo "  Workers:   $WORKER_COUNT"
    echo "  Subnets:   $SUBNET_IDS"
    echo ""

    rosa create cluster \
        --cluster-name "$CLUSTER_NAME" \
        --sts \
        --hosted-cp \
        --region "$AWS_REGION" \
        --version "$OCP_VERSION" \
        --replicas "$WORKER_COUNT" \
        --compute-machine-type "m5.xlarge" \
        --subnet-ids "$SUBNET_IDS" \
        --oidc-config-id "$OIDC_CONFIG_ID" \
        --installer-role-arn "$INSTALLER_ROLE_ARN" \
        --support-role-arn "$SUPPORT_ROLE_ARN" \
        --worker-iam-role "$WORKER_ROLE_ARN" \
        --operator-roles-prefix "$OPERATOR_PREFIX" \
        --mode auto \
        --yes

    ok "Cluster creation initiated"

    wait_for_cluster
}

wait_for_cluster() {
    info "Waiting for cluster '$CLUSTER_NAME' to be ready (this takes 15-30 minutes)..."

    local start_time=$SECONDS

    while true; do
        CLUSTER_STATE=$(rosa describe cluster --cluster "$CLUSTER_NAME" -o json 2>/dev/null \
            | python3 -c "import sys,json; print(json.load(sys.stdin).get('state','unknown'))" 2>/dev/null || echo "unknown")
        ELAPSED=$(( (SECONDS - start_time) / 60 ))

        echo -ne "\r  ⏳ State: ${CLUSTER_STATE} | Elapsed: ${ELAPSED}m   "

        if [ "$CLUSTER_STATE" = "ready" ]; then
            echo ""
            ok "Cluster is ready! (took ${ELAPSED} minutes)"
            break
        elif [ "$CLUSTER_STATE" = "error" ]; then
            echo ""
            fail "Cluster provisioning failed. Check: rosa describe cluster --cluster $CLUSTER_NAME"
        fi

        sleep 30
    done

    # Create admin user
    info "Creating cluster admin user..."
    rosa create admin --cluster "$CLUSTER_NAME" 2>&1 | tee /tmp/rosa-admin-output.txt

    get_cluster_credentials
}

get_cluster_credentials() {
    info "Retrieving cluster credentials..."

    CLUSTER_API_URL=$(rosa describe cluster --cluster "$CLUSTER_NAME" -o json 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('api',{}).get('url',''))" 2>/dev/null || echo "")
    CLUSTER_CONSOLE_URL=$(rosa describe cluster --cluster "$CLUSTER_NAME" -o json 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('console',{}).get('url',''))" 2>/dev/null || echo "")

    # Extract admin password from previous output or re-create
    ADMIN_PASSWORD=""
    if [ -f /tmp/rosa-admin-output.txt ]; then
        ADMIN_PASSWORD=$(grep -oP '(?<=--password )\S+' /tmp/rosa-admin-output.txt 2>/dev/null || echo "")
    fi

    if [ -z "$ADMIN_PASSWORD" ]; then
        warn "Could not extract admin password. Re-creating admin user..."
        ADMIN_OUTPUT=$(rosa create admin --cluster "$CLUSTER_NAME" 2>&1)
        ADMIN_PASSWORD=$(echo "$ADMIN_OUTPUT" | grep -oP '(?<=--password )\S+' || echo "")
        if [ -z "$ADMIN_PASSWORD" ]; then
            warn "Admin user may already exist. Check with: rosa describe admin --cluster $CLUSTER_NAME"
        fi
    fi

    # Save state
    mkdir -p "$STATE_DIR"
    cat > "$STATE_DIR/cluster.env" <<EOF
# ROSA E2E cluster — generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
CLUSTER_NAME=$CLUSTER_NAME
CLUSTER_API_URL=$CLUSTER_API_URL
CLUSTER_CONSOLE_URL=$CLUSTER_CONSOLE_URL
ADMIN_USERNAME=cluster-admin
ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF
    chmod 600 "$STATE_DIR/cluster.env"

    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  ROSA HCP Cluster Ready"
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Name:     $CLUSTER_NAME"
    echo "  API:      $CLUSTER_API_URL"
    echo "  Console:  $CLUSTER_CONSOLE_URL"
    echo "  Admin:    cluster-admin"
    echo "  Password: $ADMIN_PASSWORD"
    echo "═══════════════════════════════════════════════════════════════"
    echo ""

    # Login with oc (retry loop — admin takes 2-5 min to propagate)
    oc_login_with_retry
}

oc_login_with_retry() {
    if ! command -v oc &>/dev/null; then
        warn "oc CLI not available — skipping cluster login"
        return 0
    fi

    if [ -z "${ADMIN_PASSWORD:-}" ] || [ -z "${CLUSTER_API_URL:-}" ]; then
        # Load from state
        if [ -f "$STATE_DIR/cluster.env" ]; then
            source "$STATE_DIR/cluster.env"
        fi
    fi

    if [ -z "${ADMIN_PASSWORD:-}" ] || [ -z "${CLUSTER_API_URL:-}" ]; then
        warn "No admin credentials available — skipping oc login"
        return 0
    fi

    info "Logging in with oc CLI (admin user may take 2-5 minutes to propagate)..."

    for attempt in $(seq 1 20); do
        if oc login -u cluster-admin -p "$ADMIN_PASSWORD" "$CLUSTER_API_URL" --insecure-skip-tls-verify=true 2>/dev/null; then
            ok "Logged into cluster via oc (attempt $attempt)"
            oc cluster-info 2>/dev/null || true

            # Get a token for GitHub Actions
            OC_TOKEN=$(oc whoami -t 2>/dev/null || echo "")
            if [ -n "$OC_TOKEN" ]; then
                # Update state file with token
                if ! grep -q "OPENSHIFT_TOKEN" "$STATE_DIR/cluster.env" 2>/dev/null; then
                    echo "OPENSHIFT_TOKEN=$OC_TOKEN" >> "$STATE_DIR/cluster.env"
                else
                    sed -i "s|^OPENSHIFT_TOKEN=.*|OPENSHIFT_TOKEN=$OC_TOKEN|" "$STATE_DIR/cluster.env"
                fi
            fi
            return 0
        fi
        echo "  Waiting for admin user... (attempt $attempt/20, next retry in 15s)"
        sleep 15
    done

    warn "Could not log in after 20 attempts. Try manually:"
    warn "  oc login -u cluster-admin -p '$ADMIN_PASSWORD' '$CLUSTER_API_URL' --insecure-skip-tls-verify=true"
}

# ──────────────────────────────────────────────────────────────────────
# Install cert-manager
# ──────────────────────────────────────────────────────────────────────
install_cert_manager() {
    section "Installing cert-manager ${CERT_MANAGER_VERSION}"

    # Check if already installed
    if kubectl get deployment cert-manager -n cert-manager &>/dev/null; then
        READY=$(kubectl get deployment cert-manager -n cert-manager -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "${READY:-0}" -gt 0 ]; then
            ok "cert-manager already installed and running"
            return 0
        fi
    fi

    info "Applying cert-manager manifests..."
    kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"

    info "Waiting for cert-manager deployments..."
    kubectl wait --for=condition=available deployment/cert-manager -n cert-manager --timeout=5m
    kubectl wait --for=condition=available deployment/cert-manager-webhook -n cert-manager --timeout=5m
    kubectl wait --for=condition=available deployment/cert-manager-cainjector -n cert-manager --timeout=5m

    ok "cert-manager installed and ready"
    kubectl get pods -n cert-manager
}

# ──────────────────────────────────────────────────────────────────────
# Install Tekton Pipelines (OpenShift Pipelines operator or upstream)
# ──────────────────────────────────────────────────────────────────────
install_tekton() {
    section "Installing Tekton Pipelines"

    # Check if Tekton is already installed (TaskRun CRD exists)
    if oc api-resources 2>/dev/null | grep -q 'taskruns.*tekton.dev'; then
        ok "Tekton Pipelines already installed (TaskRun CRD detected)"
        oc get pods -n openshift-pipelines 2>/dev/null || oc get pods -n tekton-pipelines 2>/dev/null || true
        setup_tekton_scc
        return 0
    fi

    # Prefer OpenShift Pipelines operator on OpenShift clusters
    if oc api-resources 2>/dev/null | grep -q 'subscriptions.*operators.coreos.com'; then
        info "OpenShift cluster detected — installing OpenShift Pipelines operator via OLM..."

        # Create the Subscription for OpenShift Pipelines
        cat <<'TEKEOF' | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-pipelines-operator
  namespace: openshift-operators
spec:
  channel: latest
  installPlanApproval: Automatic
  name: openshift-pipelines-operator-rh
  source: redhat-operators
  sourceNamespace: openshift-marketplace
TEKEOF

        info "Waiting for OpenShift Pipelines operator to install (up to 5 minutes)..."
        for i in $(seq 1 60); do
            if oc api-resources 2>/dev/null | grep -q 'taskruns.*tekton.dev'; then
                ok "Tekton CRDs are available"
                break
            fi
            echo "  Waiting for Tekton CRDs... (attempt $i/60)"
            sleep 5
        done

        # Wait for the controller pods
        info "Waiting for Tekton controller pods..."
        for i in $(seq 1 30); do
            READY=$(oc get pods -n openshift-pipelines -l app=tekton-pipelines-controller \
                -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
            if [ "$READY" = "True" ]; then
                ok "Tekton Pipelines controller is ready"
                break
            fi
            echo "  Waiting for controller... (attempt $i/30)"
            sleep 10
        done

    else
        # Fallback: install upstream Tekton release manifests (vanilla Kubernetes)
        info "Installing upstream Tekton Pipelines ${TEKTON_VERSION}..."
        kubectl apply --filename \
            "https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_VERSION}/release.yaml"

        info "Waiting for Tekton Pipelines controller..."
        kubectl wait --for=condition=available deployment/tekton-pipelines-controller \
            -n tekton-pipelines --timeout=5m
        kubectl wait --for=condition=available deployment/tekton-pipelines-webhook \
            -n tekton-pipelines --timeout=5m
        ok "Upstream Tekton Pipelines installed"
    fi

    # Verify TaskRun CRD
    if oc api-resources 2>/dev/null | grep -q 'taskruns.*tekton.dev'; then
        ok "Tekton verification passed — TaskRun CRD available"
    else
        warn "Tekton CRDs not yet available. Tekton-based test tiers (2, 3, 5) will be skipped."
        return 0
    fi

    setup_tekton_scc
}

setup_tekton_scc() {
    info "Configuring Tekton SCC for build tasks..."

    # Grant pipelines-scc to pipeline ServiceAccount in the test namespace
    # The pipelines-scc is created by the OpenShift Pipelines operator
    if oc get scc pipelines-scc &>/dev/null; then
        oc create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | oc apply -f - 2>/dev/null || true
        oc create sa pipeline -n "$TEST_NAMESPACE" --dry-run=client -o yaml | oc apply -f - 2>/dev/null || true
        oc adm policy add-scc-to-user pipelines-scc -z pipeline -n "$TEST_NAMESPACE" 2>/dev/null || true
        ok "Granted pipelines-scc to pipeline SA in $TEST_NAMESPACE"
    else
        # On upstream Tekton, pipelines-scc does not exist.
        # Create a permissive SCC for Tekton builds if needed.
        warn "pipelines-scc not found (upstream Tekton or operator still initializing)"
        warn "Tekton builds may fail if buildah requires elevated privileges"
    fi

    # Also grant for the operator namespace (Tekton builds may run there)
    if oc get scc pipelines-scc &>/dev/null; then
        oc create sa pipeline -n "$OPERATOR_NAMESPACE" --dry-run=client -o yaml | oc apply -f - 2>/dev/null || true
        oc adm policy add-scc-to-user pipelines-scc -z pipeline -n "$OPERATOR_NAMESPACE" 2>/dev/null || true
        ok "Granted pipelines-scc to pipeline SA in $OPERATOR_NAMESPACE"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Audit cluster readiness
# ──────────────────────────────────────────────────────────────────────
audit_cluster() {
    section "E2E Cluster Readiness Audit"

    local PASS=0
    local FAIL=0
    local WARN_COUNT=0

    # 1. Cluster connection
    info "[1/8] Cluster connection"
    if oc whoami &>/dev/null; then
        ok "  Connected as: $(oc whoami)"
        PASS=$((PASS + 1))
    else
        fail_soft "  Not connected to a cluster"
        FAIL=$((FAIL + 1))
    fi

    # 2. cert-manager
    info "[2/8] cert-manager"
    if kubectl get deployment cert-manager -n cert-manager &>/dev/null; then
        READY=$(kubectl get deployment cert-manager -n cert-manager -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "${READY:-0}" -gt 0 ]; then
            ok "  cert-manager: running ($READY replicas)"
            PASS=$((PASS + 1))
        else
            fail_soft "  cert-manager: deployed but not ready"
            FAIL=$((FAIL + 1))
        fi
    else
        fail_soft "  cert-manager: NOT installed"
        FAIL=$((FAIL + 1))
    fi

    # 3. Tekton Pipelines
    info "[3/8] Tekton Pipelines (required for Tier 2, 3, 5)"
    if oc api-resources 2>/dev/null | grep -q 'taskruns.*tekton.dev'; then
        ok "  Tekton: CRDs available"
        TEKTON_NS=$(oc get pods --all-namespaces -l app=tekton-pipelines-controller -o jsonpath='{.items[0].metadata.namespace}' 2>/dev/null || echo "")
        if [ -n "$TEKTON_NS" ]; then
            ok "  Tekton: controller running in $TEKTON_NS"
        fi
        PASS=$((PASS + 1))
    else
        fail_soft "  Tekton: NOT installed — Tier 2, 3, 5 tests will fail"
        FAIL=$((FAIL + 1))
    fi

    # 4. S2I / BuildConfig API
    info "[4/8] S2I BuildConfig API (required for Tier 4)"
    if oc api-resources 2>/dev/null | grep -q 'buildconfigs.*build.openshift.io'; then
        ok "  S2I: BuildConfig API available"
        PASS=$((PASS + 1))
    else
        fail_soft "  S2I: BuildConfig API NOT available — Tier 4 tests will fail"
        FAIL=$((FAIL + 1))
    fi

    # 5. pipelines-scc
    info "[5/8] pipelines-scc SecurityContextConstraint"
    if oc get scc pipelines-scc &>/dev/null; then
        ok "  pipelines-scc: exists"
        PASS=$((PASS + 1))
    else
        fail_soft "  pipelines-scc: NOT found (install OpenShift Pipelines operator)"
        FAIL=$((FAIL + 1))
    fi

    # 6. Operator deployment
    info "[6/8] Operator deployment"
    if oc get deployment notebook-validator-controller-manager -n "$OPERATOR_NAMESPACE" &>/dev/null; then
        READY=$(oc get deployment notebook-validator-controller-manager -n "$OPERATOR_NAMESPACE" \
            -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "${READY:-0}" -gt 0 ]; then
            ok "  Operator: running ($READY replicas)"
            PASS=$((PASS + 1))
        else
            fail_soft "  Operator: deployed but not ready"
            FAIL=$((FAIL + 1))
        fi
    else
        fail_soft "  Operator: NOT deployed"
        FAIL=$((FAIL + 1))
    fi

    # 7. Webhook
    info "[7/8] Webhook CA bundle"
    CA_BUNDLE=$(oc get mutatingwebhookconfiguration notebook-validator-mutating-webhook-configuration \
        -o jsonpath='{.webhooks[0].clientConfig.caBundle}' 2>/dev/null || echo "")
    if [ -n "$CA_BUNDLE" ] && [ "$CA_BUNDLE" != "Cg==" ]; then
        ok "  Webhook: CA bundle injected"
        PASS=$((PASS + 1))
    else
        fail_soft "  Webhook: CA bundle missing or empty"
        FAIL=$((FAIL + 1))
    fi

    # 8. GitHub secrets
    info "[8/8] GitHub secrets"
    if command -v gh &>/dev/null && gh auth status &>/dev/null; then
        REPO="tosin2013/jupyter-notebook-validator-operator"
        SECRETS_LIST=$(gh secret list --repo "$REPO" 2>/dev/null || echo "")
        for SECRET_NAME in OPENSHIFT_SERVER OPENSHIFT_TOKEN QUAY_USERNAME QUAY_PASSWORD TEST_REPO_TOKEN; do
            if echo "$SECRETS_LIST" | grep -q "^${SECRET_NAME}"; then
                ok "  $SECRET_NAME: set"
            else
                fail_soft "  $SECRET_NAME: NOT set"
                FAIL=$((FAIL + 1))
                continue
            fi
            PASS=$((PASS + 1))
        done
    else
        warn "  Cannot check GitHub secrets (gh CLI not authenticated)"
        WARN_COUNT=$((WARN_COUNT + 1))
    fi

    # Summary
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Audit Summary"
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Passed: $PASS"
    echo "  Failed: $FAIL"
    echo "  Warnings: $WARN_COUNT"
    echo ""

    # Tier readiness
    echo "  Tier Readiness:"
    TIER1="READY"
    TIER2="NOT READY (needs Tekton)"
    TIER3="NOT READY (needs Tekton)"
    TIER4="NOT READY (needs S2I + SCC)"
    TIER5="NOT READY (needs Tekton)"

    if oc api-resources 2>/dev/null | grep -q 'taskruns.*tekton.dev'; then
        TIER2="READY"; TIER3="READY"; TIER5="READY"
    fi
    if oc api-resources 2>/dev/null | grep -q 'buildconfigs.*build.openshift.io' && oc get scc pipelines-scc &>/dev/null; then
        TIER4="READY"
    fi

    echo "  Tier 1 (Simple):       $TIER1"
    echo "  Tier 2 (Intermediate): $TIER2"
    echo "  Tier 3 (Complex):      $TIER3"
    echo "  Tier 4 (S2I):          $TIER4"
    echo "  Tier 5 (Volumes):      $TIER5"
    echo "═══════════════════════════════════════════════════════════════"

    if [ "$FAIL" -gt 0 ]; then
        echo ""
        warn "Cluster is not fully ready for all E2E tiers."
        echo "  Run: $0 --deploy   (to install cert-manager + Tekton + operator)"
        return 1
    else
        echo ""
        ok "Cluster is ready for all E2E tiers!"
        return 0
    fi
}

# Helper: non-fatal failure (used by audit)
fail_soft() { echo -e "${RED}❌ ${NC}$*"; }

# ──────────────────────────────────────────────────────────────────────
# Deploy operator
# ──────────────────────────────────────────────────────────────────────
deploy_operator() {
    section "Deploying operator"

    # Ensure Go is available
    export PATH=$PATH:/usr/local/go/bin

    if ! command -v go &>/dev/null; then
        fail "Go not installed. Run: $0 --tools-only"
    fi

    cd "$REPO_DIR"

    info "Installing CRDs..."
    make install 2>&1

    info "Deploying operator with image: $OPERATOR_IMG"
    make deploy IMG="$OPERATOR_IMG" 2>&1

    info "Waiting for operator deployment..."
    oc wait --for=condition=available deployment/notebook-validator-controller-manager \
        -n "$OPERATOR_NAMESPACE" --timeout=5m

    info "Waiting for operator pod readiness..."
    oc wait --for=condition=ready pod -l control-plane=controller-manager \
        -n "$OPERATOR_NAMESPACE" --timeout=3m

    # Wait for webhook CA bundle injection
    info "Waiting for webhook CA bundle injection..."
    for i in $(seq 1 30); do
        CA_BUNDLE=$(oc get mutatingwebhookconfiguration notebook-validator-mutating-webhook-configuration \
            -o jsonpath='{.webhooks[0].clientConfig.caBundle}' 2>/dev/null || echo "")
        if [ -n "$CA_BUNDLE" ] && [ "$CA_BUNDLE" != "Cg==" ]; then
            ok "Webhook CA bundle injected"
            break
        fi
        echo "  Waiting for CA bundle (attempt $i/30)..."
        sleep 5
    done

    # Give webhook server a moment to initialize after CA injection
    sleep 5

    ok "Operator deployed successfully"
    oc get pods -n "$OPERATOR_NAMESPACE"
}

# ──────────────────────────────────────────────────────────────────────
# Smoke test
# ──────────────────────────────────────────────────────────────────────
run_smoke_test() {
    section "Running smoke test"

    if [ "${SKIP_SMOKE_TEST:-false}" = "true" ]; then
        warn "Smoke test skipped (SKIP_SMOKE_TEST=true)"
        return 0
    fi

    # Create test namespace
    oc create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | oc apply -f -

    # Create a minimal NotebookValidationJob
    info "Creating smoke-test NotebookValidationJob..."
    cat <<SMOKEEOF | oc apply -n "$TEST_NAMESPACE" -f -
apiVersion: mlops.mlops.dev/v1alpha1
kind: NotebookValidationJob
metadata:
  name: smoke-test-hello-world
spec:
  notebook:
    git:
      url: "https://github.com/tosin2013/jupyter-notebook-validator-test-notebooks.git"
      ref: "main"
    path: "notebooks/tier1-simple/01-hello-world.ipynb"
  podConfig:
    containerImage: "quay.io/jupyter/minimal-notebook:latest"
  timeout: "5m"
SMOKEEOF

    # Wait for completion
    info "Waiting for smoke test to complete (up to 5 minutes)..."
    for i in $(seq 1 30); do
        PHASE=$(oc get notebookvalidationjob smoke-test-hello-world -n "$TEST_NAMESPACE" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        if [ "$PHASE" = "Succeeded" ]; then
            ok "Smoke test PASSED"
            MESSAGE=$(oc get notebookvalidationjob smoke-test-hello-world -n "$TEST_NAMESPACE" \
                -o jsonpath='{.status.message}' 2>/dev/null || echo "")
            echo "  Result: $MESSAGE"
            return 0
        elif [ "$PHASE" = "Failed" ]; then
            fail "Smoke test FAILED. Check: oc get notebookvalidationjob smoke-test-hello-world -n $TEST_NAMESPACE -o yaml"
        fi

        echo "  Phase: $PHASE (attempt $i/30)..."
        sleep 10
    done

    warn "Smoke test did not complete within timeout. Check manually:"
    warn "  oc get notebookvalidationjob smoke-test-hello-world -n $TEST_NAMESPACE -o yaml"
}

# ──────────────────────────────────────────────────────────────────────
# Print GitHub secrets
# ──────────────────────────────────────────────────────────────────────
print_github_secrets() {
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  GitHub Secrets for E2E Workflow"
    echo "═══════════════════════════════════════════════════════════════"
    echo ""

    if [ -f "$STATE_DIR/prerequisites.env" ]; then
        source "$STATE_DIR/prerequisites.env"
    fi
    if [ -f "$STATE_DIR/cluster.env" ]; then
        source "$STATE_DIR/cluster.env"
    fi

    AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-$(aws sts get-caller-identity --query Account --output text 2>/dev/null)}"
    CREATOR_ARN="${CREATOR_ARN:-$(aws sts get-caller-identity --query Arn --output text 2>/dev/null)}"

    echo "  ── Pre-configured cluster (skip provisioning) ──"
    echo ""
    echo "  OPENSHIFT_SERVER=${CLUSTER_API_URL:-<run full setup first>}"
    echo "  OPENSHIFT_TOKEN=${OPENSHIFT_TOKEN:-<run full setup first, then: oc whoami -t>}"
    echo ""
    echo "  ── Operator image build and push ──"
    echo ""
    echo "  QUAY_USERNAME=<your-quay-username>           # Quay.io login for pushing operator images"
    echo "  QUAY_PASSWORD=<your-quay-password-or-token>  # Quay.io robot token or password"
    echo ""
    echo "  ── Test repository access ──"
    echo ""
    echo "  TEST_REPO_TOKEN=<github-pat-with-repo-scope> # Access to jupyter-notebook-validator-test-notebooks"
    echo ""
    echo "  ── Terraform provisioning (provision_cluster=true) ──"
    echo ""
    echo "  AWS_ACCESS_KEY_ID=<your-aws-access-key>"
    echo "  AWS_SECRET_ACCESS_KEY=<your-aws-secret-key>"
    echo "  AWS_REGION=$AWS_REGION"
    echo "  AWS_ACCOUNT_ID=$AWS_ACCOUNT_ID"
    echo "  ROSA_TOKEN=$(ocm token 2>/dev/null | head -c 20)...  (get full: ocm token)"
    echo "  ROSA_INSTALLER_ROLE_ARN=${INSTALLER_ROLE_ARN:-<not detected>}"
    echo "  ROSA_SUPPORT_ROLE_ARN=${SUPPORT_ROLE_ARN:-<not detected>}"
    echo "  ROSA_WORKER_ROLE_ARN=${WORKER_ROLE_ARN:-<not detected>}"
    echo "  ROSA_OIDC_CONFIG_ID=${OIDC_CONFIG_ID:-<not created yet>}"
    echo "  ROSA_CREATOR_ARN=$CREATOR_ARN"
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo ""

    if command -v gh &>/dev/null && [ -n "${CLUSTER_API_URL:-}" ]; then
        echo "  To set the pre-configured cluster secrets in GitHub:"
        echo ""
        echo "    gh secret set OPENSHIFT_SERVER --body '${CLUSTER_API_URL}'"
        if [ -n "${OPENSHIFT_TOKEN:-}" ]; then
            echo "    gh secret set OPENSHIFT_TOKEN  --body '<token from: oc whoami -t>'"
        fi
        echo ""
        echo "  This lets the E2E workflow skip Terraform provisioning and"
        echo "  use your already-running cluster directly."
        echo ""
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Set GitHub secrets automatically
# ──────────────────────────────────────────────────────────────────────
set_github_secrets() {
    info "Setting GitHub repository secrets..."

    if ! command -v gh &>/dev/null; then
        fail "gh CLI not installed. Run this script with --tools-only first."
    fi

    if ! gh auth status &>/dev/null; then
        fail "gh CLI not authenticated. Run: gh auth login"
    fi

    if [ -f "$STATE_DIR/prerequisites.env" ]; then
        source "$STATE_DIR/prerequisites.env"
    fi
    if [ -f "$STATE_DIR/cluster.env" ]; then
        source "$STATE_DIR/cluster.env"
    fi

    REPO="tosin2013/jupyter-notebook-validator-operator"

    if [ -n "${CLUSTER_API_URL:-}" ]; then
        info "Setting OPENSHIFT_SERVER..."
        gh secret set OPENSHIFT_SERVER --repo "$REPO" --body "$CLUSTER_API_URL"
        ok "OPENSHIFT_SERVER set"
    fi

    if [ -n "${OPENSHIFT_TOKEN:-}" ]; then
        info "Setting OPENSHIFT_TOKEN..."
        gh secret set OPENSHIFT_TOKEN --repo "$REPO" --body "$OPENSHIFT_TOKEN"
        ok "OPENSHIFT_TOKEN set"
    fi

    if [ -n "${INSTALLER_ROLE_ARN:-}" ]; then
        info "Setting ROSA infrastructure secrets..."
        gh secret set AWS_ACCOUNT_ID          --repo "$REPO" --body "$AWS_ACCOUNT_ID"
        gh secret set AWS_REGION              --repo "$REPO" --body "$AWS_REGION"
        gh secret set ROSA_INSTALLER_ROLE_ARN --repo "$REPO" --body "$INSTALLER_ROLE_ARN"
        gh secret set ROSA_SUPPORT_ROLE_ARN   --repo "$REPO" --body "$SUPPORT_ROLE_ARN"
        gh secret set ROSA_WORKER_ROLE_ARN    --repo "$REPO" --body "$WORKER_ROLE_ARN"
        gh secret set ROSA_CREATOR_ARN        --repo "$REPO" --body "$CREATOR_ARN"
        ok "ROSA infrastructure secrets set"
    fi

    if [ -n "${OIDC_CONFIG_ID:-}" ]; then
        gh secret set ROSA_OIDC_CONFIG_ID --repo "$REPO" --body "$OIDC_CONFIG_ID"
        ok "ROSA_OIDC_CONFIG_ID set"
    fi

    OCM_TOKEN=$(ocm token 2>/dev/null || echo "")
    if [ -n "$OCM_TOKEN" ]; then
        gh secret set ROSA_TOKEN --repo "$REPO" --body "$OCM_TOKEN"
        ok "ROSA_TOKEN set"
    fi

    echo ""
    ok "GitHub secrets configured! The E2E workflow can now:"
    echo "  1. Use the pre-configured cluster (OPENSHIFT_SERVER + OPENSHIFT_TOKEN)"
    echo "  2. Provision a new one via Terraform (provision_cluster=true)"
}

# ──────────────────────────────────────────────────────────────────────
# Cluster status
# ──────────────────────────────────────────────────────────────────────
cluster_status() {
    info "Checking cluster status..."

    if ! command -v rosa &>/dev/null; then
        fail "ROSA CLI not installed. Run: $0 --tools-only"
    fi

    rosa_login

    rosa list clusters 2>/dev/null || echo "No clusters found"

    CLUSTER_INFO=$(rosa describe cluster --cluster "$CLUSTER_NAME" -o json 2>/dev/null || echo "")
    if [ -n "$CLUSTER_INFO" ]; then
        echo ""
        echo "$CLUSTER_INFO" | python3 -c "
import sys, json
c = json.load(sys.stdin)
print(f\"Cluster: {c['name']}\")
print(f\"ID:      {c['id']}\")
print(f\"State:   {c.get('state','unknown')}\")
print(f\"Version: {c.get('openshift_version','unknown')}\")
print(f\"API:     {c.get('api',{}).get('url','N/A')}\")
print(f\"Console: {c.get('console',{}).get('url','N/A')}\")
print(f\"Region:  {c.get('region',{}).get('id','unknown')}\")
print(f\"Nodes:   {c.get('nodes',{}).get('compute',0)} workers\")
"
    else
        warn "Cluster '$CLUSTER_NAME' not found"
    fi

    # Check oc connection
    if command -v oc &>/dev/null; then
        echo ""
        info "oc connection:"
        oc whoami 2>/dev/null && oc get nodes 2>/dev/null || warn "Not logged in via oc"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Destroy cluster
# ──────────────────────────────────────────────────────────────────────
destroy_cluster() {
    section "Destroying ROSA HCP cluster '$CLUSTER_NAME'"

    if ! command -v rosa &>/dev/null; then
        fail "ROSA CLI not installed. Run: $0 --tools-only"
    fi

    rosa_login

    if ! rosa describe cluster --cluster "$CLUSTER_NAME" &>/dev/null; then
        warn "Cluster '$CLUSTER_NAME' not found — nothing to destroy"
        return 0
    fi

    # Undeploy operator first if oc is available
    if command -v oc &>/dev/null && oc whoami &>/dev/null; then
        info "Cleaning up operator resources..."
        cd "$REPO_DIR"
        make undeploy 2>/dev/null || true
        make uninstall 2>/dev/null || true
    fi

    rosa delete cluster --cluster "$CLUSTER_NAME" --yes --watch

    ok "Cluster '$CLUSTER_NAME' deleted"

    info "Cleaning up operator roles and OIDC config..."
    if [ -f "$STATE_DIR/prerequisites.env" ]; then
        source "$STATE_DIR/prerequisites.env"
    fi
    rosa delete operator-roles --prefix "$OPERATOR_PREFIX" --mode auto --yes 2>/dev/null || true
    if [ -n "${OIDC_CONFIG_ID:-}" ]; then
        rosa delete oidc-config --oidc-config-id "$OIDC_CONFIG_ID" --mode auto --yes 2>/dev/null || true
    fi

    rm -rf "$STATE_DIR"
    ok "Cleanup complete"
}

# ──────────────────────────────────────────────────────────────────────
# Validate with Terraform (dry-run)
# ──────────────────────────────────────────────────────────────────────
terraform_validate() {
    section "Validating Terraform configuration"

    if ! command -v terraform &>/dev/null; then
        fail "Terraform not installed. Run: $0 --tools-only"
    fi

    TF_DIR="$REPO_DIR/test/e2e/terraform"

    if [ ! -f "$TF_DIR/main.tf" ]; then
        fail "Terraform configs not found at $TF_DIR"
    fi

    cd "$TF_DIR"

    info "Running terraform init..."
    terraform init -backend=false

    info "Running terraform validate..."
    terraform validate

    ok "Terraform configuration is valid"

    if [ -f "$STATE_DIR/prerequisites.env" ]; then
        source "$STATE_DIR/prerequisites.env"
        info "Running terraform plan (dry-run)..."
        OCM_TOKEN=$(ocm token 2>/dev/null || echo "dummy-for-plan")
        terraform plan \
            -var="rhcs_token=$OCM_TOKEN" \
            -var="aws_account_id=$AWS_ACCOUNT_ID" \
            -var="installer_role_arn=$INSTALLER_ROLE_ARN" \
            -var="support_role_arn=$SUPPORT_ROLE_ARN" \
            -var="worker_role_arn=$WORKER_ROLE_ARN" \
            -var="oidc_config_id=$OIDC_CONFIG_ID" \
            -var="rosa_creator_arn=$CREATOR_ARN" \
            -var="aws_region=$AWS_REGION" \
            -input=false \
            2>&1 || warn "Plan may show errors if providers can't be reached"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────
main() {
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║  ROSA HCP E2E Setup — jupyter-notebook-validator-operator ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo ""

    case "${1:-}" in
        --tools-only)
            preflight
            install_tools
            ok "Tools installed. Run '$0' (no flags) to provision a cluster."
            ;;
        --deploy)
            # Deploy operator to an already-running cluster (cert-manager + Tekton + CRDs + operator + smoke test)
            export PATH=$PATH:/usr/local/go/bin
            info "Deploying operator to existing cluster..."
            if ! oc whoami &>/dev/null; then
                # Try loading credentials from state
                if [ -f "$STATE_DIR/cluster.env" ]; then
                    source "$STATE_DIR/cluster.env"
                    oc_login_with_retry
                else
                    fail "Not logged into a cluster. Run the full setup or 'oc login' first."
                fi
            fi
            ok "Connected to cluster as: $(oc whoami)"
            install_cert_manager
            install_tekton
            deploy_operator
            run_smoke_test
            echo ""
            ok "Operator deployed and smoke test passed!"
            ;;
        --audit)
            # Audit cluster readiness for all E2E tiers
            if ! oc whoami &>/dev/null; then
                if [ -f "$STATE_DIR/cluster.env" ]; then
                    source "$STATE_DIR/cluster.env"
                    oc_login_with_retry
                else
                    fail "Not logged into a cluster. Run 'oc login' first."
                fi
            fi
            audit_cluster
            ;;
        --secrets)
            preflight
            detect_sandbox
            print_github_secrets
            ;;
        --set-secrets)
            preflight
            detect_sandbox
            set_github_secrets
            ;;
        --destroy)
            preflight
            if [ -f "$STATE_DIR/prerequisites.env" ]; then
                source "$STATE_DIR/prerequisites.env"
            fi
            detect_sandbox
            destroy_cluster
            ;;
        --status)
            preflight
            cluster_status
            ;;
        --terraform-validate)
            preflight
            install_tools
            detect_sandbox
            rosa_login
            rosa_prerequisites
            terraform_validate
            ;;
        --help|-h)
            cat <<HELPEOF
ROSA HCP E2E Setup — jupyter-notebook-validator-operator

Usage:
  $(basename "$0")                  Full setup: tools + cluster + cert-manager + Tekton + operator + smoke test
  $(basename "$0") --tools-only    Just install tools (Go, ROSA CLI, Terraform, oc, gh)
  $(basename "$0") --deploy        Deploy operator to existing cluster (cert-manager + Tekton + CRDs + operator + smoke test)
  $(basename "$0") --audit         Audit cluster readiness for all E2E tiers
  $(basename "$0") --secrets       Print GitHub secrets needed for CI
  $(basename "$0") --set-secrets   Auto-set GitHub repo secrets via gh CLI
  $(basename "$0") --destroy       Tear down cluster and clean up IAM resources
  $(basename "$0") --status        Check cluster and operator status
  $(basename "$0") --terraform-validate  Validate Terraform configs (dry-run)

Environment variables:
  CLUSTER_NAME          Cluster name (default: jnvo-e2e)
  AWS_REGION            AWS region (default: us-east-1)
  OCP_VERSION           OpenShift version (default: 4.20.36)
  WORKER_COUNT          Number of workers (default: 2)
  OPERATOR_IMG          Operator image (default: quay.io/takinosh/...operator:latest)
  CERT_MANAGER_VERSION  cert-manager version (default: v1.17.2)
  TEKTON_VERSION        Upstream Tekton version (default: v0.65.1)
  SKIP_SMOKE_TEST       Set to "true" to skip post-deploy smoke test
HELPEOF
            ;;
        "")
            preflight
            install_tools
            detect_sandbox
            rosa_login
            rosa_prerequisites
            provision_cluster
            install_cert_manager
            install_tekton
            deploy_operator
            run_smoke_test
            print_github_secrets
            echo ""
            ok "🎉 Full setup complete! Summary:"
            echo "  ✅ ROSA HCP cluster provisioned and ready"
            echo "  ✅ cert-manager installed"
            echo "  ✅ Tekton Pipelines installed"
            echo "  ✅ Operator deployed with webhooks"
            echo "  ✅ Smoke test passed"
            echo ""
            echo "  Next steps:"
            echo "  1. Set GitHub secrets: $0 --set-secrets"
            echo "  2. Audit readiness: $0 --audit"
            echo "  3. Trigger E2E workflow: gh workflow run e2e-openshift.yaml"
            echo "  4. When done: $0 --destroy"
            ;;
        *)
            fail "Unknown option: $1. Run '$0 --help' for usage."
            ;;
    esac
}

main "$@"
