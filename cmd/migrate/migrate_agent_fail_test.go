//go:build agent

package migrate

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/migration"
)

func TestMigrateAgentFailStderrOnlyJSON(t *testing.T) {
	// Workspace-local temp: sandbox may block mkdir under system $TMPDIR/.cursor.
	home, err := os.MkdirTemp(".", "migrate-agent-fail-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	target := filepath.Join(home, ".config", "codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(`{"gate-info":{"command":"x"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(home, "backup")
	if err := os.MkdirAll(backupDir, 0o500); err != nil { // no write: backup fails on apply
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Cleanup(func() {
		_ = os.Unsetenv("HOME")
		_ = os.Unsetenv("GATE_CLI_AGENT")
	})

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().Int64("max-output-bytes", 0, "")
	root.AddCommand(Cmd)

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{
		"migrate",
		"--apply",
		"--yes",
		"--provider", "codex",
		"--backup-dir", backupDir,
		"--format", "json",
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = w.Close()
	})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error exit")
	}
	_ = w.Close()
	stderrBytes, _ := io.ReadAll(r)
	if stdout.Len() > 0 {
		t.Fatalf("agent fail must not write stdout, got %q", stdout.String())
	}
	if !strings.Contains(string(stderrBytes), `"error"`) {
		t.Fatalf("expected stderr GateError JSON, got %q", stderrBytes)
	}
	var wrap map[string]interface{}
	if err := json.Unmarshal(stderrBytes, &wrap); err != nil {
		t.Fatalf("stderr JSON: %v body=%q", err, stderrBytes)
	}
	errObj, _ := wrap["error"].(map[string]interface{})
	if errObj["label"] != "MIGRATE_FAILED" {
		t.Fatalf("expected MIGRATE_FAILED, got %#v", errObj)
	}
	if retry, _ := errObj["retryable"].(bool); retry {
		t.Fatalf("MIGRATE_FAILED must not be retryable: %#v", errObj)
	}
	msg, _ := errObj["message"].(string)
	if !strings.Contains(msg, "migrate failed") {
		t.Fatalf("expected provider failure detail in message, got %q", msg)
	}
}

func TestMigrateFailMessageUsesProviderDetail(t *testing.T) {
	t.Parallel()
	msg := migrateFailMessage(migration.MigrateReport{
		Providers: []migration.MigrateProviderResult{
			{ProviderID: "cursor", Status: "fail", ManualPatch: "failed to backup file: permission denied"},
		},
	})
	if !strings.Contains(msg, "permission denied") {
		t.Fatalf("got %q", msg)
	}
}
