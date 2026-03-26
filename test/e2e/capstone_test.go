package e2e

import (
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
