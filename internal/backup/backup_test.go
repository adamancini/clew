package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adamancini/clew/internal/state"
)

func TestManager_Create(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"official": {
				Name:   "official",
				Source: "github",
				Repo:   "anthropic/claude-plugins",
			},
		},
		Plugins: map[string]state.PluginState{
			"test-plugin@official": {
				Name:        "test-plugin",
				Marketplace: "official",
				Scope:       "user",
				Enabled:     true,
			},
		},
		MCPServers: map[string]state.MCPServerState{
			"filesystem": {
				Name:      "filesystem",
				Transport: "stdio",
				Command:   "npx",
				Args:      []string{"-y", "@anthropic/mcp-server-filesystem"},
				Scope:     "user",
			},
		},
	}

	// Test creating backup without note
	bak, err := manager.Create(currentState, "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if bak.ID == "" {
		t.Error("Create() backup ID is empty")
	}
	if bak.ClewVersion != "v1.0.0" {
		t.Errorf("Create() ClewVersion = %v, want v1.0.0", bak.ClewVersion)
	}
	if bak.Note != "" {
		t.Errorf("Create() Note = %v, want empty", bak.Note)
	}

	// Verify state was captured
	if len(bak.State.Marketplaces) != 1 {
		t.Errorf("Create() Marketplaces count = %v, want 1", len(bak.State.Marketplaces))
	}
	if len(bak.State.Plugins) != 1 {
		t.Errorf("Create() Plugins count = %v, want 1", len(bak.State.Plugins))
	}
	if len(bak.State.MCPServers) != 1 {
		t.Errorf("Create() MCPServers count = %v, want 1", len(bak.State.MCPServers))
	}

	// Verify file was created
	filename := bak.ID + ".json"
	path := filepath.Join(tmpDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Create() backup file not created at %s", path)
	}
}

func TestManager_CreateWithNote(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: make(map[string]state.MarketplaceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	bak, err := manager.Create(currentState, "Before major update")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if bak.Note != "Before major update" {
		t.Errorf("Create() Note = %v, want 'Before major update'", bak.Note)
	}
}

func TestManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: make(map[string]state.MarketplaceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	// Create multiple backups
	_, err := manager.Create(currentState, "First")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait a second so we get different timestamps
	time.Sleep(time.Second)

	_, err = manager.Create(currentState, "Second")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// List backups
	backups, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("List() count = %v, want 2", len(backups))
	}

	// Verify order (newest first)
	if len(backups) >= 2 {
		if backups[0].Note != "Second" {
			t.Errorf("List() first backup Note = %v, want 'Second' (newest)", backups[0].Note)
		}
		if backups[1].Note != "First" {
			t.Errorf("List() second backup Note = %v, want 'First' (oldest)", backups[1].Note)
		}
	}
}

func TestManager_ListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	backups, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("List() count = %v, want 0", len(backups))
	}
}

func TestManager_Get(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: make(map[string]state.MarketplaceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	created, err := manager.Create(currentState, "Test backup")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get by ID
	bak, err := manager.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if bak.ID != created.ID {
		t.Errorf("Get() ID = %v, want %v", bak.ID, created.ID)
	}
	if bak.Note != "Test backup" {
		t.Errorf("Get() Note = %v, want 'Test backup'", bak.Note)
	}
}

func TestManager_GetLatest(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: make(map[string]state.MarketplaceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	_, err := manager.Create(currentState, "First")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	time.Sleep(time.Second)

	_, err = manager.Create(currentState, "Latest")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get latest
	bak, err := manager.Get("latest")
	if err != nil {
		t.Fatalf("Get('latest') error = %v", err)
	}

	if bak.Note != "Latest" {
		t.Errorf("Get('latest') Note = %v, want 'Latest'", bak.Note)
	}
}

func TestManager_GetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	_, err := manager.Get("nonexistent")
	if err == nil {
		t.Error("Get() expected error for nonexistent backup")
	}
}

func TestManager_GetLatestNoBackups(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	_, err := manager.Get("latest")
	if err == nil {
		t.Error("Get('latest') expected error when no backups exist")
	}
}

func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Marketplaces: make(map[string]state.MarketplaceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	bak, err := manager.Create(currentState, "To delete")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete
	err = manager.Delete(bak.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = manager.Get(bak.ID)
	if err == nil {
		t.Error("Get() expected error after delete")
	}
}

func TestManager_DeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	err := manager.Delete("nonexistent")
	if err == nil {
		t.Error("Delete() expected error for nonexistent backup")
	}
}

func TestBackup_ToState(t *testing.T) {
	bak := &Backup{
		ID: "test",
		State: BackupState{
			Marketplaces: map[string]state.MarketplaceState{
				"test": {Name: "test"},
			},
			Plugins: map[string]state.PluginState{
				"plugin": {Name: "plugin"},
			},
			MCPServers: map[string]state.MCPServerState{
				"server": {Name: "server"},
			},
		},
	}

	s := bak.ToState()

	if len(s.Marketplaces) != 1 {
		t.Errorf("ToState() Marketplaces count = %v, want 1", len(s.Marketplaces))
	}
	if len(s.Plugins) != 1 {
		t.Errorf("ToState() Plugins count = %v, want 1", len(s.Plugins))
	}
	if len(s.MCPServers) != 1 {
		t.Errorf("ToState() MCPServers count = %v, want 1", len(s.MCPServers))
	}
}

func TestManager_BackupDir(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	if manager.BackupDir() != tmpDir {
		t.Errorf("BackupDir() = %v, want %v", manager.BackupDir(), tmpDir)
	}
}
