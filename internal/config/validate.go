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
//   - Marketplace repo: non-empty string (validateMarketplaces)
//   - Plugin scopes: user, project (validatePlugin)
//   - Plugin name format: plugin@marketplace (validatePluginReferences)
//   - MCP transports: stdio, http, sse (validateMCPServer)
//   - MCP scopes: user, project (validateMCPServer)
package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/adamancini/clew/internal/types"
)

// pluginNamePattern validates plugin names in the format "plugin@marketplace"
var pluginNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+$`)

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

	// Validate marketplaces
	if err := validateMarketplaces(c.Marketplaces); err != nil {
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

func validateMarketplaces(marketplaces map[string]Marketplace) error {
	for alias, m := range marketplaces {
		// Validate alias (map key) is not empty
		if alias == "" {
			return ValidationError{
				Field:   "marketplaces",
				Message: "marketplace alias cannot be empty",
			}
		}

		// Validate repo is required and non-empty
		if m.Repo == "" {
			return ValidationError{
				Field:   fmt.Sprintf("marketplaces.%s.repo", alias),
				Message: "repo is required",
			}
		}

		// Note: ref is optional, git will validate if it exists
	}

	return nil
}

func validatePluginReferences(c *Clewfile) error {
	for i, p := range c.Plugins {
		// Check if plugin name contains @marketplace reference
		if strings.Contains(p.Name, "@") {
			parts := strings.SplitN(p.Name, "@", 2)
			marketplaceRef := parts[1]

			if _, err := c.GetMarketplace(marketplaceRef); err != nil {
				return ValidationError{
					Field:   fmt.Sprintf("plugins[%d].name", i),
					Message: fmt.Sprintf("references unknown marketplace '%s'", marketplaceRef),
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

	// Validate plugin name format (plugin@marketplace)
	if !pluginNamePattern.MatchString(p.Name) {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].name", index),
			Message: fmt.Sprintf("invalid plugin name '%s' (must be plugin@marketplace format)", p.Name),
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
