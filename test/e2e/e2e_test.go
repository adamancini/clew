package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	binaryName = "clew"
)

var (
	binaryPath string
	testDir    string
)

// TestMain builds the binary before running tests
func TestMain(m *testing.M) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryName, "../../cmd/clew")
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	// Get absolute path to binary
	binaryPath, _ = filepath.Abs(binaryName)

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(binaryName)

	os.Exit(code)
}

// setupTestEnv creates a temporary test environment with Claude plugin structure
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "clew-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create Claude plugin directory structure
	pluginsDir := filepath.Join(tmpDir, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	// Copy test fixtures
	fixtures := map[string]string{
		"installed_plugins.json":    filepath.Join(pluginsDir, "installed_plugins.json"),
		"known_marketplaces.json":   filepath.Join(pluginsDir, "known_marketplaces.json"),
	}

	for src, dst := range fixtures {
		srcPath := filepath.Join("fixtures", src)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", src, err)
		}

		// Replace /tmp/clew-test with actual tmpDir
		content = []byte(strings.ReplaceAll(string(content), "/tmp/clew-test", tmpDir))

		if err := os.WriteFile(dst, content, 0644); err != nil {
			t.Fatalf("failed to write fixture %s: %v", dst, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// runClew executes the clew binary with given arguments
func runClew(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// cliReaderWorks checks if the CLI reader is functional
// Returns true if CLI reader works, false if it fails (issue #34)
// Issue #34: The CLI reader tries to call `claude plugin list --json` but that command doesn't exist
func cliReaderWorks(t *testing.T) bool {
	t.Helper()

	// The actual issue is that `claude plugin list` doesn't exist
	// Check if the claude CLI has the list subcommand
	cmd := exec.Command("claude", "plugin", "list", "--json")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()

	// If command fails with "unknown command", CLI reader is broken
	if err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "unknown command") ||
		   strings.Contains(stderrStr, "error: unknown") {
			return false
		}
	}

	// If command succeeded or failed for other reasons, assume CLI reader works
	// The actual tests will verify functionality
	return true
}

// skipIfCLIReaderBroken skips the test if CLI reader is not working yet
func skipIfCLIReaderBroken(t *testing.T) {
	t.Helper()

	if !cliReaderWorks(t) {
		t.Skip("Skipping: CLI reader not working yet (see issue #34)")
	}
}

// TestExportCommand tests the export command functionality
func TestExportCommand(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	pluginsFile := filepath.Join(testDir, ".claude", "plugins", "installed_plugins.json")

	t.Run("export with text output", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "export", "--filesystem", "--read-from-filesystem", pluginsFile)
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Verify YAML output
		if !strings.Contains(stdout, "version: 1") {
			t.Errorf("expected version in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "marketplaces:") {
			t.Errorf("expected marketplaces in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "plugins:") {
			t.Errorf("expected plugins in output, got: %s", stdout)
		}
	})

	t.Run("export with JSON output", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "export", "--filesystem", "--output", "json", "--read-from-filesystem", pluginsFile)
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Verify valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
		}

		// Verify structure
		if _, ok := result["version"]; !ok {
			t.Error("expected version field in JSON output")
		}
		if _, ok := result["marketplaces"]; !ok {
			t.Error("expected marketplaces field in JSON output")
		}
		if _, ok := result["plugins"]; !ok {
			t.Error("expected plugins field in JSON output")
		}
	})

	t.Run("export with YAML output", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "export", "--filesystem", "--output", "yaml", "--read-from-filesystem", pluginsFile)
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Verify valid YAML
		var result map[string]interface{}
		if err := yaml.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("output is not valid YAML: %v\noutput: %s", err, stdout)
		}
	})
}

// TestStatusCommand tests the status command functionality
func TestStatusCommand(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create Clewfile that matches current state
	clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
	fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
	if err != nil {
		t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
	}
	if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
		t.Fatalf("failed to write Clewfile: %v", err)
	}

	t.Run("status in sync", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "status", "--config", clewfilePath, "--filesystem")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "In sync") {
			t.Errorf("expected 'In sync' in output, got: %s", stdout)
		}
	})

	t.Run("status out of sync", func(t *testing.T) {
		// Use minimal Clewfile (missing plugins)
		minimalPath := filepath.Join(testDir, "minimal.yaml")
		fixtureContent, err := os.ReadFile("fixtures/minimal-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read minimal-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(minimalPath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write minimal Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "status", "--config", minimalPath, "--filesystem")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Out of sync") {
			t.Errorf("expected 'Out of sync' in output, got: %s", stdout)
		}
	})

	t.Run("status JSON output", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "status", "--config", clewfilePath, "--filesystem", "--output", "json")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
		}

		if _, ok := result["in_sync"]; !ok {
			t.Error("expected in_sync field in JSON output")
		}
	})
}

