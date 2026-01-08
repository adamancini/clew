// Package state handles detection of current Claude Code configuration state.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fsMarketplaces represents the structure of known_marketplaces.json.
type fsMarketplaces map[string]fsMarketplace

type fsMarketplace struct {
	Source          fsMarketplaceSource `json:"source"`
	InstallLocation string              `json:"installLocation"`
	LastUpdated     string              `json:"lastUpdated"`
}

type fsMarketplaceSource struct {
	Source string `json:"source"`
	Repo   string `json:"repo,omitempty"`
	Path   string `json:"path,omitempty"`
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
		Marketplaces: make(map[string]MarketplaceState),
		Plugins:      make(map[string]PluginState),
		MCPServers:   make(map[string]MCPServerState),
	}

	// Read marketplaces
	if err := r.readMarketplaces(claudeDir, state); err != nil {
		// Non-fatal, continue with empty marketplaces
		fmt.Fprintf(os.Stderr, "Warning: could not read marketplaces: %v\n", err)
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

	// Read MCP servers
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine home directory: %v\n", err)
	} else if err := r.readMCPServers(home, state); err != nil {
		// Non-fatal, continue with empty MCP servers
		fmt.Fprintf(os.Stderr, "Warning: could not read MCP servers: %v\n", err)
	}

	return state, nil
}

func (r *FilesystemReader) readMarketplaces(claudeDir string, state *State) error {
	path := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No marketplaces file is okay
		}
		return err
	}

	var marketplaces fsMarketplaces
	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return fmt.Errorf("failed to parse known_marketplaces.json: %w", err)
	}

	for name, m := range marketplaces {
		state.Marketplaces[name] = MarketplaceState{
			Name:            name,
			Source:          m.Source.Source,
			Repo:            m.Source.Repo,
			Path:            m.Source.Path,
			InstallLocation: m.InstallLocation,
			LastUpdated:     m.LastUpdated,
		}
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

	for fullName, installs := range plugins.Plugins {
		// fullName is like "plugin@marketplace"
		parts := strings.SplitN(fullName, "@", 2)
		pluginName := parts[0]
		marketplace := ""
		if len(parts) > 1 {
			marketplace = parts[1]
		}

		// Use the first (most recent) install for each plugin
		if len(installs) > 0 {
			install := installs[0]
			state.Plugins[fullName] = PluginState{
				Name:        pluginName,
				Marketplace: marketplace,
				Scope:       install.Scope,
				Enabled:     true, // Default to true, will be updated by settings
				Version:     install.Version,
				InstallPath: install.InstallPath,
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
			Scope:     "user",
		}
	}

	return nil
}
