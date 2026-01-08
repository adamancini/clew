# Clew as a Claude Code Plugin

Clew can be distributed as a Claude Code plugin, providing:
- Pre-built binaries for all platforms (darwin/linux, arm64/amd64)
- Automatic binary installation on SessionStart
- AI-friendly skill for conversational config management

## Plugin Structure

```
clew/
├── .claude-plugin/
│   └── plugin.json          # Plugin manifest
├── skills/
│   └── clew/
│       └── SKILL.md         # AI-friendly skill for config management
├── hooks/
│   ├── hooks.json           # SessionStart hook configuration
│   └── session_start.sh     # Binary installation script
└── bin/                     # Pre-built binaries (built with `make plugin`)
    ├── clew-darwin-arm64
    ├── clew-darwin-amd64
    ├── clew-linux-amd64
    └── clew-linux-arm64
```

## Building the Plugin

```bash
# Build plugin binaries for all platforms
make plugin

# Or just build the binaries
make plugin-binaries
```

This creates the `bin/` directory with cross-compiled binaries.

## Installation Methods

### Method 1: From GitHub (Future)

Once published to a marketplace:

```bash
claude plugin install clew@marketplace-name
```

### Method 2: Local Development

Test the plugin locally:

```bash
# From the clew repository directory
claude --plugin-dir .
```

Or add to your local plugins:

```bash
# Copy to Claude plugins directory
cp -r . ~/.claude/plugins/repos/clew

# Add to installed_plugins.json or use claude plugin commands
```

### Method 3: Install Binary Only

If you only want the binary without the plugin:

```bash
# Via go install
go install github.com/adamancini/clew@latest

# Or download release binary
curl -L https://github.com/adamancini/clew/releases/latest/download/clew-$(uname -s)-$(uname -m) \
  -o ~/.local/bin/clew && chmod +x ~/.local/bin/clew
```

## How the Plugin Works

### SessionStart Hook

When a Claude Code session starts, the `session_start.sh` hook:

1. Detects your platform (darwin/linux) and architecture (arm64/amd64)
2. Creates `~/.local/bin` if it doesn't exist
3. Symlinks the appropriate binary: `bin/clew-<platform>-<arch>` -> `~/.local/bin/clew`
4. Makes the binary executable
5. Warns if `~/.local/bin` is not in your PATH

The hook is **idempotent** - safe to run multiple times. It will:
- Skip if already correctly installed
- Update the symlink if pointing to wrong binary
- Back up existing file if not a symlink

### Clew Skill

The skill enables conversational config management:

**Example interactions:**
- "Sync my Claude Code config" - runs status check and sync
- "What's the status of my plugins?" - shows comparison with Clewfile
- "Export my config to a Clewfile" - exports current state

The skill:
- Uses `clew status -o json` to assess state
- Presents findings in readable tables
- Confirms before making changes
- Highlights items needing manual attention (OAuth MCP servers)

## Testing the Plugin

### Test Hook Installation

```bash
# Simulate SessionStart hook
CLAUDE_PLUGIN_ROOT=$(pwd) bash hooks/session_start.sh

# Verify installation
which clew
clew version
```

### Test Skill Loading

```bash
# Start Claude Code with plugin loaded
claude --plugin-dir .

# Ask Claude to use the skill
> Check my plugin configuration status
```

### Test Full Workflow

```bash
# Create a test Clewfile
cat > ~/.config/claude/Clewfile.test << 'EOF'
version: 1
marketplaces:
  claude-plugins-official:
    source: github
    repo: anthropics/claude-plugins-official
plugins:
  - context7@claude-plugins-official
EOF

# Test with clew
clew status --config ~/.config/claude/Clewfile.test
```

## PATH Configuration

The SessionStart hook installs to `~/.local/bin`. Ensure this is in your PATH:

**bash (~/.bashrc or ~/.bash_profile):**
```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

**zsh (~/.zshrc):**
```zsh
export PATH="${HOME}/.local/bin:${PATH}"
```

**fish (~/.config/fish/config.fish):**
```fish
fish_add_path ~/.local/bin
```

## Troubleshooting

### Binary Not Found

If `clew` command is not found after plugin installation:

1. Check if `~/.local/bin` is in PATH: `echo $PATH | tr ':' '\n' | grep local`
2. Check if symlink exists: `ls -la ~/.local/bin/clew`
3. Check if binary exists: `ls -la ~/.claude/plugins/cache/*/clew/*/bin/`

### Wrong Platform Binary

If you get "binary cannot execute" errors:

1. Check detected platform: `uname -s` (should be Darwin or Linux)
2. Check detected arch: `uname -m` (should be x86_64, arm64, or aarch64)
3. Manually run hook to see detection: `bash hooks/session_start.sh`

### Hook Not Running

If the hook doesn't run on session start:

1. Verify hooks.json syntax: `cat hooks/hooks.json | jq .`
2. Check plugin is enabled: `/plugins` in Claude Code
3. Enable debug mode: `claude --debug`

## Contributing

### Adding New Platform Support

1. Add build target to Makefile:
   ```makefile
   GOOS=newos GOARCH=newarch go build $(LDFLAGS) -o bin/clew-newos-newarch ./cmd/clew
   ```

2. Update `session_start.sh` platform detection:
   ```bash
   newos) echo "newos" ;;
   ```

### Updating the Skill

Edit `skills/clew/SKILL.md` to:
- Add new command examples
- Update workflow instructions
- Improve error handling guidance

## License

MIT - see LICENSE file in repository root.
