// Package config handles Clewfile parsing and location resolution.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Clewfile represents the parsed configuration file.
type Clewfile struct {
	Version      int                    `yaml:"version" toml:"version" json:"version"`
	Marketplaces map[string]Marketplace `yaml:"marketplaces" toml:"marketplaces" json:"marketplaces"`
	Plugins      []Plugin               `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers   map[string]MCPServer   `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}

// Marketplace represents a plugin marketplace source.
type Marketplace struct {
	Source string `yaml:"source" toml:"source" json:"source"` // github, local
	Repo   string `yaml:"repo,omitempty" toml:"repo,omitempty" json:"repo,omitempty"`
	Path   string `yaml:"path,omitempty" toml:"path,omitempty" json:"path,omitempty"`
}

// Plugin represents a plugin to install.
// Can be specified as a simple string "name@marketplace" or as a struct.
type Plugin struct {
	Name    string `yaml:"name" toml:"name" json:"name"`
	Enabled *bool  `yaml:"enabled,omitempty" toml:"enabled,omitempty" json:"enabled,omitempty"`
	Scope   string `yaml:"scope,omitempty" toml:"scope,omitempty" json:"scope,omitempty"`
}

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Transport string            `yaml:"transport" toml:"transport" json:"transport"` // stdio, http
	Command   string            `yaml:"command,omitempty" toml:"command,omitempty" json:"command,omitempty"`
	Args      []string          `yaml:"args,omitempty" toml:"args,omitempty" json:"args,omitempty"`
	URL       string            `yaml:"url,omitempty" toml:"url,omitempty" json:"url,omitempty"`
	Env       map[string]string `yaml:"env,omitempty" toml:"env,omitempty" json:"env,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty" toml:"headers,omitempty" json:"headers,omitempty"`
	Scope     string            `yaml:"scope,omitempty" toml:"scope,omitempty" json:"scope,omitempty"`
}

// FindClewfile searches for a Clewfile in the standard locations.
// Returns the path to the first Clewfile found, or an error if none exists.
func FindClewfile(explicitPath string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("specified Clewfile not found: %s", explicitPath)
		}
		return explicitPath, nil
	}

	// Check CLEWFILE environment variable
	if envPath := os.Getenv("CLEWFILE"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// Build search paths in order of precedence
	var searchPaths []string

	// XDG_CONFIG_HOME or default
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}
	searchPaths = append(searchPaths, filepath.Join(xdgConfig, "claude"))

	// ~/.claude
	home, _ := os.UserHomeDir()
	searchPaths = append(searchPaths, filepath.Join(home, ".claude"))

	// Home directory root
	searchPaths = append(searchPaths, home)

	// File name variants
	fileNames := []string{
		"Clewfile",
		"Clewfile.yaml",
		"Clewfile.yml",
		"Clewfile.toml",
		"Clewfile.json",
		".Clewfile",
		".Clewfile.yaml",
		".Clewfile.yml",
		".Clewfile.toml",
		".Clewfile.json",
	}

	for _, dir := range searchPaths {
		for _, name := range fileNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no Clewfile found in standard locations")
}

// Load reads and parses a Clewfile from the given path.
func Load(path string) (*Clewfile, error) {
	// TODO: Implement parsing based on file extension
	// For now, return a placeholder
	return nil, fmt.Errorf("Load not yet implemented")
}

// InferScope determines the default scope based on Clewfile location.
func InferScope(clewfilePath string) string {
	home, _ := os.UserHomeDir()
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}

	dir := filepath.Dir(clewfilePath)

	// If in home config directories, default to user scope
	if dir == home || dir == filepath.Join(home, ".claude") ||
	   filepath.HasPrefix(dir, xdgConfig) {
		return "user"
	}

	// Otherwise, assume project scope
	return "project"
}
