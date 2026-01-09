# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- `clew init` command - use `clew export` to generate Clewfile from existing setup
- Template files (minimal.yaml, developer.yaml, full.yaml)
- `internal/templates` package

### Changed
- Simplified getting started workflow: run `clew export` instead of `clew init`

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
