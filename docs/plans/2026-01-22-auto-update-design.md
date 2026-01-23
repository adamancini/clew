# Auto-Update and Distribution Enhancement Design

**Date:** 2026-01-22
**Author:** Ada Mancini
**Status:** Planning
**Related Issue:** #60

## Overview

This document describes the implementation of a self-update mechanism for clew, allowing users to update to the latest version with a single command. Additionally, we explore adding GitHub Packages support to provide multiple distribution channels.

## Problem Statement

Currently, users must manually:
1. Check for new releases on GitHub
2. Download the correct binary for their platform
3. Replace the installed binary
4. Verify the installation

This creates friction for keeping clew up to date and reduces user adoption of new features and bug fixes.

## Solution

Implement `clew version --update` command that:
1. Checks GitHub releases for the latest version
2. Detects the current platform (OS/architecture)
3. Downloads the appropriate binary
4. Verifies integrity using checksums
5. Replaces the current binary safely with rollback support
6. Verifies the new installation

## Architecture

### Phase 1: Auto-Update via GitHub Releases (Primary)

**Components:**

1. **Version Checker** (`internal/update/checker.go`)
   - Query GitHub API for latest release
   - Parse version tags (semantic versioning)
   - Compare with current version
   - Return update availability status

2. **Platform Detector** (`internal/update/platform.go`)
   - Detect GOOS (darwin, linux)
   - Detect GOARCH (amd64, arm64)
   - Map to release asset names (e.g., `clew-darwin-arm64`)

3. **Binary Downloader** (`internal/update/downloader.go`)
   - Download release assets from GitHub
   - Stream to temporary location
   - Verify checksums against `checksums.txt`
   - Handle network errors and retries

4. **Binary Replacer** (`internal/update/replacer.go`)
   - Create backup of current binary
   - Atomically replace binary (rename operation)
   - Handle permissions (may require sudo)
   - Rollback on failure

5. **Update Command** (`internal/cmd/version.go`)
   - Add `--update` flag to version command
   - Add `--check` flag for dry-run
   - Orchestrate update workflow
   - Provide user feedback

### Phase 2: GitHub Packages Support (Optional Enhancement)

**Additional Distribution Channels:**

1. **GitHub Container Registry (GHCR)**
   - Publish clew as OCI container image
   - Allows: `docker run ghcr.io/adamancini/clew:latest`
   - Use case: CI/CD pipelines, containerized environments

2. **GitHub Release Assets** (Current)
   - Continue publishing standalone binaries
   - Primary distribution for direct installation

**Release Workflow Updates:**

```yaml
# .github/workflows/release.yml additions
- name: Build and push container image
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: ghcr.io/adamancini/clew:${{ github.ref_name }}
```

## Implementation Details

### Version Checking

```go
// internal/update/checker.go
type Checker struct {
    currentVersion string
    githubToken    string // optional, for rate limiting
}

func (c *Checker) CheckForUpdate() (*UpdateInfo, error) {
    // GET https://api.github.com/repos/adamancini/clew/releases/latest
    // Parse JSON response
    // Compare versions
    // Return UpdateInfo
}

type UpdateInfo struct {
    Available      bool
    CurrentVersion string
    LatestVersion  string
    ReleaseURL     string
    ReleaseNotes   string
}
```

### Platform Detection

```go
// internal/update/platform.go
type Platform struct {
    OS   string // darwin, linux
    Arch string // amd64, arm64
}

func Detect() Platform {
    return Platform{
        OS:   runtime.GOOS,
        Arch: runtime.GOARCH,
    }
}

func (p Platform) BinaryName() string {
    return fmt.Sprintf("clew-%s-%s", p.OS, p.Arch)
}
```

### Binary Download & Verification

```go
// internal/update/downloader.go
type Downloader struct {
    client *http.Client
}

func (d *Downloader) Download(url string, dst string) error {
    // HTTP GET with progress reporting
    // Stream to temporary file
    // Return path to downloaded file
}

func (d *Downloader) VerifyChecksum(file, checksum string) error {
    // Calculate SHA256 of downloaded file
    // Compare against expected checksum
    // Return error if mismatch
}
```

### Safe Binary Replacement

