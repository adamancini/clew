#!/usr/bin/env bash
set -euo pipefail

# Version Bump Validation Script
# Validates that plugin.json, CHANGELOG.md, and git tags are in sync
# and that versions follow semantic versioning rules.

# Color output for readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function error() { echo -e "${RED}ERROR: $1${NC}" >&2; }
function success() { echo -e "${GREEN}✓ $1${NC}"; }
function warn() { echo -e "${YELLOW}⚠ $1${NC}"; }
function info() { echo "INFO: $1"; }

# Exit codes
EXIT_SUCCESS=0
EXIT_VERSION_MISMATCH=1
EXIT_VERSION_NOT_GREATER=2
EXIT_TAG_EXISTS=3
EXIT_MISSING_DATE=4
EXIT_MISSING_DEPENDENCY=5

# Check for required dependencies
if ! command -v jq &> /dev/null; then
    error "jq is not installed. Install with: brew install jq (macOS) or apt-get install jq (Linux)"
    exit $EXIT_MISSING_DEPENDENCY
fi

if ! command -v git &> /dev/null; then
    error "git is not installed"
    exit $EXIT_MISSING_DEPENDENCY
fi

info "Starting version bump validation..."
echo ""

# 1. Get latest git tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
info "Latest git tag: $LATEST_TAG"

# 2. Extract plugin.json version
PLUGIN_JSON=".claude-plugin/plugin.json"
if [[ ! -f "$PLUGIN_JSON" ]]; then
    error "File not found: $PLUGIN_JSON"
    exit $EXIT_MISSING_DEPENDENCY
fi

PLUGIN_VERSION=$(jq -r '.version' "$PLUGIN_JSON")
if [[ -z "$PLUGIN_VERSION" || "$PLUGIN_VERSION" == "null" ]]; then
    error "Could not read version from $PLUGIN_JSON"
    exit $EXIT_MISSING_DEPENDENCY
fi
info "plugin.json version: $PLUGIN_VERSION"

# 3. Extract CHANGELOG.md top entry (skips [Unreleased] section, finds first X.Y.Z version)
CHANGELOG_FILE="CHANGELOG.md"
if [[ ! -f "$CHANGELOG_FILE" ]]; then
    error "File not found: $CHANGELOG_FILE"
    exit $EXIT_MISSING_DEPENDENCY
fi

CHANGELOG_VERSION=$(grep -m 1 '^## \[[0-9]' "$CHANGELOG_FILE" | sed -E 's/^## \[([0-9]+\.[0-9]+\.[0-9]+)\].*/\1/' || echo "")
CHANGELOG_DATE=$(grep -m 1 '^## \[[0-9]' "$CHANGELOG_FILE" | sed -E 's/.*- ([0-9]{4}-[0-9]{2}-[0-9]{2}).*/\1/' || echo "")

if [[ -z "$CHANGELOG_VERSION" ]]; then
    error "Could not find version entry in $CHANGELOG_FILE (expected format: ## [X.Y.Z] - YYYY-MM-DD)"
    exit $EXIT_MISSING_DEPENDENCY
fi

info "CHANGELOG.md version: $CHANGELOG_VERSION (dated $CHANGELOG_DATE)"
echo ""
