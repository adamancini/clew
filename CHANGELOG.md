# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.1] - 2026-01-16

### Changed

- Release workflow now includes both CHANGELOG content and git commit list in release notes
- Provides better release documentation with migration guides and detailed commit history

## [0.8.0] - 2026-01-15

### Breaking Changes

This release introduces a major schema change from `sources` array to `marketplaces` map format.

**Why this change?**
- Simplified configuration with direct name-to-config mapping
- Removed redundant indirection (`source.type`, `source.url` -> `url`)
- Eliminated unused fields (`kind`, `alias`, `ref`, `path`)
- Cleaner plugin references (`plugin@marketplace` works directly with map keys)

### Migration Guide

**Before (v0.7.x):**
```yaml
sources:
  - name: anthropics-marketplace
    alias: anthropics
    kind: marketplace
    source:
      type: github
      url: github.com/anthropics/claude-code-plugins

plugins:
  - plugin-dev@anthropics
```

**After (v0.8.0):**
```yaml
marketplaces:
  anthropics:
    url: github.com/anthropics/claude-code-plugins

plugins:
  - plugin-dev@anthropics
```

### Removed

- `SourceKind` type and constants (`marketplace`, `plugin`)
- `SourceType` type and constants (`github`)
- `Source.Kind` field - marketplaces map implies kind
- `Source.Type` field - only GitHub supported, now implicit
- `Source.Alias` field - map key serves as the alias
- `Source.Ref` field - unused, ref pinning not implemented
- `Source.Path` field - unused, subpath support not implemented
- `Source.Source` nested struct - flattened to direct `URL` field
- `validateSource()` and `validateSourceConfig()` functions
- `internal/types/source_kind.go` and `internal/types/source_type.go`

### Changed

- `Clewfile.Sources` (array) replaced with `Clewfile.Marketplaces` (map)
- `State.Sources` (array) replaced with `State.Marketplaces` (map)
- `Marketplace` struct simplified: only `Name` and `URL` fields
- `MarketplaceState` struct simplified: only `Name` and `URL` fields
- Plugin `@source` references now use marketplace map keys
- Diff computation updated for map-based marketplace comparison
- Sync executor updated for new marketplace structure
- Filesystem reader updated to build marketplace map from JSON files
- Export command updated to output new format
- All validation simplified - only URL validation needed

## [0.7.1] - 2026-01-15

### Fixed

- Plugin-kind sources (single-plugin GitHub repos) were being incorrectly skipped during sync
- Command generation was only including marketplace-kind sources, ignoring plugin-kind sources
- Both executor and command generation now properly handle both marketplace and plugin kinds

## [0.7.0] - 2026-01-14

### ⚠️ BREAKING CHANGES

- **Removed local plugin support**: The `source: local` type is no longer supported for plugins or marketplaces
  - Local plugin repositories (`~/.claude/plugins/repos/`) are no longer synced
  - Only GitHub sources (`source: github`) are supported
  - **Rationale**: Local plugins created version sync issues when plugin.json was bumped, breaking installations
  - **Migration**: Develop plugins in normal project directories (e.g., `~/projects/my-plugin/`), push to GitHub, and install via Clewfile with GitHub URL

### Removed

- `SourceTypeLocal` constant and validation
- `SourceKindLocal` constant and validation
- Local plugin installation logic from sync executor
- Local plugin reading from filesystem reader
- Local source examples from schema and documentation

### Changed

- `SourceType` now only supports `github` (was: `github` and `local`)
- `SourceKind` now only supports `marketplace` and `plugin` (was: `marketplace`, `plugin`, and `local`)
- Validation now rejects any `source.type` other than `github`
- Schema updated to remove `local` from allowed enum values

## [0.6.1] - 2026-01-13

### Changed
- Refactored codebase with new `internal/types` package for type-safe constants
- Improved validation using type methods instead of manual string comparisons
- Extracted SyncService for better separation of concerns and testability
- Reduced code complexity (sync.go: 237 → 91 lines, validate.go: ~60 lines removed)

### Added
- Type-safe constants with validation methods (SourceType, SourceKind, Scope, TransportType)
- Helper methods on types (IsGitHub(), RequiresCommand(), etc.)
- Comprehensive tests for SyncService (sync_service_test.go, sync_integration_test.go)
- Dependency injection support for better testing

## [0.6.0] - 2026-01-09

### Added
- Local repository plugin support with `source: local` format (#65)
- Direct editing of `installed_plugins.json` for local plugins
- Version detection from plugin.json for local plugins
- Git commit SHA tracking for local plugin installations

## [0.5.1] - 2026-01-09

### Fixed
- E2E tests updated for unified source model (fixes CI failures)
- Added validation for invalid source types (github/local only)
- Updated test fixtures to use sources format
- Fixed `known_marketplaces.json` test fixture to match actual Claude Code structure

### Removed
- `clew init` command - use `clew export` to generate Clewfile from existing setup
- Template files (minimal.yaml, developer.yaml, full.yaml)
- `internal/templates` package (1,076 lines removed)
- Unused `Marketplace` type from config package
- Duplicate state reader selection code across 5 command files

### Changed
- Simplified getting started workflow: run `clew export` instead of `clew init`
- Renamed git checker fields to use consistent Sources terminology
- Extracted `getStateReader()` helper to reduce code duplication
- Exported plugins now sorted by source name for better readability

## [0.5.0] - 2026-01-09

### Changed
- **BREAKING**: Replaced `marketplaces` field with unified `sources` field
- **BREAKING**: Sources now require `kind` field (marketplace, plugin, or local)
- **BREAKING**: Source configuration moved to nested `source` object with `type`, `url`, `ref`, and `path` fields
- Plugin `source` field changed from string to object for inline source definitions
- State detection now converts old marketplace format to unified sources format

### Added
- Source `alias` field for collision handling and short references
- Support for plugin-kind sources (standalone plugin repositories)
- Support for local-kind sources (already-installed plugins)
- Inline source syntax for one-off plugins
- Optional `@source` reference for plugins when names match

### Removed
- Legacy `marketplaces` field and associated types
- `Marketplace` and `MarketplaceState` types
- `validateMarketplace()` function

### Migration Guide

**Before (v0.x):**
```yaml
marketplaces:
  anthropics:
    source: github
    repo: github.com/anthropics/anthropic-marketplace

plugins:
  - plugin-dev@anthropics
```

**After (v1.0):**
```yaml
sources:
  - name: anthropics-marketplace
    alias: anthropics
    kind: marketplace
    source:
      type: github
      url: github.com/anthropics/anthropic-marketplace

plugins:
  - plugin-dev@anthropics
```

See issue #59 for full design rationale and examples.

## [0.4.4] - 2026-01-09

### Changed
- Updated CLAUDE.md with version bump validation system documentation

## [0.4.3] - 2026-01-09

### Fixed
- CHANGELOG date validation now properly checks format (YYYY-MM-DD pattern)

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
