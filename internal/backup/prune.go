package backup

import (
	"fmt"
)

// DefaultKeepCount is the default number of backups to retain.
const DefaultKeepCount = 30

// PruneResult contains information about what was pruned.
type PruneResult struct {
	Deleted []BackupInfo
	Kept    int
}

// Prune removes old backups, keeping only the most recent N backups.
func (m *Manager) Prune(keep int) (*PruneResult, error) {
	if keep < 0 {
		return nil, fmt.Errorf("keep count must be non-negative")
	}

	backups, err := m.List()
	if err != nil {
		return nil, err
	}

	result := &PruneResult{}

	// Backups are already sorted newest first
	if len(backups) <= keep {
		result.Kept = len(backups)
		return result, nil
	}

	// Keep the first 'keep' backups, delete the rest
	toDelete := backups[keep:]
	result.Kept = keep

	for _, backup := range toDelete {
		if err := m.Delete(backup.ID); err != nil {
			return nil, fmt.Errorf("failed to delete backup %s: %w", backup.ID, err)
		}
		result.Deleted = append(result.Deleted, backup)
	}

	return result, nil
}
