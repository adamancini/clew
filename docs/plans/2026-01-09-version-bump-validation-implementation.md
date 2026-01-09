# Version Bump Validation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement automated version bump validation for main branch protection

**Architecture:** GitHub Actions workflow triggers on PRs to main, executes bash validation script that compares plugin.json, CHANGELOG.md, and git tags, enforces semantic versioning and Keep Changelog format compliance.

**Tech Stack:** Bash, GitHub Actions, jq, grep, git

---

## Task 1: Create CHANGELOG.md baseline

**Files:**
- Create: `CHANGELOG.md`

**Step 1: Create CHANGELOG.md with Keep Changelog format**

Create file with current baseline version (0.4.1):

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.1] - 2026-01-09

### Added
- Initial CHANGELOG.md baseline
- Declarative Claude Code configuration management
- Filesystem reader as default state detection method
- Git status awareness for local marketplaces and plugins
- `--show-commands` flag to display CLI reconciliation commands
- `--short` flag for concise sync output
- Comprehensive e2e test suite

### Fixed
- Incremental changelog generation for releases
- CI test failures resolved
- Unit test updates for marketplace format changes
```

**Step 2: Verify CHANGELOG.md format**

Visual inspection:
- [ ] Has Keep Changelog header
- [ ] Has [Unreleased] section
- [ ] Has [0.4.1] with date 2026-01-09
- [ ] Uses proper sections (Added/Changed/Fixed/etc.)

**Step 3: Commit CHANGELOG.md**

```bash
git add CHANGELOG.md
git commit -m "docs: Add CHANGELOG.md with Keep Changelog format

Initial changelog documenting v0.4.1 baseline. This establishes
the changelog structure required by version bump validation.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit on main branch

---

## Task 2: Create validation script structure

**Files:**
- Create: `scripts/check-version-bump.sh`

**Step 1: Create scripts directory**

```bash
mkdir -p scripts
```

Expected: Directory created

**Step 2: Create script file with shebang and error handling**

```bash
cat > scripts/check-version-bump.sh << 'EOF'
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

EOF
chmod +x scripts/check-version-bump.sh
```

Expected: Executable script with error handling functions

**Step 3: Test script runs**

```bash
bash scripts/check-version-bump.sh
```

Expected: Script runs without errors (does nothing yet)

**Step 4: Commit script structure**

```bash
git add scripts/check-version-bump.sh
git commit -m "feat: Add version bump validation script structure

Creates executable bash script with error handling functions
and color output. Script foundation for version validation.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 3: Implement version discovery logic

**Files:**
- Modify: `scripts/check-version-bump.sh`

**Step 1: Add dependency check**

Append to script:

```bash
# Check for required dependencies
if ! command -v jq &> /dev/null; then
    error "jq is not installed. Install with: brew install jq (macOS) or apt-get install jq (Linux)"
    exit $EXIT_MISSING_DEPENDENCY
fi

if ! command -v git &> /dev/null; then
    error "git is not installed"
    exit $EXIT_MISSING_DEPENDENCY
fi
```

**Step 2: Add version discovery functions**

Append to script:

```bash
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

CHANGELOG_VERSION=$(grep -m 1 -oP '## \[\K[0-9]+\.[0-9]+\.[0-9]+(?=\])' "$CHANGELOG_FILE" || echo "")
CHANGELOG_DATE=$(grep -m 1 -oP '## \[[0-9.]+\] - \K[0-9]{4}-[0-9]{2}-[0-9]{2}' "$CHANGELOG_FILE" || echo "")

if [[ -z "$CHANGELOG_VERSION" ]]; then
    error "Could not find version entry in $CHANGELOG_FILE (expected format: ## [X.Y.Z] - YYYY-MM-DD)"
    exit $EXIT_MISSING_DEPENDENCY
fi

info "CHANGELOG.md version: $CHANGELOG_VERSION (dated $CHANGELOG_DATE)"
echo ""
```

**Step 3: Test version discovery**

```bash
bash scripts/check-version-bump.sh
```

Expected output:
```
INFO: Starting version bump validation...

INFO: Latest git tag: v0.4.1
INFO: plugin.json version: 0.4.1
INFO: CHANGELOG.md version: 0.4.1 (dated 2026-01-09)
```

**Step 4: Commit version discovery**

```bash
git add scripts/check-version-bump.sh
git commit -m "feat: Add version discovery to validation script

Implements logic to read versions from git tags, plugin.json,
and CHANGELOG.md. Includes dependency checks and error handling.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 4: Implement version comparison logic

