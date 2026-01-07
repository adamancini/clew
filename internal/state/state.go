// Package state handles detection of current Claude Code configuration state.
package state

// State represents the current Claude Code configuration.
type State struct {
	Marketplaces map[string]MarketplaceState
	Plugins      map[string]PluginState
	MCPServers   map[string]MCPServerState
}

// MarketplaceState represents a marketplace's current state.
type MarketplaceState struct {
	Name            string
	Source          string
	Repo            string
	Path            string
	InstallLocation string
	LastUpdated     string
}

// PluginState represents a plugin's current state.
type PluginState struct {
	Name        string
	Marketplace string
	Scope       string
	Enabled     bool
	Version     string
	InstallPath string
}

// MCPServerState represents an MCP server's current state.
type MCPServerState struct {
	Name      string
	Transport string
	Command   string
	Args      []string
	URL       string
	Scope     string
}

// Reader defines the interface for reading current state.
type Reader interface {
	Read() (*State, error)
}

// CLIReader reads state by invoking claude CLI commands.
type CLIReader struct{}

// Read implements Reader using claude CLI.
func (r *CLIReader) Read() (*State, error) {
	// TODO: Implement CLI-based state reading
	// claude plugin list --json
	// claude mcp list --json
	return nil, nil
}

// FilesystemReader reads state directly from Claude Code's files.
type FilesystemReader struct {
	ClaudeDir string // typically ~/.claude
}

// Read implements Reader using filesystem access.
func (r *FilesystemReader) Read() (*State, error) {
	// TODO: Implement filesystem-based state reading
	// Read ~/.claude/plugins/installed_plugins.json
	// Read ~/.claude/plugins/known_marketplaces.json
	// Read ~/.claude.json for MCP servers
	return nil, nil
}
