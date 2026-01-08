package sync

import (
	"fmt"
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

func TestAddMarketplaceGitHub(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MarketplaceDiff{
		Name:   "test-marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Marketplace{
			Source: "github",
			Repo:   "owner/test-marketplace",
		},
	}

	err := syncer.addMarketplace(m)
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
}

func TestAddMarketplaceLocal(t *testing.T) {
	syncer, mock := newMockSyncer()

	m := diff.MarketplaceDiff{
		Name:   "local-marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Marketplace{
			Source: "local",
			Path:   "/path/to/plugins",
		},
	}

	err := syncer.addMarketplace(m)
	if err != nil {
		t.Fatalf("addMarketplace() error = %v", err)
	}

	expected := "claude plugin marketplace add /path/to/plugins"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
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

	err := syncer.installPlugin(p)
	if err != nil {
		t.Fatalf("installPlugin() error = %v", err)
	}

	expected := "claude plugin install test-plugin@marketplace --scope user"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
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

	err := syncer.updatePluginState(p)
	if err != nil {
		t.Fatalf("updatePluginState() error = %v", err)
	}

	expected := "claude plugin enable test-plugin@marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
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

	err := syncer.updatePluginState(p)
	if err != nil {
		t.Fatalf("updatePluginState() error = %v", err)
	}

	expected := "claude plugin disable test-plugin@marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
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

	err := syncer.addMCPServer(m)
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

	err := syncer.addMCPServer(m)
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

	err := syncer.addMCPServer(m)
	if err != nil {
		t.Fatalf("addMCPServer() error = %v", err)
	}

	cmd := mock.Commands[0]
	if !strings.Contains(cmd, "--env AIRTABLE_API_KEY=secret") {
		t.Errorf("Command should contain env var: %s", cmd)
	}
}

func TestExecuteFullSync(t *testing.T) {
	syncer, _ := newMockSyncer()

	d := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Name:    "new-marketplace",
				Action:  diff.ActionAdd,
				Desired: &config.Marketplace{Source: "github", Repo: "owner/new"},
			},
			{
				Name:   "extra-marketplace",
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
				Name:    "failing",
				Action:  diff.ActionAdd,
				Desired: &config.Marketplace{Source: "github", Repo: "owner/failing"},
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