```go
// internal/update/replacer.go
type Replacer struct {
    currentPath string
    backupPath  string
}

func (r *Replacer) Replace(newBinary string) error {
    // 1. Create backup of current binary
    // 2. Copy new binary to temporary location
    // 3. Atomic rename operation
    // 4. Set executable permissions
    // 5. Verify new binary works (run --version)
    // 6. Remove backup on success, restore on failure
}

func (r *Replacer) Rollback() error {
    // Restore from backup if update fails
}
```

### Update Command Integration

```go
// internal/cmd/version.go

func init() {
    versionCmd.Flags().Bool("check", false, "Check for updates without installing")
    versionCmd.Flags().Bool("update", false, "Update to the latest version")
    versionCmd.Flags().Bool("pre", false, "Include pre-release versions")
}

func runVersionUpdate(cmd *cobra.Command, args []string) error {
    check, _ := cmd.Flags().GetBool("check")
    update, _ := cmd.Flags().GetBool("update")
    includePre, _ := cmd.Flags().GetBool("pre")

    // Check for updates
    checker := update.NewChecker(version)
    info, err := checker.CheckForUpdate()
    if err != nil {
        return err
    }

    if !info.Available {
        fmt.Println("Already running latest version:", version)
        return nil
    }

    fmt.Printf("Update available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)

    if check {
        // Dry-run mode
        fmt.Println("Run 'clew version --update' to install")
        return nil
    }

    if update {
        // Perform update
        return performUpdate(info)
    }

    return nil
}
```

## Security Considerations

1. **Checksum Verification**
   - Always verify SHA256 checksums before replacing binary
   - Fail fast if checksums don't match
   - Use checksums from trusted source (GitHub releases)

2. **HTTPS Only**
   - Only download over HTTPS
   - Verify GitHub's SSL certificate
   - Use GitHub API authentication token when available

3. **Backup Strategy**
   - Always create backup before replacement
   - Keep backup until verification succeeds
   - Provide rollback mechanism

4. **Permission Handling**
   - Detect if current binary is in system path (e.g., `/usr/local/bin`)
   - Provide clear error if elevated permissions needed
   - Don't automatically escalate privileges
   - Suggest: `sudo clew version --update` for system installs

5. **Code Signing** (Future Enhancement)
   - Consider signing binaries with GPG
   - Verify signatures during update
   - Add to GitHub release workflow

## User Experience

### Check for Updates

```bash
$ clew version --check
Current version: 0.8.2
Latest version: 0.9.0 available

Release notes:
- Added auto-update command
- Fixed bug in marketplace sync
- Improved error messages

Run 'clew version --update' to install
```

### Perform Update

```bash
$ clew version --update
Current version: 0.8.2
Latest version: 0.9.0 available

Downloading clew-darwin-arm64...
✓ Downloaded (2.4 MB)
✓ Checksum verified
✓ Backup created: /usr/local/bin/clew.backup

Installing update...
✓ Binary replaced
✓ Verification passed

Successfully updated to v0.9.0!
```

### Update with Pre-release

```bash
$ clew version --update --pre
Current version: 0.8.2
Latest version: 0.9.0-rc.1 available (pre-release)

Note: This is a pre-release version and may be unstable.

Continue? [y/N]: y
...
```

### Update Failure with Rollback

```bash
$ clew version --update
Current version: 0.8.2
Latest version: 0.9.0 available

Downloading clew-darwin-arm64...
✓ Downloaded (2.4 MB)
✗ Checksum verification failed!

Expected: a1b2c3d4...
Received: e5f6g7h8...

Update aborted. Current version (0.8.2) unchanged.
```

## GitHub Packages Implementation (Optional)

### Container Image

**Dockerfile:**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY clew /usr/local/bin/clew
RUN chmod +x /usr/local/bin/clew
ENTRYPOINT ["/usr/local/bin/clew"]
```

**Usage:**
```bash
# Pull and run
docker run ghcr.io/adamancini/clew:latest export

# Use in CI/CD
docker run ghcr.io/adamancini/clew:v0.9.0 sync --config /config/Clewfile.yaml
```

### Release Workflow Updates

Add container build/push to `.github/workflows/release.yml`:

```yaml
- name: Log in to GitHub Container Registry
  uses: docker/login-action@v3
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}

- name: Build and push container image
  uses: docker/build-push-action@v5
  with:
    context: .
    push: true
    tags: |
      ghcr.io/adamancini/clew:${{ github.ref_name }}
      ghcr.io/adamancini/clew:latest
    platforms: linux/amd64,linux/arm64
