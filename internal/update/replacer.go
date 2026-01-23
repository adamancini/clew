package update

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// BinaryReplacer safely replaces the binary with rollback support
type BinaryReplacer struct {
	currentPath string
	backupPath  string
}

// NewBinaryReplacer creates a new binary replacer
func NewBinaryReplacer(currentPath string) *BinaryReplacer {
	return &BinaryReplacer{
		currentPath: currentPath,
		backupPath:  currentPath + ".backup",
	}
}

// Replace replaces the current binary with the new one
func (r *BinaryReplacer) Replace(newBinary string) error {
	// 1. Create backup of current binary
	if err := r.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 2. Replace with new binary (atomic rename)
	if err := os.Rename(newBinary, r.currentPath); err != nil {
		// Attempt rollback
		_ = r.Rollback()
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// 3. Set executable permissions
	if err := os.Chmod(r.currentPath, 0755); err != nil {
		// Attempt rollback
		_ = r.Rollback()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// 4. Verify new binary works
	if err := r.verifyBinary(r.currentPath); err != nil {
		// Rollback on verification failure
		_ = r.Rollback()
		return fmt.Errorf("new binary verification failed: %w", err)
	}

	// 5. Remove backup on success
	_ = os.Remove(r.backupPath)

	return nil
}

// Rollback restores the backup if update fails
func (r *BinaryReplacer) Rollback() error {
	// 1. Check if backup exists
	if _, err := os.Stat(r.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", r.backupPath)
	}

	// 2. Restore from backup
	if err := os.Rename(r.backupPath, r.currentPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// 3. Set permissions
	if err := os.Chmod(r.currentPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on restored binary: %w", err)
	}

	// 4. Verify restored binary works
	if err := r.verifyBinary(r.currentPath); err != nil {
		return fmt.Errorf("restored binary verification failed: %w", err)
	}

	return nil
}

// createBackup creates a backup of the current binary
func (r *BinaryReplacer) createBackup() error {
	// Open source file
	src, err := os.Open(r.currentPath)
	if err != nil {
		return fmt.Errorf("failed to open current binary: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Get source file info for permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %w", err)
	}

	// Create backup file
	dst, err := os.OpenFile(r.backupPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	// Copy contents
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(r.backupPath) // Clean up partial backup
		return fmt.Errorf("failed to copy binary to backup: %w", err)
	}

	return nil
}

// verifyBinary verifies a binary works by running --version
func (r *BinaryReplacer) verifyBinary(path string) error {
	cmd := exec.Command(path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}
	return nil
}
