package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestAddSourceGitHub(t *testing.T) {
	syncer, mock := newMockSyncer()

	src := diff.SourceDiff{
		Name:   "test-marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Source{
			Name: "test-marketplace",
			Kind: config.SourceKindMarketplace,
			Source: config.SourceConfig{
				Type: config.SourceTypeGitHub,
				URL:  "owner/test-marketplace",
			},
		},
	}

	op, err := syncer.addSource(src)
	if err != nil {
		t.Fatalf("addSource() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	expected := "claude plugin marketplace add owner/test-marketplace"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "source" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "source")
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

func TestAddSourceLocal(t *testing.T) {
	syncer, mock := newMockSyncer()

	src := diff.SourceDiff{
		Name:   "local-marketplace",
		Action: diff.ActionAdd,
		Desired: &config.Source{
			Name: "local-marketplace",
			Kind: config.SourceKindMarketplace,
			Source: config.SourceConfig{
				Type: config.SourceTypeLocal,
				Path: "/path/to/plugins",
			},
		},
	}

	op, err := syncer.addSource(src)
	if err != nil {
		t.Fatalf("addSource() error = %v", err)
	}

	expected := "claude plugin marketplace add /path/to/plugins"
	if mock.Commands[0] != expected {
		t.Errorf("Command = %q, want %q", mock.Commands[0], expected)
	}

	// Verify Operation struct
	if op.Type != "source" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "source")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
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
		Sources: []diff.SourceDiff{
			{
				Name:   "new-marketplace",
				Action: diff.ActionAdd,
				Desired: &config.Source{
					Name: "new-marketplace",
					Kind: config.SourceKindMarketplace,
					Source: config.SourceConfig{
						Type: config.SourceTypeGitHub,
						URL:  "owner/new",
					},
				},
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

	// Should have executed: source add, plugin install, plugin enable, mcp add
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
		t.Errorf("Attention items = %d, want 2 (extra source + OAuth server)", len(result.Attention))
	}
}

func TestExecuteWithErrors(t *testing.T) {
	syncer, mock := newMockSyncer()
	// Set up error for source add command
	mock.Errors["claude plugin marketplace add owner/failing"] = fmt.Errorf("connection failed")

	d := &diff.Result{
		Sources: []diff.SourceDiff{
			{
				Name:   "failing",
				Action: diff.ActionAdd,
				Desired: &config.Source{
					Name: "failing",
					Kind: config.SourceKindMarketplace,
					Source: config.SourceConfig{
						Type: config.SourceTypeGitHub,
						URL:  "owner/failing",
					},
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

func newMockFileEditor() *MockFileEditor {
	return &MockFileEditor{
		Files: make(map[string][]byte),
	}
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

func TestInstallLocalPlugin(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	reposDir := filepath.Join(pluginsDir, "repos")
	pluginDir := filepath.Join(reposDir, "test-local-plugin")

	// Create directories
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock plugin.json
	pluginJSON := `{
		"name": "test-local-plugin",
		"version": "1.2.3",
		"description": "A test plugin"
	}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize mock file editor with test data
	mockEditor := newMockFileEditor()
	mockEditor.Files[filepath.Join(pluginDir, "plugin.json")] = []byte(pluginJSON)

	// Create syncer with mock editor
	mockRunner := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	syncer := NewSyncerWithRunnerAndEditor(mockRunner, mockEditor, claudeDir)

	// Create plugin diff for local plugin
	p := diff.PluginDiff{
		Name:   "test-local-plugin",
		Action: diff.ActionAdd,
		Desired: &config.Plugin{
			Name:  "test-local-plugin",
			Scope: "user",
			Source: &config.SourceConfig{
				Type: config.SourceTypeLocal,
				Path: pluginDir,
			},
		},
	}

	op, err := syncer.installLocalPlugin(p)
	if err != nil {
		t.Fatalf("installLocalPlugin() error = %v", err)
	}

	// Verify Operation struct
	if op.Type != "plugin" {
		t.Errorf("Operation.Type = %q, want %q", op.Type, "plugin")
	}
	if op.Name != "test-local-plugin" {
		t.Errorf("Operation.Name = %q, want %q", op.Name, "test-local-plugin")
	}
	if op.Action != "add" {
		t.Errorf("Operation.Action = %q, want %q", op.Action, "add")
	}
	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}
	if !strings.Contains(op.Command, "installed_plugins.json") {
		t.Errorf("Operation.Command should mention installed_plugins.json: %s", op.Command)
	}
	if !strings.Contains(op.Command, "1.2.3") {
		t.Errorf("Operation.Command should contain version: %s", op.Command)
	}

	// Verify no claude commands were run
	if len(mockRunner.Commands) != 0 {
		t.Errorf("Expected 0 CLI commands for local plugin, got %d: %v", len(mockRunner.Commands), mockRunner.Commands)
	}

	// Verify installed_plugins.json was created
	installedPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	data, ok := mockEditor.Files[installedPath]
	if !ok {
		t.Fatal("installed_plugins.json was not created")
	}

	var installed installedPluginsFile
	if err := json.Unmarshal(data, &installed); err != nil {
		t.Fatalf("Failed to parse installed_plugins.json: %v", err)
	}

	// Verify plugin entry
	entries, ok := installed.Plugins["test-local-plugin"]
	if !ok {
		t.Fatal("Plugin entry not found in installed_plugins.json")
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", entry.Version, "1.2.3")
	}
	if entry.Scope != "user" {
		t.Errorf("Scope = %q, want %q", entry.Scope, "user")
	}
	if entry.InstallPath != pluginDir {
		t.Errorf("InstallPath = %q, want %q", entry.InstallPath, pluginDir)
	}
}

func TestInstallLocalPluginUpdateExisting(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	reposDir := filepath.Join(pluginsDir, "repos")
	pluginDir := filepath.Join(reposDir, "test-local-plugin")

	// Create directories
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock plugin.json with new version
	pluginJSON := `{
		"name": "test-local-plugin",
		"version": "2.0.0",
		"description": "Updated plugin"
	}`

	// Create existing installed_plugins.json
	existingInstalled := installedPluginsFile{
		Version: 2,
		Plugins: map[string][]pluginInstallInfo{
			"test-local-plugin": {
				{
					Scope:        "user",
					InstallPath:  pluginDir,
					Version:      "1.0.0",
					InstalledAt:  "2025-01-01T00:00:00Z",
					LastUpdated:  "2025-01-01T00:00:00Z",
					GitCommitSha: "abc123",
				},
			},
		},
	}
	existingData, _ := json.Marshal(existingInstalled)

	// Initialize mock file editor
	mockEditor := newMockFileEditor()
	mockEditor.Files[filepath.Join(pluginDir, "plugin.json")] = []byte(pluginJSON)
	mockEditor.Files[filepath.Join(claudeDir, "plugins", "installed_plugins.json")] = existingData

	// Create syncer with mock editor
	mockRunner := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	syncer := NewSyncerWithRunnerAndEditor(mockRunner, mockEditor, claudeDir)

	// Create plugin diff for local plugin
	p := diff.PluginDiff{
		Name:   "test-local-plugin",
		Action: diff.ActionAdd,
		Desired: &config.Plugin{
			Name:  "test-local-plugin",
			Scope: "user",
			Source: &config.SourceConfig{
				Type: config.SourceTypeLocal,
				Path: pluginDir,
			},
		},
	}

	op, err := syncer.installLocalPlugin(p)
	if err != nil {
		t.Fatalf("installLocalPlugin() error = %v", err)
	}

	if !op.Success {
		t.Errorf("Operation.Success = %v, want true", op.Success)
	}

	// Verify installed_plugins.json was updated
	installedPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	data := mockEditor.Files[installedPath]

	var installed installedPluginsFile
	if err := json.Unmarshal(data, &installed); err != nil {
		t.Fatalf("Failed to parse installed_plugins.json: %v", err)
	}

	// Verify plugin entry was updated
	entries := installed.Plugins["test-local-plugin"]
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q (should be updated)", entry.Version, "2.0.0")
	}
	// Original install time should be preserved
	if entry.InstalledAt != "2025-01-01T00:00:00Z" {
		t.Errorf("InstalledAt = %q, want original time preserved", entry.InstalledAt)
	}
}

func TestInstallLocalPluginMissingPluginJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginDir := filepath.Join(tmpDir, "nonexistent-plugin")

	// Initialize mock file editor with NO plugin.json
	mockEditor := newMockFileEditor()

	// Create syncer with mock editor
	mockRunner := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	syncer := NewSyncerWithRunnerAndEditor(mockRunner, mockEditor, claudeDir)

	// Create plugin diff for local plugin
	p := diff.PluginDiff{
		Name:   "test-local-plugin",
		Action: diff.ActionAdd,
		Desired: &config.Plugin{
			Name:  "test-local-plugin",
			Scope: "user",
			Source: &config.SourceConfig{
				Type: config.SourceTypeLocal,
				Path: pluginDir,
			},
		},
	}

	op, err := syncer.installLocalPlugin(p)
	if err == nil {
		t.Fatal("Expected error for missing plugin.json, got nil")
	}

	if op.Success {
		t.Errorf("Operation.Success = %v, want false", op.Success)
	}
	if op.Error == "" {
		t.Error("Operation.Error should contain error message")
	}
}

func TestExecuteLocalPluginAdd(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginDir := filepath.Join(tmpDir, "my-local-plugin")

	// Create mock plugin.json
	pluginJSON := `{"name": "my-local-plugin", "version": "1.0.0"}`

	// Initialize mock file editor
	mockEditor := newMockFileEditor()
	mockEditor.Files[filepath.Join(pluginDir, "plugin.json")] = []byte(pluginJSON)

	// Create syncer with mock editor
	mockRunner := &MockCommandRunner{
		Commands: []string{},
		Outputs:  make(map[string][]byte),
		Errors:   make(map[string]error),
	}
	syncer := NewSyncerWithRunnerAndEditor(mockRunner, mockEditor, claudeDir)

	// Create diff with local plugin
	d := &diff.Result{
		Sources: []diff.SourceDiff{},
		Plugins: []diff.PluginDiff{
			{
				Name:   "my-local-plugin",
				Action: diff.ActionAdd,
				Desired: &config.Plugin{
					Name:  "my-local-plugin",
					Scope: "user",
					Source: &config.SourceConfig{
						Type: config.SourceTypeLocal,
						Path: pluginDir,
					},
				},
			},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	result, err := syncer.Execute(d, Options{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should have installed 1 plugin
	if result.Installed != 1 {
		t.Errorf("Installed = %d, want 1", result.Installed)
	}

	// Should NOT have run any claude CLI commands
	if len(mockRunner.Commands) != 0 {
		t.Errorf("Expected 0 CLI commands for local plugin, got %d: %v", len(mockRunner.Commands), mockRunner.Commands)
	}

	// Verify installed_plugins.json was created
	installedPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	if _, ok := mockEditor.Files[installedPath]; !ok {
		t.Error("installed_plugins.json should have been created")
	}
}
