// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adamancini/clew/internal/diff"
)

// localPluginJSON represents the structure of a plugin.json file.
type localPluginJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// installedPluginsFile represents the structure of installed_plugins.json.
type installedPluginsFile struct {
	Version int                            `json:"version"`
	Plugins map[string][]pluginInstallInfo `json:"plugins"`
}

// pluginInstallInfo represents a single plugin installation entry.
type pluginInstallInfo struct {
	Scope        string `json:"scope"`
	ProjectPath  string `json:"projectPath,omitempty"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha,omitempty"`
}

// CommandRunner is an interface for running external commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultCommandRunner uses os/exec to run commands.
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// addSource executes `claude plugin marketplace add <source>` for marketplace-kind sources.
func (s *Syncer) addSource(src diff.SourceDiff) (Operation, error) {
	op := Operation{
		Type:   "source",
		Name:   src.Name,
		Action: "add",
	}

	if src.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for source %s", src.Name)
		return op, fmt.Errorf("no desired state for source %s", src.Name)
	}

	// Only marketplace-kind sources can be added via CLI
	if src.Desired.Kind != "marketplace" {
		op.Success = true
		op.Skipped = true
		op.Description = fmt.Sprintf("Skip non-marketplace source (kind=%s): %s", src.Desired.Kind, src.Name)
		return op, nil
	}

	var source string
	switch src.Desired.Source.Type {
	case "github":
		source = src.Desired.Source.URL
		op.Description = fmt.Sprintf("Add GitHub source: %s", source)
	case "local":
		source = src.Desired.Source.Path
		op.Description = fmt.Sprintf("Add local source: %s", source)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unknown source type: %s", src.Desired.Source.Type)
		return op, fmt.Errorf("unknown source type: %s", src.Desired.Source.Type)
	}

	// Build command string before executing
	op.Command = fmt.Sprintf("claude plugin marketplace add %s", source)

	output, err := s.runner.Run("claude", "plugin", "marketplace", "add", source)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add source %s: %v\nOutput: %s", src.Name, err, string(output))
		return op, fmt.Errorf("failed to add source %s: %w\nOutput: %s", src.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// installPlugin executes `claude plugin install <plugin>`.
func (s *Syncer) installPlugin(p diff.PluginDiff) (Operation, error) {
	op := Operation{
		Type:        "plugin",
		Name:        p.Name,
		Action:      "add",
		Description: fmt.Sprintf("Install plugin: %s", p.Name),
	}

	if p.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for plugin %s", p.Name)
		return op, fmt.Errorf("no desired state for plugin %s", p.Name)
	}

	args := []string{"plugin", "install", p.Desired.Name}

	// Add scope if specified
	if p.Desired.Scope != "" {
		args = append(args, "--scope", p.Desired.Scope)
	}

	// Build command string before executing
	op.Command = "claude " + strings.Join(args, " ")

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to install plugin %s: %v\nOutput: %s", p.Name, err, string(output))
		return op, fmt.Errorf("failed to install plugin %s: %w\nOutput: %s", p.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// updatePluginState executes `claude plugin enable/disable <plugin>`.
func (s *Syncer) updatePluginState(p diff.PluginDiff) (Operation, error) {
	op := Operation{
		Type: "plugin",
		Name: p.Name,
	}

	var action string
	switch p.Action {
	case diff.ActionEnable:
		action = "enable"
		op.Action = "enable"
		op.Description = fmt.Sprintf("Enable plugin: %s", p.Name)
	case diff.ActionDisable:
		action = "disable"
		op.Action = "disable"
		op.Description = fmt.Sprintf("Disable plugin: %s", p.Name)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unexpected action for plugin state update: %s", p.Action)
		return op, fmt.Errorf("unexpected action for plugin state update: %s", p.Action)
	}

	// Build command string before executing
	op.Command = fmt.Sprintf("claude plugin %s %s", action, p.Name)

	output, err := s.runner.Run("claude", "plugin", action, p.Name)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to %s plugin %s: %v\nOutput: %s", action, p.Name, err, string(output))
		return op, fmt.Errorf("failed to %s plugin %s: %w\nOutput: %s", action, p.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// addMCPServer executes `claude mcp add <name> <command|url> [args...]`.
func (s *Syncer) addMCPServer(m diff.MCPServerDiff) (Operation, error) {
	op := Operation{
		Type:        "mcp",
		Name:        m.Name,
		Action:      "add",
		Description: fmt.Sprintf("Add MCP server: %s", m.Name),
	}

	if m.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for MCP server %s", m.Name)
		return op, fmt.Errorf("no desired state for MCP server %s", m.Name)
	}

	args := []string{"mcp", "add", "--transport", m.Desired.Transport}

	// Add scope if specified
	if m.Desired.Scope != "" {
		args = append(args, "--scope", m.Desired.Scope)
	}

	// Add environment variables
	for key, value := range m.Desired.Env {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
	}

	// Add headers (for HTTP/SSE)
	for key, value := range m.Desired.Headers {
		args = append(args, "--header", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the server name
	args = append(args, m.Name)

	// Add command/URL and args based on transport
	switch m.Desired.Transport {
	case "stdio":
		if m.Desired.Command == "" {
			op.Success = false
			op.Error = fmt.Sprintf("stdio MCP server %s requires a command", m.Name)
			return op, fmt.Errorf("stdio MCP server %s requires a command", m.Name)
		}
		// Add -- separator before command to prevent flag parsing issues
		args = append(args, "--")
		args = append(args, m.Desired.Command)
		args = append(args, m.Desired.Args...)
		op.Description = fmt.Sprintf("Add stdio MCP server: %s (command: %s)", m.Name, m.Desired.Command)
	case "http", "sse":
		if m.Desired.URL == "" {
			op.Success = false
			op.Error = fmt.Sprintf("http/sse MCP server %s requires a URL", m.Name)
			return op, fmt.Errorf("http/sse MCP server %s requires a URL", m.Name)
		}
		args = append(args, m.Desired.URL)
		op.Description = fmt.Sprintf("Add %s MCP server: %s (url: %s)", m.Desired.Transport, m.Name, m.Desired.URL)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unknown MCP transport: %s", m.Desired.Transport)
		return op, fmt.Errorf("unknown MCP transport: %s", m.Desired.Transport)
	}

	// Build command string before executing
	op.Command = "claude " + strings.Join(args, " ")

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add MCP server %s: %v\nOutput: %s", m.Name, err, string(output))
		return op, fmt.Errorf("failed to add MCP server %s: %w\nOutput: %s", m.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// installLocalPlugin installs a local repository plugin by directly editing installed_plugins.json.
// This is used for plugins that are cloned into ~/.claude/plugins/repos/ and don't use the marketplace.
func (s *Syncer) installLocalPlugin(p diff.PluginDiff) (Operation, error) {
	op := Operation{
		Type:   "plugin",
		Name:   p.Name,
		Action: "add",
	}

	if p.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for local plugin %s", p.Name)
		return op, fmt.Errorf("no desired state for local plugin %s", p.Name)
	}

	if p.Desired.Source == nil {
		op.Success = false
		op.Error = fmt.Sprintf("local plugin %s requires source configuration", p.Name)
		return op, fmt.Errorf("local plugin %s requires source configuration", p.Name)
	}

	// Expand the path (handle ~)
	pluginPath := expandPath(p.Desired.Source.Path)
	op.Description = fmt.Sprintf("Install local plugin: %s from %s", p.Name, pluginPath)

	// Read plugin.json to get version
	version, err := s.readPluginVersion(pluginPath)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to read plugin.json for %s: %v", p.Name, err)
		return op, fmt.Errorf("failed to read plugin.json for %s: %w", p.Name, err)
	}

	// Get git commit SHA
	gitSha := s.getGitCommitSha(pluginPath)

	// Determine scope (default to "user" if not specified)
	scope := p.Desired.Scope
	if scope == "" {
		scope = "user"
	}

	// Update installed_plugins.json
	if err := s.updateInstalledPlugins(p.Name, pluginPath, version, gitSha, scope); err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to update installed_plugins.json for %s: %v", p.Name, err)
		return op, fmt.Errorf("failed to update installed_plugins.json for %s: %w", p.Name, err)
	}

	op.Command = fmt.Sprintf("Edit installed_plugins.json: add %s (version: %s, sha: %s)", p.Name, version, gitSha)
	op.Success = true
	return op, nil
}

// readPluginVersion reads the version from plugin.json in the plugin directory.
func (s *Syncer) readPluginVersion(pluginPath string) (string, error) {
	// Try plugin.json at root first
	jsonPath := filepath.Join(pluginPath, "plugin.json")
	data, err := s.editor.ReadFile(jsonPath)
	if err != nil {
		// Try .claude-plugin/plugin.json as fallback
		jsonPath = filepath.Join(pluginPath, ".claude-plugin", "plugin.json")
		data, err = s.editor.ReadFile(jsonPath)
		if err != nil {
			return "", fmt.Errorf("plugin.json not found: %w", err)
		}
	}

	var plugin localPluginJSON
	if err := json.Unmarshal(data, &plugin); err != nil {
		return "", fmt.Errorf("failed to parse plugin.json: %w", err)
	}

	if plugin.Version == "" {
		return "0.0.0", nil // Default version if not specified
	}

	return plugin.Version, nil
}

// getGitCommitSha returns the current git commit SHA for the plugin repository.
func (s *Syncer) getGitCommitSha(pluginPath string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%H")
	cmd.Dir = pluginPath
	output, err := cmd.Output()
	if err != nil {
		return "" // Return empty string if not a git repo or git fails
	}
	return strings.TrimSpace(string(output))
}

// updateInstalledPlugins adds or updates a plugin entry in installed_plugins.json.
func (s *Syncer) updateInstalledPlugins(name, installPath, version, gitSha, scope string) error {
	installedPath := filepath.Join(s.claudeDir, "plugins", "installed_plugins.json")

	// Read existing file or create new structure
	var installed installedPluginsFile
	data, err := s.editor.ReadFile(installedPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read installed_plugins.json: %w", err)
		}
		// File doesn't exist, create new structure
		installed = installedPluginsFile{
			Version: 2,
			Plugins: make(map[string][]pluginInstallInfo),
		}
	} else {
		if err := json.Unmarshal(data, &installed); err != nil {
			return fmt.Errorf("failed to parse installed_plugins.json: %w", err)
		}
	}

	// Ensure plugins map exists
	if installed.Plugins == nil {
		installed.Plugins = make(map[string][]pluginInstallInfo)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Create new install entry
	newEntry := pluginInstallInfo{
		Scope:        scope,
		InstallPath:  installPath,
		Version:      version,
		InstalledAt:  now,
		LastUpdated:  now,
		GitCommitSha: gitSha,
	}

	// For user scope, project path is empty
	// For project scope, we'd need the current project path

	// Check if plugin already exists
	if existing, ok := installed.Plugins[name]; ok && len(existing) > 0 {
		// Update existing entry - find matching scope or update first entry
		found := false
		for i, entry := range existing {
			if entry.Scope == scope {
				newEntry.InstalledAt = entry.InstalledAt // Preserve original install time
				existing[i] = newEntry
				found = true
				break
			}
		}
		if !found {
			// Add new scope entry
			installed.Plugins[name] = append([]pluginInstallInfo{newEntry}, existing...)
		}
	} else {
		// Add new plugin
		installed.Plugins[name] = []pluginInstallInfo{newEntry}
	}

	// Write back to file
	output, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal installed_plugins.json: %w", err)
	}

	if err := s.editor.WriteFile(installedPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write installed_plugins.json: %w", err)
	}

	return nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