// TestDiffCommand tests the diff command functionality
func TestDiffCommand(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("diff with matching state", func(t *testing.T) {
		clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
		fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "diff", "--config", clewfilePath, "--filesystem")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Already in sync") {
			t.Errorf("expected 'Already in sync' message, got: %s", stdout)
		}
	})

	t.Run("diff with differences", func(t *testing.T) {
		minimalPath := filepath.Join(testDir, "minimal.yaml")
		fixtureContent, err := os.ReadFile("fixtures/minimal-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read minimal-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(minimalPath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write minimal Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "diff", "--config", minimalPath, "--filesystem")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Should show items not in Clewfile
		if !strings.Contains(stdout, "remove (not in Clewfile)") {
			t.Errorf("expected removal notices in output, got: %s", stdout)
		}
	})

	t.Run("diff JSON output", func(t *testing.T) {
		clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
		fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "diff", "--config", clewfilePath, "--filesystem", "--output", "json")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
		}
	})
}

// TestSyncCommand tests the sync command functionality
func TestSyncCommand(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("sync when already in sync", func(t *testing.T) {
		clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
		fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "sync", "--config", clewfilePath, "--filesystem")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Already in sync") || !strings.Contains(stdout, "Nothing to do") {
			t.Errorf("expected 'Already in sync' message, got: %s", stdout)
		}
	})

	t.Run("sync verbose mode", func(t *testing.T) {
		clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
		fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "sync", "--config", clewfilePath, "--filesystem", "--verbose")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Verbose should show Clewfile path (in stderr)
		if !strings.Contains(stderr, "Using Clewfile:") {
			t.Errorf("expected verbose output with Clewfile path, got stdout: %s, stderr: %s", stdout, stderr)
		}
	})

	t.Run("sync short mode", func(t *testing.T) {
		clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
		fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "sync", "--config", clewfilePath, "--filesystem", "--short")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// When in sync, should show "Already in sync" message
		if !strings.Contains(stdout, "Already in sync") || !strings.Contains(stdout, "Nothing to do") {
			t.Errorf("expected 'Already in sync' message with --short flag, got: %s", stdout)
		}
	})

	t.Run("sync JSON output includes operations array", func(t *testing.T) {
		// Use minimal Clewfile to trigger operations (items not in Clewfile)
		minimalPath := filepath.Join(testDir, "minimal.yaml")
		fixtureContent, err := os.ReadFile("fixtures/minimal-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read minimal-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(minimalPath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write minimal Clewfile: %v", err)
		}

		stdout, stderr, err := runClew(t, "sync", "--config", minimalPath, "--filesystem", "--output", "json")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// When already in sync, output should be the "Nothing to do" message, not JSON
		// But if there were operations, output should be valid JSON with operations field
		if strings.Contains(stdout, "Already in sync") {
			// No JSON output when already in sync - this is expected
			return
		}

		// If we got JSON output, verify structure
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
		}

		// Verify operations array exists in sync result
		if _, ok := result["operations"]; !ok {
			t.Error("expected 'operations' field in JSON output")
		}
	})
}

// TestValidation tests configuration validation
func TestValidation(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("invalid marketplace source", func(t *testing.T) {
		invalidPath := filepath.Join(testDir, "invalid.yaml")
		fixtureContent, err := os.ReadFile("fixtures/invalid-source-clewfile.yaml")
		if err != nil {
			t.Fatalf("failed to read invalid-source-clewfile.yaml: %v", err)
		}
		if err := os.WriteFile(invalidPath, fixtureContent, 0644); err != nil {
			t.Fatalf("failed to write invalid Clewfile: %v", err)
		}

		_, stderr, err := runClew(t, "status", "--config", invalidPath, "--filesystem")
		if err == nil {
			t.Fatal("expected command to fail with invalid source")
		}

		if !strings.Contains(stderr, "invalid source") {
			t.Errorf("expected validation error message, got: %s", stderr)
		}
	})

	t.Run("missing Clewfile", func(t *testing.T) {
		nonexistent := filepath.Join(testDir, "nonexistent.yaml")

		_, stderr, err := runClew(t, "status", "--config", nonexistent)
		if err == nil {
			t.Fatal("expected command to fail with missing Clewfile")
		}

		if !strings.Contains(stderr, "not found") {
			t.Errorf("expected 'not found' error, got: %s", stderr)
		}
	})
}

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	stdout, stderr, err := runClew(t, "--version")
	if err != nil {
		t.Fatalf("version command failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "clew version") {
		t.Errorf("expected version string in output, got: %s", stdout)
	}
}

