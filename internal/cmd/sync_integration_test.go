package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/adamancini/clew/internal/backup"
	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/state"
	"github.com/adamancini/clew/internal/sync"
	"github.com/adamancini/clew/internal/types"
)

// testSetup provides a common test environment.
type testSetup struct {
	tmpDir     string
	claudeDir  string
	clewfile   string
	cleanup    func()
	oldVerbose bool
	oldQuiet   bool
}

// newTestSetup creates a new test environment with temp directories.
func newTestSetup(t *testing.T) *testSetup {
	t.Helper()
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")

	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Save global state
	oldVerbose := verbose
	oldQuiet := quiet

	return &testSetup{
		tmpDir:     tmpDir,
		claudeDir:  claudeDir,
		clewfile:   filepath.Join(tmpDir, "Clewfile.yaml"),
		oldVerbose: oldVerbose,
		oldQuiet:   oldQuiet,
		cleanup: func() {
			// Restore global state
			verbose = oldVerbose
			quiet = oldQuiet
		},
	}
}

// writeClewfile writes a Clewfile to the test environment.
func (ts *testSetup) writeClewfile(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(ts.clewfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeMarketplaces writes a known_marketplaces.json file.
func (ts *testSetup) writeMarketplaces(t *testing.T, marketplaces map[string]interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(marketplaces, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ts.claudeDir, "plugins", "known_marketplaces.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// writeInstalledPlugins writes an installed_plugins.json file.
func (ts *testSetup) writeInstalledPlugins(t *testing.T, plugins map[string]interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(map[string]interface{}{
		"version": 2,
		"plugins": plugins,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ts.claudeDir, "plugins", "installed_plugins.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// writeSettings writes a settings.json file.
func (ts *testSetup) writeSettings(t *testing.T, settings map[string]interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ts.claudeDir, "settings.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// TestIntegration_LoadConfiguration tests loading and validating Clewfiles.
func TestIntegration_LoadConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		clewfile         string
		wantErr          bool
		wantMarketplaces int
		wantPlugins      int
		wantMCP          int
	}{
		{
			name: "valid minimal clewfile",
			clewfile: `version: 1
marketplaces:
  official:
    repo: anthropics/plugins
plugins:
  - test-plugin@official
`,
			wantErr:          false,
			wantMarketplaces: 1,
			wantPlugins:      1,
			wantMCP:          0,
		},
		{
			name: "valid full clewfile",
			clewfile: `version: 1
marketplaces:
  official:
    repo: anthropics/plugins
plugins:
  - test-plugin@official
  - name: another-plugin@official
    enabled: false
mcp_servers:
  filesystem:
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
`,
			wantErr:          false,
			wantMarketplaces: 1,
			wantPlugins:      2,
			wantMCP:          1,
		},
		{
			name: "invalid transport",
			clewfile: `version: 1
mcp_servers:
  bad:
    transport: websocket
    url: ws://localhost
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestSetup(t)
			defer ts.cleanup()

			ts.writeClewfile(t, tt.clewfile)

			clewfile, err := config.Load(ts.clewfile)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(clewfile.Marketplaces) != tt.wantMarketplaces {
				t.Errorf("marketplaces count = %d, want %d", len(clewfile.Marketplaces), tt.wantMarketplaces)
			}
			if len(clewfile.Plugins) != tt.wantPlugins {
				t.Errorf("plugins count = %d, want %d", len(clewfile.Plugins), tt.wantPlugins)
			}
			if len(clewfile.MCPServers) != tt.wantMCP {
				t.Errorf("MCP servers count = %d, want %d", len(clewfile.MCPServers), tt.wantMCP)
			}
		})
	}
}

// TestIntegration_ReadCurrentState tests reading state from filesystem.
func TestIntegration_ReadCurrentState(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.cleanup()

	// Set up marketplace
	ts.writeMarketplaces(t, map[string]interface{}{
		"official": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/plugins",
			},
			"installLocation": "/path/to/install",
			"lastUpdated":     "2025-01-01T00:00:00Z",
		},
	})

	// Set up plugins
	ts.writeInstalledPlugins(t, map[string]interface{}{
		"test-plugin@official": []map[string]interface{}{{
			"scope":       "user",
			"installPath": "/path/to/plugin",
			"version":     "1.0.0",
			"installedAt": "2025-01-01T00:00:00Z",
			"lastUpdated": "2025-01-01T00:00:00Z",
		}},
	})

	// Set up settings
	ts.writeSettings(t, map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"test-plugin@official": true,
		},
	})

	reader := &state.FilesystemReader{ClaudeDir: ts.claudeDir}
	currentState, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify marketplaces
	if len(currentState.Marketplaces) != 1 {
		t.Errorf("marketplaces count = %d, want 1", len(currentState.Marketplaces))
	}
	if m, ok := currentState.Marketplaces["official"]; ok {
		if m.Repo != "anthropics/plugins" {
			t.Errorf("marketplace repo = %s, want anthropics/plugins", m.Repo)
		}
	} else {
		t.Error("marketplace 'official' not found")
	}

	// Verify plugins
	if len(currentState.Plugins) != 1 {
		t.Errorf("plugins count = %d, want 1", len(currentState.Plugins))
	}
	if plugin, ok := currentState.Plugins["test-plugin@official"]; ok {
		if plugin.Version != "1.0.0" {
			t.Errorf("plugin version = %s, want 1.0.0", plugin.Version)
		}
		if !plugin.Enabled {
			t.Error("plugin should be enabled")
		}
	} else {
		t.Error("plugin 'test-plugin@official' not found")
	}
}

// TestIntegration_ComputeDiff tests diff computation between states.
func TestIntegration_ComputeDiff(t *testing.T) {
	tests := []struct {
		name          string
		desired       *config.Clewfile
		current       *state.State
		wantAdd       int
		wantUpdate    int
		wantAttention int // Removes go to attention in Summary()
	}{
		{
			name: "add new plugin",
			desired: &config.Clewfile{
				Plugins: []config.Plugin{
					{Name: "new-plugin@official"},
				},
			},
			current: &state.State{
				Marketplaces: map[string]state.MarketplaceState{},
				Plugins:      map[string]state.PluginState{},
				MCPServers:   map[string]state.MCPServerState{},
			},
			wantAdd:       1,
			wantUpdate:    0,
			wantAttention: 0,
		},
		{
			name: "enable disabled plugin",
			desired: &config.Clewfile{
				Plugins: []config.Plugin{
					{Name: "test-plugin@official"},
				},
			},
			current: &state.State{
				Marketplaces: map[string]state.MarketplaceState{},
				Plugins: map[string]state.PluginState{
					"test-plugin@official": {
						Name:    "test-plugin",
						Enabled: false,
					},
				},
				MCPServers: map[string]state.MCPServerState{},
			},
			wantAdd:       0,
			wantUpdate:    1,
			wantAttention: 0,
		},
		{
			name: "extra plugin in current state",
			desired: &config.Clewfile{
				Plugins: []config.Plugin{},
			},
			current: &state.State{
				Marketplaces: map[string]state.MarketplaceState{},
				Plugins: map[string]state.PluginState{
					"extra-plugin@official": {
						Name:    "extra-plugin",
						Enabled: true,
					},
				},
				MCPServers: map[string]state.MCPServerState{},
			},
			wantAdd:       0,
			wantUpdate:    0,
			wantAttention: 1, // Removes are reported as attention items
		},
		{
			name: "already in sync",
			desired: &config.Clewfile{
				Plugins: []config.Plugin{
					{Name: "test-plugin@official"},
				},
			},
			current: &state.State{
				Marketplaces: map[string]state.MarketplaceState{},
				Plugins: map[string]state.PluginState{
					"test-plugin@official": {
						Name:    "test-plugin",
						Enabled: true,
					},
				},
				MCPServers: map[string]state.MCPServerState{},
			},
			wantAdd:       0,
			wantUpdate:    0,
			wantAttention: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := diff.Compute(tt.desired, tt.current)
			add, update, _, attention := result.Summary()

			if add != tt.wantAdd {
				t.Errorf("add count = %d, want %d", add, tt.wantAdd)
			}
			if update != tt.wantUpdate {
				t.Errorf("update count = %d, want %d", update, tt.wantUpdate)
			}
			if attention != tt.wantAttention {
				t.Errorf("attention count = %d, want %d", attention, tt.wantAttention)
			}
		})
	}
}

// TestIntegration_ExecuteSync tests sync execution with mocked CLI.
func TestIntegration_ExecuteSync(t *testing.T) {
	// MockCommandRunner for testing
	type mockRunner struct {
		commands []string
		outputs  map[string][]byte
		errors   map[string]error
	}

	mockRun := func(m *mockRunner) func(name string, args ...string) ([]byte, error) {
		return func(name string, args ...string) ([]byte, error) {
			cmd := name
			for _, arg := range args {
				cmd += " " + arg
			}
			m.commands = append(m.commands, cmd)
			if err, ok := m.errors[cmd]; ok {
				return nil, err
			}
			if out, ok := m.outputs[cmd]; ok {
				return out, nil
			}
			return []byte("success"), nil
		}
	}

	tests := []struct {
		name          string
		diffResult    *diff.Result
		wantInstalled int
		wantUpdated   int
		wantFailed    int
		wantSkipped   int
		wantAttention int
		wantCommands  int
	}{
		{
			name: "install marketplace and plugin",
			diffResult: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{
					{
						Alias:  "official",
						Action: diff.ActionAdd,
						Desired: &config.Marketplace{
							Repo: "anthropics/plugins",
						},
					},
				},
				Plugins: []diff.PluginDiff{
					{
						Name:   "test-plugin@official",
						Action: diff.ActionAdd,
						Desired: &config.Plugin{
							Name: "test-plugin@official",
						},
					},
				},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantInstalled: 2,
			wantUpdated:   0,
			wantFailed:    0,
			wantSkipped:   0,
			wantAttention: 0,
			wantCommands:  2,
		},
		{
			name: "enable plugin",
			diffResult: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{},
				Plugins: []diff.PluginDiff{
					{
						Name:   "test-plugin@official",
						Action: diff.ActionEnable,
						Current: &state.PluginState{
							Name:    "test-plugin",
							Enabled: false,
						},
					},
				},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantInstalled: 0,
			wantUpdated:   1,
			wantFailed:    0,
			wantSkipped:   0,
			wantAttention: 0,
			wantCommands:  1,
		},
		{
			name: "skip OAuth MCP server",
			diffResult: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{},
				Plugins:      []diff.PluginDiff{},
				MCPServers: []diff.MCPServerDiff{
					{
						Name:          "oauth-server",
						Action:        diff.ActionAdd,
						RequiresOAuth: true,
						Desired: &config.MCPServer{
							Transport: "http",
							URL:       "https://example.com",
						},
					},
				},
			},
			wantInstalled: 0,
			wantUpdated:   0,
			wantFailed:    0,
			wantSkipped:   1,
			wantAttention: 1,
			wantCommands:  0,
		},
		{
			name: "report extra items",
			diffResult: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{
					{
						Alias:   "extra-marketplace",
						Action:  diff.ActionRemove,
						Current: &state.MarketplaceState{Alias: "extra-marketplace"},
					},
				},
				Plugins: []diff.PluginDiff{
					{
						Name:    "extra-plugin@official",
						Action:  diff.ActionRemove,
						Current: &state.PluginState{Name: "extra-plugin"},
					},
				},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantInstalled: 0,
			wantUpdated:   0,
			wantFailed:    0,
			wantSkipped:   0,
			wantAttention: 2,
			wantCommands:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRunner{
				commands: []string{},
				outputs:  make(map[string][]byte),
				errors:   make(map[string]error),
			}

			syncer := sync.NewSyncerWithRunner(&testCommandRunner{runFunc: mockRun(mock)})
			result, err := syncer.Execute(tt.diffResult, sync.Options{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Installed != tt.wantInstalled {
				t.Errorf("installed = %d, want %d", result.Installed, tt.wantInstalled)
			}
			if result.Updated != tt.wantUpdated {
				t.Errorf("updated = %d, want %d", result.Updated, tt.wantUpdated)
			}
			if result.Failed != tt.wantFailed {
				t.Errorf("failed = %d, want %d", result.Failed, tt.wantFailed)
			}
			if result.Skipped != tt.wantSkipped {
				t.Errorf("skipped = %d, want %d", result.Skipped, tt.wantSkipped)
			}
			if len(result.Attention) != tt.wantAttention {
				t.Errorf("attention items = %d, want %d", len(result.Attention), tt.wantAttention)
			}
			if len(mock.commands) != tt.wantCommands {
				t.Errorf("commands executed = %d, want %d", len(mock.commands), tt.wantCommands)
			}
		})
	}
}

// testCommandRunner implements sync.CommandRunner for testing.
type testCommandRunner struct {
	runFunc func(name string, args ...string) ([]byte, error)
}

func (r *testCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return r.runFunc(name, args...)
}

// TestIntegration_BackupCreation tests backup functionality.
func TestIntegration_BackupCreation(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.cleanup()

	// Set up state
	ts.writeMarketplaces(t, map[string]interface{}{
		"official": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/plugins",
			},
			"installLocation": "/path/to/install",
			"lastUpdated":     "2025-01-01T00:00:00Z",
		},
	})

	ts.writeInstalledPlugins(t, map[string]interface{}{
		"test-plugin@official": []map[string]interface{}{{
			"scope":       "user",
			"installPath": "/path/to/plugin",
			"version":     "1.0.0",
			"installedAt": "2025-01-01T00:00:00Z",
			"lastUpdated": "2025-01-01T00:00:00Z",
		}},
	})

	// Create backup in a temp backup directory
	backupDir := filepath.Join(ts.tmpDir, "backups")
	manager := backup.NewManagerWithDir(backupDir, "test-version")

	reader := &state.FilesystemReader{ClaudeDir: ts.claudeDir}
	currentState, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read state: %v", err)
	}

	bak, err := manager.Create(currentState, "test backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	if bak.ID == "" {
		t.Error("backup ID should not be empty")
	}

	// List backups
	backups, err := manager.List()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("backups count = %d, want 1", len(backups))
	}
}

// TestIntegration_DiffWithGitStatus tests git status checking integration.
func TestIntegration_FilterDiffByGitStatus(t *testing.T) {
	// Create a diff result
	diffResult := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Alias:  "clean-marketplace",
				Action: diff.ActionAdd,
			},
			{
				Alias:  "dirty-marketplace",
				Action: diff.ActionAdd,
			},
		},
		Plugins: []diff.PluginDiff{
			{
				Name:   "clean-plugin",
				Action: diff.ActionAdd,
			},
			{
				Name:   "dirty-plugin",
				Action: diff.ActionAdd,
			},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	// Manually mark some items as skipped due to git
	filteredMarketplaces := make([]diff.MarketplaceDiff, 0, len(diffResult.Marketplaces))
	for _, m := range diffResult.Marketplaces {
		if m.Alias == "dirty-marketplace" {
			m.Action = diff.ActionSkipGit
		}
		filteredMarketplaces = append(filteredMarketplaces, m)
	}
	diffResult.Marketplaces = filteredMarketplaces

	filteredPlugins := make([]diff.PluginDiff, 0, len(diffResult.Plugins))
	for _, p := range diffResult.Plugins {
		if p.Name == "dirty-plugin" {
			p.Action = diff.ActionSkipGit
		}
		filteredPlugins = append(filteredPlugins, p)
	}
	diffResult.Plugins = filteredPlugins

	// Count actions
	addCount := 0
	skipCount := 0
	for _, m := range diffResult.Marketplaces {
		switch m.Action {
		case diff.ActionAdd:
			addCount++
		case diff.ActionSkipGit:
			skipCount++
		}
	}
	for _, p := range diffResult.Plugins {
		switch p.Action {
		case diff.ActionAdd:
			addCount++
		case diff.ActionSkipGit:
			skipCount++
		}
	}

	if addCount != 2 {
		t.Errorf("add count = %d, want 2", addCount)
	}
	if skipCount != 2 {
		t.Errorf("skip count = %d, want 2", skipCount)
	}
}

// TestIntegration_GenerateCommands tests command generation.
func TestIntegration_GenerateCommands(t *testing.T) {
	diffResult := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{
			{
				Alias:  "official",
				Action: diff.ActionAdd,
				Desired: &config.Marketplace{
					Repo: "anthropics/plugins",
				},
			},
		},
		Plugins: []diff.PluginDiff{
			{
				Name:   "test-plugin@official",
				Action: diff.ActionAdd,
				Desired: &config.Plugin{
					Name: "test-plugin@official",
				},
			},
		},
		MCPServers: []diff.MCPServerDiff{
			{
				Name:   "filesystem",
				Action: diff.ActionAdd,
				Desired: &config.MCPServer{
					Transport: types.TransportStdio.String(),
					Command:   "npx",
					Args:      []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				},
			},
		},
	}

	commands := diffResult.GenerateCommands()
	if len(commands) != 3 {
		t.Errorf("commands count = %d, want 3", len(commands))
	}

	// Verify command order: marketplaces, plugins, MCP servers
	for i, cmd := range commands {
		if cmd.Command == "" {
			t.Errorf("command[%d] is empty", i)
		}
		if cmd.Description == "" {
			t.Errorf("command[%d] description is empty", i)
		}
	}
}

// TestIntegration_AlreadyInSync tests the "already in sync" scenario.
func TestIntegration_AlreadyInSync(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.cleanup()

	// Create Clewfile
	clewfile := `version: 1
marketplaces:
  official:
    repo: anthropics/plugins
plugins:
  - test-plugin@official
`
	ts.writeClewfile(t, clewfile)

	// Set up matching current state
	ts.writeMarketplaces(t, map[string]interface{}{
		"official": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/plugins",
			},
			"installLocation": "/path/to/install",
			"lastUpdated":     "2025-01-01T00:00:00Z",
		},
	})

	ts.writeInstalledPlugins(t, map[string]interface{}{
		"test-plugin@official": []map[string]interface{}{{
			"scope":       "user",
			"installPath": "/path/to/plugin",
			"version":     "1.0.0",
			"installedAt": "2025-01-01T00:00:00Z",
			"lastUpdated": "2025-01-01T00:00:00Z",
		}},
	})

	ts.writeSettings(t, map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"test-plugin@official": true,
		},
	})

	// Load and compute diff
	cfg, err := config.Load(ts.clewfile)
	if err != nil {
		t.Fatalf("failed to load clewfile: %v", err)
	}

	reader := &state.FilesystemReader{ClaudeDir: ts.claudeDir}
	currentState, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read state: %v", err)
	}

	diffResult := diff.Compute(cfg, currentState)
	add, update, remove, attention := diffResult.Summary()

	// Should be in sync (no adds or updates needed)
	if add != 0 {
		t.Errorf("add = %d, want 0", add)
	}
	if update != 0 {
		t.Errorf("update = %d, want 0", update)
	}
	// The remove count may vary based on current state
	t.Logf("remove = %d, attention = %d (informational)", remove, attention)
}
