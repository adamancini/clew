// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/adamancini/clew/internal/diff"
)

// CommandRunner is an interface for running external commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// DefaultCommandRunner uses os/exec to run commands.
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// addSource executes `claude plugin marketplace add <source>` for marketplace-kind sources.
func (s *Syncer) addSource(src diff.SourceDiff) (Operation, error) {
	op := Operation{
		Type:   "source",
		Name:   src.Name,
		Action: "add",
	}

	if src.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for source %s", src.Name)
		return op, fmt.Errorf("no desired state for source %s", src.Name)
	}

	// Only marketplace-kind sources can be added via CLI
	if src.Desired.Kind != "marketplace" {
		op.Success = true
		op.Skipped = true
		op.Description = fmt.Sprintf("Skip non-marketplace source (kind=%s): %s", src.Desired.Kind, src.Name)
		return op, nil
	}

	var source string
	switch src.Desired.Source.Type {
	case "github":
		source = src.Desired.Source.URL
		op.Description = fmt.Sprintf("Add GitHub source: %s", source)
	case "local":
		source = src.Desired.Source.Path
		op.Description = fmt.Sprintf("Add local source: %s", source)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unknown source type: %s", src.Desired.Source.Type)
		return op, fmt.Errorf("unknown source type: %s", src.Desired.Source.Type)
	}

	// Build command string before executing
	op.Command = fmt.Sprintf("claude plugin marketplace add %s", source)

	output, err := s.runner.Run("claude", "plugin", "marketplace", "add", source)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add source %s: %v\nOutput: %s", src.Name, err, string(output))
		return op, fmt.Errorf("failed to add source %s: %w\nOutput: %s", src.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// installPlugin executes `claude plugin install <plugin>`.
func (s *Syncer) installPlugin(p diff.PluginDiff) (Operation, error) {
	op := Operation{
		Type:        "plugin",
		Name:        p.Name,
		Action:      "add",
		Description: fmt.Sprintf("Install plugin: %s", p.Name),
	}

	if p.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for plugin %s", p.Name)
		return op, fmt.Errorf("no desired state for plugin %s", p.Name)
	}

	args := []string{"plugin", "install", p.Desired.Name}

	// Add scope if specified
	if p.Desired.Scope != "" {
		args = append(args, "--scope", p.Desired.Scope)
	}

	// Build command string before executing
	op.Command = "claude " + strings.Join(args, " ")

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to install plugin %s: %v\nOutput: %s", p.Name, err, string(output))
		return op, fmt.Errorf("failed to install plugin %s: %w\nOutput: %s", p.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// updatePluginState executes `claude plugin enable/disable <plugin>`.
func (s *Syncer) updatePluginState(p diff.PluginDiff) (Operation, error) {
	op := Operation{
		Type: "plugin",
		Name: p.Name,
	}

	var action string
	switch p.Action {
	case diff.ActionEnable:
		action = "enable"
		op.Action = "enable"
		op.Description = fmt.Sprintf("Enable plugin: %s", p.Name)
	case diff.ActionDisable:
		action = "disable"
		op.Action = "disable"
		op.Description = fmt.Sprintf("Disable plugin: %s", p.Name)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unexpected action for plugin state update: %s", p.Action)
		return op, fmt.Errorf("unexpected action for plugin state update: %s", p.Action)
	}

	// Build command string before executing
	op.Command = fmt.Sprintf("claude plugin %s %s", action, p.Name)

	output, err := s.runner.Run("claude", "plugin", action, p.Name)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to %s plugin %s: %v\nOutput: %s", action, p.Name, err, string(output))
		return op, fmt.Errorf("failed to %s plugin %s: %w\nOutput: %s", action, p.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

// addMCPServer executes `claude mcp add <name> <command|url> [args...]`.
func (s *Syncer) addMCPServer(m diff.MCPServerDiff) (Operation, error) {
	op := Operation{
		Type:        "mcp",
		Name:        m.Name,
		Action:      "add",
		Description: fmt.Sprintf("Add MCP server: %s", m.Name),
	}

	if m.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for MCP server %s", m.Name)
		return op, fmt.Errorf("no desired state for MCP server %s", m.Name)
	}

	args := []string{"mcp", "add", "--transport", m.Desired.Transport}

	// Add scope if specified
	if m.Desired.Scope != "" {
		args = append(args, "--scope", m.Desired.Scope)
	}

	// Add environment variables
	for key, value := range m.Desired.Env {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
	}

	// Add headers (for HTTP/SSE)
	for key, value := range m.Desired.Headers {
		args = append(args, "--header", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the server name
	args = append(args, m.Name)

	// Add command/URL and args based on transport
	switch m.Desired.Transport {
	case "stdio":
		if m.Desired.Command == "" {
			op.Success = false
			op.Error = fmt.Sprintf("stdio MCP server %s requires a command", m.Name)
			return op, fmt.Errorf("stdio MCP server %s requires a command", m.Name)
		}
		// Add -- separator before command to prevent flag parsing issues
		args = append(args, "--")
		args = append(args, m.Desired.Command)
		args = append(args, m.Desired.Args...)
		op.Description = fmt.Sprintf("Add stdio MCP server: %s (command: %s)", m.Name, m.Desired.Command)
	case "http", "sse":
		if m.Desired.URL == "" {
			op.Success = false
			op.Error = fmt.Sprintf("http/sse MCP server %s requires a URL", m.Name)
			return op, fmt.Errorf("http/sse MCP server %s requires a URL", m.Name)
		}
		args = append(args, m.Desired.URL)
		op.Description = fmt.Sprintf("Add %s MCP server: %s (url: %s)", m.Desired.Transport, m.Name, m.Desired.URL)
	default:
		op.Success = false
		op.Error = fmt.Sprintf("unknown MCP transport: %s", m.Desired.Transport)
		return op, fmt.Errorf("unknown MCP transport: %s", m.Desired.Transport)
	}

	// Build command string before executing
	op.Command = "claude " + strings.Join(args, " ")

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add MCP server %s: %v\nOutput: %s", m.Name, err, string(output))
		return op, fmt.Errorf("failed to add MCP server %s: %w\nOutput: %s", m.Name, err, string(output))
	}

	op.Success = true
	return op, nil
}

