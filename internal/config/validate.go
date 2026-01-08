// Package config handles Clewfile parsing and location resolution.
package config

import (
	"fmt"
	"strings"
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

	// Validate marketplaces
	for name, m := range c.Marketplaces {
		if err := validateMarketplace(name, m); err != nil {
			errors = append(errors, err.Error())
		}
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

func validateMarketplace(name string, m Marketplace) error {
	if m.Source == "" {
		return ValidationError{
			Field:   fmt.Sprintf("marketplaces.%s.source", name),
			Message: "source is required (github or local)",
		}
	}

	switch m.Source {
	case "github":
		if m.Repo == "" {
			return ValidationError{
				Field:   fmt.Sprintf("marketplaces.%s.repo", name),
				Message: "repo is required for github source",
			}
		}
	case "local":
		if m.Path == "" {
			return ValidationError{
				Field:   fmt.Sprintf("marketplaces.%s.path", name),
				Message: "path is required for local source",
			}
		}
	default:
		return ValidationError{
			Field:   fmt.Sprintf("marketplaces.%s.source", name),
			Message: fmt.Sprintf("invalid source '%s' (must be github or local)", m.Source),
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

	if p.Scope != "" && p.Scope != "user" && p.Scope != "project" {
		return ValidationError{
			Field:   fmt.Sprintf("plugins[%d].scope", index),
			Message: fmt.Sprintf("invalid scope '%s' (must be user or project)", p.Scope),
		}
	}

	return nil
}

func validateMCPServer(name string, s MCPServer) error {
	if s.Transport == "" {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.transport", name),
			Message: "transport is required (stdio or http)",
		}
	}

	switch s.Transport {
	case "stdio":
		if s.Command == "" {
			return ValidationError{
				Field:   fmt.Sprintf("mcp_servers.%s.command", name),
				Message: "command is required for stdio transport",
			}
		}
	case "http", "sse":
		if s.URL == "" {
			return ValidationError{
				Field:   fmt.Sprintf("mcp_servers.%s.url", name),
				Message: "url is required for http/sse transport",
			}
		}
	default:
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.transport", name),
			Message: fmt.Sprintf("invalid transport '%s' (must be stdio, http, or sse)", s.Transport),
		}
	}

	if s.Scope != "" && s.Scope != "user" && s.Scope != "project" {
		return ValidationError{
			Field:   fmt.Sprintf("mcp_servers.%s.scope", name),
			Message: fmt.Sprintf("invalid scope '%s' (must be user or project)", s.Scope),
		}
	}

	return nil
}
