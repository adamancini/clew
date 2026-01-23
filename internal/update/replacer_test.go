package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewBinaryReplacer(t *testing.T) {
	replacer := NewBinaryReplacer("/usr/local/bin/clew")

	if replacer.currentPath != "/usr/local/bin/clew" {
		t.Errorf("currentPath = %s, want /usr/local/bin/clew", replacer.currentPath)
	}

	expectedBackup := "/usr/local/bin/clew.backup"
	if replacer.backupPath != expectedBackup {
		t.Errorf("backupPath = %s, want %s", replacer.backupPath, expectedBackup)
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "clew")
	testContent := []byte("original binary content")

	// Create current binary
	if err := os.WriteFile(currentBinary, testContent, 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	replacer := NewBinaryReplacer(currentBinary)
	err := replacer.createBackup()
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Verify backup exists
	backupContent, err := os.ReadFile(replacer.backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != string(testContent) {
		t.Errorf("Backup content mismatch: got %s, want %s", backupContent, testContent)
	}

	// Verify permissions
	info, err := os.Stat(replacer.backupPath)
	if err != nil {
		t.Fatalf("Failed to stat backup: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("Backup permissions = %o, want 0755", info.Mode().Perm())
	}
}

func TestCreateBackup_FileNotFound(t *testing.T) {
	replacer := NewBinaryReplacer("/path/that/does/not/exist")
	err := replacer.createBackup()
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestRollback_Success(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "test-binary")
	originalContent := []byte("#!/bin/sh\necho test version 1.0.0\n")

	// Create original binary
	if err := os.WriteFile(currentBinary, originalContent, 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	replacer := NewBinaryReplacer(currentBinary)

	// Create backup
	if err := replacer.createBackup(); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Simulate failed update by replacing with bad binary
	badContent := []byte("#!/bin/sh\nexit 1\n")
	if err := os.WriteFile(currentBinary, badContent, 0755); err != nil {
		t.Fatalf("Failed to write bad binary: %v", err)
	}

	// Rollback
	err := replacer.Rollback()
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Verify original content restored
	restoredContent, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("Failed to read restored binary: %v", err)
	}

	if string(restoredContent) != string(originalContent) {
		t.Errorf("Restored content mismatch")
	}

	// Verify backup was consumed (renamed to current)
	if _, err := os.Stat(replacer.backupPath); !os.IsNotExist(err) {
		t.Error("Backup should not exist after rollback")
	}
}

func TestRollback_NoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "test-binary")

	replacer := NewBinaryReplacer(currentBinary)

	// Try to rollback without backup
	err := replacer.Rollback()
	if err == nil {
		t.Error("Expected error when backup doesn't exist")
	}
}

func TestReplace_Success(t *testing.T) {
	// Skip this test if we can't create test scripts
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "clew")
	newBinary := filepath.Join(tmpDir, "clew-new")

	// Create current binary (a simple script that responds to --version)
	currentScript := `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "clew version 0.8.2"
	exit 0
fi
exit 1
`
	if err := os.WriteFile(currentBinary, []byte(currentScript), 0755); err != nil {
		t.Fatalf("Failed to create current binary: %v", err)
	}

	// Create new binary
	newScript := `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "clew version 0.9.0"
	exit 0
fi
exit 1
`
	if err := os.WriteFile(newBinary, []byte(newScript), 0755); err != nil {
		t.Fatalf("Failed to create new binary: %v", err)
	}

	replacer := NewBinaryReplacer(currentBinary)
	err := replacer.Replace(newBinary)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	// Verify new binary is in place
	content, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("Failed to read replaced binary: %v", err)
	}

	if string(content) != newScript {
		t.Error("Binary was not replaced")
	}

	// Verify backup was removed
	if _, err := os.Stat(replacer.backupPath); !os.IsNotExist(err) {
		t.Error("Backup should be removed after successful replacement")
	}

	// Verify new binary is executable
	info, err := os.Stat(currentBinary)
	if err != nil {
		t.Fatalf("Failed to stat binary: %v", err)
	}

	if info.Mode().Perm()&0111 == 0 {
		t.Error("Binary should be executable")
	}
}

func TestReplace_VerificationFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "clew")
	newBinary := filepath.Join(tmpDir, "clew-new")

	// Create working current binary
	currentScript := `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "clew version 0.8.2"
	exit 0
fi
exit 1
`
	if err := os.WriteFile(currentBinary, []byte(currentScript), 0755); err != nil {
		t.Fatalf("Failed to create current binary: %v", err)
	}

	// Create broken new binary (fails on --version)
	brokenScript := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(newBinary, []byte(brokenScript), 0755); err != nil {
		t.Fatalf("Failed to create new binary: %v", err)
	}

	replacer := NewBinaryReplacer(currentBinary)
	err := replacer.Replace(newBinary)
	if err == nil {
		t.Error("Expected error for broken binary")
	}

	// Verify original binary is still in place (rollback happened)
	content, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("Failed to read binary: %v", err)
	}

	if string(content) != currentScript {
		t.Error("Original binary should be restored after failed verification")
	}
}

func TestReplace_BackupFails(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "clew")
	newBinary := filepath.Join(tmpDir, "clew-new")

	// Don't create current binary - backup will fail

	// Create new binary
	if err := os.WriteFile(newBinary, []byte("new content"), 0755); err != nil {
		t.Fatalf("Failed to create new binary: %v", err)
	}

	replacer := NewBinaryReplacer(currentBinary)
	err := replacer.Replace(newBinary)
	if err == nil {
		t.Error("Expected error when backup fails")
	}

	// Verify new binary still exists (wasn't consumed)
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		t.Error("New binary should still exist after failed backup")
	}
}

func TestVerifyBinary_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	testBinary := filepath.Join(tmpDir, "test")

	// Create a simple script that succeeds for --version
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
	echo "test version 1.0.0"
	exit 0
fi
exit 1
`
	if err := os.WriteFile(testBinary, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	replacer := NewBinaryReplacer(testBinary)
	err := replacer.verifyBinary(testBinary)
	if err != nil {
		t.Errorf("verifyBinary() error = %v", err)
	}
}

func TestVerifyBinary_Fails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	testBinary := filepath.Join(tmpDir, "test")

	// Create a script that fails
	script := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(testBinary, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	replacer := NewBinaryReplacer(testBinary)
	err := replacer.verifyBinary(testBinary)
	if err == nil {
		t.Error("Expected error for failing binary")
	}
}

func TestVerifyBinary_NotExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	testBinary := filepath.Join(tmpDir, "test")

	// Create a file without execute permissions
	if err := os.WriteFile(testBinary, []byte("#!/bin/sh\necho test\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	replacer := NewBinaryReplacer(testBinary)
	err := replacer.verifyBinary(testBinary)
	if err == nil {
		t.Error("Expected error for non-executable binary")
	}
}

func TestVerifyBinary_NotFound(t *testing.T) {
	replacer := NewBinaryReplacer("/path/that/does/not/exist")
	err := replacer.verifyBinary("/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent binary")
	}
}

// Test that we can actually run the real clew binary if it exists
func TestVerifyBinary_RealBinary(t *testing.T) {
	// Find clew in PATH
	clewPath, err := exec.LookPath("clew")
	if err != nil {
		t.Skip("clew not found in PATH, skipping real binary test")
	}

	replacer := NewBinaryReplacer(clewPath)
	err = replacer.verifyBinary(clewPath)
	if err != nil {
		t.Errorf("Failed to verify real clew binary: %v", err)
	}
}
