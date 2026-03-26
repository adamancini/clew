package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adamancini/clew/internal/state"
)

// captureStderr captures stderr output during function execution.
func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stderr = w

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	os.Stderr = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	return buf.String()
}

// setupMarketplaceDir creates a temporary directory structure that mimics
// the Claude Code marketplace layout:
//
//	<tmpdir>/
//	  <marketplace>/
//	    plugins/
//	      <plugin>/
//	        (empty directory)
func setupMarketplaceDir(t *testing.T, marketplaces map[string][]string) string {
	t.Helper()

	tmpDir := t.TempDir()
	for marketplace, plugins := range marketplaces {
		for _, plugin := range plugins {
			dir := filepath.Join(tmpDir, marketplace, "plugins", plugin)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("failed to create marketplace plugin dir %s: %v", dir, err)
			}
		}
	}
	return tmpDir
}

func TestConvertStateToClewfile_PluginExistsInMarketplace(t *testing.T) {
	// Set up a marketplace directory with one plugin present
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{
		"test-marketplace": {"good-plugin"},
	})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"test-marketplace": {
				Alias: "test-marketplace",
				Repo:  "owner/test-marketplace",
			},
		},
		Plugins: map[string]state.PluginState{
			"good-plugin@test-marketplace": {
				Name:        "good-plugin",
				Marketplace: "test-marketplace",
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Plugin should be included
		if len(exported.Plugins) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(exported.Plugins))
		}
		if exported.Plugins[0].Name != "good-plugin@test-marketplace" {
			t.Errorf("expected plugin name 'good-plugin@test-marketplace', got %q", exported.Plugins[0].Name)
		}
	})

	// No orphan warnings expected
	if strings.Contains(stderr, "not found in marketplace directory") {
		t.Errorf("unexpected orphan warning in stderr: %s", stderr)
	}
}

func TestConvertStateToClewfile_OrphanedPluginSkipped(t *testing.T) {
	// Set up a marketplace directory WITHOUT the orphaned plugin
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{
		"test-marketplace": {"other-plugin"},
	})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"test-marketplace": {
				Alias: "test-marketplace",
				Repo:  "owner/test-marketplace",
			},
		},
		Plugins: map[string]state.PluginState{
			"orphaned-plugin@test-marketplace": {
				Name:        "orphaned-plugin",
				Marketplace: "test-marketplace",
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Plugin should NOT be included
		if len(exported.Plugins) != 0 {
			t.Fatalf("expected 0 plugins, got %d: %+v", len(exported.Plugins), exported.Plugins)
		}
	})

	// Should see orphan warning
	if !strings.Contains(stderr, "not found in marketplace directory") {
		t.Errorf("expected orphan warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "orphaned-plugin@test-marketplace") {
		t.Errorf("expected orphaned plugin name in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Skipped 1 plugin(s)") {
		t.Errorf("expected skip count of 1 in stderr, got: %s", stderr)
	}
}

func TestConvertStateToClewfile_MixedPlugins(t *testing.T) {
	// Set up marketplace with only some plugins present
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{
		"marketplace-a": {"existing-plugin"},
		"marketplace-b": {"another-good-plugin"},
	})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"marketplace-a": {
				Alias: "marketplace-a",
				Repo:  "owner/marketplace-a",
			},
			"marketplace-b": {
				Alias: "marketplace-b",
				Repo:  "owner/marketplace-b",
			},
		},
		Plugins: map[string]state.PluginState{
			"existing-plugin@marketplace-a": {
				Name:        "existing-plugin",
				Marketplace: "marketplace-a",
				Scope:       "user",
				Enabled:     true,
			},
			"gone-plugin@marketplace-a": {
				Name:        "gone-plugin",
				Marketplace: "marketplace-a",
				Scope:       "user",
				Enabled:     true,
			},
			"another-good-plugin@marketplace-b": {
				Name:        "another-good-plugin",
				Marketplace: "marketplace-b",
				Scope:       "user",
				Enabled:     true,
			},
			"removed-plugin@marketplace-b": {
				Name:        "removed-plugin",
				Marketplace: "marketplace-b",
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Only the 2 existing plugins should be included
		if len(exported.Plugins) != 2 {
			t.Fatalf("expected 2 plugins, got %d: %+v", len(exported.Plugins), exported.Plugins)
		}

		// Verify the correct plugins are included (sorted by marketplace then name)
		pluginNames := make([]string, len(exported.Plugins))
		for i, p := range exported.Plugins {
			pluginNames[i] = p.Name
		}

		expectedNames := []string{
			"existing-plugin@marketplace-a",
			"another-good-plugin@marketplace-b",
		}
		for _, expected := range expectedNames {
			found := false
			for _, name := range pluginNames {
				if name == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected plugin %q in exported list, got: %v", expected, pluginNames)
			}
		}
	})

	// Should see orphan warnings for both missing plugins
	if !strings.Contains(stderr, "not found in marketplace directory") {
		t.Errorf("expected orphan warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Skipped 2 plugin(s)") {
		t.Errorf("expected skip count of 2 in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "gone-plugin@marketplace-a") {
		t.Errorf("expected 'gone-plugin@marketplace-a' in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "removed-plugin@marketplace-b") {
		t.Errorf("expected 'removed-plugin@marketplace-b' in stderr, got: %s", stderr)
	}
}

func TestConvertStateToClewfile_LocalMarketplaceSkipStillWorks(t *testing.T) {
	// Ensure the existing local marketplace skip logic still works
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"local-marketplace": {
				Alias: "local-marketplace",
				Repo:  "", // Local marketplace has no repo
			},
			"remote-marketplace": {
				Alias: "remote-marketplace",
				Repo:  "owner/remote-marketplace",
			},
		},
		Plugins: map[string]state.PluginState{
			"local-plugin@local-marketplace": {
				Name:        "local-plugin",
				Marketplace: "local-marketplace",
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Local marketplace should not be exported
		if _, ok := exported.Marketplaces["local-marketplace"]; ok {
			t.Error("local marketplace should not be exported")
		}

		// Remote marketplace should be exported
		if _, ok := exported.Marketplaces["remote-marketplace"]; !ok {
			t.Error("remote marketplace should be exported")
		}

		// Plugin referencing local marketplace should be skipped
		if len(exported.Plugins) != 0 {
			t.Fatalf("expected 0 plugins (local plugin should be skipped), got %d", len(exported.Plugins))
		}
	})

	// Should see local marketplace skip message
	if !strings.Contains(stderr, "local marketplace") {
		t.Errorf("expected local marketplace warning in stderr, got: %s", stderr)
	}
	// Should see non-marketplace plugin skip message
	if !strings.Contains(stderr, "referencing non-marketplace sources") {
		t.Errorf("expected non-marketplace sources warning in stderr, got: %s", stderr)
	}
}

func TestConvertStateToClewfile_PluginWithoutMarketplace(t *testing.T) {
	// Plugins without @marketplace suffix should pass through
	// (they don't reference a marketplace, so no filesystem check needed)
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{},
		Plugins: map[string]state.PluginState{
			"standalone-plugin": {
				Name:    "standalone-plugin",
				Scope:   "user",
				Enabled: true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Plugin without marketplace reference should still be exported
		if len(exported.Plugins) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(exported.Plugins))
		}
		if exported.Plugins[0].Name != "standalone-plugin" {
			t.Errorf("expected plugin name 'standalone-plugin', got %q", exported.Plugins[0].Name)
		}
	})

	// No orphan warnings expected
	if strings.Contains(stderr, "not found in marketplace directory") {
		t.Errorf("unexpected orphan warning in stderr: %s", stderr)
	}
}

