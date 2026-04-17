package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFileReadsViaOpenReducesTOCTOU(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.json")
	if err := os.WriteFile(src, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(dir, "bak")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatal(err)
	}

	dst, err := backupFile(src, backupDir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"x":1}` {
		t.Fatalf("unexpected backup: %q", string(raw))
	}
}
