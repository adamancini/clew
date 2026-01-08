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

## Usage

```bash
# Sync system to match Clewfile
clew sync

# Show what would change (dry-run)
clew diff

# Export current state to Clewfile
clew export -o yaml > ~/.config/claude/Clewfile

# Check status
clew status
```

### Flags

```bash
-o, --output <format>       # Output: text (default), json, yaml
-f, --filesystem            # Read state from files instead of claude CLI
--config <path>             # Explicit Clewfile path
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

## License

MIT
