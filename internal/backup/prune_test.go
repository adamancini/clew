package backup

import (
	"testing"
	"time"

	"github.com/adamancini/clew/internal/state"
)

func TestManager_Prune(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Sources: make(map[string]state.SourceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	// Create 5 backups
	for i := 0; i < 5; i++ {
		_, err := manager.Create(currentState, "")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Second)
	}

	// Prune to keep only 2
	result, err := manager.Prune(2)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if result.Kept != 2 {
		t.Errorf("Prune() Kept = %v, want 2", result.Kept)
	}
	if len(result.Deleted) != 3 {
		t.Errorf("Prune() Deleted count = %v, want 3", len(result.Deleted))
	}

	// Verify remaining backups
	backups, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(backups) != 2 {
		t.Errorf("List() after prune = %v, want 2", len(backups))
	}
}

func TestManager_PruneNoOp(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Sources: make(map[string]state.SourceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	// Create 2 backups
	for i := 0; i < 2; i++ {
		_, err := manager.Create(currentState, "")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Second)
	}

	// Prune with keep=5 (more than we have)
	result, err := manager.Prune(5)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if result.Kept != 2 {
		t.Errorf("Prune() Kept = %v, want 2", result.Kept)
	}
	if len(result.Deleted) != 0 {
		t.Errorf("Prune() Deleted count = %v, want 0", len(result.Deleted))
	}
}

func TestManager_PruneKeepZero(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	currentState := &state.State{
		Sources: make(map[string]state.SourceState),
		Plugins:      make(map[string]state.PluginState),
		MCPServers:   make(map[string]state.MCPServerState),
	}

	// Create 3 backups
	for i := 0; i < 3; i++ {
		_, err := manager.Create(currentState, "")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Second)
	}

	// Prune with keep=0 (delete all)
	result, err := manager.Prune(0)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if result.Kept != 0 {
		t.Errorf("Prune() Kept = %v, want 0", result.Kept)
	}
	if len(result.Deleted) != 3 {
		t.Errorf("Prune() Deleted count = %v, want 3", len(result.Deleted))
	}

	// Verify no backups remain
	backups, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("List() after prune all = %v, want 0", len(backups))
	}
}

func TestManager_PruneNegativeKeep(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	_, err := manager.Prune(-1)
	if err == nil {
		t.Error("Prune(-1) expected error for negative keep count")
	}
}

func TestManager_PruneEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManagerWithDir(tmpDir, "v1.0.0")

	result, err := manager.Prune(5)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if result.Kept != 0 {
		t.Errorf("Prune() Kept = %v, want 0", result.Kept)
	}
	if len(result.Deleted) != 0 {
		t.Errorf("Prune() Deleted count = %v, want 0", len(result.Deleted))
	}
}

func TestDefaultKeepCount(t *testing.T) {
	if DefaultKeepCount != 30 {
		t.Errorf("DefaultKeepCount = %v, want 30", DefaultKeepCount)
	}
}