```

## Implementation Plan

### Phase 1: Core Auto-Update (Issue #60)

**Epic:** Auto-Update Command Implementation

**Subtasks:**

1. **Design & Structure** (1-2 hours)
   - [ ] Create `internal/update/` package structure
   - [ ] Define interfaces and types
   - [ ] Add tests scaffolding

2. **Version Checker** (1-2 hours)
   - [ ] Implement GitHub API client for releases
   - [ ] Parse release versions
   - [ ] Compare semantic versions
   - [ ] Handle rate limiting
   - [ ] Add unit tests

3. **Platform Detection & Download** (2-3 hours)
   - [ ] Implement platform detection
   - [ ] Map platforms to binary names
   - [ ] Implement binary downloader with progress
   - [ ] Implement checksum verification
   - [ ] Add unit tests with mock HTTP server

4. **Binary Replacement** (2-3 hours)
   - [ ] Implement backup creation
   - [ ] Implement atomic replacement
   - [ ] Handle permissions
   - [ ] Implement rollback
   - [ ] Add integration tests

5. **Command Integration** (1-2 hours)
   - [ ] Add flags to version command
   - [ ] Implement update workflow
   - [ ] Add error handling
   - [ ] Update help text

6. **Documentation** (1 hour)
   - [ ] Update README with update instructions
   - [ ] Add troubleshooting guide
   - [ ] Update CLAUDE.md

7. **Testing & Polish** (2-3 hours)
   - [ ] Manual testing on all platforms
   - [ ] E2E test for update workflow
   - [ ] Error message improvements
   - [ ] Progress indicators

**Total Estimate:** 10-16 hours

### Phase 2: GitHub Packages Support (Optional)

**Subtasks:**

1. **Container Image** (2-3 hours)
   - [ ] Create Dockerfile
   - [ ] Multi-platform build setup
   - [ ] Test container locally

2. **Release Workflow** (1-2 hours)
   - [ ] Add container build/push to workflow
   - [ ] Configure GHCR authentication
   - [ ] Test on tag push

3. **Documentation** (1 hour)
   - [ ] Update README with container usage
   - [ ] Add CI/CD examples

**Total Estimate:** 4-6 hours

## Testing Strategy

### Unit Tests

- Version comparison logic
- Platform detection
- Checksum verification
- Backup/restore logic

### Integration Tests

- Mock GitHub API responses
- Temporary file handling
- Binary replacement workflow

### E2E Tests

- Full update cycle on test binary
- Checksum mismatch handling
- Permission errors
- Network failure scenarios

### Manual Testing

Test on all supported platforms:
- ✅ macOS arm64
- ✅ macOS amd64
- ✅ Linux amd64
- ✅ Linux arm64

Test scenarios:
- Fresh install location
- System install location (/usr/local/bin)
- Custom install location
- Update from old version to new
- Update when already latest
- Update with network failures
- Update with checksum mismatch

## Success Criteria

1. ✅ `clew version --check` correctly identifies updates
2. ✅ `clew version --update` successfully updates binary
3. ✅ Checksum verification prevents corrupted downloads
4. ✅ Backup/rollback works on update failures
5. ✅ Clear error messages for all failure modes
6. ✅ Works on all supported platforms
7. ✅ Handles permission issues gracefully
8. ✅ All tests pass (unit + integration + e2e)
9. ✅ Documentation complete and accurate

## Future Enhancements

1. **Auto-update notifications**
   - Check for updates in background
   - Notify user on command execution
   - Configurable frequency

2. **Update channels**
   - Stable, beta, alpha channels
   - Pin to specific channel
   - Auto-update within channel

3. **Binary signing**
   - GPG signature verification
   - Stronger security guarantees

4. **Plugin integration**
   - Hook into Claude Code plugin
   - Notify on SessionStart
   - Update via skill command

5. **Homebrew/apt support**
   - Publish to package managers
   - Integrate with system package updates

## References

- Issue #60: https://github.com/adamancini/clew/issues/60
- Self-update implementations:
  - [rhysd/go-github-selfupdate](https://github.com/rhysd/go-github-selfupdate)
  - [inconshreveable/go-update](https://github.com/inconshreveable/go-update)
- GitHub API: https://docs.github.com/en/rest/releases/releases
- GitHub Packages: https://docs.github.com/en/packages
