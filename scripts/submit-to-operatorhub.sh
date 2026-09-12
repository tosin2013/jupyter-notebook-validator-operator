#!/usr/bin/env bash
# submit-to-operatorhub.sh — Automate OperatorHub bundle submission
#
# Usage:
#   scripts/submit-to-operatorhub.sh VERSION [OPTIONS]
#
# Options:
#   --target community-operators|community-operators-prod|both  (default: both)
#   --dry-run          Print what would be done without making changes
#   --bundle-dir DIR   Path to bundle directory (default: bundle/)
#   --help             Show this help message
#
# Prerequisites:
#   - gh CLI authenticated (gh auth status)
#   - Git configured with DCO-compliant email/name
#   - Fork remotes configured (origin = your fork, upstream = upstream repo)
#
# See docs/operations/RELEASE.md for the full release runbook.

set -euo pipefail

# --- Constants ---
OPERATOR="jupyter-notebook-validator-operator"
UPSTREAM_PROD="redhat-openshift-ecosystem/community-operators-prod"
UPSTREAM_COMMUNITY="k8s-operatorhub/community-operators"

# Required metadata values (from docs/operations/RELEASE.md)
REQUIRED_PROVIDER_NAME="Decision Crafters"
REQUIRED_MAINTAINER_EMAIL="takinosh@redhat.com"

# --- Defaults ---
TARGET="both"
DRY_RUN=false
BUNDLE_DIR="bundle"
VERSION=""

# --- Color output ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
dry()   { echo -e "${YELLOW}[DRY-RUN]${NC} $*"; }

# --- Usage ---
usage() {
    cat <<EOF
Usage: $(basename "$0") VERSION [OPTIONS]

Automate OperatorHub bundle submission for ${OPERATOR}.

Arguments:
  VERSION              Version to submit (e.g., 1.0.9)

Options:
  --target TARGET      Target repo: community-operators, community-operators-prod, or both (default: both)
  --dry-run            Print planned actions without executing
  --bundle-dir DIR     Path to bundle directory (default: bundle/)
  --help               Show this help message

Examples:
  $(basename "$0") 1.0.9 --dry-run
  $(basename "$0") 1.0.9 --target community-operators-prod
  $(basename "$0") 1.0.9 --target both

Prerequisites:
  - gh auth status must succeed
  - git user.email and user.name must be set for DCO
  - Fork repos cloned with 'upstream' remote pointing to the upstream repo
EOF
    exit 0
}

# --- Argument parsing ---
parse_args() {
    if [[ $# -lt 1 ]]; then
        error "VERSION argument is required"
        usage
    fi

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --target)
                TARGET="${2:?--target requires a value}"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --bundle-dir)
                BUNDLE_DIR="${2:?--bundle-dir requires a value}"
                shift 2
                ;;
            --help|-h)
                usage
                ;;
            -*)
                error "Unknown option: $1"
                usage
                ;;
            *)
                if [[ -z "$VERSION" ]]; then
                    VERSION="$1"
                else
                    error "Unexpected argument: $1"
                    usage
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$VERSION" ]]; then
        error "VERSION argument is required"
        usage
    fi

    case "$TARGET" in
        community-operators|community-operators-prod|both) ;;
        *)
            error "Invalid --target: $TARGET (must be community-operators, community-operators-prod, or both)"
            exit 1
            ;;
    esac
}

# --- Prerequisite checks ---
check_prerequisites() {
    info "Checking prerequisites..."
    local failed=false

    # gh CLI
    if ! command -v gh &>/dev/null; then
        error "gh CLI is not installed. Install from https://cli.github.com/"
        failed=true
    elif ! gh auth status &>/dev/null; then
        error "gh CLI is not authenticated. Run: gh auth login"
        failed=true
    else
        ok "gh CLI authenticated"
    fi

    # Git DCO config
    local git_email git_name
    git_email=$(git config user.email 2>/dev/null || true)
    git_name=$(git config user.name 2>/dev/null || true)

    if [[ -z "$git_email" ]]; then
        error "git user.email is not set. Run: git config user.email \"$REQUIRED_MAINTAINER_EMAIL\""
        failed=true
    else
        ok "git user.email = $git_email"
    fi

    if [[ -z "$git_name" ]]; then
        error "git user.name is not set. Run: git config user.name \"Tosin Akinosho\""
        failed=true
    else
        ok "git user.name = $git_name"
    fi

    # Bundle directory
    if [[ ! -d "$BUNDLE_DIR/manifests" ]]; then
        error "Bundle directory not found: $BUNDLE_DIR/manifests"
        error "Run 'make bundle' first to generate the bundle"
        failed=true
    else
        ok "Bundle directory exists: $BUNDLE_DIR"
    fi

    if [[ "$failed" == "true" ]]; then
        error "Prerequisite checks failed. Fix the issues above and retry."
        exit 1
    fi

    ok "All prerequisites satisfied"
}

