---
name: clew
description: This skill should be used when the user asks to "sync my Claude Code config", "check my plugin configuration", "manage my Claude Code plugins", "sync plugins across machines", "export my plugin state", "check clew status", "reconcile my Clewfile", or mentions declarative Claude Code configuration management.
version: 1.0.0
---

# Clew Configuration Management Skill

## Overview

Clew is a declarative Claude Code configuration manager - like Brewfile for Homebrew. It reads a Clewfile describing desired state (plugins, marketplaces, MCP servers) and reconciles the system to match. This skill enables conversational management of Claude Code configuration.

## When to Use This Skill

Activate this skill when users want to:
- Sync their Claude Code configuration across machines
- Check the status of their plugins, marketplaces, or MCP servers
- Export their current configuration to a Clewfile
- Compare desired state (Clewfile) with current state
- Add or remove plugins, marketplaces, or MCP servers

## Prerequisites

Verify the clew binary is available before proceeding:

```bash
command -v clew >/dev/null 2>&1
```

If clew is not found, inform the user:
- "The clew binary is not installed or not in PATH"
- "Install with: `go install github.com/adamancini/clew@latest`"
- "Or download from GitHub releases"

## Core Workflow

### Step 1: Assess Current State

Run status command to understand the current configuration:

```bash
clew status -o json
```

Parse the JSON output to understand:
- **Clewfile location**: Where the configuration file was found
- **Marketplaces**: Registered plugin sources (GitHub repos, local paths)
- **Plugins**: Installed plugins with enabled/disabled state
- **MCP Servers**: Configured Model Context Protocol servers
- **Discrepancies**: Items in Clewfile but not installed, or vice versa

### Step 2: Present Findings Conversationally

Format the status for readability. Use tables for comparing state:

**Marketplaces:**
| Name | Source | Status |
|------|--------|--------|
| claude-plugins-official | github:anthropics/claude-plugins-official | Registered |
| superpowers-marketplace | github:obra/superpowers-marketplace | Registered |

**Plugins:**
| Plugin | Marketplace | Desired | Current | Action |
|--------|-------------|---------|---------|--------|
| context7 | claude-plugins-official | enabled | enabled | Up to date |
| linear | claude-plugins-official | disabled | enabled | Will disable |
| pr-review-toolkit | claude-plugins-official | enabled | missing | Will install |

**MCP Servers:**
| Server | Transport | Status | Notes |
|--------|-----------|--------|-------|
| episodic-memory | stdio | Configured | |
| linear | http | Attention | Requires OAuth setup |

### Step 3: Highlight Items Needing Attention

Explicitly call out:
- **OAuth-required MCP servers**: Cannot be automated; user must run `/mcp` to configure
- **Missing marketplaces**: Must be added before plugins from them can install
- **Plugin state mismatches**: Will be reconciled during sync
- **Items not in Clewfile**: Present locally but not declared (will be reported but not removed)

### Step 4: Confirm Before Sync

Before running sync, summarize planned changes and ask for confirmation:

"I found the following changes to apply:
- Install 2 new plugins: pr-review-toolkit, hookify
- Disable 1 plugin: linear
- Add 1 marketplace: devops-toolkit

The following items require manual setup:
- MCP server 'linear' needs OAuth - run `/mcp` after sync

Proceed with sync?"

### Step 5: Execute Sync

After user confirmation, run:

```bash
clew sync -o json
```

Parse the result:
```json
{
  "status": "complete",
  "installed": 2,
  "updated": 1,
  "skipped": 0,
  "failed": 0,
  "attention": ["linear"]
}
```

### Step 6: Report Results

Present the sync results:

"Sync complete:
- 2 plugins installed successfully
- 1 plugin state updated
- 0 failures

Items needing attention:
- **linear** MCP server: Requires OAuth. Run `/mcp` and select 'linear' to complete setup."

## Command Reference

### Status Check
```bash
clew status -o json
```
Returns current state comparison with Clewfile.

### Show Differences (Dry Run)
```bash
clew diff -o json
```
Shows what would change without making changes.

