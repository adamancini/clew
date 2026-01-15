// Package config handles Clewfile parsing and location resolution.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamancini/clew/internal/types"
)

// Type aliases for backward compatibility.
// These types are now defined in internal/types and re-exported here.
type (
	// Scope represents the installation scope.
	Scope = types.Scope
	// TransportType represents the MCP server transport protocol.
	TransportType = types.TransportType
)

// Scope constants - re-exported from types package.
const (
	ScopeUser    = types.ScopeUser
	ScopeProject = types.ScopeProject
)

// Transport type constants - re-exported from types package.
const (
	TransportStdio = types.TransportStdio
	TransportHTTP  = types.TransportHTTP
	TransportSSE   = types.TransportSSE
)

// Marketplace represents a plugin marketplace source.
// Marketplaces are repositories containing multiple plugins that can be installed.
type Marketplace struct {
	Repo string `yaml:"repo" toml:"repo" json:"repo"` // Repository URL (e.g., "owner/repo", "https://gitlab.com/company/plugins.git")
	Ref  string `yaml:"ref,omitempty" toml:"ref,omitempty" json:"ref,omitempty"` // Optional git ref (branch/tag/SHA)
}

// Clewfile represents the parsed configuration file.
type Clewfile struct {
	Version      int                    `yaml:"version" toml:"version" json:"version"`
	Marketplaces map[string]Marketplace `yaml:"marketplaces,omitempty" toml:"marketplaces,omitempty" json:"marketplaces,omitempty"`
	Plugins      []Plugin               `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers   map[string]MCPServer   `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}

// GetMarketplace finds a marketplace by its alias (map key).
func (c *Clewfile) GetMarketplace(alias string) (*Marketplace, error) {
	if m, ok := c.Marketplaces[alias]; ok {
		return &m, nil
	}
	return nil, fmt.Errorf("marketplace not found: %s", alias)
}

// Plugin represents a plugin to install.
// Can be specified as:
//   - Simple string: "name@marketplace" (e.g., "context7@official")
//   - Struct with name, enabled, and scope
//
// The name must be in "plugin@marketplace" format where marketplace
// refers to a key in the marketplaces map.
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

	// Get home directory (required for standard locations)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	// Build search paths in order of precedence
	var searchPaths []string

	// XDG_CONFIG_HOME or default
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	searchPaths = append(searchPaths, filepath.Join(xdgConfig, "claude"))

	// ~/.claude
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
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Clewfile: %w", err)
	}

	format := detectFormat(path, content)
	if format == FormatUnknown {
		return nil, fmt.Errorf("unable to detect file format for %s", path)
	}

	clewfile, err := parse(content, format)
	if err != nil {
		return nil, err
	}

	if err := Validate(clewfile); err != nil {
		return nil, err
	}

	return clewfile, nil
}

// InferScope determines the default scope based on Clewfile location.
// Returns "user" for home directory locations, "project" otherwise.
func InferScope(clewfilePath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't determine home directory, default to project scope (safer)
		return "project"
	}

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}

	dir := filepath.Dir(clewfilePath)

	// If in home config directories, default to user scope
	if dir == home || dir == filepath.Join(home, ".claude") ||
		strings.HasPrefix(dir, xdgConfig+string(os.PathSeparator)) || dir == xdgConfig {
		return "user"
	}

	// Otherwise, assume project scope
	return "project"
}
