# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
