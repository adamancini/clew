// Package sync handles reconciliation of current state to match Clewfile.
package sync

import (
	"fmt"
	"os/exec"

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

// addMarketplace executes `claude plugin marketplace add <source>`.
func (s *Syncer) addMarketplace(m diff.MarketplaceDiff) error {
	if m.Desired == nil {
		return fmt.Errorf("no desired state for marketplace %s", m.Name)
	}

	var source string
	switch m.Desired.Source {
	case "github":
		source = m.Desired.Repo
	case "local":
		source = m.Desired.Path
	default:
		return fmt.Errorf("unknown marketplace source: %s", m.Desired.Source)
	}

	output, err := s.runner.Run("claude", "plugin", "marketplace", "add", source)
	if err != nil {
		return fmt.Errorf("failed to add marketplace %s: %w\nOutput: %s", m.Name, err, string(output))
	}

	return nil
}

// installPlugin executes `claude plugin install <plugin>`.
func (s *Syncer) installPlugin(p diff.PluginDiff) error {
	if p.Desired == nil {
		return fmt.Errorf("no desired state for plugin %s", p.Name)
	}

	args := []string{"plugin", "install", p.Desired.Name}

	// Add scope if specified
	if p.Desired.Scope != "" {
		args = append(args, "--scope", p.Desired.Scope)
	}

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		return fmt.Errorf("failed to install plugin %s: %w\nOutput: %s", p.Name, err, string(output))
	}

	return nil
}

// updatePluginState executes `claude plugin enable/disable <plugin>`.
func (s *Syncer) updatePluginState(p diff.PluginDiff) error {
	var action string
	switch p.Action {
	case diff.ActionEnable:
		action = "enable"
	case diff.ActionDisable:
		action = "disable"
	default:
		return fmt.Errorf("unexpected action for plugin state update: %s", p.Action)
	}

	output, err := s.runner.Run("claude", "plugin", action, p.Name)
	if err != nil {
		return fmt.Errorf("failed to %s plugin %s: %w\nOutput: %s", action, p.Name, err, string(output))
	}

	return nil
}

// addMCPServer executes `claude mcp add <name> <command|url> [args...]`.
func (s *Syncer) addMCPServer(m diff.MCPServerDiff) error {
	if m.Desired == nil {
		return fmt.Errorf("no desired state for MCP server %s", m.Name)
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
			return fmt.Errorf("stdio MCP server %s requires a command", m.Name)
		}
		// Add -- separator before command to prevent flag parsing issues
		args = append(args, "--")
		args = append(args, m.Desired.Command)
		args = append(args, m.Desired.Args...)
	case "http", "sse":
		if m.Desired.URL == "" {
			return fmt.Errorf("http/sse MCP server %s requires a URL", m.Name)
		}
		args = append(args, m.Desired.URL)
	default:
		return fmt.Errorf("unknown MCP transport: %s", m.Desired.Transport)
	}

	output, err := s.runner.Run("claude", args...)
	if err != nil {
		return fmt.Errorf("failed to add MCP server %s: %w\nOutput: %s", m.Name, err, string(output))
	}

	return nil
}