# --- Metadata validation ---
validate_metadata() {
    info "Validating bundle metadata..."
    local csv="$BUNDLE_DIR/manifests/${OPERATOR}.clusterserviceversion.yaml"
    local failed=false

    if [[ ! -f "$csv" ]]; then
        error "CSV not found: $csv"
        exit 1
    fi

    # Provider name
    if grep -q "name: ${REQUIRED_PROVIDER_NAME}" "$csv"; then
        ok "Provider name: ${REQUIRED_PROVIDER_NAME}"
    else
        error "Provider name is not '${REQUIRED_PROVIDER_NAME}'"
        failed=true
    fi

    # Maintainer email
    if grep -q "email: ${REQUIRED_MAINTAINER_EMAIL}" "$csv"; then
        ok "Maintainer email: ${REQUIRED_MAINTAINER_EMAIL}"
    else
        error "Maintainer email is not '${REQUIRED_MAINTAINER_EMAIL}'"
        failed=true
    fi

    # containerImage annotation
    if grep -q "containerImage:.*${OPERATOR}" "$csv"; then
        local container_image
        container_image=$(grep "containerImage:" "$csv" | head -1 | awk '{print $2}')
        ok "containerImage: $container_image"
    else
        error "containerImage annotation is missing or does not reference ${OPERATOR}"
        failed=true
    fi

    # Icon check (non-empty base64data)
    if grep -q "base64data:" "$csv"; then
        ok "Icon base64data present"
    else
        warn "No icon base64data found — verify icon is set correctly"
    fi

    if [[ "$failed" == "true" ]]; then
        error "Metadata validation failed. Fix the CSV and retry."
        exit 1
    fi

    ok "Bundle metadata is valid"
}

# --- Check for existing PRs (one bundle per PR constraint) ---
check_existing_prs() {
    local upstream_repo="$1"
    info "Checking for existing PRs in $upstream_repo..."

    local existing_prs
    existing_prs=$(gh pr list \
        --repo "$upstream_repo" \
        --search "operator ${OPERATOR}" \
        --state open \
        --json number,title \
        --jq '.[].title' 2>/dev/null || true)

    if [[ -n "$existing_prs" ]]; then
        warn "Open PR(s) found in $upstream_repo:"
        echo "$existing_prs" | while read -r title; do
            warn "  - $title"
        done
        warn "OperatorHub enforces one bundle per PR. Ensure previous PRs are merged first."
    else
        ok "No open PRs for $OPERATOR in $upstream_repo"
    fi
}

