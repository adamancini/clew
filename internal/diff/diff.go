// Package diff computes differences between desired and current state.
package diff

import (
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/state"
)

// Action represents what needs to be done for an item.
type Action string

const (
	ActionNone    Action = "none"    // Already in desired state
	ActionAdd     Action = "add"     // Needs to be added
	ActionRemove  Action = "remove"  // Exists but not in Clewfile (info only)
	ActionUpdate  Action = "update"  // Needs configuration update
	ActionEnable  Action = "enable"  // Needs to be enabled
	ActionDisable Action = "disable" // Needs to be disabled
)

// MarketplaceDiff represents the diff for a marketplace.
type MarketplaceDiff struct {
	Name   string
	Action Action
	Current *state.MarketplaceState
	Desired *config.Marketplace
}

// PluginDiff represents the diff for a plugin.
type PluginDiff struct {
	Name    string
	Action  Action
	Current *state.PluginState
	Desired *config.Plugin
}

// MCPServerDiff represents the diff for an MCP server.
type MCPServerDiff struct {
	Name    string
	Action  Action
	Current *state.MCPServerState
	Desired *config.MCPServer
	// RequiresOAuth indicates the server needs manual OAuth setup
	RequiresOAuth bool
}

// Result contains the complete diff between desired and current state.
type Result struct {
	Marketplaces []MarketplaceDiff
	Plugins      []PluginDiff
	MCPServers   []MCPServerDiff
}

// Compute calculates the diff between a Clewfile and current state.
func Compute(clewfile *config.Clewfile, current *state.State) *Result {
	// TODO: Implement diff computation
	return &Result{}
}

// Summary returns counts of actions needed.
func (r *Result) Summary() (add, update, remove, attention int) {
	for _, m := range r.Marketplaces {
		switch m.Action {
		case ActionAdd:
			add++
		case ActionUpdate:
			update++
		case ActionRemove:
			attention++
		}
	}
	for _, p := range r.Plugins {
		switch p.Action {
		case ActionAdd:
			add++
		case ActionUpdate, ActionEnable, ActionDisable:
			update++
		case ActionRemove:
			attention++
		}
	}
	for _, m := range r.MCPServers {
		switch m.Action {
		case ActionAdd:
			if m.RequiresOAuth {
				attention++
			} else {
				add++
			}
		case ActionUpdate:
			update++
		case ActionRemove:
			attention++
		}
	}
	return
}
