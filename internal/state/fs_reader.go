// Package state handles detection of current Claude Code configuration state.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	}

	// Read marketplaces from known_marketplaces.json
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

	var marketplaces map[string]fsMarketplaceEntry
	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return fmt.Errorf("failed to parse known_marketplaces.json: %w", err)
	}

	for alias, m := range marketplaces {
		state.Marketplaces[alias] = MarketplaceState{
			Alias:           alias,
			Repo:            m.Source.Repo,
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