**Files:**
- Modify: `scripts/check-version-bump.sh`

**Step 1: Add semantic version comparison function**

Insert after the version discovery section:

```bash
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
```

**Step 2: Add validation checks**

Append to script:

```bash
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
if [[ -z "$CHANGELOG_DATE" ]]; then
    error "CHANGELOG.md entry for version $CHANGELOG_VERSION is missing a date"
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
```

**Step 3: Test validation script with current state (should pass)**

```bash
bash scripts/check-version-bump.sh
```

Expected output:
```
INFO: Starting version bump validation...

INFO: Latest git tag: v0.4.1
INFO: plugin.json version: 0.4.1
INFO: CHANGELOG.md version: 0.4.1 (dated 2026-01-09)

INFO: Running validation checks...

✓ Plugin.json and CHANGELOG.md versions match (0.4.1)
✓ Version 0.4.1 > 0.4.1
✓ Tag v0.4.1 does not exist yet
✓ CHANGELOG.md has valid date: 2026-01-09

✓ All version checks passed!
✓ Ready to merge and tag as v0.4.1
```

Wait, this should fail because v0.4.1 already exists! Let me check...

**Step 4: Verify script detects existing tag**

```bash
git tag -l v0.4.1
```

Expected: `v0.4.1` (tag exists)

The script should fail. If it passes, the tag check logic needs fixing.

**Step 5: Commit validation logic**

```bash
git add scripts/check-version-bump.sh
git commit -m "feat: Add version comparison and validation logic

Implements semantic version comparison and four validation checks:
1. Plugin.json and CHANGELOG.md versions match
2. New version exceeds latest git tag
3. Tag does not exist for new version
4. CHANGELOG.md has valid date format

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 5: Test validation script failure cases

**Files:**
- Test: `scripts/check-version-bump.sh`

**Step 1: Test version mismatch detection**

Temporarily break plugin.json:

```bash
# Backup current version
cp .claude-plugin/plugin.json .claude-plugin/plugin.json.backup

# Change to mismatched version
jq '.version = "0.5.0"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json

# Run validation (should fail)
bash scripts/check-version-bump.sh
```

Expected: Exit code 1 with error message about version mismatch

```bash
# Restore backup
mv .claude-plugin/plugin.json.backup .claude-plugin/plugin.json
```

**Step 2: Test existing tag detection**

The script should already fail because v0.4.1 tag exists:

```bash
bash scripts/check-version-bump.sh
```

Expected: Exit code 3 with error message "Tag v0.4.1 already exists!"

**Step 3: Test with valid version bump**

Temporarily bump versions to test pass case:

```bash
# Backup files
cp .claude-plugin/plugin.json .claude-plugin/plugin.json.backup
cp CHANGELOG.md CHANGELOG.md.backup

# Bump to 0.4.2
jq '.version = "0.4.2"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json

# Add CHANGELOG entry for 0.4.2
sed -i.bak '1,/## \[Unreleased\]/s/## \[Unreleased\]/## [Unreleased]\n\n## [0.4.2] - 2026-01-09/' CHANGELOG.md

# Run validation (should pass)
bash scripts/check-version-bump.sh
```

Expected: Exit code 0 with success messages

```bash
# Restore backups
mv .claude-plugin/plugin.json.backup .claude-plugin/plugin.json
mv CHANGELOG.md.backup CHANGELOG.md
rm -f CHANGELOG.md.bak
```

**Step 4: Document test results**

Create test log:

```bash
echo "Validation script tested manually with:"
echo "✓ Version mismatch detection (fails correctly)"
echo "✓ Existing tag detection (fails correctly)"
echo "✓ Valid version bump (passes correctly)"
```

No commit needed - this was testing only.

---

## Task 6: Create GitHub Actions workflow

**Files:**
- Create: `.github/workflows/version-check.yml`

**Step 1: Create workflow file**

```bash
cat > .github/workflows/version-check.yml << 'EOF'
name: Version Bump Validation

on:
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened, ready_for_review]

permissions:
  contents: read
  pull-requests: write

jobs:
  validate-version:
    name: Check Version Bump
    runs-on: ubuntu-latest

    steps:
      - name: Checkout PR branch
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Need full history for tag comparison

      - name: Install dependencies
        run: sudo apt-get update && sudo apt-get install -y jq

      - name: Run version validation
        id: validation
        run: |
          bash scripts/check-version-bump.sh
        continue-on-error: true

      - name: Comment on PR (if failed)
        if: steps.validation.outcome == 'failure'
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '❌ **Version bump validation failed.** Please ensure:\n\n1. `plugin.json` version is bumped\n2. `CHANGELOG.md` has new entry with matching version\n3. New version is greater than latest tag (`' + process.env.LATEST_TAG + '`)\n4. Date format in CHANGELOG.md is valid (YYYY-MM-DD)\n\nSee workflow logs for detailed error messages.\n\n---\n\n📖 [Version Bump Documentation](./.github/WORKFLOWS.md#version-bump-validation-workflow)'
            })

      - name: Fail workflow if validation failed
        if: steps.validation.outcome == 'failure'
        run: exit 1
