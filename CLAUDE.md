# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**clew** is a declarative Claude Code configuration manager - like Brewfile for Homebrew, but for Claude Code plugins and marketplaces. It reads a Clewfile describing desired state and reconciles the system to match.

## Build Commands

```bash
make build              # Build binary to ./clew
make install            # Install to GOPATH/bin
make test               # Run unit tests only
make test-unit          # Run unit tests only
make test-e2e           # Run e2e tests only
make test-all           # Run all tests (unit + e2e)
make test-coverage      # Run tests with coverage report
make test-e2e-verbose   # Run e2e tests with verbose output
make lint               # Run golangci-lint
make fmt                # Format code
make tidy               # Tidy go.mod
make build-all          # Cross-compile for darwin/linux (amd64/arm64)
make clean              # Remove build artifacts
make plugin-binaries    # Build binaries for plugin distribution
make plugin-clean       # Clean plugin binaries
make plugin             # Build complete plugin package
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/config/...
```

## Architecture

```
clew/
├── cmd/clew/main.go      # Entry point, version injection via ldflags
└── internal/
    ├── cmd/              # Cobra commands (root, sync, diff, export, status, backup, version, completion)
    ├── config/           # Clewfile parsing, location resolution, validation
    ├── types/            # Shared types and constants
    ├── state/            # Current state detection via filesystem reader
    ├── diff/             # Compute differences between desired and current state
    ├── sync/             # Reconciliation logic to apply changes
    ├── backup/           # Backup and restore functionality
    ├── interactive/      # Interactive approval prompts
    ├── git/              # Git status checking for local repos
    ├── output/           # Formatters for text/json/yaml output
    └── update/           # Self-update via GitHub releases
```

### Data Flow

1. **config** - Load and parse Clewfile (YAML/TOML/JSON)
2. **state** - Read current state via `FilesystemReader` (reads `~/.claude/plugins/` JSON files)
3. **diff** - Compare Clewfile against current state, produce action list
4. **sync** - Execute actions: add marketplaces first (plugins depend on them), then plugins
5. **output** - Format results for display

### Key Types

- `config.Clewfile` - Parsed configuration with Marketplaces and Plugins
- `state.State` - Current system state with same structure
- `diff.Result` - List of MarketplaceDiff and PluginDiff with Actions
- `sync.Result` - Counts of installed/updated/skipped/failed plus unmanaged items

### State Detection

Single reader in `internal/state/`:
- `FilesystemReader` - Reads `~/.claude/plugins/` JSON files directly (stable, reliable)

### Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| Sync behavior | Non-destructive | Items not in Clewfile are reported, not removed |
| Scope | User scope only | All plugins installed at user scope; project scope deferred to post-1.0 |
| Git status checking | Local repos checked | Skips sync if uncommitted changes in local marketplaces/plugins |
| Auto-backup | Enabled by default on sync | Creates backup before changes; use --no-backup to skip |
| Interactive mode | Available for sync/diff | Approve each change individually with -i/--interactive flag |
| Exit codes | 0=success, 1=failure, 2=strict mode failure | Partial success exits 0 unless --strict |
| Version management | Required for main branch PRs | All PRs require version bump in plugin.json and CHANGELOG.md |

## Implementation Status

### Core Functionality
- Config parsing (YAML/TOML/JSON) with environment variable expansion
- FilesystemReader - reads state from `~/.claude/plugins/` JSON files
- Diff computation - compares Clewfile against current state
- Sync execution - installs/updates marketplaces and plugins via claude CLI
- Command runner abstraction - allows mocking for tests

### Commands
- `clew sync` - reconcile system to match Clewfile (with auto-backup)
- `clew diff` - dry-run preview of changes
- `clew export` - export current state to Clewfile format
- `clew status` - show current configuration status
- `clew completion` - shell completion (bash/zsh/fish)
- `clew backup` - backup/restore functionality
  - `create` - create backup snapshot
  - `list` - list all backups
  - `restore` - restore from backup
  - `delete` - delete specific backup
  - `prune` - remove old backups
- `clew version` - version information and auto-update
  - `--check` - check for updates without installing
  - `--update` - download and install latest version

