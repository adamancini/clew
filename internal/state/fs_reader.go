// Package state handles detection of current Claude Code configuration state.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamancini/clew/internal/types"
)

// fsMarketplaceEntry represents a single marketplace in known_marketplaces.json.
type fsMarketplaceEntry struct {
	Source struct {
		Source string `json:"source"` // "github" or "local"
		Repo   string `json:"repo,omitempty"`
		Path   string `json:"path,omitempty"`
	} `json:"source"`
	InstallLocation string `json:"installLocation"`
	LastUpdated     string `json:"lastUpdated"`
}

// fsInstalledPlugins represents the structure of installed_plugins.json.
type fsInstalledPlugins struct {
	Version int                            `json:"version"`
	Plugins map[string][]fsPluginInstall   `json:"plugins"`
}

type fsPluginInstall struct {
	Scope        string `json:"scope"`
	ProjectPath  string `json:"projectPath,omitempty"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha,omitempty"`
}

// fsSettings represents the relevant parts of settings.json.
type fsSettings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// fsClaudeConfig represents the structure of ~/.claude.json.
type fsClaudeConfig struct {
	Projects map[string]fsProjectConfig `json:"projects"`
}

type fsProjectConfig struct {
	MCPServers map[string]fsMCPServer `json:"mcpServers"`
}

type fsMCPServer struct {
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// Read implements Reader using filesystem access.
func (r *FilesystemReader) Read() (*State, error) {
	claudeDir := r.ClaudeDir
	if claudeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		claudeDir = filepath.Join(home, ".claude")
	}

	state := &State{
		Sources:    make(map[string]SourceState),
		Plugins:    make(map[string]PluginState),
		MCPServers: make(map[string]MCPServerState),
	}

	// Read sources (marketplaces converted to sources)
	if err := r.readSources(claudeDir, state); err != nil {
		// Non-fatal, continue with empty sources
		fmt.Fprintf(os.Stderr, "Warning: could not read sources: %v\n", err)
	}

	// Read plugin repositories from repos/ directory
	if err := r.readPluginRepos(claudeDir, state); err != nil {
		// Non-fatal, continue without plugin repos
		fmt.Fprintf(os.Stderr, "Warning: could not read plugin repos: %v\n", err)
	}

	// Read plugins
	if err := r.readPlugins(claudeDir, state); err != nil {
		// Non-fatal, continue with empty plugins
		fmt.Fprintf(os.Stderr, "Warning: could not read plugins: %v\n", err)
	}

	// Read enabled state from settings
	if err := r.readSettings(claudeDir, state); err != nil {
		// Non-fatal, continue with default enabled state
		fmt.Fprintf(os.Stderr, "Warning: could not read settings: %v\n", err)
	}

	// Read MCP servers - derive home from claudeDir to respect HOME env var
	// claudeDir is ~/.claude, so home is parent directory
	home := filepath.Dir(claudeDir)
	if err := r.readMCPServers(home, state); err != nil {
		// Non-fatal, continue with empty MCP servers
		fmt.Fprintf(os.Stderr, "Warning: could not read MCP servers: %v\n", err)
	}

	return state, nil
}

func (r *FilesystemReader) readSources(claudeDir string, state *State) error {
	path := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No sources file is okay
		}
		return err
	}

	// Read marketplace format and convert to sources
	var marketplaces map[string]fsMarketplaceEntry
	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return fmt.Errorf("failed to parse known_marketplaces.json: %w", err)
	}

	// Convert marketplaces to sources with kind="marketplace"
	for name, m := range marketplaces {
		source := SourceState{
			Name:            name,
			Kind:            types.SourceKindMarketplace.String(), // All items in known_marketplaces.json are marketplace kind
			Type:            m.Source.Source,                      // github or local
			InstallLocation: m.InstallLocation,
			LastUpdated:     m.LastUpdated,
		}

		// Set URL or Path based on type
		sourceType := types.SourceType(m.Source.Source)
		switch {
		case sourceType.IsGitHub():
			source.URL = m.Source.Repo
		case sourceType.IsLocal():
			source.Path = m.Source.Path
		}

		state.Sources[name] = source
	}

	return nil
}

func (r *FilesystemReader) readPluginRepos(claudeDir string, state *State) error {
	reposDir := filepath.Join(claudeDir, "plugins", "repos")

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No repos directory is okay
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		repoPath := filepath.Join(reposDir, name)

		// All repos in ~/.claude/plugins/repos/ are local plugin-kind sources
		source := SourceState{
			Name:            name,
			Kind:            types.SourceKindPlugin.String(),
			Type:            types.SourceTypeLocal.String(),
			Path:            repoPath,
			InstallLocation: repoPath,
		}

		state.Sources[name] = source
	}

	return nil
}

func (r *FilesystemReader) readPlugins(claudeDir string, state *State) error {
	path := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No plugins file is okay
		}
		return err
	}

	var plugins fsInstalledPlugins
	if err := json.Unmarshal(data, &plugins); err != nil {
		return fmt.Errorf("failed to parse installed_plugins.json: %w", err)
	}

	reposDir := filepath.Join(claudeDir, "plugins", "repos")

	for fullName, installs := range plugins.Plugins {
		// fullName is like "plugin@marketplace" or just "plugin" for local plugins
		parts := strings.SplitN(fullName, "@", 2)
		pluginName := parts[0]
		marketplace := ""
		if len(parts) > 1 {
			marketplace = parts[1]
		}

		// Use the first (most recent) install for each plugin
		if len(installs) > 0 {
			install := installs[0]

			// Detect if this is a local plugin:
			// 1. If installPath is in the repos/ directory, OR
			// 2. If the plugin name doesn't have @marketplace (no marketplace association)
			//    and has a valid local path
			isLocal := strings.HasPrefix(install.InstallPath, reposDir) ||
				(marketplace == "" && install.InstallPath != "")

			state.Plugins[fullName] = PluginState{
				Name:         pluginName,
				Marketplace:  marketplace,
				Scope:        install.Scope,
				Enabled:      true, // Default to true, will be updated by settings
				Version:      install.Version,
				InstallPath:  install.InstallPath,
				IsLocal:      isLocal,
				GitCommitSha: install.GitCommitSha,
			}
		}
	}

	return nil
}

func (r *FilesystemReader) readSettings(claudeDir string, state *State) error {
	path := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No settings file is okay
		}
		return err
	}

	var settings fsSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse settings.json: %w", err)
	}

	// Update enabled state for plugins
	for name, enabled := range settings.EnabledPlugins {
		if plugin, ok := state.Plugins[name]; ok {
			plugin.Enabled = enabled
			state.Plugins[name] = plugin
		}
	}

	return nil
}

func (r *FilesystemReader) readMCPServers(homeDir string, state *State) error {
	path := filepath.Join(homeDir, ".claude.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No claude.json file is okay
		}
		return err
	}

	var config fsClaudeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse .claude.json: %w", err)
	}

	// Collect MCP servers from all projects
	// For now, use homeDir project as user-scope servers
	homeProject := config.Projects[homeDir]
	for name, server := range homeProject.MCPServers {
		state.MCPServers[name] = MCPServerState{
			Name:      name,
			Transport: server.Transport,
			Command:   server.Command,
			Args:      server.Args,
			URL:       server.URL,
			Scope:     types.ScopeUser.String(),
		}
	}

	return nil
}
