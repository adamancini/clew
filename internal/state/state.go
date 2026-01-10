// Package state handles detection of current Claude Code configuration state.
package state

// State represents the current Claude Code configuration.
type State struct {
	Sources    map[string]SourceState
	Plugins    map[string]PluginState
	MCPServers map[string]MCPServerState
}

// SourceState represents a source's current state.
type SourceState struct {
	Name            string
	Kind            string // marketplace, plugin, local
	Type            string // github, local
	URL             string
	Ref             string
	Path            string
	InstallLocation string
	LastUpdated     string
}

// PluginState represents a plugin's current state.
type PluginState struct {
	Name         string
	Marketplace  string
	Scope        string
	Enabled      bool
	Version      string
	InstallPath  string
	IsLocal      bool   // True for local repository plugins (not marketplace)
	GitCommitSha string // Git commit SHA for the plugin
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

// FilesystemReader reads state directly from Claude Code's files.
type FilesystemReader struct {
	ClaudeDir string // typically ~/.claude
}
