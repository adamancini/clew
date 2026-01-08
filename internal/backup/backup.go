// Package backup handles backup and restore operations for Claude Code configuration.
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/adamancini/clew/internal/state"
)

// Backup represents a single backup snapshot.
type Backup struct {
	ID          string       `json:"id"`
	CreatedAt   time.Time    `json:"created_at"`
	Note        string       `json:"note,omitempty"`
	ClewVersion string       `json:"clew_version"`
	State       BackupState  `json:"state"`
}

// BackupState contains the configuration state at backup time.
type BackupState struct {
	Marketplaces map[string]state.MarketplaceState `json:"marketplaces"`
	Plugins      map[string]state.PluginState      `json:"plugins"`
	MCPServers   map[string]state.MCPServerState   `json:"mcp_servers"`
}

// BackupInfo provides summary information about a backup for listing.
type BackupInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Note      string    `json:"note,omitempty"`
	Size      int64     `json:"size"`
}

// Manager handles backup operations.
type Manager struct {
	backupDir   string
	clewVersion string
}

// NewManager creates a new backup manager.
func NewManager(version string) (*Manager, error) {
	backupDir, err := getBackupDir()
	if err != nil {
		return nil, err
	}
	return &Manager{
		backupDir:   backupDir,
		clewVersion: version,
	}, nil
}

// NewManagerWithDir creates a backup manager with a custom directory (for testing).
func NewManagerWithDir(backupDir, version string) *Manager {
	return &Manager{
		backupDir:   backupDir,
		clewVersion: version,
	}
}

// getBackupDir returns the default backup directory path.
func getBackupDir() (string, error) {
	// Use XDG_CACHE_HOME or default to ~/.cache
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "clew", "backups"), nil
}

// Create creates a new backup from the current state.
func (m *Manager) Create(currentState *state.State, note string) (*Backup, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup ID and timestamp
	now := time.Now()
	id := now.Format("2006-01-02-150405")

	backup := &Backup{
		ID:          id,
		CreatedAt:   now,
		Note:        note,
		ClewVersion: m.clewVersion,
		State: BackupState{
			Marketplaces: currentState.Marketplaces,
			Plugins:      currentState.Plugins,
			MCPServers:   currentState.MCPServers,
		},
	}

	// Write backup to file
	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(m.backupDir, filename)

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	return backup, nil
}

// List returns all backups sorted by creation time (newest first).
func (m *Manager) List() ([]BackupInfo, error) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(m.backupDir, entry.Name())
		backup, err := m.loadBackup(path)
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			ID:        backup.ID,
			CreatedAt: backup.CreatedAt,
			Note:      backup.Note,
			Size:      info.Size(),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Get retrieves a backup by ID. Use "latest" to get the most recent backup.
func (m *Manager) Get(id string) (*Backup, error) {
	if id == "latest" {
		backups, err := m.List()
		if err != nil {
			return nil, err
		}
		if len(backups) == 0 {
			return nil, fmt.Errorf("no backups found")
		}
		id = backups[0].ID
	}

	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(m.backupDir, filename)

	return m.loadBackup(path)
}

// Delete removes a backup by ID.
func (m *Manager) Delete(id string) error {
	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(m.backupDir, filename)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", id)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// loadBackup reads and parses a backup file.
func (m *Manager) loadBackup(path string) (*Backup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("backup not found: %s", filepath.Base(path))
		}
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup file: %w", err)
	}

	return &backup, nil
}

// ToState converts a backup's state to a state.State for comparison.
func (b *Backup) ToState() *state.State {
	return &state.State{
		Marketplaces: b.State.Marketplaces,
		Plugins:      b.State.Plugins,
		MCPServers:   b.State.MCPServers,
	}
}

// BackupDir returns the backup directory path.
func (m *Manager) BackupDir() string {
	return m.backupDir
}
