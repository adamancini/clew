package cmd

import (
	"testing"

	"github.com/adamancini/clew/internal/config"
	"github.com/adamancini/clew/internal/diff"
	"github.com/adamancini/clew/internal/state"
)

// TestSyncServiceIsInSync tests the IsInSync method.
func TestSyncServiceIsInSync(t *testing.T) {
	tests := []struct {
		name   string
		diff   *diff.Result
		wantIn bool
	}{
		{
			name: "in sync - no actions needed",
			diff: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{
					{Alias: "official", Action: diff.ActionNone},
				},
				Plugins:    []diff.PluginDiff{},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantIn: true,
		},
		{
			name: "not in sync - add needed",
			diff: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{},
				Plugins: []diff.PluginDiff{
					{Name: "new-plugin", Action: diff.ActionAdd},
				},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantIn: false,
		},
		{
			name: "not in sync - update needed",
			diff: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{},
				Plugins: []diff.PluginDiff{
					{Name: "plugin", Action: diff.ActionEnable},
				},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantIn: false,
		},
		{
			name: "not in sync - attention needed",
			diff: &diff.Result{
				Marketplaces: []diff.MarketplaceDiff{
					{Alias: "extra", Action: diff.ActionRemove},
				},
				Plugins:    []diff.PluginDiff{},
				MCPServers: []diff.MCPServerDiff{},
			},
			wantIn: false,
		},
	}

	service := &SyncService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsInSync(tt.diff)
			if got != tt.wantIn {
				t.Errorf("IsInSync() = %v, want %v", got, tt.wantIn)
			}
		})
	}
}

// TestSyncServiceComputeDiff tests the ComputeDiff method.
func TestSyncServiceComputeDiff(t *testing.T) {
	service := &SyncService{}

	clewfile := &config.Clewfile{
		Plugins: []config.Plugin{
			{Name: "new-plugin@official"},
		},
	}

	current := &state.State{
		Marketplaces: map[string]state.MarketplaceState{},
		Plugins:      map[string]state.PluginState{},
		MCPServers:   map[string]state.MCPServerState{},
	}

	result := service.ComputeDiff(clewfile, current)

	if result == nil {
		t.Fatal("ComputeDiff() returned nil")
	}

	if len(result.Plugins) != 1 {
		t.Errorf("expected 1 plugin diff, got %d", len(result.Plugins))
	}

	if result.Plugins[0].Action != diff.ActionAdd {
		t.Errorf("expected ActionAdd, got %s", result.Plugins[0].Action)
	}
}

// TestSyncServiceGenerateCommands tests the GenerateCommands method.
func TestSyncServiceGenerateCommands(t *testing.T) {
	service := &SyncService{}

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
				Name:   "test@official",
				Action: diff.ActionAdd,
				Desired: &config.Plugin{
					Name: "test@official",
				},
			},
		},
		MCPServers: []diff.MCPServerDiff{},
	}

	commands := service.GenerateCommands(diffResult)

	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}
}

// TestSyncServiceFilterDiffByGitStatus tests git filtering.
func TestSyncServiceFilterDiffByGitStatus(t *testing.T) {
	service := &SyncService{}

	// Test with nil git result
	diffResult := &diff.Result{
		Marketplaces: []diff.MarketplaceDiff{{Alias: "marketplace"}},
		Plugins:      []diff.PluginDiff{{Name: "plugin"}},
		MCPServers:   []diff.MCPServerDiff{},
	}

	filtered := service.FilterDiffByGitStatus(diffResult, nil)

	// Should return the same diff when git result is nil
	if filtered != diffResult {
		t.Error("FilterDiffByGitStatus should return original diff when git result is nil")
	}
}

// TestSyncOptions tests the SyncOptions struct.
func TestSyncOptions(t *testing.T) {
	opts := SyncOptions{
		Strict:       true,
		Interactive:  true,
		CreateBackup: true,
		Short:        false,
		ShowCommands: false,
		SkipGitCheck: false,
		OutputFormat: "json",
		Verbose:      true,
		Quiet:        false,
	}

	if !opts.Strict {
		t.Error("Strict should be true")
	}
	if !opts.Interactive {
		t.Error("Interactive should be true")
	}
	if !opts.CreateBackup {
		t.Error("CreateBackup should be true")
	}
	if opts.OutputFormat != "json" {
		t.Errorf("OutputFormat = %s, want json", opts.OutputFormat)
	}
}

// TestNewSyncService tests service creation.
func TestNewSyncService(t *testing.T) {
	service := NewSyncService("test-config", "1.0.0")

	if service == nil {
		t.Fatal("NewSyncService returned nil")
	}
	if service.configPath != "test-config" {
		t.Errorf("configPath = %s, want test-config", service.configPath)
	}
	if service.version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", service.version)
	}
	if service.stateReader == nil {
		t.Error("stateReader should not be nil")
	}
	if service.syncer == nil {
		t.Error("syncer should not be nil")
	}
	if service.gitChecker == nil {
		t.Error("gitChecker should not be nil")
	}
}
