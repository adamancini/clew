// Package diff computes differences between desired and current state.
package diff

import (
	"strings"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
)

// Compute calculates the diff between a Clewfile and current state.
func compute(clewfile *config.Clewfile, current *state.State) *Result {
	result := &Result{
		Marketplaces: computeMarketplaceDiffs(clewfile.Marketplaces, current.Marketplaces),
		Plugins:      computePluginDiffs(clewfile.Plugins, current.Plugins),
		MCPServers:   computeMCPServerDiffs(clewfile.MCPServers, current.MCPServers),
	}
	return result
}

func computeMarketplaceDiffs(desired map[string]config.Marketplace, current map[string]state.MarketplaceState) []MarketplaceDiff {
	var diffs []MarketplaceDiff
	seen := make(map[string]bool)

	// Check each desired marketplace
	for name, d := range desired {
		seen[name] = true
		desiredCopy := d

		if c, exists := current[name]; exists {
			currentCopy := c
			// Check if update needed (source changed)
			if marketplaceNeedsUpdate(d, c) {
				diffs = append(diffs, MarketplaceDiff{
					Name:    name,
					Action:  ActionUpdate,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			} else {
				diffs = append(diffs, MarketplaceDiff{
					Name:    name,
					Action:  ActionNone,
					Current: &currentCopy,
					Desired: &desiredCopy,
				})
			}
		} else {
			// Needs to be added
			diffs = append(diffs, MarketplaceDiff{
				Name:    name,
				Action:  ActionAdd,
				Desired: &desiredCopy,
			})
		}
	}

	// Check for extra marketplaces not in Clewfile
	for name, c := range current {
		if !seen[name] {
			currentCopy := c
			diffs = append(diffs, MarketplaceDiff{
				Name:    name,
				Action:  ActionRemove,
				Current: &currentCopy,
			})
		}
	}

	return diffs
}

func marketplaceNeedsUpdate(desired config.Marketplace, current state.MarketplaceState) bool {
	// Check if source type changed
	if desired.Source != current.Source {
		return true
	}
	// Check if repo changed for github source
	if desired.Source == "github" && desired.Repo != current.Repo {
		return true
	}
	// Check if path changed for local source
	if desired.Source == "local" && desired.Path != current.Path {
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
	// Only HTTP/SSE transports can require OAuth
	if server.Transport != "http" && server.Transport != "sse" {
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

	// For stdio, check command and args
	if desired.Transport == "stdio" {
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

	// For HTTP/SSE, check URL
	if (desired.Transport == "http" || desired.Transport == "sse") && desired.URL != current.URL {
		return true
	}

	return false
}
