# Marketplace Schema Simplification Design

**Date:** 2026-01-15
**Status:** Approved
**Version:** v0.8.0 (breaking change)

## Overview

Simplify the Clewfile schema by reverting from the `sources:` array model back to the `marketplaces:` map model, removing unnecessary complexity introduced in v0.5.0-v0.7.1.

## Motivation

The current `sources:` model has excess complexity:
- `SourceKind` enum (marketplace/plugin) - unnecessary distinction for MVP
- `SourceType` enum (github) - can infer from repo format
- Nested `source:` object - adds verbosity
- `name` + `alias` - redundant (map key can be the alias)

After removing local plugin support in v0.7.0, we're left with only GitHub-based sources, making the unified model overcomplicated for the current feature set.

## Requirements

**MVP (v0.8.0):**
- Support marketplace-based plugins (the opinionated way to manage plugins)
- Conform to Claude Code CLI: `claude plugin marketplace add <repo>`
- Simple, clear schema with minimal boilerplate

**Future (v0.9.0+):**
- Add `repos:` for standalone plugin repos (single-plugin repositories)
- No breaking changes when adding this feature

## Schema Design

### Marketplace Configuration

```yaml
marketplaces:
  <alias>:
    repo: <repository-url>
    ref: <optional-git-ref>
```

