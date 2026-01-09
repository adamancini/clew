package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemReaderMarketplaces(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write test marketplaces
	marketplacesJSON := `{
  "repositories": {
    "test-marketplace": {
      "name": "test-marketplace",
      "source": "github",
      "repo": "owner/test-marketplace",
      "installLocation": "/path/to/marketplace",
      "lastUpdated": "2025-01-01T00:00:00Z"
    },
    "local-marketplace": {
      "name": "local-marketplace",
      "source": "local",
      "path": "/local/path",
      "installLocation": "/local/path",
      "lastUpdated": "2025-01-01T00:00:00Z"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), []byte(marketplacesJSON), 0644); err != nil {
		t.Fatal(err)
	}

	reader := &FilesystemReader{ClaudeDir: claudeDir}
	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if len(state.Marketplaces) != 2 {
		t.Errorf("Marketplaces count = %d, want 2", len(state.Marketplaces))
	}

	if m, ok := state.Marketplaces["test-marketplace"]; ok {
		if m.Source != "github" {
			t.Errorf("Marketplace source = %s, want github", m.Source)
		}
		if m.Repo != "owner/test-marketplace" {
			t.Errorf("Marketplace repo = %s, want owner/test-marketplace", m.Repo)
		}
	} else {
		t.Error("Missing test-marketplace")
	}

	if m, ok := state.Marketplaces["local-marketplace"]; ok {
		if m.Source != "local" {
			t.Errorf("Marketplace source = %s, want local", m.Source)
		}
		if m.Path != "/local/path" {
			t.Errorf("Marketplace path = %s, want /local/path", m.Path)
		}
	} else {
		t.Error("Missing local-marketplace")
	}
}

func TestFilesystemReaderPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write test plugins
	pluginsJSON := `{
  "version": 2,
  "plugins": {
    "test-plugin@test-marketplace": [
      {
        "scope": "user",
        "installPath": "/path/to/plugin",
        "version": "1.0.0",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z"
      }
    ],
    "another-plugin@test-marketplace": [
      {
        "scope": "project",
        "projectPath": "/project",
        "installPath": "/path/to/another",
        "version": "2.0.0",
        "installedAt": "2025-01-01T00:00:00Z",
        "lastUpdated": "2025-01-01T00:00:00Z"
      }
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(pluginsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Write settings for enabled state
	settingsJSON := `{
  "enabledPlugins": {
    "test-plugin@test-marketplace": true,
    "another-plugin@test-marketplace": false
  }
}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	reader := &FilesystemReader{ClaudeDir: claudeDir}
	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if len(state.Plugins) != 2 {
		t.Errorf("Plugins count = %d, want 2", len(state.Plugins))
	}

	if p, ok := state.Plugins["test-plugin@test-marketplace"]; ok {
		if p.Name != "test-plugin" {
			t.Errorf("Plugin name = %s, want test-plugin", p.Name)
		}
		if p.Marketplace != "test-marketplace" {
			t.Errorf("Plugin marketplace = %s, want test-marketplace", p.Marketplace)
		}
		if p.Scope != "user" {
			t.Errorf("Plugin scope = %s, want user", p.Scope)
		}
		if !p.Enabled {
			t.Error("Plugin should be enabled")
		}
		if p.Version != "1.0.0" {
			t.Errorf("Plugin version = %s, want 1.0.0", p.Version)
		}
	} else {
		t.Error("Missing test-plugin@test-marketplace")
	}

	if p, ok := state.Plugins["another-plugin@test-marketplace"]; ok {
		if p.Enabled {
			t.Error("Plugin should be disabled")
		}
	}
}

func TestFilesystemReaderMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	reader := &FilesystemReader{ClaudeDir: claudeDir}
	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v, should handle missing files gracefully", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}

	if len(state.Marketplaces) != 0 {
		t.Errorf("Marketplaces should be empty, got %d", len(state.Marketplaces))
	}
	if len(state.Plugins) != 0 {
		t.Errorf("Plugins should be empty, got %d", len(state.Plugins))
	}
	if len(state.MCPServers) != 0 {
		t.Errorf("MCPServers should be empty, got %d", len(state.MCPServers))
	}
}

func TestFilesystemReaderMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write malformed JSON
	if err := os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := &FilesystemReader{ClaudeDir: claudeDir}
	// Should not error fatally, just warn and continue
	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error = %v, should handle malformed JSON gracefully", err)
	}

	// Should still return empty state
	if state == nil {
		t.Fatal("State should not be nil")
	}
}
