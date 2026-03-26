package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCapstoneRemovedCommands verifies that removed commands and flags produce
// appropriate errors, confirming they are no longer part of the CLI surface.
func TestCapstoneRemovedCommands(t *testing.T) {
	t.Run("removed subcommands produce unknown command errors", func(t *testing.T) {
		removedCommands := []string{"add", "remove", "init"}
		for _, cmd := range removedCommands {
			t.Run(cmd, func(t *testing.T) {
				_, stderr, err := runClew(t, "", cmd)
				if err == nil {
					t.Fatalf("expected error for removed command %q, but it succeeded", cmd)
				}
				if !strings.Contains(stderr, "unknown command") {
					t.Errorf("expected 'unknown command' in stderr for %q, got: %s", cmd, stderr)
				}
			})
		}
	})

	t.Run("removed flags produce unknown flag errors", func(t *testing.T) {
		removedFlags := []struct {
			flag    string
			errText string
		}{
			{"--cli", "unknown flag"},
			{"-f", "unknown shorthand flag"},
			{"--filesystem", "unknown flag"},
			{"--read-from-filesystem", "unknown flag"},
		}
		for _, rf := range removedFlags {
			t.Run(rf.flag, func(t *testing.T) {
				// Use diff as a representative command to test flags
				_, stderr, err := runClew(t, "", "diff", rf.flag)
				if err == nil {
					t.Fatalf("expected error for removed flag %q, but it succeeded", rf.flag)
				}
				if !strings.Contains(stderr, rf.errText) {
					t.Errorf("expected %q in stderr for flag %q, got: %s", rf.errText, rf.flag, stderr)
				}
			})
		}
	})
}

// TestCapstoneScopeRejection verifies that scope: project is rejected with
// a clear error message, since clew 1.0 only supports user scope.
func TestCapstoneScopeRejection(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a Clewfile with scope: project on a plugin entry
	clewfileContent := `version: 1
marketplaces:
  official:
    repo: anthropics/claude-plugins-official
plugins:
  - name: superpowers@official
    scope: project
`
	clewfilePath := filepath.Join(testDir, "scope-project.yaml")
	if err := os.WriteFile(clewfilePath, []byte(clewfileContent), 0644); err != nil {
		t.Fatalf("failed to write Clewfile: %v", err)
	}

	_, stderr, err := runClew(t, testDir, "diff", "--config", clewfilePath)
	if err == nil {
		t.Fatal("expected diff to fail with scope: project, but it succeeded")
	}

	if !strings.Contains(stderr, "clew 1.0 only supports user scope") {
		t.Errorf("expected scope rejection message in stderr, got: %s", stderr)
	}
}

