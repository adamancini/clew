package update

import "fmt"

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
// Implementation coming in next phase
func (r *BinaryReplacer) Replace(newBinary string) error {
	// TODO: Implement:
	// 1. Create backup of current binary
	// 2. Atomically replace with new binary
	// 3. Set executable permissions
	// 4. Verify new binary works
	// 5. Remove backup on success
	return fmt.Errorf("not implemented yet")
}

// Rollback restores the backup if update fails
// Implementation coming in next phase
func (r *BinaryReplacer) Rollback() error {
	// TODO: Implement:
	// 1. Check if backup exists
	// 2. Restore from backup
	// 3. Set permissions
	// 4. Verify restored binary works
	return fmt.Errorf("not implemented yet")
}