func TestConvertStateToClewfile_BothSkipTypesReported(t *testing.T) {
	// Test that both skip types (no-marketplace and orphaned) are reported separately
	marketplacesDir := setupMarketplaceDir(t, map[string][]string{
		"valid-marketplace": {"real-plugin"},
	})

	s := &state.State{
		Marketplaces: map[string]state.MarketplaceState{
			"valid-marketplace": {
				Alias: "valid-marketplace",
				Repo:  "owner/valid-marketplace",
			},
			"local-only": {
				Alias: "local-only",
				Repo:  "", // Local marketplace
			},
		},
		Plugins: map[string]state.PluginState{
			"real-plugin@valid-marketplace": {
				Name:        "real-plugin",
				Marketplace: "valid-marketplace",
				Scope:       "user",
				Enabled:     true,
			},
			"ghost-plugin@valid-marketplace": {
				Name:        "ghost-plugin",
				Marketplace: "valid-marketplace",
				Scope:       "user",
				Enabled:     true,
			},
			"local-plugin@local-only": {
				Name:        "local-plugin",
				Marketplace: "local-only",
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	stderr := captureStderr(t, func() {
		exported := convertStateToClewfile(s, marketplacesDir)

		// Only real-plugin should survive
		if len(exported.Plugins) != 1 {
			t.Fatalf("expected 1 plugin, got %d: %+v", len(exported.Plugins), exported.Plugins)
		}
		if exported.Plugins[0].Name != "real-plugin@valid-marketplace" {
			t.Errorf("expected 'real-plugin@valid-marketplace', got %q", exported.Plugins[0].Name)
		}
	})

	// Should have separate messages for each skip type
	if !strings.Contains(stderr, "referencing non-marketplace sources") {
		t.Errorf("expected non-marketplace skip message, got: %s", stderr)
	}
	if !strings.Contains(stderr, "not found in marketplace directory") {
		t.Errorf("expected orphan skip message, got: %s", stderr)
	}
	if !strings.Contains(stderr, "ghost-plugin@valid-marketplace") {
		t.Errorf("expected ghost-plugin in orphan message, got: %s", stderr)
	}
	if !strings.Contains(stderr, "local-plugin@local-only") {
		t.Errorf("expected local-plugin in non-marketplace message, got: %s", stderr)
	}
}