// TestCapstoneHelpText verifies that the help output reflects the 1.0 CLI
// surface: only supported commands are listed and removed ones are absent.
func TestCapstoneHelpText(t *testing.T) {
	stdout, stderr, err := runClew(t, "", "--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}

	t.Run("lists all supported commands", func(t *testing.T) {
		expectedCommands := []string{
			"sync", "diff", "export", "status",
			"backup", "version", "completion",
		}
		for _, cmd := range expectedCommands {
			if !strings.Contains(stdout, cmd) {
				t.Errorf("expected %q in help output, but not found.\nHelp output:\n%s", cmd, stdout)
			}
		}
	})

	t.Run("does not list removed commands", func(t *testing.T) {
		removedCommands := []string{"add", "remove", "init"}
		for _, cmd := range removedCommands {
			// Check for the command as a word boundary to avoid false matches
			// in descriptive text. The help output lists commands indented with
			// their name as the first non-space word on a line.
			lines := strings.Split(stdout, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				// A listed command starts with the command name followed by spaces
				if strings.HasPrefix(trimmed, cmd+" ") || trimmed == cmd {
					t.Errorf("removed command %q should not appear as a listed command in help output.\nLine: %q\nFull help:\n%s", cmd, line, stdout)
				}
			}
		}
	})

	t.Run("does not list removed flags", func(t *testing.T) {
		removedFlags := []string{"--cli", "--filesystem"}
		for _, flag := range removedFlags {
			if strings.Contains(stdout, flag) {
				t.Errorf("removed flag %q should not appear in help output.\nHelp output:\n%s", flag, stdout)
			}
		}
	})
}

// TestCapstoneExportDiffRoundTripOrphanedPlugin verifies that an orphaned plugin
// (listed in installed_plugins.json but whose directory is missing from its
// marketplace) is excluded from export output, and that the exported Clewfile
// round-trips through diff without errors.
func TestCapstoneExportDiffRoundTripOrphanedPlugin(t *testing.T) {
	// Create a dedicated temp directory for this test (not using setupTestEnv
	// because we need a custom installed_plugins.json with an orphaned entry).
	tmpDir, err := os.MkdirTemp("", "clew-e2e-orphan-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pluginsDir := filepath.Join(tmpDir, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	// Write known_marketplaces.json with one marketplace.
	marketplaces := map[string]interface{}{
		"test-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "example/test-marketplace",
			},
			"installLocation": filepath.Join(pluginsDir, "marketplaces", "test-marketplace"),
			"lastUpdated":     "2026-01-01T00:00:00.000Z",
		},
	}
	marketplacesJSON, _ := json.MarshalIndent(marketplaces, "", "  ")
	if err := os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), marketplacesJSON, 0644); err != nil {
		t.Fatalf("failed to write known_marketplaces.json: %v", err)
	}

	// Write installed_plugins.json with two plugins:
	// - "good-plugin@test-marketplace" whose directory WILL exist
	// - "orphaned-plugin@test-marketplace" whose directory will NOT exist
	installedPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"good-plugin@test-marketplace": []map[string]interface{}{
				{
					"scope":        "user",
					"installPath":  filepath.Join(pluginsDir, "cache", "test-marketplace", "good-plugin", "1.0.0"),
					"version":      "1.0.0",
					"installedAt":  "2026-01-01T00:00:00.000Z",
					"lastUpdated":  "2026-01-01T00:00:00.000Z",
					"gitCommitSha": "abc123",
				},
			},
			"orphaned-plugin@test-marketplace": []map[string]interface{}{
				{
					"scope":        "user",
					"installPath":  filepath.Join(pluginsDir, "cache", "test-marketplace", "orphaned-plugin", "1.0.0"),
					"version":      "1.0.0",
					"installedAt":  "2026-01-01T00:00:00.000Z",
					"lastUpdated":  "2026-01-01T00:00:00.000Z",
					"gitCommitSha": "def456",
				},
			},
		},
	}
	installedJSON, _ := json.MarshalIndent(installedPlugins, "", "  ")
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644); err != nil {
		t.Fatalf("failed to write installed_plugins.json: %v", err)
	}

	// Create the marketplace directory with ONLY good-plugin present.
	// orphaned-plugin's directory is intentionally absent.
	goodPluginDir := filepath.Join(pluginsDir, "marketplaces", "test-marketplace", "plugins", "good-plugin")
	if err := os.MkdirAll(goodPluginDir, 0755); err != nil {
		t.Fatalf("failed to create good-plugin marketplace dir: %v", err)
	}
	// Do NOT create orphaned-plugin directory -- that is the whole point.

	t.Run("export excludes orphaned plugin", func(t *testing.T) {
		stdout, stderr, err := runClew(t, tmpDir, "export")
		if err != nil {
			t.Fatalf("export failed: %v\nstderr: %s", err, stderr)
		}

		// Orphaned plugin must NOT appear in the exported output
		if strings.Contains(stdout, "orphaned-plugin") {
			t.Errorf("orphaned plugin should not appear in export output, but it did.\nstdout:\n%s", stdout)
		}

		// Good plugin MUST appear in the exported output
		if !strings.Contains(stdout, "good-plugin") {
			t.Errorf("good-plugin should appear in export output, but it did not.\nstdout:\n%s", stdout)
		}

		// stderr should mention the orphaned plugin skip
		if !strings.Contains(stderr, "not found in marketplace directory") {
			t.Errorf("expected orphan skip note in stderr, got: %s", stderr)
		}
		if !strings.Contains(stderr, "orphaned-plugin@test-marketplace") {
			t.Errorf("expected orphaned plugin name in stderr, got: %s", stderr)
		}
	})

	t.Run("exported Clewfile round-trips through diff without errors", func(t *testing.T) {
		// Step 1: Export current state
		exportedYAML, stderr, err := runClew(t, tmpDir, "export")
		if err != nil {
			t.Fatalf("export failed: %v\nstderr: %s", err, stderr)
		}

		// Step 2: Write exported YAML to a temp Clewfile
		clewfilePath := filepath.Join(tmpDir, "Clewfile.yaml")
		if err := os.WriteFile(clewfilePath, []byte(exportedYAML), 0644); err != nil {
			t.Fatalf("failed to write exported Clewfile: %v", err)
		}

		// Step 3: Run diff with the exported Clewfile -- must not error
		stdout, stderr, err := runClew(t, tmpDir, "diff", "--config", clewfilePath)
		if err != nil {
			t.Fatalf("diff with exported Clewfile failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// The diff may show the orphaned plugin as "remove (not in Clewfile)"
		// since it is installed but not in the exported Clewfile.
		// The key assertion: no error exit code (checked above via err == nil).
		t.Logf("diff output:\n%s", stdout)
	})
}

// TestCapstoneSyncOutputLabel verifies that sync output uses "Unmanaged items:"
// (not the old "Items needing attention:" label) when there are plugins installed
// that are not listed in the Clewfile.
func TestCapstoneSyncOutputLabel(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Use the minimal Clewfile which only includes one marketplace and one plugin.
	// The other installed plugins (hookify, plugin-dev) and their marketplace
	// (claude-code-plugins) will appear as unmanaged items.
	minimalPath := filepath.Join(testDir, "minimal.yaml")
	fixtureContent, err := os.ReadFile("fixtures/minimal-clewfile.yaml")
	if err != nil {
		t.Fatalf("failed to read minimal-clewfile.yaml: %v", err)
	}
	if err := os.WriteFile(minimalPath, fixtureContent, 0644); err != nil {
		t.Fatalf("failed to write minimal Clewfile: %v", err)
	}

	stdout, stderr, err := runClew(t, testDir, "sync", "--config", minimalPath, "--no-backup")
	if err != nil {
		t.Fatalf("sync failed: %v\nstderr: %s", err, stderr)
	}

	t.Run("uses Unmanaged items label", func(t *testing.T) {
		if !strings.Contains(stdout, "Unmanaged items:") {
			t.Errorf("expected 'Unmanaged items:' in sync output, got:\n%s", stdout)
		}
	})

	t.Run("does not use old Items needing attention label", func(t *testing.T) {
		if strings.Contains(stdout, "Items needing attention:") {
			t.Errorf("sync output should NOT contain old label 'Items needing attention:', got:\n%s", stdout)
		}
	})
}
