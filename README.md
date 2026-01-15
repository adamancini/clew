# clew

[![CI](https://github.com/adamancini/clew/actions/workflows/ci.yml/badge.svg)](https://github.com/adamancini/clew/actions/workflows/ci.yml)
[![CodeQL](https://github.com/adamancini/clew/actions/workflows/codeql.yml/badge.svg)](https://github.com/adamancini/clew/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/adamancini/clew/branch/main/graph/badge.svg)](https://codecov.io/gh/adamancini/clew)

Declarative Claude Code configuration management - like Brewfile for Homebrew, but for Claude Code plugins, marketplaces, and MCP servers.

## The Problem

When syncing Claude Code config across workstations with yadm (or similar):
- `installed_plugins.json` references paths that don't exist on other machines
- Plugin cache directories aren't synced, so plugins appear "installed" but won't load
- You're syncing *state* rather than *intent*

## The Solution

Declare your desired configuration in a **Clewfile**, sync that across machines, run `clew sync`.

```yaml
# ~/.config/claude/Clewfile
# yaml-language-server: $schema=https://raw.githubusercontent.com/adamancini/clew/main/schema/clewfile.schema.json
---
version: 1

marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official

plugins:
  - context7@claude-plugins-official
  - superpowers@superpowers-marketplace
  - name: linear@claude-plugins-official
    enabled: false

mcp_servers:
  episodic-memory:
    transport: stdio
    command: npx
    args: ["-y", "@anthropic/episodic-memory-mcp"]
```

## Installation

```bash
go install github.com/adamancini/clew@latest
```

Or download a release binary:

```bash
curl -L https://github.com/adamancini/clew/releases/latest/download/clew-$(uname -s)-$(uname -m) \
  -o ~/.local/bin/clew && chmod +x ~/.local/bin/clew
```

## Quick Start

```bash
# Export your current Claude Code setup to a Clewfile
clew export > ~/.claude/Clewfile.yaml

# Sync another machine to match your Clewfile
clew sync

# Check what would change before syncing
clew diff
```

## Usage

```bash
# Export current state to Clewfile
clew export > ~/.claude/Clewfile.yaml

# Sync system to match Clewfile
clew sync

# Show what would change (dry-run)
clew diff

# Check status
clew status
```

### Create a Clewfile

**Option 1: Export from existing setup (recommended)**
```bash
clew export > ~/.claude/Clewfile.yaml
```

**Option 2: Write manually**
```yaml
version: 1

marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official

plugins:
  - context7@claude-plugins-official
  - episodic-memory@claude-plugins-official

mcp_servers: {}
```

### Interactive Mode

Use `--interactive` or `-i` to review and approve each change individually:

```bash
# Interactive sync - approve each change before applying
clew sync --interactive
clew sync -i

# Interactive diff - preview changes with prompts (dry-run)
clew diff --interactive
```

Interactive mode prompts for each marketplace, plugin, and MCP server change:

```
$ clew sync --interactive

Marketplaces:
  + private-marketplace (will add)
    -> Add private-marketplace from github:you/plugins? [y/n/a/q] y

Plugins:
  + pr-review-toolkit@claude-plugins-official (will add)
    -> Add pr-review-toolkit@claude-plugins-official? [y/n/a/q] y

  - linear@claude-plugins-official (will disable)
    -> Disable linear@claude-plugins-official? [y/n/a/q] n
    - Skipped

MCP Servers:
  + filesystem (will add)
    -> Add MCP server filesystem? [y/n/a/q] y

Summary:
  Will apply: 3 changes
  Skipped: 1

Proceed with sync? [y/n] y

Installed: 3
```

**Prompt options:**
- `y` - Yes, approve this change
- `n` - No, skip this change
- `a` - All, approve all remaining changes
- `q` - Quit, abort interactive mode

**Non-TTY fallback:** When not running in a terminal (e.g., in scripts or CI), interactive mode automatically falls back to non-interactive mode with a warning.

### Output Modes

By default, sync shows verbose output with commands and descriptions:

```
$ clew sync

Add: Installing plugin context7
→ claude plugin install context7@claude-plugins-official
✓ Success

Enable: Enabling plugin linear
→ claude plugin enable linear@claude-plugins-official
✓ Success

Summary:
  Installed: 1
  Updated: 1
  Failed: 0
```

Use `--short` for concise one-line-per-item output:

```
$ clew sync --short

✓ context7 (plugin add)
✓ linear (plugin enable)

Summary: 1 installed, 1 updated
```

The short format is ideal for scripts and CI pipelines where you want minimal output.

## Backup and Restore

clew can backup your Claude Code configuration before making changes, allowing easy rollback if something goes wrong.

```bash
# Create a backup
clew backup create
clew backup create --note "Before plugin update"

# List all backups
clew backup list
clew backup list -o json

# Restore from a backup
clew backup restore <id>
clew backup restore latest

# Delete a specific backup
clew backup delete <id>

# Remove old backups (keep last N)
clew backup prune --keep=10
```

### Auto-Backup on Sync

By default, `clew sync` creates a backup before making changes:

```bash
# Default behavior - backup created automatically
clew sync

# Explicitly request backup
clew sync --backup

# Skip backup creation
clew sync --no-backup
```

### Backup Storage

Backups are stored in `~/.cache/clew/backups/` as JSON files named with timestamps (e.g., `2024-01-08-143022.json`).

Each backup contains:
- Timestamp and optional note
- clew version that created the backup
- Complete state: marketplaces, plugins, and MCP servers

### Flags

```bash
-o, --output <format>       # Output: text (default), json, yaml
-f, --filesystem            # Read state from files instead of claude CLI
-i, --interactive           # Interactive mode (sync/diff only)
--config <path>             # Explicit Clewfile path
--strict                    # Exit non-zero on any failure (sync only)
--short                     # One-line per item output (sync only)
--verbose                   # Detailed output
--quiet                     # Errors only
```

## Shell Completion

clew supports shell completion for bash, zsh, and fish.

### Installation

**Bash:**

```bash
# Linux (system-wide)
clew completion bash | sudo tee /etc/bash_completion.d/clew > /dev/null

# macOS (Homebrew)
clew completion bash > $(brew --prefix)/etc/bash_completion.d/clew

# Load in current session
source <(clew completion bash)
```

**Zsh:**

```bash
# Oh My Zsh
mkdir -p ~/.oh-my-zsh/completions
clew completion zsh > ~/.oh-my-zsh/completions/_clew

# Standard zsh (add to fpath)
clew completion zsh > /usr/local/share/zsh/site-functions/_clew

# Load in current session
source <(clew completion zsh)
```

**Fish:**

```bash
clew completion fish > ~/.config/fish/completions/clew.fish
```

After installation, restart your shell or source the completion script.

## Clewfile Location

clew searches (first found wins):
1. `--config` flag or `CLEWFILE` env var
2. `$XDG_CONFIG_HOME/claude/Clewfile[.yaml|.toml|.json]`
3. `~/.claude/Clewfile[.yaml|.toml|.json]`
4. `~/.Clewfile[.yaml|.toml|.json]`

Supports YAML, TOML, and JSON formats (auto-detected by extension).

## IDE Support

clew includes a [JSON Schema](schema/clewfile.schema.json) for Clewfile validation and auto-completion:

**YAML files** - Add schema reference at the top:
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/adamancini/clew/main/schema/clewfile.schema.json
---
version: 1
```

**JSON files** - Add `$schema` property:
```json
{
  "$schema": "https://raw.githubusercontent.com/adamancini/clew/main/schema/clewfile.schema.json",
  "version": 1
}
```

See [schema/README.md](schema/README.md) for IDE setup details and examples.

## Documentation

See [docs/design.md](docs/design.md) for full architecture and specification.

## Contributing

All pull requests to `main` require a version bump. Before creating a PR:

1. Update version in `.claude-plugin/plugin.json`
2. Add entry to `CHANGELOG.md` with new version and changes
3. Ensure version is greater than latest git tag

See [.github/WORKFLOWS.md](./.github/WORKFLOWS.md#version-bump-validation-workflow) for detailed workflow documentation.

## License

MIT