### Features
- Interactive mode (`-i/--interactive`) for sync and diff
- Git status awareness - skips local repos with uncommitted changes
- Auto-backup before sync (configurable with --backup/--no-backup)
- Multiple output formats (text, json, yaml)
- Environment variable expansion (`${VAR}` and `${VAR:-default}`)
- Flexible plugin format (string or object with enabled field)
- `--show-commands` flag to display CLI reconciliation commands
- Comprehensive e2e test suite
- JSON Schema for IDE validation and auto-completion
- Version bump validation system
  - Automated PR checks via GitHub Actions
  - Validation script (`scripts/check-version-bump.sh`)
  - CHANGELOG.md in Keep a Changelog format
  - Branch protection for main branch
  - Semantic version validation
- Self-update capability
  - Check for updates via GitHub releases API
  - Download and verify binaries with SHA256 checksums
  - Safe binary replacement with automatic rollback
  - Support for all platforms (darwin/linux, amd64/arm64)

### Plugin Integration
- Claude Code plugin structure (`.claude-plugin/plugin.json`)
- SessionStart hook for auto-execution
- Skill for user-invokable `/clew` command
- Multi-platform binaries (darwin/linux, amd64/arm64)

## Clewfile Locations (precedence order)

1. `--config` flag or `CLEWFILE` env var
2. `$XDG_CONFIG_HOME/claude/Clewfile[.yaml|.toml|.json]`
3. `~/.claude/Clewfile[.yaml|.toml|.json]`
4. `~/.Clewfile[.yaml|.toml|.json]`

## Schema Maintenance

Validation rules exist in two places that must stay synchronized:

| File | Purpose |
|------|---------|
| `internal/config/validate.go` | Runtime validation (Go code) |
| `schema/clewfile.schema.json` | IDE validation (JSON Schema) |

### When to Update Both Files

Update both files when changing:
- Allowed enum values (source types)
- Required fields for any configuration type
- Field patterns or formats
- New configuration options

### Update Checklist

When modifying validation rules:

- [ ] Update `internal/config/validate.go` with new validation logic
- [ ] Update `schema/clewfile.schema.json` with matching constraints:
  - Update `enum` arrays for new allowed values
  - Update `oneOf` blocks for conditional requirements
  - Update `description` fields to document changes
- [ ] Update `schema/examples/advanced.yaml` with examples of new features
- [ ] Run `make test` to verify Go validation
- [ ] Validate schema is valid JSON: `python3 -m json.tool schema/clewfile.schema.json > /dev/null`

### Current Synced Rules

| Rule | Go Location | Schema Location |
|------|-------------|-----------------|
| Marketplace sources | `validateMarketplace()` | `marketplaces.*.source.enum` |
| Plugin format | `validatePlugin()` | `plugins` array items |

## Version Bump Validation

All PRs to `main` require version bumps. This is enforced by automated validation.

### Components

| Component | Purpose |
|-----------|---------|
| `scripts/check-version-bump.sh` | Bash validation script |
| `.github/workflows/version-check.yml` | GitHub Actions workflow |
| `CHANGELOG.md` | Keep a Changelog format |
| Branch protection | Requires "Check Version Bump" status |

### Validation Checks

The validation script performs four checks:

1. **Version match**: plugin.json version equals CHANGELOG.md top entry
2. **Version bump**: New version > latest git tag (semantic versioning)
3. **Tag uniqueness**: No git tag exists for new version
4. **Date format**: CHANGELOG.md entry has valid date (YYYY-MM-DD)

### Workflow

```bash
# 1. Bump version in plugin.json
jq '.version = "0.5.0"' .claude-plugin/plugin.json > tmp && mv tmp .claude-plugin/plugin.json

# 2. Update CHANGELOG.md
## [0.5.0] - 2026-01-09
### Added
- Feature description

# 3. Test locally
bash scripts/check-version-bump.sh

# 4. Create PR - validation runs automatically
# 5. After merge, tag triggers release
git tag v0.5.0 && git push origin v0.5.0
```

### Exit Codes

| Code | Meaning | Fix |
|------|---------|-----|
| 0 | All checks passed | Ready to merge |
| 1 | Version mismatch | Sync plugin.json and CHANGELOG.md |
| 2 | Version not bumped | Increase version above latest tag |
| 3 | Tag already exists | Use a higher version number |
| 4 | Missing/invalid date | Add YYYY-MM-DD date to CHANGELOG |
| 5 | Missing dependency | Install jq or git |

### References

- Design: `docs/plans/2026-01-09-version-bump-validation-design.md`
- Workflow docs: `.github/WORKFLOWS.md#version-bump-validation-workflow`
- Contributing: See README.md Contributing section