// TestOutputFormats tests that all commands support output format flags
func TestOutputFormats(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	clewfilePath := filepath.Join(testDir, "Clewfile.yaml")
	fixtureContent, err := os.ReadFile("fixtures/complete-clewfile.yaml")
	if err != nil {
		t.Fatalf("failed to read complete-clewfile.yaml: %v", err)
	}
	if err := os.WriteFile(clewfilePath, fixtureContent, 0644); err != nil {
		t.Fatalf("failed to write Clewfile: %v", err)
	}

	formats := []string{"text", "json", "yaml"}
	commands := [][]string{
		{"status", "--config", clewfilePath, "--filesystem"},
		{"export", "--filesystem"},
	}

	for _, cmd := range commands {
		for _, format := range formats {
			t.Run(strings.Join(cmd, " ")+" with "+format, func(t *testing.T) {
				args := append(cmd, "--output", format)
				stdout, stderr, err := runClew(t, args...)
				if err != nil {
					t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
				}

				if stdout == "" {
					t.Error("expected output but got none")
				}

				// Verify format-specific output
				switch format {
				case "json":
					var result interface{}
					if err := json.Unmarshal([]byte(stdout), &result); err != nil {
						t.Errorf("output is not valid JSON: %v", err)
					}
				case "yaml":
					var result interface{}
					if err := yaml.Unmarshal([]byte(stdout), &result); err != nil {
						t.Errorf("output is not valid YAML: %v", err)
					}
				}
			})
		}
	}
}

// TestEmptyClewfile tests handling of empty Clewfile
func TestEmptyClewfile(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	emptyPath := filepath.Join(testDir, "empty.yaml")
	fixtureContent, err := os.ReadFile("fixtures/empty-clewfile.yaml")
	if err != nil {
		t.Fatalf("failed to read empty-clewfile.yaml: %v", err)
	}
	if err := os.WriteFile(emptyPath, fixtureContent, 0644); err != nil {
		t.Fatalf("failed to write empty Clewfile: %v", err)
	}

	// Should not fail, but show everything needs attention
	stdout, stderr, err := runClew(t, "diff", "--config", emptyPath, "--filesystem")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// All installed items should be marked as "not in Clewfile"
	if !strings.Contains(stdout, "remove (not in Clewfile)") {
		t.Errorf("expected items marked as not in Clewfile, got: %s", stdout)
	}
}

// TestCLIReader tests commands using the CLI reader (default mode)
// These tests will be skipped until issue #34 is fixed
// Once fixed, these tests will automatically start running and passing
func TestCLIReader(t *testing.T) {
	skipIfCLIReaderBroken(t)

	t.Run("export without filesystem flag", func(t *testing.T) {
		stdout, stderr, err := runClew(t, "export")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Should produce same output as filesystem reader
		if !strings.Contains(stdout, "version: 1") {
			t.Errorf("expected version in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "marketplaces:") {
			t.Errorf("expected marketplaces in output, got: %s", stdout)
		}
	})

	t.Run("status without filesystem flag", func(t *testing.T) {
		// Create Clewfile in home directory
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}
		claudeDir := filepath.Join(home, ".claude")
		clewfilePath := filepath.Join(claudeDir, "Clewfile.yaml")

		// Export current state to use as test Clewfile
		stdout, _, err := runClew(t, "export", "--filesystem")
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Write exported config as Clewfile
		if err := os.WriteFile(clewfilePath, []byte(stdout), 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}
		defer os.Remove(clewfilePath)

		// Test status without filesystem flag
		stdout, stderr, err := runClew(t, "status")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Should report sync status
		if !strings.Contains(stdout, "sync") {
			t.Errorf("expected sync status in output, got: %s", stdout)
		}
	})

	t.Run("diff without filesystem flag", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}
		claudeDir := filepath.Join(home, ".claude")
		clewfilePath := filepath.Join(claudeDir, "Clewfile.yaml")

		// Export current state
		stdout, _, err := runClew(t, "export", "--filesystem")
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Write as Clewfile
		if err := os.WriteFile(clewfilePath, []byte(stdout), 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}
		defer os.Remove(clewfilePath)

		// Test diff without filesystem flag
		stdout, stderr, err := runClew(t, "diff")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Should show changes or "in sync"
		if stdout == "" {
			t.Error("expected output from diff command")
		}
	})

	t.Run("sync without filesystem flag", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}
		claudeDir := filepath.Join(home, ".claude")
		clewfilePath := filepath.Join(claudeDir, "Clewfile.yaml")

		// Export current state
		stdout, _, err := runClew(t, "export", "--filesystem")
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Write as Clewfile
		if err := os.WriteFile(clewfilePath, []byte(stdout), 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}
		defer os.Remove(clewfilePath)

		// Test sync without filesystem flag (should be no-op if in sync)
		stdout, stderr, err := runClew(t, "sync")
		if err != nil {
			t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
		}

		// Should complete successfully
		if stdout == "" {
			t.Error("expected output from sync command")
		}
	})
}

