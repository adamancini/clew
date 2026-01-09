// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"github.com/adamancini/clew/internal/diff"
)

// Operation represents a single sync operation performed.
type Operation struct {
	Type        string `json:"type"`            // "marketplace", "plugin", or "mcp"
	Name        string `json:"name"`            // Item name
	Action      string `json:"action"`          // "add", "enable", "disable"
	Command     string `json:"command"`         // CLI command executed
	Description string `json:"description"`     // Human-readable description
	Success     bool   `json:"success"`         // Whether operation succeeded
	Error       string `json:"error,omitempty"` // Error message if failed
}

// Result represents the outcome of a sync operation.
type Result struct {
	Installed  int
	Updated    int
	Skipped    int
	Failed     int
	Attention  []string    // Items needing manual attention
	Errors     []error     // Detailed error objects (not serialized to JSON)
	Operations []Operation `json:"operations"` // Individual operations performed (always included in JSON)
}

// Options configures sync behavior.
type Options struct {
	Strict  bool // Exit non-zero on any failure
	Verbose bool
	Quiet   bool
	Short   bool // One-line-per-item output format
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
	result := &Result{
		Operations: []Operation{},
	}

	// Process marketplaces first (plugins depend on them)
	for _, m := range d.Marketplaces {
		switch m.Action {
		case diff.ActionAdd:
			op, err := s.addMarketplace(m)
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "marketplace: "+m.Name)
		case diff.ActionSkipGit:
			// Skipped due to git status issues
			result.Skipped++
			result.Attention = append(result.Attention, "marketplace (git): "+m.Name+" - has uncommitted changes")
		}
	}

	// Process plugins
	for _, p := range d.Plugins {
		switch p.Action {
		case diff.ActionAdd:
			op, err := s.installPlugin(p)
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Installed++
			}
		case diff.ActionEnable, diff.ActionDisable:
			op, err := s.updatePluginState(p)
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else {
				result.Updated++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "plugin: "+p.Name)
		case diff.ActionSkipGit:
			// Skipped due to git status issues
			result.Skipped++
			result.Attention = append(result.Attention, "plugin (git): "+p.Name+" - has uncommitted changes")
		}
	}

	// Process MCP servers
	for _, m := range d.MCPServers {
		switch m.Action {
		case diff.ActionAdd:
			if m.RequiresOAuth {
				result.Attention = append(result.Attention, "mcp (oauth): "+m.Name)
				result.Skipped++
			} else {
				op, err := s.addMCPServer(m)
				result.Operations = append(result.Operations, op)
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, err)
				} else {
					result.Installed++
				}
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "mcp: "+m.Name)
		}
	}

	return result, nil
}

