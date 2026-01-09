---
name: Support configuring plugin and marketplace auto-update behavior
about: Track future work for auto-update configuration
title: 'feat: Support configuring plugin and marketplace auto-update behavior'
labels: enhancement, future
---

## Problem

Currently, clew syncs plugins and marketplaces to match the Clewfile declaratively, but doesn't provide control over automatic updates of those components. Users may want to:

- Pin specific plugin versions to avoid breaking changes
- Configure update channels (stable/beta/latest)
- Control marketplace update frequency
- Disable auto-updates for specific plugins while keeping others current

## Proposed Solution

Add optional configuration fields to Clewfile for controlling update behavior:

### Plugin Version Pinning

```yaml
plugins:
  - name: context7@claude-plugins-official
    version: "1.2.3"  # Pin to specific version
    auto_update: false

  - name: superpowers@superpowers-marketplace
    version: "latest"  # Always update to latest (default)
    auto_update: true
```

### Marketplace Update Channels

```yaml
marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official
    branch: stable  # Track stable branch instead of main
    auto_update: daily  # Update frequency: never/manual/daily/weekly
```

## Implementation Approach

1. **Extend Clewfile schema** (`internal/config/`)
   - Add `version` field to plugin config
   - Add `branch` and `auto_update` fields to marketplace config
   - Update JSON schema for validation

2. **Version resolution logic** (`internal/sync/`)
   - Check if plugin has version pinning
   - Compare current vs. requested version
   - Skip update if version matches or auto_update disabled

3. **Marketplace update logic** (`internal/sync/`)
   - Respect `auto_update` frequency settings
   - Track last update time in state
   - Skip git pull if within update window

4. **Update schema** (`schema/clewfile.schema.json`)
   - Add new optional fields
   - Update examples with version pinning

## Example Use Cases

**Use Case 1: Pin production plugins**
```yaml
plugins:
  - name: production-plugin@internal
    version: "2.1.0"  # Known stable version
    auto_update: false
```

**Use Case 2: Beta testing**
```yaml
marketplaces:
  superpowers-marketplace:
    source: github
    repo: superpowers-dev/superpowers
    branch: beta  # Track beta branch
    auto_update: daily
```

**Use Case 3: Offline development**
```yaml
marketplaces:
  local-plugins:
    source: local
    path: ~/dev/plugins
    auto_update: never  # Never update local dev plugins
```

## Alternatives Considered

1. **Separate update command** - `clew update --plugins --marketplaces`
   - Pro: Explicit control over when updates happen
   - Con: Requires manual intervention, defeats declarative model

2. **Lock file** (like package-lock.json)
   - Pro: Reproducible installs across machines
   - Con: Additional complexity, another file to sync

3. **Global config** (~/.config/clew/config.yml)
   - Pro: Centralized update preferences
   - Con: Doesn't allow per-project customization

## Related

- Current sync behavior: Non-destructive (doesn't remove items not in Clewfile)
- See CLAUDE.md "Design Decisions" table for context
- Schema validation in `internal/config/validate.go` and `schema/clewfile.schema.json`

## References

- [Clewfile Schema](../../schema/clewfile.schema.json)
- [Sync Implementation](../../internal/sync/)
- [CLAUDE.md Design Decisions](../../CLAUDE.md#design-decisions)