# --- Submit to a single target repo ---
submit_to_repo() {
    local fork_dir="$1"
    local upstream_repo="$2"
    local branch="add-${OPERATOR}-${VERSION}"

    info "=== Submitting to $upstream_repo ==="

    # Check fork directory
    if [[ ! -d "$fork_dir" ]]; then
        error "Fork directory not found: $fork_dir"
        error "Clone your fork first: gh repo clone <your-fork> $fork_dir"
        return 1
    fi

    check_existing_prs "$upstream_repo"

    pushd "$fork_dir" > /dev/null

    # Check upstream remote
    if ! git remote get-url upstream &>/dev/null; then
        error "No 'upstream' remote configured in $fork_dir"
        error "Run: git remote add upstream https://github.com/$upstream_repo.git"
        popd > /dev/null
        return 1
    fi

    # Check if branch already exists (idempotency)
    if git show-ref --verify --quiet "refs/heads/$branch" 2>/dev/null; then
        warn "Branch '$branch' already exists locally"
        # Check if PR already exists
        local existing_pr
        existing_pr=$(gh pr list \
            --repo "$upstream_repo" \
            --head "$branch" \
            --json url \
            --jq '.[0].url' 2>/dev/null || true)
        if [[ -n "$existing_pr" ]]; then
            ok "PR already exists: $existing_pr"
            popd > /dev/null
            return 0
        fi
        info "No PR found for branch — will reuse existing branch"
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        dry "Would sync fork: git fetch upstream && git checkout main && git rebase upstream/main"
        dry "Would create branch: $branch"
        dry "Would copy bundle to: operators/${OPERATOR}/${VERSION}/"
        dry "Would commit with DCO sign-off: operator ${OPERATOR} (${VERSION})"
        dry "Would open PR to $upstream_repo with title: operator ${OPERATOR} (${VERSION})"
        popd > /dev/null
        return 0
    fi

    # Sync fork
    info "Syncing fork with upstream..."
    git fetch upstream
    git checkout main
    git rebase upstream/main
    git push origin main

    # Create branch (or checkout existing)
    if git show-ref --verify --quiet "refs/heads/$branch" 2>/dev/null; then
        git checkout "$branch"
    else
        git checkout -b "$branch"
    fi

    # Copy bundle
    local target_dir="operators/${OPERATOR}/${VERSION}"
    info "Copying bundle to $target_dir..."
    mkdir -p "$target_dir"

    local source_bundle
    source_bundle="$(dirs -l +1)/$BUNDLE_DIR"
    cp -r "$source_bundle/manifests" "$target_dir/"
    cp -r "$source_bundle/metadata" "$target_dir/"

    # Ensure ci.yaml exists in operator directory
    if [[ ! -f "operators/${OPERATOR}/ci.yaml" ]]; then
        local prev_version_dir
        prev_version_dir=$(ls -d "operators/${OPERATOR}/"*/ 2>/dev/null | head -1)
        if [[ -n "$prev_version_dir" && -f "${prev_version_dir}ci.yaml" ]]; then
            cp "${prev_version_dir}ci.yaml" "operators/${OPERATOR}/ci.yaml"
            info "Copied ci.yaml from previous version"
        fi
    fi

    # Commit with DCO sign-off
    git add "operators/${OPERATOR}/"
    git commit -s -m "operator ${OPERATOR} (${VERSION})"

    # Push
    info "Pushing branch..."
    git push origin "$branch"

    # Open PR
    info "Opening PR..."
    local pr_url
    pr_url=$(gh pr create \
        --repo "$upstream_repo" \
        --title "operator ${OPERATOR} (${VERSION})" \
        --body "Adding ${OPERATOR} version ${VERSION}." \
        --head "$branch" 2>&1)

    ok "PR created: $pr_url"

    popd > /dev/null
}

# --- Main ---
main() {
    parse_args "$@"

    echo ""
    info "OperatorHub Submission Script"
    info "Operator: $OPERATOR"
    info "Version:  $VERSION"
    info "Target:   $TARGET"
    info "Dry run:  $DRY_RUN"
    info "Bundle:   $BUNDLE_DIR"
    echo ""

    check_prerequisites
    echo ""
    validate_metadata
    echo ""

    # Determine fork directories (convention from docs/operations/RELEASE.md)
    local fork_prod="${HOME}/forks/community-operators-prod"
    local fork_community="${HOME}/forks/community-operators"

    case "$TARGET" in
        community-operators-prod)
            submit_to_repo "$fork_prod" "$UPSTREAM_PROD"
            ;;
        community-operators)
            submit_to_repo "$fork_community" "$UPSTREAM_COMMUNITY"
            ;;
        both)
            submit_to_repo "$fork_prod" "$UPSTREAM_PROD"
            echo ""
            submit_to_repo "$fork_community" "$UPSTREAM_COMMUNITY"
            ;;
    esac

    echo ""
    ok "Submission complete!"
    if [[ "$DRY_RUN" == "true" ]]; then
        dry "No changes were made (dry-run mode)"
    fi
}

main "$@"
