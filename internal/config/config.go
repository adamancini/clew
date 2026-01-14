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
	// SourceKind represents the type of source.
	SourceKind = types.SourceKind
	// SourceType represents how a source is accessed.
	SourceType = types.SourceType
	// Scope represents the installation scope.
	Scope = types.Scope
	// TransportType represents the MCP server transport protocol.
	TransportType = types.TransportType
)

// Source kind constants - re-exported from types package.
const (
	SourceKindMarketplace = types.SourceKindMarketplace
	SourceKindPlugin      = types.SourceKindPlugin
	SourceKindLocal       = types.SourceKindLocal
)

// Source type constants - re-exported from types package.
const (
	SourceTypeGitHub = types.SourceTypeGitHub
	SourceTypeLocal  = types.SourceTypeLocal
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

// SourceConfig defines where a source comes from.
type SourceConfig struct {
	Type SourceType `yaml:"type" toml:"type" json:"type"`
	URL  string     `yaml:"url,omitempty" toml:"url,omitempty" json:"url,omitempty"`   // For github: org/repo or full URL
	Ref  string     `yaml:"ref,omitempty" toml:"ref,omitempty" json:"ref,omitempty"`   // Optional git ref (branch/tag)
	Path string     `yaml:"path,omitempty" toml:"path,omitempty" json:"path,omitempty"` // For local: filesystem path
}

// Source represents a unified source (marketplace, plugin repo, or local plugin).
type Source struct {
	Name   string       `yaml:"name" toml:"name" json:"name"`
	Alias  string       `yaml:"alias,omitempty" toml:"alias,omitempty" json:"alias,omitempty"`
	Kind   SourceKind   `yaml:"kind" toml:"kind" json:"kind"`
	Source SourceConfig `yaml:"source" toml:"source" json:"source"`
}

// GetAlias returns the alias if set, otherwise returns the name.
func (s *Source) GetAlias() string {
	if s.Alias != "" {
		return s.Alias
	}
	return s.Name
}

// Clewfile represents the parsed configuration file.
type Clewfile struct {
	Version    int                  `yaml:"version" toml:"version" json:"version"`
	Sources    []Source             `yaml:"sources,omitempty" toml:"sources,omitempty" json:"sources,omitempty"`
	Plugins    []Plugin             `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers map[string]MCPServer `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}

// GetSourceByAliasOrName finds a source by its alias or name.
func (c *Clewfile) GetSourceByAliasOrName(ref string) (*Source, error) {
	for i := range c.Sources {
		if c.Sources[i].GetAlias() == ref || c.Sources[i].Name == ref {
			return &c.Sources[i], nil
		}
	}
	return nil, fmt.Errorf("source not found: %s", ref)
}

// Plugin represents a plugin to install.
// Can be specified as:
//   - Simple string: "name@source" or "name" (for plugin-kind sources with matching names)
//   - Struct with inline source for one-off plugins
//   - Struct with reference to named source
type Plugin struct {
	Name    string        `yaml:"name" toml:"name" json:"name"`
	Source  *SourceConfig `yaml:"source,omitempty" toml:"source,omitempty" json:"source,omitempty"` // Inline source for one-off plugins
	Enabled *bool         `yaml:"enabled,omitempty" toml:"enabled,omitempty" json:"enabled,omitempty"`
	Scope   string        `yaml:"scope,omitempty" toml:"scope,omitempty" json:"scope,omitempty"`
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
