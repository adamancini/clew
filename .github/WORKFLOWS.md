# GitHub Actions Workflows

This document describes the CI/CD workflows for the clew project.

## CI Workflow (`.github/workflows/ci.yml`)

**Triggers:** Push to main, Pull requests to main

**Purpose:** Continuous integration testing and code quality checks

**Jobs:**

1. **Test Matrix**
   - Runs on: Ubuntu 22.04 and macOS latest
   - Go versions: 1.21, 1.22, 1.23
   - Executes: `make test`
   - Collects: Test coverage reports
   - Uploads: Coverage to Codecov (main build only)

2. **Lint**
   - Runs on: Ubuntu 22.04
   - Executes: golangci-lint with all checks enabled
   - Timeout: 5 minutes
   - Fails: Build if linting issues found

3. **Build Verification**
   - Runs on: Ubuntu 22.04
   - Executes: Cross-platform binary builds via `make plugin-binaries`
   - Verifies: All binaries are executable and respond to `--help`
   - Artifacts: Uploaded for 24 hours

**Caching:**
- Go modules cached via actions/setup-go
- Build cache not used (compilation is fast for clew)

**Success Criteria:**
- All tests pass on all Go versions and platforms
- No linting violations
- All binaries build successfully

## Release Workflow (`.github/workflows/release.yml`)

**Triggers:** Git tags matching `v*` (e.g., `v0.2.0`, `v1.0.0`)

**Purpose:** Automated binary releases with checksums

**Process:**

1. **Build & Package**
   - Checks out code with full history (needed for changelog)
   - Builds all platform binaries via `make plugin-binaries`
   - Generates SHA256 checksums for all binaries

2. **Release Notes**
   - Extracts commit log for release (last 20 commits or since previous tag)
   - Adds installation instructions
   - Includes checksums for verification

3. **GitHub Release**
   - Creates release from git tag
   - Uploads binaries: `clew-darwin-{arm64,amd64}` and `clew-linux-{amd64,arm64}`
   - Uploads: `checksums.txt` (SHA256 format)
   - Uses body_path to include formatted release notes

4. **Artifact Retention**
   - Artifacts retained for 30 days in GitHub Actions

**Manual Release Workflow:**

```bash
# Create and push a version tag
git tag v0.2.0
git push --tags

# GitHub Actions automatically creates release with binaries
```

**Verifying Release:**

```bash
# Download binary
curl -L https://github.com/adamancini/clew/releases/download/v0.2.0/clew-linux-amd64 \
  -o clew && chmod +x clew

# Verify checksum (download checksums.txt from release)
sha256sum -c checksums.txt
```

## CodeQL Workflow (`.github/workflows/codeql.yml`)

**Triggers:**
- Push to main branch
- Pull requests to main branch
- Weekly schedule (Sunday midnight UTC)

**Purpose:** Security code scanning and vulnerability detection

**Configuration:**
- Language: Go
- Query suite: security-and-quality
- Autobuild: Yes (automatic Go detection)

**Results:**
- Available in Security tab on GitHub
- Blocks merge on security issues (if configured in branch protection)

## Dependabot Configuration (`.github/dependabot.yml`)

**Purpose:** Automated dependency updates

**Go Dependencies:**
- Schedule: Weekly (Monday 4:00 UTC)
- Label: `dependencies`, `go`
- Reviewers: adamancini
- Commit prefix: `chore(deps)`

**GitHub Actions:**
- Schedule: Weekly (Monday 4:00 UTC)
- Label: `ci`, `github-actions`
- Reviewers: adamancini
- Commit prefix: `ci(actions)`

## Status Badges

The README includes three status badges:

1. **CI Badge** - Shows status of test and lint workflows
   ```
   ![CI](https://github.com/adamancini/clew/actions/workflows/ci.yml/badge.svg)
   ```

2. **CodeQL Badge** - Shows status of security scanning
   ```
   ![CodeQL](https://github.com/adamancini/clew/actions/workflows/codeql.yml/badge.svg)
   ```

3. **Code Coverage Badge** - Shows test coverage from Codecov
   ```
   ![codecov](https://codecov.io/gh/adamancini/clew/branch/main/graph/badge.svg)
   ```

## Action Versions

All actions are pinned to major versions for stability:

| Action | Version | Purpose |
|--------|---------|---------|
| `actions/checkout` | v4 | Clone repository |
| `actions/setup-go` | v5 | Configure Go environment |
| `golangci/golangci-lint-action` | v4 | Lint Go code |
| `codecov/codecov-action` | v4 | Upload coverage |
| `github/codeql-action` | v3 | Security scanning |
| `softprops/action-gh-release` | v2 | Create releases |
| `actions/upload-artifact` | v4 | Store build artifacts |

## Troubleshooting

### CI fails on Go 1.23 but passes on 1.22

Check for deprecated APIs or Go version-specific issues:

```bash
go mod tidy
go test ./...
```

### Release workflow doesn't trigger

Ensure the tag matches the pattern `v*`:

```bash
git tag v0.2.0  # Creates release
git tag 0.2.0   # Does NOT create release
```

### Codecov integration not working

1. Repository must be public (codecov requires public repos for free tier)
2. CODECOV_TOKEN is optional for public repos but can be added to secrets if needed
3. Check codecov.io dashboard for upload status

### Dependabot PRs not being created

1. Enable Dependabot in repository settings
2. Check for branch protection rules that might require review before Dependabot can create PRs
3. Verify schedule is set correctly

## Future Enhancements

Potential improvements:

1. **SBOM Generation** - Generate CycloneDX SBOM for binaries using syft
2. **Signature Verification** - Sign releases using cosign with Keyless signing
3. **Multi-matrix Coverage** - Different test coverage reporting per Go version
4. **Performance Benchmarks** - Track performance metrics over releases
5. **Windows Support** - Add windows-latest to test matrix
6. **Docker Images** - Build and push Docker images on release

## Related Documentation

- [Makefile targets](../Makefile) - Build commands used by workflows
- [CLAUDE.md](../CLAUDE.md) - Project architecture and design decisions
- [GitHub Actions Docs](https://docs.github.com/en/actions) - Official reference
