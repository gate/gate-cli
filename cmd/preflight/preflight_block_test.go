package preflight

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/migration"
	"github.com/gate/gate-cli/internal/version"
)

func TestPreflightBlockStderrOnlyJSON(t *testing.T) {
	prev := version.Version
	version.Version = "0.1.0"
	t.Cleanup(func() { version.Version = prev })

	res := migration.BuildPreflight(migration.PreflightOptions{
		Installed: func(string) bool { return true },
		Version:   version.Version,
	})
	if res.Route != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", res.Route)
	}

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "pretty", "")
	root.PersistentFlags().Int64("max-output-bytes", 0, "")
	root.AddCommand(Cmd)

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"preflight", "--format", "json"})

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
		t.Fatalf("BLOCK must not write stdout, got %q", stdout.String())
	}
	if !strings.Contains(string(stderrBytes), `"error"`) {
		t.Fatalf("expected stderr GateError JSON, got %q", stderrBytes)
	}
	var wrap map[string]interface{}
	if err := json.Unmarshal(stderrBytes, &wrap); err != nil {
		t.Fatalf("stderr JSON: %v body=%q", err, stderrBytes)
	}
	errObj, _ := wrap["error"].(map[string]interface{})
	if errObj["error_type"] == nil || errObj["suggested_next_action"] == nil {
		t.Fatalf("missing convergence fields: %#v", errObj)
	}
	if retry, _ := errObj["retryable"].(bool); retry {
		t.Fatalf("PREFLIGHT_BLOCKED must not be retryable: %#v", errObj)
	}
}
