package migration

import (
	"os"
	"path/filepath"
	"strings"
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

func TestContainsGateTokenStructuredJSONExactMatch(t *testing.T) {
	raw := `{"mcpServers":{"gate-info":{"command":"x"}}}`
	if !containsGateToken(strings.ToLower(raw), raw, "gate-info") {
		t.Fatalf("expected structured JSON exact match")
	}
}

func TestContainsGateTokenStructuredJSONAvoidsURLFalsePositive(t *testing.T) {
	raw := `{"url":"https://example.com/gate-info"}`
	if containsGateToken(strings.ToLower(raw), raw, "gate-info") {
		t.Fatalf("did not expect URL-only match")
	}
}

func TestScannerAllProvidersUsesStableOrder(t *testing.T) {
	sc := NewScannerWithHome(t.TempDir())
	items := sc.Scan(nil)
	if len(items) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(items))
	}
	if items[0].ProviderID != "codex" || items[1].ProviderID != "cursor" || items[2].ProviderID != "claude_desktop" {
		t.Fatalf("unexpected provider order: %#v", []string{items[0].ProviderID, items[1].ProviderID, items[2].ProviderID})
	}
}

func TestScanProviderClearsErrorWhenEntryFound(t *testing.T) {
	home := t.TempDir()
	good := filepath.Join(home, ".cursor", "mcp.json")
	bad := filepath.Join(home, ".cursor", "config.json")
	if err := os.MkdirAll(filepath.Dir(good), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte(`{"mcpServers":{"gate-info":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(`{"mcpServers":{"x":{"command":"y"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(good, bad); err != nil {
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
	if items[0].Error != "" {
		t.Fatalf("expected empty error when entries found, got %q", items[0].Error)
	}
	// Second path is a symlink into the first file; scanner follows it within home (CR-1009).
	if len(items[0].Warnings) == 0 {
		t.Fatalf("expected symlink follow warning, got none")
	}
}

func TestScannerFollowsSymlinkWithinHome(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(home, ".cursor", "real-mcp.json")
	link := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte(`{"mcpServers":{"gate-news":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	sc := NewScannerWithHome(home)
	items := sc.Scan([]string{"cursor"})
	if len(items) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(items))
	}
	if items[0].EntriesFound == 0 {
		t.Fatalf("expected entries via symlink")
	}
	if len(items[0].Warnings) == 0 || !strings.Contains(items[0].Warnings[0], "followed symlink") {
		t.Fatalf("expected followed symlink warning, got %#v", items[0].Warnings)
	}
}

func TestScannerRefusesSymlinkTargetOutsideHome(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	outFile := filepath.Join(outside, "secret.json")
	if err := os.WriteFile(outFile, []byte(`{"mcpServers":{"gate-info":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outFile, link); err != nil {
		t.Fatal(err)
	}

	sc := NewScannerWithHome(home)
	items := sc.Scan([]string{"cursor"})
	if len(items) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(items))
	}
	if items[0].EntriesFound != 0 {
		t.Fatalf("expected no entries when symlink escapes home")
	}
	if items[0].Error == "" || !strings.Contains(items[0].Error, "outside home") {
		t.Fatalf("expected outside-home error, got Error=%q", items[0].Error)
	}
}