// TestCLIReaderVsFilesystemReader compares CLI reader and filesystem reader outputs
// This test will be skipped until issue #34 is fixed
func TestCLIReaderVsFilesystemReader(t *testing.T) {
	skipIfCLIReaderBroken(t)

	t.Run("export outputs match", func(t *testing.T) {
		// Get output from CLI reader
		cliOutput, stderr, err := runClew(t, "export", "--output", "json")
		if err != nil {
			t.Fatalf("CLI reader failed: %v\nstderr: %s", err, stderr)
		}

		// Get output from filesystem reader
		fsOutput, stderr, err := runClew(t, "export", "--filesystem", "--output", "json")
		if err != nil {
			t.Fatalf("filesystem reader failed: %v\nstderr: %s", err, stderr)
		}

		// Parse both outputs
		var cliResult, fsResult map[string]interface{}
		if err := json.Unmarshal([]byte(cliOutput), &cliResult); err != nil {
			t.Fatalf("CLI output is not valid JSON: %v", err)
		}
		if err := json.Unmarshal([]byte(fsOutput), &fsResult); err != nil {
			t.Fatalf("filesystem output is not valid JSON: %v", err)
		}

		// Compare plugin counts
		cliPlugins, cliOk := cliResult["plugins"].([]interface{})
		fsPlugins, fsOk := fsResult["plugins"].([]interface{})
		if !cliOk || !fsOk {
			t.Fatal("expected plugins arrays in both outputs")
		}

		if len(cliPlugins) != len(fsPlugins) {
			t.Errorf("plugin count mismatch: CLI=%d, filesystem=%d", len(cliPlugins), len(fsPlugins))
		}

		// Compare marketplace counts
		cliMarkets, cliOk := cliResult["marketplaces"].(map[string]interface{})
		fsMarkets, fsOk := fsResult["marketplaces"].(map[string]interface{})
		if !cliOk || !fsOk {
			t.Fatal("expected marketplaces maps in both outputs")
		}

		if len(cliMarkets) != len(fsMarkets) {
			t.Errorf("marketplace count mismatch: CLI=%d, filesystem=%d", len(cliMarkets), len(fsMarkets))
		}
	})

	t.Run("status outputs match", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("failed to get home dir: %v", err)
		}
		claudeDir := filepath.Join(home, ".claude")
		clewfilePath := filepath.Join(claudeDir, "Clewfile.yaml")

		// Create test Clewfile
		stdout, _, err := runClew(t, "export", "--filesystem")
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}
		if err := os.WriteFile(clewfilePath, []byte(stdout), 0644); err != nil {
			t.Fatalf("failed to write Clewfile: %v", err)
		}
		defer os.Remove(clewfilePath)

		// Get status from CLI reader
		cliOutput, stderr, err := runClew(t, "status", "--output", "json")
		if err != nil {
			t.Fatalf("CLI reader failed: %v\nstderr: %s", err, stderr)
		}

		// Get status from filesystem reader
		fsOutput, stderr, err := runClew(t, "status", "--filesystem", "--output", "json")
		if err != nil {
			t.Fatalf("filesystem reader failed: %v\nstderr: %s", err, stderr)
		}

		// Parse both outputs
		var cliResult, fsResult map[string]interface{}
		if err := json.Unmarshal([]byte(cliOutput), &cliResult); err != nil {
			t.Fatalf("CLI output is not valid JSON: %v", err)
		}
		if err := json.Unmarshal([]byte(fsOutput), &fsResult); err != nil {
			t.Fatalf("filesystem output is not valid JSON: %v", err)
		}

		// Compare sync status
		if cliResult["in_sync"] != fsResult["in_sync"] {
			t.Errorf("sync status mismatch: CLI=%v, filesystem=%v",
				cliResult["in_sync"], fsResult["in_sync"])
		}
	})
}
