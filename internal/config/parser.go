// Package config handles Clewfile parsing and location resolution.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Format represents the file format of a Clewfile.
type Format int

const (
	FormatUnknown Format = iota
	FormatYAML
	FormatTOML
	FormatJSON
)

// detectFormat determines the file format based on extension or content.
func detectFormat(path string, content []byte) Format {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".toml":
		return FormatTOML
	case ".json":
		return FormatJSON
	}

	// Content sniffing for extensionless files
	return sniffFormat(content)
}

// sniffFormat attempts to detect format from content.
func sniffFormat(content []byte) Format {
	trimmed := strings.TrimSpace(string(content))

	// JSON starts with { or [
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return FormatJSON
	}

	// TOML typically has [sections] or key = value with = sign
	// YAML uses key: value with : sign
	// Check for TOML indicators first
	if strings.Contains(trimmed, " = ") || strings.HasPrefix(trimmed, "[") {
		// Verify it's valid TOML by checking for = on non-comment lines
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, " = ") || strings.HasPrefix(line, "[") {
				return FormatTOML
			}
			// If we see : without =, it's likely YAML
			if strings.Contains(line, ":") && !strings.Contains(line, "=") {
				return FormatYAML
			}
		}
	}

	// Default to YAML if we see colons
	if strings.Contains(trimmed, ":") {
		return FormatYAML
	}

	return FormatUnknown
}

// rawClewfile is an intermediate representation for parsing.
// It handles the flexible Plugin format (string or struct).
type rawClewfile struct {
	Version      int                    `yaml:"version" toml:"version" json:"version"`
	Marketplaces map[string]Marketplace `yaml:"marketplaces" toml:"marketplaces" json:"marketplaces"`
	Plugins      []interface{}          `yaml:"plugins" toml:"plugins" json:"plugins"`
	MCPServers   map[string]MCPServer   `yaml:"mcp_servers" toml:"mcp_servers" json:"mcp_servers"`
}

// parsePlugins converts the flexible plugin format to Plugin structs.
// Plugins can be specified as:
//   - Simple string: "name@marketplace" (e.g., "context7@official")
//   - Struct with name, enabled, and scope fields
func parsePlugins(raw []interface{}) ([]Plugin, error) {
	plugins := make([]Plugin, 0, len(raw))

	for i, item := range raw {
		switch v := item.(type) {
		case string:
			// Simple string format: "name@marketplace"
			plugins = append(plugins, Plugin{Name: v})

		case map[string]interface{}:
			// Struct format with name, enabled, scope
			plugin := Plugin{}

			if name, ok := v["name"].(string); ok {
				plugin.Name = name
			} else {
				return nil, fmt.Errorf("plugin[%d]: missing or invalid 'name' field", i)
			}

			if enabled, ok := v["enabled"].(bool); ok {
				plugin.Enabled = &enabled
			}

			if scope, ok := v["scope"].(string); ok {
				plugin.Scope = scope
			}

			plugins = append(plugins, plugin)

		default:
			return nil, fmt.Errorf("plugin[%d]: invalid format (expected string or object)", i)
		}
	}

	return plugins, nil
}

// envVarPattern matches ${VAR} and ${VAR:-default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

// expandEnvVars replaces ${VAR} and ${VAR:-default} patterns in content.
func expandEnvVars(content []byte) []byte {
	result := envVarPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		parts := envVarPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := string(parts[1])
		value := os.Getenv(varName)

		if value == "" && len(parts) >= 3 && len(parts[2]) > 0 {
			// Use default value
			value = string(parts[2])
		}

		return []byte(value)
	})

	return result
}

// parse parses the content according to the specified format.
func parse(content []byte, format Format) (*Clewfile, error) {
	// Expand environment variables first
	content = expandEnvVars(content)

	var raw rawClewfile

	switch format {
	case FormatYAML:
		if err := yaml.Unmarshal(content, &raw); err != nil {
			return nil, fmt.Errorf("YAML parse error: %w", err)
		}
	case FormatTOML:
		if err := toml.Unmarshal(content, &raw); err != nil {
			return nil, fmt.Errorf("TOML parse error: %w", err)
		}
	case FormatJSON:
		if err := json.Unmarshal(content, &raw); err != nil {
			return nil, fmt.Errorf("JSON parse error: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown file format")
	}

	// Convert plugins from flexible format
	plugins, err := parsePlugins(raw.Plugins)
	if err != nil {
		return nil, err
	}

	clewfile := &Clewfile{
		Version:      raw.Version,
		Marketplaces: raw.Marketplaces,
		Plugins:      plugins,
		MCPServers:   raw.MCPServers,
	}

	// Initialize nil maps
	if clewfile.Marketplaces == nil {
		clewfile.Marketplaces = make(map[string]Marketplace)
	}
	if clewfile.MCPServers == nil {
		clewfile.MCPServers = make(map[string]MCPServer)
	}

	return clewfile, nil
}
