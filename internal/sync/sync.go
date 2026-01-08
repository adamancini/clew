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

// Syncer executes sync operations with a configurable command runner.
type Syncer struct {
	runner CommandRunner
}

// NewSyncer creates a Syncer with the default command runner.
func NewSyncer() *Syncer {
	return &Syncer{runner: &DefaultCommandRunner{}}
}

// NewSyncerWithRunner creates a Syncer with a custom command runner (for testing).
func NewSyncerWithRunner(runner CommandRunner) *Syncer {
	return &Syncer{runner: runner}
}

// Execute applies the diff to bring current state in line with Clewfile.
func (s *Syncer) Execute(d *diff.Result, opts Options) (*Result, error) {
	result := &Result{}

	// Process marketplaces first (plugins depend on them)
	for _, m := range d.Marketplaces {
		switch m.Action {
		case diff.ActionAdd:
			if err := s.addMarketplace(m); err != nil {
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
			if err := s.installPlugin(p); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionEnable, diff.ActionDisable:
			if err := s.updatePluginState(p); err != nil {
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
			} else if err := s.addMCPServer(m); err != nil {
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

