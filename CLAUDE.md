# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**clew** is a declarative Claude Code configuration manager - like Brewfile for Homebrew, but for Claude Code plugins, marketplaces, and MCP servers. It reads a Clewfile describing desired state and reconciles the system to match.

## Build Commands

```bash
make build              # Build binary to ./clew
make install            # Install to GOPATH/bin
make test               # Run all tests
make test-coverage      # Run tests with coverage report
make lint               # Run golangci-lint
make fmt                # Format code
make tidy               # Tidy go.mod
make build-all          # Cross-compile for darwin/linux (amd64/arm64)
make clean              # Remove build artifacts
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
- `CLIReader` (default) - Invokes `claude plugin list --json`, `claude mcp list --json`
- `FilesystemReader` (-f flag) - Reads `~/.claude/plugins/installed_plugins.json` directly

### Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| Sync behavior | Non-destructive | Items not in Clewfile are reported, not removed |
| Scope inference | From Clewfile location | `~/` or `~/.config/` = user scope; project dir = project scope |
| OAuth MCP servers | Skipped with info | Cannot be automated; require manual `/mcp` setup |
| Exit codes | 0=success, 1=failure, 2=strict mode failure | Partial success exits 0 unless --strict |

## Implementation Status

The scaffolding is complete with TODOs marking unimplemented sections:
- `config.Load()` - needs YAML/TOML/JSON parsing
- `state.CLIReader.Read()` - needs claude CLI invocation
- `state.FilesystemReader.Read()` - needs file parsing
- `diff.Compute()` - needs comparison logic
- `sync.addMarketplace/installPlugin/updatePluginState/addMCPServer` - need claude CLI calls

## Clewfile Locations (precedence order)

1. `--config` flag or `CLEWFILE` env var
2. `$XDG_CONFIG_HOME/claude/Clewfile[.yaml|.toml|.json]`
3. `~/.claude/Clewfile[.yaml|.toml|.json]`
4. `~/.Clewfile[.yaml|.toml|.json]`
