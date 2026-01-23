package update

import "testing"

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

func TestBinaryReplacerReplace(t *testing.T) {
	replacer := NewBinaryReplacer("/usr/local/bin/clew")

	// Currently returns not implemented
	err := replacer.Replace("/tmp/clew-new")
	if err == nil {
		t.Error("Expected not implemented error")
	}
}

func TestBinaryReplacerRollback(t *testing.T) {
	replacer := NewBinaryReplacer("/usr/local/bin/clew")

	// Currently returns not implemented
	err := replacer.Rollback()
	if err == nil {
		t.Error("Expected not implemented error")
	}
}
