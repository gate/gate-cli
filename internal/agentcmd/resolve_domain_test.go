//go:build agent

package agentcmd

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAgentResolveDomainFiltersResolvedLeaves(t *testing.T) {
	t.Parallel()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.AddCommand(NewResolveCmd())
	root.SetArgs([]string{"agent-resolve", "--query", "BTC brief", "--domain", "news", "--format", "json"})

	execErr := root.Execute()
	_ = w.Close()
	os.Stdout = oldStdout
	stdoutBytes, _ := io.ReadAll(r)
	if execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdoutBytes, &out); err != nil {
		t.Fatalf("parse stdout: %v body=%s", err, stdoutBytes)
	}
	if out["domain"] != "news" {
		t.Fatalf("expected domain news, got %#v", out["domain"])
	}
	leaves, _ := out["resolved_leaves"].([]interface{})
	if len(leaves) == 0 {
		t.Fatalf("expected resolved_leaves, got %s", stdoutBytes)
	}
	for _, item := range leaves {
		m, _ := item.(map[string]interface{})
		cmdStr, _ := m["command"].(string)
		if cmdStr != "" && !strings.HasPrefix(cmdStr, "gate-cli news ") {
			t.Fatalf("resolved_leaves must be news-only, got %q", cmdStr)
		}
	}
}
