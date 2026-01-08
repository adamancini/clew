# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**clew** is a declarative Claude Code configuration manager - like Brewfile for Homebrew, but for Claude Code plugins, marketplaces, and MCP servers. It reads a Clewfile describing desired state and reconciles the system to match.

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
    ├── cmd/              # Cobra commands (root, sync, diff, export, status, add, remove)
    ├── config/           # Clewfile parsing, location resolution, scope inference
    ├── state/            # Current state detection via Claude CLI or filesystem
    ├── diff/             # Compute differences between desired and current state
    ├── sync/             # Reconciliation logic to apply changes
    └── output/           # Formatters for text/json/yaml output
```

### Data Flow

1. **config** - Load and parse Clewfile (YAML/TOML/JSON)
2. **state** - Read current state via `CLIReader` (claude CLI) or `FilesystemReader` (-f flag)
3. **diff** - Compare Clewfile against current state, produce action list
4. **sync** - Execute actions: add marketplaces first (plugins depend on them), then plugins, then MCP servers
5. **output** - Format results for display

### Key Types

- `config.Clewfile` - Parsed configuration with Marketplaces, Plugins, MCPServers
- `state.State` - Current system state with same structure
- `diff.Result` - List of MarketplaceDiff, PluginDiff, MCPServerDiff with Actions
- `sync.Result` - Counts of installed/updated/skipped/failed plus attention items

### State Detection Strategy

Two readers in `internal/state/`:
- `FilesystemReader` (default) - Reads `~/.claude/plugins/` JSON files directly (stable)
- `CLIReader` (--cli flag) - Invokes `claude plugin marketplace list` and `claude mcp list` (experimental, currently broken - see issue #34)

The filesystem reader is now the default because it's more reliable and doesn't depend on the Claude CLI's human-readable output parsing.

### Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| Sync behavior | Non-destructive | Items not in Clewfile are reported, not removed |
| Scope inference | From Clewfile location | `~/` or `~/.config/` = user scope; project dir = project scope |
| OAuth MCP servers | Skipped with info | Cannot be automated; require manual `/mcp` setup |
| Git status checking | Local repos checked | Skips sync if uncommitted changes in local marketplaces/plugins |
| Auto-backup | Enabled by default on sync | Creates backup before changes; use --no-backup to skip |
| Interactive mode | Available for sync/diff | Approve each change individually with -i/--interactive flag |
| Exit codes | 0=success, 1=failure, 2=strict mode failure | Partial success exits 0 unless --strict |

## Implementation Status

**✅ Fully Implemented:**

### Core Functionality
- ✅ Config parsing (YAML/TOML/JSON) with environment variable expansion
- ✅ FilesystemReader - reads state from `~/.claude/` JSON files (default, stable)
- ✅ CLIReader - parses `claude plugin marketplace list` and `claude mcp list` output (experimental, issue #34)
- ✅ Diff computation - compares Clewfile against current state
- ✅ Sync execution - installs/updates marketplaces, plugins, and MCP servers via claude CLI
- ✅ Command runner abstraction - allows mocking for tests

### Commands
- ✅ `clew init` - create Clewfile from templates (minimal, developer, full, or URL)
- ✅ `clew sync` - reconcile system to match Clewfile (with auto-backup)
- ✅ `clew diff` - dry-run preview of changes
- ✅ `clew export` - export current state to Clewfile format
- ✅ `clew status` - show current configuration status
- 🚧 `clew add` - add marketplace to Clewfile (plugin/MCP support pending)
- 🚧 `clew remove` - remove marketplace from Clewfile (plugin/MCP support pending)
- ✅ `clew completion` - shell completion (bash/zsh/fish)
- ✅ `clew backup` - backup/restore functionality
  - `create` - create backup snapshot
  - `list` - list all backups
  - `restore` - restore from backup
  - `delete` - delete specific backup
  - `prune` - remove old backups

### Features
- ✅ Interactive mode (`-i/--interactive`) for sync and diff
- ✅ Git status awareness - skips local repos with uncommitted changes
- ✅ Auto-backup before sync (configurable with --backup/--no-backup)
- ✅ Multiple output formats (text, json, yaml)
- ✅ Environment variable expansion (`${VAR}` and `${VAR:-default}`)
- ✅ Flexible plugin format (string or object with enabled/scope)
- ✅ OAuth MCP server detection and skip
- ✅ Scope support (user/project) for plugins and MCP servers
- ✅ `--show-commands` flag to display CLI reconciliation commands
- ✅ Comprehensive e2e test suite
- ✅ JSON Schema for IDE validation and auto-completion

### Plugin Integration
- ✅ Claude Code plugin structure (`.claude-plugin/plugin.json`)
- ✅ SessionStart hook for auto-execution
- ✅ Skill for user-invokable `/clew` command
- ✅ Multi-platform binaries (darwin/linux, amd64/arm64)

**⚠️ Known Issues:**
- CLIReader is experimental and currently broken (issue #34) - use filesystem reader (default)

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
- Allowed enum values (transport types, source types, scopes)
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
| Plugin scopes | `validatePlugin()` | `plugins.*.scope.enum` |
| MCP transports | `validateMCPServer()` | `mcp_servers.*.transport.enum` |
| MCP scopes | `validateMCPServer()` | `mcp_servers.*.scope.enum` |