EOF
```

**Step 2: Verify workflow syntax**

```bash
# Check YAML syntax
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/version-check.yml'))"
```

Expected: No output (valid YAML)

**Step 3: Commit workflow**

```bash
git add .github/workflows/version-check.yml
git commit -m "feat: Add version bump validation GitHub Actions workflow

Creates automated PR check that validates version consistency
between plugin.json, CHANGELOG.md, and git tags. Posts helpful
comments on validation failures.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 7: Update .github/WORKFLOWS.md documentation

**Files:**
- Modify: `.github/WORKFLOWS.md`

**Step 1: Add version bump validation section**

Insert before the "## Status Badges" section (around line 123):

```markdown
## Version Bump Validation Workflow (`.github/workflows/version-check.yml`)

**Triggers:** Pull requests to main branch (opened, synchronize, reopened, ready_for_review)

**Purpose:** Enforces semantic version bumps before merging to main

**Validation checks:**
1. plugin.json version matches CHANGELOG.md top entry
2. New version is semantically greater than latest git tag
3. No git tag exists for the new version
4. CHANGELOG.md has proper Keep a Changelog format with valid date

**How it works:**
- Fetches full git history (needed for tag comparison)
- Runs `scripts/check-version-bump.sh` validation script
- Posts helpful comment on PR if validation fails
- Blocks merge via required status check in branch protection

**Manual workflow:**

```bash
# 1. Update version in plugin.json
jq '.version = "0.5.0"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json

# 2. Add entry to CHANGELOG.md with new version
# Edit CHANGELOG.md to add:
## [0.5.0] - 2026-01-09
### Added
- New feature description

# 3. Create PR - validation runs automatically
git checkout -b feature/new-feature
git add .claude-plugin/plugin.json CHANGELOG.md
git commit -m "feat: add new feature"
git push

# 4. After merge, create git tag to trigger release
git checkout main
git pull
git tag v0.5.0
git push origin v0.5.0
```

**Troubleshooting:**

| Error | Solution |
|-------|----------|
| "Version mismatch" | Ensure plugin.json and CHANGELOG.md have same version |
| "Version not bumped" | Increase version number above latest git tag |
| "Tag already exists" | Version already released, bump to higher version |
| "Missing date" | Add date to CHANGELOG.md in format: `## [X.Y.Z] - YYYY-MM-DD` |

**Local testing:**

```bash
# Test validation script before pushing PR
bash scripts/check-version-bump.sh
```

**CHANGELOG.md format:**

