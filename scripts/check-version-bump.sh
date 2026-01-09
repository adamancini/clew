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