### Sync Configuration
```bash
clew sync -o json
```
Reconciles system to match Clewfile.

### Export Current State
```bash
clew export -o yaml
```
Outputs current state as a Clewfile. Useful for initial setup:
```bash
clew export -o yaml > ~/.config/claude/Clewfile
```

### Add Items
```bash
# Add marketplace
clew add marketplace devops-toolkit --source=local --path=~/.claude/plugins/repos/devops-toolkit

# Add plugin
clew add plugin context7@claude-plugins-official

# Add MCP server
clew add mcp episodic-memory --transport=stdio --command=npx --args="-y,@anthropic/episodic-memory-mcp"
```

### Remove Items
```bash
clew remove plugin linear@claude-plugins-official
clew remove mcp episodic-memory
```

## Output Formats

Clew supports three output formats via `-o` flag:
- **text** (default): Human-readable with icons
- **json**: Machine-readable, best for skill parsing
- **yaml**: Structured, good for exports

Always use `-o json` when invoking clew from this skill for reliable parsing.

## Error Handling

### Clewfile Not Found
```json
{"error": "clewfile_not_found", "searched": ["~/.config/claude/Clewfile", "~/.claude/Clewfile", "~/.Clewfile"]}
```
Guide user to create one: `clew export -o yaml > ~/.config/claude/Clewfile`

### Parse Error
```json
{"error": "parse_error", "file": "~/.config/claude/Clewfile", "line": 15, "message": "invalid YAML"}
```
Help user fix syntax error at indicated location.

### Partial Failure
```json
{"status": "partial", "installed": 1, "failed": 1, "failures": [{"item": "broken-plugin@bad-marketplace", "error": "marketplace not found"}]}
```
Report successes and failures separately. Non-destructive by default.

## Clewfile Format Reference

```yaml
version: 1

marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official

  devops-toolkit:
    source: local
    path: ~/.claude/plugins/repos/devops-toolkit

plugins:
  # Simple form - installed & enabled
  - context7@claude-plugins-official
  - superpowers@superpowers-marketplace

  # Extended form - explicit options
  - name: linear@claude-plugins-official
    enabled: false
    scope: user

mcp_servers:
  episodic-memory:
    transport: stdio
    command: npx
    args: ["-y", "@anthropic/episodic-memory-mcp"]
    env:
      EMBEDDING_MODEL: ${EMBEDDING_MODEL:-text-embedding-3-small}

  # OAuth servers - skipped during sync with note
  linear:
    transport: http
    url: https://mcp.linear.app/mcp
```

## Workflow Examples

### Initial Setup on New Machine
1. User: "I just set up a new machine, help me sync my Claude Code config"
2. Check if clew binary exists
3. Check if Clewfile exists (should be synced via yadm/dotfiles)
4. Run `clew status -o json` to show what will be installed
5. Confirm with user
6. Run `clew sync -o json`
7. Report results and highlight OAuth servers needing manual setup

### Export Current Configuration
1. User: "Export my current Claude Code setup to a Clewfile"
2. Run `clew export -o yaml`
3. Present the output
4. Suggest saving to `~/.config/claude/Clewfile`
5. Remind to track with yadm: `yadm add ~/.config/claude/Clewfile`

### Check Status Only
1. User: "What's the status of my Claude Code config?"
2. Run `clew status -o json`
3. Present findings in tables
4. Highlight any discrepancies
5. Offer to sync if changes detected

## Best Practices

### Always Use JSON Output
Parse JSON output for reliable data extraction. Text output is for human display only.

### Confirm Before Sync
Always show planned changes and get user confirmation before running sync.

### Handle OAuth Servers Gracefully
OAuth-required MCP servers cannot be automated. Always inform users they need manual setup via `/mcp`.

### Non-Destructive by Default
Clew reports items not in Clewfile but does not remove them. This is intentional - users may have local-only plugins.

### Suggest yadm Integration
When users export or create Clewfiles, remind them to track with yadm for cross-machine sync.
