# Version Bump Validation Design

**Date:** 2026-01-09
**Author:** Ada Mancini
**Status:** Draft

## Overview

This document describes a branch protection mechanism that requires version bumps on all PRs targeting `main`. The system validates three components: git tags, plugin.json version, and CHANGELOG.md entries. This prevents releases without proper version management.

## Problem Statement

Currently, merges to `main` do not require version updates. This creates risks:

- Releases without version bumps
- Out-of-sync versions between plugin.json and git tags
- Missing changelog entries
- Unclear release history

## Solution

We enforce version bumps through GitHub branch protection with a required status check. A validation script runs on every PR, checking that:

1. plugin.json version matches CHANGELOG.md top entry
2. The new version exceeds the latest git tag semantically
3. No git tag exists for the new version yet
4. CHANGELOG.md follows Keep Changelog format with a valid date

## Architecture

### Components

**GitHub Actions Workflow** (`.github/workflows/version-check.yml`)
- Triggers on all PRs targeting `main`
- Fetches full git history for tag comparison
- Executes validation script
- Posts helpful comments on validation failures
- Serves as required status check in branch protection

**Validation Script** (`scripts/check-version-bump.sh`)
- Reads latest git tag via `git describe --tags --abbrev=0`
- Parses plugin.json version with `jq`
- Extracts CHANGELOG.md top entry with grep
- Compares versions semantically using `sort -V`
- Exits with status code 0 on success, non-zero with error message on failure

**Branch Protection Rule**
- Requires "Check Version Bump" status to pass before merge
- Configured in repository settings or via GitHub API

### Validation Logic

The script validates versions in this sequence:

1. **Discover current version** - Query git for latest tag (e.g., `v0.4.1`)
2. **Parse plugin.json** - Extract version field (e.g., `0.5.0`)
3. **Parse CHANGELOG.md** - Find topmost `## [X.Y.Z] - YYYY-MM-DD` entry
4. **Compare semantically** - Split versions, compare numerically (0.5.0 > 0.4.1)
5. **Check consistency** - Verify plugin.json and CHANGELOG versions match
6. **Verify tag absence** - Confirm the new version has no existing tag

**Special cases:**
- First release: Allow any version ≥ 0.1.0
- Prerelease versions: Support `-alpha`, `-beta`, `-rc` suffixes
- PR from main to main: Skip validation (no version bump needed)

## Implementation

### GitHub Workflow

```yaml
name: Version Bump Validation

on:
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened, ready_for_review]

permissions:
  contents: read
  pull-requests: read

jobs:
  validate-version:
    name: Check Version Bump
    runs-on: ubuntu-latest

    steps:
      - name: Checkout PR branch
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install dependencies
        run: sudo apt-get update && sudo apt-get install -y jq

      - name: Run version validation
        run: bash scripts/check-version-bump.sh

      - name: Comment on PR (if failed)
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '❌ **Version bump validation failed.** Please ensure:\n\n1. `plugin.json` version is bumped\n2. `CHANGELOG.md` has new entry with matching version\n3. New version is greater than latest tag\n\nSee workflow logs for details.'
            })
```

### Validation Script