**Field Specifications:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `<alias>` | Map key | Yes | Short name for referencing in plugins (e.g., "superpowers") |
| `repo` | String | Yes | Repository in any format Claude CLI supports |
| `ref` | String | No | Git branch, tag, or SHA (omit for repo's default branch) |

**Supported `repo` Formats:**

1. **Short GitHub form:** `owner/repo` (assumes github.com)
2. **HTTPS URL:** `https://gitlab.com/company/plugins.git`
3. **SSH URL:** `git@github.com:owner/repo.git`
4. **Any valid git URL** - validation is lenient, git is the authority

**Validation:**
- `repo`: Non-empty string (git validates actual format)
- `ref`: Optional string (git validates ref exists)
- Map keys must be unique

### Plugin Configuration

```yaml
plugins:
  - <plugin>@<marketplace-alias>
  - name: <plugin>@<marketplace-alias>
    enabled: <true|false>
    scope: <user|project>
```

**Simple form (most common):**
```yaml
plugins:
  - context7@official
  - brainstorming@superpowers
```
Defaults: `enabled: true`, `scope: user`

**Extended form:**
```yaml
plugins:
  - name: linear@official
    enabled: false
  - name: hookify@official
    scope: project
```

**Validation:**
- Plugin name must match pattern: `^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+$`
- Marketplace alias must exist in `marketplaces:` section
- `scope` must be `user` or `project` (if specified)

### Complete Example

```yaml
version: 1

marketplaces:
  superpowers:
    repo: obra/superpowers-marketplace

  official:
    repo: anthropics/claude-plugins-official
    ref: v1.2.0

  gitlab-plugins:
    repo: https://gitlab.com/mycompany/plugins.git
    ref: main

plugins:
  - context7@official
  - name: linear@official
    enabled: false
  - name: brainstorming@superpowers
    scope: project

mcp_servers:
  filesystem:
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
```

## Changes from v0.7.1

### Removed

**Types (from internal/types/constants.go):**
- `SourceType` type and all methods (~67 lines)
- `SourceKind` type and all methods (~126 lines)
- All related tests (~200 lines)

**Structs (from internal/config/config.go):**
- `SourceConfig` struct
- `Source` struct
- `GetSourceByAliasOrName()` method

**Complexity:**
- Nested source configuration
- Kind/type validation logic
- Conditional logic based on kind
- Type inference from repo format

### Changed

**Config Structure:**
```go
// OLD
type Clewfile struct {
    Sources []Source  // Array
}

type Source struct {
    Name   string
    Alias  string
    Kind   SourceKind
    Source SourceConfig
}

// NEW
type Clewfile struct {
    Marketplaces map[string]Marketplace  // Map
}

type Marketplace struct {
    Repo string
    Ref  string
}
```

**Plugin Structure:**
```go
// OLD
type Plugin struct {
    Name    string
    Source  *SourceConfig  // Inline source for local plugins
    Enabled *bool
    Scope   string
}

// NEW
type Plugin struct {
    Name    string  // Must match plugin@marketplace format
    Enabled *bool
    Scope   string
}
```

### Migration Path

**v0.7.1 → v0.8.0:**

```yaml
# Before
sources:
  - name: superpowers-marketplace
    alias: superpowers
    kind: marketplace
    source:
      type: github
      url: obra/superpowers-marketplace

# After
marketplaces:
  superpowers:
    repo: obra/superpowers-marketplace
```

**Single-plugin repos (like devops-toolkit):**
- **v0.7.1**: Declared in `sources:` with `kind: plugin`
- **v0.8.0**: NOT in Clewfile - installed manually
- **v0.9.0**: Will be supported via `repos:` top-level key

## Implementation Changes

### Files Requiring Major Changes

1. **internal/types/constants.go**
   - Remove SourceType (~67 lines)
   - Remove SourceKind (~126 lines)
   - Keep only Scope and TransportType

2. **internal/config/config.go**
   - Replace `Sources []Source` with `Marketplaces map[string]Marketplace`
   - Remove Source, SourceConfig structs
   - Add simple Marketplace struct
   - Remove GetSourceByAliasOrName (map lookup instead)

3. **internal/config/parser.go**
   - Parse `marketplaces:` map instead of `sources:` array
   - Remove source parsing logic

4. **internal/config/validate.go**
   - Validate marketplaces map (simpler)
   - Remove kind/type validation
   - Just check: non-empty repo, valid scope

5. **internal/diff/compute.go**
   - `computeSourceDiffs()` → `computeMarketplaceDiffs()`
   - Compare desired map vs current map
   - No kind checks needed

6. **internal/diff/diff.go**
   - `SourceDiff` → `MarketplaceDiff`
   - Remove IsLocal() method on PluginDiff (always false now)
   - Simpler action logic

7. **internal/diff/commands.go**
   - Generate marketplace add commands (no kind checks)
   - Simpler command generation

8. **internal/sync/executor.go**
   - `addSource()` → `addMarketplace()`
   - Remove kind/type checks
   - Just pass repo to CLI

9. **internal/state/*.go**
   - Read marketplaces from `known_marketplaces.json`
   - Return `map[string]MarketplaceState` instead of `map[string]SourceState`

10. **schema/clewfile.schema.json**
    - `sources:` array → `marketplaces:` map
    - Remove kind/type enums
    - Simpler structure

### Test Files (~20 files)
- Update fixtures to use marketplaces map
- Remove source kind/type test cases
- Simpler test data

### Documentation
- Update CLAUDE.md
- Update README.md examples
- Update CHANGELOG.md with v0.8.0 breaking changes

## Estimated Scope

| Category | Files | Lines Changed |
|----------|-------|---------------|
| Core logic | 10 | ~500 |
| Test files | 20 | ~300 |
| Types removal | 2 | -193 |
| Documentation | 4 | ~100 |
| **Total** | **~36** | **~700 net** |

**Time estimate:** 2-3 hours focused work

## Future: Plugin Repos (v0.9.0)

When we add support for standalone plugin repositories:

```yaml
version: 1

marketplaces:
  superpowers:
    repo: obra/superpowers-marketplace

repos:                              # ← New top-level key
  devops-toolkit:
    repo: git@github.com:adamancini/devops-toolkit.git

  custom-plugin:
    repo: https://gitlab.com/me/plugin.git

plugins:
  - brainstorming@superpowers        # From marketplace
  - devops-toolkit                   # From repos (no @ needed)
  - custom-plugin                    # From repos
```

**Implementation path:**
1. Add `Repos map[string]Repository` to Clewfile
2. Repository struct identical to Marketplace (same fields)
3. Plugin resolution: if no `@`, check repos first, then marketplaces
4. No changes to existing marketplace logic
5. Non-breaking: existing Clewfiles continue working

## Design Validation Checklist

- ✅ Conforms to Claude Code CLI behavior
- ✅ Simpler than current design (fewer types, less nesting)
- ✅ Supports all git hosting providers (GitHub, GitLab, etc.)
- ✅ Flexible repo formats (short, HTTPS, SSH)
- ✅ Optional ref support (branches, tags, SHAs)
- ✅ Future-proof for plugin repos without breaking changes
- ✅ Lenient validation (git is authority)
- ✅ Clear migration path from v0.7.1

## Success Criteria

**Functionality:**
- [ ] Parse marketplaces map from YAML/TOML/JSON
- [ ] Validate marketplace configuration
- [ ] Compute diffs (add/update/remove marketplaces)
- [ ] Generate CLI commands for marketplace operations
- [ ] Execute sync to add/update marketplaces
- [ ] Export current state to new format

**Quality:**
- [ ] All existing tests pass (updated for new schema)
- [ ] Schema validation with JSON Schema
- [ ] Migration guide in CHANGELOG
- [ ] Updated documentation

**Verification:**
- [ ] `clew diff` shows marketplace changes correctly
- [ ] `clew sync` adds marketplaces via CLI
- [ ] `clew export` generates new format
- [ ] Invalid configs rejected with clear errors