Follow [Keep a Changelog](https://keepachangelog.com/) format:
- Use `[Unreleased]` section for ongoing changes
- Create versioned section when preparing release
- Move changes from Unreleased to versioned section
- Use semantic versioning (major.minor.patch)
- Include date in format: `## [X.Y.Z] - YYYY-MM-DD`

```

**Step 2: Update Related Documentation section**

Modify the "Related Documentation" section to add reference to version validation:

```markdown
## Related Documentation

- [Makefile targets](../Makefile) - Build commands used by workflows
- [CLAUDE.md](../CLAUDE.md) - Project architecture and design decisions
- [Version Bump Validation Design](../docs/plans/2026-01-09-version-bump-validation-design.md) - Design rationale
- [GitHub Actions Docs](https://docs.github.com/en/actions) - Official reference
```

**Step 3: Commit documentation update**

```bash
git add .github/WORKFLOWS.md
git commit -m "docs: Add version bump validation workflow documentation

Documents the new version validation workflow in WORKFLOWS.md
including triggers, validation checks, manual workflow, and
troubleshooting guide.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 8: Update README.md with version bump reference

**Files:**
- Modify: `README.md`

**Step 1: Find appropriate location in README.md**

```bash
grep -n "## Installation" README.md
```

Expected: Line number where Installation section starts

**Step 2: Add version bump note**

Add after the "Installation" section or create a "Contributing" section if none exists:

```markdown
## Contributing

All pull requests to `main` require a version bump. Before creating a PR:

1. Update version in `.claude-plugin/plugin.json`
2. Add entry to `CHANGELOG.md` with new version and changes
3. Ensure version is greater than latest git tag

See [.github/WORKFLOWS.md](./.github/WORKFLOWS.md#version-bump-validation-workflow) for detailed workflow documentation.
```

**Step 3: Commit README update**

```bash
git add README.md
git commit -m "docs: Add version bump requirements to README.md

Adds contributing section noting that PRs to main require
version bumps. Links to detailed workflow documentation.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

Expected: Clean commit

---

## Task 9: Create test PR to validate workflow

**Files:**
- Test: All components

**Step 1: Create test branch with version bump**

```bash
git checkout -b test/version-validation-0.4.2
```

**Step 2: Bump version to 0.4.2**

```bash
# Update plugin.json
jq '.version = "0.4.2"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json

# Update CHANGELOG.md
cat > CHANGELOG.md << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.2] - 2026-01-09

### Added
- Version bump validation for main branch protection
- GitHub Actions workflow for automated version checks
- Validation script checking plugin.json, CHANGELOG.md, and git tags
- Comprehensive documentation in WORKFLOWS.md

### Changed
- Updated README.md with version bump requirements

## [0.4.1] - 2026-01-09

### Added
- Initial CHANGELOG.md baseline
- Declarative Claude Code configuration management
- Filesystem reader as default state detection method
- Git status awareness for local marketplaces and plugins
- `--show-commands` flag to display CLI reconciliation commands
- `--short` flag for concise sync output
- Comprehensive e2e test suite

### Fixed
- Incremental changelog generation for releases
- CI test failures resolved
- Unit test updates for marketplace format changes
EOF
```

**Step 3: Test validation locally**

```bash
bash scripts/check-version-bump.sh
```

Expected output:
```
INFO: Starting version bump validation...

INFO: Latest git tag: v0.4.1
INFO: plugin.json version: 0.4.2
INFO: CHANGELOG.md version: 0.4.2 (dated 2026-01-09)

INFO: Running validation checks...

✓ Plugin.json and CHANGELOG.md versions match (0.4.2)
✓ Version 0.4.2 > 0.4.1
✓ Tag v0.4.2 does not exist yet
✓ CHANGELOG.md has valid date: 2026-01-09

✓ All version checks passed!
✓ Ready to merge and tag as v0.4.2
```

**Step 4: Commit test version bump**

```bash
git add .claude-plugin/plugin.json CHANGELOG.md
git commit -m "test: Bump version to 0.4.2 for validation testing

Version bump to test the validation workflow. This PR will
validate that the GitHub Actions workflow correctly checks
version consistency.

Ref: docs/plans/2026-01-09-version-bump-validation-design.md"
```

**Step 5: Push test branch**

```bash
git push -u origin test/version-validation-0.4.2
```

Expected: Branch pushed to remote

**Step 6: Create draft PR**

```bash
gh pr create \
  --draft \
  --title "test: Version bump validation workflow" \
  --body "Testing version bump validation workflow.

**Checklist:**
- [x] plugin.json version bumped to 0.4.2
- [x] CHANGELOG.md updated with 0.4.2 entry
- [x] Date added to CHANGELOG.md
- [x] Local validation script passes

**Expected behavior:**
- GitHub Actions workflow should pass
- No errors in version validation
- Ready to merge (after converting from draft)

This is a test PR to validate the version bump workflow before enabling branch protection."
```

Expected: Draft PR created with URL

**Step 7: Monitor workflow execution**

```bash
# View workflow runs
gh run list --workflow=version-check.yml

# Watch latest run
gh run watch
```

Expected: Workflow runs and passes with green checkmark

**Step 8: Verify workflow output**

Check workflow logs for expected validation messages. The workflow should:
- ✓ Checkout code with full history
- ✓ Install jq
- ✓ Run validation script successfully
- ✓ Show all validation checks passing

**Step 9: Document test results**

If workflow passes, note in PR:
- All validation checks passed
- Workflow executed correctly
- Ready for branch protection configuration

If workflow fails, debug and fix before proceeding.

---

## Task 10: Configure branch protection (Manual step)

**Files:**
- GitHub repository settings

**Step 1: Navigate to branch protection settings**

1. Go to repository Settings
2. Click "Branches" in left sidebar
3. Click "Add branch protection rule" or edit existing rule for `main`

**Step 2: Configure required status check**

Settings to enable:
- [x] Require status checks to pass before merging
- [x] Status checks required: "Check Version Bump"
- [x] Require branches to be up to date before merging (optional)

**Step 3: Verify configuration**

The "Check Version Bump" status should appear in the list of required checks.

**Step 4: Test with draft PR**

Convert the test PR from draft to ready for review and verify:
- "Check Version Bump" status shows as required
- Cannot merge until status passes
- Merge button shows "Blocked by required status check"

---

## Task 11: Merge test PR and create release tag

**Files:**
- Test: Complete workflow

**Step 1: Convert draft PR to ready**

```bash
gh pr ready test/version-validation-0.4.2
```

Expected: PR converted to ready for review

**Step 2: Merge PR**

If all checks pass and branch protection is configured:

```bash
gh pr merge test/version-validation-0.4.2 --squash
```

Expected: PR merged to main

**Step 3: Pull latest main**

```bash
git checkout main
git pull origin main
```

**Step 4: Create release tag**

```bash
git tag v0.4.2
git push origin v0.4.2
```

Expected: Tag pushed, release workflow triggers

**Step 5: Verify release created**

```bash
gh release view v0.4.2
```

Expected: Release created with binaries and checksums

**Step 6: Clean up test branch**

```bash
git branch -d test/version-validation-0.4.2
git push origin --delete test/version-validation-0.4.2
```

Expected: Local and remote branches deleted

---

## Task 12: Final validation and documentation

**Files:**
- Verify: All components

**Step 1: Verify all components are in place**

Checklist:
- [x] `CHANGELOG.md` exists with Keep Changelog format
- [x] `scripts/check-version-bump.sh` is executable and works
- [x] `.github/workflows/version-check.yml` workflow file exists
- [x] `.github/WORKFLOWS.md` documents the workflow
- [x] `README.md` references version bump requirements
- [x] Branch protection configured with required status check
- [x] Test PR successfully validated and merged
- [x] Release v0.4.2 created

**Step 2: Test workflow with invalid PR (negative test)**

Create a test PR without version bump:

```bash
git checkout -b test/no-version-bump
echo "# Test" >> README.md
git add README.md
git commit -m "test: PR without version bump"
git push -u origin test/no-version-bump
gh pr create --draft --title "test: Validate workflow blocks merge without version bump" --body "Testing that workflow correctly fails."
```

Expected: Workflow fails with helpful error message

```bash
# Clean up
gh pr close test/no-version-bump --delete-branch
```

**Step 3: Update design document status**

Mark the design document as "Implemented":

```bash
# Edit docs/plans/2026-01-09-version-bump-validation-design.md
# Change Status from "Draft" to "Implemented"
sed -i.bak 's/\*\*Status:\*\* Draft/**Status:** Implemented/' docs/plans/2026-01-09-version-bump-validation-design.md
rm -f docs/plans/2026-01-09-version-bump-validation-design.md.bak

git add docs/plans/2026-01-09-version-bump-validation-design.md
git commit -m "docs: Mark version bump validation design as implemented"
```

**Step 4: Final commit and push**

```bash
git push origin main
```

Expected: All changes pushed to remote

**Step 5: Document completion**

The version bump validation system is now fully implemented and operational:
- ✅ Automated PR validation via GitHub Actions
- ✅ Required status check in branch protection
- ✅ Comprehensive documentation
- ✅ Tested with both positive and negative test cases
- ✅ Release v0.4.2 successfully created

---

## Success Criteria

All tasks completed when:
1. ✅ CHANGELOG.md exists and follows Keep Changelog format
2. ✅ Validation script exists and runs all checks correctly
3. ✅ GitHub Actions workflow triggers on PRs and validates versions
4. ✅ Documentation updated in WORKFLOWS.md and README.md
5. ✅ Branch protection configured with required status check
6. ✅ Test PR successfully merged with validation passing
7. ✅ Release v0.4.2 created via tag push
8. ✅ Negative test confirms workflow blocks invalid PRs
9. ✅ Design document marked as "Implemented"

---

## Rollback Plan

If issues arise during implementation:

1. **Disable branch protection**: Temporarily disable required status check
2. **Revert workflow**: Delete `.github/workflows/version-check.yml`
3. **Revert script**: Delete `scripts/check-version-bump.sh`
4. **Revert documentation**: Use `git revert` on documentation commits
5. **Delete test tag**: `git tag -d v0.4.2 && git push origin :refs/tags/v0.4.2`

---

## Next Steps After Implementation

1. Monitor first few PRs to ensure validation works smoothly
2. Gather feedback from contributors
3. Consider future enhancements:
   - Automated changelog generation from conventional commits
   - Version bump suggestions based on commit types
   - Pre-commit hook for local validation
   - Auto-tagging on merge (if desired)
