// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"github.com/adamancini/clew/internal/diff"
)

// Result represents the outcome of a sync operation.
type Result struct {
	Installed  int
	Updated    int
	Skipped    int
	Failed     int
	Attention  []string // Items needing manual attention
	Errors     []error
}

// Options configures sync behavior.
type Options struct {
	Strict  bool // Exit non-zero on any failure
	Verbose bool
	Quiet   bool
}

// Execute applies the diff to bring current state in line with Clewfile.
func Execute(d *diff.Result, opts Options) (*Result, error) {
	result := &Result{}

	// Process marketplaces first (plugins depend on them)
	for _, m := range d.Marketplaces {
		switch m.Action {
		case diff.ActionAdd:
			if err := addMarketplace(m); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "marketplace: "+m.Name)
		}
	}

	// Process plugins
	for _, p := range d.Plugins {
		switch p.Action {
		case diff.ActionAdd:
			if err := installPlugin(p); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionEnable, diff.ActionDisable:
			if err := updatePluginState(p); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Updated++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "plugin: "+p.Name)
		}
	}

	// Process MCP servers
	for _, m := range d.MCPServers {
		switch m.Action {
		case diff.ActionAdd:
			if m.RequiresOAuth {
				result.Attention = append(result.Attention, "mcp (oauth): "+m.Name)
				result.Skipped++
			} else if err := addMCPServer(m); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "mcp: "+m.Name)
		}
	}

	return result, nil
}

func addMarketplace(m diff.MarketplaceDiff) error {
	// TODO: Execute claude plugin marketplace add
	return nil
}

func installPlugin(p diff.PluginDiff) error {
	// TODO: Execute claude plugin install
	return nil
}

func updatePluginState(p diff.PluginDiff) error {
	// TODO: Execute claude plugin enable/disable
	return nil
}

func addMCPServer(m diff.MCPServerDiff) error {
	// TODO: Execute claude mcp add
	return nil
}
