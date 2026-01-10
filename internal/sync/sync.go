// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"os"
	"path/filepath"

	"github.com/adamancini/clew/internal/diff"
)

// Operation represents a single sync operation performed.
type Operation struct {
	Type        string `json:"type"`            // "source", "plugin", or "mcp"
	Name        string `json:"name"`            // Item name
	Action      string `json:"action"`          // "add", "enable", "disable"
	Command     string `json:"command"`         // CLI command executed
	Description string `json:"description"`     // Human-readable description
	Success     bool   `json:"success"`         // Whether operation succeeded
	Skipped     bool   `json:"skipped"`         // Whether operation was skipped
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

// FileEditor is an interface for filesystem operations.
// This allows for mocking in tests.
type FileEditor interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

// DefaultFileEditor uses os package for file operations.
type DefaultFileEditor struct{}

func (e *DefaultFileEditor) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (e *DefaultFileEditor) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Syncer executes sync operations with a configurable command runner.
type Syncer struct {
	runner    CommandRunner
	editor    FileEditor
	claudeDir string // Path to ~/.claude directory
}

// NewSyncer creates a Syncer with the default command runner and file editor.
func NewSyncer() *Syncer {
	home, _ := os.UserHomeDir()
	return &Syncer{
		runner:    &DefaultCommandRunner{},
		editor:    &DefaultFileEditor{},
		claudeDir: filepath.Join(home, ".claude"),
	}
}

// NewSyncerWithRunner creates a Syncer with a custom command runner (for testing).
func NewSyncerWithRunner(runner CommandRunner) *Syncer {
	home, _ := os.UserHomeDir()
	return &Syncer{
		runner:    runner,
		editor:    &DefaultFileEditor{},
		claudeDir: filepath.Join(home, ".claude"),
	}
}

// NewSyncerWithRunnerAndEditor creates a Syncer with custom runner and editor (for testing).
func NewSyncerWithRunnerAndEditor(runner CommandRunner, editor FileEditor, claudeDir string) *Syncer {
	return &Syncer{
		runner:    runner,
		editor:    editor,
		claudeDir: claudeDir,
	}
}

// Execute applies the diff to bring current state in line with Clewfile.
func (s *Syncer) Execute(d *diff.Result, opts Options) (*Result, error) {
	result := &Result{
		Operations: []Operation{},
	}

	// Process sources first (plugins depend on them)
	for _, src := range d.Sources {
		switch src.Action {
		case diff.ActionAdd:
			op, err := s.addSource(src)
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else if op.Skipped {
				result.Skipped++
			} else {
				result.Installed++
			}
		case diff.ActionRemove:
			// Info only - don't remove
			result.Attention = append(result.Attention, "source: "+src.Name)
		case diff.ActionSkipGit:
			// Skipped due to git status issues
			result.Skipped++
			result.Attention = append(result.Attention, "source (git): "+src.Name+" - has uncommitted changes")
		}
	}

	// Process plugins
	for _, p := range d.Plugins {
		switch p.Action {
		case diff.ActionAdd:
			var op Operation
			var err error
			if p.IsLocal() {
				// Local plugins are installed by editing installed_plugins.json directly
				op, err = s.installLocalPlugin(p)
			} else {
				// Marketplace plugins use claude CLI
				op, err = s.installPlugin(p)
			}
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else if op.Skipped {
				result.Skipped++
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
		case diff.ActionUpdate:
			var op Operation
			var err error
			if p.IsLocal() {
				// Local plugin updates are handled by editing installed_plugins.json
				op, err = s.installLocalPlugin(p)
			} else {
				// Marketplace plugins - update would need reinstall
				op = Operation{
					Type:        "plugin",
					Name:        p.Name,
					Action:      "update",
					Description: "Marketplace plugin update requires manual reinstall",
					Success:     true,
					Skipped:     true,
				}
			}
			result.Operations = append(result.Operations, op)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err)
			} else if op.Skipped {
				result.Skipped++
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

