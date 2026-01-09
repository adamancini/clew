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

# Semantic version comparison function
# Returns 0 (success) if $1 > $2, non-zero (failure) otherwise
function version_gt() {
    local ver1="$1"
    local ver2="$2"

    # Use sort -V for semantic version comparison
    # If ver1 is NOT the minimum (first) when sorted, then ver1 > ver2
    if [[ "$(printf '%s\n' "$ver1" "$ver2" | sort -V | head -n1)" != "$ver1" ]]; then
        return 0  # ver1 > ver2
    else
        return 1  # ver1 <= ver2
    fi
}

# Validation checks
info "Running validation checks..."
echo ""

# Check 1: Plugin.json and CHANGELOG.md versions must match
if [[ "$PLUGIN_VERSION" != "$CHANGELOG_VERSION" ]]; then
    error "Version mismatch!"
    error "  plugin.json: $PLUGIN_VERSION"
    error "  CHANGELOG.md: $CHANGELOG_VERSION"
    error ""
    error "Fix: Update both files to the same version"
    exit $EXIT_VERSION_MISMATCH
fi
success "Plugin.json and CHANGELOG.md versions match ($PLUGIN_VERSION)"

# Check 2: New version must be greater than latest tag
LATEST_TAG_STRIPPED="${LATEST_TAG#v}"  # Remove 'v' prefix

# Special case: First release (no real tags yet)
if [[ "$LATEST_TAG" == "v0.0.0" ]]; then
    warn "No previous tags found - this appears to be the first release"
    success "Version $PLUGIN_VERSION will be the first tagged release"
elif ! version_gt "$PLUGIN_VERSION" "$LATEST_TAG_STRIPPED"; then
    error "Version not bumped!"
    error "  New version: $PLUGIN_VERSION"
    error "  Latest tag: $LATEST_TAG_STRIPPED"
    error ""
    error "Fix: Bump version to be greater than $LATEST_TAG_STRIPPED"
    exit $EXIT_VERSION_NOT_GREATER
else
    success "Version $PLUGIN_VERSION > $LATEST_TAG_STRIPPED"
fi

# Check 3: Tag must not already exist for new version
if git rev-parse "v$PLUGIN_VERSION" >/dev/null 2>&1; then
    error "Tag v$PLUGIN_VERSION already exists!"
    error ""
    error "This version has already been released."
    error "Fix: Bump to a higher version number"
    exit $EXIT_TAG_EXISTS
fi
success "Tag v$PLUGIN_VERSION does not exist yet"

# Check 4: CHANGELOG must have valid date
if [[ -z "$CHANGELOG_DATE" ]] || ! [[ "$CHANGELOG_DATE" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    error "CHANGELOG.md entry for version $CHANGELOG_VERSION is missing a valid date"
    error ""
    error "Expected format: ## [$CHANGELOG_VERSION] - YYYY-MM-DD"
    error "Fix: Add date to CHANGELOG.md entry"
    exit $EXIT_MISSING_DATE
fi
success "CHANGELOG.md has valid date: $CHANGELOG_DATE"

echo ""
success "All version checks passed!"
success "Ready to merge and tag as v$PLUGIN_VERSION"
exit $EXIT_SUCCESS
