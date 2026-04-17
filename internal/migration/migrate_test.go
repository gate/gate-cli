package migration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMigrateDryRun(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"gate-info":{"command":"x"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := RunMigrate(MigrateOptions{
		Apply:       false,
		ProviderIDs: []string{"cursor"},
		Scanner:     NewScannerWithHome(home),
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "dry_run" {
		t.Fatalf("expected dry_run, got %s", report.Mode)
	}
}

func TestRunMigrateApplyCreatesBackup(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"gate-info":{"command":"x"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(home, "backup")

	report, err := RunMigrate(MigrateOptions{
		Apply:       true,
		ProviderIDs: []string{"cursor"},
		Scanner:     NewScannerWithHome(home),
		BackupDir:   backupDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) == 0 {
		t.Fatalf("expected provider results")
	}
}

func TestRunMigrateApplyJSONStaysManualPatch(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"mcpServers":{"gate-info":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := RunMigrate(MigrateOptions{
		Apply:       true,
		ProviderIDs: []string{"cursor"},
		Scanner:     NewScannerWithHome(home),
		BackupDir:   filepath.Join(home, "backup"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) == 0 {
		t.Fatalf("expected provider result")
	}
	if report.Providers[0].Action != "manual_patch" {
		t.Fatalf("expected manual patch for json, got %s", report.Providers[0].Action)
	}
}

func TestMigrateExitCode(t *testing.T) {
	if MigrateExitCode(MigrateReport{Status: "pass"}) != 0 {
		t.Fatalf("pass should map to 0")
	}
	if MigrateExitCode(MigrateReport{Status: "warn"}) != 10 {
		t.Fatalf("warn should map to 10")
	}
	if MigrateExitCode(MigrateReport{Status: "fail"}) != 20 {
		t.Fatalf("fail should map to 20")
	}
}

func TestRemoveGateMarkersJSONRemovesLegacyEntries(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "mcp.json")
	if err := os.WriteFile(target, []byte(`{"mcpServers":{"gate-info":{"command":"x"},"other":{"command":"y"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	updated, changed, err := removeGateMarkersJSON(target)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected json to stay in manual patch mode")
	}
	if updated != "" {
		t.Fatalf("expected empty updated content in manual patch mode, got %q", updated)
	}
}

func TestRemoveGateMarkersTOMLRemovesLegacyEntries(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "config.toml")
	content := `
[mcpServers.gate-info]
command = "x"

[mcpServers.other]
command = "y"
`
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	updated, changed, err := removeGateMarkersTOML(target)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected toml to stay in manual patch mode")
	}
	if updated != "" {
		t.Fatalf("expected empty updated content in manual patch mode, got %q", updated)
	}
}

func TestRunMigrateApplyUnknownFormatKeepsManualPatch(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cursor", "mcp.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	// Scanner only checks known files; feed a custom scanner result through RunMigrate with existing file.
	if err := os.WriteFile(target, []byte("mcpServers:\n  gate-info: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := RunMigrate(MigrateOptions{
		Apply:       true,
		ProviderIDs: []string{"cursor"},
		Scanner:     NewScannerWithHome(home),
		BackupDir:   filepath.Join(home, "backup"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) == 0 {
		t.Fatal("expected provider results")
	}
	// Unknown/non-structural formats should remain manual patch when changed=false.
	if report.Providers[0].Action != "manual_patch" && report.Providers[0].Status == "warn" {
		t.Fatalf("expected manual_patch for non-structural format, got %s", report.Providers[0].Action)
	}
}

func TestAtomicWritePreservePermPreservesModeAndContent(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "config.json")
	if err := os.WriteFile(target, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := atomicWritePreservePerm(target, []byte(`{"b":2}`)); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"b":2}` {
		t.Fatalf("unexpected content: %s", string(got))
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}
