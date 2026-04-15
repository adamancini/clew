package sync

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/state"
)

// MockCommandRunner records commands for testing.
type MockCommandRunner struct {
	Commands []string
	Outputs  map[string][]byte
	Errors   map[string]error
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.Commands = append(m.Commands, cmd)

	if err, ok := m.Errors[cmd]; ok {
		return nil, err
	}
	if output, ok := m.Outputs[cmd]; ok {
		return output, nil
	}
	return []byte("success"), nil
}

func (m *MockCommandRunner) RunInDir(dir, name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.Commands = append(m.Commands, cmd)

	if err, ok := m.Errors[cmd]; ok {
		return nil, err
	}
	if output, ok := m.Outputs[cmd]; ok {
		return output, nil
	}
	return []byte("Already up to date."), nil
}

func newMockSyncer() (*Syncer, *MockCommandRunner) {
	mock := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	return NewSyncerWithRunner(mock), mock
}

func TestAddLocalPathMarketplace(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MarketplaceDiff{
		Alias:  "local-tools",
		Action: diff.ActionAdd,
		Desired: &config.Marketplace{
			Path: "/tmp/local-tools",
		},
	}

	op, err := syncer.addMarketplace(m)
	if err != nil {
		t.Fatalf("addMarketplace() error = %v", err)
	}

	expected := "claude plugin marketplace add /tmp/local-tools"
	if len(mock.Commands) != 1 || mock.Commands[0] != expected {
		t.Errorf("Command = %v, want [%q]", mock.Commands, expected)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
}

func TestGitPullMarketplace(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MarketplaceDiff{
		Alias:  "local-tools",
		Action: diff.ActionGitUpdate,
		Desired: &config.Marketplace{
			Path: "/tmp/local-tools",
		},
	}

	op, err := syncer.gitPullMarketplace(m)
	if err != nil {
		t.Fatalf("gitPullMarketplace() error = %v", err)
	}

	if len(mock.Commands) != 1 || mock.Commands[0] != "git pull" {
		t.Errorf("Commands = %v, want [\"git pull\"]", mock.Commands)
	}
	if op.Action != "git_update" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "git_update")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
}

func TestExecuteGitUpdate(t *testing.T) {
	syncer, _ := newMockSyncer()

	d := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Alias:  "local-tools",
				Action: diff.ActionGitUpdate,
				Desired: &config.Marketplace{
					Path: "/tmp/local-tools",
				},
			},
		},
		Plugins: []diff.PluginDiff{},
	}

	result, err := syncer.Execute(d, Options{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
}

func TestAddMarketplace(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MarketplaceDiff{
		Alias:  "test-marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Marketplace{
			Repo: "owner/test-marketplace",
		},
	}

	op, err := syncer.addMarketplace(m)
	if err != nil {
		t.Fatalf("addMarketplace() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	expected := "claude plugin marketplace add owner/test-marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "marketplace" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "marketplace")
	}
	if op.Name != "test-marketplace" {
		t.Errorf("Operation.Name = %q, want %q", op.Name, "test-marketplace")
	}
	if op.Action != "add" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "add")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
}

func TestInstallPlugin(t *testing.T) {
	syncer, mock := newMockSyncer()

	p := diff.PluginDiff{
		Name:   "test-plugin@marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Plugin{
			Name:  "test-plugin@marketplace",
			Scope: "user",
		},
	}

	op, err := syncer.installPlugin(p)
	if err != nil {
		t.Fatalf("installPlugin() error = %v", err)
	}

	expected := "claude plugin install test-plugin@marketplace --scope user"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "plugin" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "plugin")
	}
	if op.Name != "test-plugin@marketplace" {
		t.Errorf("Operation.Name = %q, want %q", op.Name, "test-plugin@marketplace")
	}
	if op.Action != "add" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "add")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
}

func TestUpdatePluginStateEnable(t *testing.T) {
	syncer, mock := newMockSyncer()

	p := diff.PluginDiff{
		Name:   "test-plugin@marketplace",
		Action: diff.ActionEnable,
		Current: &state.PluginState{
			Name:    "test-plugin",
			Enabled: false,
		},
	}

	op, err := syncer.updatePluginState(p)
	if err != nil {
		t.Fatalf("updatePluginState() error = %v", err)
	}

	expected := "claude plugin enable test-plugin@marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "plugin" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "plugin")
	}
	if op.Action != "enable" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "enable")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
}

func TestUpdatePluginStateDisable(t *testing.T) {
	syncer, mock := newMockSyncer()

	p := diff.PluginDiff{
		Name:   "test-plugin@marketplace",
		Action: diff.ActionDisable,
		Current: &state.PluginState{
			Name:    "test-plugin",
			Enabled: true,
		},
	}

	op, err := syncer.updatePluginState(p)
	if err != nil {
		t.Fatalf("updatePluginState() error = %v", err)
	}

	expected := "claude plugin disable test-plugin@marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "plugin" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "plugin")
	}
	if op.Action != "disable" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "disable")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
}

func TestExecuteFullSync(t *testing.T) {
	syncer, _ := newMockSyncer()

	d := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Alias:  "new-marketplace",
				Action: diff.ActionAdd,
				Desired: &config.Marketplace{
					Repo: "owner/new",
				},
			},
			{
				Alias:  "extra-marketplace",
				Action: diff.ActionRemove,
			},
		},
		Plugins: []diff.PluginDiff{
			{
				Name:    "new-plugin@new-marketplace",
				Action:  diff.ActionAdd,
				Desired: &config.Plugin{Name: "new-plugin@new-marketplace"},
			},
			{
				Name:   "enable-plugin@marketplace",
				Action: diff.ActionEnable,
			},
		},
	}

	result, err := syncer.Execute(d, Options{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should have executed: marketplace add, plugin install, plugin enable
	if result.Installed != 2 {
		t.Errorf("Installed = %d, want 2", result.Installed)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}
	if len(result.Attention) != 1 {
		t.Errorf("Attention items = %d, want 1 (extra marketplace)", len(result.Attention))
	}
}

func TestExecuteWithErrors(t *testing.T) {
	syncer, mock := newMockSyncer()
	// Set up error for marketplace add command
	mock.Errors["claude plugin marketplace add owner/failing"] = fmt.Errorf("connection failed")

	d := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Alias:  "failing",
				Action: diff.ActionAdd,
				Desired: &config.Marketplace{
					Repo: "owner/failing",
				},
			},
		},
		Plugins: []diff.PluginDiff{},
	}

	result, err := syncer.Execute(d, Options{})
	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors count = %d, want 1", len(result.Errors))
	}
}

func TestInstallPluginAlwaysUserScope(t *testing.T) {
	syncer, mock := newMockSyncer()

	// Even with empty scope, install should always include --scope user
	p := diff.PluginDiff{
		Name:   "test-plugin@marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Plugin{
			Name:  "test-plugin@marketplace",
			Scope: "",
		},
	}

	op, err := syncer.installPlugin(p)
	if err != nil {
		t.Fatalf("installPlugin() error = %v", err)
	}

	expected := "claude plugin install test-plugin@marketplace --scope user"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if op.Command != expected {
		t.Errorf("Operation.Command = %q, want %q", op.Command, expected)
	}
}

// MockFileEditor records file operations for testing.
type MockFileEditor struct {
	Files map[string][]byte
}

func (m *MockFileEditor) ReadFile(path string) ([]byte, error) {
	if data, ok := m.Files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileEditor) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}
