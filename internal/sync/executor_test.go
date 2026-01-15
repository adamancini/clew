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

func newMockSyncer() (*Syncer, *MockCommandRunner) {
	mock := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	return NewSyncerWithRunner(mock), mock
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

func TestAddMCPServerStdio(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MCPServerDiff{
		Name:   "filesystem",
		Action: diff.ActionAdd,
		Desired: &config.MCPServer{
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		},
	}

	op, err := syncer.addMCPServer(m)
	if err != nil {
		t.Fatalf("addMCPServer() error = %v", err)
	}

	// Check the command was formed correctly
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if !strings.Contains(cmd, "--transport stdio") {
		t.Errorf("Command should contain --transport stdio: %s", cmd)
	}
	if !strings.Contains(cmd, "filesystem") {
		t.Errorf("Command should contain server name: %s", cmd)
	}
	if !strings.Contains(cmd, "npx") {
		t.Errorf("Command should contain npx: %s", cmd)
	}

	// Verify Operation struct
	if op.Type != "mcp" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "mcp")
	}
	if op.Name != "filesystem" {
		t.Errorf("Operation.Name = %q, want %q", op.Name, "filesystem")
	}
	if op.Action != "add" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "add")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
}

func TestAddMCPServerHTTP(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MCPServerDiff{
		Name:   "sentry",
		Action: diff.ActionAdd,
		Desired: &config.MCPServer{
			Transport: "http",
			URL:       "https://mcp.sentry.dev/mcp",
		},
	}

	op, err := syncer.addMCPServer(m)
	if err != nil {
		t.Fatalf("addMCPServer() error = %v", err)
	}

	cmd := mock.Commands[0]
	if !strings.Contains(cmd, "--transport http") {
		t.Errorf("Command should contain --transport http: %s", cmd)
	}
	if !strings.Contains(cmd, "https://mcp.sentry.dev/mcp") {
		t.Errorf("Command should contain URL: %s", cmd)
	}

	// Verify Operation struct
	if op.Type != "mcp" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "mcp")
	}
	if op.Name != "sentry" {
		t.Errorf("Operation.Name = %q, want %q", op.Name, "sentry")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
}

func TestAddMCPServerWithEnv(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MCPServerDiff{
		Name:   "airtable",
		Action: diff.ActionAdd,
		Desired: &config.MCPServer{
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "airtable-mcp-server"},
			Env:       map[string]string{"AIRTABLE_API_KEY": "secret"},
		},
	}

	op, err := syncer.addMCPServer(m)
	if err != nil {
		t.Fatalf("addMCPServer() error = %v", err)
	}

	cmd := mock.Commands[0]
	if !strings.Contains(cmd, "--env AIRTABLE_API_KEY=secret") {
		t.Errorf("Command should contain env var: %s", cmd)
	}

	// Verify Operation struct
	if op.Type != "mcp" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "mcp")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
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
		MCPServers: []diff.MCPServerDiff{
			{
				Name:          "oauth-server",
				Action:        diff.ActionAdd,
				RequiresOAuth: true,
				Desired:       &config.MCPServer{Transport: "http", URL: "https://example.com"},
			},
			{
				Name:    "stdio-server",
				Action:  diff.ActionAdd,
				Desired: &config.MCPServer{Transport: "stdio", Command: "cmd"},
			},
		},
	}

	result, err := syncer.Execute(d, Options{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should have executed: marketplace add, plugin install, plugin enable, mcp add
	// OAuth server should be skipped
	if result.Installed != 3 {
		t.Errorf("Installed = %d, want 3", result.Installed)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (OAuth server)", result.Skipped)
	}
	if len(result.Attention) != 2 {
		t.Errorf("Attention items = %d, want 2 (extra marketplace + OAuth server)", len(result.Attention))
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
		Plugins:    []diff.PluginDiff{},
		MCPServers: []diff.MCPServerDiff{},
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
