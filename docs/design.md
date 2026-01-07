# Clew: Declarative Claude Code Configuration Management

## Overview

**clew** is a Go binary that manages Claude Code plugins, marketplaces, and MCP servers declaratively. It reads a Clewfile describing desired state and reconciles the system to match—like Brewfile for Homebrew.

### The Problem

When syncing Claude Code config via yadm across workstations:
- `installed_plugins.json` references paths that don't exist on other machines
- Plugin cache directories aren't synced, so plugins appear "installed" but won't load
- You're syncing *state* (what's installed) rather than *intent* (what should be installed)

### The Solution

Declare desired state in a Clewfile, sync that via yadm, run `clew sync` on each machine.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User                                │
│                           │                                 │
│              ┌────────────┴────────────┐                    │
│              ▼                         ▼                    │
│      $ clew sync              Claude Code Skill             │
│      $ clew export            (wraps clew -o json)          │
│              │                         │                    │
│              └────────────┬────────────┘                    │
│                           ▼                                 │
│                    ┌─────────────┐                          │
│                    │  clew CLI   │  ← Go binary             │
│                    └─────────────┘                          │
│                           │                                 │
│              ┌────────────┼────────────┐                    │
│              ▼            ▼            ▼                    │
│         Clewfile     claude CLI    State Files              │
│         (source)     (preferred)   (fallback)               │
└─────────────────────────────────────────────────────────────┘
```

**Components:**
- **clew binary** - Core logic, cross-platform (darwin/linux, amd64/arm64)
- **Clewfile** - Declarative config in YAML, TOML, or JSON
- **Claude CLI** - Primary interface for state detection and mutations
- **Claude Code Skill** - AI-friendly wrapper invoking `clew -o json`

## Clewfile Format

### Location Precedence

First found wins:
1. `--config` flag or `CLEWFILE` env var
2. `$XDG_CONFIG_HOME/claude/Clewfile[.yaml|.toml|.json]`
3. `~/.claude/Clewfile[.yaml|.toml|.json]`
4. `~/.Clewfile[.yaml|.toml|.json]`

### Format Detection

- By extension if present (`.yaml`, `.toml`, `.json`)
- Extensionless: try YAML → TOML → JSON

### Example (YAML)

```yaml
version: 1

marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official

  superpowers-marketplace:
    source: github
    repo: obra/superpowers-marketplace

  claude-code-workflows:
    source: github
    repo: wshobson/agents

  claude-code-plugins:
    source: github
    repo: anthropics/claude-code

  devops-toolkit:
    source: local
    path: ~/.claude/plugins/repos/devops-toolkit

plugins:
  # Simple form - installed & enabled, scope inferred from Clewfile location
  - context7@claude-plugins-official
  - superpowers@superpowers-marketplace
  - episodic-memory@superpowers-marketplace
  - feature-dev@claude-plugins-official
  - commit-commands@claude-plugins-official
  - pr-review-toolkit@claude-plugins-official
  - hookify@claude-plugins-official

  # Extended form - explicit options
  - name: linear@claude-plugins-official
    enabled: false
    scope: user

  - name: playwright@claude-plugins-official
    enabled: false

  - name: devops-toolkit@devops-toolkit
    scope: user

mcp_servers:
  episodic-memory:
    transport: stdio
    command: npx
    args: ["-y", "@anthropic/episodic-memory-mcp"]
    env:
      EMBEDDING_MODEL: ${EMBEDDING_MODEL:-text-embedding-3-small}

  # OAuth servers - will be skipped with note during sync
  linear:
    transport: http
    url: https://mcp.linear.app/mcp
```

### Scope Inference

- Clewfile in `~/.config/` or `~/` → default `user` scope
- Clewfile in project directory → default `project` scope
- Explicit `scope:` in entry overrides inference

## CLI Interface

### Commands

```bash
# Core operations
clew sync              # Reconcile system to match Clewfile
clew diff              # Show what would change (dry-run)
clew export            # Output current state as Clewfile
clew status            # Show sync status summary

# Management
clew add marketplace <name> --source=github --repo=owner/repo
clew add plugin <plugin@marketplace> [--enabled=false] [--scope=user]
clew add mcp <name> --transport=stdio --command=... [--args=...]
clew remove plugin <plugin@marketplace>
clew remove mcp <name>
```

### Global Flags

```bash
-o, --output <format>              # Output format: text (default), json, yaml
-f, --filesystem,
    --read-from-filesystem         # Read state from files instead of claude CLI
--config <path>                    # Explicit Clewfile path
--verbose                          # Detailed progress output
--quiet                            # Errors only
--strict                           # Exit non-zero on any failure
```

### Output Examples

**Human-readable (default):**
```
$ clew sync
✓ Marketplace claude-plugins-official (up to date)
✓ Marketplace superpowers-marketplace (up to date)
✓ Plugin context7@claude-plugins-official (up to date)
+ Plugin pr-review-toolkit@claude-plugins-official (installing...)
✓ Plugin pr-review-toolkit@claude-plugins-official (installed)
ℹ Plugin playwright@claude-plugins-official (not in Clewfile, ignoring)
✓ MCP server episodic-memory (configured)
ℹ MCP server linear requires OAuth - configure via /mcp