```bash
#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

function error() { echo -e "${RED}ERROR: $1${NC}" >&2; }
function success() { echo -e "${GREEN}✓ $1${NC}"; }

# Get latest git tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Latest git tag: $LATEST_TAG"

# Extract plugin.json version
PLUGIN_VERSION=$(jq -r '.version' .claude-plugin/plugin.json)
echo "plugin.json version: $PLUGIN_VERSION"

# Extract CHANGELOG.md top entry
CHANGELOG_VERSION=$(grep -m 1 -oP '## \[\K[0-9]+\.[0-9]+\.[0-9]+(?=\])' CHANGELOG.md || echo "")
CHANGELOG_DATE=$(grep -m 1 -oP '## \[[0-9.]+\] - \K[0-9]{4}-[0-9]{2}-[0-9]{2}' CHANGELOG.md || echo "")
echo "CHANGELOG.md version: $CHANGELOG_VERSION (dated $CHANGELOG_DATE)"

# Semantic version comparison
function version_gt() {
  [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" != "$1" ]
}

# Validate versions match
if [[ "$PLUGIN_VERSION" != "$CHANGELOG_VERSION" ]]; then
  error "Version mismatch: plugin.json ($PLUGIN_VERSION) != CHANGELOG.md ($CHANGELOG_VERSION)"
  exit 1
fi

# Ensure new version > latest tag
LATEST_TAG_STRIPPED="${LATEST_TAG#v}"
if ! version_gt "$PLUGIN_VERSION" "$LATEST_TAG_STRIPPED"; then
  error "New version ($PLUGIN_VERSION) must be > latest tag ($LATEST_TAG_STRIPPED)"
  exit 1
fi

# Ensure tag doesn't exist
if git rev-parse "v$PLUGIN_VERSION" >/dev/null 2>&1; then
  error "Tag v$PLUGIN_VERSION already exists. Version already released."
  exit 1
fi

# Validate changelog date
if [[ -z "$CHANGELOG_DATE" ]]; then
  error "CHANGELOG.md missing date for version $CHANGELOG_VERSION"
  exit 1
fi

success "All version checks passed!"
success "Ready to merge and tag as v$PLUGIN_VERSION"
```

## Migration Plan

### Initial Setup

**Step 1: Sync versions**
- Update plugin.json from `1.0.0` to `0.4.1` (current git tag)
- Create CHANGELOG.md with Keep Changelog structure
- Document existing releases or start fresh with v0.4.1 baseline

**Step 2: Add validation components**
- Create `scripts/check-version-bump.sh` with execute permissions
- Add `.github/workflows/version-check.yml`
- Test locally: `bash scripts/check-version-bump.sh`

**Step 3: Configure branch protection**
- Enable "Require status checks to pass before merging"
- Select "Check Version Bump" as required check
- Test with a draft PR bumping version to 0.4.2

### Developer Workflow

When preparing a release:

1. Create feature branch: `git checkout -b feature/something`
2. Make changes
3. Update `plugin.json`: `"version": "0.5.0"`
4. Update `CHANGELOG.md`:
   ```markdown
   ## [0.5.0] - 2026-01-09
   ### Added
   - New feature X
   ### Fixed
   - Bug Y
   ```
5. Push PR—GitHub validates versions
6. Merge when approved
7. Create tag: `git tag v0.5.0 && git push origin v0.5.0`
8. Release workflow runs automatically

### Documentation

Add to CONTRIBUTING.md or README.md:

- Version bump requirements for main branch
- CHANGELOG format (link to keepachangelog.com)
- Example of proper version update PR
- Note about branch protection enforcement

## CHANGELOG Format

Follow Keep Changelog format:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-01-09
### Added
- Version bump validation for main branch protection
- GitHub Actions workflow for automated checks

### Changed
- Updated plugin.json to track with git tags

### Fixed
- Version sync between plugin.json and git tags

## [0.4.1] - 2025-XX-XX
(Previous releases documented here)
```

## Testing Strategy

**Local validation:**
```bash
# Test script directly
bash scripts/check-version-bump.sh

# Should pass with valid version bump
# Should fail with invalid/missing version
```

**Integration testing:**
1. Create test branch with version bump
2. Open draft PR to main
3. Verify workflow runs and reports results
4. Test failure cases: mismatched versions, no bump, existing tag
5. Verify helpful error messages appear

## Rollout Timeline

1. Implement sync and validation (1 session)
2. Test with draft PR (1 session)
3. Enable branch protection (immediate)
4. Monitor first release cycle (ongoing)

## Future Enhancements

- Validate semantic version type (major/minor/patch) matches changes
- Auto-suggest version bump based on conventional commits
- Automated changelog generation from commit messages
- Version bump reminder bot that comments on PRs without bumps

## References

- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
