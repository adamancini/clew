// Package config handles Clewfile parsing and location resolution.
//
// SYNC REQUIREMENT: Validation rules in this file must stay in sync with
// the JSON Schema at schema/clewfile.schema.json.
//
// When updating validation rules:
//  1. Update this file (validate.go) with the new validation logic
//  2. Update schema/clewfile.schema.json with matching constraints
//  3. Update schema/examples/advanced.yaml if adding new features
//  4. See CLAUDE.md "Schema Maintenance" section for full checklist
//
// Synced validation rules:
//   - Source kinds: marketplace, plugin, local (validateSources)
//   - Source types: github, local (validateSources)
//   - Plugin scopes: user, project (validatePlugin)
//   - MCP transports: stdio, http, sse (validateMCPServer)
//   - MCP scopes: user, project (validateMCPServer)
package config

import (
	"fmt"
	"strings"

	"github.com/adamancini/clew/internal/types"
)

// ValidationError represents a Clewfile validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks the Clewfile for required fields and valid values.
func Validate(c *Clewfile) error {
	var errors []string

	// Validate sources
	if err := validateSources(c.Sources); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate plugin references
	if err := validatePluginReferences(c); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate plugins
	for i, p := range c.Plugins {
		if err := validatePlugin(i, p); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate MCP servers
	for name, s := range c.MCPServers {
		if err := validateMCPServer(name, s); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

func validateSources(sources []Source) error {
	aliases := make(map[string]bool)

	for i, s := range sources {
		// Validate required fields
		if s.Name == "" {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].name", i),
				Message: "name is required",
			}
		}

		// Validate kind using the type's Validate method
		if err := s.Kind.Validate(); err != nil {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].kind", i),
				Message: err.Error(),
			}
		}

		// Check alias uniqueness
		alias := s.GetAlias()
		if aliases[alias] {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].alias", i),
				Message: fmt.Sprintf("duplicate alias '%s'", alias),
			}
		}
		aliases[alias] = true

		// Validate source type using the type's Validate method
		if err := s.Source.Type.Validate(); err != nil {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].source.type", i),
				Message: err.Error(),
			}
		}

		// All source kinds (marketplace, plugin) require github source type
		if !s.Source.Type.IsGitHub() {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].source.type", i),
				Message: fmt.Sprintf("source type must be 'github' (got '%s')", s.Source.Type),
			}
		}

		if s.Source.URL == "" {
			return ValidationError{
				Field:   fmt.Sprintf("sources[%d].source.url", i),
				Message: "github source requires url",
			}
		}
	}

	return nil
}

func validatePluginReferences(c *Clewfile) error {
	for i, p := range c.Plugins {
		// Skip plugins with inline sources
		if p.Source != nil {
			continue
		}

		// Check if plugin name contains @source reference
		if strings.Contains(p.Name, "@") {
			parts := strings.SplitN(p.Name, "@", 2)
			sourceRef := parts[1]

			if _, err := c.GetSourceByAliasOrName(sourceRef); err != nil {
				return ValidationError{
					Field:   fmt.Sprintf("plugins[%d].name", i),
					Message: fmt.Sprintf("references unknown source '%s'", sourceRef),
				}
			}
		}
	}

	return nil
}

func validatePlugin(index int, p Plugin) error {
	if p.Name == "" {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].name", index),
			Message: "name is required",
		}
	}

	// Validate inline source if present
	if p.Source != nil {
		// Validate source type using the type's Validate method
		if err := p.Source.Type.Validate(); err != nil {
			return ValidationError{
				Field:   fmt.Sprintf("plugins[%d].source", index),
				Message: err.Error(),
			}
		}

		// Only github sources are supported
		if p.Source.URL == "" {
			return ValidationError{
				Field:   fmt.Sprintf("plugins[%d].source.url", index),
				Message: "github source requires url",
			}
		}
	}

	// Validate scope using the type's Validate method
	if err := types.Scope(p.Scope).Validate(); err != nil {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].scope", index),
			Message: err.Error(),
		}
	}

	return nil
}

func validateMCPServer(name string, s MCPServer) error {
	// Validate transport using the type's Validate method
	transport := types.TransportType(s.Transport)
	if err := transport.Validate(); err != nil {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.transport", name),
			Message: err.Error(),
		}
	}

	// Validate transport-specific requirements using helper methods
	if transport.RequiresCommand() && s.Command == "" {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.command", name),
			Message: "command is required for stdio transport",
		}
	}

	if transport.RequiresURL() && s.URL == "" {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.url", name),
			Message: "url is required for http/sse transport",
		}
	}

	// Validate scope using the type's Validate method
	if err := types.Scope(s.Scope).Validate(); err != nil {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.scope", name),
			Message: err.Error(),
		}
	}

	return nil
}