Sync complete: 1 installed, 0 removed, 1 needs attention
```

**JSON output:**
```bash
$ clew sync -o json
{"status":"complete","installed":1,"removed":0,"attention":["linear"]}
```

## Sync Behavior

### Algorithm

```
1. Load Clewfile (find location, parse format)
2. Get current state (claude CLI or filesystem with -f)
3. Compute diff:
   - Marketplaces: missing, present, extra
   - Plugins: missing, present, extra, enabled mismatch
   - MCP servers: missing, present, extra, config mismatch
4. Apply changes (fast, resilient):
   - Add missing marketplaces first (plugins depend on them)
   - Install missing plugins
   - Configure missing MCP servers (skip OAuth, report)
   - Update enabled/disabled state
5. Report results
```

### Failure Handling

Prioritizes speed and resilience:
- Failures are logged and skipped, sync continues
- OAuth-requiring MCP servers are skipped with info message
- Partial success still exits 0
- Use `--strict` for exit 1 on any failure

### Conflict Resolution

Plugins installed locally but not in Clewfile:
- Reported at INFO level
- Not removed (non-destructive)
- User can add to Clewfile or uninstall manually

### Exit Codes

- `0` - Success (including partial success with warnings)
- `1` - Complete failure (Clewfile not found, parse error)
- `2` - `--strict` mode and something failed

## Secrets Handling

**Current approach:**
- Environment variable references: `${VAR}` or `${VAR:-default}`
- OAuth-requiring MCP servers skipped during sync with note

**Future possibilities:**
- Separate `Clewfile.secrets.yaml` (gitignored)
- System keychain integration

## Distribution

### Initial (go install)

```bash
go install github.com/adamancini/clew@latest
```

### Release Binary

```bash
curl -L https://github.com/adamancini/clew/releases/latest/download/clew-$(uname -s)-$(uname -m) \
  -o ~/.local/bin/clew && chmod +x ~/.local/bin/clew
```

### Future Plugin

```
clew-plugin/
├── .claude-plugin/
│   └── plugin.json
├── skills/
│   └── clew/
│       └── SKILL.md
├── bin/
│   ├── clew-darwin-arm64
│   ├── clew-darwin-amd64
│   └── clew-linux-amd64
└── hooks/
    └── hooks.json          # Post-install: symlink correct binary
```

## Claude Code Skill

```markdown
# ~/.claude/skills/clew/SKILL.md
---
name: clew
description: Manage Claude Code plugins, marketplaces, and MCP servers declaratively
---

When user asks to sync, check, or manage their Claude Code configuration:

1. Run `clew status -o json` to assess current state
2. Present findings conversationally
3. Run `clew sync -o json` if user confirms
4. Report results, highlight items needing manual attention
```

## Workflow

### Initial Setup (primary workstation)

```bash
# Export current state
clew export -o yaml > ~/.config/claude/Clewfile

# Track in yadm
yadm add ~/.config/claude/Clewfile
yadm commit -m "Add Clewfile for Claude Code config"
yadm push
```

### Sync on Other Workstations

```bash
yadm pull
clew sync
```

### Adding New Plugins

```bash
# Install via Claude Code as normal
/plugin install some-plugin@marketplace

# Update Clewfile
clew export -o yaml > ~/.config/claude/Clewfile

# Or add directly
clew add plugin some-plugin@marketplace
```

## What to Track in yadm

**Track (sync across machines):**
- `~/.config/claude/Clewfile`
- `~/.claude/settings.json` (non-plugin settings)
- `~/.claude/CLAUDE.md`
- `~/.claude/plugins/repos/devops-toolkit/` (personal plugin repo)

**Don't track (regenerated by clew sync):**
- `~/.claude/plugins/installed_plugins.json`
- `~/.claude/plugins/known_marketplaces.json`
- `~/.claude/plugins/cache/`
- `~/.claude/plugins/marketplaces/`
- `~/.claude/plugins/install-counts-cache.json`

## Project Structure

```
clew/
├── cmd/clew/
│   └── main.go
├── internal/
│   ├── config/       # Clewfile parsing (yaml/toml/json)
│   ├── state/        # Current state detection (CLI + filesystem)
│   ├── diff/         # State comparison
│   ├── sync/         # Reconciliation logic
│   └── output/       # Formatters (text/json/yaml)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Design Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, cross-platform, no runtime deps |
| Output format | Human default, `-o` flag | kubectl pattern, familiar |
| Clewfile location | XDG first with fallbacks | Standards-compliant, flexible |
| Clewfile format | YAML/TOML/JSON | User preference |
| State detection | Claude CLI default | Stable interface, `-f` escape hatch |
| Sync behavior | Non-destructive | Safe, report extras at info level |
| Secrets | Environment variables | Simple, integrates with existing setup |
| Scope inference | From Clewfile location | Smart defaults, explicit override |
| Export | Complete snapshot | User can prune afterward |
