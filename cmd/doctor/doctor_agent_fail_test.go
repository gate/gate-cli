//go:build agent

package doctor

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

func TestDoctorAgentFailStderrOnlyJSON(t *testing.T) {
	prev := version.Version
	version.Version = "0.0.1"
	t.Cleanup(func() { version.Version = prev })

	t.Setenv("GATE_CLI_AGENT", "1")
	t.Cleanup(func() { _ = os.Unsetenv("GATE_CLI_AGENT") })

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	root.PersistentFlags().Int64("max-output-bytes", 0, "")
	root.AddCommand(Cmd)

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"doctor", "--check", "version", "--format", "json"})

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
	if errObj["label"] != "DOCTOR_FAILED" {
		t.Fatalf("expected DOCTOR_FAILED, got %#v", errObj)
	}
	if retry, _ := errObj["retryable"].(bool); retry {
		t.Fatalf("DOCTOR_FAILED must not be retryable: %#v", errObj)
	}
	msg, _ := errObj["message"].(string)
	if !strings.Contains(msg, "doctor checks failed") {
		t.Fatalf("expected check detail in message, got %q", msg)
	}
}

func TestDoctorFailMessageUsesFirstFailCheck(t *testing.T) {
	t.Parallel()
	msg := doctorFailMessage(migration.DoctorReport{
		Checks: []migration.DoctorCheck{
			{ID: "cli.version", Status: "fail", Message: "cli version below minimum requirement"},
		},
	})
	if !strings.Contains(msg, "cli version below minimum requirement") {
		t.Fatalf("got %q", msg)
	}
}
