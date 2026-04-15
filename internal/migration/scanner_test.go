package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerDetectsGateMarkers(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"mcpServers":{"gate-info":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	sc := NewScannerWithHome(home)
	items := sc.Scan([]string{"cursor"})
	if len(items) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(items))
	}
	if items[0].EntriesFound == 0 {
		t.Fatalf("expected entries found")
	}
}
