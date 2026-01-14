// Package diff computes differences between desired and current state.
package diff

import (
	"strings"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
	"github.com/adamancini/clew/internal/types"
)

// Compute calculates the diff between a Clewfile and current state.
func compute(clewfile *config.Clewfile, current *state.State) *Result {
	result := &Result{
		Sources:    computeSourceDiffs(clewfile.Sources, current.Sources),
		Plugins:    computePluginDiffs(clewfile.Plugins, current.Plugins),
		MCPServers: computeMCPServerDiffs(clewfile.MCPServers, current.MCPServers),
	}
	return result
}

func computeSourceDiffs(desired []config.Source, current map[string]state.SourceState) []SourceDiff {
	var diffs []SourceDiff
	seen := make(map[string]bool)

	// Check each desired source
	for _, d := range desired {
		seen[d.Name] = true
		desiredCopy := d

		if c, exists := current[d.Name]; exists {
			currentCopy := c
			// Check if update needed (source changed)
			if sourceNeedsUpdate(d, c) {
				diffs = append(diffs, SourceDiff{
					Name:    d.Name,
					Action:  ActionUpdate,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			} else {
				diffs = append(diffs, SourceDiff{
					Name:    d.Name,
					Action:  ActionNone,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			}
		} else {
			// Needs to be added
			diffs = append(diffs, SourceDiff{
				Name:    d.Name,
				Action:  ActionAdd,
				Desired: &desiredCopy,
			})
		}
	}

	// Check for extra sources not in Clewfile
	for name, c := range current {
		if !seen[name] {
			currentCopy := c
			diffs = append(diffs, SourceDiff{
				Name:    name,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}

func sourceNeedsUpdate(desired config.Source, current state.SourceState) bool {
	// Check if kind changed
	if string(desired.Kind) != current.Kind {
		return true
	}
	// Check if source type changed
	if string(desired.Source.Type) != current.Type {
		return true
	}
	// Check if URL changed (only github sources are supported)
	if desired.Source.URL != current.URL {
		return true
	}
	// Check if ref changed
	if desired.Source.Ref != current.Ref {
		return true
	}
	return false
}

func computePluginDiffs(desired []config.Plugin, current map[string]state.PluginState) []PluginDiff {
	var diffs []PluginDiff
	seen := make(map[string]bool)

	// Check each desired plugin
	for _, d := range desired {
		desiredCopy := d
		fullName := d.Name // Already includes @marketplace if specified

		seen[fullName] = true

		if c, exists := current[fullName]; exists {
			currentCopy := c
			action := ActionNone

			// Check enabled state
			desiredEnabled := d.Enabled == nil || *d.Enabled
			if desiredEnabled && !c.Enabled {
				action = ActionEnable
			} else if !desiredEnabled && c.Enabled {
				action = ActionDisable
			}

			// Check scope mismatch (would need reinstall)
			if d.Scope != "" && d.Scope != c.Scope {
				action = ActionUpdate
			}

			diffs = append(diffs, PluginDiff{
				Name:    fullName,
				Action:  action,
				Current: &currentCopy,
				Desired: &desiredCopy,
			})
		} else {
			// Needs to be installed
			diffs = append(diffs, PluginDiff{
				Name:    fullName,
				Action:  ActionAdd,
				Desired: &desiredCopy,
			})
		}
	}

	// Check for extra plugins not in Clewfile
	for name, c := range current {
		if !seen[name] {
			currentCopy := c
			diffs = append(diffs, PluginDiff{
				Name:    name,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}

func computeMCPServerDiffs(desired map[string]config.MCPServer, current map[string]state.MCPServerState) []MCPServerDiff {
	var diffs []MCPServerDiff
	seen := make(map[string]bool)

	// Check each desired MCP server
	for name, d := range desired {
		seen[name] = true
		desiredCopy := d

		requiresOAuth := serverRequiresOAuth(d)

		if c, exists := current[name]; exists {
			currentCopy := c

			// Check if update needed
			if mcpServerNeedsUpdate(d, c) {
				diffs = append(diffs, MCPServerDiff{
					Name:          name,
					Action:        ActionUpdate,
					Current:       &currentCopy,
					Desired:       &desiredCopy,
					RequiresOAuth: requiresOAuth,
				})
			} else {
				diffs = append(diffs, MCPServerDiff{
					Name:          name,
					Action:        ActionNone,
					Current:       &currentCopy,
					Desired:       &desiredCopy,
					RequiresOAuth: requiresOAuth,
				})
			}
		} else {
			// Needs to be added
			diffs = append(diffs, MCPServerDiff{
				Name:          name,
				Action:        ActionAdd,
				Desired:       &desiredCopy,
				RequiresOAuth: requiresOAuth,
			})
		}
	}

	// Check for extra MCP servers not in Clewfile
	for name, c := range current {
		if !seen[name] {
			currentCopy := c
			diffs = append(diffs, MCPServerDiff{
				Name:    name,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}

// serverRequiresOAuth detects if an HTTP MCP server likely requires OAuth.
// This is a heuristic based on common patterns.
func serverRequiresOAuth(server config.MCPServer) bool {
	// Parse transport type for helper method access
	transport := types.TransportType(server.Transport)

	// Only HTTP-based transports can require OAuth
	if !transport.IsHTTPBased() {
		return false
	}

	// If there are env vars for auth, assume it's handled
	for key := range server.Env {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "auth") ||
			strings.Contains(lowerKey, "secret") {
			return false
		}
	}

	// If there are headers for auth, assume it's handled
	for key := range server.Headers {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "auth") ||
			strings.Contains(lowerKey, "token") {
			return false
		}
	}

	// HTTP/SSE without apparent auth config likely needs OAuth
	return true
}

func mcpServerNeedsUpdate(desired config.MCPServer, current state.MCPServerState) bool {
	// Check transport changed
	if desired.Transport != current.Transport {
		return true
	}

	// Parse transport type for helper method access
	transport := types.TransportType(desired.Transport)

	// For stdio, check command and args
	if transport.IsStdio() {
		if desired.Command != current.Command {
			return true
		}
		// Simple args comparison
		if len(desired.Args) != len(current.Args) {
			return true
		}
		for i, arg := range desired.Args {
			if arg != current.Args[i] {
				return true
			}
		}
	}

	// For HTTP-based transports, check URL
	if transport.IsHTTPBased() && desired.URL != current.URL {
		return true
	}

	return false
}
