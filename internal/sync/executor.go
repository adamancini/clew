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
	RunInDir(dir, name string, args ...string) ([]byte, error)
}

// DefaultCommandRunner uses os/exec to run commands.
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func (r *DefaultCommandRunner) RunInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// addMarketplace executes `claude plugin marketplace add <source>`.
// Source is the repo URL for remote marketplaces or the local path for git checkouts.
func (s *Syncer) addMarketplace(m diff.MarketplaceDiff) (Operation, error) {
	op := Operation{
		Type:   "marketplace",
		Name:   m.Alias,
		Action: "add",
	}

	if m.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for marketplace %s", m.Alias)
		return op, fmt.Errorf("no desired state for marketplace %s", m.Alias)
	}

	source := m.Desired.Source()
	op.Description = fmt.Sprintf("Add marketplace: %s (%s)", m.Alias, source)
	op.Command = fmt.Sprintf("claude plugin marketplace add %s", source)

	output, err := s.runner.Run("claude", "plugin", "marketplace", "add", source)
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("failed to add marketplace %s: %v\nOutput: %s", m.Alias, err, string(output))
		return op, fmt.Errorf("failed to add marketplace %s: %w\nOutput: %s", m.Alias, err, string(output))
	}

	op.Success = true
	return op, nil
}

// gitPullMarketplace runs `git pull` inside the local marketplace checkout.
// It is used for ActionGitUpdate — local marketplaces with auto_update: true.
func (s *Syncer) gitPullMarketplace(m diff.MarketplaceDiff) (Operation, error) {
	op := Operation{
		Type:   "marketplace",
		Name:   m.Alias,
		Action: "git_update",
	}

	if m.Desired == nil {
		op.Success = false
		op.Error = fmt.Sprintf("no desired state for marketplace %s", m.Alias)
		return op, fmt.Errorf("no desired state for marketplace %s", m.Alias)
	}

	path := m.Desired.Source()
	op.Description = fmt.Sprintf("Pull latest changes for local marketplace: %s (%s)", m.Alias, path)
	op.Command = fmt.Sprintf("git -C %s pull", path)

	output, err := s.runner.RunInDir(path, "git", "pull")
	if err != nil {
		op.Success = false
		op.Error = fmt.Sprintf("git pull failed for marketplace %s (%s): %v\nOutput: %s", m.Alias, path, err, string(output))
		return op, fmt.Errorf("git pull failed for marketplace %s: %w\nOutput: %s", m.Alias, err, string(output))
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

	// clew 1.0 always installs at user scope
	args = append(args, "--scope", "user")

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
